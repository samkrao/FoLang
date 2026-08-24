// Package main is the entry point for the fo-lang compiler frontend.
package main

import (
	"os"

	entry "github.com/samkrao/fo-lang/src/entry"
)

func main() {
	entry.Run(os.Args)
}
