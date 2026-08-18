package svgdecode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// parseBounds parses data-pptx-bounds="x y width height" in viewBox units.
// It returns a model.Rect in percentage coordinates.
func (c *compiler) parseBounds(n *svgNode) (model.Rect, bool) {
	raw := strings.TrimSpace(n.attr("data-pptx-bounds"))
	if raw == "" {
		return model.Rect{}, false
	}
	nums := ParseNumbers(raw)
	if len(nums) != 4 || nums[2] <= 0 || nums[3] <= 0 {
		return model.Rect{}, false
	}
	return model.Rect{
		X: c.pct(nums[0], c.canvas.ViewW),
		Y: c.pct(nums[1], c.canvas.ViewH),
		W: c.pct(nums[2], c.canvas.ViewW),
		H: c.pct(nums[3], c.canvas.ViewH),
	}, true
}

// nativeMetadata is the JSON payload carried by data-pptx-replace-with regions.
type nativeMetadata struct {
	Type       string         `json:"type"`
	Categories []string       `json:"categories"`
	Values     []float64      `json:"values"`
	X          []float64      `json:"x"`
	Y          []float64      `json:"y"`
	Size       []float64      `json:"size"`
	Series     []nativeSeries `json:"series"`
	Headers    []string       `json:"headers"`
	Rows       [][]string     `json:"rows"`
	Style      struct {
		Colors []string `json:"colors"`
		Color  string   `json:"color"`
	} `json:"style"`
}

type nativeSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color"`
}

// tryNativeObject inspects a region carrying data-pptx-replace-with and, when
// the embedded metadata maps to a supported native PowerPoint object, emits one
// native chart/table element and returns true. The compiler then skips the
// shape fallback children. Unsupported kinds return false and keep the editable
// shape fallback path.
func (c *compiler) tryNativeObject(n *svgNode) bool {
	replace := strings.TrimSpace(n.attr("data-pptx-replace-with"))
	if replace == "" {
		return false
	}

	rect, ok := c.parseBounds(n)
	if !ok {
		return false
	}

	meta, ok := c.readNativeMetadata(n)
	if !ok {
		return false
	}

	switch replace {
	case "chart":
		chart := nativeChart(meta)
		if chart == nil {
			return false
		}
		c.add(&model.Element{
			ID:    c.idFor(n, "chart"),
			Type:  model.ElementChart,
			Rect:  rect,
			Chart: chart,
		})
		return true
	case "table":
		table := nativeTable(meta)
		if table == nil {
			return false
		}
		c.add(&model.Element{
			ID:    c.idFor(n, "table"),
			Type:  model.ElementTable,
			Rect:  rect,
			Table: table,
		})
		return true
	default:
		return false
	}
}

func (c *compiler) readNativeMetadata(n *svgNode) (*nativeMetadata, bool) {
	for i := range n.Children {
		child := &n.Children[i]
		if child.XMLName.Local != "metadata" {
			continue
		}
		var meta nativeMetadata
		if err := json.Unmarshal([]byte(strings.TrimSpace(child.Text)), &meta); err != nil {
			c.skip(fmt.Sprintf("<metadata> invalid json: %v", err))
			return nil, false
		}
		if meta.Type == "" {
			return nil, false
		}
		return &meta, true
	}
	return nil, false
}

var nativeChartTypes = map[string]model.ChartType{
	"bar":      model.ChartBar,
	"column":   model.ChartColumn,
	"line":     model.ChartLine,
	"pie":      model.ChartPie,
	"doughnut": model.ChartDoughnut,
	"scatter":  model.ChartScatter,
}

func nativeChart(meta *nativeMetadata) *model.ChartData {
	ct, ok := nativeChartTypes[strings.ToLower(meta.Type)]
	if !ok {
		return nil
	}

	// Scatter with a bubble-size channel cannot be represented natively yet;
	// keep the shape fallback so the size encoding is preserved.
	if ct == model.ChartScatter && len(meta.Size) > 0 {
		return nil
	}

	cd := &model.ChartData{
		ChartType:      ct,
		Categories:     meta.Categories,
		ShowLegend:     true,
		ShowDataLabels: false,
	}

	if len(meta.Series) > 0 {
		for i, s := range meta.Series {
			cs := model.ChartSeries{Name: s.Name, Values: s.Values, Color: s.Color}
			if cs.Color == "" && i < len(meta.Style.Colors) {
				cs.Color = meta.Style.Colors[i]
			}
			cd.Series = append(cd.Series, cs)
		}
	} else if len(meta.Values) > 0 {
		name := strings.ToLower(meta.Type)
		cs := model.ChartSeries{Name: name, Values: meta.Values}
		if len(meta.Style.Colors) > 0 {
			cs.Color = meta.Style.Colors[0]
		}
		cd.Series = append(cd.Series, cs)
	}

	// Scatter: x values belong on the X axis, y values on Y.
	if ct == model.ChartScatter && len(meta.X) > 0 && len(meta.Series) > 0 {
		cd.Series[0].XValues = meta.X
	}

	if len(cd.Series) == 0 {
		return nil
	}
	return cd
}

func nativeTable(meta *nativeMetadata) *model.TableData {
	if len(meta.Headers) == 0 || len(meta.Rows) == 0 {
		return nil
	}
	td := &model.TableData{}
	for _, h := range meta.Headers {
		td.Headers = append(td.Headers, model.TableCell{Text: h})
	}
	for _, row := range meta.Rows {
		cells := make([]model.TableCell, 0, len(row))
		for _, cell := range row {
			cells = append(cells, model.TableCell{Text: cell})
		}
		td.Rows = append(td.Rows, cells)
	}
	return td
}
