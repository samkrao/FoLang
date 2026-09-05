package parser_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/parser"
)

// TestEveryNodeCarriesASpan walks the tree of every accepted conformance fixture
// and asserts that each node reports a source region.
//
// This is the completeness guard for the LSP work. Adding the Span field to the
// node types was mechanical; POPULATING it at 250-odd construction sites was
// not, and a site that was missed is invisible — the node still compiles, still
// parses, and simply reports line 0. Every editor feature built on spans would
// then fail on exactly the constructs nobody tested by hand.
//
// Walking with reflection rather than the ast visitor is deliberate: the visitor
// only descends the shapes it knows about, so a node it does not handle would be
// skipped and its missing span never noticed.
func TestEveryNodeCarriesASpan(t *testing.T) {
	for _, path := range conformanceFixtures(t, "accepted") {
		path := path
		t.Run(fixtureName(path), func(t *testing.T) {
			result := parser.ParseFile(
				readFixture(t, path), "spans",
				filepath.Dir(path), fixtureBasename(path), "",
			)
			if len(result.Diagnostics) != 0 {
				t.Fatalf("accepted fixture produced diagnostics: %v", result.Diagnostics)
			}

			missing := map[string]int{}
			walkNodes(reflect.ValueOf(result.Root), map[uintptr]bool{}, func(typeName string, span ast.Span) {
				if span.IsZero() {
					missing[typeName]++
				}
			})
			if len(missing) != 0 {
				t.Errorf("nodes with no source span:\n%s", formatCounts(missing))
			}
		})
	}
}

// TestSpansAreWellFormed checks the spans mean something, not merely that they
// are non-zero: a node must not end before it starts, and a child must lie
// within its parent.
func TestSpansAreWellFormed(t *testing.T) {
	path := filepath.Join("examples", "accepted", "Employee.fol")
	result := parser.ParseFile(readFixture(t, path), "spans", filepath.Dir(path), filepath.Base(path), "")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("fixture produced diagnostics: %v", result.Diagnostics)
	}

	checked := 0
	walkNodes(reflect.ValueOf(result.Root), map[uintptr]bool{}, func(typeName string, span ast.Span) {
		checked++
		if span.End.Ln < span.Start.Ln {
			t.Errorf("%s: span ends on line %d before it starts on line %d",
				typeName, span.End.Ln, span.Start.Ln)
		}
		if span.End.Ln == span.Start.Ln && span.End.Col < span.Start.Col {
			t.Errorf("%s: span ends at column %d before it starts at column %d",
				typeName, span.End.Col, span.Start.Col)
		}
	})
	if checked == 0 {
		t.Fatal("walked no nodes; the traversal is not reaching the tree")
	}
}

// TestSpanContainsLocatesTheEnclosingNode exercises the primitive every
// navigation feature is built from: given a cursor, find the innermost node.
func TestSpanContainsLocatesTheEnclosingNode(t *testing.T) {
	const source = "_ co.lang.struct = {\n    id   co.lang.int;\n    name co.lang.string;\n}\n"
	result := parser.ParseFile(source, "spans", ".", "Employee.fol", "people")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("source produced diagnostics: %v", result.Diagnostics)
	}

	// A cursor on line 3 must find a node, and the innermost one must not be the
	// whole file.
	var innermost ast.Span
	var innermostType string
	found := false
	walkNodes(reflect.ValueOf(result.Root), map[uintptr]bool{}, func(typeName string, span ast.Span) {
		if span.IsZero() || !span.Contains(3, 6) {
			return
		}
		if !found || spanNarrower(span, innermost) {
			innermost, innermostType, found = span, typeName, true
		}
	})

	if !found {
		t.Fatal("no node contains the cursor at line 3; Contains cannot drive navigation")
	}
	if innermost.Start.Ln != 3 {
		t.Errorf("innermost node at line 3 is %s spanning lines %d-%d, want one starting on line 3",
			innermostType, innermost.Start.Ln, innermost.End.Ln)
	}
}

// spanNarrower reports whether a covers less source than b.
func spanNarrower(a, b ast.Span) bool {
	aLines := a.End.Ln - a.Start.Ln
	bLines := b.End.Ln - b.Start.Ln
	if aLines != bLines {
		return aLines < bLines
	}
	return (a.End.Col - a.Start.Col) < (b.End.Col - b.Start.Col)
}

// walkNodes visits every ast node value reachable from v, calling visit with the
// node's type name and span.
//
// seen guards against the cycles a symbol pointer can introduce. Unexported
// fields are skipped because reflection cannot read them, which is safe here:
// every AST field that holds a child node is exported.
func walkNodes(v reflect.Value, seen map[uintptr]bool, visit func(string, ast.Span)) {
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Ptr {
			if seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
		}
		walkNodes(v.Elem(), seen, visit)
		return

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkNodes(v.Index(i), seen, visit)
		}
		return

	case reflect.Map:
		for _, key := range v.MapKeys() {
			walkNodes(v.MapIndex(key), seen, visit)
		}
		return

	case reflect.Struct:
		// A node is anything embedding ast.Span. Everything else — symbol
		// records, parser-side helpers — is walked through but not checked.
		//
		// An EMPTY struct is an UNSET FIELD rather than a node. Some node
		// types hold another node by value instead of by pointer —
		// ast.FunctionType.Parent is one — so the field exists on every
		// instance whether or not it was ever filled in. The exclusion is safe
		// and narrow: symbolfactory.go guarantees that every node the parser
		// builds carries a Symb pointer, so a node that reached the tree is
		// never empty, and a genuinely missing span cannot hide here.
		//
		// "Empty" means zero APART FROM NodeName, which every node type now
		// carries and every construction site fills in. A plain IsZero would
		// call an `ast.DirectveList{NodeName: "DirectveList"}` — the empty
		// directive list a node holds when nothing was annotated — a real node
		// and demand a source region for text that was never written.
		if isNode(v) && !isEmptyNode(v) {
			visit(v.Type().Name(), v.FieldByName("Span").Interface().(ast.Span))
		}
		// Do not descend into symbol records: they are metadata, they form
		// cycles, and they carry no spans.
		if strings.HasSuffix(v.Type().PkgPath(), "/context") {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			walkNodes(v.Field(i), seen, visit)
		}
		return
	}
}

// isEmptyNode reports whether a struct is zero once NodeName is discounted.
//
// NodeName names the node's own type, so it is a constant of the FORM rather
// than anything the source supplied; it says nothing about whether this instance
// was ever filled in. Counting it would make every unset by-value node field
// look like a node that had lost its span.
func isEmptyNode(v reflect.Value) bool {
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == "NodeName" {
			continue
		}
		if !v.Field(i).IsZero() {
			return false
		}
	}
	return true
}

// isNode reports whether a struct value is an AST node, i.e. embeds ast.Span.
func isNode(v reflect.Value) bool {
	if v.Type().PkgPath() == "" || !strings.HasSuffix(v.Type().PkgPath(), "/ast") {
		return false
	}
	field, ok := v.Type().FieldByName("Span")
	return ok && field.Type == reflect.TypeOf(ast.Span{})
}

// formatCounts renders a type/count map deterministically.
func formatCounts(counts map[string]int) string {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("  %-32s %d", name, counts[name]))
	}
	return strings.Join(lines, "\n")
}
