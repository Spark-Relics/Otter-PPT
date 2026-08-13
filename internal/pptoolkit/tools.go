package pptoolkit

import (
	"github.com/sashabaranov/go-openai"
)

// ToolDefinitions returns all available tools in OpenAI function-calling format.
// These are the "design operations" the AI agent can use.
func ToolDefinitions() []openai.Tool {
	return []openai.Tool{
		// ────────── Presentation ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_title",
			Description: "Set the presentation title (shown on cover page).",
			Parameters: params(map[string]prop{
				"title": {typ: "string", desc: "Presentation title", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_theme",
			Description: "Set the global color scheme and fonts for the entire presentation. Call this first before adding slides.",
			Parameters: params(map[string]prop{
				"name":             {typ: "string", desc: "Theme name"},
				"primary_color":    {typ: "string", desc: "Primary color hex, e.g. #1A73E8", req: true},
				"secondary_color":  {typ: "string", desc: "Secondary color hex"},
				"accent_color":     {typ: "string", desc: "Accent/highlight color hex"},
				"background_color": {typ: "string", desc: "Default background color hex"},
				"text_color":       {typ: "string", desc: "Default body text color hex"},
				"title_font":       {typ: "string", desc: "Title font name"},
				"body_font":        {typ: "string", desc: "Body font name"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_slide_size",
			Description: "Set slide dimensions in inches. Use 13.333x7.5 for 16:9 or 10x7.5 for 4:3.",
			Parameters: params(map[string]prop{
				"width":  {typ: "number", desc: "Width in inches", req: true},
				"height": {typ: "number", desc: "Height in inches", req: true},
			}),
		}},

		// ────────── Slides ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_slide",
			Description: "Add a new slide. Returns the slide ID. Layout options: title, title_content, two_column, image_left, image_right, image_full, section.",
			Parameters: params(map[string]prop{
				"layout": {typ: "string", desc: "Slide layout type", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "delete_slide",
			Description: "Delete a slide by ID.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "duplicate_slide",
			Description: "Duplicate an existing slide. Returns the new slide ID.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID to duplicate", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "move_slide",
			Description: "Reorder a slide to a new position.",
			Parameters: params(map[string]prop{
				"slide_id":  {typ: "string", desc: "Slide ID", req: true},
				"new_index": {typ: "integer", desc: "New 0-based position", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_notes",
			Description: "Set speaker notes for a slide.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
				"notes":    {typ: "string", desc: "Speaker notes text", req: true},
			}),
		}},

		// ────────── Background ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_bg_color",
			Description: "Set a solid background color for a slide.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
				"color":    {typ: "string", desc: "Hex color e.g. #1A237E", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_bg_gradient",
			Description: "Set a gradient background. Provide 2+ color stops.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
				"type":     {typ: "string", desc: "linear or radial", req: true},
				"angle":    {typ: "number", desc: "Angle in degrees (for linear)"},
				"stops":    {typ: "array", desc: "Color stops [{color, position}]", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_bg_image",
			Description: "Set an image as the slide background.",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"image_path": {typ: "string", desc: "Local path or URL to image", req: true},
			}),
		}},

		// ────────── Transition ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_transition",
			Description: "Set slide transition effect.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
				"type":     {typ: "string", desc: "fade, push, wipe, split, cover, zoom, morph", req: true},
				"duration": {typ: "number", desc: "Duration in seconds"},
			}),
		}},

		// ────────── Text ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_title",
			Description: "Add a title text element to a slide.",
			Parameters: rectParams(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
				"text":     {typ: "string", desc: "Title text", req: true},
				"font_size":  {typ: "integer", desc: "Font size in pt (default 36)"},
				"color":      {typ: "string", desc: "Text color hex"},
				"font_name":  {typ: "string", desc: "Font name"},
				"align":      {typ: "string", desc: "left, center, right"},
				"bold":       {typ: "boolean", desc: "Bold (default true for titles)"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_text",
			Description: "Add a body text box to a slide.",
			Parameters: rectParams(map[string]prop{
				"slide_id":  {typ: "string", desc: "Slide ID", req: true},
				"text":      {typ: "string", desc: "Text content", req: true},
				"font_size": {typ: "integer", desc: "Font size in pt"},
				"color":     {typ: "string", desc: "Text color hex"},
				"align":     {typ: "string", desc: "left, center, right, justify"},
				"bold":      {typ: "boolean", desc: "Bold"},
				"italic":    {typ: "boolean", desc: "Italic"},
				"line_spacing": {typ: "number", desc: "Line spacing multiplier (1.0 = single)"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_bullet_list",
			Description: "Add a bullet list to a slide.",
			Parameters: rectParams(map[string]prop{
				"slide_id":  {typ: "string", desc: "Slide ID", req: true},
				"items":     {typ: "array", desc: "Array of bullet point strings", req: true},
				"font_size": {typ: "integer", desc: "Font size in pt"},
				"color":     {typ: "string", desc: "Text color hex"},
				"bullet_char": {typ: "string", desc: "Bullet character (default •)"},
			}),
		}},

		// ────────── Visual Elements ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_image",
			Description: "Add an image to a slide. Provide image_path for an existing asset, or image_prompt when an image model is configured.",
			Parameters: rectParams(map[string]prop{
				"slide_id":    {typ: "string", desc: "Slide ID", req: true},
				"image_path":  {typ: "string", desc: "Existing local path or URL"},
				"image_prompt": {typ: "string", desc: "Prompt for the configured image model"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_shape",
			Description: "Add a shape. Types: rectangle, rounded_rectangle, ellipse, triangle, diamond, arrow, pentagon, hexagon, star, callout, line.",
			Parameters: rectParams(map[string]prop{
				"slide_id":     {typ: "string", desc: "Slide ID", req: true},
				"shape_type":   {typ: "string", desc: "Shape type", req: true},
				"fill_color":   {typ: "string", desc: "Fill color hex (empty = no fill)"},
				"border_color": {typ: "string", desc: "Border color hex"},
				"border_width": {typ: "number", desc: "Border width in pt"},
				"corner_radius": {typ: "number", desc: "Corner radius 0-1 (rounded_rectangle)"},
				"text":          {typ: "string", desc: "Optional text inside shape"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_table",
			Description: "Add a data table.",
			Parameters: rectParams(map[string]prop{
				"slide_id":      {typ: "string", desc: "Slide ID", req: true},
				"headers":       {typ: "array", desc: "Header cell texts", req: true},
				"rows":          {typ: "array", desc: "2D array of row cells", req: true},
				"header_color":  {typ: "string", desc: "Header row background color"},
				"alt_row_color": {typ: "string", desc: "Alternating row color"},
				"font_size":     {typ: "integer", desc: "Font size in pt"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_chart",
			Description: "Add a chart. Types: bar, column, line, pie, area, doughnut.",
			Parameters: rectParams(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"chart_type": {typ: "string", desc: "Chart type", req: true},
				"categories": {typ: "array", desc: "Category labels (x-axis)", req: true},
				"series":     {typ: "array", desc: "Series [{name, values:[num], color}]", req: true},
				"title":      {typ: "string", desc: "Chart title"},
				"show_legend": {typ: "boolean", desc: "Show legend (default true)"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_connector",
			Description: "Add a connector line or arrow between two points.",
			Parameters: params(map[string]prop{
				"slide_id":       {typ: "string", desc: "Slide ID", req: true},
				"connector_type": {typ: "string", desc: "line, arrow, double_arrow", req: true},
				"color":          {typ: "string", desc: "Line color hex", req: true},
				"width":          {typ: "number", desc: "Line width in pt", req: true},
				"start_x":        {typ: "number", desc: "Start X percentage 0-100", req: true},
				"start_y":        {typ: "number", desc: "Start Y percentage 0-100", req: true},
				"end_x":          {typ: "number", desc: "End X percentage 0-100", req: true},
				"end_y":          {typ: "number", desc: "End Y percentage 0-100", req: true},
			}),
		}},

		// ────────── Element Manipulation ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "update_text",
			Description: "Update text content of an existing element.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"text":     {typ: "string", desc: "New text", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "update_style",
			Description: "Update font/style of an existing element.",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"font_size":  {typ: "integer", desc: "Font size in pt"},
				"color":      {typ: "string", desc: "Text color hex"},
				"bold":       {typ: "boolean", desc: "Bold"},
				"italic":     {typ: "boolean", desc: "Italic"},
				"align":      {typ: "string", desc: "left, center, right"},
				"line_spacing": {typ: "number", desc: "Line spacing multiplier"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "update_position",
			Description: "Move or resize an element. Coordinates are percentages 0-100.",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"x":          {typ: "number", desc: "Left position %", req: true},
				"y":          {typ: "number", desc: "Top position %", req: true},
				"w":          {typ: "number", desc: "Width %", req: true},
				"h":          {typ: "number", desc: "Height %", req: true},
				"rotation":   {typ: "number", desc: "Rotation in degrees"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "delete_element",
			Description: "Delete an element from a slide.",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "bring_to_front",
			Description: "Bring an element to the front (highest z-order).",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "send_to_back",
			Description: "Send an element to the back (lowest z-order).",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_rotation",
			Description: "Set rotation angle of an element.",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"degrees":    {typ: "number", desc: "Rotation in degrees", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_opacity",
			Description: "Set element opacity/transparency (0 = invisible, 1 = opaque).",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"opacity":    {typ: "number", desc: "Opacity 0-1", req: true},
			}),
		}},

		// ────────── Animation ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "set_animation",
			Description: "Add an animation effect to an element. Types: fade, fly_in, zoom_in, bounce, rotate, wipe, appear.",
			Parameters: params(map[string]prop{
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"type":       {typ: "string", desc: "Animation type", req: true},
				"trigger":    {typ: "string", desc: "on_click, after_previous, with_previous"},
				"direction":  {typ: "string", desc: "left, right, top, bottom, center"},
				"duration":   {typ: "number", desc: "Duration in seconds"},
				"delay":      {typ: "number", desc: "Delay in seconds"},
			}),
		}},

		// ────────── State / Export ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "get_state",
			Description: "Get the current presentation state as JSON (for review).",
			Parameters: params(map[string]prop{}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "done",
			Description: "Signal that the presentation is complete and ready for export.",
			Parameters: params(map[string]prop{}),
		}},
	}
}
