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
func aliasDefinition(t *testing.T, source string) ast.Type {
	t.Helper()
	root, _, _, _ := parser.Parse(source, "derived", ".", "Probe.fol", "", "program", "program", true)

	pkg, ok := root.(ast.PackageStmt)
	if !ok {
		t.Fatalf("root is %T, want ast.PackageStmt", root)
	}
	for _, decl := range pkg.Body {
		if td, ok := decl.(ast.TypeDeclarationStmt); ok {
			return td.Type_
		}
	}
	t.Fatal("no type declaration found")
	return nil
}

// logicalName strips the scanner's "_fo" mangling suffix from a declared name.
func logicalName(scanned string) string {
	if idx := len(scanned) - len("_fo"); idx > 0 && scanned[idx:] == "_fo" {
		return scanned[:idx]
	}
	return scanned
}

