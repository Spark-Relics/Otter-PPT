// renderings.go implements the AI-image rendering catalog (ppt-master
// image-renderings concept). A "rendering" is the deck-wide visual style
// family for AI-generated images — orthogonal to the slide style (shape
// language) and palette (colors). Every AI image in one deck shares one
// rendering; prompts assemble: rendering style paragraph + deck color
// anchors (semantic roles, never visible text) + per-image composition.
package design

import (
	"fmt"
	"sort"
	"strings"
)

// Rendering is one AI-image style preset.
type Rendering struct {
	Key    string
	Name   string
	Brief  string // selection rule for the planner
	// Paragraph is the paste-ready 80-120 word style paragraph embedded in
	// every image prompt for this rendering.
	Paragraph string
	// ColorBehavior explains how deck HEX anchors map into the image.
	ColorBehavior string
	// Fewshot is one ready-to-paste example prompt (hero/backdrop).
	Fewshot string
}

var renderings = map[string]*Rendering{
	"vector-illustration": {
		Key:   "vector-illustration",
		Name:  "Vector Illustration",
		Brief: "Default safe pick for general decks: clean flat vector art with bold shapes, confident negative space, scales across 15+ slides without fatigue.",
		Paragraph: "Clean flat vector illustration with bold geometric shapes and confident solid fills. Crisp outlines (1.5-2px equivalent stroke weight, consistent across all elements) define every form. No gradients within shapes — color is applied as flat blocks. Subtle shadow only where it adds depth (soft 8% opacity drop, no harsh edges). Composition is grid-aware and balanced, with deliberate negative space carrying as much weight as filled areas. Iconography is simplified to essential geometry — recognizable at small sizes. Overall feel is modern, professional, and confidently restrained.",
		ColorBehavior: "Colors are flat coded zones, not gradients: primary HEX fills main shapes, background HEX carries the field, accent HEX appears only as one or two emphasis elements. Apply HEX values exactly — no tinting, no shading. Outlines in dark neutral unless primary is dark enough.",
		Fewshot:      "Clean flat vector illustration backdrop. Bold geometric shapes in flat solid fills — primary deep navy forming a confident diagonal across the lower third, light gray occupying the upper two-thirds as calm breathing space, accent gold appearing only as one or two thin geometric lines drawing the eye toward the center. Crisp 2px outlines on all shapes. No gradients, no shadows beyond a single 8% soft drop. The central 70% of the canvas is deliberately calm, ready to receive a title overlay. NO text of any kind anywhere in the image — no letters, numbers, signs, watermarks, or written symbols.",
	},
	"flat": {
		Key:   "flat",
		Name:  "Flat Minimal",
		Brief: "Pick for swiss_minimal / mono_ink decks: even more reduced than vector-illustration — almost no outlines, pure silhouette shapes, maximum whitespace.",
		Paragraph: "Minimal flat design with pure silhouette shapes and no outlines at all. Each form is a single solid color block, reduced to its most iconic geometry. Extremely generous negative space — the composition breathes. At most 3-4 shapes per image, each carrying one clear job. No shadows, no gradients, no texture, no stroke. The feel is signage-grade clarity: instantly readable, poster-like, almost brutalist in its reduction.",
		ColorBehavior: "Two colors do almost everything: background HEX carries 70%+ of the canvas, primary HEX carries the main silhouette. Accent HEX may appear once, on a single small shape. Never more than three colors per image.",
		Fewshot:      "Minimal flat illustration. One bold primary-color silhouette occupying roughly a third of the canvas, positioned off-center with deliberate asymmetry, set against a calm flat background field. A single small accent-color shape balances the composition. No outlines, no gradients, no shadows, no texture. Poster-like reduction — every shape instantly readable. NO text of any kind anywhere in the image.",
	},
	"watercolor": {
		Key:   "watercolor",
		Name:  "Watercolor",
		Brief: "Pick for editorial / soft / humanities decks: soft washes, paper grain, gentle pigment bleeds — emotional and warm, not corporate.",
		Paragraph: "Delicate watercolor illustration with soft translucent washes layered over visible cold-press paper grain. Pigment bleeds naturally at shape edges — wet-on-wet blooms and gentle cauliflowers — with granulation where pigment settles into paper texture. Brush strokes remain visible and hand-held: loose confident strokes, not labored detail. Composition is airy and off-balance in a deliberate way, with white paper showing through as breathing space. The overall feel is quiet, human, editorial — like a quality picture book or a literary magazine spot illustration.",
		ColorBehavior: "Deck anchors are pigment starting points, not flat codes: primary appears as the richest wash, background stays warm paper white, accent is a thin saturated stroke. Colors may mix and dilute freely — this is the one rendering where derived tints are encouraged. Never let an image exceed 4 hue families.",
		Fewshot:      "Delicate watercolor illustration. Soft translucent washes in the deck's primary hue blooming gently at the edges, layered over visible cold-press paper grain, with generous white paper showing through as breathing space. One thin saturated accent stroke draws the eye. Loose confident hand-held brush strokes, natural pigment bleeds, no labored detail. Quiet, human, editorial feel. NO text of any kind anywhere in the image.",
	},
	"3d-isometric": {
		Key:   "3d-isometric",
		Name:  "3D Isometric",
		Brief: "Pick for tech / architecture / product / infrastructure decks: isometric dioramas with soft studio lighting, explaining systems as miniature scenes.",
		Paragraph: "Isometric 3D illustration rendered in soft studio lighting, using true isometric projection (30-degree axes, no perspective distortion). Objects are simplified rounded geometric forms with matte clay-like materials and subtle ambient occlusion where surfaces meet. A soft directional key light casts gentle shadows in one consistent direction. Depth comes from layered platforms and floating elements, not from dramatic perspective. The feel is a polished product diorama — like a premium SaaS explainer or a board-game-quality miniature scene.",
		ColorBehavior: "Primary HEX colors the main structures, background HEX or a slightly tinted variant forms the base plate, accent HEX marks the single most important element. Materials stay matte — no metallic, no glass unless the deck style explicitly says so. Lighting may warm or cool the anchors slightly but never replaces them.",
		Fewshot:      "Isometric 3D illustration in soft studio lighting, true isometric projection. Simplified rounded geometric forms in matte clay-like materials on a calm base plate, subtle ambient occlusion where surfaces meet, one soft key light casting gentle consistent shadows. Primary-colored main structures with one accent-colored focal element, layered platforms creating depth without perspective distortion. Polished product-diorama feel. NO text of any kind anywhere in the image.",
	},
	"corporate-photo": {
		Key:   "corporate-photo",
		Name:  "Corporate Photo",
		Brief: "Pick for business / consulting / case-study decks: photorealistic editorial business photography, shallow depth of field, natural light.",
		Paragraph: "Editorial corporate photography with a documentary eye. Natural available light — soft window light or overcast diffusion — never flash-flat. Shallow depth of field with the subject crisply separated from a softly blurred environment. Real people mid-action, unposed: collaborating, thinking, presenting — never grinning at the camera. Muted authentic color grade with restrained contrast; no HDR, no heavy filters. Composition follows the rule of thirds with clean negative space reserved for text overlay where needed. The feel is a quality business magazine photo essay, not stock photography.",
		ColorBehavior: "Color anchors guide wardrobe, environment, and props: primary tone appears in clothing or key objects, background stays neutral and desaturated, accent appears as at most one small saturated element. The grade keeps everything cohesive — muted, slightly cool or warm to match the deck mood — but never forces objects to impossible colors.",
		Fewshot:      "Editorial corporate photograph, natural window light with soft overcast diffusion. Real people collaborating mid-action around a table, unposed, shallow depth of field separating them from a softly blurred modern office. Muted authentic color grade, restrained contrast, rule-of-thirds composition with clean negative space on one side for text overlay. Business-magazine photo-essay feel, absolutely not stock photography. NO text of any kind anywhere in the image — no signs, no screens with readable words.",
	},
	"sketch-notes": {
		Key:   "sketch-notes",
		Name:  "Sketch Notes",
		Brief: "Pick for education / workshop / agile decks: hand-drawn whiteboard energy, marker strokes, arrows and containers — thinking made visible.",
		Paragraph: "Hand-drawn sketch-note illustration with visible marker strokes on a clean whiteboard surface. Confident single-weight line work, slightly imperfect circles and connectors — human wobble, not chaotic scribble. Ideas are organized into hand-drawn containers: rounded boxes, cloud bubbles, and curved arrows linking them. Limited spot color fills (roughly 25% coverage) with the rest left as line work. The feel is a live whiteboard captured mid-workshop — thinking made visible, energetic but legible.",
		ColorBehavior: "Line work in a dark neutral (near the deck's text color). Spot fills use the primary HEX loosely — slightly uneven, like marker on whiteboard — and the accent HEX highlights the single most important container or arrow. Background stays white/near-white even in dark decks (sketch notes live on their own white canvas).",
		Fewshot:      "Hand-drawn sketch-note illustration on a clean whiteboard surface. Confident marker line work with human wobble, ideas organized into hand-drawn rounded boxes and cloud bubbles connected by curved arrows flowing left to right. Loose primary-color spot fills on roughly a quarter of the containers, one accent-colored arrow marking the key path. Energetic workshop whiteboard feel, legible structure. NO text of any kind anywhere in the image — no labels, no letters, containers stay empty of words.",
	},
	"warm-scene": {
		Key:   "warm-scene",
		Name:  "Warm Scene",
		Brief: "Pick for storytelling / brand / lifestyle decks: cozy narrative scenes with golden-hour atmosphere and cinematic depth.",
		Paragraph: "Warm narrative scene illustration with golden-hour atmosphere and cinematic depth of field. Soft gradient light — amber through rose — bathing the scene, with a gentle haze that separates foreground silhouettes from an atmospheric background. Shapes are simplified but organic: no harsh geometry, gentle curves and soft edges. One clear focal subject, with supporting elements arranged to lead the eye toward it. The feel is a storybook film still or a premium brand film frame — emotional, inviting, quietly cinematic.",
		ColorBehavior: "The palette anchors become light: primary HEX shapes the key subject, accent HEX glows as the light source or its halo, and the background dissolves into warm derived tones of the deck background. Atmospheric blending is expected — this rendering treats colors as light and air, not as flat codes.",
		Fewshot:      "Warm narrative scene illustration with golden-hour atmosphere. Soft amber-through-rose gradient light bathing the scene, gentle haze separating a simplified organic foreground subject from an atmospheric background. One clear focal point with supporting elements leading the eye toward it, soft edges and gentle curves throughout. Cinematic storybook film-still feel, emotional and inviting. NO text of any kind anywhere in the image.",
	},
	"digital-dashboard": {
		Key:   "digital-dashboard",
		Name:  "Digital Dashboard",
		Brief: "Pick for dark_tech / data / AI decks: dark UI surfaces with glowing accents, futuristic HUD elements, high-tech precision.",
		Paragraph: "Dark futuristic interface illustration with glowing accent light on deep near-black surfaces. Thin luminous lines, holographic wireframes, and subtle grid textures define structure. Glows are soft bloom effects around emissive elements — neon without noise. Depth comes from layered translucent panels floating in dark space, slightly overlapping, each catching rim light. Precision is the aesthetic: everything aligns to a grid, curves are mathematically clean. The feel is a premium sci-fi HUD or a next-gen analytics dashboard rendered as art.",
		ColorBehavior: "Background stays near-black (derived from the deck's dark background). Primary HEX and accent HEX become the two glow colors — accent is the brightest, used on the single focal element only. Never more than two glow hues per image; everything else stays neutral dark grays. In a light-palette deck this rendering should not be chosen.",
		Fewshot:      "Dark futuristic interface illustration. Thin luminous lines and holographic wireframes on a deep near-black surface, soft bloom glows around emissive elements, layered translucent panels floating in dark space with subtle rim light. Precise grid-aligned structure, mathematically clean curves. Primary-hue glow on supporting elements, the accent hue reserved for the single brightest focal point. Premium sci-fi HUD feel. NO text of any kind anywhere in the image — no readable characters on any panel or display.",
	},
}

