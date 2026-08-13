package pptoolkit

import "encoding/json"

// prop describes a single JSON schema property.
type prop struct {
	typ  string
	desc string
	req  bool
}

// props builds a JSON schema (as RawMessage) from a property map.
func props(m map[string]prop) json.RawMessage {
	properties := map[string]any{}
	required := []string{}
	for name, p := range m {
		properties[name] = map[string]any{
			"type":        p.typ,
			"description": p.desc,
		}
		if p.req {
			required = append(required, name)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
	b, _ := json.Marshal(schema)
	return b
}

// alias for clarity in tool definitions
var params = props

// rectProps adds rect coordinates to the property map.
func rectParams(base map[string]prop) json.RawMessage {
	all := map[string]prop{}
	for k, v := range base {
		all[k] = v
	}
	all["x"] = prop{typ: "number", desc: "Left position (% of slide width 0-100)", req: true}
	all["y"] = prop{typ: "number", desc: "Top position (% of slide height 0-100)", req: true}
	all["w"] = prop{typ: "number", desc: "Width (% of slide width 0-100)", req: true}
	all["h"] = prop{typ: "number", desc: "Height (% of slide height 0-100)", req: true}
	return props(all)
}
