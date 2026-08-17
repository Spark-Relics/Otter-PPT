// ppbiset4: open intro JSON and save decks with slide subsets (prefix bisect).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	data, err := os.ReadFile("examples/project_intro.json")
	if err != nil {
		panic(err)
	}
	var full model.Presentation
	if err := json.Unmarshal(data, &full); err != nil {
		panic(err)
	}

	out := os.Args[1]
	n := len(full.Slides)

	// prefix decks: slides 1..k
	for k := 1; k <= n; k++ {
		p := full
		p.Slides = full.Slides[:k]
		if err := builder.New(&p).Save(filepath.Join(out, fmt.Sprintf("intro_%02d_slides.pptx", k))); err != nil {
			panic(err)
		}
	}

	// suffix deck: slides k..n (to catch header/presentation-level issues vs slide issues)
	p := full
	p.Slides = full.Slides[n-1:]
	if err := builder.New(&p).Save(filepath.Join(out, "intro_last_only.pptx")); err != nil {
		panic(err)
	}

	// all slides but strip per-slide extra features: rebuild each slide keeping
	// only text-like elements (drop shapes/charts/images) to isolate content type
	strip := full
	for i, s := range strip.Slides {
		kept := s.Elements[:0]
		for _, e := range s.Elements {
			switch e.Type {
			case model.ElementTitle, model.ElementSubtitle, model.ElementBody, model.ElementBullet:
				kept = append(kept, e)
			}
		}
		clone := *s
		clone.Elements = kept
		clone.Background = nil
		clone.Transition = nil
		clone.Notes = ""
		strip.Slides[i] = &clone
	}
	if err := builder.New(&strip).Save(filepath.Join(out, "intro_textonly.pptx")); err != nil {
		panic(err)
	}

	fmt.Println("generated", n, "prefix decks + suffix + textonly; total slides:", strings.Repeat("", 0)+fmt.Sprint(n))
}
