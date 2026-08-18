package svgdecode

import (
	"math"
	"testing"
)

func TestParsePathBasics(t *testing.T) {
	pd, err := ParsePath("M 10 20 L 30 40 L 50 40 Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(pd.Subpaths) != 1 {
		t.Fatalf("subpaths = %d, want 1", len(pd.Subpaths))
	}
	sp := pd.Subpaths[0]
	if !sp.Closed {
		t.Error("subpath should be closed")
	}
	if sp.Start != (Point{10, 20}) {
		t.Errorf("start = %v", sp.Start)
	}
	if len(sp.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(sp.Nodes))
	}
	if sp.Nodes[0].Pt != (Point{30, 40}) || sp.Nodes[1].Pt != (Point{50, 40}) {
		t.Errorf("nodes = %+v", sp.Nodes)
	}
}

func TestParsePathRelativeAndImplicit(t *testing.T) {
	// m 10 10 20 20 → move + implicit relative lineTo
	pd, err := ParsePath("m 10 10 20 20 l 5 0")
	if err != nil {
		t.Fatal(err)
	}
	sp := pd.Subpaths[0]
	if sp.Start != (Point{10, 10}) {
		t.Errorf("start = %v", sp.Start)
	}
	want := []Point{{30, 30}, {35, 30}}
	for i, pt := range want {
		if sp.Nodes[i].Pt != pt {
			t.Errorf("node %d = %v, want %v", i, sp.Nodes[i].Pt, pt)
		}
	}
}

func TestParsePathCurves(t *testing.T) {
	pd, err := ParsePath("M 0 0 C 10 0 10 10 20 10 S 30 20 30 30 Q 40 40 50 50 T 60 60")
	if err != nil {
		t.Fatal(err)
	}
	if len(pd.Subpaths[0].Nodes) != 4 {
		t.Fatalf("nodes = %d, want 4", len(pd.Subpaths[0].Nodes))
	}
	// S reflects previous C2 around current point (20,10): 2*20-10, 2*10-10
	n2 := pd.Subpaths[0].Nodes[1]
	if n2.Cmd != 'C' || n2.C1 != (Point{30, 10}) {
		t.Errorf("S node = %+v", n2)
	}
}

func TestFlattenArc(t *testing.T) {
	// Semicircle arc.
	pd, err := ParsePath("M 0 0 A 50 50 0 0 1 100 0")
	if err != nil {
		t.Fatal(err)
	}
	polys := pd.Flatten()
	if len(polys) != 1 {
		t.Fatalf("polys = %d", len(polys))
	}
	// Midpoint of upper semicircle should be near (50, -50).
	minY := math.Inf(1)
	for _, p := range polys[0] {
		if p.Y < minY {
			minY = p.Y
		}
	}
	if math.Abs(minY-(-50)) > 2 {
		t.Errorf("arc apex Y = %.2f, want ≈ -50", minY)
	}
}

func TestCompileRect(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720">
	  <rect x="100" y="100" width="400" height="300" fill="#1A73E8" stroke="#000000" stroke-width="2"/>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(res.Elements))
	}
	e := res.Elements[0]
	if e.Type != "shape" || e.Shape == nil {
		t.Fatalf("type = %v", e.Type)
	}
	if e.Shape.ShapeType != "rectangle" {
		t.Errorf("shapeType = %v", e.Shape.ShapeType)
	}
	// 100/1280*100 = 7.8125, 100/720*100 = 13.888…
	if math.Abs(e.Rect.X-7.8125) > 0.01 || math.Abs(e.Rect.Y-13.8889) > 0.01 {
		t.Errorf("rect = %+v", e.Rect)
	}
	if e.Shape.Fill == nil || e.Shape.Fill.Color != "#1A73E8" {
		t.Errorf("fill = %+v", e.Shape.Fill)
	}
	if e.Shape.Line == nil || e.Shape.Line.Width != 2 {
		t.Errorf("line = %+v", e.Shape.Line)
	}
}

func TestCompileGroupTransform(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 500">
	  <g transform="translate(100 50) scale(2)">
	    <rect x="0" y="0" width="50" height="40" fill="#FF0000"/>
	  </g>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := res.Elements[0]
	// Final rect: x=100..200, y=50..130 → pct X=10, Y=10, W=10, H=16
	if math.Abs(e.Rect.X-10) > 0.01 || math.Abs(e.Rect.Y-10) > 0.01 {
		t.Errorf("rect = %+v", e.Rect)
	}
	if math.Abs(e.Rect.W-10) > 0.01 || math.Abs(e.Rect.H-16) > 0.01 {
		t.Errorf("rect = %+v", e.Rect)
	}
}

