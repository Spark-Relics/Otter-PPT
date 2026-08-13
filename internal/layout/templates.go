package layout

import (
	"fmt"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// TemplateID identifies a predefined smart layout.
type TemplateID string

const (
	TemplateTitle         TemplateID = "title"          // bold title + subtitle on full bg
	TemplateTitleContent  TemplateID = "title_content"  // title on top, content below
	TemplateTwoColumn     TemplateID = "two_column"     // title + 2 content columns
	TemplateImageLeft     TemplateID = "image_left"     // image on left, text on right
	TemplateImageRight    TemplateID = "image_right"    // text on left, image on right
	TemplateImageFull     TemplateID = "image_full"     // full-bleed image with title overlay
	TemplateSection       TemplateID = "section"        // section divider
	TemplateBullets       TemplateID = "bullets"        // title + bullet list
	TemplateQuote         TemplateID = "quote"          // large quote center
	TemplateThreeCards    TemplateID = "three_cards"    // title + 3 card shapes
	TemplateFourCards     TemplateID = "four_cards"     // title + 4 card shapes in 2x2
	TemplateTimeline       TemplateID = "timeline"       // horizontal timeline with milestones
	TemplateComparison    TemplateID = "comparison"     // two-column comparison
	TemplateStats         TemplateID = "stats"          // title + 3 big number stats
	TemplateChart         TemplateID = "chart"          // title + chart placeholder
	TemplateAgenda        TemplateID = "agenda"         // title + numbered list
	TemplateThankYou      TemplateID = "thank_you"      // "Thank You" centered
	TemplateContact       TemplateID = "contact"        // contact info centered
)

// TemplateSlot describes one content placeholder within a template.
type TemplateSlot struct {
	Role  string     // "title", "content", "image", "stat", "card", "quote", etc.
	Rect  model.Rect // position in percentages
	Style string     // hint for font sizing/style
}

// TemplateDef is a predefined layout with named slots.
type TemplateDef struct {
	ID     TemplateID
	Name   string
	Slots  []TemplateSlot
}

// allTemplates maps template IDs to their definitions.
var allTemplates = map[TemplateID]TemplateDef{
	TemplateTitle: {
		ID:   TemplateTitle,
		Name: "Title Slide",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 10, Y: 35, W: 80, H: 20}, Style: "large"},
			{Role: "subtitle", Rect: model.Rect{X: 15, Y: 56, W: 70, H: 10}, Style: "medium"},
		},
	},
	TemplateTitleContent: {
		ID:   TemplateTitleContent,
		Name: "Title + Content",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 8, W: 88, H: 12}, Style: "large"},
			{Role: "content", Rect: model.Rect{X: 8, Y: 25, W: 84, H: 60}, Style: "normal"},
		},
	},
	TemplateTwoColumn: {
		ID:   TemplateTwoColumn,
		Name: "Two Column",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 8, W: 88, H: 10}, Style: "large"},
			{Role: "column_left", Rect: model.Rect{X: 6, Y: 25, W: 41, H: 60}, Style: "normal"},
			{Role: "column_right", Rect: model.Rect{X: 53, Y: 25, W: 41, H: 60}, Style: "normal"},
		},
	},
	TemplateImageLeft: {
		ID:   TemplateImageLeft,
		Name: "Image Left",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 52, Y: 10, W: 42, H: 10}},
			{Role: "content", Rect: model.Rect{X: 52, Y: 25, W: 42, H: 55}},
			{Role: "image", Rect: model.Rect{X: 5, Y: 10, W: 42, H: 80}},
		},
	},
	TemplateImageRight: {
		ID:   TemplateImageRight,
		Name: "Image Right",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 10, W: 42, H: 10}},
			{Role: "content", Rect: model.Rect{X: 6, Y: 25, W: 42, H: 55}},
			{Role: "image", Rect: model.Rect{X: 53, Y: 10, W: 42, H: 80}},
		},
	},
	TemplateImageFull: {
		ID:   TemplateImageFull,
		Name: "Full Image with Overlay",
		Slots: []TemplateSlot{
			{Role: "image", Rect: model.Rect{X: 0, Y: 0, W: 100, H: 100}},
			{Role: "title", Rect: model.Rect{X: 8, Y: 38, W: 84, H: 18}, Style: "large"},
			{Role: "subtitle", Rect: model.Rect{X: 12, Y: 58, W: 76, H: 8}, Style: "medium"},
		},
	},
	TemplateSection: {
		ID:   TemplateSection,
		Name: "Section Divider",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 15, Y: 38, W: 70, H: 15}, Style: "large"},
			{Role: "subtitle", Rect: model.Rect{X: 20, Y: 55, W: 60, H: 8}, Style: "medium"},
		},
	},
	TemplateBullets: {
		ID:   TemplateBullets,
		Name: "Title + Bullets",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 8, W: 88, H: 12}},
			{Role: "bullets", Rect: model.Rect{X: 10, Y: 25, W: 80, H: 60}},
		},
	},
	TemplateQuote: {
		ID:   TemplateQuote,
		Name: "Quote",
		Slots: []TemplateSlot{
			{Role: "quote", Rect: model.Rect{X: 15, Y: 25, W: 70, H: 30}, Style: "large"},
			{Role: "author", Rect: model.Rect{X: 20, Y: 58, W: 60, H: 8}, Style: "medium"},
		},
	},
	TemplateThreeCards: {
		ID:   TemplateThreeCards,
		Name: "Three Cards",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 6, W: 88, H: 10}},
			{Role: "card1", Rect: model.Rect{X: 5, Y: 22, W: 28, H: 60}},
			{Role: "card2", Rect: model.Rect{X: 36, Y: 22, W: 28, H: 60}},
			{Role: "card3", Rect: model.Rect{X: 67, Y: 22, W: 28, H: 60}},
		},
	},
	TemplateFourCards: {
		ID:   TemplateFourCards,
		Name: "Four Cards (2x2)",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 5, W: 88, H: 8}},
			{Role: "card1", Rect: model.Rect{X: 5, Y: 18, W: 43, H: 34}},
			{Role: "card2", Rect: model.Rect{X: 52, Y: 18, W: 43, H: 34}},
			{Role: "card3", Rect: model.Rect{X: 5, Y: 56, W: 43, H: 34}},
			{Role: "card4", Rect: model.Rect{X: 52, Y: 56, W: 43, H: 34}},
		},
	},
	TemplateTimeline: {
		ID:   TemplateTimeline,
		Name: "Timeline",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 8, W: 88, H: 10}},
			{Role: "milestone1", Rect: model.Rect{X: 4, Y: 35, W: 20, H: 30}},
			{Role: "milestone2", Rect: model.Rect{X: 27, Y: 35, W: 20, H: 30}},
			{Role: "milestone3", Rect: model.Rect{X: 50, Y: 35, W: 20, H: 30}},
			{Role: "milestone4", Rect: model.Rect{X: 73, Y: 35, W: 20, H: 30}},
		},
	},
	TemplateComparison: {
		ID:   TemplateComparison,
		Name: "Comparison",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 6, W: 88, H: 10}},
			{Role: "left_header", Rect: model.Rect{X: 5, Y: 20, W: 41, H: 8}},
			{Role: "left_content", Rect: model.Rect{X: 5, Y: 30, W: 41, H: 45}},
			{Role: "right_header", Rect: model.Rect{X: 54, Y: 20, W: 41, H: 8}},
			{Role: "right_content", Rect: model.Rect{X: 54, Y: 30, W: 41, H: 45}},
		},
	},
	TemplateStats: {
		ID:   TemplateStats,
		Name: "Stats Dashboard",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 6, W: 88, H: 10}},
			{Role: "stat1", Rect: model.Rect{X: 5, Y: 30, W: 28, H: 30}},
			{Role: "stat2", Rect: model.Rect{X: 36, Y: 30, W: 28, H: 30}},
			{Role: "stat3", Rect: model.Rect{X: 67, Y: 30, W: 28, H: 30}},
			{Role: "footer", Rect: model.Rect{X: 10, Y: 72, W: 80, H: 8}},
		},
	},
	TemplateChart: {
		ID:   TemplateChart,
		Name: "Title + Chart",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 6, W: 88, H: 10}},
			{Role: "chart", Rect: model.Rect{X: 5, Y: 20, W: 62, H: 65}},
			{Role: "commentary", Rect: model.Rect{X: 70, Y: 25, W: 25, H: 50}},
		},
	},
	TemplateAgenda: {
		ID:   TemplateAgenda,
		Name: "Agenda",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 6, Y: 8, W: 88, H: 10}},
			{Role: "item1", Rect: model.Rect{X: 15, Y: 25, W: 70, H: 8}},
			{Role: "item2", Rect: model.Rect{X: 15, Y: 35, W: 70, H: 8}},
			{Role: "item3", Rect: model.Rect{X: 15, Y: 45, W: 70, H: 8}},
			{Role: "item4", Rect: model.Rect{X: 15, Y: 55, W: 70, H: 8}},
			{Role: "item5", Rect: model.Rect{X: 15, Y: 65, W: 70, H: 8}},
		},
	},
	TemplateThankYou: {
		ID:   TemplateThankYou,
		Name: "Thank You",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 20, Y: 35, W: 60, H: 18}, Style: "large"},
			{Role: "subtitle", Rect: model.Rect{X: 25, Y: 56, W: 50, H: 10}, Style: "medium"},
		},
	},
	TemplateContact: {
		ID:   TemplateContact,
		Name: "Contact",
		Slots: []TemplateSlot{
			{Role: "title", Rect: model.Rect{X: 20, Y: 25, W: 60, H: 12}, Style: "large"},
			{Role: "email", Rect: model.Rect{X: 25, Y: 42, W: 50, H: 6}},
			{Role: "phone", Rect: model.Rect{X: 25, Y: 52, W: 50, H: 6}},
			{Role: "web", Rect: model.Rect{X: 25, Y: 62, W: 50, H: 6}},
		},
	},
}

