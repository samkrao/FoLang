package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/parser"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Concurrency contract.
//
// A language server parses the files the user has open, and it parses them while
// answering requests about others. That means several ParseFile calls in flight
// at once, on distinct sources. These tests establish that this is safe.
//
// They are written to CONTEND rather than merely to run in parallel: every
// goroutine starts from a released barrier, so the calls overlap in the window
// where a shared write would actually collide. Run them with -race; without it
// they still catch corruption and panics, but a benign-looking race will pass.

// concurrency is the number of goroutines each test runs. It is deliberately
// larger than a plausible core count so that goroutines are preempted mid-parse.
const concurrency = 32

// TestConcurrentParsesOfDistinctFiles is the case a server actually hits.
func TestConcurrentParsesOfDistinctFiles(t *testing.T) {
	sources := make([]string, concurrency)
	names := make([]string, concurrency)
	for i := range sources {
		// Distinct content and distinct filenames, so nothing is shared except
		// the frontend itself.
		sources[i] = fmt.Sprintf(`_ co.lang.unit = {
    compute%d(value co.lang.int)->(co.lang.int) = {
        total := value + %d;
        this.return total;
    }
}
`, i, i)
		names[i] = fmt.Sprintf("shape_%d.unit.fol", i)
	}

	results := make([]parser.Result, concurrency)
	runConcurrently(t, func(i int) {
		results[i] = parser.ParseFile(sources[i], "concurrent", ".", names[i], "shapes")
	})

	for i, result := range results {
		if len(result.Diagnostics) != 0 {
			t.Errorf("file %d produced diagnostics: %v", i, result.Diagnostics)
		}
		want := fmt.Sprintf("compute%d", i)
		if !declaredFunctionNames(result.Root)[want] {
			t.Errorf("file %d lost its declaration %q; concurrent parses interfered", i, want)
		}
	}
}

// TestConcurrentParsesOfTheSameFile covers the other real pattern: a server
// re-parsing one buffer while an earlier parse of it is still running.
func TestConcurrentParsesOfTheSameFile(t *testing.T) {
	const source = `_ co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}
`
	results := make([]parser.Result, concurrency)
	runConcurrently(t, func(i int) {
		results[i] = parser.ParseFile(source, "concurrent", ".", "Employee.fol", "people")
	})

	// Every parse of identical input must produce identical spans. A shared
	// counter or buffer would show up here as a divergence even without -race.
	reference := collectSpans(results[0].Root)
	for i, result := range results {
		if len(result.Diagnostics) != 0 {
			t.Errorf("parse %d produced diagnostics: %v", i, result.Diagnostics)
		}
		if !equalSpans(reference, collectSpans(result.Root)) {
			t.Errorf("parse %d produced different spans from parse 0", i)
		}
	}
}

// TestConcurrentParsesOfMalformedFiles contends on the diagnostic and recovery
// paths, which allocate and unwind more than the clean path does.
func TestConcurrentParsesOfMalformedFiles(t *testing.T) {
	sources := make([]string, concurrency)
	for i := range sources {
		var b strings.Builder
		b.WriteString("_ co.lang.unit = {\n")
		for j := 0; j < 20; j++ {
			b.WriteString("    &&& broken &&&\n")
		}
		fmt.Fprintf(&b, "    ok%d()->(co.lang.int) = { this.return %d; }\n}\n", i, i)
		sources[i] = b.String()
	}

	results := make([]parser.Result, concurrency)
	runConcurrently(t, func(i int) {
		results[i] = parser.ParseFile(sources[i], "concurrent", ".",
			fmt.Sprintf("broken_%d.unit.fol", i), "shapes")
	})

	for i, result := range results {
		if len(result.Diagnostics) == 0 {
			t.Errorf("malformed file %d produced no diagnostics", i)
		}
		want := fmt.Sprintf("ok%d", i)
		if !declaredFunctionNames(result.Root)[want] {
			t.Errorf("file %d lost %q; recovery is not concurrency-safe", i, want)
		}
	}
}

// TestConcurrentTokenizationAndParsing mixes the two public entry points, since
// a server tokenizes for semantic highlighting and parses for everything else,
// often at the same time.
func TestConcurrentTokenizationAndParsing(t *testing.T) {
	const source = `_ co.lang.unit = {
    run(a co.lang.int, b co.lang.int)->(co.lang.int) = {
        this.return a + b;
    }
}
`
	tokenCounts := make([]int, concurrency)
	runConcurrently(t, func(i int) {
		if i%2 == 0 {
			// The highlighting path: tokens only.
			toks, diags := scanlex.TokenizeCollecting(source, "mixed.unit.fol", nil)
			if len(diags) != 0 {
				t.Errorf("goroutine %d: clean source produced lexical diagnostics: %v", i, diags)
			}
			tokenCounts[i] = len(toks)
			return
		}
		// The everything-else path: a full parse.
		result := parser.ParseFile(source, "concurrent", ".", "mixed.unit.fol", "shapes")
		if len(result.Diagnostics) != 0 {
			t.Errorf("goroutine %d: clean source produced diagnostics: %v", i, result.Diagnostics)
		}
		tokenCounts[i] = len(result.Tokens)
	})

	// Both paths tokenize the same source, so every goroutine must have seen the
	// same stream length. A shared lexer buffer would show up as a divergence.
	for i, n := range tokenCounts {
		if n != tokenCounts[0] {
			t.Fatalf("goroutine %d saw %d tokens but goroutine 0 saw %d; tokenization is not concurrency-safe",
				i, n, tokenCounts[0])
		}
	}
}

