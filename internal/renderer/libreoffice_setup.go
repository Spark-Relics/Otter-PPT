// libreoffice_setup.go handles automatic LibreOffice (and poppler) provisioning.
//
// When the renderer can't find LibreOffice on the system, it downloads a
// portable version into a local cache directory (~/.otter-ppt/libreoffice/).
// This makes otter-ppt work on any machine without pre-installing anything.
//
// Platform specifics:
//   - Linux:   download .tar.gz, extract → soffice
//   - Windows: download .msi, silent install to cache → soffice.exe
//              download poppler-windows zip → pdftoppm.exe
//   - macOS:   download .dmg, mount + copy .app → soffice
//              (poppler must be installed via Homebrew)
package renderer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LOVersion is the LibreOffice version to download.
const LOVersion = "24.8.4"

// cacheDir returns the otter-ppt cache directory.
func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".otter-ppt")
}

// loCacheDir returns the directory where LibreOffice is cached.
func loCacheDir() string {
	return filepath.Join(cacheDir(), "libreoffice")
}

// popplerCacheDir returns the directory where poppler is cached (Windows only).
func popplerCacheDir() string {
	return filepath.Join(cacheDir(), "poppler")
}

// loDownloadURL returns the platform-specific LibreOffice download URL.
func loDownloadURL() string {
	base := fmt.Sprintf("https://download.documentfoundation.org/libreoffice/stable/%s", LOVersion)
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s/win/x86_64/LibreOffice_%s_Win_x86-64.msi", base, LOVersion)
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("%s/mac/aarch64/LibreOffice_%s_MacOS_aarch64.dmg", base, LOVersion)
		}
		return fmt.Sprintf("%s/mac/x86_64/LibreOffice_%s_MacOS_X86-64.dmg", base, LOVersion)
	default: // linux
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("%s/deb/aarch64/LibreOffice_%s_Linux_aarch64_deb.tar.gz", base, LOVersion)
		}
		return fmt.Sprintf("%s/deb/x86_64/LibreOffice_%s_Linux_x86-64_deb.tar.gz", base, LOVersion)
	}
}

// popplerDownloadURL returns the poppler-windows download URL (Windows only).
// Uses the oschwartz10612/poppler-windows GitHub releases.
func popplerDownloadURL() string {
	// Use a stable release tag.
	return "https://github.com/oschwartz10612/poppler-windows/releases/download/v24.08.0-0/Release-24.08.0-0.zip"
}

// downloadFile downloads a URL to a local path with a progress indicator.
func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "otter-ppt-setup/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractTarGz extracts a .tar.gz to a destination directory.
func extractTarGz(srcPath, destDir string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.Create(target)
			if err != nil {
				continue
			}
			io.Copy(outFile, tr)
			outFile.Close()
		case tar.TypeSymlink:
			// Skip symlinks for security
		}
	}
	return nil
}

// extractZip extracts a .zip to a destination directory.
func extractZip(srcPath, destDir string) error {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		if strings.HasSuffix(f.Name, "/") {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		outFile, err := os.Create(target)
		if err != nil {
			rc.Close()
			continue
		}
		io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
	return nil
}

// installLibreOfficeLinux downloads and extracts the .tar.gz, then extracts .debs.
func installLibreOfficeLinux(cacheDir string) (string, error) {
	url := loDownloadURL()
	tmpArchive := filepath.Join(cacheDir, "libreoffice.tar.gz")

	fmt.Printf("  Downloading LibreOffice %s for Linux...\n", LOVersion)
	if err := downloadFile(url, tmpArchive); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	extractDir := filepath.Join(cacheDir, "lo-extract")
	os.RemoveAll(extractDir)
	os.MkdirAll(extractDir, 0755)

	fmt.Println("  Extracting...")
	if err := extractTarGz(tmpArchive, extractDir); err != nil {
		return "", fmt.Errorf("extract: %w", err)
	}
	os.Remove(tmpArchive)

	// Find the .deb files and extract them
	installDir := filepath.Join(cacheDir, "installed")
	os.RemoveAll(installDir)
	os.MkdirAll(installDir, 0755)

	// Walk to find DEBS folder and extract each .deb
	filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".deb") {
			// Use dpkg-deb to extract (available on most Linux)
			cmd := exec.Command("dpkg-deb", "-x", path, installDir)
			cmd.Run()
		}
		return nil
	})
	os.RemoveAll(extractDir)

	// Find soffice binary
	sofficePath := findSofficeInDir(installDir)
	if sofficePath == "" {
		return "", fmt.Errorf("soffice not found after extraction")
	}
	return sofficePath, nil
}

// installLibreOfficeWindows downloads the .msi and installs silently.
func installLibreOfficeWindows(cacheDir string) (string, error) {
	url := loDownloadURL()
	tmpMSI := filepath.Join(cacheDir, "libreoffice.msi")

	fmt.Printf("  Downloading LibreOffice %s for Windows...\n", LOVersion)
	if err := downloadFile(url, tmpMSI); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	installDir := filepath.Join(cacheDir, "installed")
	fmt.Printf("  Installing to %s...\n", installDir)

	// Silent install with msiexec
	cmd := exec.Command("msiexec", "/i", tmpMSI,
		"INSTALLLOCATION="+installDir,
		"/quiet", "/norestart")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("msiexec: %w: %s", err, output)
	}
	os.Remove(tmpMSI)

	// Find soffice.exe
	sofficePath := filepath.Join(installDir, "program", "soffice.exe")
	if _, err := os.Stat(sofficePath); err != nil {
		// Try default install location
		sofficePath = findSofficeInDir(installDir)
		if sofficePath == "" {
			return "", fmt.Errorf("soffice.exe not found after install")
		}
	}
	return sofficePath, nil
}

