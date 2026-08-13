package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/otter-ppt/otter-ppt/internal/ai"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
	"github.com/sashabaranov/go-openai"
)

func main() {
	apiKey := os.Getenv("TEXT_MODEL_API_KEY")
	if apiKey == "" {
		apiKey = "REMOVED-LEAKED-KEY"
	}
	model := os.Getenv("TEXT_MODEL_NAME")
	if model == "" {
		model = "agnes-2.5-pro"
	}
	baseURL := os.Getenv("TEXT_MODEL_BASE_URL")
	if baseURL == "" {
		baseURL = "https://apihub.agnes-ai.com/v1"
	}

	tools := pptoolkit.ToolDefinitions()
	toolJSON, _ := json.Marshal(tools)
	fmt.Printf("Tool count: %d, tools JSON size: %.1f KB\n", len(tools), float64(len(toolJSON))/1024)

	systemPrompt := `You are a presentation designer. Call set_theme first, then add slides. Call done when finished.`
	userPrompt := `Create a 2-slide presentation about AI.`

	// Build the request body
	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Tools:       tools,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	reqBody, _ := json.Marshal(req)
	fmt.Printf("Full request body size: %.1f KB\n", float64(len(reqBody))/1024)

	// Send with CompatTransport
	transport := ai.NewCompatTransport()
	client := &http.Client{Transport: transport, Timeout: 180 * time.Second}

	httpReq, _ := http.NewRequest("POST", baseURL+"/chat/completions", nil)
	// Re-marshal into a simple map to set body
	var rawReq map[string]json.RawMessage
	json.Unmarshal(reqBody, &rawReq)
	finalBody, _ := json.Marshal(rawReq)

	httpReq, _ = http.NewRequest("POST", baseURL+"/chat/completions", bytesReader(finalBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	fmt.Printf("Sending request to %s ...\n", baseURL+"/chat/completions")
	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("ERROR after %.1fs: %v\n", elapsed.Seconds(), err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response after %.1fs: Status=%d, Body=[%.500s]\n", elapsed.Seconds(), resp.StatusCode, string(body))
}

func bytesReader(data []byte) io.Reader {
	return &byteReader{data: data}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
