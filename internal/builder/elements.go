package builder

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// writeElement dispatches to the correct writer based on element type.
func (b *Builder) writeElement(buf *strings.Builder, elem *model.Element) {
	switch elem.Type {
	case model.ElementTitle, model.ElementSubtitle, model.ElementBody:
		b.writeTextBox(buf, elem)
	case model.ElementBullet:
		b.writeBulletList(buf, elem)
	case model.ElementImage:
		b.writeImage(buf, elem)
	case model.ElementShape:
		b.writeShape(buf, elem)
	case model.ElementTable:
		b.writeTable(buf, elem)
	case model.ElementChart:
		b.writeChart(buf, elem)
	case model.ElementConnector:
		b.writeConnector(buf, elem)
	case model.ElementVideo:
		b.writeVideo(buf, elem)
	case model.ElementAudio:
		b.writeAudio(buf, elem)
	case model.ElementGroup:
		// Groups are logical; children are rendered independently
	default:
		b.writeTextBox(buf, elem)
	}
}

// ooxmlObjectID returns a stable positive numeric ID required by cNvPr.
func ooxmlObjectID(id string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return h.Sum32()%2147483646 + 2
}

// transform returns the position, size, and rotation for a DrawingML shape.
func (b *Builder) transform(elem *model.Element) drawingTransform {
	return drawingTransform{
		x:        pctToEMU(elem.Rect.X, b.pres.SlideWidth),
		y:        pctToEMU(elem.Rect.Y, b.pres.SlideHeight),
		cx:       pctToEMU(elem.Rect.W, b.pres.SlideWidth),
		cy:       pctToEMU(elem.Rect.H, b.pres.SlideHeight),
		rotation: int(elem.Rotation * 60000),
	}
}

// writeTextBox writes a plain text box shape (sp).
func (b *Builder) writeTextBox(buf *strings.Builder, elem *model.Element) {
	transform := b.transform(elem)
	style := b.textStyle(elem.Type, elem.Style)
	fontSize := style.FontSize
	align := alignXML(style.Align)
	if align == "" {
		align = "l"
	}

	buf.WriteString(fmt.Sprintf(
		`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>`,
		ooxmlObjectID(elem.ID), xmlEscape(elem.ID)))
	shapeProperties{
		transform: transform,
		geometry:  presetGeometryXML("rect", ""),
	}.writeTo(buf)

	buf.WriteString(`<p:txBody>` + textBodyPropertiesXML(style) + `<a:lstStyle/>`)
	if len(elem.Paragraphs) > 0 {
		for _, paragraph := range elem.Paragraphs {
			b.writeParagraph(buf, paragraph, style)
		}
	} else {
		buf.WriteString(fmt.Sprintf(`<a:p><a:pPr algn="%s">`, align))
		if style.LineSpacing > 0 {
			fmt.Fprintf(buf, `<a:lnSpc><a:spcPct val="%d"/></a:lnSpc>`, int(style.LineSpacing*100000))
		}
		buf.WriteString(`</a:pPr>`)
		buf.WriteString(buildRunXML(elem.Text, style, fontSize))
		buf.WriteString(`</a:p>`)
	}

	buf.WriteString(`</p:txBody></p:sp>`)
}

// writeBulletList writes a text box with multiple bullet paragraphs.
func (b *Builder) writeBulletList(buf *strings.Builder, elem *model.Element) {
	transform := b.transform(elem)
	style := b.textStyle(elem.Type, elem.Style)
	fontSize := style.FontSize
	align := alignXML(style.Align)
	if align == "" {
		align = "l"
	}

	bulletChar := style.BulletChar
	if bulletChar == "" {
		bulletChar = "•"
	}

	buf.WriteString(fmt.Sprintf(
		`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>`,
		ooxmlObjectID(elem.ID), xmlEscape(elem.ID)))
	shapeProperties{
		transform: transform,
		geometry:  presetGeometryXML("rect", ""),
	}.writeTo(buf)

	buf.WriteString(`<p:txBody>` + textBodyPropertiesXML(style) + `<a:lstStyle/>`)

	for _, item := range elem.Items {
		buf.WriteString(fmt.Sprintf(
			`<a:p><a:pPr algn="%s" marL="342900" indent="-342900">`, align))
		buf.WriteString(fmt.Sprintf(
			`<a:buFont typeface="Arial" pitchFamily="34" charset="0"/><a:buChar char="%s"/>`,
			xmlEscape(bulletChar)))
		buf.WriteString(`</a:pPr>`)
		buf.WriteString(buildRunXML(item, style, fontSize))
		buf.WriteString(`</a:p>`)
	}

	buf.WriteString(`</p:txBody></p:sp>`)
}