// GetTemplate returns a template definition by ID.
func GetTemplate(id TemplateID) (TemplateDef, bool) {
	t, ok := allTemplates[id]
	return t, ok
}

// ListTemplates returns all available template definitions.
func ListTemplates() []TemplateDef {
	result := make([]TemplateDef, 0, len(allTemplates))
	for _, t := range allTemplates {
		result = append(result, t)
	}
	return result
}

// TemplateNames returns a summary of available templates for the AI agent.
func TemplateNames() string {
	result := "Available layouts:\n"
	for _, t := range allTemplates {
		result += fmt.Sprintf("  - %s (%s): %d slots\n", t.ID, t.Name, len(t.Slots))
	}
	return result
}

// ApplyTemplate repositions existing elements on a slide to match a template.
// It maps elements to template slots by type/role matching:
//   - Title elements → title/subtitle slots
//   - Body/bullet elements → content/bullets/column slots
//   - Image elements → image slots
//   - Chart elements → chart slots
//   - Shapes → card slots
// Elements not matched by type are distributed to remaining slots in order.
func ApplyTemplate(slide *model.Slide, tplID TemplateID) (string, error) {
	tpl, ok := GetTemplate(tplID)
	if !ok {
		return "", fmt.Errorf("unknown template: %s", tplID)
	}

	slots := tpl.Slots
	elements := slide.Elements
	if len(elements) == 0 || len(slots) == 0 {
		return fmt.Sprintf("Applied template %s (no elements to position)", tpl.Name), nil
	}

	// Build priority queues by element type
	var titles, images, charts, shapes, others []*model.Element
	for _, elem := range elements {
		switch {
		case elem.Type == model.ElementTitle || elem.Type == model.ElementSubtitle:
			titles = append(titles, elem)
		case elem.Type == model.ElementImage:
			images = append(images, elem)
		case elem.Type == model.ElementChart:
			charts = append(charts, elem)
		case elem.Type == model.ElementShape:
			shapes = append(shapes, elem)
		default:
			others = append(others, elem)
		}
	}

	// Assign elements to slots
	assigned := make(map[int]bool)
	assignSlot := func(elem *model.Element, slotIdx int) {
		if slotIdx < len(slots) && !assigned[slotIdx] {
			elem.Rect = slots[slotIdx].Rect
			assigned[slotIdx] = true
		}
	}

	// First pass: match titles to title/subtitle slots
	titleIdx := 0
	subtitleIdx := 0
	for i, slot := range slots {
		if slot.Role == "title" && titleIdx < len(titles) {
			assignSlot(titles[titleIdx], i)
			titleIdx++
		}
		if slot.Role == "subtitle" && subtitleIdx < len(titles) {
			if titleIdx < len(titles) {
				assignSlot(titles[titleIdx], i)
				titleIdx++
			}
		}
	}

	// Images to image slots
	imgIdx := 0
	for i, slot := range slots {
		if slot.Role == "image" && imgIdx < len(images) {
			assignSlot(images[imgIdx], i)
			imgIdx++
		}
	}

	// Charts to chart slots
	chartIdx := 0
	for i, slot := range slots {
		if slot.Role == "chart" && chartIdx < len(charts) {
			assignSlot(charts[chartIdx], i)
			chartIdx++
		}
	}

	// Shapes to card slots
	shapeIdx := 0
	for i, slot := range slots {
		if contains([]string{"card1", "card2", "card3", "card4", "left_header", "right_header", "left_content", "right_content", "milestone1", "milestone2", "milestone3", "milestone4", "stat1", "stat2", "stat3", "footer"}, slot.Role) && shapeIdx < len(shapes) {
			assignSlot(shapes[shapeIdx], i)
			shapeIdx++
		}
	}

	// Remaining elements (others + unassigned) to unassigned slots
	remaining := append(others, unassignedElements(elements, assigned, slots)...)
	remIdx := 0
	for i, slot := range slots {
		if assigned[i] {
			continue
		}
		if remIdx < len(remaining) {
			remaining[remIdx].Rect = slot.Rect
			assigned[i] = true
			remIdx++
		}
	}

	// Update slide layout to match template
	slide.Layout = model.SlideLayout(tplID)

	return fmt.Sprintf("Applied template '%s' to slide %s: positioned %d elements into %d slots", tpl.Name, slide.ID, len(elements), len(slots)), nil
}

func unassignedElements(elements []*model.Element, assigned map[int]bool, slots []TemplateSlot) []*model.Element {
	// Elements whose rect hasn't been matched to a slot
	var result []*model.Element
	for _, elem := range elements {
		if !elemAssignedToSlot(elem, assigned, slots) {
			result = append(result, elem)
		}
	}
	return result
}

func elemAssignedToSlot(elem *model.Element, assigned map[int]bool, slots []TemplateSlot) bool {
	for i, slot := range slots {
		if assigned[i] && rectsEqual(elem.Rect, slot.Rect) {
			return true
		}
	}
	return false
}

func rectsEqual(a, b model.Rect) bool {
	return a.X == b.X && a.Y == b.Y && a.W == b.W && a.H == b.H
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
