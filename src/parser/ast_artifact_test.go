package parser

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/samkrao/fo-lang/src/project"
)

// The frontend artifact — docs/language-ref.md, "Compiler and Backend".
//
// "the frontend/backend interchange artifact is written beneath the reserved
// root-level `build/` domain", and `build/` is compiler-managed output the
// compiler "may create when absent". These tests pin the location, the creation,
// and the one property that makes the file usable: that the ids in it resolve
// inside it.

// writeProject lays out a minimal discoverable project and returns its root.
func writeProject(t *testing.T, entry string) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("creating src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "fol-conf.yaml"), []byte("project: artifact\n"), 0o644); err != nil {
		t.Fatalf("writing the project marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "appl.fol"), []byte(entry), 0o644); err != nil {
		t.Fatalf("writing the entry file: %v", err)
	}
	return root
}

func TestDebugTraceWritesFunctionFlowBesideASTArtifact(t *testing.T) {
	root := writeProject(t, "total co.lang.int = 1;\n")

	var humanTrace bytes.Buffer
	previousOutput, previousEnabled := debugTraceOutput, DEBUG_TRACE
	debugTraceOutput, DEBUG_TRACE = &humanTrace, true
	defer func() {
		debugTraceOutput, DEBUG_TRACE = previousOutput, previousEnabled
		resetDebugTraceEvents()
	}()

	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root)
	if err != nil {
		t.Fatalf("compiling with parser trace: %v", err)
	}
	tracePath := filepath.Join(root, project.BuildDomain, "appl"+debugTraceArtifactExtension)
	if artifact != filepath.Join(root, project.BuildDomain, "appl"+astArtifactExtension) {
		t.Fatalf("AST artifact path = %q", artifact)
	}
	content, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("reading parser trace artifact: %v", err)
	}
	var trace serializedDebugTrace
	if err := json.Unmarshal(content, &trace); err != nil {
		t.Fatalf("trace artifact is not valid JSON: %v", err)
	}
	if trace.Kind != "parser-function-flow" || len(trace.Events) == 0 {
		t.Fatalf("trace = %#v, want non-empty parser-function-flow", trace)
	}
	stack := make([]debugTraceEvent, 0)
	for index, event := range trace.Events {
		if event.Sequence != index+1 {
			t.Fatalf("event sequence = %d at index %d", event.Sequence, index)
		}
		switch event.Event {
		case "ENTER":
			if event.Depth != len(stack) {
				t.Fatalf("ENTER %s depth = %d, want %d", event.Function, event.Depth, len(stack))
			}
			stack = append(stack, event)
		case "EXIT":
			if len(stack) == 0 {
				t.Fatalf("unmatched EXIT for %s", event.Function)
			}
			entry := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if event.Function != entry.Function || event.Depth != entry.Depth {
				t.Fatalf("EXIT %#v does not match ENTER %#v", event, entry)
			}
		default:
			t.Fatalf("unknown trace event %q", event.Event)
		}
	}
	if len(stack) != 0 {
		t.Fatalf("trace ended with %d open functions", len(stack))
	}
}

func TestFocmainWritesTheArtifactBeneathBuild(t *testing.T) {
	root := writeProject(t, "total co.lang.int = 1;\n")

	_, artifact, serialized, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root)
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	want := filepath.Join(root, project.BuildDomain, "appl"+astArtifactExtension)
	if artifact != want {
		t.Fatalf("artifact path = %q, want %q", artifact, want)
	}

	written, readErr := os.ReadFile(artifact)
	if readErr != nil {
		t.Fatalf("reading the artifact: %v", readErr)
	}
	// The returned string and the file are the same bytes. A consumer that takes
	// one and a backend that reads the other must not see different programs.
	if string(written) != serialized {
		t.Error("the artifact on disk differs from the serialized value returned")
	}
}

