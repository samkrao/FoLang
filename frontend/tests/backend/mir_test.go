// Package backend_test tests the fo-lang MIR (mid-level IR) node types defined
// in the backend package. These tests focus on node construction, field access,
// and the Node interface contract without exercising the full compilation pipeline.
package backend_test

import (
	"os"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/backend"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

func TestMain(m *testing.M) {
	foerrors.GenPanic = true
	os.Exit(m.Run())
}

// uid creates a minimal UIDNode for test fixtures.
func uid(id string) *backend.UIDNode {
	return &backend.UIDNode{ID: id, UID: id}
}

// ---------------------------------------------------------------------------
// UIDNode
// ---------------------------------------------------------------------------

func TestUIDNode_Fields(t *testing.T) {
	u := backend.UIDNode{ID: "abc", UID: "uid-abc", Provided: true}
	if u.ID != "abc" {
		t.Errorf("expected ID %q, got %q", "abc", u.ID)
	}
	if u.UID != "uid-abc" {
		t.Errorf("expected UID %q, got %q", "uid-abc", u.UID)
	}
	if !u.Provided {
		t.Error("expected Provided=true")
	}
}

// ---------------------------------------------------------------------------
// BlockNode
// ---------------------------------------------------------------------------

func TestBlockNode_NodeInterface(t *testing.T) {
	// BlockNode.GetType() returns b.Type_ (not NodeType); set it explicitly.
	b := backend.BlockNode{
		UID:   uid("block-1"),
		Type_: "block",
	}
	var _ backend.Node = b // compile-time check
	if b.GetUID() != "block-1" {
		t.Errorf("expected GetUID %q, got %q", "block-1", b.GetUID())
	}
	if b.GetType() != "block" {
		t.Errorf("expected GetType 'block', got %q", b.GetType())
	}
}

func TestBlockNode_NodeSlice(t *testing.T) {
	child := backend.BuiltInStmtNode{
		UID:      uid("c1"),
		Value:    "stmt",
		NodeType: helpers.GetType(backend.BuiltInStmtNode{}),
		Type_:    "builtin",
	}
	b := backend.BlockNode{
		UID:   uid("blk"),
		Node_: []backend.Node{child},
	}
	if len(b.Node_) != 1 {
		t.Fatalf("expected 1 node, got %d", len(b.Node_))
	}
	if b.Node_[0].GetType() != "builtin" {
		t.Errorf("expected node type 'builtin', got %q", b.Node_[0].GetType())
	}
}

// ---------------------------------------------------------------------------
// BuiltInStmtNode
// ---------------------------------------------------------------------------

func TestBuiltInStmtNode_GetType(t *testing.T) {
	n := backend.BuiltInStmtNode{
		UID:      uid("bis1"),
		Value:    "print",
		Type_:    "io.print",
		NodeType: helpers.GetType(backend.BuiltInStmtNode{}),
	}
	if n.GetType() != "io.print" {
		t.Errorf("expected GetType 'io.print', got %q", n.GetType())
	}
}

func TestBuiltInStmtNode_GetId_Empty(t *testing.T) {
	n := backend.BuiltInStmtNode{UID: uid("bis2")}
	if n.GetId() != "" {
		t.Errorf("expected empty GetId for BuiltInStmtNode, got %q", n.GetId())
	}
}

// ---------------------------------------------------------------------------
// PackageNode
// ---------------------------------------------------------------------------

func TestPackageNode_Fields(t *testing.T) {
	pkg := backend.PackageNode{
		UID:      uid("pkg1"),
		Name:     "mypackage",
		NodeType: helpers.GetType(backend.PackageNode{}),
	}
	if pkg.Name != "mypackage" {
		t.Errorf("expected name 'mypackage', got %q", pkg.Name)
	}
	if pkg.GetUID() != "pkg1" {
		t.Errorf("expected uid 'pkg1', got %q", pkg.GetUID())
	}
}

// ---------------------------------------------------------------------------
// DeclNode (variable declaration)
// ---------------------------------------------------------------------------

func TestDeclNode_Fields(t *testing.T) {
	v := backend.DeclNode{
		UID:          uid("var1"),
		VarName:      "x",
		HasInitValue: true,
		NodeType:     helpers.GetType(backend.DeclNode{}),
	}
	if v.VarName != "x" {
		t.Errorf("expected VarName 'x', got %q", v.VarName)
	}
	if !v.HasInitValue {
		t.Error("expected HasInitValue=true")
	}
}

func TestDeclNode_LocalBinding(t *testing.T) {
	v := backend.DeclNode{UID: uid("v2"), VarName: "y", LocalBinding: true}
	if !v.LocalBinding {
		t.Error("expected LocalBinding=true")
	}
}

// ---------------------------------------------------------------------------
// SymbolNode
// ---------------------------------------------------------------------------

func TestSymbolNode_Value(t *testing.T) {
	sym := backend.SymbolNode{
		UID:      uid("sym1"),
		Value:    "mySymbol",
		NodeType: helpers.GetType(backend.SymbolNode{}),
	}
	if sym.Value != "mySymbol" {
		t.Errorf("expected Value 'mySymbol', got %q", sym.Value)
	}
}

// ---------------------------------------------------------------------------
// PtrNode — fat pointer kind and meta
// ---------------------------------------------------------------------------

func TestPtrNode_SinglePointer(t *testing.T) {
	sym := backend.SymbolNode{UID: uid("t"), Value: "int"}
	ptr := backend.PtrNode{
		UID:     uid("ptr1"),
		Count_:  1,
		VarName: "p",
		Type_:   sym,
	}
	if ptr.Count_ != 1 {
		t.Errorf("expected Count_=1, got %d", ptr.Count_)
	}
}

func TestPtrNode_FatPointer_Kind(t *testing.T) {
	ptr := backend.PtrNode{
		UID:         uid("fp1"),
		VarName:     "buf",
		PointerKind: "slice",
	}
	if ptr.PointerKind != "slice" {
		t.Errorf("expected PointerKind 'slice', got %q", ptr.PointerKind)
	}
}

func TestPtrNode_FatPointer_Meta(t *testing.T) {
	ptr := backend.PtrNode{
		UID:         uid("fp2"),
		VarName:     "obj",
		PointerKind: "trait",
		Meta:        map[string]string{"vtab": "MyTrait"},
	}
	if ptr.Meta["vtab"] != "MyTrait" {
		t.Errorf("expected Meta[vtab]='MyTrait', got %q", ptr.Meta["vtab"])
	}
}

func TestPtrNode_FatPointer_AllKinds(t *testing.T) {
	kinds := []string{"slice", "region", "trait", "weak", "shared", "unique"}
	for _, kind := range kinds {
		ptr := backend.PtrNode{UID: uid("fp-" + kind), PointerKind: kind}
		if ptr.PointerKind != kind {
			t.Errorf("expected PointerKind %q, got %q", kind, ptr.PointerKind)
		}
	}
}

// ---------------------------------------------------------------------------
// BindVariableNode (new in session 3)
// ---------------------------------------------------------------------------

func TestBindVariableNode_BareBindVar(t *testing.T) {
	b := backend.BindVariableNode{
		UID:      uid("bv0"),
		Name:     "$",
		Index:    -1,
		NodeType: helpers.GetType(backend.BindVariableNode{}),
	}
	if b.Name != "$" {
		t.Errorf("expected Name '$', got %q", b.Name)
	}
	if b.Index != -1 {
		t.Errorf("expected Index -1 for bare '$', got %d", b.Index)
	}
}

func TestBindVariableNode_IndexedBindVar(t *testing.T) {
	cases := []struct {
		name  string
		index int
	}{
		{"$0", 0},
		{"$1", 1},
		{"$5", 5},
	}
	for _, c := range cases {
		b := backend.BindVariableNode{
			UID:   uid("bv-" + c.name),
			Name:  c.name,
			Index: c.index,
		}
		if b.Name != c.name {
			t.Errorf("expected Name %q, got %q", c.name, b.Name)
		}
		if b.Index != c.index {
			t.Errorf("expected Index %d for %q, got %d", c.index, c.name, b.Index)
		}
	}
}

func TestBindVariableNode_NodeInterface(t *testing.T) {
	b := backend.BindVariableNode{
		UID:      uid("bv-iface"),
		Name:     "$0",
		Index:    0,
		NodeType: helpers.GetType(backend.BindVariableNode{}),
	}
	var _ backend.Node = b // compile-time interface check
	if b.GetUID() != "bv-iface" {
		t.Errorf("expected GetUID 'bv-iface', got %q", b.GetUID())
	}
	if !strings.Contains(b.GetType(), "BindVariableNode") {
		t.Errorf("expected GetType to contain 'BindVariableNode', got %q", b.GetType())
	}
}

// ---------------------------------------------------------------------------
// MatcherDeclNode (new in session 3)
// ---------------------------------------------------------------------------

func TestMatcherDeclNode_Fields(t *testing.T) {
	fn := backend.FunctionDeclarationNode{
		UID:      uid("fn1"),
		Name:     "match_fn",
		NodeType: helpers.GetType(backend.FunctionDeclarationNode{}),
	}
	m := backend.MatcherDeclNode{
		UID:                     uid("md1"),
		FunctionDeclarationNode: fn,
		TypeParam:               "T",
		NodeType:                helpers.GetType(backend.MatcherDeclNode{}),
	}
	if m.TypeParam != "T" {
		t.Errorf("expected TypeParam 'T', got %q", m.TypeParam)
	}
	if m.FunctionDeclarationNode.Name != "match_fn" {
		t.Errorf("expected embedded Name 'match_fn', got %q", m.FunctionDeclarationNode.Name)
	}
}

func TestMatcherDeclNode_NodeInterface(t *testing.T) {
	m := backend.MatcherDeclNode{
		UID:      uid("md-iface"),
		NodeType: helpers.GetType(backend.MatcherDeclNode{}),
	}
	var _ backend.Node = m
	if m.GetUID() != "md-iface" {
		t.Errorf("expected GetUID 'md-iface', got %q", m.GetUID())
	}
}

// ---------------------------------------------------------------------------
// DependentTypeNode (new in session 3)
// ---------------------------------------------------------------------------

func TestDependentTypeNode_Fields(t *testing.T) {
	base := backend.SymbolNode{UID: uid("base"), Value: "Array"}
	expr := backend.VarAccessNode{UID: uid("len"), VarName: "n"}
	d := backend.DependentTypeNode{
		UID:      uid("dt1"),
		BaseType: base,
		Expr:     expr,
		Kind:     "length",
		NodeType: helpers.GetType(backend.DependentTypeNode{}),
	}
	if d.Kind != "length" {
		t.Errorf("expected Kind 'length', got %q", d.Kind)
	}
	if d.BaseType == nil {
		t.Error("expected non-nil BaseType")
	}
	if d.Expr == nil {
		t.Error("expected non-nil Expr")
	}
}

func TestDependentTypeNode_NilExpr(t *testing.T) {
	d := backend.DependentTypeNode{
		UID:      uid("dt-nil"),
		BaseType: backend.SymbolNode{UID: uid("t"), Value: "T"},
		Expr:     nil,
		Kind:     "param",
	}
	if d.Expr != nil {
		t.Error("expected Expr=nil for no-expr dependent type")
	}
}

func TestDependentTypeNode_NodeInterface(t *testing.T) {
	d := backend.DependentTypeNode{
		UID:      uid("dt-iface"),
		NodeType: helpers.GetType(backend.DependentTypeNode{}),
	}
	var _ backend.Node = d
	if d.GetUID() != "dt-iface" {
		t.Errorf("expected GetUID 'dt-iface', got %q", d.GetUID())
	}
}

// ---------------------------------------------------------------------------
// ForComprehensionNode
// ---------------------------------------------------------------------------

func TestForComprehensionNode_Fields(t *testing.T) {
	src := backend.VarAccessNode{UID: uid("src"), VarName: "myList"}
	yield := backend.BuiltInStmtNode{UID: uid("y"), Value: "x*2"}
	fc := backend.ForComprehensionNode{
		UID:     uid("fc1"),
		Binding: "x",
		Source:  src,
		Yield:   yield,
	}
	if fc.Binding != "x" {
		t.Errorf("expected Binding 'x', got %q", fc.Binding)
	}
}

// ---------------------------------------------------------------------------
// ForAllNode
// ---------------------------------------------------------------------------

func TestForAllNode_TypeParams(t *testing.T) {
	fa := backend.ForAllNode{
		UID:        uid("fa1"),
		TypeParams: []string{"T", "U"},
	}
	if len(fa.TypeParams) != 2 {
		t.Fatalf("expected 2 type params, got %d", len(fa.TypeParams))
	}
	if fa.TypeParams[0] != "T" {
		t.Errorf("expected TypeParams[0]='T', got %q", fa.TypeParams[0])
	}
}

// ---------------------------------------------------------------------------
// LambdaNode
// ---------------------------------------------------------------------------

func TestLambdaNode_Fields(t *testing.T) {
	param := backend.ParameterNode{
		UID:   uid("p1"),
		Name_: "x",
	}
	body := backend.BlockNode{UID: uid("body")}
	lambda := backend.LambdaNode{
		UID:        uid("lam1"),
		Parameters: []backend.ParameterNode{param},
		Body:       body,
	}
	if len(lambda.Parameters) != 1 {
		t.Fatalf("expected 1 param, got %d", len(lambda.Parameters))
	}
	if lambda.Parameters[0].Name_ != "x" {
		t.Errorf("expected param Name_ 'x', got %q", lambda.Parameters[0].Name_)
	}
}

// ---------------------------------------------------------------------------
// PatternMatchNode / PatternCaseNode
// ---------------------------------------------------------------------------

func TestPatternMatchNode_Cases(t *testing.T) {
	sym1 := backend.SymbolNode{UID: uid("p1"), Value: "Some(x)"}
	sym2 := backend.SymbolNode{UID: uid("p2"), Value: "None"}
	c1 := backend.PatternCaseNode{UID: uid("c1"), Pattern: sym1, Binding: "x"}
	c2 := backend.PatternCaseNode{UID: uid("c2"), Pattern: sym2}
	pm := backend.PatternMatchNode{
		UID:   uid("pm1"),
		Cases: []backend.PatternCaseNode{c1, c2},
	}
	if len(pm.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(pm.Cases))
	}
	if pm.Cases[0].Pattern.(backend.SymbolNode).Value != "Some(x)" {
		t.Errorf("expected first case pattern value 'Some(x)', got %q", pm.Cases[0].Pattern.(backend.SymbolNode).Value)
	}
}

// ---------------------------------------------------------------------------
// helpers.GetType — verify type names contain type info
// ---------------------------------------------------------------------------

func TestHelpers_GetType_NodeTypes(t *testing.T) {
	cases := []struct {
		node     backend.Node
		wantSubs string
	}{
		{backend.BlockNode{UID: uid("x")}, "BlockNode"},
		{backend.BindVariableNode{UID: uid("x")}, "BindVariableNode"},
		{backend.MatcherDeclNode{UID: uid("x")}, "MatcherDeclNode"},
		{backend.DependentTypeNode{UID: uid("x")}, "DependentTypeNode"},
		{backend.PtrNode{UID: uid("x")}, "PtrNode"},
	}
	for _, c := range cases {
		typeName := helpers.GetType(c.node)
		if !strings.Contains(typeName, c.wantSubs) {
			t.Errorf("GetType(%T) = %q, expected to contain %q", c.node, typeName, c.wantSubs)
		}
	}
}
