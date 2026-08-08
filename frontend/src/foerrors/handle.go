// Package foerrors provides error handling utilities for the fo-lang compiler.
package foerrors

import (
	"fmt"
	"os"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// MaxParseErrors is the maximum number of diagnostics a single parse collects
// before it stops recording more.
//
// The cap exists so that one early mistake whose cascade never resynchronises
// cannot make the parser allocate without bound. Reaching it is reported by
// Truncated on the diagnostic list, so a caller can tell "these are all the
// problems" apart from "these are the first fifty".
const MaxParseErrors = 50

// GenPanic controls whether HandleErrors panics instead of calling os.Exit.
//
// It exists for the batch compiler and its tests, and it is process-global, so
// it is not a safe lever for a library consumer. Anything embedding the frontend
// — a language server above all — should call the non-fatal entry points
// (parser.ParseFile, scanlex.TokenizeQuiet) and never touch this.
var GenPanic bool = false

// HandleErrors prints any non-nil errors and terminates the process if errors
// are present.
//
// This is the BATCH path: it is how the command-line compiler stops. It never
// returns when a real error is present. Callers that must survive a malformed
// file use the non-fatal entry points instead.
//
// Diagnostics go to stderr, not stdout. The frontend is a library, and a
// consumer that speaks a protocol over stdout cannot have compiler output
// interleaved into that stream.
func HandleErrors(errors ...helpers.ErrorInterface) bool {
	var flag bool = false
	if errors != nil {
		reported := 0
		for _, err := range errors {
			if err == nil {
				continue
			}
			fmt.Fprintln(os.Stderr, err.AsString())
			flag = true
			reported++
		}
		if flag {
			// Report the count that was actually produced. The old message
			// claimed the fifty-error cap had been exceeded no matter how many
			// diagnostics there were, so a single typo announced "too many
			// errors (50)".
			if reported >= MaxParseErrors {
				fmt.Fprintf(os.Stderr,
					"too many errors (%d), stopping parsing\n", reported)
			} else {
				fmt.Fprintf(os.Stderr, "%s found, stopping\n", pluralErrors(reported))
			}
			if !GenPanic {
				os.Exit(1)
			}
			panic("Error")
		}
	}
	return true
}

// pluralErrors renders a diagnostic count in words.
func pluralErrors(n int) string {
	if n == 1 {
		return "1 error"
	}
	return fmt.Sprintf("%d errors", n)
}
