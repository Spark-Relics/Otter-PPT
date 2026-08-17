// ppbiset: bisect which builder feature triggers PowerPoint repair.
// Generates minimal decks with incremental features; the caller opens each
// via PowerPoint COM to identify which feature trips the repair dialog.
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
		Theme: model.Theme{
			Name:            "Bisect",
			PrimaryColor:    "#2563EB",
			SecondaryColor:  "#0F172A",
			AccentColor:     "#F59E0B",
			BackgroundColor: "#F8FAFC",
			TextColor:       "#0F172A",
		},
		SlideWidth:  w,
		SlideHeight: h,
	}
}

func darkBG() *model.Background {
	return &model.Background{
		Type: model.BgGradient,
		Gradient: &model.Gradient{
			Type:  model.GradientLinear,
			Angle: 135,
			Stops: []model.GradientStop{
				{Position: 0, Color: "#0A0E1A"},
				{Position: 60, Color: "#141B2E"},
				{Position: 100, Color: "#1E293B"},
			},
		},
	}
}

func save(p *model.Presentation, path string) {
	if err := builder.New(p).Save(path); err != nil {
		panic(err)
	}
}

func main() {
	out := os.Args[1]

	// Case A: single slide, plain text only, no notes
	pa := base()
	pa.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Elements: []*model.Element{
			{Type: model.ElementTitle, Text: "Plain", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}},
		},
	}}
	save(pa, filepath.Join(out, "case_a_text.pptx"))

	// Case B: + gradient background
	pb := base()
	pb.Slides = []*model.Slide{{
		Layout:     model.LayoutTitle,
		Background: darkBG(),
		Elements: []*model.Element{
			{Type: model.ElementTitle, Text: "GradBg", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}, Style: model.TextStyle{Color: "#E2E8F0"}},
		},
	}}
	save(pb, filepath.Join(out, "case_b_gradbg.pptx"))

	// Case C: + colored run + font name + shadow (the rPr combo)
	pc := base()
	pc.Slides = []*model.Slide{{
		Layout:     model.LayoutTitle,
		Background: darkBG(),
		Elements: []*model.Element{{
			Type: model.ElementTitle,
			Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10},
			Paragraphs: []model.Paragraph{{
				Runs: []model.RichTextRun{{
					Text:  "Colored",
					Style: model.TextStyle{Color: "#22D3EE", FontName: "Noto Sans SC", FontSize: 32, Shadow: true},
				}},
			}},
		}},
	}}
	save(pc, filepath.Join(out, "case_c_rpr.pptx"))

	// Case D: + notes (notesMaster path)
	pd := base()
	pd.Slides = []*model.Slide{
		pc.Slides[0],
		{
			Layout: model.LayoutTitle,
			Notes:  "speaker notes here",
			Elements: []*model.Element{
				{Type: model.ElementTitle, Text: "Notes slide", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}},
			},
		},
	}
	save(pd, filepath.Join(out, "case_d_notes.pptx"))

	// Case E: + animation timing
	pe := base()
	anim := &model.Animation{Type: model.AnimFade, Trigger: model.TriggerOnClick, Duration: 0.5}
	pe.Slides = []*model.Slide{{
		Layout:     model.LayoutTitle,
		Background: darkBG(),
		Elements: []*model.Element{
			{Type: model.ElementTitle, Text: "Animated", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}, Style: model.TextStyle{Color: "#E2E8F0"}, Animation: anim},
		},
	}}
	save(pe, filepath.Join(out, "case_e_anim.pptx"))

	// Case F: + gradient-filled decorative shape (like intro)
	pf := base()
	pf.Slides = []*model.Slide{{
		Layout:     model.LayoutTitle,
		Background: darkBG(),
		Elements: []*model.Element{
			{
				Type: model.ElementShape,
				Rect: model.Rect{X: 70, Y: -5, W: 30, H: 20},
				Shape: &model.ShapeData{
					ShapeType: model.ShapeEllipse,
					Fill: &model.FillStyle{
						Gradient: &model.Gradient{
							Type: model.GradientRadial,
							Stops: []model.GradientStop{
								{Position: 0, Color: "#22D3EE", Opacity: 0.35},
								{Position: 70, Color: "#22D3EE", Opacity: 0.05},
								{Position: 100, Color: "#22D3EE"},
							},
						},
					},
				},
			},
			{Type: model.ElementTitle, Text: "Decor", Rect: model.Rect{X: 10, Y: 40, W: 80, H: 10}, Style: model.TextStyle{Color: "#E2E8F0"}},
		},
	}}
	save(pf, filepath.Join(out, "case_f_shape.pptx"))

	fmt.Println("generated all cases")
}
