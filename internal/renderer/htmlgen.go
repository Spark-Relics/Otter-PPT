// htmlgen.go generates a self-contained HTML document from the presentation
// model. Each slide becomes an absolutely-positioned page; elements are
// rendered as divs/SVG with percentage coordinates matching the PPTX layout.
// The HTML is then screenshotted by a headless browser (browser.go), giving
// a zero-download rendering path when LibreOffice is unavailable.
//
// Design note: unlike OOXML-parsing renderers, we generate from our own
// model, so no XML round-trip is needed — positions, styles, and media are
// already structured data.
package renderer

import (
	"encoding/base64"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

const (
	htmlDPI      = 96.0
	ptPerInch    = 72.0
	defaultSlideW = 13.333
	defaultSlideH = 7.5
)

// htmlGenerator accumulates the HTML document and per-slide anchors.
type htmlGenerator struct {
	pres      *model.Presentation
	slideWIn  float64
	slideHIn  float64
	imgCache  map[string]string // path → data URI
}

// GenerateHTML renders the presentation as one self-contained HTML file.
// Slides are stacked vertically, each exactly viewport-sized, so a
// headless --screenshot with the right window size captures one slide.
func GenerateHTML(pres *model.Presentation, outPath string) error {
	g := &htmlGenerator{
		pres:     pres,
		imgCache: map[string]string{},
	}
	g.slideWIn, g.slideHIn = defaultSlideW, defaultSlideH
	if pres.SlideWidth > 0 && pres.SlideHeight > 0 {
		g.slideWIn, g.slideHIn = pres.SlideWidth, pres.SlideHeight
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"UTF-8\">\n")
	sb.WriteString("<style>\n")
	sb.WriteString(g.css())
	sb.WriteString("</style>\n</head>\n<body>\n")

	for _, slide := range pres.Slides {
		sb.WriteString(slideBackgroundStyle(g, slide))
		sb.WriteString(g.slideHTML(slide))
	}

	sb.WriteString("</body>\n</html>\n")
	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

func (g *htmlGenerator) css() string {
	return fmt.Sprintf(`
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body { background: #333; }
.slide {
  position: relative;
  width: %.2fin;
  height: %.2fin;
  overflow: hidden;
  margin: 0 auto;
  page-break-after: always;
  background: #fff;
}
.slide + .slide { margin-top: 24px; }
.el {
  position: absolute;
}
.text-el {
  display: flex;
  flex-direction: column;
}
`, g.slideWIn, g.slideHIn)
}

// slideHTML renders one slide (full-document mode).
func (g *htmlGenerator) slideHTML(slide *model.Slide) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"slide\">\n")
	for _, elem := range slide.Elements {
		sb.WriteString(g.elementHTML(elem))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// slideBody renders the slide's inner elements (no wrapper), shared by the
// full-document and single-slide-screenshot paths.
func (g *htmlGenerator) slideBody(slide *model.Slide) string {
	var sb strings.Builder
	for _, elem := range slide.Elements {
		sb.WriteString(g.elementHTML(elem))
	}
	return sb.String()
}

// elementHTML dispatches on element type.
func (g *htmlGenerator) elementHTML(elem *model.Element) string {
	switch elem.Type {
	case model.ElementTitle, model.ElementSubtitle, model.ElementBody, model.ElementBullet:
		return g.textElementHTML(elem)
	case model.ElementShape:
		return g.shapeElementHTML(elem)
	case model.ElementImage:
		return g.imageElementHTML(elem)
	case model.ElementTable:
		return g.tableElementHTML(elem)
	case model.ElementChart:
		return g.chartElementHTML(elem)
	case model.ElementConnector:
		return g.connectorElementHTML(elem)
	case model.ElementGroup:
		// Groups are flattened by the builder; children render independently.
		return ""
	case model.ElementVideo, model.ElementAudio:
		return g.mediaElementHTML(elem)
	case model.ElementIcon:
		return g.iconElementHTML(elem)
	default:
		return g.textElementHTML(elem)
	}
}

// posStyle builds the absolute-position CSS fragment for an element rect.
func posStyle(r model.Rect) string {
	return fmt.Sprintf("left:%.4f%%;top:%.4f%%;width:%.4f%%;height:%.4f%%;",
		r.X, r.Y, r.W, r.H)
}

// rotationStyle converts an OOXML-style rotation (degrees, clockwise) to CSS.
func rotationStyle(deg float64) string {
	if deg == 0 {
		return ""
	}
	return fmt.Sprintf("transform:rotate(%.2fdeg);", -deg)
}

// textElementHTML renders text, bullet lists, and rich-text paragraphs.
func (g *htmlGenerator) textElementHTML(elem *model.Element) string {
	style := g.textStyleCSS(elem.Style, elem.Type)
	inner := g.textContentHTML(elem)

	align := cssTextAlign(elem.Style.Align)
	valign := cssVAlign(elem.Style.VAlign, elem.Type)

	return fmt.Sprintf(
		"<div class=\"el text-el\" style=\"%s%s%sjustify-content:%s;\">%s</div>\n",
		posStyle(elem.Rect), rotationStyle(elem.Rotation), style, valign, inner+align)
}

// textContentHTML renders paragraphs / plain text / bullet items.
func (g *htmlGenerator) textContentHTML(elem *model.Element) string {
	var sb strings.Builder

	if len(elem.Paragraphs) > 0 {
		for _, p := range elem.Paragraphs {
			pStyle := g.textStyleCSS(mergeStyle(elem.Style, p.Style), "")
			align := cssTextAlign(firstNonEmpty(p.Style.Align, elem.Style.Align))
			sb.WriteString(fmt.Sprintf("<p style=\"%s%s\">", pStyle, align))
			if p.Bullet != "" || p.Level > 0 {
				sb.WriteString("<span style=\"margin-right:0.4em;\">" + html.EscapeString(p.Bullet) + "</span>")
			}
			if len(p.Runs) > 0 {
				for _, run := range p.Runs {
					rs := g.textStyleCSS(mergeStyle(elem.Style, p.Style, run.Style), "")
					sb.WriteString(fmt.Sprintf("<span style=\"%s\">%s</span>", rs, html.EscapeString(run.Text)))
				}
			} else {
				sb.WriteString(html.EscapeString(p.Text))
			}
			sb.WriteString("</p>")
		}
		return sb.String()
	}

	if len(elem.Items) > 0 {
		itemStyle := g.textStyleCSS(elem.Style, "")
		for _, item := range elem.Items {
			bullet := elem.Style.BulletChar
			if bullet == "" {
				bullet = "•"
			}
			sb.WriteString(fmt.Sprintf(
				"<p style=\"%s\"><span style=\"margin-right:0.4em;\">%s</span>%s</p>",
				itemStyle, html.EscapeString(bullet), html.EscapeString(item)))
		}
		return sb.String()
	}

	return fmt.Sprintf("<p style=\"%s%s\">%s</p>",
		g.textStyleCSS(elem.Style, ""), "", html.EscapeString(elem.Text))
}

// shapeElementHTML renders shapes (rect, ellipse, rounded, etc.).
func (g *htmlGenerator) shapeElementHTML(elem *model.Element) string {
	if elem.Shape == nil {
		return ""
	}
	sd := elem.Shape

	// Lines render better as SVG for arrowheads.
	if sd.ShapeType == model.ShapeLine || sd.ShapeType == model.ShapeArrow || sd.ShapeType == model.ShapeDoubleArrow {
		return g.lineShapeSVG(elem)
	}

	style := posStyle(elem.Rect) + rotationStyle(elem.Rotation)

	// Fill
	fillColor := sd.FillColor
	var fillOpacity float64 = 1
	if sd.Fill != nil {
		if sd.Fill.Gradient != nil {
			style += "background:" + cssGradient(sd.Fill.Gradient) + ";"
		} else {
			fillColor = firstNonEmpty(sd.Fill.Color, fillColor)
			if sd.Fill.Opacity > 0 {
				fillOpacity = sd.Fill.Opacity
			}
		}
	}
	if fillColor != "" {
		style += "background:" + cssRGBA(fillColor, fillOpacity) + ";"
	}

	// Border
	lineColor, lineWidth := sd.BorderColor, sd.BorderWidth
	if sd.Line != nil {
		lineColor = firstNonEmpty(sd.Line.Color, lineColor)
		if sd.Line.Width > 0 {
			lineWidth = sd.Line.Width
		}
	}
	if lineColor != "" {
		style += fmt.Sprintf("border:%.2fpt solid %s;", math.Max(lineWidth, 0.25), cssColor(lineColor))
	}

	// Radius
	if sd.ShapeType == model.ShapeRoundedRectangle || sd.CornerRadius > 0 {
		r := sd.CornerRadius
		if r <= 0 {
			r = 0.1
		}
		style += fmt.Sprintf("border-radius:%.1f%%;", math.Min(r*100, 50))
	}

	// Shadow
	if sd.Shadow != nil {
		style += "box-shadow:" + cssShadow(sd.Shadow) + ";"
	}

	// Non-rect geometry via clip-path or border-radius
	switch sd.ShapeType {
	case model.ShapeEllipse:
		style += "border-radius:50%;"
	case model.ShapeTriangle:
		style += "clip-path:polygon(50% 0,100% 100%,0 100%);"
	case model.ShapeDiamond:
		style += "clip-path:polygon(50% 0,100% 50%,50% 100%,0 50%);"
	case model.ShapePentagon:
		style += "clip-path:polygon(50% 0,100% 38%,82% 100%,18% 100%,0 38%);"
	case model.ShapeHexagon:
		style += "clip-path:polygon(25% 0,75% 0,100% 50%,75% 100%,25% 100%,0 50%);"
	case model.ShapeStar:
		style += "clip-path:polygon(50% 0,61% 35%,98% 35%,68% 57%,79% 91%,50% 70%,21% 91%,32% 57%,2% 35%,39% 35%);"
	}

	opacityCSS := ""
	if elem.Opacity > 0 && elem.Opacity < 1 {
		opacityCSS = fmt.Sprintf("opacity:%.2f;", elem.Opacity)
	}

	inner := ""
	if sd.Text != "" {
		ts := g.textStyleCSS(mergeStyle(sd.Style, elem.Style), "")
		inner = fmt.Sprintf(
			"<div style=\"display:flex;width:100%%;height:100%%;align-items:center;justify-content:center;\"><p style=\"%s%s\">%s</p></div>",
			ts, cssTextAlign(sd.Style.Align), html.EscapeString(sd.Text))
	}

	return fmt.Sprintf("<div class=\"el\" style=\"%s%s%s\">%s</div>\n", style, opacityCSS, "", inner)
}

// lineShapeSVG renders line/arrow shapes as inline SVG with arrowheads.
func (g *htmlGenerator) lineShapeSVG(elem *model.Element) string {
	sd := elem.Shape
	w := elem.Rect.W / 100 * g.slideWIn * htmlDPI
	h := elem.Rect.H / 100 * g.slideHIn * htmlDPI
	x2, y2 := w, h
	// A flat rect with positive width and zero-ish height is a horizontal line.
	if h < 1 {
		y2 = 0
	}
	if w < 1 {
		x2 = 0
	}

	color := firstNonEmpty(sd.Line.Color, sd.BorderColor, "#333333")
	strokeW := sd.BorderWidth
	if sd.Line != nil && sd.Line.Width > 0 {
		strokeW = sd.Line.Width
	}
	dash := ""
	if sd.Line != nil {
		dash = cssDashArray(sd.Line.Dash, strokeW)
	}

	startArrow := ""
	endArrow := ""
	if sd.ShapeType == model.ShapeArrow || sd.ShapeType == model.ShapeDoubleArrow {
		endArrow = arrowMarkerDef("arrowEnd", color)
	}
	if sd.ShapeType == model.ShapeDoubleArrow {
		startArrow = arrowMarkerDef("arrowStart", color)
	}

	uid := elem.ID
	svg := fmt.Sprintf(
		`<svg class="el" style="%s%s" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">`+
			`<defs>%s%s</defs>`+
			`<line x1="0" y1="0" x2="%.0f" y2="%.0f" stroke="%s" stroke-width="%.2f"%s marker-end="url(#%s-end)" marker-start="url(#%s-start)"/></svg>`,
		posStyle(elem.Rect), rotationStyle(elem.Rotation), math.Max(w, 1), math.Max(h, 1), math.Max(w, 1), math.Max(h, 1),
		endArrow, startArrow,
		x2, y2, cssColor(color), math.Max(strokeW, 0.5), dashAttr(dash), uid, uid)

	return svg + "\n"
}

func arrowMarkerDef(uid, color string) string {
	return fmt.Sprintf(`<marker id="%s-end" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 z" fill="%s"/></marker>`, uid, cssColor(color))
}

func dashAttr(dash string) string {
	if dash == "" {
		return ""
	}
	return fmt.Sprintf(` stroke-dasharray="%s"`, dash)
}

func cssDashArray(dash string, width float64) string {
	switch dash {
	case "dash":
		return fmt.Sprintf("%.1f,%.1f", width*3, width*2)
	case "dot":
		return fmt.Sprintf("%.1f,%.1f", width, width*1.5)
	case "dash_dot":
		return fmt.Sprintf("%.1f,%.1f,%.1f,%.1f", width*3, width*2, width, width*2)
	default:
		return ""
	}
}

// imageElementHTML renders images as data URIs with fit modes.
func (g *htmlGenerator) imageElementHTML(elem *model.Element) string {
	uri := g.dataURI(elem.ImagePath)
	if uri == "" {
		// Placeholder for missing images
		return fmt.Sprintf(
			"<div class=\"el\" style=\"%sbackground:#eee;display:flex;align-items:center;justify-content:center;color:#999;font-size:11pt;border:1px dashed #ccc;\">%s</div>\n",
			posStyle(elem.Rect), html.EscapeString(firstNonEmpty(elem.ImageAlt, "[image]")))
	}

	fit := elem.ImageFit
	objFit := "contain"
	switch fit {
	case "cover":
		objFit = "cover"
	case "stretch":
		objFit = "fill"
	}

	radius := ""
	if elem.ImageRadius > 0 {
		radius = fmt.Sprintf("border-radius:%.1f%%;", math.Min(elem.ImageRadius*100, 50))
	}

	clip := ""
	if elem.ImageCrop != nil {
		c := elem.ImageCrop
		// CSS crop via a scale transform on an oversized image inside overflow:hidden
		inv := func(v float64) float64 { return math.Max(1-v, 0.0001) }
		sw := inv(c.Left + c.Right)
		sh := inv(c.Top + c.Bottom)
		clip = fmt.Sprintf(
			"object-fit:fill;transform:scale(%.4f,%.4f);transform-origin:0 0;object-position:%.2f%% %.2f%%;",
			1/sw, 1/sh, c.Left*100/(sw*100)*100, c.Top*100/(sh*100)*100)
		// Simplified: rely on object-fit cover + clip-path approximation
		clip = fmt.Sprintf("clip-path:inset(%.2f%% %.2f%% %.2f%% %.2f%%);", c.Top*100, c.Right*100, c.Bottom*100, c.Left*100)
		_ = objFit
		objFit = "cover"
	}

	return fmt.Sprintf(
		"<div class=\"el\" style=\"%s%s%soverflow:hidden;\"><img src=\"%s\" style=\"width:100%%;height:100%%;object-fit:%s;\"/></div>\n",
		posStyle(elem.Rect), rotationStyle(elem.Rotation), radius+clip, uri, objFit)
}

// tableElementHTML renders tables.
func (g *htmlGenerator) tableElementHTML(elem *model.Element) string {
	if elem.Table == nil {
		return ""
	}
	td := elem.Table

	var sb strings.Builder
	fontSize := td.FontSize
	if fontSize == 0 {
		fontSize = 12
	}
	headerColor := firstNonEmpty(td.HeaderColor, "#2c3e50")
	borderColor := firstNonEmpty(td.BorderColor, "#bbbbbb")

	sb.WriteString(fmt.Sprintf(
		"<div class=\"el\" style=\"%s\"><table style=\"width:100%%;height:100%%;border-collapse:collapse;font-size:%dpt;\">",
		posStyle(elem.Rect), fontSize))

	// Header row
	sb.WriteString(fmt.Sprintf("<tr style=\"background:%s;\">", cssColor(headerColor)))
	for _, c := range td.Headers {
		sb.WriteString(g.tableCellHTML(c, true, borderColor))
	}
	sb.WriteString("</tr>")

	for ri, row := range td.Rows {
		bg := ""
		if td.AltRowColor != "" && ri%2 == 1 {
			bg = fmt.Sprintf(" background:%s;", cssColor(td.AltRowColor))
		}
		sb.WriteString(fmt.Sprintf("<tr style=\"%s\">", bg))
		for _, c := range row {
			sb.WriteString(g.tableCellHTML(c, false, borderColor))
		}
		sb.WriteString("</tr>")
	}

	sb.WriteString("</table></div>\n")
	return sb.String()
}

func (g *htmlGenerator) tableCellHTML(c model.TableCell, header bool, borderColor string) string {
	style := fmt.Sprintf("border:0.5pt solid %s;padding:0.3em 0.5em;", cssColor(borderColor))
	if c.Style.BgColor != "" && !header {
		style += "background:" + cssColor(c.Style.BgColor) + ";"
	}
	color := c.Style.Color
	if color == "" && header {
		color = "#ffffff"
	}
	if color != "" {
		style += "color:" + cssColor(color) + ";"
	}
	if c.Style.Bold || header {
		style += "font-weight:bold;"
	}
	align := cssTextAlign(c.Style.Align)
	if header && align == "" {
		align = "text-align:center;"
	}
	return fmt.Sprintf("<td style=\"%s%s\"%s>%s</td>",
		style, align, spanAttrs(c), html.EscapeString(c.Text))
}

func spanAttrs(c model.TableCell) string {
	s := ""
	if c.ColSpan > 1 {
		s += fmt.Sprintf(" colspan=\"%d\"", c.ColSpan)
	}
	if c.RowSpan > 1 {
		s += fmt.Sprintf(" rowspan=\"%d\"", c.RowSpan)
	}
	return s
}

// chartElementHTML renders charts as inline SVG.
func (g *htmlGenerator) chartElementHTML(elem *model.Element) string {
	if elem.Chart == nil || len(elem.Chart.Series) == 0 {
		return ""
	}
	cd := elem.Chart
	w := elem.Rect.W / 100 * g.slideWIn * htmlDPI
	h := elem.Rect.H / 100 * g.slideHIn * htmlDPI

	svg := g.chartSVG(cd, w, h)
	return fmt.Sprintf("<div class=\"el\" style=\"%s%s\">%s</div>\n",
		posStyle(elem.Rect), rotationStyle(elem.Rotation), svg)
}

// connectorElementHTML renders connectors as SVG lines.
func (g *htmlGenerator) connectorElementHTML(elem *model.Element) string {
	if elem.Connector == nil {
		return ""
	}
	cd := elem.Connector

	// Bounding box from start/end points
	x1, y1, x2, y2 := cd.StartX, cd.StartY, cd.EndX, cd.EndY
	minX, minY := math.Min(x1, x2), math.Min(y1, y2)
	maxX, maxY := math.Max(x1, x2), math.Max(y1, y2)
	bw := (maxX - minX) / 100 * g.slideWIn * htmlDPI
	bh := (maxY - minY) / 100 * g.slideHIn * htmlDPI
	lx := (x1 - minX) / math.Max(maxX-minX, 0.0001) * bw
	ly := (y1 - minY) / math.Max(maxY-minY, 0.0001) * bh
	ex := (x2 - minX) / math.Max(maxX-minX, 0.0001) * bw
	ey := (y2 - minY) / math.Max(maxY-minY, 0.0001) * bh

	color := cssColor(firstNonEmpty(cd.Color, "#333333"))
	uid := "cn" + elem.ID
	marker := ""
	if cd.ConnectorType == model.ShapeArrow || cd.ConnectorType == model.ShapeDoubleArrow {
		marker = ` marker-end="url(#` + uid + `-end)"`
	}
	markerDef := ""
	if marker != "" {
		markerDef = fmt.Sprintf(`<defs><marker id="%s-end" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 z" fill="%s"/></marker></defs>`, uid, color)
	}

	return fmt.Sprintf(
		`<svg class="el" style="left:%.4f%%;top:%.4f%%;width:%.2fpx;height:%.2fpx;" width="%.2f" height="%.2f" xmlns="http://www.w3.org/2000/svg">%s<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.2f"%s/></svg>`,
		minX, minY, math.Max(bw, 1), math.Max(bh, 1), math.Max(bw, 1), math.Max(bh, 1),
		markerDef, lx, ly, ex, ey, color, math.Max(cd.Width, 0.75), marker) + "\n"
}

// mediaElementHTML renders video/audio placeholders with poster frames.
func (g *htmlGenerator) mediaElementHTML(elem *model.Element) string {
	label := "▶ Video"
	if elem.Type == model.ElementAudio {
		label = "♪ Audio"
	}
	poster := ""
	if elem.Media != nil && elem.Media.PosterPath != "" {
		if uri := g.dataURI(elem.Media.PosterPath); uri != "" {
			poster = fmt.Sprintf("<img src=\"%s\" style=\"width:100%%;height:100%%;object-fit:cover;opacity:0.85;\"/>", uri)
		}
	}
	return fmt.Sprintf(
		"<div class=\"el\" style=\"%sbackground:#111;display:flex;align-items:center;justify-content:center;overflow:hidden;\">%s<span style=\"position:absolute;color:#fff;font-size:14pt;text-shadow:0 1px 3px #000;\">%s</span></div>\n",
		posStyle(elem.Rect), poster, label)
}

// iconElementHTML renders icon placeholders.
func (g *htmlGenerator) iconElementHTML(elem *model.Element) string {
	return fmt.Sprintf(
		"<div class=\"el\" style=\"%s%sdisplay:flex;align-items:center;justify-content:center;font-size:%dpt;color:%s;\">★</div>\n",
		posStyle(elem.Rect), rotationStyle(elem.Rotation),
		firstPositive(elem.Style.FontSize, 24), cssColor(firstNonEmpty(elem.Style.Color, "#666666")))
}

// ──────────── Text style helpers ────────────

// mergeStyle overlays non-zero fields of extra styles onto base.
func mergeStyle(base model.TextStyle, extras ...model.TextStyle) model.TextStyle {
	out := base
	for _, e := range extras {
		if e.FontSize > 0 {
			out.FontSize = e.FontSize
		}
		if e.FontName != "" {
			out.FontName = e.FontName
		}
		if e.Bold {
			out.Bold = true
		}
		if e.Italic {
			out.Italic = true
		}
		if e.Underline {
			out.Underline = true
		}
		if e.Strike {
			out.Strike = true
		}
		if e.Color != "" {
			out.Color = e.Color
		}
		if e.Opacity > 0 {
			out.Opacity = e.Opacity
		}
		if e.Align != "" {
			out.Align = e.Align
		}
		if e.LineSpacing > 0 {
			out.LineSpacing = e.LineSpacing
		}
		if e.LetterSpacing > 0 {
			out.LetterSpacing = e.LetterSpacing
		}
		if e.VAlign != "" {
			out.VAlign = e.VAlign
		}
		if e.Shadow {
			out.Shadow = true
		}
	}
	return out
}

// textStyleCSS converts a TextStyle to CSS declarations.
func (g *htmlGenerator) textStyleCSS(ts model.TextStyle, elemType model.ElementType) string {
	var b strings.Builder

	if ts.FontSize > 0 {
		b.WriteString(fmt.Sprintf("font-size:%dpt;", ts.FontSize))
	} else if elemType == model.ElementTitle {
		b.WriteString("font-size:36pt;font-weight:bold;")
	} else if elemType == model.ElementSubtitle {
		b.WriteString("font-size:20pt;")
	}

	fontName := ts.FontName
	if fontName == "" {
		if elemType == model.ElementTitle && g.pres.Theme.TitleFont != "" {
			fontName = g.pres.Theme.TitleFont
		} else if g.pres.Theme.BodyFont != "" {
			fontName = g.pres.Theme.BodyFont
		}
	}
	if fontName != "" {
		b.WriteString(fmt.Sprintf("font-family:%s,%s;", cssQuoteFont(fontName), cjkFallbackChain(fontName)))
	} else {
		b.WriteString(fmt.Sprintf("font-family:%s;", cjkFallbackChain("")))
	}

	color := ts.Color
	if color == "" {
		if elemType == model.ElementTitle && g.pres.Theme.PrimaryColor != "" && isLight(g.pres.Theme.BackgroundColor) {
			color = g.pres.Theme.PrimaryColor
		} else if g.pres.Theme.TextColor != "" {
			color = g.pres.Theme.TextColor
		}
	}
	if color != "" {
		b.WriteString("color:" + cssColor(color) + ";")
	}
	if ts.Bold {
		b.WriteString("font-weight:bold;")
	}
	if ts.Italic {
		b.WriteString("font-style:italic;")
	}
	if ts.Underline {
		b.WriteString("text-decoration:underline;")
	}
	if ts.Strike {
		b.WriteString("text-decoration:line-through;")
	}
	if ts.LineSpacing > 0 {
		b.WriteString(fmt.Sprintf("line-height:%.2f;", ts.LineSpacing))
	}
	if ts.LetterSpacing > 0 {
		b.WriteString(fmt.Sprintf("letter-spacing:%.2fpt;", ts.LetterSpacing))
	}
	if ts.Shadow {
		b.WriteString("text-shadow:1pt 1pt 3pt rgba(0,0,0,0.45);")
	}
	if ts.Opacity > 0 && ts.Opacity < 1 {
		b.WriteString(fmt.Sprintf("opacity:%.2f;", ts.Opacity))
	}

	// Text insets
	if ts.MarginLeft != 0 || ts.MarginTop != 0 || ts.MarginRight != 0 || ts.MarginBottom != 0 {
		b.WriteString(fmt.Sprintf("padding:%.1fpt %.1fpt %.1fpt %.1fpt;",
			ts.MarginTop, ts.MarginRight, ts.MarginBottom, ts.MarginLeft))
	}

	return b.String()
}

// cjkFallbackChain builds a CSS font-family fallback so CJK glyphs render
// on any OS even when the declared font is missing.
func cjkFallbackChain(primary string) string {
	if primary != "" && isCjkFont(primary) {
		return "'PingFang SC','Hiragino Sans','Yu Gothic','Apple SD Gothic Neo','Microsoft YaHei','Noto Sans CJK SC',sans-serif"
	}
	return "'Microsoft YaHei','PingFang SC','Noto Sans CJK SC','Hiragino Sans',sans-serif"
}

func isCjkFont(name string) bool {
	n := strings.ToLower(name)
	for _, k := range []string{"yahei", "pingfang", "noto sans cjk", "noto serif", "songti", "simhei", "simsun", "gothic", "mincho"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func cssQuoteFont(name string) string {
	if strings.ContainsAny(name, " ") && !strings.HasPrefix(name, "'") {
		return "'" + name + "'"
	}
	return name
}

func cssTextAlign(align string) string {
	switch strings.ToLower(align) {
	case "left":
		return "text-align:left;"
	case "center", "middle":
		return "text-align:center;"
	case "right":
		return "text-align:right;"
	case "justify":
		return "text-align:justify;"
	default:
		return ""
	}
}

func cssVAlign(valign string, elemType model.ElementType) string {
	switch strings.ToLower(valign) {
	case "top":
		return "flex-start"
	case "middle", "center":
		return "center"
	case "bottom":
		return "flex-end"
	default:
		if elemType == model.ElementTitle || elemType == model.ElementSubtitle {
			return "center"
		}
		return "flex-start"
	}
}

// ──────────── Color / fill helpers ────────────

func cssColor(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "#000000"
	}
	if strings.HasPrefix(c, "#") && len(c) == 4 {
		// Expand 3-digit hex (#abc → #aabbcc)
		c = "#" + strings.Repeat(string(c[1]), 2) + strings.Repeat(string(c[2]), 2) + strings.Repeat(string(c[3]), 2)
	}
	if !strings.HasPrefix(c, "#") && !strings.HasPrefix(c, "rgb") {
		c = "#" + c
	}
	return c
}

// cssRGBA converts hex + opacity to rgba() notation.
func cssRGBA(c string, opacity float64) string {
	if opacity >= 1 || opacity <= 0 {
		return cssColor(c)
	}
	r, g, b, _ := parseHexRGB(c)
	return fmt.Sprintf("rgba(%d,%d,%d,%.2f)", r, g, b, opacity)
}

func parseHexRGB(c string) (int, int, int, bool) {
	c = strings.TrimPrefix(strings.TrimSpace(c), "#")
	if len(c) == 3 {
		c = string([]byte{c[0], c[0], c[1], c[1], c[2], c[2]})
	}
	if len(c) != 6 {
		return 0, 0, 0, false
	}
	var r, g, b int
	fmt.Sscanf(c, "%02x%02x%02x", &r, &g, &b)
	return r, g, b, true
}

// isLight reports whether a hex background is light (for default title color).
func isLight(hex string) bool {
	r, g, b, ok := parseHexRGB(hex)
	if !ok {
		return true
	}
	return 0.299*float64(r)+0.587*float64(g)+0.114*float64(b) > 140
}

func cssGradient(grad *model.Gradient) string {
	var stops []string
	for _, s := range grad.Stops {
		pos := s.Position
		if pos > 1 && pos <= 100 {
			pos = pos / 100
		}
		stops = append(stops, fmt.Sprintf("%s %.1f%%", cssRGBA(s.Color, s.Opacity), pos*100))
	}
	shape := "linear-gradient"
	dir := "180deg"
	if grad.Type == model.GradientRadial {
		shape = "radial-gradient"
		dir = "circle"
	} else if grad.Angle != 0 {
		// OOXML angle: 0 = left→right, clockwise. CSS: 0deg = bottom→top.
		dir = fmt.Sprintf("%.0fdeg", grad.Angle)
	}
	return fmt.Sprintf("%s(%s, %s)", shape, dir, strings.Join(stops, ", "))
}

func cssShadow(s *model.ShadowStyle) string {
	color := firstNonEmpty(s.Color, "#000000")
	opacity := s.Opacity
	if opacity <= 0 {
		opacity = 0.4
	}
	dist := s.Distance
	if dist == 0 {
		dist = 2
	}
	blur := s.Blur
	if blur == 0 {
		blur = 4
	}
	angleRad := s.Angle * math.Pi / 180
	dx := -dist * math.Sin(angleRad)
	dy := dist * math.Cos(angleRad)
	return fmt.Sprintf("%.1fpt %.1fpt %.1fpt %s", dx, dy, blur, cssRGBA(color, opacity))
}

// ──────────── Media helpers ────────────

// dataURI loads an image and returns a data: URI, with caching.
func (g *htmlGenerator) dataURI(path string) string {
	if path == "" {
		return ""
	}
	if uri, ok := g.imgCache[path]; ok {
		return uri
	}
	uri := buildDataURI(path)
	g.imgCache[path] = uri
	return uri
}

func buildDataURI(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	var mime string
	switch ext {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".svg":
		mime = "image/svg+xml"
	case ".webp":
		mime = "image/webp"
	case ".bmp":
		mime = "image/bmp"
	default:
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// Remote URLs are used directly (requires network in browser).
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return path
		}
		return ""
	}
	if len(data) > 8<<20 {
		return ""
	}
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstPositive(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
