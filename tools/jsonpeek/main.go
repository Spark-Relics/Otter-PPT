//go:build ignore

// jsonpeek reports the size and last bytes of a file.
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
	fmt.Println("size:", len(data))
	fmt.Printf("first bytes: %q\n", string(data[:20]))
	tail := 40
	if len(data) < tail {
		tail = len(data)
	}
	fmt.Printf("last bytes: %q\n", string(data[len(data)-tail:]))
}
