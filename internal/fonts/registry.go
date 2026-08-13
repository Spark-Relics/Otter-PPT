// Package fonts provides font discovery, metadata parsing, and catalog management.
// It scans the `assets/fonts/` directory for .ttf/.otf files, parses their metadata,
// and exposes them through a registry that the builder can use to embed fonts into PPTX files.
package fonts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FontEntry represents a discovered font file with its metadata.
type FontEntry struct {
	// Display name (e.g. "Inter")
	Name string `json:"name"`
	// Full name (e.g. "Inter Bold")
	FullName string `json:"full_name"`
	// PostScript name (e.g. "Inter-Bold") — used for PPTX embedding
	PostScriptName string `json:"postscript_name"`
	// Subfamily: Regular, Bold, Italic, etc.
	Subfamily string `json:"subfamily"`
	// Weight class: 100-900
	Weight string `json:"weight"`
	// File path relative to fonts directory
	FileName string `json:"file_name"`
	// Absolute path on disk
	Path string `json:"-"`
	// File size in bytes
	Size int64 `json:"size"`
	// Whether this is a CJK font
	CJK bool `json:"cjk"`
	// Whether this font is bold
	Bold bool `json:"bold"`
	// Whether this font is italic
	Italic bool `json:"italic"`
	// Style tags for categorization
	Tags []string `json:"tags,omitempty"`
}

// Registry manages font discovery and lookup.
type Registry struct {
	mu       sync.RWMutex
	fontsDir string
	entries  []FontEntry
	byName   map[string]*FontEntry // key: lowercased family name
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// SetRegistry replaces the global registry instance.
// Useful for testing or when the fonts directory path is known at startup.
func SetRegistry(r *Registry) {
	globalRegistry = r
}

// GetRegistry returns the singleton font registry.
func GetRegistry() *Registry {
	once.Do(func() {
		// Try common locations relative to working directory
		candidates := []string{
			"assets/fonts",
			"../assets/fonts",
			"../../assets/fonts",
			"fonts",
		}
		for _, dir := range candidates {
			if abs, err := filepath.Abs(dir); err == nil {
				if info, err := os.Stat(abs); err == nil && info.IsDir() {
					globalRegistry = NewRegistry(abs)
					return
				}
			}
		}
		// Fallback: create with default path
		abs, _ := filepath.Abs("assets/fonts")
		globalRegistry = NewRegistry(abs)
	})
	return globalRegistry
}

// NewRegistry creates a font registry for the given directory.
func NewRegistry(fontsDir string) *Registry {
	return &Registry{
		fontsDir: fontsDir,
		byName:   make(map[string]*FontEntry),
	}
}

// Scan discovers all .ttf/.otf/.ttc font files in the fonts directory.
func (r *Registry) Scan() ([]FontEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = nil
	r.byName = make(map[string]*FontEntry)

	// Ensure directory exists
	if _, err := os.Stat(r.fontsDir); os.IsNotExist(err) {
		// Return empty list — no fonts installed yet
		return nil, nil
	}

	err := filepath.Walk(r.fontsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ttf" && ext != ".otf" && ext != ".ttc" {
			return nil
		}

		entry, parseErr := parseFontFile(path, info)
		if parseErr != nil {
			// If parsing fails, still include with filename-based name
			entry = &FontEntry{
				Name:           strings.TrimSuffix(filepath.Base(path), ext),
				FullName:       strings.TrimSuffix(filepath.Base(path), ext),
				PostScriptName: strings.TrimSuffix(filepath.Base(path), ext),
				Subfamily:      guessSubfamily(filepath.Base(path)),
				FileName:       relPath(r.fontsDir, path),
				Path:           path,
				Size:           info.Size(),
				Weight:         guessWeight(filepath.Base(path)),
				Tags:           guessTags(filepath.Base(path)),
			}
		}
		if entry.FileName == "" {
			entry.FileName = relPath(r.fontsDir, path)
		}
		if entry.Subfamily == "" {
			entry.Subfamily = guessSubfamily(filepath.Base(path))
		}
		if entry.Weight == "" {
			entry.Weight = guessWeight(filepath.Base(path))
		}
		if len(entry.Tags) == 0 {
			entry.Tags = guessTags(filepath.Base(path))
		}

		// Detect CJK
		lower := strings.ToLower(entry.Name)
		for _, cjkKeyword := range []string{"noto sans sc", "noto sans cjk", "noto serif sc", "noto serif cjk",
			"source han", "思源", "微软雅黑", "msyh", "simsun", "simhei", "fangsong", "kaiti",
			"noto sans jp", "noto sans kr", "noto sans tc", "sarasa", "霞鹜"} {
			if strings.Contains(lower, cjkKeyword) {
				entry.CJK = true
				break
			}
		}

		r.entries = append(r.entries, *entry)
		key := strings.ToLower(entry.Name)
		if _, exists := r.byName[key]; !exists {
			e := *entry
			r.byName[key] = &e
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("scan fonts dir: %w", err)
	}

	// Sort: CJK fonts first, then alphabetical
	sort.SliceStable(r.entries, func(i, j int) bool {
		if r.entries[i].CJK != r.entries[j].CJK {
			return r.entries[i].CJK
		}
		return r.entries[i].Name < r.entries[j].Name
	})

	return r.entries, nil
}

// All returns all discovered font entries.
func (r *Registry) All() []FontEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries
}

