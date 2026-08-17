package parser_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

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
	expectations := refBlockExpectations(t)

	for _, path := range paths {
		path := path
		name := refBlockName(path)
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			result := parser.ParseFile(string(source), "refblocks",
				filepath.Dir(path), filepath.Base(path), "")
			if len(result.Diagnostics) == 0 {
				t.Fatalf("parsed without a diagnostic; this file must be rejected\n%s", source)
			}
			first := result.Diagnostics[0].AsString()

			// A fixture rejected by a FILENAME rule is not proving anything about
			// the rule it was written for: the parse stopped before reaching the
			// source text. That is a corpus defect wherever it appears, and it is
			// checkable without knowing which rule the fixture targets, so it is
			// applied to the extracted blocks too — their L<line> names move
			// whenever the reference is edited and cannot be pinned individually.
			for _, artifact := range filenameArtifacts {
				if strings.Contains(first, artifact) {
					t.Fatalf("rejected by a filename rule rather than by the rule under test;\n"+
						"give the fixture a folder holding the filename FoLang requires\n  %s\n\nsource:\n%s",
						firstLine(first), source)
				}
			}

			expected, pinned := expectations[name]
			if !pinned {
				// Only the flat hand-written half is pinned; see EXPECTATIONS.tsv.
				if !strings.Contains(name, "/") {
					t.Errorf("%s has no row in the invalid corpus manifest", name)
				}
				return
			}
			if !strings.Contains(first, expected) {
				t.Errorf("first diagnostic does not match the expected rule\n"+
					"  expected to contain: %s\n  got: %s\n\nsource:\n%s",
					expected, firstLine(first), source)
			}
		})
	}
}

// filenameArtifacts are the diagnostics that mean a fixture was stopped by the
// external filename grammar before its source text was read.
var filenameArtifacts = []string{
	"is not a valid FoLang filename identifier",
	"a primary declaration takes its name from the filename",
	"denotes the same declaration as",
}

// refBlockExpectations reads the hand-written half's manifest. The format
// matches examples/rejected/EXPECTATIONS.tsv: name, tab, expected substring.
func refBlockExpectations(t *testing.T) map[string]string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "refblocks", "invalid", "EXPECTATIONS.tsv")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	expectations := make(map[string]string)
	for number, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, expected, ok := strings.Cut(line, "\t")
		name, expected = strings.TrimSpace(name), strings.TrimSpace(expected)
		if !ok || name == "" || expected == "" {
			t.Fatalf("%s line %d is not a tab-separated fixture and expectation: %q", path, number+1, line)
		}
		expectations[name] = expected
	}
	return expectations
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
