package layout

import (
	"fmt"
	"math"
	"sort"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// AutoFixSlide automatically corrects common layout problems on a slide.
// It returns the number of fixes applied.
func AutoFixSlide(slide *model.Slide) int {
	fixes := 0
	fixes += fixBounds(slide)
	fixes += fixOverlaps(slide)
	fixes += fixTitlePosition(slide)
	return fixes
}

// AutoFixPresentation runs AutoFixSlide on every slide.
// Returns total fixes applied.
func AutoFixPresentation(pres *model.Presentation) int {
	total := 0
	for _, slide := range pres.Slides {
		total += AutoFixSlide(slide)
	}
	return total
}

// fixBounds clamps any element that extends beyond safe slide boundaries (0-100%).
func fixBounds(slide *model.Slide) int {
	fixes := 0
	margin := 3.0 // safe margin
	maxRight := 97.0
	maxBottom := 95.0

	for _, elem := range slide.Elements {
		if elem.Type == model.ElementConnector {
			continue
		}
		rect := &elem.Rect

		// X: negative → push to margin; too far right → shift left
		if rect.X < margin {
			rect.X = margin
			fixes++
		}
		if rect.X+rect.W > maxRight {
			if rect.W < maxRight-margin {
				rect.X = maxRight - rect.W
			} else {
				rect.W = maxRight - margin
				rect.X = margin
			}
			fixes++
		}

		// Y: negative → push to margin; too far down → shift up
		if rect.Y < margin {
			rect.Y = margin
			fixes++
		}
		if rect.Y+rect.H > maxBottom {
			if rect.H < maxBottom-margin {
				rect.Y = maxBottom - rect.H
			} else {
				rect.H = maxBottom - margin
				rect.Y = margin
			}
			fixes++
		}
	}
	return fixes
}

// fixOverlaps resolves text element overlaps by shifting later elements down.
func fixOverlaps(slide *model.Slide) int {
	fixes := 0

	// Collect text-bearing elements sorted by Y position
	type item struct {
		elem *model.Element
	}
	var items []item
	for _, elem := range slide.Elements {
		if isTextBearing(elem) {
			items = append(items, item{elem: elem})
		}
	}
	if len(items) < 2 {
		return 0
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].elem.Rect.Y < items[j].elem.Rect.Y
	})

	// Try to resolve overlaps by pushing elements down
	gap := 2.0 // minimum gap between text elements
	for i := 1; i < len(items); i++ {
		prev := items[i-1].elem
		cur := items[i].elem

		// Check horizontal overlap (do their X ranges intersect?)
		if cur.Rect.X >= prev.Rect.X+prev.Rect.W || cur.Rect.X+cur.Rect.W <= prev.Rect.X {
			continue // No horizontal overlap, fine
		}

		// Check vertical overlap
		prevBottom := prev.Rect.Y + prev.Rect.H
		if cur.Rect.Y < prevBottom+gap {
			shift := prevBottom + gap - cur.Rect.Y
			cur.Rect.Y = prevBottom + gap
			fixes++

			// Clamp to bounds
			if cur.Rect.Y+cur.Rect.H > 95 {
				cur.Rect.Y = 95 - cur.Rect.H
			}
			_ = shift
		}
	}
	return fixes
}

// fixTitlePosition ensures title elements are in the top portion of the slide.
func fixTitlePosition(slide *model.Slide) int {
	fixes := 0
	for _, elem := range slide.Elements {
		if elem.Type != model.ElementTitle {
			continue
		}
		if elem.Rect.Y > 40 {
			elem.Rect.Y = 8 // Standard title position
			fixes++
		}
		// Title should span reasonable width
		if elem.Rect.W < 40 {
			elem.Rect.W = 70
			elem.Rect.X = 6
			fixes++
		}
	}
	return fixes
}

// SmartDistribute evenly distributes N elements across a horizontal row.
// Elements are placed with equal spacing starting from startX.
func SmartDistribute(elements []*model.Element, startX, startY, totalWidth, height, gap float64) {
	n := len(elements)
	if n == 0 {
		return
	}
	itemWidth := (totalWidth - gap*float64(n-1)) / float64(n)
	for i, elem := range elements {
		elem.Rect.X = startX + float64(i)*(itemWidth+gap)
		elem.Rect.Y = startY
		elem.Rect.W = itemWidth
		elem.Rect.H = height
	}
}

// SmartGrid arranges elements in a rows×cols grid within the given bounds.
func SmartGrid(elements []*model.Element, startX, startY, totalWidth, totalHeight float64, rows, cols int) {
	n := len(elements)
	if n == 0 || rows == 0 || cols == 0 {
		return
	}
	gapX := 2.0
	gapY := 2.0
	itemW := (totalWidth - gapX*float64(cols-1)) / float64(cols)
	itemH := (totalHeight - gapY*float64(rows-1)) / float64(rows)
	for i, elem := range elements {
		if i >= rows*cols {
			break
		}
		row := i / cols
		col := i % cols
		elem.Rect.X = startX + float64(col)*(itemW+gapX)
		elem.Rect.Y = startY + float64(row)*(itemH+gapY)
		elem.Rect.W = itemW
		elem.Rect.H = itemH
	}
}

// SuggestFixes returns a summary string of what AutoFix would do.
func SuggestFixes(report *PresentationReport) string {
	var sb stringBuilder
	fixCount := 0
	sb.Printf("Auto-fix recommendations:\n")
	for _, sr := range report.Slides {
		for _, iss := range sr.Issues {
			if iss.Severity == SeverityError || iss.Severity == SeverityWarning {
				sb.Printf("  Slide %d [%s]: %s\n", sr.SlideNum, iss.Severity, iss.Message)
				fixCount++
			}
		}
	}
	if fixCount == 0 {
		sb.Printf("  No automatic fixes needed.\n")
	} else {
		sb.Printf("\nCall auto_fix_layout to resolve %d issues automatically.\n", fixCount)
	}
	return sb.String()
}

// unused but kept for potential future use
var _ = math.Round
var _ = fmt.Sprintf
