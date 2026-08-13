// Package builder converts a model.Presentation into an actual .pptx file.
//
// It uses raw OOXML generation (ZIP + XML) rather than relying on a third-party
// library, giving us full control over every feature: shapes, gradients, charts,
// animations, transitions, etc.
//
// The PPTX format is an OPC ZIP package containing XML parts:
//
//	ppt/presentation.xml        — slide list, dimensions, master refs
//	ppt/slides/slide1.xml       — each slide's content
//	ppt/slideLayouts/slideLayout1.xml
//	ppt/slideMasters/slideMaster1.xml
//	ppt/theme/theme1.xml
//	[Content_Types].xml
//	ppt/_rels/presentation.xml.rels
//	…
package builder

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// EMU (English Metric Units) conversion constants.
// PPTX uses EMU internally: 914400 EMU = 1 inch, 9525 EMU = 1 pixel @96dpi.
const (
	EMUPerInch = 914400
)

// Builder renders a Presentation to PPTX.
type Builder struct {
	pres *model.Presentation
}

// New creates a builder for the given presentation.
func New(pres *model.Presentation) *Builder {
	if pres.SlideWidth == 0 || pres.SlideHeight == 0 {
		pres.SlideWidth, pres.SlideHeight = model.DefaultSlideSize()
	}
	return &Builder{pres: pres}
}

// Save writes the presentation to a .pptx file.
func (b *Builder) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	return b.Write(f)
}

// Write writes the PPTX to an io.Writer.
func (b *Builder) Write(w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Write each part
	writers := []func(*zip.Writer) error{
		b.writeContentTypes,
		b.writeRels,
		b.writePresentation,
		b.writePresentationRels,
		b.writeTheme,
		b.writeSlideMaster,
		b.writeSlideMasterRels,
		b.writeSlideLayout,
	}

	for _, fn := range writers {
		if err := fn(zw); err != nil {
			return err
		}
	}

	// Write slides
	for i, slide := range b.pres.Slides {
		if err := b.writeSlide(zw, i+1, slide); err != nil {
			return err
		}
		if err := b.writeSlideRels(zw, i+1); err != nil {
			return err
		}
	}

	return nil
}

// ─────────── Helper conversion functions ───────────

// pctToEMU converts a percentage (0-100) to EMU for the given total dimension (inches).
func pctToEMU(pct, totalInches float64) int64 {
	return int64(pct/100.0 * totalInches * float64(EMUPerInch))
}

// inchToEMU converts inches to EMU.
func inchToEMU(inches float64) int64 {
	return int64(inches * float64(EMUPerInch))
}

// hexToRGB parses "#RRGGBB" to r,g,b.
func hexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return
}

// xmlEscape escapes XML special characters.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// fmtFloat formats a float for XML, trimming trailing zeros.
func fmtFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%.4f", f)
}