// installLibreOfficeMac downloads the .dmg, mounts it, and copies the app.
func installLibreOfficeMac(cacheDir string) (string, error) {
	url := loDownloadURL()
	tmpDMG := filepath.Join(cacheDir, "libreoffice.dmg")

	fmt.Printf("  Downloading LibreOffice %s for macOS...\n", LOVersion)
	if err := downloadFile(url, tmpDMG); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	// Mount the DMG
	mountPoint := filepath.Join(cacheDir, "dmg-mount")
	os.MkdirAll(mountPoint, 0755)

	fmt.Println("  Mounting DMG...")
	cmd := exec.Command("hdiutil", "attach", "-nobrowse", "-mountpoint", mountPoint, tmpDMG)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpDMG)
		return "", fmt.Errorf("hdiutil attach: %w: %s", err, output)
	}

	// Copy LibreOffice.app
	appSrc := filepath.Join(mountPoint, "LibreOffice.app")
	appDst := filepath.Join(cacheDir, "LibreOffice.app")
	os.RemoveAll(appDst)

	fmt.Println("  Copying LibreOffice.app...")
	cmd = exec.Command("cp", "-R", appSrc, appDst)
	if err := cmd.Run(); err != nil {
		exec.Command("hdiutil", "detach", mountPoint).Run()
		os.Remove(tmpDMG)
		return "", fmt.Errorf("copy app: %w", err)
	}

	// Unmount and cleanup
	exec.Command("hdiutil", "detach", mountPoint).Run()
	os.Remove(tmpDMG)
	os.Remove(mountPoint)

	sofficePath := filepath.Join(appDst, "Contents", "MacOS", "soffice")
	if _, err := os.Stat(sofficePath); err != nil {
		return "", fmt.Errorf("soffice not found: %w", err)
	}
	return sofficePath, nil
}

// installPopplerWindows downloads and extracts poppler for Windows.
func installPopplerWindows(cacheDir string) (string, error) {
	url := popplerDownloadURL()
	tmpZip := filepath.Join(cacheDir, "poppler.zip")

	fmt.Println("  Downloading poppler for Windows...")
	if err := downloadFile(url, tmpZip); err != nil {
		return "", fmt.Errorf("download poppler: %w", err)
	}

	popplerDir := popplerCacheDir()
	os.RemoveAll(popplerDir)
	os.MkdirAll(popplerDir, 0755)

	fmt.Println("  Extracting poppler...")
	if err := extractZip(tmpZip, popplerDir); err != nil {
		return "", fmt.Errorf("extract poppler: %w", err)
	}
	os.Remove(tmpZip)

	// Find pdftoppm.exe
	var pdftoppmPath string
	filepath.Walk(popplerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, "pdftoppm.exe") {
			pdftoppmPath = path
		}
		return nil
	})

	if pdftoppmPath == "" {
		return "", fmt.Errorf("pdftoppm.exe not found after extraction")
	}
	return pdftoppmPath, nil
}

// findSofficeInDir searches for the soffice binary in a directory tree.
func findSofficeInDir(dir string) string {
	var sofficeName string
	switch runtime.GOOS {
	case "windows":
		sofficeName = "soffice.exe"
	case "darwin":
		sofficeName = "soffice"
	default:
		sofficeName = "soffice"
	}

	var found string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == sofficeName {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// EnsureLibreOffice checks for LibreOffice and downloads it if missing.
// Returns the paths to soffice and pdftoppm, or error.
func EnsureLibreOffice() (sofficePath, pdftoppmPath string, err error) {
	// Step 1: Check system-installed
	soffice := findExecutable("soffice", "libreoffice")
	pdftoppm := findExecutable("pdftoppm", "")

	if soffice != "" && pdftoppm != "" {
		return soffice, pdftoppm, nil
	}

	// Step 2: Check cache
	cacheDir := loCacheDir()
	cachedSoffice := findSofficeInDir(cacheDir)
	if cachedSoffice != "" {
		soffice = cachedSoffice
	}

	// Check poppler cache (Windows)
	if runtime.GOOS == "windows" && pdftoppm == "" {
		filepath.Walk(popplerCacheDir(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.Name() == "pdftoppm.exe" {
				pdftoppm = path
				return filepath.SkipAll
			}
			return nil
		})
	}

	if soffice != "" && pdftoppm != "" {
		return soffice, pdftoppm, nil
	}

	// Step 3: Download and install
	os.MkdirAll(cacheDir, 0755)

	fmt.Println("LibreOffice not found — setting up automatically...")

	if soffice == "" {
		switch runtime.GOOS {
		case "windows":
			soffice, err = installLibreOfficeWindows(cacheDir)
		case "darwin":
			soffice, err = installLibreOfficeMac(cacheDir)
		default:
			soffice, err = installLibreOfficeLinux(cacheDir)
		}
		if err != nil {
			return "", "", fmt.Errorf("install LibreOffice: %w", err)
		}
		fmt.Printf("  LibreOffice ready: %s\n", soffice)
	}

	if pdftoppm == "" {
		switch runtime.GOOS {
		case "windows":
			pdftoppm, err = installPopplerWindows(cacheDir)
			if err != nil {
				fmt.Printf("  WARNING: poppler install failed: %v\n", err)
				fmt.Println("  Rendering will use LibreOffice PDF→image fallback")
			}
		case "darwin":
			fmt.Println("  NOTE: Install poppler for macOS: brew install poppler")
		default:
			fmt.Println("  NOTE: Install poppler: sudo apt install poppler-utils")
		}
		if pdftoppm != "" {
			fmt.Printf("  poppler ready: %s\n", pdftoppm)
		}
	}

	return soffice, pdftoppm, nil
}
