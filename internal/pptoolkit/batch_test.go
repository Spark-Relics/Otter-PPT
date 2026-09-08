package pptoolkit

import "testing"

func TestExecuteBatchAtomicRollback(t *testing.T) {
	s := NewSessionFromPresentation(nil)
	before := len(s.Presentation().Slides)

	res := s.ExecuteBatch([]ToolCall{
		{Name: "add_slide", Arguments: map[string]any{"layout": "blank"}},
		{Name: "add_text", Arguments: map[string]any{"slide_id": "nope", "text": "x"}},
	})
	if res.AllSuccess {
		t.Fatalf("expected batch failure, got success")
	}
	if res.FailedIndex != 1 {
		t.Fatalf("FailedIndex = %d, want 1", res.FailedIndex)
	}
	if !res.RolledBack {
		t.Fatalf("expected RolledBack=true")
	}
	if got := len(s.Presentation().Slides); got != before {
		t.Fatalf("slide count after rollback = %d, want %d (atomicity violated)", got, before)
	}
}

func TestExecuteBatchSuccessSingleUndoStep(t *testing.T) {
	s := NewSessionFromPresentation(nil)
	before := len(s.Presentation().Slides)

	res := s.ExecuteBatch([]ToolCall{
		{Name: "add_slide", Arguments: map[string]any{"layout": "blank"}},
		{Name: "add_slide", Arguments: map[string]any{"layout": "title"}},
	})
	if !res.AllSuccess {
		t.Fatalf("expected batch success, got failure at %d: %+v", res.FailedIndex, res.Results)
	}
	if got := len(s.Presentation().Slides); got != before+2 {
		t.Fatalf("slide count = %d, want %d", got, before+2)
	}

	undoDepth, _ := s.HistoryStatus()
	if undoDepth != 1 {
		t.Fatalf("undo depth = %d, want 1 (batch should be a single undo step)", undoDepth)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if got := len(s.Presentation().Slides); got != before {
		t.Fatalf("slide count after undo = %d, want %d", got, before)
	}
}

func TestExecuteBatchEmpty(t *testing.T) {
	s := NewSessionFromPresentation(nil)
	res := s.ExecuteBatch(nil)
	if !res.AllSuccess || res.FailedIndex != -1 || len(res.Results) != 0 {
		t.Fatalf("unexpected empty batch result: %+v", res)
	}
}
