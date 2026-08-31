package design

import (
	"strings"
	"testing"
)

func TestRegistries(t *testing.T) {
	if len(styles) < 5 {
		t.Errorf("expected at least 5 styles, got %d", len(styles))
	}
	if len(palettes) < 5 {
		t.Errorf("expected at least 5 palettes, got %d", len(palettes))
	}
	for k, s := range styles {
		if s.Key != k {
			t.Errorf("style %s has mismatched Key %s", k, s.Key)
		}
		if len(s.Rules) == 0 {
			t.Errorf("style %s has no hard rules", k)
		}
		if len(s.Recipes) == 0 {
			t.Errorf("style %s has no composition recipes", k)
		}
	}
	for k, p := range palettes {
		if p.Key != k {
			t.Errorf("palette %s has mismatched Key %s", k, p.Key)
		}
		for name, hex := range map[string]string{
			"background": p.Background, "secondary": p.BackgroundSecondary,
			"primary": p.Primary, "accent": p.Accent, "text": p.Text,
		} {
			if !strings.HasPrefix(hex, "#") || len(hex) != 7 {
				t.Errorf("palette %s role %s has invalid hex %q", k, name, hex)
			}
		}
	}
}

func TestGetters(t *testing.T) {
	if GetStyle("DARK_TECH") == nil {
		t.Error("style lookup should be case-insensitive")
	}
	if GetStyle("nope") != nil {
		t.Error("unknown style should return nil")
	}
	if GetPalette("TECH_NEON") == nil {
		t.Error("palette lookup should be case-insensitive")
	}
	if StyleKeys()[0] == "" || PaletteKeys()[0] == "" {
		t.Error("key listings should be non-empty")
	}
}

func TestLock(t *testing.T) {
	lock := Lock("dark_tech", "tech_neon")
	for _, want := range []string{"STYLE: dark_tech", "PALETTE: tech_neon", "#0A0E1A", "DESIGN LOCK", "Composition recipes",
		"TYPOGRAPHY ROLES", "PAGE RHYTHM", "±2px", "breathing", "hero", "footnote"} {
		if !strings.Contains(lock, want) {
			t.Errorf("lock missing %q", want)
		}
	}
	// Unknown keys must degrade gracefully, not panic or print empty sections
	// in a broken way.
	if l := Lock("nope", "nope"); !strings.Contains(l, "DESIGN LOCK") {
		t.Error("lock with unknown keys should still render header")
	}
	// Unknown keys must NOT print typography anchors derived from a nil style.
	if l := Lock("nope", "nope"); strings.Contains(l, "TYPOGRAPHY ROLES") {
		t.Error("lock with unknown style should not render typography roles")
	}
}

func TestTypeRoles(t *testing.T) {
	s := GetStyle("dark_tech")
	roles := TypeRoles(s)
	if len(roles) != 7 {
		t.Fatalf("expected 7 typography roles, got %d", len(roles))
	}
	byRole := map[string]int{}
	for _, r := range roles {
		byRole[r.Role] = r.Size
		if r.Size <= 0 {
			t.Errorf("role %s has non-positive size %d", r.Role, r.Size)
		}
	}
	// dark_tech: body=14, title=32 → hero=28, section=21, lead=16, caption=11, footnote=9
	want := map[string]int{"hero": 28, "title": 32, "section": 21, "lead": 16, "body": 14, "caption": 11, "footnote": 9}
	for role, size := range want {
		if byRole[role] != size {
			t.Errorf("role %s = %d, want %d", role, byRole[role], size)
		}
	}
	if TypeRoles(nil) != nil {
		t.Error("TypeRoles(nil) should return nil")
	}
}

func TestCatalogs(t *testing.T) {
	if !strings.Contains(StyleCatalog(), "Pick for") {
		t.Error("style catalog should contain selection rules")
	}
	if !strings.Contains(PaletteCatalog(), "tech_neon") {
		t.Error("palette catalog should list tech_neon")
	}
}
