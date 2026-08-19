// Package parser performs reverse parsing: .pptx → model.Presentation.
//
// It is the inverse of internal/builder: reads the OPC zip package,
// walks ppt/presentation.xml, ppt/theme/theme1.xml, ppt/slides/slideN.xml
// (plus notes and relationships), and reconstructs an editable
// model.Presentation in percentage coordinates.
//
// Supported element types (round-trip with builder output):
//   - text boxes (title/subtitle/body via name heuristic) with runs,
//     font size/color/bold/italic/underline/strike/align/line spacing
//   - bullet lists (paragraphs with buChar) and plain multi-paragraph bodies
//   - shapes (preset geometry, fill/line, rotation, text inside shape)
//   - freeform shapes (custGeom contours)
//   - pictures (embedded media referenced as package paths)
//   - tables (text grid; merged cells noted as warnings)
//   - charts (graphicFrame chart rel → chart XML: type, categories,
//     series with cached values, legend/data-labels flags)
//   - connectors (cxnSp with straight connector geometry)
//   - slide backgrounds: solid color and gradient fills
//   - speaker notes (notesSlide body placeholder)
//
// Unsupported constructs are skipped with a Warning instead of failing,
// so partial decks still load.
package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// Result is the outcome of parsing a .pptx package.
type Result struct {
	Presentation *model.Presentation
	// Warnings lists non-fatal issues (skipped elements, approximations).
	Warnings []string
}

// ParseFile opens a .pptx file and parses it into a model.Presentation.
func ParseFile(pptxPath string) (*Result, error) {
	zr, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	defer zr.Close()
	return parseZip(&zr.Reader)
}

// ParseBytes parses an in-memory .pptx.
func ParseBytes(data []byte) (*Result, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	return parseZip(zr)
}

// opcPackage is a fully-read OPC zip package.
type opcPackage struct {
	files map[string][]byte // full path -> content
}

func (p *opcPackage) has(name string) bool { _, ok := p.files[name]; return ok }

// relsFor parses "xxx/_rels/<base>.xml.rels" for the part at partPath and
// returns rId → package-absolute target path.
func (p *opcPackage) relsFor(partPath string) map[string]string {
	dir := path.Dir(partPath)
	base := path.Base(partPath)
	relsPath := dir + "/_rels/" + base + ".rels"
	raw, ok := p.files[relsPath]
	if !ok {
		return nil
	}
	var doc struct {
		Relationships []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if xml.Unmarshal(raw, &doc) != nil {
		return nil
	}
	m := make(map[string]string, len(doc.Relationships))
	for _, r := range doc.Relationships {
		m[r.ID] = normalizeTarget(dir, r.Target)
	}
	return m
}

// normalizeTarget resolves a rel target (possibly ../media/x.png) against
// the part directory into a package-absolute path.
func normalizeTarget(partDir string, target string) string {
	target = strings.TrimPrefix(target, "/")
	if strings.HasPrefix(target, "../") {
		dir := partDir
		for strings.HasPrefix(target, "../") {
			dir = path.Dir(dir)
			target = strings.TrimPrefix(target, "../")
		}
		return path.Join(dir, target)
	}
	return path.Join(partDir, target)
}

func parseZip(z *zip.Reader) (*Result, error) {
	pkg := &opcPackage{files: make(map[string][]byte)}
	for _, f := range z.File {
		if f.FileInfo().IsDir() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(r)
		r.Close()
		pkg.files[f.Name] = data
	}

	if !pkg.has("ppt/presentation.xml") {
		return nil, fmt.Errorf("not a pptx package: ppt/presentation.xml missing")
	}

	res := &Result{}
	pres := &model.Presentation{
		Theme:       defaultTheme(),
		SlideWidth:  13.333,
		SlideHeight: 7.5,
	}

	// Slide size + slide id list from presentation.xml.
	presRaw := pkg.files["ppt/presentation.xml"]
	var presDoc xmlPresentation
	if err := xml.Unmarshal(presRaw, &presDoc); err == nil && presDoc.SldSz.CX > 0 {
		pres.SlideWidth = float64(presDoc.SldSz.CX) / 914400
		pres.SlideHeight = float64(presDoc.SldSz.CY) / 914400
	}

	// Theme.
	if themeRaw, ok := pkg.files["ppt/theme/theme1.xml"]; ok {
		if t, warn := parseTheme(themeRaw); t.Name != "" {
			pres.Theme = t
			if warn != "" {
				res.Warnings = append(res.Warnings, warn)
			}
		}
	}

	// Slides in presentation order.
	slideNames := orderedSlideNames(pkg, presDoc)
	if len(slideNames) == 0 {
		return nil, fmt.Errorf("no slides found in package")
	}

	for i, slideName := range slideNames {
		slide, slideWarnings := parseSlide(pkg, slideName, pres.SlideWidth, pres.SlideHeight)
		for _, w := range slideWarnings {
			res.Warnings = append(res.Warnings, fmt.Sprintf("slide %d: %s", i+1, w))
		}
		pres.Slides = append(pres.Slides, slide)
	}

	// Presentation title: first title-ish text found.
	for _, sl := range pres.Slides {
		for _, el := range sl.Elements {
			if el.Type == model.ElementTitle && strings.TrimSpace(el.Text) != "" {
				pres.Title = el.Text
				break
			}
		}
		if pres.Title != "" {
			break
		}
	}

	res.Presentation = pres
	return res, nil
}

// ---------- presentation.xml ----------

type xmlPresentation struct {
	SldIDs []struct {
		RID string `xml:"id,attr"`
	} `xml:"sldIdLst>sldId"`
	SldSz struct {
		CX int `xml:"cx,attr"`
		CY int `xml:"cy,attr"`
	} `xml:"sldSz"`
}

// orderedSlideNames returns slide part names in presentation order.
func orderedSlideNames(pkg *opcPackage, presDoc xmlPresentation) []string {
	rels := pkg.relsFor("ppt/presentation.xml")
	var names []string
	for _, s := range presDoc.SldIDs {
		target, ok := rels[s.RID]
		if !ok {
			continue
		}
		names = append(names, target)
	}
	if len(names) == 0 {
		// Fallback: numeric order of ppt/slides/slideN.xml.
		for name := range pkg.files {
			if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
				names = append(names, name)
			}
		}
		sort.Slice(names, func(i, j int) bool {
			return slideIndex(names[i]) < slideIndex(names[j])
		})
	}
	return names
}

