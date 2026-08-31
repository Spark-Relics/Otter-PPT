//go:build ignore

// jsonscan reports the offset reached before a JSON syntax error.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var depthTokens int
	for {
		var v any
		if err := dec.Decode(&v); err != nil {
			off := dec.InputOffset()
			line, col := 1, 0
			for i := 0; i < int(off) && i < len(data); i++ {
				if data[i] == '\n' {
					line++
					col = 0
				} else {
					col++
				}
			}
			fmt.Printf("stopped at offset=%d line=%d col=%d err=%v (tokens=%d)\n", off, line, col, err, depthTokens)
			s := int(off) - 60
			if s < 0 {
				s = 0
			}
			e := int(off) + 60
			if e > len(data) {
				e = len(data)
			}
			fmt.Printf("ctx: %q\n", string(data[s:e]))
			return
		}
		depthTokens++
		if _, err := dec.Token(); err != nil {
			fmt.Println("eof token err:", err)
			return
		}
	}
}
