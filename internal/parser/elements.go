package parser

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// parseShape converts a p:sp into a model.Element.
// A shape is classified as:
//   - title/subtitle/body text box: txBox=1 with rect geometry and no
//     explicit fill/line, classified further by cNvPr name heuristic
//   - bullet list: txBox with paragraphs carrying buChar
//   - shape: everything else (prstGeom), with optional text inside
func parseShape(sp xmlShape, slideW, slideH float64, warnings *[]string) *model.Element {
	rect, rot, ok := rectFromXfrm(sp.SpPr.Xfrm, slideW, slideH)
	if !ok {
		*warnings = append(*warnings, fmt.Sprintf("shape %q without geometry skipped", sp.Name()))
		return nil
	}

	paragraphs, fullText, style := parseBody(sp.Body)
	isBulletList := hasBullet(sp.Body)

	// Text-box family: txBox flag or plain rect with no fill/line/custGeom.
	isTextBox := sp.TxBox() || (sp.SpPr.PrstGeom != nil && sp.SpPr.PrstGeom.Prst == "rect" &&
		sp.SpPr.SolidFill == nil && sp.SpPr.GradFill == nil && sp.SpPr.Ln == nil && !sp.SpPr.CustGeomPresent)

	elem := &model.Element{
		Rect:     rect,
		Rotation: rot,
	}

	switch {
	case isTextBox && isBulletList:
		elem.Type = model.ElementBullet
		for _, p := range sp.Body {
			elem.Items = append(elem.Items, paragraphPlainText(p))
		}
		elem.Style = style
	case isTextBox:
		elem.Type = classifyTextBox(sp.Name())
		if len(sp.Body) > 1 {
			elem.Paragraphs = paragraphs
		} else {
			elem.Text = fullText
		}
		elem.Style = style
	default:
		elem.Type = model.ElementShape
		shape := &model.ShapeData{}
		if sp.SpPr.PrstGeom != nil {
			shape.ShapeType = prstToShapeType(sp.SpPr.PrstGeom.Prst)
		} else if sp.SpPr.CustGeomPresent {
			shape.ShapeType = model.ShapeFreeform
			*warnings = append(*warnings, fmt.Sprintf("freeform %q approximated as rectangle (custGeom import not yet supported)", sp.Name()))
		}
		if sp.SpPr.SolidFill != nil {
			shape.FillColor = srgbToHex(sp.SpPr.SolidFill.Color)
			if c := sp.SpPr.SolidFill.Color; c != nil && c.Alpha != nil {
				shape.Fill = &model.FillStyle{Color: shape.FillColor, Opacity: float64(c.Alpha.Val) / 100000}
			}
		}
		if sp.SpPr.GradFill != nil && len(sp.SpPr.GradFill.Stops) > 0 {
			g := &model.Gradient{Type: model.GradientLinear}
			if sp.SpPr.GradFill.Lin != nil {
				g.Angle = float64(sp.SpPr.GradFill.Lin.Ang) / 60000
			}
			for _, st := range sp.SpPr.GradFill.Stops {
				if st.Color != nil {
					g.Stops = append(g.Stops, model.GradientStop{Color: srgbToHex(st.Color), Position: float64(st.Pos) / 1000})
				}
			}
			shape.Fill = &model.FillStyle{Gradient: g}
		}
		if sp.SpPr.Ln != nil {
			line := &model.LineStyle{}
			if sp.SpPr.Ln.W > 0 {
				line.Width = float64(sp.SpPr.Ln.W) / 12700 // EMU → pt
			}
			if sp.SpPr.Ln.Solid != nil {
				line.Color = srgbToHex(sp.SpPr.Ln.Solid.Color)
			}
			if sp.SpPr.Ln.PrstDash != nil {
				line.Dash = dashFromXML(sp.SpPr.Ln.PrstDash.Val)
			}
			shape.Line = line
			if shape.BorderColor == "" {
				shape.BorderColor = line.Color
			}
		}
		if fullText != "" {
			shape.Text = fullText
			shape.Style = style
		}
		elem.Shape = shape
	}

	return elem
}

