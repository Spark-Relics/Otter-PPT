// Package svgdecode compiles slide-authored SVG markup into native otter-ppt
// model elements. This mirrors the svg_to_pptx compilation idea from
// ppt-master: the AI authors vector primitives per slide, and the compiler
// turns them into editable native PPTX shapes instead of a bitmap.
package svgdecode

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// svgNode is a generic decoded SVG element node.
type svgNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []svgNode  `xml:",any"`
	Text     string     `xml:",chardata"`
}

func (n *svgNode) attr(name string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func (n *svgNode) id() string { return n.attr("id") }

// Canvas is the parsed coordinate system of one SVG page.
type Canvas struct {
	ViewW, ViewH float64 // viewBox dimensions
}

// Options tweak compilation.
type Options struct {
	// SlideW/SlideH are the target presentation size in inches
	// (defaults to 13.333 x 7.5).
	SlideW, SlideH float64
	// IDPrefix is prepended to generated element IDs.
	IDPrefix string
}

// Result summarizes one compilation run.
type Result struct {
	Elements []*model.Element
	Skipped  []string // reasons for skipped nodes
	Canvas   Canvas
}

// Compile parses an SVG document and emits model elements positioned in
// percentage coordinates (0-100) relative to the slide.
func Compile(svgText string, opts Options) (*Result, error) {
	if opts.SlideW == 0 {
		opts.SlideW, opts.SlideH = model.DefaultSlideSize()
	}
	var root svgNode
	if err := xml.Unmarshal([]byte(svgText), &root); err != nil {
		return nil, fmt.Errorf("parse SVG: %w", err)
	}
	if root.XMLName.Local != "svg" {
		return nil, fmt.Errorf("root element is %q, want svg", root.XMLName.Local)
	}

	canvas, err := parseCanvas(&root)
	if err != nil {
		return nil, err
	}

	c := &compiler{opts: opts, canvas: canvas, res: &Result{Canvas: canvas}}
	c.walk(&root, identityTransform())
	if len(c.res.Elements) == 0 {
		return nil, fmt.Errorf("no compilable elements found (skipped: %s)", strings.Join(c.res.Skipped, "; "))
	}
	return c.res, nil
}

func parseCanvas(root *svgNode) (Canvas, error) {
	vb := root.attr("viewBox")
	nums := ParseNumbers(vb)
	if len(nums) == 4 && nums[2] > 0 && nums[3] > 0 {
		return Canvas{ViewW: nums[2], ViewH: nums[3]}, nil
	}
	// Fall back to width/height without units.
	w, _ := strconv.ParseFloat(strings.TrimRight(root.attr("width"), "px"), 64)
	h, _ := strconv.ParseFloat(strings.TrimRight(root.attr("height"), "px"), 64)
	if w > 0 && h > 0 {
		return Canvas{ViewW: w, ViewH: h}, nil
	}
	return Canvas{}, fmt.Errorf("SVG needs a viewBox or width/height")
}

// transform is an accumulated 2D affine matrix [a c e; b d f; 0 0 1].
type transform struct{ a, b, c, d, e, f float64 }

func identityTransform() transform {
	return transform{a: 1, d: 1}
}

func (t transform) apply(x, y float64) (float64, float64) {
	return t.a*x + t.c*y + t.e, t.b*x + t.d*y + t.f
}

func (t transform) mul(o transform) transform {
	return transform{
		a: t.a*o.a + t.c*o.b, b: t.b*o.a + t.d*o.b,
		c: t.a*o.c + t.c*o.d, d: t.b*o.c + t.d*o.d,
		e: t.a*o.e + t.c*o.f + t.e, f: t.b*o.e + t.d*o.f + t.f,
	}
}

func parseTransform(s string) (transform, bool) {
	t := identityTransform()
	if strings.TrimSpace(s) == "" {
		return t, false
	}
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == '(' || r == ')' || r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	i := 0
	for i < len(fields)-1 {
		name := strings.TrimSpace(fields[i])
		switch name {
		case "translate":
			tx, _ := strconv.ParseFloat(fields[i+1], 64)
			ty := 0.0
			if i+2 < len(fields) && isNum(fields[i+2]) {
				ty, _ = strconv.ParseFloat(fields[i+2], 64)
				i++
			}
			t = t.mul(transform{a: 1, d: 1, e: tx, f: ty})
			i += 2
		case "scale":
			sx, _ := strconv.ParseFloat(fields[i+1], 64)
			sy := sx
			if i+2 < len(fields) && isNum(fields[i+2]) {
				sy, _ = strconv.ParseFloat(fields[i+2], 64)
				i++
			}
			t = t.mul(transform{a: sx, d: sy})
			i += 2
		case "rotate":
			ang, _ := strconv.ParseFloat(fields[i+1], 64)
			r := ang * math.Pi / 180
			cos, sin := math.Cos(r), math.Sin(r)
			rot := transform{a: cos, b: sin, c: -sin, d: cos}
			// Optional rotation center.
			if i+3 < len(fields) && isNum(fields[i+2]) && isNum(fields[i+3]) {
				cx, _ := strconv.ParseFloat(fields[i+2], 64)
				cy, _ := strconv.ParseFloat(fields[i+3], 64)
				t = t.mul(transform{a: 1, d: 1, e: cx, f: cy}).
					mul(rot).
					mul(transform{a: 1, d: 1, e: -cx, f: -cy})
				i += 2
			} else {
				t = t.mul(rot)
			}
			i += 2
		case "matrix":
			if i+6 > len(fields) {
				return t, false
			}
			var v [6]float64
			for j := 0; j < 6; j++ {
				v[j], _ = strconv.ParseFloat(fields[i+1+j], 64)
			}
			t = t.mul(transform{a: v[0], b: v[1], c: v[2], d: v[3], e: v[4], f: v[5]})
			i += 7
		default:
			return t, false // unsupported transform function
		}
	}
	return t, true
}

func isNum(s string) bool {
	_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return err == nil
}

type compiler struct {
	opts   Options
	canvas Canvas
	res    *Result
	seq    int
}

func (c *compiler) skip(reason string) {
	if len(c.res.Skipped) < 32 {
		c.res.Skipped = append(c.res.Skipped, reason)
	}
}

// pct converts viewBox units to percentage coordinates on the slide.
func (c *compiler) pct(v, total float64) float64 {
	if total == 0 {
		return 0
	}
	return v / total * 100
}

func (c *compiler) nextID(kind string) string {
	c.seq++
	return fmt.Sprintf("%ssvg_%s_%d", c.opts.IDPrefix, kind, c.seq)
}

func (c *compiler) walk(n *svgNode, parent transform) {
	tr, ok := parseTransform(n.attr("transform"))
	if !ok && n.attr("transform") != "" {
		c.skip(fmt.Sprintf("<%s id=%s unsupported transform>", n.XMLName.Local, n.id()))
		return
	}
	tr = parent.mul(tr)

	switch n.XMLName.Local {
	case "svg", "g", "a":
		for i := range n.Children {
			c.walk(&n.Children[i], tr)
		}
	case "rect":
		c.compileRect(n, tr)
	case "circle":
		c.compileCircle(n, tr)
	case "ellipse":
		c.compileEllipse(n, tr)
	case "line":
		c.compileLine(n, tr)
	case "path":
		c.compilePath(n, tr)
	case "text":
		c.compileText(n, tr)
	case "image":
		c.compileImage(n, tr)
	default:
		// defs, use, style, linearGradient... — not compiled.
		c.skip(fmt.Sprintf("<%s>", n.XMLName.Local))
	}
}

// ─────────── primitives ───────────

func (c *compiler) compileRect(n *svgNode, tr transform) {
	x, _ := strconv.ParseFloat(n.attr("x"), 64)
	y, _ := strconv.ParseFloat(n.attr("y"), 64)
	w, _ := strconv.ParseFloat(n.attr("width"), 64)
	h, _ := strconv.ParseFloat(n.attr("height"), 64)
	if w <= 0 || h <= 0 {
		return
	}
	x1, y1 := tr.apply(x, y)
	x2, y2 := tr.apply(x+w, y+h)
	bx, by, bw, bh := normRect(x1, y1, x2, y2)
	if tr != identityTransform() && skew(tr) {
		// Rotated rect falls back to freeform.
		c.emitFreeformFromPoly(n, tr, rectPoly(x, y, w, h))
		return
	}
	rx, _ := strconv.ParseFloat(n.attr("rx"), 64)
	elem := &model.Element{
		ID:   c.idFor(n, "rect"),
		Type: model.ElementShape,
		Rect: model.Rect{
			X: c.pct(bx, c.canvas.ViewW), Y: c.pct(by, c.canvas.ViewH),
			W: c.pct(bw, c.canvas.ViewW), H: c.pct(bh, c.canvas.ViewH),
		},
		Shape: &model.ShapeData{
			ShapeType: model.ShapeRectangle,
			Fill:      c.fill(n),
			Line:      c.line(n),
		},
	}
	if rx > 0 && rx <= w/2 && rx <= h/2 {
		elem.Shape.ShapeType = model.ShapeRoundedRectangle
		elem.Shape.CornerRadius = clamp01(rx / math.Min(w, h))
	}
	c.add(elem)
}

func (c *compiler) compileCircle(n *svgNode, tr transform) {
	cx, _ := strconv.ParseFloat(n.attr("cx"), 64)
	cy, _ := strconv.ParseFloat(n.attr("cy"), 64)
	r, _ := strconv.ParseFloat(n.attr("r"), 64)
	if r <= 0 {
		return
	}
	x1, y1 := tr.apply(cx-r, cy-r)
	x2, y2 := tr.apply(cx+r, cy+r)
	if skew(tr) || math.Abs((x2-x1)-(y2-y1)) > math.Abs(x2-x1)*0.01 {
		c.emitFreeformFromPoly(n, tr, EllipseContour(cx, cy, r, r))
		return
	}
	bx, by, bw, bh := normRect(x1, y1, x2, y2)
	c.add(&model.Element{
		ID:   c.idFor(n, "circle"),
		Type: model.ElementShape,
		Rect: model.Rect{
			X: c.pct(bx, c.canvas.ViewW), Y: c.pct(by, c.canvas.ViewH),
			W: c.pct(bw, c.canvas.ViewW), H: c.pct(bh, c.canvas.ViewH),
		},
		Shape: &model.ShapeData{
			ShapeType: model.ShapeEllipse,
			Fill:      c.fill(n),
			Line:      c.line(n),
		},
	})
}

func (c *compiler) compileEllipse(n *svgNode, tr transform) {
	cx, _ := strconv.ParseFloat(n.attr("cx"), 64)
	cy, _ := strconv.ParseFloat(n.attr("cy"), 64)
	rx, _ := strconv.ParseFloat(n.attr("rx"), 64)
	ry, _ := strconv.ParseFloat(n.attr("ry"), 64)
	if rx <= 0 || ry <= 0 {
		return
	}
	if skew(tr) {
		c.emitFreeformFromPoly(n, tr, EllipseContour(cx, cy, rx, ry))
		return
	}
	x1, y1 := tr.apply(cx-rx, cy-ry)
	x2, y2 := tr.apply(cx+rx, cy+ry)
	bx, by, bw, bh := normRect(x1, y1, x2, y2)
	c.add(&model.Element{
		ID:   c.idFor(n, "ellipse"),
		Type: model.ElementShape,
		Rect: model.Rect{
			X: c.pct(bx, c.canvas.ViewW), Y: c.pct(by, c.canvas.ViewH),
			W: c.pct(bw, c.canvas.ViewW), H: c.pct(bh, c.canvas.ViewH),
		},
		Shape: &model.ShapeData{
			ShapeType: model.ShapeEllipse,
			Fill:      c.fill(n),
			Line:      c.line(n),
		},
	})
}

func (c *compiler) compileLine(n *svgNode, tr transform) {
	x1, _ := strconv.ParseFloat(n.attr("x1"), 64)
	y1, _ := strconv.ParseFloat(n.attr("y1"), 64)
	x2, _ := strconv.ParseFloat(n.attr("x2"), 64)
	y2, _ := strconv.ParseFloat(n.attr("y2"), 64)
	ax1, ay1 := tr.apply(x1, y1)
	ax2, ay2 := tr.apply(x2, y2)
	line := c.line(n)
	if line == nil {
		line = &model.LineStyle{Width: 1}
	}
	if line.Color == "" {
		line.Color = "#000000"
	}
	c.add(&model.Element{
		ID:   c.idFor(n, "line"),
		Type: model.ElementConnector,
		Rect: model.Rect{W: 100, H: 100},
		Connector: &model.ConnectorData{
			ConnectorType: model.ShapeLine,
			Color:         line.Color,
			Width:         line.Width,
			StartX:        c.pct(ax1, c.canvas.ViewW),
			StartY:        c.pct(ay1, c.canvas.ViewH),
			EndX:          c.pct(ax2, c.canvas.ViewW),
			EndY:          c.pct(ay2, c.canvas.ViewH),
		},
	})
}

func (c *compiler) compilePath(n *svgNode, tr transform) {
	d := n.attr("d")
	pd, err := ParsePath(d)
	if err != nil {
		c.skip(fmt.Sprintf("path id=%s: %v", n.id(), err))
		return
	}
	if tr != identityTransform() {
		pd = transformPath(pd, tr)
	}

	// Line-only, unclosed, stroke-only path → connector.
	if isStrokedOnly(n) {
		if lines := asSinglePolyline(pd); len(lines) == 1 && len(lines[0]) == 2 {
			c.emitConnector(n, lines[0][0], lines[0][1])
			return
		}
	}

	polys := pd.Flatten()
	if len(polys) == 0 {
		c.skip(fmt.Sprintf("path id=%s: no renderable geometry", n.id()))
		return
	}
	// Single closed rect contour → rectangle shape.
	if len(polys) == 1 && pd.Subpaths[0].Closed && IsRectContour(polys[0]) {
		minX, minY, maxX, maxY := bounds(polys[0])
		c.add(&model.Element{
			ID:   c.idFor(n, "path"),
			Type: model.ElementShape,
			Rect: model.Rect{
				X: c.pct(minX, c.canvas.ViewW), Y: c.pct(minY, c.canvas.ViewH),
				W: c.pct(maxX-minX, c.canvas.ViewW), H: c.pct(maxY-minY, c.canvas.ViewH),
			},
			Shape: &model.ShapeData{
				ShapeType: model.ShapeRectangle,
				Fill:      c.fill(n),
				Line:      c.line(n),
			},
		})
		return
	}
	c.emitFreeform(n, polys, pd)
}

func (c *compiler) compileText(n *svgNode, tr transform) {
	if len(n.Children) > 0 && n.Children[0].XMLName.Local == "tspan" {
		c.skip(fmt.Sprintf("text id=%s: nested tspan unsupported", n.id()))
		return
	}
	text := strings.TrimSpace(n.attr("raw"))
	_ = text
	content := strings.TrimSpace(n.Text)
	if content == "" {
		return
	}
	x, _ := strconv.ParseFloat(n.attr("x"), 64)
	y, _ := strconv.ParseFloat(n.attr("y"), 64)
	fontSize, _ := strconv.ParseFloat(n.attr("font-size"), 64)
	if fontSize <= 0 {
		fontSize = 16
	}
	anchor := n.attr("text-anchor")
	// Approximate text box around the anchor point.
	estW := fontSize * 0.6 * float64(len([]rune(content)))
	estH := fontSize * 1.4
	var bx, by float64
	switch anchor {
	case "middle":
		bx = x - estW/2
	case "end":
		bx = x - estW
	default:
		bx = x
	}
	by = y - fontSize // baseline → top approx
	ax1, ay1 := tr.apply(bx, by)
	ax2, ay2 := tr.apply(bx+estW, by+estH)

	style := model.TextStyle{
		FontSize: int(math.Round(fontSize * ptPerUnit(c.canvas.ViewH, c.opts.SlideH))),
		Color:    colorOr(n.attr("fill"), "#000000"),
		FontName: n.attr("font-family"),
		VAlign:   "t",
	}
	switch anchor {
	case "middle":
		style.Align = "center"
	case "end":
		style.Align = "right"
	}
	if fw := n.attr("font-weight"); fw == "bold" || fw == "700" || fw == "800" || fw == "900" {
		style.Bold = true
	}
	if n.attr("font-style") == "italic" {
		style.Italic = true
	}
	if n.attr("text-decoration") == "underline" {
		style.Underline = true
	}

	c.add(&model.Element{
		ID:   c.idFor(n, "text"),
		Type: model.ElementBody,
		Rect: model.Rect{
			X: c.pct(ax1, c.canvas.ViewW), Y: c.pct(ay1, c.canvas.ViewH),
			W: c.pct(ax2-ax1, c.canvas.ViewW), H: c.pct(ay2-ay1, c.canvas.ViewH),
		},
		Text:  content,
		Style: style,
	})
}

func (c *compiler) compileImage(n *svgNode, tr transform) {
	href := n.attr("href")
	if href == "" {
		href = n.attr("{http://www.w3.org/1999/xlink}href")
	}
	if href == "" {
		href = n.attr("xlink:href")
	}
	if href == "" {
		c.skip("image without href")
		return
	}
	x, _ := strconv.ParseFloat(n.attr("x"), 64)
	y, _ := strconv.ParseFloat(n.attr("y"), 64)
	w, _ := strconv.ParseFloat(n.attr("width"), 64)
	h, _ := strconv.ParseFloat(n.attr("height"), 64)
	if w <= 0 || h <= 0 {
		c.skip("image without width/height")
		return
	}
	x1, y1 := tr.apply(x, y)
	x2, y2 := tr.apply(x+w, y+h)
	bx, by, bw, bh := normRect(x1, y1, x2, y2)

	elem := &model.Element{
		ID:   c.idFor(n, "image"),
		Type: model.ElementImage,
		Rect: model.Rect{
			X: c.pct(bx, c.canvas.ViewW), Y: c.pct(by, c.canvas.ViewH),
			W: c.pct(bw, c.canvas.ViewW), H: c.pct(bh, c.canvas.ViewH),
		},
		ImagePath: href,
		ImageFit:  "contain",
	}
	c.add(elem)
}

// ─────────── freeform emission ───────────

func (c *compiler) emitFreeform(n *svgNode, polys [][]Point, pd *PathData) {
	var minX, minY, maxX, maxY float64
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range polys {
		a, b, e, f := bounds(p)
		minX, minY = math.Min(minX, a), math.Min(minY, b)
		maxX, maxY = math.Max(maxX, e), math.Max(maxY, f)
	}
	w, h := maxX-minX, maxY-minY
	if w <= 0 || h <= 0 {
		c.skip(fmt.Sprintf("path id=%s: degenerate bounds", n.id()))
		return
	}

	contours := make([][]model.Vec2, len(polys))
	for i, poly := range polys {
		pts := make([]model.Vec2, len(poly))
		for j, p := range poly {
			pts[j] = model.Vec2{X: (p.X - minX) / w, Y: (p.Y - minY) / h}
		}
		contours[i] = pts
	}
	closed := make([]bool, len(pd.Subpaths))
	for i, sp := range pd.Subpaths {
		closed[i] = sp.Closed
	}

	c.add(&model.Element{
		ID:   c.idFor(n, "freeform"),
		Type: model.ElementShape,
		Rect: model.Rect{
			X: c.pct(minX, c.canvas.ViewW), Y: c.pct(minY, c.canvas.ViewH),
			W: c.pct(w, c.canvas.ViewW), H: c.pct(h, c.canvas.ViewH),
		},
		Shape: &model.ShapeData{
			ShapeType: model.ShapeFreeform,
			Fill:      c.fill(n),
			Line:      c.line(n),
			Freeform:  &model.FreeformData{Contours: contours, Closed: closed},
		},
	})
}

func (c *compiler) emitFreeformFromPoly(n *svgNode, tr transform, poly []Point) {
	pd := &PathData{}
	for _, p := range poly {
		pd.expand(p)
	}
	tPoly := make([]Point, len(poly))
	for i, p := range poly {
		x, y := tr.apply(p.X, p.Y)
		tPoly[i] = Point{X: x, Y: y}
	}
	pd.MinX, pd.MinY, pd.MaxX, pd.MaxY = bounds(tPoly)
	sp := Subpath{Closed: true, Start: tPoly[0]}
	for _, p := range tPoly[1:] {
		sp.Nodes = append(sp.Nodes, PathNode{Cmd: 'L', Pt: p})
	}
	pd.Subpaths = []Subpath{sp}
	c.emitFreeform(n, [][]Point{tPoly}, pd)
}

func (c *compiler) emitConnector(n *svgNode, from, to Point) {
	line := c.line(n)
	if line == nil {
		line = &model.LineStyle{Width: 1}
	}
	if line.Color == "" {
		line.Color = "#000000"
	}
	c.add(&model.Element{
		ID:   c.idFor(n, "conn"),
		Type: model.ElementConnector,
		Rect: model.Rect{W: 100, H: 100},
		Connector: &model.ConnectorData{
			ConnectorType: model.ShapeLine,
			Color:         line.Color,
			Width:         line.Width,
			StartX:        c.pct(from.X, c.canvas.ViewW),
			StartY:        c.pct(from.Y, c.canvas.ViewH),
			EndX:          c.pct(to.X, c.canvas.ViewW),
			EndY:          c.pct(to.Y, c.canvas.ViewH),
		},
	})
}

func (c *compiler) add(elem *model.Element) {
	c.res.Elements = append(c.res.Elements, elem)
}

func (c *compiler) idFor(n *svgNode, kind string) string {
	if id := n.id(); id != "" {
		return c.opts.IDPrefix + "svg_" + id
	}
	return c.nextID(kind)
}

// ─────────── paint helpers ───────────

func (c *compiler) fill(n *svgNode) *model.FillStyle {
	fill := strings.TrimSpace(n.attr("fill"))
	switch {
	case fill == "" || fill == "none":
		return nil
	case strings.HasPrefix(fill, "url("):
		c.skip(fmt.Sprintf("<%s id=%s gradient fill approximated with solid>", n.XMLName.Local, n.id()))
		return &model.FillStyle{Color: "#808080"}
	case strings.HasPrefix(fill, "rgb("):
		return &model.FillStyle{Color: rgbToHex(fill)}
	case strings.HasPrefix(fill, "#"):
		return &model.FillStyle{Color: normalizeHex(fill)}
	default:
		return &model.FillStyle{Color: namedColor(fill)}
	}
}

func (c *compiler) line(n *svgNode) *model.LineStyle {
	stroke := strings.TrimSpace(n.attr("stroke"))
	if stroke == "" || stroke == "none" {
		return nil
	}
	w, _ := strconv.ParseFloat(n.attr("stroke-width"), 64)
	if w <= 0 {
		w = 1
	}
	ls := &model.LineStyle{Width: w}
	switch {
	case strings.HasPrefix(stroke, "rgb("):
		ls.Color = rgbToHex(stroke)
	case strings.HasPrefix(stroke, "#"):
		ls.Color = normalizeHex(stroke)
	default:
		ls.Color = namedColor(stroke)
	}
	switch n.attr("stroke-dasharray") {
	case "":
	case "none":
	default:
		ls.Dash = "dash"
	}
	return ls
}

// ─────────── misc helpers ───────────

func transformPath(pd *PathData, tr transform) *PathData {
	out := &PathData{Subpaths: make([]Subpath, len(pd.Subpaths))}
	applyPt := func(p Point) Point {
		x, y := tr.apply(p.X, p.Y)
		return Point{X: x, Y: y}
	}
	for i, sp := range pd.Subpaths {
		ns := Subpath{Closed: sp.Closed, Start: applyPt(sp.Start)}
		ns.Nodes = make([]PathNode, len(sp.Nodes))
		for j, node := range sp.Nodes {
			nn := node
			nn.Pt = applyPt(node.Pt)
			nn.C1 = applyPt(node.C1)
			nn.C2 = applyPt(node.C2)
			ns.Nodes[j] = nn
		}
		out.Subpaths[i] = ns
	}
	out.MinX, out.MinY, out.MaxX, out.MaxY = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, sp := range out.Subpaths {
		out.expand(sp.Start)
		for _, node := range sp.Nodes {
			out.expand(node.Pt)
			out.expand(node.C1)
			out.expand(node.C2)
		}
	}
	return out
}

func skew(t transform) bool {
	return math.Abs(t.b) > 1e-9 || math.Abs(t.c) > 1e-9 ||
		math.Abs(t.a-t.d) > 1e-9 || t.a < 0
}

func normRect(x1, y1, x2, y2 float64) (bx, by, bw, bh float64) {
	return math.Min(x1, x2), math.Min(y1, y2), math.Abs(x2 - x1), math.Abs(y2 - y1)
}

func rectPoly(x, y, w, h float64) []Point {
	return []Point{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}, {x, y}}
}

