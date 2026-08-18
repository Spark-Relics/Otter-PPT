package builder

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
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

func Test3DColumnChart(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-3d", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartColumn3D, Categories: []string{"A", "B", "C"}, Title: "3D Revenue", ShowLegend: true,
			Series: []model.ChartSeries{
				{Name: "2026", Values: []float64{10, 20, 30}, Color: "#4472C4"},
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
	// Must use bar3DChart element
	if !strings.Contains(chartXML, `<c:bar3DChart>`) {
		t.Fatalf("3D chart element missing: %s", chartXML)
	}
	// Must have view3D
	if !strings.Contains(chartXML, `<c:view3D>`) {
		t.Fatalf("view3D missing: %s", chartXML)
	}
	// Must have rotX and rotY
	if !strings.Contains(chartXML, `<c:rotX val="15"/>`) || !strings.Contains(chartXML, `<c:rotY val="20"/>`) {
		t.Fatalf("3D rotation missing: %s", chartXML)
	}
	// Must have catAx and valAx
	if !strings.Contains(chartXML, `<c:catAx>`) || !strings.Contains(chartXML, `<c:valAx>`) {
		t.Fatalf("axes missing in 3D chart: %s", chartXML)
	}
	// Must close with bar3DChart
	if !strings.Contains(chartXML, `</c:bar3DChart>`) {
		t.Fatalf("3D chart closing tag missing: %s", chartXML)
	}
}

func Test3DPieChart(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-3dpie", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartPie3D, Categories: []string{"A", "B"}, Title: "Share", ShowLegend: true,
			Series: []model.ChartSeries{
				{Name: "Share", Values: []float64{60, 40}, Color: "#4472C4"},
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
	// Must use pie3DChart
	if !strings.Contains(chartXML, `<c:pie3DChart>`) {
		t.Fatalf("3D pie chart element missing: %s", chartXML)
	}
	// Must have view3D
	if !strings.Contains(chartXML, `<c:view3D>`) {
		t.Fatalf("view3D missing: %s", chartXML)
	}
	// Must NOT have axes (pie charts don't use axes)
	if strings.Contains(chartXML, `<c:catAx>`) || strings.Contains(chartXML, `<c:valAx>`) {
		t.Fatalf("3D pie should not have axes: %s", chartXML)
	}
}

func TestElementAnimation(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{
		{
			ID: "title-1", Type: model.ElementTitle, Text: "Hello",
			Rect: model.Rect{X: 10, Y: 10, W: 80, H: 20},
			Animation: &model.Animation{
				Type:      model.AnimFade,
				Trigger:   model.TriggerOnClick,
				Duration:  0.5,
			},
		},
		{
			ID: "body-1", Type: model.ElementBody, Text: "World",
			Rect: model.Rect{X: 10, Y: 40, W: 80, H: 20},
			Animation: &model.Animation{
				Type:      model.AnimFlyIn,
				Trigger:   model.TriggerAfterPrev,
				Direction: model.DirFromLeft,
				Duration:  0.8,
				Delay:     0.2,
			},
		},
	}}}}
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
	slideXML := parts["ppt/slides/slide1.xml"]
	// Must have <p:timing>
	if !strings.Contains(slideXML, `<p:timing>`) {
		t.Fatalf("timing element missing: %s", slideXML)
	}
	// Must have main sequence
	if !strings.Contains(slideXML, `nodeType="mainSeq"`) {
		t.Fatalf("main sequence missing: %s", slideXML)
	}
	// Must have animEffect with "fade" for the first element
	if !strings.Contains(slideXML, `filter="fade"`) {
		t.Fatalf("fade animation filter missing: %s", slideXML)
	}
	// Must have animEffect with "wipe(left)" for the second element (fly_in from left)
	if !strings.Contains(slideXML, `filter="wipe(left)"`) {
		t.Fatalf("wipe animation filter missing: %s", slideXML)
	}
	// Must have proper timing node types
	if !strings.Contains(slideXML, `nodeType="clickEffect"`) {
		t.Fatalf("clickEffect node type missing: %s", slideXML)
	}
	if !strings.Contains(slideXML, `nodeType="afterEffect"`) {
		t.Fatalf("afterEffect node type missing: %s", slideXML)
	}
	// Must close timing
	if !strings.Contains(slideXML, `</p:timing>`) {
		t.Fatalf("timing closing tag missing: %s", slideXML)
	}
}

