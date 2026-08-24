package parser_test

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/parser"
)

// TestPredefinedUnregisteredOperatorIsUnsupportedInEveryPosition asserts the
// Disclaimer's operator rule: an operator spelling the language pre-defines but
// has not registered with a meaning — a reserved-operator — raises an
// unsupported-feature diagnostic wherever it is USED, not only where it happens
// to start an operand.
//
// Infix position previously escaped the rule. `a ->> b` left the Pratt loop with
// no infix entry for "->>", so the statement parser reported a missing ";" and
// named the terminator rather than the reserved operator that stopped the parse,
// which is the "silently unrecognised" outcome the reference forbids.
//
// The pre-declared glyphs are deliberately absent from this list. `∪` and `∩` are
// ENABLED operators, not reserved ones: see
// TestPredeclaredGlyphsAreActiveInfixOperators below.
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
		{"declarator-definition", "x ::= 1;\n", "::="},
		{"operand-pipeline", "x := ->> 2;\n", "->>"},
		{"operand-hash", "x := # 2;\n", "#"},
		{"postfix-pipeline", "x := 1 ->>;\n", "->>"},
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

// TestPredeclaredGlyphsAreActiveInfixOperators fixes the difference between a
// pre-declared glyph and a reserved future spelling.
//
// `∪` and `∩` are enabled expression operators: the parser applies their
// language-defined fixity, precedence, associativity and arity, so a use PARSES
// even when no implementation exists. A missing overload fails during operator
// resolution rather than during lexing or parsing
// (docs/language-ref.md, "Pre-Declared Operator Glyphs").
//
// Only their infix position is active. Neither has a prefix or postfix form, so a
// glyph without a left operand is still an error — just not a "reserved" one.
func TestPredeclaredGlyphsAreActiveInfixOperators(t *testing.T) {
	for _, source := range []string{"x := 1 ∪ 2;\n", "x := 1 ∩ 2;\n", "x := 1 ∪ 2 ∩ 3;\n"} {
		source := source
		t.Run("infix"+source, func(t *testing.T) {
			result := parser.ParseFile(source, "glyph", ".", "appl.fol", "")
			if len(result.Diagnostics) != 0 {
				t.Fatalf("%q reported %q; a pre-declared glyph is an enabled infix operator and must parse",
					source, diagnosticText(result.Diagnostics))
			}
		})
	}

	for _, source := range []string{"x := ∪ 2;\n", "x := 1 ∪;\n"} {
		source := source
		t.Run("no-operand"+source, func(t *testing.T) {
			result := parser.ParseFile(source, "glyph", ".", "appl.fol", "")
			if len(result.Diagnostics) == 0 {
				t.Fatalf("%q parsed without a diagnostic; a pre-declared glyph is binary infix and needs both operands", source)
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
