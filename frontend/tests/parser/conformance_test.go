package parser_test

import (
	"os"
	"path/filepath"
	"slices"
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
		fixtures := rejectedFixtures(t)
		expectations := rejectedExpectations(t)

		// The manifest and the corpus are checked against each other in both
		// directions. A fixture with no row would otherwise be asserted only to
		// fail somehow, which is the weakness this manifest exists to remove; a
		// row with no fixture is an expectation nothing is proving.
		for _, fixture := range fixtures {
			if _, ok := expectations[fixture.name]; !ok {
				t.Errorf("%s has no row in %s; every rejected fixture must state the diagnostic it expects",
					fixture.name, rejectedManifest)
			}
		}
		for name := range expectations {
			if !slices.ContainsFunc(fixtures, func(f rejectedFixture) bool { return f.name == name }) {
				t.Errorf("%s names %q, which is not a fixture in the rejected corpus", rejectedManifest, name)
			}
		}

		for _, fixture := range fixtures {
			fixture := fixture
			expected, ok := expectations[fixture.name]
			if !ok {
				continue
			}
			t.Run(fixture.name, func(t *testing.T) {
				source := readFixture(t, fixture.path)

				// ParseFile rather than Parse: the batch entry point ends the
				// process at the first diagnostic, which leaves a test able to
				// observe only THAT it failed. The collecting entry point hands
				// back the findings themselves, which is what makes the
				// assertion below possible.
				result := parser.ParseFile(source, "conformance",
					filepath.Dir(fixture.path), filepath.Base(fixture.path), "")

				if len(result.Diagnostics) == 0 {
					t.Fatalf("parsed without a diagnostic; this fixture must be rejected\n%s", source)
				}

				// The FIRST diagnostic is what is asserted, not any diagnostic.
				// A fixture that dies earlier than intended still reports
				// something, and matching anywhere in the list would let that
				// pass — which is exactly how a fifth of this corpus came to be
				// exercising the filename rules instead of the rule each case is
				// named for.
				first := result.Diagnostics[0].AsString()
				if !strings.Contains(first, expected) {
					t.Errorf("first diagnostic does not match the expected rule\n"+
						"  expected to contain: %s\n  got: %s\n\nsource:\n%s",
						expected, firstLine(first), source)
				}
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

// rejectedManifest names the file that pairs each rejected fixture with the
// diagnostic it must produce.
const rejectedManifest = "examples/rejected/EXPECTATIONS.tsv"

// rejectedFixture is one entry of the rejected corpus.
type rejectedFixture struct {
	// name is the case name used in the manifest and in the subtest: the
	// containing folder for a nested fixture, the filename for a flat one.
	name string
	path string
}

// rejectedFixtures lists the rejected corpus, which has two layouts.
//
// A FLAT `<case>.fol` is parsed under its own hyphenated name. That name is not
// a filename-identifier, so it is never read as a file-backed primary
// declaration, and the file classifies as an application entry file by its
// syntax. That is the right home for a fixture whose rule is about a statement
// or an expression.
//
// A fixture whose rule needs a different classification lives in a folder,
// `<case>/<Name>.fol`, holding one file under the name FoLang requires — the
// same device testdata/refblocks/ uses. A rule about a struct body needs
// `Employee.fol`, one about a companion unit needs `Employee.comp.unit.fol`,
// and one about a block needs a unit file, because an entry file admits no
// function to put a block in. Parsing those under a synthesized hyphenated name
// stops the parse at the filename rules and never reaches the rule under test.
//
// One file per folder also keeps validateFilenameIsUnique out of the way, since
// no two fixtures are then siblings.
func rejectedFixtures(t *testing.T) []rejectedFixture {
	t.Helper()

	flat, err := filepath.Glob(filepath.Join("examples", "rejected", "*.fol"))
	if err != nil {
		t.Fatalf("discover rejected fixtures: %v", err)
	}
	nested, err := filepath.Glob(filepath.Join("examples", "rejected", "*", "*.fol"))
	if err != nil {
		t.Fatalf("discover rejected fixtures: %v", err)
	}
	if len(flat)+len(nested) == 0 {
		t.Fatal("no rejected fixtures found")
	}

	fixtures := make([]rejectedFixture, 0, len(flat)+len(nested))
	for _, path := range flat {
		fixtures = append(fixtures, rejectedFixture{name: fixtureName(path), path: path})
	}
	for _, path := range nested {
		fixtures = append(fixtures, rejectedFixture{name: filepath.Base(filepath.Dir(path)), path: path})
	}
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].name < fixtures[j].name })

	seen := make(map[string]string, len(fixtures))
	for _, fixture := range fixtures {
		if previous, clash := seen[fixture.name]; clash {
			t.Errorf("rejected case %q is claimed by both %s and %s", fixture.name, previous, fixture.path)
		}
		seen[fixture.name] = fixture.path
	}
	return fixtures
}

// rejectedExpectations reads the manifest: one case name and one diagnostic
// substring per line, tab separated, with "#" comments and blank lines ignored.
func rejectedExpectations(t *testing.T) map[string]string {
	t.Helper()

	content, err := os.ReadFile(filepath.FromSlash(rejectedManifest))
	if err != nil {
		t.Fatalf("read %s: %v", rejectedManifest, err)
	}

	expectations := make(map[string]string)
	for number, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, expected, ok := strings.Cut(line, "\t")
		name, expected = strings.TrimSpace(name), strings.TrimSpace(expected)
		if !ok || name == "" || expected == "" {
			t.Fatalf("%s line %d is not a tab-separated case and expectation: %q",
				rejectedManifest, number+1, line)
		}
		if _, duplicate := expectations[name]; duplicate {
			t.Errorf("%s lists %q twice", rejectedManifest, name)
		}
		expectations[name] = expected
	}
	return expectations
}

// firstLine trims a rendered diagnostic to its message, dropping the file
// location and the source excerpt beneath it.
func firstLine(diagnostic string) string {
	line, _, _ := strings.Cut(diagnostic, "\n")
	return line
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
