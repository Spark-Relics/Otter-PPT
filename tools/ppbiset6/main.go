// ppbiset6: exact replicas of intro slide 4 with variations.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	out := os.Args[1]

	data, err := os.ReadFile("examples/project_intro.json")
	if err != nil {
		panic(err)
	}
	var full model.Presentation
	if err := json.Unmarshal(data, &full); err != nil {
		panic(err)
	}
	s4 := full.Slides[3]

	save := func(p *model.Presentation, name string) {
		if err := builder.New(p).Save(filepath.Join(out, name)); err != nil {
			panic(err)
		}
		fmt.Println("saved", name)
	}

	w, h := model.DefaultSlideSize()

	// U: exact slide 4 alone
	pu := &model.Presentation{Theme: full.Theme, SlideWidth: full.SlideWidth, SlideHeight: full.SlideHeight}
	pu.Slides = []*model.Slide{s4}
	save(pu, "case_u_exact_s4.pptx")

	// V: slide 4 with layout switched to title
	pv := &model.Presentation{Theme: full.Theme, SlideWidth: full.SlideWidth, SlideHeight: full.SlideHeight}
	s4v := *s4
	s4v.Layout = model.LayoutTitle
	pv.Slides = []*model.Slide{&s4v}
	save(pv, "case_v_s4_titlelayout.pptx")

	// W: slide 4 without notes
	pw := &model.Presentation{Theme: full.Theme, SlideWidth: full.SlideWidth, SlideHeight: full.SlideHeight}
	s4w := *s4
	s4w.Notes = ""
	pw.Slides = []*model.Slide{&s4w}
	save(pw, "case_w_s4_nonotes.pptx")

	// X: slides 1-3 + slide 4 (replicating intro_04 failure) with default size
	px := &model.Presentation{Theme: full.Theme, SlideWidth: w, SlideHeight: h}
	px.Slides = full.Slides[:4]
	save(px, "case_x_defaultsize4.pptx")
}
