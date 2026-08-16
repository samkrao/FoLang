package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/project"
)

func TestPrepareProjectRootBuildsIsolatedStages(t *testing.T) {
	root := t.TempDir()
	writePreparedProjectFile(t, root, "src/appl.fol", `@co.ddap.import(component="native")
value := 1;`)
	writePreparedProjectFile(t, root, "components/native/component.fol", `_ co.lang.component = {
    allocate(size co.lang.int)->(co.lang.address) = {}
}`)
	writePreparedProjectFile(t, root, "components/native/impl/Memory.fol", `_ co.lang.struct = { address co.lang.address; }`)
	writePreparedProjectFile(t, root, "components/packaged/component.fol", `_ co.lang.component = {
    @co.dap.export(packages={hr: {recurse=true}})
}`)
	writePreparedProjectFile(t, root, "components/packaged/hr/Employee.fol", `_ co.lang.struct = { id co.lang.int; }`)
	writePreparedProjectFile(t, root, "components/packaged/hr/detail/Record.fol", `_ co.lang.struct = { id co.lang.int; }`)
	writePreparedProjectFile(t, root, "components/packaged/secret/Hidden.fol", `_ co.lang.struct = { id co.lang.int; }`)
	writePreparedProjectFile(t, root, "components/operators/component.fol", `_ co.lang.component = {
    <+> co.lang.operator = {
        fixity: co.operator.fixity.infix,
        precedence: 60,
        associativity: co.operator.associativity.left,
        arity: co.operator.arity.binary
    };
}`)
	writePreparedProjectFile(t, root, "lib/runtime.folenc", "")

	prepared, err := PrepareProjectRoot(filepath.Join(root, "src", "appl.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(prepared.Order, []project.CompilationStage{
		project.StageComponents, project.StageLibraries, project.StagePrimarySource,
	}) {
		t.Fatalf("preparation order = %#v", prepared.Order)
	}
	native := prepared.Components["native"]
	if native == nil || native.Surface == nil || native.ProjectedAPI == nil {
		t.Fatalf("native component surface was not prepared: %#v", native)
	}
	if len(native.PrivatePackages["impl"]) != 1 {
		t.Fatalf("native private packages = %#v", native.PrivatePackages)
	}
	packaged := prepared.Components["packaged"]
	if packaged == nil || len(packaged.PackagedExports["hr"]) != 1 || len(packaged.PackagedExports["hr.detail"]) != 1 {
		t.Fatalf("packaged exports = %#v", packaged)
	}
	if _, leaked := packaged.PackagedExports["secret"]; leaked {
		t.Fatal("unselected packaged component package leaked into exports")
	}
	operators := prepared.Components["operators"]
	if operators == nil || operators.Surface == nil || operators.ProjectedAPI != nil {
		t.Fatalf("operator component must retain its parsed surface but publish no ordinary API: %#v", operators)
	}
	if len(prepared.Operators) != 1 {
		t.Fatalf("separate operator table contains %d declarations, want 1", len(prepared.Operators))
	}
	if prepared.Environment.ProjectedComponents["native"] == nil {
		t.Fatal("primary environment is missing the native projected surface API")
	}
	if len(prepared.Environment.PackagedComponents["hr"]) != 1 || len(prepared.Environment.PackagedComponents["hr.detail"]) != 1 {
		t.Fatalf("primary packaged environment = %#v", prepared.Environment.PackagedComponents)
	}
	if _, leaked := prepared.Environment.PackagedComponents["secret"]; leaked {
		t.Fatal("primary environment received a private packaged-component package")
	}
	if len(prepared.Environment.Operators) != 1 {
		t.Fatalf("primary operator environment contains %d declarations", len(prepared.Environment.Operators))
	}
	if library, ok := prepared.Libraries[filepath.Join(root, "lib", "runtime.folenc")]; !ok || !library.Pending {
		t.Fatalf("deferred-codec library = %#v, present=%t", library, ok)
	}
	if len(prepared.Primary) != 1 || prepared.Primary[0].Symbols == nil {
		t.Fatalf("primary sources = %#v", prepared.Primary)
	}
	if prepared.Primary[0].RootSymbolTable == nil || prepared.Primary[0].SymbolGraph == nil {
		t.Fatal("primary source did not retain its isolated symbol table graph")
	}
	if len(prepared.Findings) != 0 {
		t.Fatalf("valid application component composition findings: %v", prepared.Findings)
	}
}

