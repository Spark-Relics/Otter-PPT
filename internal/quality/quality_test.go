package quality

import (
	"strings"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func woff() *bool {
	v := false
	return &v
}

func TestEstimateTextWidth(t *testing.T) {
	// CJK chars are 1em wide: 8 CJK chars at 18pt ≈ 144pt
	w := EstimateTextWidth("哈哈哈哈哈哈哈哈", 18)
	if w < 140 || w > 148 {
		t.Errorf("CJK width estimate = %.1f, want ~144", w)
	}
	// Latin is narrower than CJK for the same count
	if EstimateTextWidth("abcdefgh", 18) >= w {
		t.Error("latin should estimate narrower than CJK")
	}
}

func TestOutOfBounds(t *testing.T) {
	s := &model.Slide{Elements: []*model.Element{
		{ID: "ok", Type: model.ElementShape, Rect: model.Rect{X: 10, Y: 10, W: 50, H: 50}},
		{ID: "oob", Type: model.ElementTitle, Rect: model.Rect{X: 80, Y: 80, W: 30, H: 30}, Text: "outside"},
	}}
	issues := CheckSlide(s, 1, 10, 7.5)
	found := false
	for _, i := range issues {
		if i.ElementID == "oob" && i.Kind == KindOutOfBounds {
			found = true
			if i.Severity != SeverityError {
				t.Error("text element out of bounds should be error")
			}
		}
	}
	if !found {
		t.Error("expected out_of_bounds issue for 'oob'")
	}
	// A decorative shape bleeding off-canvas is only a warning.
	s.Elements = []*model.Element{{ID: "deco", Type: model.ElementShape, Rect: model.Rect{X: 90, Y: 90, W: 20, H: 20}}}
	for _, i := range CheckSlide(s, 1, 10, 7.5) {
		if i.Kind == KindOutOfBounds && i.Severity != SeverityWarn {
			t.Error("decorative bleed should be warn, not error")
		}
	}
}

func TestOverflowVertical(t *testing.T) {
	// Box: 20% of 7.5in tall = 108pt. 40 lines of text at 14pt won't fit.
	items := make([]string, 40)
	for i := range items {
		items[i] = "项目条目文本"
	}
	s := &model.Slide{Elements: []*model.Element{
		{ID: "list", Type: model.ElementBody, Rect: model.Rect{X: 5, Y: 10, W: 40, H: 20}, Style: model.TextStyle{FontSize: 14}, Items: items},
	}}
	var found *Issue
	for i, is := range CheckSlide(s, 1, 10, 7.5) {
		if is.Kind == KindOverflowV {
			found = &CheckSlide(s, 1, 10, 7.5)[i]
		}
	}
	if found == nil {
		t.Fatal("expected vertical overflow for 40 items in 20% box")
	}
	if found.Severity != SeverityError {
		t.Errorf("40 overflowing lines should be error severity, got %s", found.Severity)
	}
}

func TestOverflowHorizontalNoWrap(t *testing.T) {
	long := strings.Repeat("很长很长的中文标题", 10) // ~80 CJK chars
	s := &model.Slide{Elements: []*model.Element{
		{ID: "nowrap", Type: model.ElementTitle, Rect: model.Rect{X: 10, Y: 10, W: 20, H: 8},
			Style: model.TextStyle{FontSize: 18, WordWrap: woff()}, Text: long},
	}}
	for _, i := range CheckSlide(s, 1, 10, 7.5) {
		if i.Kind == KindOverflowH && i.Severity == SeverityError {
			return // expected
		}
	}
	t.Error("expected horizontal overflow error for unwrapped long text")
}

func TestTinyText(t *testing.T) {
	s := &model.Slide{Elements: []*model.Element{
		{ID: "tiny", Type: model.ElementTitle, Rect: model.Rect{X: 10, Y: 10, W: 30, H: 5},
			Style: model.TextStyle{FontSize: 7}, Text: "too small"},
	}}
	errs := 0
	for _, i := range CheckSlide(s, 1, 10, 7.5) {
		if i.Kind == KindTinyText && i.Severity == SeverityError {
			errs++
		}
	}
	if errs != 1 {
		t.Errorf("expected 1 tiny_text error, got %d", errs)
	}
}

func TestCoverGate(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{
		{Elements: []*model.Element{
			{ID: "t", Type: model.ElementTitle, Rect: model.Rect{X: 5, Y: 30, W: 90, H: 20},
				Style: model.TextStyle{FontSize: 18}, Text: "weak cover"},
		}},
	}}
	issues := CheckCover(pres)
	if len(issues) == 0 || issues[0].Kind != KindCoverGate || issues[0].Severity != SeverityError {
		t.Fatalf("expected cover gate error, got %+v", issues)
	}
	// Now with a hero title it must pass.
	pres.Slides[0].Elements[0].Style.FontSize = 40
	if issues := CheckCover(pres); len(issues) != 0 {
		t.Errorf("cover with 40pt hero should pass, got %+v", issues)
	}
}

func TestCheckReport(t *testing.T) {
	pres := &model.Presentation{Slides: []*model.Slide{
		{Elements: []*model.Element{
			{ID: "hero", Type: model.ElementTitle, Rect: model.Rect{X: 5, Y: 30, W: 90, H: 20},
				Style: model.TextStyle{FontSize: 44}, Text: "标题"},
		}},
	}}
	r := Check(pres)
	if !r.Pass || r.Score < 90 {
		t.Errorf("clean deck should pass with high score, got pass=%v score=%d issues=%+v", r.Pass, r.Score, r.Issues)
	}
	if !strings.Contains(r.Summary(), "PASS") {
		t.Errorf("summary should say PASS, got %q", r.Summary())
	}
}
