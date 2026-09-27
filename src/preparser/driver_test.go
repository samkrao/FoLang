package preparser

import (
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func TestLexerOperatorsProjectsSemanticRegistry(t *testing.T) {
	symbols := &symboltable.FolangSymbols{}
	symbols.CreateFolangSymbols()
	if err := symbols.Operators.Register("%%", &symboltable.OperatorSymbol{
		Fixity:        symboltable.OperatorInfix,
		Precedence:    60,
		Associativity: symboltable.OperatorLeft,
		Arity:         symboltable.OperatorBinary,
	}); err != nil {
		t.Fatal(err)
	}

	stream := scanlex.NewTokenStream(
		[]byte("left %% right"),
		"test.fol",
		&symbols.Operators,
	)

	for !stream.AtEOF() {
		token := stream.Next()
		if token.Value != "%%" {
			continue
		}
		if token.Kind != scanlex.CUSTOM_OPERATOR || token.SubKind != scanlex.NA {
			t.Fatalf("operator token = (%v, %v), want CUSTOM_OPERATOR/NA", token.Kind, token.SubKind)
		}
		return
	}
	t.Fatal("custom operator token was not produced")
}

func TestPreParseInitializesOperatorRegistry(t *testing.T) {
	symbols := PreParse("install", "project")
	if symbols == nil || symbols.Operators.BySpelling == nil {
		t.Fatal("PreParse returned no initialized operator registry")
	}
}
