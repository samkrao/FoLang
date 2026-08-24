package parser

import (
	"sort"
	"strings"
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

// What the parse binds into the segments scope_test.go builds.
//
// A segment's Symboldetails map is what a name lookup searches
// (docs/language-ref.md, B.4), so these tests ask the question the structural
// ones cannot: not whether the right segments exist, but whether each name
// landed in the one that owns it.

// TestDeclarationsBindIntoTheirOwnSegment parses the B.1 reference unit and
// checks the drawing's symbol lists, segment by segment.
//
// The interesting one is the block's `j`: the outer `j` stays in the function's
// second segment, so the two are distinct bindings rather than one name declared
// twice.
func TestDeclarationsBindIntoTheirOwnSegment(t *testing.T) {
	_, p := parsePackageSource(t, referenceUnit, "some.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the reference unit produced diagnostics: %v", p.diags)
	}

	unit := onlyChild(t, p.fs, p.ctx)
	function := p.fs.GetContext(unit.ChildCtxIds[0])
	segments := segmentChain(p.fs, function)

	if got := boundNames(segments[1]); got != "k, v" {
		t.Errorf("the function's first segment binds %q, want %q", got, "k, v")
	}
	if got := boundNames(segments[0]); got != "j" {
		t.Errorf("the function's second segment binds %q, want %q", got, "j")
	}

	block := onlyChild(t, p.fs, function)
	if got := boundNames(segmentChain(p.fs, block)[0]); got != "j" {
		t.Errorf("the block segment binds %q, want the block-local %q", got, "j")
	}
}

// TestFunctionNameBindsWhereItIsDeclaredAndItsSignatureInsideItself covers the
// split B.1 draws: `firstfun` is a member of the unit, while what its parameter
// list introduces belongs to the function's own context.
//
// Binding a parameter in the declaring scope instead would leak it to every
// sibling and make two functions that share a parameter name collide.
func TestFunctionNameBindsWhereItIsDeclaredAndItsSignatureInsideItself(t *testing.T) {
	source := `_ co.lang.unit = {
    scale(factor co.lang.int)->(scaled co.lang.int) = {
        this.return factor * 2;
    }

    shift(factor co.lang.int)->(co.lang.int) = {
        this.return factor + 1;
    }
}`

	_, p := parsePackageSource(t, source, "members.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the source produced diagnostics: %v", p.diags)
	}

	unit := onlyChild(t, p.fs, p.ctx)
	if got := boundNames(segmentChain(p.fs, unit)[0]); got != "scale, shift" {
		t.Errorf("the unit body binds %q, want both function names and nothing else", got)
	}

	scale := p.fs.GetContext(unit.ChildCtxIds[0])
	if got := boundNames(segmentChain(p.fs, scale)[0]); got != "factor, scaled" {
		t.Errorf("the function context binds %q, want its parameter and its named result", got)
	}
}

// TestOverloadsBindSideBySide checks that a function's key carries its signature:
// two declarations of one name that differ in their parameters are two bindings,
// and a third that repeats a signature is the redeclaration.
func TestOverloadsBindSideBySide(t *testing.T) {
	source := `_ co.lang.unit = {
    show(value co.lang.int)->() = { }

    show(value co.lang.string)->() = { }
}`

	_, p := parsePackageSource(t, source, "overloads.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("two overloads produced diagnostics: %v", p.diags)
	}

	unit := onlyChild(t, p.fs, p.ctx)
	if bound := len(segmentChain(p.fs, unit)[0].Symboldetails); bound != 2 {
		t.Errorf("the unit body holds %d bindings, want one per overload", bound)
	}

	_, repeated := parsePackageSource(t, `_ co.lang.unit = {
    show(value co.lang.int)->() = { }

    show(other co.lang.int)->() = { }
}`, "overloads.unit.fol")
	assertDiagnostic(t, repeated, "show is already declared in this scope with the same parameter signature")
}

// TestReturnTypeDoesNotDistinguishTwoOverloads is the reference's own invalid
// example: two declarations that differ only in their result.
//
// A return type never participates in overload selection, so these are one
// signature declared twice rather than two overloads.
func TestReturnTypeDoesNotDistinguishTwoOverloads(t *testing.T) {
	_, p := parsePackageSource(t, `_ co.lang.unit = {
    select(x Animal)->(Animal) = { }

    select(x Animal)->(Dog) = { }
}`, "returns.unit.fol")

	assertDiagnostic(t, p, "select is already declared in this scope with the same parameter signature")
}

// TestOverloadFamilyKeepsOneReturnSignature covers the invariant that holds across
// siblings: parameters may vary, the declared result contract may not.
func TestOverloadFamilyKeepsOneReturnSignature(t *testing.T) {
	_, ok := parsePackageSource(t, `_ co.lang.unit = {
    collide(a Animal, b Animal)->(co.lang.bool) = { }

    collide(a Dog, b Cat)->(co.lang.bool) = { }
}`, "family.unit.fol")
	if len(ok.diags) != 0 {
		t.Fatalf("siblings sharing a return signature produced diagnostics: %v", ok.diags)
	}

	_, p := parsePackageSource(t, `_ co.lang.unit = {
    transform(a Animal)->(Animal) = { }

    transform(a Dog)->(Dog) = { }
}`, "family.unit.fol")

	assertDiagnostic(t, p, "every declaration of transform must declare the same return signature")
}

