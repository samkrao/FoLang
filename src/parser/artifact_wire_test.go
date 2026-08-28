package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/project"
)

// backendConfig renders a backend interchange contract selecting one wire.
func backendConfig(wire string) string {
	return `{"protocol":"` + project.BackendProtocol +
		`","hir_schema":"` + project.BackendHIRSchema +
		`","wire":"` + wire +
		`","runtime_operations":"` + project.BackendRuntimeOperations + `"}`
}

// compileWithWire builds a one-file project under the given contract and returns
// the artifact path and the compile error.
func compileWithWire(t *testing.T, wire, source string) (string, error) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, project.MarkerFilename), "project: demo\n")
	if wire != "" {
		write(t, filepath.Join(root, project.BackendConfigFilename), backendConfig(wire))
	}
	write(t, filepath.Join(root, "src", "appl.fol"), source)

	_, artifact, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root)
	return artifact, err
}

// The backend contract selects the artifact's encoding, and protobuf is what an
// absent contract means.
func TestBackendContractSelectsTheArtifactEncoding(t *testing.T) {
	for name, test := range map[string]struct{ wire, suffix string }{
		"no contract": {"", ".ast.pb"},
		"protobuf":    {project.WireProtobuf, ".ast.pb"},
		"json":        {project.WireJSON, ".ast.json"},
	} {
		t.Run(name, func(t *testing.T) {
			artifact, err := compileWithWire(t, test.wire, "total co.lang.int = 1;\n")
			if err != nil {
				t.Fatalf("compiling: %v", err)
			}
			if !strings.HasSuffix(artifact, test.suffix) {
				t.Errorf("artifact = %s, want one ending %s", filepath.Base(artifact), test.suffix)
			}
		})
	}
}

// An integer literal past 2^53 must not reach the backend as a different number.
//
// co.lang.int is 64-bit and ast.IntegerLiteral.Value is an int64, but a protobuf
// google.protobuf.Value stores every number as a double. Encoding 9007199254740993
// through it used to yield 9007199254740992 with nothing reported, so the backend
// compiled a program the source never wrote.
func TestALargeIntegerLiteralIsNeverSilentlyRounded(t *testing.T) {
	const source = "big co.lang.int = 9007199254740993;\n"

	// The JSON wire carries it exactly.
	artifact, err := compileWithWire(t, project.WireJSON, source)
	if err != nil {
		t.Fatalf("compiling with the json wire: %v", err)
	}
	written, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), `"Value": 9007199254740993`) {
		t.Error("the json artifact does not carry the literal exactly")
	}

	// The protobuf wire cannot, and says so instead of rounding.
	if _, err := compileWithWire(t, project.WireProtobuf, source); err == nil {
		t.Fatal("the protobuf wire encoded a literal it cannot represent")
	} else if !strings.Contains(err.Error(), "9007199254740993") {
		t.Errorf("the failure does not name the literal: %v", err)
	}
}

// The boundary itself still compiles under protobuf: 2^53 is the largest integer
// a double holds exactly, so refusing it would reject a program needlessly.
func TestTheLargestExactIntegerStillCompilesToProtobuf(t *testing.T) {
	if _, err := compileWithWire(t, project.WireProtobuf, "big co.lang.int = 9007199254740992;\n"); err != nil {
		t.Fatalf("2^53 must still compile to protobuf: %v", err)
	}
}
