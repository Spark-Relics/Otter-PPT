package parser

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// ---------- slide XML structures ----------

type xmlPoint struct {
	X int `xml:"x,attr"`
	Y int `xml:"y,attr"`
}

type xmlExt struct {
	CX int `xml:"cx,attr"`
	CY int `xml:"cy,attr"`
}

type xmlXfrm struct {
	Off   xmlPoint `xml:"off"`
	Ext   xmlExt   `xml:"ext"`
	Rot   int      `xml:"rot,attr"`
	FlipH bool     `xml:"flipH,attr"`
	FlipV bool     `xml:"flipV,attr"`
}

type xmlSrgbClr struct {
	Val   string `xml:"val,attr"`
	Alpha *struct {
		Val int `xml:"val,attr"`
	} `xml:"alpha"`
}

type xmlGradientStop struct {
	Pos   int         `xml:"pos,attr"`
	Color *xmlSrgbClr `xml:"srgbClr"`
}

type xmlGradFill struct {
	Lin *struct {
		Ang int `xml:"ang,attr"`
	} `xml:"lin"`
	Stops []xmlGradientStop `xml:"gsLst>gs"`
}

type xmlSolidFill struct {
	Color *xmlSrgbClr `xml:"srgbClr"`
}

type xmlPrstDash struct {
	Val string `xml:"val,attr"`
}

type xmlLn struct {
	W      int           `xml:"w,attr"` // EMU
	Solid  *xmlSolidFill `xml:"solidFill"`
	PrstDash *xmlPrstDash `xml:"prstDash"`
}

type xmlPrstGeom struct {
	Prst string `xml:"prst,attr"`
}

// xmlSpPr is the shared shape properties node.
type xmlSpPr struct {
	Xfrm              *xmlXfrm      `xml:"xfrm"`
	PrstGeom          *xmlPrstGeom  `xml:"prstGeom"`
	CustGeomPresent   bool          `xml:"custGeom"`
	SolidFill         *xmlSolidFill `xml:"solidFill"`
	GradFill          *xmlGradFill  `xml:"gradFill"`
	NoFillPresent     bool          `xml:"noFill"`
	Ln                *xmlLn        `xml:"ln"`
}

// xmlRunProps captures a:rPr styling. b/i/u/strike are attributes in OOXML.
type xmlRunProps struct {
	Sz        int          `xml:"sz,attr"` // hundredths of a point
	B         string       `xml:"b,attr"`
	I         string       `xml:"i,attr"`
	U         string       `xml:"u,attr"`
	Strike    string       `xml:"strike,attr"`
	SolidFill *xmlSolidFill `xml:"solidFill"`
	Latin     *struct {
		Typeface string `xml:"typeface,attr"`
	} `xml:"latin"`
}

type xmlRun struct {
	Text string       `xml:"t"`
	RPr  *xmlRunProps `xml:"rPr"`
}

type xmlSpcPct struct {
	Val int `xml:"val,attr"`
}

type xmlParagraphProps struct {
	Algn  string `xml:"algn,attr"`
	Lvl   int    `xml:"lvl,attr"`
	LnSpc *struct {
		SpcPct *xmlSpcPct `xml:"spcPct"`
	} `xml:"lnSpc"`
	BuChar *struct {
		Char string `xml:"char,attr"`
	} `xml:"buChar"`
	BuNonePresent bool `xml:"buNone"`
}

type xmlParagraph struct {
	PPr  *xmlParagraphProps `xml:"pPr"`
	Runs []xmlRun           `xml:"r"`
}

// xmlCNvPr is the common non-visual props.
type xmlCNvPr struct {
	Name string `xml:"name,attr"`
}

// xmlShape is a p:sp (shape or text box).
type xmlShape struct {
	NVSpPr struct {
		CNvPr   xmlCNvPr `xml:"cNvPr"`
		CNvSpPr struct {
			TxBox bool `xml:"txBox,attr"`
		} `xml:"cNvSpPr"`
	} `xml:"nvSpPr"`
	SpPr xmlSpPr      `xml:"spPr"`
	Body []xmlParagraph `xml:"txBody>p"`
}

// Name returns the cNvPr name.
func (s *xmlShape) Name() string { return s.NVSpPr.CNvPr.Name }
func (s *xmlShape) TxBox() bool  { return s.NVSpPr.CNvSpPr.TxBox }

// xmlPicture is a p:pic.
type xmlPicture struct {
	NVPicPr struct {
		CNvPr xmlCNvPr `xml:"cNvPr"`
	} `xml:"nvPicPr"`
	BlipFill struct {
		SrcRect *struct {
			L int `xml:"l,attr"`
			T int `xml:"t,attr"`
			R int `xml:"r,attr"`
			B int `xml:"b,attr"`
		} `xml:"srcRect"`
	} `xml:"blipFill"`
	SpPr xmlSpPr `xml:"spPr"`
}

func (p *xmlPicture) Name() string { return p.NVPicPr.CNvPr.Name }

