package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
)

// TestGenericContainerParametersAreRetained verifies that the common primary
// declaration prefix does not merely consume generic parameters: every named
// type/container form that admits the clause carries it into its concrete AST.
func TestGenericContainerParametersAreRetained(t *testing.T) {
	tests := []struct {
		name   string
		source string
		params func(ast.Stmt) []symboltable.GenericTypeParam
	}{
		{"unit", `Generic(F(_)) co.lang.unit = {}`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.TypeDeclarationStmt).TypeParams
		}},
		{"module", `Generic(F(_)) co.lang.module = {}`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.ModuleStmt).TypeParams
		}},
		{"object", `Generic(F(_)) co.lang.object = {}`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.ObjectDeclStmt).TypeParams
		}},
		{"instance", `Generic(F(_)) co.lang.instance = {}`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.TypeclassInstanceStmt).TypeParams
		}},
		{"matcher", `Generic(F(_)) co.lang.matcher = {}`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.MatcherInstanceStmt).TypeParams
		}},
		{"function-object", `Generic(F(_)) co.lang.function = target;`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.FunctionDeclarationStmt).TypeParams
		}},
		{"delegate", `Generic(F(_)) co.lang.delegate = (F(co.lang.int))->(F(co.lang.int));`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.DelegateStmt).TypeParams
		}},
		{"named-block", `Generic(F(_)) co.lang.block = {}`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(*ast.BlockStmt).TypeParams
		}},
		{"annotated-typeclass", "@co.dap.typeclass\nGeneric(F(_)) = {}", func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.TypeclassStmt).TypeParams
		}},
		{"data", `Generic(F(_)) co.lang.data = Present(F(co.lang.int)) | Absent();`, func(stmt ast.Stmt) []symboltable.GenericTypeParam {
			return stmt.(ast.TypeConstructorStmt).GenericParams
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decl := packagePrimary(t, tc.source, "Generic.fol")
			params := tc.params(decl)
			if len(params) != 1 {
				t.Fatalf("stored %d generic parameters, want 1", len(params))
			}
			if got := logicalName(params[0].Name); got != "F" {
				t.Fatalf("generic parameter = %q, want F", got)
			}
			if params[0].TypeKind != "type" || params[0].Types != "_" {
				t.Fatalf("generic arity = kind %q slots %q, want type(_) ", params[0].TypeKind, params[0].Types)
			}
		})
	}
}

// TestDataDeclarationRetainsCompleteGenericParameters guards against reducing
// co.lang.data parameters to names. Higher-kinded arity and ordinary bounds are
// both needed by later type checking.
func TestDataDeclarationRetainsCompleteGenericParameters(t *testing.T) {
	decl := packagePrimary(t,
		`Generic(F(_), T: Orderable) co.lang.data = Present(F(T)) | Absent();`,
		"Generic.fol",
	).(ast.TypeConstructorStmt)

	if len(decl.GenericParams) != 2 {
		t.Fatalf("stored %d generic parameters, want 2", len(decl.GenericParams))
	}
	constructor := decl.GenericParams[0]
	if logicalName(constructor.Name) != "F" || constructor.TypeKind != "type" || constructor.Types != "_" {
		t.Fatalf("higher-kinded parameter = %#v, want F(_)", constructor)
	}
	bounded := decl.GenericParams[1]
	if logicalName(bounded.Name) != "T" || logicalName(bounded.Constraint) != "Orderable" {
		t.Fatalf("bounded parameter = %#v, want T: Orderable", bounded)
	}
	if len(decl.TypeParams) != 2 || logicalName(decl.TypeParams[0]) != "F" || logicalName(decl.TypeParams[1]) != "T" {
		t.Fatalf("legacy generic names = %#v, want [F T]", decl.TypeParams)
	}
}

func TestPackageAndLibraryDeclarationsRejectGenericClauses(t *testing.T) {
	rejected := []struct {
		name   string
		source string
	}{
		{"package", `Alias(T) co.lang.package;`},
		{"library", `Surface(T) co.lang.library = {}`},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				packagePrimary(t, tc.source, "Generic.fol")
			})
		})
	}
}

