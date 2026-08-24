package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedCorporaAreCurrent(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(old, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	blocks, err := extractBlocks(referencePath)
	if err != nil {
		t.Fatal(err)
	}
	known, _, err := loadExistingClassifications()
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range blocks {
		category, ok := known[hashOf(block.content)]
		if !ok {
			t.Fatalf("reference block L%04d is absent from the generated corpus; run go run ./cmd/refblocks -write", block.line)
		}
		expected := filepath.Join(corpusRoot, string(category), block.directory(), block.filename())
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("reference block L%04d has stale corpus placement %s: %v; run go run ./cmd/refblocks -write", block.line, expected, err)
		}
	}

	actual := 0
	for _, category := range []category{catParsing, catInvalid, catExcluded} {
		entries, err := os.ReadDir(filepath.Join(corpusRoot, string(category)))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				actual++
			}
		}
	}
	if actual != len(blocks) {
		t.Fatalf("generated corpus contains %d blocks, reference contains %d; run go run ./cmd/refblocks -write", actual, len(blocks))
	}
}
