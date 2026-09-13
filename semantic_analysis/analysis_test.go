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
