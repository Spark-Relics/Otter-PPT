// fontdl downloads curated open-source fonts into assets/fonts/ for PPTX embedding.
//
// It uses the Google Fonts CSS2 API to resolve font URLs from fonts.gstatic.com,
// then downloads the actual TTF files. It can also download CJK fonts (Noto Sans SC/TC/JP/KR)
// from GitHub releases, and optionally copy select fonts from the local system
// fonts directory on Windows, macOS, or Linux.
//
// Usage:
//
//	go run ./cmd/fontdl                    # download all curated + CJK fonts
//	go run ./cmd/fontdl -dir custom/path   # custom output directory
//	go run ./cmd/fontdl -no-gstatic        # skip Google Fonts gstatic downloads
//	go run ./cmd/fontdl -no-cjk            # skip CJK font downloads from GitHub
//	go run ./cmd/fontdl -sysfonts          # copy select system fonts (cross-platform)
package main

import (
	"compress/flate"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type fontSpec struct {
	family   string
	weight   string
	filename string
}

// Curated Latin fonts to download from Google Fonts CSS2 API.
var curatedFonts = []fontSpec{
	{"Inter", "400", "Inter-Regular.ttf"},
	{"Inter", "700", "Inter-Bold.ttf"},
	{"Montserrat", "400", "Montserrat-Regular.ttf"},
	{"Montserrat", "700", "Montserrat-Bold.ttf"},
	{"Roboto", "400", "Roboto-Regular.ttf"},
	{"Open Sans", "400", "OpenSans-Regular.ttf"},
	{"Lato", "400", "Lato-Regular.ttf"},
	{"Poppins", "400", "Poppins-Regular.ttf"},
	{"Poppins", "700", "Poppins-Bold.ttf"},
	{"Raleway", "400", "Raleway-Regular.ttf"},
	{"Playfair Display", "400", "PlayfairDisplay-Regular.ttf"},
	{"Playfair Display", "700", "PlayfairDisplay-Bold.ttf"},
	{"Merriweather", "400", "Merriweather-Regular.ttf"},
	{"Lora", "400", "Lora-Regular.ttf"},
	{"JetBrains Mono", "400", "JetBrainsMono-Regular.ttf"},
	{"Source Code Pro", "400", "SourceCodePro-Regular.ttf"},
	{"Bebas Neue", "400", "BebasNeue-Regular.ttf"},
	{"Oswald", "400", "Oswald-Regular.ttf"},
	{"Dancing Script", "400", "DancingScript-Regular.ttf"},
	{"Caveat", "400", "Caveat-Regular.ttf"},
}

// CJK fonts downloaded directly from GitHub (Google Noto CJK releases).
// These are large (10-20 MB each) but are the most reliable cross-platform
// source for Chinese/Japanese/Korean fonts without relying on the OS.
type cjkFontSpec struct {
	name string
	url  string
	// sha256 is optional for verification (empty = skip checksum).
	sha256 string
}

var cjkFonts = []cjkFontSpec{
	{
		name: "NotoSansSC-Regular.otf",
		url:  "https://github.com/notofonts/noto-cjk/raw/main/Sans/OTF/SimplifiedChinese/NotoSansCJKsc-Regular.otf",
	},
	{
		name: "NotoSansSC-Bold.otf",
		url:  "https://github.com/notofonts/noto-cjk/raw/main/Sans/OTF/SimplifiedChinese/NotoSansCJKsc-Bold.otf",
	},
	{
		name: "NotoSansTC-Regular.otf",
		url:  "https://github.com/notofonts/noto-cjk/raw/main/Sans/OTF/TraditionalChinese/NotoSansCJKtc-Regular.otf",
	},
	{
		name: "NotoSansJP-Regular.otf",
		url:  "https://github.com/notofonts/noto-cjk/raw/main/Sans/OTF/Japanese/NotoSansCJKjp-Regular.otf",
	},
	{
		name: "NotoSansKR-Regular.otf",
		url:  "https://github.com/notofonts/noto-cjk/raw/main/Sans/OTF/Korean/NotoSansCJKkr-Regular.otf",
	},
	{
		name: "NotoSerifSC-Regular.otf",
		url:  "https://github.com/notofonts/noto-cjk/raw/main/Serif/OTF/SimplifiedChinese/NotoSerifCJKsc-Regular.otf",
	},
}

// systemFontSpec maps a display name to a list of possible source filenames
// across different operating systems. The tool searches each candidate
// in the system font directories for the current OS.
type systemFontSpec struct {
	displayName string
	candidates  []string // OS-agnostic candidate filenames, checked in order
}

// Common CJK system fonts across platforms.
// - Windows: msyh.ttc, simhei.ttf, ...
// - macOS:   PingFang.ttc, STHeiti Light.ttc, ...
// - Linux:   NotoSansCJK-Regular.ttc, wqy-zenhei.ttc, ...
var systemFontSources = []systemFontSpec{
	{displayName: "MicrosoftYaHei.ttc", candidates: []string{"msyh.ttc", "Microsoft Yahei.ttf", "PingFang.ttc", "NotoSansCJKsc-Regular.otf"}},
	{displayName: "MicrosoftYaHei-Bold.ttc", candidates: []string{"msyhbd.ttc", "Microsoft Yahei Bold.ttf", "PingFang-Bold.ttc", "NotoSansCJKsc-Bold.otf"}},
	{displayName: "SimHei.ttf", candidates: []string{"simhei.ttf", "STHeiti Medium.ttc", "wqy-zenhei.ttc"}},
	{displayName: "SimSun.ttc", candidates: []string{"simsun.ttc", "Songti.ttc", "NotoSerifCJKsc-Regular.otf"}},
	{displayName: "KaiTi.ttf", candidates: []string{"simkai.ttf", "STKaiti.ttf", "Kaiti.ttc"}},
}

const legacyUA = "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1; SV1)"

var srcURLRe = regexp.MustCompile(`src:\s*url\((https://fonts\.gstatic\.com[^)]+)\)`)

func getFontURL(family, weight string) (string, error) {
	cssURL := fmt.Sprintf("https://fonts.googleapis.com/css2?family=%s:wght@%s",
		strings.ReplaceAll(family, " ", "+"), weight)

	req, err := http.NewRequest("GET", cssURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", legacyUA)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	matches := srcURLRe.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("no font URL in CSS")
	}
	return matches[1], nil
}

func downloadFont(url, destPath string) error {
	// Use a custom transport to handle the gstatic compression
	transport := &http.Transport{
		DisableCompression: false,
	}
	client := &http.Client{Timeout: 120 * time.Second, Transport: transport}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", legacyUA)
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	// Handle gzip/deflate manually if Content-Encoding is set
	enc := resp.Header.Get("Content-Encoding")
	switch enc {
	case "gzip":
		// Go Transport already decompressed this when DisableCompression=false
		// and Accept-Encoding wasn't set manually
	case "deflate":
		reader = flate.NewReader(resp.Body)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if len(data) < 100 {
		return fmt.Errorf("file too small (%d bytes)", len(data))
	}

	return os.WriteFile(destPath, data, 0644)
}

// downloadCJKFont downloads a CJK font directly from a GitHub URL with a
// generous timeout (files are 10-20 MB).
func downloadCJKFont(url, destPath string) error {
	client := &http.Client{Timeout: 300 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "otter-ppt-fontdl/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(data) < 10000 {
		return fmt.Errorf("file too small (%d bytes), likely an error page", len(data))
	}

	return os.WriteFile(destPath, data, 0644)
}

// systemFontDirs returns the OS-specific directories where system fonts live.
func systemFontDirs() []string {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		sysDir := os.Getenv("WINDIR")
		if sysDir == "" {
			sysDir = `C:\Windows`
		}
		return []string{
			filepath.Join(sysDir, "Fonts"),
		}

	case "darwin":
		return []string{
			"/System/Library/Fonts",
			"/Library/Fonts",
			filepath.Join(home, "Library", "Fonts"),
			"/System/Library/Fonts/Supplemental",
		}

	default: // linux and other unix-like
		return []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			filepath.Join(home, ".local", "share", "fonts"),
			filepath.Join(home, ".fonts"),
		}
	}
}

