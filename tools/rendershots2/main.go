//go:build ignore

// rendershots2 renders an arbitrary .pptx to per-slide PNGs using the
// project's renderer (browser → structured fallback).
// Usage: go run tools/rendershots2/main.go <pptx-path> <out-dir>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/renderer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: rendershots2 <pptx-path> [out-dir]")
		os.Exit(1)
	}
	pptx := os.Args[1]
	outDir := "output/shots"
	if len(os.Args) > 2 {
		outDir = os.Args[2]
	}
	r := renderer.NewRenderer()
	var pres *model.Presentation
	// Prefer the sibling JSON source (enables HTML render path with full fidelity).
	jsonPath := strings.TrimSuffix(pptx, ".pptx") + ".json"
	if _, err := os.Stat(jsonPath); err == nil {
		data, err := os.ReadFile(jsonPath)
		if err == nil {
			var p model.Presentation
			if json.Unmarshal(data, &p) == nil {
				pres = &p
			}
		}
	}
	imgs, err := r.RenderPresentation(pptx, pres)
	if err != nil {
		panic(err)
	}
	os.MkdirAll(outDir, 0o755)
	var lines []string
	for _, img := range imgs {
		if img.Path != "" {
			b, _ := os.ReadFile(img.Path)
			dst := fmt.Sprintf("%s/slide%02d.png", outDir, img.SlideNum)
			if err := os.WriteFile(dst, b, 0o644); err != nil {
				panic(err)
			}
			lines = append(lines, fmt.Sprintf("saved %s (%d bytes)", dst, len(b)))
		} else {
			lines = append(lines, fmt.Sprintf("slide %d: fallback (%d chars)", img.SlideNum, len(img.FallbackDescription)))
		}
	}
	fmt.Println(strings.Join(lines, "\n"))
	_ = json.Marshal
}
