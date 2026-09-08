package pptoolkit

import (
	"encoding/json"
	"fmt"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// historyLimit caps the undo stack depth (each entry is a JSON snapshot of
// the whole presentation, so a bounded stack keeps memory predictable).
const historyLimit = 50

// mutatingTools lists tool names that change presentation state.
// Only these get an undo checkpoint. Read-only tools (get_state,
// render_slides, get_design_reference) and the history tools themselves
// are excluded so undo steps are meaningful single edits.
var mutatingTools = map[string]bool{
	"set_title":            true,
	"set_theme":            true,
	"set_slide_size":       true,
	"add_slide":            true,
	"delete_slide":         true,
	"duplicate_slide":      true,
	"move_slide":           true,
	"set_notes":            true,
	"set_bg_color":         true,
	"set_bg_gradient":      true,
	"set_bg_image":         true,
	"set_transition":       true,
	"add_title":            true,
	"add_text":             true,
	"add_bullet_list":      true,
	"update_text":          true,
	"update_style":         true,
	"update_position":      true,
	"add_image":            true,
	"add_shape":            true,
	"add_table":            true,
	"add_chart":            true,
	"add_connector":        true,
	"delete_element":       true,
	"set_animation":        true,
	"bring_to_front":       true,
	"send_to_back":         true,
	"set_rotation":         true,
	"set_opacity":          true,
	"group_elements":       true,
	"import_svg":           true,
	"import_pptx":          true,
	"load_template":        true,
	"add_image_generated":  true,
	"generate_image":       true,
	"set_element_position": true,
}

// snapshot serializes the current presentation for the history stack.
// The model is a plain JSON-tagged data tree, so a JSON round-trip is a
// reliable deep copy.
func (s *Session) snapshot() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.Marshal(s.pres)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// pushHistory records a pre-mutation snapshot onto the undo stack.
// Any redo branch is discarded (a new edit invalidates it).
func (s *Session) pushHistory(snap string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.undoStack = append(s.undoStack, snap)
	if len(s.undoStack) > historyLimit {
		s.undoStack = s.undoStack[len(s.undoStack)-historyLimit:]
	}
	s.redoStack = s.redoStack[:0]
}

// restore replaces the live presentation with a stored snapshot.
func (s *Session) restore(snap string) error {
	var pres model.Presentation
	if err := json.Unmarshal([]byte(snap), &pres); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pres = &pres
	return nil
}

// currentSnapshot returns a snapshot of the current state (used to build
// the redo stack when undoing).
func (s *Session) currentSnapshot() (string, error) {
	return s.snapshot()
}

// Undo reverts the last mutating tool call. Returns an error message
// string when there is nothing to undo.
func (s *Session) Undo() error {
	s.mu.Lock()
	if len(s.undoStack) == 0 {
		s.mu.Unlock()
		return fmt.Errorf("nothing to undo")
	}
	snap := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.mu.Unlock()

	cur, err := s.currentSnapshot()
	if err != nil {
		return err
	}
	if err := s.restore(snap); err != nil {
		return err
	}
	s.mu.Lock()
	s.redoStack = append(s.redoStack, cur)
	s.mu.Unlock()
	return nil
}

// Redo re-applies the most recently undone change.
func (s *Session) Redo() error {
	s.mu.Lock()
	if len(s.redoStack) == 0 {
		s.mu.Unlock()
		return fmt.Errorf("nothing to redo")
	}
	snap := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]
	s.mu.Unlock()

	cur, err := s.currentSnapshot()
	if err != nil {
		return err
	}
	if err := s.restore(snap); err != nil {
		return err
	}
	s.mu.Lock()
	s.undoStack = append(s.undoStack, cur)
	s.mu.Unlock()
	return nil
}

// HistoryStatus reports the current undo/redo depth (for tool messages).
func (s *Session) HistoryStatus() (undoDepth, redoDepth int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.undoStack), len(s.redoStack)
}
