//go:build ignore

// zippeek lists media entries inside a .pptx and checks slide1 rels.
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	r, err := zip.OpenReader(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if strings.Contains(f.Name, "media") {
			fmt.Println("MEDIA:", f.Name, f.FileInfo().Size())
			rc, _ := f.Open()
			head := make([]byte, 60)
			n, _ := io.ReadFull(rc, head)
			fmt.Printf("  head: %q\n", string(head[:n]))
			rc.Close()
		}
	}
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "slides/_rels/slide2.xml.rels") {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			fmt.Println("SLIDE2 RELS:", string(b))
			rc.Close()
		}
	}
}
