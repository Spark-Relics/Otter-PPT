// Package agent implements the AI agent loop that drives PPT creation
// via tool calling. The agent receives a user prompt, then iteratively
// calls pptoolkit tools until the presentation is complete.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
	"github.com/sashabaranov/go-openai"
)

// AgentConfig holds the settings for an agent run.
type AgentConfig struct {
	APIKey   string
	BaseURL  string
	Model    string
	Language string // "zh" or "en"
	MaxSteps int    // safety limit for tool-call rounds
}

// Agent drives the PPT creation process.
type Agent struct {
	cfg     AgentConfig
	client  *openai.Client
	session *pptoolkit.Session
}

// NewAgent creates a new agent.
func NewAgent(cfg AgentConfig) *Agent {
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
	return &Agent{
		cfg:    cfg,
		client: openai.NewClientWithConfig(ocfg),
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
	Steps       []StepLog `json:"steps"`
	TotalSteps  int       `json:"total_steps"`
	Done        bool      `json:"done"`
}

// Run executes the full agent loop: prompt → tool calls → done.
func (a *Agent) Run(ctx context.Context, userPrompt string) (*AgentResult, error) {
	a.session = pptoolkit.NewSession()
	tools := pptoolkit.ToolDefinitions()

	systemMsg := a.buildSystemPrompt()
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemMsg},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	result := &AgentResult{Steps: []StepLog{}}

	for step := 0; step < a.cfg.MaxSteps; step++ {
		resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    a.cfg.Model,
			Messages: messages,
			Tools:    tools,
			Temperature: 0.7,
			MaxTokens: 4096,
		})
		if err != nil {
			return result, fmt.Errorf("LLM call failed at step %d: %w", step, err)
		}

		if len(resp.Choices) == 0 {
			return result, fmt.Errorf("LLM returned no choices at step %d", step)
		}

		choice := resp.Choices[0]
		// Append assistant message (with tool calls) to history
		messages = append(messages, choice.Message)

		// If no tool calls, check if the model is done
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
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    resultStr,
				ToolCallID: tc.ID,
			})

			// Check for done signal
			if toolName == "done" {
				result.Done = true
				result.TotalSteps = len(result.Steps)
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

// Session returns the session after Run completes.
func (a *Agent) Session() *pptoolkit.Session {
	return a.session
}

func (a *Agent) buildSystemPrompt() string {
	langName := "Chinese"
	if a.cfg.Language == "en" {
		langName = "English"
	}

	return fmt.Sprintf(`You are Otter PPT, an expert presentation designer AI.

You have a set of tools to build a complete, beautiful, editable PowerPoint presentation.
Use these tools step by step, exactly like a human designer would in PowerPoint.

## Workflow
1. Call set_theme first to define the color scheme and fonts.
2. Call add_slide for each page, then add elements to each slide.
3. For each slide:
   - Set background (set_bg_color, set_bg_gradient, or set_bg_image)
   - Add title, text, bullet lists
   - Add shapes, images, tables, charts as needed
   - Add transitions and animations
   - Set speaker notes
4. Call done when the presentation is complete.

## Design Principles
- Use cohesive color schemes (2-3 colors max + neutrals)
- Vary layouts: not every slide should look the same
- Cover slide (slide 1): bold title, subtitle, attractive background
- Agenda slide: list of topics
- Content slides: title + 3-5 bullet points or visual elements
- Use shapes for visual interest (cards, accent bars, icons)
- Use charts for data visualization
- Last slide: thank you / Q&A

## Coordinate System
All positions use percentages (0-100) relative to slide size:
- x: left edge (0 = far left)
- y: top edge (0 = very top)
- w: width percentage
- h: height percentage
- Keep content within margins: x+w <= 95, y+h <= 92

## Language
Write ALL text content in %s.

## Important
- Call ONE tool at a time to build up the presentation gradually.
- After add_slide, you get the slide_id back. Use it for subsequent elements.
- After add_*, you get element_id back. Use it for updates.
- Make text concise: titles <20 chars, bullets <40 chars each.
- Be creative and make it visually appealing.

Start building the presentation now!`, langName)
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
