//go:build ignore

// buildcap compiles examples/capability_overview.json into a .pptx and
// renders per-slide screenshots for visual verification.
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
	data, err := os.ReadFile("examples/capability_overview.json")
	if err != nil {
		panic(err)
	}
	var pres model.Presentation
	if err := json.Unmarshal(data, &pres); err != nil {
		panic(err)
	}
	out := "output/OtterPPT-能力全景.pptx"
	if err := builder.New(&pres).Save(out); err != nil {
		panic(err)
	}
	fmt.Println("saved:", out, "slides:", len(pres.Slides))

	r := renderer.NewRenderer()
	imgs, err := r.RenderPresentation(out, &pres)
	if err != nil {
		panic(err)
	}
	outDir := "output/cap_shots"
	os.MkdirAll(outDir, 0o755)
	for _, img := range imgs {
		if img.Path != "" {
			b, _ := os.ReadFile(img.Path)
			dst := fmt.Sprintf("%s/slide%02d.png", outDir, img.SlideNum)
			os.WriteFile(dst, b, 0o644)
			fmt.Println("saved", dst, len(b), "bytes")
		} else {
			fmt.Printf("slide %d: fallback (%d chars)\n", img.SlideNum, len(img.FallbackDescription))
		}
	}
}
