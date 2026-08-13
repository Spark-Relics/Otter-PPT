package builder

import (
	"fmt"
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
	case model.ElementGroup:
		// Groups are logical; children are rendered independently
	default:
		b.writeTextBox(buf, elem)
	}
}

// offset returns the off/ext XML for an element rect.
func (b *Builder) offset(elem *model.Element) (string, string) {
	offX := pctToEMU(elem.Rect.X, b.pres.SlideWidth)
	offY := pctToEMU(elem.Rect.Y, b.pres.SlideHeight)
	extW := pctToEMU(elem.Rect.W, b.pres.SlideWidth)
	extH := pctToEMU(elem.Rect.H, b.pres.SlideHeight)
	off := fmt.Sprintf(`<a:off x="%d" y="%d"/>`, offX, offY)
	ext := fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, extW, extH)
	return off, ext
}

// rotationXML returns the rot attribute for shapes, if any.
func rotationXML(elem *model.Element) string {
	if elem.Rotation != 0 {
		return fmt.Sprintf(` rot="%d"`, int(elem.Rotation*60000))
	}
	return ""
}

// writeTextBox writes a plain text box shape (sp).
func (b *Builder) writeTextBox(buf *strings.Builder, elem *model.Element) {
	off, ext := b.offset(elem)
	fontSize := elem.Style.FontSize
	if fontSize == 0 {
		fontSize = 18
	}
	align := alignXML(elem.Style.Align)
	if align == "" {
		align = "l"
	}

	buf.WriteString(fmt.Sprintf(
		`<p:sp><p:nvSpPr><p:cNvPr id="%s" name="%s"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>`,
		elem.ID, xmlEscape(elem.ID)))
	buf.WriteString(fmt.Sprintf(
		`<p:spPr>%s%s<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr>`,
		off, ext))

	// Text body
	buf.WriteString(`<p:txBody><a:bodyPr wrap="square" rtlCol="0">`)
	buf.WriteString(fmt.Sprintf(`<a:normAutofit/>`))
	buf.WriteString(`</a:bodyPr><a:lstStyle/>`)

	// Paragraph
	buf.WriteString(fmt.Sprintf(
		`<a:p><a:pPr algn="%s">`, align))
	if elem.Style.LineSpacing > 0 {
		spc := int(elem.Style.LineSpacing * 100)
		fmt.Fprintf(buf, `<a:lnSpc><a:spcPct val="%d"/></a:lnSpc>`, spc)
	}
	buf.WriteString(`</a:pPr>`)
	buf.WriteString(buildRunXML(elem.Text, elem.Style, fontSize))
	buf.WriteString(`</a:p>`)

	buf.WriteString(`</p:txBody></p:sp>`)
}

// writeBulletList writes a text box with multiple bullet paragraphs.
func (b *Builder) writeBulletList(buf *strings.Builder, elem *model.Element) {
	off, ext := b.offset(elem)
	fontSize := elem.Style.FontSize
	if fontSize == 0 {
		fontSize = 18
	}
	align := alignXML(elem.Style.Align)
	if align == "" {
		align = "l"
	}

	bulletChar := elem.Style.BulletChar
	if bulletChar == "" {
		bulletChar = "•"
	}

	buf.WriteString(fmt.Sprintf(
		`<p:sp><p:nvSpPr><p:cNvPr id="%s" name="%s"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>`,
		elem.ID, xmlEscape(elem.ID)))
	buf.WriteString(fmt.Sprintf(
		`<p:spPr>%s%s<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr>`,
		off, ext))

	buf.WriteString(`<p:txBody><a:bodyPr wrap="square"><a:normAutofit/></a:bodyPr><a:lstStyle/>`)

	for _, item := range elem.Items {
		buf.WriteString(fmt.Sprintf(
			`<a:p><a:pPr algn="%s" marL="342900" indent="-342900">`, align))
		buf.WriteString(fmt.Sprintf(
			`<a:buFont typeface="Arial" pitchFamily="34" charset="0"/><a:buChar char="%s"/>`,
			xmlEscape(bulletChar)))
		buf.WriteString(`</a:pPr>`)
		buf.WriteString(buildRunXML(item, elem.Style, fontSize))
		buf.WriteString(`</a:p>`)
	}

	buf.WriteString(`</p:txBody></p:sp>`)
}

