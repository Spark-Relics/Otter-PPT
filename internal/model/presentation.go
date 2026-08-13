package model

// Rect represents a rectangle in percentage coordinates (0-100).
// Using percentages makes layouts resolution-independent.
// x/y is the top-left corner, w/h is width/height.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// TextStyle defines font properties for a text element.
type TextStyle struct {
	FontSize      int     `json:"font_size,omitempty"` // pt
	FontName      string  `json:"font_name,omitempty"`
	Bold          bool    `json:"bold,omitempty"`
	Italic        bool    `json:"italic,omitempty"`
	Underline     bool    `json:"underline,omitempty"`
	Strike        bool    `json:"strike,omitempty"`
	Color         string  `json:"color,omitempty"`
	Opacity       float64 `json:"opacity,omitempty"`
	Align         string  `json:"align,omitempty"`
	LineSpacing   float64 `json:"line_spacing,omitempty"`
	LetterSpacing float64 `json:"letter_spacing,omitempty"`
	BulletChar    string  `json:"bullet_char,omitempty"`
	VAlign        string  `json:"valign,omitempty"`
	WordWrap      *bool   `json:"word_wrap,omitempty"`
	Shadow        bool    `json:"shadow,omitempty"`
	MarginLeft    float64 `json:"margin_left,omitempty"`   // pt
	MarginRight   float64 `json:"margin_right,omitempty"`  // pt
	MarginTop     float64 `json:"margin_top,omitempty"`    // pt
	MarginBottom  float64 `json:"margin_bottom,omitempty"` // pt
}

// RichTextRun is a single styled text fragment within a paragraph.
type RichTextRun struct {
	Text  string    `json:"text"`
	Style TextStyle `json:"style,omitempty"`
}

// Paragraph is a block of text with optional mixed formatting.
type Paragraph struct {
	Runs        []RichTextRun `json:"runs,omitempty"`
	Text        string        `json:"text,omitempty"`
	Style       TextStyle     `json:"style,omitempty"`
	Level       int           `json:"level,omitempty"`
	SpaceBefore float64       `json:"space_before,omitempty"` // pt
	SpaceAfter  float64       `json:"space_after,omitempty"`  // pt
	Bullet      string        `json:"bullet,omitempty"`
}

// TableCellStyle for individual table cells.
type TableCellStyle struct {
	BgColor  string `json:"bg_color,omitempty"`
	Bold     bool   `json:"bold,omitempty"`
	Align    string `json:"align,omitempty"`
	FontSize int    `json:"font_size,omitempty"`
	Color    string `json:"color,omitempty"`
}

// TableCell with value and optional styling.
type TableCell struct {
	Text    string         `json:"text"`
	Style   TableCellStyle `json:"style,omitempty"`
	ColSpan int            `json:"col_span,omitempty"`
	RowSpan int            `json:"row_span,omitempty"`
}

// TableData holds a table structure with full styling control.
type TableData struct {
	Headers     []TableCell   `json:"headers"`
	Rows        [][]TableCell `json:"rows"`
	HeaderColor string        `json:"header_color,omitempty"` // bg color for header row
	BorderColor string        `json:"border_color,omitempty"`
	AltRowColor string        `json:"alt_row_color,omitempty"` // alternating row color
	FontSize    int           `json:"font_size,omitempty"`
}

// ImageCrop specifies source cropping percentages.
type ImageCrop struct {
	Left   float64 `json:"left,omitempty"`
	Top    float64 `json:"top,omitempty"`
	Right  float64 `json:"right,omitempty"`
	Bottom float64 `json:"bottom,omitempty"`
}

// Element is a single placed object on a slide.
type Element struct {
	ID   string      `json:"id"`
	Type ElementType `json:"type"`

	// Position in percentage coordinates (0-100 relative to slide).
	Rect     Rect    `json:"rect"`
	Rotation float64 `json:"rotation,omitempty"` // degrees

	// Z-order: higher = in front. Auto-assigned by index if 0.
	ZOrder int `json:"z_order,omitempty"`

	// Text content (for text-like elements: title, subtitle, body).
	Text       string      `json:"text,omitempty"`
	Style      TextStyle   `json:"style,omitempty"`
	Paragraphs []Paragraph `json:"paragraphs,omitempty"` // for rich text

	// Image path (for image elements). Can be local path or URL.
	ImagePath string     `json:"image_path,omitempty"`
	ImageFit  string     `json:"image_fit,omitempty"` // contain, cover, stretch
	ImageAlt  string     `json:"image_alt,omitempty"`
	ImageCrop *ImageCrop `json:"image_crop,omitempty"`

	// For bullet lists, each item is a separate string.
	Items []string `json:"items,omitempty"`

	// Table data.
	Table *TableData `json:"table,omitempty"`

	// Shape data.
	Shape *ShapeData `json:"shape,omitempty"`

	// Chart data.
	Chart *ChartData `json:"chart,omitempty"`

	// Connector data.
	Connector *ConnectorData `json:"connector,omitempty"`

	// Animation for this element.
	Animation *Animation `json:"animation,omitempty"`

	// Group children (for grouped elements).
	Children []string `json:"children,omitempty"` // element IDs

	// Opacity (0-1).
	Opacity float64 `json:"opacity,omitempty"`
}

// Slide is a single page in the presentation.
type Slide struct {
	ID         string      `json:"id"`
	Layout     SlideLayout `json:"layout"`
	Background *Background `json:"background,omitempty"`
	Transition *Transition `json:"transition,omitempty"`
	Notes      string      `json:"notes,omitempty"`
	Elements   []*Element  `json:"elements"`
}

// Theme defines the visual style for the entire presentation.
type Theme struct {
	Name            string `json:"name,omitempty"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	AccentColor     string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	TitleFont       string `json:"title_font"`
	BodyFont        string `json:"body_font"`
}

// Presentation is the top-level structure for a complete PPT.
type Presentation struct {
	Title       string   `json:"title"`
	Theme       Theme    `json:"theme"`
	Slides      []*Slide `json:"slides"`
	SlideWidth  float64  `json:"slide_width,omitempty"`
	SlideHeight float64  `json:"slide_height,omitempty"`
}

// DefaultSlideSize returns standard 16:9 dimensions in inches.
func DefaultSlideSize() (w, h float64) {
	return 13.333, 7.5
}

// FindSlide returns the slide with the given ID, or nil.
func (p *Presentation) FindSlide(id string) *Slide {
	for _, s := range p.Slides {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// FindElement returns the element with the given ID on the given slide, or nil.
func (s *Slide) FindElement(id string) *Element {
	for _, e := range s.Elements {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// RemoveElement deletes the element with the given ID. Returns true if found.
func (s *Slide) RemoveElement(id string) bool {
	for i, e := range s.Elements {
		if e.ID == id {
			s.Elements = append(s.Elements[:i], s.Elements[i+1:]...)
			return true
		}
	}
	return false
}
