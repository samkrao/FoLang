package parser

import (
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
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
	graph.AddFolContext(&symboltable.FolContext{Id: "lookup-project", SymbolTable_: rootTable.Id, Context_: root.Id, Kind: "application"})
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

func TestInstanceTypeLookupInheritsAliasesFromItsTypeclass(t *testing.T) {
	graph := &symboltable.FolangSymbols{}
	graph.CreateFolangSymbols()
	root, rootTable := CreateNewContext("", symboltable.S_Program, "typeclass-root")
	contractContext, contractTable := CreateNewContext(root.Id, symboltable.S_TypeclassSymbol, "functor-contract")
	instanceContext, instanceTable := CreateNewContext(root.Id, symboltable.S_InstanceSymbol, "list-functor")
	methodContext, methodTable := CreateNewContext(instanceContext.Id, symboltable.S_FunctionSymbol, "map-method")
	contractContext.ParentCtxSymbolTableId = rootTable.Id
	instanceContext.ParentCtxSymbolTableId = rootTable.Id
	methodContext.ParentCtxSymbolTableId = instanceTable.Id
	for _, context := range []*symboltable.Context{root, contractContext, instanceContext, methodContext} {
		graph.AddContext(context)
	}
	for _, table := range []*symboltable.SymbolTable{rootTable, contractTable, instanceTable, methodTable} {
		graph.AddSymbolTable(table)
	}

	contract := &symboltable.TypeclassSymbol{SymbolDetails: symboltable.SymbolDetails{
		SymbolId_: "functor", SymbolType_: string(symboltable.S_TypeclassSymbol), Name_: "Functor",
		SymbolTableId: rootTable.Id, OwnedContextId: contractContext.Id,
	}, AliasNames: []string{"MapFunction", "InputContainer", "ResultContainer"}}
	graph.Declare(rootTable.Id, symboltable.SymbolKey("Functor", string(symboltable.S_TypeclassSymbol)), contract)
	alias := &symboltable.TypeSymbol{SymbolDetails: symboltable.SymbolDetails{
		SymbolId_: "result-container", SymbolType_: string(symboltable.S_TypeSymbol), Name_: "ResultContainer",
		SymbolTableId: contractTable.Id, Type_: "F(B)",
	}, Alias: true}
	graph.Declare(contractTable.Id, symboltable.SymbolKey("ResultContainer", string(symboltable.S_TypeSymbol)), alias)

	instance := &symboltable.InstanceSymbol{SymbolDetails: symboltable.SymbolDetails{
		SymbolId_: "list-functor", SymbolType_: string(symboltable.S_InstanceSymbol), Name_: "ListFunctor",
		SymbolTableId: rootTable.Id, OwnedContextId: instanceContext.Id,
	}, TypeClassName: "Functor", ForTypes: []string{"co.core.List"}}
	graph.RegisterSymbol(instance)
	instanceContext.OwnerSymbolId = instance.SymbolId_
	methodContext.OwnerSymbolId = "map-method-symbol"
	occurrence := ast.SymbolTypeNode{Value: "ResultContainer", Symb: &symboltable.TypeSymbol{SymbolDetails: symboltable.SymbolDetails{SymbolTableId: methodTable.Id}}}

	if got := resolvedTypeSymbolID(occurrence, graph); got != alias.SymbolId_ {
		t.Fatalf("instance contract alias resolved %q, want %q", got, alias.SymbolId_)
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
	graph.AddFolContext(&symboltable.FolContext{Id: "surface-project", SymbolTable_: surfaceTable.Id, Context_: root.Id, Kind: "library"})
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