// writeImage writes a picture shape (pic).
func (b *Builder) writeImage(buf *strings.Builder, elem *model.Element) {
	off, ext := b.offset(elem)

	// For image, we use a placeholder rectangle if no relationship is set up.
	// In a full implementation, images would be embedded in ppt/media/ and
	// referenced via relationship IDs. For now, we render a placeholder shape.
	r, g, bl := 100, 100, 100
	fmt.Fprintf(buf,
		`<p:sp><p:nvSpPr><p:cNvPr id="%s" name="Image Placeholder"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>`,
		elem.ID)
	fmt.Fprintf(buf,
		`<p:spPr>%s%s<a:prstGeom prst="rect"><a:avLst/></a:prstGeom>`+
			`<a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`+
			`<a:ln w="9525"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln>`+
			`</p:spPr>`, off, ext, r, g, bl)

	// Label text
	if elem.ImagePath != "" {
		buf.WriteString(`<p:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`)
		fmt.Fprintf(buf,
			`<a:p><a:pPr algn="ctr"/><a:r><a:rPr lang="en-US" sz="1200">`+
				`<a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:rPr>`+
				`<a:t>📷 %s</a:t></a:r></a:p>`, xmlEscape(elem.ImagePath))
		buf.WriteString(`</p:txBody>`)
	}
	buf.WriteString(`</p:sp>`)
}

// writeShape writes an auto shape (sp) with fill/border/text.
func (b *Builder) writeShape(buf *strings.Builder, elem *model.Element) {
	if elem.Shape == nil {
		return
	}
	off, ext := b.offset(elem)
	rot := rotationXML(elem)

	prst := shapeToPrst(elem.Shape.ShapeType)

	fmt.Fprintf(buf,
		`<p:sp><p:nvSpPr><p:cNvPr id="%s" name="Shape"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>`,
		elem.ID)
	fmt.Fprintf(buf, `<p:spPr%s>%s%s`, rot, off, ext)

	// Shape geometry
	adjXML := ""
	if elem.Shape.ShapeType == model.ShapeRoundedRectangle && elem.Shape.CornerRadius > 0 {
		rad := int(elem.Shape.CornerRadius * 50000) // adjustment in 60000ths approx
		adjXML = fmt.Sprintf(`<a:avLst><a:gd name="adj" fmla="val %d"/></a:avLst>`, rad)
	} else {
		adjXML = `<a:avLst/>`
	}
	fmt.Fprintf(buf, `<a:prstGeom prst="%s">%s</a:prstGeom>`, prst, adjXML)

	// Fill
	if elem.Shape.FillColor != "" {
		r, g, bl := hexToRGB(elem.Shape.FillColor)
		fmt.Fprintf(buf, `<a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`, r, g, bl)
	} else {
		buf.WriteString(`<a:noFill/>`)
	}

	// Border
	if elem.Shape.BorderColor != "" {
		w := int(elem.Shape.BorderWidth * 9525) // pt to... rough EMU
		if w == 0 {
			w = 9525
		}
		r, g, bl := hexToRGB(elem.Shape.BorderColor)
		fmt.Fprintf(buf,
			`<a:ln w="%d"><a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill></a:ln>`,
			w, r, g, bl)
	} else {
		buf.WriteString(`<a:ln><a:noFill/></a:ln>`)
	}

	buf.WriteString(`</p:spPr>`)

	// Text inside shape
	if elem.Shape.Text != "" {
		fontSize := elem.Shape.Style.FontSize
		if fontSize == 0 {
			fontSize = 14
		}
		align := alignXML(elem.Shape.Style.Align)
		if align == "" {
			align = "ctr"
		}
		buf.WriteString(`<p:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`)
		fmt.Fprintf(buf, `<a:p><a:pPr algn="%s"/>`, align)
		buf.WriteString(buildRunXML(elem.Shape.Text, elem.Shape.Style, fontSize))
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
	off, ext := b.offset(elem)

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
		`<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="%s" name="Table"/><p:cNvGraphicFramePr/>`+
			`<p:nvPr/></p:nvGraphicFramePr>%s%s`, elem.ID, off, ext)

	fmt.Fprintf(buf,
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table">`+
			`<a:tbl><a:tblPr firstRow="1" bandRow="1"><a:tableStyleId/>`+
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
			fmt.Fprintf(buf,
				`<a:tc><a:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`+
					`<a:p><a:pPr algn="ctr"/><a:r><a:rPr lang="zh-CN" sz="%d" b="1">`+
					`<a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:rPr>`+
					`<a:t>%s</a:t></a:r></a:p></a:txBody>`+
					`<a:tcPr><a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill></a:tcPr></a:tc>`,
				fontSize*100, xmlEscape(cell.Text), hr, hg, hb)
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
			fmt.Fprintf(buf,
				`<a:tc><a:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`+
					`<a:p><a:pPr algn="l"/><a:r><a:rPr lang="zh-CN" sz="%d">`+
					`<a:solidFill><a:srgbClr val="333333"/></a:solidFill></a:rPr>`+
					`<a:t>%s</a:t></a:r></a:p></a:txBody>`+
					`<a:tcPr>%s</a:tcPr></a:tc>`,
				fontSize*100, xmlEscape(cell.Text), bgXML)
		}
		buf.WriteString(`</a:tr>`)
	}

	buf.WriteString(`</a:tbl></a:graphicData></a:graphic></p:graphicFrame>`)
}

