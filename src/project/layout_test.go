package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLayout materializes a project tree from a path -> content map. A path ending in
// "/" creates an empty directory, which the presence invariant distinguishes from an
// absent one.
func writeLayout(t *testing.T, entries map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for path, content := range entries {
		full := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(path, "/")))
		if strings.HasSuffix(path, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func findingsText(layout Layout) string {
	parts := make([]string, len(layout.Findings))
	for i, finding := range layout.Findings {
		parts[i] = finding.Error()
	}
	return strings.Join(parts, "\n")
}

func TestValidLayoutsProduceNoFindings(t *testing.T) {
	tests := []struct {
		name    string
		entries map[string]string
		kind    ProjectKind
	}{
		{
			name:    "application with only src",
			entries: map[string]string{"src/appl.fol": "value := 1;\n"},
			kind:    KindApplication,
		},
		{
			name:    "standalone library",
			entries: map[string]string{"src/library.fol": "_ co.lang.library = {}\n"},
			kind:    KindStandaloneLibrary,
		},
		{
			name: "every optional domain populated",
			entries: map[string]string{
				"src/appl.fol":                         "value := 1;\n",
				"src/hr/employee/Employee.fol":         "_ co.lang.struct = {}\n",
				"srclib/ffi/library.fol":               "_ co.lang.library = {}\n",
				"srclib/ffi/native/marshal/M.unit.fol": "_ co.lang.unit = {}\n",
				"srclib/operators/library.fol":         "_ co.lang.library = {}\n",
				"lib/vendor.folenc":                    "binary",
				"build/":                               "",
			},
			kind: KindApplication,
		},
		{
			// build/ is compiler-owned, so an empty one is a valid generated state
			// while an empty srclib/ or lib/ is not.
			name: "empty build directory",
			entries: map[string]string{
				"src/appl.fol": "value := 1;\n",
				"build/":       "",
			},
			kind: KindApplication,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			layout := ValidateLayout(writeLayout(t, test.entries))
			if len(layout.Findings) != 0 {
				t.Fatalf("valid layout reported findings:\n%s", findingsText(layout))
			}
			if layout.Kind != test.kind {
				t.Fatalf("project kind = %v, want %v", layout.Kind, test.kind)
			}
		})
	}
}

func TestLayoutViolationsAreReported(t *testing.T) {
	tests := []struct {
		name    string
		entries map[string]string
		want    string
	}{
		{
			name: "project-local co artifact shadows standard package",
			entries: map[string]string{
				"src/appl.fol":  "value := 1;\n",
				"lib/co.folenc": "binary",
			},
			want: "cannot shadow the installed standard package",
		},
		{
			name:    "src is missing",
			entries: map[string]string{"README.md": "not source"},
			want:    "src/ is missing",
		},
		{
			name:    "src has no structural surface",
			entries: map[string]string{"src/Employee.fol": "_ co.lang.struct = {}\n"},
			want:    "no structural surface",
		},
		{
			// A project is an application or a standalone library; having both
			// leaves nothing to say which it is.
			name: "both structural surfaces",
			entries: map[string]string{
				"src/appl.fol":    "value := 1;\n",
				"src/library.fol": "_ co.lang.library = {}\n",
			},
			want: "contains both",
		},
		{
			name: "srclib present but empty",
			entries: map[string]string{
				"src/appl.fol": "value := 1;\n",
				"srclib/":      "",
			},
			want: "srclib/ is present but empty",
		},
		{
			name: "srclib holds an unstandardized slot",
			entries: map[string]string{
				"src/appl.fol":               "value := 1;\n",
				"srclib/helpers/library.fol": "_ co.lang.library = {}\n",
			},
			want: "not a standardized source-library slot",
		},
		{
			name: "source-library slot has no surface",
			entries: map[string]string{
				"src/appl.fol":                 "value := 1;\n",
				"srclib/ffi/native/M.unit.fol": "_ co.lang.unit = {}\n",
			},
			want: "has no library.fol",
		},
		{
			// The operator slot is a parser bootstrap, not an API with an
			// implementation behind it, so it holds one file and nothing else.
			name: "operator slot holds more than its surface",
			entries: map[string]string{
				"src/appl.fol":                 "value := 1;\n",
				"srclib/operators/library.fol": "_ co.lang.library = {}\n",
				"srclib/operators/extra.fol":   "value := 1;\n",
			},
			want: "holds exactly one file",
		},
		{
			name: "nested library surface below a source-library root",
			entries: map[string]string{
				"src/appl.fol":                  "value := 1;\n",
				"srclib/ffi/library.fol":        "_ co.lang.library = {}\n",
				"srclib/ffi/native/library.fol": "_ co.lang.library = {}\n",
			},
			want: "nested library.fol",
		},
		{
			name: "lib present but empty",
			entries: map[string]string{
				"src/appl.fol": "value := 1;\n",
				"lib/":         "",
			},
			want: "lib/ is present but empty",
		},
		{
			name: "lib holds source",
			entries: map[string]string{
				"src/appl.fol":   "value := 1;\n",
				"lib/vendor.fol": "_ co.lang.struct = {}\n",
			},
			want: "is FoLang source",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			layout := ValidateLayout(writeLayout(t, test.entries))
			got := findingsText(layout)
			if !strings.Contains(got, test.want) {
				t.Fatalf("findings did not mention %q:\n%s", test.want, got)
			}
		})
	}
}

// A package path is measured from the DOMAIN root, because src/ and srclib/<slot>/ are
// filesystem domains rather than packages and contribute no namespace component.
func TestPackagePathsAreRelativeToTheOwningDomain(t *testing.T) {
	root := writeLayout(t, map[string]string{
		"src/appl.fol":                         "value := 1;\n",
		"src/hr/employee/Employee.fol":         "_ co.lang.struct = {}\n",
		"srclib/ffi/library.fol":               "_ co.lang.library = {}\n",
		"srclib/ffi/native/marshal/M.unit.fol": "_ co.lang.unit = {}\n",
	})

	tests := []struct {
		base        string
		packagePath string
		atRoot      bool
		domain      string
		slot        string
	}{
		{"appl.fol", "", true, SourceDomain, ""},
		{"Employee.fol", "hr.employee", false, SourceDomain, ""},
		{"library.fol", "", true, SourceLibraryDomain, "ffi"},
		{"M.unit.fol", "native.marshal", false, SourceLibraryDomain, "ffi"},
	}

	discovered, err := Discover(filepath.Join(root, "src", "appl.fol"), root)
	if err != nil {
		t.Fatal(err)
	}
	byBase := map[string]File{}
	for _, f := range discovered.Files {
		byBase[f.Base] = f
	}

	for _, test := range tests {
		t.Run(test.base, func(t *testing.T) {
			f, ok := byBase[test.base]
			if !ok {
				t.Fatalf("%s was not discovered", test.base)
			}
			if f.PackagePath != test.packagePath || f.AtRoot != test.atRoot {
				t.Errorf("package path/atRoot = %q/%v, want %q/%v", f.PackagePath, f.AtRoot, test.packagePath, test.atRoot)
			}
			if f.Domain != test.domain || f.LibrarySlot != test.slot {
				t.Errorf("domain/slot = %q/%q, want %q/%q", f.Domain, f.LibrarySlot, test.domain, test.slot)
			}
		})
	}
}