// The artifact has to carry the scope graph, not just a reference to it. A
// context stores its symbol table by ID, so an envelope holding only the root
// context names a symbol table it does not contain — which is what this checks
// has not come back.
func TestArtifactCarriesAResolvableSymbolGraph(t *testing.T) {
	root := writeProject(t, "count co.lang.int = 1;\nname := \"folang\";\n")

	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root)
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	content, readErr := os.ReadFile(artifact)
	if readErr != nil {
		t.Fatalf("reading the artifact: %v", readErr)
	}

	var envelope struct {
		FolangSymbols struct {
			RootContextID  string `json:"RootContextId"`
			SymboltableMap map[string]json.RawMessage
			ContextMap     map[string]json.RawMessage
		} `json:"FolangSymbols"`
		AST json.RawMessage `json:"AST"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("the artifact is not valid JSON: %v", err)
	}

	if len(envelope.AST) == 0 {
		t.Error("the artifact carries no AST")
	}
	if envelope.FolangSymbols.RootContextID == "" {
		t.Error("the artifact carries no root context")
	}
	var rootContext struct {
		SymbolTable string `json:"SymbolTable_"`
	}
	rootWire, resolves := envelope.FolangSymbols.ContextMap[envelope.FolangSymbols.RootContextID]
	if !resolves {
		t.Fatalf("the root context %q is absent from the artifact's context map", envelope.FolangSymbols.RootContextID)
	}
	if err := json.Unmarshal(rootWire, &rootContext); err != nil {
		t.Fatal(err)
	}
	if _, resolves := envelope.FolangSymbols.SymboltableMap[rootContext.SymbolTable]; !resolves {
		t.Errorf("the root context names symbol table %q, which the artifact does not contain", rootContext.SymbolTable)
	}
}

func TestArtifactASTContainsOnlySymbolIDs(t *testing.T) {
	root := writeProject(t, "count co.lang.int = 1;\n")
	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		SymbolFormatVersion int `json:"symbolFormatVersion"`
		FolangSymbols       struct {
			SymbolsByID map[string]json.RawMessage `json:"SymbolsById"`
		} `json:"FolangSymbols"`
		AST any `json:"AST"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.SymbolFormatVersion != 1 || len(envelope.FolangSymbols.SymbolsByID) == 0 {
		t.Fatalf("symbol envelope = %#v", envelope)
	}
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if _, legacy := value["Symb"]; legacy {
				t.Error("AST contains an inline Symb record")
			}
			if id, ok := value["SymbolId"].(string); ok {
				if _, resolves := envelope.FolangSymbols.SymbolsByID[id]; !resolves {
					t.Errorf("AST symbol id %q does not resolve", id)
				}
			}
			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(envelope.AST)
}

// An in-process caller — a language server above all — parses a buffer per
// keystroke and must not touch the project tree to do it. A zero astArtifact is
// how that is expressed, so it has to write nothing at all.
func TestSerializeASTWritesNothingWithoutADestination(t *testing.T) {
	root := t.TempDir()

	source := "value co.lang.int = 1;\n"
	parsed := parseCollecting(nil, source, "artifact", root, "appl.fol", "", true, parseConfiguration{})
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("parsing produced diagnostics: %v", parsed.Diagnostics)
	}

	encoded, written, err := serializeAST(parsed.Root, parsed.Context, parsed.Symbols, false, astArtifact{})
	if err != nil {
		t.Fatalf("serializing: %v", err)
	}
	if written != "" {
		t.Errorf("serializeAST reported writing %q with no destination configured", written)
	}
	if encoded == "" {
		t.Error("serializeAST returned nothing to its caller")
	}
	if _, err := os.Stat(filepath.Join(root, project.BuildDomain)); !os.IsNotExist(err) {
		t.Errorf("serializeAST created a %s domain with no destination configured", project.BuildDomain)
	}
}

// Binary output belongs to the serialization layer, which is not wired up. It
// must not leave a JSON file behind under a name that promised protobuf.
func TestBinaryOutputWritesNoArtifact(t *testing.T) {
	root := t.TempDir()

	source := "value co.lang.int = 1;\n"
	parsed := parseCollecting(nil, source, "artifact", root, "appl.fol", "", true, parseConfiguration{})

	_, written, err := serializeAST(parsed.Root, parsed.Context, parsed.Symbols, true, astArtifact{
		Root: root,
		Stem: "appl",
	})
	if err == nil {
		t.Fatal("binary output reported success while the encoding is unimplemented")
	}
	if written != "" {
		t.Errorf("binary output wrote %q", written)
	}
	if _, err := os.Stat(filepath.Join(root, project.BuildDomain)); !os.IsNotExist(err) {
		t.Errorf("binary output created a %s domain", project.BuildDomain)
	}
}

// build/ is a project-ROOT domain, and Discover succeeds for a loose source file
// by rooting a one-file project at that file's own directory. Treating that as a
// project root put the artifact wherever the file happened to sit — for a package
// file, `src/hr/build/`, a compiler-managed directory inside a package, which
// "Project Layout" does not admit: every direct entry under src/ must be a package
// directory.
//
// With no marker and no override there is no known root, so nothing is written and
// the caller still receives the serialized envelope.
func TestNoArtifactWithoutADiscoveredProjectRoot(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "src", "hr")
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("creating the package directory: %v", err)
	}
	source := filepath.Join(packageDir, "Employee.fol")
	if err := os.WriteFile(source, []byte("_ co.lang.struct = {\n    id co.lang.int;\n}\n"), 0o644); err != nil {
		t.Fatalf("writing the source: %v", err)
	}

	// No fol-conf.yaml anywhere, and no explicit root.
	_, artifact, serialized, _, err := Focmain(source, false, false, "", false, "")
	if err != nil {
		t.Fatalf("compiling: %v", err)
	}

	if artifact != "" {
		t.Errorf("wrote %q for a file outside any discovered project", artifact)
	}
	if serialized == "" {
		t.Error("the caller received no serialized envelope")
	}
	if _, err := os.Stat(filepath.Join(packageDir, project.BuildDomain)); !os.IsNotExist(err) {
		t.Errorf("created a %s directory inside the package %s", project.BuildDomain, packageDir)
	}
	if _, err := os.Stat(filepath.Join(root, project.BuildDomain)); !os.IsNotExist(err) {
		t.Errorf("created a %s directory at %s, whose root was never discovered", project.BuildDomain, root)
	}
}
