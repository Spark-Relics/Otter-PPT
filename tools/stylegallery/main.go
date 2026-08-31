//go:build ignore

// stylegallery builds a visual specimen deck for every design style preset
// and renders each to PNG. Output: output/stylegallery/<style>/<style>.pptx
// + <style>.json + slide01.png / slide02.png — the "see it, don't read it"
// companion to the design style catalog.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/gallery"
	"github.com/otter-ppt/otter-ppt/internal/renderer"
)

func main() {
	outDir := "output/stylegallery"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	r := renderer.NewRenderer()
	fail := 0
	for _, spec := range gallery.Specimens() {
		dir := filepath.Join(outDir, spec.Style)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		pres, err := gallery.Build(spec)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", spec.Style, err)
			fail++
			continue
		}
		jsonPath := filepath.Join(dir, spec.Style+".json")
		data, _ := json.MarshalIndent(pres, "", "  ")
		if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
			panic(err)
		}
		pptxPath := filepath.Join(dir, spec.Style+".pptx")
		if err := builder.New(pres).Save(pptxPath); err != nil {
			fmt.Printf("✗ %s: build: %v\n", spec.Style, err)
			fail++
			continue
		}
		imgs, err := r.RenderPresentation(pptxPath, pres)
		if err != nil {
			fmt.Printf("✗ %s: render: %v\n", spec.Style, err)
			fail++
			continue
		}
		for _, img := range imgs {
			if img.Path == "" {
				fmt.Printf("! %s slide %d: structured fallback (no PNG)\n", spec.Style, img.SlideNum)
				continue
			}
			b, _ := os.ReadFile(img.Path)
			dst := filepath.Join(dir, fmt.Sprintf("slide%02d.png", img.SlideNum))
			if err := os.WriteFile(dst, b, 0o644); err != nil {
				panic(err)
			}
			fmt.Printf("✓ %s → %s (%d bytes)\n", spec.Style, dst, len(b))
		}
	}
	if fail > 0 {
		fmt.Printf("done with %d failures\n", fail)
		os.Exit(1)
	}
	fmt.Println("style gallery complete")
}
