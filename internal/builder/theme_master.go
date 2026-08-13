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

// writeSlideMaster writes a minimal slide master.
func (b *Builder) writeSlideMaster(zw *zip.Writer) error {
	w, err := zw.Create("ppt/slideMasters/slideMaster1.xml")
	if err != nil {
		return err
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
<p:sldLayoutIdLst><p:sldLayoutId id="2147483649" r:id="rId1"/></p:sldLayoutIdLst>
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
func (b *Builder) writeSlideMasterRels(zw *zip.Writer) error {
	w, err := zw.Create("ppt/slideMasters/_rels/slideMaster1.xml.rels")
	if err != nil {
		return err
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}

// writeSlideLayout writes a minimal slide layout.
func (b *Builder) writeSlideLayout(zw *zip.Writer) error {
	w, err := zw.Create("ppt/slideLayouts/slideLayout1.xml")
	if err != nil {
		return err
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="blank" preserve="1">
<p:cSld name="Blank">
<p:spTree>
<p:nvGrpSpPr>
<p:cNvPr id="1" name=""/>
<p:cNvGrpSpPr/>
<p:nvPr/>
</p:nvGrpSpPr>
<p:grpSpPr/>
</p:spTree>
</p:cSld>
</p:sldLayout>`

	_, err = w.Write([]byte(xml))
	return err
}

// writeSlideLayoutRels writes ppt/slideLayouts/_rels/slideLayout1.xml.rels.
func (b *Builder) writeSlideLayoutRels(zw *zip.Writer) error {
	w, err := zw.Create("ppt/slideLayouts/_rels/slideLayout1.xml.rels")
	if err != nil {
		return err
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}

// writeSlideRels writes ppt/slides/_rels/slideN.xml.rels.
func (b *Builder) writeSlideRels(zw *zip.Writer, slideNum int, slide *model.Slide) error {
	path := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)
	w, err := zw.Create(path)
	if err != nil {
		return err
	}

	var mediaRels strings.Builder
	for _, elem := range slide.Elements {
		if asset := b.mediaByElement[elem]; asset != nil {
			fmt.Fprintf(&mediaRels, `<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>`, asset.relID, asset.fileName)
		}
	}
	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>` + mediaRels.String() + `
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}
