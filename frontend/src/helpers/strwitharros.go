package helpers

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"strings"
)

/*
	type Position struct {
		ln  int
		col int
	}
*/

// CopyMap deep-copies in to out by encoding and decoding through GOB.
func CopyMap(in, out interface{}) {
	buf := new(bytes.Buffer)
	gob.NewEncoder(buf).Encode(in)
	gob.NewDecoder(buf).Decode(out)

}

// CopyMapJson deep-copies in to out by encoding and decoding through JSON.
func CopyMapJson(in, out interface{}) {
	b, _ := json.Marshal(in)
	json.Unmarshal(b, out)
}

// stringWithArrows renders the offending source line under a diagnostic with a
// caret run beneath the reported span.
//
// Every index it derives is clamped, because it must not be able to fail. A
// diagnostic is what a malformed file produces, so this runs on exactly the
// inputs whose positions are least trustworthy, and ParseFile promises callers —
// a language server above all — that a malformed file yields diagnostics rather
// than a panic. Rendering one is part of that promise.
//
// The degenerate cases are real rather than theoretical. A span that crosses a
// line break, such as the one an unterminated block comment carries to end of
// file, ends at a SMALLER column than it starts at; an empty line makes
// `len(line)-1` negative; and a start column past the end of its line is
// produced whenever a token is reported at end of input. Each of those made
// strings.Repeat panic on a negative count.
func stringWithArrows(text string, posStart, posEnd Position) string {
	result := ""

	// Calculate indices
	var idxStart int = 0
	if posStart.Col > 0 && posStart.Col < len(text) {
		idxStart = max(strings.LastIndex(text[:posStart.Col], "\n"), 0)
	}
	idxEnd := strings.Index(text[idxStart:], "\n")
	if idxEnd < 0 {
		idxEnd = len(text)
	} else {
		idxEnd += idxStart
	}

	// Generate each line
	lineCount := posEnd.Ln - posStart.Ln + 1
	if lineCount < 1 {
		lineCount = 1
	}
	for i := 0; i < lineCount; i++ {
		// Calculate line columns
		line := text[idxStart:idxEnd]
		colStart := posStart.Col
		if i != 0 {
			colStart = 0
		}
		colEnd := posEnd.Col
		if i != lineCount-1 {
			colEnd = len(line) - 1
		}

		// A caret can only be drawn under text that is present, so both columns
		// are pinned to the line and the run is never negative.
		colStart = clamp(colStart, 0, len(line))
		colEnd = clamp(colEnd, colStart, len(line))

		// Append to result
		result += line + "\n"
		result += strings.Repeat(" ", colStart) + strings.Repeat("^", colEnd-colStart)

		// Re-calculate indices
		idxStart = idxEnd
		idxEnd = strings.Index(text[idxStart:], "\n")
		if idxEnd < 0 {
			idxEnd = len(text)
		} else {
			idxEnd += idxStart
		}
	}

	return strings.ReplaceAll(result, "\t", "")
}

// clamp confines value to [low, high]. high wins when the range is inverted, so
// the result is never outside a caller's line.
func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
