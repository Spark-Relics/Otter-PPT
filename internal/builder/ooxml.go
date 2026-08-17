package builder

import (
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// drawingTransform is the only supported representation of DrawingML position
// and size. Keeping off/ext private prevents callers from emitting them outside
// the required a:xfrm wrapper.
type drawingTransform struct {
	x, y, cx, cy int64
	rotation     int
}

func (t drawingTransform) xml() string {
	rotation := ""
	if t.rotation != 0 {
		rotation = fmt.Sprintf(` rot="%d"`, t.rotation)
	}
	return fmt.Sprintf(`<a:xfrm%s>%s</a:xfrm>`, rotation, t.childrenXML())
}

func (t drawingTransform) childrenXML() string {
	return fmt.Sprintf(
		`<a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/>`,
		t.x, t.y, t.cx, t.cy,
	)
}

func (t drawingTransform) graphicFrameXML() string {
	return `<p:xfrm>` + t.childrenXML() + `</p:xfrm>`
}

// shapeProperties writes a complete p:spPr node. A transform is mandatory, so
// shape writers cannot accidentally place a:off/a:ext directly under p:spPr.
type shapeProperties struct {
	transform drawingTransform
	geometry  string
	fill      string
	line      string
	effects   string
}

func (p shapeProperties) writeTo(buf *strings.Builder) {
	buf.WriteString(`<p:spPr>`)
	buf.WriteString(p.transform.xml())
	buf.WriteString(p.geometry)
	buf.WriteString(p.fill)
	buf.WriteString(p.line)
	buf.WriteString(p.effects)
	buf.WriteString(`</p:spPr>`)
}

func presetGeometryXML(preset string, adjustments string) string {
	if adjustments == "" {
		adjustments = `<a:avLst/>`
	}
	return fmt.Sprintf(`<a:prstGeom prst="%s">%s</a:prstGeom>`, xmlEscape(preset), adjustments)
}

func colorXMLWithOpacity(color string, opacity float64) string {
	r, g, b := hexToRGB(color)
	alpha := ""
	if opacity > 0 && opacity < 1 {
		alpha = fmt.Sprintf(`<a:alpha val="%d"/>`, int(opacity*100000))
	}
	return fmt.Sprintf(`<a:srgbClr val="%02X%02X%02X">%s</a:srgbClr>`, r, g, b, alpha)
}

// normalizeHex strips a leading "#" from a hex color string, uppercasing it.
// OOXML srgbClr val must be pure hex without the "#" prefix.
func normalizeHex(color string) string {
	return strings.ToUpper(strings.TrimPrefix(color, "#"))
}

func solidFillXML(color string) string {
	return solidFillOpacityXML(color, 0)
}

func solidFillOpacityXML(color string, opacity float64) string {
	return `<a:solidFill>` + colorXMLWithOpacity(color, opacity) + `</a:solidFill>`
}

func gradientFillXML(gradient *model.Gradient) string {
	if gradient == nil || len(gradient.Stops) == 0 {
		return `<a:noFill/>`
	}
	var buf strings.Builder
	buf.WriteString(`<a:gradFill rotWithShape="1"><a:gsLst>`)
	for _, stop := range gradient.Stops {
		fmt.Fprintf(&buf, `<a:gs pos="%d">%s</a:gs>`, int(stop.Position*1000), colorXMLWithOpacity(stop.Color, stop.Opacity))
	}
	buf.WriteString(`</a:gsLst>`)
	if gradient.Type == model.GradientRadial {
		buf.WriteString(`<a:path path="circle"><a:fillToRect l="50000" t="50000" r="50000" b="50000"/></a:path>`)
	} else {
		fmt.Fprintf(&buf, `<a:lin ang="%d" scaled="1"/>`, int(gradient.Angle*60000))
	}
	buf.WriteString(`</a:gradFill>`)
	return buf.String()
}

func fillStyleXML(fill *model.FillStyle, legacyColor string) string {
	if fill != nil {
		if fill.Gradient != nil {
			return gradientFillXML(fill.Gradient)
		}
		if fill.Color != "" {
			return solidFillOpacityXML(fill.Color, fill.Opacity)
		}
	}
	if legacyColor != "" {
		return solidFillXML(legacyColor)
	}
	return `<a:noFill/>`
}

func solidLineXML(color string, width int) string {
	return fmt.Sprintf(`<a:ln w="%d">%s</a:ln>`, width, solidFillXML(color))
}

func lineStyleXML(line *model.LineStyle, legacyColor string, legacyWidth float64) string {
	color, width, opacity, dash, beginArrow, endArrow := legacyColor, legacyWidth, 0.0, "", "", ""
	if line != nil {
		if line.Color != "" {
			color = line.Color
		}
		if line.Width > 0 {
			width = line.Width
		}
		opacity, dash, beginArrow, endArrow = line.Opacity, line.Dash, line.BeginArrow, line.EndArrow
	}
	if color == "" {
		return `<a:ln><a:noFill/></a:ln>`
	}
	if width <= 0 {
		width = 1
	}
	var buf strings.Builder
	fmt.Fprintf(&buf, `<a:ln w="%d">%s`, int(width*12700), solidFillOpacityXML(color, opacity))
	if dash != "" && dash != "solid" {
		fmt.Fprintf(&buf, `<a:prstDash val="%s"/>`, xmlEscape(dash))
	}
	if beginArrow != "" {
		fmt.Fprintf(&buf, `<a:headEnd type="%s"/>`, xmlEscape(beginArrow))
	}
	if endArrow != "" {
		fmt.Fprintf(&buf, `<a:tailEnd type="%s"/>`, xmlEscape(endArrow))
	}
	buf.WriteString(`</a:ln>`)
	return buf.String()
}

// effectsXML generates an <a:effectLst> containing glow, shadow, and reflection effects.
func effectsXML(shadow *model.ShadowStyle, glow *model.GlowStyle, reflection *model.ReflectionStyle) string {
	var inner strings.Builder

	// Glow must come before shadow in effectLst
	if glow != nil {
		color := glow.Color
		if color == "" {
			color = "6366F1"
		}
		opacity := glow.Opacity
		if opacity == 0 {
			opacity = 0.5
		}
		radius := int(glow.Radius * 12700)
		if radius == 0 {
			radius = 63500
		}
		fmt.Fprintf(&inner, `<a:glow rad="%d">%s</a:glow>`, radius, colorXMLWithOpacity(color, opacity))
	}

	// Inner shadow
	if shadow != nil {
		color := shadow.Color
		if color == "" {
			color = "000000"
		}
		opacity := shadow.Opacity
		if opacity == 0 {
			opacity = 0.35
		}
		blur, distance := int(shadow.Blur*12700), int(shadow.Distance*12700)
		if blur == 0 {
			blur = 63500
		}
		if distance == 0 {
			distance = 25400
		}
		fmt.Fprintf(&inner, `<a:outerShdw blurRad="%d" dist="%d" dir="%d" algn="ctr" rotWithShape="0">%s</a:outerShdw>`,
			blur, distance, int(shadow.Angle*60000), colorXMLWithOpacity(color, opacity))
	}

	// Reflection
	if reflection != nil {
		opacity := reflection.Opacity
		if opacity == 0 {
			opacity = 0.5
		}
		blur := int(reflection.Blur * 12700)
		if blur == 0 {
			blur = 25400
		}
		dist := int(reflection.Distance * 12700)
		if dist == 0 {
			dist = 38100
		}
		// sx (horizontal scaling), sy (vertical), dir=5400000 (downward mirror)
		fmt.Fprintf(&inner, `<a:reflection blurRad="%d" stA="%d" stPos="0" dist="%d" dir="5400000" fadeDir="5400000" sy="-1000000" algn="b" rotWithShape="0"/>`,
			blur, int(opacity*100000), dist)
	}

	if inner.Len() == 0 {
		return ""
	}
	return fmt.Sprintf(`<a:effectLst>%s</a:effectLst>`, inner.String())
}
