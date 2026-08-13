package pptoolkit

import (
	"encoding/json"
	"fmt"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// ToolResult is the return value from executing a tool.
type ToolResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (r ToolResult) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

func ok(msg string, data ...any) ToolResult {
	r := ToolResult{Success: true, Message: msg}
	if len(data) > 0 {
		r.Data = data[0]
	}
	return r
}

func fail(msg string) ToolResult {
	return ToolResult{Success: false, Message: msg}
}

// ExecuteTool dispatches a tool call to the appropriate handler.
// name is the tool/function name, args is the parsed JSON arguments.
func (s *Session) ExecuteTool(name string, args map[string]any) ToolResult {
	switch name {

	// ──────── Presentation ────────
	case "set_title":
		title, _ := args["title"].(string)
		return ok(s.SetTitle(title))

	case "set_theme":
		theme := mapToTheme(args)
		return ok(s.SetTheme(theme))

	case "set_slide_size":
		w := toFloat(args["width"])
		h := toFloat(args["height"])
		return ok(s.SetSlideSize(w, h))

	// ──────── Slides ────────
	case "add_slide":
		layout, _ := args["layout"].(string)
		id := s.AddSlide(layout)
		return ok(fmt.Sprintf("Slide added (id=%s, layout=%s)", id, layout), map[string]string{"slide_id": id})

	case "delete_slide":
		id, _ := args["slide_id"].(string)
		if err := s.DeleteSlide(id); err != nil {
			return fail(err.Error())
		}
		return ok("Slide deleted")

	case "duplicate_slide":
		id, _ := args["slide_id"].(string)
		newID, err := s.DuplicateSlide(id)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Slide duplicated (new id=%s)", newID), map[string]string{"slide_id": newID})

	case "move_slide":
		id, _ := args["slide_id"].(string)
		idx := int(toFloat(args["new_index"]))
		if err := s.MoveSlide(id, idx); err != nil {
			return fail(err.Error())
		}
		return ok("Slide moved")

	case "set_notes":
		id, _ := args["slide_id"].(string)
		notes, _ := args["notes"].(string)
		if err := s.SetSlideNotes(id, notes); err != nil {
			return fail(err.Error())
		}
		return ok("Notes set")

	// ──────── Background ────────
	case "set_bg_color":
		id, _ := args["slide_id"].(string)
		color, _ := args["color"].(string)
		bg := &model.Background{Type: model.BgSolid, Color: color}
		if err := s.SetSlideBackground(id, bg); err != nil {
			return fail(err.Error())
		}
		return ok("Background color set")

	case "set_bg_gradient":
		id, _ := args["slide_id"].(string)
		gType, _ := args["type"].(string)
		angle := toFloat(args["angle"])
		stops := mapToGradientStops(args["stops"])
		bg := &model.Background{
			Type: model.BgGradient,
			Gradient: &model.Gradient{
				Type:  model.GradientType(gType),
				Angle: angle,
				Stops: stops,
			},
		}
		if err := s.SetSlideBackground(id, bg); err != nil {
			return fail(err.Error())
		}
		return ok("Gradient background set")

	case "set_bg_image":
		id, _ := args["slide_id"].(string)
		imgPath, _ := args["image_path"].(string)
		bg := &model.Background{Type: model.BgImage, ImagePath: imgPath}
		if err := s.SetSlideBackground(id, bg); err != nil {
			return fail(err.Error())
		}
		return ok("Background image set")

	// ──────── Transition ────────
	case "set_transition":
		id, _ := args["slide_id"].(string)
		tType, _ := args["type"].(string)
		dur := toFloat(args["duration"])
		t := &model.Transition{Type: model.TransitionType(tType), Duration: dur}
		if err := s.SetSlideTransition(id, t); err != nil {
			return fail(err.Error())
		}
		return ok("Transition set")

	// ──────── Text ────────
	case "add_title":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		text, _ := args["text"].(string)
		style := mapToStyle(args)
		elemID, err := s.AddTitle(id, rect, text, style)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Title added (id=%s)", elemID), map[string]string{"element_id": elemID})

	case "add_text":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		text, _ := args["text"].(string)
		style := mapToStyle(args)
		elemID, err := s.AddText(id, rect, text, style)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Text added (id=%s)", elemID), map[string]string{"element_id": elemID})

	case "add_bullet_list":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		items := toStrSlice(args["items"])
		style := mapToStyle(args)
		elemID, err := s.AddBulletList(id, rect, items, style)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Bullet list added (id=%s)", elemID), map[string]string{"element_id": elemID})

	// ──────── Visual ────────
	case "add_image":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		imgPath, _ := args["image_path"].(string)
		elemID, err := s.AddImage(id, rect, imgPath)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Image added (id=%s)", elemID), map[string]string{"element_id": elemID})

	case "add_shape":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		shape := mapToShape(args)
		elemID, err := s.AddShape(id, rect, shape)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Shape added (id=%s)", elemID), map[string]string{"element_id": elemID})

	case "add_table":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		table := mapToTable(args)
		elemID, err := s.AddTable(id, rect, table)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Table added (id=%s)", elemID), map[string]string{"element_id": elemID})

	case "add_chart":
		id, _ := args["slide_id"].(string)
		rect := mapToRect(args)
		chart := mapToChart(args)
		elemID, err := s.AddChart(id, rect, chart)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Chart added (id=%s)", elemID), map[string]string{"element_id": elemID})

	case "add_connector":
		id, _ := args["slide_id"].(string)
		conn := mapToConnector(args)
		elemID, err := s.AddConnector(id, conn)
		if err != nil {
			return fail(err.Error())
		}
		return ok(fmt.Sprintf("Connector added (id=%s)", elemID), map[string]string{"element_id": elemID})

	// ──────── Element Manipulation ────────
	case "update_text":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		text, _ := args["text"].(string)
		if err := s.UpdateText(id, elemID, text); err != nil {
			return fail(err.Error())
		}
		return ok("Text updated")

	case "update_style":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		style := mapToStyle(args)
		if err := s.UpdateStyle(id, elemID, style); err != nil {
			return fail(err.Error())
		}
		return ok("Style updated")

	case "update_position":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		rect := mapToRect(args)
		rot := toFloat(args["rotation"])
		if err := s.UpdatePosition(id, elemID, rect, rot); err != nil {
			return fail(err.Error())
		}
		return ok("Position updated")

	case "delete_element":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		if err := s.DeleteElement(id, elemID); err != nil {
			return fail(err.Error())
		}
		return ok("Element deleted")

	case "bring_to_front":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		if err := s.BringToFront(id, elemID); err != nil {
			return fail(err.Error())
		}
		return ok("Element brought to front")

	case "send_to_back":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		if err := s.SendToBack(id, elemID); err != nil {
			return fail(err.Error())
		}
		return ok("Element sent to back")

	case "set_rotation":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		deg := toFloat(args["degrees"])
		if err := s.SetRotation(id, elemID, deg); err != nil {
			return fail(err.Error())
		}
		return ok("Rotation set")

	case "set_opacity":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		op := toFloat(args["opacity"])
		if err := s.SetOpacity(id, elemID, op); err != nil {
			return fail(err.Error())
		}
		return ok("Opacity set")

	case "set_animation":
		id, _ := args["slide_id"].(string)
		elemID, _ := args["element_id"].(string)
		anim := mapToAnimation(args)
		if err := s.SetElementAnimation(id, elemID, anim); err != nil {
			return fail(err.Error())
		}
		return ok("Animation set")

	// ──────── State / Export ────────
	case "get_state":
		return ok("Current state", s.Presentation())

	case "done":
		return ok("Presentation complete")

	default:
		return fail(fmt.Sprintf("Unknown tool: %s", name))
	}
}

// ============================================================
// Helpers for map → struct conversion
// ============================================================

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func toStrSlice(v any) []string {
	if arr, ok := v.([]any); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

func mapToRect(args map[string]any) model.Rect {
	return model.Rect{
		X: toFloat(args["x"]),
		Y: toFloat(args["y"]),
		W: toFloat(args["w"]),
		H: toFloat(args["h"]),
	}
}

func mapToStyle(args map[string]any) model.TextStyle {
	fs := int(toFloat(args["font_size"]))
	return model.TextStyle{
		FontSize:      fs,
		FontName:      strOr(args, "font_name", ""),
		Bold:          boolOr(args, "bold"),
		Italic:        boolOr(args, "italic"),
		Underline:     boolOr(args, "underline"),
		Color:         strOr(args, "color", ""),
		Align:         strOr(args, "align", ""),
		LineSpacing:   toFloat(args["line_spacing"]),
		LetterSpacing: toFloat(args["letter_spacing"]),
		BulletChar:    strOr(args, "bullet_char", ""),
		VAlign:        strOr(args, "valign", ""),
		Shadow:        boolOr(args, "shadow"),
	}
}

func mapToTheme(args map[string]any) model.Theme {
	return model.Theme{
		Name:            strOr(args, "name", ""),
		PrimaryColor:    strOr(args, "primary_color", "#1A73E8"),
		SecondaryColor:  strOr(args, "secondary_color", "#424242"),
		AccentColor:     strOr(args, "accent_color", "#FF6D00"),
		BackgroundColor: strOr(args, "background_color", "#FFFFFF"),
		TextColor:       strOr(args, "text_color", "#212121"),
		TitleFont:       strOr(args, "title_font", "Microsoft YaHei"),
		BodyFont:        strOr(args, "body_font", "Microsoft YaHei"),
	}
}

func mapToGradientStops(v any) []model.GradientStop {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	stops := make([]model.GradientStop, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			stops = append(stops, model.GradientStop{
				Color:    strOr(m, "color", "#FFFFFF"),
				Position: toFloat(m["position"]),
				Opacity:  toFloat(m["opacity"]),
			})
		}
	}
	return stops
}

func mapToGradient(v any) *model.Gradient {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	stops := mapToGradientStops(m["stops"])
	if len(stops) == 0 {
		return nil
	}
	return &model.Gradient{Type: model.GradientType(strOr(m, "type", "linear")), Angle: toFloat(m["angle"]), Stops: stops}
}

func mapToShape(args map[string]any) *model.ShapeData {
	shape := &model.ShapeData{
		ShapeType: model.ShapeType(strOr(args, "shape_type", "rectangle")), FillColor: strOr(args, "fill_color", ""),
		BorderColor: strOr(args, "border_color", ""), BorderWidth: toFloat(args["border_width"]),
		Text: strOr(args, "text", ""), CornerRadius: toFloat(args["corner_radius"]),
	}
	if gradient := mapToGradient(args["gradient"]); gradient != nil {
		shape.Fill = &model.FillStyle{Gradient: gradient, Opacity: toFloat(args["fill_opacity"])}
	} else if opacity := toFloat(args["fill_opacity"]); opacity > 0 {
		shape.Fill = &model.FillStyle{Color: shape.FillColor, Opacity: opacity}
	}
	if boolOr(args, "shadow") {
		shape.Shadow = &model.ShadowStyle{Color: "#000000", Opacity: 0.25, Blur: 6, Distance: 2, Angle: 45}
	}
	return shape
}

func mapToTable(args map[string]any) *model.TableData {
	td := &model.TableData{
		HeaderColor: strOr(args, "header_color", ""),
		AltRowColor: strOr(args, "alt_row_color", ""),
		FontSize:    int(toFloat(args["font_size"])),
	}
	// Headers
	for _, h := range toStrSlice(args["headers"]) {
		td.Headers = append(td.Headers, model.TableCell{Text: h})
	}
	// Rows
	if rows, ok := args["rows"].([]any); ok {
		for _, row := range rows {
			cells := toStrSlice(row)
			var tableRow []model.TableCell
			for _, c := range cells {
				tableRow = append(tableRow, model.TableCell{Text: c})
			}
			td.Rows = append(td.Rows, tableRow)
		}
	}
	return td
}

func mapToChart(args map[string]any) *model.ChartData {
	cd := &model.ChartData{
		ChartType:  model.ChartType(strOr(args, "chart_type", "column")),
		Categories: toStrSlice(args["categories"]),
		Title:      strOr(args, "title", ""),
		ShowLegend: true,
	}
	if v, ok := args["show_legend"]; ok {
		cd.ShowLegend = v.(bool)
	}
	if series, ok := args["series"].([]any); ok {
		for _, s := range series {
			if m, ok := s.(map[string]any); ok {
				cs := model.ChartSeries{
					Name:  strOr(m, "name", ""),
					Color: strOr(m, "color", ""),
				}
				if vals, ok := m["values"].([]any); ok {
					for _, v := range vals {
						cs.Values = append(cs.Values, toFloat(v))
					}
				}
				cd.Series = append(cd.Series, cs)
			}
		}
	}
	return cd
}

func mapToConnector(args map[string]any) *model.ConnectorData {
	return &model.ConnectorData{
		ConnectorType: model.ShapeType(strOr(args, "connector_type", "line")),
		Color:         strOr(args, "color", "#333333"),
		Width:         toFloat(args["width"]),
		StartX:        toFloat(args["start_x"]),
		StartY:        toFloat(args["start_y"]),
		EndX:          toFloat(args["end_x"]),
		EndY:          toFloat(args["end_y"]),
	}
}

func mapToAnimation(args map[string]any) *model.Animation {
	anim := &model.Animation{
		Type:      model.AnimationType(strOr(args, "type", "fade")),
		Direction: model.AnimationDirection(strOr(args, "direction", "")),
		Duration:  toFloat(args["duration"]),
		Delay:     toFloat(args["delay"]),
	}
	trigger := strOr(args, "trigger", "on_click")
	anim.Trigger = model.AnimationTrigger(trigger)
	return anim
}

func strOr(m map[string]any, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func boolOr(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
