//go:build ignore

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := os.Args[1]
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".xml") {
			return nil
		}
		data, _ := os.ReadFile(path)
		dec := xml.NewDecoder(bytes.NewReader(data))
		var syntaxErr error
		for {
			_, e := dec.Token()
			if e != nil {
				syntaxErr = e
				break
			}
		}
		if syntaxErr != nil && syntaxErr.Error() != "EOF" {
			// Re-scan to locate line/col of the error offset.
			offset := dec.InputOffset()
			line, col := offsetToLineCol(string(data), int(offset))
			lines := strings.Split(string(data), "\n")
			ctx := ""
			if line-1 < len(lines) {
				l := lines[line-1]
				lo := col - 40
				if lo < 0 {
					lo = 0
				}
				hi := col + 40
				if hi > len(l) {
					hi = len(l)
				}
				ctx = l[lo:hi]
			}
			fmt.Printf("FAIL %s line=%d col=%d err=%v\n  ctx=%q\n", filepath.Base(path), line, col, syntaxErr, ctx)
		}
		return nil
	})
}

func offsetToLineCol(s string, offset int) (int, int) {
	if offset > len(s) {
		offset = len(s)
	}
	line, col := 1, 1
	for i := 0; i < offset; i++ {
		if s[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
