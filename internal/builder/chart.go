package builder

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func (b *Builder) writeChartPart(zw *zip.Writer, asset *chartAsset) error {
	if asset.data == nil {
		return nil
	}
	w, err := zw.Create(fmt.Sprintf("ppt/charts/chart%d.xml", asset.index))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(b.chartXML(asset.data, asset.index)))
	return err
}

func (b *Builder) chartXML(chart *model.ChartData, chartIndex int) string {
	isScatter := chart.ChartType == model.ChartScatter
	isCombo := chart.ChartType == model.ChartCombo
	chartType := nativeChartType(chart.ChartType)
	catID, valID := chartIndex*2+100000, chartIndex*2+100001
	secValID := chartIndex*2+100002 // secondary axis for combo charts

	var plot strings.Builder

	if isCombo {
		plot.WriteString(comboPlotXML(chart, catID, valID, secValID))
	} else {
		fmt.Fprintf(&plot, `<c:%s>`, chartType)
		switch chart.ChartType {
		case model.ChartBar:
			plot.WriteString(`<c:barDir val="bar"/><c:grouping val="clustered"/><c:varyColors val="0"/>`)
		case model.ChartColumn:
			plot.WriteString(`<c:barDir val="col"/><c:grouping val="clustered"/><c:varyColors val="0"/>`)
		case model.ChartPie, model.ChartDoughnut:
			plot.WriteString(`<c:varyColors val="1"/>`)
		case model.ChartLine:
			plot.WriteString(`<c:grouping val="standard"/><c:varyColors val="0"/>`)
		case model.ChartArea:
			plot.WriteString(`<c:grouping val="standard"/><c:varyColors val="0"/>`)
		case model.ChartScatter:
			plot.WriteString(`<c:scatterStyle val="lineMarker"/><c:varyColors val="0"/>`)
		default:
			plot.WriteString(`<c:grouping val="standard"/><c:varyColors val="0"/>`)
		}
		// Add smooth for line charts
		if chart.ChartType == model.ChartLine && chart.Smooth {
			plot.WriteString(`<c:smooth val="1"/>`)
		}
		for i, series := range chart.Series {
			if isScatter {
				plot.WriteString(scatterSeriesXML(chart, series, i))
			} else {
				plot.WriteString(chartSeriesXML(chart, series, i))
			}
		}
		if chart.ChartType == model.ChartDoughnut {
			plot.WriteString(`<c:holeSize val="50"/>`)
		}
		if chart.ShowDataLabels {
			if chart.ChartType == model.ChartPie || chart.ChartType == model.ChartDoughnut {
				fmt.Fprintf(&plot, `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="0"/><c:showCatName val="1"/><c:showSerName val="0"/><c:showPercent val="1"/><c:showBubbleSize val="0"/></c:dLbls>`)
			} else {
				fmt.Fprintf(&plot, `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`)
			}
		}
		if chart.ChartType != model.ChartPie && chart.ChartType != model.ChartDoughnut {
			fmt.Fprintf(&plot, `<c:axId val="%d"/><c:axId val="%d"/>`, catID, valID)
		}
		fmt.Fprintf(&plot, `</c:%s>`, chartType)
	}

	axes := ""
	if chart.ChartType != model.ChartPie && chart.ChartType != model.ChartDoughnut {
		if isScatter {
			axes = fmt.Sprintf(
				`<c:valAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="b"/><c:majorGridlines/><c:numFmt formatCode="General" sourceLinked="1"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx>`+
					`<c:valAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="l"/><c:majorGridlines/><c:numFmt formatCode="General" sourceLinked="1"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx>`,
				catID, valID, valID, catID)
		} else if isCombo {
			// Combo: catAx + primary valAx (left) + secondary valAx (right)
			axes = fmt.Sprintf(
				`<c:catAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="b"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:auto val="1"/><c:lblAlgn val="ctr"/><c:lblOffset val="100"/></c:catAx>`+
					`<c:valAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="l"/><c:majorGridlines/><c:numFmt formatCode="General" sourceLinked="1"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx>`+
					`<c:valAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="r"/><c:numFmt formatCode="General" sourceLinked="1"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="max"/></c:valAx>`,
				catID, valID, valID, catID, secValID, catID)
		} else {
			axes = fmt.Sprintf(`<c:catAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="b"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:auto val="1"/><c:lblAlgn val="ctr"/><c:lblOffset val="100"/></c:catAx><c:valAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="l"/><c:majorGridlines/><c:numFmt formatCode="General" sourceLinked="1"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx>`, catID, valID, valID, catID)
		}
	}
	title := ""
	if chart.Title != "" {
		title = fmt.Sprintf(`<c:title><c:tx><c:rich><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr lang="zh-CN"/><a:t>%s</a:t></a:r></a:p></c:rich></c:tx><c:layout/><c:overlay val="0"/></c:title>`, xmlEscape(chart.Title))
	}
	legend := ""
	if chart.ShowLegend {
		legend = `<c:legend><c:legendPos val="r"/><c:layout/><c:overlay val="0"/></c:legend>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><c:chartSpace xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><c:date1904 val="0"/><c:lang val="zh-CN"/><c:roundedCorners val="0"/><c:chart>` + title + `<c:autoTitleDeleted val="0"/><c:plotArea><c:layout/>` + plot.String() + axes + `</c:plotArea>` + legend + `<c:plotVisOnly val="1"/><c:dispBlanksAs val="gap"/><c:showDLblsOverMax val="0"/></c:chart><c:printSettings><c:headerFooter/><c:pageMargins b="0.75" l="0.7" r="0.7" t="0.75" header="0.3" footer="0.3"/><c:pageSetup/></c:printSettings></c:chartSpace>`
}

func nativeChartType(chartType model.ChartType) string {
	switch chartType {
	case model.ChartBar, model.ChartColumn:
		return "barChart"
	case model.ChartLine:
		return "lineChart"
	case model.ChartPie:
		return "pieChart"
	case model.ChartArea:
		return "areaChart"
	case model.ChartDoughnut:
		return "doughnutChart"
	case model.ChartScatter:
		return "scatterChart"
	case model.ChartCombo:
		return "barChart" // combo uses custom rendering
	default:
		return "lineChart"
	}
}

// seriesColor returns the hex color (without #) for a chart series, using a
// default palette when not specified.
func seriesColor(series model.ChartSeries, index int) string {
	color := strings.TrimPrefix(series.Color, "#")
	if color == "" {
		palette := []string{"4472C4", "ED7D31", "A5A5A5", "FFC000", "5B9BD5", "70AD47"}
		color = palette[index%len(palette)]
	}
	return color
}

// markerXML builds a c:marker element for line/scatter charts.
func markerXML(color string) string {
	return fmt.Sprintf(`<c:marker><c:symbol val="circle"/><c:size val="7"/><c:spPr><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:ln><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:marker>`, color, color)
}

// trendlineXML builds a c:trendline element.
func trendlineXML(trendlineType string) string {
	trendVal := "linear"
	switch trendlineType {
	case "exponential":
		trendVal = "exp"
	case "movingAvg":
		trendVal = "movingAvg"
	case "polynomial":
		trendVal = "poly"
	default:
		trendVal = "linear"
	}
	return fmt.Sprintf(`<c:trendline><c:trendlineType val="%s"/><c:dispEq val="0"/><c:dispRSqr val="0"/></c:trendline>`, trendVal)
}

// categoryCacheXML generates c:cat with c:strRef wrapping c:strCache.
func categoryCacheXML(categories []string) string {
	var pts strings.Builder
	for i, cat := range categories {
		fmt.Fprintf(&pts, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, xmlEscape(cat))
	}
	return fmt.Sprintf(`<c:cat><c:strRef><c:f></c:f><c:strCache><c:ptCount val="%d"/>%s</c:strCache></c:strRef></c:cat>`, len(categories), pts.String())
}

// valueCacheXML generates c:val with c:numRef wrapping c:numCache.
func valueCacheXML(values []float64) string {
	var pts strings.Builder
	for i, v := range values {
		fmt.Fprintf(&pts, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, fmtFloat(v))
	}
	return fmt.Sprintf(`<c:val><c:numRef><c:f></c:f><c:numCache><c:formatCode>General</c:formatCode><c:ptCount val="%d"/>%s</c:numCache></c:numRef></c:val>`, len(values), pts.String())
}

// numCacheOnlyXML generates a bare c:numRef for scatter xVal/yVal.
func numCacheOnlyXML(values []float64) string {
	var pts strings.Builder
	for i, v := range values {
		fmt.Fprintf(&pts, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, fmtFloat(v))
	}
	return fmt.Sprintf(`<c:numRef><c:f></c:f><c:numCache><c:formatCode>General</c:formatCode><c:ptCount val="%d"/>%s</c:numCache></c:numRef>`, len(values), pts.String())
}

func chartSeriesXML(chart *model.ChartData, series model.ChartSeries, index int) string {
	color := seriesColor(series, index)
	dLbls := ""
	if chart.ShowDataLabels {
		dLbls = `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`
	}
	// Markers for line charts
	marker := ""
	if chart.ChartType == model.ChartLine {
		marker = markerXML(color)
	}
	// Per-series smooth
	smooth := ""
	if (chart.ChartType == model.ChartLine) && (chart.Smooth || series.Smooth) {
		smooth = `<c:smooth val="1"/>`
	}
	// Per-series trendline
	trendline := ""
	if series.Trendline != "" {
		trendline = trendlineXML(series.Trendline)
	}
	catXML := categoryCacheXML(chart.Categories)
	valXML := valueCacheXML(series.Values)
	return fmt.Sprintf(`<c:ser><c:idx val="%d"/><c:order val="%d"/><c:tx><c:v>%s</c:v></c:tx><c:spPr><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:ln><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:spPr>%s%s%s%s%s%s</c:ser>`,
		index, index, xmlEscape(series.Name), color, color, marker, smooth, trendline, dLbls, catXML, valXML)
}

// scatterSeriesXML builds a c:ser element for scatter charts using c:xVal/c:yVal.
func scatterSeriesXML(chart *model.ChartData, series model.ChartSeries, index int) string {
	color := seriesColor(series, index)

	xData := series.XValues
	if len(xData) == 0 {
		xData = make([]float64, len(series.Values))
		for i := range series.Values {
			xData[i] = float64(i)
		}
	}

	dLbls := ""
	if chart.ShowDataLabels {
		dLbls = `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`
	}

	marker := markerXML(color)
	xValXML := numCacheOnlyXML(xData)
	yValXML := numCacheOnlyXML(series.Values)

	// Per-series smooth
	smooth := ""
	if chart.Smooth || series.Smooth {
		smooth = `<c:smooth val="1"/>`
	}

	// Per-series trendline
	trendline := ""
	if series.Trendline != "" {
		trendline = trendlineXML(series.Trendline)
	}

	return fmt.Sprintf(`<c:ser><c:idx val="%d"/><c:order val="%d"/><c:tx><c:v>%s</c:v></c:tx><c:spPr><a:ln><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:spPr>%s%s%s%s<c:xVal>%s</c:xVal><c:yVal>%s</c:yVal></c:ser>`,
		index, index, xmlEscape(series.Name), color, marker, smooth, trendline, dLbls, xValXML, yValXML)
}

// comboPlotXML renders a combo chart with bar+line sub-charts and optional secondary axis.
func comboPlotXML(chart *model.ChartData, catID, valID, secValID int) string {
	var barSeries, linePrimarySeries, lineSecondarySeries []model.ChartSeries
	var barIdx, lineIdx int

	for _, s := range chart.Series {
		seriesChartType := s.ChartType
		if seriesChartType == "" {
			seriesChartType = model.ChartColumn // default to column
		}
		if seriesChartType == model.ChartLine {
			if s.SecondaryAxis {
				lineSecondarySeries = append(lineSecondarySeries, s)
			} else {
				linePrimarySeries = append(linePrimarySeries, s)
			}
		} else {
			barSeries = append(barSeries, s)
		}
	}

	var plot strings.Builder

	// Render bar chart sub-element
	if len(barSeries) > 0 {
		plot.WriteString(`<c:barChart><c:barDir val="col"/><c:grouping val="clustered"/><c:varyColors val="0"/>`)
		for _, s := range barSeries {
			plot.WriteString(comboSeriesXML(chart, s, barIdx, catID, valID))
			barIdx++
		}
		if chart.ShowDataLabels {
			plot.WriteString(`<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`)
		}
		fmt.Fprintf(&plot, `<c:axId val="%d"/><c:axId val="%d"/></c:barChart>`, catID, valID)
	}

	// Render line chart sub-element (primary axis)
	if len(linePrimarySeries) > 0 {
		plot.WriteString(`<c:lineChart><c:grouping val="standard"/><c:varyColors val="0"/>`)
		if chart.Smooth {
			plot.WriteString(`<c:smooth val="1"/>`)
		}
		for _, s := range linePrimarySeries {
			plot.WriteString(comboSeriesXML(chart, s, barIdx+lineIdx, catID, valID))
			lineIdx++
		}
		if chart.ShowDataLabels {
			plot.WriteString(`<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`)
		}
		fmt.Fprintf(&plot, `<c:axId val="%d"/><c:axId val="%d"/></c:lineChart>`, catID, valID)
	}

	// Render line chart sub-element (secondary axis)
	if len(lineSecondarySeries) > 0 {
		plot.WriteString(`<c:lineChart><c:grouping val="standard"/><c:varyColors val="0"/>`)
		if chart.Smooth {
			plot.WriteString(`<c:smooth val="1"/>`)
		}
		for _, s := range lineSecondarySeries {
			plot.WriteString(comboSeriesXML(chart, s, barIdx+lineIdx, catID, secValID))
			lineIdx++
		}
		if chart.ShowDataLabels {
			plot.WriteString(`<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`)
		}
		fmt.Fprintf(&plot, `<c:axId val="%d"/><c:axId val="%d"/></c:lineChart>`, catID, secValID)
	}

	return plot.String()
}

// comboSeriesXML renders a single series for combo charts (bar or line).
func comboSeriesXML(chart *model.ChartData, series model.ChartSeries, index int, catID, valID int) string {
	color := seriesColor(series, index)
	isLine := series.ChartType == model.ChartLine

	marker := ""
	solidFill := fmt.Sprintf(`<a:solidFill><a:srgbClr val="%s"/></a:solidFill>`, color)
	if isLine {
		marker = markerXML(color)
		solidFill = "" // lines only need line color, not fill
	}

	smooth := ""
	if isLine && (chart.Smooth || series.Smooth) {
		smooth = `<c:smooth val="1"/>`
	}

	trendline := ""
	if series.Trendline != "" {
		trendline = trendlineXML(series.Trendline)
	}

	catXML := categoryCacheXML(chart.Categories)
	valXML := valueCacheXML(series.Values)

	spPr := fmt.Sprintf(`<c:spPr>%s<a:ln><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:spPr>`, solidFill, color)

	return fmt.Sprintf(`<c:ser><c:idx val="%d"/><c:order val="%d"/><c:tx><c:v>%s</c:v></c:tx>%s%s%s%s%s%s</c:ser>`,
		index, index, xmlEscape(series.Name), spPr, marker, smooth, trendline, catXML, valXML)
}
