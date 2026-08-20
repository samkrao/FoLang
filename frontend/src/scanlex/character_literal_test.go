package scanlex

import (
	"strings"
	"testing"
)

// Character literals — docs/language-ref.md, "Alpha Character and String
// Literals": "A character literal contains exactly one non-backslash character."
//
// There are two ways to break that rule and they are reported apart. Both `''`
// and `'ab'` hold the wrong NUMBER of characters, but a reader looking at `''`
// is not helped by being told it encloses more than one, so the count is named.
func TestMalformedCharacterLiteralNamesTheCount(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   string
	}{
		{`''`, "a character literal contains exactly one character; '' encloses none"},
		{`'ab'`, "a character literal contains exactly one character; 'ab' encloses more than one"},
		{`'abc'`, "a character literal contains exactly one character; 'abc' encloses more than one"},
		{`'  '`, "a character literal contains exactly one character; '  ' encloses more than one"},
	} {
		tc := tc
		t.Run(tc.source, func(t *testing.T) {
			_, findings := TokenizeCollecting("letter := "+tc.source+";", "probe.fol", nil)
			if len(findings) == 0 {
				t.Fatalf("%s produced no finding", tc.source)
			}
			if got := findings[0].AsString(); !strings.Contains(got, tc.want) {
				t.Errorf("finding does not name the count\n  want contains: %s\n  got: %s", tc.want, got)
			}
		})
	}
}

// The rule must not capture the spellings around it. A one-character literal is
// one CHARACTER rather than one byte, a label carries no closing apostrophe, and
// an escape has its own unsupported-feature diagnostic that is more specific
// than the count rule.
func TestMalformedCharacterLiteralLeavesValidSpellingsAlone(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   TokenKind
	}{
		{"ascii", `'A'`, CHAR},
		{"space", `' '`, CHAR},
		{"comma", `','`, CHAR},
		{"non-ascii", `'∪'`, CHAR},
		{"label reference", `'outer`, LABEL_IDENTIFIER},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			toks, findings := TokenizeCollecting(tc.source, "probe.fol", nil)
			if len(findings) != 0 {
				t.Fatalf("%s was reported: %v", tc.source, findings[0].AsString())
			}
			if len(toks) == 0 || toks[0].Kind != tc.want {
				t.Fatalf("%s tokens = %#v, want a leading kind %d", tc.source, toks, tc.want)
			}
		})
	}

	// An escape keeps the diagnostic that names the withdrawn feature rather
	// than being counted as two characters.
	_, findings := TokenizeCollecting(`letter := '\n';`, "probe.fol", nil)
	if len(findings) == 0 {
		t.Fatal(`'\n' produced no finding`)
	}
	if got := findings[0].AsString(); !strings.Contains(got, "escaped character literals are not supported") {
		t.Errorf(`'\n' reported %q, want the unsupported-escape diagnostic`, got)
	}
}
