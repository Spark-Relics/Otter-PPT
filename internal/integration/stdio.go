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
	"github.com/otter-ppt/otter-ppt/internal/renderer"
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
	session  *pptoolkit.Session
	renderer *renderer.Renderer
	in       io.Reader
	out      io.Writer
}

func NewStdioServer(in io.Reader, out io.Writer) *StdioServer {
	return &StdioServer{
		session:  pptoolkit.NewSession(),
		renderer: renderer.NewRenderer(),
		in:       in,
		out:      out,
	}
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
			// render_slides returns MCP image content blocks for multimodal AI
			if call.Name == "render_slides" {
				result, rpcErr := s.handleRenderSlidesMCP(call)
				if rpcErr != nil {
					return nil, rpcErr
				}
				return result, nil
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
	case "render_slides":
		return s.handleRenderSlides(call.Arguments)
	default:
		result := s.session.ExecuteTool(call.Name, call.Arguments)
		if !result.Success {
			return result, fmt.Errorf("%s", result.Message)
		}
		return result, nil
	}
}

// handleRenderSlides builds the current presentation to PPTX, renders to images,
// and returns slide data for JSON-RPC clients.
func (s *StdioServer) handleRenderSlides(args map[string]any) (any, error) {
	pres := s.session.Presentation()
	if len(pres.Slides) == 0 {
		return nil, fmt.Errorf("no slides to render — add slides first")
	}

	// Build PPTX to temp file
	tmpFile, err := os.CreateTemp("", "otter-render-*.pptx")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := builder.New(pres).Save(tmpPath); err != nil {
		return nil, fmt.Errorf("build PPTX for render: %w", err)
	}

	// Render
	images, err := s.renderer.RenderPresentation(tmpPath, pres)
	if err != nil {
		return nil, fmt.Errorf("render slides: %w", err)
	}

	backend := "native"
	if s.renderer.IsAvailable() {
		backend = "libreoffice"
	}

	slides := make([]map[string]any, 0, len(images))
	for _, img := range images {
		entry := map[string]any{
			"slide_num":   img.SlideNum,
			"width":       img.Width,
			"height":      img.Height,
			"image_base64": img.Base64,
		}
		if img.FallbackDescription != "" {
			entry["description"] = img.FallbackDescription
		}
		if img.Path != "" {
			entry["image_path"] = img.Path
		}
		slides = append(slides, entry)
	}

	return map[string]any{
		"success":  true,
		"backend":  backend,
		"slide_count": len(slides),
		"slides":   slides,
		"hint":     "Images are base64-encoded PNG. If you have vision capability, analyze each slide for design issues (overlaps, alignment, whitespace, color contrast) and fix them using update_position, update_style, etc.",
	}, nil
}

// handleRenderSlidesMCP returns MCP-formatted content with image blocks
// so multimodal AI clients (Claude, GPT-4o) can visually inspect slides.
func (s *StdioServer) handleRenderSlidesMCP(call toolCallParams) (any, *rpcError) {
	if call.Arguments == nil {
		call.Arguments = map[string]any{}
	}

	pres := s.session.Presentation()
	if len(pres.Slides) == 0 {
		return map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": "No slides to render. Add slides first using add_slide."},
			},
			"isError": true,
		}, nil
	}

	// Build PPTX to temp file
	tmpFile, err := os.CreateTemp("", "otter-render-*.pptx")
	if err != nil {
		return map[string]any{
			"content": []map[string]string{{"type": "text", "text": "Failed to create temp file: " + err.Error()}},
			"isError": true,
		}, nil
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := builder.New(pres).Save(tmpPath); err != nil {
		return map[string]any{
			"content": []map[string]string{{"type": "text", "text": "Failed to build PPTX: " + err.Error()}},
			"isError": true,
		}, nil
	}

	// Render
	images, err := s.renderer.RenderPresentation(tmpPath, pres)
	if err != nil || len(images) == 0 {
		return map[string]any{
			"content": []map[string]string{{"type": "text", "text": "Rendering failed: " + fmt.Sprintf("%v", err)}},
			"isError": true,
		}, nil
	}

	// Build MCP content array: text intro + image blocks + text descriptions
	backend := "native"
	if s.renderer.IsAvailable() {
		backend = "libreoffice"
	}

	content := []map[string]any{
		{"type": "text", "text": fmt.Sprintf("Rendered %d slides using %s backend. Review each image for design quality (overlaps, alignment, whitespace, contrast). Fix issues with update_position/update_style/delete_element, then re-render to verify.", len(images), backend)},
	}

	for _, img := range images {
		// Add image content block for multimodal AI
		if img.Base64 != "" {
			content = append(content, map[string]any{
				"type":     "image",
				"data":     img.Base64,
				"mimeType": "image/png",
			})
		}
		// Always add text description as fallback
		desc := img.FallbackDescription
		if desc == "" {
			desc = fmt.Sprintf("(Slide %d — image rendered, no text description available)", img.SlideNum)
		}
		content = append(content, map[string]any{
			"type": "text",
			"text": fmt.Sprintf("Slide %d:", img.SlideNum),
		})
		content = append(content, map[string]any{
			"type": "text",
			"text": desc,
		})
	}

	return map[string]any{
		"content": content,
		"structuredContent": map[string]any{
			"backend":     backend,
			"slide_count": len(images),
		},
	}, nil
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
			"name": "render_slides", "description": "Render the current presentation to images for visual review. Returns slide images (base64 PNG) and structural descriptions. Use this to visually inspect your design — check for overlaps, alignment issues, whitespace balance, and color contrast. After reviewing, fix issues with update_position/update_style/delete_element, then re-render to verify. This enables a visual feedback loop.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		map[string]any{
			"name": "reset_session", "description": "Clear the current presentation and start a new session.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
	)
	return tools
}