func isStrokedOnly(n *svgNode) bool {
	f := strings.TrimSpace(n.attr("fill"))
	return f == "" || f == "none"
}

// asSinglePolyline returns point lists if the path is purely straight-line
// segments without curves or arcs, grouped per subpath.
func asSinglePolyline(pd *PathData) [][]Point {
	var out [][]Point
	for _, sp := range pd.Subpaths {
		for _, node := range sp.Nodes {
			if node.Cmd != 'L' {
				return nil
			}
		}
		if sp.Closed {
			return nil
		}
		pts := make([]Point, 0, len(sp.Nodes)+1)
		pts = append(pts, sp.Start)
		for _, node := range sp.Nodes {
			pts = append(pts, node.Pt)
		}
		out = append(out, pts)
	}
	return out
}

func colorOr(c, def string) string {
	if c == "" || c == "none" {
		return def
	}
	return normalizeHex(c)
}

func normalizeHex(c string) string {
	c = strings.TrimSpace(c)
	if len(c) == 4 && strings.HasPrefix(c, "#") {
		return fmt.Sprintf("#%c%c%c%c%c%c", c[1], c[1], c[2], c[2], c[3], c[3])
	}
	return strings.ToUpper(c)
}

func rgbToHex(rgb string) string {
	nums := ParseNumbers(rgb)
	if len(nums) < 3 {
		return "#808080"
	}
	return fmt.Sprintf("#%02X%02X%02X", uint8(nums[0]), uint8(nums[1]), uint8(nums[2]))
}

