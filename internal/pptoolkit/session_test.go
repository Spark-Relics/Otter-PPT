package pptoolkit

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

func TestAddCardProducesDesignedCard(t *testing.T) {
	s := NewSession()
	s.SetTheme(model.Theme{
		Name:            "Dark Tech",
		PrimaryColor:    "#22D3EE",
		SecondaryColor:  "#141B2E",
		AccentColor:     "#A78BFA",
		BackgroundColor: "#0A0E1A",
		TextColor:       "#E2E8F0",
		TitleFont:       "Microsoft YaHei",
		BodyFont:        "Microsoft YaHei",
	})
	slideID := s.AddSlide("title_content")
	panelID, titleID, descID, accentID, err := s.AddCard(
		slideID,
		model.Rect{X: 5, Y: 22, W: 28, H: 60},
		"内容易，排版难",
		"LLM 写文字很强，但排成专业 PPT 仍靠人肉拖拽。",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if panelID == "" || titleID == "" || descID == "" || accentID == "" {
		t.Fatalf("expected non-empty element ids, got panel=%q title=%q desc=%q accent=%q",
			panelID, titleID, descID, accentID)
	}

	pres := s.Presentation()
	sl := pres.FindSlide(slideID)
	if len(sl.Elements) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(sl.Elements))
	}

	var buf bytes.Buffer
	if err := builder.New(pres).Write(&buf); err != nil {
		t.Fatalf("build PPTX: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	slideXML := readZipEntry(t, zr, "ppt/slides/slide1.xml")

	for _, want := range []string{
		`prst="roundRect"`,
		"内容易，排版难",
		"LLM 写文字很强",
		"141B2E", // themed panel fill
		"22D3EE", // themed panel border
		"A78BFA", // accent bar
	} {
		if !strings.Contains(slideXML, want) {
			t.Errorf("slide1.xml missing %q", want)
		}
	}
}

func TestApplyShapeDefaultsThemesPanels(t *testing.T) {
	s := NewSession()
	s.SetTheme(model.Theme{
		Name:            "Dark Tech",
		PrimaryColor:    "#22D3EE",
		SecondaryColor:  "#141B2E",
		AccentColor:     "#A78BFA",
		BackgroundColor: "#0A0E1A",
		TextColor:       "#E2E8F0",
		TitleFont:       "Microsoft YaHei",
		BodyFont:        "Microsoft YaHei",
	})
	slideID := s.AddSlide("title_content")

	// No explicit fill/border: should inherit the themed panel look.
	id, err := s.AddShape(slideID, model.Rect{X: 5, Y: 20, W: 30, H: 40}, &model.ShapeData{
		ShapeType: model.ShapeRoundedRectangle,
		Text:      "默认面板",
	})
	if err != nil {
		t.Fatal(err)
	}
	sl := s.Presentation().FindSlide(slideID)
	elem := sl.FindElement(id)
	if elem.Shape.FillColor != "#141B2E" {
		t.Errorf("expected themed panel fill, got %q", elem.Shape.FillColor)
	}
	if elem.Shape.BorderColor != "#22D3EE" {
		t.Errorf("expected themed panel border, got %q", elem.Shape.BorderColor)
	}
}

func readZipEntry(t *testing.T, zr *zip.Reader, name string) string {
	t.Helper()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
	}
	t.Fatalf("zip entry %q not found", name)
	return ""
}
