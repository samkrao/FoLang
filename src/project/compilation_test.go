package project

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCompilationInputsRejectInvalidComponentLayouts(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{"unknown kind", []string{"components/unknown/component.fol"}, "not a standardized component kind"},
		{"missing surface", []string{"components/native/impl/Memory.fol"}, "has no component.fol surface"},
		{"operator implementation package", []string{"components/operators/component.fol", "components/operators/impl/X.fol"}, "no implementation packages"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			files := make([]File, 0, len(test.paths))
			for _, relative := range test.paths {
				files = append(files, File{Path: filepath.Join(root, filepath.FromSlash(relative))})
			}
			_, err := CompilationInputs(root, files)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text containing %q", err, test.want)
			}
		})
	}
}

func TestCompilationInputsOrdersOwnerBeforeCompanionBeforeUnit(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "src", "hr", "Z.unit.fol"),
		filepath.Join(root, "src", "hr", "Employee.comp.unit.fol"),
		filepath.Join(root, "src", "hr", "Employee.fol"),
	}
	files := make([]File, 0, len(paths))
	for _, path := range paths {
		files = append(files, File{Path: path})
	}
	inputs, err := CompilationInputs(root, files)
	if err != nil {
		t.Fatal(err)
	}
	if got := filepath.Base(inputs[0].Path); got != "Employee.fol" {
		t.Fatalf("first source = %s, want Employee.fol", got)
	}
	if got := filepath.Base(inputs[1].Path); got != "Employee.comp.unit.fol" {
		t.Fatalf("second source = %s, want Employee.comp.unit.fol", got)
	}
	if got := filepath.Base(inputs[2].Path); got != "Z.unit.fol" {
		t.Fatalf("third source = %s, want Z.unit.fol", got)
	}
}

func TestCompilationInputsRejectsCompanionWithoutOwner(t *testing.T) {
	root := t.TempDir()
	_, err := CompilationInputs(root, []File{{Path: filepath.Join(root, "src", "hr", "Employee.comp.unit.fol")}})
	if err == nil || !strings.Contains(err.Error(), "requires owner type file") {
		t.Fatalf("error = %v, want missing companion owner diagnostic", err)
	}
}

func TestValidateCompilationRootStructuralExclusivity(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantKind CompilationProjectKind
		want     string
	}{
		{"application", []string{"appl.fol"}, CompilationApplication, ""},
		{"standalone", []string{"component.fol"}, CompilationStandaloneComponent, ""},
		{"both", []string{"appl.fol", "component.fol"}, 0, "both"},
		{"neither", nil, 0, "no structural surface"},
		{"loose root source", []string{"Employee.fol", "appl.fol"}, 0, "loose file"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range test.files {
				if err := os.WriteFile(filepath.Join(root, "src", name), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			kind, findings := ValidateCompilationRoot(root)
			if kind != test.wantKind {
				t.Fatalf("kind = %v, want %v", kind, test.wantKind)
			}
			if test.want == "" {
				if len(findings) != 0 {
					t.Fatalf("unexpected findings: %v", findings)
				}
				return
			}
			if len(findings) == 0 || !strings.Contains(findings[0].Error(), test.want) {
				t.Fatalf("findings = %v, want text containing %q", findings, test.want)
			}
		})
	}
}
