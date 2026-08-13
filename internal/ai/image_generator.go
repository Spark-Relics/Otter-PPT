package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxGeneratedImageBytes = 20 << 20

// ImageConfig configures an OpenAI-compatible image generation endpoint.
type ImageConfig struct {
	APIKey string
	BaseURL string
	Model string
}

// ImageGenerator turns an image prompt into a local image path.
type ImageGenerator interface {
	Generate(context.Context, string) (string, error)
}

// OpenAIImageGenerator calls an OpenAI-compatible /images/generations endpoint.
type OpenAIImageGenerator struct {
	cfg ImageConfig
	client *http.Client
}

func NewImageGenerator(cfg ImageConfig) *OpenAIImageGenerator {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-image-1"
	}
	return &OpenAIImageGenerator{cfg: cfg, client: &http.Client{Timeout: 120 * time.Second}}
}

func (g *OpenAIImageGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	if g.cfg.APIKey == "" {
		return "", fmt.Errorf("image model API key is not configured")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("image prompt is required")
	}

	body, err := json.Marshal(map[string]any{
		"model": g.cfg.Model, "prompt": prompt, "size": "1536x1024", "response_format": "b64_json",
	})
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(g.cfg.BaseURL, "/") + "/images/generations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("image model call failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxGeneratedImageBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > maxGeneratedImageBytes {
		return "", fmt.Errorf("image model response exceeds %d bytes", maxGeneratedImageBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("image model returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var result struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("decode image model response: %w", err)
	}
	if len(result.Data) == 0 || result.Data[0].B64JSON == "" {
		return "", fmt.Errorf("image model returned no base64 image")
	}
	imageBytes, err := base64.StdEncoding.DecodeString(result.Data[0].B64JSON)
	if err != nil {
		return "", fmt.Errorf("decode generated image: %w", err)
	}
	if len(imageBytes) > maxGeneratedImageBytes {
		return "", fmt.Errorf("generated image exceeds %d bytes", maxGeneratedImageBytes)
	}
	file, err := os.CreateTemp("", "otter-ppt-image-*.png")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if _, err := file.Write(imageBytes); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}
