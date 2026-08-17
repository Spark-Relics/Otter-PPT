// htmlchart.go renders ChartData to inline SVG for the HTML preview path.
// This is a faithful-but-simplified vector rendering covering all 13 chart
// types otter-ppt supports: geometry is exact (bars, lines, pie slices),
// decorative extras (3D perspective, trendlines, error bars) are
// approximated. Good enough for AI visual feedback loops.
package renderer

import (
	"fmt"
	"math"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

var chartPalette = []string{
	"#4472C4", "#ED7D31", "#A5A5A5", "#FFC000", "#5B9BD5",
	"#70AD47", "#264478", "#9E480E", "#636363", "#997300",
}

func seriesColor(cd *model.ChartData, i int) string {
	if i < len(cd.Series) && cd.Series[i].Color != "" {
		return cssColor(cd.Series[i].Color)
	}
	return chartPalette[i%len(chartPalette)]
}

// chartSVG renders the chart into an SVG string sized w×h (px).
func (g *htmlGenerator) chartSVG(cd *model.ChartData, w, h float64) string {
	w = math.Max(w, 40)
	h = math.Max(h, 40)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`<svg width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg" style="width:100%%;height:100%%;">`,
		w, h, w, h))

	// Title
	top := 6.0
	if cd.Title != "" {
		sb.WriteString(fmt.Sprintf(
			`<text x="%.0f" y="18" text-anchor="middle" font-size="13" font-weight="bold" fill="#333">%s</text>`,
			w/2, escapeXML(cd.Title)))
		top = 28
	}

	// Legend
	legendH := 0.0
	if cd.ShowLegend && len(cd.Series) > 1 {
		lx := 8.0
		for i, s := range cd.Series {
			label := s.Name
			if label == "" {
				label = fmt.Sprintf("Series %d", i+1)
			}
			sb.WriteString(fmt.Sprintf(
				`<rect x="%.0f" y="%.1f" width="9" height="9" fill="%s"/><text x="%.0f" y="%.1f" font-size="9" fill="#555">%s</text>`,
				lx, h-12, seriesColor(cd, i), lx+12, h-4, escapeXML(label)))
			lx += 18 + 5.2*float64(len([]rune(label)))
		}
		legendH = 16
	}

	plotX, plotY := 36.0, top
	plotW := w - plotX - 10
	plotH := h - top - 18 - legendH
	if plotW < 20 || plotH < 20 {
		sb.WriteString("</svg>")
		return sb.String()
	}

	ct := effectiveChartType(cd)
	switch ct {
	case model.ChartPie, model.ChartPie3D, model.ChartDoughnut:
		g.pieSVG(&sb, cd, w/2, plotY+plotH/2, math.Min(plotW, plotH)/2, ct == model.ChartDoughnut)
	case model.ChartScatter:
		g.scatterSVG(&sb, cd, plotX, plotY, plotW, plotH)
	default:
		g.axisChartSVG(&sb, cd, ct, plotX, plotY, plotW, plotH)
	}

	sb.WriteString("</svg>")
	return sb.String()
}

func effectiveChartType(cd *model.ChartData) model.ChartType {
	switch cd.ChartType {
	case model.ChartBar, model.ChartBar3D:
		return model.ChartBar
	case model.ChartColumn, model.ChartColumn3D:
		return model.ChartColumn
	case model.ChartLine, model.ChartLine3D:
		return model.ChartLine
	case model.ChartArea, model.ChartArea3D:
		return model.ChartArea
	case model.ChartPie, model.ChartPie3D:
		return cd.ChartType
	case model.ChartDoughnut:
		return model.ChartDoughnut
	case model.ChartCombo:
		return model.ChartCombo
	default:
		return cd.ChartType
	}
}