// writeChart writes a chart placeholder (simplified — renders as text summary).
func (b *Builder) writeChart(buf *strings.Builder, elem *model.Element) {
	if elem.Chart == nil {
		return
	}
	off, ext := b.offset(elem)
	cd := elem.Chart

	// Render chart as a styled card with data summary
	// Full chart embedding requires chart XML parts + embedded xlsx data
	r, g, bl := hexToRGB(b.pres.Theme.PrimaryColor)
	fmt.Fprintf(buf,
		`<p:sp><p:nvSpPr><p:cNvPr id="%s" name="Chart"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>`,
		elem.ID)
	fmt.Fprintf(buf,
		`<p:spPr>%s%s<a:prstGeom prst="round2SameRect"><a:avLst/></a:prstGeom>`,
		off, ext)
	fmt.Fprintf(buf,
		`<a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`, r, g, bl)
	buf.WriteString(`<a:ln w="12700"><a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:ln></p:spPr>`)

	buf.WriteString(`<p:txBody><a:bodyPr anchor="ctr"/><a:lstStyle/>`)
	if cd.Title != "" {
		fmt.Fprintf(buf,
			`<a:p><a:pPr algn="ctr"/><a:r><a:rPr lang="zh-CN" sz="2000" b="1">`+
				`<a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:rPr>`+
				`<a:t>%s</a:t></a:r></a:p>`, xmlEscape(cd.Title))
	}
	// Data rows as text
	for _, s := range cd.Series {
		vals := ""
		for i, v := range s.Values {
			if i > 0 {
				vals += " | "
			}
			vals += fmt.Sprintf("%.0f", v)
		}
		fmt.Fprintf(buf,
			`<a:p><a:pPr algn="ctr"/><a:r><a:rPr lang="zh-CN" sz="1400">`+
				`<a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:rPr>`+
				`<a:t>%s: %s</a:t></a:r></a:p>`, xmlEscape(s.Name), vals)
	}
	buf.WriteString(`</p:txBody></p:sp>`)
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

	off := fmt.Sprintf(`<a:off x="%d" y="%d"/>`, startX, startY)
	ext := fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, endX-startX, endY-startY)

	prst := "line"
	if conn.ConnectorType == model.ShapeArrow {
		prst = "straightConnector1"
	}

	fmt.Fprintf(buf,
		`<p:cxnSp><p:nvCxnSpPr><p:cNvPr id="%s" name="Connector"/><p:cNvCxnSpPr/><p:nvPr/></p:nvCxnSpPr>`,
		elem.ID)
	fmt.Fprintf(buf, `<p:spPr>%s%s`, off, ext)
	fmt.Fprintf(buf, `<a:prstGeom prst="%s"><a:avLst/></a:prstGeom>`, prst)

	r, g, bl := hexToRGB(conn.Color)
	w := int(conn.Width * 12700) // pt to EMU
	if w == 0 {
		w = 9525
	}
	fmt.Fprintf(buf,
		`<a:ln w="%d"><a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill></a:ln>`,
		w, r, g, bl)

	// Arrow head
	if conn.ConnectorType == model.ShapeArrow || conn.ConnectorType == model.ShapeDoubleArrow {
		fmt.Fprintf(buf,
			`<a:tailEnd type="triangle" w="med" len="med"/>`)
	}

	buf.WriteString(`</p:spPr></p:cxnSp>`)
}

// ============================================================
// Helpers
// ============================================================

// buildRunXML builds a single text run (<a:r>) element.
func buildRunXML(text string, style model.TextStyle, fontSize int) string {
	var sb strings.Builder
	sb.WriteString(`<a:r><a:rPr lang="zh-CN"`)
	if fontSize > 0 {
		fmt.Fprintf(&sb, ` sz="%d"`, fontSize*100)
	}
	if style.Bold {
		sb.WriteString(` b="1"`)
	}
	if style.Italic {
		sb.WriteString(` i="1"`)
	}
	if style.FontName != "" {
		fmt.Fprintf(&sb, `><a:latin typeface="%s"/><a:ea typeface="%s"`,
			style.FontName, style.FontName)
	}
	if style.Color != "" {
		r, g, bl := hexToRGB(style.Color)
		fmt.Fprintf(&sb, `><a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`,
			r, g, bl)
	}
	sb.WriteString(`/>`)
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
