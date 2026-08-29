package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/project"
)

// backendConfig renders a backend interchange contract selecting one wire.
func backendConfig(wire string) string {
	return `{"protocol":"` + project.BackendProtocol +
		`","hir_schema":"` + project.BackendHIRSchema +
		`","wire":"` + wire +
		`","runtime_operations":"` + project.BackendRuntimeOperations + `"}`
}

// installBackendContract stands a toolchain up in a temporary directory with the
// given contract beside its executable, for the duration of one test.
//
// The contract belongs to the INSTALLATION, not to the project: the backend that
// will read the artifact decides its encoding, so a project cannot choose one its
// backend does not accept.
func installBackendContract(t *testing.T, wire string) {
	t.Helper()
	installRoot := t.TempDir()
	binDirectory := filepath.Join(installRoot, "bin")
	if err := os.MkdirAll(binDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(binDirectory, "folcc")
	write(t, executable, "compiler")
	if wire != "" {
		write(t, filepath.Join(binDirectory, project.BackendConfigFilename), backendConfig(wire))
	}
	project.UseInstallationForTest(t, executable)
}

// compileWithWire builds a one-file project under the given installed contract
// and returns the artifact path and the compile error.
func compileWithWire(t *testing.T, wire, source string) (string, error) {
	t.Helper()
	installBackendContract(t, wire)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, project.MarkerFilename), "project: demo")
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

// A 64-bit integer literal reaches the backend unchanged, on either wire.
//
// co.lang.int is 64-bit and ast.IntegerLiteral.Value is an int64. The artifact
// used to be encoded as a google.protobuf.Value, which stores every number as a
// double, so 9007199254740993 arrived as 9007199254740992 with nothing reported
// and the backend compiled a program the source never wrote. The artifact schema
// in src/shared/folang-artifact.proto carries the integer as an integer.
func TestALargeIntegerLiteralSurvivesEitherWire(t *testing.T) {
	const source = "big co.lang.int = 9007199254740993;\n"

	for _, wire := range []string{project.WireJSON, project.WireProtobuf} {
		artifact, err := compileWithWire(t, wire, source)
		if err != nil {
			t.Fatalf("compiling with the %s wire: %v", wire, err)
		}
		written, err := os.ReadFile(artifact)
		if err != nil {
			t.Fatal(err)
		}
		var envelope map[string]any
		if err := helpers.DeserializeArtifact(written, &envelope); err != nil {
			t.Fatalf("decoding the %s artifact: %v", wire, err)
		}
		got := integerLiteralValue(t, envelope)
		number, isNumber := got.(json.Number)
		if !isNumber || number.String() != "9007199254740993" {
			t.Errorf("%s wire carried the literal as %v (%T), want 9007199254740993", wire, got, got)
		}
	}
}

// integerLiteralValue finds the one IntegerLiteral node's Value in a decoded
// artifact. The digits also appear as the literal's symbol NAME, a string, so a
// text search over the artifact cannot tell a preserved value from a rounded one.
func integerLiteralValue(t *testing.T, envelope map[string]any) any {
	t.Helper()
	var found []any
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			if typed["NodeName"] == "IntegerLiteral" {
				found = append(found, typed["Value"])
			}
			for _, member := range typed {
				walk(member)
			}
		case []any:
			for _, member := range typed {
				walk(member)
			}
		}
	}
	walk(envelope)
	if len(found) != 1 {
		t.Fatalf("expected one IntegerLiteral, found %d", len(found))
	}
	return found[0]
}

// The boundary a double could still hold is unaffected, so nothing regressed for
// ordinary values while the large ones were fixed.
func TestTheLargestExactIntegerStillCompilesToProtobuf(t *testing.T) {
	if _, err := compileWithWire(t, project.WireProtobuf, "big co.lang.int = 9007199254740992;\n"); err != nil {
		t.Fatalf("2^53 must still compile to protobuf: %v", err)
	}
}

// A float literal and an integer literal stay different kinds of number.
//
// ast.NumberLiteral.Value is a float64 and ast.IntegerLiteral.Value an int64, but
// Go writes float64(3) as "3" — the same text an integer produces. The artifact
// carried `ratio co.lang.float = 3.0;` as an integer because of it, so a backend
// reading the wire saw a different kind of literal from the one written.
func TestAWholeFloatLiteralIsNotCarriedAsAnInteger(t *testing.T) {
	const source = "ratio co.lang.float = 3.0;\nwhole co.lang.int = 3;\n"

	for _, wire := range []string{project.WireJSON, project.WireProtobuf} {
		artifact, err := compileWithWire(t, wire, source)
		if err != nil {
			t.Fatalf("compiling with the %s wire: %v", wire, err)
		}
		written, err := os.ReadFile(artifact)
		if err != nil {
			t.Fatal(err)
		}
		var envelope map[string]any
		if err := helpers.DeserializeArtifact(written, &envelope); err != nil {
			t.Fatalf("decoding the %s artifact: %v", wire, err)
		}

		float, integer := literalValue(t, envelope, "NumberLiteral"), literalValue(t, envelope, "IntegerLiteral")
		if float != "3.0" {
			t.Errorf("%s wire carried the float literal as %q, want 3.0", wire, float)
		}
		if integer != "3" {
			t.Errorf("%s wire carried the integer literal as %q, want 3", wire, integer)
		}
	}
}

// literalValue returns the written form of the one node of the given kind.
func literalValue(t *testing.T, envelope map[string]any, nodeName string) string {
	t.Helper()
	var found []string
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			if typed["NodeName"] == nodeName {
				if number, ok := typed["Value"].(json.Number); ok {
					found = append(found, number.String())
				} else {
					found = append(found, fmt.Sprintf("%v (%T)", typed["Value"], typed["Value"]))
				}
			}
			for _, member := range typed {
				walk(member)
			}
		case []any:
			for _, member := range typed {
				walk(member)
			}
		}
	}
	walk(envelope)
	if len(found) != 1 {
		t.Fatalf("expected one %s, found %d", nodeName, len(found))
	}
	return found[0]
}