// writeImage writes an embedded picture when a local asset is available.
func (b *Builder) writeImage(buf *strings.Builder, elem *model.Element) {
	transform := b.transform(elem)
	if asset := b.mediaByElement[elem]; asset != nil {
		crop := ""
		if elem.ImageCrop != nil {
			crop = fmt.Sprintf(`<a:srcRect l="%d" t="%d" r="%d" b="%d"/>`, int(elem.ImageCrop.Left*1000), int(elem.ImageCrop.Top*1000), int(elem.ImageCrop.Right*1000), int(elem.ImageCrop.Bottom*1000))
		}
		fmt.Fprintf(buf, `<p:pic><p:nvPicPr><p:cNvPr id="%d" name="%s" descr="%s"/><p:cNvPicPr><a:picLocks noChangeAspect="1"/></p:cNvPicPr><p:nvPr/></p:nvPicPr>`, ooxmlObjectID(elem.ID), xmlEscape(elem.ID), xmlEscape(elem.ImageAlt))
		// SVG images get an extension block for svgBlip
		isSVG := asset.ext == "svg"
		blipInner := fmt.Sprintf(` r:embed="%s"`, asset.relID)
		if isSVG {
			// SVG requires a rasterized fallback; we use the same SVG reference with the asvg extension.
			// Modern PowerPoint renders this correctly.
			blipInner = fmt.Sprintf(` r:embed="%s"><a:extLst><a:ext uri="{96DAC541-7B7A-43D3-8B79-37D633B846F1}"><asvg:svgBlip xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main" r:embed="%s"/></a:ext></a:extLst>`, asset.relID, asset.relID)
		}
		fmt.Fprintf(buf, `<p:blipFill><a:blip%s</a:blip>%s<a:stretch><a:fillRect/></a:stretch></p:blipFill>`, blipInner, crop)
		// Use rounded rectangle geometry if ImageRadius > 0
		geometry := presetGeometryXML("rect", "")
		if elem.ImageRadius > 0 {
			radVal := int(elem.ImageRadius * 50000)
			if radVal > 50000 {
				radVal = 50000
			}
			geometry = fmt.Sprintf(`<a:prstGeom prst="roundRect"><a:avLst><a:gd name="adj" fmla="val %d"/></a:avLst></a:prstGeom>`, radVal)
		}
		buf.WriteString(`<p:spPr>` + transform.xml() + geometry + `</p:spPr></p:pic>`)
		return
	}

	fmt.Fprintf(buf, `<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Image Placeholder"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>`, ooxmlObjectID(elem.ID))
	shapeProperties{transform: transform, geometry: presetGeometryXML("rect", ""), fill: solidFillXML("#646464"), line: solidLineXML("#CCCCCC", 9525)}.writeTo(buf)
	buf.WriteString(`<p:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/><a:p><a:pPr algn="ctr"/>`)
	buf.WriteString(buildRunXML("Image unavailable", model.TextStyle{Color: "#FFFFFF"}, 12))
	buf.WriteString(`</a:p></p:txBody></p:sp>`)
}

