// planner.go implements the pre-build planning phase. Before the agent starts
// calling design tools, the LLM creates a structured plan covering:
//   - Slide outline (content structure, key points per slide)
//   - Design strategy (theme, color palette, typographic direction)
//   - Asset requirements (which slides need AI-generated images, charts, etc.)
//
// The plan is injected into the build-phase prompt so the agent has a clear
// roadmap rather than improvising from scratch.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// SlidePlan describes the design intention for one slide.
type SlidePlan struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Layout      string   `json:"layout"`
	KeyPoints   []string `json:"key_points"`
	VisualNeeds string   `json:"visual_needs"` // e.g. "AI-generated hero image of...", "3-card layout with icons"
	ImagePrompt string   `json:"image_prompt"` // ready-to-use prompt if image generation is needed
	Notes       string   `json:"notes"`        // speaker notes outline
}

// DesignPlan is the complete pre-build strategy.
type DesignPlan struct {
	Title          string      `json:"title"`
	StyleDirection string      `json:"style_direction"` // overall visual style description
	ColorPalette   string      `json:"color_palette"`
	FontStrategy   string      `json:"font_strategy"`
	TargetAudience string      `json:"target_audience"`
	Slides         []SlidePlan `json:"slides"`
	ImageNeeds     []ImageNeed `json:"image_needs"`
}

// ImageNeed describes an image that should be generated.
type ImageNeed struct {
	SlideNum int    `json:"slide_num"`
	Role     string `json:"role"`     // "background", "hero", "illustration", "icon"
	Prompt   string `json:"prompt"`   // generation prompt
}

// Planner creates a structured design plan before building.
type Planner struct {
	client *openai.Client
	model  string
}

// NewPlanner creates a planner.
func NewPlanner(client *openai.Client, model string) *Planner {
	return &Planner{client: client, model: model}
}

// Plan asks the LLM to create a detailed presentation plan.
func (p *Planner) Plan(ctx context.Context, topic string, slideCount int, style string, language string) (*DesignPlan, error) {
	langName := "English"
	if language == "zh" || language == "" {
		langName = "Chinese"
	}

	if slideCount <= 0 {
		slideCount = 8
	}

	systemPrompt := fmt.Sprintf(`You are a presentation strategist and art director.
Create a detailed plan for a %d-slide presentation about: "%s".
Style: %s. All content in %s.

Output JSON with this exact structure:
{
  "title": "Presentation title",
  "style_direction": "Brief description of overall visual direction (e.g. 'Modern minimalist with bold typography, generous whitespace, and tech-inspired gradient accents')",
  "color_palette": "Specific hex colors and their usage (e.g. '#1A73E8 primary, #F1F3F4 background, #EA4335 accent, #202124 text')",
  "font_strategy": "Font choices and hierarchy (e.g. 'Montserrat Bold for titles, Inter Regular for body')",
  "target_audience": "Who is this presentation for",
  "slides": [
    {
      "number": 1,
      "title": "Slide title (concise, <20 chars)",
      "layout": "title|title_content|two_column|image_left|image_right|image_full|section|bullets|quote|three_cards|four_cards|timeline|comparison|stats|chart|agenda|thank_you|contact",
      "key_points": ["Main point 1", "Main point 2", "Main point 3"],
      "visual_needs": "What visual elements this slide needs (shapes, icons, decorative elements)",
      "image_prompt": "Detailed prompt for AI image generation if needed, empty string if no image needed",
      "notes": "Speaker notes outline for this slide"
    }
  ],
  "image_needs": [
    {"slide_num": 1, "role": "background", "prompt": "Abstract gradient mesh background in blue and purple, professional, high resolution"}
  ]
}

Design Rules:
- Slide 1 is always "title" layout (cover page with hero image or gradient bg)
- Slide 2 is "agenda" layout (table of contents)
- For data-heavy slides, use "chart" or "stats" layouts
- For concept slides, use "three_cards", "four_cards", or "timeline"
- For section breaks, use "section" layout
- Last slide is "thank_you" or "contact"
- Only suggest image_prompt for slides that truly benefit from visuals (cover, section dividers, hero slides)
- Keep image prompts detailed and professional (photographic, illustration style, mood, colors)
- Vary layouts across slides — no more than 2 consecutive same-layout slides
- Ensure content flows logically from introduction → problem → solution → details → conclusion`, slideCount, topic, style, langName)

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a presentation strategist and art director. Respond ONLY with valid JSON."},
			{Role: openai.ChatMessageRoleUser, Content: systemPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   8000,
	})
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("planner returned no choices")
	}

	var plan DesignPlan
	raw := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		start := strings.Index(raw, "{")
		end := strings.LastIndex(raw, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(raw[start:end+1]), &plan); err2 != nil {
				return nil, fmt.Errorf("parse plan: %w", err2)
			}
		} else {
			return nil, fmt.Errorf("parse plan: %w", err)
		}
	}

	return &plan, nil
}

// FormatPlan converts a DesignPlan to a rich instruction string for the build phase.
func FormatPlan(plan *DesignPlan) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 DESIGN PLAN\n\n"))
	sb.WriteString(fmt.Sprintf("Title: %s\n", plan.Title))
	sb.WriteString(fmt.Sprintf("Style Direction: %s\n", plan.StyleDirection))
	sb.WriteString(fmt.Sprintf("Color Palette: %s\n", plan.ColorPalette))
	sb.WriteString(fmt.Sprintf("Font Strategy: %s\n\n", plan.FontStrategy))

	sb.WriteString("SLIDE BREAKDOWN:\n")
	for _, sp := range plan.Slides {
		sb.WriteString(fmt.Sprintf("\nSlide %d [%s]: %s\n", sp.Number, sp.Layout, sp.Title))
		for _, kp := range sp.KeyPoints {
			sb.WriteString(fmt.Sprintf("  • %s\n", kp))
		}
		if sp.VisualNeeds != "" {
			sb.WriteString(fmt.Sprintf("  Visual: %s\n", sp.VisualNeeds))
		}
		if sp.ImagePrompt != "" {
			sb.WriteString(fmt.Sprintf("  Image Prompt: %s\n", sp.ImagePrompt))
		}
	}

	if len(plan.ImageNeeds) > 0 {
		sb.WriteString("\nIMAGE ASSETS NEEDED:\n")
		for _, in := range plan.ImageNeeds {
			sb.WriteString(fmt.Sprintf("  Slide %d [%s]: %s\n", in.SlideNum, in.Role, in.Prompt))
		}
	}

	sb.WriteString("\nFollow this plan closely. Use the specified layouts, colors, and content for each slide.")
	sb.WriteString("For slides with image_prompt, use that exact prompt in add_image.")
	sb.WriteString(" Apply professional transitions between slides. Set speaker notes from the plan.\n")

	return sb.String()
}
