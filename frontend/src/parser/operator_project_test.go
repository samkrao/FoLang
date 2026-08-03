package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/project"
)

func TestProjectOperatorCatalogParsesUseWithoutLocalDeclaration(t *testing.T) {
	declaration := operatorDeclaration{Options: map[string]any{
		"symbol": "<+>", "fixity": "infix",
		"precedence": int64(65), "associativity": "left", "arity": "binary",
	}}

	root, _, _, _ := parseIntoConfigured(nil,
		"result := left <+> right;\n",
		"operators", ".", "main.fol", "", "program", "program", true,
		parseConfiguration{locationKnown: true, atRoot: true, operators: []operatorDeclaration{declaration}},
	)
	application, ok := root.(ast.Application)
	if !ok || len(application.Body) != 1 {
		t.Fatalf("project operator use returned %T with unexpected body", root)
	}
}

func TestOrdinarySourceCannotPopulateOperatorCatalog(t *testing.T) {
	collection := declaredOperatorsIn(
		`combine co.lang.operator->(symbol="<+>", mode=define, fixity=infix, precedence=65, associativity=left, arity=binary) = co.lang.int;`,
		"Combine.fol", nil,
	)
	if !collection.Custom.Empty() {
		t.Fatal("ordinary co.lang.operator syntax populated the source-only catalog")
	}
	if len(collection.Declarations) != 0 {
		t.Fatalf("declarations = %d, want 0", len(collection.Declarations))
	}
}

func TestProjectOperatorRequiresRealCompanionStruct(t *testing.T) {
	unit := scanDeclarationSurface(`Math co.lang.unit = {
    @co.dap.operator(symbol='+', mode=overload)
    add(left Math, right Math)->(Math) = { this.return left; }
}`, project.File{Base: "Math.unit.fol", Stem: "Math.unit", PackagePath: "example"})

	if findings := validateOperatorCompanions([]declarationSurface{unit}); len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}

	structure := scanDeclarationSurface(`Math co.lang.struct = { value co.lang.int; }`,
		project.File{Base: "Math.struct.fol", Stem: "Math.struct", PackagePath: "example"})
	if findings := validateOperatorCompanions([]declarationSurface{unit, structure}); len(findings) != 0 {
		t.Fatalf("same-package companion produced %d findings", len(findings))
	}
}

func TestProjectOperatorRejectsAmbiguousMultiFileCompanions(t *testing.T) {
	operatorUnit := func(base string) declarationSurface {
		return scanDeclarationSurface(`Employee co.lang.unit = {
    @co.dap.operator(symbol='+', mode=overload)
    add(left Employee, right Employee)->(Employee) = { this.return left; }
}`, project.File{Base: base, Stem: strings.TrimSuffix(base, ".fol"), PackagePath: "hr"})
	}
	structure := func(base string) declarationSurface {
		return scanDeclarationSurface(`Employee co.lang.struct = { id co.lang.int; }`,
			project.File{Base: base, Stem: strings.TrimSuffix(base, ".fol"), PackagePath: "hr"})
	}
	plainUnit := func(base string) declarationSurface {
		return scanDeclarationSurface(`Employee co.lang.unit = {
    label(value Employee)->(co.lang.string) = { this.return "employee"; }
}`, project.File{Base: base, Stem: strings.TrimSuffix(base, ".fol"), PackagePath: "hr"})
	}

	tests := []struct {
		name     string
		surfaces []declarationSurface
		want     string
	}{
		{
			name: "duplicate structs",
			surfaces: []declarationSurface{
				operatorUnit("Employee.unit.fol"),
				structure("Employee.fol"),
				structure("Employee.struct.fol"),
			},
			want: "found 2 struct declarations and 1 unit declarations",
		},
		{
			name: "plain duplicate unit",
			surfaces: []declarationSurface{
				operatorUnit("Employee.unit.fol"),
				plainUnit("Employee.extra.fol"),
				structure("Employee.struct.fol"),
			},
			want: "found 1 struct declarations and 2 unit declarations",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			findings := validateOperatorCompanions(test.surfaces)
			if len(findings) != 1 {
				t.Fatalf("findings = %d, want one deterministic ambiguity diagnostic", len(findings))
			}
			if diagnostic := findings[0].Error(); !strings.Contains(diagnostic, test.want) {
				t.Fatalf("diagnostic = %q, want %q", diagnostic, test.want)
			}
		})
	}
}

func TestProjectBuiltInOperatorExtensionDoesNotRequireCompanionStruct(t *testing.T) {
	extension := scanDeclarationSurface(`Strings co.lang.unit = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(fortype=co.lang.string, what=extends)
    concat(left co.lang.string, right co.lang.string)->(co.lang.string) = { this.return left; }
}`, project.File{Base: "Strings.unit.fol", Stem: "Strings.unit", PackagePath: "example"})

	if !extension.HasOperator {
		t.Fatal("operator extension was not discovered")
	}
	if extension.HasCompanionOperator {
		t.Fatal("built-in operator extension was classified as a struct companion operator")
	}
	if findings := validateOperatorCompanions([]declarationSurface{extension}); len(findings) != 0 {
		t.Fatalf("built-in operator extension produced %d companion findings", len(findings))
	}
}

func TestProjectMixedOperatorUnitStillRequiresCompanionStruct(t *testing.T) {
	mixed := scanDeclarationSurface(`Strings co.lang.unit = {
    @co.dap.extension(fortype=co.lang.string, what=extends)
    @co.dap.operator(symbol='+')
    concat(left co.lang.string, right co.lang.string)->(co.lang.string) = { this.return left; }

    @co.dap.operator(symbol='-')
    subtract(left Strings, right Strings)->(Strings) = { this.return left; }
}`, project.File{Base: "Strings.unit.fol", Stem: "Strings.unit", PackagePath: "example"})

	if !mixed.HasCompanionOperator {
		t.Fatal("ordinary operator in a mixed unit was hidden by an unrelated extension operator")
	}
	if findings := validateOperatorCompanions([]declarationSurface{mixed}); len(findings) != 1 {
		t.Fatalf("mixed unit produced %d companion findings, want 1", len(findings))
	}
}
