// Package agent implements the AI agent loop that drives PPT creation
// via tool calling. The agent receives a user prompt, then iteratively
// calls pptoolkit tools until the presentation is complete.
//
// The package also provides a multi-phase Workflow (see workflow.go)
// that adds planning, vision review, and iterative refinement on top
// of the basic agent loop.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/otter-ppt/otter-ppt/internal/ai"
	"github.com/otter-ppt/otter-ppt/internal/layout"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
	"github.com/sashabaranov/go-openai"
)

// AgentConfig holds the settings for an agent run.
type AgentConfig struct {
	APIKey         string
	BaseURL        string
	Model          string
	Language       string // "zh" or "en"
	MaxSteps       int    // safety limit for tool-call rounds
	ImageGenerator interface {
		Generate(context.Context, string) (string, error)
	}
}

// Agent drives the PPT creation process.
type Agent struct {
	cfg     AgentConfig
	client  *openai.Client
	session *pptoolkit.Session
	// messages holds the conversation history so Refine() can continue.
	messages []openai.ChatCompletionMessage
	tools    []openai.Tool
}

// NewAgent creates a new agent.
func NewAgent(cfg AgentConfig) *Agent {
	// Trim whitespace from config strings (Windows env vars often have trailing spaces).
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Model = strings.TrimSpace(cfg.Model)

	ocfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		ocfg.BaseURL = cfg.BaseURL
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	if cfg.MaxSteps == 0 {
		cfg.MaxSteps = 60
	}
	if cfg.Language == "" {
		cfg.Language = "zh"
	}
	// Use compat transport for non-OpenAI providers (Gemini, etc.)
	transport := ai.NewCompatTransport()
	// Auto-detect Gemini free-tier: enforce 12s pacing to stay under 5 RPM.
	if strings.Contains(cfg.BaseURL, "generativelanguage.googleapis.com") {
		transport.MinInterval = 13 * time.Second
		log.Printf("[Agent] Gemini detected: request pacing set to %.0fs (free-tier 5 RPM)", transport.MinInterval.Seconds())
	}
	ocfg.HTTPClient = &http.Client{Transport: transport}
	return &Agent{
		cfg:    cfg,
		client: openai.NewClientWithConfig(ocfg),
		tools:  pptoolkit.ToolDefinitions(),
	}
}

// StepLog records one tool call for debugging/display.
type StepLog struct {
	ToolName string `json:"tool_name"`
	Args     any    `json:"args"`
	Result   string `json:"result"`
}

// AgentResult is the final output of an agent run.
type AgentResult struct {
	Steps      []StepLog `json:"steps"`
	TotalSteps int       `json:"total_steps"`
	Done       bool      `json:"done"`
}

// Session returns the session after Run completes.
func (a *Agent) Session() *pptoolkit.Session {
	return a.session
}

// Run executes the full agent loop: prompt → tool calls → done.
// After "done", it runs layout validation + auto-fix and feeds back
// issues if the layout quality score is below 80.
func (a *Agent) Run(ctx context.Context, userPrompt string) (*AgentResult, error) {
	a.session = pptoolkit.NewSession()
	a.tools = pptoolkit.ToolDefinitions()

	systemMsg := a.buildSystemPrompt()
	a.messages = []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemMsg},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	result := &AgentResult{Steps: []StepLog{}}
	return a.runLoop(ctx, result)
}

// Refine continues the conversation with visual or layout feedback.
// The feedback string is injected as a user message, and the agent
// gets up to MaxSteps/3 additional rounds to fix issues.
func (a *Agent) Refine(ctx context.Context, feedback string) (*AgentResult, error) {
	if a.session == nil {
		return nil, fmt.Errorf("no active session — call Run first")
	}

	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf("Visual review completed. Here is the feedback:\n\n%s\n\nPlease fix the issues above by modifying slides (use update_position, update_style, update_text, delete_element, add_shape, etc.), then call done when finished.", feedback),
	})

	result := &AgentResult{Steps: []StepLog{}}
	// Use fewer steps for refinement
	prevMax := a.cfg.MaxSteps
	a.cfg.MaxSteps = prevMax / 3
	if a.cfg.MaxSteps < 10 {
		a.cfg.MaxSteps = 10
	}
	defer func() { a.cfg.MaxSteps = prevMax }()

	return a.runLoop(ctx, result)
}

