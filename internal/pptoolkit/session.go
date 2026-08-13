// Package pptoolkit is the core innovation of otter-ppt.
//
// Instead of asking the AI to generate a complete JSON in one shot,
// we expose granular design tools (like PowerPoint's UI operations)
// as function-calling tools. The AI agent calls them iteratively
// to build a presentation with full creative control.
//
// Tool categories:
//   - Presentation: set_theme, set_slide_size
//   - Slide:        add_slide, delete_slide, duplicate_slide, move_slide
//   - Background:   set_bg_color, set_bg_gradient, set_bg_image
//   - Text:         add_text, add_title, add_bullet_list, update_text
//   - Styling:      update_style, update_position
//   - Visual:       add_image, add_shape, add_table, add_chart, add_connector
//   - Effects:      set_transition, set_animation, set_rotation, set_opacity
//   - Layout:       bring_to_front, send_to_back, group_elements
//   - Export:       get_state, export_pptx, done
package pptoolkit

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

// Session holds the in-progress presentation state.
// It is the "canvas" that AI tools operate on.
type Session struct {
	mu   sync.Mutex
	pres *model.Presentation
}

// NewSession creates a new empty session.
func NewSession() *Session {
	w, h := model.DefaultSlideSize()
	return &Session{
		pres: &model.Presentation{
			Theme: model.Theme{
				Name:            "Modern Editorial",
				PrimaryColor:    "#2563EB",
				SecondaryColor:  "#0F172A",
				AccentColor:     "#F59E0B",
				BackgroundColor: "#F8FAFC",
				TextColor:       "#0F172A",
				TitleFont:       "Microsoft YaHei UI",
				BodyFont:        "Microsoft YaHei UI",
			},
			SlideWidth:  w,
			SlideHeight: h,
		},
	}
}

// NewSessionFromPresentation wraps an existing presentation.
func NewSessionFromPresentation(pres *model.Presentation) *Session {
	if pres == nil {
		return NewSession()
	}
	if pres.SlideWidth == 0 || pres.SlideHeight == 0 {
		pres.SlideWidth, pres.SlideHeight = model.DefaultSlideSize()
	}
	return &Session{pres: pres}
}

// Presentation returns the current presentation state.
func (s *Session) Presentation() *model.Presentation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pres
}

// genID generates a short unique ID.
func genID() string {
	return uuid.New().String()[:8]
}

// ============================================================
// Presentation-level tools
// ============================================================

// SetTitle sets the presentation title.
func (s *Session) SetTitle(title string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pres.Title = title
	return fmt.Sprintf("Presentation title set to: %s", title)
}

// SetTheme sets the color scheme and fonts.
func (s *Session) SetTheme(theme model.Theme) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pres.Theme = theme
	return fmt.Sprintf("Theme '%s' applied: primary=%s, accent=%s",
		theme.Name, theme.PrimaryColor, theme.AccentColor)
}

// SetSlideSize sets slide dimensions in inches.
func (s *Session) SetSlideSize(w, h float64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pres.SlideWidth = w
	s.pres.SlideHeight = h
	return fmt.Sprintf("Slide size set to %.3f x %.3f inches", w, h)
}

// ============================================================
// Slide tools
// ============================================================

// AddSlide adds a new slide and returns its ID.
func (s *Session) AddSlide(layout string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	slide := &model.Slide{
		ID:     genID(),
		Layout: model.SlideLayout(layout),
	}
	if slide.Layout == "" {
		slide.Layout = model.LayoutTitleContent
	}
	s.pres.Slides = append(s.pres.Slides, slide)
	return slide.ID
}

// DeleteSlide removes a slide by ID.
func (s *Session) DeleteSlide(slideID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sl := range s.pres.Slides {
		if sl.ID == slideID {
			s.pres.Slides = append(s.pres.Slides[:i], s.pres.Slides[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("slide not found: %s", slideID)
}

// DuplicateSlide copies a slide and inserts it right after the original.
func (s *Session) DuplicateSlide(slideID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sl := range s.pres.Slides {
		if sl.ID == slideID {
			// Deep copy via re-marshal
			// For simplicity, create a shallow copy with new IDs
			newSlide := &model.Slide{
				ID:         genID(),
				Layout:     sl.Layout,
				Background: sl.Background,
				Transition: sl.Transition,
				Notes:      sl.Notes,
			}
			for _, elem := range sl.Elements {
				cp := *elem
				cp.ID = genID()
				newSlide.Elements = append(newSlide.Elements, &cp)
			}
			// Insert after original
			tail := append([]*model.Slide{newSlide}, s.pres.Slides[i+1:]...)
			s.pres.Slides = append(s.pres.Slides[:i+1], tail...)
			return newSlide.ID, nil
		}
	}
	return "", fmt.Errorf("slide not found: %s", slideID)
}

// MoveSlide reorders a slide to a new position (0-based).
func (s *Session) MoveSlide(slideID string, newIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldIndex := -1
	for i, sl := range s.pres.Slides {
		if sl.ID == slideID {
			oldIndex = i
			break
		}
	}
	if oldIndex == -1 {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	if newIndex < 0 || newIndex >= len(s.pres.Slides) {
		return fmt.Errorf("invalid index %d (0-%d)", newIndex, len(s.pres.Slides)-1)
	}
	sl := s.pres.Slides[oldIndex]
	s.pres.Slides = append(s.pres.Slides[:oldIndex], s.pres.Slides[oldIndex+1:]...)
	s.pres.Slides = append(s.pres.Slides[:newIndex], append([]*model.Slide{sl}, s.pres.Slides[newIndex:]...)...)
	return nil
}

// SetSlideBackground sets the background of a slide.
func (s *Session) SetSlideBackground(slideID string, bg *model.Background) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	sl.Background = bg
	return nil
}

// SetSlideTransition sets the transition for a slide.
func (s *Session) SetSlideTransition(slideID string, t *model.Transition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	sl.Transition = t
	return nil
}

// SetSlideNotes sets speaker notes for a slide.
func (s *Session) SetSlideNotes(slideID, notes string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	sl.Notes = notes
	return nil
}

// ============================================================
// Element tools
// ============================================================

// addElement is the internal helper for adding any element.
func (s *Session) addElement(slideID string, elem *model.Element) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return "", fmt.Errorf("slide not found: %s", slideID)
	}
	if elem.ID == "" {
		elem.ID = genID()
	}
	elem.ZOrder = len(sl.Elements) + 1
	sl.Elements = append(sl.Elements, elem)
	return elem.ID, nil
}

// AddText adds a text box to a slide.
func (s *Session) AddText(slideID string, rect model.Rect, text string, style model.TextStyle) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:  model.ElementBody,
		Rect:  rect,
		Text:  text,
		Style: style,
	})
}

