// Package gallery builds executable visual specimens for every design style
// preset. Where design.StyleSpec describes a style in prose, gallery turns the
// prose into an actual deck (cover + signature content page) so that style
// selection — by humans or AI agents — can be made by looking, not reading.
//
// Every specimen deck strictly obeys its own style's hard rules: shapes,
// colors, corner radii, opacities and composition are derived from the
// StyleSpec + Palette pair, not hand-picked per style.
package gallery

import (
	"fmt"

	"github.com/otter-ppt/otter-ppt/internal/design"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

// Specimens lists the canonical style × palette pairings shown in the gallery.
// Pairings follow each style's own selection brief (e.g. dark_tech → tech_neon).
func Specimens() []Spec {
	return []Spec{
		{Style: "swiss_minimal", Palette: "cool_corporate"},
		{Style: "dark_tech", Palette: "tech_neon"},
		{Style: "editorial", Palette: "editorial_classic"},
		{Style: "glassmorphism", Palette: "sunset_gradient"},
		{Style: "soft_rounded", Palette: "warm_earth"},
		{Style: "gradient_modern", Palette: "sunset_gradient"},
		{Style: "blueprint", Palette: "cool_corporate"},
	}
}

// Spec is one style × palette pairing.
type Spec struct {
	Style   string
	Palette string
}

// Build returns a 2-slide specimen deck for the given pairing:
// slide 1 = anchor (cover, style signature), slide 2 = dense (signature content recipe).
func Build(spec Spec) (*model.Presentation, error) {
	s := design.GetStyle(spec.Style)
	if s == nil {
		return nil, fmt.Errorf("unknown style %q (available: %v)", spec.Style, design.StyleKeys())
	}
	p := design.GetPalette(spec.Palette)
	if p == nil {
		return nil, fmt.Errorf("unknown palette %q (available: %v)", spec.Palette, design.PaletteKeys())
	}

	pres := &model.Presentation{
		Title: fmt.Sprintf("OtterPPT Style Gallery — %s", s.Name),
		Theme: model.Theme{
			Name:            fmt.Sprintf("%s × %s", s.Name, p.Name),
			PrimaryColor:    p.Primary,
			SecondaryColor:  p.BackgroundSecondary,
			AccentColor:     p.Accent,
			BackgroundColor: p.Background,
			TextColor:       p.Text,
			TitleFont:       s.TitleFont,
			BodyFont:        s.BodyFont,
			StyleKey:        s.Key,
			PaletteKey:      p.Key,
		},
		Slides: []*model.Slide{
			coverSlide(s, p),
			contentSlide(s, p),
		},
	}
	return pres, nil
}

// ─────────────────────────────────────────────────────────────
// element helpers
// ─────────────────────────────────────────────────────────────

func text(id, txt string, r model.Rect, st model.TextStyle) *model.Element {
	return &model.Element{ID: id, Type: model.ElementBody, Rect: r, Text: txt, Style: st}
}

func shape(id string, r model.Rect, sd *model.ShapeData) *model.Element {
	return &model.Element{ID: id, Type: model.ElementShape, Rect: r, Shape: sd}
}

func rect(x, y, w, h float64) model.Rect {
	return model.Rect{X: x, Y: y, W: w, H: h}
}

func solid(color string, opacity float64) *model.FillStyle {
	return &model.FillStyle{Color: color, Opacity: opacity}
}

func outline(color string, width float64) *model.LineStyle {
	return &model.LineStyle{Color: color, Width: width}
}


// ─────────────────────────────────────────────────────────────
// cover slide (anchor rhythm) — per-style signature
// ─────────────────────────────────────────────────────────────

func coverSlide(s *design.StyleSpec, p *design.Palette) *model.Slide {
	els := []*model.Element{}
	switch s.Key {

	case "swiss_minimal":
		// Rule: oversized headline top-left, thin primary rule, giant page numeral.
		els = append(els,
			text("eyebrow", "STYLE SPECIMEN · 01", rect(8, 14, 50, 4), model.TextStyle{
				FontSize: 11, Color: p.Accent, Bold: true, LetterSpacing: 3}),
			text("title", "Swiss Minimal", rect(8, 22, 70, 16), model.TextStyle{
				FontSize: s.TitleSize * 2, Bold: true, Color: p.Text}),
			shape("rule", rect(8, 44, 84, 0.4), &model.ShapeData{ // accent bar (the ONE decoration)
				ShapeType: model.ShapeRectangle, Fill: solid(p.Primary, 1)}),
			text("lead", "Grid. Hierarchy. Nothing else.", rect(8, 50, 60, 6), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Color: p.Text, Opacity: 0.75}),
			text("numeral", "01", rect(70, 60, 22, 28), model.TextStyle{
				FontSize: s.BodySize * 8, Bold: true, Color: p.Primary, Opacity: 0.12, Align: "right"}),
		)

	case "dark_tech":
		// Recipe: hero orbit — concentric ring stage + monospace-feel labels.
		els = append(els,
			shape("ring-outer", rect(30, 8, 40, 71), &model.ShapeData{ // concentric rings stage
				ShapeType: model.ShapeEllipse, Fill: solid(p.Primary, 0.04), Line: outline(p.Primary, 0.75)}),
			shape("ring-inner", rect(37, 22, 26, 46), &model.ShapeData{
				ShapeType: model.ShapeEllipse, Fill: solid(p.Accent, 0.06), Line: outline(p.Accent, 0.75)}),
			text("eyebrow", "STYLE SPECIMEN // 02", rect(8, 12, 50, 4), model.TextStyle{
				FontSize: 11, Color: p.Primary, Bold: true, LetterSpacing: 3}),
			text("title", "Dark Tech", rect(8, 20, 50, 10), model.TextStyle{
				FontSize: int(float64(s.TitleSize) * 1.5), Bold: true, Color: p.Text}),
			text("lead", "Signals in the dark.", rect(8, 33, 40, 5), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Color: p.Text, Opacity: 0.7}),
			text("hero", "∞", rect(41, 38, 18, 10), model.TextStyle{
				FontSize: 40, Bold: true, Color: p.Primary, Align: "center"}),
			text("stat", "UPTIME 99.99%", rect(36, 60, 28, 4), model.TextStyle{
				FontSize: 11, Color: p.Accent, Align: "center", LetterSpacing: 2}),
		)

	case "editorial":
		// Recipe: magazine spread — serif headline, deck, pull-quote bar.
		els = append(els,
			text("eyebrow", "STYLE SPECIMEN · No. 03", rect(8, 12, 50, 4), model.TextStyle{
				FontSize: 11, Color: p.Accent, LetterSpacing: 2}),
			text("title", "The Editorial", rect(8, 20, 62, 16), model.TextStyle{
				FontSize: s.TitleSize * 2, Color: p.Text, FontName: s.TitleFont}),
			text("deck", "Stories set like a magazine spread — serif headlines, generous margins, quiet rules.", rect(8, 42, 52, 10), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Color: p.Text, LineSpacing: 1.5}),
			shape("quotbar", rect(8, 60, 0.5, 20), &model.ShapeData{ // pull-quote accent bar
				ShapeType: model.ShapeRectangle, Fill: solid(p.Accent, 1)}),
			text("quote", "Typography is the voice.\nLayout is the pause.", rect(11, 60, 44, 20), model.TextStyle{
				FontSize: s.BodySize * 3 / 2, Italic: true, Color: p.Text, LineSpacing: 1.4}),
			text("pageno", "— 03 —", rect(44, 88, 12, 3), model.TextStyle{
				FontSize: 10, Color: p.Text, Opacity: 0.5, Align: "center"}),
		)

	case "glassmorphism":
		// Recipe: full-bleed vivid gradient + one large frosted title card.
		els = append(els,
			shape("glass", rect(10, 22, 80, 56), &model.ShapeData{
				ShapeType:    model.ShapeRoundedRectangle, CornerRadius: 0.1,
				Fill:         solid("#FFFFFF", 0.55),
				Line:         outline("#FFFFFF", 1),
				Shadow:       &model.ShadowStyle{Color: "#000000", Opacity: 0.15, Blur: 30, Distance: 8},
			}),
			text("eyebrow", "STYLE SPECIMEN · 04", rect(18, 32, 50, 4), model.TextStyle{
				FontSize: 11, Color: p.Primary, Bold: true, LetterSpacing: 3}),
			text("title", "Glassmorphism", rect(18, 39, 64, 12), model.TextStyle{
				FontSize: s.TitleSize * 2, Bold: true, Color: p.Text}),
			text("lead", "Frosted panels floating over vivid light.", rect(18, 56, 60, 6), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Color: p.Text, Opacity: 0.85}),
		)

	case "soft_rounded":
		// Recipe: center hero — soft blobs at low opacity + rounded title card.
		els = append(els,
			shape("blob1", rect(-8, -10, 30, 40), &model.ShapeData{
				ShapeType: model.ShapeEllipse, Fill: solid(p.Accent, 0.18)}),
			shape("blob2", rect(78, 72, 30, 40), &model.ShapeData{
				ShapeType: model.ShapeEllipse, Fill: solid(p.Primary, 0.15)}),
			shape("card", rect(14, 28, 72, 44), &model.ShapeData{
				ShapeType:    model.ShapeRoundedRectangle, CornerRadius: 0.16,
				Fill:         solid(p.BackgroundSecondary, 1),
				Shadow:       &model.ShadowStyle{Color: "#000000", Opacity: 0.10, Blur: 24, Distance: 6},
			}),
			text("eyebrow", "Style Specimen · 05", rect(0, 36, 100, 4), model.TextStyle{
				FontSize: 11, Color: p.Accent, Align: "center"}),
			text("title", "Soft & Rounded", rect(0, 42, 100, 10), model.TextStyle{
				FontSize: s.TitleSize * 2, Bold: true, Color: p.Text, Align: "center"}),
			text("lead", "Friendly shapes, gentle shadows, pastel calm.", rect(20, 56, 60, 5), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Color: p.Text, Opacity: 0.8, Align: "center"}),
		)

	case "gradient_modern":
		// Recipe: gradient hero — full-bleed diagonal gradient + bold headline.
		els = append(els,
			shape("band", rect(0, 0, 100, 62), &model.ShapeData{
				ShapeType: model.ShapeRectangle,
				Fill: &model.FillStyle{Gradient: &model.Gradient{
					Type: model.GradientLinear, Angle: 45,
					Stops: []model.GradientStop{
						{Color: p.Primary, Position: 0},
						{Color: p.Accent, Position: 100},
					},
				}},
			}),
			text("eyebrow", "STYLE SPECIMEN · 06", rect(8, 18, 50, 4), model.TextStyle{
				FontSize: 11, Color: "#FFFFFF", Bold: true, LetterSpacing: 3}),
			text("lead", "Motion you can almost feel.", rect(8, 43, 60, 5), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Color: "#FFFFFF", Opacity: 0.9}),
			shape("slant", rect(70, 70, 30, 2), &model.ShapeData{ // slanted accent strip
				ShapeType: model.ShapeRectangle, Fill: solid(p.Accent, 1)}),
		)

	case "blueprint":
		// Recipe: schematic block — labeled outline blocks + thin connectors.
		els = append(els,
			text("eyebrow", "STYLE SPECIMEN / 07", rect(8, 12, 50, 4), model.TextStyle{
				FontSize: 11, Color: p.Primary, Bold: true, LetterSpacing: 2}),
			text("title", "Blueprint", rect(8, 19, 50, 10), model.TextStyle{
				FontSize: s.TitleSize * 2, Bold: true, Color: p.Text}),
			shape("blk-in", rect(8, 52, 24, 14), &model.ShapeData{ // schematic blocks
				ShapeType: model.ShapeRectangle, Fill: solid(p.Primary, 0.06), Line: outline(p.Primary, 1)}),
			text("blk-in-l", "INPUT", rect(8, 56, 24, 5), model.TextStyle{
				FontSize: 12, Bold: true, Color: p.Text, Align: "center", LetterSpacing: 2}),
			shape("blk-core", rect(40, 46, 26, 26), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Accent, 0.08), Line: outline(p.Accent, 1.25)}),
			text("blk-core-l", "CORE\nENGINE", rect(40, 53, 26, 10), model.TextStyle{
				FontSize: 12, Bold: true, Color: p.Text, Align: "center", LetterSpacing: 2}),
			shape("blk-out", rect(74, 52, 24, 14), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Primary, 0.06), Line: outline(p.Primary, 1)}),
			text("blk-out-l", "OUTPUT", rect(74, 56, 24, 5), model.TextStyle{
				FontSize: 12, Bold: true, Color: p.Text, Align: "center", LetterSpacing: 2}),
			shape("cn1", rect(32, 59, 8, 0.3), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Text, 0.7)}),
			shape("cn2", rect(66, 59, 8, 0.3), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Text, 0.7)}),
			text("dims", "SCALE 1:1 · REV A", rect(8, 84, 40, 3), model.TextStyle{
				FontSize: 10, Color: p.Text, Opacity: 0.55, LetterSpacing: 2}),
		)
	}
	return &model.Slide{
		ID:         "cover",
		Background: &model.Background{Type: model.BgSolid, Color: p.Background},
		Elements:   els,
	}
}

