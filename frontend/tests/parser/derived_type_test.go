package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// TestDerivedTypesReachTheAST asserts that a derivation written in a position with no
// declaration statement to record it on — a parameter, a result, a receiver, a type
// alias, a function type's components — survives into the AST as an ast.DerivedType.
//
// Parsing alone does not prove this: the conformance fixtures accepted these forms
// while the parser silently kept only the element type, so `p co.lang.int->(**)` became
// an ordinary co.lang.int parameter. Only inspecting the node catches that.
func TestDerivedTypesReachTheAST(t *testing.T) {
	const source = `_ co.lang.unit = {
    derived(
        p co.lang.int->(**),
        r co.lang.int->(&&),
        a co.lang.int->([2][3]),
        s co.lang.int->([:]),
        plain co.lang.int
    )->(co.lang.int->(*), named co.lang.int->([5])) = {
        this.return p, a;
    }

    (recv Employee->(&)) method()->(co.lang.int) = { this.return 0; }
}
`

	fn := unitFunction(t, source, "derived")
	params := flatParameters(fn)

	assertDerived(t, "parameter p", params["p"], ast.DerivePointer, func(d ast.DerivedType) {
		if d.PointerCount != 2 {
			t.Errorf("parameter p: pointer depth = %d, want 2", d.PointerCount)
		}
	})

	assertDerived(t, "parameter r", params["r"], ast.DeriveReference, func(d ast.DerivedType) {
		if d.RefCount != 2 {
			t.Errorf("parameter r: reference count = %d, want 2", d.RefCount)
		}
	})

	// The jagged array is the case a single dimension slot cannot express: recording
	// only the first group knew there were two but not that the second was 3.
	assertDerived(t, "parameter a", params["a"], ast.DeriveArray, func(d ast.DerivedType) {
		if !d.IsJagged() {
			t.Errorf("parameter a: IsJagged() = false, want true")
		}
		if got := len(d.DimGroups); got != 2 {
			t.Fatalf("parameter a: dimension groups = %d, want 2", got)
		}
		if got := dimensionValue(t, d.DimGroups[0]); got != 2 {
			t.Errorf("parameter a: first group = %d, want 2", got)
		}
		if got := dimensionValue(t, d.DimGroups[1]); got != 3 {
			t.Errorf("parameter a: second group = %d, want 3", got)
		}
	})

	assertDerived(t, "parameter s", params["s"], ast.DeriveSlice, nil)

	// An undecorated type must NOT gain a wrapper, so the common case is unchanged.
	if _, wrapped := params["plain"].(ast.DerivedType); wrapped {
		t.Errorf("parameter plain: got ast.DerivedType, want the bare element type")
	}

	if len(fn.ReturnType) != 2 {
		t.Fatalf("results = %d, want 2", len(fn.ReturnType))
	}
	assertDerived(t, "result 0", fn.ReturnType[0].Type_, ast.DerivePointer, func(d ast.DerivedType) {
		if d.PointerCount != 1 {
			t.Errorf("result 0: pointer depth = %d, want 1", d.PointerCount)
		}
	})
	assertDerived(t, "named result", fn.ReturnType[1].Type_, ast.DeriveArray, nil)

	// The receiver clause is a third position with nowhere else to put a derivation.
	method := unitFunction(t, source, "method")
	if method.AssociatedReceiver == nil {
		t.Fatal("method: no receiver recorded")
	}
	recv, ok := method.AssociatedReceiver.SymbolStmt.(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("method: receiver is %T, want ast.VarDeclarationStmt", method.AssociatedReceiver.SymbolStmt)
	}
	assertDerived(t, "receiver", recv.Type_, ast.DeriveReference, nil)
}

