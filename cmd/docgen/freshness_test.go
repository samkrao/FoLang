package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type validationProvenance struct {
	SourceSHA256  string `json:"source_sha256"`
	SourceBytes   int    `json:"source_bytes"`
	SourceLines   int    `json:"source_lines"`
	GrammarSHA256 string `json:"grammar_sha256"`
	GrammarBytes  int    `json:"bytes"`
	GrammarLines  int    `json:"lines"`
}

func TestGeneratedGrammarAndCallgraphAreCurrent(t *testing.T) {
	pkgDir := filepath.Join("..", "..", "src", "parser")
	grammar := filepath.Join("..", "..", "docs", "grammar", "folang.ebnf")
	docs := filepath.Join("..", "..", "docs")

	gmap, err := buildGrammarMap(pkgDir, grammar)
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedJSON(t, filepath.Join(docs, "grammar-map.json"), gmap)

	callgraph, err := buildCallGraph(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedJSON(t, filepath.Join(docs, "callgraph.json"), callgraph)
}

func TestConformanceValidationProvenanceIsCurrent(t *testing.T) {
	root := filepath.Join("..", "..")
	reference, err := os.ReadFile(filepath.Join(root, "docs", "language-ref.md"))
	if err != nil {
		t.Fatal(err)
	}
	grammar, err := os.ReadFile(filepath.Join(root, "docs", "grammar", "folang.ebnf"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(filepath.Join(root, "docs", "grammar", "folang-conformance-validation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var provenance validationProvenance
	if err := json.Unmarshal(report, &provenance); err != nil {
		t.Fatal(err)
	}

	canonicalReference := canonicalLF(reference)
	refHash := sha256.Sum256(canonicalReference)
	if provenance.SourceSHA256 != fmt.Sprintf("%x", refHash) || provenance.SourceBytes != len(canonicalReference) || provenance.SourceLines != bytes.Count(canonicalReference, []byte{'\n'}) {
		t.Fatal("folang-conformance-validation.json has stale language-reference provenance")
	}
	canonicalGrammar := canonicalLF(grammar)
	grammarHash := sha256.Sum256(canonicalGrammar)
	if provenance.GrammarSHA256 != fmt.Sprintf("%x", grammarHash) || provenance.GrammarBytes != len(canonicalGrammar) || provenance.GrammarLines != bytes.Count(canonicalGrammar, []byte{'\n'}) {
		t.Fatal("folang-conformance-validation.json has stale grammar provenance")
	}
}

func assertGeneratedJSON(t *testing.T, path string, value any) {
	t.Helper()
	want, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, '\n')
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonicalLF(got), canonicalLF(want)) {
		t.Fatalf("%s is stale; regenerate it with go run ./cmd/docgen", path)
	}
}

// canonicalLF makes generated-artifact checks independent of core.autocrlf and
// of the platform on which a worktree was checked out.
func canonicalLF(content []byte) []byte {
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}
