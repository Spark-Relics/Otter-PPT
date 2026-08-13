package builder

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// writeSlide writes ppt/slides/slideN.xml — the core slide content.
func (b *Builder) writeSlide(zw *zip.Writer, slideNum int, slide *model.Slide) error {
	path := fmt.Sprintf("ppt/slides/slide%d.xml", slideNum)
	w, err := zw.Create(path)
	if err != nil {
		return err
	}

	var body strings.Builder

	// Background
	b.writeBackground(&body, slide)

	// Elements sorted by z-order
	elements := sortedElements(slide.Elements)

	body.WriteString(`<p:spTree>`)
	body.WriteString(`<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/>`)

	for _, elem := range elements {
		b.writeElement(&body, elem)
	}

	body.WriteString(`</p:spTree>`)

	// Transition (optional)
	transitionXML := ""
	if slide.Transition != nil && slide.Transition.Type != model.TransitionNone {
		transitionXML = b.buildTransition(slide.Transition)
	}

	// Notes
	notesXML := ""
	if slide.Notes != "" {
		notesXML = fmt.Sprintf(`<p:notes>`+
			`<p:cSld><p:spTree>`+
			`<p:sp><p:nvSpPr><p:cNvPr id="2" name="Notes"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>`+
			`<p:spPr/><p:txBody><a:bodyPr/><a:lstStyle/>`+
			`<a:p><a:r><a:rPr lang="zh-CN"/><a:t>%s</a:t></a:r></a:p>`+
			`</p:txBody></p:sp>`+
			`</p:spTree></p:cSld></p:notes>`, xmlEscape(slide.Notes))
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
<p:cSld>%s</p:cSld>%s%s
</p:sld>`, body.String(), transitionXML, "")

	_ = notesXML // notes need separate notesSlide parts; left for future enhancement

	_, err = w.Write([]byte(xml))
	return err
}

// writeBackground writes the slide background XML.
func (b *Builder) writeBackground(buf *strings.Builder, slide *model.Slide) {
	if slide.Background == nil {
		return
	}

	bg := slide.Background
	switch bg.Type {
	case model.BgSolid:
		r, g, bl := hexToRGB(bg.Color)
		fmt.Fprintf(buf,
			`<p:bg><p:bgPr><a:solidFill><a:srgbClr val="%02X%02X%02X"/></a:solidFill>`+
				`<a:effectLst/></p:bgPr></p:bg>`, r, g, bl)

	case model.BgGradient:
		if bg.Gradient != nil && len(bg.Gradient.Stops) >= 2 {
			buf.WriteString(`<p:bg><p:bgPr>`)
			b.writeGradientFill(buf, bg.Gradient)
			buf.WriteString(`<a:effectLst/></p:bgPr></p:bg>`)
		}

	case model.BgImage:
		// Image backgrounds require relationship parts; using solid fallback for now
		// Future: embed image and add blipFill reference
	}
}

// writeGradientFill writes an OOXML gradient fill element.
func (b *Builder) writeGradientFill(buf *strings.Builder, grad *model.Gradient) {
	gradFill := `<a:gradFill>`
	if grad.Type == model.GradientLinear {
		ang := int(grad.Angle * 60000) // degrees to 60000ths
		gradFill = fmt.Sprintf(`<a:gradFill flip="none" rotWithShape="1"><a:gsLst>`)
		_ = ang
		for _, stop := range grad.Stops {
			r, g, bl := hexToRGB(stop.Color)
			pos := int(stop.Position * 1000)
			gradFill += fmt.Sprintf(
				`<a:gs pos="%d"><a:srgbClr val="%02X%02X%02X"/></a:gs>`,
				pos, r, g, bl)
		}
		gradFill += `</a:gsLst>`
		gradFill += fmt.Sprintf(`<a:lin ang="%d" scaled="1"/>`, ang)
		gradFill += `</a:gradFill>`
	}
	buf.WriteString(gradFill)
}

// sortedElements returns elements sorted by z-order.
func sortedElements(elements []*model.Element) []*model.Element {
	result := make([]*model.Element, len(elements))
	copy(result, elements)
	// Simple insertion sort by z-order
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].ZOrder < result[j-1].ZOrder; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result
}

// buildTransition builds the transition XML for a slide.
func (b *Builder) buildTransition(t *model.Transition) string {
	dur := int(t.Duration * 1000)
	if dur == 0 {
		dur = 700
	}

	var inner string
	switch t.Type {
	case model.TransitionFade:
		inner = `<p:fade/>`
	case model.TransitionPush:
		inner = `<p:push dir="l"/>`
	case model.TransitionWipe:
		inner = `<p:wipe dir="l"/>`
	case model.TransitionSplit:
		inner = `<p:split orient="horz" dir="out"/>`
	case model.TransitionCover:
		inner = `<p:cover dir="l"/>`
	case model.TransitionZoom:
		inner = `<p:zoom dir="in"/>`
	case model.TransitionMorph:
		inner = `<p:morph option="byObject"/>`
	default:
		return ""
	}

	return fmt.Sprintf(`<p:transition xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" spd="med">%s</p:transition>`, inner)
}