// TestDerivedTypesInAliasesAndFunctionTypes covers the two remaining slots: a type
// alias's definition and the parameters and results of a function type.
func TestDerivedTypesInAliasesAndFunctionTypes(t *testing.T) {
	alias := aliasDefinition(t, "ptrAlias co.lang.type = co.lang.int->(*);\n")
	assertDerived(t, "alias definition", alias, ast.DerivePointer, func(d ast.DerivedType) {
		if d.PointerCount != 1 {
			t.Errorf("alias: pointer depth = %d, want 1", d.PointerCount)
		}
	})

	fnAlias := aliasDefinition(t,
		"fnAlias co.lang.type = (co.lang.int->(*), co.lang.int->([4]))->(co.lang.int->(&));\n")
	fnType, ok := fnAlias.(ast.FunctionType)
	if !ok {
		t.Fatalf("fnAlias: definition is %T, want ast.FunctionType", fnAlias)
	}

	var params []ast.Parameter
	for _, list := range fnType.Params {
		params = append(params, list...)
	}
	if len(params) != 2 {
		t.Fatalf("fnAlias: parameters = %d, want 2", len(params))
	}
	assertDerived(t, "fnAlias parameter 0", params[0].Type_, ast.DerivePointer, nil)
	assertDerived(t, "fnAlias parameter 1", params[1].Type_, ast.DeriveArray, nil)

	if len(fnType.Results) != 1 {
		t.Fatalf("fnAlias: results = %d, want 1", len(fnType.Results))
	}
	assertDerived(t, "fnAlias result 0", fnType.Results[0].Type_, ast.DeriveReference, nil)
}

// TestTypeArgumentsKeepDerivations covers the type-ARGUMENT slot, which has no
// declaration to record a derivation on either: Vector(co.lang.int->(*)) must keep the
// pointer on its argument.
func TestTypeArgumentsKeepDerivations(t *testing.T) {
	fn := unitFunction(t, `_ co.lang.unit = {
    f(v Vector(co.lang.int->(*)))->(co.lang.int) = { this.return 0; }
}
`, "f")

	applied, ok := flatParameters(fn)["v"].(ast.CompoundType)
	if !ok {
		t.Fatalf("parameter v: type is %T, want ast.CompoundType", flatParameters(fn)["v"])
	}
	assertDerived(t, "type argument", applied.Right, ast.DerivePointer, func(d ast.DerivedType) {
		if d.PointerCount != 1 {
			t.Errorf("type argument: pointer depth = %d, want 1", d.PointerCount)
		}
	})
}

// TestDerivedTypesInComposedTypeExpressions covers the recursive type-expression slots.
// Each slot must retain the complete arm/body/result type because no declaration statement
// exists inside a forall, union or function type to carry a derivation separately.
func TestDerivedTypesInComposedTypeExpressions(t *testing.T) {
	poly := aliasDefinition(t, "poly co.lang.type = forall(T).T->(*);\n")
	forall, ok := poly.(ast.ForAllType)
	if !ok {
		t.Fatalf("poly: definition is %T, want ast.ForAllType", poly)
	}
	assertDerived(t, "forall body", forall.Inner, ast.DerivePointer, nil)

	sum := aliasDefinition(t, "sum co.lang.type = Left->(*) | Right->(&);\n")
	union, ok := sum.(ast.CompoundType)
	if !ok || union.Op != "|" {
		t.Fatalf("sum: definition is %#v, want ast.CompoundType union", sum)
	}
	assertDerived(t, "left union arm", union.Left, ast.DerivePointer, nil)
	assertDerived(t, "right union arm", union.Right, ast.DeriveReference, nil)

	arrow := aliasDefinition(t, "arrow co.lang.type = (Input)->Output->(*);\n")
	function, ok := arrow.(ast.FunctionType)
	if !ok {
		t.Fatalf("arrow: definition is %T, want ast.FunctionType", arrow)
	}
	if len(function.Results) != 1 {
		t.Fatalf("arrow: results = %d, want 1", len(function.Results))
	}
	assertDerived(t, "bare arrow result", function.Results[0].Type_, ast.DerivePointer, nil)

	// A second derivation is expressed by grouping the already-derived base. The
	// outer wrapper must point to the inner wrapper rather than replacing it.
	nested := aliasDefinition(t, "nested co.lang.type = (Element->(*))->(&);\n")
	assertDerived(t, "outer grouped derivation", nested, ast.DeriveReference, func(outer ast.DerivedType) {
		assertDerived(t, "inner grouped derivation", outer.Underlying, ast.DerivePointer, nil)
	})
}

