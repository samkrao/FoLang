package scanlex

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/helpers"
)

func TestLexerDiagnosticNamesAreRegisteredAndSourceIndependent(t *testing.T) {
	_, first := TokenizeCollecting("value \x01 = 1;", "first.fol", nil)
	_, second := TokenizeCollecting("other \x02 = 2;", "second.fol", nil)
	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("lexer did not report invalid bytes: first=%v second=%v", first, second)
	}
	for _, diagnostic := range append(first, second...) {
		if !helpers.IsRegisteredDiagnosticName(diagnostic.DiagnosticName()) {
			t.Fatalf("lexer emitted unregistered diagnostic name %q", diagnostic.DiagnosticName())
		}
		if strings.Contains(diagnostic.DiagnosticName(), "lexer error") {
			t.Fatalf("lexer leaked display text into diagnostic name %q", diagnostic.DiagnosticName())
		}
	}
	if first[0].DiagnosticName() != second[0].DiagnosticName() {
		t.Fatalf("diagnostic names vary with source: %q != %q", first[0].DiagnosticName(), second[0].DiagnosticName())
	}
}
