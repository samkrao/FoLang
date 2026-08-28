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

// BackendConfig is the frontend/backend interchange contract. Protobuf is the
// default when backend-conf.json is absent.
type BackendConfig struct {
	Protocol          string `json:"protocol"`
	HIRSchema         string `json:"hir_schema"`
	Wire              string `json:"wire"`
	RuntimeOperations string `json:"runtime_operations"`
}

func DefaultBackendConfig() BackendConfig {
	return BackendConfig{Protocol: BackendProtocol, HIRSchema: BackendHIRSchema, Wire: WireProtobuf, RuntimeOperations: BackendRuntimeOperations}
}

func LoadBackendConfig(root string) (BackendConfig, error) {
	config := DefaultBackendConfig()
	if root == "" {
		return config, nil
	}
	path := filepath.Join(root, BackendConfigFilename)
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
