package svgdecode

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Point is an absolute 2D coordinate in SVG user units.
type Point struct{ X, Y float64 }

// Subpath is a sequence of nodes belonging to one M..Z contour.
type Subpath struct {
	Closed bool
	Start  Point // contour start point (move-to target)
	Nodes  []PathNode
}

// PathNode is one path command node with absolute coordinates.
type PathNode struct {
	Cmd byte // M L H V C S Q T A Z (uppercase, absolute)
	Pt  Point
	// Cubic / quadrilateral control points (absolute).
	C1, C2 Point
	Rx, Ry, Rot, LargeArc, Sweep float64
}

// PathData is the parsed form of an SVG `d` attribute.
type PathData struct {
	Subpaths []Subpath
	// Bounds of all nodes (including control points).
	MinX, MinY, MaxX, MaxY float64
}

// ParsePath parses an SVG path data string (M L H V C S Q T A Z, relative
// variants included) into absolute subpaths.
func ParsePath(d string) (*PathData, error) {
	p := &pathParser{s: d}
	return p.parse()
}

type pathParser struct {
	s          string
	pos        int
	cur        Point // current point
	start      Point // start of current subpath
	lastCmd    byte
	lastC2     Point  // last cubic control point (for S)
	lastQCtrl  Point  // last quadratic control point (for T)
	hasLastC2  bool
	hasLastQ   bool
	sub        *Subpath
	out        *PathData
}

func (p *pathParser) parse() (*PathData, error) {
	p.out = &PathData{}
	p.out.MinX, p.out.MinY = math.Inf(1), math.Inf(1)
	p.out.MaxX, p.out.MaxY = math.Inf(-1), math.Inf(-1)

	for {
		p.skipSpace()
		if p.pos >= len(p.s) {
			break
		}
		cmd := p.s[p.pos]
		switch cmd {
		case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v', 'C', 'c',
			'S', 's', 'Q', 'q', 'T', 't', 'A', 'a', 'Z', 'z':
			p.pos++
			p.lastCmd = cmd
		case ',', ' ', '\t', '\n', '\r':
			p.pos++
			continue
		default:
			// Implicit repeat of previous command (number follows).
			switch p.lastCmd {
			case 'M':
				cmd = 'L'
			case 'm':
				cmd = 'l'
			default:
				cmd = p.lastCmd
			}
			if cmd == 0 || cmd == 'Z' || cmd == 'z' {
				return nil, fmt.Errorf("unexpected character %q at offset %d", cmd, p.pos)
			}
		}

		rel := cmd >= 'a' && cmd <= 'z'
		switch cmd {
		case 'M', 'm':
			x, y, err := p.num2()
			if err != nil {
				return nil, err
			}
			pt := p.abs(rel, x, y)
			p.moveTo(pt)
			// Subsequent implicit pairs are lineTo.
			p.lastCmd = map[byte]byte{'M': 'L', 'm': 'l'}[cmd]
		case 'L', 'l':
			x, y, err := p.num2()
			if err != nil {
				return nil, err
			}
			p.node('L', p.abs(rel, x, y))
		case 'H', 'h':
			x, err := p.num1()
			if err != nil {
				return nil, err
			}
			if rel {
				x += p.cur.X
			}
			p.node('L', Point{X: x, Y: p.cur.Y})
		case 'V', 'v':
			y, err := p.num1()
			if err != nil {
				return nil, err
			}
			if rel {
				y += p.cur.Y
			}
			p.node('L', Point{X: p.cur.X, Y: y})
		case 'C', 'c':
			nums, err := p.numN(6)
			if err != nil {
				return nil, err
			}
			c1 := p.abs(rel, nums[0], nums[1])
			c2 := p.abs(rel, nums[2], nums[3])
			end := p.abs(rel, nums[4], nums[5])
			p.node('C', end)
			n := &p.sub.Nodes[len(p.sub.Nodes)-1]
			n.C1, n.C2 = c1, c2
			p.lastC2, p.hasLastC2 = c2, true
		case 'S', 's':
			nums, err := p.numN(4)
			if err != nil {
				return nil, err
			}
			c2 := p.abs(rel, nums[0], nums[1])
			end := p.abs(rel, nums[2], nums[3])
			var c1 Point
			if p.hasLastC2 {
				c1 = Point{X: 2*p.cur.X - p.lastC2.X, Y: 2*p.cur.Y - p.lastC2.Y}
			} else {
				c1 = p.cur
			}
			p.node('C', end)
			n := &p.sub.Nodes[len(p.sub.Nodes)-1]
			n.C1, n.C2 = c1, c2
			p.lastC2, p.hasLastC2 = c2, true
		case 'Q', 'q':
			nums, err := p.numN(4)
			if err != nil {
				return nil, err
			}
			c := p.abs(rel, nums[0], nums[1])
			end := p.abs(rel, nums[2], nums[3])
			p.node('Q', end)
			n := &p.sub.Nodes[len(p.sub.Nodes)-1]
			n.C1 = c
			p.lastQCtrl, p.hasLastQ = c, true
		case 'T', 't':
			x, y, err := p.num2()
			if err != nil {
				return nil, err
			}
			end := p.abs(rel, x, y)
			var c Point
			if p.hasLastQ {
				c = Point{X: 2*p.cur.X - p.lastQCtrl.X, Y: 2*p.cur.Y - p.lastQCtrl.Y}
			} else {
				c = p.cur
			}
			p.node('Q', end)
			n := &p.sub.Nodes[len(p.sub.Nodes)-1]
			n.C1 = c
			p.lastQCtrl, p.hasLastQ = c, true
		case 'A', 'a':
			nums, err := p.numN(7)
			if err != nil {
				return nil, err
			}
			end := p.abs(rel, nums[5], nums[6])
			p.node('A', end)
			n := &p.sub.Nodes[len(p.sub.Nodes)-1]
			n.Rx, n.Ry, n.Rot = nums[0], nums[1], nums[2]
			n.LargeArc, n.Sweep = nums[3], nums[4]
		case 'Z', 'z':
			if p.sub != nil {
				p.sub.Closed = true
				p.cur = p.start
			}
		default:
			return nil, fmt.Errorf("unsupported path command %q", cmd)
		}
	}
	if p.sub != nil && len(p.sub.Nodes) > 0 {
		p.out.Subpaths = append(p.out.Subpaths, *p.sub)
	}
	if len(p.out.Subpaths) == 0 {
		return nil, fmt.Errorf("empty path data")
	}
	return p.out, nil
}

