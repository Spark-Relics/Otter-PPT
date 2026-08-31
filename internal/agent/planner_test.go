package agent

import "testing"

func TestRhythmOrDefault(t *testing.T) {
	if rhythmOrDefault("breathing") != "breathing" || rhythmOrDefault("anchor") != "anchor" {
		t.Error("valid rhythm tags must pass through")
	}
	if rhythmOrDefault("") != "dense" || rhythmOrDefault("bogus") != "dense" {
		t.Error("unknown/empty rhythm must default to dense")
	}
}

func TestFormatPlanRhythm(t *testing.T) {
	plan := &DesignPlan{
		Title: "T",
		Slides: []SlidePlan{
			{Number: 1, Title: "Cover", Layout: "title", Rhythm: "anchor"},
			{Number: 2, Title: "Data", Layout: "chart", Rhythm: "dense"},
			{Number: 3, Title: "Impact", Layout: "quote", Rhythm: "breathing"},
			{Number: 4, Title: "Missing", Layout: "bullets"},
		},
	}
	out := FormatPlan(plan)
	for _, want := range []string{"(anchor)", "(dense)", "(breathing)", "(dense): Missing"} {
		if !contains(out, want) {
			t.Errorf("FormatPlan output missing %q", want)
		}
	}
	if !contains(out, "rhythm tag") {
		t.Error("FormatPlan should remind the builder to obey rhythm tags")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