func namedColor(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "black":
		return "#000000"
	case "white":
		return "#FFFFFF"
	case "red":
		return "#FF0000"
	case "green":
		return "#008000"
	case "blue":
		return "#0000FF"
	case "yellow":
		return "#FFFF00"
	case "orange":
		return "#FFA500"
	case "purple":
		return "#800080"
	case "gray", "grey":
		return "#808080"
	case "silver":
		return "#C0C0C0"
	case "navy":
		return "#000080"
	case "teal":
		return "#008080"
	default:
		return "#808080"
	}
}

// ptPerUnit scales SVG user units to typographic points given the slide
// height mapping (viewBox height → SlideH inches → 72pt/inch).
func ptPerUnit(viewH, slideH float64) float64 {
	if viewH <= 0 || slideH <= 0 {
		return 0.75 // assume 96dpi-ish
	}
	return slideH * 72 / viewH
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// DecodeDataURL extracts bytes from a data: URI (used by image elements).
func DecodeDataURL(uri string) ([]byte, string, error) {
	if !strings.HasPrefix(uri, "data:") {
		return nil, "", fmt.Errorf("not a data URI")
	}
	rest := uri[len("data:"):]
	semi := strings.Index(rest, ",")
	if semi < 0 {
		return nil, "", fmt.Errorf("malformed data URI")
	}
	meta := rest[:semi]
	payload := rest[semi+1:]
	mime := "application/octet-stream"
	if i := strings.Index(meta, ";"); i > 0 {
		mime = meta[:i]
	} else if meta != "base64" && meta != "" {
		mime = meta
	}
	if strings.HasSuffix(meta, ";base64") || meta == "base64" {
		data, err := base64.StdEncoding.DecodeString(payload)
		return data, mime, err
	}
	return []byte(payload), mime, nil
}