// runLoop is the core iterative tool-calling engine.
func (a *Agent) runLoop(ctx context.Context, result *AgentResult) (*AgentResult, error) {
	for step := 0; step < a.cfg.MaxSteps; step++ {
		resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       a.cfg.Model,
			Messages:    a.messages,
			Tools:       a.tools,
			Temperature: 0.7,
			MaxTokens:   4096,
		})
		if err != nil {
			return result, fmt.Errorf("LLM call failed at step %d: %w", step, err)
		}

		if len(resp.Choices) == 0 {
			return result, fmt.Errorf("LLM returned no choices at step %d", step)
		}

		choice := resp.Choices[0]
		a.messages = append(a.messages, choice.Message)

		// If no tool calls, check if done
		if len(choice.Message.ToolCalls) == 0 {
			log.Printf("[Agent] Model responded without tool calls, assuming done")
			result.Done = true
			break
		}

		// Process each tool call
		for _, tc := range choice.Message.ToolCalls {
			toolName := tc.Function.Name
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]any{"_raw": tc.Function.Arguments}
			}

			log.Printf("[Agent] Tool: %s args: %v", toolName, args)

			// Resolve generated images
			if toolName == "add_image" || toolName == "generate_image" {
				imagePath, _ := args["image_path"].(string)
				imagePrompt, _ := args["image_prompt"].(string)
				if imagePath == "" && imagePrompt != "" {
					if a.cfg.ImageGenerator == nil {
						result.Steps = append(result.Steps, StepLog{ToolName: toolName, Args: args, Result: "error: no image model configured"})
						a.messages = append(a.messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    "Error: image generation requires an image model. Use image_path with a local file instead.",
							ToolCallID: tc.ID,
						})
						continue
					}
					generatedPath, err := a.cfg.ImageGenerator.Generate(ctx, imagePrompt)
					if err != nil {
						errMsg := fmt.Sprintf("Image generation failed: %v", err)
						a.messages = append(a.messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    "Error: " + errMsg,
							ToolCallID: tc.ID,
						})
						result.Steps = append(result.Steps, StepLog{ToolName: toolName, Args: args, Result: errMsg})
						continue
					}
					args["image_path"] = generatedPath
				}
			}

			// Execute the tool
			toolResult := a.session.ExecuteTool(toolName, args)
			resultStr := toolResult.String()

			log.Printf("[Agent] Result: %s", resultStr)

			result.Steps = append(result.Steps, StepLog{
				ToolName: toolName,
				Args:     args,
				Result:   resultStr,
			})

			// Add tool result to conversation
			a.messages = append(a.messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    resultStr,
				ToolCallID: tc.ID,
			})

			// Check for done signal
			if toolName == "done" {
				result.Done = true
				result.TotalSteps = len(result.Steps)

				// Post-build: auto-fix + validate
				pres := a.session.Presentation()
				fixes := layout.AutoFixPresentation(pres)
				report := layout.ValidatePresentation(pres)

				log.Printf("[Agent] Post-build: auto-fixed %d issues, layout score: %.0f/100", fixes, report.Score)

				if report.Score < 80 {
					log.Printf("[Agent] Layout score below 80, feeding report back")
					feedback := fmt.Sprintf("Layout auto-fix applied %d corrections. Current score: %.0f/100.\n\n%s\nFix the remaining issues with update_position or apply_smart_layout, then call done.", fixes, report.Score, layout.FormatReport(report))
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleUser,
						Content: feedback,
					})
					result.Done = false
					break // let the loop continue
				}

				return result, nil
			}
		}
	}

	result.TotalSteps = len(result.Steps)
	if !result.Done {
		log.Printf("[Agent] Reached max steps (%d) without explicit done", a.cfg.MaxSteps)
	}

	return result, nil
}

