package model

// FillStyle defines a reusable solid or gradient DrawingML fill.
type FillStyle struct {
	Color    string    `json:"color,omitempty"`
	Gradient *Gradient `json:"gradient,omitempty"`
	Opacity  float64   `json:"opacity,omitempty"` // 0-1, 0 means opaque for compatibility
}

// LineStyle defines shape and connector outlines.
type LineStyle struct {
	Color      string  `json:"color,omitempty"`
	Width      float64 `json:"width,omitempty"` // pt
	Opacity    float64 `json:"opacity,omitempty"`
	Dash       string  `json:"dash,omitempty"` // solid, dash, dot, dash_dot
	BeginArrow string  `json:"begin_arrow,omitempty"`
	EndArrow   string  `json:"end_arrow,omitempty"`
}

// ShadowStyle defines an outer shadow.
type ShadowStyle struct {
	Color    string  `json:"color,omitempty"`
	Opacity  float64 `json:"opacity,omitempty"`
	Blur     float64 `json:"blur,omitempty"`     // pt
	Distance float64 `json:"distance,omitempty"` // pt
	Angle    float64 `json:"angle,omitempty"`    // degrees
}

// ShapeData holds shape-specific properties.
type ShapeData struct {
	ShapeType    ShapeType    `json:"shape_type"`
	FillColor    string       `json:"fill_color,omitempty"`   // legacy solid fill
	BorderColor  string       `json:"border_color,omitempty"` // legacy line color
	BorderWidth  float64      `json:"border_width,omitempty"` // legacy line width, pt
	Fill         *FillStyle   `json:"fill,omitempty"`
	Line         *LineStyle   `json:"line,omitempty"`
	Shadow       *ShadowStyle `json:"shadow,omitempty"`
	Text         string       `json:"text,omitempty"`
	Style        TextStyle    `json:"style,omitempty"`
	CornerRadius float64      `json:"corner_radius,omitempty"`
}

// ChartSeries is one data series in a chart.
type ChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"` // hex
}

// ChartData holds chart-specific properties.
type ChartData struct {
	ChartType  ChartType     `json:"chart_type"`
	Categories []string      `json:"categories"`
	Series     []ChartSeries `json:"series"`
	Title      string        `json:"title,omitempty"`
	ShowLegend bool          `json:"show_legend,omitempty"`
}

// ConnectorData defines a line/arrow connecting two points.
type ConnectorData struct {
	ConnectorType ShapeType `json:"connector_type"` // line, arrow, double_arrow
	Color         string    `json:"color"`
	Width         float64   `json:"width"` // pt
	// Start/End in percentage coords (separate from Rect for flexibility).
	StartX float64 `json:"start_x"`
	StartY float64 `json:"start_y"`
	EndX   float64 `json:"end_x"`
	EndY   float64 `json:"end_y"`
}
