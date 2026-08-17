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
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

// EMU (English Metric Units) conversion constants.
// PPTX uses EMU internally: 914400 EMU = 1 inch, 9525 EMU = 1 pixel @96dpi.
const (
	EMUPerInch = 914400
)

// Builder renders a Presentation to PPTX.
type mediaAsset struct {
	data     []byte
	fileName string
	relID    string
	ext      string
}

type chartAsset struct {
	index int
	relID string
	data  *model.ChartData
}

type Builder struct {
	pres            *model.Presentation
	mediaByElement  map[*model.Element]*mediaAsset
	mediaAssets     []*mediaAsset
	chartByElement  map[*model.Element]*chartAsset
	chartAssets     []*chartAsset
	bgImageBySlide  map[int]*mediaAsset
	posterByElement map[*model.Element]*mediaAsset
	embeddedFonts   []embeddedFont
}

// New creates a builder for the given presentation.
func New(pres *model.Presentation) *Builder {
	if pres.SlideWidth == 0 || pres.SlideHeight == 0 {
		pres.SlideWidth, pres.SlideHeight = model.DefaultSlideSize()
	}
	b := &Builder{
		pres:            pres,
		mediaByElement:  make(map[*model.Element]*mediaAsset),
		chartByElement:  make(map[*model.Element]*chartAsset),
		bgImageBySlide:  make(map[int]*mediaAsset),
		posterByElement: make(map[*model.Element]*mediaAsset),
	}
	b.embeddedFonts = b.prepareEmbeddedFonts()
	return b
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
	if err := b.prepareAssets(); err != nil {
		return err
	}
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
		b.writeSlideLayoutRels,
		b.writeNotesMaster,
		b.writeNotesMasterRels,
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
		if err := b.writeSlideRels(zw, i+1, slide); err != nil {
			return err
		}
		// Write notes slides
		if slide.Notes != "" {
			if err := b.writeNotesSlide(zw, i+1, slide); err != nil {
				return err
			}
			if err := b.writeNotesSlideRels(zw, i+1); err != nil {
				return err
			}
		}
	}
	for _, asset := range b.mediaAssets {
		entry, err := zw.Create("ppt/media/" + asset.fileName)
		if err != nil {
			return err
		}
		if _, err := entry.Write(asset.data); err != nil {
			return err
		}
	}
	for _, asset := range b.chartAssets {
		if err := b.writeChartPart(zw, asset); err != nil {
			return err
		}
	}

	// Write embedded fonts
	if len(b.embeddedFonts) > 0 {
		if err := b.writeEmbeddedFonts(zw, b.embeddedFonts); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) prepareAssets() error {
	b.mediaAssets = nil
	b.mediaByElement = make(map[*model.Element]*mediaAsset)
	b.chartAssets = nil
	b.chartByElement = make(map[*model.Element]*chartAsset)
	b.bgImageBySlide = make(map[int]*mediaAsset)
	mediaIndex := 1
	chartIndex := 1
	for slideIdx, slide := range b.pres.Slides {
		relIndex := 2

		// Background image
		if slide.Background != nil && slide.Background.Type == model.BgImage && slide.Background.ImagePath != "" {
			data, ext, err := loadImageData(slide.Background.ImagePath)
			if err == nil {
				asset := &mediaAsset{data: data, fileName: fmt.Sprintf("image%d%s", mediaIndex, ext), relID: fmt.Sprintf("rId%d", relIndex), ext: strings.TrimPrefix(ext, ".")}
				b.mediaAssets = append(b.mediaAssets, asset)
				b.bgImageBySlide[slideIdx] = asset
				mediaIndex++
				relIndex++
			}
		}

		for _, elem := range slide.Elements {
			if elem.Type == model.ElementChart && elem.Chart != nil {
				asset := &chartAsset{index: chartIndex, relID: fmt.Sprintf("rId%d", relIndex), data: elem.Chart}
				b.chartAssets = append(b.chartAssets, asset)
				b.chartByElement[elem] = asset
				chartIndex++
				relIndex++
				continue
			}
			// Video / Audio media
			if (elem.Type == model.ElementVideo || elem.Type == model.ElementAudio) && elem.Media != nil && elem.Media.MediaPath != "" {
				data, ext, err := loadMediaData(elem.Media.MediaPath)
				if err == nil {
					asset := &mediaAsset{data: data, fileName: fmt.Sprintf("media%d%s", mediaIndex, ext), relID: fmt.Sprintf("rId%d", relIndex), ext: strings.TrimPrefix(ext, ".")}
					b.mediaAssets = append(b.mediaAssets, asset)
					b.mediaByElement[elem] = asset
					mediaIndex++
					relIndex++
					// Load poster image if provided
					if elem.Media.PosterPath != "" {
						pData, pExt, err := loadImageData(elem.Media.PosterPath)
						if err == nil {
							poster := &mediaAsset{data: pData, fileName: fmt.Sprintf("image%d%s", mediaIndex, pExt), relID: fmt.Sprintf("rId%d", relIndex), ext: strings.TrimPrefix(pExt, ".")}
							b.mediaAssets = append(b.mediaAssets, poster)
							b.posterByElement[elem] = poster
							mediaIndex++
							relIndex++
						}
					}
					continue
				}
				continue
			}
			if elem.Type != model.ElementImage || elem.ImagePath == "" {
				continue
			}
			data, ext, err := loadImageData(elem.ImagePath)
			if err != nil {
				return fmt.Errorf("load image %q: %w", elem.ImagePath, err)
			}
			asset := &mediaAsset{data: data, fileName: fmt.Sprintf("image%d%s", mediaIndex, ext), relID: fmt.Sprintf("rId%d", relIndex), ext: strings.TrimPrefix(ext, ".")}
			b.mediaAssets = append(b.mediaAssets, asset)
			b.mediaByElement[elem] = asset
			mediaIndex++
			relIndex++
		}
	}
	return nil
}