// slideIndex extracts the numeric suffix of ppt/slides/slideN.xml.
func slideIndex(name string) int {
	s := strings.TrimSuffix(path.Base(name), ".xml")
	s = strings.TrimPrefix(s, "slide")
	n, _ := strconv.Atoi(s)
	return n
}

// defaultTheme mirrors template.go fallbacks.
func defaultTheme() model.Theme {
	return model.Theme{
		Name:            "Imported",
		PrimaryColor:    "#2563EB",
		SecondaryColor:  "#0F172A",
		AccentColor:     "#F59E0B",
		BackgroundColor: "#FFFFFF",
		TextColor:       "#0F172A",
	}
}

// ---------- theme1.xml ----------

type xmlColorScheme struct {
	DK1     *xmlSrgb `xml:"dk1>srgbClr"`
	LT1     *xmlSrgb `xml:"lt1>srgbClr"`
	DK2     *xmlSrgb `xml:"dk2>srgbClr"`
	LT2     *xmlSrgb `xml:"lt2>srgbClr"`
	Accent1 *xmlSrgb `xml:"accent1>srgbClr"`
	Accent2 *xmlSrgb `xml:"accent2>srgbClr"`
}

type xmlFontScheme struct {
	MajorLat *xmlFont `xml:"majorFont>latin"`
	MinorLat *xmlFont `xml:"minorFont>latin"`
}

type xmlSrgb struct {
	Val string `xml:"val,attr"`
}

type xmlFont struct {
	Typeface string `xml:"typeface,attr"`
}

type xmlThemeDoc struct {
	Colors xmlColorScheme `xml:"themeElements>clrScheme"`
	Fonts  xmlFontScheme  `xml:"themeElements>fontScheme"`
	Name   string         `xml:"name,attr"`
}

// parseTheme maps a DrawingML theme onto model.Theme using the same role
// mapping as internal/template (inverse of builder.writeTheme).
func parseTheme(raw []byte) (model.Theme, string) {
	var doc xmlThemeDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return model.Theme{}, fmt.Sprintf("theme parse: %v", err)
	}

	c := func(s *xmlSrgb, fallback string) string {
		if s == nil || s.Val == "" {
			return fallback
		}
		return "#" + strings.ToUpper(strings.TrimPrefix(s.Val, "#"))
	}
	f := func(x *xmlFont, fallback string) string {
		if x == nil || x.Typeface == "" {
			return fallback
		}
		return x.Typeface
	}

	theme := model.Theme{
		Name:            doc.Name,
		PrimaryColor:    c(doc.Colors.Accent1, "#2563EB"),
		SecondaryColor:  c(doc.Colors.DK2, "#0F172A"),
		AccentColor:     c(doc.Colors.Accent2, "#F59E0B"),
		BackgroundColor: c(doc.Colors.LT1, "#FFFFFF"),
		TextColor:       c(doc.Colors.DK1, "#0F172A"),
		TitleFont:       f(doc.Fonts.MajorLat, ""),
		BodyFont:        f(doc.Fonts.MinorLat, ""),
	}
	if theme.Name == "" {
		theme.Name = "Imported"
	}
	return theme, ""
}
