package viz

import (
	"fmt"
	"sort"
	"strings"
)

// indexEntry is the selection rule for one canonical visualization template.
// The summary is deliberately a selection rule ("Pick for ... Skip if ..."),
// not a description — this mirrors ppt-master's visualization recall and keeps
// the LLM choosing by data shape instead of aesthetic preference.
type indexEntry struct {
	Kind    string // chart | table
	Summary string
}

var index = map[string]indexEntry{
	// ────────── charts ──────────
	"column_chart": {
		Kind: "chart",
		Summary: "Pick for single-series category comparison with 3-8 categories. " +
			"Skip for long labels or >12 items (use horizontal_bar_chart) or multi-series comparison (use add_chart with multiple series).",
	},
	"horizontal_bar_chart": {
		Kind: "chart",
		Summary: "Pick for ranking 5-12 items, especially when labels are long. " +
			"Skip for <=8 short labels (use column_chart).",
	},
	"line_chart": {
		Kind: "chart",
		Summary: "Pick for 1-3 time-series on a continuous axis showing direction. " +
			"Skip if cumulative volume matters (use add_chart area) or the two metrics use different units (use add_chart combo).",
	},
	"pie_chart": {
		Kind: "chart",
		Summary: "Pick for 3-6 simple proportions of one whole. " +
			"Skip for >=7 parts (use donut_chart) or when a center total deserves emphasis (use donut_chart).",
	},
	"donut_chart": {
		Kind: "chart",
		Summary: "Pick for 3-6 part proportions where a center KPI/total deserves emphasis. " +
			"Skip if there is no meaningful center value (use pie_chart).",
	},
	"funnel_chart": {
		Kind: "chart",
		Summary: "Pick for 3-5 sequential conversion stages whose values drive a monotonic drop-off. " +
			"Skip if stages do not encode loss or qualitative branches carry the message (build a diagram structure instead).",
	},
	"gantt_chart": {
		Kind: "chart",
		Summary: "Pick for a project schedule with 6-12 tasks whose dates/durations determine bar position and length. " +
			"Skip for milestones without duration or qualitative lane-stage handoffs (build a structure instead).",
	},
	"radar_chart": {
		Kind: "chart",
		Summary: "Pick for 4-8 capability dimensions scored across 1-3 entities. " +
			"Skip for >3 entities (use grouped bars) or <4 dimensions.",
	},
	"matrix_2x2": {
		Kind: "chart",
		Summary: "Pick for 4-10 items whose x/y positions encode two dimensions and bubble radius a third magnitude inside a 2x2 quadrant frame. " +
			"Skip for fixed qualitative regions with text assigned to zones (build a structure instead).",
	},
	"waterfall_chart": {
		Kind: "chart",
		Summary: "Pick for showing how an initial value rises/falls through sequential contributions to a final total. " +
			"Skip if the running-total story is not the message (use column_chart).",
	},

	// ────────── tables ──────────
	"record_table": {
		Kind: "table",
		Summary: "Pick for one flat record per row and one stable heterogeneous field per column. " +
			"Skip for KPI scanning (use metric_table) or grouped rows with totals (use hierarchical_table).",
	},
	"metric_table": {
		Kind: "table",
		Summary: "Pick for operating metrics by entity with current values, changes, statuses, or target progress inside cells. " +
			"Skip if the marks leave the grid and encode magnitude (use a chart).",
	},
	"comparison_matrix": {
		Kind: "table",
		Summary: "Pick for criteria × alternatives with prose, exact values, or mixed facts at intersections. " +
			"Skip if cells encode only feature states (use feature_matrix) or ordinal ratings (use rating_matrix).",
	},
	"feature_matrix": {
		Kind: "table",
		Summary: "Pick for capabilities × offerings with supported, unsupported, partial, or exception states. " +
			"Skip for mixed facts (use comparison_matrix) or ordinal scores (use rating_matrix).",
	},
	"rating_matrix": {
		Kind: "table",
		Summary: "Pick for criteria × alternatives using one repeated ordinal scale. " +
			"Skip for feature states (use feature_matrix) or exact facts (use comparison_matrix).",
	},
	"hierarchical_table": {
		Kind: "table",
		Summary: "Pick for grouped or indented row hierarchies across stable measure columns, including subtotals/totals. " +
			"Skip for flat records (use record_table) or geometry driven by numeric magnitude (use a chart).",
	},
}

// List returns all visualization entries sorted by kind then key.
func List() []Entry {
	out := make([]Entry, 0, len(index))
	for key, e := range index {
		out = append(out, Entry{Key: key, Kind: e.Kind, Summary: e.Summary})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// Catalog renders the selection rules for LLM prompts.
func Catalog() string {
	var sb strings.Builder
	sb.WriteString("CHART templates (value-driven geometry — pick by data shape, not preference):\n")
	for _, e := range List() {
		if e.Kind == "chart" {
			fmt.Fprintf(&sb, "- %s: %s\n", e.Key, e.Summary)
		}
	}
	sb.WriteString("\nTABLE templates (row × column fact grid):\n")
	for _, e := range List() {
		if e.Kind == "table" {
			fmt.Fprintf(&sb, "- %s: %s\n", e.Key, e.Summary)
		}
	}
	return sb.String()
}

// canonicalKey maps friendly aliases to registry keys.
func canonicalKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	aliases := map[string]string{
		"bar":                 "horizontal_bar_chart",
		"bar_chart":           "horizontal_bar_chart",
		"column":              "column_chart",
		"line":                "line_chart",
		"pie":                 "pie_chart",
		"donut":               "donut_chart",
		"doughnut":            "donut_chart",
		"funnel":              "funnel_chart",
		"gantt":               "gantt_chart",
		"radar":               "radar_chart",
		"matrix":              "matrix_2x2",
		"waterfall":           "waterfall_chart",
		"record":              "record_table",
		"metric":              "metric_table",
		"comparison":          "comparison_matrix",
		"comparison_table":    "comparison_matrix",
		"feature":             "feature_matrix",
		"rating":              "rating_matrix",
		"hierarchical":        "hierarchical_table",
		"hierarchical_table_": "hierarchical_table",
	}
	if canon, ok := aliases[key]; ok {
		return canon
	}
	if _, ok := index[key]; ok {
		return key
	}
	return key
}
