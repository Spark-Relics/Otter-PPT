// Package design implements a two-level design system borrowed from
// ppt-master's architecture: a visual STYLE (shape language, composition
// discipline, typography character — deliberately color-free) is orthogonal
// to a PALETTE (six semantic color roles). The AI picks one of each; the
// resolved combination locks the deck's look while staying composable.
package design

import (
	"fmt"
	"sort"
	"strings"
)

// Palette is a six-role semantic color scheme. Roles are stable across
// palettes so styles and palettes can be mixed freely.
type Palette struct {
	Key                 string `json:"key"`
	Name                string `json:"name"`
	Background          string `json:"background"`           // page background
	BackgroundSecondary string `json:"background_secondary"` // cards, panels, bands
	Primary             string `json:"primary"`              // primary brand / structural color
	Accent              string `json:"accent"`               // the ONE attention color
	AccentSecondary     string `json:"accent_secondary"`     // secondary highlight
	Text                string `json:"text"`                 // default text on background
	Dark                bool   `json:"dark"`                 // dark background discipline
	Brief               string `json:"brief"`                // selection rule for the AI
}

// SemanticRole is one orthogonal color reference: status colors for
// positive/negative/warning and the neutral slate grey scale.
type SemanticRole struct {
	Hex   string
	Use   string
	Rules string
}

// SemanticStatusColors returns the fixed status palette. It is orthogonal to
// Palette: positive/negative/warning must never be re-themed to brand colors,
// otherwise dashboards lose their universal "good/bad/caution" meaning.
func SemanticStatusColors() []SemanticRole {
	return []SemanticRole{
		{Hex: "#059669", Use: "positive / up / complete / on-track", Rules: "use ONLY for rising, done, met-target, supported"},
		{Hex: "#E11D48", Use: "negative / down / anomaly / missed", Rules: "use ONLY for falling, error, unmet-target, unsupported"},
		{Hex: "#D97706", Use: "warning / at-risk / pending", Rules: "use ONLY for risk, partial, in-review"},
	}
}

// NeutralReference returns the slate grey scale used by templates and previews.
// It is not a palette replacement — it guarantees every preview stays legible
// before the deck's identity colors are applied.
func NeutralReference() []SemanticRole {
	return []SemanticRole{
		{Hex: "#0F172A", Use: "primary text / titles / key values", Rules: "headings, hero numbers"},
		{Hex: "#475569", Use: "body text / legends", Rules: "descriptions, axis legends"},
		{Hex: "#64748B", Use: "secondary text / axis labels", Rules: "captions, annotations"},
		{Hex: "#94A3B8", Use: "muted text / placeholders", Rules: "footnotes, disabled, sources"},
		{Hex: "#CBD5E1", Use: "hairlines / gridlines", Rules: "dividers, table borders"},
		{Hex: "#E2E8F0", Use: "soft fills / table bands", Rules: "header bands, alternating rows"},
		{Hex: "#F8FAFC", Use: "near-white panel background", Rules: "card surfaces on white"},
	}
}

// StatusCatalog renders status + neutral reference colors for LLM prompts.
func StatusCatalog() string {
	var sb strings.Builder
	sb.WriteString("STATUS COLORS (fixed, orthogonal to palette — never re-theme):\n")
	for _, c := range SemanticStatusColors() {
		fmt.Fprintf(&sb, "- %s: %s — %s\n", c.Hex, c.Use, c.Rules)
	}
	sb.WriteString("\nNEUTRAL REFERENCE SCALE (for previews/templates, not a palette):\n")
	for _, c := range NeutralReference() {
		fmt.Fprintf(&sb, "- %s: %s\n", c.Hex, c.Use)
	}
	return sb.String()
}

// StyleSpec is the shape/composition/typography discipline. It names no
// colors — the palette owns colors. This is what keeps long decks coherent.
type StyleSpec struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Brief         string   `json:"brief"`          // selection rule for the AI
	ShapeLanguage string   `json:"shape_language"` // shape vocabulary & corner rounding
	Composition   string   `json:"composition"`    // composition patterns
	Decoration    string   `json:"decoration"`     // decoration discipline
	Typography    string   `json:"typography"`     // type character & tracking
	Depth         string   `json:"depth"`          // how elevation is expressed
	TitleFont     string   `json:"title_font"`
	BodyFont      string   `json:"body_font"`
	TitleSize     int      `json:"title_size"` // pt, content-page title anchor
	BodySize      int      `json:"body_size"`  // pt, body text anchor
	Rules         []string `json:"rules"`      // hard discipline rules for the build agent
	Recipes       []string `json:"recipes"`    // composition motifs / page recipes for the build agent
}

