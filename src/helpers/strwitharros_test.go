package helpers

import (
	"strings"
	"testing"
)

// Diagnostic rendering must not be able to fail.
//
// A diagnostic exists BECAUSE the source is malformed, so the caret renderer
// runs on exactly the positions least likely to be well formed, and parser.
// ParseFile promises an embedding consumer — a language server above all — that
// a malformed file yields diagnostics instead of a panic. A renderer that
// panics on a degenerate span breaks that promise at the last possible moment,
// after the diagnostic has already been produced.
//
// Each case below made strings.Repeat panic with a negative count. The
// unterminated block comment is the one a user actually types.
func TestStringWithArrowsSurvivesDegenerateSpans(t *testing.T) {
	const text = "_ co.lang.unit = {\n    subject()->() = {\n    x := 1; /* never closed\n    }\n}"

	for _, tc := range []struct {
		name       string
		start, end Position
	}{
		{
			// An unterminated block comment runs to end of file, so its span
			// ends at a SMALLER column than it starts at.
			name:  "span ends left of where it starts",
			start: Position{Ln: 3, Col: 12},
			end:   Position{Ln: 5, Col: 1},
		},
		{
			name:  "end column precedes start column on one line",
			start: Position{Ln: 1, Col: 10},
			end:   Position{Ln: 1, Col: 2},
		},
		{
			name:  "start column past the end of its line",
			start: Position{Ln: 1, Col: 400},
			end:   Position{Ln: 1, Col: 401},
		},
		{
			name:  "negative columns",
			start: Position{Ln: 1, Col: -5},
			end:   Position{Ln: 1, Col: -1},
		},
		{
			name:  "end line precedes start line",
			start: Position{Ln: 4, Col: 0},
			end:   Position{Ln: 1, Col: 0},
		},
		{
			name:  "empty source",
			start: Position{Ln: 1, Col: 0},
			end:   Position{Ln: 1, Col: 0},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source := text
			if tc.name == "empty source" {
				source = ""
			}

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("rendering a diagnostic panicked: %v", r)
				}
			}()

			// The result is unconstrained beyond not crashing and not inventing
			// carets: a degenerate span has no correct caret run to draw.
			got := stringWithArrows(source, tc.start, tc.end)
			if strings.Count(got, "^") > len(source) {
				t.Errorf("caret run is longer than the source it underlines:\n%s", got)
			}
		})
	}
}

// The ordinary case must keep rendering exactly as before.
func TestStringWithArrowsUnderlinesAnOrdinarySpan(t *testing.T) {
	const text = "alpha\nbravo\ncharlie"

	got := stringWithArrows(text, Position{Ln: 1, Col: 1}, Position{Ln: 1, Col: 4})
	want := "alpha\n ^^^"
	if got != want {
		t.Errorf("stringWithArrows = %q, want %q", got, want)
	}
}
