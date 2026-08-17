// Package renderer converts PPTX slides into images for AI visual evaluation.
//
// It uses LibreOffice headless to convert PPTX → PDF → PNG.
// When LibreOffice is unavailable, it falls back to a rich structural
// description so the vision pipeline can still provide feedback.
package renderer

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/otter-ppt/otter-ppt/internal/model"
)

// SlideImage holds the rendered image and metadata for one slide.
type SlideImage struct {
	SlideNum int    // 1-based
	Path     string // local path to PNG, empty if using fallback
	Base64   string // base64-encoded PNG data
	Width    int
	Height   int
	// FallbackDescription is populated when rendering isn't available.
	FallbackDescription string
}

// Renderer converts slides into images.
type Renderer struct {
	// LibreOffice binary path (auto-detected if empty).
	libreOfficePath string
	// pdftoppm binary path (auto-detected if empty).
	pdftoppmPath string
	// Headless-capable browser for the HTML rendering path (auto-detected).
	browser *BrowserBackend
	// Temp directory for intermediate files.
	tempDir string
}

// NewRenderer creates a renderer that auto-detects tooling.
func NewRenderer() *Renderer {
	soffice := findExecutable("soffice", "libreoffice")
	pdftoppm := findExecutable("pdftoppm", "")
	return &Renderer{
		libreOfficePath: soffice,
		pdftoppmPath:    pdftoppm,
		browser:         FindBrowser(),
	}
}

// EnsureTooling checks if LibreOffice is available. If not, downloads and
// installs it automatically. Call this before rendering for zero-setup UX.
func (r *Renderer) EnsureTooling() error {
	if r.IsAvailable() {
		return nil
	}
	soffice, pdftoppm, err := EnsureLibreOffice()
	if err != nil {
		return err
	}
	if soffice != "" {
		r.libreOfficePath = soffice
	}
	if pdftoppm != "" {
		r.pdftoppmPath = pdftoppm
	}
	return nil
}

// IsAvailable returns true if LibreOffice rendering is available.
func (r *Renderer) IsAvailable() bool {
	return r.libreOfficePath != "" && r.pdftoppmPath != ""
}

// RenderPresentation takes a PPTX file path and returns slide images.
// Rendering priority: LibreOffice (highest fidelity, if already installed)
// → HTML + headless browser screenshot (zero-download, fast) → Go structural
// fallback (always available).
func (r *Renderer) RenderPresentation(pptxPath string, pres *model.Presentation) ([]SlideImage, error) {
	if !r.IsAvailable() {
		// Only auto-download LibreOffice when there is no browser path —
		// the HTML renderer covers the zero-setup case without a 350MB download.
		if r.browser == nil {
			r.EnsureTooling()
		}
	}
	if r.IsAvailable() {
		return r.renderWithLibreOffice(pptxPath, pres)
	}
	if r.browser != nil && pres != nil {
		images, err := r.renderWithBrowser(pres)
		if err == nil && len(images) > 0 {
			return images, nil
		}
	}
	return r.renderStructuralFallback(pres), nil
}

// renderWithBrowser generates HTML from the presentation model and
// screenshots it with the detected headless browser.
func (r *Renderer) renderWithBrowser(pres *model.Presentation) ([]SlideImage, error) {
	tempDir, err := os.MkdirTemp("", "otter-html-*")
	if err != nil {
		return nil, err
	}
	r.tempDir = tempDir

	htmlPath := filepath.Join(tempDir, "slides.html")
	if err := GenerateHTML(pres, htmlPath); err != nil {
		return nil, fmt.Errorf("generate html: %w", err)
	}

	wIn, hIn := defaultSlideW, defaultSlideH
	if pres.SlideWidth > 0 && pres.SlideHeight > 0 {
		wIn, hIn = pres.SlideWidth, pres.SlideHeight
	}
	// Render at ~120 DPI for crisp text, capped so the long edge ≤ 1920px.
	wPx := int(math.Round(wIn * 120))
	hPx := int(math.Round(hIn * 120))
	longEdge := math.Max(float64(wPx), float64(hPx))
	if longEdge > 1920 {
		scale := 1920.0 / float64(longEdge)
		wPx = int(float64(wPx) * scale)
		hPx = int(float64(hPx) * scale)
	}

	var images []SlideImage
	for i := range pres.Slides {
		slideHTMLPath := filepath.Join(tempDir, fmt.Sprintf("slide-%d.html", i+1))
		if err := writeSingleSlideHTML(pres, i, htmlPath, slideHTMLPath, wPx, hPx); err != nil {
			return nil, err
		}
		pngPath := filepath.Join(tempDir, fmt.Sprintf("slide-%02d.png", i+1))
		if err := r.browser.ScreenshotHTML(slideHTMLPath, wPx, hPx, pngPath); err != nil {
			return nil, fmt.Errorf("slide %d: %w", i+1, err)
		}
		img, err := imaging.Open(pngPath)
		if err != nil {
			continue
		}
		b64, err := fileToBase64(pngPath)
		if err != nil {
			continue
		}
		images = append(images, SlideImage{
			SlideNum: i + 1,
			Path:     pngPath,
			Base64:   b64,
			Width:    img.Bounds().Dx(),
			Height:   img.Bounds().Dy(),
		})
	}
	return images, nil
}

