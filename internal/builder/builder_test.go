package builder

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func TestEmbeddedImageCreatesMediaAndRelationship(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "sample.png")
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 20, G: 80, B: 180, A: 255})
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()

	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{ID: "image-1", Type: model.ElementImage, Rect: model.Rect{X: 10, Y: 10, W: 40, H: 40}, ImagePath: imagePath, ImageAlt: "sample"}}}}}
	var output bytes.Buffer
	if err := New(pres).Write(&output); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := make(map[string]string)
	for _, part := range zr.File {
		r, _ := part.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		parts[part.Name] = string(data)
	}
	if _, ok := parts["ppt/media/image1.png"]; !ok {
		t.Fatal("embedded image media part missing")
	}
	if !strings.Contains(parts["ppt/slides/slide1.xml"], `<p:pic>`) || !strings.Contains(parts["ppt/slides/slide1.xml"], `r:embed="rId2"`) {
		t.Fatal("slide picture reference missing")
	}
	if !strings.Contains(parts["ppt/slides/_rels/slide1.xml.rels"], `Target="../media/image1.png"`) {
		t.Fatal("image relationship missing")
	}
	if !strings.Contains(parts["[Content_Types].xml"], `Extension="png" ContentType="image/png"`) {
		t.Fatal("image content type missing")
	}
}

func TestNativeChartCreatesEditableChartPart(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-1", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartColumn, Categories: []string{"Q1", "Q2"}, Title: "Revenue", ShowLegend: true, Series: []model.ChartSeries{{Name: "2026", Values: []float64{12.5, 18}, Color: "#4472C4"}}},
	}}}}}
	var output bytes.Buffer
	if err := New(pres).Write(&output); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := make(map[string]string)
	for _, part := range zr.File {
		r, _ := part.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		parts[part.Name] = string(data)
	}
	if !strings.Contains(parts["ppt/slides/slide1.xml"], `<c:chart`) || !strings.Contains(parts["ppt/slides/slide1.xml"], `r:id="rId2"`) {
		t.Fatal("native chart graphic frame reference missing")
	}
	chartXML := parts["ppt/charts/chart1.xml"]
	if !strings.Contains(chartXML, `<c:barChart>`) || !strings.Contains(chartXML, `<c:v>Q1</c:v>`) || !strings.Contains(chartXML, `<c:v>12.5</c:v>`) {
		t.Fatalf("native chart data missing: %s", chartXML)
	}
	if !strings.Contains(parts["ppt/slides/_rels/slide1.xml.rels"], `relationships/chart`) || !strings.Contains(parts["ppt/slides/_rels/slide1.xml.rels"], `../charts/chart1.xml`) {
		t.Fatal("chart relationship missing")
	}
	if !strings.Contains(parts["[Content_Types].xml"], `drawingml.chart+xml`) {
		t.Fatal("chart content type missing")
	}
}

func TestScatterChartCreatesScatterChartPart(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-scatter", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartScatter, Title: "Correlation", ShowLegend: true,
			Series: []model.ChartSeries{
				{Name: "Series A", XValues: []float64{1, 2, 3, 4}, Values: []float64{2, 4, 5, 4}, Color: "#4472C4"},
				{Name: "Series B", XValues: []float64{1, 2, 3}, Values: []float64{3, 5, 7}, Color: "#ED7D31"},
			}},
	}}}}}
	var output bytes.Buffer
	if err := New(pres).Write(&output); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := make(map[string]string)
	for _, part := range zr.File {
		r, _ := part.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		parts[part.Name] = string(data)
	}
	chartXML := parts["ppt/charts/chart1.xml"]
	// Must use scatterChart element
	if !strings.Contains(chartXML, `<c:scatterChart>`) {
		t.Fatalf("scatter chart element missing: %s", chartXML)
	}
	// Must have scatterStyle
	if !strings.Contains(chartXML, `<c:scatterStyle val="lineMarker"/>`) {
		t.Fatalf("scatterStyle missing: %s", chartXML)
	}
	// Must have xVal/yVal with numeric data
	if !strings.Contains(chartXML, `<c:xVal>`) || !strings.Contains(chartXML, `<c:yVal>`) {
		t.Fatalf("xVal/yVal missing in scatter series: %s", chartXML)
	}
	// Must have two valAx (no catAx)
	if strings.Contains(chartXML, `<c:catAx>`) {
		t.Fatalf("scatter chart should not have catAx: %s", chartXML)
	}
	if strings.Count(chartXML, `<c:valAx>`) != 2 {
		t.Fatalf("scatter chart should have 2 valAx, got %d", strings.Count(chartXML, `<c:valAx>`))
	}
	// Must have markers
	if !strings.Contains(chartXML, `<c:marker>`) {
		t.Fatalf("marker missing in scatter series: %s", chartXML)
	}
	// Verify data values are present
	if !strings.Contains(chartXML, `<c:v>4</c:v>`) {
		t.Fatalf("scatter y-value missing: %s", chartXML)
	}
}

