package parser

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEveryParserMethodIsTraced prevents new parser paths from becoming
// invisible in the entry/exit trace. The tracing implementation is excluded
// because tracing it would recursively invoke itself. cur is the single cursor
// primitive used by the debug tracer and therefore has the same restriction.
func TestEveryParserMethodIsTraced(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate parser source directory")
	}
	dir := filepath.Dir(thisFile)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	excludedFiles := map[string]bool{
		"debug_trace.go":  true,
		"partrace_on.go":  true,
		"partrace_off.go": true,
	}
	for _, path := range files {
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") || excludedFiles[base] {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := goparser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil || function.Name.Name == "cur" {
				continue
			}
			star, ok := function.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			receiver, ok := star.X.(*ast.Ident)
			if !ok || receiver.Name != "parser" {
				continue
			}
			start := fset.Position(function.Body.Lbrace).Offset
			end := fset.Position(function.Body.Rbrace).Offset
			if end > start+300 {
				end = start + 300
			}
			if !strings.Contains(string(source[start:end]), "defer p.traceEnd(p.traceBegin())") {
				t.Errorf("%s: parser method %s is missing entry/exit tracing", base, function.Name.Name)
			}
		}
	}
}
