package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestCompilationUnitClassificationUsesProjectLocation(t *testing.T) {

	tests := []struct {
		name     string
		source   string
		basename string
		atRoot   bool
		want     unitKind
	}{
		{"root struct is entry profile", `_ co.lang.struct = { id co.lang.int; }`, "Employee.fol", true, unitEntry},
		{"package statement is package profile", `value := 1;`, "Values.fol", false, unitPackage},
		{"root library remains surface", `_ co.lang.library = {}`, "Api.fol", true, unitLibrary},
		{"nested library remains surface", `_ co.lang.library = {}`, "Api.fol", false, unitLibrary},
		// A unit filename is decisive on its own: a `.unit.fol` file is a package
		// source file wherever it sits, including at the project root.
		{"unit filename overrides root location", `_ co.lang.unit = {}`, "arithmetic.unit.fol", true, unitPackage},
		{"companion filename overrides root location", `_ co.lang.unit = {}`, "Employee.comp.unit.fol", true, unitPackage},
		{"package.fol overrides root location", `_ co.lang.package = { name: "emp" };`, "package.fol", true, unitPackage},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toks := normalizeTokens(scanlex.Tokenize(test.source, test.basename))
			p, _ := newParser(toks)
			p.file = fileinfo{
				Basename:      test.basename,
				LocationKnown: true,
				AtRoot:        test.atRoot,
				Source:        classifySourceFilename(test.basename),
			}
			if got := p.classifyCompilationUnit(); got != test.want {
				t.Fatalf("classification = %v, want %v", got, test.want)
			}
		})
	}
}

// DECISION-ENTRY-001 admits parameterized co.lang.type constructors in an entry
// file. Revision 19 had rejected every declaration-head parameter clause there;
// revision 23 kept the restriction only for the kinds that never had one.
func TestEntryFileAdmitsParameterizedTypeConstructor(t *testing.T) {
	root, p := parseEntrySource(t, `Option(T) co.lang.type = Some(T) | None(); value Option(co.lang.int);`)

	if _, ok := root.(ast.Application); !ok {
		t.Fatalf("root = %T, want ast.Application", root)
	}
	if len(p.diags) != 0 {
		t.Fatalf("parameterized entry type constructor produced diagnostics: %v", p.diags)
	}
}

func TestEntryFileRejectsParameterizedNonTypeKind(t *testing.T) {
	_, p := parseEntrySource(t, `Alias(F(_)) co.lang.newtype = co.lang.int;`)

	if len(p.diags) != 1 {
		t.Fatalf("diagnostics = %d, want exactly one parameterized-kind diagnostic", len(p.diags))
	}
	if got := p.diags[0].Error(); !strings.Contains(got, "only a co.lang.type declaration may be parameterized") {
		t.Fatalf("diagnostic = %q, want the parameterized-kind restriction", got)
	}
}

func TestEntryFileDeclarationStillAllowsForallTypeAlias(t *testing.T) {
	root, p := parseEntrySource(t, `PolyId co.lang.type = forall(T).(T)->(T); value PolyId;`)

	if _, ok := root.(ast.Application); !ok {
		t.Fatalf("root = %T, want ast.Application", root)
	}
	if len(p.diags) != 0 {
		t.Fatalf("forall entry-file alias produced diagnostics: %v", p.diags)
	}
}

// parseEntrySource parses source as the application entry file at the project root.
func parseEntrySource(t *testing.T, source string) (ast.Stmt, *parser) {
	t.Helper()

	toks := normalizeTokens(scanlex.Tokenize(source, "app.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{
		Basename:      "app.fol",
		LocationKnown: true,
		AtRoot:        true,
		Source:        classifySourceFilename("app.fol"),
	}
	return p.parseCompilationUnit(), p
}

func TestLibraryBodyImportsReachImportSurface(t *testing.T) {
	surface := ScanImportSurface(`_ co.lang.library = {
    @co.ddap.import(package="internal.values", as="values")
    Value co.lang.struct = { id co.lang.int; }
}`, "Api.fol", "Api", "", true)

	if len(surface.Imports) != 1 {
		t.Fatalf("imports = %d, want 1", len(surface.Imports))
	}
	if got := surface.Imports[0].Package; got != "internal.values" {
		t.Fatalf("package = %q, want internal.values", got)
	}
	if got := surface.Imports[0].Alias; got != "values" {
		t.Fatalf("alias = %q, want values", got)
	}
}

func TestFilenameDerivedNameIsValidatedAndLowered(t *testing.T) {
	root, _, _, _ := parseIntoConfigured(nil,
		`_ co.lang.struct = { id co.lang.int; }`,
		"Employee", ".", "Employee.fol", "people", "program", "program", true,
		parseConfiguration{locationKnown: true, atRoot: false},
	)
	pkg, ok := root.(ast.PackageStmt)
	if !ok || len(pkg.Body) != 1 {
		t.Fatalf("root = %T with unexpected body", root)
	}
	decl, ok := pkg.Body[0].(ast.TypeDeclarationStmt)
	if !ok {
		t.Fatalf("declaration = %T", pkg.Body[0])
	}
	if decl.Name != "Employee_fo" {
		t.Fatalf("name = %q, want Employee_fo", decl.Name)
	}
}

func TestPackageIdentityDistinguishesLegacyCallersFromKnownProjectRoot(t *testing.T) {
	tests := []struct {
		name string
		file fileinfo
		want string
	}{
		{"legacy API keeps basename fallback", fileinfo{Basename: "people"}, "people"},
		{"known project root is not a package", fileinfo{Basename: "people", LocationKnown: true, AtRoot: true}, ""},
		{"known subfolder uses package path", fileinfo{Basename: "Employee", PackagePath: "people", LocationKnown: true}, "people"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := parser{file: test.file}
			if got := p.packageIdentity(); got != test.want {
				t.Fatalf("package identity = %q, want %q", got, test.want)
			}
		})
	}
}
