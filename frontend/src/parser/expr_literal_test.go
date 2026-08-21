package parser

import (
	"math"
	"strings"
	"testing"
)

// TestFloatLexemeStaysFinite pins the NumberLiteral.Value invariant. An
// out-of-range floating literal is a typing/representation concern rather than
// malformed syntax, so it must still parse — but the float64 convenience slot
// has to hold a finite value, exactly as IntegerLiteral.Value holds zero for a
// literal too wide for int64. The authoritative lexeme lives on the expression
// symbol either way.
func TestFloatLexemeStaysFinite(t *testing.T) {
	tests := []struct {
		lexeme string
		want   float64
	}{
		{"1e5", 100000},
		{"3.14", 3.14},
		{"2.5f32", 2.5},
		{"0x1.8p3", 12},
		// Overflow: ParseFloat reports ±Inf with strconv.ErrRange.
		{"1e9999", 0},
		{"1.7976931348623159e308", 0},
		// Underflow already yields a finite zero and needs no special case.
		{"1e-9999", 0},
	}

	for _, test := range tests {
		t.Run(test.lexeme, func(t *testing.T) {
			value, ok := parseFloatLexeme(test.lexeme)
			if !ok {
				t.Fatalf("parseFloatLexeme(%q) rejected an accepted floating literal", test.lexeme)
			}
			if math.IsInf(value, 0) || math.IsNaN(value) {
				t.Fatalf("parseFloatLexeme(%q) = %v, want a finite value", test.lexeme, value)
			}
			if value != test.want {
				t.Fatalf("parseFloatLexeme(%q) = %v, want %v", test.lexeme, value, test.want)
			}
		})
	}
}

// TestOutOfRangeFloatLiteralSerializes covers the failure the invariant exists
// to prevent. A non-finite float64 has no JSON encoding, so storing one made
// the driver's AST emission fail with "json: unsupported value: +Inf" — an
// internal error carrying no source position, for a literal the grammar
// accepts. This drives the real emission path rather than the helper.
func TestOutOfRangeFloatLiteralSerializes(t *testing.T) {
	const source = "wideFloat := 1e9999;\nnegWide := -1e9999;\n"

	root, _, ctx, _ := Parse(source, "literals", ".", "literals.fol", "", "program", "program", true)
	if root == nil {
		t.Fatal("parsing an out-of-range floating literal produced no AST")
	}

	// A zero astArtifact writes nothing: this exercises the encoding, and a test
	// has no project tree to drop a build/ domain into.
	encoded, written, err := serializeAST(root, ctx, nil, false, astArtifact{})
	if err != nil {
		t.Fatalf("serializing an out-of-range floating literal: %v", err)
	}
	if written != "" {
		t.Errorf("serializeAST wrote %q with no destination configured", written)
	}

	// encoding/json rejects a non-finite float outright, so the error above is
	// the real guard. These check the JSON *value* forms a hand-rolled encoder
	// might emit instead; matching on a bare "Inf" would false-positive on field
	// names such as "Inferred" and "TypeInfo".
	for _, leak := range []string{`:+Inf`, `:-Inf`, `:NaN`, `"+Inf"`, `"-Inf"`, `"NaN"`} {
		if strings.Contains(encoded, leak) {
			t.Fatalf("serialized AST leaked the non-finite value %s", leak)
		}
	}
}
