package parser_test

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/parser"
)

const standardPrivateReference = `_ co.unit = {
    Value co.type = fΦλ.lang.int;
}`

func TestOrdinarySourceRejectsStandardPrivateNamespace(t *testing.T) {
	result := parser.ParseFile(standardPrivateReference, "ordinary", ".", "types.unit.fol", "pkg")
	if len(result.Diagnostics) == 0 {
		t.Fatal("ordinary source accepted compiler-private fΦλ.* reference")
	}
	if got := diagnosticText(result.Diagnostics); !strings.Contains(got, "only while building co.folenc") {
		t.Fatalf("diagnostic = %q, want standard-bootstrap restriction", got)
	}
}

func TestStandardBootstrapAcceptsPrivateNamespace(t *testing.T) {
	result := parser.ParseStandardBootstrapFile(standardPrivateReference, "standard", ".", "types.unit.fol", "fΦλ.lang")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("standard bootstrap rejected private reference: %s", diagnosticText(result.Diagnostics))
	}
}
