package parser

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/project"
)

// How much of a library or a component the project may see.
//
// Its AST always arrives whole; its SYMBOLS arrive whole only when no surface
// stands between the project and its contents. A projected library publishes its
// surface and hides everything behind it, which is the point of the form.

// TestOnlyALibraryHidesBehindItsSurface covers the hiding half of the rule at the
// decision that makes it, rather than end to end.
//
// No SOURCE form reaches the projected branch today: the `library.fol` surface
// form was withdrawn, and the reference forbids `@co.dap.library` below
// components/ outright, since a component's kind already comes from its folder.
// The branch is wired for the lib/ artifacts that will carry a projected surface
// once they can be deserialized.
func TestOnlyALibraryHidesBehindItsSurface(t *testing.T) {
	projected := ast.ComponentDeclarationStmt{NodeName: "ComponentDeclarationStmt", Projected: true}
	assembly := &projectAssembly{}

	for _, unit := range []struct {
		name    string
		subject *externalUnit
		isolate bool
	}{
		{"a projected library", &externalUnit{key: "ffi", domain: project.PackagedLibraryDomain}, true},
		{"an application component", &externalUnit{key: componentKindApplication, domain: componentDomain}, false},
		{"a packaged component", &externalUnit{key: componentKindPackaged, domain: componentDomain}, false},
		{"the operator component", &externalUnit{key: componentKindOperators, domain: componentDomain}, false},
	} {
		t.Run(unit.name, func(t *testing.T) {
			if got := assembly.publishesSurfaceOnly(unit.subject, projected); got != unit.isolate {
				t.Errorf("publishes surface only = %v, want %v", got, unit.isolate)
			}
		})
	}
}

// TestAProjectedSurfaceIsRecognisedByItsExposureModel covers the other input to
// that decision.
func TestAProjectedSurfaceIsRecognisedByItsExposureModel(t *testing.T) {
	if !declaresProjectedLibrary(ast.ComponentDeclarationStmt{NodeName: "ComponentDeclarationStmt", Projected: true}) {
		t.Error("a surface whose exposure model is projected was not recognised")
	}
	if declaresProjectedLibrary(ast.ComponentDeclarationStmt{NodeName: "ComponentDeclarationStmt"}) {
		t.Error("a surface with no projected exposure model was read as projected")
	}
	if declaresProjectedLibrary(nil) {
		t.Error("an absent surface was read as projected")
	}
}

// TestAComponentIsASurfaceAndItsPackages covers the component node's shape and
// the symbols it contributes: a component is compiled as part of the project that
// owns it, so its whole table reaches the project's model.
func TestAComponentIsASurfaceAndItsPackages(t *testing.T) {
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
	write("src/appl.fol", "value := 1;")
	write("components/packaged/component.fol", `_ co.lang.component = {
}`)
	write("components/packaged/internals/Helper.fol", `_ co.lang.struct = {
    slot co.lang.int;
}`)

	parsed, _, err := ParseProject(root)
	if err != nil {
		t.Fatalf("ParseProject: %v", err)
	}
	stmt := parsed.(ast.ProjectStmt)

	node, known := stmt.ComponentStmt[componentKindPackaged]
	if !known {
		t.Fatalf("the project has no packaged component")
	}
	component, isComponent := node.(ast.ComponentDeclarationStmt)
	if !isComponent {
		t.Fatalf("the component node is %T, want ast.ComponentDeclarationStmt", node)
	}

	if !bound(stmt.FolangSymbols, "Helper") {
		t.Error("the project's symbol model does not hold Helper; a component publishes its whole table")
	}
	if component.Kind != componentKindPackaged {
		t.Errorf("the component's kind is %q, want %q from its folder", component.Kind, componentKindPackaged)
	}
	if _, held := component.SubPackage["internals"]; !held {
		t.Errorf("the component's subpackages are %v, want internals", subPackageKeys(component))
	}
	if _, isSurface := component.SurfaceFile.(ast.PackageStmt); !isSurface {
		t.Errorf("the component's surface file is %T, want the ast.PackageStmt holding its own declarations", component.SurfaceFile)
	}
}

// subPackageKeys lists a component's subpackage names, sorted.
func subPackageKeys(component ast.ComponentDeclarationStmt) []string {
	names := make([]string, 0, len(component.SubPackage))
	for name := range component.SubPackage {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// bound reports whether a name is declared anywhere in a symbol model.
func bound(symbols *symboltable.FolangSymbols, name string) bool {
	for _, table := range symbols.SymboltableMap {
		for _, symbol := range symbols.Bindings(table.Id) {
			if logicalName(symbol.GetName()) == name {
				return true
			}
		}
	}
	return false
}

// hasPackage reports whether a unit holds a top-level package of the given name.
func hasPackage(unit ast.ProjectStmt, name string) bool {
	_, known := unit.PackageStmts[name]
	return known
}

func packageNames(unit ast.ProjectStmt) []string {
	return topLevelNames(unit)
}
