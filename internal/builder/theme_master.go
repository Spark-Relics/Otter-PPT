package builder

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// writeTheme writes ppt/theme/theme1.xml.
func (b *Builder) writeTheme(zw *zip.Writer) error {
	w, err := zw.Create("ppt/theme/theme1.xml")
	if err != nil {
		return err
	}
	primary := strings.TrimPrefix(b.pres.Theme.PrimaryColor, "#")
	if primary == "" {
		primary = "1A73E8"
	}
	secondary := strings.TrimPrefix(b.pres.Theme.SecondaryColor, "#")
	if secondary == "" {
		secondary = "424242"
	}
	accent := strings.TrimPrefix(b.pres.Theme.AccentColor, "#")
	if accent == "" {
		accent = "FF6D00"
	}
	background := strings.TrimPrefix(b.pres.Theme.BackgroundColor, "#")
	if background == "" {
		background = "FFFFFF"
	}
	textColor := strings.TrimPrefix(b.pres.Theme.TextColor, "#")
	if textColor == "" {
		textColor = "212121"
	}
	titleFont := b.pres.Theme.TitleFont
	if titleFont == "" {
		titleFont = "Aptos Display"
	}
	bodyFont := b.pres.Theme.BodyFont
	if bodyFont == "" {
		bodyFont = "Aptos"
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Office Theme">
<a:themeElements>
<a:clrScheme name="Office">
<a:dk1><a:srgbClr val="%s"/></a:dk1>
<a:lt1><a:srgbClr val="%s"/></a:lt1>
<a:dk2><a:srgbClr val="%s"/></a:dk2>
<a:lt2><a:srgbClr val="E8E8E8"/></a:lt2>
<a:accent1><a:srgbClr val="%s"/></a:accent1>
<a:accent2><a:srgbClr val="%s"/></a:accent2>
<a:accent3><a:srgbClr val="00ACC1"/></a:accent3>
<a:accent4><a:srgbClr val="43A047"/></a:accent4>
<a:accent5><a:srgbClr val="FB8C00"/></a:accent5>
<a:accent6><a:srgbClr val="8E24AA"/></a:accent6>
<a:hlink><a:srgbClr val="1A73E8"/></a:hlink>
<a:folHlink><a:srgbClr val="FF6D00"/></a:folHlink>
</a:clrScheme>
<a:fontScheme name="Office">
<a:majorFont>
<a:latin typeface="%s"/><a:ea typeface="%s"/><a:cs typeface="%s"/>
</a:majorFont>
<a:minorFont>
<a:latin typeface="%s"/><a:ea typeface="%s"/><a:cs typeface="%s"/>
</a:minorFont>
</a:fontScheme>
<a:fmtScheme name="Office">
<a:fillStyleLst>
<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
</a:fillStyleLst>
<a:lnStyleLst>
<a:ln w="9525" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
<a:ln w="25400" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
<a:ln w="38100" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
</a:lnStyleLst>
<a:effectStyleLst>
<a:effectStyle><a:effectLst/></a:effectStyle>
<a:effectStyle><a:effectLst/></a:effectStyle>
<a:effectStyle><a:effectLst/></a:effectStyle>
</a:effectStyleLst>
<a:bgFillStyleLst>
<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
</a:bgFillStyleLst>
</a:fmtScheme>
</a:themeElements>
<a:objectDefaults/>
<a:extraClrSchemeLst/>
</a:theme>`, textColor, background, secondary, primary, accent, xmlEscape(titleFont), xmlEscape(titleFont), xmlEscape(titleFont), xmlEscape(bodyFont), xmlEscape(bodyFont), xmlEscape(bodyFont))

	_, err = w.Write([]byte(xml))
	return err
}

// numLayouts is the total number of slide layouts generated.
const numLayouts = 5

// layoutNumberForSlide maps a SlideLayout to a 1-based layout file number.
// 1=title, 2=titleContent, 3=twoColumn, 4=section, 5=blank.
func layoutNumberForSlide(layout model.SlideLayout) int {
	switch layout {
	case model.LayoutTitle:
		return 1
	case model.LayoutTitleContent:
		return 2
	case model.LayoutTwoColumn, model.LayoutImageLeft, model.LayoutImageRight, model.LayoutImageFull:
		return 3
	case model.LayoutSection:
		return 4
	default:
		return 5 // blank
	}
}

// writeSlideMaster writes the slide master referencing all layout parts.
func (b *Builder) writeSlideMaster(zw *zip.Writer) error {
	w, err := zw.Create("ppt/slideMasters/slideMaster1.xml")
	if err != nil {
		return err
	}

	// Build sldLayoutIdLst with all layouts.
	var layoutIDs strings.Builder
	for i := 1; i <= numLayouts; i++ {
		fmt.Fprintf(&layoutIDs, `<p:sldLayoutId id="%d" r:id="rId%d"/>`, 2147483648+uint32(i), i)
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
<p:cSld>
<p:bg>
<p:bgRef idx="1001"><a:schemeClr val="bg1"/></p:bgRef>
</p:bg>
<p:spTree>
<p:nvGrpSpPr>
<p:cNvPr id="1" name=""/>
<p:cNvGrpSpPr/>
<p:nvPr/>
</p:nvGrpSpPr>
<p:grpSpPr/>
</p:spTree>
</p:cSld>
<p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
<p:sldLayoutIdLst>` + layoutIDs.String() + `</p:sldLayoutIdLst>
<p:txStyles>
<p:titleStyle>
<a:lvl1pPr algn="l" defTabSz="914400" rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1">
<a:spcBef><a:spcPct val="0"/></a:spcBef>
<a:buNone/>
<a:defRPr sz="4400" kern="1200">
<a:solidFill><a:schemeClr val="tx1"/></a:solidFill>
<a:latin typeface="+mj-lt"/><a:ea typeface="+mj-ea"/><a:cs typeface="+mj-cs"/>
</a:defRPr>
</a:lvl1pPr>
</p:titleStyle>
<p:bodyStyle>
<a:lvl1pPr marL="342900" indent="-342900" algn="l" defTabSz="914400" rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1">
<a:spcBef><a:spcPct val="20000"/></a:spcBef>
<a:buFont typeface="Arial" pitchFamily="34" charset="0"/>
<a:buChar char="•"/>
<a:defRPr sz="2800" kern="1200">
<a:solidFill><a:schemeClr val="tx1"/></a:solidFill>
<a:latin typeface="+mn-lt"/><a:ea typeface="+mn-ea"/><a:cs typeface="+mn-cs"/>
</a:defRPr>
</a:lvl1pPr>
</p:bodyStyle>
<p:otherStyle/>
</p:txStyles>
</p:sldMaster>`

	_, err = w.Write([]byte(xml))
	return err
}

// writeSlideMasterRels writes ppt/slideMasters/_rels/slideMaster1.xml.rels.
// rId1..rId5 = layouts, rId6 = theme.
func (b *Builder) writeSlideMasterRels(zw *zip.Writer) error {
	w, err := zw.Create("ppt/slideMasters/_rels/slideMaster1.xml.rels")
	if err != nil {
		return err
	}

	var rels strings.Builder
	for i := 1; i <= numLayouts; i++ {
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout%d.xml"/>`, i, i)
	}
	themeRels := fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>`, numLayouts+1)

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		rels.String() + themeRels + `
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}

// placeholderXML returns a <p:ph> element with optional idx and sz.
func placeholderXML(phType, idx string, sz int) string {
	idxAttr := ""
	if idx != "" {
		idxAttr = ` idx="` + idx + `"`
	}
	szAttr := ""
	if sz > 0 {
		szAttr = fmt.Sprintf(` sz="%d"`, sz)
	}
	return fmt.Sprintf(`<p:ph type="%s"%s%s/>`, phType, idxAttr, szAttr)
}

// layoutPlaceholderSp builds a placeholder shape (p:sp) with given parameters.
func layoutPlaceholderSp(id int, name, phType, phIdx string, x, y, cx, cy int64, fontSize int) string {
	ph := placeholderXML(phType, phIdx, 0)
	geo := presetGeometryXML("rect", "")

	bodyPr := `<a:bodyPr anchor="ctr" lIns="91440" tIns="45720" bIns="45720" rIns="91440"><a:normAutofit/></a:bodyPr>`
	if phType == "title" || phType == "ctrTitle" {
		bodyPr = `<a:bodyPr anchor="b" lIns="91440" tIns="45720" bIns="45720" rIns="91440"><a:normAutofit/></a:bodyPr>`
	}

	fontSz := fontSize
	if fontSz == 0 {
		switch phType {
		case "title", "ctrTitle":
			fontSz = 44
		case "subTitle":
			fontSz = 32
		default:
			fontSz = 28
		}
	}

	return fmt.Sprintf(
		`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr/><p:nvPr>%s</p:nvPr></p:nvSpPr>`+
			`<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>%s</p:spPr>`+
			`<p:txBody>%s<a:lstStyle/><a:p><a:pPr algn="ctr"/><a:endParaRPr lang="zh-CN" sz="%d"/></a:p></p:txBody></p:sp>`,
		id, name, ph, x, y, cx, cy, geo, bodyPr, fontSz*100)
}

// writeSlideLayout writes all 5 slide layout files with proper placeholders.
func (b *Builder) writeSlideLayout(zw *zip.Writer) error {
	slideW := inchToEMU(b.pres.SlideWidth)
	slideH := inchToEMU(b.pres.SlideHeight)

	// Common margins
	margin := inchToEMU(0.5) // 0.5 inch

	layouts := b.buildAllLayouts(slideW, slideH, margin)

	for i := 1; i <= numLayouts; i++ {
		path := fmt.Sprintf("ppt/slideLayouts/slideLayout%d.xml", i)
		w, err := zw.Create(path)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(layouts[i-1]))
		if err != nil {
			return err
		}
	}
	return nil
}

