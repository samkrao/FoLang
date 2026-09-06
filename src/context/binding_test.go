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
	graph.FolProject = &FolProject{Id: "project", SymbolTable_: "table", Context_: "context", Kind: "application"}
	graph.AddContext(&Context{Id: "context", SymbolTable_: "table"})
	graph.AddSymbolTable(&SymbolTable{Id: "table", ContextId: "context"})
	symbol := &SymbolDetails{
		SymbolId_: "symbol_a", SymbolType_: string(S_SymbolDetails), Name_: "answer",
		Type_: "co.lang.int", SymbolTableId: "table", IsInternal_: true,
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
	if restored.FolProject == nil || !reflect.DeepEqual(restored.FolProject, graph.FolProject) {
		t.Fatalf("FoLang project descriptor did not round-trip: %#v", restored.FolProject)
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

// Undeclare is the parser's speculation rollback. The registry, not the table, is
// where a record lives now, so unbinding a name has to reach it too: a record left
// behind is referenced by nothing and still travels to the backend inside the
// artifact, presenting itself there as a declaration the program never made.
func TestUndeclareRemovesTheRecordFromTheRegistry(t *testing.T) {
	graph := &FolangSymbols{}
	graph.CreateFolangSymbols()
	table := &SymbolTable{Id: "table", ContextId: "context"}
	graph.AddSymbolTable(table)

	symbol := &SymbolDetails{SymbolId_: "symbol_a", Name_: "value", SymbolType_: string(S_VarSymbol)}
	key := SymbolKey("value", symbol.SymbolType_)
	if _, ok := graph.Declare(table.Id, key, symbol); !ok {
		t.Fatal("declaration was rejected")
	}

	graph.Undeclare(table.Id, key)

	if got := graph.GetSymbol("symbol_a"); got != nil {
		t.Fatalf("rolled-back declaration survives in the registry: %#v", got)
	}
	if len(table.SymbolIds) != 0 {
		t.Fatalf("declaration order = %v, want empty", table.SymbolIds)
	}
	if _, present := table.SymbolsByName[key]; present {
		t.Fatal("rolled-back declaration survives in the name index")
	}
}

// One key holds one id today, but SymbolsByName is a slice so that an overload
// family may bind siblings together. A partial unwind would leave the order list
// and the name index disagreeing about what the segment contains.
func TestUndeclareUnwindsEverySiblingUnderOneKey(t *testing.T) {
	graph := &FolangSymbols{}
	graph.CreateFolangSymbols()
	table := &SymbolTable{Id: "table", ContextId: "context", SymbolsByName: map[string][]string{}}
	graph.AddSymbolTable(table)

	key := FunctionFamily("work", "")
	for _, id := range []string{"symbol_a", "symbol_b"} {
		symbol := &SymbolDetails{SymbolId_: id, Name_: "work", SymbolType_: string(S_FunctionSymbol)}
		graph.RegisterSymbol(symbol)
		table.SymbolsByName[key] = append(table.SymbolsByName[key], id)
		table.SymbolIds = append(table.SymbolIds, id)
	}

	graph.Undeclare(table.Id, key)

	if len(table.SymbolIds) != 0 {
		t.Fatalf("declaration order = %v, want empty", table.SymbolIds)
	}
	if graph.GetSymbol("symbol_a") != nil || graph.GetSymbol("symbol_b") != nil {
		t.Fatal("a sibling survived the rollback")
	}
}