// axisChartSVG renders bar/column/line/area/combo with axes and gridlines.
func (g *htmlGenerator) axisChartSVG(sb *strings.Builder, cd *model.ChartData, ct model.ChartType,
	px, py, pw, ph float64) {

	// Compute value range across all series.
	minV, maxV := 0.0, 0.0
	first := true
	for _, s := range cd.Series {
		for _, v := range s.Values {
			if first {
				minV, maxV = v, v
				first = false
			} else {
				minV = math.Min(minV, v)
				maxV = math.Max(maxV, v)
			}
		}
	}
	if first || minV == maxV {
		if minV == maxV {
			maxV = minV + 1
			if minV > 0 {
				minV = 0
			}
		}
	}
	if minV > 0 {
		minV = 0
	}
	if maxV < 0 {
		maxV = 0
	}
	pad := (maxV - minV) * 0.05
	maxV += pad
	if minV < 0 {
		minV -= pad
	}

	yOf := func(v float64) float64 {
		return py + ph - (v-minV)/(maxV-minV)*ph
	}

	// Gridlines + Y axis labels (5 ticks)
	sb.WriteString(`<g stroke="#e8e8e8" stroke-width="0.5">`)
	for i := 0; i <= 4; i++ {
		v := minV + (maxV-minV)*float64(i)/4
		y := yOf(v)
		sb.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, px, y, px+pw, y))
	}
	sb.WriteString(`</g>`)
	for i := 0; i <= 4; i++ {
		v := minV + (maxV-minV)*float64(i)/4
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="8" fill="#888">%s</text>`,
			px-3, yOf(v)+3, formatNum(v)))
	}

	// Axes
	sb.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#999" stroke-width="0.75"/>`+
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#999" stroke-width="0.75"/>`,
		px, py, px, py+ph, px, py+ph, px+pw, py+ph))

	n := len(cd.Categories)
	if n == 0 {
		n = 1
	}

	// Category labels
	catStep := pw / float64(n)
	for i, c := range cd.Categories {
		x := px + catStep*float64(i) + catStep/2
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="8" fill="#666">%s</text>`,
			x, py+ph+11, escapeXML(truncateLabel(c))))
	}

	switch ct {
	case model.ChartColumn, model.ChartCombo:
		// Columns (bar-series only in combo)
		groupW := catStep * 0.72
		barSeries := 0
		for _, s := range cd.Series {
			if ct != model.ChartCombo || s.ChartType == "" || s.ChartType == model.ChartBar || s.ChartType == model.ChartColumn {
				barSeries++
			}
		}
		if barSeries == 0 {
			barSeries = 1
		}
		barW := groupW / float64(barSeries)
		bi := 0
		for si, s := range cd.Series {
			isBar := ct != model.ChartCombo || s.ChartType == "" || s.ChartType == model.ChartBar || s.ChartType == model.ChartColumn
			if !isBar {
				continue
			}
			for ci, v := range s.Values {
				if ci >= n {
					break
				}
				x := px + catStep*float64(ci) + (catStep-groupW)/2 + barW*float64(bi)
				y := yOf(math.Max(v, 0))
				bh := math.Abs(yOf(v) - yOf(0))
				if bh < 0.5 && v != 0 {
					bh = 0.5
				}
				sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
					x, y, barW*0.92, bh, seriesColor(cd, si)))
				if cd.ShowDataLabels {
					sb.WriteString(fmt.Sprintf(
						`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="7" fill="#555">%s</text>`,
						x+barW*0.46, y-2, formatNum(v)))
				}
			}
			bi++
		}
		if ct == model.ChartCombo {
			g.lineSeriesSVG(sb, cd, px, py, catStep, yOf, true, model.ChartLine)
		}
	case model.ChartBar:
		// Horizontal bars
		rowH := ph / float64(n)
		seriesCount := float64(len(cd.Series))
		barH := rowH * 0.7 / seriesCount
		for si, s := range cd.Series {
			for ci, v := range s.Values {
				if ci >= n {
					break
				}
				y := py + rowH*float64(ci) + rowH*0.15 + barH*float64(si)
				xw := (v - minV) / (maxV - minV) * pw
				sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
					px, y, math.Max(xw, 0.5), barH*0.92, seriesColor(cd, si)))
			}
		}
	case model.ChartLine, model.ChartArea:
		if ct == model.ChartArea {
			// Filled area per series (drawn back-to-front)
			for si := len(cd.Series) - 1; si >= 0; si-- {
				s := cd.Series[si]
				var pts []string
				for ci, v := range s.Values {
					if ci >= n {
						break
					}
					x := px + catStep*float64(ci) + catStep/2
					pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, yOf(v)))
				}
				if len(pts) == 0 {
					continue
				}
				sb.WriteString(fmt.Sprintf(`<polygon points="%s %.1f,%.1f" fill="%s" opacity="0.35"/>`,
					strings.Join(pts, " "), px+catStep*float64(len(pts)-1)+catStep/2, yOf(0), seriesColor(cd, si)))
			}
		}
		g.lineSeriesSVG(sb, cd, px, py, catStep, yOf, false, ct)
	}
}

// lineSeriesSVG draws polylines with optional markers/data labels.
// onlyLine=true draws only series explicitly typed as line (combo mode).
func (g *htmlGenerator) lineSeriesSVG(sb *strings.Builder, cd *model.ChartData,
	px, py, catStep float64, yOf func(float64) float64, onlyLine bool, ct model.ChartType) {

	n := len(cd.Categories)
	for si, s := range cd.Series {
		if onlyLine {
			if s.ChartType != model.ChartLine {
				continue
			}
		}
		var pts []string
		for ci, v := range s.Values {
			if ci >= n {
				break
			}
			x := px + catStep*float64(ci) + catStep/2
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, yOf(v)))
		}
		if len(pts) < 1 {
			continue
		}
		sb.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="%s" stroke-width="1.6"/>`,
			strings.Join(pts, " "), seriesColor(cd, si)))
		for ci := range s.Values {
			if ci >= n {
				break
			}
			x := px + catStep*float64(ci) + catStep/2
			y := yOf(s.Values[ci])
			sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="2.2" fill="%s"/>`, x, y, seriesColor(cd, si)))
			if cd.ShowDataLabels {
				sb.WriteString(fmt.Sprintf(
					`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="7" fill="#555">%s</text>`,
					x, y-5, formatNum(s.Values[ci])))
			}
		}
	}
}

// pieSVG renders pie and doughnut charts with labels.
func (g *htmlGenerator) pieSVG(sb *strings.Builder, cd *model.ChartData,
	cx, cy, r float64, doughnut bool) {

	// Pie uses the first series; categories are labels.
	var s *model.ChartSeries
	if len(cd.Series) > 0 {
		s = &cd.Series[0]
	}
	if s == nil || len(s.Values) == 0 {
		return
	}

	total := 0.0
	for _, v := range s.Values {
		total += math.Abs(v)
	}
	if total <= 0 {
		return
	}

	start := -math.Pi / 2 // 12 o'clock
	for i, v := range s.Values {
		frac := math.Abs(v) / total
		angle := frac * 2 * math.Pi
		x1 := cx + r*math.Cos(start)
		y1 := cy + r*math.Sin(start)
		x2 := cx + r*math.Cos(start+angle)
		y2 := cy + r*math.Sin(start+angle)
		large := 0
		if angle > math.Pi {
			large = 1
		}
		color := chartPalette[i%len(chartPalette)]
		if s.Color != "" && len(cd.Series) == 1 && len(cd.Categories) <= 1 {
			color = cssColor(s.Color)
		}

		if doughnut {
			ir := r * 0.55
			ix1 := cx + ir*math.Cos(start+angle)
			iy1 := cy + ir*math.Sin(start+angle)
			ix2 := cx + ir*math.Cos(start)
			iy2 := cy + ir*math.Sin(start)
			sb.WriteString(fmt.Sprintf(
				`<path d="M %.1f %.1f A %.1f %.1f 0 %d 1 %.1f %.1f L %.1f %.1f A %.1f %.1f 0 %d 0 %.1f %.1f Z" fill="%s" stroke="#fff" stroke-width="0.8"/>`,
				x1, y1, r, r, large, x2, y2, ix1, iy1, ir, ir, large, ix2, iy2, color))
		} else {
			sb.WriteString(fmt.Sprintf(
				`<path d="M %.1f %.1f A %.1f %.1f 0 %d 1 %.1f %.1f L %.1f %.1f Z" fill="%s" stroke="#fff" stroke-width="0.8"/>`,
				x1, y1, r, r, large, x2, y2, cx, cy, color))
		}

		// Category label outside the slice
		if i < len(cd.Categories) {
			mid := start + angle/2
			lr := r + 12
			lx := cx + lr*math.Cos(mid)
			ly := cy + lr*math.Sin(mid)
			anchor := "middle"
			if math.Cos(mid) > 0.3 {
				anchor = "start"
			} else if math.Cos(mid) < -0.3 {
				anchor = "end"
			}
			sb.WriteString(fmt.Sprintf(
				`<text x="%.1f" y="%.1f" text-anchor="%s" font-size="8" fill="#555">%s</text>`,
				lx, ly+3, anchor, escapeXML(truncateLabel(cd.Categories[i]))))
			if cd.ShowDataLabels {
				pct := frac * 100
				sb.WriteString(fmt.Sprintf(
					`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="8" font-weight="bold" fill="#333">%.0f%%</text>`,
					cx+(r*0.68)*math.Cos(mid), cy+(r*0.68)*math.Sin(mid), pct))
			}
		}
		start += angle
	}

	if doughnut {
		sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="#fff"/>`, cx, cy, r*0.53))
	}
}

