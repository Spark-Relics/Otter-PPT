// Package ai handles LLM communication to generate structured slide JSON.
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/sashabaranov/go-openai"
)

// Generator calls an LLM to produce a structured Presentation.
type Generator struct {
	client    *openai.Client
	model     string
	maxTokens int
}

// NewGenerator creates an AI generator with the given API key.
// baseURL can be empty to use the default OpenAI endpoint.
func NewGenerator(apiKey, baseURL, model string) *Generator {
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &Generator{
		client:    openai.NewClientWithConfig(cfg),
		model:     model,
		maxTokens: 8000,
	}
}

// GenerateRequest is what the user asks for.
type GenerateRequest struct {
	Topic    string `json:"topic"`
	Language string `json:"language"` // "zh" or "en"
	Slides   int    `json:"slides"`   // number of slides, 0 = auto
	Style    string `json:"style"`    // e.g. "科技感", "商务简约"
}

// Generate calls the LLM and returns a complete Presentation.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*model.Presentation, error) {
	if req.Slides == 0 {
		req.Slides = 8
	}
	if req.Language == "" {
		req.Language = "zh"
	}

	prompt := buildPrompt(req)

	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: g.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature:      0.7,
		MaxTokens:        g.maxTokens,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}

	var pres model.Presentation
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &pres); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w", err)
	}

	// Ensure slide dimensions
	if pres.SlideWidth == 0 || pres.SlideHeight == 0 {
		pres.SlideWidth, pres.SlideHeight = model.DefaultSlideSize()
	}

	// Assign IDs if missing
	for i, slide := range pres.Slides {
		if slide.ID == "" {
			slide.ID = fmt.Sprintf("slide_%d", i+1)
		}
		for j, elem := range slide.Elements {
			if elem.ID == "" {
				elem.ID = fmt.Sprintf("s%d_e%d", i+1, j+1)
			}
		}
	}

	return &pres, nil
}

const systemPrompt = `You are a professional presentation designer.
You output ONLY valid JSON following the Presentation schema.
All text content must be in the language requested by the user.
Every text element must have a Rect with x, y, w, h as percentages (0-100) of the slide.
Every text element must have a Style with at minimum font_size, color, bold, and align.
Colors are hex strings like "#1A2B3C".
Be creative but keep layouts clean and readable.`

func buildPrompt(req GenerateRequest) string {
	lang := "Chinese"
	if req.Language == "en" {
		lang = "English"
	}

	style := req.Style
	if style == "" {
		style = "现代简约"
	}

	return fmt.Sprintf(`Create a %d-slide presentation about "%s".
Write all text in %s.
Visual style: %s.

Return JSON in this exact structure:
{
  "title": "演示文稿标题",
  "theme": {
    "primary_color": "#hex",
    "secondary_color": "#hex",
    "accent_color": "#hex",
    "background_color": "#hex",
    "text_color": "#hex",
    "title_font": "Microsoft YaHei",
    "body_font": "Microsoft YaHei"
  },
  "slides": [
    {
      "id": "slide_1",
      "layout": "title_content",
      "background": "",
      "notes": "演讲备注",
      "elements": [
        {
          "id": "s1_e1",
          "type": "title",
          "rect": {"x": 10, "y": 8, "w": 80, "h": 12},
          "text": "标题文字",
          "style": {
            "font_size": 36,
            "font_name": "Microsoft YaHei",
            "bold": true,
            "color": "#FFFFFF",
            "align": "left"
          }
        },
        {
          "id": "s1_e2",
          "type": "bullet",
          "rect": {"x": 10, "y": 25, "w": 80, "h": 60},
          "items": ["要点一", "要点二", "要点三"],
          "style": {
            "font_size": 18,
            "color": "#E0E0E0",
            "align": "left"
          }
        }
      ]
    }
  ]
}

Rules:
- Slide 1 should be layout "title" (cover page).
- Slide 2 should be layout "section" (table of contents / agenda).
- Last slide should be layout "title" (thank you page).
- Middle slides use "title_content" or "two_column" layouts.
- Each slide should have 3-6 elements.
- Keep text concise: titles <15 chars, bullets <30 chars each.
- Vary element positions for visual interest, but keep within bounds (x+w <= 95, y+h <= 92).
- Generate %d slides total.`,
		req.Slides, req.Topic, lang, style, req.Slides)
}
