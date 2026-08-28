package parser

import (
	"encoding/json"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

// Projection preserves an explicitly populated NodeName and uses the concrete
// struct name only as a fallback for an omitted value. Parser coverage tests
// separately reject incorrect non-empty names in parser-produced trees.
func TestProjectionDerivesOnlyMissingNodeNames(t *testing.T) {
	for name, test := range map[string]struct {
		node ast.Stmt
		want string
	}{
		"never stamped":   {node: ast.IfStmt{}, want: "IfStmt"},
		"stamped wrongly": {node: ast.IfStmt{NodeName: "ForeachStmt"}, want: "ForeachStmt"},
		"stamped rightly": {node: ast.IfStmt{NodeName: "IfStmt"}, want: "IfStmt"},
	} {
		encoded, err := json.Marshal(projectAST(test.node, nil))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var tree map[string]any
		if err := json.Unmarshal(encoded, &tree); err != nil {
			t.Fatalf("%s: decode projection: %v", name, err)
		}
		if got := tree["NodeName"]; got != test.want {
			t.Errorf("%s: NodeName = %v, want %q", name, got, test.want)
		}
	}
}

// Only AST nodes are named. A span, a symbol record and a parser-side helper are
// structs in the same tree and must not grow a NodeName they never declared.
func TestProjectionNamesOnlyASTNodes(t *testing.T) {
	encoded, err := json.Marshal(projectAST(ast.IfStmt{NodeName: "IfStmt"}, nil))
	if err != nil {
		t.Fatal(err)
	}
	var tree map[string]any
	if err := json.Unmarshal(encoded, &tree); err != nil {
		t.Fatal(err)
	}
	span, ok := tree["Span"].(map[string]any)
	if !ok {
		t.Fatalf("Span is not an object: %v", tree["Span"])
	}
	if _, named := span["NodeName"]; named {
		t.Errorf("Span was given a NodeName: %v", span)
	}
}
