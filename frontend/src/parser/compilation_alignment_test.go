package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestCompilationUnitClassificationUsesProjectLocation(t *testing.T) {
	tests := []struct {
		name   string
		source string
		atRoot bool
		want   unitKind
	}{
		{"root struct is entry profile", `Employee co.lang.struct = { id co.lang.int; }`, true, unitEntry},
		{"package statement is package profile", `value := 1;`, false, unitPackage},
		{"root library remains surface", `Api co.lang.library = {}`, true, unitLibrary},
		{"nested library remains surface", `Api co.lang.library = {}`, false, unitLibrary},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toks := normalizeTokens(scanlex.Tokenize(test.source, "test.fol"))
			p, _ := newParser(toks)
			p.file = fileinfo{LocationKnown: true, AtRoot: test.atRoot}
			if got := p.classifyCompilationUnit(); got != test.want {
				t.Fatalf("classification = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEntryFileDeclarationRejectsGenericParameterClause(t *testing.T) {
	toks := normalizeTokens(scanlex.Tokenize(
		`Alias(F(_)) co.lang.type = co.lang.int;`,
		"app.fol",
	))
	p, _ := newParser(toks)
	p.file = fileinfo{Basename: "app.fol", LocationKnown: true, AtRoot: true}

	root := p.parseCompilationUnit()
	if _, ok := root.(ast.Application); !ok {
		t.Fatalf("root = %T, want ast.Application", root)
	}
	if len(p.diags) != 1 {
		t.Fatalf("diagnostics = %d, want exactly one generic entry-file diagnostic", len(p.diags))
	}
	if got := p.diags[0].Error(); !strings.Contains(got, "generic parameter clauses are not allowed in an application entry file") {
		t.Fatalf("diagnostic = %q, want the entry-file generic restriction", got)
	}
}

func TestEntryFileDeclarationStillAllowsForallTypeAlias(t *testing.T) {
	toks := normalizeTokens(scanlex.Tokenize(
		`PolyId co.lang.type = forall(T).(T)->(T); value PolyId;`,
		"app.fol",
	))
	p, _ := newParser(toks)
	p.file = fileinfo{Basename: "app.fol", LocationKnown: true, AtRoot: true}

	root := p.parseCompilationUnit()
	if _, ok := root.(ast.Application); !ok {
		t.Fatalf("root = %T, want ast.Application", root)
	}
	if len(p.diags) != 0 {
		t.Fatalf("forall entry-file alias produced diagnostics: %v", p.diags)
	}
}

func TestLibraryBodyImportsReachImportSurface(t *testing.T) {
	surface := ScanImportSurface(`Api co.lang.library = {
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
		"Employee", ".", "Employee.struct.fol", "people", "program", "program", true,
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
