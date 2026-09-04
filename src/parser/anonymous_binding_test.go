package parser

import (
	"strings"
	"testing"
)

func TestAnonymousFunctionRequiresBindingInitializer(t *testing.T) {
	valid := `_ co.lang.unit = {
    run()->() = {
        stored := (x co.lang.int)->(co.lang.int) { this.return x; };
        result := (x co.lang.int)->(co.lang.int) { this.return x; }(10);
    }
}`
	_, parsed := parsePackageSource(t, valid, "anonymous_binding.unit.fol")
	if len(parsed.diags) != 0 {
		t.Fatalf("bound anonymous functions produced diagnostics: %v", parsed.diags)
	}

	for _, test := range []struct {
		name, statement string
		bindingMessage  bool
	}{
		{"direct argument", `consume((x co.lang.int)->(co.lang.int) { this.return x; });`, true},
		{"direct return", `this.return (x co.lang.int)->(co.lang.int) { this.return x; };`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := `_ co.lang.unit = { run()->() = { ` + test.statement + ` } }`
			_, parsed := parsePackageSource(t, source, "anonymous_binding.unit.fol")
			if len(parsed.diags) == 0 {
				t.Fatal("unbound anonymous function was accepted")
			}
			if !test.bindingMessage {
				return
			}
			found := false
			for _, diagnostic := range parsed.diags {
				if strings.Contains(diagnostic.Error(), "anonymous function must be the value of a variable or function-object binding") {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("diagnostics = %v, want anonymous-function binding diagnostic", parsed.diags)
			}
		})
	}
}
