//go:build ignore

// jsonbalance reports running depth of {} and [] outside strings,
// printing line numbers where depth returns to unexpected values.
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
	for i, c := range data {
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
			if depth == 0 {
				fmt.Printf("depth 0 at line %d offset %d\n", line, i)
			}
			if depth < 0 {
				fmt.Printf("NEGATIVE depth at line %d offset %d\n", line, i)
				return
			}
		}
	}
	fmt.Println("final depth:", depth, "inString:", inString, "total lines:", line)
}