// buildAllLayouts returns XML strings for all 5 slide layouts.
func (b *Builder) buildAllLayouts(slideW, slideH, margin int64) []string {
	// Helper: full-width title bar
	titleY := margin
	titleH := inchToEMU(1.0)
	titleW := slideW - 2*margin
	titleX := margin

	// Content area below title
	contentY := titleY + titleH + inchToEMU(0.3)
	contentH := slideH - contentY - margin
	contentW := slideW - 2*margin
	contentX := margin

	// Half-width for two-column
	colGap := inchToEMU(0.4)
	halfW := (contentW - colGap) / 2

	grpSpPr := `<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/>`

	header := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" `

	// Layout 1: Title (centered title + subtitle)
	layout1 := header + `type="title" preserve="1">
<p:cSld name="Title Slide">
<p:spTree>` + grpSpPr +
		layoutPlaceholderSp(2, "Title 1", "ctrTitle", "", slideW/4, slideH/5, slideW/2, inchToEMU(1.5), 44) +
		layoutPlaceholderSp(3, "Subtitle 2", "subTitle", "1", slideW/4, slideH/2-slideH/10, slideW/2, inchToEMU(1.2), 32) +
		`</p:spTree>
</p:cSld>
</p:sldLayout>`

	// Layout 2: Title + Content
	layout2 := header + `type="obj" preserve="1">
<p:cSld name="Title and Content">
<p:spTree>` + grpSpPr +
		layoutPlaceholderSp(2, "Title 1", "title", "", titleX, titleY, titleW, titleH, 44) +
		layoutPlaceholderSp(3, "Content Placeholder 2", "body", "", contentX, contentY, contentW, contentH, 28) +
		`</p:spTree>
</p:cSld>
</p:sldLayout>`

	// Layout 3: Two Content (title + left body + right body)
	layout3 := header + `type="twoObj" preserve="1">
<p:cSld name="Two Content">
<p:spTree>` + grpSpPr +
		layoutPlaceholderSp(2, "Title 1", "title", "", titleX, titleY, titleW, titleH, 44) +
		layoutPlaceholderSp(3, "Left Content 2", "body", "1", contentX, contentY, halfW, contentH, 28) +
		layoutPlaceholderSp(4, "Right Content 3", "body", "2", contentX+halfW+colGap, contentY, halfW, contentH, 28) +
		`</p:spTree>
</p:cSld>
</p:sldLayout>`

	// Layout 4: Section (centered large title)
	layout4 := header + `type="secHead" preserve="1">
<p:cSld name="Section Header">
<p:spTree>` + grpSpPr +
		layoutPlaceholderSp(2, "Title 1", "ctrTitle", "", slideW/6, slideH/4, 2*slideW/3, inchToEMU(2.0), 60) +
		layoutPlaceholderSp(3, "Text Placeholder 2", "body", "1", slideW/6, slideH/2+inchToEMU(0.3), 2*slideW/3, inchToEMU(1.0), 28) +
		`</p:spTree>
</p:cSld>
</p:sldLayout>`

	// Layout 5: Blank
	layout5 := header + `type="blank" preserve="1">
<p:cSld name="Blank">
<p:spTree>` + grpSpPr + `</p:spTree>
</p:cSld>
</p:sldLayout>`

	return []string{layout1, layout2, layout3, layout4, layout5}
}

