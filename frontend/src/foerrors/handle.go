// Package foerrors provides error handling utilities for the fo-lang compiler.
package foerrors

import (
	"fmt"
	"os"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// MaxParseErrors is the maximum number of errors the parser will collect before stopping.
const MaxParseErrors = 50

// GenPanic controls whether HandleErrors panics instead of calling os.Exit.
// Set to true in tests so errors can be recovered rather than terminating the process.
var GenPanic bool = false

// HandleErrors prints any non-nil errors and terminates the process if errors are present.
func HandleErrors(errors ...helpers.ErrorInterface) bool {

	var flag bool = false
	if errors != nil {
		for _, err := range errors {

			if err != nil {
				fmt.Println(err.AsString())
				flag = true
			}
		}
		fmt.Fprintf(os.Stderr,
			"too many errors (%d), stopping parsing\n", MaxParseErrors)
		if flag && !GenPanic {
			os.Exit(1)
		} else if flag && GenPanic {
			panic("Error")
		}
	}
	return true

}
