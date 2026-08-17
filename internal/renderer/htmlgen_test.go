package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func samplePresentation() *model.Presentation {
	return &model.Presentation{
		Title: "Render Test",
		Theme: model.Theme{
			PrimaryColor:    "#1a5276",
			SecondaryColor:  "#2e86c1",
			AccentColor:     "#f39c12",
			BackgroundColor: "#ffffff",
			TextColor:       "#2c3e50",
		},
		Slides: []*model.Slide{
			{
				ID:     "s1",
				Layout: model.LayoutTitle,
				Background: &model.Background{
					Type: model.BgGradient,
					Gradient: &model.Gradient{
						Type: model.GradientLinear,
						Angle: 90,
						Stops: []model.GradientStop{
							{Color: "#1a5276", Position: 0},
							{Color: "#2e86c1", Position: 100},
						},
					},
				},
				Elements: []*model.Element{
					{ID: "t1", Type: model.ElementTitle, Text: "标题测试 中文",
						Rect: model.Rect{X: 10, Y: 30, W: 80, H: 20},
						Style: model.TextStyle{FontSize: 44, Color: "#ffffff", Align: "center"}},
					{ID: "sub1", Type: model.ElementSubtitle, Text: "HTML Render Path <b>escape</b> test",
						Rect: model.Rect{X: 10, Y: 55, W: 80, H: 12},
						Style: model.TextStyle{FontSize: 20, Color: "#d6eaf8", Align: "center"}},
				},
			},
			{
				ID:     "s2",
				Layout: model.LayoutTitleContent,
				Elements: []*model.Element{
					{ID: "t2", Type: model.ElementTitle, Text: "Content Slide",
						Rect: model.Rect{X: 5, Y: 5, W: 90, H: 15}},
					{ID: "b1", Type: model.ElementBullet,
						Rect: model.Rect{X: 5, Y: 25, W: 45, H: 40},
						Items: []string{"First point 要点一", "Second point", "Third point"},
						Style: model.TextStyle{FontSize: 16}},
					{ID: "shp1", Type: model.ElementShape,
						Rect: model.Rect{X: 55, Y: 25, W: 15, H: 20},
						Shape: &model.ShapeData{
							ShapeType: model.ShapeRoundedRectangle,
							Fill:      &model.FillStyle{Color: "#f39c12"},
							Line:      &model.LineStyle{Color: "#9c640c", Width: 1.5},
							Text:      "Shape",
						}},
					{ID: "shp2", Type: model.ElementShape,
						Rect: model.Rect{X: 75, Y: 25, W: 15, H: 20},
						Shape: &model.ShapeData{
							ShapeType: model.ShapeEllipse,
							Fill:      &model.FillStyle{Gradient: &model.Gradient{
								Type:  model.GradientRadial,
								Stops: []model.GradientStop{{Color: "#e74c3c", Position: 0}, {Color: "#c0392b", Position: 100}},
							}},
						}},
					{ID: "tbl1", Type: model.ElementTable,
						Rect: model.Rect{X: 5, Y: 70, W: 60, H: 25},
						Table: &model.TableData{
							Headers: []model.TableCell{
								{Text: "名称"}, {Text: "值"}, {Text: "备注"},
							},
							Rows: [][]model.TableCell{
								{{Text: "Alpha"}, {Text: "1.0"}, {Text: "first"}},
								{{Text: "Beta", Style: model.TableCellStyle{Bold: true}}, {Text: "2.5"}, {Text: "second", ColSpan: 1}},
							},
							HeaderColor: "#1a5276",
							AltRowColor: "#eaf2f8",
							FontSize:    11,
						}},
					{ID: "cht1", Type: model.ElementChart,
						Rect: model.Rect{X: 70, Y: 55, W: 28, H: 40},
						Chart: &model.ChartData{
							ChartType:  model.ChartColumn,
							Title:      "Quarterly",
							Categories: []string{"Q1", "Q2", "Q3", "Q4"},
							Series: []model.ChartSeries{
								{Name: "Sales", Values: []float64{12, 25, 18, 30}},
								{Name: "Cost", Values: []float64{8, 15, 11, 22}, Color: "#e67e22"},
							},
							ShowLegend:     true,
							ShowDataLabels: false,
						}},
					{ID: "cn1", Type: model.ElementConnector,
						Rect: model.Rect{},
						Connector: &model.ConnectorData{
							ConnectorType: model.ShapeArrow,
							Color:         "#7f8c8d",
							Width:         1.5,
							StartX:        5, StartY: 65,
							EndX:          40, EndY: 65,
						}},
				},
			},
			{
				ID:     "s3",
				Layout: model.LayoutTitleContent,
				Elements: []*model.Element{
					{ID: "t3", Type: model.ElementTitle, Text: "More Charts",
						Rect: model.Rect{X: 5, Y: 5, W: 90, H: 15}},
					{ID: "pie", Type: model.ElementChart,
						Rect: model.Rect{X: 5, Y: 25, W: 28, H: 45},
						Chart: &model.ChartData{
							ChartType:  model.ChartPie,
							Categories: []string{"A", "B", "C", "D"},
							Series:     []model.ChartSeries{{Name: "Share", Values: []float64{30, 25, 25, 20}}},
							ShowDataLabels: true,
						}},
					{ID: "line", Type: model.ElementChart,
						Rect: model.Rect{X: 36, Y: 25, W: 30, H: 45},
						Chart: &model.ChartData{
							ChartType:  model.ChartLine,
							Categories: []string{"Jan", "Feb", "Mar", "Apr", "May"},
							Series: []model.ChartSeries{
								{Name: "Trend", Values: []float64{3, 7, 5, 9, 12}},
							},
						}},
					{ID: "scat", Type: model.ElementChart,
						Rect: model.Rect{X: 69, Y: 25, W: 28, H: 45},
						Chart: &model.ChartData{
							ChartType: model.ChartScatter,
							Series: []model.ChartSeries{
								{Name: "Points", XValues: []float64{1, 2, 3, 4, 5}, Values: []float64{2, 4.5, 3, 7, 6}},
							},
						}},
				},
			},
		},
	}
}

