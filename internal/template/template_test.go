package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

// buildSamplePPTX round-trips through the real builder so the parser is
// tested against genuine otter-ppt output.
func buildSamplePPTX(t *testing.T) string {
	t.Helper()
	pres := &model.Presentation{
		Theme: model.Theme{
			Name:            "Sample",
			PrimaryColor:    "#1A73E8",
			SecondaryColor:  "#424242",
			AccentColor:     "#FF6D00",
			BackgroundColor: "#FFFFFF",
			TextColor:       "#212121",
			TitleFont:       "Georgia",
			BodyFont:        "Arial",
		},
		Slides: []*model.Slide{{
			ID:     "s1",
			Layout: model.LayoutTitleContent,
			Elements: []*model.Element{
				{ID: "e1", Type: model.ElementTitle, Text: "Hello", Rect: model.Rect{X: 10, Y: 8, W: 80, H: 12}},
			},
		}},
	}
	path := filepath.Join(t.TempDir(), "sample.pptx")
	if err := builder.New(pres).Save(path); err != nil {
		t.Fatalf("build sample pptx: %v", err)
	}
	return path
}

func TestParseFile(t *testing.T) {
	extracted, err := ParseFile(buildSamplePPTX(t))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	if extracted.Theme.PrimaryColor != "#1A73E8" {
		t.Errorf("primary = %s, want #1A73E8", extracted.Theme.PrimaryColor)
	}
	if extracted.Theme.AccentColor != "#FF6D00" {
		t.Errorf("accent = %s, want #FF6D00", extracted.Theme.AccentColor)
	}
	if extracted.Theme.BackgroundColor != "#FFFFFF" {
		t.Errorf("background = %s, want #FFFFFF", extracted.Theme.BackgroundColor)
	}
	if extracted.Theme.TextColor != "#212121" {
		t.Errorf("text = %s, want #212121", extracted.Theme.TextColor)
	}
	if extracted.Theme.TitleFont != "Georgia" {
		t.Errorf("title font = %s, want Georgia", extracted.Theme.TitleFont)
	}
	if extracted.Theme.BodyFont != "Arial" {
		t.Errorf("body font = %s, want Arial", extracted.Theme.BodyFont)
	}

	if extracted.SlideWidth < 13.3 || extracted.SlideWidth > 13.4 {
		t.Errorf("slide width = %f, want ~13.333", extracted.SlideWidth)
	}
	if extracted.SlideHeight < 7.4 || extracted.SlideHeight > 7.6 {
		t.Errorf("slide height = %f, want ~7.5", extracted.SlideHeight)
	}

	if len(extracted.Layouts) != 5 {
		t.Errorf("layout count = %d, want 5 (got %+v)", len(extracted.Layouts), extracted.Layouts)
	}
}

func TestParseBytesMatchesFile(t *testing.T) {
	path := buildSamplePPTX(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	fromBytes, err := ParseBytes(raw)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	fromFile, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if fromBytes.Theme != fromFile.Theme {
		t.Errorf("themes differ: %+v vs %+v", fromBytes.Theme, fromFile.Theme)
	}
	if len(fromBytes.Layouts) != len(fromFile.Layouts) {
		t.Errorf("layout counts differ: %d vs %d", len(fromBytes.Layouts), len(fromFile.Layouts))
	}
}

func TestParseFileRejectsNonPPTX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.pptx")
	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path); err == nil {
		t.Error("expected error for non-zip input")
	}
}

func TestLayoutNum(t *testing.T) {
	cases := map[string]int{
		"slideLayout1.xml":  1,
		"slideLayout9.xml":  9,
		"slideLayout10.xml": 10,
		"weird.xml":         -1,
	}
	for name, want := range cases {
		if got := layoutNum(name); got != want {
			t.Errorf("layoutNum(%s) = %d, want %d", name, got, want)
		}
	}
}

func TestSortLayoutNamesNumeric(t *testing.T) {
	names := []string{"slideLayout10.xml", "slideLayout2.xml", "slideLayout1.xml"}
	sortLayoutNames(names)
	want := []string{"slideLayout1.xml", "slideLayout2.xml", "slideLayout10.xml"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("sort = %v, want %v", names, want)
		}
	}
}
