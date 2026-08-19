package parser

import (
	"bytes"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

// buildSample creates a presentation exercising every element type the
// parser supports, builds it to PPTX bytes, then parses it back.
func buildSample(t *testing.T) *model.Presentation {
	t.Helper()
	pres := &model.Presentation{
		Title:       "Roundtrip",
		SlideWidth:  13.333,
		SlideHeight: 7.5,
		Theme: model.Theme{
			Name:            "RT",
			PrimaryColor:    "#123456",
			SecondaryColor:  "#0F172A",
			AccentColor:     "#F59E0B",
			BackgroundColor: "#FFFFFF",
			TextColor:       "#0F172A",
		},
	}

	bulletSize := 18
	slide1 := &model.Slide{
		ID:     "slide-1",
		Layout: model.LayoutTitleContent,
		Notes:  "hello notes",
		Elements: []*model.Element{
			{ID: "title1", Type: model.ElementTitle, Rect: model.Rect{X: 10, Y: 8, W: 80, H: 12},
				Text: "Roundtrip Title",
				Style: model.TextStyle{FontSize: 40, Bold: true, Color: "#123456"}},
			{ID: "body1", Type: model.ElementBody, Rect: model.Rect{X: 10, Y: 30, W: 40, H: 10},
				Text:  "Plain body text",
				Style: model.TextStyle{FontSize: 16, Color: "#0F172A"}},
			{ID: "bul1", Type: model.ElementBullet, Rect: model.Rect{X: 10, Y: 45, W: 40, H: 30},
				Items: []string{"one", "two", "three"},
				Style: model.TextStyle{FontSize: bulletSize}},
			{ID: "shp1", Type: model.ElementShape, Rect: model.Rect{X: 60, Y: 30, W: 20, H: 15},
				Rotation: 15,
				Shape: &model.ShapeData{
					ShapeType: model.ShapeRoundedRectangle,
					Fill:      &model.FillStyle{Color: "#22D3EE"},
					Line:      &model.LineStyle{Color: "#475569", Width: 1.5},
					Text:      "shape text",
				}},
			{ID: "tbl1", Type: model.ElementTable, Rect: model.Rect{X: 55, Y: 50, W: 40, H: 25},
				Table: &model.TableData{
					Headers: []model.TableCell{{Text: "A"}, {Text: "B"}},
					Rows: [][]model.TableCell{
						{{Text: "1"}, {Text: "2"}},
						{{Text: "3"}, {Text: "4"}},
					},
				}},
			{ID: "cht1", Type: model.ElementChart, Rect: model.Rect{X: 5, Y: 80, W: 40, H: 18},
				Chart: &model.ChartData{
					ChartType:  model.ChartColumn,
					Title:      "Sales",
					ShowLegend: true,
					Categories: []string{"Q1", "Q2"},
					Series: []model.ChartSeries{
						{Name: "rev", Values: []float64{10, 20}},
					},
				}},
		},
	}

	_ = slide1
	pres.Slides = append(pres.Slides, slide1)
	return pres
}

func TestRoundtrip(t *testing.T) {
	pres := buildSample(t)

	var buf bytes.Buffer
	if err := builder.New(pres).Write(&buf); err != nil {
		t.Fatalf("build: %v", err)
	}

	res, err := ParseBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(res.Presentation.Slides) != 1 {
		t.Fatalf("want 1 slide, got %d (warnings: %v)", len(res.Presentation.Slides), res.Warnings)
	}
	slide := res.Presentation.Slides[0]

	// Elements are classified by content (IDs are not restored).
	var title, body, bullet, shape, table, chart bool
	for _, el := range slide.Elements {
		switch el.Type {
		case model.ElementTitle:
			title = true
			if el.Text != "Roundtrip Title" {
				t.Errorf("title text = %q", el.Text)
			}
			if el.Style.FontSize != 40 || !el.Style.Bold {
				t.Errorf("title style = %+v", el.Style)
			}
		case model.ElementBody:
			body = true
			if el.Text != "Plain body text" {
				t.Errorf("body text = %q", el.Text)
			}
		case model.ElementBullet:
			bullet = true
			if len(el.Items) != 3 || el.Items[0] != "one" {
				t.Errorf("bullet items = %v", el.Items)
			}
		case model.ElementShape:
			shape = true
			if el.Shape == nil || el.Shape.ShapeType != model.ShapeRoundedRectangle {
				t.Errorf("shape type = %+v", el.Shape)
			}
			if el.Rotation != 15 {
				t.Errorf("rotation = %v", el.Rotation)
			}
			if el.Shape.Text != "shape text" {
				t.Errorf("shape text = %q", el.Shape.Text)
			}
		case model.ElementTable:
			table = true
			if el.Table == nil || len(el.Table.Headers) != 2 || len(el.Table.Rows) != 2 {
				t.Errorf("table = %+v", el.Table)
			}
			if el.Table.Headers[0].Text != "A" || el.Table.Rows[1][1].Text != "4" {
				t.Errorf("table content = %+v", el.Table)
			}
		case model.ElementChart:
			chart = true
			if el.Chart == nil || el.Chart.ChartType != model.ChartColumn {
				t.Errorf("chart type = %+v", el.Chart)
			}
			if el.Chart.Title != "Sales" {
				t.Errorf("chart title = %q", el.Chart.Title)
			}
			if len(el.Chart.Categories) != 2 || el.Chart.Categories[0] != "Q1" {
				t.Errorf("chart cats = %v", el.Chart.Categories)
			}
			if len(el.Chart.Series) != 1 || len(el.Chart.Series[0].Values) != 2 || el.Chart.Series[0].Values[1] != 20 {
				t.Errorf("chart series = %+v", el.Chart.Series)
			}
		}
	}

	if !title || !body || !bullet || !shape || !table || !chart {
		t.Errorf("missing element types: title=%v body=%v bullet=%v shape=%v table=%v chart=%v (warnings: %v)",
			title, body, bullet, shape, table, chart, res.Warnings)
	}

	// Speaker notes round-trip.
	if slide.Notes != "hello notes" {
		t.Errorf("notes = %q", slide.Notes)
	}

	// Theme round-trip.
	if res.Presentation.Theme.PrimaryColor != "#123456" {
		t.Errorf("theme primary = %s", res.Presentation.Theme.PrimaryColor)
	}

	// Presentation title from first title element.
	if res.Presentation.Title != "Roundtrip Title" {
		t.Errorf("presentation title = %q", res.Presentation.Title)
	}
}

func TestRoundtripGeometry(t *testing.T) {
	pres := buildSample(t)
	var buf bytes.Buffer
	if err := builder.New(pres).Write(&buf); err != nil {
		t.Fatalf("build: %v", err)
	}
	res, err := ParseBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// All parsed elements should land within ~0.5% of the original rects.
	want := map[string]model.Rect{
		"title": {X: 10, Y: 8, W: 80, H: 12},
		"chart": {X: 5, Y: 80, W: 40, H: 18},
	}
	for _, el := range res.Presentation.Slides[0].Elements {
		var key string
		switch {
		case el.Type == model.ElementTitle:
			key = "title"
		case el.Type == model.ElementChart:
			key = "chart"
		}
		if key == "" {
			continue
		}
		w := want[key]
		d := func(a, b float64) float64 {
			if a-b > 0 {
				return a - b
			}
			return b - a
		}
		for _, pair := range [][2]float64{{el.Rect.X, w.X}, {el.Rect.Y, w.Y}, {el.Rect.W, w.W}, {el.Rect.H, w.H}} {
			if d(pair[0], pair[1]) > 0.5 {
				t.Errorf("%s rect off: %+v (want %+v)", key, el.Rect, w)
				break
			}
		}
	}
}

func TestGradientBackground(t *testing.T) {
	pres := &model.Presentation{
		Title: "bg", SlideWidth: 13.333, SlideHeight: 7.5,
		Slides: []*model.Slide{{
			ID: "s1", Layout: model.LayoutTitleContent,
			Background: &model.Background{
				Type: model.BgGradient,
				Gradient: &model.Gradient{
					Type:  model.GradientLinear,
					Angle: 45,
					Stops: []model.GradientStop{
						{Color: "#0EA5E9", Position: 0},
						{Color: "#1E1B4B", Position: 100},
					},
				},
			},
		}},
	}
	var buf bytes.Buffer
	if err := builder.New(pres).Write(&buf); err != nil {
		t.Fatalf("build: %v", err)
	}
	res, err := ParseBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bg := res.Presentation.Slides[0].Background
	if bg == nil || bg.Type != model.BgGradient || bg.Gradient == nil {
		t.Fatalf("background = %+v", bg)
	}
	if len(bg.Gradient.Stops) != 2 || bg.Gradient.Stops[0].Color != "#0EA5E9" {
		t.Errorf("gradient stops = %+v", bg.Gradient.Stops)
	}
}

func TestParseFileErrors(t *testing.T) {
	if _, err := ParseFile("does-not-exist.pptx"); err == nil {
		t.Error("expected error for missing file")
	}
	if _, err := ParseBytes([]byte("not a zip")); err == nil {
		t.Error("expected error for non-zip input")
	}
}
