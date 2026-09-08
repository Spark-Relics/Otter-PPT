package builder

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// goldenPresentation builds a deck that exercises the main element types
// and style features, so the golden files catch rendering regressions.
func goldenPresentation() *model.Presentation {
	accent := "#22D3EE"
	return &model.Presentation{
		Title: "Golden Deck",
		Theme: model.Theme{
			Name:            "dark_tech",
			PrimaryColor:    "#2563EB",
			SecondaryColor:  "#0F172A",
			AccentColor:     accent,
			BackgroundColor: "#0B1020",
			TextColor:       "#E2E8F0",
			TitleFont:       "Microsoft YaHei UI",
			BodyFont:        "Microsoft YaHei UI",
			StyleKey:        "dark_tech",
			PaletteKey:      "tech_neon",
		},
		SlideWidth:  13.333,
		SlideHeight: 7.5,
		Slides: []*model.Slide{
			{
				ID:     "s1",
				Layout: model.LayoutTitle,
				Background: &model.Background{
					Type: model.BgGradient,
					Gradient: &model.Gradient{
						Type:  model.GradientLinear,
						Angle: 135,
						Stops: []model.GradientStop{
							{Color: "#0B1020", Position: 0},
							{Color: "#1E293B", Position: 1},
						},
					},
				},
				Elements: []*model.Element{
					{
						ID:   "t1",
						Type: model.ElementTitle,
						Rect: model.Rect{X: 8, Y: 18, W: 84, H: 14},
						Text: "标题 Golden 标题",
						Style: model.TextStyle{
							FontSize: 44, Bold: true, Color: "#F8FAFC",
							Align: "left", FontName: "Microsoft YaHei UI",
						},
					},
					{
						ID:   "b1",
						Type: model.ElementBody,
						Rect: model.Rect{X: 8, Y: 36, W: 84, H: 20},
						Text: "第一个要点 中文混排\n第二个要点 bullet two",
						Style: model.TextStyle{
							FontSize: 18, Color: "#94A3B8",
							Align: "left", BulletChar: "•",
						},
					},
					{
						ID:   "sh1",
						Type: model.ElementShape,
						Rect: model.Rect{X: 70, Y: 60, W: 22, H: 12},
						Shape: &model.ShapeData{
							ShapeType:    model.ShapeRoundedRectangle,
							FillColor:    accent,
							CornerRadius: 0.25,
							Fill:         &model.FillStyle{Color: accent, Opacity: 0.9},
						},
					},
					{
						ID:   "c1",
						Type: model.ElementShape,
						Rect: model.Rect{X: 10, Y: 60, W: 10, H: 10},
						Shape: &model.ShapeData{
							ShapeType: model.ShapeEllipse,
							Fill:      &model.FillStyle{Color: accent, Opacity: 0.35},
						},
					},
					{
						ID:   "ln1",
						Type: model.ElementShape,
						Rect: model.Rect{X: 8, Y: 56, W: 84, H: 1},
						Shape: &model.ShapeData{
							ShapeType: model.ShapeLine,
							Line:      &model.LineStyle{Color: "#475569", Width: 1, Dash: "dash"},
						},
					},
				},
			},
			{
				ID:     "s2",
				Layout: model.LayoutTitleContent,
				Elements: []*model.Element{
					{
						ID:    "t2",
						Type:  model.ElementTitle,
						Rect:  model.Rect{X: 6, Y: 6, W: 88, H: 10},
						Text:  "图表与表格页",
						Style: model.TextStyle{FontSize: 32, Bold: true, Color: "#F8FAFC"},
					},
					{
						ID:   "ch1",
						Type: model.ElementChart,
						Rect: model.Rect{X: 6, Y: 20, W: 50, H: 55},
						Chart: &model.ChartData{
							ChartType:  "column",
							Title:      "季度数据",
							Categories: []string{"Q1", "Q2", "Q3", "Q4"},
							Series: []model.ChartSeries{
								{Name: "营收", Values: []float64{120, 150, 180, 210}},
								{Name: "成本", Values: []float64{80, 90, 110, 120}},
							},
						},
					},
					{
						ID:   "tb1",
						Type: model.ElementTable,
						Rect: model.Rect{X: 60, Y: 20, W: 34, H: 30},
						Table: &model.TableData{
							Headers: []model.TableCell{
								{Text: "模块"}, {Text: "状态"},
							},
							Rows: [][]model.TableCell{
								{{Text: "核心"}, {Text: "完成"}},
								{{Text: "预览"}, {Text: "完成"}},
							},
							HeaderColor: "#1E293B",
							BorderColor: "#334155",
						},
					},
					{
						ID:   "cn1",
						Type: model.ElementShape,
						Rect: model.Rect{X: 60, Y: 56, W: 34, H: 18},
						Shape: &model.ShapeData{
							ShapeType: model.ShapeRectangle,
							Fill:      &model.FillStyle{Color: "#111C33"},
							Line:      &model.LineStyle{Color: accent, Width: 1},
							Text:      "结论卡片",
							Style: model.TextStyle{
								FontSize: 16, Color: "#E2E8F0",
								Align: "center", VAlign: "middle",
							},
						},
					},
				},
			},
			{
				ID:     "s3",
				Layout: model.LayoutSection,
				Notes:  "备注内容 golden notes",
				Elements: []*model.Element{
					{
						ID:    "t3",
						Type:  model.ElementBody,
						Rect:  model.Rect{X: 10, Y: 10, W: 80, H: 10},
						Text:  "动画元素",
						Style: model.TextStyle{FontSize: 24, Color: "#F8FAFC"},
						Animation: &model.Animation{
							Type:    model.AnimFade,
							Trigger: model.TriggerOnClick,
						},
					},
				},
			},
		},
	}
}

// TestGoldenPPTX compares every XML/rels part of the built .pptx against
// golden files. Run with UPDATE_GOLDEN=1 to refresh the baseline, then
// review the diff in git like any code change.
func TestGoldenPPTX(t *testing.T) {
	pres := goldenPresentation()

	dir := t.TempDir()
	out := filepath.Join(dir, "golden.pptx")
	if err := New(pres).Save(out); err != nil {
		t.Fatalf("build: %v", err)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()

	goldenDir := "testdata/golden"
	update := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, f := range zr.File {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".xml" && ext != ".rels" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}

		name := strings.ReplaceAll(f.Name, "/", "__")
		goldenPath := filepath.Join(goldenDir, name)
		content := string(raw) + "\n"

		if update {
			if err := os.MkdirAll(goldenDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, []byte(content), 0o644); err != nil {
				t.Fatalf("write golden %s: %v", goldenPath, err)
			}
			continue
		}

		want, err := os.ReadFile(goldenPath)
		if os.IsNotExist(err) {
			t.Errorf("golden file missing for %s — run with UPDATE_GOLDEN=1 to generate", f.Name)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != content {
			t.Errorf("golden mismatch for %s\n--- want (first 300 chars)\n%s\n--- got (first 300 chars)\n%s",
				f.Name, truncate(string(want), 300), truncate(content, 300))
		}
	}
	if update {
		t.Log("golden files updated")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return fmt.Sprintf("%s…", s[:n])
}
