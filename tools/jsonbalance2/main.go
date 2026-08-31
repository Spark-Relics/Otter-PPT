//go:build ignore

// jsonbalance2 prints every line where depth transitions to a suspicious value.
package main

import (
	"fmt"
	"os"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	inString, escape := false, false
	depth := 0
	line := 1
	depthAtLineStart := 0
	prevLine := 0
	for i, c := range data {
		if line != prevLine {
			depthAtLineStart = depth
			prevLine = line
		}
		switch {
		case escape:
			escape = false
		case inString && c == '\\':
			escape = true
		case c == '"':
			inString = !inString
		case c == '\n':
			line++
		case inString:
			// ignore
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		}
	}
	// walk again printing depth at each element id line
	inString, escape = false, false
	depth, line = 0, 1
	idLine := map[int]string{}
	cur := ""
	for i, c := range data {
		switch {
		case escape:
			escape = false
		case inString && c == '\\':
			escape = true
		case c == '"':
			if !inString {
				cur = ""
			} else {
				if len(cur) < 30 {
					idLine[line] = cur
				}
			}
			inString = !inString
		case c == '\n':
			line++
		case inString:
			if len(cur) < 40 {
				cur += string(c)
			}
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		}
		_ = i
		_ = depthAtLineStart
	}
	fmt.Println("final depth:", depth)
}