// TestDerivedTypesInRemainingDeclarationSlots verifies the declaration forms that do not
// lower through lowerDeclarator. They all store an ast.Type directly and therefore must use
// typeRef.fullType rather than its element-only Node field.
func TestDerivedTypesInRemainingDeclarationSlots(t *testing.T) {
	structDecl := packagePrimary(t, "_ co.lang.struct = { (Element->(*))->(&); }\n", "Container.fol")
	container, ok := structDecl.(ast.TypeDeclarationStmt)
	if !ok || len(container.Body) != 1 {
		t.Fatalf("embedded-field fixture produced %#v", structDecl)
	}
	embedded, ok := container.Body[0].(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("embedded field is %T, want ast.VarDeclarationStmt", container.Body[0])
	}
	assertDerived(t, "embedded field", embedded.Type_, ast.DeriveReference, func(outer ast.DerivedType) {
		assertDerived(t, "embedded field element", outer.Underlying, ast.DerivePointer, nil)
	})

	kindDecl := packagePrimary(t, "_ co.lang.kind = co.lang.int->(*);\n", "PointerKind.fol")
	kind, ok := kindDecl.(ast.TypeDeclarationStmt)
	if !ok {
		t.Fatalf("general kind is %T, want ast.TypeDeclarationStmt", kindDecl)
	}
	assertDerived(t, "general-kind binding", kind.Type_, ast.DerivePointer, nil)

	fn := unitFunction(t, `_ co.lang.unit = {
    keep(xs Values)->() = {
        let p co.lang.int->(*) = xs;
        xs.map(|q co.lang.int->(*)| => q);
    }
}
`, "keep")
	if len(fn.Body) != 2 {
		t.Fatalf("keep body has %d statements, want 2", len(fn.Body))
	}
	letDecl, ok := fn.Body[0].(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("typed let is %T, want ast.VarDeclarationStmt", fn.Body[0])
	}
	assertDerived(t, "typed let", letDecl.Type_, ast.DerivePointer, nil)

	exprStmt, ok := fn.Body[1].(ast.ExpressionStmt)
	if !ok {
		t.Fatalf("lambda statement is %T, want ast.ExpressionStmt", fn.Body[1])
	}
	call, ok := exprStmt.Expression.(ast.CallExpr)
	if !ok || len(call.Arguments) != 1 {
		t.Fatalf("lambda expression is %#v, want one-argument ast.CallExpr", exprStmt.Expression)
	}
	lambda, ok := call.Arguments[0].(ast.LambdaExpr)
	if !ok || len(lambda.Parameters) != 1 {
		t.Fatalf("callback is %#v, want one-parameter ast.LambdaExpr", call.Arguments[0])
	}
	assertDerived(t, "lambda parameter", lambda.Parameters[0].Type_, ast.DerivePointer, nil)
}

// TestDependentTypeConstructorKeepsSignatureAndBinding asserts on information that syntax-only
// conformance cannot see. The value parameter `n` must remain available to resolve the array
// dimension in the constructed type.
func TestDependentTypeConstructorKeepsSignatureAndBinding(t *testing.T) {
	primary := unitMember(t, `_ co.lang.unit = {
    Vector(n co.lang.int)->(co.lang.dependentType) = co.lang.int->([n]);
}
`)
	constructor, ok := primary.(ast.TypeDeclarationStmt)
	if !ok {
		t.Fatalf("constructor is %T, want ast.TypeDeclarationStmt", primary)
	}
	if constructor.SubType_ != "TYPE_CONSTRUCTOR" {
		t.Fatalf("constructor subtype = %q, want TYPE_CONSTRUCTOR", constructor.SubType_)
	}
	if len(constructor.Parameters) != 1 || len(constructor.Parameters[0]) != 1 {
		t.Fatalf("constructor parameters = %#v, want one parameter list containing n", constructor.Parameters)
	}
	if got := logicalName(constructor.Parameters[0][0].Name_); got != "n" {
		t.Fatalf("constructor parameter = %q, want n", got)
	}
	if len(constructor.ReturnType) != 1 {
		t.Fatalf("constructor results = %d, want 1", len(constructor.ReturnType))
	}
	assertDerived(t, "constructed type", constructor.Type_, ast.DeriveArray, func(array ast.DerivedType) {
		if len(array.DimGroups) != 1 || len(array.DimGroups[0]) != 1 {
			t.Fatalf("constructed array dimensions = %#v, want one group containing n", array.DimGroups)
		}
		index, ok := array.DimGroups[0][0].(ast.SymbolExpr)
		if !ok || logicalName(index.Value) != "n" {
			t.Fatalf("constructed array index = %#v, want symbol n", array.DimGroups[0][0])
		}
	})
}