// writeShape writes an auto shape (sp) with fill/border/text.
func (b *Builder) writeShape(buf *strings.Builder, elem *model.Element) {
	if elem.Shape == nil {
		return
	}
	transform := b.transform(elem)

	var geometry string
	if elem.Shape.ShapeType == model.ShapeFreeform && elem.Shape.Freeform != nil {
		hasFill := elem.Shape.Fill != nil && elem.Shape.Fill.Color != ""
		geometry = freeformGeometryXML(elem.Shape.Freeform, hasFill)
	} else {
		prst := shapeToPrst(elem.Shape.ShapeType)
		adjustments := ""
		if elem.Shape.ShapeType == model.ShapeRoundedRectangle && elem.Shape.CornerRadius > 0 {
			rad := int(elem.Shape.CornerRadius * 50000)
			adjustments = fmt.Sprintf(`<a:avLst><a:gd name="adj" fmla="val %d"/></a:avLst>`, rad)
		}
		geometry = presetGeometryXML(prst, adjustments)
	}
	fmt.Fprintf(buf,
		`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Shape"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>`,
		ooxmlObjectID(elem.ID))
	shapeProperties{
		transform: transform,
		geometry:  geometry,
		fill:      fillStyleXML(elem.Shape.Fill, elem.Shape.FillColor),
		line:      lineStyleXML(elem.Shape.Line, elem.Shape.BorderColor, elem.Shape.BorderWidth),
		effects:   effectsXML(elem.Shape.Shadow, elem.Shape.Glow, elem.Shape.Reflection),
	}.writeTo(buf)

	// Text inside shape
	if elem.Shape.Text != "" {
		style := b.shapeTextStyle(elem)
		fontSize := style.FontSize
		align := alignXML(style.Align)
		if align == "" {
			align = "ctr"
		}
		buf.WriteString(`<p:txBody>` + textBodyPropertiesXML(style) + `<a:lstStyle/>`)
		fmt.Fprintf(buf, `<a:p><a:pPr algn="%s"/>`, align)
		buf.WriteString(buildRunXML(elem.Shape.Text, style, fontSize))
		buf.WriteString(`</a:p></p:txBody>`)
	} else {
		buf.WriteString(`<p:txBody><a:bodyPr/><a:lstStyle/><a:p/></p:txBody>`)
	}

	buf.WriteString(`</p:sp>`)
}

// writeTable writes a table shape (graphicFrame).
func (b *Builder) writeTable(buf *strings.Builder, elem *model.Element) {
	if elem.Table == nil {
		return
	}
	transform := b.transform(elem)

	td := elem.Table
	numCols := len(td.Headers)
	if numCols == 0 && len(td.Rows) > 0 {
		numCols = len(td.Rows[0])
	}
	if numCols == 0 {
		return
	}

	fontSize := td.FontSize
	if fontSize == 0 {
		fontSize = 12
	}

	fmt.Fprintf(buf,
		`<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="%d" name="Table"/><p:cNvGraphicFramePr/>`+
			`<p:nvPr/></p:nvGraphicFramePr>%s`, ooxmlObjectID(elem.ID), transform.graphicFrameXML())

	fmt.Fprintf(buf,
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table">`+
			`<a:tbl><a:tblPr firstRow="1" bandRow="1"><a:tableStyleId>`+
			`{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}</a:tableStyleId></a:tblPr>`)

	// Column widths
	colW := pctToEMU(elem.Rect.W, b.pres.SlideWidth) / int64(numCols)
	fmt.Fprintf(buf, `<a:tblGrid>`)
	for i := 0; i < numCols; i++ {
		fmt.Fprintf(buf, `<a:gridCol w="%d"/>`, colW)
	}
	fmt.Fprintf(buf, `</a:tblGrid>`)

	// Header row
	if len(td.Headers) > 0 {
		buf.WriteString(`<a:tr h="370840">`)
		hdrColor := td.HeaderColor
		if hdrColor == "" {
			hdrColor = b.pres.Theme.PrimaryColor
		}
		hr, hg, hb := hexToRGB(hdrColor)
		for _, cell := range td.Headers {
			gridSpan := ""
			if cell.ColSpan > 1 {
				gridSpan = fmt.Sprintf(` gridSpan="%d"`, cell.ColSpan)
			}
			rowSpan := ""
			if cell.RowSpan > 1 {
				rowSpan = fmt.Sprintf(` rowSpan="%d"`, cell.RowSpan)
			}
			fmt.Fprintf(buf,
				`<a:tc%s%s><a:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`+
					`<a:p><a:pPr algn="ctr"/><a:r><a:rPr lang="zh-CN" sz="%d" b="1">`+
					`<a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:rPr>`+
					`<a:t>%s</a:t></a:r></a:p></a:txBody>`+
					`<a:tcPr><a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill></a:tcPr></a:tc>`,
				gridSpan, rowSpan, fontSize*100, xmlEscape(cell.Text), hr, hg, hb)
		}
		buf.WriteString(`</a:tr>`)
	}

	// Data rows
	for rowIdx, row := range td.Rows {
		buf.WriteString(`<a:tr h="370840">`)
		for _, cell := range row {
			bgXML := ""
			if td.AltRowColor != "" && rowIdx%2 == 1 {
				ar, ag, ab := hexToRGB(td.AltRowColor)
				bgXML = fmt.Sprintf(`<a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`, ar, ag, ab)
			}

			// Determine text color: use cell style color, or default to a light color for dark themes
			textColor := cell.Style.Color
			if textColor == "" {
				textColor = "E2E8F0"
			}
			textColor = normalizeHex(textColor)
			// Determine alignment: use cell style align, or default center
			cellAlign := alignXML(cell.Style.Align)
			if cellAlign == "" {
				cellAlign = "ctr"
			}
			// Determine bold
			boldAttr := ""
			if cell.Style.Bold {
				boldAttr = ` b="1"`
			}
			// Determine font size override
			cellFontSize := fontSize
			if cell.Style.FontSize > 0 {
				cellFontSize = cell.Style.FontSize
			}

			// Cell merge attributes
			gridSpan := ""
			if cell.ColSpan > 1 {
				gridSpan = fmt.Sprintf(` gridSpan="%d"`, cell.ColSpan)
			}
			rowSpan := ""
			if cell.RowSpan > 1 {
				rowSpan = fmt.Sprintf(` rowSpan="%d"`, cell.RowSpan)
			}
			// hMerge: this cell is a continuation of a vertically merged cell above
			hMerge := ""
			if cell.RowSpan < 0 {
				hMerge = ` hMerge="1"`
			}

			fmt.Fprintf(buf,
				`<a:tc%s%s%s><a:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`+
					`<a:p><a:pPr algn="%s"/><a:r><a:rPr lang="zh-CN" sz="%d"%s>`+
					`<a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:rPr>`+
					`<a:t>%s</a:t></a:r></a:p></a:txBody>`+
					`<a:tcPr>%s</a:tcPr></a:tc>`,
				gridSpan, rowSpan, hMerge, cellAlign, cellFontSize*100, boldAttr, textColor, xmlEscape(cell.Text), bgXML)
		}
		buf.WriteString(`</a:tr>`)
	}

	buf.WriteString(`</a:tbl></a:graphicData></a:graphic></p:graphicFrame>`)
}

