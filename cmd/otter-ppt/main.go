package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/agent"
	"github.com/otter-ppt/otter-ppt/internal/ai"
	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/integration"
	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/server"
)

const version = "0.4.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
	case "gen":
		cmdGen(os.Args[2:])
	case "mcp":
		if err := integration.NewStdioServer(os.Stdin, os.Stdout).RunMCP(); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}
	case "stdio":
		if err := integration.NewStdioServer(os.Stdin, os.Stdout).RunJSONRPC(); err != nil {
			log.Fatalf("STDIO server failed: %v", err)
		}
	case "version":
		fmt.Printf("otter-ppt v%s\n", version)
	case "help", "-h", "----help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`otter-ppt - AI-powered presentation generator

Usage:
  otter-ppt <command> [flags]

Commands:
  serve    Start the HTTP API server
  gen      Generate a PPTX from a topic prompt
  mcp      Start an MCP server over stdio (Claude Code / Cursor)
  stdio    Start the generic JSON-RPC server over stdio
  version  Show version info
  help     Show this help

Examples:
  otter-ppt serve --port 8080
  otter-ppt gen --topic "AI的未来" --slides 8 --style 科技感 --output my.pptx
  otter-ppt gen --topic "AI未来" --mode simple  # disable planning/vision

Environment:
  TEXT_MODEL_API_KEY / OPENAI_API_KEY       Text model API key (required for gen)
  TEXT_MODEL_BASE_URL / OPENAI_BASE_URL     OpenAI-compatible text endpoint
  TEXT_MODEL_NAME / OPENAI_MODEL            Text model name (default: gpt-4o)
  IMAGE_MODEL_API_KEY                       Optional image model API key
  IMAGE_MODEL_BASE_URL                      Optional OpenAI-compatible image endpoint
  IMAGE_MODEL_NAME                          Optional image model name`)
}

// ──────────────────────────────────────────────────────────────
// serve
// ──────────────────────────────────────────────────────────────

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.String("port", "8080", "HTTP server port")
	fs.Parse(args)

	srv := server.New(server.Config{
		Port:         *port,
		APIKey:       envOr("TEXT_MODEL_API_KEY", "OPENAI_API_KEY"),
		BaseURL:      envOr("TEXT_MODEL_BASE_URL", "OPENAI_BASE_URL"),
		Model:        envOr("TEXT_MODEL_NAME", "OPENAI_MODEL"),
		ImageAPIKey:  os.Getenv("IMAGE_MODEL_API_KEY"),
		ImageBaseURL: os.Getenv("IMAGE_MODEL_BASE_URL"),
		ImageModel:   os.Getenv("IMAGE_MODEL_NAME"),
	})

	log.Printf("Otter PPT server starting on :%s", *port)
	log.Printf("Endpoints: POST /api/v1/generate, POST /api/v1/execute, POST /api/v1/build, GET /api/v1/tools")
	log.Printf("Health check: GET /health")

	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────
// gen
// ──────────────────────────────────────────────────────────────

func cmdGen(args []string) {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	topic := fs.String("topic", "", "Presentation topic (required)")
	slides := fs.Int("slides", 8, "Number of slides")
	style := fs.String("style", "", "Visual style description")
	language := fs.String("language", "zh", "Content language (zh/en)")
	output := fs.String("output", "presentation.pptx", "Output .pptx file path")
	baseURL := fs.String("base-url", "", "Custom LLM API endpoint")
	modelName := fs.String("model", "", "Model name (default: gpt-4o)")
	maxSteps := fs.Int("max-steps", 60, "Maximum agent steps")
	mode := fs.String("mode", "workflow", "Generation mode: 'workflow' (plan+vision+refine) or 'simple' (direct agent loop)")
	fs.Parse(args)

	if *topic == "" {
		fmt.Fprintln(os.Stderr, "Error: --topic is required")
		fs.Usage()
		os.Exit(1)
	}

	apiKey := envOr("TEXT_MODEL_API_KEY", "OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: TEXT_MODEL_API_KEY or OPENAI_API_KEY is required")
		os.Exit(1)
	}

	if *baseURL == "" {
		*baseURL = envOr("TEXT_MODEL_BASE_URL", "OPENAI_BASE_URL")
	}
	if *modelName == "" {
		*modelName = firstNonEmpty(envOr("TEXT_MODEL_NAME", "OPENAI_MODEL"), "gpt-4o")
	}

	log.Printf("Generating: %s (%d slides, style=%s, lang=%s, mode=%s)",
		*topic, *slides, *style, *language, *mode)

	ag := agent.NewAgent(agent.AgentConfig{
		APIKey:         apiKey,
		BaseURL:        *baseURL,
		Model:          *modelName,
		Language:       *language,
		MaxSteps:       *maxSteps,
		ImageGenerator: makeImageGenerator(),
	})

	switch *mode {
	case "workflow":
		runGenWorkflow(ag, topic, slides, style, language, output)
	default:
		runGenSimple(ag, topic, slides, style, output)
	}
}

func runGenWorkflow(ag *agent.Agent, topic *string, slides *int, style, language, output *string) {
	log.Printf("Mode: workflow (plan → gather → build → review → refine → polish)")

	wfCfg := agent.DefaultWorkflowConfig()
	wfCfg.Topic = *topic
	wfCfg.SlideCount = *slides
	wfCfg.Style = *style
	wfCfg.Language = *language

	wfResult, err := agent.NewWorkflow(ag, wfCfg).Run(context.Background())
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	pres := ag.Session().Presentation()
	saveOutput(pres, *output)

	log.Printf("✅ Generated %d slides (%d agent steps, %d refine rounds) → %s",
		len(pres.Slides), wfResult.AgentResult.TotalSteps, wfResult.RefineRounds, *output)
	if wfResult.VisionReport != nil {
		log.Printf("   Vision score: %.0f/100", wfResult.VisionReport.OverallScore)
	}
	if wfResult.LayoutReport != nil {
		log.Printf("   Layout score: %.0f/100", wfResult.LayoutReport.Score)
	}
}

func runGenSimple(ag *agent.Agent, topic *string, slides *int, style, output *string) {
	log.Printf("Mode: simple (direct agent loop)")

	result, err := ag.Run(context.Background(), agent.SplitPrompt(*topic, *slides, *style))
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	pres := ag.Session().Presentation()
	saveOutput(pres, *output)

	log.Printf("✅ Generated %d slides in %d steps → %s",
		len(pres.Slides), result.TotalSteps, *output)
}

// ──────────────────────────────────────────────────────────────
// Shared helpers
// ──────────────────────────────────────────────────────────────

// saveOutput writes the presentation to the given path, creating parent dirs.
func saveOutput(pres *model.Presentation, output string) {
	if dir := filepath.Dir(output); dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}
	if err := builder.New(pres).Save(output); err != nil {
		log.Fatalf("Failed to save PPTX: %v", err)
	}
}

// makeImageGenerator returns an image generator from env vars, or nil if not configured.
func makeImageGenerator() ai.ImageGenerator {
	if key := os.Getenv("IMAGE_MODEL_API_KEY"); key != "" {
		return ai.NewImageGenerator(ai.ImageConfig{
			APIKey:  key,
			BaseURL: os.Getenv("IMAGE_MODEL_BASE_URL"),
			Model:   os.Getenv("IMAGE_MODEL_NAME"),
		})
	}
	return nil
}

func envOr(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
