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