// TestConcurrentParsesShareOneOperatorCatalog covers the sharing pattern a
// server actually uses, and the only object it is expected to hold across
// parses.
//
// A custom operator cannot be recognised from one file alone, so the catalog is
// loaded once at project open and passed to every subsequent parse. That makes
// it the one piece of state genuinely shared between concurrent parses — every
// other input is per-call — so it is the place a mutation would do damage.
func TestConcurrentParsesShareOneOperatorCatalog(t *testing.T) {
	root := t.TempDir()
	writeOperatorProject(t, root)

	operators, findings := parser.LoadOperators(root)
	if len(findings) != 0 {
		t.Fatalf("loading the operator catalog reported findings: %v", findings)
	}

	const source = `_ co.lang.unit = {
    @co.dap.operator(symbol="<+>", mode=overload)
    merge(left Vector, right Vector)->(Vector) = { this.return left; }
}
`
	results := make([]parser.Result, concurrency)
	runConcurrently(t, func(i int) {
		results[i] = parser.ParseFileWithOperators(
			source, "concurrent", root, "Vector.comp.unit.fol", "vectors", operators)
	})

	// Every parse read the same catalog, so every parse must have classified the
	// custom symbol identically. A catalog mutated under contention would show up
	// as one goroutine failing to recognise "<+>" and reporting it as unknown.
	for i, result := range results {
		if len(result.Diagnostics) != 0 {
			t.Errorf("parse %d produced diagnostics with a shared catalog: %v", i, result.Diagnostics)
		}
		if !declaredFunctionNames(result.Root)["merge"] {
			t.Errorf("parse %d lost its operator implementation", i)
		}
	}
}

// writeOperatorProject lays out the minimum project a custom operator needs: the fixed
// bootstrap surface at its one standardized location.
//
// The location is not configurable. `components/operators/component.fol` is the only place a
// project-local operator symbol may be declared, and the surface carries no library-kind
// annotation because the `operators/` slot already establishes what it is.
func writeOperatorProject(t *testing.T, root string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, "fol-conf.yaml"),
		[]byte("fol-lang:\n  name: fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := filepath.Join(root, "components", "operators")
	if err := os.MkdirAll(area, 0o755); err != nil {
		t.Fatal(err)
	}
	const catalog = `_ co.lang.component = {
    <+> co.lang.operator = {
        fixity: co.operator.fixity.infix,
        precedence: 60,
        associativity: co.operator.associativity.left,
        arity: co.operator.arity.binary
    };
}
`
	if err := os.WriteFile(filepath.Join(area, "component.fol"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestConcurrentSpanWalksDoNotMutate confirms the tree is safe to READ from
// several goroutines, which is what a server does when it answers hover,
// document-symbol and folding requests against one cached parse.
func TestConcurrentSpanWalksDoNotMutate(t *testing.T) {
	const source = `_ co.lang.unit = {
    outer(value co.lang.int)->(co.lang.int) = {
        inner := value * 2;
        this.return inner;
    }
}
`
	result := parser.ParseFile(source, "concurrent", ".", "shapes.unit.fol", "shapes")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("source produced diagnostics: %v", result.Diagnostics)
	}

	counts := make([]int, concurrency)
	runConcurrently(t, func(i int) {
		walkNodes(reflect.ValueOf(result.Root), map[uintptr]bool{}, func(_ string, span ast.Span) {
			if !span.IsZero() {
				counts[i]++
			}
		})
	})

	for i, n := range counts {
		if n != counts[0] {
			t.Fatalf("walk %d saw %d nodes but walk 0 saw %d; the tree is being mutated by readers",
				i, n, counts[0])
		}
	}
	if counts[0] == 0 {
		t.Fatal("walked no nodes")
	}
}

// runConcurrently releases all goroutines at once so their work overlaps.
//
// Starting them with a barrier matters: goroutines launched in a plain loop
// often run to completion one at a time on a lightly loaded machine, which would
// make a shared-state bug invisible.
func runConcurrently(t *testing.T, body func(i int)) {
	t.Helper()

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			body(i)
		}(i)
	}
	close(start)
	wg.Wait()
}