// scatterSVG renders scatter plots with X/Y value axes.
func (g *htmlGenerator) scatterSVG(sb *strings.Builder, cd *model.ChartData,
	px, py, pw, ph float64) {

	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, s := range cd.Series {
		for i, yv := range s.Values {
			var xv float64
			if i < len(s.XValues) {
				xv = s.XValues[i]
			} else {
				xv = float64(i + 1)
			}
			minX, maxX = math.Min(minX, xv), math.Max(maxX, xv)
			minY, maxY = math.Min(minY, yv), math.Max(maxY, yv)
		}
	}
	if math.IsInf(minX, 1) {
		return
	}
	if minX == maxX {
		maxX = minX + 1
	}
	if minY == maxY {
		maxY = minY + 1
	}

	// Grid + ticks
	for i := 0; i <= 4; i++ {
		f := float64(i) / 4
		gy := py + f*ph
		gx := px + f*pw
		yLabel := formatNum(maxY - (maxY-minY)*f)
		xLabel := formatNum(minX + (maxX-minX)*f)
		sb.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#e8e8e8" stroke-width="0.5"/>`+
				`<text x="%.1f" y="%.1f" text-anchor="end" font-size="8" fill="#888">%s</text>`+
				`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="8" fill="#888">%s</text>`,
			px, gy, px+pw, gy,
			px-3, gy+3, yLabel,
			gx, py+ph+11, xLabel))
	}
	sb.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#999" stroke-width="0.75"/>`+
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#999" stroke-width="0.75"/>`,
		px, py, px, py+ph, px, py+ph, px+pw, py+ph))

	for si, s := range cd.Series {
		var pts []string
		for i, yv := range s.Values {
			var xv float64
			if i < len(s.XValues) {
				xv = s.XValues[i]
			} else {
				xv = float64(i + 1)
			}
			x := px + (xv-minX)/(maxX-minX)*pw
			y := py + ph - (yv-minY)/(maxY-minY)*ph
			sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="2.5" fill="%s"/>`, x, y, seriesColor(cd, si)))
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		// Smooth-ish connect line when enabled
		if len(pts) > 1 {
			sb.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="%s" stroke-width="1" opacity="0.5"/>`,
				strings.Join(pts, " "), seriesColor(cd, si)))
		}
	}
}

// ──────────── helpers ────────────

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}

func truncateLabel(s string) string {
	r := []rune(s)
	if len(r) > 10 {
		return string(r[:9]) + "…"
	}
	return s
}

func formatNum(v float64) string {
	av := math.Abs(v)
	switch {
	case av >= 1000000:
		return fmt.Sprintf("%.0fM", v/1000000)
	case av >= 1000:
		return fmt.Sprintf("%.1fk", v/1000)
	case av >= 10:
		return fmt.Sprintf("%.0f", v)
	case av >= 1:
		return fmt.Sprintf("%.1f", v)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}