// ─────────────────────────────────────────────────────────────
// content slide (dense rhythm) — the style's most representative recipe
// ─────────────────────────────────────────────────────────────

func contentSlide(s *design.StyleSpec, p *design.Palette) *model.Slide {
	els := []*model.Element{}
	switch s.Key {

	case "swiss_minimal":
		// Recipe: KPI strip on a hairline grid.
		els = append(els,
			text("title", "KPI Strip", rect(8, 10, 60, 8), model.TextStyle{
				FontSize: s.TitleSize, Bold: true, Color: p.Text}),
			shape("rule", rect(8, 20, 84, 0.3), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Primary, 1)}),
		)
		kpis := []struct{ v, l string }{{"84%", "COVERAGE"}, {"3.2×", "THROUGHPUT"}, {"12ms", "LATENCY"}}
		for i, k := range kpis {
			x := 8 + float64(i)*28.7
			if i > 0 { // hairline dividers
				els = append(els, shape(fmt.Sprintf("div%d", i), rect(x-2.2, 30, 0.2, 24), &model.ShapeData{
					ShapeType: model.ShapeRectangle, Fill: solid(p.Text, 0.25)}))
			}
			els = append(els,
				text(fmt.Sprintf("kpi-v%d", i), k.v, rect(x, 32, 24, 12), model.TextStyle{
					FontSize: s.BodySize * 3, Bold: true, Color: p.Primary}),
				text(fmt.Sprintf("kpi-l%d", i), k.l, rect(x, 45, 24, 4), model.TextStyle{
					FontSize: s.BodySize, Color: p.Text, Opacity: 0.6, LetterSpacing: 2}),
			)
		}
		els = append(els, text("body", "Metrics sit on an invisible grid. No cards, no shadows — whitespace does the separating.", rect(8, 66, 70, 8), model.TextStyle{
			FontSize: s.BodySize, Color: p.Text, Opacity: 0.8, LineSpacing: 1.4}))

	case "dark_tech":
		// Recipe: node grid — translucent panels connected by thin lines.
		els = append(els,
			text("title", "Node Grid", rect(8, 10, 60, 8), model.TextStyle{
				FontSize: s.TitleSize, Bold: true, Color: p.Text}),
			text("eyebrow", "SIGNATURE RECIPE // NODES", rect(8, 19, 60, 3), model.TextStyle{
				FontSize: 10, Color: p.Primary, LetterSpacing: 2}),
		)
		nodes := []struct {
			x, y float64
			c    string
		}{{8, 30, p.Primary}, {40, 30, p.Accent}, {72, 30, p.AccentSecondary}}
		for i, n := range nodes {
			els = append(els,
				shape(fmt.Sprintf("node%d", i), rect(n.x, n.y, 20, 26), &model.ShapeData{
					ShapeType: model.ShapeRoundedRectangle, CornerRadius: 0.06,
					Fill:      solid(n.c, 0.12), Line: outline(n.c, 0.75)}),
				text(fmt.Sprintf("node-l%d", i), fmt.Sprintf("NODE-%02d", i+1), rect(n.x, n.y+3, 20, 4), model.TextStyle{
					FontSize: 11, Color: n.c, Bold: true, LetterSpacing: 2, Align: "center"}),
				text(fmt.Sprintf("node-v%d", i), []string{"128ms", "99.9%", "42/s"}[i], rect(n.x, n.y+11, 20, 8), model.TextStyle{
					FontSize: 22, Bold: true, Color: p.Text, Align: "center"}),
			)
			if i > 0 {
				px := nodes[i-1].x + 20
				els = append(els, shape(fmt.Sprintf("link%d", i), rect(px, 42.6, float64(int(n.x)-int(px)), 0.25), &model.ShapeData{
					ShapeType: model.ShapeRectangle, Fill: solid(p.Primary, 0.5)}))
			}
		}
		els = append(els, text("body", "Panels float on dark negative space; connectors glow instead of shadows.", rect(8, 66, 70, 6), model.TextStyle{
			FontSize: s.BodySize, Color: p.Text, Opacity: 0.7}))

	case "editorial":
		els = append(els,
			text("title", "The Column Argument", rect(8, 10, 70, 8), model.TextStyle{
				FontSize: s.TitleSize, Color: p.Text, FontName: s.TitleFont}),
			shape("rule", rect(8, 20, 84, 0.3), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Accent, 1)}),
			text("col1", "The first column argues in prose. Long-form reading wants measure, rhythm and a quiet margin — not chrome.", rect(8, 26, 38, 30), model.TextStyle{
				FontSize: s.BodySize, Color: p.Text, LineSpacing: 1.5}),
			shape("hair", rect(50, 26, 0.2, 42), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Text, 0.3)}),
			text("col2", "The second column answers. Two voices, one page. The hairline between them is the only decoration allowed.", rect(54, 26, 38, 30), model.TextStyle{
				FontSize: s.BodySize, Color: p.Text, LineSpacing: 1.5}),
			shape("qbar", rect(54, 62, 0.5, 8), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Accent, 1)}),
			text("quote", "White space is content.", rect(57, 62, 35, 8), model.TextStyle{
				FontSize: s.BodySize * 6 / 5, Italic: true, Color: p.Text}),
			text("pageno", "04 · SPECIMEN", rect(66, 90, 26, 3), model.TextStyle{
				FontSize: 9, Color: p.Text, Opacity: 0.5, Align: "right", LetterSpacing: 1.5}),
		)

	case "glassmorphism":
		// Recipe: glass card grid — 3 frosted tiles over vivid gradient.
		els = append(els,
			text("title", "Glass Tiles", rect(8, 10, 60, 8), model.TextStyle{
				FontSize: s.TitleSize, Bold: true, Color: "#FFFFFF"}),
		)
		labels := []struct{ t, d string }{{"Translucent", "0.55–0.75 fill opacity"}, {"Light border", "1pt near-white"}, {"Vivid backdrop", "gradient, never flat"}}
		for i, l := range labels {
			x := 8 + float64(i)*29.3
			els = append(els,
				shape(fmt.Sprintf("glass%d", i), rect(x, 26, 26, 36), &model.ShapeData{
					ShapeType: model.ShapeRoundedRectangle, CornerRadius: 0.1,
					Fill:      solid("#FFFFFF", 0.55), Line: outline("#FFFFFF", 1)}),
				text(fmt.Sprintf("glass-t%d", i), l.t, rect(x+2, 30, 22, 5), model.TextStyle{
					FontSize: 15, Bold: true, Color: p.Text}),
				text(fmt.Sprintf("glass-d%d", i), l.d, rect(x+2, 37, 22, 10), model.TextStyle{
					FontSize: 12, Color: p.Text, Opacity: 0.8, LineSpacing: 1.3}),
			)
		}

	case "soft_rounded":
		// Recipe: pill steps with numbers in soft circles.
		els = append(els,
			text("title", "Pill Steps", rect(8, 10, 60, 8), model.TextStyle{
				FontSize: s.TitleSize, Bold: true, Color: p.Text}),
		)
		steps := []string{"Pick", "Soften", "Ship"}
		for i, st := range steps {
			x := 8 + float64(i)*29.3
			els = append(els,
				shape(fmt.Sprintf("pill%d", i), rect(x, 30, 26, 14), &model.ShapeData{
					ShapeType: model.ShapeRoundedRectangle, CornerRadius: 0.5,
					Fill:      solid(p.BackgroundSecondary, 1),
					Shadow:    &model.ShadowStyle{Color: "#000000", Opacity: 0.10, Blur: 18, Distance: 5},
				}),
				shape(fmt.Sprintf("dot%d", i), rect(x+2, 33, 8, 8), &model.ShapeData{
					ShapeType: model.ShapeEllipse, Fill: solid(p.Accent, 1)}),
				text(fmt.Sprintf("dot-n%d", i), fmt.Sprint(i+1), rect(x+2, 33, 8, 8), model.TextStyle{
					FontSize: 12, Bold: true, Color: "#FFFFFF", Align: "center"}),
				text(fmt.Sprintf("pill-l%d", i), st, rect(x+11, 33, 13, 8), model.TextStyle{
					FontSize: 15, Bold: true, Color: p.Text}),
			)
		}
		els = append(els, text("body", "Every radius is generous. Shadows are wide, soft and quiet. Nothing sharp, nothing loud.", rect(8, 56, 66, 8), model.TextStyle{
			FontSize: s.BodySize, Color: p.Text, Opacity: 0.8, LineSpacing: 1.4}))

	case "gradient_modern":
		// Recipe: card grid with gradient headers.
		els = append(els,
			text("title", "Gradient Cards", rect(8, 10, 60, 8), model.TextStyle{
				FontSize: s.TitleSize, Bold: true, Color: p.Text}),
		)
		cards := []struct{ t, d, c1, c2 string }{
			{"Diagonal", "One gradient direction per deck", p.Primary, p.Accent},
			{"Vivid", "2–3 stops on heroes and key shapes", p.Accent, p.AccentSecondary},
			{"Readable", "Body text stays on neutral panels", "#475569", "#94A3B8"},
		}
		for i, c := range cards {
			x := 8 + float64(i)*29.3
			els = append(els,
				shape(fmt.Sprintf("card%d", i), rect(x, 26, 26, 32), &model.ShapeData{
					ShapeType: model.ShapeRoundedRectangle, CornerRadius: 0.08,
					Fill:      solid(p.BackgroundSecondary, 1)}),
				shape(fmt.Sprintf("card-h%d", i), rect(x, 26, 26, 3), &model.ShapeData{ // gradient header strip
					ShapeType: model.ShapeRectangle,
					Fill: &model.FillStyle{Gradient: &model.Gradient{
						Type: model.GradientLinear, Angle: 45,
						Stops: []model.GradientStop{{Color: c.c1, Position: 0}, {Color: c.c2, Position: 1}},
					}},
				}),
				text(fmt.Sprintf("card-t%d", i), c.t, rect(x+2, 32, 22, 5), model.TextStyle{
					FontSize: 15, Bold: true, Color: p.Text}),
				text(fmt.Sprintf("card-d%d", i), c.d, rect(x+2, 39, 22, 14), model.TextStyle{
					FontSize: 12, Color: p.Text, Opacity: 0.8, LineSpacing: 1.3}),
			)
		}

	case "blueprint":
		// Recipe: process rail — numbered callout nodes along a thin spine.
		els = append(els,
			text("title", "Process Rail", rect(8, 10, 60, 8), model.TextStyle{
				FontSize: s.TitleSize, Bold: true, Color: p.Text}),
			text("eyebrow", "SIGNATURE RECIPE / RAIL", rect(8, 19, 60, 3), model.TextStyle{
				FontSize: 10, Color: p.Primary, LetterSpacing: 2}),
			shape("spine", rect(8, 44, 84, 0.3), &model.ShapeData{
				ShapeType: model.ShapeRectangle, Fill: solid(p.Text, 0.6)}),
		)
		for i := 0; i < 4; i++ {
			cx := 12 + float64(i)*22
			els = append(els,
				shape(fmt.Sprintf("tick%d", i), rect(cx, 42.6, 0.3, 3.5), &model.ShapeData{
					ShapeType: model.ShapeRectangle, Fill: solid(p.Primary, 1)}),
				shape(fmt.Sprintf("callout%d", i), rect(cx-3, 28, 6, 6), &model.ShapeData{ // numbered callout
					ShapeType: model.ShapeEllipse, Fill: solid(p.Background, 1), Line: outline(p.Primary, 1)}),
				text(fmt.Sprintf("callout-n%d", i), fmt.Sprint(i+1), rect(cx-3, 28, 6, 6), model.TextStyle{
					FontSize: 11, Bold: true, Color: p.Primary, Align: "center"}),
				text(fmt.Sprintf("step%d", i), []string{"MEASURE", "DRAFT", "REVIEW", "ISSUE"}[i], rect(cx-8, 50, 16, 3), model.TextStyle{
					FontSize: 10, Color: p.Text, Align: "center", LetterSpacing: 1.5}),
			)
		}
		els = append(els, text("body", "Thin lines, dashed guides, uppercase labels — the page reads like a drawing sheet.", rect(8, 62, 70, 6), model.TextStyle{
			FontSize: s.BodySize, Color: p.Text, Opacity: 0.7}))
	}
	return &model.Slide{
		ID:         "content",
		Background: &model.Background{Type: model.BgSolid, Color: p.Background},
		Elements:   els,
	}
}