func (a *Agent) buildSystemPrompt() string {
	langName := "Chinese"
	if a.cfg.Language == "en" {
		langName = "English"
	}

	hasImageModel := a.cfg.ImageGenerator != nil

	imageInstruction := ""
	if hasImageModel {
		imageInstruction = `
## Image Strategy
- You have an AI image generation model available.
- Use image_prompt in add_image to generate professional images inline.
- For cover slides: generate abstract/professional backgrounds or hero imagery
- For section dividers: generate mood-setting backgrounds
- For content slides: generate relevant illustrations or diagrams when they add value
- Write detailed English prompts for image generation (style, mood, composition, colors)
- Example: "Modern abstract gradient background, deep blue to purple, geometric shapes, professional, 16:9"
- Alternate between image slides and text-focused slides for visual rhythm`
	} else {
		imageInstruction = `
## Image Strategy  
- No AI image model is configured.
- Create visual interest using shapes (cards, bands, dividers, decorative elements).
- Use gradients and colors as "visual imagery" instead of photos.
- Focus on typography-driven design with geometric compositions.`
	}

	return fmt.Sprintf(`You are Otter PPT, an expert presentation designer AI with the skills of a senior art director.

You have tools to build a complete, beautiful, editable PowerPoint presentation.
Use them step by step, exactly like a professional designer crafting slides in PowerPoint.

## Workflow
1. Call set_theme AND set_slide_size together in one response. set_theme MUST use style + palette preset keys (see Design System below); do not invent ad-hoc colors.
2. For each slide: add_slide, then set_bg_gradient (derive stops from the palette), then add ALL elements. Use add_card for card layouts instead of stacking add_shape + add_text manually.
3. Use apply_smart_layout if helpful, or position elements manually with precise coordinates.
4. Set notes (and transitions if desired).
5. Call done. A post-build auto-fix handles minor layout issues.

## Design System (binding)
Pick ONE style + ONE palette from the set_theme tool catalog and keep it for the whole deck. Common pairings:
- Tech / AI / dev tools / dark launch → style=dark_tech + palette=tech_neon
- Corporate / consulting / data → style=swiss_minimal + palette=cool_corporate
- Storytelling / report / thought leadership → style=editorial + palette=editorial_classic
- Modern product / SaaS / futuristic → style=glassmorphism + palette=tech_neon (or gradient_modern)
- Education / consumer / health → style=soft_rounded + palette=warm_earth
The design lock returned by set_theme is the rulebook for the entire deck: obey its shape language, corner radius, decoration, and depth rules on EVERY slide. Never mix styles.

## SPEED RULES
- Call MULTIPLE tools per response (e.g., set_theme + set_slide_size together, or add_title + add_text + add_shape together in one response).
- NEVER delete a slide to rebuild it. Fix issues with update_* tools instead.
- Do NOT call validate_layout manually — the system auto-validates after done.

## CONTENT RICHNESS (CRITICAL — DO NOT SKIP)
Every slide MUST be visually rich. A slide with only a title + one line of text is UNACCEPTABLE.

**Cover slide** (minimum 5 elements): gradient/image background + large title + subtitle + accent line/shape + decorative shape or logo text.
**Content slide** (minimum 6 elements): background + title + subtitle/intro text + 3-4 cards built with add_card (each card = panel + accent bar + title + description in one call) + optional decorative shapes.
**Stats slide** (minimum 8 elements): background + title + 3 big numbers (font 48-60pt) + 3 labels + optional shapes.
**Timeline slide** (minimum 10 elements): background + title + arrow shape + 4-5 milestone shapes + dates + descriptions.
**Section divider** (minimum 4 elements): distinctive background + large centered text + accent shapes.

Do NOT call done until every slide meets the minimum element count above. A sparse slide is worse than a slightly imperfect but rich slide.
%s

## Smart Layout Templates
- **title**: Bold title + subtitle (cover slide)
- **title_content**: Title on top, content area below
- **two_column**: Title + two content columns side by side
- **image_left** / **image_right**: Image on one half, text on other
- **image_full**: Full-bleed image with title overlay
- **section**: Section divider with large centered text
- **bullets**: Title + bulleted list
- **three_cards** / **four_cards**: Title + content cards
- **timeline**: Horizontal timeline with milestones
- **comparison**: Two-column comparison layout
- **stats**: Title + 3 big-number statistics
- **agenda**: Title + numbered list (for table of contents)
- **chart**: Title + chart + commentary sidebar

## Design Excellence Rules
1. **Visual Hierarchy**: Title > key points > supporting details (use size, weight, color)
2. **Whitespace**: Don't crowd elements. 8%%+ margins minimum.
3. **Color Discipline**: 2-3 colors + neutrals. Consistent across all slides.
4. **Layout Variety**: Never use the same layout 2 slides in a row (except intentional pairs).
5. **Consistent Grid**: Align elements to the same vertical/horizontal positions across slides.
6. **Typographic Contrast**: Mix font sizes (32-40pt titles, 24pt headings, 16-18pt body).
7. **Decorative Elements**: Use shapes (rounded rectangles) as card backgrounds, accent bars, divider lines.
8. **Data Visualization**: Use charts for numbers, tables for comparisons, timelines for progression.
9. **Transitions**: Use fade between content slides, push for section changes, morph for related slides.
10. **Speaker Notes**: Add concise notes for each slide — this makes the PPT useful for presenters.

## Composition Patterns (like a real designer)
- Cover: Full gradient/image bg + bold title + subtitle + accent line
- Agenda: Numbered list with accent-colored numbers on the left
- Content + Cards: use add_card for 3-4 themed cards (panel + left accent bar + bold title + muted description); lay them in a row with 2-3%% gaps, equal heights
- Stats: Three big numbers (48-60pt) in accent color + small labels below
- Quote: Large italic centered text with a subtle accent bar above
- Timeline: Horizontal arrow shape with 4-5 milestone circles (ellipse shapes) with dates + labels
- Comparison: Two contrasting columns (different background colors, "vs" in center)
- Thank You: Large centered text on branded background

## Coordinate System
Percentages (0-100): x = left, y = top, w = width, h = height.
Safe area: x ≥ 5, x+w ≤ 95, y ≥ 5, y+h ≤ 92.

## Language
Write ALL content in %s.

## Important Rules
- Call MULTIPLE tools per response to speed up building. Example: call set_theme + set_slide_size together, or add_title + add_text + add_shape + add_shape together.
- After add_slide, you get slide_id. After add_*, you get element_id.
- Keep text concise: titles <20 chars, bullets <40 chars.
- Be creative and professional — make it look like a $5000 presentation.
- Use shapes generously as visual containers and decorative elements.
- NEVER delete and rebuild an entire slide. Fix issues with update_* tools instead.
- Each slide MUST be visually rich — see CONTENT RICHNESS rules above. No slide should have fewer than 5 elements.
- Always call done when finished.

Start building the presentation now!`, imageInstruction, langName)
}

// SplitPrompt creates a structured prompt from user input.
func SplitPrompt(topic string, slideCount int, style string) string {
	parts := []string{fmt.Sprintf("Create a presentation about: %s", topic)}
	if slideCount > 0 {
		parts = append(parts, fmt.Sprintf("Make it %d slides.", slideCount))
	}
	if style != "" {
		parts = append(parts, fmt.Sprintf("Visual style: %s.", style))
	}
	return strings.Join(parts, " ")
}
