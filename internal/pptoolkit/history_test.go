package pptoolkit

import (
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func mustTool(t *testing.T, s *Session, name string, args map[string]any) ToolResult {
	t.Helper()
	r := s.ExecuteTool(name, args)
	if !r.Success {
		t.Fatalf("%s failed: %s", name, r.Message)
	}
	return r
}

func TestUndoRedoAddSlide(t *testing.T) {
	s := NewSession()
	r := mustTool(t, s, "add_slide", map[string]any{"layout": "blank"})
	sid := r.Data.(map[string]string)["slide_id"]

	mustTool(t, s, "add_text", map[string]any{
		"slide_id": sid, "x": 5, "y": 5, "w": 40, "h": 10, "text": "hello",
	})
	if got := len(s.Presentation().Slides); got != 1 {
		t.Fatalf("slides = %d, want 1", got)
	}

	mustTool(t, s, "undo", nil)
	if got := len(s.Presentation().Slides); got != 1 {
		// add_text undone: slide remains but element gone
		t.Fatalf("after undo slides = %d, want 1", got)
	}
	if got := len(s.Presentation().Slides[0].Elements); got != 0 {
		t.Fatalf("after undo elements = %d, want 0", got)
	}

	mustTool(t, s, "redo", nil)
	if got := len(s.Presentation().Slides[0].Elements); got != 1 {
		t.Fatalf("after redo elements = %d, want 1", got)
	}

	// undo twice: back before add_text and add_slide
	mustTool(t, s, "undo", nil)
	mustTool(t, s, "undo", nil)
	if got := len(s.Presentation().Slides); got != 0 {
		t.Fatalf("after double undo slides = %d, want 0", got)
	}
	mustTool(t, s, "redo", nil)
	if got := len(s.Presentation().Slides); got != 1 {
		t.Fatalf("after redo slides = %d, want 1", got)
	}
}

func TestUndoRestoresElementContent(t *testing.T) {
	s := NewSession()
	r := mustTool(t, s, "add_slide", map[string]any{"layout": "blank"})
	sid := r.Data.(map[string]string)["slide_id"]
	r = mustTool(t, s, "add_text", map[string]any{
		"slide_id": sid, "x": 5, "y": 5, "w": 40, "h": 10, "text": "before",
	})
	elemID := r.Data.(map[string]string)["element_id"]

	mustTool(t, s, "update_text", map[string]any{
		"slide_id": sid, "element_id": elemID, "text": "after",
	})
	if got := s.Presentation().Slides[0].Elements[0].Text; got != "after" {
		t.Fatalf("text = %q, want after", got)
	}

	mustTool(t, s, "undo", nil)
	if got := s.Presentation().Slides[0].Elements[0].Text; got != "before" {
		t.Fatalf("after undo text = %q, want before", got)
	}
}

func TestUndoEmpty(t *testing.T) {
	s := NewSession()
	if r := s.ExecuteTool("undo", nil); r.Success {
		t.Fatal("expected failure on empty undo stack")
	}
	if r := s.ExecuteTool("redo", nil); r.Success {
		t.Fatal("expected failure on empty redo stack")
	}
}

func TestFailedToolDoesNotCheckpoint(t *testing.T) {
	s := NewSession()
	r := s.ExecuteTool("update_text", map[string]any{
		"slide_id": "nope", "element_id": "nope", "text": "x",
	})
	if r.Success {
		t.Fatal("expected failure for unknown slide")
	}
	if r := s.ExecuteTool("undo", nil); r.Success {
		t.Fatal("failed tool must not create an undo step")
	}
}

func TestNewEditClearsRedoBranch(t *testing.T) {
	s := NewSession()
	r := mustTool(t, s, "add_slide", map[string]any{"layout": "blank"})
	sid := r.Data.(map[string]string)["slide_id"]

	// undo add_slide, redo it back, then undo again
	// deep enough history: slide + title
	mustTool(t, s, "add_title", map[string]any{
		"slide_id": sid, "x": 0, "y": 0, "w": 50, "h": 10, "text": "t",
	})
	mustTool(t, s, "undo", nil) // title gone
	mustTool(t, s, "redo", nil) // title back
	mustTool(t, s, "undo", nil) // title gone again, redo branch has it

	// a new edit invalidates the redo branch
	mustTool(t, s, "add_title", map[string]any{
		"slide_id": sid, "x": 0, "y": 0, "w": 50, "h": 10, "text": "t2",
	})
	if r := s.ExecuteTool("redo", nil); r.Success {
		t.Fatal("redo should be unavailable after a new edit")
	}
}

// Direct method calls (used by server preview) still snapshot correctly
// when routed through ExecuteTool; verify the model round-trip fidelity.
func TestSnapshotFidelity(t *testing.T) {
	s := NewSession()
	sid := s.AddSlide("blank")
	style := model.TextStyle{FontSize: 24, Bold: true, Color: "#22D3EE"}
	if _, err := s.AddText(sid, model.Rect{X: 1, Y: 2, W: 30, H: 8}, "鸟 turtle", style); err != nil {
		t.Fatal(err)
	}
	before, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	s.Undo() // no-op on direct calls, but must not corrupt state
	s.restore(before)
	after := s.Presentation()
	if got := after.Slides[0].Elements[0].Text; got != "鸟 turtle" {
		t.Fatalf("text = %q", got)
	}
	if got := after.Slides[0].Elements[0].Style.FontSize; got != 24 {
		t.Fatalf("fontsize = %v, want 24", got)
	}
}