// writeChart writes an editable native DrawingML chart reference.
func (b *Builder) writeChart(buf *strings.Builder, elem *model.Element) {
	asset := b.chartByElement[elem]
	if elem.Chart == nil || asset == nil {
		return
	}
	transform := b.transform(elem)
	fmt.Fprintf(buf,
		`<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="%d" name="Chart %d"/><p:cNvGraphicFramePr/><p:nvPr/></p:nvGraphicFramePr>%s`,
		ooxmlObjectID(elem.ID), asset.index, transform.graphicFrameXML())
	fmt.Fprintf(buf,
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:id="%s"/></a:graphicData></a:graphic></p:graphicFrame>`,
		asset.relID)
}

// writeConnector writes a connector line/arrow (cxnSp).
func (b *Builder) writeConnector(buf *strings.Builder, elem *model.Element) {
	if elem.Connector == nil {
		return
	}
	conn := elem.Connector

	startX := pctToEMU(conn.StartX, b.pres.SlideWidth)
	startY := pctToEMU(conn.StartY, b.pres.SlideHeight)
	endX := pctToEMU(conn.EndX, b.pres.SlideWidth)
	endY := pctToEMU(conn.EndY, b.pres.SlideHeight)

	transform := drawingTransform{
		x: startX, y: startY,
		cx: endX - startX, cy: endY - startY,
	}

	prst := "line"
	if conn.ConnectorType == model.ShapeArrow {
		prst = "straightConnector1"
	}

	fmt.Fprintf(buf,
		`<p:cxnSp><p:nvCxnSpPr><p:cNvPr id="%d" name="Connector"/><p:cNvCxnSpPr/><p:nvPr/></p:nvCxnSpPr>`,
		ooxmlObjectID(elem.ID))
	width := int(conn.Width * 12700)
	if width == 0 {
		width = 9525
	}
	line := solidLineXML(conn.Color, width)
	if conn.ConnectorType == model.ShapeArrow || conn.ConnectorType == model.ShapeDoubleArrow {
		line = strings.Replace(line, `</a:ln>`, `<a:tailEnd type="triangle" w="med" len="med"/></a:ln>`, 1)
	}
	shapeProperties{
		transform: transform,
		geometry:  presetGeometryXML(prst, ""),
		line:      line,
	}.writeTo(buf)
	buf.WriteString(`</p:cxnSp>`)
}

// ============================================================
// Helpers
// ============================================================

func (b *Builder) textStyle(elementType model.ElementType, style model.TextStyle) model.TextStyle {
	if style.FontName == "" {
		if elementType == model.ElementTitle || elementType == model.ElementSubtitle {
			style.FontName = b.pres.Theme.TitleFont
		} else {
			style.FontName = b.pres.Theme.BodyFont
		}
	}
	if style.Color == "" {
		style.Color = b.pres.Theme.TextColor
	}
	if style.FontSize == 0 {
		if elementType == model.ElementTitle {
			style.FontSize = 32
		} else if elementType == model.ElementSubtitle {
			style.FontSize = 22
		} else {
			style.FontSize = 18
		}
	}
	return style
}

// shapeTextStyle derives a readable, theme-aware text style for text rendered
// inside a shape. Card-like shapes frequently have dark or saturated fills, so
// the text color is picked from fill luminance and comfortable insets are
// applied so text never touches the shape edges.
func (b *Builder) shapeTextStyle(elem *model.Element) model.TextStyle {
	style := elem.Shape.Style
	if style.FontName == "" {
		style.FontName = b.pres.Theme.BodyFont
	}
	if style.FontSize == 0 {
		style.FontSize = 14
	}
	if style.Color == "" {
		fill := elem.Shape.FillColor
		if fill == "" && elem.Shape.Fill != nil {
			switch {
			case elem.Shape.Fill.Color != "":
				fill = elem.Shape.Fill.Color
			case elem.Shape.Fill.Gradient != nil && len(elem.Shape.Fill.Gradient.Stops) > 0:
				fill = elem.Shape.Fill.Gradient.Stops[0].Color
			}
		}
		if fill != "" {
			style.Color = contrastTextColor(fill)
		} else {
			style.Color = b.pres.Theme.TextColor
		}
	}
	if style.VAlign == "" {
		style.VAlign = "middle"
	}
	// Comfortable panel insets (pt → EMU conversion happens in
	// textBodyPropertiesXML). Explicit user margins are preserved.
	if style.MarginLeft == 0 {
		style.MarginLeft = 6
	}
	if style.MarginRight == 0 {
		style.MarginRight = 6
	}
	if style.MarginTop == 0 {
		style.MarginTop = 4
	}
	if style.MarginBottom == 0 {
		style.MarginBottom = 4
	}
	return style
}

// contrastTextColor returns a readable text color for the given fill color:
// dark text on light fills, light text on dark fills.
func contrastTextColor(hex string) string {
	r, g, b := hexToRGB(hex)
	lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
	if lum > 0.62 {
		return "#0F172A"
	}
	return "#FFFFFF"
}

func textBodyPropertiesXML(style model.TextStyle) string {
	anchor := "t"
	if style.VAlign == "middle" || style.VAlign == "center" {
		anchor = "ctr"
	} else if style.VAlign == "bottom" {
		anchor = "b"
	}
	wrap := "square"
	if style.WordWrap != nil && !*style.WordWrap {
		wrap = "none"
	}
	return fmt.Sprintf(`<a:bodyPr wrap="%s" anchor="%s" lIns="%d" rIns="%d" tIns="%d" bIns="%d"><a:normAutofit/></a:bodyPr>`, wrap, anchor, int(style.MarginLeft*12700), int(style.MarginRight*12700), int(style.MarginTop*12700), int(style.MarginBottom*12700))
}

func (b *Builder) writeParagraph(buf *strings.Builder, paragraph model.Paragraph, fallback model.TextStyle) {
	style := paragraph.Style
	if style.FontName == "" {
		style.FontName = fallback.FontName
	}
	if style.FontSize == 0 {
		style.FontSize = fallback.FontSize
	}
	if style.Color == "" {
		style.Color = fallback.Color
	}
	align := alignXML(style.Align)
	if align == "" {
		align = alignXML(fallback.Align)
	}
	if align == "" {
		align = "l"
	}
	fmt.Fprintf(buf, `<a:p><a:pPr algn="%s" lvl="%d">`, align, paragraph.Level)
	if paragraph.SpaceBefore > 0 {
		fmt.Fprintf(buf, `<a:spcBef><a:spcPts val="%d"/></a:spcBef>`, int(paragraph.SpaceBefore*100))
	}
	if paragraph.SpaceAfter > 0 {
		fmt.Fprintf(buf, `<a:spcAft><a:spcPts val="%d"/></a:spcAft>`, int(paragraph.SpaceAfter*100))
	}
	bullet := paragraph.Bullet
	if bullet == "" {
		bullet = style.BulletChar
	}
	if bullet != "" {
		fmt.Fprintf(buf, `<a:buChar char="%s"/>`, xmlEscape(bullet))
	}
	buf.WriteString(`</a:pPr>`)
	if len(paragraph.Runs) > 0 {
		for _, run := range paragraph.Runs {
			runStyle := run.Style
			if runStyle.FontName == "" {
				runStyle.FontName = style.FontName
			}
			if runStyle.Color == "" {
				runStyle.Color = style.Color
			}
			if runStyle.FontSize == 0 {
				runStyle.FontSize = style.FontSize
			}
			buf.WriteString(buildRunXML(run.Text, runStyle, runStyle.FontSize))
		}
	} else {
		buf.WriteString(buildRunXML(paragraph.Text, style, style.FontSize))
	}
	buf.WriteString(`</a:p>`)
}

// buildRunXML builds a single text run (<a:r>) element.
func buildRunXML(text string, style model.TextStyle, fontSize int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<a:r><a:rPr lang="zh-CN" sz="%d"`, fontSize*100)
	if style.Bold {
		sb.WriteString(` b="1"`)
	}
	if style.Italic {
		sb.WriteString(` i="1"`)
	}
	if style.Underline {
		sb.WriteString(` u="sng"`)
	}
	if style.Strike {
		sb.WriteString(` strike="sngStrike"`)
	}
	if style.LetterSpacing != 0 {
		fmt.Fprintf(&sb, ` spc="%d"`, int(style.LetterSpacing*100))
	}
	sb.WriteString(`>`)
	// Child order must follow CT_TextCharacterProperties schema:
	// fill → effectLst → latin/ea/cs (PowerPoint enforces this order).
	if style.Color != "" {
		sb.WriteString(solidFillOpacityXML(style.Color, style.Opacity))
	}
	if style.Shadow {
		sb.WriteString(`<a:effectLst><a:outerShdw blurRad="38100" dist="19050" dir="2700000"><a:srgbClr val="000000"><a:alpha val="35000"/></a:srgbClr></a:outerShdw></a:effectLst>`)
	}
	if style.FontName != "" {
		fontName := xmlEscape(style.FontName)
		fmt.Fprintf(&sb, `<a:latin typeface="%s"/><a:ea typeface="%s"/><a:cs typeface="%s"/>`, fontName, fontName, fontName)
	}
	sb.WriteString(`</a:rPr>`)
	fmt.Fprintf(&sb, `<a:t>%s</a:t></a:r>`, xmlEscape(text))
	return sb.String()
}

