package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

func TestDynamicBindingDeclaration(t *testing.T) {
	body := parseRegressionBody(t, `value ::= 10;`)
	if len(body) != 1 {
		t.Fatalf("dynamic declaration produced %d statements, want 1", len(body))
	}
	declaration, ok := body[0].(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("dynamic declaration parsed as %T, want ast.VarDeclarationStmt", body[0])
	}
	if declaration.DefinitionOperator != "::=" {
		t.Errorf("definition operator = %q, want ::=", declaration.DefinitionOperator)
	}
	if declaration.Symb == nil || !declaration.Symb.Dynamic {
		t.Fatalf("dynamic declaration symbol = %#v, want Dynamic=true", declaration.Symb)
	}
	if declaration.Symb.Inferred {
		t.Error("dynamic binding must not be marked as a statically inferred binding")
	}
}

func TestDynamicBindingRequiresInitializer(t *testing.T) {
	mustPanic(t, func() {
		parseRegressionBody(t, `value ::=`)
	})
}
