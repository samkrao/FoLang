package symboltable

import "testing"

func TestSymbolUtilitiesUseCanonicalSymbolKinds(t *testing.T) {
	fs := FolangSymbols{}
	fs.CreateFolangSymbols()

	variable := &Variable{SymbolDetails: SymbolDetails{
		SymbolId_:     "variable-id",
		SymbolType_:   "identifier",
		Name_:         "item",
		SymbolTableId: "table",
	}}
	function := &FunctionSymbol{AbstractFunctionShape: AbstractFunctionShape{SymbolDetails: SymbolDetails{
		SymbolId_:     "function-id",
		SymbolType_:   "callable",
		Name_:         "transform",
		SymbolTableId: "table",
	}}}
	fs.RegisterSymbol(variable)
	fs.RegisterSymbol(function)

	table := &SymbolTable{
		Id:        "table",
		SymbolIds: []SymbolID{variable.GetSymbolID(), function.GetSymbolID()},
		SymbolsByName: map[SymbolName][]SymbolID{
			"item_identifier":       {variable.GetSymbolID()},
			"transform_Fun(co.int)": {function.GetSymbolID()},
		},
	}
	fs.AddSymbolTable(table)

	if got := table.GetVarDetails(fs, "item"); got.GetSymbolID() != variable.GetSymbolID() {
		t.Fatalf("GetVarDetails returned %q, want %q", got.GetSymbolID(), variable.GetSymbolID())
	}
	if got := table.GetFunDetails(fs, "transform"); got.GetSymbolID() != function.GetSymbolID() {
		t.Fatalf("GetFunDetails returned %q, want %q", got.GetSymbolID(), function.GetSymbolID())
	}
	if !table.Exists(fs, "item", "identifier") {
		t.Fatal("Exists did not match the symbol's current SymbolType")
	}
	if !table.ExistsVar(fs, "item") || !table.ExistsFun(fs, "transform") {
		t.Fatal("typed existence lookup did not find registered symbols")
	}
}

func TestSymbolUtilitiesWalkParentTablesSafely(t *testing.T) {
	fs := FolangSymbols{}
	fs.CreateFolangSymbols()

	parentVariable := &Variable{SymbolDetails: SymbolDetails{
		SymbolId_:     "parent-variable",
		Name_:         "value",
		SymbolTableId: "parent",
	}}
	localFunction := &FunctionSymbol{AbstractFunctionShape: AbstractFunctionShape{SymbolDetails: SymbolDetails{
		SymbolId_:     "local-function",
		Name_:         "value",
		SymbolTableId: "child",
	}}}
	fs.RegisterSymbol(parentVariable)
	fs.RegisterSymbol(localFunction)

	parent := &SymbolTable{
		Id:            "parent",
		SymbolIds:     []SymbolID{parentVariable.GetSymbolID()},
		SymbolsByName: map[SymbolName][]SymbolID{"value_Var": {parentVariable.GetSymbolID()}},
	}
	child := &SymbolTable{
		Id:            "child",
		ParentId:      parent.Id,
		SymbolIds:     []SymbolID{localFunction.GetSymbolID()},
		SymbolsByName: map[SymbolName][]SymbolID{"value_Fun()": {localFunction.GetSymbolID()}},
	}
	fs.AddSymbolTable(parent)
	fs.AddSymbolTable(child)

	if got := child.GetVarDetails(fs, "value"); got.GetSymbolID() != parentVariable.GetSymbolID() {
		t.Fatalf("variable lookup stopped at a same-named function: got %q", got.GetSymbolID())
	}

	parent.ParentId = child.Id
	if got := child.GetVarDetails(fs, "missing"); got.GetSymbolID() != "" {
		t.Fatalf("cyclic missing lookup returned %q", got.GetSymbolID())
	}
}

func TestSymbolUtilitiesFollowContextBoundaryAndHideInternalSymbols(t *testing.T) {
	fs := FolangSymbols{}
	fs.CreateFolangSymbols()

	internal := &Variable{SymbolDetails: SymbolDetails{
		SymbolId_:     "internal-variable",
		Name_:         "compilerValue",
		IsInternal_:   true,
		SymbolTableId: "outer",
	}}
	fs.RegisterSymbol(internal)

	outer := &SymbolTable{
		Id:            "outer",
		SymbolIds:     []SymbolID{internal.GetSymbolID()},
		SymbolsByName: map[SymbolName][]SymbolID{"compilerValue_Var": {internal.GetSymbolID()}},
	}
	inner := &SymbolTable{Id: "inner", ContextId: "inner-context"}
	ctx := &Context{Id: "inner-context", ParentCtxSymbolTableId: outer.Id}
	fs.AddSymbolTable(outer)
	fs.AddSymbolTable(inner)
	fs.AddContext(ctx)

	if got := inner.GetVarDetails(fs, "compilerValue"); got.GetSymbolID() != internal.GetSymbolID() {
		t.Fatalf("context-boundary lookup returned %q, want %q", got.GetSymbolID(), internal.GetSymbolID())
	}
	if inner.ExistsVar(fs, "compilerValue") {
		t.Fatal("ExistsVar exposed an internal symbol")
	}
}