// alignXML converts align string to OOXML code.
func alignXML(align string) string {
	switch strings.ToLower(align) {
	case "left", "l":
		return "l"
	case "center", "centre", "c":
		return "ctr"
	case "right", "r":
		return "r"
	case "justify", "j":
		return "just"
	default:
		return ""
	}
}

// colorXML returns srgbClr XML for a hex color.
func colorXML(hex string) string {
	r, g, b := hexToRGB(hex)
	return fmt.Sprintf(`<a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`, r, g, b)
}

// shapeToPrst maps our ShapeType to OOXML preset geometry.
func shapeToPrst(st model.ShapeType) string {
	switch st {
	case model.ShapeRectangle:
		return "rect"
	case model.ShapeRoundedRectangle:
		return "roundRect"
	case model.ShapeEllipse:
		return "ellipse"
	case model.ShapeTriangle:
		return "triangle"
	case model.ShapeDiamond:
		return "diamond"
	case model.ShapeLine:
		return "line"
	case model.ShapeArrow:
		return "rightArrow"
	case model.ShapeDoubleArrow:
		return "leftRightArrow"
	case model.ShapePentagon:
		return "pentagon"
	case model.ShapeHexagon:
		return "hexagon"
	case model.ShapeStar:
		return "star5"
	case model.ShapeCallout:
		return "wedgeRectCallout"
	case model.ShapeHeart:
		return "heart"
	case model.ShapeCloud:
		return "cloud"
	default:
		return "rect"
	}
}

