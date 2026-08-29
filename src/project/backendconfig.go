package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// LoadBackendConfig reads the installed contract, deriving its location the same
// way the standard package is found.
//
// Two places are looked at, in order:
//
//	<install-root>/backend-conf.json        beside stdlib/, where the toolchain
//	                                        keeps what belongs to the installation
//	<install-root>/bin/backend-conf.json    beside the executable itself
//
// The install root comes first because that is where the rest of the
// installation lives: the standard package is <install-root>/stdlib/co.folenc,
// and a contract describing the whole toolchain belongs at the same level rather
// than inside the directory that holds one binary. The bin/ location is still
// read, so an installation that placed it there keeps working.
//
// An absent contract is the default one. A malformed or unsupported contract
// stops compilation rather than being replaced by a guess: writing an encoding
// the installed backend does not accept produces an artifact nothing can read.
func LoadBackendConfig() (BackendConfig, error) {
	binDirectory, err := ExecutableDirectory()
	if err != nil {
		return BackendConfig{}, err
	}
	installRoot := filepath.Dir(binDirectory)

	for _, directory := range []string{installRoot, binDirectory} {
		config, found, err := readBackendConfig(directory)
		if err != nil {
			return BackendConfig{}, err
		}
		if found {
			return config, nil
		}
	}
	return DefaultBackendConfig(), nil
}

// readBackendConfig reads the contract from one directory, reporting whether one
// was there. A contract that exists but cannot be used is an error rather than a
// miss, so a broken file is never passed over in favour of the next location.
//
// A contract that exists must STATE all four fields. Decoding into the default
// contract would let a partial document inherit the rest and pass every check
// below — `{}` would validate as the default backend, and `{"wire":"json"}`
// would be accepted as declaring a protocol and an HIR schema it never mentions.
// The checks would be reading the defaults rather than the file.
func readBackendConfig(directory string) (BackendConfig, bool, error) {
	if directory == "" {
		return DefaultBackendConfig(), false, nil
	}
	path := filepath.Join(directory, BackendConfigFilename)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return BackendConfig{}, false, nil
	}
	if err != nil {
		return BackendConfig{}, false, fmt.Errorf("reading backend configuration %s: %w", path, err)
	}
	var config BackendConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return BackendConfig{}, false, fmt.Errorf("decoding backend configuration %s: %w", path, err)
	}
	// One document, and nothing after it. A second object following the first
	// would otherwise be dropped in silence, so a file holding two contracts
	// would compile under whichever one happened to be written first.
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return BackendConfig{}, false, fmt.Errorf("backend configuration %s: a second document follows the contract; the file holds exactly one", path)
		}
		return BackendConfig{}, false, fmt.Errorf("backend configuration %s: reading past the contract: %w", path, err)
	}
	for _, missing := range []struct{ field, value string }{
		{"protocol", config.Protocol},
		{"hir_schema", config.HIRSchema},
		{"wire", config.Wire},
		{"runtime_operations", config.RuntimeOperations},
	} {
		if missing.value == "" {
			return BackendConfig{}, false, fmt.Errorf("backend configuration %s: %s is missing; an installed contract states all four of protocol, hir_schema, wire and runtime_operations", path, missing.field)
		}
	}
	if config.Protocol != BackendProtocol {
		return BackendConfig{}, false, fmt.Errorf("backend configuration %s: protocol %q is unsupported; want %q", path, config.Protocol, BackendProtocol)
	}
	if config.HIRSchema != BackendHIRSchema {
		return BackendConfig{}, false, fmt.Errorf("backend configuration %s: hir_schema %q is unsupported; want %q", path, config.HIRSchema, BackendHIRSchema)
	}
	if config.RuntimeOperations != BackendRuntimeOperations {
		return BackendConfig{}, false, fmt.Errorf("backend configuration %s: runtime_operations %q is unsupported; want %q", path, config.RuntimeOperations, BackendRuntimeOperations)
	}
	if config.Wire != WireProtobuf && config.Wire != WireJSON {
		return BackendConfig{}, false, fmt.Errorf("backend configuration %s: wire %q is unsupported; use %q or %q", path, config.Wire, WireProtobuf, WireJSON)
	}
	return config, true, nil
}