func TestPrepareProjectRootRecognizesStandaloneComponentSurface(t *testing.T) {
	root := t.TempDir()
	writePreparedProjectFile(t, root, "src/component.fol", `@co.dap.library(type=native)
_ co.lang.component = { allocate(size co.lang.int)->(co.lang.address) = {} }`)
	writePreparedProjectFile(t, root, "src/impl/Memory.fol", `_ co.lang.struct = { address co.lang.address; }`)

	prepared, err := PrepareProjectRoot(filepath.Join(root, "src", "component.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Kind != project.CompilationStandaloneComponent {
		t.Fatalf("project kind = %v, want standalone component", prepared.Kind)
	}
	if len(prepared.Primary) != 2 {
		t.Fatalf("standalone primary inputs = %d, want surface and implementation package", len(prepared.Primary))
	}
	if len(prepared.Findings) != 0 {
		t.Fatalf("valid standalone component findings: %v", prepared.Findings)
	}
	if prepared.StandaloneProjectedAPI == nil || len(prepared.StandalonePackagedExports) != 0 {
		t.Fatal("projected standalone surface did not publish only its API context")
	}
}

func TestPrepareProjectRootSelectsStandalonePackagedExports(t *testing.T) {
	root := t.TempDir()
	writePreparedProjectFile(t, root, "src/component.fol", `_ co.lang.component = {
    @co.dap.export(packages={hr: {recurse=true}})
}`)
	writePreparedProjectFile(t, root, "src/hr/Employee.fol", `_ co.lang.struct = { id co.lang.int; }`)
	writePreparedProjectFile(t, root, "src/private/Hidden.fol", `_ co.lang.struct = { id co.lang.int; }`)

	prepared, err := PrepareProjectRoot(filepath.Join(root, "src", "component.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.StandaloneProjectedAPI != nil || len(prepared.StandalonePackagedExports["hr"]) != 1 {
		t.Fatalf("standalone packaged exports = %#v", prepared.StandalonePackagedExports)
	}
	if _, leaked := prepared.StandalonePackagedExports["private"]; leaked {
		t.Fatal("unselected standalone implementation package leaked into artifact exports")
	}
}

func TestStandaloneNativeProjectRejectsProjectLocalComponents(t *testing.T) {
	root := t.TempDir()
	writePreparedProjectFile(t, root, "src/component.fol", `@co.dap.library(type=native)
_ co.lang.component = {}`)
	writePreparedProjectFile(t, root, "components/application/component.fol", `_ co.lang.component = {}`)

	prepared, err := PrepareProjectRoot(filepath.Join(root, "src", "component.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Findings) == 0 {
		t.Fatal("standalone native project accepted a project-local component")
	}
}

func TestProjectedApplicationLibraryAllowsOnlyOperatorComponent(t *testing.T) {
	root := t.TempDir()
	writePreparedProjectFile(t, root, "src/component.fol", `@co.dap.library
_ co.lang.component = {}`)
	writePreparedProjectFile(t, root, "components/operators/component.fol", `_ co.lang.component = {
    <+> co.lang.operator = { fixity: co.operator.fixity.infix, precedence: 60, associativity: co.operator.associativity.left, arity: co.operator.arity.binary };
}`)

	prepared, err := PrepareProjectRoot(filepath.Join(root, "src", "component.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Findings) != 0 {
		t.Fatalf("projected application library with operator component findings: %v", prepared.Findings)
	}
	if len(prepared.Operators) != 1 || prepared.StandaloneProjectedAPI == nil {
		t.Fatal("projected application library did not retain API and isolated operator table")
	}
}

func TestProjectLocalComponentCannotImportPeerComponent(t *testing.T) {
	root := t.TempDir()
	writePreparedProjectFile(t, root, "src/appl.fol", "value := 1;")
	writePreparedProjectFile(t, root, "components/application/component.fol", `_ co.lang.component = {
    @co.ddap.import(component="native")
}`)
	writePreparedProjectFile(t, root, "components/native/component.fol", `_ co.lang.component = {}`)

	prepared, err := PrepareProjectRoot(filepath.Join(root, "src", "appl.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Findings) == 0 {
		t.Fatal("project-local component imported a peer component")
	}
}

func TestFocmainWithExplicitRootRunsPreparedProjectPipeline(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "src", "appl.fol")
	writePreparedProjectFile(t, root, "src/appl.fol", "value := 1;")
	writePreparedProjectFile(t, root, "components/application/component.fol", `_ co.lang.component = { ping()->() = {} }`)

	_, _, serialized, _, err := Focmain(entry, false, false, "", false, root)
	if err != nil {
		t.Fatal(err)
	}
	if serialized == "" {
		t.Fatal("project-root driver returned no primary AST")
	}
}

func writePreparedProjectFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
