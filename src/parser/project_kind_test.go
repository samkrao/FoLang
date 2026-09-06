package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

// ProjectStmt.ProjectKind names what the project IS.
//
// Appendix B.7.1 lists Kind on ProjectStatement — "effective project/library kind
// according to the project model" — and src/ holds exactly one primary structural
// surface that decides it. src/component.fol then picks between the two mutually
// exclusive standalone exposure models from its own metadata: @co.dap.library
// makes it projected, its absence makes it packaged
// (docs/language-ref.md, "Form Exclusivity").
//
// components/ and lib/ contribute no kind. A project-local component compiles into
// its owning application and never produces its own artifact; lib/ holds compiled
// dependencies that arrive as imported libraries and surface symbols. Neither is
// ever a project root.
func TestProjectKindComesFromTheStructuralSurface(t *testing.T) {
	for name, test := range map[string]struct {
		files       map[string]string
		want        string
		wantLibrary bool
	}{
		"application entry": {
			want:        ast.ProjectKindApplication,
			wantLibrary: false,
			files: map[string]string{
				"fol-conf.yaml": "project: demo\n",
				"src/appl.fol":  "total co.lang.int = 1;\n",
			},
		},
		"projected library": {
			want:        ast.ProjectKindLibrary,
			wantLibrary: true,
			files: map[string]string{
				"fol-conf.yaml": "project: demo\n",
				"src/component.fol": `@co.dap.library
_ co.lang.component = {
}`,
			},
		},
		"packaged library": {
			want:        ast.ProjectKindPackagedLibrary,
			wantLibrary: true,
			files: map[string]string{
				"fol-conf.yaml": "project: demo\n",
				"src/component.fol": `_ co.lang.component = {
    @co.dap.export(
        packages={
            hr.employee={recurse=true}
        }
    )
}`,
				"src/hr/employee/Employee.fol": "_ co.lang.struct = {\n}",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := writeTree(t, test.files)
			stmt, diagnostics, err := ParseProject(root)
			if err != nil {
				t.Fatalf("parsing the project: %v", err)
			}
			if len(diagnostics) != 0 {
				t.Fatalf("project produced diagnostics: %v", diagnostics)
			}
			project, isProject := stmt.(ast.ProjectStmt)
			if !isProject {
				t.Fatalf("project root = %T, want ast.ProjectStmt", stmt)
			}
			if project.ProjectKind != test.want {
				t.Errorf("ProjectKind = %q, want %q", project.ProjectKind, test.want)
			}
			// Both exposure models are STANDALONE libraries, so a packaged
			// library answers IsLibrary exactly as a projected one does.
			if project.IsLibrary != test.wantLibrary {
				t.Errorf("IsLibrary = %v, want %v", project.IsLibrary, test.wantLibrary)
			}
			folContext := project.FolangSymbols.RootFolContext()
			if test.want == ast.ProjectKindPackagedLibrary {
				if len(folContext.ChildCtxIds) != 0 || folContext.ExportedPackages["hr.employee"] == "" {
					t.Errorf("packaged publication = children %v, exports %v", folContext.ChildCtxIds, folContext.ExportedPackages)
				}
			} else if len(folContext.ExportedPackages) != 0 {
				t.Errorf("non-packaged project unexpectedly exports packages: %v", folContext.ExportedPackages)
			}
		})
	}
}

// The kind is read from the surface, so a project whose entry did not parse has
// none to report. Naming it "application" would put a claim in the artifact that
// nothing in the source established.
func TestProjectKindIsEmptyWithoutARecognizedSurface(t *testing.T) {
	if got := projectKind(nil); got != "" {
		t.Errorf("projectKind(nil) = %q, want the empty kind", got)
	}
	if got := projectKind(ast.BlockStmt{NodeName: "BlockStmt"}); got != "" {
		t.Errorf("projectKind(a non-surface node) = %q, want the empty kind", got)
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