// classifyTextBox names a text element by cNvPr name heuristic. Builder
// writes the element ID as cNvPr name; agent-authored IDs often contain
// "title"/"subtitle".
func classifyTextBox(name string) model.ElementType {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "title"):
		return model.ElementTitle
	case strings.Contains(n, "subtitle"):
		return model.ElementSubtitle
	default:
		return model.ElementBody
	}
}

func dashFromXML(v string) string {
	switch v {
	case "dash", "lgDash", "sysDash":
		return "dash"
	case "dot", "sysDot":
		return "dot"
	case "dashDot", "lgDashDot":
		return "dash_dot"
	default:
		return "solid"
	}
}

// hasBullet reports whether any paragraph carries a buChar.
func hasBullet(paras []xmlParagraph) bool {
	for _, p := range paras {
		if p.PPr != nil && p.PPr.BuChar != nil {
			return true
		}
	}
	return false
}

func paragraphPlainText(p xmlParagraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// parseBody extracts paragraphs, flattened text and an aggregated style
// from a txBody. The aggregated style comes from the first styled run.
func parseBody(paras []xmlParagraph) ([]model.Paragraph, string, model.TextStyle) {
	var out []model.Paragraph
	var full strings.Builder
	var style model.TextStyle
	styleSet := false

	for _, p := range paras {
		mp := model.Paragraph{}
		if p.PPr != nil {
			switch p.PPr.Algn {
			case "l":
				mp.Style.Align = "left"
			case "ctr":
				mp.Style.Align = "center"
			case "r":
				mp.Style.Align = "right"
			case "just":
				mp.Style.Align = "justify"
			}
			if p.PPr.Lvl > 0 {
				mp.Level = p.PPr.Lvl
			}
			if p.PPr.LnSpc != nil && p.PPr.LnSpc.SpcPct != nil {
				ls := float64(p.PPr.LnSpc.SpcPct.Val) / 100000
				if ls > 0 {
					mp.Style.LineSpacing = ls
				}
			}
		}
		for _, r := range p.Runs {
			run := model.RichTextRun{Text: r.Text}
			if r.RPr != nil {
				run.Style = runStyle(r.RPr)
				if !styleSet {
					style = run.Style
					styleSet = true
				}
			}
			mp.Runs = append(mp.Runs, run)
			full.WriteString(r.Text)
		}
		if mp.Style.Align != "" && !styleSet {
			style.Align = mp.Style.Align
		}
		if len(mp.Runs) == 1 {
			mp.Text = mp.Runs[0].Text
		}
		out = append(out, mp)
	}

	return out, full.String(), style
}

// runStyle converts a:rPr to model.TextStyle.
func runStyle(r *xmlRunProps) model.TextStyle {
	st := model.TextStyle{}
	if r.Sz > 0 {
		st.FontSize = r.Sz / 100
	}
	if r.B == "1" || r.B == "true" {
		st.Bold = true
	}
	if r.I == "1" || r.I == "true" {
		st.Italic = true
	}
	if r.U != "" && r.U != "none" {
		st.Underline = true
	}
	if r.Strike != "" && r.Strike != "noStrike" {
		st.Strike = true
	}
	if r.SolidFill != nil {
		st.Color = srgbToHex(r.SolidFill.Color)
	}
	if r.Latin != nil && r.Latin.Typeface != "" {
		st.FontName = r.Latin.Typeface
	}
	return st
}

// parseConnector converts a p:cxnSp into a connector element.
func parseConnector(cx xmlConnector, slideW, slideH float64) *model.Element {
	rect, rot, ok := rectFromXfrm(cx.SpPr.Xfrm, slideW, slideH)
	if !ok {
		return nil
	}
	conn := &model.ConnectorData{
		ConnectorType: model.ShapeLine,
		StartX:        rect.X,
		StartY:        rect.Y,
		EndX:          round2(rect.X + rect.W),
		EndY:          round2(rect.Y + rect.H),
	}
	if cx.SpPr.Ln != nil {
		if cx.SpPr.Ln.W > 0 {
			conn.Width = float64(cx.SpPr.Ln.W) / 12700
		}
		if cx.SpPr.Ln.Solid != nil {
			conn.Color = srgbToHex(cx.SpPr.Ln.Solid.Color)
		}
	}
	return &model.Element{
		Type:      model.ElementConnector,
		Rect:      rect,
		Rotation:  rot,
		Connector: conn,
	}
}

// parsePicture converts a p:pic into an image element. mediaFile is the
// package-absolute target (may be empty if the rel was not found).
func parsePicture(p xmlPicture, mediaFile string, slideW, slideH float64, warnings *[]string) *model.Element {
	rect, rot, ok := rectFromXfrm(p.SpPr.Xfrm, slideW, slideH)
	if !ok {
		*warnings = append(*warnings, fmt.Sprintf("picture %q without geometry skipped", p.Name()))
		return nil
	}
	elem := &model.Element{
		Type:     model.ElementImage,
		Rect:     rect,
		Rotation: rot,
		ImageAlt: p.Name(),
	}
	if mediaFile != "" {
		// Reference the embedded part by its package path; re-export resolves
		// these at build time.
		elem.ImagePath = "pptx:" + mediaFile
	} else {
		*warnings = append(*warnings, fmt.Sprintf("picture %q: embedded media not found in rels", p.Name()))
	}
	if sr := p.BlipFill.SrcRect; sr != nil && (sr.L|sr.T|sr.R|sr.B) != 0 {
		elem.ImageCrop = &model.ImageCrop{
			Left:   float64(sr.L) / 1000,
			Top:    float64(sr.T) / 1000,
			Right:  float64(sr.R) / 1000,
			Bottom: float64(sr.B) / 1000,
		}
	}
	return elem
}

// parseTable converts a graphicFrame hosting a:tbl into a table element.
func parseTable(gf xmlGraphicFrame, slideW, slideH float64, warnings *[]string) *model.Element {
	rect, _, ok := rectFromXfrm(gf.Xfrm, slideW, slideH)
	if !ok {
		*warnings = append(*warnings, "table without geometry skipped")
		return nil
	}
	t := gf.Graphic.GraphicData.Table
	table := &model.TableData{}

	cellText := func(c xmlCell) string {
		var b strings.Builder
		for _, p := range c.Paragraphs {
			b.WriteString(paragraphPlainText(p))
		}
		return b.String()
	}
	cellModel := func(c xmlCell) model.TableCell {
		tc := model.TableCell{Text: cellText(c)}
		if c.TCPr != nil {
			tc.ColSpan = c.TCPr.GridSpan
			tc.RowSpan = c.TCPr.RowSpan
			if c.TCPr.HMerge || c.TCPr.VMerge {
				*warnings = append(*warnings, "table merged cell approximated (spans kept, no merge semantics)")
			}
		}
		return tc
	}

	for ri, row := range t.Rows {
		var cells []model.TableCell
		for _, c := range row.Cells {
			cells = append(cells, cellModel(c))
		}
		if ri == 0 {
			table.Headers = cells
		} else {
			table.Rows = append(table.Rows, cells)
		}
	}
	if len(table.Headers) == 0 && len(table.Rows) > 0 {
		table.Headers = table.Rows[0]
		table.Rows = table.Rows[1:]
	}

	return &model.Element{
		Type:  model.ElementTable,
		Rect:  rect,
		Table: table,
	}
}

// ---------- chart parsing ----------

type xmlChartVal struct {
	V float64 `xml:"v"`
}

type xmlChartStr struct {
	V string `xml:"v"`
}

type xmlChartSeries struct {
	Tx *struct {
		StrRef *struct {
			StrCache *struct {
				Pts []xmlChartStr `xml:"pt"`
			} `xml:"strCache"`
		} `xml:"strRef"`
		V string `xml:"v"`
	} `xml:"tx"`
	Cat *struct {
		StrRef *struct {
			StrCache *struct {
				Pts []xmlChartStr `xml:"pt"`
			} `xml:"strCache"`
		} `xml:"strRef"`
		NumRef *struct {
			NumCache *struct {
				Pts []xmlChartStr `xml:"pt"`
			} `xml:"numCache"`
		} `xml:"numRef"`
	} `xml:"cat"`
	Val *struct {
		NumRef *struct {
			NumCache *struct {
				Pts []xmlChartVal `xml:"pt"`
			} `xml:"numCache"`
		} `xml:"numRef"`
	} `xml:"val"`
	XVal *struct {
		NumRef *struct {
			NumCache *struct {
				Pts []xmlChartVal `xml:"pt"`
			} `xml:"numCache"`
		} `xml:"numRef"`
	} `xml:"xVal"`
	YVal *struct {
		NumRef *struct {
			NumCache *struct {
				Pts []xmlChartVal `xml:"pt"`
			} `xml:"numCache"`
		} `xml:"numRef"`
	} `xml:"yVal"`
	Smooth *struct {
		Val bool `xml:"val,attr"`
	} `xml:"smooth"`
}

func (s *xmlChartSeries) name() string {
	if s.Tx != nil {
		if s.Tx.StrRef != nil && s.Tx.StrRef.StrCache != nil && len(s.Tx.StrRef.StrCache.Pts) > 0 {
			return s.Tx.StrRef.StrCache.Pts[0].V
		}
		return s.Tx.V
	}
	return ""
}

func (s *xmlChartSeries) categories() []string {
	var out []string
	if s.Cat != nil {
		if s.Cat.StrRef != nil && s.Cat.StrRef.StrCache != nil {
			for _, p := range s.Cat.StrRef.StrCache.Pts {
				out = append(out, p.V)
			}
		}
		if s.Cat.NumRef != nil && s.Cat.NumRef.NumCache != nil {
			for _, p := range s.Cat.NumRef.NumCache.Pts {
				out = append(out, p.V)
			}
		}
	}
	return out
}

func (s *xmlChartSeries) values() []float64 {
	var out []float64
	if s.Val != nil && s.Val.NumRef != nil && s.Val.NumRef.NumCache != nil {
		for _, p := range s.Val.NumRef.NumCache.Pts {
			out = append(out, p.V)
		}
	}
	if s.YVal != nil && s.YVal.NumRef != nil && s.YVal.NumRef.NumCache != nil {
		for _, p := range s.YVal.NumRef.NumCache.Pts {
			out = append(out, p.V)
		}
	}
	return out
}

func (s *xmlChartSeries) xValues() []float64 {
	var out []float64
	if s.XVal != nil && s.XVal.NumRef != nil && s.XVal.NumRef.NumCache != nil {
		for _, p := range s.XVal.NumRef.NumCache.Pts {
			out = append(out, p.V)
		}
	}
	return out
}

func seriesToModel(list []xmlChartSeries, ct model.ChartType) []model.ChartSeries {
	var out []model.ChartSeries
	for i := range list {
		s := &list[i]
		ms := model.ChartSeries{
			Name:      s.name(),
			Values:    s.values(),
			XValues:   s.xValues(),
			ChartType: ct,
		}
		if s.Smooth != nil {
			ms.Smooth = s.Smooth.Val
		}
		out = append(out, ms)
	}
	return out
}

// xmlChartBlock models one plot block (barChart, lineChart, ...) with its
// barDir and series. Single-element tags avoid path-tag conflicts.
type xmlChartBlock struct {
	BarDir struct {
		Val string `xml:"val,attr"`
	} `xml:"barDir"`
	Sers []xmlChartSeries `xml:"ser"`
}

type xmlPlotArea struct {
	Bar       *xmlChartBlock `xml:"barChart"`
	Bar3D     *xmlChartBlock `xml:"bar3DChart"`
	Line      *xmlChartBlock `xml:"lineChart"`
	Line3D    *xmlChartBlock `xml:"line3DChart"`
	Pie       *xmlChartBlock `xml:"pieChart"`
	Pie3D     *xmlChartBlock `xml:"pie3DChart"`
	Area      *xmlChartBlock `xml:"areaChart"`
	Area3D    *xmlChartBlock `xml:"area3DChart"`
	Doughnut  *xmlChartBlock `xml:"doughnutChart"`
	Scatter   *xmlChartBlock `xml:"scatterChart"`
}

type xmlChartDoc struct {
	PlotArea xmlPlotArea `xml:"chart>plotArea"`
	Legend   *struct{}   `xml:"chart>legend"`
	Title    struct {
		Text string `xml:"tx>rich>p>r>t"`
	} `xml:"chart>title"`
}

// parseChart converts a chart part (chartN.xml) + its host graphicFrame
// into a chart element.
func parseChart(chartRaw []byte, gf xmlGraphicFrame, slideW, slideH float64, warnings *[]string) *model.Element {
	var doc xmlChartDoc
	if err := xml.Unmarshal(chartRaw, &doc); err != nil {
		*warnings = append(*warnings, fmt.Sprintf("chart %q parse failed: %v", gf.Name(), err))
		return nil
	}

	chart := &model.ChartData{
		Title:      doc.Title.Text,
		ShowLegend: doc.Legend != nil,
	}

	pa := &doc.PlotArea
	barDir := ""
	if pa.Bar != nil {
		barDir = pa.Bar.BarDir.Val
	} else if pa.Bar3D != nil {
		barDir = pa.Bar3D.BarDir.Val
	}

	var series []model.ChartSeries
	var rawSeries []xmlChartSeries
	switch {
	case pa.Bar != nil && pa.Line != nil:
		chart.ChartType = model.ChartCombo
		series = append(series, seriesToModel(pa.Bar.Sers, model.ChartBar)...)
		series = append(series, seriesToModel(pa.Line.Sers, model.ChartLine)...)
		rawSeries = append(pa.Bar.Sers, pa.Line.Sers...)
	case pa.Bar != nil:
		if barDir == "bar" {
			chart.ChartType = model.ChartBar
		} else {
			chart.ChartType = model.ChartColumn
		}
		series = seriesToModel(pa.Bar.Sers, chart.ChartType)
		rawSeries = pa.Bar.Sers
	case pa.Bar3D != nil:
		if barDir == "bar" {
			chart.ChartType = model.ChartBar3D
		} else {
			chart.ChartType = model.ChartColumn3D
		}
		series = seriesToModel(pa.Bar3D.Sers, chart.ChartType)
		rawSeries = pa.Bar3D.Sers
	case pa.Line3D != nil:
		chart.ChartType = model.ChartLine3D
		series = seriesToModel(pa.Line3D.Sers, chart.ChartType)
		rawSeries = pa.Line3D.Sers
	case pa.Line != nil:
		chart.ChartType = model.ChartLine
		series = seriesToModel(pa.Line.Sers, chart.ChartType)
		rawSeries = pa.Line.Sers
	case pa.Pie3D != nil:
		chart.ChartType = model.ChartPie3D
		series = seriesToModel(pa.Pie3D.Sers, chart.ChartType)
		rawSeries = pa.Pie3D.Sers
	case pa.Pie != nil:
		chart.ChartType = model.ChartPie
		series = seriesToModel(pa.Pie.Sers, chart.ChartType)
		rawSeries = pa.Pie.Sers
	case pa.Area3D != nil:
		chart.ChartType = model.ChartArea3D
		series = seriesToModel(pa.Area3D.Sers, chart.ChartType)
		rawSeries = pa.Area3D.Sers
	case pa.Area != nil:
		chart.ChartType = model.ChartArea
		series = seriesToModel(pa.Area.Sers, chart.ChartType)
		rawSeries = pa.Area.Sers
	case pa.Doughnut != nil:
		chart.ChartType = model.ChartDoughnut
		series = seriesToModel(pa.Doughnut.Sers, chart.ChartType)
		rawSeries = pa.Doughnut.Sers
	case pa.Scatter != nil:
		chart.ChartType = model.ChartScatter
		series = seriesToModel(pa.Scatter.Sers, chart.ChartType)
		rawSeries = pa.Scatter.Sers
	default:
		*warnings = append(*warnings, fmt.Sprintf("chart %q: unrecognized plot type, skipped", gf.Name()))
		return nil
	}

	chart.Series = series
	if len(rawSeries) > 0 {
		chart.Categories = (&rawSeries[0]).categories()
	}

	rect, _, ok := rectFromXfrm(gf.Xfrm, slideW, slideH)
	if !ok {
		*warnings = append(*warnings, fmt.Sprintf("chart %q without geometry, defaulting rect", gf.Name()))
		rect = model.Rect{X: 5, Y: 15, W: 90, H: 70}
	}

	return &model.Element{
		Type:  model.ElementChart,
		Rect:  rect,
		Chart: chart,
	}
}
