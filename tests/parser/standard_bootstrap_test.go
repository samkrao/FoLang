package parser_test

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/parser"
)

const ordinaryPrivateReference = `_ co.unit = {
    Value co.type = fΦλ.lang.int;
}`

const standardPrivateReference = `_ fΦλ.lang.unit = {
    Value fΦλ.lang.type = fΦλ.lang.int;
}`

func TestOrdinarySourceRejectsStandardPrivateNamespace(t *testing.T) {
	result := parser.ParseFile(ordinaryPrivateReference, "ordinary", ".", "types.unit.fol", "pkg")
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

func TestStandardBootstrapAcceptsProjectionSurface(t *testing.T) {
	source := `_ fΦλ.lang.component = {
    @co.dap.export(
        packages={forall={recurse=true}, for={recurse=true}, let={recurse=true}, self={recurse=true}, this={recurse=true}, fΦλ.lang={recurse=true}},
        as=co
    )
    @co.dap.export(package=fΦλ.const={recurse=true}, as=co.const)
}`
	result := parser.ParseStandardBootstrapFile(source, "standard", ".", "component.fol", "")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("standard bootstrap rejected projection surface: %s", diagnosticText(result.Diagnostics))
	}
}