// xmlConnector is a p:cxnSp.
type xmlConnector struct {
	NVCxnSpPr struct {
		CNvPr xmlCNvPr `xml:"cNvPr"`
	} `xml:"nvCxnSpPr"`
	SpPr xmlSpPr `xml:"spPr"`
}

func (c *xmlConnector) Name() string { return c.NVCxnSpPr.CNvPr.Name }

// xmlCell is a table cell.
type xmlCell struct {
	Paragraphs []xmlParagraph `xml:"txBody>p"`
	TCPr       *struct {
		GridSpan int  `xml:"gridSpan,attr"`
		RowSpan  int  `xml:"rowSpan,attr"`
		HMerge   bool `xml:"hMerge,attr"`
		VMerge   bool `xml:"vMerge,attr"`
	} `xml:"tcPr"`
}

// xmlTable is the a:tbl inside a graphicFrame.
type xmlTable struct {
	Rows []struct {
		Cells []xmlCell `xml:"tc"`
	} `xml:"tr"`
}

// xmlGraphicFrame hosts a table or chart.
type xmlGraphicFrame struct {
	NVGraphicFramePr struct {
		CNvPr xmlCNvPr `xml:"cNvPr"`
	} `xml:"nvGraphicFramePr"`
	Xfrm      *xmlXfrm `xml:"xfrm"`
	Graphic   struct {
		GraphicData struct {
			Table *xmlTable `xml:"tbl"`
		} `xml:"graphicData"`
	} `xml:"graphic"`
}

func (g *xmlGraphicFrame) Name() string { return g.NVGraphicFramePr.CNvPr.Name }

// xmlSlide is the full p:sld document.
type xmlSlide struct {
	Background *struct {
		BgPr struct {
			Solid *xmlSolidFill `xml:"solidFill"`
			Grad  *xmlGradFill  `xml:"gradFill"`
		} `xml:"bgPr"`
	} `xml:"cSld>bg"`
	SpTree struct {
		Shapes     []xmlShape        `xml:"sp"`
		Pictures   []xmlPicture      `xml:"pic"`
		Connectors []xmlConnector    `xml:"cxnSp"`
		Frames     []xmlGraphicFrame `xml:"graphicFrame"`
	} `xml:"cSld>spTree"`
}

// parseSlide parses one slide part into a model.Slide.
func parseSlide(pkg *opcPackage, slideName string, slideW, slideH float64) (*model.Slide, []string) {
	var warnings []string
	slide := &model.Slide{ID: "s" + strconv.Itoa(slideIndex(slideName)), Layout: model.LayoutTitleContent}

	raw := pkg.files[slideName]
	var doc xmlSlide
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return slide, []string{fmt.Sprintf("slide XML parse failed: %v", err)}
	}

	// Background.
	if doc.Background != nil {
		slide.Background = parseBackground(doc.Background.BgPr.Solid, doc.Background.BgPr.Grad)
	}

	// Speaker notes: notesSlide via rels.
	rels := pkg.relsFor(slideName)
	for _, target := range rels {
		if strings.Contains(target, "notesSlides/") && strings.HasSuffix(target, ".xml") {
			if notesRaw, ok := pkg.files[target]; ok {
				slide.Notes = parseNotes(notesRaw)
			}
		}
	}

	type pendingPic struct {
		pic  xmlPicture
		file string
	}
	var pics []pendingPic
	for _, p := range doc.SpTree.Pictures {
		pics = append(pics, pendingPic{pic: p, file: findEmbedTarget(raw, p.Name(), rels)})
	}

	// Shapes (text boxes + shapes).
	for _, sp := range doc.SpTree.Shapes {
		elem := parseShape(sp, slideW, slideH, &warnings)
		if elem != nil {
			slide.Elements = append(slide.Elements, elem)
		}
	}

	// Connectors.
	for _, cx := range doc.SpTree.Connectors {
		if elem := parseConnector(cx, slideW, slideH); elem != nil {
			slide.Elements = append(slide.Elements, elem)
		} else {
			warnings = append(warnings, "connector without geometry skipped")
		}
	}

	// Graphic frames: tables parsed inline; charts via rel + chart part.
	for _, gf := range doc.SpTree.Frames {
		if gf.Graphic.GraphicData.Table != nil {
			elem := parseTable(gf, slideW, slideH, &warnings)
			if elem != nil {
				slide.Elements = append(slide.Elements, elem)
			}
			continue
		}
		chartTarget := findChartTarget(raw, gf.Name(), rels)
		if chartTarget == "" {
			warnings = append(warnings, fmt.Sprintf("graphic frame %q: no chart relationship, skipped", gf.Name()))
			continue
		}
		chartRaw, ok := pkg.files[chartTarget]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("chart part %q missing", chartTarget))
			continue
		}
		elem := parseChart(chartRaw, gf, slideW, slideH, &warnings)
		if elem != nil {
			slide.Elements = append(slide.Elements, elem)
		}
	}

	// Pictures last (z-order approximation for builder-generated decks).
	for _, p := range pics {
		elem := parsePicture(p.pic, p.file, slideW, slideH, &warnings)
		if elem != nil {
			slide.Elements = append(slide.Elements, elem)
		}
	}

	return slide, warnings
}

