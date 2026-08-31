//go:build ignore

// jsoncheck validates JSON like buildintro does.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/otter-ppt/otter-ppt/internal/model"
)

func main() {
	data, err := os.ReadFile("examples/adnify_intro.json")
	if err != nil {
		fmt.Println("read:", err)
		return
	}
	fmt.Println("read size:", len(data))
	var pres model.Presentation
	if err := json.Unmarshal(data, &pres); err != nil {
		fmt.Println("unmarshal:", err)
		return
	}
	fmt.Println("ok, slides:", len(pres.Slides))
}
