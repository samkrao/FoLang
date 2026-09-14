package parser

import (
	"os"
	"path/filepath"
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/project"
)

func TestInstalledStandardArtifactLoadsAndMergesCanonicalGraph(t *testing.T) {
	installRoot := t.TempDir()
	executable := filepath.Join(installRoot, "bin", "folcc")
	artifactPath := filepath.Join(installRoot, "stdlib", "co.folenc")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("future-codec-payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	previousDecode := standardArtifactDecode
	t.Cleanup(func() { standardArtifactDecode = previousDecode })
	project.UseInstallationForTest(t, executable)
	standardArtifactDecode = func(raw []byte, out any) error {
		artifact := out.(*CompiledArtifact)
		artifact.SymbolFormatVersion = symboltable.SymbolFormatVersion
		graph := &symboltable.FolangSymbols{}
		graph.CreateFolangSymbols()
		graph.AddSymbolTable(&symboltable.SymbolTable{Id: "co:sym:root", ContextId: "co:ctx:root", SymbolsByName: map[string][]string{}})
		graph.AddContext(&symboltable.Context{Id: "co:ctx:root", Prefix: "co", SymbolTable_: "co:sym:root", ImportedContextIds: map[string]string{}})
		graph.AddFolContext(&symboltable.FolContext{Id: "co:fol:root", SymbolTable_: "co:sym:root", Context_: "co:ctx:root", Kind: "packaged", ExportedPackages: map[string]string{"co": "co:ctx:root"}})
		artifact.Name = "co"
		artifact.FolangSymbols = graph
		artifact.RootContextID = "co:ctx:root"
		return nil
	}

	artifact, gotPath, err := loadInstalledStandardArtifact()
	if err != nil {
		t.Fatalf("loading installed standard artifact: %v", err)
	}
	if gotPath != artifactPath {
		t.Fatalf("artifact path = %q, want %q", gotPath, artifactPath)
	}

	destination := &symboltable.FolangSymbols{}
	destination.CreateFolangSymbols()
	projectRoot := &symboltable.Context{Id: "ctx_project", SymbolTable_: "sym_project", ImportedContextIds: map[string]string{}}
	destination.AddContext(projectRoot)
	destination.AddSymbolTable(&symboltable.SymbolTable{Id: "sym_project", ContextId: projectRoot.Id, SymbolsByName: map[string][]string{}})
	if err := mergeInstalledStandardSymbols(destination, projectRoot, artifact); err != nil {
		t.Fatalf("merging standard symbols: %v", err)
	}
	if destination.GetContext("co:ctx:root") == nil || destination.GetSymbolTable("co:sym:root") == nil {
		t.Fatal("standard context or symbol table was not added to FolangSymbols")
	}
	if got := projectRoot.ImportedContextIds["co"]; got != "co:ctx:root" {
		t.Fatalf("implicit co context = %q, want co:ctx:root", got)
	}
	if got := destination.GetContext("co:ctx:root").ParentId; got != "" {
		t.Fatalf("co root parent = %q, want an independent imported root", got)
	}
}

func TestStandardMergePreservesCanonicalExportAncestry(t *testing.T) {
	standard := &symboltable.FolangSymbols{}
	standard.CreateFolangSymbols()
	standard.AddSymbolTable(&symboltable.SymbolTable{Id: "private-root-table", ContextId: "private-root", SymbolsByName: map[string][]string{}})
	standard.AddContext(&symboltable.Context{Id: "private-root", Prefix: "fΦλ", SymbolTable_: "private-root-table"})
	standard.AddSymbolTable(&symboltable.SymbolTable{Id: "private-out-table", ContextId: "private-out", SymbolsByName: map[string][]string{}})
	standard.AddContext(&symboltable.Context{Id: "private-out", ParentId: "private-root", ParentCtxSymbolTableId: "private-root-table", Prefix: "fΦλ.out", SymbolTable_: "private-out-table"})
	standard.AddFolContext(&symboltable.FolContext{Id: "standard-project", SymbolTable_: "private-root-table", Context_: "private-root", Kind: "packaged", ExportedPackages: map[string]string{"co.out": "private-out"}})
	artifact := &CompiledArtifact{SymbolFormatVersion: symboltable.SymbolFormatVersion, Name: "co", FolangSymbols: standard}

	destination := &symboltable.FolangSymbols{}
	destination.CreateFolangSymbols()
	projectRoot := &symboltable.Context{Id: "consumer-root", SymbolTable_: "consumer-table", ImportedContextIds: map[string]string{}}
	destination.AddContext(projectRoot)
	destination.AddSymbolTable(&symboltable.SymbolTable{Id: "consumer-table", ContextId: projectRoot.Id, SymbolsByName: map[string][]string{}})

	if err := mergeInstalledStandardSymbols(destination, projectRoot, artifact); err != nil {
		t.Fatalf("merging standard symbols: %v", err)
	}
	exported := destination.GetContext("private-out")
	if exported == nil || exported.ParentId != "private-root" || exported.ParentCtxSymbolTableId != "private-root-table" {
		t.Fatalf("canonical export ancestry was changed during merge: %#v", exported)
	}
	if got := projectRoot.ImportedContextIds["co.out"]; got != "private-out" {
		t.Fatalf("public projection = %q, want private-out", got)
	}
}

func TestMissingInstalledStandardArtifactIsBootstrapCompatible(t *testing.T) {
	installRoot := t.TempDir()
	executable := filepath.Join(installRoot, "bin", "folcc")
	project.UseInstallationForTest(t, executable)

	artifact, path, err := loadInstalledStandardArtifact()
	if err != nil {
		t.Fatalf("missing bootstrap artifact: %v", err)
	}
	if artifact != nil {
		t.Fatalf("artifact = %#v, want nil while co.folenc is unavailable", artifact)
	}
	if want := filepath.Join(installRoot, "stdlib", "co.folenc"); path != want {
		t.Fatalf("artifact path = %q, want %q", path, want)
	}
}
