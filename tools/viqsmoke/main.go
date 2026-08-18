package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
	"github.com/otter-ppt/otter-ppt/internal/svgdecode"
	"github.com/otter-ppt/otter-ppt/internal/viz"
)

func dataMap(r pptoolkit.ToolResult) map[string]any {
	switch d := r.Data.(type) {
	case map[string]any:
		return d
	case map[string]string:
		m := make(map[string]any, len(d))
		for k, v := range d {
			m[k] = v
		}
		return m
	default:
		return map[string]any{}
	}
}

func main() {
	fail := 0
	check := func(name string, ok bool, extra string) {
		status := "OK"
		if !ok {
			status = "FAIL"
			fail++
		}
		fmt.Printf("[%s] %s %s\n", status, name, extra)
	}

	// 1. Tool registered in ToolDefinitions
	tools := pptoolkit.ToolDefinitions()
	names := map[string]bool{}
	for _, t := range tools {
		if t.Function != nil {
			names[t.Function.Name] = true
		}
	}
	check("tool get_viz_template registered", names["get_viz_template"], "")

	// 2. ExecuteTool returns template svg
	sess := pptoolkit.NewSession()
	res := sess.ExecuteTool("get_viz_template", map[string]any{"template": "column_chart"})
	svg, _ := dataMap(res)["svg"].(string)
	check("execute get_viz_template", res.Success && svg != "", fmt.Sprintf("(len(svg)=%d)", len(svg)))

	// 3. Viz templates are compileable + upgrade to native chart
	compiled, err := svgdecode.Compile(svg, svgdecode.Options{})
	check("compile column_chart.svg", err == nil, fmt.Sprintf("(err=%v)", err))
	hasChart := false
	if err == nil {
		for _, e := range compiled.Elements {
			if e.Type == model.ElementChart {
				hasChart = true
			}
		}
		check("compiled to native chart", hasChart, fmt.Sprintf("(elements=%d)", len(compiled.Elements)))
	}

	// 4. import_svg handler end-to-end on a slide
	sess.ExecuteTool("set_theme", map[string]any{"style": "dark_tech", "palette": "tech_neon"})
	slideRes := sess.ExecuteTool("add_slide", map[string]any{"layout": "title_content"})
	slideID, _ := dataMap(slideRes)["slide_id"].(string)
	imp := sess.ExecuteTool("import_svg", map[string]any{"slide_id": slideID, "svg": svg})
	dm := dataMap(imp)
	plot := dm["plot_area"]
	check("import_svg handler", imp.Success && len(compiled.Elements) > 0, fmt.Sprintf("(plot_area=%v)", plot))

	// 5. Catalog sanity: keys resolve, aliases work
	check("catalog has 16 entries", len(viz.List()) == 16, fmt.Sprintf("(got %d)", len(viz.List())))
	_, kind, _, ok := viz.Get("comparison")
	check("alias comparison→table", ok && kind == "table", "")

	// 6. Build the deck to a file
	out := "output/_viz_smoke.pptx"
	build := sess.ExecuteTool("export_pptx", map[string]any{"output_path": out})
	check("export_pptx", build.Success, fmt.Sprintf("(msg=%s)", strings.Split(build.Message, "\n")[0]))
	if build.Success {
		if fi, err := os.Stat(out); err == nil {
			check("pptx on disk", fi.Size() > 10000, fmt.Sprintf("(%d bytes)", fi.Size()))
		}
	}

	_ = json.Marshal
	if fail > 0 {
		fmt.Printf("\n%d CHECK(S) FAILED\n", fail)
		os.Exit(1)
	}
	fmt.Println("\nALL CHECKS PASSED")
}