// Lookup finds a font by family name (case-insensitive).
// Returns nil if not found.
func (r *Registry) Lookup(name string) *FontEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[strings.ToLower(name)]
}

// HasFont checks if a font family exists in the registry.
func (r *Registry) HasFont(name string) bool {
	return r.Lookup(name) != nil
}

// Names returns a deduplicated list of font family names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]bool)
	var names []string
	for _, e := range r.entries {
		key := strings.ToLower(e.Name)
		if !seen[key] {
			seen[key] = true
			names = append(names, e.Name)
		}
	}
	return names
}

// Catalog groups font entries by family name, useful for UI display.
type FontFamily struct {
	Name    string      `json:"name"`
	Members []FontEntry `json:"members"`
	CJK     bool        `json:"cjk"`
}

// Catalog returns fonts grouped by family.
func (r *Registry) Catalog() []FontFamily {
	r.mu.RLock()
	defer r.mu.RUnlock()

	groups := make(map[string]*FontFamily)
	var order []string

	for _, e := range r.entries {
		fam, exists := groups[e.Name]
		if !exists {
			fam = &FontFamily{Name: e.Name, CJK: e.CJK}
			groups[e.Name] = fam
			order = append(order, e.Name)
		}
		fam.Members = append(fam.Members, e)
		if e.CJK {
			fam.CJK = true
		}
	}

	result := make([]FontFamily, 0, len(order))
	for _, name := range order {
		result = append(result, *groups[name])
	}
	return result
}

// FontPath returns the file path for a given font name, or empty if not found.
func (r *Registry) FontPath(name string) string {
	e := r.Lookup(name)
	if e == nil {
		return ""
	}
	return e.Path
}

// FontsDir returns the font directory path.
func (r *Registry) FontsDir() string {
	return r.fontsDir
}

// FontData returns the raw font file bytes for embedding.
func (r *Registry) FontData(name string) ([]byte, error) {
	e := r.Lookup(name)
	if e == nil {
		return nil, fmt.Errorf("font %q not found", name)
	}
	return os.ReadFile(e.Path)
}

// ── helpers ──

func parseFontFile(path string, info os.FileInfo) (*FontEntry, error) {
	ttfInfo, err := ParseTTF(path)
	if err != nil {
		return nil, err
	}
	return &FontEntry{
		Name:           ttfInfo.FamilyName,
		FullName:       ttfInfo.FullName,
		PostScriptName: ttfInfo.PostScript,
		Subfamily:      ttfInfo.SubfamilyName,
		Bold:           ttfInfo.IsBold,
		Italic:         ttfInfo.IsItalic,
		FileName:       filepath.Base(path),
		Path:           path,
		Size:           info.Size(),
		Weight:         guessWeight(ttfInfo.SubfamilyName),
		Tags:           guessTags(ttfInfo.FamilyName),
	}, nil
}

func relPath(base, full string) string {
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return filepath.Base(full)
	}
	return rel
}

func guessSubfamily(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.Contains(lower, "bold"):
		return "Bold"
	case strings.Contains(lower, "italic") || strings.Contains(lower, "oblique"):
		return "Italic"
	case strings.Contains(lower, "light"):
		return "Light"
	case strings.Contains(lower, "medium"):
		return "Medium"
	case strings.Contains(lower, "black") || strings.Contains(lower, "heavy"):
		return "Black"
	case strings.Contains(lower, "thin"):
		return "Thin"
	default:
		return "Regular"
	}
}

func guessWeight(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "thin"):
		return "100"
	case strings.Contains(lower, "extralight") || strings.Contains(lower, "extra-light"):
		return "200"
	case strings.Contains(lower, "light"):
		return "300"
	case strings.Contains(lower, "medium"):
		return "500"
	case strings.Contains(lower, "semibold") || strings.Contains(lower, "demibold"):
		return "600"
	case strings.Contains(lower, "bold"):
		return "700"
	case strings.Contains(lower, "extrabold") || strings.Contains(lower, "heavy"):
		return "800"
	case strings.Contains(lower, "black"):
		return "900"
	default:
		return "400"
	}
}

func guessTags(name string) []string {
	lower := strings.ToLower(name)
	var tags []string

	// Script type
	switch {
	case strings.Contains(lower, "mono") || strings.Contains(lower, "courier") || strings.Contains(lower, "jetbrains") || strings.Contains(lower, "fira code"):
		tags = append(tags, "mono", "code")
	case strings.Contains(lower, "serif"):
		tags = append(tags, "serif", "elegant")
	case strings.Contains(lower, "sans"):
		tags = append(tags, "sans", "clean")
	}

	// Mood
	switch {
	case strings.Contains(lower, "playfair") || strings.Contains(lower, "cormorant") || strings.Contains(lower, "lora"):
		tags = append(tags, "elegant", "editorial")
	case strings.Contains(lower, "inter") || strings.Contains(lower, "roboto") || strings.Contains(lower, "open sans"):
		tags = append(tags, "modern", "ui")
	case strings.Contains(lower, "pacifico") || strings.Contains(lower, "caveat") || strings.Contains(lower, "dancing"):
		tags = append(tags, "handwriting", "playful")
	}

	// CJK
	for _, kw := range []string{"noto", "source han", "思源", "霞鹜", "sarasa"} {
		if strings.Contains(lower, kw) {
			tags = append(tags, "cjk")
			break
		}
	}

	if len(tags) == 0 {
		tags = []string{"sans"}
	}

	return tags
}
