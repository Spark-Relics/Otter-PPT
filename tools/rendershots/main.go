//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/renderer"
)

func main() {
	data, err := os.ReadFile("examples/project_intro.json")
	if err != nil {
		panic(err)
	}
	var pres model.Presentation
	if err := json.Unmarshal(data, &pres); err != nil {
		panic(err)
	}
	pptxPath := "output/OtterPPT-项目介绍.pptx"
	if _, err := os.Stat(pptxPath); os.IsNotExist(err) {
		// Build the pptx first so we can render it.
		if err := builder.New(&pres).Save(pptxPath); err != nil {
			panic(err)
		}
	}
	r := renderer.NewRenderer()
	imgs, err := r.RenderPresentation(pptxPath, &pres)
	if err != nil {
		panic(err)
	}
	outDir := "output/intro_shots"
	os.MkdirAll(outDir, 0o755)
	for _, img := range imgs {
		if img.Path != "" {
			// copy to output dir
			b, _ := os.ReadFile(img.Path)
			dst := fmt.Sprintf("%s/slide%02d.png", outDir, img.SlideNum)
			os.WriteFile(dst, b, 0o644)
			fmt.Println("saved", dst, len(b), "bytes")
		} else {
			fmt.Printf("slide %d: fallback (%d chars)\n", img.SlideNum, len(img.FallbackDescription))
		}
	}
}
