package parser_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// The contract an embedding consumer depends on.
//
// Each test here asserts a property a language server needs and the batch
// compiler does not. They are behavioural: the batch path still exits the
// process on a diagnostic, and these prove the non-fatal path does not.

// TestParseFileSurvivesSyntaxErrors is the headline property. Parse terminates
// the process at the first diagnostic; ParseFile must return.
func TestParseFileSurvivesSyntaxErrors(t *testing.T) {
	// A malformed member between two well-formed ones.
	const source = `_ co.lang.struct = {
    id   co.lang.int;
    &&& broken &&&
    name co.lang.string;
}
`
	result := parser.ParseFile(source, "lsp", ".", "Employee.fol", "people")

	if len(result.Diagnostics) == 0 {
		t.Fatal("malformed source produced no diagnostics")
	}
	if _, dummy := result.Root.(ast.DummyStmt); dummy {
		t.Fatal("recovery produced no tree; the partial parse was discarded")
	}
	if len(result.Tokens) == 0 {
		t.Fatal("no token stream returned")
	}
}

// TestParseFileRecoversTheWellFormedMembers proves recovery is USEFUL, not
// merely that it returns: the members either side of a broken one must survive
// into the tree, since that is what keeps an editor's outline and navigation
// working while the user is mid-edit.
func TestParseFileRecoversTheWellFormedMembers(t *testing.T) {
	const source = `_ co.lang.unit = {
    first()->(co.lang.int) = { this.return 1; }
    &&& broken &&&
    second()->(co.lang.int) = { this.return 2; }
}
`
	result := parser.ParseFile(source, "lsp", ".", "shapes.unit.fol", "shapes")
	if len(result.Diagnostics) == 0 {
		t.Fatal("malformed source produced no diagnostics")
	}

	names := declaredFunctionNames(result.Root)
	for _, want := range []string{"first", "second"} {
		if !names[want] {
			t.Errorf("function %q did not survive recovery; recovered: %v", want, keysOf(names))
		}
	}
}

// TestParseFileSurvivesAMalformedDeclarationHead covers the case that used to
// discard the entire tree: the head is what is incomplete while the user types.
func TestParseFileSurvivesAMalformedDeclarationHead(t *testing.T) {
	// No kind token: the head is half-typed.
	const source = `_ = {
    id co.lang.int;
}
`
	result := parser.ParseFile(source, "lsp", ".", "Employee.fol", "people")

	if len(result.Diagnostics) == 0 {
		t.Fatal("a headless declaration produced no diagnostic")
	}
	// The tree may be partial, but the call must return rather than exit, and it
	// must not panic. Reaching this line is the assertion.
}

// TestParseFileSurvivesLexicalErrors covers the scanner half: one bad byte used
// to end the process before the parser ran at all.
func TestParseFileSurvivesLexicalErrors(t *testing.T) {
	// An identifier ending in an underscore is a lexical error, and the file
	// continues past it.
	const source = `_ co.lang.unit = {
    broken_()->(co.lang.int) = { this.return 1; }
    fine()->(co.lang.int) = { this.return 2; }
}
`
	result := parser.ParseFile(source, "lsp", ".", "shapes.unit.fol", "shapes")

	if len(result.Diagnostics) == 0 {
		t.Fatal("a lexical error produced no diagnostic")
	}
	if len(result.Tokens) == 0 {
		t.Fatal("a lexical error discarded the token stream")
	}
	if !declaredFunctionNames(result.Root)["fine"] {
		t.Error("the declaration after a lexical error did not survive")
	}
}

// TestDiagnosticsCarryUsableRanges checks the ranges an editor renders.
func TestDiagnosticsCarryUsableRanges(t *testing.T) {
	const source = "_ co.lang.struct = {\n    id co.lang.int\n}\n"
	result := parser.ParseFile(source, "lsp", ".", "Employee.fol", "people")

	if len(result.Diagnostics) == 0 {
		t.Fatal("a missing terminator produced no diagnostic")
	}
	text := result.Diagnostics[0].AsString()
	if !strings.Contains(text, "Employee.fol") {
		t.Errorf("diagnostic does not name its file: %s", text)
	}
	if strings.Contains(text, "line 0") {
		t.Errorf("diagnostic has no usable line: %s", text)
	}
}

// TestDiagnosticsAreCapped proves MaxParseErrors is enforced and that reaching
// it is reported rather than silent.
func TestDiagnosticsAreCapped(t *testing.T) {
	var b strings.Builder
	b.WriteString("_ co.lang.unit = {\n")
	for i := 0; i < 200; i++ {
		b.WriteString("    &&& broken &&&\n")
	}
	b.WriteString("}\n")

	result := parser.ParseFile(b.String(), "lsp", ".", "shapes.unit.fol", "shapes")

	if len(result.Diagnostics) > 60 {
		t.Errorf("diagnostics = %d, want the list capped near MaxParseErrors", len(result.Diagnostics))
	}
	if !result.Truncated {
		t.Error("the list was capped but Truncated is false, so a caller cannot tell")
	}
}