// findEmbedTarget extracts the r:embed rel target for the picture with the
// given cNvPr name by string-scanning the slide XML.
func findEmbedTarget(slideRaw []byte, picName string, rels map[string]string) string {
	s := string(slideRaw)
	idx := strings.Index(s, `name="`+picName+`"`)
	if idx < 0 {
		return ""
	}
	rest, ok := strings.CutPrefix(s[idx:], `embed="`)
	// search after the name position
	if !ok {
		j := strings.Index(s[idx:], `embed="`)
		if j < 0 {
			return ""
		}
		rest = s[idx+j+len(`embed="`):]
	}
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rels[rest[:end]]
}

// findChartTarget extracts the chart rel target for a graphicFrame.
func findChartTarget(slideRaw []byte, frameName string, rels map[string]string) string {
	s := string(slideRaw)
	idx := strings.Index(s, `name="`+frameName+`"`)
	if idx < 0 {
		return ""
	}
	after := s[idx:]
	chartIdx := strings.Index(after, "chart")
	if chartIdx < 0 {
		return ""
	}
	idAt := strings.Index(after[chartIdx:], `id="`)
	if idAt < 0 {
		return ""
	}
	rest := after[chartIdx+idAt+len(`id="`):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rels[rest[:end]]
}

// parseBackground converts bg fill nodes.
func parseBackground(solid *xmlSolidFill, grad *xmlGradFill) *model.Background {
	if solid != nil && solid.Color != nil {
		return &model.Background{Type: model.BgSolid, Color: srgbToHex(solid.Color)}
	}
	if grad != nil && len(grad.Stops) > 0 {
		g := &model.Gradient{Type: model.GradientLinear}
		if grad.Lin != nil {
			g.Angle = float64(grad.Lin.Ang) / 60000
		}
		for _, st := range grad.Stops {
			if st.Color != nil {
				g.Stops = append(g.Stops, model.GradientStop{
					Color:    srgbToHex(st.Color),
					Position: float64(st.Pos) / 1000,
				})
			}
		}
		return &model.Background{Type: model.BgGradient, Gradient: g}
	}
	return nil
}

// parseNotes extracts the body placeholder text from a notesSlide.
func parseNotes(raw []byte) string {
	var doc struct {
		Shapes []xmlShape `xml:"cSld>spTree>sp"`
	}
	if xml.Unmarshal(raw, &doc) != nil {
		return ""
	}
	var parts []string
	for _, sp := range doc.Shapes {
		for _, p := range sp.Body {
			if t := paragraphPlainText(p); t != "" {
				parts = append(parts, t)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// ---------- EMU/percent helpers ----------

func emuToPctX(v int, slideW float64) float64 { return float64(v) / 914400 / slideW * 100 }
func emuToPctY(v int, slideH float64) float64 { return float64(v) / 914400 / slideH * 100 }

func srgbToHex(c *xmlSrgbClr) string {
	if c == nil || c.Val == "" {
		return ""
	}
	return "#" + strings.ToUpper(strings.TrimPrefix(c.Val, "#"))
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// rectFromXfrm converts a transform into percentage coordinates.
func rectFromXfrm(x *xmlXfrm, slideW, slideH float64) (model.Rect, float64, bool) {
	if x == nil {
		return model.Rect{}, 0, false
	}
	return model.Rect{
		X: round2(emuToPctX(x.Off.X, slideW)),
		Y: round2(emuToPctY(x.Off.Y, slideH)),
		W: round2(emuToPctX(x.Ext.CX, slideW)),
		H: round2(emuToPctY(x.Ext.CY, slideH)),
	}, float64(x.Rot) / 60000, true
}

// prstToShapeType maps OOXML preset geometry back to model.ShapeType
// (inverse of builder.shapeToPrst).
func prstToShapeType(prst string) model.ShapeType {
	switch prst {
	case "rect":
		return model.ShapeRectangle
	case "roundRect":
		return model.ShapeRoundedRectangle
	case "ellipse":
		return model.ShapeEllipse
	case "triangle":
		return model.ShapeTriangle
	case "diamond":
		return model.ShapeDiamond
	case "line":
		return model.ShapeLine
	case "rightArrow":
		return model.ShapeArrow
	case "leftRightArrow":
		return model.ShapeDoubleArrow
	case "pentagon":
		return model.ShapePentagon
	case "hexagon":
		return model.ShapeHexagon
	case "star5":
		return model.ShapeStar
	case "wedgeRectCallout":
		return model.ShapeCallout
	case "heart":
		return model.ShapeHeart
	case "cloud":
		return model.ShapeCloud
	default:
		return model.ShapeRectangle
	}
}
