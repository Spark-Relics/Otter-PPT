//go:build ignore

// debug-html renders prompt-compare.json via the browser path.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/renderer"
)

func main() {
	data, err := os.ReadFile("output/prompt-compare.json")
	if err != nil {
		panic(err)
	}
	var p model.Presentation
	if err := json.Unmarshal(data, &p); err != nil {
		panic(err)
	}
	r := renderer.NewRenderer()
	browser := renderer.FindBrowser()
	if browser == nil {
		fmt.Println("NO BROWSER FOUND")
		os.Exit(1)
	}
	r.ForceBrowserOnly()
	imgs, err := r.RenderPresentation("output/prompt-compare.pptx", &p)
	if err != nil {
		panic(err)
	}
	os.MkdirAll("output/prompt-compare-shots", 0o755)
	for _, img := range imgs {
		if img.Path == "" {
			fmt.Printf("slide %d: fallback only\n", img.SlideNum)
			continue
		}
		b, _ := os.ReadFile(img.Path)
		dst := fmt.Sprintf("output/prompt-compare-shots/slide%02d.png", img.SlideNum)
		os.WriteFile(dst, b, 0o644)
		fmt.Printf("saved %s (%d bytes)\n", dst, len(b))
	}
}