// TestKindTokensRemainUsableAsTypes fixes the contextual priority rule. The
// scanner may classify an overlapping co.lang name as BUILT_IN_KIND, but after
// a variable name the parser must read it as the variable's type.
func TestKindTokensRemainUsableAsTypes(t *testing.T) {
	fn := unitFunction(t, `_ co.lang.unit = {
    receive()->() = {
        value co.lang.value;
        absent co.lang.nothing;
        payload co.lang.data;
        generated co.lang.dependentType;
    }
}`, "receive")

	if len(fn.Body) != 4 {
		t.Fatalf("function body has %d declarations, want 4", len(fn.Body))
	}
	for i, statement := range fn.Body {
		if _, ok := statement.(ast.VarDeclarationStmt); !ok {
			t.Errorf("body statement %d is %T, want ast.VarDeclarationStmt", i, statement)
		}
	}
}

func TestDependentAndMetaKindsAreDirectTypeDeclarations(t *testing.T) {
	tests := []struct {
		kind    string
		subtype string
	}{
		{"co.lang.dependentType", "dependent"},
		{"co.lang.typetype", "typetype"},
		{"co.lang.typekind", "typekind"},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			decl := packagePrimary(t, "Declared "+tc.kind+" = co.lang.type;", "Declared.fol").(ast.TypeDeclarationStmt)
			if decl.Kind != tc.kind || decl.SubType_ != tc.subtype {
				t.Fatalf("declaration kind/subtype = %q/%q, want %q/%q", decl.Kind, decl.SubType_, tc.kind, tc.subtype)
			}
		})
	}
}

func TestTypeConstructorHasExactlyOneTypeProducingResult(t *testing.T) {
	var union ast.TypeDeclarationStmt
	mustNotPanic(t, func() {
		union = packagePrimary(t,
			`Choice(n co.lang.int)->(co.lang.dependentType | co.lang.type) = co.lang.int;`,
			"Choice.fol").(ast.TypeDeclarationStmt)
	})
	if union.Kind != "co.lang.dependentType|co.lang.type" {
		t.Fatalf("constructor result kind = %q, want the one union result to be retained", union.Kind)
	}
	if !union.Symb.(*symboltable.TypeSymbol).DependentType {
		t.Fatal("constructor union containing co.lang.dependentType lost its dependent-type symbol flag")
	}

	plain := packagePrimary(t,
		`Meta(n co.lang.int)->(co.lang.typetype) = co.lang.type;`,
		"Meta.fol").(ast.TypeDeclarationStmt)
	if plain.Kind != "co.lang.typetype" {
		t.Fatalf("constructor result kind = %q, want co.lang.typetype", plain.Kind)
	}

	rejected := []struct {
		name   string
		source string
	}{
		{"multiple-results", `Bad(n co.lang.int)->(co.lang.dependentType, co.lang.type) = co.lang.int;`},
		{"named-result", `Bad(n co.lang.int)->(result co.lang.dependentType) = co.lang.int;`},
		{"non-type-union-arm", `Bad(n co.lang.int)->(co.lang.dependentType | co.lang.int) = co.lang.int;`},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				packagePrimary(t, tc.source, "Bad.fol")
			})
		})
	}
}

// TestParenthesizedExpressionsAndFunctionTypesStayDistinct records the user's
// distinction: `(a, b)` is a valid tuple expression, while a type parameter
// group becomes a function type only through a proper arrow and never admits a
// trailing comma.
func TestParenthesizedExpressionsAndFunctionTypesStayDistinct(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, `result := (left, right);`)
	})
	mustNotPanic(t, func() {
		packagePrimary(t, `Fn co.lang.type = (A, B)->(R);`, "Fn.fol")
	})

	rejected := []struct {
		name   string
		source string
	}{
		{"type-list-without-arrow", `Fn co.lang.type = (A, B);`},
		{"single-type-trailing-comma", `Fn co.lang.type = (A,);`},
		{"function-type-trailing-comma", `Fn co.lang.type = (A,)->(R);`},
		{"delegate-trailing-comma", `Fn co.lang.delegate = (A,)->(R);`},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				packagePrimary(t, tc.source, "Fn.fol")
			})
		})
	}
}
