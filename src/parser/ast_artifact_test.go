package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
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
	installBackendContract(t, project.WireJSON)
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

// A source file beneath a discovered project is compiled as that project, even
// when the caller does not repeat the root explicitly. The artifact root is the
// ProjectStmt wrapper; Application is its structural EntryStmt, not the root.
func TestDiscoveredProjectArtifactHasProjectRootAndKind(t *testing.T) {
	root := writeProject(t, "total co.lang.int = 1;\n")

	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, "")
	if err != nil {
		t.Fatalf("compiling discovered project: %v", err)
	}
	written, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		FolangSymbols struct {
			ContextMap map[string]json.RawMessage
		}
		AST struct {
			NodeName    string
			ProjectKind string
			EntryStmt   struct{ NodeName string }
		}
	}
	if err := json.Unmarshal(written, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.AST.NodeName != "ProjectStmt" {
		t.Errorf("root NodeName = %q, want ProjectStmt", envelope.AST.NodeName)
	}
	if envelope.AST.ProjectKind != "application" {
		t.Errorf("ProjectKind = %q, want application", envelope.AST.ProjectKind)
	}
	if envelope.AST.EntryStmt.NodeName != "Application" {
		t.Errorf("entry NodeName = %q, want Application", envelope.AST.EntryStmt.NodeName)
	}
	if len(envelope.FolangSymbols.ContextMap) != 2 {
		t.Errorf("single-file project contexts = %d, want independent surface and operational contexts", len(envelope.FolangSymbols.ContextMap))
	}
}

