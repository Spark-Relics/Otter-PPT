// ppbiset5: isolate what in the intro table breaks PowerPoint.
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

func slideWith(el *model.Element, bg bool) *model.Presentation {
	w, h := model.DefaultSlideSize()
	p := &model.Presentation{Theme: model.Theme{Name: "B5"}, SlideWidth: w, SlideHeight: h}
	s := &model.Slide{Layout: model.LayoutTitle, Elements: []*model.Element{el}}
	if bg {
		s.Background = &model.Background{Type: model.BgSolid, Color: "#0A0E1A"}
	}
	p.Slides = []*model.Slide{s}
	return p
}

func main() {
	out := os.Args[1]

	// P: styled table like intro (colors + font_size + alt rows)
	smallStyled := &model.Element{
		Type: model.ElementTable,
		Rect: model.Rect{X: 7, Y: 24, W: 86, H: 62},
		Table: &model.TableData{
			Headers: []model.TableCell{{Text: "A"}, {Text: "B"}, {Text: "C"}},
			Rows: [][]model.TableCell{
				{{Text: "1"}, {Text: "2"}, {Text: "3"}},
				{{Text: "4"}, {Text: "5"}, {Text: "6"}},
			},
			HeaderColor: "#141B2E",
			BorderColor: "#334155",
			AltRowColor: "#0F1524",
			FontSize:    12,
		},
	}
	save(slideWith(smallStyled, false), filepath.Join(out, "case_p_smallstyled.pptx"))

	// Q: CJK content table, no styling
	cjk := &model.Element{
		Type: model.ElementTable,
		Rect: model.Rect{X: 7, Y: 24, W: 86, H: 62},
		Table: &model.TableData{
			Headers: []model.TableCell{{Text: "能力域"}, {Text: "覆盖内容"}, {Text: "状态"}},
			Rows: [][]model.TableCell{
				{{Text: "原生图表"}, {Text: "13 种类型（含 3D）"}, {Text: "已完成"}},
			},
		},
	}
	save(slideWith(cjk, false), filepath.Join(out, "case_q_cjk.pptx"))

	// R: 8-row table like intro
	rows := [][]model.TableCell{}
	for i := 0; i < 8; i++ {
		rows = append(rows, []model.TableCell{
			{Text: fmt.Sprintf("行%d", i)},
			{Text: "渐变 / 透明度 / 阴影 / 发光 / 虚线 / 箭头 / freeform 自定义几何"},
			{Text: "已完成"},
		})
	}
	big := &model.Element{
		Type: model.ElementTable,
		Rect: model.Rect{X: 7, Y: 24, W: 86, H: 62},
		Table: &model.TableData{
			Headers:     []model.TableCell{{Text: "能力域"}, {Text: "覆盖内容"}, {Text: "状态"}},
			Rows:        rows,
			HeaderColor: "#141B2E",
			BorderColor: "#334155",
			AltRowColor: "#0F1524",
			FontSize:    12,
		},
	}
	save(slideWith(big, false), filepath.Join(out, "case_r_bigtable.pptx"))

	// S: the thin accent bar shape (s4-bar) with solid fill
	bar := &model.Element{
		Type: model.ElementShape,
		Rect: model.Rect{X: 7, Y: 16.5, W: 4, H: 0.5},
		Shape: &model.ShapeData{
			ShapeType: model.ShapeRectangle,
			Fill:      &model.FillStyle{Color: "#22D3EE"},
		},
	}
	save(slideWith(bar, false), filepath.Join(out, "case_s_bar.pptx"))

	// T: slide 4 exact minus table (title + bar + bg + notes)
	w, h := model.DefaultSlideSize()
	pt := &model.Presentation{Theme: model.Theme{Name: "B5"}, SlideWidth: w, SlideHeight: h}
	pt.Slides = []*model.Slide{{
		Layout: model.LayoutTitle,
		Background: &model.Background{
			Type: model.BgSolid, Color: "#0A0E1A",
		},
		Notes: "八项能力域全部落地。表格本身即由项目表格引擎渲染（含合并与隔行变色）。",
		Elements: []*model.Element{
			{ID: "t", Type: model.ElementTitle, Text: "核心能力矩阵", Rect: model.Rect{X: 7, Y: 6, W: 86, H: 10}, Style: model.TextStyle{FontSize: 34, Bold: true, Color: "#E2E8F0"}},
			bar,
		},
	}}
	save(pt, filepath.Join(out, "case_t_s4notable.pptx"))

	fmt.Println("generated all cases")
}