// writeVideo writes a video element as a p:pic with embedded media.
func (b *Builder) writeVideo(buf *strings.Builder, elem *model.Element) {
	if elem.Media == nil {
		return
	}
	transform := b.transform(elem)
	objID := ooxmlObjectID(elem.ID)

	// Get the video asset and optional poster image asset
	videoAsset := b.mediaByElement[elem]
	if videoAsset == nil {
		// Fallback: render as placeholder text
		b.writeTextBox(buf, elem)
		return
	}

	// Determine poster image embed (if provided)
	posterEmbed := ""
	if posterAsset := b.posterByElement[elem]; posterAsset != nil {
		posterEmbed = posterAsset.relID
	} else {
		// Use the video's own relationship for the blip fallback
		posterEmbed = videoAsset.relID
	}

	// Build the pic element with video extension
	fmt.Fprintf(buf,
		`<p:pic><p:nvPicPr><p:cNvPr id="%d" name="%s"><a:hlinkClick r:id="" action="ppaction://media"/></p:cNvPr>`,
		objID, xmlEscape(elem.ID))
	fmt.Fprintf(buf,
		`<p:cNvPicPr><a:picLocks noChangeAspect="1"/></p:cNvPicPr>`)
	fmt.Fprintf(buf,
		`<p:nvPr><a:videoFile r:link=""/><p:extLst><p:ext uri="{DAA4B4D4-6D71-4841-9C94-3DE7FCFB9230}"><p14:media xmlns:p14="http://schemas.microsoft.com/office/powerpoint/2010/main" r:embed="%s"/></p:ext></p:extLst></p:nvPr></p:nvPicPr>`,
		videoAsset.relID)

	// blipFill with poster image
	fmt.Fprintf(buf,
		`<p:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></p:blipFill>`,
		posterEmbed)

	// Shape properties
	geometry := presetGeometryXML("rect", "")
	buf.WriteString(`<p:spPr>` + transform.xml() + geometry + `</p:spPr>`)

	buf.WriteString(`</p:pic>`)
}

