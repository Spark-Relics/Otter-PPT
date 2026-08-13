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
	chartType := nativeChartType(chart.ChartType)
	var plot strings.Builder
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
	default:
		plot.WriteString(`<c:grouping val="standard"/><c:varyColors val="0"/>`)
	}
	for i, series := range chart.Series {
		plot.WriteString(chartSeriesXML(chart, series, i))
	}
	if chart.ChartType == model.ChartDoughnut {
		plot.WriteString(`<c:holeSize val="50"/>`)
	}
	if chart.ShowDataLabels {
		// For pie/doughnut, show percentage instead of value
		if chart.ChartType == model.ChartPie || chart.ChartType == model.ChartDoughnut {
			fmt.Fprintf(&plot, `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="0"/><c:showCatName val="1"/><c:showSerName val="0"/><c:showPercent val="1"/><c:showBubbleSize val="0"/></c:dLbls>`)
		} else {
			fmt.Fprintf(&plot, `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`)
		}
	}
	if chart.ChartType != model.ChartPie && chart.ChartType != model.ChartDoughnut {
		fmt.Fprintf(&plot, `<c:axId val="%d"/><c:axId val="%d"/>`, chartIndex*2+100000, chartIndex*2+100001)
	}
	fmt.Fprintf(&plot, `</c:%s>`, chartType)

	axes := ""
	if chart.ChartType != model.ChartPie && chart.ChartType != model.ChartDoughnut {
		catID, valID := chartIndex*2+100000, chartIndex*2+100001
		axes = fmt.Sprintf(`<c:catAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="b"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:auto val="1"/><c:lblAlgn val="ctr"/><c:lblOffset val="100"/></c:catAx><c:valAx><c:axId val="%d"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:delete val="0"/><c:axPos val="l"/><c:majorGridlines/><c:numFmt formatCode="General" sourceLinked="1"/><c:tickLblPos val="nextTo"/><c:crossAx val="%d"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx>`, catID, valID, valID, catID)
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
	default:
		return "lineChart"
	}
}

func chartSeriesXML(chart *model.ChartData, series model.ChartSeries, index int) string {
	color := strings.TrimPrefix(series.Color, "#")
	if color == "" {
		palette := []string{"4472C4", "ED7D31", "A5A5A5", "FFC000", "5B9BD5", "70AD47"}
		color = palette[index%len(palette)]
	}
	var categories strings.Builder
	for i, category := range chart.Categories {
		fmt.Fprintf(&categories, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, xmlEscape(category))
	}
	var values strings.Builder
	for i, value := range series.Values {
		fmt.Fprintf(&values, `<c:pt idx="%d"><c:v>%s</c:v></c:pt>`, i, fmtFloat(value))
	}
	// Data labels per-series if ShowDataLabels is enabled
	dLbls := ""
	if chart.ShowDataLabels {
		dLbls = `<c:dLbls><c:showLegendKey val="0"/><c:showVal val="1"/><c:showCatName val="0"/><c:showSerName val="0"/><c:showPercent val="0"/><c:showBubbleSize val="0"/></c:dLbls>`
	}
	return fmt.Sprintf(`<c:ser><c:idx val="%d"/><c:order val="%d"/><c:tx><c:v>%s</c:v></c:tx><c:spPr><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:ln><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:spPr>%s<c:cat><c:strLit><c:ptCount val="%d"/>%s</c:strLit></c:cat><c:val><c:numLit><c:formatCode>General</c:formatCode><c:ptCount val="%d"/>%s</c:numLit></c:val></c:ser>`, index, index, xmlEscape(series.Name), color, color, dLbls, len(chart.Categories), categories.String(), len(series.Values), values.String())
}
