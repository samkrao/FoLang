package entry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/parser"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestLoadDebugTraceConfigMissingFileDisablesTracing(t *testing.T) {
	parser.DEBUG_TRACE = true
	scanlex.DEBUG_TRACE = true

	err := loadDebugTraceConfig(filepath.Join(t.TempDir(), debugConfigFilename))
	if err != nil {
		t.Fatalf("missing configuration returned an error: %v", err)
	}
	if parser.DEBUG_TRACE || scanlex.DEBUG_TRACE {
		t.Fatal("missing configuration must disable parser and lexer tracing")
	}
}

func TestLoadDebugTraceConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), debugConfigFilename)
	data := []byte(`{"debug":{"trace":{"lexer":true,"parser":true}}}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDebugTraceConfig(path); err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	if !parser.DEBUG_TRACE || !scanlex.DEBUG_TRACE {
		t.Fatal("configuration did not enable both traces")
	}
}

func TestLoadDebugTraceConfigDefaultsOmittedPropertiesToFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), debugConfigFilename)
	if err := os.WriteFile(path, []byte(`{"debug":{"trace":{"parser":true}}}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDebugTraceConfig(path); err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	if !parser.DEBUG_TRACE || scanlex.DEBUG_TRACE {
		t.Fatalf("unexpected settings: parser=%v lexer=%v", parser.DEBUG_TRACE, scanlex.DEBUG_TRACE)
	}
}

func TestLoadDebugTraceConfigRejectsUnknownProperties(t *testing.T) {
	path := filepath.Join(t.TempDir(), debugConfigFilename)
	if err := os.WriteFile(path, []byte(`{"debug":{"trace":{"scanner":true}}}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDebugTraceConfig(path); err == nil {
		t.Fatal("unknown property was accepted")
	}
	if parser.DEBUG_TRACE || scanlex.DEBUG_TRACE {
		t.Fatal("invalid configuration must leave tracing disabled")
	}
}

func TestLoadDebugTraceConfigRejectsTrailingJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), debugConfigFilename)
	if err := os.WriteFile(path, []byte(`{} {}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDebugTraceConfig(path); err == nil {
		t.Fatal("trailing JSON value was accepted")
	}
}