// ─────────────────────────────────────────────────────────────
// Style registry
// ─────────────────────────────────────────────────────────────

var styles = map[string]*StyleSpec{
	"swiss_minimal": {
		Key:           "swiss_minimal",
		Name:          "Swiss Minimal",
		Brief:         "Pick for corporate, consulting, data-driven decks. Skip for playful consumer or artistic topics (use soft_rounded or editorial).",
		ShapeLanguage: "crisp rectangles, sharp corners (no rounding), thin hairline dividers (0.5-1pt); shapes carry structure, not decoration",
		Composition:   "strict grid alignment; strong left-aligned axis; oversized headline anchoring the page; generous asymmetric whitespace",
		Decoration:    "almost none — one accent bar or rule per page maximum; no shadows, no gradients",
		Typography:    "bold tight-tracked sans titles, neutral body; hierarchy via size/weight contrast only",
		Depth:         "flat; separation via whitespace and hairlines, never shadows",
		TitleFont:     "Microsoft YaHei",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     32,
		BodySize:      14,
		Rules: []string{
			"Sharp corners only — never use rounded_rectangle",
			"No shadows anywhere",
			"At most ONE decorative accent element per page",
			"Everything aligns to a visible grid axis",
		},
		Recipes: []string{
			"Title band: oversized headline top-left, thin primary rule beneath it, body in a left column with generous right whitespace",
			"KPI strip: 3-4 metric cards on a hairline grid — big number above a small uppercase label",
			"Two-column compare: split the page vertically with a 0.5-1pt divider; facts left, visual right",
			"Section opener: giant page number + single-line title on a near-white panel",
		},
	},
	"dark_tech": {
		Key:           "dark_tech",
		Name:          "Dark Tech",
		Brief:         "Pick for tech, AI, developer tools, launches on dark canvas. Skip for print-friendly or bright daylight contexts (use cool_corporate).",
		ShapeLanguage: "crisp geometry; slightly rounded or sharp corners; hexagon/grid/circuit motifs used sparingly; thin glowing rules",
		Composition:   "elements float on dark negative space; concentric rings or diagonal traces can stage a hero metric; oversized low-opacity numeral behind content",
		Decoration:    "glow accents, fine grid backgrounds, monospace labels, node/connector lines — restrained, precision over clutter",
		Typography:    "clean sans body + monospace for labels/figures; wide tracking on small-caps labels; high-contrast hierarchy against dark",
		Depth:         "depth via glow and layering, NOT drop shadows; gradients stay same-hue and subtle",
		TitleFont:     "Microsoft YaHei",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     32,
		BodySize:      14,
		Rules: []string{
			"A dark palette is mandatory with this style (e.g. tech_neon, dark_cinematic)",
			"Use fill_opacity 0.1-0.2 panels for cards on dark, with thin bright borders",
			"Glow > shadow: prefer colored thin borders over drop shadows",
			"Monospace-feel labels: uppercase + letter-spacing for small labels",
		},
		Recipes: []string{
			"Hero orbit: centered metric in a concentric-ring stage with monospace labels around it",
			"Node grid: 3-4 translucent panels connected by thin connector lines and small nodes",
			"Code/data panel: dark card with monospace values, thin cyan/purple border, subtle glow accent",
			"Timeline rail: horizontal glowing rule with node markers and short callouts",
		},
	},
	"editorial": {
		Key:           "editorial",
		Name:          "Editorial",
		Brief:         "Pick for storytelling, reports, thought leadership, humanities. Skip for dense data dashboards (use swiss_minimal).",
		ShapeLanguage: "classical rectangles, occasional thin rules; content sits in 'columns' like a magazine spread",
		Composition:   "magazine spread: large serif headline, deck (subtitle), drop-cap or pull-quote potential; images framed with generous margins",
		Decoration:    "hairline rules, small caps section labels, page numbers — quiet and literary",
		Typography:    "serif or serif-feel titles + humanist body; generous line spacing (1.4-1.6)",
		Depth:         "flat with paper-like layering; no heavy shadows",
		TitleFont:     "Georgia",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     34,
		BodySize:      15,
		Rules: []string{
			"Use line_spacing 1.4+ for body text",
			"Hairline rules (0.75pt) as section separators",
			"Pull-quotes: large italic text with accent left bar",
			"Restrained color: accent used for rules and emphasis only",
		},
		Recipes: []string{
			"Magazine spread: large serif headline, deck paragraph, drop-cap or pull-quote, generous margins",
			"Image feature: full-bleed image with a quiet caption strip and a small-caps section label",
			"Pull quote: oversized italic quote with an accent left bar",
			"Column argument: two reading columns separated by a hairline, page number at foot",
		},
	},
	"glassmorphism": {
		Key:           "glassmorphism",
		Name:          "Glassmorphism",
		Brief:         "Pick for modern product, SaaS, futuristic UI-flavored decks. Skip for print or conservative corporate (use swiss_minimal).",
		ShapeLanguage: "rounded cards (corner_radius 0.08-0.15) that read as frosted glass panels floating over a vivid background",
		Composition:   "full-bleed gradient or photo background; 2-4 glass panels per page layered at different sizes; overlapping allowed",
		Decoration:    "soft blur illusion via translucent fills (fill_opacity 0.55-0.75) + thin light borders (1pt, near-white at 40-60% opacity)",
		Typography:    "clean geometric sans; medium-weight titles on glass, light body",
		Depth:         "depth via transparency layering and subtle borders; shadows very soft or none",
		TitleFont:     "Microsoft YaHei",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     32,
		BodySize:      14,
		Rules: []string{
			"Cards MUST be translucent: fill_opacity 0.55-0.75 + thin light border",
			"Background is always a vivid gradient or image — never flat white",
			"corner_radius 0.08-0.15 on all cards",
			"Max 4 glass panels per page",
		},
		Recipes: []string{
			"Full-bleed hero: vivid gradient background with one large frosted title card",
			"Glass card grid: 2-4 translucent rounded cards layered over the background",
			"KPI glass tiles: compact frosted metric tiles with thin light borders",
			"Layered overlap: offset overlapping glass panels to create depth",
		},
	},
	"soft_rounded": {
		Key:           "soft_rounded",
		Name:          "Soft Rounded",
		Brief:         "Pick for education, consumer, health, friendly onboarding topics. Skip for finance/consulting formality (use swiss_minimal).",
		ShapeLanguage: "large radii (corner_radius 0.12-0.2), pill shapes, soft circles; nothing sharp",
		Composition:   "card-based layouts with breathing room; icons/numbers in soft circles; center- or card-aligned rather than strict grid",
		Decoration:    "soft pastel blobs and circles at low opacity; gentle, never loud",
		Typography:    "rounded-feel sans, medium weights; friendly hierarchy, avoid ALL-CAPS",
		Depth:         "soft diffuse shadows (large blur, low opacity) to lift cards",
		TitleFont:     "Microsoft YaHei",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     30,
		BodySize:      14,
		Rules: []string{
			"corner_radius >= 0.12 on every card",
			"Shadow=true on cards for soft lift",
			"Use soft pastel decorative circles at fill_opacity 0.15-0.3",
			"No sharp angular shapes; no ALL-CAPS body text",
		},
		Recipes: []string{
			"Card cluster: 3-4 soft rounded cards with pastel circles and icons",
			"Pill steps: horizontal pill-shaped steps with numbers in soft circles",
			"Friendly KPI: big rounded number tiles with soft shadows",
			"Center hero: centered title with decorative pastel blobs at low opacity",
		},
	},
	"gradient_modern": {
		Key:           "gradient_modern",
		Name:          "Gradient Modern",
		Brief:         "Pick for product launches, startups, innovation showcases. Skip for conservative or data-dense decks (use swiss_minimal).",
		ShapeLanguage: "clean rounded rectangles (corner_radius 0.06-0.1) with gradient fills; gradient text-panels; slanted accent bars",
		Composition:   "hero gradient background; content in clean cards; diagonal energy lines or gradient bands as movement",
		Decoration:    "vibrant multi-hue gradients (2-3 stops) on heroes and key shapes; soft glow on accent numbers",
		Typography:    "bold modern sans; extra-bold headlines, gradient-adjacent accent colors for emphasis",
		Depth:         "gradient layering + very soft shadows",
		TitleFont:     "Microsoft YaHei",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     34,
		BodySize:      14,
		Rules: []string{
			"Hero/title areas use set_bg_gradient with 2-3 vivid stops",
			"Key shapes use gradient fills, not flat colors",
			"Keep body text areas on neutral panels for readability",
			"One gradient direction per deck (e.g. always diagonal)",
		},
		Recipes: []string{
			"Gradient hero: full-bleed diagonal gradient band with a bold headline",
			"Slanted accent: diagonal gradient bars slicing card corners for motion",
			"Card grid with gradient headers: cards topped by thin gradient strips",
			"KPI with gradient numerals: big gradient-filled numbers",
		},
	},
	"blueprint": {
		Key:           "blueprint",
		Name:          "Blueprint",
		Brief:         "Pick for engineering, architecture, system design, technical deep-dives. Skip for marketing/storytelling (use editorial or gradient_modern).",
		ShapeLanguage: "thin precise outlines (0.75-1.25pt strokes), technical rectangles, connector lines with small nodes; dashed guides",
		Composition:   "schematic layouts: labeled blocks connected by thin lines; dimension-like annotations; grid background at low opacity",
		Decoration:    "measurement ticks, coordinate labels, dashed construction lines — all thin and quiet",
		Typography:    "monospace-feel labels; uppercase small labels with wide tracking; precise numbered callouts",
		Depth:         "flat — layering via line weight and dashed vs solid",
		TitleFont:     "Microsoft YaHei",
		BodyFont:      "Microsoft YaHei",
		TitleSize:     30,
		BodySize:      13,
		Rules: []string{
			"Blocks: thin outline + transparent or very light fill",
			"Connect blocks with add_connector lines (0.75-1.25pt)",
			"Uppercase + letter-spacing for all small labels",
			"Works best on a light grid-like background or cool_corporate palette",
		},
		Recipes: []string{
			"Schematic block: labeled rectangles connected by thin lines with small nodes",
			"Dimensioned layout: content blocks with dashed guides and coordinate labels",
			"Process rail: numbered callout nodes along a thin spine",
			"Annotated diagram: technical diagram with uppercase small labels and measurement ticks",
		},
	},
}

