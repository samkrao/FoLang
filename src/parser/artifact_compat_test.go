package parser

import (
	"strings"
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
)

// artifactAtVersion builds a complete, decodable co.folenc declaring the given
// symbol format version and carrying a field this reader has never heard of.
func artifactAtVersion(version int) map[string]any {
	return map[string]any{
		"SymbolFormatVersion": version,
		"Name":                "co",
		"RootContextID":       "co:ctx:root",
		"FolangSymbols": map[string]any{
			"RootContextId": "co:fol:root",
			"SymboltableMap": map[string]any{
				"co:sym:root": map[string]any{"Id": "co:sym:root", "ContextId": "co:ctx:root", "SymbolsByName": map[string]any{}},
			},
			"ContextMap": map[string]any{
				"co:fol:root": map[string]any{"Id": "co:fol:root", "Context_": "co:ctx:root", "SymbolTable_": "co:sym:root", "Kind": "packaged", "ContextKind": "fol-context"},
				"co:ctx:root": map[string]any{"Id": "co:ctx:root", "Prefix": "co", "SymbolTable_": "co:sym:root"},
			},
			"SymbolsById": map[string]any{},
		},
		// what a newer producer added and this reader does not know
		"FutureField": true,
	}
}

// An artifact reader ignores a field it does not know, and an incompatible one
// still fails — on the version it declares, which is the check that means it.
//
// The two go together. Tolerating unknown fields is what lets a newer producer
// add one without breaking every reader built before it, and it costs nothing
// only because SymbolFormatVersion gates real incompatibility explicitly. Were
// the version gate to go, an unknown field would be the accident standing in for
// it — reporting a stale artifact by whichever field name happened to differ.
func TestAnIncompatibleArtifactFailsOnItsVersionNotItsFields(t *testing.T) {
	encoded, err := helpers.SerializeArtifact(artifactAtVersion(symboltable.SymbolFormatVersion + 1))
	if err != nil {
		t.Fatal(err)
	}

	var artifact CompiledArtifact
	if err := helpers.DeserializeArtifact(encoded, &artifact); err != nil {
		t.Fatalf("the unknown field was refused rather than ignored: %v", err)
	}
	if artifact.Name != "co" {
		t.Errorf("name = %q, want the field the reader does know", artifact.Name)
	}

	err = validateInstalledStandardArtifact(&artifact)
	if err == nil {
		t.Fatal("an unsupported symbol format version was accepted")
	}
	if !strings.Contains(err.Error(), "symbol format version") {
		t.Errorf("the failure does not name the version: %v", err)
	}
}

// The same artifact at the supported version passes, so the check above is
// failing on the version rather than on the fixture being incomplete.
func TestTheSameArtifactAtTheSupportedVersionIsAccepted(t *testing.T) {
	encoded, err := helpers.SerializeArtifact(artifactAtVersion(symboltable.SymbolFormatVersion))
	if err != nil {
		t.Fatal(err)
	}
	var artifact CompiledArtifact
	if err := helpers.DeserializeArtifact(encoded, &artifact); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if err := validateInstalledStandardArtifact(&artifact); err != nil {
		t.Fatalf("a supported artifact carrying an unknown field was rejected: %v", err)
	}
}
