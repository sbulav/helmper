package main

import (
	"os"

	"github.com/ChristofferNissen/helmper/internal"
)

func main() {
	if err := internal.Program(os.Args); err != nil {
		os.Exit(1)
	}
}
