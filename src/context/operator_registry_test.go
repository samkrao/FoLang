package symboltable

import "testing"

func TestOperatorRegistryIsInitializedAndRejectsDuplicates(t *testing.T) {
	symbols := &FolangSymbols{}
	symbols.CreateFolangSymbols()
	if symbols.Operators.BySpelling == nil {
		t.Fatal("CreateFolangSymbols did not initialize the operator registry")
	}

	declaration := &OperatorSymbol{
		Fixity:        OperatorInfix,
		Precedence:    60,
		Associativity: OperatorLeft,
		Arity:         OperatorBinary,
	}
	if err := symbols.Operators.Register("%%", declaration); err != nil {
		t.Fatalf("Register returned an error: %v", err)
	}
	if got, ok := symbols.Operators.Lookup("%%"); !ok || got != declaration {
		t.Fatalf("Lookup returned (%p, %v), want (%p, true)", got, ok, declaration)
	}
	if err := symbols.Operators.Register("%%", &OperatorSymbol{}); err == nil {
		t.Fatal("duplicate operator registration succeeded")
	}
}
