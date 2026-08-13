// Package integration exposes Otter PPT through protocol-neutral stdio transports.
package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
)

const protocolVersion = "2025-03-26"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// StdioServer serves either MCP or the lightweight Otter JSON-RPC API.
type StdioServer struct {
	session *pptoolkit.Session
	in      io.Reader
	out     io.Writer
}

func NewStdioServer(in io.Reader, out io.Writer) *StdioServer {
	return &StdioServer{session: pptoolkit.NewSession(), in: in, out: out}
}

// RunMCP starts an MCP stdio server compatible with Claude Code and Cursor.
func (s *StdioServer) RunMCP() error {
	return s.run(true)
}

// RunJSONRPC starts the generic newline-delimited JSON-RPC transport.
func (s *StdioServer) RunJSONRPC() error {
	return s.run(false)
}

func (s *StdioServer) run(mcp bool) error {
	scanner := bufio.NewScanner(s.in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	encoder := json.NewEncoder(s.out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if err := encoder.Encode(rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error", Data: err.Error()}}); err != nil {
				return err
			}
			continue
		}

		result, rpcErr := s.dispatch(req, mcp)
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue
		}
		if err := encoder.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr}); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *StdioServer) dispatch(req rpcRequest, mcp bool) (any, *rpcError) {
	if mcp {
		switch req.Method {
		case "initialize":
			return map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities": map[string]any{"tools": map[string]any{"listChanged": false}},
				"serverInfo": map[string]string{"name": "otter-ppt", "version": "0.1.0"},
			}, nil
		case "ping":
			return map[string]any{}, nil
		case "tools/list":
			return map[string]any{"tools": portableTools()}, nil
		case "tools/call":
			var call toolCallParams
			if err := json.Unmarshal(req.Params, &call); err != nil {
				return nil, invalidParams(err)
			}
			result, err := s.callTool(call)
			if err != nil {
				return map[string]any{"content": []map[string]string{{"type": "text", "text": err.Error()}}, "isError": true}, nil
			}
			payload, _ := json.Marshal(result)
			return map[string]any{"content": []map[string]string{{"type": "text", "text": string(payload)}}, "structuredContent": result}, nil
		default:
			return nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method}
		}
	}

	switch req.Method {
	case "tools.list":
		return map[string]any{"tools": portableTools()}, nil
	case "tools.call":
		var call toolCallParams
		if err := json.Unmarshal(req.Params, &call); err != nil {
			return nil, invalidParams(err)
		}
		result, err := s.callTool(call)
		if err != nil {
			return nil, &rpcError{Code: -32000, Message: err.Error()}
		}
		return result, nil
	case "session.reset":
		s.session = pptoolkit.NewSession()
		return map[string]bool{"reset": true}, nil
	case "presentation.get":
		return s.session.Presentation(), nil
	default:
		return nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
}

func (s *StdioServer) callTool(call toolCallParams) (any, error) {
	if call.Arguments == nil {
		call.Arguments = map[string]any{}
	}
	switch call.Name {
	case "reset_session":
		s.session = pptoolkit.NewSession()
		return map[string]bool{"reset": true}, nil
	case "export_pptx":
		output, _ := call.Arguments["output_path"].(string)
		if output == "" {
			return nil, fmt.Errorf("output_path is required")
		}
		absolute, err := filepath.Abs(output)
		if err != nil {
			return nil, fmt.Errorf("resolve output path: %w", err)
		}
		if dir := filepath.Dir(absolute); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("create output directory: %w", err)
			}
		}
		if err := builder.New(s.session.Presentation()).Save(absolute); err != nil {
			return nil, fmt.Errorf("export PPTX: %w", err)
		}
		return map[string]any{"success": true, "output_path": absolute}, nil
	default:
		result := s.session.ExecuteTool(call.Name, call.Arguments)
		if !result.Success {
			return result, fmt.Errorf("%s", result.Message)
		}
		return result, nil
	}
}

func invalidParams(err error) *rpcError {
	return &rpcError{Code: -32602, Message: "invalid params", Data: err.Error()}
}

func portableTools() []map[string]any {
	definitions := pptoolkit.ToolDefinitions()
	tools := make([]map[string]any, 0, len(definitions)+2)
	for _, definition := range definitions {
		if definition.Function == nil {
			continue
		}
		var schema any
		raw, _ := json.Marshal(definition.Function.Parameters)
		_ = json.Unmarshal(raw, &schema)
		tools = append(tools, map[string]any{
			"name":        definition.Function.Name,
			"description": definition.Function.Description,
			"inputSchema": schema,
		})
	}
	tools = append(tools,
		map[string]any{
			"name": "export_pptx", "description": "Export the current presentation to an editable PPTX file.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"output_path": map[string]string{"type": "string", "description": "Destination .pptx path"}}, "required": []string{"output_path"}},
		},
		map[string]any{
			"name": "reset_session", "description": "Clear the current presentation and start a new session.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
	)
	return tools
}
