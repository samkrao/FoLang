// Package parser_test exercises the fo-lang parser via its public API.
//
// NOTE: parser.Parse(src, name, base, true) (full-parse mode) currently hangs
// on some simple inputs due to a pre-existing loop in parse_code_stmt when
// the parser cannot match a grammar rule and the sync heuristic does not
// advance. Until that is addressed, these tests cover:
//   - Tokenize-only mode (parse=false) via the parser entry point.
//   - Token-sequence assertions that cross-validate the scanner and parser API.
package parser_test

import (
	"os"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/parser"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestMain(m *testing.M) {
	foerrors.GenPanic = true
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseTokensOnly calls parser.Parse with parse=false and returns the token
// slice. The returned AST is always a DummyStmt in this mode.
func parseTokensOnly(t *testing.T, src string) []scanlex.Token {
	t.Helper()
	stmt, toks, ctx, _ := parser.Parse("\n"+src, "test", "test", "", "program", "program", "program", false)
	if _, ok := stmt.(ast.DummyStmt); !ok {
		t.Errorf("expected DummyStmt in tokenize-only mode, got %T", stmt)
	}
	if ctx != nil {
		t.Errorf("expected nil context in tokenize-only mode")
	}
	return toks
}

func findKind(toks []scanlex.Token, kind scanlex.TokenKind) (scanlex.Token, bool) {
	for _, t := range toks {
		if t.Kind == kind {
			return t, true
		}
	}
	return scanlex.Token{}, false
}

func countKind(toks []scanlex.Token, kind scanlex.TokenKind) int {
	n := 0
	for _, t := range toks {
		if t.Kind == kind {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Tokenize-only mode — basic token production
// ---------------------------------------------------------------------------

func TestParseTokensOnly_DummyStmt(t *testing.T) {
	stmt, _, ctx, _ := parser.Parse("\nlet x := 42;", "test", "test", "", "program", "program", "program", false)
	if _, ok := stmt.(ast.DummyStmt); !ok {
		t.Errorf("expected DummyStmt from tokenize-only Parse, got %T", stmt)
	}
	if ctx != nil {
		t.Errorf("expected nil context from tokenize-only Parse")
	}
}

func TestParseTokensOnly_WalrusDecl(t *testing.T) {
	toks := parseTokensOnly(t, "let x := 42;")
	// Must have: KEYWORD("let"), IDENTIFIER, WALRUS, NUMBER, SEMI_COLON
	_, hasLet := findKind(toks, scanlex.KEYWORD)
	if !hasLet {
		t.Error("expected KEYWORD token for 'let'")
	}
	_, hasWalrus := findKind(toks, scanlex.WALRUS)
	if !hasWalrus {
		t.Error("expected WALRUS token ':='")
	}
	_, hasNum := findKind(toks, scanlex.NUMBER)
	if !hasNum {
		t.Error("expected NUMBER token '42'")
	}
}

func TestParseTokensOnly_VarDecl_WithType(t *testing.T) {
	// "someInt co.lang.int = 42;" -> IDENTIFIER BUILT_IN_TYPE ASSIGNMENT NUMBER SEMI_COLON
	toks := parseTokensOnly(t, "someInt co.lang.int = 42;")
	_, hasIdent := findKind(toks, scanlex.IDENTIFIER)
	if !hasIdent {
		t.Error("expected IDENTIFIER token")
	}
	_, hasType := findKind(toks, scanlex.BUILT_IN_TYPE)
	if !hasType {
		t.Error("expected BUILT_IN_TYPE token 'co.lang.int'")
	}
	_, hasAssign := findKind(toks, scanlex.ASSIGNMENT)
	if !hasAssign {
		t.Error("expected ASSIGNMENT token '='")
	}
}

func TestParseTokensOnly_StringVar(t *testing.T) {
	toks := parseTokensOnly(t, `let s := "hello";`)
	_, hasStr := findKind(toks, scanlex.STRING)
	if !hasStr {
		t.Error("expected STRING token")
	}
}

func TestParseTokensOnly_ArrowType(t *testing.T) {
	// Function type annotation: (co.lang.int)->(co.lang.bool)
	toks := parseTokensOnly(t, "f (co.lang.int)->(co.lang.bool) = {}")
	_, hasArrow := findKind(toks, scanlex.ARROW)
	if !hasArrow {
		t.Error("expected ARROW token '->'")
	}
}

func TestParseTokensOnly_DelegateDecl(t *testing.T) {
	toks := parseTokensOnly(t, `@co.dap.delegate someDelegate co.lang.delegate = (a co.lang.int, b co.lang.int)->(co.lang.int);`)
	hasDirective := false
	for _, tok := range toks {
		if tok.Value == "@co.dap.delegate" {
			hasDirective = true
			break
		}
	}
	if !hasDirective {
		t.Error("expected folded token for '@co.dap.delegate'")
	}
	delegateKinds := 0
	for _, tok := range toks {
		if tok.Value == "co.lang.delegate" {
			delegateKinds++
		}
	}
	if delegateKinds != 1 {
		t.Errorf("expected exactly one co.lang.delegate token, got %d", delegateKinds)
	}
}

// ---------------------------------------------------------------------------
// Tokenize-only — new operators added in recent sessions
// ---------------------------------------------------------------------------

func TestParseTokensOnly_PipelineArrow(t *testing.T) {
	toks := parseTokensOnly(t, "let f ->> g")
	_, found := findKind(toks, scanlex.MINUS_ARROW_GT)
	if !found {
		t.Error("expected MINUS_ARROW_GT '->>` token")
	}
}

func TestParseTokensOnly_BidirArrow(t *testing.T) {
	toks := parseTokensOnly(t, "let a <-> b")
	_, found := findKind(toks, scanlex.BIDIR_ARROW)
	if !found {
		t.Error("expected BIDIR_ARROW '<->' token")
	}
}

func TestParseTokensOnly_LeftArrow_ForComprehension(t *testing.T) {
	toks := parseTokensOnly(t, "let x <- xs")
	_, found := findKind(toks, scanlex.LEFT_ARROW)
	if !found {
		t.Error("expected LEFT_ARROW '<-' token")
	}
}

func TestParseTokensOnly_BindVar(t *testing.T) {
	toks := parseTokensOnly(t, "$0 + $1")
	n := countKind(toks, scanlex.BIND_VAR)
	if n != 2 {
		t.Errorf("expected 2 BIND_VAR tokens, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Tokenize-only — keyword tokens
// ---------------------------------------------------------------------------

func TestParseTokensOnly_ForKeyword(t *testing.T) {
	toks := parseTokensOnly(t, "for x in xs {}")
	tok, found := findKind(toks, scanlex.KEYWORD)
	if !found {
		t.Fatal("expected KEYWORD token")
	}
	if tok.Value != "for" {
		t.Errorf("expected keyword 'for', got %q", tok.Value)
	}
}

func TestParseTokensOnly_ForallKeyword(t *testing.T) {
	toks := parseTokensOnly(t, "forall")
	tok, found := findKind(toks, scanlex.KEYWORD)
	if !found {
		t.Fatal("expected KEYWORD for 'forall'")
	}
	if tok.Value != "forall" {
		t.Errorf("expected keyword value 'forall', got %q", tok.Value)
	}
}

// ---------------------------------------------------------------------------
// Tokenize-only — built-in types
// ---------------------------------------------------------------------------

func TestParseTokensOnly_BuiltInTypes(t *testing.T) {
	types := []string{
		"co.lang.int", "co.lang.string", "co.lang.bool",
		"co.lang.float", "co.lang.char",
	}
	for _, tp := range types {
		src := "x " + tp + ";"
		toks := parseTokensOnly(t, src)
		tok, found := findKind(toks, scanlex.BUILT_IN_TYPE)
		if !found {
			t.Errorf("expected BUILT_IN_TYPE for %q", tp)
			continue
		}
		if tok.Value != tp {
			t.Errorf("expected BUILT_IN_TYPE value %q, got %q", tp, tok.Value)
		}
	}
}

// ---------------------------------------------------------------------------
// Tokenize-only — punctuation in declarations
// ---------------------------------------------------------------------------

func TestParseTokensOnly_SemiColon(t *testing.T) {
	toks := parseTokensOnly(t, "x co.lang.int;")
	_, found := findKind(toks, scanlex.SEMI_COLON)
	if !found {
		t.Error("expected SEMI_COLON token")
	}
}

func TestParseTokensOnly_CurlyBraces(t *testing.T) {
	toks := parseTokensOnly(t, "f () -> (co.lang.int) = { }")
	_, hasOpen := findKind(toks, scanlex.OPEN_CURLY)
	_, hasClose := findKind(toks, scanlex.CLOSE_CURLY)
	if !hasOpen {
		t.Error("expected OPEN_CURLY token")
	}
	if !hasClose {
		t.Error("expected CLOSE_CURLY token")
	}
}

// ---------------------------------------------------------------------------
// AST node construction (independent of parser, no parsing needed)
// ---------------------------------------------------------------------------

func TestASTNode_Prog_Empty(t *testing.T) {
	prog := ast.Prog{
		Body:      []ast.Stmt{},
		Name:      "test",
		Package:   "",
		CodeBlock: true,
	}
	if len(prog.Body) != 0 {
		t.Errorf("expected empty Body, got %d", len(prog.Body))
	}
	if !prog.CodeBlock {
		t.Error("expected CodeBlock=true for a code-file Prog")
	}
}

func TestASTNode_DummyStmt(t *testing.T) {
	var s ast.Stmt = ast.DummyStmt{}
	if s == nil {
		t.Error("DummyStmt should satisfy ast.Stmt interface")
	}
}

func TestASTNode_VarDeclarationStmt_Fields(t *testing.T) {
	symb := &symboltable.VarSymbol{}
	symb.HasInitValue = true
	v := ast.VarDeclarationStmt{
		BasicVarStmt: ast.BasicVarStmt{Identifier: "x"},
		Symb:         symb,
	}
	if v.Identifier != "x" {
		t.Errorf("expected identifier 'x', got %q", v.Identifier)
	}
	if !v.IsInitialize() {
		t.Error("expected HasInitValue=true (via IsInitialize)")
	}
}

func TestASTNode_BindVariableExpr_Index(t *testing.T) {
	b0 := ast.BindVariableExpr{Name: "$0", Index: 0}
	b1 := ast.BindVariableExpr{Name: "$1", Index: 1}
	bare := ast.BindVariableExpr{Name: "$", Index: -1}

	if b0.Index != 0 {
		t.Errorf("expected index 0 for '$0', got %d", b0.Index)
	}
	if b1.Index != 1 {
		t.Errorf("expected index 1 for '$1', got %d", b1.Index)
	}
	if bare.Index != -1 {
		t.Errorf("expected index -1 for bare '$', got %d", bare.Index)
	}
}

func TestASTNode_ForComprehensionExpr_Fields(t *testing.T) {
	fc := ast.ForComprehensionExpr{
		Bindings: []ast.ForBinding{{Name: "x"}},
	}
	if len(fc.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(fc.Bindings))
	}
	if fc.Bindings[0].Name != "x" {
		t.Errorf("expected binding name 'x', got %q", fc.Bindings[0].Name)
	}
}
