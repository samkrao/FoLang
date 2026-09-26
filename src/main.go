// Package main is the entry point for the fo-lang compiler frontend.
package main

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/preparser"
)

func main() {

	preparser.PreParse("", "")
	parser := preparser.Init("")
	ast := parser.Parse("")

	fmt.Sprint(ast)
}
