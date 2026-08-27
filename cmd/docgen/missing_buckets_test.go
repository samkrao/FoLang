package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingBucketsSummaryMatchesGrammarMap(t *testing.T) {
	repo := filepath.Join("..", "..")
	gmap, err := buildGrammarMap(filepath.Join(repo, "src", "parser"), filepath.Join(repo, "docs", "grammar", "folang.ebnf"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo, "docs", "MISSING-BUCKETS.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	want := []string{
		fmt.Sprintf("| Grammar productions | %d |", len(gmap.Productions)+len(gmap.Missing)),
		fmt.Sprintf("| Productions claimed by indexed functions | %d |", len(gmap.Productions)),
		fmt.Sprintf("| Productions reported as `MISSING` | %d |", len(gmap.Missing)),
		fmt.Sprintf("| Parser functions reported as `EXTRA` | %d |", len(gmap.Extra)),
		fmt.Sprintf("| Productions with conflicting claims | %d |", len(gmap.Conflicts)),
	}
	for _, line := range want {
		if !strings.Contains(doc, line) {
			t.Errorf("MISSING-BUCKETS.md is stale: expected summary line %q", line)
		}
	}
}
