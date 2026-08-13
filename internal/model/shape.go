package model

// ShapeData holds shape-specific properties.
type ShapeData struct {
	ShapeType   ShapeType `json:"shape_type"`
	FillColor   string    `json:"fill_color,omitempty"`   // hex, empty = no fill
	BorderColor string    `json:"border_color,omitempty"` // hex
	BorderWidth float64   `json:"border_width,omitempty"` // pt
	// Optional text inside the shape.
	Text  string    `json:"text,omitempty"`
	Style TextStyle `json:"style,omitempty"`
	// Corner radius for rounded rectangle (0-1 relative).
	CornerRadius float64 `json:"corner_radius,omitempty"`
}

// ChartSeries is one data series in a chart.
type ChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"` // hex
}

// ChartData holds chart-specific properties.
type ChartData struct {
	ChartType ChartType     `json:"chart_type"`
	Categories []string     `json:"categories"`
	Series     []ChartSeries `json:"series"`
	Title      string       `json:"title,omitempty"`
	ShowLegend bool         `json:"show_legend,omitempty"`
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
