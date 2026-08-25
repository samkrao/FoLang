package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var retiredControlSyntax = []*regexp.Regexp{
	regexp.MustCompile(`\.do\s*\(`),
	regexp.MustCompile(`\.otherwise(?:\s*\([^)]*\))?\s*\.loop\s*\(`),
	regexp.MustCompile(`\.each\s*\([^)]*\)\s*\.loop\s*\(`),
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
