// vision.go provides AI visual evaluation of rendered slides using a
// multimodal model. It sends slide images or structural descriptions to a
// vision-capable LLM and receives structured design feedback.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/renderer"
	"github.com/sashabaranov/go-openai"
)

// VisionFeedback is the structured design critique for one slide.
type VisionFeedback struct {
	SlideNum     int      `json:"slide_num"`
	DesignScore  float64  `json:"design_score"`  // 0-100
	ContentScore float64  `json:"content_score"` // 0-100
	Issues       []string `json:"issues"`
	Suggestions  []string `json:"suggestions"`
}

// VisionReport aggregates feedback across all slides.
type VisionReport struct {
	OverallScore float64         `json:"overall_score"`
	SlideFeedback []VisionFeedback `json:"slide_feedback"`
	OverallIssues []string        `json:"overall_issues"`
	OverallSuggestions []string   `json:"overall_suggestions"`
}

// VisionEvaluator uses a multimodal model to assess slide design.
type VisionEvaluator struct {
	client *openai.Client
	model  string
}

// NewVisionEvaluator creates a vision evaluator that reuses the agent's LLM client.
func NewVisionEvaluator(client *openai.Client, model string) *VisionEvaluator {
	if model == "" {
		model = "gpt-4o"
	}
	return &VisionEvaluator{client: client, model: model}
}

// EvaluateSlides sends rendered slide images to the vision model and returns feedback.
func (ve *VisionEvaluator) EvaluateSlides(ctx context.Context, slides []renderer.SlideImage, topic, style string) (*VisionReport, error) {
	if len(slides) == 0 {
		return &VisionReport{}, nil
	}

	langNote := "Chinese"
	// Detect from style/topic is complex; default to Chinese based on project config

	systemPrompt := fmt.Sprintf(`You are a senior presentation design critic and art director.
You evaluate slides for visual design quality, content clarity, and professional polish.
Analyze each slide image carefully. Consider:
- Visual hierarchy and information flow
- Whitespace, balance, and alignment
- Color harmony and contrast
- Text readability and font choices
- Element positioning and potential overlaps
- Content density (too crowded vs. too sparse)
- Overall professional appearance vs. amateur look

Rate each slide on Design (visual quality, 0-100) and Content (clarity and structure, 0-100).
List specific, actionable issues and suggestions.

Respond in %s, as JSON:
{
  "overall_score": number,
  "slide_feedback": [
    {"slide_num": 1, "design_score": 85, "content_score": 90, "issues": ["..."], "suggestions": ["..."]}
  ],
  "overall_issues": ["..."],
  "overall_suggestions": ["..."]
}`, langNote)

	// Build message content: add images
	content := []openai.ChatMessagePart{
		{Type: openai.ChatMessagePartTypeText, Text: fmt.Sprintf("Topic: %s\nStyle: %s\nEvaluate these %d slides:", topic, style, len(slides))},
	}

	for _, slide := range slides {
		if slide.Base64 != "" {
			content = append(content, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL:    fmt.Sprintf("data:image/png;base64,%s", slide.Base64),
					Detail: openai.ImageURLDetailLow,
				},
			})
		} else if slide.FallbackDescription != "" {
			content = append(content, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: slide.FallbackDescription,
			})
		}
	}

	resp, err := ve.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: ve.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, MultiContent: content},
		},
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("vision evaluation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("vision model returned no choices")
	}

	var report VisionReport
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &report); err != nil {
		// Try to extract JSON from text
		raw := resp.Choices[0].Message.Content
		start := strings.Index(raw, "{")
		end := strings.LastIndex(raw, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(raw[start:end+1]), &report); err2 != nil {
				return nil, fmt.Errorf("parse vision feedback: %w", err2)
			}
		} else {
			return nil, fmt.Errorf("parse vision feedback: %w", err)
		}
	}

	// Ensure slide numbers are set
	for i := range report.SlideFeedback {
		if report.SlideFeedback[i].SlideNum == 0 {
			report.SlideFeedback[i].SlideNum = i + 1
		}
	}

	return &report, nil
}

// FormatVisionReport converts a VisionReport to a feedback string for the agent loop.
func FormatVisionReport(report *VisionReport) string {
	if report == nil {
		return "No vision feedback available."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔬 Visual Design Review — Overall Score: %.0f/100\n\n", report.OverallScore))

	for _, sf := range report.SlideFeedback {
		sb.WriteString(fmt.Sprintf("Slide %d: Design %.0f/100, Content %.0f/100\n", sf.SlideNum, sf.DesignScore, sf.ContentScore))
		for _, issue := range sf.Issues {
			sb.WriteString(fmt.Sprintf("  ⚠ %s\n", issue))
		}
		for _, sugg := range sf.Suggestions {
			sb.WriteString(fmt.Sprintf("  💡 %s\n", sugg))
		}
		sb.WriteString("\n")
	}

	if len(report.OverallIssues) > 0 {
		sb.WriteString("Overall Issues:\n")
		for _, issue := range report.OverallIssues {
			sb.WriteString(fmt.Sprintf("  ⚠ %s\n", issue))
		}
	}
	if len(report.OverallSuggestions) > 0 {
		sb.WriteString("\nOverall Suggestions:\n")
		for _, sugg := range report.OverallSuggestions {
			sb.WriteString(fmt.Sprintf("  💡 %s\n", sugg))
		}
	}

	return sb.String()
}

// ShouldRefine returns true if the vision score is below threshold.
func ShouldRefine(report *VisionReport, threshold float64) bool {
	if report == nil {
		return false
	}
	return report.OverallScore < threshold
}
