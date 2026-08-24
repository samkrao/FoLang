// Command docgen writes the three generated parser documentation files.
//
//	docs/trace.json         function -> up to 5 deduplicated source snippets
//	docs/callgraph.json     function -> callers, callees
//	docs/grammar-map.json   grammar production -> implementing function
//
// The files contain data only, no prose. They are inputs to documentation
// tooling and must never be hand-edited.
//
// trace.json requires the recording build of the parser, because the spans it
// reports come from the instrumentation in src/parser/partrace_on.go:
//
//	go run -tags partrace ./cmd/docgen
//
// Without the tag the parser records nothing, so docgen writes the other two
// files, reports that trace.json was skipped, and leaves any existing
// trace.json untouched rather than overwriting it with an empty object.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		pkgDir  = flag.String("pkg", filepath.Join("src", "parser"), "parser package directory")
		corpus  = flag.String("corpus", filepath.Join("tests", "parser", "examples", "accepted"), "directory of .fol sources to trace")
		grammar = flag.String("grammar", filepath.Join("docs", "grammar", "folang.ebnf"), "EBNF grammar file")
		out     = flag.String("out", "docs", "output directory")
		limit   = flag.Int("limit", 5, "maximum snippets per function in trace.json")
	)
	flag.Parse()

	if err := run(*pkgDir, *corpus, *grammar, *out, *limit); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: %v\n", err)
		os.Exit(1)
	}
}

func run(pkgDir, corpus, grammar, out string, limit int) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	// grammar-map.json
	gmap, err := buildGrammarMap(pkgDir, grammar)
	if err != nil {
		return fmt.Errorf("grammar map: %w", err)
	}
	if err := writeJSON(filepath.Join(out, "grammar-map.json"), gmap); err != nil {
		return err
	}
	fmt.Printf("grammar-map.json  mapped=%d missing=%d extra=%d\n",
		len(gmap.Productions), len(gmap.Missing), len(gmap.Extra))

	// callgraph.json
	cg, err := buildCallGraph(pkgDir)
	if err != nil {
		return fmt.Errorf("call graph: %w", err)
	}
	if err := writeJSON(filepath.Join(out, "callgraph.json"), cg); err != nil {
		return err
	}
	fmt.Printf("callgraph.json    functions=%d\n", len(cg))

	// trace.json — only in a partrace build.
	snippets, ok, err := collectTrace(corpus, limit)
	if err != nil {
		return fmt.Errorf("trace: %w", err)
	}
	if !ok {
		fmt.Println("trace.json        SKIPPED (rebuild with -tags partrace)")
		return nil
	}
	if err := writeJSON(filepath.Join(out, "trace.json"), snippets); err != nil {
		return err
	}
	fmt.Printf("trace.json        functions=%d\n", len(snippets))
	return nil
}

// writeJSON writes v as indented JSON with a trailing newline. Map keys are
// sorted by encoding/json, so repeated runs over the same inputs produce
// byte-identical files and a regeneration shows an empty diff.
func writeJSON(path string, v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}
