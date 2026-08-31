//go:build ignore

// jsonmin minimally reproduces the decode failure with a small subset.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var pres model.Presentation
	err = json.Unmarshal(data, &pres)
	fmt.Println("full unmarshal err:", err)

	// Try decoding just an element containing a shape
	elem := `{"id":"x","type":"shape","rect":{"x":1},"shape":{"shape_type":"rectangle","fill":{"type":"solid","color":"#111"}}}`
	var e model.Element
	fmt.Println("element unmarshal err:", json.Unmarshal([]byte(elem), &e))

	var p2 model.Presentation
	err = json.Unmarshal(data[:1024], &p2)
	fmt.Println("1KB prefix err:", err)
}
