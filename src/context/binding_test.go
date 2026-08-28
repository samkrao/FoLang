package symboltable

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSymbolTableStoresOnlyOrderedCanonicalIDs(t *testing.T) {
	graph := &FolangSymbols{}
	graph.CreateFolangSymbols()
	table := &SymbolTable{Id: "table", ContextId: "context"}
	graph.AddSymbolTable(table)

	first := &SymbolDetails{SymbolId_: "symbol_a", Name_: "value", SymbolType_: string(S_VarSymbol)}
	second := &SymbolDetails{SymbolId_: "symbol_b", Name_: "work", SymbolType_: string(S_FunctionSymbol)}
	if _, ok := graph.Declare(table.Id, SymbolKey("value", first.SymbolType_), first); !ok {
		t.Fatal("first declaration was rejected")
	}
	if _, ok := graph.Declare(table.Id, FunctionKey("work", "", []string{"int"}), second); !ok {
		t.Fatal("second declaration was rejected")
	}

	if want := []string{"symbol_a", "symbol_b"}; !reflect.DeepEqual(table.SymbolIds, want) {
		t.Fatalf("declaration order = %v, want %v", table.SymbolIds, want)
	}
	if graph.GetSymbol("symbol_a") != first || graph.GetSymbol("symbol_b") != second {
		t.Fatal("table declarations were not registered canonically")
	}
	if got := table.GetVarDetails(*graph, "value"); got != first {
		t.Fatalf("lookup resolved %v, want first symbol", got)
	}

	wire, err := json.Marshal(table)
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]json.RawMessage
	if err := json.Unmarshal(wire, &encoded); err != nil {
		t.Fatal(err)
	}
	if _, duplicated := encoded["Symboldetails"]; duplicated {
		t.Fatal("serialized table duplicated complete symbol records")
	}
	if _, ok := encoded["SymbolIds"]; !ok {
		t.Fatal("serialized table omitted ordered SymbolIds")
	}
	if _, ok := encoded["SymbolsByName"]; !ok {
		t.Fatal("serialized table omitted SymbolsByName index")
	}
}

func TestFolangSymbolsJSONRoundTripUsesPortableRecords(t *testing.T) {
	graph := &FolangSymbols{}
	graph.CreateFolangSymbols()
	graph.RootContextId = "context"
	graph.AddContext(&Context{Id: "context", SymbolTable_: "table"})
	graph.AddSymbolTable(&SymbolTable{Id: "table", ContextId: "context"})
	symbol := &SymbolDetails{
		SymbolId_: "symbol_a", SymbolType_: string(S_SymbolDetails), Name_: "answer",
		Type_: "co.lang.int", State: Resolved, SymbolTableId: "table", IsInternal_: true,
	}
	if _, ok := graph.Declare("table", SymbolKey("answer", symbol.SymbolType_), symbol); !ok {
		t.Fatal("declaration was rejected")
	}

	wire, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	var restored FolangSymbols
	if err := json.Unmarshal(wire, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.RootContextId != graph.RootContextId {
		t.Fatalf("root context = %q, want %q", restored.RootContextId, graph.RootContextId)
	}
	got := restored.GetSymbol("symbol_a")
	if got == nil || got.GetName() != "answer" || got.GetType() != "co.lang.int" || !got.IsInternal() {
		t.Fatalf("portable symbol did not round-trip: %#v", got)
	}
	if bound := restored.Bindings("table")[SymbolKey("answer", symbol.SymbolType_)]; bound != got {
		t.Fatal("restored table index did not resolve through the global registry")
	}
}

func TestRegisterSymbolIncludesReferencedSymbolRecords(t *testing.T) {
	graph := &FolangSymbols{}
	graph.CreateFolangSymbols()
	dependency := &MacroSymbol{SymbolDetails: SymbolDetails{SymbolId_: "dependency", SymbolType_: string(S_MacroSymbol)}}
	macro := &MacroSymbol{
		SymbolDetails: SymbolDetails{SymbolId_: "macro", SymbolType_: string(S_MacroSymbol)},
		Depends:       dependency,
	}
	graph.RegisterSymbol(macro)
	if graph.GetSymbol("macro") != macro || graph.GetSymbol("dependency") != dependency {
		t.Fatal("referenced symbol was not included in the canonical registry")
	}
}
