package parser_test

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// TestPredefinedUnregisteredOperatorIsUnsupportedInEveryPosition asserts the
// Disclaimer's operator rule: an operator spelling the language pre-defines but
// has not registered with a meaning — a reserved-operator or a
// predeclared-operator-glyph — raises an unsupported-feature diagnostic wherever
// it is USED, not only where it happens to start an operand.
//
// Infix position previously escaped the rule. `a ->> b` left the Pratt loop with
// no infix entry for "->>", so the statement parser reported a missing ";" and
// named the terminator rather than the reserved operator that stopped the parse,
// which is the "silently unrecognised" outcome C.7 forbids.
func TestPredefinedUnregisteredOperatorIsUnsupportedInEveryPosition(t *testing.T) {
	cases := []struct {
		name   string
		source string
		symbol string
	}{
		{"infix-pipeline", "x := 1 ->> 2;\n", "->>"},
		{"infix-bidirectional", "x := 1 <-> 2;\n", "<->"},
		{"infix-hash", "x := 1 # 2;\n", "#"},
		{"infix-backtick", "x := 1 ` 2;\n", "`"},
		{"infix-backslash", "x := 1 \\ 2;\n", "\\"},
		{"infix-glyph", "x := 1 ∪ 2;\n", "∪"},
		{"declarator-definition", "x ::= 1;\n", "::="},
		{"operand-pipeline", "x := ->> 2;\n", "->>"},
		{"operand-hash", "x := # 2;\n", "#"},
		{"operand-glyph", "x := ∪ 2;\n", "∪"},
		{"postfix-pipeline", "x := 1 ->>;\n", "->>"},
		{"postfix-glyph", "x := 1 ∪;\n", "∪"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			result := parser.ParseFile(c.source, "reserved", ".", "appl.fol", "")
			if len(result.Diagnostics) == 0 {
				t.Fatalf("%q parsed without a diagnostic; a pre-defined unregistered operator must be rejected on use", c.source)
			}
			text := diagnosticText(result.Diagnostics)
			if !strings.Contains(text, "reserved") || !strings.Contains(text, c.symbol) {
				t.Errorf("%q reported %q; want a diagnostic naming %q as reserved",
					c.source, text, c.symbol)
			}
		})
	}
}

// diagnosticText joins every diagnostic's rendered form so a test can assert on
// what the user is actually shown.
func diagnosticText(diagnostics []helpers.ErrorInterface) string {
	parts := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		parts = append(parts, d.AsString())
	}
	return strings.Join(parts, "\n")
}
