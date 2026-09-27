package preparser

import (
	"os"
	"path/filepath"
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

func TestFetchBackendConfUsesEmbeddedDefaultsWhenFileIsAbsent(t *testing.T) {
	symbols := &symboltable.FolangSymbols{}
	if err := fetchBackendConf(t.TempDir(), symbols); err != nil {
		t.Fatal(err)
	}
	want := symboltable.BackendConfig{
		Protocol:          "folang-plugin/1.0",
		HIRSchema:         "folang-hir/1",
		Wire:              "protobuf/1.0",
		RuntimeOperations: "folang-runtime-operations/1",
	}
	if symbols.BackendConf != want {
		t.Fatalf("BackendConf = %#v, want %#v", symbols.BackendConf, want)
	}
}

func TestFetchBackendConfReadsInstalledContract(t *testing.T) {
	installDir := t.TempDir()
	confDir := filepath.Join(installDir, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{
        "protocol": "folang-plugin/2.0",
        "hir_schema": "folang-hir/2",
        "wire": "protobuf/2.0",
        "runtime_operations": "folang-runtime-operations/2"
    }`)
	if err := os.WriteFile(filepath.Join(confDir, "backend-conf.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	symbols := &symboltable.FolangSymbols{}
	if err := fetchBackendConf(installDir, symbols); err != nil {
		t.Fatal(err)
	}
	if symbols.BackendConf.Protocol != "folang-plugin/2.0" ||
		symbols.BackendConf.HIRSchema != "folang-hir/2" ||
		symbols.BackendConf.Wire != "protobuf/2.0" ||
		symbols.BackendConf.RuntimeOperations != "folang-runtime-operations/2" {
		t.Fatalf("BackendConf = %#v", symbols.BackendConf)
	}
}
