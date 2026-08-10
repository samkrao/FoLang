package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// TestDeclarationHeadParametersAreRestrictedToTheirThreeForms records
// DECISION-GEN-001. Revision 23 removed the declaration-head parameter clause
// from every named type and container form; @co.dap.generic is now the sole
// mechanism for a generic struct, class, function or method. Only three
// declaration forms keep a head clause, and each keeps it for a reason a
// filename or an annotation cannot supply.
func TestDeclarationHeadParametersAreRestrictedToTheirThreeForms(t *testing.T) {
	t.Run("retained", func(t *testing.T) {
		tests := []struct {
			name     string
			source   string
			basename string
			params   func(ast.Stmt) []symboltable.GenericTypeParam
		}{
			{
				// A typeclass parameter clause is its own grammar component, and
				// its parameters may declare arity (DECISION-TCLASS-001).
				name:     "typeclass",
				source:   "@co.dap.typeclass(kind=Functor)\n_ (F(_)) co.lang.typeclass = {}",
				basename: "Generic.fol",
				params: func(stmt ast.Stmt) []symboltable.GenericTypeParam {
					return stmt.(ast.TypeclassStmt).TypeParams
				},
			},
			{
				// A filename cannot carry `(F(_))`, so a parameterized type
				// constructor names itself and lives in a unit.
				name:     "parameterized-type",
				source:   "_ co.lang.unit = {\n    Generic(F(_)) co.lang.type = F(co.lang.int);\n}",
				basename: "generic.unit.fol",
				params: func(stmt ast.Stmt) []symboltable.GenericTypeParam {
					return stmt.(ast.TypeDeclarationStmt).TypeParams
				},
			},
			{
				// A data declaration names its variants in the head, so it
				// cannot take a filename-derived name either.
				name:     "data",
				source:   "_ co.lang.unit = {\n    Generic(F(_)) co.lang.data = Present(F(co.lang.int)) | Absent();\n}",
				basename: "generic.unit.fol",
				params: func(stmt ast.Stmt) []symboltable.GenericTypeParam {
					return stmt.(ast.TypeConstructorStmt).GenericParams
				},
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				var decl ast.Stmt
				if tc.basename == "Generic.fol" {
					decl = packagePrimary(t, tc.source, tc.basename)
				} else {
					decl = unitMember(t, tc.source)
				}
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
	})

	t.Run("rejected", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			source string
		}{
			{"struct", `_(F(_)) co.lang.struct = {}`},
			{"class", `_(F(_)) co.lang.class = {}`},
			{"unit", `_(F(_)) co.lang.unit = {}`},
			{"module", `_(F(_)) co.lang.module = {}`},
			{"object", `_(F(_)) co.lang.object = {}`},
			{"instance", `_(F(_)) co.lang.instance = {}`},
			{"matcher", `_(F(_)) co.lang.matcher->(type=co.lang.int) = {}`},
			{"function-object", `_(F(_)) co.lang.function = target;`},
			{"delegate", `_(F(_)) co.lang.delegate = (F(co.lang.int))->(F(co.lang.int));`},
			{"named-block", `_(F(_)) co.lang.block = {}`},
			{"library", `_(T) co.lang.library = {}`},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				mustPanic(t, func() {
					packagePrimary(t, tc.source, "Generic.fol")
				})
			})
		}
	})
}