// TestTypeListsKeepDerivedPayloads covers both consumers of the type-list production: enum
// constructor payloads and algebraic data variants.
func TestTypeListsKeepDerivedPayloads(t *testing.T) {
	enumPrimary := packagePrimary(t,
		"_ co.lang.enum = { Item(co.lang.int->(*)) }\n", "Payload.fol")
	enumDecl := enumPrimary.(ast.TypeDeclarationStmt)
	variant := enumDecl.Body[0].(ast.VarDeclarationStmt)
	variantType, ok := variant.Type_.(ast.FunctionType)
	if !ok || len(variantType.Params) != 1 || len(variantType.Params[0]) != 1 {
		t.Fatalf("enum payload type = %#v, want one-parameter ast.FunctionType", variant.Type_)
	}
	assertDerived(t, "enum payload", variantType.Params[0][0].Type_, ast.DerivePointer, nil)

	dataPrimary := unitMember(t,
		"_ co.lang.unit = {\n    PayloadData co.lang.data = Item(co.lang.int->(*));\n}\n")
	dataDecl, ok := dataPrimary.(ast.TypeConstructorStmt)
	if !ok || len(dataDecl.Variants) != 1 || len(dataDecl.Variants[0].PayloadTypes) != 1 {
		t.Fatalf("data payload = %#v, want one lossless payload type", dataPrimary)
	}
	assertDerived(t, "data payload", dataDecl.Variants[0].PayloadTypes[0], ast.DerivePointer, nil)
}

// TestRecordedTypeNamesAreNames asserts that a declaration records the NAME of its
// type, not a symbol category.
//
// GetActType returns a pair whose halves mean different things per node, and reading
// the wrong half collapsed every user-defined type to the literal "Type" and every
// applied generic to "CDT". Those strings then reached symbol metadata and the
// declaration nodes, so two unrelated types became indistinguishable.
func TestRecordedTypeNamesAreNames(t *testing.T) {
	fn := unitFunction(t, `_ co.lang.unit = {
    f(a Employee, b co.lang.int, c Vector(co.lang.int), d co.lang.int->(*))->(co.lang.int) = {
        this.return 0;
    }
}
`, "f")

	want := map[string]string{
		"a": "Employee", // was "Type"
		"b": "co.lang.int",
		"c": "Vector", // was "CDT"
		"d": "co.lang.int",
	}

	for _, list := range fn.Parameters {
		for _, prm := range list {
			decl, ok := prm.SymbolDeclStmt.(ast.VarDeclarationStmt)
			if !ok {
				continue
			}
			name := logicalName(prm.Name_)
			expected, tracked := want[name]
			if !tracked {
				continue
			}
			if got := logicalName(decl.VarType); got != expected {
				t.Errorf("parameter %s: recorded type = %q, want %q", name, got, expected)
			}
		}
	}
}

// flatParameters indexes a function's parameters by their logical name, flattening the
// curried lists so a caller does not care which list a parameter came from.
func flatParameters(fn ast.FunctionDeclarationStmt) map[string]ast.Type {
	out := map[string]ast.Type{}
	for _, list := range fn.Parameters {
		for _, prm := range list {
			out[logicalName(prm.Name_)] = prm.Type_
		}
	}
	return out
}