func TestCompileCircleAndText(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 500">
	  <circle cx="500" cy="250" r="100" fill="#00AA00"/>
	  <text x="500" y="100" font-size="40" fill="#123456" text-anchor="middle" font-weight="bold">Hello</text>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Elements) != 2 {
		t.Fatalf("elements = %d, want 2", len(res.Elements))
	}
	circ := res.Elements[0]
	if circ.Shape == nil || circ.Shape.ShapeType != "ellipse" {
		t.Fatalf("circle shapeType = %+v", circ.Shape)
	}
	txt := res.Elements[1]
	if txt.Text != "Hello" {
		t.Errorf("text = %q", txt.Text)
	}
	if !txt.Style.Bold || txt.Style.Align != "center" {
		t.Errorf("style = %+v", txt.Style)
	}
	// Font size: 40 units * (7.5*72/500) = 43.2pt ≈ 43
	if txt.Style.FontSize != 43 {
		t.Errorf("fontSize = %d, want 43", txt.Style.FontSize)
	}
}

func TestCompileFreeformPath(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 500">
	  <path d="M 100 100 L 300 100 L 250 300 L 150 300 Z" fill="#3333CC"/>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := res.Elements[0]
	if e.Shape == nil || e.Shape.ShapeType != "freeform" || e.Shape.Freeform == nil {
		t.Fatalf("shape = %+v", e.Shape)
	}
	ff := e.Shape.Freeform
	if len(ff.Contours) != 1 || len(ff.Closed) != 1 || !ff.Closed[0] {
		t.Fatalf("freeform = %+v", ff)
	}
	// Normalized first point: (100-100)/200 = 0, (100-100)/200 = 0
	if ff.Contours[0][0].X != 0 || ff.Contours[0][0].Y != 0 {
		t.Errorf("first contour point = %+v", ff.Contours[0][0])
	}
}

func TestCompileLineToConnector(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 500">
	  <line x1="100" y1="100" x2="400" y2="300" stroke="#FF8800" stroke-width="3"/>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := res.Elements[0]
	if e.Type != "connector" || e.Connector == nil {
		t.Fatalf("type = %v", e.Type)
	}
	if e.Connector.Color != "#FF8800" || e.Connector.Width != 3 {
		t.Errorf("connector = %+v", e.Connector)
	}
	if e.Connector.StartX != 10 || e.Connector.StartY != 20 ||
		e.Connector.EndX != 40 || e.Connector.EndY != 60 {
		t.Errorf("connector pts = %+v", e.Connector)
	}
}

func TestCompileImageDataURL(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 500">
	  <image x="200" y="100" width="300" height="200" href="data:image/png;base64,aGVsbG8="/>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e := res.Elements[0]
	if e.Type != "image" {
		t.Fatalf("type = %v", e.Type)
	}
	data, _, err := DecodeDataURL(e.ImagePath)
	if err != nil || string(data) != "hello" {
		t.Errorf("decode = %q, %v", data, err)
	}
}

func TestCompileSkippedGradient(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 500">
	  <rect x="10" y="10" width="100" height="100" fill="url(#g1)"/>
	</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) == 0 {
		t.Error("expected gradient fill to be reported as skipped/approximated")
	}
}

func TestChartPlotAreaMarker(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720">
  <!-- chart-plot-area: 300,180,640,380 -->
  <rect x="100" y="160" width="1080" height="470" fill="#F8FAFC"/>
</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.PlotArea == nil {
		t.Fatal("expected chart-plot-area marker to be parsed")
	}
	// 300/1280*100 = 23.4375, 180/720*100 = 25, 640/1280*100 = 50, 380/720*100 ≈ 52.778
	if math.Abs(res.PlotArea.X-23.4375) > 0.01 || math.Abs(res.PlotArea.Y-25) > 0.01 {
		t.Errorf("plot area = %+v", res.PlotArea)
	}
	if math.Abs(res.PlotArea.W-50) > 0.01 || math.Abs(res.PlotArea.H-52.7778) > 0.1 {
		t.Errorf("plot area = %+v", res.PlotArea)
	}
}
