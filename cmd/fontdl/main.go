// fontdl downloads curated open-source fonts into assets/fonts/ for PPTX embedding.
//
// It uses the Google Fonts CSS2 API to resolve font URLs from fonts.gstatic.com,
// then downloads the actual TTF files. It also copies select fonts from the
// local Windows system fonts directory when available.
//
// Usage:
//
//	go run ./cmd/fontdl                    # download all curated fonts
//	go run ./cmd/fontdl -dir custom/path   # custom output directory
//	go run ./cmd/fontdl -no-gstatic        # skip gstatic downloads
//	go run ./cmd/fontdl -sysfonts          # copy select Windows fonts
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

// Curated fonts to download from Google Fonts CSS2 API.
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

// Windows system fonts to copy (full, unmodified TTF files).
var sysFontSources = map[string]string{
	"msyh.ttc":   "MicrosoftYaHei.ttc",
	"msyhbd.ttc": "MicrosoftYaHei-Bold.ttc",
	"simhei.ttf": "SimHei.ttf",
	"simsun.ttc": "SimSun.ttc",
	"simkai.ttf": "KaiTi.ttf",
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

func copySysFonts(fontsDir string) int {
	if runtime.GOOS != "windows" {
		fmt.Println("  (System font copy is Windows-only, skipping)")
		return 0
	}

	sysDir := os.Getenv("WINDIR")
	if sysDir == "" {
		sysDir = `C:\Windows`
	}
	sysFontsDir := filepath.Join(sysDir, "Fonts")

	copied := 0
	for src, dst := range sysFontSources {
		srcPath := filepath.Join(sysFontsDir, src)
		dstPath := filepath.Join(fontsDir, dst)

		if _, err := os.Stat(dstPath); err == nil {
			fmt.Printf("  SKIP %s (exists)\n", dst)
			continue
		}

		if _, err := os.Stat(srcPath); err != nil {
			fmt.Printf("  MISS %s (not found in system fonts)\n", dst)
			continue
		}

		fmt.Printf("  COPY %s ... ", dst)
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
	copySys := flag.Bool("sysfonts", false, "copy select Windows system fonts")
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

	// Copy system fonts first (full CJK TTFs)
	if *copySys {
		fmt.Println("Copying Windows system fonts:")
		success += copySysFonts(absDir)
		fmt.Println()
	}

	// Download from Google Fonts gstatic
	if !*noGstatic {
		fmt.Println("Downloading from Google Fonts:")
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