// GetRendering returns the rendering preset for a key, or nil.
func GetRendering(key string) *Rendering { return renderings[strings.ToLower(key)] }

// RenderingKeys lists available rendering keys in stable order.
func RenderingKeys() []string {
	keys := make([]string, 0, len(renderings))
	for k := range renderings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// RenderingCatalog renders the image-rendering selection rules for LLM prompts.
func RenderingCatalog() string {
	var sb strings.Builder
	for _, k := range RenderingKeys() {
		r := renderings[k]
		fmt.Fprintf(&sb, "- %s (%s): %s\n", r.Key, r.Name, r.Brief)
	}
	return sb.String()
}

// ImagePromptRules are the hard rules for assembling AI-image prompts —
// borrowed from ppt-master image-generator.md core principles.
func ImagePromptRules() string {
	return `AI IMAGE PROMPT RULES (hard):
1. ONE rendering per deck — every AI image in the deck shares the selected rendering's style paragraph; do not mix renderings.
2. A prompt is ONE coherent prose paragraph, NOT a tag soup ("cute puppy, fluffy, 4k, professional" is a failure mode — a model-output reality, not an aesthetic choice).
3. Assemble each prompt as: rendering style paragraph (adapted, not copied verbatim) + subject and composition in prose + deck color anchors used as rendering guidance.
4. HEX codes and color names are rendering guidance — they must NEVER appear as visible text in the image.
5. NO text of any kind in images by default ("NO text of any kind anywhere in the image — no letters, numbers, signs, watermarks, or written symbols" as the closing line). All editable text (titles, labels, body) stays as native PPT text elements — changing one in-image word costs an image regeneration, one native text element costs a keystroke.
6. Derive color behavior from the deck palette's semantic roles (background carries the field, primary carries main forms, accent stays scarce) — never invent an unrelated image-only palette.
7. Match the slide's rhythm: hero/anchor pages get calm compositions with quiet regions reserved for title overlay; content images stay supporting, never busy.`
}

// FormatImagePromptGuidance renders the full build-phase image section:
// rendering lock + color anchors + hard rules + one fewshot.
func FormatImagePromptGuidance(renderingKey, paletteKey string) string {
	r := GetRendering(renderingKey)
	p := GetPalette(paletteKey)
	if r == nil {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "🎨 IMAGE RENDERING LOCK: %s (%s)\n", r.Key, r.Name)
	fmt.Fprintf(&sb, "Style paragraph (adapt into EVERY image prompt):\n%s\n\n", r.Paragraph)
	fmt.Fprintf(&sb, "Color behavior: %s\n", r.ColorBehavior)
	if p != nil {
		fmt.Fprintf(&sb, "Deck color anchors: background=%s secondary-bg=%s primary=%s accent=%s secondary-accent=%s text=%s\n",
			p.Background, p.BackgroundSecondary, p.Primary, p.Accent, p.AccentSecondary, p.Text)
	}
	fmt.Fprintf(&sb, "Fewshot (reference only — do not copy the subject):\n%s\n\n", r.Fewshot)
	sb.WriteString(ImagePromptRules())
	return sb.String()
}