func (p *pathParser) moveTo(pt Point) {
	if p.sub != nil && !p.sub.Closed && len(p.sub.Nodes) == 0 {
		// Degenerate empty subpath: reuse it.
		p.sub = nil
	}
	if p.sub != nil {
		p.out.Subpaths = append(p.out.Subpaths, *p.sub)
	}
	p.sub = &Subpath{Start: pt}
	p.cur = pt
	p.start = pt
	p.hasLastC2, p.hasLastQ = false, false
}

func (p *pathParser) node(cmd byte, pt Point) {
	if p.sub == nil {
		p.sub = &Subpath{Start: p.cur}
		p.start = p.cur
	}
	p.sub.Nodes = append(p.sub.Nodes, PathNode{Cmd: cmd, Pt: pt})
	p.cur = pt
	p.out.expand(pt)
}

func (d *PathData) expand(pt Point) {
	if pt.X < d.MinX {
		d.MinX = pt.X
	}
	if pt.Y < d.MinY {
		d.MinY = pt.Y
	}
	if pt.X > d.MaxX {
		d.MaxX = pt.X
	}
	if pt.Y > d.MaxY {
		d.MaxY = pt.Y
	}
}

func (p *pathParser) abs(rel bool, dx, dy float64) Point {
	if rel {
		return Point{X: p.cur.X + dx, Y: p.cur.Y + dy}
	}
	return Point{X: dx, Y: dy}
}

func (p *pathParser) skipSpace() {
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case ' ', '\t', '\n', '\r', ',':
			p.pos++
		default:
			return
		}
	}
}

