package builder

import (
	"archive/zip"
	"fmt"
)

// hasNotes reports whether any slide carries speaker notes, in which case
// the package must contain a notesMaster part (required by PowerPoint).
func (b *Builder) hasNotes() bool {
	for _, slide := range b.pres.Slides {
		if slide.Notes != "" {
			return true
		}
	}
	return false
}

// writeNotesMaster writes ppt/notesMasters/notesMaster1.xml.
// Required whenever notesSlide parts exist; PowerPoint refuses to open the
// package without it ("needs repair").
func (b *Builder) writeNotesMaster(zw *zip.Writer) error {
	if !b.hasNotes() {
		return nil
	}
	w, err := zw.Create("ppt/notesMasters/notesMaster1.xml")
	if err != nil {
		return err
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:notesMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
<p:cSld>
<p:bg><p:bgRef idx="1001"><a:schemeClr val="bg1"/></p:bgRef></p:bg>
<p:spTree>
<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
<p:grpSpPr/>
<p:sp><p:nvSpPr><p:cNvPr id="2" name="Slide Image Placeholder"/><p:cNvSpPr><a:spLocks noGrp="1" noRot="1" noChangeAspect="1"/></p:cNvSpPr><p:nvPr><p:ph type="sldImg"/></p:nvPr></p:nvSpPr>
<p:spPr/></p:sp>
<p:sp><p:nvSpPr><p:cNvPr id="3" name="Notes Placeholder"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr><p:ph type="body" idx="1"/></p:nvPr></p:nvSpPr>
<p:spPr/></p:sp>
<p:sp><p:nvSpPr><p:cNvPr id="4" name="Slide Number Placeholder"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr><p:nvPr><p:ph type="sldNum" sz="quarter" idx="10"/></p:nvPr></p:nvSpPr>
<p:spPr/></p:sp>
</p:spTree>
</p:cSld>
<p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
<p:notesStyle>
<a:lvl1pPr marL="0" indent="0" defTabSz="914400" rtl="0" algn="l" eaLnBrk="1" latinLnBrk="0" hangingPunct="1">
<a:buNone/><a:defRPr sz="1200" kern="1200"><a:solidFill><a:schemeClr val="tx1"/></a:solidFill><a:latin typeface="+mn-lt"/><a:ea typeface="+mn-ea"/><a:cs typeface="+mn-cs"/></a:defRPr>
</a:lvl1pPr>
</p:notesStyle>
</p:notesMaster>`

	_, err = w.Write([]byte(xml))
	return err
}

// writeNotesMasterRels writes ppt/notesMasters/_rels/notesMaster1.xml.rels.
// rId1 = theme1.xml (the only relationship a notesMaster needs).
func (b *Builder) writeNotesMasterRels(zw *zip.Writer) error {
	if !b.hasNotes() {
		return nil
	}
	w, err := zw.Create("ppt/notesMasters/_rels/notesMaster1.xml.rels")
	if err != nil {
		return err
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}

// notesMasterRelID returns the rId for the notesMaster relationship in
// presentation.xml.rels. Layout: rId1=master, rId2..rId(N+1)=slides,
// next free = N+2 (embedded fonts use 100+).
func (b *Builder) notesMasterRelID() string {
	return fmt.Sprintf("rId%d", len(b.pres.Slides)+2)
}
