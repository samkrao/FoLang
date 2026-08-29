package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validContract = `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"json","runtime_operations":"folang-runtime-operations/1"}`

// installationWith stands a toolchain up in a temporary directory and points the
// installation derivation at its executable. contract is written beside that
// executable when non-empty.
// installationWithAt stands a toolchain up and writes the contract into one of
// its directories: "" for the install root, "bin" for beside the executable.
func installationWithAt(t *testing.T, where, contract string) string {
	t.Helper()
	root := t.TempDir()
	binDirectory := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(binDirectory, "folcc")
	if err := os.WriteFile(executable, []byte("compiler"), 0o755); err != nil {
		t.Fatal(err)
	}
	if contract != "" {
		if err := os.WriteFile(filepath.Join(root, where, BackendConfigFilename), []byte(contract), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	UseInstallationForTest(t, executable)
	return root
}

func installationWith(t *testing.T, contract string) string {
	t.Helper()
	root := t.TempDir()
	binDirectory := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(binDirectory, "folcc")
	if err := os.WriteFile(executable, []byte("compiler"), 0o755); err != nil {
		t.Fatal(err)
	}
	if contract != "" {
		if err := os.WriteFile(filepath.Join(binDirectory, BackendConfigFilename), []byte(contract), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	UseInstallationForTest(t, executable)
	return root
}

// The contract is a property of the INSTALLATION, not of a project: the backend
// that will read the artifact decides its encoding, so a project cannot choose
// one its backend does not accept. It is read from beside the compiler
// executable (docs/language-ref.md, "Installed Backend Interchange Contract").
func TestBackendConfigIsReadFromBesideTheCompiler(t *testing.T) {
	installationWith(t, validContract)

	config, err := LoadBackendConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Wire != WireJSON {
		t.Fatalf("wire = %q, want json", config.Wire)
	}
}

// An absent contract is the default backend's answer rather than an error, so a
// toolchain that ships no contract still compiles.
func TestBackendConfigDefaultsToProtobuf(t *testing.T) {
	installationWith(t, "")

	config, err := LoadBackendConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config != DefaultBackendConfig() || config.Wire != WireProtobuf {
		t.Fatalf("default backend config = %#v", config)
	}
}

// A contract in the PROJECT is not read. Putting one there would look like it
// worked while the installed backend went on expecting its own encoding.
func TestABackendConfigInTheProjectIsIgnored(t *testing.T) {
	installationWith(t, "")

	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, BackendConfigFilename), []byte(validContract), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadBackendConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Wire != WireProtobuf {
		t.Fatalf("a project-local contract was read: wire = %q", config.Wire)
	}
}

// An unusable contract stops compilation rather than being replaced by a guess:
// writing an encoding the installed backend does not accept produces an artifact
// nothing can read.
func TestBackendConfigRejectsAnUnusableContract(t *testing.T) {
	for name, test := range map[string]struct{ contract, want string }{
		"an unsupported wire": {
			contract: `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"bson","runtime_operations":"folang-runtime-operations/1"}`,
			want:     "wire",
		},
		"an unsupported protocol": {
			contract: `{"protocol":"folang-plugin/2.0","hir_schema":"folang-hir/1","wire":"json","runtime_operations":"folang-runtime-operations/1"}`,
			want:     "protocol",
		},
		"an unknown field": {
			contract: `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"json","runtime_operations":"folang-runtime-operations/1","extra":1}`,
			want:     "extra",
		},
		"malformed json": {
			contract: `{`,
			want:     "decoding backend configuration",
		},
		"an empty object": {
			contract: `{}`,
			want:     "is missing",
		},
		"only a wire": {
			contract: `{"wire":"json"}`,
			want:     "protocol is missing",
		},
		"a missing runtime_operations": {
			contract: `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"json"}`,
			want:     "runtime_operations is missing",
		},
		"a second contract after the first": {
			contract: validContract + ` ` + validContract,
			want:     "a second document follows the contract",
		},
	} {
		t.Run(name, func(t *testing.T) {
			installationWith(t, test.contract)

			_, err := LoadBackendConfig()
			if err == nil {
				t.Fatal("an unusable contract was accepted")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("error does not mention %q: %v", test.want, err)
			}
		})
	}
}

// The standard package and the backend contract are located from ONE derivation,
// so they can never disagree about which toolchain is running.
func TestTheInstallationIsDerivedFromTheRealExecutable(t *testing.T) {
	root := installationWith(t, "")

	binDirectory, err := ExecutableDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if binDirectory != filepath.Join(root, "bin") {
		t.Errorf("executable directory = %q, want %q", binDirectory, filepath.Join(root, "bin"))
	}
	installRoot, err := InstallRoot()
	if err != nil {
		t.Fatal(err)
	}
	if installRoot != root {
		t.Errorf("install root = %q, want %q", installRoot, root)
	}
}

// The contract is read from the install root, beside stdlib/, and also from
// beside the executable so an installation that placed it there keeps working.
func TestBackendConfigIsReadFromEitherInstalledLocation(t *testing.T) {
	for name, where := range map[string]string{
		"the install root, beside stdlib/": "",
		"beside the executable, in bin/":   "bin",
	} {
		t.Run(name, func(t *testing.T) {
			installationWithAt(t, where, validContract)

			config, err := LoadBackendConfig()
			if err != nil {
				t.Fatal(err)
			}
			if config.Wire != WireJSON {
				t.Fatalf("wire = %q, want json", config.Wire)
			}
		})
	}
}

// The install root wins, so a contract left in bin/ by an older installation
// cannot quietly override the one the toolchain now ships.
func TestTheInstallRootContractWinsOverBin(t *testing.T) {
	root := installationWithAt(t, "", validContract)
	stale := `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"protobuf","runtime_operations":"folang-runtime-operations/1"}`
	if err := os.WriteFile(filepath.Join(root, "bin", BackendConfigFilename), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadBackendConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Wire != WireJSON {
		t.Fatalf("wire = %q, want the install root's json", config.Wire)
	}
}

// A contract that exists but cannot be used is an error, not a miss: passing
// over a broken file in favour of the next location would compile against a
// contract the installation did not mean.
func TestABrokenContractIsNotPassedOver(t *testing.T) {
	root := installationWithAt(t, "", `{"wire":"json"}`)
	if err := os.WriteFile(filepath.Join(root, "bin", BackendConfigFilename), []byte(validContract), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadBackendConfig(); err == nil {
		t.Fatal("a broken contract was skipped in favour of another location")
	}
}
