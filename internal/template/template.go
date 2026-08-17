// Package template imports design DNA (palette, fonts, slide size, master/layout
// inventory) from an existing .pptx file so a new deck can inherit its look.
//
// V1 scope: read-only extraction of the theme color scheme, font scheme,
// slide dimensions, and master/layout inventory. The extracted theme maps
// onto model.Theme so it can be applied with SetTheme — no low-level
// OOXML part replacement yet (a future version may splice raw parts).
package template

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

// LayoutInfo describes one slide layout found in the package.
type LayoutInfo struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // sldLayout @type: title, obj, secHead, blank...
}

// Extracted is the design summary pulled from a template .pptx.
type Extracted struct {
	Theme      model.Theme `json:"theme"`
	SlideWidth float64     `json:"slide_width"`  // inches
	SlideHeight float64    `json:"slide_height"` // inches
	Layouts    []LayoutInfo `json:"layouts"`
	// Warnings lists non-fatal issues encountered while parsing.
	Warnings []string `json:"warnings,omitempty"`
}

// ParseFile opens a .pptx file and extracts the design summary.
func ParseFile(pptxPath string) (*Extracted, error) {
	zr, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	defer zr.Close()
	return parseZip(&zr.Reader)
}

// ParseBytes extracts the design summary from an in-memory .pptx.
func ParseBytes(data []byte) (*Extracted, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	return parseZip(zr)
}

func parseZip(z *zip.Reader) (*Extracted, error) {
	out := &Extracted{
		SlideWidth:  13.333,
		SlideHeight: 7.5,
	}

	var themeRaw, presRaw []byte
	layoutFiles := map[string][]byte{} // basename -> content

	for _, f := range z.File {
		base := path.Base(f.Name)
		switch {
		case f.Name == "ppt/theme/theme1.xml":
			themeRaw = mustRead(f)
		case f.Name == "ppt/presentation.xml":
			presRaw = mustRead(f)
		case strings.HasPrefix(f.Name, "ppt/slideLayouts/") && strings.HasSuffix(f.Name, ".xml"):
			layoutFiles[base] = mustRead(f)
		}
	}

	if themeRaw == nil {
		return nil, fmt.Errorf("not a pptx template: ppt/theme/theme1.xml missing")
	}

	theme, themeWarn := parseThemeXML(themeRaw)
	out.Theme = theme
	if themeWarn != "" {
		out.Warnings = append(out.Warnings, themeWarn)
	}

	if presRaw != nil {
		w, h, err := parseSlideSize(presRaw)
		if err == nil {
			out.SlideWidth, out.SlideHeight = w, h
		} else {
			out.Warnings = append(out.Warnings, err.Error())
		}
	}

	// Layout inventory, sorted by numeric suffix (slideLayout1, 2, ...).
	for _, name := range sortedLayoutNames(layoutFiles) {
		info := parseLayoutInfo(layoutFiles[name])
		out.Layouts = append(out.Layouts, info)
	}

	return out, nil
}

func mustRead(f *zip.File) []byte {
	r, err := f.Open()
	if err != nil {
		return nil
	}
	defer r.Close()
	data, _ := io.ReadAll(r)
	return data
}

// ---------- theme1.xml parsing ----------

type xmlColorScheme struct {
	XMLName xml.Name `xml:"clrScheme"`
	Name    string   `xml:"name,attr"`
	DK1     *xmlSrgb `xml:"dk1>srgbClr"`
	LT1     *xmlSrgb `xml:"lt1>srgbClr"`
	DK2     *xmlSrgb `xml:"dk2>srgbClr"`
	LT2     *xmlSrgb `xml:"lt2>srgbClr"`
	Accent1 *xmlSrgb `xml:"accent1>srgbClr"`
	Accent2 *xmlSrgb `xml:"accent2>srgbClr"`
	Accent3 *xmlSrgb `xml:"accent3>srgbClr"`
	Accent4 *xmlSrgb `xml:"accent4>srgbClr"`
	Accent5 *xmlSrgb `xml:"accent5>srgbClr"`
	Accent6 *xmlSrgb `xml:"accent6>srgbClr"`
}

