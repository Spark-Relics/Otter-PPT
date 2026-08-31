//go:build ignore

// jsondepth prints depth at the start of each line (outside strings),
// to locate where bracket balance goes wrong.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	// compute cumulative depth per char
	inString, escape := false, false
	depths := make([]int, len(data))
	depth := 0
	for i, c := range data {
		switch {
		case escape:
			escape = false
		case inString && c == '\\':
			escape = true
		case c == '"':
			inString = !inString
		case inString:
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		}
		depths[i] = depth
	}
	// print depth after each line
	offset := 0
	for ln, l := range lines {
		off2 := offset + len(l)
		d := depths[off2-1]
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "\"id\"") || strings.HasPrefix(trimmed, "\"slides\"") || trimmed == "{" || trimmed == "}" || strings.HasPrefix(trimmed, "\"title\"") {
			if strings.Contains(trimmed, "id") || trimmed == "{" || trimmed == "}" || strings.HasPrefix(trimmed, "\"slides\"") || strings.HasPrefix(trimmed, "\"title\"") {
				fmt.Printf("line %4d depth %2d | %s\n", ln+1, d, trimmed)
			}
		}
		offset = off2 + 1
	}
	fmt.Println("final:", depth)
}
