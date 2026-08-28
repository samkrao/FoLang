package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedSymbolFlagLayoutIsCurrent(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "symbol-flag-layout.md")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, render()) {
		t.Fatalf("%s is stale; run go run ./cmd/symbolflags", path)
	}
}
