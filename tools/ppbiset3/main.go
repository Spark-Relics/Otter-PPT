// ppbiset3: test embedded fonts (the feature unique to the intro deck).
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func save(p *model.Presentation, path string) {
	if err := builder.New(p).Save(path); err != nil {
		panic(err)
	}
}

func main() {
	out := os.Args[1]
	w, h := model.DefaultSlideSize()

	// Case M: embedded fonts via theme + element font names
	pm := &model.Presentation{
		Theme: model.Theme{
			Name:        "Bisect3",
			TitleFont:   "Noto Sans SC",
			BodyFont:    "Noto Sans SC",
			PrimaryColor: "#2563EB",
		},
		SlideWidth:  w,
		SlideHeight: h,
	}
	pm.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{Type: model.ElementTitle, Text: "Embedded", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}, Style: model.TextStyle{FontName: "Noto Sans SC"}},
		},
	}}
	save(pm, filepath.Join(out, "case_m_embedfont.pptx"))

	// Case N: 8 slides with notes + charts (intro-scale combination)
	pn := &model.Presentation{
		Theme:       model.Theme{Name: "Bisect3"},
		SlideWidth:  w,
		SlideHeight: h,
	}
	for i := 0; i < 8; i++ {
		pn.Slides = append(pn.Slides, &model.Slide{
			Layout: model.LayoutTitle,
			Notes:  fmt.Sprintf("notes %d", i),
			Elements: []*model.Element{
				{Type: model.ElementTitle, Text: fmt.Sprintf("Slide %d", i), Rect: model.Rect{X: 10, Y: 10, W: 80, H: 10}},
			},
		})
	}
	pn.Slides[3].Elements = append(pn.Slides[3].Elements, &model.Element{
		Type: model.ElementChart,
		Rect: model.Rect{X: 10, Y: 30, W: 80, H: 50},
		Chart: &model.ChartData{
			ChartType:  model.ChartBar,
			Categories: []string{"A", "B"},
			Series:     []model.ChartSeries{{Name: "S", Values: []float64{1, 2}}},
		},
	})
	save(pn, filepath.Join(out, "case_n_scale.pptx"))

	fmt.Println("generated all cases")
}