func (p *pathParser) num1() (float64, error) {
	p.skipSpace()
	start := p.pos
	// Sign
	if p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
		p.pos++
	}
	digits := false
	for p.pos < len(p.s) && p.s[p.pos] >= '0' && p.s[p.pos] <= '9' {
		p.pos++
		digits = true
	}
	if p.pos < len(p.s) && p.s[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.s) && p.s[p.pos] >= '0' && p.s[p.pos] <= '9' {
			p.pos++
			digits = true
		}
	}
	if p.pos < len(p.s) && (p.s[p.pos] == 'e' || p.s[p.pos] == 'E') {
		save := p.pos
		p.pos++
		if p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
			p.pos++
		}
		expDigits := false
		for p.pos < len(p.s) && p.s[p.pos] >= '0' && p.s[p.pos] <= '9' {
			p.pos++
			expDigits = true
		}
		if !expDigits {
			p.pos = save
		}
	}
	if !digits {
		return 0, fmt.Errorf("invalid number at offset %d", start)
	}
	return strconv.ParseFloat(p.s[start:p.pos], 64)
}

func (p *pathParser) num2() (float64, float64, error) {
	x, err := p.num1()
	if err != nil {
		return 0, 0, err
	}
	y, err := p.num1()
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func (p *pathParser) numN(n int) ([]float64, error) {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		v, err := p.num1()
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// FlattenTolerance is the max deviation used when flattening curves.
const FlattenTolerance = 0.25

// Flatten converts all curve nodes into polylines with straight segments.
// Returns subpaths whose nodes are only 'L' nodes (plus leading virtual move).
func (pd *PathData) Flatten() [][]Point {
	var out [][]Point
	for _, sp := range pd.Subpaths {
		var pts []Point
		if len(sp.Nodes) == 0 {
			continue
		}
		prev := sp.Start
		pts = append(pts, prev)
		for _, n := range sp.Nodes {
			switch n.Cmd {
			case 'L':
				pts = append(pts, n.Pt)
			case 'C':
				pts = append(pts, flattenCubic(prev, n.C1, n.C2, n.Pt)...)
				pts = append(pts, n.Pt)
			case 'Q':
				pts = append(pts, flattenQuad(prev, n.C1, n.Pt)...)
				pts = append(pts, n.Pt)
			case 'A':
				pts = append(pts, flattenArc(prev, n)...)
				pts = append(pts, n.Pt)
			}
			prev = n.Pt
		}
		if sp.Closed {
			// Ensure closure by repeating the start point.
			if len(pts) > 0 {
				first := pts[0]
				last := pts[len(pts)-1]
				if math.Abs(first.X-last.X) > 1e-9 || math.Abs(first.Y-last.Y) > 1e-9 {
					pts = append(pts, first)
				}
			}
		}
		if len(pts) >= 2 {
			out = append(out, pts)
		}
	}
	return out
}

func flattenCubic(p0, p1, p2, p3 Point) []Point {
	n := cubicSegments(p0, p1, p2, p3)
	var out []Point
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		x := u*u*u*p0.X + 3*u*u*t*p1.X + 3*u*t*t*p2.X + t*t*t*p3.X
		y := u*u*u*p0.Y + 3*u*u*t*p1.Y + 3*u*t*t*p2.Y + t*t*t*p3.Y
		out = append(out, Point{X: x, Y: y})
	}
	return out
}

func cubicSegments(p0, p1, p2, p3 Point) int {
	// Estimate control polygon length.
	d := dist(p0, p1) + dist(p1, p2) + dist(p2, p3)
	n := int(math.Ceil(d / FlattenTolerance))
	if n < 4 {
		n = 4
	}
	if n > 256 {
		n = 256
	}
	return n
}

func flattenQuad(p0, p1, p2 Point) []Point {
	d := dist(p0, p1) + dist(p1, p2)
	n := int(math.Ceil(d / FlattenTolerance))
	if n < 4 {
		n = 4
	}
	if n > 256 {
		n = 256
	}
	var out []Point
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		x := u*u*p0.X + 2*u*t*p1.X + t*t*p2.X
		y := u*u*p0.Y + 2*u*t*p1.Y + t*t*p2.Y
		out = append(out, Point{X: x, Y: y})
	}
	return out
}