func TestTableMergedCells(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "table-1", Type: model.ElementTable, Rect: model.Rect{X: 10, Y: 10, W: 80, H: 60},
		Table: &model.TableData{
			Headers: []model.TableCell{
				{Text: "Category", ColSpan: 2},
				{Text: ""},
				{Text: "Value"},
			},
			Rows: [][]model.TableCell{
				{{Text: "A", ColSpan: 2}, {Text: ""}, {Text: "100"}},
				{{Text: "B"}, {Text: "B2"}, {Text: "200", RowSpan: 2}},
				{{Text: "C"}, {Text: "C2"}, {Text: "", RowSpan: -1}},
			},
		},
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
	slideXML := parts["ppt/slides/slide1.xml"]
	// Must have gridSpan for col_span
	if !strings.Contains(slideXML, `gridSpan="2"`) {
		t.Fatalf("gridSpan (col_span) missing: %s", slideXML)
	}
	// Must have rowSpan for row_span
	if !strings.Contains(slideXML, `rowSpan="2"`) {
		t.Fatalf("rowSpan missing: %s", slideXML)
	}
	// Must have hMerge for continuation cells (row_span=-1)
	if !strings.Contains(slideXML, `hMerge="1"`) {
		t.Fatalf("hMerge missing: %s", slideXML)
	}
}

