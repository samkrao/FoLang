package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompilationInputsOrderComponentsLibrariesThenSource(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "runtime.folenc"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(root, "src", "appl.fol"),
		filepath.Join(root, "src", "hr", "Employee.fol"),
		filepath.Join(root, "components", "native", "component.fol"),
		filepath.Join(root, "components", "native", "impl", "Native.unit.fol"),
	}
	files := make([]File, 0, len(paths))
	for _, path := range paths {
		files = append(files, File{Path: path})
	}

	inputs, err := CompilationInputs(root, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 5 {
		t.Fatalf("inputs = %d, want 5", len(inputs))
	}
	if inputs[0].Stage != StageComponents || !inputs[0].Surface || inputs[0].ComponentKind != "native" {
		t.Fatalf("first input = %#v, want native component surface", inputs[0])
	}
	if inputs[1].Stage != StageComponents || inputs[1].Surface {
		t.Fatalf("second input = %#v, want native private source", inputs[1])
	}
	if inputs[2].Stage != StageLibraries {
		t.Fatalf("third input = %#v, want compiled library", inputs[2])
	}
	if inputs[3].Stage != StagePrimarySource || inputs[4].Stage != StagePrimarySource {
		t.Fatalf("primary inputs are not last: %#v", inputs)
	}
}
