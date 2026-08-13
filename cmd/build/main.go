package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: build <input.json> <output.pptx>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to read JSON: %v", err)
	}

	var pres model.Presentation
	if err := json.Unmarshal(data, &pres); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	b := builder.New(&pres)
	if err := b.Save(os.Args[2]); err != nil {
		log.Fatalf("Failed to save PPTX: %v", err)
	}

	log.Printf("✅ Generated %d slides → %s", len(pres.Slides), os.Args[2])
}