// loadImageData reads image data from a local path or downloads from a URL.
// Returns the raw bytes and the file extension (e.g. ".png").
func loadImageData(path string) ([]byte, string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, "", fmt.Errorf("download image: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, "", fmt.Errorf("download image: HTTP %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20)) // 50MB limit
		if err != nil {
			return nil, "", fmt.Errorf("read image response: %w", err)
		}
		// Determine extension from Content-Type or URL
		ext := ".png"
		ct := resp.Header.Get("Content-Type")
		switch {
		case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
			ext = ".jpg"
		case strings.Contains(ct, "gif"):
			ext = ".gif"
		case strings.Contains(ct, "webp"):
			ext = ".png" // convert webp to png placeholder
		case strings.Contains(ct, "svg"):
			ext = ".svg"
		default:
			uExt := strings.ToLower(filepath.Ext(path))
			if uExt == ".jpeg" {
				uExt = ".jpg"
			}
			if uExt == ".png" || uExt == ".jpg" || uExt == ".gif" || uExt == ".svg" {
				ext = uExt
			}
		}
		return data, ext, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	if ext != ".png" && ext != ".jpg" && ext != ".gif" && ext != ".svg" {
		return nil, "", fmt.Errorf("unsupported image format %q", ext)
	}
	return data, ext, nil
}

// loadMediaData reads video/audio data from a local path or downloads from a URL.
// Returns the raw bytes and the file extension (e.g. ".mp4").
func loadMediaData(path string) ([]byte, string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, "", fmt.Errorf("download media: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, "", fmt.Errorf("download media: HTTP %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 200<<20)) // 200MB limit
		if err != nil {
			return nil, "", fmt.Errorf("read media response: %w", err)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == "" {
			ext = ".mp4"
		}
		return data, ext, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = ".mp4"
	}
	return data, ext, nil
}

func mediaContentType(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "svg":
		return "image/svg+xml"
	case "mp4":
		return "video/mp4"
	case "avi":
		return "video/x-msvideo"
	case "mov":
		return "video/quicktime"
	case "wmv":
		return "video/x-ms-wmv"
	case "webm":
		return "video/webm"
	case "mkv":
		return "video/x-matroska"
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "aac":
		return "audio/aac"
	case "m4a":
		return "audio/mp4"
	case "ogg":
		return "audio/ogg"
	case "flac":
		return "audio/flac"
	}
	if value := mime.TypeByExtension("." + ext); value != "" {
		return value
	}
	return "application/octet-stream"
}

// ─────────── Helper conversion functions ───────────

// pctToEMU converts a percentage (0-100) to EMU for the given total dimension (inches).
func pctToEMU(pct, totalInches float64) int64 {
	return int64(pct / 100.0 * totalInches * float64(EMUPerInch))
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
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return s
}
