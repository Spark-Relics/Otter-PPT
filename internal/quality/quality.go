// Package quality implements the Go SVG/element quality checker borrowed
// from ppt-master's gate architecture: text-width estimation, module
// boundary (overflow) detection, out-of-bounds detection, and the cover /
// final gates. It checks compiled native elements (model.Element) rather
// than raw SVG, so it covers transforms, freeforms, and any other source
// of elements alike.
package quality

import (
	"fmt"
	"math"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// Severity levels for issues.
const (
	SeverityError = "error"
	SeverityWarn  = "warn"
)

// Issue kinds.
const (
	KindOverflowH   = "overflow_horizontal"
	KindOverflowV   = "overflow_vertical"
	KindOutOfBounds = "out_of_bounds"
	KindTinyText    = "tiny_text"
	KindCoverGate   = "cover_gate"
)

// Issue is one quality finding on one element of one slide.
type Issue struct {
	Slide     int    `json:"slide"` // 1-based
	ElementID string `json:"element_id,omitempty"`
	Kind      string `json:"kind"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

// Report aggregates issues with a pass/fail gate verdict and a score.
type Report struct {
	Pass   bool    `json:"pass"` // no error-severity issues
	Score  int     `json:"score"`
	Issues []Issue `json:"issues,omitempty"`
}

// Summary renders a compact one-line verdict.
func (r *Report) Summary() string {
	errs, warns := 0, 0
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			errs++
		} else {
			warns++
		}
	}
	return fmt.Sprintf("%d errors, %d warnings, score %d/100, gate %s",
		errs, warns, r.Score, map[bool]string{true: "PASS", false: "FAIL"}[r.Pass])
}

// ─────────────────────────────────────────────────────────────
// Text metrics
// ─────────────────────────────────────────────────────────────

// charWidthFrac estimates a rune's advance width as a fraction of font size.
// CJK/fullwidth glyphs occupy a full em; latin/digits are narrower.
func charWidthFrac(r rune) float64 {
	switch {
	case r >= 0x2E80 && r <= 0x9FFF, // CJK blocks
		r >= 0xAC00 && r <= 0xD7AF, // Hangul
		r >= 0x3000 && r <= 0x303F, // CJK punctuation
		r >= 0xFF00 && r <= 0xFFEF: // fullwidth forms
		return 1.0
	case r == ' ':
		return 0.30
	case r >= '0' && r <= '9':
		return 0.55
	case r >= 'A' && r <= 'Z':
		return 0.65
	default:
		return 0.50
	}
}

// EstimateTextWidth estimates the rendered width of a single line in points.
func EstimateTextWidth(text string, fontSize int) float64 {
	w := 0.0
	for _, r := range text {
		w += charWidthFrac(r) * float64(fontSize)
	}
	return w
}

// textLines estimates how many lines a text block needs inside availWidthPt
// (0 disables wrapping: each \n segment counts as one line).
func textLines(text string, fontSize int, availWidthPt float64) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	segments := strings.Split(text, "\n")
	lines := 0
	for _, seg := range segments {
		w := EstimateTextWidth(seg, fontSize)
		if availWidthPt <= 0 || w <= availWidthPt {
			lines++
			continue
		}
		lines += int(math.Ceil(w / availWidthPt))
	}
	if lines == 0 {
		lines = 1
	}
	return lines
}

// textHeightPt estimates the rendered height of a wrapped text block in points.
func textHeightPt(text string, fontSize int, availWidthPt, lineSpacing float64) float64 {
	if lineSpacing < 1.0 {
		lineSpacing = 1.2
	}
	return float64(textLines(text, fontSize, availWidthPt)) * float64(fontSize) * lineSpacing
}

// ─────────────────────────────────────────────────────────────
// Checks
// ─────────────────────────────────────────────────────────────

const defaultSlideWidthInches = 10.0
const defaultSlideHeightInches = 7.5
const defaultMarginPt = 5.0 // default text-box inner margins combined L+R, approximate

func slideDims(pres *model.Presentation) (wIn, hIn float64) {
	wIn, hIn = pres.SlideWidth, pres.SlideHeight
	if wIn <= 0 {
		wIn = defaultSlideWidthInches
	}
	if hIn <= 0 {
		if wIn > 11 {
			hIn = 7.5 // 16:9
		} else {
			hIn = defaultSlideHeightInches
		}
	}
	return
}

// textOf returns the effective text content of an element.
func textOf(e *model.Element) string {
	if e.Text != "" {
		return e.Text
	}
	if len(e.Items) > 0 {
		return strings.Join(e.Items, "\n")
	}
	if len(e.Paragraphs) > 0 {
		parts := make([]string, 0, len(e.Paragraphs))
		for _, p := range e.Paragraphs {
			if p.Text != "" {
				parts = append(parts, p.Text)
				continue
			}
			var sb strings.Builder
			for _, r := range p.Runs {
				sb.WriteString(r.Text)
			}
			parts = append(parts, sb.String())
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// effectiveFontSize resolves the font size of an element's first styled text.
func effectiveFontSize(e *model.Element) int {
	if e.Style.FontSize > 0 {
		return e.Style.FontSize
	}
	for _, p := range e.Paragraphs {
		if p.Style.FontSize > 0 {
			return p.Style.FontSize
		}
		for _, r := range p.Runs {
			if r.Style.FontSize > 0 {
				return r.Style.FontSize
			}
		}
	}
	return 18 // builder default body size
}

// outOfBounds reports elements whose rect leaves the 0-100% slide area.
func outOfBounds(slideNum int, elements []*model.Element) []Issue {
	var issues []Issue
	for _, e := range elements {
		r := e.Rect
		oob := r.X < -0.5 || r.Y < -0.5 || r.W <= 0 || r.H <= 0 ||
			r.X+r.W > 100.5 || r.Y+r.H > 100.5
		if !oob {
			continue
		}
		sev := SeverityError
		// Decorative shapes intentionally bleeding off-canvas are a common
		// design pattern — downgrade to warn when there is no text.
		if textOf(e) == "" {
			sev = SeverityWarn
		}
		issues = append(issues, Issue{
			Slide: slideNum, ElementID: e.ID, Kind: KindOutOfBounds, Severity: sev,
			Message: fmt.Sprintf("element %q rect (%.1f,%.1f %.1fx%.1f%%) exceeds slide bounds", e.ID, r.X, r.Y, r.W, r.H),
		})
	}
	return issues
}

// overflow detects text that cannot fit its box: horizontally when wrap is
// off, vertically when wrapped lines exceed the box height.
func overflow(slideNum int, e *model.Element, slideWIn, slideHIn float64) []Issue {
	text := textOf(e)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	size := effectiveFontSize(e)
	boxW := e.Rect.W / 100 * slideWIn * 72 // pt
	boxH := e.Rect.H / 100 * slideHIn * 72 // pt
	availW := boxW - defaultMarginPt

	var issues []Issue

	// Horizontal: only meaningful when wrapping is disabled.
	if e.Style.WordWrap != nil && !*e.Style.WordWrap {
		longest := 0.0
		for _, seg := range strings.Split(text, "\n") {
			if w := EstimateTextWidth(seg, size); w > longest {
				longest = w
			}
		}
		if availW > 0 && longest > availW*1.05 {
			issues = append(issues, Issue{
				Slide: slideNum, ElementID: e.ID, Kind: KindOverflowH, Severity: SeverityError,
				Message: fmt.Sprintf("text %.0fpt wide exceeds %.0fpt box (word_wrap off)", longest, boxW),
			})
		}
	}

	// Vertical: estimated wrapped height vs box height (15% tolerance).
	if boxH > 0 && availW > 0 {
		need := textHeightPt(text, size, availW, e.Style.LineSpacing)
		if need > boxH*1.15 {
			sev := SeverityWarn
			if need > boxH*1.5 {
				sev = SeverityError
			}
			issues = append(issues, Issue{
				Slide: slideNum, ElementID: e.ID, Kind: KindOverflowV, Severity: sev,
				Message: fmt.Sprintf("text needs ~%.0fpt height, box is %.0fpt (%d lines @%dpt)", need, boxH, textLines(text, size, availW), size),
			})
		}
	}

	return issues
}

// tinyText flags text below projection-readable sizes.
func tinyText(slideNum int, e *model.Element) []Issue {
	text := textOf(e)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	size := effectiveFontSize(e)
	switch {
	case size < 9:
		return []Issue{{
			Slide: slideNum, ElementID: e.ID, Kind: KindTinyText, Severity: SeverityError,
			Message: fmt.Sprintf("font %dpt below 9pt minimum (illegible when projected)", size),
		}}
	case size < 10:
		return []Issue{{
			Slide: slideNum, ElementID: e.ID, Kind: KindTinyText, Severity: SeverityWarn,
			Message: fmt.Sprintf("font %dpt is very small; prefer ≥10pt", size),
		}}
	}
	return nil
}

// CheckSlide runs all per-element checks on one slide (1-based number).
func CheckSlide(s *model.Slide, slideNum int, slideWIn, slideHIn float64) []Issue {
	var issues []Issue
	issues = append(issues, outOfBounds(slideNum, s.Elements)...)
	for _, e := range s.Elements {
		issues = append(issues, overflow(slideNum, e, slideWIn, slideHIn)...)
		issues = append(issues, tinyText(slideNum, e)...)
	}
	return issues
}

// CheckCover is the cover gate: the first slide must carry a hero-size
// headline (≥24pt) — the ppt-master "first page quality gate".
func CheckCover(pres *model.Presentation) []Issue {
	if len(pres.Slides) == 0 {
		return []Issue{{Slide: 1, Kind: KindCoverGate, Severity: SeverityError, Message: "deck has no slides"}}
	}
	cover := pres.Slides[0]
	hero := 0
	for _, e := range cover.Elements {
		t := textOf(e)
		if strings.TrimSpace(t) == "" {
			continue
		}
		if size := effectiveFontSize(e); size > hero {
			hero = size
		}
	}
	if hero < 24 {
		return []Issue{{
			Slide: 1, Kind: KindCoverGate, Severity: SeverityError,
			Message: fmt.Sprintf("cover has no headline ≥24pt (largest text is %dpt) — the first page needs a hero title", hero),
		}}
	}
	return nil
}

// Check runs all checks including gates over the whole presentation.
func Check(pres *model.Presentation) *Report {
	wIn, hIn := slideDims(pres)
	var issues []Issue
	for i, s := range pres.Slides {
		issues = append(issues, CheckSlide(s, i+1, wIn, hIn)...)
	}
	issues = append(issues, CheckCover(pres)...)

	r := &Report{Issues: issues}
	errs := 0
	for _, i := range issues {
		if i.Severity == SeverityError {
			errs++
		}
	}
	r.Pass = errs == 0
	r.Score = 100 - errs*8 - (len(issues)-errs)*3
	if r.Score < 0 {
		r.Score = 0
	}
	return r
}
