package parser

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/project"
)

// The project tree ParseProject assembles.
//
// The shape under test is the one the layout dictates rather than the one any
// single file produces: a package per FOLDER, an ordinary unit spliced into it, a
// companion unit folded into the type it belongs to, and one scope model for the
// whole thing.

// projectFixture writes a small but complete project and returns its root.
//
// It exercises every placement rule at once: an entry surface, a package with a
// primary declaration and its companion, an ordinary unit beside them, a nested
// subpackage, and a source library.
func projectFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	write := func(relative, contents string) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(project.MarkerFilename, "")
	write("src/appl.fol", `main()->() = {
    co.out.println("hello");
}`)
	write("src/hr/Employee.fol", `_ co.lang.struct = {
    id co.lang.int;
}`)
	write("src/hr/Employee.comp.unit.fol", `_ co.lang.unit = {
    promote(e Employee)->() = { }
}`)
	write("src/hr/rules.unit.fol", `_ co.lang.unit = {
    eligible()->(co.lang.bool) = { this.return co.const.true; }
}`)
	write("src/hr/payroll/Rate.fol", `_ co.lang.struct = {
    amount co.lang.float;
}`)
	return root
}

// parseFixtureProject parses the fixture and returns its root node.
func parseFixtureProject(t *testing.T) ast.ProjectStmt {
	t.Helper()
	root := projectFixture(t)

	parsed, _, err := ParseProject(root)
	if err != nil {
		t.Fatalf("ParseProject: %v", err)
	}
	stmt, isProject := parsed.(ast.ProjectStmt)
	if !isProject {
		t.Fatalf("ParseProject returned %T, want ast.ProjectStmt", parsed)
	}
	return stmt
}

// TestParseProjectReturnsOneProjectStatement covers the contract the whole layer
// exists for: whatever the project's shape, one statement comes back and it is the
// project.
func TestParseProjectReturnsOneProjectStatement(t *testing.T) {
	stmt := parseFixtureProject(t)

	if stmt.EntryStmt == nil {
		t.Error("the project has no entry statement; src/appl.fol is its structural surface")
	}
	if _, isApplication := stmt.EntryStmt.(ast.Application); !isApplication {
		t.Errorf("the entry statement is %T, want ast.Application for src/appl.fol", stmt.EntryStmt)
	}
	if stmt.IsLibrary {
		t.Error("a project whose surface is appl.fol is an application, not a library")
	}
	if stmt.FolangSymbols == nil {
		t.Fatal("the project carries no symbol model, so the ids in its tree resolve to nothing")
	}
}

