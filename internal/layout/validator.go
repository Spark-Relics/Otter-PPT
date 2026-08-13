// Package layout provides intelligent layout analysis, validation, auto-fix,
// and smart templates for PPT slides. It works with the model package to
// detect common design issues (overlaps, out-of-bounds, poor spacing) and
// either fix them automatically or suggest improvements to the AI agent.
package layout

import (
	"fmt"
	"sort"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// Severity rates how important an issue is.
type Severity string

const (
	SeverityError   Severity = "error"   // must fix: overlapping text, out-of-bounds
	SeverityWarning Severity = "warning" // should fix: tight spacing, inconsistent sizing
	SeverityInfo    Severity = "info"    // nice to have: whitespace imbalance
)

// Issue describes a single layout problem found on a slide.
type Issue struct {
	Severity   Severity `json:"severity"`
	Category   string   `json:"category"`    // overlap, bounds, spacing, title, density
	Message    string   `json:"message"`
	ElementIDs []string `json:"element_ids"` // involved element IDs
}

// SlideReport contains the analysis for one slide.
type SlideReport struct {
	SlideID  string  `json:"slide_id"`
	SlideNum int     `json:"slide_num"` // 1-based
	Issues   []Issue `json:"issues"`
	Score    float64 `json:"score"` // 0-100 quality score
}

// PresentationReport aggregates reports for all slides.
type PresentationReport struct {
	Slides []SlideReport `json:"slides"`
	Score  float64        `json:"score"` // average
}

// ──────────────────── Validation ────────────────────

// ValidateSlide analyzes one slide for layout issues and returns a report.
func ValidateSlide(slide *model.Slide, slideNum int) *SlideReport {
	r := &SlideReport{
		SlideID:  slide.ID,
		SlideNum: slideNum,
		Issues:   []Issue{},
		Score:    100,
	}

	if len(slide.Elements) == 0 {
		r.Issues = append(r.Issues, Issue{
			Severity: SeverityWarning,
			Category: "empty",
			Message:  "Slide has no elements",
		})
		r.Score = 30
		return r
	}

	for _, elem := range slide.Elements {
		// 1. Bounds check
		checkBounds(r, elem)
	}

	// 2. Overlap detection (only for text-bearing elements)
	checkOverlaps(r, slide)

	// 3. Title position guidance
	checkTitle(slide, r)

	// 4. Density check
	checkDensity(slide, r)

	// 5. Alignment / consistency checks
	checkAlignment(slide, r)

	// Compute score
	r.Score = computeScore(r)

	return r
}

// ValidatePresentation analyzes all slides and returns a full report.
func ValidatePresentation(pres *model.Presentation) *PresentationReport {
	report := &PresentationReport{Slides: []SlideReport{}}
	totalScore := 0.0
	for i, slide := range pres.Slides {
		sr := ValidateSlide(slide, i+1)
		report.Slides = append(report.Slides, *sr)
		totalScore += sr.Score
	}
	if len(report.Slides) > 0 {
		report.Score = totalScore / float64(len(report.Slides))
	}
	return report
}

// ──────────────────── Individual Checks ────────────────────

func checkBounds(r *SlideReport, elem *model.Element) {
	// Connectors use percentage coords directly, skip bounds for them
	if elem.Type == model.ElementConnector {
		return
	}
	rect := elem.Rect
	var issues []Issue
	if rect.X < 0 {
		issues = append(issues, Issue{
			Severity: SeverityError, Category: "bounds",
			Message:  fmt.Sprintf("Element %s starts off-screen left (x=%.1f)", elem.ID, rect.X),
			ElementIDs: []string{elem.ID},
		})
	}
	if rect.Y < 0 {
		issues = append(issues, Issue{
			Severity: SeverityError, Category: "bounds",
			Message:  fmt.Sprintf("Element %s starts off-screen top (y=%.1f)", elem.ID, rect.Y),
			ElementIDs: []string{elem.ID},
		})
	}
	if rect.X+rect.W > 100 {
		issues = append(issues, Issue{
			Severity: SeverityError, Category: "bounds",
			Message:  fmt.Sprintf("Element %s extends off-screen right (x+w=%.1f)", elem.ID, rect.X+rect.W),
			ElementIDs: []string{elem.ID},
		})
	}
	if rect.Y+rect.H > 100 {
		issues = append(issues, Issue{
			Severity: SeverityError, Category: "bounds",
			Message:  fmt.Sprintf("Element %s extends off-screen bottom (y+h=%.1f)", elem.ID, rect.Y+rect.H),
			ElementIDs: []string{elem.ID},
		})
	}
	// Margins: content should keep at least 3% from edges
	margin := 3.0
	if rect.X > 0 && rect.X < margin && elem.Type != model.ElementShape {
		issues = append(issues, Issue{
			Severity: SeverityWarning, Category: "bounds",
			Message:  fmt.Sprintf("Element %s is too close to left edge (x=%.1f, min=%.1f)", elem.ID, rect.X, margin),
			ElementIDs: []string{elem.ID},
		})
	}
	if rect.Y > 0 && rect.Y < margin && elem.Type != model.ElementShape {
		issues = append(issues, Issue{
			Severity: SeverityWarning, Category: "bounds",
			Message:  fmt.Sprintf("Element %s is too close to top edge (y=%.1f, min=%.1f)", elem.ID, rect.Y, margin),
			ElementIDs: []string{elem.ID},
		})
	}

	for _, iss := range issues {
		r.Issues = append(r.Issues, iss)
	}
}

func checkOverlaps(r *SlideReport, slide *model.Slide) {
	type textElem struct {
		elem *model.Element
	}
	var textElems []textElem
	for _, elem := range slide.Elements {
		if isTextBearing(elem) {
			textElems = append(textElems, textElem{elem: elem})
		}
	}
	for i := 0; i < len(textElems); i++ {
		for j := i + 1; j < len(textElems); j++ {
			a, b := textElems[i].elem, textElems[j].elem
			if rectsOverlap(a.Rect, b.Rect) {
				overlapArea := overlapRect(a.Rect, b.Rect)
				r.Issues = append(r.Issues, Issue{
					Severity:   SeverityError,
					Category:   "overlap",
					Message:    fmt.Sprintf("Text elements %s and %s overlap (area=%.1f%%)", a.ID, b.ID, overlapArea),
					ElementIDs: []string{a.ID, b.ID},
				})
			}
		}
	}
}

func checkTitle(slide *model.Slide, r *SlideReport) {
	var titles []*model.Element
	for _, elem := range slide.Elements {
		if elem.Type == model.ElementTitle {
			titles = append(titles, elem)
		}
	}
	if len(titles) == 0 {
		// It's acceptable for some slides (e.g. section dividers) to not have a title element
		return
	}
	if len(titles) > 1 {
		r.Issues = append(r.Issues, Issue{
			Severity: SeverityWarning,
			Category: "title",
			Message:  fmt.Sprintf("Slide has %d title elements; consider using one", len(titles)),
		})
	}
	for _, t := range titles {
		// Title should be in the top 40% of the slide
		if t.Rect.Y > 45 {
			r.Issues = append(r.Issues, Issue{
				Severity:   SeverityWarning,
				Category:   "title",
				Message:    fmt.Sprintf("Title element %s is positioned low (y=%.1f); titles should be in the top portion", t.ID, t.Rect.Y),
				ElementIDs: []string{t.ID},
			})
		}
	}
}

func checkDensity(slide *model.Slide, r *SlideReport) {
	count := len(slide.Elements)
	if count > 12 {
		r.Issues = append(r.Issues, Issue{
			Severity: SeverityWarning,
			Category: "density",
			Message:  fmt.Sprintf("Slide has %d elements; consider splitting into multiple slides for clarity", count),
		})
	}
	// Check for empty space utilization
	if count <= 2 && slide.Layout != model.LayoutTitle && slide.Layout != model.LayoutSection {
		r.Issues = append(r.Issues, Issue{
			Severity: SeverityInfo,
			Category: "density",
			Message:  "Slide has very few elements; consider adding visual content",
		})
	}
}

func checkAlignment(slide *model.Slide, r *SlideReport) {
	// Check if multiple text elements share the same left edge (left-align)
	type pos struct {
		id string
		x  float64
	}
	var positions []pos
	for _, elem := range slide.Elements {
		if isTextBearing(elem) {
			positions = append(positions, pos{elem.ID, elem.Rect.X})
		}
	}
	if len(positions) < 3 {
		return
	}
	// Group by similar X (within 2%)
	sort.Slice(positions, func(i, j int) bool { return positions[i].x < positions[j].x })
	groupStart := 0
	for i := 1; i <= len(positions); i++ {
		if i == len(positions) || positions[i].x-positions[groupStart].x > 2 {
			if i-groupStart >= 3 {
				// Good alignment found
				return
			}
			groupStart = i
		}
	}
	r.Issues = append(r.Issues, Issue{
		Severity: SeverityInfo,
		Category: "alignment",
		Message:  "Text elements are not aligned to a common left margin; left-aligning text improves readability",
	})
}

// ──────────────────── Score Computation ────────────────────

func computeScore(r *SlideReport) float64 {
	score := 100.0
	for _, iss := range r.Issues {
		switch iss.Severity {
		case SeverityError:
			score -= 15
		case SeverityWarning:
			score -= 7
		case SeverityInfo:
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// ──────────────────── Helpers ────────────────────

func isTextBearing(elem *model.Element) bool {
	switch elem.Type {
	case model.ElementTitle, model.ElementSubtitle, model.ElementBody, model.ElementBullet:
		return true
	default:
		return false
	}
}

func rectsOverlap(a, b model.Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func overlapRect(a, b model.Rect) float64 {
	dx := min(a.X+a.W, b.X+b.W) - max(a.X, b.X)
	dy := min(a.Y+a.H, b.Y+b.H) - max(a.Y, b.Y)
	if dx <= 0 || dy <= 0 {
		return 0
	}
	return dx * dy
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// FormatReport turns a PresentationReport into a concise text summary for the AI agent.
func FormatReport(report *PresentationReport) string {
	var sb stringBuilder
	sb.Printf("Layout Quality Score: %.0f/100\n\n", report.Score)
	for _, sr := range report.Slides {
		if len(sr.Issues) == 0 {
			sb.Printf("Slide %d (%s): OK ✓ (score: %.0f)\n", sr.SlideNum, sr.SlideID, sr.Score)
			continue
		}
		sb.Printf("Slide %d (%s) [score: %.0f]:\n", sr.SlideNum, sr.SlideID, sr.Score)
		for _, iss := range sr.Issues {
			icon := "⚠"
			switch iss.Severity {
			case SeverityError:
				icon = "✗"
			case SeverityWarning:
				icon = "⚠"
			case SeverityInfo:
				icon = "ℹ"
			}
			sb.Printf("  %s [%s] %s\n", icon, iss.Category, iss.Message)
		}
	}
	return sb.String()
}

// stringBuilder is a minimal strings.Builder wrapper with Printf.
type stringBuilder struct {
	data []byte
}

func (s *stringBuilder) Printf(format string, args ...any) {
	s.data = append(s.data, []byte(fmt.Sprintf(format, args...))...)
}

func (s *stringBuilder) String() string {
	return string(s.data)
}
