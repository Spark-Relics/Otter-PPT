package pptoolkit

import (
	"testing"
)

// TestAddTextInlineWarnings verifies the quality gate fires at add time:
// a tiny box with lots of text should come back with warnings, a clean
// element should not.
func TestAddTextInlineWarnings(t *testing.T) {
	s := NewSession()
	sid := s.AddSlide("blank")

	// Overflowing text in a tiny box
	r := s.ExecuteTool("add_text", map[string]any{
		"slide_id": sid, "x": 5, "y": 5, "w": 20, "h": 4,
		"text":      "这是一段非常长的文本，塞进一个很小的盒子里，必然会发生垂直方向上的溢出，应当触发警告",
		"font_size": 18,
	})
	if !r.Success {
		t.Fatalf("add_text failed: %s", r.Message)
	}
	data, _ := r.Data.(map[string]any)
	if data == nil {
		t.Fatal("add_text result missing data")
	}
	warnings, _ := data["warnings"].([]string)
	if len(warnings) == 0 {
		t.Fatal("expected overflow warnings for cramped text, got none")
	}

	// Clean element: no warnings
	r = s.ExecuteTool("add_title", map[string]any{
		"slide_id": sid, "x": 5, "y": 5, "w": 80, "h": 12,
		"text": "正常标题", "font_size": 32,
	})
	if !r.Success {
		t.Fatalf("add_title failed: %s", r.Message)
	}
	data, _ = r.Data.(map[string]any)
	if data == nil {
		t.Fatal("add_title result missing data")
	}
	if w, _ := data["warnings"].([]string); len(w) != 0 {
		t.Fatalf("clean title should have no warnings, got %v", w)
	}
}