func TestOrdinaryUnitOverloadRestrictionsAreValidatedInPackageContext(t *testing.T) {
	root := projectFixture(t)
	first := filepath.Join(root, "src", "hr", "format_int.unit.fol")
	second := filepath.Join(root, "src", "hr", "format_float.unit.fol")
	if err := os.WriteFile(first, []byte("_ co.lang.unit = {\nformat(~value co.lang.int)->(co.lang.string) = { this.return \"int\"; }\n}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("_ co.lang.unit = {\nformat(~value co.lang.float)->(co.lang.string) = { this.return \"float\"; }\n}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, diagnostics, err := ParseProject(root)
	if err != nil {
		t.Fatalf("ParseProject: %v", err)
	}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Error(), "format cannot be overloaded across ordinary unit files") &&
			strings.Contains(diagnostic.Error(), "named parameters") {
			return
		}
	}
	t.Fatalf("diagnostics = %v, want package-owned restricted-overload error", diagnostics)
}

func TestOrdinaryUnitOverloadsSharePackageFamily(t *testing.T) {
	root := projectFixture(t)
	for filename, source := range map[string]string{
		"format_int.unit.fol":   "_ co.lang.unit = {\nformat(value co.lang.int)->(co.lang.string) = { this.return \"int\"; }\n}",
		"format_float.unit.fol": "_ co.lang.unit = {\nformat(value co.lang.float)->(co.lang.string) = { this.return \"float\"; }\n}",
	} {
		if err := os.WriteFile(filepath.Join(root, "src", "hr", filename), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, diagnostics, err := ParseProject(root)
	if err != nil {
		t.Fatalf("ParseProject: %v", err)
	}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Error(), "format cannot be overloaded") ||
			strings.Contains(diagnostic.Error(), "every declaration of format") ||
			strings.Contains(diagnostic.Error(), "format is already declared") {
			t.Fatalf("valid cross-unit overload family produced diagnostic: %v", diagnostic)
		}
	}
}

// TestProjectBodyHoldsOnePackagePerFolder checks that the package tree keeps the
// shape of the folder tree, with a subfolder reachable through its parent rather
// than as a second top-level package.
func TestProjectBodyHoldsOnePackagePerFolder(t *testing.T) {
	stmt := parseFixtureProject(t)

	if len(stmt.PackageStmts) != 1 {
		t.Fatalf("the project has %d top-level packages, want 1 for src/hr", len(stmt.PackageStmts))
	}

	node, known := stmt.PackageStmts["hr"]
	if !known {
		t.Fatalf("the project's top-level packages are %v, want hr", topLevelNames(stmt))
	}
	hr, isPackage := node.(ast.PackageStmt)
	if !isPackage {
		t.Fatalf("the project holds %T under hr, want ast.PackageStmt", node)
	}
	if hr.Name != "hr" {
		t.Errorf("the package is named %q, want %q", hr.Name, "hr")
	}

	sub, known := hr.SubPackage["payroll"]
	if !known {
		t.Fatalf("hr has subpackages %v, want payroll among them", subPackageNames(hr))
	}
	if payroll := sub.(ast.PackageStmt); payroll.Name != "hr.payroll" {
		t.Errorf("the subpackage is named %q, want its path %q", payroll.Name, "hr.payroll")
	}
}

// TestUnitMembersAreSplicedAndCompanionMembersAreFolded is the placement rule that
// makes a file stop being a scope.
//
// `rules.unit.fol` declares no type: its members are the package's own. The
// companion's are not — they belong to the struct `Employee.fol` declares, so the
// package sees one Employee rather than a struct and a loose unit beside it.
func TestUnitMembersAreSplicedAndCompanionMembersAreFolded(t *testing.T) {
	stmt := parseFixtureProject(t)
	hr := stmt.PackageStmts["hr"].(ast.PackageStmt)

	var employee ast.TypeDeclarationStmt
	found := false
	functions := []string{}
	units := 0
	for _, item := range hr.Body {
		switch member := item.(type) {
		case ast.TypeDeclarationStmt:
			if member.Kind == "co.lang.unit" {
				units++
			}
			if logicalName(member.Name) == "Employee" {
				employee, found = member, true
			}
		case ast.FunctionDeclarationStmt:
			functions = append(functions, logicalName(member.Name))
		}
	}

	if units != 0 {
		t.Errorf("the package holds %d unit declarations; a unit is a file wrapper and must not survive assembly", units)
	}
	sort.Strings(functions)
	if got := strings.Join(functions, ", "); got != "eligible" {
		t.Errorf("the package's own functions are %q, want the ordinary unit's %q spliced in", got, "eligible")
	}
	if !found {
		t.Fatal("the package has no Employee declaration")
	}

	folded := false
	for _, member := range employee.Body {
		if function, isFunction := member.(ast.FunctionDeclarationStmt); isFunction && logicalName(function.Name) == "promote" {
			folded = true
		}
	}
	if !folded {
		t.Error("the companion unit's promote is not in Employee's body; a companion is part of the type it names")
	}
}

// TestEveryFileSharesOneScopeModel is what lets two files in a package see each
// other. A per-file model would also leave the assembled tree naming tables it
// does not contain.
func TestEveryFileSharesOneScopeModel(t *testing.T) {
	stmt := parseFixtureProject(t)
	symbols := stmt.FolangSymbols

	roots := 0
	for _, ctx := range symbols.ContextMap {
		if ctx.ParentId == "" {
			roots++
		}
	}
	if roots < 2 {
		t.Errorf("the model has %d structural roots, want independent project surface and operational root", roots)
	}
	root := ProjectRootContext(symbols)
	if root == nil || root.ParentId != "" || root.ParentCtxSymbolTableId != "" || len(root.ChildCtxIds) != 0 {
		t.Fatalf("operational root is not independent: %#v", root)
	}

	for id, table := range symbols.SymboltableMap {
		if symbols.GetContext(table.ContextId) == nil {
			t.Errorf("symbol table %s belongs to context %s, which is not in this model", id, table.ContextId)
		}
	}
}

// TestProjectContextsFollowSemanticOwners pins Appendix B.8/B.10: source files
// and unit wrappers are parse-time devices, while packages and primary types are
// the contexts that survive in the assembled graph.
func TestProjectContextsFollowSemanticOwners(t *testing.T) {
	stmt := parseFixtureProject(t)
	symbols := stmt.FolangSymbols
	if stmt.FolContext == nil || stmt.FolContext != symbols.FolContext {
		t.Fatal("project statement does not carry the canonical FolContext descriptor")
	}
	if symbols.GetSymbolTable(stmt.FolContext.SymbolTable_) == nil || symbols.GetContext(stmt.FolContext.Context_) == nil {
		t.Fatalf("FolContext entry points are not present in the symbol graph: %#v", stmt.FolContext)
	}
	var hrContext *symboltable.Context
	for _, context := range symbols.ContextMap {
		if context != nil && context.ContextType_ == symboltable.S_PackageSymbol && context.ParentId == "" {
			hrContext = context
			break
		}
	}
	if hrContext == nil {
		t.Fatal("the project has no package context for src/hr")
	}

	hr := stmt.PackageStmts["hr"].(ast.PackageStmt)
	var employee ast.TypeDeclarationStmt
	var eligible ast.FunctionDeclarationStmt
	for _, item := range hr.Body {
		switch declaration := item.(type) {
		case ast.TypeDeclarationStmt:
			if logicalName(declaration.Name) == "Employee" {
				employee = declaration
			}
		case ast.FunctionDeclarationStmt:
			if logicalName(declaration.Name) == "eligible" {
				eligible = declaration
			}
		}
	}

	eligibleTable := symbols.GetSymbolTable(eligible.Symb.SymbolTableId)
	if eligibleTable == nil || eligibleTable.ContextId != hrContext.Id {
		t.Fatalf("ordinary-unit member eligible belongs to context %v, want package context %s", eligibleTable, hrContext.Id)
	}

	var employeeContext *symboltable.Context
	for _, childID := range hrContext.ChildCtxIds {
		child := symbols.GetContext(childID)
		if child != nil && child.ContextType_ == symboltable.S_StructSymbol {
			employeeContext = child
			break
		}
	}
	if employeeContext == nil {
		t.Fatal("Employee's struct context is not a child of the hr package context")
	}

	for _, member := range employee.Body {
		function, ok := member.(ast.FunctionDeclarationStmt)
		if !ok || logicalName(function.Name) != "promote" {
			continue
		}
		table := symbols.GetSymbolTable(function.Symb.SymbolTableId)
		if table == nil || table.ContextId != employeeContext.Id {
			t.Fatalf("companion member promote belongs to context %v, want Employee context %s", table, employeeContext.Id)
		}
		return
	}
	t.Fatal("the folded companion member promote was not found")
}

// topLevelNames lists a project's top-level package names, sorted.
func topLevelNames(stmt ast.ProjectStmt) []string {
	names := make([]string, 0, len(stmt.PackageStmts))
	for name := range stmt.PackageStmts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func subPackageNames(pkg ast.PackageStmt) []string {
	names := make([]string, 0, len(pkg.SubPackage))
	for name := range pkg.SubPackage {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func libraryNames(stmt ast.ProjectStmt) []string {
	names := make([]string, 0, len(stmt.LibraryStmt))
	for name := range stmt.LibraryStmt {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