// writeSingleSlideHTML writes a wrapper HTML page that shows exactly one
// slide (via CSS isolation of the Nth .slide) scaled to fill the viewport.
func writeSingleSlideHTML(pres *model.Presentation, slideIdx int, allHTMLPath, outPath string, wPx, hPx int) error {
	// Re-generate per-slide HTML: cheaper than post-processing the full doc.
	g := &htmlGenerator{pres: pres, imgCache: map[string]string{}}
	g.slideWIn, g.slideHIn = defaultSlideW, defaultSlideH
	if pres.SlideWidth > 0 && pres.SlideHeight > 0 {
		g.slideWIn, g.slideHIn = pres.SlideWidth, pres.SlideHeight
	}
	slide := pres.Slides[slideIdx]

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"UTF-8\">\n<style>\n")
	sb.WriteString("* { margin:0; padding:0; box-sizing:border-box; }\n")
	sb.WriteString("html, body { width:" + fmt.Sprintf("%dpx", wPx) + "; height:" + fmt.Sprintf("%dpx", hPx) + "; overflow:hidden; }\n")
	sb.WriteString(fmt.Sprintf(
		".stage { width:%.4fin; height:%.4fin; transform:scale(%.6f); transform-origin:0 0; position:absolute; top:0; left:0; }\n",
		g.slideWIn, g.slideHIn, float64(wPx)/(g.slideWIn*htmlDPI)))
	sb.WriteString(fmt.Sprintf(
		".slide { position:relative; width:%.4fin; height:%.4fin; overflow:hidden; background:#fff; }\n",
		g.slideWIn, g.slideHIn))
	sb.WriteString(".el { position:absolute; }\n")
	sb.WriteString(".text-el { display:flex; flex-direction:column; }\n")
	sb.WriteString("</style>\n</head>\n<body>\n<div class=\"stage\">\n")
	sb.WriteString(slideBackgroundStyle(g, slide))
	sb.WriteString(g.slideBody(slide))
	sb.WriteString("</div>\n</body>\n</html>\n")
	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

// slideBackgroundStyle returns a <style> block for the slide background
// applying to .slide (used by the single-slide wrapper).
func slideBackgroundStyle(g *htmlGenerator, slide *model.Slide) string {
	bg := slide.Background
	if bg == nil {
		if g.pres.Theme.BackgroundColor != "" {
			return fmt.Sprintf("<style>.slide { background: %s; }</style>\n", cssColor(g.pres.Theme.BackgroundColor))
		}
		return ""
	}
	switch bg.Type {
	case model.BgSolid:
		return fmt.Sprintf("<style>.slide { background: %s; }</style>\n", cssColor(bg.Color))
	case model.BgGradient:
		if bg.Gradient != nil && len(bg.Gradient.Stops) > 0 {
			return fmt.Sprintf("<style>.slide { background: %s; }</style>\n", cssGradient(bg.Gradient))
		}
	case model.BgImage:
		if uri := g.dataURI(bg.ImagePath); uri != "" {
			return fmt.Sprintf("<style>.slide { background: url(%s) center/cover no-repeat; }</style>\n", uri)
		}
	}
	return ""
}

