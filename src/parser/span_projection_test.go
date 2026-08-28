package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/project"
)

// The artifact keeps a source region on every node and drops the frontend's
// bookkeeping from it.
//
// Span itself is not optional: Appendix B.7 lists it on the AST nodes it defines,
// and a diagnostic instance must carry a primary source span — a rule that binds
// the backend, which can still reject an accepted program with
// UnsupportedBackendFeature and has to say where.
//
// Its CONTENTS are unspecified, and two fields serve only the frontend. Ftxt is
// the whole source line, held twice per node and again for every nested node on
// that line; it is read only by the caret-underline in helpers/error.go while the
// frontend still has the file open. Idx is Pos under another name.
func TestProjectedSpanKeepsLocationAndDropsFrontendBookkeeping(t *testing.T) {
	start := helpers.NewPosition(11, 3, 5, 11, "demo.unit.fol", "    total co.lang.int = 1;", false)
	end := helpers.NewPosition(19, 3, 13, 19, "demo.unit.fol", "    total co.lang.int = 1;", false)
	node := ast.BreakStmt{NodeName: "BreakStmt", Span: ast.Span{Start: *start, End: *end}}

	encoded, err := json.Marshal(projectAST(node, nil, true))
	if err != nil {
		t.Fatal(err)
	}
	var tree map[string]any
	if err := json.Unmarshal(encoded, &tree); err != nil {
		t.Fatal(err)
	}
	span, ok := tree["Span"].(map[string]any)
	if !ok {
		t.Fatalf("node lost its Span: %s", encoded)
	}

	for _, edge := range []string{"Start", "End"} {
		position, ok := span[edge].(map[string]any)
		if !ok {
			t.Fatalf("Span.%s is not an object: %v", edge, span[edge])
		}
		for _, kept := range []string{"Ln", "Col", "Pos", "Fn"} {
			if _, present := position[kept]; !present {
				t.Errorf("Span.%s dropped %s, which a consumer needs to report against source", edge, kept)
			}
		}
		for _, dropped := range []string{"Ftxt", "Idx"} {
			if _, present := position[dropped]; present {
				t.Errorf("Span.%s still carries %s, which no reader of the artifact can use", edge, dropped)
			}
		}
	}

	if got := span["Start"].(map[string]any)["Ln"]; got != float64(3) {
		t.Errorf("Start.Ln = %v, want 3", got)
	}
	if got := span["End"].(map[string]any)["Col"]; got != float64(13) {
		t.Errorf("End.Col = %v, want 13", got)
	}
	if got := span["Start"].(map[string]any)["Fn"]; got != "demo.unit.fol" {
		t.Errorf("Start.Fn = %v, want the source file name", got)
	}
}

// The trim is a projection concern only. The in-memory position keeps Ftxt so the
// frontend's own diagnostics can still underline the offending text.
func TestTrimDoesNotTouchTheInMemoryPosition(t *testing.T) {
	source := `_ co.lang.unit = {
    run()->() = {
        x co.lang.int = 1;
    }
}`
	root, p := parsePackageSource(t, source, "demo.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("diags: %v", p.diags)
	}
	unit, ok := root.(ast.PackageStmt)
	if !ok || len(unit.Body) == 0 {
		t.Fatalf("unexpected root %T", root)
	}
	span := unit.Body[0].(ast.TypeDeclarationStmt).GetSpan()
	if span.Start.Ftxt == "" {
		t.Error("the parsed tree lost Ftxt; frontend diagnostics need it to draw the caret")
	}
}

// `span: off` in fol-conf.yaml drops the source region from the artifact — the
// analogue of building without debug information. It is the whole Span that goes,
// not its contents, and nothing else about the tree changes.
func TestSpanConfigControlsTheArtifact(t *testing.T) {
	for _, test := range []struct {
		config    string
		wantSpans bool
	}{
		{"project: demo\n", true},
		{"project: demo\nspan: on\n", true},
		{"project: demo\nspan: off\n", false},
		{"project: demo\nspan: off   # no debug information\n", false},
	} {
		artifact := compileWithConfig(t, test.config)
		if got := strings.Contains(artifact, `"Span"`); got != test.wantSpans {
			t.Errorf("config %q: artifact carries spans = %v, want %v", test.config, got, test.wantSpans)
		}

		// Whatever the setting, the tree itself is intact and still names its nodes.
		var envelope map[string]any
		if err := json.Unmarshal([]byte(artifact), &envelope); err != nil {
			t.Fatalf("config %q: %v", test.config, err)
		}
		root, ok := envelope["AST"].(map[string]any)
		if !ok {
			t.Fatalf("config %q: artifact has no AST root", test.config)
		}
		if root["NodeName"] != "ProjectStmt" {
			t.Errorf("config %q: AST root = %v, want ProjectStmt", test.config, root["NodeName"])
		}
		if _, present := root["EntryStmt"]; !present {
			t.Errorf("config %q: turning spans off disturbed the rest of the node", test.config)
		}
	}
}

// An unusable setting stops the compilation rather than quietly choosing one.
func TestSpanConfigRejectsAnUnknownSetting(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, project.MarkerFilename), "project: demo\nspan: maybe\n")
	write(t, filepath.Join(root, "src", "appl.fol"), "total co.lang.int = 1;\n")

	_, _, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root)
	if err == nil {
		t.Fatal("an unrecognized span setting compiled anyway")
	}
	if !strings.Contains(err.Error(), "span") {
		t.Errorf("error does not name the offending key: %v", err)
	}
}

// compileWithConfig builds a one-file project under config and returns its artifact.
func compileWithConfig(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, project.MarkerFilename), config)
	write(t, filepath.Join(root, project.BackendConfigFilename), `{"protocol":"folang-plugin/1.0","hir_schema":"folang-hir/1","wire":"json","runtime_operations":"folang-runtime-operations/1"}`)
	write(t, filepath.Join(root, "src", "appl.fol"), "total co.lang.int = 1;\n")

	if _, _, _, _, err := Focmain(filepath.Join(root, "src", "appl.fol"), false, false, "", false, root); err != nil {
		t.Fatalf("config %q: %v", config, err)
	}
	content, err := os.ReadFile(filepath.Join(root, project.BuildDomain, "appl"+astArtifactExtension))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
