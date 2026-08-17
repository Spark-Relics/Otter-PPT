package builder

import (
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// freeformGeometryXML builds an <a:custGeom> element from normalized
// contours. Each contour becomes one path block: moveTo → lineTo… → close.
// Coordinates are in the custGeom local coordinate space where the shape
// bounding box maps to 0..100000 (w/h percentages of shape extent).
// hasFill controls the per-path fill attribute ("none" disables fill).
func freeformGeometryXML(ff *model.FreeformData, hasFill bool) string {
	pathFill := ` fill="none"`
	if hasFill {
		pathFill = ""
	}
	var path strings.Builder
	path.WriteString(`<a:pathLst>`)
	for i, contour := range ff.Contours {
		if len(contour) < 2 {
			continue
		}
		first := contour[0]
		fmt.Fprintf(&path, `<a:path w="100000" h="100000"%s extrusionOk="0"><a:moveTo><a:pt x="%d" y="%d"/></a:moveTo>`,
			pathFill, coordToGeom(first.X), coordToGeom(first.Y))
		for _, pt := range contour[1:] {
			fmt.Fprintf(&path, `<a:lnTo><a:pt x="%d" y="%d"/></a:lnTo>`,
				coordToGeom(pt.X), coordToGeom(pt.Y))
		}
		if i < len(ff.Closed) && ff.Closed[i] {
			path.WriteString(`<a:close/>`)
		}
		path.WriteString(`</a:path>`)
	}
	path.WriteString(`</a:pathLst>`)

	// Bounding box covers the full normalized unit square.
	return `<a:custGeom><a:avLst/><a:gdLst>` +
		`<a:gd name="x1" fmla="*/ w 0 100000"/><a:gd name="y1" fmla="*/ h 0 100000"/>` +
		`<a:gd name="x2" fmla="*/ w 100000 100000"/><a:gd name="y2" fmla="*/ h 100000 100000"/>` +
		`</a:gdLst>` + path.String() + `</a:custGeom>`
}

// coordToGeom maps a normalized 0-1 coordinate to the custGeom integer space.
func coordToGeom(v float64) int {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return int(v*100000 + 0.5)
}
