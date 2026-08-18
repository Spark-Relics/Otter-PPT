// Package viz provides a Go-native visualization template library: canonical,
// standalone-renderable SVG skeletons for the most common charts and tables,
// plus "Pick for / Skip if" selection rules. This is the piece borrowed from
// ppt-master that made the biggest quality difference: the AI no longer draws
// charts from scratch, it adapts a neutral, well-composed skeleton.
//
// Every template is a complete SVG document with:
//   - viewBox="0 0 1280 720" and a white preview background
//   - neutral reference colors (slate greys + semantic green/red/amber)
//   - data-pptx-bounds on logical regions
//   - data-pptx-replace-with="chart" plus embedded metadata JSON where a native
//     PowerPoint chart/table can replace the shape fallback
//   - a chart-plot-area comment carrying the plot rectangle
//
// The svgdecode compiler consumes these markers; the import_svg handler then
// tells the agent when a native-object upgrade is available.
package viz

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/charts/*.svg templates/tables/*.svg
var templatesFS embed.FS

// Entry is one catalog item.
type Entry struct {
	Key     string `json:"key"`
	Kind    string `json:"kind"` // chart | table
	Summary string `json:"summary"`
}

// Get returns the canonical SVG source for a key plus its kind and selection
// rule. Key matching is case-insensitive and accepts common aliases.
func Get(key string) (svg string, kind string, summary string, ok bool) {
	key = canonicalKey(key)
	e, exists := index[key]
	if !exists {
		return "", "", "", false
	}
	path := fmt.Sprintf("templates/%ss/%s.svg", e.Kind, key)
	b, err := templatesFS.ReadFile(path)
	if err != nil {
		return "", "", "", false
	}
	return string(b), e.Kind, e.Summary, true
}

// PreviewHint returns the authoring guidance appended to a returned template.
func PreviewHint(kind string) string {
	if kind == "chart" {
		return "Neutral reference colors only. Re-theme to the active design lock, keep the value-to-mark mapping, and preserve data-pptx-bounds / data-pptx-replace-with / chart-plot-area markers."
	}
	return "Neutral reference colors only. Re-theme to the active design lock, keep the full row/column topology, and preserve data-pptx-bounds / data-pptx-replace-with markers."
}

// IsTemplateKey reports whether key resolves to a library template.
func IsTemplateKey(key string) bool {
	_, _, _, ok := Get(key)
	return ok
}

// trimIndent is a tiny helper for tests and inline display.
func trimIndent(s string) string {
	return strings.TrimSpace(s)
}
