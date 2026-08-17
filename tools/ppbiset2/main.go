// ppbiset2: bisect chart/table/icon features of the intro deck.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func base() *model.Presentation {
	w, h := model.DefaultSlideSize()
	return &model.Presentation{
		Theme:       model.Theme{Name: "Bisect2"},
		SlideWidth:  w,
		SlideHeight: h,
	}
}

func save(p *model.Presentation, path string) {
	if err := builder.New(p).Save(path); err != nil {
		panic(err)
	}
}

func main() {
	out := os.Args[1]

	// Case G: bar chart
	pg := base()
	pg.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{
				Type: model.ElementChart,
				Rect: model.Rect{X: 10, Y: 20, W: 80, H: 60},
				Chart: &model.ChartData{
					ChartType:  model.ChartBar,
					Title:      "Bar",
					Categories: []string{"A", "B", "C"},
					Series: []model.ChartSeries{
						{Name: "S1", Values: []float64{1, 2, 3}},
					},
				},
			},
		},
	}}
	save(pg, filepath.Join(out, "case_g_chart.pptx"))

	// Case H: table
	ph := base()
	ph.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{
				Type: model.ElementTable,
				Rect: model.Rect{X: 10, Y: 20, W: 80, H: 60},
				Table: &model.TableData{
					Headers: []model.TableCell{{Text: "H1"}, {Text: "H2"}},
					Rows: [][]model.TableCell{
						{{Text: "a"}, {Text: "b"}},
						{{Text: "c"}, {Text: "d"}},
					},
				},
			},
		},
	}}
	save(ph, filepath.Join(out, "case_h_table.pptx"))

	// Case I: icon element
	pi := base()
	pi.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{Type: model.ElementIcon, Text: "check", Rect: model.Rect{X: 10, Y: 20, W: 10, H: 10}},
		},
	}}
	save(pi, filepath.Join(out, "case_i_icon.pptx"))

	// Case J: connector
	pj := base()
	pj.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{Type: model.ElementConnector, Rect: model.Rect{X: 10, Y: 50, W: 80, H: 1}},
		},
	}}
	save(pj, filepath.Join(out, "case_j_connector.pptx"))

	// Case K: transition on slide
	pk := base()
	pk.Slides = []*model.Slide{{
		Layout:      model.LayoutTitle,
		Transition:  &model.Transition{Type: model.TransitionFade, Duration: 0.7},
		Elements: []*model.Element{
			{Type: model.ElementTitle, Text: "Trans", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}},
		},
	}}
	save(pk, filepath.Join(out, "case_k_transition.pptx"))

	// Case L: rounded rect with border + shadow style
	pl := base()
	pl.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{
				Type: model.ElementShape,
				Rect: model.Rect{X: 10, Y: 20, W: 40, H: 30},
				Shape: &model.ShapeData{
					ShapeType:    model.ShapeRoundedRectangle,
					Fill:         &model.FillStyle{Color: "#1E293B"},
					Line:         &model.LineStyle{Color: "#22D3EE", Width: 1.5},
					CornerRadius: 0.2,
				},
			},
		},
	}}
	save(pl, filepath.Join(out, "case_l_roundrect.pptx"))

	fmt.Println("generated all cases")
}
