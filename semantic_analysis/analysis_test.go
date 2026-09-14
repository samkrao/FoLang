package semantic_analysis

import (
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
)

func TestAnalyzeRequiresSymbolGraph(t *testing.T) {
	result := Analyze(ast.ProjectStmt{})
	if !result.HasErrors() || result.Diagnostics[0].Code != "SEM0001" {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestFlowReportsReadBeforeInitialization(t *testing.T) {
	graph := &symboltable.FolangSymbols{}
	graph.CreateFolangSymbols()
	graph.AddFolContext(&symboltable.FolContext{Id: "project", Context_: "root"})
	graph.AddContext(&symboltable.Context{Id: "root"})
	variable := &symboltable.VarSymbol{VariableDetails: symboltable.VariableDetails{
		SymbolDetails: symboltable.SymbolDetails{Name_: "x"},
	}}
	function := ast.FunctionDeclarationStmt{Name: "f", Symb: &symboltable.FunctionSymbol{}, Body: []ast.Stmt{
		ast.VarDeclarationStmt{BasicVarStmt: ast.BasicVarStmt{Identifier: "x"}, Symb: variable},
		ast.ExpressionStmt{Expression: ast.SymbolExpr{Value: "x"}},
	}}
	result := Analyze(ast.ProjectStmt{FolangSymbols: graph, EntryStmt: ast.Application{Body: []ast.Stmt{function}}})
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "SEM3002" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected SEM3002, diagnostics = %#v", result.Diagnostics)
	}
}

func TestLookupUsesLongestImportedQualifier(t *testing.T) {
	graph := lookupTestGraph()
	addLookupContext(graph, "co-context", "co-table", "", nil)
	addLookupContext(graph, "co-out-context", "co-out-table", "", nil)
	declareLookupSymbol(graph, "co-table", "out.println", "broad")
	declareLookupSymbol(graph, "co-out-table", "println", "specific")
	graph.GetContext("root").ImportedContextIds = map[string]string{
		"co":     "co-context",
		"co.out": "co-out-context",
	}

	matches := lookup(graph, "use-table", "co.out.println")
	if len(matches) != 1 || matches[0].GetSymbolID() != "specific" {
		t.Fatalf("matches = %#v, want the longest co.out import", matches)
	}
}

func TestLookupFindsImportsOnLexicalParents(t *testing.T) {
	graph := lookupTestGraph()
	addLookupContext(graph, "container", "container-table", "", map[string]string{"emp": "employee-context"})
	addLookupContext(graph, "function", "function-table", "container", nil)
	addLookupContext(graph, "employee-context", "employee-table", "", nil)
	declareLookupSymbol(graph, "employee-table", "Employee", "employee")

	matches := lookup(graph, "function-table", "emp.Employee")
	if len(matches) != 1 || matches[0].GetSymbolID() != "employee" {
		t.Fatalf("matches = %#v, want the import declared by the parent context", matches)
	}
}

func TestLookupUsesTransparentOperationalRootImports(t *testing.T) {
	graph := lookupTestGraph()
	addLookupContext(graph, "co-context", "co-table", "", nil)
	declareLookupSymbol(graph, "co-table", "int", "co-int")
	graph.GetContext("root").ImportedContextIds = map[string]string{"co": "co-context"}

	matches := lookup(graph, "use-table", "co.int")
	if len(matches) != 1 || matches[0].GetSymbolID() != "co-int" {
		t.Fatalf("matches = %#v, want the automatic root co import", matches)
	}
}

func TestLookupDoesNotTraverseAnImportedContextsImports(t *testing.T) {
	graph := lookupTestGraph()
	addLookupContext(graph, "library", "library-table", "", map[string]string{"hidden": "hidden-context"})
	addLookupContext(graph, "hidden-context", "hidden-table", "", nil)
	declareLookupSymbol(graph, "hidden-table", "Secret", "secret")
	graph.GetContext("root").ImportedContextIds = map[string]string{"lib": "library"}

	if matches := lookup(graph, "use-table", "lib.hidden.Secret"); len(matches) != 0 {
		t.Fatalf("matches = %#v, imported-imported contexts must remain private", matches)
	}
}

func lookupTestGraph() *symboltable.FolangSymbols {
	graph := &symboltable.FolangSymbols{}
	graph.CreateFolangSymbols()
	graph.AddFolContext(&symboltable.FolContext{Id: "project", Context_: "root"})
	addLookupContext(graph, "root", "root-table", "", nil)
	addLookupContext(graph, "use", "use-table", "", nil)
	return graph
}

func addLookupContext(graph *symboltable.FolangSymbols, contextID, tableID, parentID string, imports map[string]string) {
	context := &symboltable.Context{Id: contextID, ParentId: parentID, SymbolTable_: tableID, ImportedContextIds: imports}
	if parent := graph.GetContext(parentID); parent != nil {
		context.ParentCtxSymbolTableId = parent.SymbolTable_
	}
	graph.AddContext(context)
	graph.AddSymbolTable(&symboltable.SymbolTable{Id: tableID, ContextId: contextID, SymbolsByName: map[string][]string{}})
}

func declareLookupSymbol(graph *symboltable.FolangSymbols, tableID, name, id string) {
	symbol := &symboltable.VarSymbol{VariableDetails: symboltable.VariableDetails{SymbolDetails: symboltable.SymbolDetails{
		SymbolId_: id, SymbolType_: string(symboltable.S_VarSymbol), Name_: name, SymbolTableId: tableID,
	}}}
	graph.Declare(tableID, symboltable.SymbolKey(name, string(symboltable.S_VarSymbol)), symbol)
}