// renderWithLibreOffice uses LibreOffice + pdftoppm to generate PNGs.
func (r *Renderer) renderWithLibreOffice(pptxPath string, pres *model.Presentation) ([]SlideImage, error) {
	tempDir, err := os.MkdirTemp("", "otter-render-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	r.tempDir = tempDir

	// Step 1: PPTX → PDF via LibreOffice
	pdfDir, err := os.MkdirTemp("", "otter-pdf-*")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(r.libreOfficePath,
		"--headless", "--convert-to", "pdf",
		"--outdir", pdfDir, pptxPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return r.renderStructuralFallback(pres), fmt.Errorf("libreoffice convert failed: %v: %s", err, output)
	}

	baseName := strings.TrimSuffix(filepath.Base(pptxPath), filepath.Ext(pptxPath))
	pdfPath := filepath.Join(pdfDir, baseName+".pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		return r.renderStructuralFallback(pres), fmt.Errorf("PDF not found after conversion: %w", err)
	}

	// Step 2: PDF → PNG via pdftoppm
	pngPrefix := filepath.Join(tempDir, "slide")
	cmd = exec.Command(r.pdftoppmPath,
		"-png", "-r", "150",
		pdfPath, pngPrefix)
	if output, err := cmd.CombinedOutput(); err != nil {
		return r.renderStructuralFallback(pres), fmt.Errorf("pdftoppm failed: %v: %s", err, output)
	}

	// Collect generated PNGs
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, err
	}

	var images []SlideImage
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		imgPath := filepath.Join(tempDir, entry.Name())
		img, err := imaging.Open(imgPath)
		if err != nil {
			continue
		}
		b64, err := fileToBase64(imgPath)
		if err != nil {
			continue
		}
		// Extract slide number from filename (slide-01.png → 1)
		num := extractSlideNumber(entry.Name())

		// Resize for API if too large
		bounds := img.Bounds()
		if bounds.Dx() > 1920 {
			img = imaging.Resize(img, 1920, 0, imaging.Lanczos)
			resizedPath := filepath.Join(tempDir, fmt.Sprintf("resized_%s", entry.Name()))
			if err := imaging.Save(img, resizedPath); err == nil {
				b64, _ = fileToBase64(resizedPath)
			}
		}

		images = append(images, SlideImage{
			SlideNum: num,
			Path:     imgPath,
			Base64:   b64,
			Width:    img.Bounds().Dx(),
			Height:   img.Bounds().Dy(),
		})
	}

	// Sort by slide number
	for i := 0; i < len(images); i++ {
		for j := i + 1; j < len(images); j++ {
			if images[i].SlideNum > images[j].SlideNum {
				images[i], images[j] = images[j], images[i]
			}
		}
	}

	return images, nil
}

// renderStructuralFallback generates slide "previews" using Go image rendering
// when LibreOffice is unavailable. Produces simplified visual representations.
func (r *Renderer) renderStructuralFallback(pres *model.Presentation) []SlideImage {
	var images []SlideImage
	for i, slide := range pres.Slides {
		img := renderSlidePreview(slide, pres)
		var buf strings.Builder
		if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
			continue
		}
		b64 := base64.StdEncoding.EncodeToString([]byte(buf.String()))
		images = append(images, SlideImage{
			SlideNum:  i + 1,
			Base64:    b64,
			Width:     img.Bounds().Dx(),
			Height:    img.Bounds().Dy(),
			FallbackDescription: describeSlide(slide, pres, i+1),
		})
	}
	return images
}