func TestComboChartWithSecondaryAxis(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-combo", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartCombo, Categories: []string{"Q1", "Q2", "Q3"}, Title: "Revenue vs Growth", ShowLegend: true,
			Series: []model.ChartSeries{
				{Name: "Revenue", Values: []float64{100, 150, 200}, Color: "#4472C4", ChartType: model.ChartColumn},
				{Name: "Growth %", Values: []float64{10, 15, 25}, Color: "#ED7D31", ChartType: model.ChartLine, SecondaryAxis: true, Trendline: "linear"},
			}},
	}}}}}
	var output bytes.Buffer
	if err := New(pres).Write(&output); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := make(map[string]string)
	for _, part := range zr.File {
		r, _ := part.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		parts[part.Name] = string(data)
	}
	chartXML := parts["ppt/charts/chart1.xml"]
	// Must have both barChart and lineChart sub-elements
	if !strings.Contains(chartXML, `<c:barChart>`) {
		t.Fatalf("combo chart missing barChart: %s", chartXML)
	}
	if !strings.Contains(chartXML, `<c:lineChart>`) {
		t.Fatalf("combo chart missing lineChart: %s", chartXML)
	}
	// Must have 3 axes: catAx + primary valAx + secondary valAx
	if strings.Count(chartXML, `<c:catAx>`) != 1 {
		t.Fatalf("combo chart should have 1 catAx, got %d", strings.Count(chartXML, `<c:catAx>`))
	}
	if strings.Count(chartXML, `<c:valAx>`) != 2 {
		t.Fatalf("combo chart should have 2 valAx, got %d", strings.Count(chartXML, `<c:valAx>`))
	}
	// Secondary valAx must be on the right
	if !strings.Contains(chartXML, `<c:axPos val="r"/>`) {
		t.Fatalf("secondary axis should be positioned right: %s", chartXML)
	}
	// Must have trendline
	if !strings.Contains(chartXML, `<c:trendline>`) {
		t.Fatalf("trendline missing: %s", chartXML)
	}
	// Must use numCache/strCache
	if !strings.Contains(chartXML, `<c:numCache>`) || !strings.Contains(chartXML, `<c:strCache>`) {
		t.Fatalf("numCache/strCache missing: %s", chartXML)
	}
}

func TestLineChartSmoothAndTrendline(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-line", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartLine, Categories: []string{"A", "B", "C"}, Title: "Trend", Smooth: true,
			Series: []model.ChartSeries{
				{Name: "Data", Values: []float64{1, 4, 9}, Color: "#4472C4", Trendline: "polynomial"},
			}},
	}}}}}
	var output bytes.Buffer
	if err := New(pres).Write(&output); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := make(map[string]string)
	for _, part := range zr.File {
		r, _ := part.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		parts[part.Name] = string(data)
	}
	chartXML := parts["ppt/charts/chart1.xml"]
	// Must have smooth
	if !strings.Contains(chartXML, `<c:smooth val="1"/>`) {
		t.Fatalf("smooth flag missing: %s", chartXML)
	}
	// Must have trendline with polynomial type
	if !strings.Contains(chartXML, `<c:trendlineType val="poly"/>`) {
		t.Fatalf("polynomial trendline missing: %s", chartXML)
	}
	// Must have marker
	if !strings.Contains(chartXML, `<c:marker>`) {
		t.Fatalf("marker missing: %s", chartXML)
	}
}

func TestGeneratedPartsAreWellFormedXML(t *testing.T) {
	pres := &model.Presentation{
		Title: "OOXML validation",
		Slides: []*model.Slide{{
			ID:     "slide-1",
			Layout: model.LayoutTitleContent,
			Background: &model.Background{Type: model.BgGradient, Gradient: &model.Gradient{
				Type: model.GradientLinear, Angle: 135,
				Stops: []model.GradientStop{{Color: "#07111F", Position: 0}, {Color: "#0E7490", Position: 100}},
			}},
			Transition: &model.Transition{Type: model.TransitionFade, Duration: 0.7},
			Elements: []*model.Element{
				{
					ID: "text-alpha", Type: model.ElementTitle, Text: "Otter PPT",
					Rect: model.Rect{X: 10, Y: 10, W: 80, H: 20},
				},
				{
					ID: "shape-alpha", Type: model.ElementShape, Rect: model.Rect{X: 10, Y: 35, W: 30, H: 20},
					Rotation: 15, Shape: &model.ShapeData{ShapeType: model.ShapeRoundedRectangle, FillColor: "#3B82F6"},
				},
			},
		}},
	}

	var output bytes.Buffer
	if err := New(pres).Write(&output); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatalf("open PPTX zip: %v", err)
	}
	for _, part := range zr.File {
		if len(part.Name) < 4 || (part.Name[len(part.Name)-4:] != ".xml" && part.Name[len(part.Name)-5:] != ".rels") {
			continue
		}
		r, err := part.Open()
		if err != nil {
			t.Fatalf("open %s: %v", part.Name, err)
		}
		data, readErr := io.ReadAll(r)
		r.Close()
		if readErr != nil {
			t.Fatalf("read %s: %v", part.Name, readErr)
		}
		var node any
		if err := xml.Unmarshal(data, &node); err != nil {
			t.Errorf("invalid XML in %s: %v", part.Name, err)
		}
		if part.Name == "ppt/slides/slide1.xml" {
			xmlText := string(data)
			if !strings.Contains(xmlText, `<a:xfrm><a:off x="`) || !strings.Contains(xmlText, `<a:ext cx="`) {
				t.Errorf("slide shape is missing required a:xfrm transform: %s", xmlText)
			}
			if strings.Contains(xmlText, `<p:spPr><a:off`) {
				t.Errorf("shape transform children must be wrapped in a:xfrm: %s", xmlText)
			}
			if count := strings.Count(xmlText, `<p:spPr><a:xfrm`); count != 2 {
				t.Errorf("expected every shape to use the OOXML property builder, got %d transforms", count)
			}
			if !strings.Contains(xmlText, `<a:xfrm rot="900000">`) {
				t.Errorf("shape rotation must be encoded on a:xfrm: %s", xmlText)
			}
		}
	}
}