type xmlFontScheme struct {
	XMLName  xml.Name `xml:"fontScheme"`
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

// parseThemeXML maps a DrawingML theme onto model.Theme.
//
// Role mapping mirrors writeTheme: primary = accent1, accent = accent2,
// secondary = dk2, background = lt1, text = dk1. If dk1/lt1 use sysClr
// (windowText/window) instead of srgbClr they arrive empty and fall back
// to sensible neutrals.
func parseThemeXML(raw []byte) (model.Theme, string) {
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

	theme := model.Theme{
		Name:            doc.Name,
		PrimaryColor:    c(doc.Colors.Accent1, "#2563EB"),
		SecondaryColor:  c(doc.Colors.DK2, "#0F172A"),
		AccentColor:     c(doc.Colors.Accent2, "#F59E0B"),
		BackgroundColor: c(doc.Colors.LT1, "#FFFFFF"),
		TextColor:       c(doc.Colors.DK1, "#0F172A"),
	}
	if theme.Name == "" {
		theme.Name = "Imported Template"
	}
	if doc.Fonts.MajorLat != nil && doc.Fonts.MajorLat.Typeface != "" && doc.Fonts.MajorLat.Typeface != "+mj-lt" {
		theme.TitleFont = doc.Fonts.MajorLat.Typeface
	}
	if doc.Fonts.MinorLat != nil && doc.Fonts.MinorLat.Typeface != "" && doc.Fonts.MinorLat.Typeface != "+mn-lt" {
		theme.BodyFont = doc.Fonts.MinorLat.Typeface
	}
	return theme, ""
}

// ---------- presentation.xml parsing ----------

type xmlPresentation struct {
	SlideSize *xmlSlideSize `xml:"sldSz"`
}

type xmlSlideSize struct {
	Cx int64 `xml:"cx,attr"`
	Cy int64 `xml:"cy,attr"`
}

func parseSlideSize(raw []byte) (w, h float64, err error) {
	var doc xmlPresentation
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return 0, 0, fmt.Errorf("presentation parse: %w", err)
	}
	if doc.SlideSize == nil || doc.SlideSize.Cx == 0 || doc.SlideSize.Cy == 0 {
		return 0, 0, fmt.Errorf("slide size missing in presentation.xml")
	}
	// EMU → inches
	return float64(doc.SlideSize.Cx) / 914400.0, float64(doc.SlideSize.Cy) / 914400.0, nil
}

// ---------- slideLayoutN.xml parsing ----------

type xmlLayoutDoc struct {
	Type string `xml:"type,attr"`
	Name string `xml:"cSld>name,attr"`
}

func parseLayoutInfo(raw []byte) LayoutInfo {
	var doc xmlLayoutDoc
	info := LayoutInfo{}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return info
	}
	info.Type = doc.Type
	info.Name = doc.Name
	return info
}

func sortedLayoutNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sortLayoutNames(names)
	return names
}

// layoutNum extracts the trailing digits from a layout file name
// ("slideLayout12.xml" → 12). Returns -1 when no digits are present.
func layoutNum(name string) int {
	trimmed := strings.TrimSuffix(name, ".xml")
	i := len(trimmed)
	for i > 0 && trimmed[i-1] >= '0' && trimmed[i-1] <= '9' {
		i--
	}
	if i == len(trimmed) {
		return -1
	}
	n, err := strconv.Atoi(trimmed[i:])
	if err != nil {
		return -1
	}
	return n
}

// sortLayoutNames orders layout file names numerically by suffix so
// "slideLayout10.xml" sorts after "slideLayout9.xml".
func sortLayoutNames(names []string) {
	sort.Slice(names, func(a, b int) bool {
		na, nb := layoutNum(names[a]), layoutNum(names[b])
		if na == -1 || nb == -1 {
			return names[a] < names[b]
		}
		if na != nb {
			return na < nb
		}
		return names[a] < names[b]
	})
}
