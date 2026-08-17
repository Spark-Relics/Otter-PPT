package pptoolkit

import (
	"github.com/sashabaranov/go-openai"
	"github.com/otter-ppt/otter-ppt/internal/design"
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
		Description: "Set the global color scheme and fonts for the entire presentation. Call this first before adding slides. Two modes: (a) pass style + palette preset keys from the catalog below for a professionally locked design system, or (b) pass explicit hex colors. Preset mode is strongly preferred.\n\nSTYLE presets (shape language / composition discipline):\n" + design.StyleCatalog() + "\nPALETTE presets (six semantic color roles):\n" + design.PaletteCatalog() + "\nWhen style/palette keys are given, individual color args are optional overrides.",
		Parameters: params(map[string]prop{
			"name":             {typ: "string", desc: "Theme name"},
			"style":            {typ: "string", desc: "Style preset key (e.g. dark_tech, swiss_minimal, glassmorphism)"},
			"palette":          {typ: "string", desc: "Palette preset key (e.g. tech_neon, cool_corporate)"},
			"primary_color":    {typ: "string", desc: "Primary color hex, e.g. #1A73E8 (overrides palette)"},
			"secondary_color":  {typ: "string", desc: "Secondary color hex"},
			"accent_color":     {typ: "string", desc: "Accent/highlight color hex (overrides palette)"},
			"background_color": {typ: "string", desc: "Default background color hex (overrides palette)"},
			"text_color":       {typ: "string", desc: "Default body text color hex"},
			"title_font":       {typ: "string", desc: "Title font name (overrides style)"},
			"body_font":        {typ: "string", desc: "Body font name (overrides style)"},
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
				"slide_id":  {typ: "string", desc: "Slide ID", req: true},
				"text":      {typ: "string", desc: "Title text", req: true},
				"font_size": {typ: "integer", desc: "Font size in pt (default 36)"},
				"color":     {typ: "string", desc: "Text color hex"},
				"font_name": {typ: "string", desc: "Font name"},
				"align":     {typ: "string", desc: "left, center, right"},
				"bold":      {typ: "boolean", desc: "Bold (default true for titles)"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_text",
			Description: "Add a body text box to a slide.",
			Parameters: rectParams(map[string]prop{
				"slide_id":     {typ: "string", desc: "Slide ID", req: true},
				"text":         {typ: "string", desc: "Text content", req: true},
				"font_size":    {typ: "integer", desc: "Font size in pt"},
				"color":        {typ: "string", desc: "Text color hex"},
				"align":        {typ: "string", desc: "left, center, right, justify"},
				"bold":         {typ: "boolean", desc: "Bold"},
				"italic":       {typ: "boolean", desc: "Italic"},
				"line_spacing": {typ: "number", desc: "Line spacing multiplier (1.0 = single)"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_bullet_list",
			Description: "Add a bullet list to a slide.",
			Parameters: rectParams(map[string]prop{
				"slide_id":    {typ: "string", desc: "Slide ID", req: true},
				"items":       {typ: "array", desc: "Array of bullet point strings", req: true},
				"font_size":   {typ: "integer", desc: "Font size in pt"},
				"color":       {typ: "string", desc: "Text color hex"},
				"bullet_char": {typ: "string", desc: "Bullet character (default •)"},
			}),
		}},

		// ────────── Visual Elements ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_image",
			Description: "Add an image to a slide. Provide image_path for an existing asset, or image_prompt when an image model is configured.",
			Parameters: rectParams(map[string]prop{
				"slide_id":     {typ: "string", desc: "Slide ID", req: true},
				"image_path":   {typ: "string", desc: "Existing local path or URL"},
				"image_prompt": {typ: "string", desc: "Prompt for the configured image model"},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "add_shape",
			Description: "Add a shape. Types: rectangle, rounded_rectangle, ellipse, triangle, diamond, arrow, pentagon, hexagon, star, callout, line.",
			Parameters: rectParams(map[string]prop{
				"slide_id":      {typ: "string", desc: "Slide ID", req: true},
				"shape_type":    {typ: "string", desc: "Shape type", req: true},
				"fill_color":    {typ: "string", desc: "Fill color hex (empty = no fill)"},
				"border_color":  {typ: "string", desc: "Border color hex"},
				"border_width":  {typ: "number", desc: "Border width in pt"},
				"corner_radius": {typ: "number", desc: "Corner radius 0-1 (rounded_rectangle)"},
				"gradient":      {typ: "object", desc: "Gradient {type, angle, stops:[{color,position,opacity}]}"},
				"fill_opacity":  {typ: "number", desc: "Fill opacity 0-1"},
				"shadow":        {typ: "boolean", desc: "Add a subtle outer shadow"},
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
		Description: "Add a native chart. Selection rules (pick by data shape, not preference): " +
			"column: single-series category comparison, 3-8 categories; skip for >12 long labels (use bar) or multi-series (use grouped bar via multi-series column). " +
			"bar (horizontal): ranking 5-12 items, especially long labels; skip for <=8 short labels (use column). " +
			"line: 1-3 time-series showing direction; skip if cumulative volume matters (use area) or units differ (use combo with secondary axis). " +
			"area: 1-2 cumulative trend series emphasizing volume; skip for >=3 series. " +
			"pie: 3-6 simple proportions of one whole; skip for >=7 parts (use doughnut). " +
			"doughnut: 3-6 proportions where a center KPI deserves emphasis. " +
			"scatter: x-y correlation between two numeric variables; not for categories. " +
			"combo: bar+line mix, or two metrics with different units/scales (line series on secondary axis). " +
			"3D variants (bar_3d, column_3d, line_3d, pie_3d, area_3d): only when explicitly requested; 2D reads cleaner.",
		Parameters: rectParams(map[string]prop{
			"slide_id":    {typ: "string", desc: "Slide ID", req: true},
			"chart_type":  {typ: "string", desc: "Chart type: bar, column, line, pie, area, doughnut, scatter, combo, bar_3d, column_3d, line_3d, pie_3d, area_3d", req: true},
				"categories":  {typ: "array", desc: "Category labels (x-axis)", req: true},
				"series":      {typ: "array", desc: "Series [{name, values:[num], color}]", req: true},
				"title":       {typ: "string", desc: "Chart title"},
				"show_legend":       {typ: "boolean", desc: "Show legend (default true)"},
				"show_data_labels":  {typ: "boolean", desc: "Show data labels on chart"},
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
				"slide_id":   {typ: "string", desc: "Slide ID", req: true},
				"element_id": {typ: "string", desc: "Element ID", req: true},
				"text":       {typ: "string", desc: "New text", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "update_style",
			Description: "Update font/style of an existing element.",
			Parameters: params(map[string]prop{
				"slide_id":     {typ: "string", desc: "Slide ID", req: true},
				"element_id":   {typ: "string", desc: "Element ID", req: true},
				"font_size":    {typ: "integer", desc: "Font size in pt"},
				"color":        {typ: "string", desc: "Text color hex"},
				"bold":         {typ: "boolean", desc: "Bold"},
				"italic":       {typ: "boolean", desc: "Italic"},
				"align":        {typ: "string", desc: "left, center, right"},
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

		// ────────── Layout Intelligence ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "validate_layout",
			Description: "Validate the layout quality of a specific slide or all slides. Returns issues (overlaps, out-of-bounds, title placement) and a quality score 0-100. Call this after adding elements to check for problems.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID to validate. If empty, validates all slides."},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "auto_fix_layout",
			Description: "Automatically fix layout issues on a slide: clamps out-of-bounds elements, resolves overlaps by repositioning, and ensures titles are in the top portion. Returns the number of fixes applied.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Slide ID to fix. If empty, fixes all slides."},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "apply_smart_layout",
			Description: "Apply a predefined smart layout template to a slide. Repositions existing elements to match professional spacing. Templates: title, title_content, two_column, image_left, image_right, image_full, section, bullets, quote, three_cards, four_cards, timeline, comparison, stats, chart, agenda, thank_you, contact.",
			Parameters: params(map[string]prop{
				"slide_id":  {typ: "string", desc: "Slide ID", req: true},
				"template":  {typ: "string", desc: "Template ID (e.g. three_cards, two_column, image_left)", req: true},
			}),
		}},

		// ────────── AI Image Generation ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "import_svg",
			Description: "Compile an SVG slide design into native editable PPTX elements on a slide. The SVG must use viewBox coordinates (e.g. viewBox=\"0 0 1280 720\"). Supported primitives map to native shapes: rect→rectangle/rounded-rectangle, circle/ellipse→ellipse, line/stroked 2-point path→connector, arbitrary path (curves flattened)→editable freeform (custom geometry), text→text box, image→picture. Gradient fills are approximated as solid gray. Transform support: translate/scale/rotate/matrix. Returns the number of created elements plus any skipped constructs.",
			Parameters: params(map[string]prop{
				"slide_id": {typ: "string", desc: "Target slide ID", req: true},
				"svg":      {typ: "string", desc: "Complete SVG document markup", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "generate_image",
			Description: "Generate a professional AI image from a text prompt and return the local path. Use this to create backgrounds, illustrations, and visual assets before adding them to slides.",
			Parameters: params(map[string]prop{
				"image_prompt": {typ: "string", desc: "Detailed English prompt for image generation (describe style, mood, composition, colors)", req: true},
			}),
		}},

		// ────────── State / Export ──────────
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "get_state",
			Description: "Get the current presentation state as JSON (for review).",
			Parameters:  params(map[string]prop{}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "render_slides",
			Description: "Render the current presentation to slide images for visual review. Returns base64 PNG images and structural descriptions of each slide. Use this to visually inspect your design — check for element overlaps, alignment issues, whitespace balance, text readability, and color contrast. After reviewing the images, fix any issues using update_position/update_style/delete_element, then call render_slides again to verify your fixes. This creates a visual feedback loop for iterative design improvement.",
			Parameters:  params(map[string]prop{}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "export_pptx",
			Description: "Export the current presentation to an editable .pptx file on disk.",
			Parameters: params(map[string]prop{
				"output_path": {typ: "string", desc: "Destination .pptx file path", req: true},
			}),
		}},
		{Type: openai.ToolTypeFunction, Function: &openai.FunctionDefinition{
			Name:        "done",
			Description: "Signal that the presentation is complete and ready for export.",
			Parameters:  params(map[string]prop{}),
		}},
	}
}