// ─────────────────────────────────────────────────────────────
// Palette registry
// ─────────────────────────────────────────────────────────────

var palettes = map[string]*Palette{
	"tech_neon": {
		Key: "tech_neon", Name: "Tech Neon", Dark: true,
		Background: "#0A0E1A", BackgroundSecondary: "#141B2E",
		Primary: "#22D3EE", Accent: "#A78BFA", AccentSecondary: "#F472B6",
		Text:  "#E2E8F0",
		Brief: "Pick with dark_tech for AI/dev products. Skip for print or daylight rooms.",
	},
	"cool_corporate": {
		Key: "cool_corporate", Name: "Cool Corporate", Dark: false,
		Background: "#FFFFFF", BackgroundSecondary: "#F1F5F9",
		Primary: "#1E5AA8", Accent: "#0EA5E9", AccentSecondary: "#64748B",
		Text:  "#1E293B",
		Brief: "Pick for business, consulting, finance, reports. The safe professional default.",
	},
	"dark_cinematic": {
		Key: "dark_cinematic", Name: "Dark Cinematic", Dark: true,
		Background: "#0D0D0F", BackgroundSecondary: "#1C1C21",
		Primary: "#D4AF37", Accent: "#C0392B", AccentSecondary: "#9AA0A6",
		Text:  "#F5F5F0",
		Brief: "Pick for keynote drama, luxury, awards, film. Skip for friendly/casual decks.",
	},
	"editorial_classic": {
		Key: "editorial_classic", Name: "Editorial Classic", Dark: false,
		Background: "#FAF7F2", BackgroundSecondary: "#F0EBE0",
		Primary: "#1A1A1A", Accent: "#B33A3A", AccentSecondary: "#8A7968",
		Text:  "#262626",
		Brief: "Pick with editorial for essays, reports, humanities, publishing.",
	},
	"warm_earth": {
		Key: "warm_earth", Name: "Warm Earth", Dark: false,
		Background: "#FDF8F2", BackgroundSecondary: "#F3E9DC",
		Primary: "#A66A2C", Accent: "#C75B39", AccentSecondary: "#7D8C5C",
		Text:  "#3E2F23",
		Brief: "Pick for sustainability, food, agriculture, wellness, culture.",
	},
	"sunset_gradient": {
		Key: "sunset_gradient", Name: "Sunset Gradient", Dark: false,
		Background: "#FFF7ED", BackgroundSecondary: "#FFE4D6",
		Primary: "#EA580C", Accent: "#DB2777", AccentSecondary: "#7C3AED",
		Text:  "#431407",
		Brief: "Pick with gradient_modern for launches and vibrant product stories.",
	},
	"jewel_tone": {
		Key: "jewel_tone", Name: "Jewel Tone", Dark: false,
		Background: "#F8FAFC", BackgroundSecondary: "#EDEFF7",
		Primary: "#1D4ED8", Accent: "#7C3AED", AccentSecondary: "#0D9488",
		Text:  "#0F172A",
		Brief: "Pick for rich premium variety: multiple distinct content streams needing distinct colors.",
	},
	"mono_ink": {
		Key: "mono_ink", Name: "Mono Ink", Dark: false,
		Background: "#FFFFFF", BackgroundSecondary: "#F5F5F5",
		Primary: "#111111", Accent: "#444444", AccentSecondary: "#888888",
		Text:  "#111111",
		Brief: "Pick with swiss_minimal for brutalist clarity or photography-led decks where color must stay neutral.",
	},
}

