package builder

import (
	"archive/zip"
	"fmt"
)

// writePresentation writes ppt/presentation.xml — the main presentation part.
func (b *Builder) writePresentation(zw *zip.Writer) error {
	w, err := zw.Create("ppt/presentation.xml")
	if err != nil {
		return err
	}

	slideW := inchToEMU(b.pres.SlideWidth)
	slideH := inchToEMU(b.pres.SlideHeight)

	sldIDs := ""
	for i := range b.pres.Slides {
		sldIDs += fmt.Sprintf(
			`<p:sldId id="%d" r:id="rId%d"/>`,
			256+i, i+2) // rId1=master, slides start at rId2
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats-officedocument.presentationml/2006/main" saveSubsetFonts="1">
<p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst>
<p:sldIdLst>%s</p:sldIdLst>
<p:sldSz cx="%d" cy="%d" type="screen16x9"/>
<p:notesSz cx="6858000" cy="9144000"/>
<p:defaultTextStyle/>
</p:presentation>`, sldIDs, slideW, slideH)

	_, err = w.Write([]byte(xml))
	return err
}

// writePresentationRels writes ppt/_rels/presentation.xml.rels.
func (b *Builder) writePresentationRels(zw *zip.Writer) error {
	w, err := zw.Create("ppt/_rels/presentation.xml.rels")
	if err != nil {
		return err
	}

	slideRels := ""
	for i := range b.pres.Slides {
		slideRels += fmt.Sprintf(
			`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`,
			i+2, i+1)
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>%s
</Relationships>`, slideRels)

	_, err = w.Write([]byte(xml))
	return err
}
