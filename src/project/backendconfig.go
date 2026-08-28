package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const BackendConfigFilename = "backend-conf.json"

const (
	BackendProtocol          = "folang-plugin/1.0"
	BackendHIRSchema         = "folang-hir/1"
	BackendRuntimeOperations = "folang-runtime-operations/1"
	WireProtobuf             = "protobuf"
	WireJSON                 = "json"
)

// BackendConfig is the frontend/backend interchange contract.
//
// The selected backend supplies it "to tell the frontend which FoLang/plugin
// protocol, HIR schema, and wire format it accepts", and installing a backend
// places it "in the same installation directory as the FoLang compiler
// executable" (docs/language-ref.md, "Installed Backend Interchange Contract").
//
// It is therefore a property of the TOOLCHAIN, not of a project: which encoding
// the frontend writes is decided by the backend that will read it, so a project
// cannot choose one its backend does not accept. Protobuf is what an absent
// contract means, which is the default backend's own answer.
type BackendConfig struct {
	Protocol          string `json:"protocol"`
	HIRSchema         string `json:"hir_schema"`
	Wire              string `json:"wire"`
	RuntimeOperations string `json:"runtime_operations"`
}

func DefaultBackendConfig() BackendConfig {
	return BackendConfig{Protocol: BackendProtocol, HIRSchema: BackendHIRSchema, Wire: WireProtobuf, RuntimeOperations: BackendRuntimeOperations}
}

// LoadBackendConfig reads the installed contract from beside the running
// compiler, deriving that location the same way the standard package is found.
// An absent contract is the default one; a malformed or unsupported contract
// stops compilation rather than being replaced by a guess.
func LoadBackendConfig() (BackendConfig, error) {
	directory, err := ExecutableDirectory()
	if err != nil {
		return BackendConfig{}, err
	}
	return loadBackendConfigFrom(directory)
}

// loadBackendConfigFrom reads the contract from one directory.
func loadBackendConfigFrom(directory string) (BackendConfig, error) {
	config := DefaultBackendConfig()
	if directory == "" {
		return config, nil
	}
	path := filepath.Join(directory, BackendConfigFilename)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return BackendConfig{}, fmt.Errorf("reading backend configuration %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return BackendConfig{}, fmt.Errorf("decoding backend configuration %s: %w", path, err)
	}
	if config.Protocol != BackendProtocol {
		return BackendConfig{}, fmt.Errorf("backend configuration %s: protocol %q is unsupported; want %q", path, config.Protocol, BackendProtocol)
	}
	if config.HIRSchema != BackendHIRSchema {
		return BackendConfig{}, fmt.Errorf("backend configuration %s: hir_schema %q is unsupported; want %q", path, config.HIRSchema, BackendHIRSchema)
	}
	if config.RuntimeOperations != BackendRuntimeOperations {
		return BackendConfig{}, fmt.Errorf("backend configuration %s: runtime_operations %q is unsupported; want %q", path, config.RuntimeOperations, BackendRuntimeOperations)
	}
	if config.Wire != WireProtobuf && config.Wire != WireJSON {
		return BackendConfig{}, fmt.Errorf("backend configuration %s: wire %q is unsupported; use %q or %q", path, config.Wire, WireProtobuf, WireJSON)
	}
	return config, nil
}
