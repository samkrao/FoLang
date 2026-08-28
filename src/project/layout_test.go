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
			entries: map[string]string{"src/component.fol": "_ co.lang.component = {}\n"},
			kind:    KindStandaloneLibrary,
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
				"src/appl.fol":      "value := 1;\n",
				"src/component.fol": "_ co.lang.component = {}\n",
			},
			want: "contains both",
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

// A package path is measured from the DOMAIN root, because src/ is a filesystem
// domain rather than a package and contributes no namespace component.
func TestPackagePathsAreRelativeToTheOwningDomain(t *testing.T) {
	root := writeLayout(t, map[string]string{
		"src/appl.fol":                 "value := 1;\n",
		"src/hr/employee/Employee.fol": "_ co.lang.struct = {}\n",
	})

	tests := []struct {
		base        string
		packagePath string
		atRoot      bool
		domain      string
	}{
		{"appl.fol", "", true, SourceDomain},
		{"Employee.fol", "hr.employee", false, SourceDomain},
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
			if f.Domain != test.domain {
				t.Errorf("domain = %q, want %q", f.Domain, test.domain)
			}
		})
	}
}

// src/ holds one of exactly two structural surfaces: appl.fol for an application,
// component.fol for a standalone library under either exposure model
// (docs/language-ref.md, "Project Layout"). `library.fol` was a withdrawn spelling
// that appears nowhere in the reference, and validating against it rejected the
// very form the reference defines.
func TestSourceDomainAcceptsTheComponentSurface(t *testing.T) {
	root := writeLayout(t, map[string]string{
		"src/component.fol": "_ co.lang.component = {}\n",
	})
	layout := ValidateLayout(root)
	if len(layout.Findings) != 0 {
		t.Fatalf("a standalone library was rejected:\n%s", findingsText(layout))
	}
	if layout.Kind != KindStandaloneLibrary {
		t.Errorf("project kind = %v, want KindStandaloneLibrary", layout.Kind)
	}
	if layout.LibrarySurface == "" {
		t.Error("the library surface path was not recorded")
	}
}

// library.fol carries no structural meaning now, so a project holding only it has
// no surface at all rather than a library one.
func TestSourceDomainNoLongerRecognizesLibraryFol(t *testing.T) {
	root := writeLayout(t, map[string]string{
		"src/library.fol": "_ co.lang.component = {}\n",
	})
	layout := ValidateLayout(root)
	if layout.Kind == KindStandaloneLibrary {
		t.Error("library.fol was still read as a structural library surface")
	}
	if !strings.Contains(findingsText(layout), "no structural surface") {
		t.Errorf("expected a missing-surface finding:\n%s", findingsText(layout))
	}
}

// The layout rules that are about DIRECTORIES rather than files.
//
// Discovery only ever sees .fol files, so an empty components/, an unknown
// component kind, a component holding no surface, a package directory with
// nothing under it, and a stray directory in lib/ are all invisible to it. They
// are stated as layout rules for exactly that reason
// (docs/language-ref.md, "Project Layout").
func TestDirectoryShapedViolationsAreReported(t *testing.T) {
	for _, test := range []struct {
		name    string
		entries map[string]string
		want    string
	}{
		{
			name: "a file directly in src/ that is not a surface",
			entries: map[string]string{
				"src/appl.fol":  "value := 1;\n",
				"src/notes.txt": "hello\n",
			},
			want: "occurs directly in src/",
		},
		{
			name: "a package directory with no source at any depth",
			entries: map[string]string{
				"src/appl.fol":          "value := 1;\n",
				"src/hr/empty/.gitkeep": "",
			},
			want: "holding no FoLang source at any depth",
		},
		{
			name: "components/ present but empty",
			entries: map[string]string{
				"src/appl.fol": "value := 1;\n",
				"components/":  "",
			},
			want: "components/ is present but empty",
		},
		{
			name: "an unknown component kind",
			entries: map[string]string{
				"src/appl.fol":                     "value := 1;\n",
				"components/helpers/component.fol": "_ co.lang.component = {}\n",
			},
			want: "is not a standardized component kind",
		},
		{
			name: "a component kind with no surface",
			entries: map[string]string{
				"src/appl.fol":                     "value := 1;\n",
				"components/application/pkg/A.fol": "_ co.lang.struct = {}\n",
			},
			want: "has no component.fol",
		},
		{
			name: "the operator component holding a package directory",
			entries: map[string]string{
				"src/appl.fol":                       "value := 1;\n",
				"components/operators/component.fol": "_ co.lang.component = {}\n",
				"components/operators/pkg/A.fol":     "_ co.lang.struct = {}\n",
			},
			want: "the operator component permits none",
		},
		{
			name: "a directory inside lib/",
			entries: map[string]string{
				"src/appl.fol":            "value := 1;\n",
				"lib/vendor/inner.folenc": "",
			},
			want: "is a directory",
		},
		{
			name: "unrelated content inside lib/",
			entries: map[string]string{
				"src/appl.fol":   "value := 1;\n",
				"lib/notes.txt":  "hello\n",
				"lib/dep.folenc": "",
			},
			want: "is not a compiled .folenc artifact",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			layout := ValidateLayout(writeLayout(t, test.entries))
			got := findingsText(layout)
			if !strings.Contains(got, test.want) {
				t.Fatalf("findings did not mention %q:\n%s", test.want, got)
			}
		})
	}
}

// An intermediate package that only groups subpackages is ordinary — src/hr/
// holding just src/hr/employee/ is the reference's own Package Identity example.
func TestAPackageGroupingOnlySubpackagesIsValid(t *testing.T) {
	layout := ValidateLayout(writeLayout(t, map[string]string{
		"src/appl.fol":                 "value := 1;\n",
		"src/hr/employee/Employee.fol": "_ co.lang.struct = {}\n",
		"lib/dep.folenc":               "",
	}))
	if len(layout.Findings) != 0 {
		t.Fatalf("a grouping package was reported:\n%s", findingsText(layout))
	}
}

// A component kind holds exactly one direct source file, and it is component.fol.
// Implementation source belongs in a descendant package directory, where it is
// component-private.
func TestASecondSourceFileBesideAComponentSurfaceIsReported(t *testing.T) {
	layout := ValidateLayout(writeLayout(t, map[string]string{
		"src/appl.fol":                    "value := 1;\n",
		"components/native/component.fol": "_ co.lang.component = {}\n",
		"components/native/Extra.fol":     "_ co.lang.struct = {}\n",
	}))
	if !strings.Contains(findingsText(layout), "sits beside component.fol") {
		t.Fatalf("a second direct source file was accepted:\n%s", findingsText(layout))
	}
}

// The same file one level down is ordinary: a descendant package directory is
// exactly where a component's implementation source belongs.
func TestSourceInAComponentPackageDirectoryIsValid(t *testing.T) {
	layout := ValidateLayout(writeLayout(t, map[string]string{
		"src/appl.fol":                        "value := 1;\n",
		"components/native/component.fol":     "_ co.lang.component = {}\n",
		"components/native/marshal/Extra.fol": "_ co.lang.struct = {}\n",
	}))
	if len(layout.Findings) != 0 {
		t.Fatalf("component package source was reported:\n%s", findingsText(layout))
	}
}