// GetStyle returns the style spec for a key, or nil.
func GetStyle(key string) *StyleSpec { return styles[strings.ToLower(key)] }

// GetPalette returns the palette for a key, or nil.
func GetPalette(key string) *Palette { return palettes[strings.ToLower(key)] }

// StyleKeys lists available style keys in stable order.
func StyleKeys() []string {
	keys := make([]string, 0, len(styles))
	for k := range styles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// PaletteKeys lists available palette keys in stable order.
func PaletteKeys() []string {
	keys := make([]string, 0, len(palettes))
	for k := range palettes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// StyleCatalog renders the style selection rules for LLM prompts.
func StyleCatalog() string {
	var sb strings.Builder
	for _, k := range StyleKeys() {
		s := styles[k]
		fmt.Fprintf(&sb, "- %s (%s): %s\n", s.Key, s.Name, s.Brief)
	}
	return sb.String()
}

// PaletteCatalog renders the palette selection rules for LLM prompts.
func PaletteCatalog() string {
	var sb strings.Builder
	for _, k := range PaletteKeys() {
		p := palettes[k]
		mood := "light"
		if p.Dark {
			mood = "dark"
		}
		fmt.Fprintf(&sb, "- %s (%s, %s): bg=%s primary=%s accent=%s — %s\n",
			p.Key, p.Name, mood, p.Background, p.Primary, p.Accent, p.Brief)
	}
	return sb.String()
}

// Lock renders the full design lock: the immutable per-deck contract the
// build agent must obey on every page. Borrowed from ppt-master's spec_lock
// concept — a derived snapshot that prevents style drift across long decks.
func Lock(styleKey, paletteKey string) string {
	s := GetStyle(styleKey)
	p := GetPalette(paletteKey)
	var sb strings.Builder
	sb.WriteString("🔒 DESIGN LOCK — obey on EVERY slide. Do not invent colors/fonts outside this lock.\n\n")
	if s != nil {
		fmt.Fprintf(&sb, "STYLE: %s (%s)\n", s.Key, s.Name)
		fmt.Fprintf(&sb, "- Shape language: %s\n", s.ShapeLanguage)
		fmt.Fprintf(&sb, "- Composition: %s\n", s.Composition)
		fmt.Fprintf(&sb, "- Decoration: %s\n", s.Decoration)
		fmt.Fprintf(&sb, "- Typography: %s\n", s.Typography)
		fmt.Fprintf(&sb, "- Depth: %s\n", s.Depth)
		fmt.Fprintf(&sb, "- Fonts: title=%s %dpt / body=%s %dpt\n", s.TitleFont, s.TitleSize, s.BodyFont, s.BodySize)
		sb.WriteString("- Hard rules:\n")
		for _, r := range s.Rules {
			fmt.Fprintf(&sb, "  * %s\n", r)
		}
		sb.WriteString("- Composition recipes (page motifs):\n")
		for _, r := range s.Recipes {
			fmt.Fprintf(&sb, "  * %s\n", r)
		}
	}
	if p != nil {
		fmt.Fprintf(&sb, "\nPALETTE: %s (%s)\n", p.Key, p.Name)
		fmt.Fprintf(&sb, "- background=%s (page bg)\n", p.Background)
		fmt.Fprintf(&sb, "- background_secondary=%s (cards/panels/bands)\n", p.BackgroundSecondary)
		fmt.Fprintf(&sb, "- primary=%s (structure, titles, key shapes)\n", p.Primary)
		fmt.Fprintf(&sb, "- accent=%s (THE attention color — sparingly)\n", p.Accent)
		fmt.Fprintf(&sb, "- accent_secondary=%s (secondary highlight)\n", p.AccentSecondary)
		fmt.Fprintf(&sb, "- text=%s\n", p.Text)
		if p.Dark {
			sb.WriteString("- Dark discipline: never place dark text on dark panels; borders and dividers use light tints\n")
		}
	}
	sb.WriteString("\n" + StatusCatalog() + "\n")
	return sb.String()
}
