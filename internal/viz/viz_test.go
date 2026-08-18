package viz

import (
	"strings"
	"testing"
)

func TestRegistry(t *testing.T) {
	entries := List()
	if len(entries) != len(index) {
		t.Fatalf("List returned %d entries, want %d", len(entries), len(index))
	}

	charts := 0
	tables := 0
	for _, e := range entries {
		if e.Kind == "chart" {
			charts++
		} else if e.Kind == "table" {
			tables++
		} else {
			t.Errorf("entry %s has unknown kind %q", e.Key, e.Kind)
		}
		if !strings.Contains(e.Summary, "Pick for") {
			t.Errorf("entry %s summary is not a selection rule: %q", e.Key, e.Summary)
		}
	}
	if charts < 10 {
		t.Errorf("expected at least 10 chart templates, got %d", charts)
	}
	if tables != 6 {
		t.Errorf("expected 6 table templates, got %d", tables)
	}
}

func TestGet(t *testing.T) {
	for _, key := range []string{
		"column_chart", "horizontal_bar_chart", "line_chart", "pie_chart",
		"donut_chart", "funnel_chart", "gantt_chart", "radar_chart",
		"matrix_2x2", "waterfall_chart",
		"record_table", "metric_table", "comparison_matrix",
		"feature_matrix", "rating_matrix", "hierarchical_table",
	} {
		svg, kind, _, ok := Get(key)
		if !ok {
			t.Errorf("Get(%q) not found", key)
			continue
		}
		if !strings.Contains(svg, "viewBox=\"0 0 1280 720\"") {
			t.Errorf("%s missing canonical viewBox", key)
		}
		if !strings.Contains(svg, "data-pptx-bounds") {
			t.Errorf("%s missing data-pptx-bounds markers", key)
		}
		wantReplace := "chart"
		if kind == "table" {
			wantReplace = "table"
		}
		if !strings.Contains(svg, "data-pptx-replace-with=\""+wantReplace+"\"") {
			t.Errorf("%s missing data-pptx-replace-with=%s", key, wantReplace)
		}
	}
}

func TestAliases(t *testing.T) {
	for _, key := range []string{"bar", "pie", "donut", "funnel", "gantt", "matrix", "waterfall", "comparison_table"} {
		if !IsTemplateKey(key) {
			t.Errorf("alias %q should resolve", key)
		}
	}
	if IsTemplateKey("not_a_template") {
		t.Error("unknown key should not resolve")
	}
}

func TestCatalog(t *testing.T) {
	c := Catalog()
	for _, want := range []string{"CHART templates", "TABLE templates", "column_chart", "comparison_matrix"} {
		if !strings.Contains(c, want) {
			t.Errorf("catalog missing %q", want)
		}
	}
}