// TestNonOverloadableFormsHaveNoFamily covers the signature categories that cannot
// be overloaded at all, so that a second declaration is invalid however its
// parameters differ.
func TestNonOverloadableFormsHaveNoFamily(t *testing.T) {
	for _, form := range []struct {
		name    string
		source  string
		because string
	}{
		{
			name: "named return",
			source: `_ co.lang.unit = {
    total(a co.lang.int)->(sum co.lang.int) = { }

    total(a co.lang.float)->(sum co.lang.int) = { }
}`,
			because: "a named return",
		},
		{
			name: "multiple returns",
			source: `_ co.lang.unit = {
    split(a co.lang.int)->(co.lang.int, co.lang.bool) = { }

    split(a co.lang.float)->(co.lang.int, co.lang.bool) = { }
}`,
			because: "multiple returns",
		},
		{
			name: "pointer in the signature",
			source: `_ co.lang.unit = {
    store(a co.lang.int->(*))->() = { }

    store(a co.lang.float)->() = { }
}`,
			because: "a pointer in its signature",
		},
	} {
		t.Run(form.name, func(t *testing.T) {
			_, p := parsePackageSource(t, form.source, "restricted.unit.fol")
			assertDiagnostic(t, p, "cannot be overloaded")
			assertDiagnostic(t, p, form.because)
		})
	}
}

// TestRedeclarationInOneSegmentIsReported covers the collision itself. A name
// bound twice in one segment would otherwise lose one of the two declarations
// with nothing said about it.
func TestRedeclarationInOneSegmentIsReported(t *testing.T) {
	_, p := parsePackageSource(t, `_ co.lang.unit = {
    subject()->() = {
        total co.lang.int = 1;
        total co.lang.int = 2;
    }
}`, "clash.unit.fol")

	assertDiagnostic(t, p, "total is already declared in this scope")
}

// TestRedeclarationAcrossSegmentsIsNotAClash is the other half of that rule: a
// declaration written after a statement opens a new frontier, so it binds in a
// segment of its own and shadows rather than collides.
func TestRedeclarationAcrossSegmentsIsNotAClash(t *testing.T) {
	_, p := parsePackageSource(t, `_ co.lang.unit = {
    subject()->() = {
        total := 1;
        co.out.println(total);
        total co.lang.int = 2;
    }
}`, "shadow.unit.fol")

	if len(p.diags) != 0 {
		t.Fatalf("a declaration in a later frontier produced diagnostics: %v", p.diags)
	}

	function := p.fs.GetContext(onlyChild(t, p.fs, p.ctx).ChildCtxIds[0])
	segments := segmentChain(p.fs, function)
	if len(segments) != 2 {
		t.Fatalf("the function owns %d segments, want 2", len(segments))
	}
	for _, segment := range segments {
		if got := boundNames(segment); got != "total" {
			t.Errorf("segment %s binds %q, want each frontier to hold its own %q", segment.Id, got, "total")
		}
	}
}

// TestSpeculationLeavesNoBindingBehind is the binding half of
// TestSpeculationLeavesNoContextBehind.
//
// The lambda body is read twice, once speculatively. A binding left by the
// abandoned reading would sit in a context nothing references, and — worse — a
// name bound twice by one declaration would report the accepted reading as a
// redeclaration of the rejected one.
func TestSpeculationLeavesNoBindingBehind(t *testing.T) {
	_, p := parsePackageSource(t, `_ co.lang.unit = {
    subject()->(co.lang.int) = {
        base := 1;
        values := [1, 2, 3];
        values.map(|v| => { v + base })
    }
}`, "tail.unit.fol")

	if len(p.diags) != 0 {
		t.Fatalf("the source produced diagnostics: %v", p.diags)
	}

	seen := map[string]int{}
	for _, table := range p.fs.SymboltableMap {
		for _, symbol := range table.Symboldetails {
			seen[logicalName(symbol.GetName())]++
		}
	}
	for _, declared := range []string{"base", "values", "v"} {
		if seen[declared] != 1 {
			t.Errorf("%q is bound %d times across the model, want once", declared, seen[declared])
		}
	}
}

// boundNames lists a segment's bindings by their source spelling, sorted, so a
// test can state a segment's contents as one string.
func boundNames(table *symboltable.SymbolTable) string {
	names := make([]string, 0, len(table.Symboldetails))
	for _, symbol := range table.Symboldetails {
		names = append(names, logicalName(symbol.GetName()))
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// assertDiagnostic fails unless the parse reported one diagnostic containing want.
func assertDiagnostic(t *testing.T, p *parser, want string) {
	t.Helper()
	for _, diagnostic := range p.diags {
		if strings.Contains(diagnostic.Error(), want) {
			return
		}
	}
	t.Fatalf("no diagnostic contains %q; got %v", want, p.diags)
}
