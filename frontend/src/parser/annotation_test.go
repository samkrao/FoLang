package parser

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestNumericAnnotationValuesDecodeSuffixes(t *testing.T) {
	tests := []struct {
		lexeme string
		want   any
	}{
		{"65u", int64(65)},
		{"42LL", int64(42)},
		{"1.5f", float64(1.5)},
	}

	for _, tc := range tests {
		if got := numericValue(tc.lexeme); got != tc.want {
			t.Errorf("numericValue(%q) = %#v (%T), want %#v (%T)", tc.lexeme, got, got, tc.want, tc.want)
		}
	}
}

func TestAnnotationTypeSpellingPreservesNamedParameterBoundaries(t *testing.T) {
	tokens := normalizeTokens(scanlex.TokenizeQuiet(
		`(a co.lang.int,b co.lang.int)->(result co.lang.int)`, "annotation.fol"))
	p, _ := newParser(tokens)

	got := p.parseAnnotationValue()
	want := `(a co.lang.int,b co.lang.int)->(result co.lang.int)`
	if got != want {
		t.Fatalf("annotation type spelling = %#v, want %q", got, want)
	}
}