// TestParseFileWritesNothingToStdout is the property that makes the frontend
// usable over a stdio transport at all. Anything on stdout corrupts a
// JSON-RPC stream and disconnects the client.
func TestParseFileWritesNothingToStdout(t *testing.T) {
	saved := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write

	captured := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := read.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		captured <- b.String()
	}()

	// Parse both a clean file and a malformed one: the diagnostic path is the
	// one that used to print.
	parser.ParseFile("_ co.lang.struct = {\n    id co.lang.int;\n}\n", "lsp", ".", "Employee.fol", "people")
	parser.ParseFile("_ co.lang.struct = {\n    &&& broken\n}\n", "lsp", ".", "Employee.fol", "people")

	write.Close()
	os.Stdout = saved
	if output := <-captured; output != "" {
		t.Errorf("parsing wrote %d bytes to stdout, which corrupts an LSP stdio stream:\n%s",
			len(output), output)
	}
}

// TestParseFileNeverReachesTheFatalPath proves ParseFile does not depend on the
// process-global foerrors.GenPanic, which is the one piece of global state an
// embedding consumer could be bitten by.
//
// The whole test package sets GenPanic to true so that the batch path panics
// instead of exiting. This test restores the PRODUCTION default for the duration
// of one malformed parse. If ParseFile reached HandleErrors under that setting
// it would call os.Exit(1) and kill the test binary outright — an unmistakable
// failure. Returning normally is the assertion.
//
// It does not call t.Parallel: it mutates a global, and the guarantee is about
// reachability rather than about concurrent access to GenPanic itself, which no
// consumer should ever write.
func TestParseFileNeverReachesTheFatalPath(t *testing.T) {
	restore := foerrors.GenPanic
	foerrors.GenPanic = false
	defer func() { foerrors.GenPanic = restore }()

	result := parser.ParseFile("_ co.lang.struct = {\n    &&& broken &&&\n}\n",
		"lsp", ".", "Employee.fol", "people")

	if len(result.Diagnostics) == 0 {
		t.Fatal("malformed source produced no diagnostics")
	}
	// Reaching this line means the fatal path was never entered.
}

// TestParseIsDeterministic proves two parses of identical source agree, which
// is what lets a consumer diff parses or cache anything keyed on symbol identity.
func TestParseIsDeterministic(t *testing.T) {
	const source = "_ co.lang.struct = {\n    id co.lang.int;\n}\n"

	first := parser.ParseFile(source, "lsp", ".", "Employee.fol", "people")
	second := parser.ParseFile(source, "lsp", ".", "Employee.fol", "people")

	if len(first.Tokens) != len(second.Tokens) {
		t.Fatalf("token counts differ: %d then %d", len(first.Tokens), len(second.Tokens))
	}
	for i := range first.Tokens {
		if first.Tokens[i].Value != second.Tokens[i].Value {
			t.Fatalf("token %d differs: %q then %q", i, first.Tokens[i].Value, second.Tokens[i].Value)
		}
	}
	// Node spans must agree exactly; they are what an editor maps to ranges.
	if firstSpans, secondSpans := collectSpans(first.Root), collectSpans(second.Root); !equalSpans(firstSpans, secondSpans) {
		t.Error("two parses of identical source produced different node spans")
	}
}

// declaredFunctionNames collects the logical names of every function
// declaration in a tree.
//
// It walks by reflection rather than by a type switch over container kinds.
// A type switch has to enumerate both every container that can hold a function
// AND every wrapper that can hold one: an operator implementation is an
// ast.OperatorStmt embedding a FunctionDeclarationStmt, and a matcher, macro,
// lambda and generic function each have their own wrapper. Missing one makes a
// test report a lost declaration that is really present, which is exactly the
// false failure this replaced.
func declaredFunctionNames(root ast.Stmt) map[string]bool {
	names := map[string]bool{}
	collectFunctionNames(reflect.ValueOf(root), map[uintptr]bool{}, names)
	return names
}

func collectFunctionNames(v reflect.Value, seen map[uintptr]bool, names map[string]bool) {
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
		collectFunctionNames(v.Elem(), seen, names)

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectFunctionNames(v.Index(i), seen, names)
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			collectFunctionNames(v.MapIndex(key), seen, names)
		}

	case reflect.Struct:
		if fn, ok := v.Interface().(ast.FunctionDeclarationStmt); ok && fn.Name != "" {
			names[strings.TrimSuffix(fn.Name, "_fo")] = true
		}
		// Symbol records are metadata, form cycles, and declare nothing.
		if strings.HasSuffix(v.Type().PkgPath(), "/context") {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			collectFunctionNames(v.Field(i), seen, names)
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func collectSpans(root ast.Stmt) []ast.Span {
	var spans []ast.Span
	walkNodes(reflect.ValueOf(root), map[uintptr]bool{}, func(_ string, span ast.Span) {
		spans = append(spans, span)
	})
	return spans
}

func equalSpans(a, b []ast.Span) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Start.Ln != b[i].Start.Ln || a[i].Start.Col != b[i].Start.Col ||
			a[i].End.Ln != b[i].End.Ln || a[i].End.Col != b[i].End.Col {
			return false
		}
	}
	return true
}
