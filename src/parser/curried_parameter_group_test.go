package parser

import (
	"strings"
	"testing"
)

func TestCurriedFunctionsRequireEveryParameterGroupToBeNonempty(t *testing.T) {
	for _, source := range []string{
		`_ co.lang.unit = { f()(x co.lang.int)->() = {} }`,
		`_ co.lang.unit = { f(x co.lang.int)()->() = {} }`,
		`_ co.lang.unit = { f()()()()->() = {} }`,
	} {
		result := ParseFile(source, "test", ".", "functions.unit.fol", "pkg")
		joined := ""
		for _, diagnostic := range result.Diagnostics {
			joined += diagnostic.Error()
		}
		if !strings.Contains(joined, "every parameter group of a curried function must contain at least one explicitly typed parameter") {
			t.Errorf("source %q diagnostics = %v", source, result.Diagnostics)
		}
	}
}

func TestSoleEmptyParameterGroupRemainsAZeroArgumentFunction(t *testing.T) {
	result := ParseFile(`_ co.lang.unit = { f()->() = {} }`, "test", ".", "functions.unit.fol", "pkg")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("zero-argument function diagnostics: %v", result.Diagnostics)
	}
}

func TestCurriedTemplateParametersStillRequireExplicitTypes(t *testing.T) {
	result := ParseFile(`_ co.lang.unit = {
    @co.dap.template
    f(x)(y co.lang.int)->() = {}
}`, "test", ".", "functions.unit.fol", "pkg")
	joined := ""
	for _, diagnostic := range result.Diagnostics {
		joined += diagnostic.Error()
	}
	if !strings.Contains(joined, "every parameter in a curried function must have an explicit type") {
		t.Fatalf("untyped curried template diagnostics: %v", result.Diagnostics)
	}
}