// writeAudio writes an audio element as a small icon-style p:pic with embedded media.
func (b *Builder) writeAudio(buf *strings.Builder, elem *model.Element) {
	if elem.Media == nil {
		return
	}
	transform := b.transform(elem)
	objID := ooxmlObjectID(elem.ID)

	audioAsset := b.mediaByElement[elem]
	if audioAsset == nil {
		b.writeTextBox(buf, elem)
		return
	}

	posterEmbed := audioAsset.relID

	// Build the pic element with audio extension
	fmt.Fprintf(buf,
		`<p:pic><p:nvPicPr><p:cNvPr id="%d" name="%s"><a:hlinkClick r:id="" action="ppaction://media"/></p:cNvPr>`,
		objID, xmlEscape(elem.ID))
	fmt.Fprintf(buf,
		`<p:cNvPicPr><a:picLocks noChangeAspect="1"/></p:cNvPicPr>`)
	fmt.Fprintf(buf,
		`<p:nvPr><a:audioFile r:link=""/><p:extLst><p:ext uri="{DAA4B4D4-6D71-4841-9C94-3DE7FCFB9230}"><p14:media xmlns:p14="http://schemas.microsoft.com/office/powerpoint/2010/main" r:embed="%s"/></p:ext></p:extLst></p:nvPr></p:nvPicPr>`,
		audioAsset.relID)

	fmt.Fprintf(buf,
		`<p:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></p:blipFill>`,
		posterEmbed)

	geometry := presetGeometryXML("rect", "")
	buf.WriteString(`<p:spPr>` + transform.xml() + geometry + `</p:spPr>`)

	buf.WriteString(`</p:pic>`)
}