// TestDataDeclarationRetainsCompleteGenericParameters guards against reducing
// co.lang.data parameters to names. Higher-kinded arity and ordinary bounds are
// both needed by later type checking.
func TestDataDeclarationRetainsCompleteGenericParameters(t *testing.T) {
	decl := unitMember(t,
		"_ co.lang.unit = {\n    Generic(F(_), T: Orderable) co.lang.data = Present(F(T)) | Absent();\n}",
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

// type-declaration-kind is closed to the kinds the reference gives a source form. Both
// halves matter: the admitted kinds declare a type, and the table-listed names that have
// no declaration syntax anywhere in the reference stay reserved rather than becoming
// usable by resemblance to the ones that do.
func TestTypeDeclarationKindsAreClosedToTheDocumentedSourceForms(t *testing.T) {
	admitted := []struct {
		kind    string
		subtype string
	}{
		{"co.lang.dependentType", "dependent"},
		{"co.lang.newtype", "newtype"},
		{"co.lang.opaquetype", "opaque"},
		{"co.lang.subtype", "subtype"},
		{"co.lang.supertype", "supertype"},
		{"co.lang.kind", "kind"},
	}

	for _, tc := range admitted {
		t.Run(tc.kind, func(t *testing.T) {
			decl := unitMember(t,
				"_ co.lang.unit = {\n    Declared "+tc.kind+" = co.lang.type;\n}",
			).(ast.TypeDeclarationStmt)
			if decl.Kind != tc.kind || decl.SubType_ != tc.subtype {
				t.Fatalf("declaration kind/subtype = %q/%q, want %q/%q", decl.Kind, decl.SubType_, tc.kind, tc.subtype)
			}
		})
	}

	// Each of these is a row of the Builtin Kinds table with no declaration form
	// anywhere in the reference, so none may be declared.
	reserved := []string{
		"co.lang.typealias",
		"co.lang.associatedtype",
		"co.lang.refinementType",
		"co.lang.typetype",
		"co.lang.typekind",
	}
	for _, kind := range reserved {
		t.Run("reserved/"+kind, func(t *testing.T) {
			mustPanic(t, func() {
				parseUnitSource(t, "_ co.lang.unit = {\n    Declared "+kind+" = co.lang.type;\n}")
			})
		})
	}
}

// A type-level function returns exactly one unnamed type-producing result, which a `|`
// may combine into one union and a `,` may never split into two.
//
// type-level-result-kind is the two kinds the reference actually returns from one:
// co.lang.dependentType and co.lang.type.
func TestTypeLevelFunctionHasExactlyOneTypeProducingResult(t *testing.T) {
	var union ast.TypeDeclarationStmt
	mustNotPanic(t, func() {
		union = unitMember(t,
			"_ co.lang.unit = {\n    Choice(n co.lang.int)->(co.lang.dependentType | co.lang.type) = co.lang.int;\n}",
		).(ast.TypeDeclarationStmt)
	})
	if union.Kind != "co.lang.dependentType|co.lang.type" {
		t.Fatalf("result kind = %q, want the one union result to be retained", union.Kind)
	}
	if !union.Symb.(*symboltable.TypeSymbol).DependentType {
		t.Fatal("union containing co.lang.dependentType lost its dependent-type symbol flag")
	}

	plain := unitMember(t,
		"_ co.lang.unit = {\n    Meta(n co.lang.int)->(co.lang.type) = co.lang.int;\n}",
	).(ast.TypeDeclarationStmt)
	if plain.Kind != "co.lang.type" {
		t.Fatalf("result kind = %q, want co.lang.type", plain.Kind)
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseUnitSource(t, "_ co.lang.unit = {\n    "+tc.source+"\n}")
			})
		})
	}
}

// parseUnitSource parses one ordinary unit file without inspecting its members,
// for cases whose whole point is that parsing must fail.
func parseUnitSource(t *testing.T, source string) {
	t.Helper()
	parser.Parse(source, "kinds", ".", "probe.unit.fol", "types", "program", "program", true)
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
		unitMember(t, "_ co.lang.unit = {\n    Fn co.lang.type = (A, B)->(R);\n}")
	})

	for _, tc := range []struct {
		name   string
		source string
	}{
		{"type-list-without-arrow", `Fn co.lang.type = (A, B);`},
		{"single-type-trailing-comma", `Fn co.lang.type = (A,);`},
		{"function-type-trailing-comma", `Fn co.lang.type = (A,)->(R);`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseUnitSource(t, "_ co.lang.unit = {\n    "+tc.source+"\n}")
			})
		})
	}

	t.Run("delegate-trailing-comma", func(t *testing.T) {
		mustPanic(t, func() {
			packagePrimary(t, `_ co.lang.delegate = (A,)->(R);`, "Fn.fol")
		})
	})
}