// assertDerived checks that slot holds an ast.DerivedType of the expected form, then
// runs any form-specific assertions.
func assertDerived(t *testing.T, what string, slot ast.Type, want ast.DerivationForm, extra func(ast.DerivedType)) {
	t.Helper()
	derived, ok := slot.(ast.DerivedType)
	if !ok {
		t.Errorf("%s: type is %T, want ast.DerivedType (the derivation was discarded)", what, slot)
		return
	}
	if derived.Form != want {
		t.Errorf("%s: form = %q, want %q", what, derived.Form, want)
	}
	if derived.Underlying == nil {
		t.Errorf("%s: underlying element type is nil", what)
	}
	if extra != nil {
		extra(derived)
	}
}

// dimensionValue reads the single integer dimension out of a group.
func dimensionValue(t *testing.T, group []ast.Expr) int64 {
	t.Helper()
	if len(group) != 1 {
		t.Fatalf("dimension group holds %d entries, want 1", len(group))
	}
	lit, ok := group[0].(ast.IntegerLiteral)
	if !ok {
		t.Fatalf("dimension is %T, want ast.IntegerLiteral", group[0])
	}
	return lit.Value
}

// unitFunction parses source as a package source file and returns the named function.
func unitFunction(t *testing.T, source, name string) ast.FunctionDeclarationStmt {
	t.Helper()
	root, _, _, _ := parser.Parse(source, "derived", ".", "Probe.unit.fol", "", "program", "program", true)

	pkg, ok := root.(ast.PackageStmt)
	if !ok {
		t.Fatalf("root is %T, want ast.PackageStmt", root)
	}
	for _, decl := range pkg.Body {
		unit, ok := decl.(ast.TypeDeclarationStmt)
		if !ok {
			continue
		}
		for _, member := range unit.Body {
			fn, ok := member.(ast.FunctionDeclarationStmt)
			if ok && logicalName(fn.Name) == name {
				return fn
			}
		}
	}
	t.Fatalf("function %q not found", name)
	return ast.FunctionDeclarationStmt{}
}

// aliasDefinition parses a single type alias and returns its definition type.
//
// A type declaration is a unit member (DECISION-FILE-003), so the alias is wrapped
// in the unit file that now owns it rather than standing alone as a primary.
func aliasDefinition(t *testing.T, source string) ast.Type {
	t.Helper()

	member := unitMember(t, "_ co.lang.unit = {\n"+source+"}\n")
	td, ok := member.(ast.TypeDeclarationStmt)
	if !ok {
		t.Fatalf("unit member is %T, want ast.TypeDeclarationStmt", member)
	}
	return td.Type_
}

// unitMember parses one ordinary unit file and returns its single member.
func unitMember(t *testing.T, source string) ast.Stmt {
	t.Helper()
	root, _, _, _ := parser.Parse(source, "derived", ".", "probe.unit.fol", "types", "program", "program", true)

	pkg, ok := root.(ast.PackageStmt)
	if !ok {
		t.Fatalf("root is %T, want ast.PackageStmt", root)
	}
	for _, decl := range pkg.Body {
		unit, ok := decl.(ast.TypeDeclarationStmt)
		if !ok || unit.Kind != "co.lang.unit" {
			continue
		}
		if len(unit.Body) != 1 {
			t.Fatalf("unit body has %d members, want 1", len(unit.Body))
		}
		return unit.Body[0]
	}
	t.Fatal("no unit declaration found")
	return nil
}

// packagePrimary parses one package-source primary declaration and returns it without the
// package envelope. A dedicated helper keeps AST-focused tests explicit about the source form
// they expect instead of silently accepting an entry-file reclassification.
func packagePrimary(t *testing.T, source, basename string) ast.Stmt {
	t.Helper()
	root, _, _, _ := parser.Parse(source, "derived", ".", basename, "types", "program", "program", true)

	pkg, ok := root.(ast.PackageStmt)
	if !ok {
		t.Fatalf("root is %T, want ast.PackageStmt", root)
	}
	if len(pkg.Body) != 1 {
		t.Fatalf("package body has %d declarations, want 1", len(pkg.Body))
	}
	return pkg.Body[0]
}

// logicalName strips the scanner's "_fo" mangling suffix from a declared name.
func logicalName(scanned string) string {
	if idx := len(scanned) - len("_fo"); idx > 0 && scanned[idx:] == "_fo" {
		return scanned[:idx]
	}
	return scanned
}