// AddTitle adds a title text element.
func (s *Session) AddTitle(slideID string, rect model.Rect, text string, style model.TextStyle) (string, error) {
	if style.FontSize == 0 {
		style.FontSize = 36
	}
	style.Bold = true
	return s.addElement(slideID, &model.Element{
		Type:  model.ElementTitle,
		Rect:  rect,
		Text:  text,
		Style: style,
	})
}

// AddBulletList adds a bullet list element.
func (s *Session) AddBulletList(slideID string, rect model.Rect, items []string, style model.TextStyle) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:  model.ElementBullet,
		Rect:  rect,
		Items: items,
		Style: style,
	})
}

// AddImage adds an image element.
func (s *Session) AddImage(slideID string, rect model.Rect, imagePath string) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:      model.ElementImage,
		Rect:      rect,
		ImagePath: imagePath,
	})
}

// AddShape adds a shape element.
func (s *Session) AddShape(slideID string, rect model.Rect, shape *model.ShapeData) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:  model.ElementShape,
		Rect:  rect,
		Shape: shape,
	})
}

// AddTable adds a table element.
func (s *Session) AddTable(slideID string, rect model.Rect, table *model.TableData) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:  model.ElementTable,
		Rect:  rect,
		Table: table,
	})
}

// AddChart adds a chart element.
func (s *Session) AddChart(slideID string, rect model.Rect, chart *model.ChartData) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:  model.ElementChart,
		Rect:  rect,
		Chart: chart,
	})
}

// AddConnector adds a connector line/arrow.
func (s *Session) AddConnector(slideID string, conn *model.ConnectorData) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:      model.ElementConnector,
		Rect:      model.Rect{}, // computed from connector points
		Connector: conn,
	})
}

// ============================================================
// Element manipulation tools
// ============================================================

// UpdateText updates text content of an element.
func (s *Session) UpdateText(slideID, elemID, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	elem := sl.FindElement(elemID)
	if elem == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	elem.Text = text
	return nil
}

// UpdateStyle updates the style of an element.
func (s *Session) UpdateStyle(slideID, elemID string, style model.TextStyle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	elem := sl.FindElement(elemID)
	if elem == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	elem.Style = style
	return nil
}

// UpdatePosition updates the rect/rotation of an element.
func (s *Session) UpdatePosition(slideID, elemID string, rect model.Rect, rotation float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	elem := sl.FindElement(elemID)
	if elem == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	elem.Rect = rect
	elem.Rotation = rotation
	return nil
}

// DeleteElement removes an element from a slide.
func (s *Session) DeleteElement(slideID, elemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	if !sl.RemoveElement(elemID) {
		return fmt.Errorf("element not found: %s", elemID)
	}
	return nil
}

// SetElementAnimation sets animation for an element.
func (s *Session) SetElementAnimation(slideID, elemID string, anim *model.Animation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	elem := sl.FindElement(elemID)
	if elem == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	elem.Animation = anim
	return nil
}

// BringToFront moves an element to the highest z-order.
func (s *Session) BringToFront(slideID, elemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	maxZ := 0
	var target *model.Element
	for _, e := range sl.Elements {
		if e.ZOrder > maxZ {
			maxZ = e.ZOrder
		}
		if e.ID == elemID {
			target = e
		}
	}
	if target == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	target.ZOrder = maxZ + 1
	return nil
}

// SendToBack moves an element to the lowest z-order.
func (s *Session) SendToBack(slideID, elemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	minZ := 999
	var target *model.Element
	for _, e := range sl.Elements {
		if e.ZOrder < minZ {
			minZ = e.ZOrder
		}
		if e.ID == elemID {
			target = e
		}
	}
	if target == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	target.ZOrder = minZ - 1
	return nil
}

// SetOpacity sets element opacity (0-1).
func (s *Session) SetOpacity(slideID, elemID string, opacity float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	elem := sl.FindElement(elemID)
	if elem == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	elem.Opacity = opacity
	return nil
}

// SetRotation sets element rotation in degrees.
func (s *Session) SetRotation(slideID, elemID string, degrees float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sl := s.pres.FindSlide(slideID)
	if sl == nil {
		return fmt.Errorf("slide not found: %s", slideID)
	}
	elem := sl.FindElement(elemID)
	if elem == nil {
		return fmt.Errorf("element not found: %s", elemID)
	}
	elem.Rotation = degrees
	return nil
}

// GroupElements creates a group element referencing child IDs.
func (s *Session) GroupElements(slideID string, elemIDs []string) (string, error) {
	return s.addElement(slideID, &model.Element{
		Type:     model.ElementGroup,
		Children: elemIDs,
	})
}
