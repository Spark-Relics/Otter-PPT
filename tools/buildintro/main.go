//go:build ignore

// buildintro compiles examples/project_intro.json into a .pptx using the
// project's own builder — dogfooding the full pipeline.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	data, err := os.ReadFile("examples/project_intro.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	var pres model.Presentation
	if err := json.Unmarshal(data, &pres); err != nil {
		fmt.Fprintln(os.Stderr, "parse json:", err)
		os.Exit(1)
	}
	out := "output/OtterPPT-项目介绍.pptx"
	if err := os.MkdirAll("output", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if err := builder.New(&pres).Save(out); err != nil {
		fmt.Fprintln(os.Stderr, "build:", err)
		os.Exit(1)
	}
	fmt.Println("saved:", out, "slides:", len(pres.Slides))
}