// findSystemFont searches all system font directories for the first matching
// candidate filename (case-insensitive on macOS/Linux, exact on Windows).
func findSystemFont(candidates []string) string {
	dirs := systemFontDirs()

	for _, dir := range dirs {
		for _, candidate := range candidates {
			path := filepath.Join(dir, candidate)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
			// On case-sensitive filesystems, also try a case-insensitive walk
			if runtime.GOOS != "windows" {
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				lower := strings.ToLower(candidate)
				for _, entry := range entries {
					if strings.ToLower(entry.Name()) == lower {
						return filepath.Join(dir, entry.Name())
					}
				}
			}
		}
	}
	return ""
}

func copySysFonts(fontsDir string) int {
	if runtime.GOOS == "windows" {
		fmt.Println("  Platform: Windows")
	} else if runtime.GOOS == "darwin" {
		fmt.Println("  Platform: macOS")
	} else {
		fmt.Printf("  Platform: %s\n", runtime.GOOS)
	}
	fmt.Printf("  Font dirs: %s\n", strings.Join(systemFontDirs(), ", "))

	copied := 0
	for _, spec := range systemFontSources {
		dstPath := filepath.Join(fontsDir, spec.displayName)

		if _, err := os.Stat(dstPath); err == nil {
			fmt.Printf("  SKIP %s (exists)\n", spec.displayName)
			continue
		}

		srcPath := findSystemFont(spec.candidates)
		if srcPath == "" {
			fmt.Printf("  MISS %s (none of %v found)\n", spec.displayName, spec.candidates)
			continue
		}

		fmt.Printf("  COPY %s <- %s ... ", spec.displayName, filepath.Base(srcPath))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			continue
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			continue
		}
		fmt.Printf("OK (%d KB)\n", len(data)/1024)
		copied++
	}
	return copied
}

