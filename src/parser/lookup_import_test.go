package parser

import (
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

func TestQualifiedImportLookupWalksLexicalParentsOnlyAfterSymbolLookup(t *testing.T) {
	graph := &symboltable.FolangSymbols{}
	graph.CreateFolangSymbols()

	root, rootTable := CreateNewContext("", symboltable.S_Program, "lookup-root")
	child, childTable := CreateNewContext(root.Id, symboltable.S_FunctionSymbol, "lookup-child")
	child.ParentCtxSymbolTableId = rootTable.Id
	root.ChildCtxIds = append(root.ChildCtxIds, child.Id)
	target, targetTable := CreateNewContext("", symboltable.S_PackageSymbol, "lookup-target")
	graph.AddContext(root)
	graph.AddContext(child)
	graph.AddContext(target)
	graph.AddSymbolTable(rootTable)
	graph.AddSymbolTable(childTable)
	graph.AddSymbolTable(targetTable)
	root.ImportedContextIds["emp"] = target.Id

	declaration := &symboltable.SymbolDetails{
		SymbolId_: "employee-service", SymbolType_: string(symboltable.S_VarSymbol),
		Name_: "EmployeeService", SymbolTableId: targetTable.Id,
	}
	if _, ok := graph.Declare(targetTable.Id, symboltable.SymbolKey("EmployeeService", declaration.SymbolType_), declaration); !ok {
		t.Fatal("could not declare imported symbol")
	}
	occurrence := &symboltable.ExpressionSymbol{SymbolDetails: symboltable.SymbolDetails{SymbolTableId: childTable.Id}}

	if got := resolvedNameSymbolID("EmployeeService", occurrence, graph); got != "" {
		t.Fatalf("unqualified lookup entered imports and resolved %q", got)
	}
	if got := resolvedNameSymbolID("emp.EmployeeService", occurrence, graph); got != declaration.SymbolId_ {
		t.Fatalf("qualified parent import resolved %q, want %q", got, declaration.SymbolId_)
	}
	rootOnly := &symboltable.SymbolDetails{
		SymbolId_: "consumer-secret", SymbolType_: string(symboltable.S_VarSymbol),
		Name_: "ConsumerSecret", SymbolTableId: rootTable.Id,
	}
	graph.Declare(rootTable.Id, symboltable.SymbolKey("ConsumerSecret", rootOnly.SymbolType_), rootOnly)
	target.ParentId = root.Id // model a project-owned component/package attachment
	target.ParentCtxSymbolTableId = rootTable.Id
	graph.RootContextId = root.Id
	graph.FolContext = &symboltable.FolContext{Id: "lookup-project", SymbolTable_: rootTable.Id, Context_: root.Id, Kind: "application"}
	if got := resolvedNameSymbolID("emp.ConsumerSecret", occurrence, graph); got != "" {
		t.Fatalf("imported lookup leaked into consumer root and resolved %q", got)
	}
}

func TestQualifiedImportLookupUsesLongestDeclaredPackageQualifier(t *testing.T) {
	graph := &symboltable.FolangSymbols{}
	graph.CreateFolangSymbols()
	owner, ownerTable := CreateNewContext("", symboltable.S_Program, "longest-owner")
	company, companyTable := CreateNewContext("", symboltable.S_PackageSymbol, "company")
	hr, hrTable := CreateNewContext("", symboltable.S_PackageSymbol, "company.hr")
	for _, context := range []*symboltable.Context{owner, company, hr} {
		graph.AddContext(context)
	}
	for _, table := range []*symboltable.SymbolTable{ownerTable, companyTable, hrTable} {
		graph.AddSymbolTable(table)
	}
	owner.ImportedContextIds["company"] = company.Id
	owner.ImportedContextIds["company.hr"] = hr.Id
	declaration := &symboltable.SymbolDetails{
		SymbolId_: "employee", SymbolType_: string(symboltable.S_VarSymbol), Name_: "Employee", SymbolTableId: hrTable.Id,
	}
	graph.Declare(hrTable.Id, symboltable.SymbolKey("Employee", declaration.SymbolType_), declaration)
	occurrence := &symboltable.ExpressionSymbol{SymbolDetails: symboltable.SymbolDetails{SymbolTableId: ownerTable.Id}}

	if got := resolvedNameSymbolID("company.hr.Employee", occurrence, graph); got != declaration.SymbolId_ {
		t.Fatalf("longest package qualifier resolved %q, want %q", got, declaration.SymbolId_)
	}
}

func TestProjectSurfaceUsesTransparentOperationalRootImports(t *testing.T) {
	graph := &symboltable.FolangSymbols{}
	graph.CreateFolangSymbols()
	root, rootTable := CreateNewContext("", symboltable.S_Program, "environment-root")
	surface, surfaceTable := CreateNewContext("", symboltable.S_Program, "project-surface")
	function, functionTable := CreateNewContext(surface.Id, symboltable.S_FunctionSymbol, "surface-function")
	function.ParentCtxSymbolTableId = surfaceTable.Id
	surface.ChildCtxIds = append(surface.ChildCtxIds, function.Id)
	target, targetTable := CreateNewContext("", symboltable.S_PackageSymbol, "surface-import-target")
	for _, context := range []*symboltable.Context{root, surface, function, target} {
		graph.AddContext(context)
	}
	for _, table := range []*symboltable.SymbolTable{rootTable, surfaceTable, functionTable, targetTable} {
		graph.AddSymbolTable(table)
	}
	graph.RootContextId = root.Id
	graph.FolContext = &symboltable.FolContext{Id: "surface-project", SymbolTable_: surfaceTable.Id, Context_: root.Id, Kind: "library"}
	root.ImportedContextIds["emp"] = target.Id
	declaration := &symboltable.SymbolDetails{
		SymbolId_: "transparent-service", SymbolType_: string(symboltable.S_VarSymbol),
		Name_: "EmployeeService", SymbolTableId: targetTable.Id,
	}
	graph.Declare(targetTable.Id, symboltable.SymbolKey("EmployeeService", declaration.SymbolType_), declaration)
	occurrence := &symboltable.ExpressionSymbol{SymbolDetails: symboltable.SymbolDetails{SymbolTableId: functionTable.Id}}

	if got := resolvedNameSymbolID("emp.EmployeeService", occurrence, graph); got != declaration.SymbolId_ {
		t.Fatalf("transparent project-root import resolved %q, want %q", got, declaration.SymbolId_)
	}
}