func TestSVGImageEmbed(t *testing.T) {
	// Create a minimal SVG file
	svgPath := filepath.Join(t.TempDir(), "test.svg")
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect width="100" height="100" fill="blue"/></svg>`
	if err := os.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
		t.Fatal(err)
	}
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "svg-1", Type: model.ElementImage, Rect: model.Rect{X: 10, Y: 10, W: 40, H: 40},
		ImagePath: svgPath, ImageAlt: "SVG test",
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
	// SVG file must be embedded in media
	mediaKey := ""
	for k := range parts {
		if strings.Contains(k, "ppt/media/") && strings.HasSuffix(k, ".svg") {
			mediaKey = k
			break
		}
	}
	if mediaKey == "" {
		t.Fatal("SVG media file not embedded")
	}
	// Content types must include svg+xml
	if !strings.Contains(parts["[Content_Types].xml"], `image/svg+xml`) {
		t.Fatalf("SVG content type missing: %s", parts["[Content_Types].xml"])
	}
	// Slide XML must have svgBlip extension
	slideXML := parts["ppt/slides/slide1.xml"]
	if !strings.Contains(slideXML, `svgBlip`) {
		t.Fatalf("svgBlip extension missing: %s", slideXML)
	}
}

func TestVideoEmbed(t *testing.T) {
	// Create a minimal MP4 file (just some bytes)
	videoPath := filepath.Join(t.TempDir(), "test.mp4")
	if err := os.WriteFile(videoPath, []byte("fake mp4 data"), 0644); err != nil {
		t.Fatal(err)
	}
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "video-1", Type: model.ElementVideo, Rect: model.Rect{X: 10, Y: 10, W: 60, H: 50},
		Media: &model.MediaData{
			MediaPath: videoPath,
			MediaType: "video",
			MimeType:  "video/mp4",
		},
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
	// Video file must be embedded
	mediaKey := ""
	for k := range parts {
		if strings.Contains(k, "ppt/media/") && strings.HasSuffix(k, ".mp4") {
			mediaKey = k
			break
		}
	}
	if mediaKey == "" {
		t.Fatal("video media file not embedded")
	}
	// Content types must include video/mp4
	if !strings.Contains(parts["[Content_Types].xml"], `video/mp4`) {
		t.Fatalf("video content type missing: %s", parts["[Content_Types].xml"])
	}
	// Slide XML must have video reference
	slideXML := parts["ppt/slides/slide1.xml"]
	if !strings.Contains(slideXML, `p14:media`) {
		t.Fatalf("video p14:media extension missing: %s", slideXML)
	}
	// Slide rels must have video relationship type
	relsXML := parts["ppt/slides/_rels/slide1.xml.rels"]
	if !strings.Contains(relsXML, `relationships/video`) {
		t.Fatalf("video relationship missing: %s", relsXML)
	}
}

func TestAudioEmbed(t *testing.T) {
	// Create a minimal MP3 file (just some bytes)
	audioPath := filepath.Join(t.TempDir(), "test.mp3")
	if err := os.WriteFile(audioPath, []byte("fake mp3 data"), 0644); err != nil {
		t.Fatal(err)
	}
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "audio-1", Type: model.ElementAudio, Rect: model.Rect{X: 10, Y: 10, W: 30, H: 10},
		Media: &model.MediaData{
			MediaPath: audioPath,
			MediaType: "audio",
			MimeType:  "audio/mpeg",
		},
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
	// Audio file must be embedded
	mediaKey := ""
	for k := range parts {
		if strings.Contains(k, "ppt/media/") && strings.HasSuffix(k, ".mp3") {
			mediaKey = k
			break
		}
	}
	if mediaKey == "" {
		t.Fatal("audio media file not embedded")
	}
	// Content types must include audio/mpeg
	if !strings.Contains(parts["[Content_Types].xml"], `audio/mpeg`) {
		t.Fatalf("audio content type missing: %s", parts["[Content_Types].xml"])
	}
	// Slide XML must have audio reference
	slideXML := parts["ppt/slides/slide1.xml"]
	if !strings.Contains(slideXML, `p14:media`) {
		t.Fatalf("audio p14:media extension missing: %s", slideXML)
	}
	// Slide rels must have audio relationship type
	relsXML := parts["ppt/slides/_rels/slide1.xml.rels"]
	if !strings.Contains(relsXML, `relationships/audio`) {
		t.Fatalf("audio relationship missing: %s", relsXML)
	}
}

func TestColumnChartWithErrorBars(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{{ID: "slide-1", Elements: []*model.Element{{
		ID: "chart-err", Type: model.ElementChart, Rect: model.Rect{X: 10, Y: 10, W: 70, H: 60},
		Chart: &model.ChartData{ChartType: model.ChartColumn, Categories: []string{"A", "B", "C"}, Title: "Measurements",
			Series: []model.ChartSeries{
				{Name: "Data", Values: []float64{10, 20, 30}, Color: "#4472C4",
					ErrorBars: &model.ErrorBarStyle{Direction: "y", Type: "fixedVal", Value: 1.5}},
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
	// Must have error bars
	if !strings.Contains(chartXML, `<c:errBars>`) {
		t.Fatalf("error bars element missing: %s", chartXML)
	}
	if !strings.Contains(chartXML, `<c:errDir val="y"/>`) {
		t.Fatalf("error direction missing: %s", chartXML)
	}
	if !strings.Contains(chartXML, `<c:errValType val="fixedVal"/>`) {
		t.Fatalf("error value type missing: %s", chartXML)
	}
	if !strings.Contains(chartXML, `<c:v>1.5</c:v>`) {
		t.Fatalf("error value missing: %s", chartXML)
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

func TestMultiLayoutPlaceholders(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{
		{ID: "s1", Layout: model.LayoutTitle, Elements: []*model.Element{
			{ID: "t1", Type: model.ElementTitle, Text: "Title", Rect: model.Rect{X: 10, Y: 10, W: 80, H: 20}},
		}},
		{ID: "s2", Layout: model.LayoutTitleContent, Elements: []*model.Element{
			{ID: "t2", Type: model.ElementTitle, Text: "Title", Rect: model.Rect{X: 10, Y: 10, W: 80, H: 20}},
			{ID: "b2", Type: model.ElementBody, Text: "Body", Rect: model.Rect{X: 10, Y: 35, W: 80, H: 50}},
		}},
		{ID: "s3", Layout: model.LayoutTwoColumn, Elements: []*model.Element{}},
		{ID: "s4", Layout: model.LayoutSection, Elements: []*model.Element{}},
		{ID: "s5", Layout: model.LayoutImageLeft, Elements: []*model.Element{}},
	}}
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

	// All 5 layout files must exist
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("ppt/slideLayouts/slideLayout%d.xml", i)
		if _, ok := parts[key]; !ok {
			t.Fatalf("layout file %s missing", key)
		}
	}

	// Content types must include all 5 layout overrides
	ctXML := parts["[Content_Types].xml"]
	for i := 1; i <= 5; i++ {
		expected := fmt.Sprintf(`PartName="/ppt/slideLayouts/slideLayout%d.xml"`, i)
		if !strings.Contains(ctXML, expected) {
			t.Fatalf("content type missing layout %d override", i)
		}
	}

	// Master must reference all 5 layouts
	masterXML := parts["ppt/slideMasters/slideMaster1.xml"]
	for i := 1; i <= 5; i++ {
		expected := fmt.Sprintf(`Target="../slideLayouts/slideLayout%d.xml"`, i)
		if !strings.Contains(masterXML, expected) {
			// Check master rels instead
			masterRels := parts["ppt/slideMasters/_rels/slideMaster1.xml.rels"]
			if !strings.Contains(masterRels, expected) {
				t.Fatalf("master rels missing layout %d", i)
			}
		}
	}

	// Layout 1 (title) must have ctrTitle placeholder
	layout1 := parts["ppt/slideLayouts/slideLayout1.xml"]
	if !strings.Contains(layout1, `type="ctrTitle"`) {
		t.Fatalf("layout 1 (title) missing ctrTitle placeholder: %s", layout1)
	}
	if !strings.Contains(layout1, `type="subTitle"`) {
		t.Fatalf("layout 1 (title) missing subTitle placeholder: %s", layout1)
	}

	// Layout 2 (titleContent) must have title + body placeholders
	layout2 := parts["ppt/slideLayouts/slideLayout2.xml"]
	if !strings.Contains(layout2, `type="title"`) {
		t.Fatalf("layout 2 (titleContent) missing title placeholder")
	}
	if !strings.Contains(layout2, `type="body"`) {
		t.Fatalf("layout 2 (titleContent) missing body placeholder")
	}

	// Layout 3 (twoColumn) must have title + two body placeholders
	layout3 := parts["ppt/slideLayouts/slideLayout3.xml"]
	if strings.Count(layout3, `type="body"`) != 2 {
		t.Fatalf("layout 3 (twoColumn) should have 2 body placeholders, got %d", strings.Count(layout3, `type="body"`))
	}

	// Layout 4 (section) must have ctrTitle
	layout4 := parts["ppt/slideLayouts/slideLayout4.xml"]
	if !strings.Contains(layout4, `type="ctrTitle"`) {
		t.Fatalf("layout 4 (section) missing ctrTitle placeholder")
	}

	// Layout 5 (blank) should have no placeholders
	layout5 := parts["ppt/slideLayouts/slideLayout5.xml"]
	if strings.Contains(layout5, `<p:ph `) {
		t.Fatalf("layout 5 (blank) should have no placeholders")
	}

	// Slide rels must point to correct layout
	// s1=LayoutTitle → layout1, s2=LayoutTitleContent → layout2, s3=LayoutTwoColumn → layout3, s4=LayoutSection → layout4, s5=LayoutImageLeft → layout3
	slide1Rels := parts["ppt/slides/_rels/slide1.xml.rels"]
	if !strings.Contains(slide1Rels, `slideLayout1.xml`) {
		t.Fatalf("slide 1 should reference slideLayout1.xml")
	}
	slide2Rels := parts["ppt/slides/_rels/slide2.xml.rels"]
	if !strings.Contains(slide2Rels, `slideLayout2.xml`) {
		t.Fatalf("slide 2 should reference slideLayout2.xml")
	}
	slide3Rels := parts["ppt/slides/_rels/slide3.xml.rels"]
	if !strings.Contains(slide3Rels, `slideLayout3.xml`) {
		t.Fatalf("slide 3 should reference slideLayout3.xml")
	}
	slide4Rels := parts["ppt/slides/_rels/slide4.xml.rels"]
	if !strings.Contains(slide4Rels, `slideLayout4.xml`) {
		t.Fatalf("slide 4 should reference slideLayout4.xml")
	}
	slide5Rels := parts["ppt/slides/_rels/slide5.xml.rels"]
	if !strings.Contains(slide5Rels, `slideLayout3.xml`) {
		t.Fatalf("slide 5 (image_left) should reference slideLayout3.xml (twoColumn mapping)")
	}

	// All layout XML must be well-formed
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("ppt/slideLayouts/slideLayout%d.xml", i)
		var node any
		if err := xml.Unmarshal([]byte(parts[key]), &node); err != nil {
			t.Errorf("invalid XML in %s: %v", key, err)
		}
	}
}

func TestContrastTextColor(t *testing.T) {
	if got := contrastTextColor("#0F172A"); got != "#FFFFFF" {
		t.Errorf("dark fill should use light text, got %s", got)
	}
	if got := contrastTextColor("#F8FAFC"); got != "#0F172A" {
		t.Errorf("light fill should use dark text, got %s", got)
	}
}

func TestShapeTextStyleThemeAware(t *testing.T) {
	pres := &model.Presentation{
		Theme: model.Theme{BodyFont: "Microsoft YaHei", TextColor: "#E2E8F0"},
		Slides: []*model.Slide{{ID: "s1"}},
	}
	b := New(pres)

	// Dark fill without explicit text color → light readable text.
	dark := &model.Element{ID: "shape-dark", Shape: &model.ShapeData{
		ShapeType: model.ShapeRoundedRectangle,
		FillColor: "#141B2E",
		Text:      "卡牌",
	}}
	darkStyle := b.shapeTextStyle(dark)
	if darkStyle.Color != "#FFFFFF" {
		t.Errorf("dark fill text color = %s, want #FFFFFF", darkStyle.Color)
	}
	if darkStyle.FontName != "Microsoft YaHei" {
		t.Errorf("font name = %s, want theme body font", darkStyle.FontName)
	}
	if darkStyle.MarginLeft == 0 || darkStyle.MarginRight == 0 {
		t.Errorf("expected non-zero horizontal insets, got L=%v R=%v", darkStyle.MarginLeft, darkStyle.MarginRight)
	}

	// Light fill without explicit text color → dark readable text.
	light := &model.Element{ID: "shape-light", Shape: &model.ShapeData{
		ShapeType: model.ShapeRoundedRectangle,
		FillColor: "#F8FAFC",
		Text:      "卡牌",
	}}
	lightStyle := b.shapeTextStyle(light)
	if lightStyle.Color != "#0F172A" {
		t.Errorf("light fill text color = %s, want #0F172A", lightStyle.Color)
	}

	// Explicit text color must win over auto-contrast.
	explicit := &model.Element{ID: "shape-explicit", Shape: &model.ShapeData{
		ShapeType: model.ShapeRoundedRectangle,
		FillColor: "#141B2E",
		Text:      "卡牌",
		Style:     model.TextStyle{Color: "#22D3EE", FontSize: 24, Bold: true},
	}}
	explicitStyle := b.shapeTextStyle(explicit)
	if explicitStyle.Color != "#22D3EE" {
		t.Errorf("explicit text color should win, got %s", explicitStyle.Color)
	}
	if explicitStyle.FontSize != 24 {
		t.Errorf("explicit font size should win, got %d", explicitStyle.FontSize)
	}
}
