package svgdecode

import (
	"strings"
	"testing"
)

func TestNativeChartUpgrade(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720">
  <rect width="1280" height="720" fill="#FFFFFF"/>
  <g id="chartArea" data-pptx-bounds="100 160 1080 470" data-pptx-replace-with="chart">
    <metadata type="application/json">
      {"type":"column","categories":["A","B","C"],"values":[10,20,30],"style":{"color":"#2563EB"}}
    </metadata>
    <rect x="100" y="160" width="1080" height="470" fill="#F8FAFC"/>
  </g>
</svg>`

	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(res.Elements) != 2 {
		t.Fatalf("expected 2 elements (background + native chart), got %d", len(res.Elements))
	}
	// The g region must upgrade to exactly one native chart; only the outer
	// background rect remains a shape. If the group's child rect leaked through,
	// there would be two shapes.
	chartCount, shapeCount := 0, 0
	for _, e := range res.Elements {
		switch e.Type {
		case "chart":
			chartCount++
		case "shape":
			shapeCount++
		}
	}
	if chartCount != 1 || shapeCount != 1 {
		t.Fatalf("expected 1 native chart + 1 background shape, got chart=%d shape=%d", chartCount, shapeCount)
	}
}

func TestNativeTableUpgrade(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720">
  <rect width="1280" height="720" fill="#FFFFFF"/>
  <g id="tbl" data-pptx-bounds="60 150 1160 500" data-pptx-replace-with="table">
    <metadata type="application/json">
      {"type":"record","headers":["A","B"],"rows":[["1","2"],["3","4"]]}
    </metadata>
    <rect x="60" y="150" width="1160" height="500" fill="#F8FAFC"/>
  </g>
</svg>`

	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	found := false
	for _, e := range res.Elements {
		if e.Type == "table" {
			found = true
			if e.Table == nil || len(e.Table.Headers) != 2 || len(e.Table.Rows) != 2 {
				t.Fatalf("table not populated correctly: %+v", e.Table)
			}
		}
	}
	if !found {
		t.Fatal("expected a native table element")
	}
}

func TestBoundsPreservedWithoutReplacement(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720">
  <g data-pptx-bounds="40 15 1200 125">
    <text x="60" y="70" font-size="36" fill="#0F172A">Title</text>
  </g>
</svg>`
	res, err := Compile(svg, Options{})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(res.Elements) != 1 {
		t.Fatalf("expected 1 text element, got %d", len(res.Elements))
	}
	if !strings.Contains(res.Elements[0].ID, "svg") {
		t.Errorf("unexpected element id %q", res.Elements[0].ID)
	}
}
