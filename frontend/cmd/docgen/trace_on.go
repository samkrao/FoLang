//go:build partrace

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// collectTrace parses every .fol file under corpus and returns the spans the
// instrumented parser recorded.
//
// A file that fails to parse still contributes: the parse functions that
// succeeded before the failure recorded their spans, and the instrumentation
// discards only the spans that were open when the bailout unwound. The corpus is
// walked in sorted order so a rerun produces byte-identical output.
func collectTrace(corpus string, limit int) (map[string][]string, bool, error) {
	info, err := os.Stat(corpus)
	if err != nil {
		return nil, false, fmt.Errorf("corpus %q: %w", corpus, err)
	}
	if !info.IsDir() {
		return nil, false, fmt.Errorf("corpus %q is not a directory", corpus)
	}

	var files []string
	if err := filepath.Walk(corpus, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && strings.HasSuffix(path, ".fol") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, false, err
	}
	if len(files) == 0 {
		return nil, false, fmt.Errorf("corpus %q contains no .fol files", corpus)
	}
	sort.Strings(files)

	// A parse failure must not terminate docgen: HandleErrors exits the process
	// unless it is told to panic instead, and a rejected corpus file is ordinary.
	restore := foerrors.GenPanic
	foerrors.GenPanic = true
	defer func() { foerrors.GenPanic = restore }()

	parser.ResetTrace()
	parsed, rejected := 0, 0
	for _, path := range files {
		if parseCorpusFile(path) {
			parsed++
		} else {
			rejected++
		}
	}
	fmt.Printf("trace corpus      files=%d parsed=%d rejected=%d\n", len(files), parsed, rejected)

	return parser.TraceSnippets(limit), true, nil
}

// parseCorpusFile parses one file, absorbing a rejection.
func parseCorpusFile(path string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	source, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	parser.Parse(string(source), "docgen", filepath.Dir(path), filepath.Base(path),
		"", "program", "program", true)
	return true
}
