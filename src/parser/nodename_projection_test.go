package parser

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

// The serialized tree names each node from its Go type, not from the NodeName
// field the construction site wrote.
//
// The field exists so that any marshalling of a node shows its form, and it is
// filled in by hand at every construction site — the same way Span is, with the
// same hazard: a site added later and stamped wrongly, or not at all, still
// compiles and still parses. TestEveryNodeCarriesItsNodeName catches that for
// trees the parser builds, but a node built anywhere else reaches the artifact
// unchecked. Deriving the name during projection closes that off: the dump is
// correct whatever the field holds, so a reader debugging a tree can trust it.
func TestProjectedNodeNameComesFromTheType(t *testing.T) {
	for name, node := range map[string]ast.Stmt{
		"never stamped":   ast.IfStmt{},
		"stamped wrongly": ast.IfStmt{NodeName: "ForeachStmt"},
		"stamped rightly": ast.IfStmt{NodeName: "IfStmt"},
	} {
		encoded, err := json.Marshal(projectAST(node, nil))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(string(encoded), `"NodeName":"IfStmt"`) {
			t.Errorf("%s: projected tree does not name IfStmt: %s", name, encoded)
		}
		if strings.Contains(string(encoded), "ForeachStmt") {
			t.Errorf("%s: the field's stale value reached the tree: %s", name, encoded)
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