func TestGenerateHTML(t *testing.T) {
	pres := samplePresentation()
	out := filepath.Join(t.TempDir(), "slides.html")
	if err := GenerateHTML(pres, out); err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)

	checks := []struct{ name, needle string }{
		{"doctype", "<!DOCTYPE html>"},
		{"slide div", `class="slide"`},
		{"title text", "标题测试 中文"},
		{"escaped text", "&lt;b&gt;escape&lt;/b&gt;"},
		{"bullet", "First point 要点一"},
		{"gradient bg", "linear-gradient"},
		{"radial gradient", "radial-gradient"},
		{"shape radius", "border-radius"},
		{"table header bg", "#1a5276"},
		{"chart svg", "<svg"},
		{"column rect", "<rect"},
		{"pie path", "<path"},
		{"polyline", "<polyline"},
		{"connector line", "marker-end"},
		{"cjk fallback", "Microsoft YaHei"},
	}
	for _, c := range checks {
		if !strings.Contains(html, c.needle) {
			t.Errorf("HTML missing %s (looked for %q)", c.name, c.needle)
		}
	}
	if got := strings.Count(html, `class="slide"`); got != 3 {
		t.Errorf("expected 3 slides, got %d", got)
	}
}

func TestWriteSingleSlideHTML(t *testing.T) {
	pres := samplePresentation()
	dir := t.TempDir()
	all := filepath.Join(dir, "all.html")
	if err := GenerateHTML(pres, all); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "slide-1.html")
	if err := writeSingleSlideHTML(pres, 0, all, out, 1600, 900); err != nil {
		t.Fatalf("writeSingleSlideHTML: %v", err)
	}
	data, _ := os.ReadFile(out)
	html := string(data)
	if !strings.Contains(html, "标题测试 中文") {
		t.Error("single slide HTML missing title")
	}
	if !strings.Contains(html, "transform:scale(") {
		t.Error("missing viewport scale transform")
	}
	// Slide 2 content must NOT be in slide 1's page
	if strings.Contains(html, "Content Slide") {
		t.Error("slide 2 leaked into slide 1 page")
	}
}

// TestBrowserScreenshot is an integration test — it runs only when a
// headless browser is present (Edge on Windows, Chrome/Chromium elsewhere).
func TestBrowserScreenshot(t *testing.T) {
	browser := FindBrowser()
	if browser == nil {
		t.Skip("no headless-capable browser found")
	}

	pres := samplePresentation()
	r := NewRenderer()
	r.libreOfficePath = "" // force browser path
	r.pdftoppmPath = ""
	r.browser = browser

	images, err := r.renderWithBrowser(pres)
	if err != nil {
		t.Fatalf("renderWithBrowser: %v", err)
	}
	if len(images) != 3 {
		t.Fatalf("expected 3 slide images, got %d", len(images))
	}
	for i, img := range images {
		if img.SlideNum != i+1 {
			t.Errorf("image %d has SlideNum %d", i, img.SlideNum)
		}
		if len(img.Base64) < 1000 {
			t.Errorf("slide %d base64 suspiciously small: %d bytes", i+1, len(img.Base64))
		}
		if img.Width < 300 || img.Height < 200 {
			t.Errorf("slide %d too small: %dx%d", i+1, img.Width, img.Height)
		}
	}
	t.Logf("rendered %d slides via %s", len(images), browser.Name)
}

func TestCssGradient(t *testing.T) {
	g := &model.Gradient{
		Type:  model.GradientLinear,
		Angle: 45,
		Stops: []model.GradientStop{
			{Color: "#ff0000", Position: 0},
			{Color: "#0000ff", Position: 100},
		},
	}
	out := cssGradient(g)
	if !strings.Contains(out, "linear-gradient") || !strings.Contains(out, "#ff0000") {
		t.Errorf("unexpected gradient: %s", out)
	}
}

func TestCssRGBA(t *testing.T) {
	if got := cssRGBA("#ff0000", 0.5); got != "rgba(255,0,0,0.50)" {
		t.Errorf("cssRGBA: %s", got)
	}
	if got := cssRGBA("#abc", 1); got != "#aabbcc" {
		t.Errorf("cssRGBA 3-digit: %s", got)
	}
}
