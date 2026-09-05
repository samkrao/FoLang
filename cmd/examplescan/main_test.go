package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/parser"
)

var retiredControlSyntax = []*regexp.Regexp{
	regexp.MustCompile(`\.do\s*\(`),
	regexp.MustCompile(`\.otherwise(?:\s*\([^)]*\))?\s*\.loop\s*\(`),
	regexp.MustCompile(`\.each\s*\([^)]*\)\s*\.loop\s*\(`),
}

// TestExampleParseFailuresAreClassified parses every user-facing example.
// Files that still need migration are explicit debt in
// KNOWN_PARSE_EXCLUSIONS.txt; a new failure or a recovered file makes this test
// fail until the classification is updated.
func TestExampleParseFailuresAreClassified(t *testing.T) {
	manifest, err := os.ReadFile("KNOWN_PARSE_EXCLUSIONS.txt")
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, line := range strings.Split(string(manifest), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			known[line] = true
		}
	}

	root := filepath.Join("..", "..", "examples")
	actual := map[string]bool{}
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".fol" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dir, base := exampleParseContext(path, string(source))
		result := parser.ParseFile(string(source), "examples", dir, base, "")
		if len(result.Diagnostics) != 0 {
			rel, err := filepath.Rel(filepath.Join("..", ".."), path)
			if err != nil {
				return err
			}
			actual[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for path := range actual {
		if !known[path] {
			t.Errorf("new unclassified example parse failure: %s", path)
		}
	}
	for path := range known {
		if !actual[path] {
			t.Errorf("known example exclusion no longer fails; remove or reclassify it: %s", path)
		}
	}
}

// The examples directory is user-facing documentation and is separate from
// the parser fixture corpus. Keep its control-flow spelling on the current
// alpha profile even while examples with unrelated semantic dependencies may
// not yet be independently compilable.
func TestExamplesDoNotUseRetiredControlSyntax(t *testing.T) {
	root := filepath.Join("..", "..", "examples")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".fol" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, pattern := range retiredControlSyntax {
			if pattern.Match(source) {
				t.Errorf("%s uses retired control syntax matched by %s", path, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
