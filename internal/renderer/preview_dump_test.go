package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestDumpPreviewPNGs renders the sample presentation to PNGs under
// output/html-render-test for manual visual inspection. Run with:
//   go test ./internal/renderer/ -run TestDumpPreviewPNGs -v
func TestDumpPreviewPNGs(t *testing.T) {
	browser := FindBrowser()
	if browser == nil {
		t.Skip("no browser")
	}
	pres := samplePresentation()
	r := NewRenderer()
	r.libreOfficePath = ""
	r.pdftoppmPath = ""
	r.browser = browser

	images, err := r.renderWithBrowser(pres)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("output", "html-render-test")
	os.MkdirAll(dir, 0755)
	for _, img := range images {
		data, _ := os.ReadFile(img.Path)
		out := filepath.Join(dir, fmt.Sprintf("slide-%02d.png", img.SlideNum))
		os.WriteFile(out, data, 0644)
		t.Logf("wrote %s (%dx%d)", out, img.Width, img.Height)
	}
}
