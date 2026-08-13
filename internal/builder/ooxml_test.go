package builder

import (
	"strings"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func TestShapePropertiesAlwaysWrapTransform(t *testing.T) {
	var output strings.Builder
	shapeProperties{
		transform: drawingTransform{x: 1, y: 2, cx: 3, cy: 4, rotation: 60000},
		geometry:  presetGeometryXML("rect", ""),
		fill:      solidFillXML("#112233"),
	}.writeTo(&output)

	got := output.String()
	wantTransform := `<p:spPr><a:xfrm rot="60000"><a:off x="1" y="2"/><a:ext cx="3" cy="4"/></a:xfrm>`
	if !strings.HasPrefix(got, wantTransform) {
		t.Fatalf("invalid shape property structure: %s", got)
	}
	if strings.Contains(got, `<p:spPr><a:off`) {
		t.Fatalf("unwrapped transform children: %s", got)
	}
}

func TestAdvancedShapeStyles(t *testing.T) {
	fill := fillStyleXML(&model.FillStyle{Gradient: &model.Gradient{Type: model.GradientLinear, Angle: 45, Stops: []model.GradientStop{{Color: "#112233", Position: 0, Opacity: 0.5}, {Color: "#445566", Position: 100}}}}, "")
	if !strings.Contains(fill, `<a:gradFill`) || !strings.Contains(fill, `<a:alpha val="50000"/>`) || !strings.Contains(fill, `<a:lin ang="2700000"`) {
		t.Fatalf("advanced gradient missing expected OOXML: %s", fill)
	}
	line := lineStyleXML(&model.LineStyle{Color: "#ABCDEF", Width: 2, Dash: "dash", EndArrow: "triangle"}, "", 0)
	if !strings.Contains(line, `<a:prstDash val="dash"/>`) || !strings.Contains(line, `<a:tailEnd type="triangle"/>`) {
		t.Fatalf("advanced line missing expected OOXML: %s", line)
	}
	if shadow := shadowXML(&model.ShadowStyle{Color: "#000000", Opacity: 0.25, Blur: 4, Distance: 2, Angle: 45}); !strings.Contains(shadow, `<a:outerShdw`) {
		t.Fatalf("shadow missing expected OOXML: %s", shadow)
	}
}

func TestGraphicFrameTransformUsesPresentationNamespace(t *testing.T) {
	got := (drawingTransform{x: 1, y: 2, cx: 3, cy: 4}).graphicFrameXML()
	want := `<p:xfrm><a:off x="1" y="2"/><a:ext cx="3" cy="4"/></p:xfrm>`
	if got != want {
		t.Fatalf("graphic frame transform = %q, want %q", got, want)
	}
}
