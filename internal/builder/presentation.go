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

	embeddedFonts := embeddedFontLstXML(b.embeddedFonts)

	notesMasterIDs := ""
	if b.hasNotes() {
		notesMasterIDs = fmt.Sprintf(
			`<p:notesMasterIdLst><p:notesMasterId id="2147483649" r:id="%s"/></p:notesMasterIdLst>`,
			b.notesMasterRelID())
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" saveSubsetFonts="1">
<p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst>
<p:sldIdLst>%s</p:sldIdLst>
%s
<p:sldSz cx="%d" cy="%d" type="screen16x9"/>
<p:notesSz cx="6858000" cy="9144000"/>
%s
<p:defaultTextStyle/>
</p:presentation>`, sldIDs, notesMasterIDs, slideW, slideH, embeddedFonts)

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

	fontRels := embeddedFontRelationsXML(b.embeddedFonts)

	notesMasterRels := ""
	if b.hasNotes() {
		notesMasterRels = fmt.Sprintf(
			`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster" Target="notesMasters/notesMaster1.xml"/>`,
			b.notesMasterRelID())
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>%s%s%s
</Relationships>`, slideRels, notesMasterRels, fontRels)

	_, err = w.Write([]byte(xml))
	return err
}
