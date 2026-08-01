package parser

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/project"
)

func TestProjectOperatorCatalogParsesUseWithoutLocalDeclaration(t *testing.T) {
	declaration := operatorDeclaration{Options: map[string]any{
		"symbol": "<+>", "mode": "define", "fixity": "infix",
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

func TestCoLangOperatorOptionsReachPrepass(t *testing.T) {
	collection := declaredOperatorsIn(
		`combine co.lang.operator->(symbol="<+>", mode=define, fixity=infix, precedence=65, associativity=left, arity=binary) = co.lang.int;`,
		"Combine.fol", nil,
	)
	if collection.Custom.Empty() {
		t.Fatal("co.lang.operator symbol was not collected")
	}
	if len(collection.Declarations) != 1 {
		t.Fatalf("declarations = %d, want 1", len(collection.Declarations))
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
