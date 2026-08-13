package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/agent"
	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/server"
)

const version = "0.1.0"

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
	case "version":
		fmt.Printf("otter-ppt v%s\n", version)
	case "help", "-h", "--help":
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
  version  Show version info
  help     Show this help

Examples:
  otter-ppt serve --port 8080
  otter-ppt gen --topic "AI的未来" --slides 8 --style 科技感 --output my.pptx

Environment:
  OPENAI_API_KEY     OpenAI-compatible API key (required for gen)
  OPENAI_BASE_URL    Custom API endpoint (optional)
  OPENAI_MODEL       Model name (default: gpt-4o)`)
}

// ────────── serve command ──────────

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.String("port", "8080", "HTTP server port")
	fs.Parse(args)

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	srv := server.New(server.Config{
		Port:    *port,
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})

	log.Printf("Otter PPT server starting on :%s", *port)
	log.Printf("Endpoints: POST /api/v1/generate, POST /api/v1/build, GET /api/v1/tools")
	log.Printf("Health check: GET /health")

	if err := srv.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// ────────── gen command ──────────

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
	fs.Parse(args)

	if *topic == "" {
		fmt.Fprintln(os.Stderr, "Error: --topic is required")
		fs.Usage()
		os.Exit(1)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OPENAI_API_KEY environment variable is required")
		os.Exit(1)
	}

	if *baseURL == "" {
		*baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if *modelName == "" {
		*modelName = os.Getenv("OPENAI_MODEL")
		if *modelName == "" {
			*modelName = "gpt-4o"
		}
	}

	log.Printf("Generating presentation: %s (%d slides, style=%s, lang=%s)",
		*topic, *slides, *style, *language)

	ag := agent.NewAgent(agent.AgentConfig{
		APIKey:   apiKey,
		BaseURL:  *baseURL,
		Model:    *modelName,
		Language: *language,
		MaxSteps: *maxSteps,
	})

	prompt := agent.SplitPrompt(*topic, *slides, *style)
	result, err := ag.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	pres := ag.Session().Presentation()

	// Ensure output directory exists
	outDir := filepath.Dir(*output)
	if outDir != "" && outDir != "." {
		os.MkdirAll(outDir, 0755)
	}

	b := builder.New(pres)
	if err := b.Save(*output); err != nil {
		log.Fatalf("Failed to save PPTX: %v", err)
	}

	log.Printf("✅ Generated %d slides in %d steps → %s",
		len(pres.Slides), result.TotalSteps, *output)
}
