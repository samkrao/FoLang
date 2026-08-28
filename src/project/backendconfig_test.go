package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackendConfigDefaultsToProtobuf(t *testing.T) {
	config, err := LoadBackendConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if config != DefaultBackendConfig() || config.Wire != WireProtobuf {
		t.Fatalf("default backend config = %#v", config)
	}
}

func TestBackendConfigSelectsJSON(t *testing.T) {
	root := t.TempDir()
	raw := `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"json","runtime_operations":"folang-runtime-operations/1"}`
	if err := os.WriteFile(filepath.Join(root, BackendConfigFilename), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := LoadBackendConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.Wire != WireJSON {
		t.Fatalf("wire = %q, want json", config.Wire)
	}
}
