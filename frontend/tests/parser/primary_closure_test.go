package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// These tests fix the three revision-27 decisions that closed
// primary-declaration. Each one moved a declaration OUT of the file-backed
// primary set, so each has two halves that must hold together: the form is
// accepted in its new home, and the "_" spelling that used to be required is
// rejected in the old one. Testing only the rejection would pass on a parser
// that had simply dropped the construct.

// DECISION-DECL-002: a function object and a delegate are unit members. Both
// take an ordinary identifier, because one unit file carries several members and
// no filename can name them all.
//
// The forward spelling `<name> co.lang.function;` is deliberately NOT asserted
// here. Whether it is expressible at all is OQ-001 in
// docs/grammar/OPEN-QUESTIONS.md; a test either way would freeze an open
// question into the suite.
func TestFunctionObjectAndDelegateAreUnitMembers(t *testing.T) {
	members := unitMembers(t, `_ co.lang.unit = {
    someFArg co.lang.function = (a co.lang.int, b co.lang.int)->(co.lang.int) = {
        this.return a + b;
    }

    oObj co.lang.function = add;

    someDelegate co.lang.delegate = (a co.lang.int)->(co.lang.int);
}`)

	if len(members) != 3 {
		t.Fatalf("unit body has %d members, want 3", len(members))
	}

	inline, ok := members[0].(ast.FunctionDeclarationStmt)
	if !ok {
		t.Fatalf("function object is %T, want ast.FunctionDeclarationStmt", members[0])
	}
	if logicalName(inline.Name) != "someFArg" {
		t.Errorf("function object name = %q, want someFArg", logicalName(inline.Name))
	}
	if !inline.Symb.FunctionObject || !inline.Symb.IsBody {
		t.Errorf("inline function object symbol = %#v, want a function object with a body", inline.Symb)
	}

	// The other half of DECISION-SYN-007: an expression binding is not the
	// declaration's inline body, so it ends at ";" rather than at "}".
	bound, ok := members[1].(ast.FunctionDeclarationStmt)
	if !ok {
		t.Fatalf("bound function object is %T, want ast.FunctionDeclarationStmt", members[1])
	}
	if !bound.Symb.FunctionObject || bound.Symb.IsBody {
		t.Errorf("bound function object symbol = %#v, want a function object without an inline body", bound.Symb)
	}

	delegate, ok := members[2].(ast.DelegateStmt)
	if !ok {
		t.Fatalf("delegate is %T, want ast.DelegateStmt", members[2])
	}
	if logicalName(delegate.Symb.Name) != "someDelegate" {
		t.Errorf("delegate name = %q, want someDelegate", logicalName(delegate.Symb.Name))
	}
}

// DECISION-DECL-003: a named block is a statement. The reference states a block
// cannot live outside a function or method, which is precisely why it is not a
// file-backed primary.
func TestNamedBlockIsAStatement(t *testing.T) {
	fn := unitFunction(t, `_ co.lang.unit = {
    run()->() = {
        labelBlock co.lang.block = {
            x co.lang.int = 1;
        }
        labelBlock.expand();
    }
}`, "run")

	if len(fn.Body) != 2 {
		t.Fatalf("function body has %d statements, want 2", len(fn.Body))
	}
	block, ok := fn.Body[0].(*ast.BlockStmt)
	if !ok {
		t.Fatalf("named block is %T, want *ast.BlockStmt", fn.Body[0])
	}
	if !block.Symb.IsNamed || logicalName(block.Symb.Name_) != "labelBlock" {
		t.Errorf("named block symbol = %#v, want a named block called labelBlock", block.Symb)
	}
	if len(block.Body) != 1 {
		t.Errorf("named block holds %d statements, want 1", len(block.Body))
	}
}

// DECISION-DECL-001: primary-declaration is closed. Each of these was an
// alternative before revision 27 and must now be rejected in a `<Name>.fol`
// package source file.
func TestClosedPrimaryDeclarationRejectsRelocatedForms(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   string
		basename string
	}{
		{"function-object", `_ co.lang.function = add;`, "SomeFArg.fol"},
		{"function-object-inline", "_ co.lang.function = (a co.lang.int)->(co.lang.int) = {\n    this.return a;\n}", "SomeFArg.fol"},
		{"delegate", `_ co.lang.delegate = (co.lang.int)->(co.lang.string);`, "Transform.fol"},
		{"named-block", "_ co.lang.block = {\n}", "LabelBlock.fol"},
		{"annotated-contract", "@co.dap.Functor\n_ = {\n    map(v co.lang.int)->(co.lang.int);\n}", "Functor.fol"},
		// general-kind-declaration admitted every one of these; none has a
		// declaration form in the reference.
		{"general-kind-trait", "_ co.lang.trait = {\n    label co.lang.string;\n}", "Labelled.fol"},
		{"general-kind-macro", "_ co.lang.macro = { }", "Twice.fol"},
		{"general-kind-alias", `_ co.lang.alias = co.lang.int;`, "Count.fol"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				packagePrimary(t, tc.source, tc.basename)
			})
		})
	}
}

// A unit member names itself, so the "_" head that a file-backed primary
// requires is wrong there. The diagnostic has to say so: "_" was the correct
// spelling for these two declarations until revision 27, and a reader who wrote
// it has the right declaration in the wrong source form.
func TestUnitMembersRejectTheFilenameDerivedHead(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{"function-object", "_ co.lang.unit = {\n    _ co.lang.function = add;\n}"},
		{"delegate", "_ co.lang.unit = {\n    _ co.lang.delegate = (co.lang.int)->(co.lang.int);\n}"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				unitMembers(t, tc.source)
			})
		})
	}
}

// The nested-declaration guard of DECISION-SYN-008 must keep rejecting every
// OTHER kind-introduced declaration in a block. co.lang.block is the single
// exception the statement dispatcher claims before the guard runs.
func TestNamedBlockExceptionDoesNotOpenNestedKinds(t *testing.T) {
	mustPanic(t, func() {
		unitFunction(t, `_ co.lang.unit = {
    run()->() = {
        Inner co.lang.struct = {
            value co.lang.int;
        }
    }
}`, "run")
	})
}

// unitMembers parses one ordinary unit file and returns every member of its
// single unit declaration.
func unitMembers(t *testing.T, source string) []ast.Stmt {
	t.Helper()

	for _, decl := range parseRegressionFile(t, source, "probe.unit.fol") {
		if unit, ok := decl.(ast.TypeDeclarationStmt); ok && unit.Kind == "co.lang.unit" {
			return unit.Body
		}
	}
	t.Fatalf("no unit declaration found in %s", source)
	return nil
}
