package parser_test

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// TestRefBlocksInvalidAreRejected asserts the contract of
// testdata/refblocks/invalid/: every file in it must be rejected.
//
// The corpus holds the reference's error-marked examples plus hand-written
// cases for rules no reference example exercises. Each hand-written file
// carries its statement terminator so that a rejection comes from the rule
// under test rather than incidentally from a missing ";".
func TestRefBlocksInvalidAreRejected(t *testing.T) {
	paths := refBlockCorpus(t, "invalid")
	for _, path := range paths {
		path := path
		t.Run(refBlockName(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if rejected, diag := rejectsWithDiagnostic(string(source), path); !rejected {
				t.Errorf("parsed without a diagnostic; this file must be rejected\n%s", source)
			} else if testing.Verbose() {
				t.Logf("%s", diag)
			}
		})
	}
}

// TestRefBlocksParsingAreAccepted asserts the other half of the contract:
// every file in testdata/refblocks/parsing/ must parse.
//
// This half had no test, which is why it rotted. The corpus is extracted from
// the reference and named after the line each block sits on, so editing the
// reference renumbers everything below the edit; with nothing checking the
// claim, a third of the "must parse" corpus had stopped parsing before anyone
// noticed. Regenerate with `go run ./cmd/refblocks -write`.
func TestRefBlocksParsingAreAccepted(t *testing.T) {
	for _, path := range refBlockCorpus(t, "parsing") {
		path := path
		t.Run(refBlockName(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			// The entry keeps the filename the reference gave it, and FoLang
			// classifies a source file by its name, so it is parsed under that
			// name rather than under a synthesized one.
			result := parser.ParseFile(string(source), "refblocks",
				filepath.Dir(path), filepath.Base(path), "")
			if len(result.Diagnostics) != 0 {
				t.Errorf("this block must parse but reported %d diagnostic(s):\n%s\n%s",
					len(result.Diagnostics), source, result.Diagnostics[0].AsString())
			}
		})
	}
}

// refBlockCorpus lists the source files of one corpus.
//
// Two layouts coexist. An extracted block lives in its own folder so that it can
// keep the filename the reference gave it — `package.fol`, `appl.fol` and `library.fol`
// are reserved exact spellings that cannot be prefixed with a line number and
// remain themselves. Hand-written fixtures, which no reference block covers,
// sit flat beside those folders.
func refBlockCorpus(t *testing.T, corpus string) []string {
	t.Helper()

	dir := filepath.Join("..", "..", "testdata", "refblocks", corpus)
	flat, err := filepath.Glob(filepath.Join(dir, "*.fol"))
	if err != nil {
		t.Fatalf("discover %s corpus: %v", corpus, err)
	}
	nested, err := filepath.Glob(filepath.Join(dir, "L*", "*.fol"))
	if err != nil {
		t.Fatalf("discover %s corpus: %v", corpus, err)
	}

	paths := append(flat, nested...)
	if len(paths) == 0 {
		t.Fatalf("%s corpus is empty", corpus)
	}
	sort.Strings(paths)
	return paths
}

// refBlockName labels a corpus entry by its block folder and filename, so a
// failure names something findable in the reference.
func refBlockName(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".fol")
	if parent := filepath.Base(filepath.Dir(path)); strings.HasPrefix(parent, "L") {
		return parent + "/" + base
	}
	return base
}

// rejectsWithDiagnostic parses source and reports whether it was rejected,
// along with the first diagnostic. Diagnostics are printed to stdout before the
// bailout panic, so stdout is captured for the duration of the parse.
func rejectsWithDiagnostic(source, path string) (rejected bool, diag string) {
	saved := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return false, ""
	}
	os.Stdout = w

	captured := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		captured <- string(out)
	}()

	func() {
		defer func() {
			if recover() != nil {
				rejected = true
			}
		}()
		root, _, _, _ := parser.Parse(source, "refblocks", filepath.Dir(path),
			filepath.Base(path), "", "program", "program", true)
		if _, dummy := root.(ast.DummyStmt); dummy {
			rejected = true
		}
	}()

	w.Close()
	os.Stdout = saved

	for _, line := range strings.Split(<-captured, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Invalid Syntax:") || strings.HasPrefix(line, "Restricted Import:") ||
			strings.HasPrefix(line, "Unsupported") {
			return rejected, line
		}
	}
	return rejected, ""
}
