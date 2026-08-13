package builder

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/fonts"
)

// embeddedFont tracks a font that will be embedded in the PPTX package.
type embeddedFont struct {
	relID       string // rId for presentation.xml.rels
	fontPart    string // e.g. "ppt/fonts/font1.fntdata"
	fontData    []byte
	familyName  string
	psName      string // PostScript name
	regular     bool
	bold        bool
	italic      bool
	boldItalic  bool
}

// prepareEmbeddedFonts scans the presentation for font names and tries to
// match them against the font registry. Returns the list of fonts to embed.
func (b *Builder) prepareEmbeddedFonts() []embeddedFont {
	registry := fonts.GetRegistry()

	// Collect all unique font names used in the presentation
	usedFonts := make(map[string]bool)
	if b.pres.Theme.TitleFont != "" {
		usedFonts[b.pres.Theme.TitleFont] = true
	}
	if b.pres.Theme.BodyFont != "" {
		usedFonts[b.pres.Theme.BodyFont] = true
	}

	for _, slide := range b.pres.Slides {
		for _, elem := range slide.Elements {
			if elem.Style.FontName != "" {
				usedFonts[elem.Style.FontName] = true
			}
			if elem.Shape != nil && elem.Shape.Style.FontName != "" {
				usedFonts[elem.Shape.Style.FontName] = true
			}
			for _, para := range elem.Paragraphs {
				if para.Style.FontName != "" {
					usedFonts[para.Style.FontName] = true
				}
			}
		}
	}

	var result []embeddedFont
	fontIndex := 1
	relIndex := 100 // Start high to avoid collision with slide rels

	for fontName := range usedFonts {
		entry := registry.Lookup(fontName)
		if entry == nil {
			continue // Font not in registry — will rely on system fonts
		}

		data, err := os.ReadFile(entry.Path)
		if err != nil {
			continue
		}

		ef := embeddedFont{
			relID:      fmt.Sprintf("rId%d", relIndex),
			fontPart:   fmt.Sprintf("ppt/fonts/font%d.fntdata", fontIndex),
			fontData:   data,
			familyName: entry.Name,
			psName:     entry.PostScriptName,
			regular:    !entry.Bold && !entry.Italic,
			bold:       entry.Bold,
			italic:     entry.Italic,
		}
		if ef.psName == "" {
			ef.psName = entry.Name
		}

		result = append(result, ef)
		fontIndex++
		relIndex++
	}

	return result
}

// writeEmbeddedFonts writes the font .fntdata parts into the ZIP.
func (b *Builder) writeEmbeddedFonts(zw *zip.Writer, embeds []embeddedFont) error {
	for _, ef := range embeds {
		w, err := zw.Create(ef.fontPart)
		if err != nil {
			return err
		}
		if _, err := w.Write(ef.fontData); err != nil {
			return err
		}
	}
	return nil
}

// embeddedFontContentTypesXML returns the <Default> entries for font parts.
func embeddedFontContentTypesXML(embeds []embeddedFont) string {
	if len(embeds) == 0 {
		return ""
	}
	return `<Default Extension="fntdata" ContentType="application/x-fontdata"/>`
}

// embeddedFontRelationsXML returns the <Relationship> entries for presentation.xml.rels.
func embeddedFontRelationsXML(embeds []embeddedFont) string {
	if len(embeds) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, ef := range embeds {
		relPart := strings.TrimPrefix(ef.fontPart, "ppt/")
		fmt.Fprintf(&sb,
			`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/font" Target="%s"/>`,
			ef.relID, relPart)
	}
	return sb.String()
}

// embeddedFontLstXML returns the <p:embeddedFontLst> XML for presentation.xml.
func embeddedFontLstXML(embeds []embeddedFont) string {
	if len(embeds) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<p:embeddedFontLst>`)
	for _, ef := range embeds {
		sb.WriteString(fmt.Sprintf(
			`<p:embeddedFont><p:font typeface="%s"/>`, xmlEscape(ef.familyName)))
		if ef.regular {
			sb.WriteString(fmt.Sprintf(
				`<p:regular r:id="%s"/>`, ef.relID))
		}
		if ef.bold {
			sb.WriteString(fmt.Sprintf(
				`<p:bold r:id="%s"/>`, ef.relID))
		}
		if ef.italic {
			sb.WriteString(fmt.Sprintf(
				`<p:italic r:id="%s"/>`, ef.relID))
		}
		if ef.boldItalic {
			sb.WriteString(fmt.Sprintf(
				`<p:boldItalic r:id="%s"/>`, ef.relID))
		}
		sb.WriteString(`</p:embeddedFont>`)
	}
	sb.WriteString(`</p:embeddedFontLst>`)
	return sb.String()
}

// EmbedFontDirectory sets the fonts directory path for the global registry.
// Call this before building to ensure fonts from the correct location are used.
func EmbedFontDirectory(fontsDir string) error {
	abs, err := filepath.Abs(fontsDir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		// Reset the global registry to the specified directory
		reg := fonts.NewRegistry(abs)
		if _, err := reg.Scan(); err != nil {
			return err
		}
		// Replace global instance
		fonts.SetRegistry(reg)
	}
	return nil
}
