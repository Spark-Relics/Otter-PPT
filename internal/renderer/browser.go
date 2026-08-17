// browser.go implements HTML → PNG screenshot via shell-out to whichever
// headless-capable browser is available on the system (Edge/Chrome/Chromium
// → Firefox). This mirrors the approach popularized by agent-friendly
// Office renderers: no embedded browser engine, the binary stays small,
// and on Windows the pre-installed Edge covers the common case with zero
// downloads.
package renderer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BrowserBackend describes a detected browser binary.
type BrowserBackend struct {
	Path string // executable path
	Name string // "edge", "chrome", "chromium", "firefox"
}

// FindBrowser returns the best available headless-capable browser, or nil.
// Preference: Chromium family (best --screenshot support) → Firefox.
func FindBrowser() *BrowserBackend {
	for _, cand := range chromiumCandidates() {
		if p := probePath(cand.path); p != "" {
			return &BrowserBackend{Path: p, Name: cand.name}
		}
	}
	// Firefox fallback
	if p, err := exec.LookPath("firefox"); err == nil {
		return &BrowserBackend{Path: p, Name: "firefox"}
	}
	return nil
}

type browserCandidate struct {
	name string
	path string
}

// chromiumCandidates returns Chromium-family browser paths in priority order.
func chromiumCandidates() []browserCandidate {
	var cands []browserCandidate
	// PATH first
	for _, name := range []string{"chrome", "chromium", "chromium-browser", "microsoft-edge", "msedge"} {
		if p, err := exec.LookPath(name); err == nil {
			cands = append(cands, browserCandidate{name: shortName(name), path: p})
		}
	}
	if runtime.GOOS == "windows" {
		locals := []string{
			filepath.Join(os.Getenv("ProgramFiles"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("LocalAppData"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("ProgramFiles"), `Chromium\Application\chrome.exe`),
			// Edge ships with Windows — the zero-download default.
			filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("LocalAppData"), `Microsoft\Edge\Application\msedge.exe`),
		}
		for _, p := range locals {
			cands = append(cands, browserCandidate{name: shortName(p), path: p})
		}
	} else if runtime.GOOS == "darwin" {
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		} {
			cands = append(cands, browserCandidate{name: shortName(p), path: p})
		}
	} else {
		for _, p := range []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/microsoft-edge",
			"/snap/bin/chromium",
		} {
			cands = append(cands, browserCandidate{name: shortName(p), path: p})
		}
	}
	return cands
}

func shortName(p string) string {
	base := strings.ToLower(filepath.Base(p))
	switch {
	case strings.Contains(base, "edge"):
		return "edge"
	case strings.Contains(base, "chromium"):
		return "chromium"
	default:
		return "chrome"
	}
}

func probePath(p string) string {
	if p == "" {
		return ""
	}
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		return p
	}
	return ""
}

// ScreenshotHTML renders an HTML file to a PNG at width×height using the
// given browser in headless mode. Returns the PNG path.
func (b *BrowserBackend) ScreenshotHTML(htmlPath string, width, height int, outPath string) error {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	url := "file:///" + strings.TrimPrefix(filepath.ToSlash(abs), "/")
	var args []string
	if b.Name == "firefox" {
		args = []string{
			"--headless",
			fmt.Sprintf("--screenshot=%s", outPath),
			fmt.Sprintf("--window-size=%d,%d", width, height),
			url,
		}
	} else {
		args = []string{
			"--headless=new",
			"--disable-gpu",
			"--no-sandbox",
			"--hide-scrollbars",
			"--force-device-scale-factor=1",
			fmt.Sprintf("--window-size=%d,%d", width, height),
			fmt.Sprintf("--screenshot=%s", outPath),
			url,
		}
	}
	cmd := exec.Command(b.Path, args...)
	cmd.SysProcAttr = hideWindowProcAttr()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s screenshot failed: %w: %s", b.Name, err, truncateForLog(out, 500))
	}
	if info, err := os.Stat(outPath); err != nil || info.Size() == 0 {
		return fmt.Errorf("%s screenshot produced no output", b.Name)
	}
	return nil
}

func truncateForLog(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		s = s[:n] + "..."
	}
	return s
}
