package parser_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// TestEBNFConformance runs complete parsing over the canonical accepted and
// rejected examples from docs/grammar/folang.ebnf. Adding a .fol file to either
// fixture directory automatically adds another conformance case.
func TestEBNFConformance(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		for _, path := range conformanceFixtures(t, "accepted") {
			path := path
			t.Run(fixtureName(path), func(t *testing.T) {
				source := readFixture(t, path)

				var root ast.Stmt
				mustNotPanic(t, func() {
					root, _, _, _ = parser.Parse(
						source,
						"conformance",
						filepath.Dir(path),
						filepath.Base(path),
						"",
						"program",
						"program",
						true,
					)
				})

				if _, dummy := root.(ast.DummyStmt); dummy {
					t.Fatalf("accepted fixture returned a dummy AST")
				}
			})
		}
	})

	t.Run("rejected", func(t *testing.T) {
		for _, path := range conformanceFixtures(t, "rejected") {
			path := path
			t.Run(fixtureName(path), func(t *testing.T) {
				source := readFixture(t, path)

				mustPanic(t, func() {
					parser.Parse(
						source,
						"conformance",
						filepath.Dir(path),
						filepath.Base(path),
						"",
						"program",
						"program",
						true,
					)
				})
			})
		}
	})
}

// TestPointerDepthPreserved verifies more than syntactic acceptance. The EBNF
// production `pointer-stars = "*", { "*" }` permits every positive pointer
// degree, and the exact degree must survive lowering into PointerSymbol.Count.
func TestPointerDepthPreserved(t *testing.T) {
	path := filepath.Join("examples", "accepted", "pointer-depths.fol")
	source := readFixture(t, path)

	var root ast.Stmt
	mustNotPanic(t, func() {
		root, _, _, _ = parser.Parse(
			source,
			"conformance",
			filepath.Dir(path),
			filepath.Base(path),
			"",
			"program",
			"program",
			true,
		)
	})

	application, ok := root.(ast.Application)
	if !ok {
		t.Fatalf("pointer fixture returned %T, want ast.Application", root)
	}
	if len(application.Body) != 5 {
		t.Fatalf("pointer fixture produced %d statements, want 5", len(application.Body))
	}

	for index, statement := range application.Body {
		pointer, ok := statement.(ast.PointerVariableDeclStmt)
		if !ok {
			t.Fatalf("statement %d is %T, want ast.PointerVariableDeclStmt", index+1, statement)
		}
		want := index + 1
		if pointer.Symb.Count != want {
			t.Errorf("statement %d preserved pointer degree %d, want %d", index+1, pointer.Symb.Count, want)
		}
	}
}

func conformanceFixtures(t *testing.T, outcome string) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("examples", outcome, "*.fol"))
	if err != nil {
		t.Fatalf("discover %s fixtures: %v", outcome, err)
	}
	if len(paths) == 0 {
		t.Fatalf("no %s fixtures found", outcome)
	}
	sort.Strings(paths)
	return paths
}

func readFixture(t *testing.T, path string) string {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(source)
}

func fixtureName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func mustNotPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("accepted fixture panicked: %v", recovered)
		}
	}()
	fn()
}

func mustPanic(t *testing.T, fn func()) {
	t.Helper()

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		fn()
	}()

	if recovered == nil {
		t.Fatalf("rejected fixture parsed without a diagnostic")
	}
	if recovered != "Error" {
		t.Fatalf("rejected fixture caused an unexpected internal panic: %v", recovered)
	}
}
