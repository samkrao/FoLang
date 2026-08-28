package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProjectTree lays out a project and returns its root.
func writeProjectTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relative, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// parseProjectDiagnostics parses a tree and returns its diagnostics as one string.
func parseProjectDiagnostics(t *testing.T, files map[string]string) string {
	t.Helper()
	_, diagnostics, err := ParseProject(writeProjectTree(t, files))
	if err != nil {
		t.Fatalf("parsing the project: %v", err)
	}
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Error())
	}
	return strings.Join(messages, "\n")
}

// A layout violation is a fact about the project ParseProject was asked for, so
// it belongs in the diagnostics ParseProject returns.
//
// Discovery computes these findings and they used to stop there, which let the
// public parser hand back a tree for a project whose shape it had already found
// invalid. Only the CLI's separate preparation pass reported anything, so a
// caller using ParseProject directly was told nothing.
func TestParseProjectReturnsLayoutDiagnostics(t *testing.T) {
	for name, test := range map[string]struct {
		files map[string]string
		want  string
	}{
		"both structural surfaces": {
			want: "contains both appl.fol and component.fol",
			files: map[string]string{
				"fol-conf.yaml":     "project: demo\n",
				"src/appl.fol":      "total co.lang.int = 1;\n",
				"src/component.fol": "_ co.lang.component = {\n}",
			},
		},
		"a stray file directly in src/": {
			want: "occurs directly in src/",
			files: map[string]string{
				"fol-conf.yaml": "project: demo\n",
				"src/appl.fol":  "total co.lang.int = 1;\n",
				"src/notes.txt": "hello\n",
			},
		},
		"a component kind with no surface": {
			want: "has no component.fol",
			files: map[string]string{
				"fol-conf.yaml":                    "project: demo\n",
				"src/appl.fol":                     "total co.lang.int = 1;\n",
				"components/application/pkg/A.fol": "_ co.lang.struct = {\n}",
			},
		},
		"unrelated content in lib/": {
			want: "is not a compiled .folenc artifact",
			files: map[string]string{
				"fol-conf.yaml":  "project: demo\n",
				"src/appl.fol":   "total co.lang.int = 1;\n",
				"lib/notes.txt":  "hello\n",
				"lib/dep.folenc": "",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := parseProjectDiagnostics(t, test.files); !strings.Contains(got, test.want) {
				t.Fatalf("ParseProject did not report %q:\n%s", test.want, got)
			}
		})
	}
}

// A valid project reports nothing, so the propagation above cannot be satisfied
// by reporting indiscriminately.
func TestParseProjectReportsNothingForAValidLayout(t *testing.T) {
	got := parseProjectDiagnostics(t, map[string]string{
		"fol-conf.yaml":                "project: demo\n",
		"src/appl.fol":                 "total co.lang.int = 1;\n",
		"src/hr/employee/Employee.fol": "_ co.lang.struct = {\n}",
		"lib/dep.folenc":               "",
	})
	if got != "" {
		t.Fatalf("a valid project reported diagnostics:\n%s", got)
	}
}

// WHICH components a standalone library may own turns on the exposure model
// written inside src/component.fol, which a filesystem walk cannot read. Only a
// projected APPLICATION library keeps the components/operators/ exception;
// a packaged, native or dynamicvmrt library may hold no components/ tree at all.
func TestParseProjectEnforcesStandaloneComponentRestrictions(t *testing.T) {
	const operatorSurface = "_ co.lang.component = {\n}"

	for name, test := range map[string]struct {
		surface string
		kind    string
		want    string
	}{
		"a packaged library may own no component": {
			surface: "_ co.lang.component = {\n    @co.dap.export(\n        packages={\n            hr.employee={recurse=true}\n        }\n    )\n}",
			kind:    "operators",
			want:    "a standalone packaged library may not contain components/operators",
		},
		"a native library may own no component": {
			surface: "@co.dap.library(type=native)\n_ co.lang.component = {\n}",
			kind:    "operators",
			want:    "a standalone native library may not contain components/operators",
		},
		"a projected application library may not own an ordinary component": {
			surface: "@co.dap.library\n_ co.lang.component = {\n}",
			kind:    "application",
			want:    "a projected application library permits only components/operators",
		},
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{
				"fol-conf.yaml":                              "project: demo\n",
				"src/component.fol":                          test.surface,
				"src/hr/employee/Employee.fol":               "_ co.lang.struct = {\n}",
				"components/" + test.kind + "/component.fol": operatorSurface,
			}
			if got := parseProjectDiagnostics(t, files); !strings.Contains(got, test.want) {
				t.Fatalf("ParseProject did not report %q:\n%s", test.want, got)
			}
		})
	}
}

// The one permitted combination stays silent.
func TestAProjectedApplicationLibraryMayOwnTheOperatorComponent(t *testing.T) {
	got := parseProjectDiagnostics(t, map[string]string{
		"fol-conf.yaml":                      "project: demo\n",
		"src/component.fol":                  "@co.dap.library\n_ co.lang.component = {\n}",
		"src/hr/employee/Employee.fol":       "_ co.lang.struct = {\n}",
		"components/operators/component.fol": "_ co.lang.component = {\n}",
	})
	if strings.Contains(got, "components/operators") {
		t.Fatalf("the permitted operator component was reported:\n%s", got)
	}
}