// writeSlideLayoutRels writes rels files for all 5 slide layouts.
func (b *Builder) writeSlideLayoutRels(zw *zip.Writer) error {
	for i := 1; i <= numLayouts; i++ {
		path := fmt.Sprintf("ppt/slideLayouts/_rels/slideLayout%d.xml.rels", i)
		w, err := zw.Create(path)
		if err != nil {
			return err
		}

		xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`

		_, err = w.Write([]byte(xml))
		if err != nil {
			return err
		}
	}
	return nil
}

// writeSlideRels writes ppt/slides/_rels/slideN.xml.rels.
func (b *Builder) writeSlideRels(zw *zip.Writer, slideNum int, slide *model.Slide) error {
	path := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)
	w, err := zw.Create(path)
	if err != nil {
		return err
	}

	var mediaRels strings.Builder

	// Background image relationship (if any)
	if bgAsset := b.bgImageBySlide[slideNum-1]; bgAsset != nil {
		fmt.Fprintf(&mediaRels, `<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>`, bgAsset.relID, bgAsset.fileName)
	}

	// Notes slide relationship
	if slide.Notes != "" {
		fmt.Fprintf(&mediaRels, `<Relationship Id="rIdNotes%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide" Target="../notesSlides/notesSlide%d.xml"/>`, slideNum, slideNum)
	}

	for _, elem := range slide.Elements {
		if asset := b.mediaByElement[elem]; asset != nil {
			// Determine relationship type based on element type
			relType := "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
			if elem.Type == model.ElementVideo {
				relType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/video"
			} else if elem.Type == model.ElementAudio {
				relType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/audio"
			}
			fmt.Fprintf(&mediaRels, `<Relationship Id="%s" Type="%s" Target="../media/%s"/>`, asset.relID, relType, asset.fileName)
		}
		// Poster image for video/audio
		if posterAsset := b.posterByElement[elem]; posterAsset != nil {
			fmt.Fprintf(&mediaRels, `<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>`, posterAsset.relID, posterAsset.fileName)
		}
		if asset := b.chartByElement[elem]; asset != nil {
			fmt.Fprintf(&mediaRels, `<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart" Target="../charts/chart%d.xml"/>`, asset.relID, asset.index)
		}
	}
	layoutNum := layoutNumberForSlide(slide.Layout)

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout` + fmt.Sprintf("%d", layoutNum) + `.xml"/>` + mediaRels.String() + `
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}
