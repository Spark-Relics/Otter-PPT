// Package imageutil provides image manipulation helpers (crop, resize, format conversion).
package imageutil

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// OpenImage opens an image file and returns a decoded image.Image.
func OpenImage(path string) (image.Image, error) {
	return imaging.Open(path)
}

// CropAndResize crops a region from src and resizes to target dimensions.
// cropRect uses absolute pixel coordinates.
func CropAndResize(src image.Image, cropRect image.Rectangle, targetW, targetH int) (image.Image, error) {
	cropped := imaging.Crop(src, cropRect)
	resized := imaging.Resize(cropped, targetW, targetH, imaging.Lanczos)
	return resized, nil
}

// SaveImage saves an image to the given path. Format is determined by extension.
func SaveImage(img image.Image, path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	var buf bytes.Buffer

	switch ext {
	case ".jpg", ".jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
			return err
		}
	case ".png":
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// EnsureLocalImage checks if path is local; if it's a URL, caller should download first.
// Returns the local path.
func EnsureLocalImage(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", fmt.Errorf("URL images must be downloaded first: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("image file not found: %s", path)
	}
	return path, nil
}
