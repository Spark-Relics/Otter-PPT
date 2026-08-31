//go:build ignore

// buildjson compiles an arbitrary Presentation JSON into a .pptx using the
// project's own builder. Usage: go run tools/buildjson/main.go <input.json> [output.pptx]
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: buildjson <input.json> [output.pptx]")
		os.Exit(1)
	}
	in := os.Args[1]
	out := "output/presentation.pptx"
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	data, err := os.ReadFile(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	var pres model.Presentation
	if err := json.Unmarshal(data, &pres); err != nil {
		fmt.Fprintln(os.Stderr, "parse json:", err)
		os.Exit(1)
	}
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