func TestArtifactOmitsParserOnlyStatementAndApplicationSymbols(t *testing.T) {
	root := writeProject(t, "x co.lang.int = 1;\nco.out.println(x);\n")
	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		FolangSymbols struct {
			FolContext struct {
				Context string `json:"Context_"`
			} `json:"FolContext"`
			Contexts map[string]struct {
				SymbolTable string `json:"SymbolTable_"`
			} `json:"ContextMap"`
			SymbolsByID map[string]struct {
				SymbolType string         `json:"symbolType"`
				Name       string         `json:"name"`
				Fields     map[string]any `json:"fields"`
			} `json:"SymbolsById"`
			SymbolTables map[string]struct {
				SymbolIDs     []string            `json:"SymbolIds"`
				SymbolsByName map[string][]string `json:"SymbolsByName"`
			} `json:"SymboltableMap"`
		} `json:"FolangSymbols"`
		AST any `json:"AST"`
	}
	if err := json.Unmarshal(written, &envelope); err != nil {
		t.Fatal(err)
	}
	for id, symbol := range envelope.FolangSymbols.SymbolsByID {
		if symbol.SymbolType == string(symboltable.S_StatmentSymbol) {
			t.Errorf("artifact retained parser-only statement symbol %s", id)
		}
		if symbol.SymbolType == string(symboltable.S_ExpressionSymbol) {
			t.Errorf("artifact retained unbound expression-occurrence symbol %s", id)
		}
		if symbol.SymbolType == string(symboltable.S_TypeSymbol) && symbol.Name == "co.lang.int" {
			t.Errorf("artifact retained the declaration's embedded type occurrence %s", id)
		}
		if symbol.SymbolType == string(symboltable.S_PackageSymbol) && symbol.Name == "appl.fol" {
			t.Errorf("artifact retained parser-only application anchor %s", id)
		}
		if symbol.SymbolType == string(symboltable.S_ComponentSymbol) && symbol.Fields["Kind"] == "project" {
			t.Errorf("artifact retained structural project-wrapper symbol %s", id)
		}
	}
	for tableID, table := range envelope.FolangSymbols.SymbolTables {
		if len(table.SymbolIDs) == 0 && len(table.SymbolsByName) == 0 {
			rootTable := envelope.FolangSymbols.Contexts[envelope.FolangSymbols.FolContext.Context].SymbolTable
			if tableID != rootTable {
				t.Errorf("artifact retained empty symbol-table segment %s", tableID)
			}
		}
		for _, id := range table.SymbolIDs {
			if _, exists := envelope.FolangSymbols.SymbolsByID[id]; !exists {
				t.Errorf("table %s retains omitted symbol %s", tableID, id)
			}
		}
		for name, ids := range table.SymbolsByName {
			for _, id := range ids {
				if _, exists := envelope.FolangSymbols.SymbolsByID[id]; !exists {
					t.Errorf("table %s key %s retains omitted symbol %s", tableID, name, id)
				}
			}
		}
	}
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if node, _ := value["NodeName"].(string); node == "ExpressionStmt" || node == "Application" || node == "ProjectStmt" {
				if _, hasID := value["SymbolId"]; hasID {
					t.Errorf("%s retains a parser-only SymbolId", node)
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

func TestArtifactClassifiesASTNodesIndependentlyOfDataTypes(t *testing.T) {
	root := writeProject(t, "x co.lang.int = 10;\ny := x + 1;\n")
	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct{ AST any }
	if err := json.Unmarshal(written, &envelope); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ProjectStmt":        "co.lang.statement",
		"VarDeclarationStmt": "co.lang.statement",
		"BinaryExpr":         "co.lang.expression",
		"SymbolExpr":         "co.lang.symbol",
		"IntegerLiteral":     "co.lang.literal",
	}
	seen := map[string]bool{}
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if name, _ := value["NodeName"].(string); want[name] != "" {
				seen[name] = true
				if value["NodeType_"] != want[name] {
					t.Errorf("%s NodeType_ = %v, want %s", name, value["NodeType_"], want[name])
				}
			}
			if value["Value"] == "+" && value["NodeType_"] != "co.lang.operator" {
				t.Errorf("operator NodeType_ = %v", value["NodeType_"])
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
	for name := range want {
		if !seen[name] {
			t.Errorf("artifact contains no %s to classify", name)
		}
	}
}

func TestParseTimeLookupReusesVisibleDeclarationIDs(t *testing.T) {
	root := writeProject(t, "x := 1;\nco.out.println(x);\nx ?= 2;\nx = 3;\ny := x;\n")
	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, "")
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		FolangSymbols struct {
			SymbolsByID map[string]struct {
				SymbolType string          `json:"symbolType"`
				Name       string          `json:"name"`
				Type       string          `json:"type"`
				State      json.RawMessage `json:"state"`
			} `json:"SymbolsById"`
		} `json:"FolangSymbols"`
		AST any
	}
	if err := json.Unmarshal(written, &envelope); err != nil {
		t.Fatal(err)
	}
	declarations := map[string]string{}
	for id, symbol := range envelope.FolangSymbols.SymbolsByID {
		if symbol.State != nil {
			t.Errorf("symbol %s serializes occurrence resolution state: %s", id, symbol.State)
		}
		if symbol.SymbolType != string(symboltable.S_VarSymbol) {
			continue
		}
		if previous := declarations[symbol.Name]; previous != "" {
			t.Errorf("variable %s has two declaration IDs: %s and %s", symbol.Name, previous, id)
		}
		declarations[symbol.Name] = id
		if (symbol.Name == "x_fo" || symbol.Name == "y_fo") && symbol.Type != "co.lang.int" {
			t.Errorf("inferred type of %s = %q, want co.lang.int", symbol.Name, symbol.Type)
		}
	}
	if len(declarations) != 2 || declarations["x_fo"] == "" || declarations["y_fo"] == "" {
		t.Fatalf("variable declarations = %v, want exactly x and y", declarations)
	}
	var xDeclarationIDs, xReferenceIDs []string
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			name, _ := value["NodeName"].(string)
			if name == "VarDeclarationStmt" {
				basic, _ := value["BasicVarStmt"].(map[string]any)
				if value["ResolutionState_"] != string(ast.ResolutionResolved) {
					t.Errorf("variable declaration state = %v, want RESOLVED", value["ResolutionState_"])
				}
				if basic["VarType"] != "co.lang.int" {
					t.Errorf("variable declaration type = %v, want co.lang.int", basic["VarType"])
				}
				if basic["Identifier"] == "x_fo" {
					xDeclarationIDs = append(xDeclarationIDs, fmt.Sprint(value["SymbolId"]))
				}
			}
			if name == "SymbolExpr" && value["Value"] == "x_fo" {
				xReferenceIDs = append(xReferenceIDs, fmt.Sprint(value["SymbolId"]))
				if value["ResolutionState_"] != string(ast.ResolutionResolved) {
					t.Errorf("x reference state = %v, want RESOLVED", value["ResolutionState_"])
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
	for _, id := range append(xDeclarationIDs, xReferenceIDs...) {
		if id != declarations["x_fo"] {
			t.Errorf("x occurrence resolves to %q, want declaration %q", id, declarations["x_fo"])
		}
	}
	if len(xDeclarationIDs) != 2 {
		t.Errorf("x declaration-form occurrences = %d, want := and ?=", len(xDeclarationIDs))
	}
}

func TestTypeOccurrenceResolutionClassification(t *testing.T) {
	tests := []struct {
		name string
		node string
		wire map[string]any
		want string
	}{
		{"built-in type", "BuiltInDataType", map[string]any{"Value": "co.lang.int"}, string(ast.ResolutionResolved)},
		{"named type", "SymbolTypeNode", map[string]any{"Value": "Customer"}, string(ast.ResolutionPartiallyResolved)},
		{"parameter type", "Parameter", map[string]any{"Type_": map[string]any{"ResolutionState_": string(ast.ResolutionResolved)}}, string(ast.ResolutionResolved)},
		{"return type", "Returns", map[string]any{"Type_": map[string]any{"ResolutionState_": string(ast.ResolutionPartiallyResolved)}}, string(ast.ResolutionPartiallyResolved)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := astNodeResolutionState(test.node, test.wire); got != test.want {
				t.Fatalf("resolution state = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFrontendArtifactDefaultsToProtobuf(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, project.MarkerFilename), []byte("project: artifact\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(root, "src", "appl.fol")
	if err := os.WriteFile(entry, []byte("total co.lang.int = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, artifact, _, _, err := Focmain(entry, false, false, "", false, root)
	if err != nil {
		t.Fatal(err)
	}
	if artifact != filepath.Join(root, project.BuildDomain, "appl"+astProtobufExtension) {
		t.Fatalf("artifact path = %q, want protobuf default", artifact)
	}
	raw, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var decoded serializedAST
	if err := helpers.UnmarshalProtobuf(raw, &decoded); err != nil {
		t.Fatalf("decoding protobuf artifact: %v", err)
	}
	if decoded.Wire != project.WireProtobuf || decoded.AST == nil || decoded.Symbols == nil {
		t.Fatalf("decoded protobuf envelope = %#v", decoded)
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
			if _, duplicated := value["FolangSymbols"]; duplicated {
				t.Error("AST duplicates the canonical FolangSymbols graph")
			}
			if _, duplicated := value["SurfaceFileSymbols"]; duplicated {
				t.Error("AST duplicates the canonical surface-symbol index")
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

// backend-conf.json is authoritative; the legacy binary argument no longer
// overrides a configured JSON wire.
func TestLegacyBinaryFlagDoesNotOverrideBackendConfig(t *testing.T) {
	root := t.TempDir()
	installBackendContract(t, project.WireJSON)

	source := "value co.lang.int = 1;\n"
	parsed := parseCollecting(nil, source, "artifact", root, "appl.fol", "", true, parseConfiguration{})

	_, written, err := serializeAST(parsed.Root, parsed.Context, parsed.Symbols, true, astArtifact{
		Root: root,
		Stem: "appl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(written) != ".json" {
		t.Fatalf("artifact = %q, want configured JSON wire", written)
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
