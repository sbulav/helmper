package main

import (
	"fmt"
	"os"

	"github.com/ChristofferNissen/helmper/internal"
)

func main() {
	if err := internal.Program(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
