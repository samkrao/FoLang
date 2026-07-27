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

func stringWithArrows(text string, posStart, posEnd Position) string {
	result := ""

	// Calculate indices
	var idxStart int = 0
	if posStart.Col < len(text) {
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