// renderSlidePreview creates a simplified Go-rendered preview of a slide.
func renderSlidePreview(slide *model.Slide, pres *model.Presentation) image.Image {
	const W, H = 1600, 900 // 16:9 at reasonable resolution
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Background
	bgColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if slide.Background != nil {
		if slide.Background.Type == model.BgSolid && slide.Background.Color != "" {
			bc := parseHexColor(slide.Background.Color)
			bgColor = bc
		} else if slide.Background.Type == model.BgGradient && len(slide.Background.Gradient.Stops) > 0 {
			c := parseHexColor(slide.Background.Gradient.Stops[0].Color)
			bgColor = c
		}
	} else if pres.Theme.BackgroundColor != "" {
		bgColor = parseHexColor(pres.Theme.BackgroundColor)
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	// Render elements
	for _, elem := range slide.Elements {
		x := int(float64(W) * elem.Rect.X / 100)
		y := int(float64(H) * elem.Rect.Y / 100)
		w := int(float64(W) * elem.Rect.W / 100)
		h := int(float64(H) * elem.Rect.H / 100)

		switch elem.Type {
		case model.ElementShape:
			shapeColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}
			if elem.Shape != nil && elem.Shape.FillColor != "" {
				shapeColor = parseHexColor(elem.Shape.FillColor)
			}
			if elem.Shape != nil && elem.Shape.Fill != nil {
				shapeColor = parseHexColor(elem.Shape.Fill.Color)
			}
			draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{C: shapeColor}, image.Point{}, draw.Over)
		case model.ElementImage:
			// Draw placeholder rectangle
			draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{C: color.RGBA{R: 240, G: 240, B: 240, A: 200}}, image.Point{}, draw.Over)
		default:
			// Text-bearing elements: draw colored rectangle as text placeholder
			textColor := color.RGBA{R: 50, G: 50, B: 50, A: 255}
			if elem.Style.Color != "" {
				textColor = parseHexColor(elem.Style.Color)
			} else if pres.Theme.TextColor != "" {
				textColor = parseHexColor(pres.Theme.TextColor)
			}
			alpha := uint8(60)
			if elem.Type == model.ElementTitle {
				alpha = 100
			}
			tinted := color.RGBA{R: textColor.R, G: textColor.G, B: textColor.B, A: alpha}
			rect := image.Rect(x, y, x+w, y+min(h, 30))
			if elem.Type == model.ElementTitle {
				rect = image.Rect(x, y, x+w, y+min(h, 60))
			}
			draw.Draw(img, rect, &image.Uniform{C: tinted}, image.Point{}, draw.Over)
		}
	}

	return img
}

// describeSlide generates a rich textual description for vision-less evaluation.
func describeSlide(slide *model.Slide, pres *model.Presentation, num int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Slide %d [%s]:\n", num, slide.Layout))

	// Background
	if slide.Background != nil {
		switch slide.Background.Type {
		case model.BgSolid:
			sb.WriteString(fmt.Sprintf("  Background: solid %s\n", slide.Background.Color))
		case model.BgGradient:
			stops := make([]string, len(slide.Background.Gradient.Stops))
			for i, s := range slide.Background.Gradient.Stops {
				stops[i] = s.Color
			}
			sb.WriteString(fmt.Sprintf("  Background: gradient %s\n", strings.Join(stops, "→")))
		case model.BgImage:
			sb.WriteString("  Background: image\n")
		}
	}

	for _, elem := range slide.Elements {
		x2 := elem.Rect.X + elem.Rect.W
		y2 := elem.Rect.Y + elem.Rect.H
		sb.WriteString(fmt.Sprintf("  [%s] pos(%.0f,%.0f)-(%.0f,%.0f)", elem.Type, elem.Rect.X, elem.Rect.Y, x2, y2))
		if elem.Text != "" {
			preview := elem.Text
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			sb.WriteString(fmt.Sprintf(" text=\"%s\"", preview))
		}
		if elem.Style.FontSize > 0 {
			sb.WriteString(fmt.Sprintf(" size=%dpt", elem.Style.FontSize))
		}
		if elem.Style.Color != "" {
			sb.WriteString(fmt.Sprintf(" color=%s", elem.Style.Color))
		}
		if elem.Style.Bold {
			sb.WriteString(" bold")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Cleanup removes temporary files.
func (r *Renderer) Cleanup() {
	if r.tempDir != "" {
		os.RemoveAll(r.tempDir)
	}
}

// ──────────── Helpers ────────────

func findExecutable(names ...string) string {
	for _, name := range names {
		if name == "" {
			continue
		}
		// Check common Windows paths
		winPaths := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "LibreOffice", "program", name+".exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "LibreOffice", "program", name+".exe"),
		}
		for _, p := range winPaths {
			if p != "" {
				if info, err := os.Stat(p); err == nil && !info.IsDir() {
					return p
				}
			}
		}
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func fileToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// Truncate if too large
	if len(data) > 4 << 20 {
		data = data[:4 << 20]
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func extractSlideNumber(name string) int {
	// Handle formats: slide-01.png, slide-1.png
	parts := strings.Split(name, "-")
	if len(parts) >= 2 {
		numStr := strings.TrimSuffix(parts[len(parts)-1], ".png")
		numStr = strings.TrimPrefix(numStr, "0")
		var num int
		fmt.Sscanf(numStr, "%d", &num)
		if num > 0 {
			return num
		}
	}
	return 1
}

func parseHexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	var r, g, b uint8
	if len(hex) >= 6 {
		fmt.Sscanf(hex[:6], "%02x%02x%02x", &r, &g, &b)
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Suppress unused import
var _ = math.Pi