func main() {
	dir := flag.String("dir", "assets/fonts", "output directory")
	noGstatic := flag.Bool("no-gstatic", false, "skip Google Fonts gstatic downloads")
	noCJK := flag.Bool("no-cjk", false, "skip CJK font downloads from GitHub")
	copySys := flag.Bool("sysfonts", false, "copy select system fonts (cross-platform: Windows/macOS/Linux)")
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "create dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Font download tool — target: %s\n\n", absDir)

	success, failed, skipped := 0, 0, 0

	// Copy system fonts first (CJK TTFs from local OS)
	if *copySys {
		fmt.Println("Copying system fonts:")
		success += copySysFonts(absDir)
		fmt.Println()
	}

	// Download CJK fonts from GitHub (cross-platform, no system dependency)
	if !*noCJK {
		fmt.Println("Downloading CJK fonts from GitHub (Noto CJK):")
		for _, spec := range cjkFonts {
			destPath := filepath.Join(absDir, spec.name)

			if info, err := os.Stat(destPath); err == nil && info.Size() > 10000 {
				fmt.Printf("  SKIP %-30s (exists, %d KB)\n", spec.name, info.Size()/1024)
				skipped++
				continue
			}

			fmt.Printf("  GET  %-30s ", spec.name)

			if err := downloadCJKFont(spec.url, destPath); err != nil {
				fmt.Printf("FAIL (%v)\n", err)
				os.Remove(destPath)
				failed++
				continue
			}

			info, _ := os.Stat(destPath)
			fmt.Printf("OK (%d KB)\n", info.Size()/1024)
			success++
			time.Sleep(300 * time.Millisecond)
		}
		fmt.Println()
	}

	// Download Latin fonts from Google Fonts gstatic
	if !*noGstatic {
		fmt.Println("Downloading Latin fonts from Google Fonts:")
		for _, spec := range curatedFonts {
			destPath := filepath.Join(absDir, spec.filename)

			if info, err := os.Stat(destPath); err == nil && info.Size() > 1000 {
				fmt.Printf("  SKIP %-30s (exists, %d KB)\n", spec.filename, info.Size()/1024)
				skipped++
				continue
			}

			fmt.Printf("  GET  %-30s [%s] ", spec.filename, spec.weight)

			fontURL, err := getFontURL(spec.family, spec.weight)
			if err != nil {
				fmt.Printf("FAIL (%v)\n", err)
				failed++
				continue
			}

			if err := downloadFont(fontURL, destPath); err != nil {
				fmt.Printf("FAIL (%v)\n", err)
				os.Remove(destPath)
				failed++
				continue
			}

			info, _ := os.Stat(destPath)
			fmt.Printf("OK (%d KB)\n", info.Size()/1024)
			success++
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Println()
	}

	fmt.Printf("Done: %d downloaded/copied, %d skipped, %d failed\n", success, skipped, failed)
}