func flattenArc(p0 Point, n PathNode) []Point {
	// Endpoint-to-center parameterization (SVG spec F.6).
	rx, ry := math.Abs(n.Rx), math.Abs(n.Ry)
	if rx < 1e-9 || ry < 1e-9 || (p0.X == n.Pt.X && p0.Y == n.Pt.Y) {
		return nil
	}
	phi := n.Rot * math.Pi / 180
	cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)
	dx2 := (p0.X - n.Pt.X) / 2
	dy2 := (p0.Y - n.Pt.Y) / 2
	x1 := cosPhi*dx2 + sinPhi*dy2
	y1 := -sinPhi*dx2 + cosPhi*dy2

	// Scale radii up if too small.
	lambda := x1*x1/(rx*rx) + y1*y1/(ry*ry)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx *= s
		ry *= s
	}

	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	if num < 0 {
		num = 0
	}
	coef := math.Sqrt(num / den)
	if n.LargeArc == n.Sweep { // large-arc and sweep differ → negative
		coef = -coef
	}
	cx := coef * rx * y1 / ry
	cy := -coef * ry * x1 / rx
	centerX := cosPhi*cx - sinPhi*cy + (p0.X+n.Pt.X)/2
	centerY := sinPhi*cx + cosPhi*cy + (p0.Y+n.Pt.Y)/2

	theta1 := angleBetween(1, 0, (p0.X-centerX)/rx, (p0.Y-centerY)/ry)
	delta := angleBetween((p0.X-centerX)/rx, (p0.Y-centerY)/ry, (n.Pt.X-centerX)/rx, (n.Pt.Y-centerY)/ry)

	sweep := n.Sweep != 0
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	}
	if sweep && delta < 0 {
		delta += 2 * math.Pi
	}

	segs := int(math.Ceil(math.Abs(delta) / (FlattenTolerance / math.Max(rx, ry) * 1)))
	if segs < 4 {
		segs = 4
	}
	if segs > 256 {
		segs = 256
	}
	var out []Point
	for i := 1; i < segs; i++ {
		t := theta1 + delta*float64(i)/float64(segs)
		px := centerX + rx*math.Cos(t)*cosPhi - ry*math.Sin(t)*sinPhi
		py := centerY + rx*math.Cos(t)*sinPhi + ry*math.Sin(t)*cosPhi
		out = append(out, Point{X: px, Y: py})
	}
	return out
}

func angleBetween(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	lenU := math.Hypot(ux, uy)
	lenV := math.Hypot(vx, vy)
	if lenU == 0 || lenV == 0 {
		return 0
	}
	c := dot / (lenU * lenV)
	if c > 1 {
		c = 1
	}
	if c < -1 {
		c = -1
	}
	ang := math.Acos(c)
	if ux*vy-uy*vx < 0 {
		ang = -ang
	}
	return ang
}

func dist(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

// IsRectContour reports whether a flattened polygon is an axis-aligned
// rectangle (optionally with rounded corners detected by node count > 5).
func IsRectContour(pts []Point) bool {
	if len(pts) < 4 || len(pts) > 8 {
		return false
	}
	minX, minY, maxX, maxY := bounds(pts)
	tol := math.Max(maxX-minX, maxY-minY) * 0.02
	// All points must lie on one of the four rect edges.
	for _, p := range pts {
		onV := math.Abs(p.X-minX) < tol || math.Abs(p.X-maxX) < tol
		onH := math.Abs(p.Y-minY) < tol || math.Abs(p.Y-maxY) < tol
		if !onV && !onH {
			return false
		}
	}
	// Corner proximity: at least 4 points near corners.
	corners := [4]Point{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	near := 0
	for _, c := range corners {
		for _, p := range pts {
			if dist(p, c) < math.Max(tol*4, 4) {
				near++
				break
			}
		}
	}
	return near >= 3
}

func bounds(pts []Point) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return
}

// ellipseContour approximates a circle/ellipse as a closed contour
// (used to recognize circle elements).
func EllipseContour(cx, cy, rx, ry float64) []Point {
	const n = 16
	pts := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		t := 2 * math.Pi * float64(i) / n
		pts = append(pts, Point{X: cx + rx*math.Cos(t), Y: cy + ry*math.Sin(t)})
	}
	return pts
}

// ParseNumbers splits a numeric list attribute (points, viewBox...).
func ParseNumbers(s string) []float64 {
	var out []float64
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(cur.String()), 64); err == nil {
			out = append(out, v)
		}
		cur.Reset()
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '+' || r == 'e' || r == 'E' {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}
