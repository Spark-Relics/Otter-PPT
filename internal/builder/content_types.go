package builder

import (
	"archive/zip"
	"fmt"
)

// writeContentTypes writes [Content_Types].xml — the OPC manifest.
func (b *Builder) writeContentTypes(zw *zip.Writer) error {
	w, err := zw.Create("[Content_Types].xml")
	if err != nil {
		return err
	}

	slideOverrides := ""
	for i := range b.pres.Slides {
		slideOverrides += fmt.Sprintf(
			`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`,
			i+1)
	}
	chartOverrides := ""
	for _, asset := range b.chartAssets {
		chartOverrides += fmt.Sprintf(`<Override PartName="/ppt/charts/chart%d.xml" ContentType="application/vnd.openxmlformats-officedocument.drawingml.chart+xml"/>`, asset.index)
	}
	mediaDefaults := ""
	seen := make(map[string]bool)
	for _, asset := range b.mediaAssets {
		if !seen[asset.ext] {
			mediaDefaults += fmt.Sprintf(`<Default Extension="%s" ContentType="%s"/>`, asset.ext, mediaContentType(asset.ext))
			seen[asset.ext] = true
		}
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>` + mediaDefaults + `
<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
<Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
<Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
<Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>` +
		slideOverrides + chartOverrides +
		`</Types>`

	_, err = w.Write([]byte(xml))
	return err
}

// writeRels writes _rels/.rels — the root package relationship.
func (b *Builder) writeRels(zw *zip.Writer) error {
	w, err := zw.Create("_rels/.rels")
	if err != nil {
		return err
	}

	xml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`

	_, err = w.Write([]byte(xml))
	return err
}
