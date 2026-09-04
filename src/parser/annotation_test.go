package parser

import (
	"testing"

	"github.com/samkrao/fo-lang/src/scanlex"
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

func TestAnnotationDeclarationReferencePreservesOverloadSignature(t *testing.T) {
	tokens := normalizeTokens(scanlex.TokenizeQuiet(
		`find(co.lang.int)->(Employee)`, "annotation.fol"))
	p, _ := newParser(tokens)

	got := p.parseAnnotationValue()
	want := `find(co.lang.int)->(Employee)`
	if got != want {
		t.Fatalf("annotation type spelling = %#v, want %q", got, want)
	}
}

func TestAnnotationValuesRejectInlineTypes(t *testing.T) {
	for _, source := range []string{
		`Vector(co.lang.int)`,
		`co.lang.int->(*)`,
		`(co.lang.int)->(co.lang.bool)`,
	} {
		t.Run(source, func(t *testing.T) {
			tokens := normalizeTokens(scanlex.TokenizeQuiet(source, "annotation.fol"))
			p, _ := newParser(tokens)
			defer func() {
				if recover() == nil {
					t.Fatalf("annotation value %q parsed without a diagnostic", source)
				}
			}()
			p.parseAnnotationValue()
		})
	}
}

func TestGenericAnnotationAliasesAcceptTypeExpressions(t *testing.T) {
	tokens := normalizeTokens(scanlex.TokenizeQuiet(
		`@co.dap.generic(types=[{name=A},{name=B}], aliases=[{name=Mapper,type=(A)->(B)},{name=BoxOfA,type=Box->(A)}])`,
		"annotation.fol"))
	p, _ := newParser(tokens)

	annotations := p.parseAnnotations()
	if len(p.diags) != 0 {
		t.Fatalf("generic aliases produced diagnostics: %v", p.diags)
	}
	if len(annotations.genericAliases) != 2 {
		t.Fatalf("generic alias count = %d, want 2", len(annotations.genericAliases))
	}
	if got := annotations.genericAliases[0].Written; got != `(A)->(B)` {
		t.Fatalf("first alias representation = %q, want %q", got, `(A)->(B)`)
	}
	raw, ok := annotations.option("@co.dap.generic", "aliases")
	if !ok {
		t.Fatal("aliases metadata was not preserved")
	}
	entries := raw.([]any)
	if got := entries[0].(map[string]any)["type"]; got != `(A)->(B)` {
		t.Fatalf("serialized alias representation = %#v", got)
	}
}

func TestGenericAnnotationAliasRejectsForall(t *testing.T) {
	tokens := normalizeTokens(scanlex.TokenizeQuiet(
		`@co.dap.generic(types=[{name=A}], aliases=[{name=Poly,type=forall(T).(T)->(T)}])`,
		"annotation.fol"))
	p, _ := newParser(tokens)
	defer func() {
		if recover() == nil {
			t.Fatal("forall generic alias parsed without a diagnostic")
		}
	}()
	p.parseAnnotations()
}
