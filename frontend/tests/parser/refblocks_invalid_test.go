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
	dir := filepath.Join("..", "..", "testdata", "refblocks", "invalid")
	paths, err := filepath.Glob(filepath.Join(dir, "*.fol"))
	if err != nil {
		t.Fatalf("discover invalid corpus: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("invalid corpus is empty")
	}
	sort.Strings(paths)

	for _, path := range paths {
		path := path
		t.Run(strings.TrimSuffix(filepath.Base(path), ".fol"), func(t *testing.T) {
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
