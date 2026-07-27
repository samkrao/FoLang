package scanlex_test

import (
	"os"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestMain(m *testing.M) {
	// Prevent os.Exit on tokenizer/parser errors so panics can be recovered.
	foerrors.GenPanic = true
	os.Exit(m.Run())
}

// tokenize is a convenience wrapper. fo-lang requires at least one preceding
// newline before the first identifier in the stream; this helper adds it.
func tokenize(src string) []scanlex.Token {
	return scanlex.Tokenize("\n"+src, "test")
}

// meaningful filters out the trailing EOF token.
func meaningful(tokens []scanlex.Token) []scanlex.Token {
	out := make([]scanlex.Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Kind != scanlex.EOF {
			out = append(out, t)
		}
	}
	return out
}

func assertKind(t *testing.T, tok scanlex.Token, kind scanlex.TokenKind) {
	t.Helper()
	if tok.Kind != kind {
		t.Errorf("expected kind %s (%d), got %s (%d) (value=%q)",
			scanlex.TokenKindString(kind), kind,
			scanlex.TokenKindString(tok.Kind), tok.Kind, tok.Value)
	}
}

func assertKindValue(t *testing.T, tok scanlex.Token, kind scanlex.TokenKind, value string) {
	t.Helper()
	assertKind(t, tok, kind)
	if tok.Value != value {
		t.Errorf("expected value %q, got %q", value, tok.Value)
	}
}

func findKind(toks []scanlex.Token, kind scanlex.TokenKind) (scanlex.Token, bool) {
	for _, t := range toks {
		if t.Kind == kind {
			return t, true
		}
	}
	return scanlex.Token{}, false
}

// ---------------------------------------------------------------------------
// Numbers
// ---------------------------------------------------------------------------

func TestTokenize_IntegerLiteral(t *testing.T) {
	toks := meaningful(tokenize("42"))
	if len(toks) == 0 {
		t.Fatal("expected at least 1 token")
	}
	assertKind(t, toks[0], scanlex.NUMBER)
	if toks[0].Value != "42" {
		t.Errorf("expected value %q, got %q", "42", toks[0].Value)
	}
}

func TestTokenize_FloatLiteral(t *testing.T) {
	toks := meaningful(tokenize("3.14"))
	if len(toks) == 0 {
		t.Fatal("expected at least 1 token")
	}
	assertKind(t, toks[0], scanlex.NUMBER)
}

func TestTokenize_NegativeNumber(t *testing.T) {
	toks := meaningful(tokenize("-7"))
	if len(toks) == 0 {
		t.Fatal("expected at least 1 token")
	}
	found := false
	for _, tk := range toks {
		if tk.Kind == scanlex.NUMBER {
			found = true
		}
	}
	if !found {
		t.Error("expected NUMBER token in '-7'")
	}
}

// ---------------------------------------------------------------------------
// String literals
// ---------------------------------------------------------------------------

func TestTokenize_StringLiteral(t *testing.T) {
	toks := meaningful(tokenize(`"hello world"`))
	if len(toks) == 0 {
		t.Fatal("expected at least 1 token")
	}
	assertKind(t, toks[0], scanlex.STRING)
}

func TestTokenize_StringLiteral_Empty(t *testing.T) {
	toks := meaningful(tokenize(`""`))
	if len(toks) == 0 {
		t.Fatal("expected at least 1 token for empty string")
	}
	assertKind(t, toks[0], scanlex.STRING)
}

// ---------------------------------------------------------------------------
// Keywords
// ---------------------------------------------------------------------------

func TestTokenize_Keywords(t *testing.T) {
	cases := []string{"co", "let", "this", "for", "forall", "self"}
	for _, kw := range cases {
		toks := meaningful(tokenize(kw))
		if len(toks) == 0 {
			t.Errorf("no token for keyword %q", kw)
			continue
		}
		assertKind(t, toks[0], scanlex.KEYWORD)
		if toks[0].Value != kw {
			t.Errorf("keyword %q: expected value %q, got %q", kw, kw, toks[0].Value)
		}
	}
}

// ---------------------------------------------------------------------------
// Identifiers (get _fo suffix after folding)
// ---------------------------------------------------------------------------

func TestTokenize_Identifier_Kind(t *testing.T) {
	toks := meaningful(tokenize("someVar co.lang.int;"))
	if len(toks) == 0 {
		t.Fatal("expected tokens for 'someVar co.lang.int;'")
	}
	// First token is the user identifier
	assertKind(t, toks[0], scanlex.IDENTIFIER)
}

func TestTokenize_Identifier_FoSuffix(t *testing.T) {
	// The tokenizer appends _fo to user-defined identifiers after folding.
	toks := meaningful(tokenize("myVar co.lang.int;"))
	if len(toks) == 0 {
		t.Fatal("expected tokens")
	}
	if toks[0].Value != "myVar_fo" {
		t.Errorf("expected identifier value %q, got %q", "myVar_fo", toks[0].Value)
	}
}

func TestTokenize_BuiltInType_CoLangInt(t *testing.T) {
	// "co.lang.int" folds into a single BUILT_IN_TYPE token.
	toks := meaningful(tokenize("x co.lang.int;"))
	if len(toks) == 0 {
		t.Fatal("expected tokens")
	}
	_, found := findKind(toks, scanlex.BUILT_IN_TYPE)
	if !found {
		t.Errorf("expected BUILT_IN_TYPE token in 'x co.lang.int;', got %v", toks)
	}
}

func TestTokenize_BuiltInType_Value(t *testing.T) {
	toks := meaningful(tokenize("x co.lang.string;"))
	tok, found := findKind(toks, scanlex.BUILT_IN_TYPE)
	if !found {
		t.Fatal("expected BUILT_IN_TYPE token")
	}
	if tok.Value != "co.lang.string" {
		t.Errorf("expected BUILT_IN_TYPE value %q, got %q", "co.lang.string", tok.Value)
	}
}

// ---------------------------------------------------------------------------
// Assignment operators
// ---------------------------------------------------------------------------

func TestTokenize_Assignment(t *testing.T) {
	toks := meaningful(tokenize("x = 1"))
	_, found := findKind(toks, scanlex.ASSIGNMENT)
	if !found {
		t.Error("expected ASSIGNMENT token")
	}
}

func TestTokenize_WalrusOperator(t *testing.T) {
	toks := meaningful(tokenize("x := 42"))
	tok, found := findKind(toks, scanlex.WALRUS)
	if !found {
		t.Fatal("expected WALRUS token for ':='")
	}
	if tok.Value != ":=" {
		t.Errorf("expected WALRUS value %q, got %q", ":=", tok.Value)
	}
}

func TestTokenize_ColonWalrus(t *testing.T) {
	toks := meaningful(tokenize("x ::= 42"))
	_, found := findKind(toks, scanlex.COLON_WALRUS)
	if !found {
		t.Error("expected COLON_WALRUS token for '::='")
	}
}

func TestTokenize_QEQ(t *testing.T) {
	toks := meaningful(tokenize("x ?= y"))
	_, found := findKind(toks, scanlex.QEQ)
	if !found {
		t.Error("expected QEQ token for '?='")
	}
}

// ---------------------------------------------------------------------------
// Arithmetic operators
// ---------------------------------------------------------------------------

func TestTokenize_ArithmeticOperators(t *testing.T) {
	cases := []struct {
		src  string
		kind scanlex.TokenKind
	}{
		{"1+2", scanlex.PLUS},
		{"1-2", scanlex.MINUS},
		{"1*2", scanlex.STAR},   // * is STAR (39), not MUL (36)
		{"1/2", scanlex.SLASH},  // / is SLASH (35), not FORWARD_SLASH (56)
		{"1%2", scanlex.PERCENT},
	}
	for _, c := range cases {
		toks := meaningful(tokenize(c.src))
		_, found := findKind(toks, c.kind)
		if !found {
			t.Errorf("expected %s token in %q", scanlex.TokenKindString(c.kind), c.src)
		}
	}
}

// ---------------------------------------------------------------------------
// Comparison operators
// ---------------------------------------------------------------------------

func TestTokenize_ComparisonOperators(t *testing.T) {
	cases := []struct {
		src  string
		kind scanlex.TokenKind
	}{
		{"a==b", scanlex.EQUALS},
		{"a!=b", scanlex.NOT_EQUALS},
		{"a<=b", scanlex.LESS_EQUALS},
		{"a>=b", scanlex.GREATER_EQUALS},
	}
	for _, c := range cases {
		toks := meaningful(tokenize(c.src))
		_, found := findKind(toks, c.kind)
		if !found {
			t.Errorf("expected %s token in %q", scanlex.TokenKindString(c.kind), c.src)
		}
	}
}

// ---------------------------------------------------------------------------
// Arrow operators
// ---------------------------------------------------------------------------

func TestTokenize_Arrow(t *testing.T) {
	toks := meaningful(tokenize("(a)->(b)"))
	_, found := findKind(toks, scanlex.ARROW)
	if !found {
		t.Error("expected ARROW token in '(a)->(b)'")
	}
}

func TestTokenize_PipelineArrow_SingleToken(t *testing.T) {
	// ->> must tokenize as a single MINUS_ARROW_GT, not -> + >
	toks := meaningful(tokenize("->>"))
	if len(toks) != 1 {
		t.Fatalf("expected 1 token for '->>', got %d: %v", len(toks), toks)
	}
	assertKindValue(t, toks[0], scanlex.MINUS_ARROW_GT, "->>")
}

func TestTokenize_PipelineArrow_InExpr(t *testing.T) {
	// f ->> g should be: IDENTIFIER MINUS_ARROW_GT IDENTIFIER
	toks := meaningful(tokenize("let f ->> g"))
	_, found := findKind(toks, scanlex.MINUS_ARROW_GT)
	if !found {
		t.Errorf("expected MINUS_ARROW_GT in 'f ->> g', tokens: %v", toks)
	}
}

func TestTokenize_PipelineArrow_NotConfusedWithArrow(t *testing.T) {
	// "->" should NOT produce MINUS_ARROW_GT
	toks := meaningful(tokenize("(a)->(b)"))
	_, found := findKind(toks, scanlex.MINUS_ARROW_GT)
	if found {
		t.Error("did not expect MINUS_ARROW_GT in '(a)->(b)'")
	}
}

func TestTokenize_LeftArrow(t *testing.T) {
	// <- for comprehension / channel receive
	toks := meaningful(tokenize("let x <- y"))
	_, found := findKind(toks, scanlex.LEFT_ARROW)
	if !found {
		t.Errorf("expected LEFT_ARROW token in 'x <- y', tokens: %v", toks)
	}
}

func TestTokenize_BidirArrow_SingleToken(t *testing.T) {
	// <-> must tokenize as a single BIDIR_ARROW, not <- + >
	toks := meaningful(tokenize("let x <-> y"))
	_, found := findKind(toks, scanlex.BIDIR_ARROW)
	if !found {
		t.Fatalf("expected BIDIR_ARROW in 'x <-> y', tokens: %v", toks)
	}
	// Must NOT also produce a LEFT_ARROW from the same source
	tok, _ := findKind(toks, scanlex.LEFT_ARROW)
	if tok.Value == "<-" {
		t.Error("BIDIR_ARROW '<->' was split into LEFT_ARROW + '>'")
	}
}

func TestTokenize_BidirArrow_NotConfusedWithLeftArrow(t *testing.T) {
	// "a <- b" must still produce LEFT_ARROW
	toks := meaningful(tokenize("let a <- b"))
	_, found := findKind(toks, scanlex.LEFT_ARROW)
	if !found {
		t.Error("expected LEFT_ARROW in 'a <- b'")
	}
	_, foundBidir := findKind(toks, scanlex.BIDIR_ARROW)
	if foundBidir {
		t.Error("did not expect BIDIR_ARROW in 'a <- b'")
	}
}

// ---------------------------------------------------------------------------
// Bind variables ($, $0, $1)
// ---------------------------------------------------------------------------

func TestTokenize_BindVarBare(t *testing.T) {
	toks := meaningful(tokenize("$"))
	if len(toks) == 0 {
		t.Fatal("expected BIND_VAR token for '$'")
	}
	assertKind(t, toks[0], scanlex.BIND_VAR)
}

func TestTokenize_BindVarIndexed(t *testing.T) {
	cases := []string{"$0", "$1", "$2"}
	for _, src := range cases {
		toks := meaningful(tokenize(src))
		if len(toks) == 0 {
			t.Fatalf("expected BIND_VAR token for %q", src)
		}
		assertKind(t, toks[0], scanlex.BIND_VAR)
		if toks[0].Value != src {
			t.Errorf("expected bind-var value %q, got %q", src, toks[0].Value)
		}
	}
}

// ---------------------------------------------------------------------------
// Punctuation
// ---------------------------------------------------------------------------

func TestTokenize_Punctuation(t *testing.T) {
	cases := []struct {
		src  string
		kind scanlex.TokenKind
	}{
		{";", scanlex.SEMI_COLON},
		{",", scanlex.COMMA},
		{"(", scanlex.OPEN_PAREN},
		{")", scanlex.CLOSE_PAREN},
		{"{", scanlex.OPEN_CURLY},
		{"}", scanlex.CLOSE_CURLY},
		{"[", scanlex.OPEN_BRACKET},
		{"]", scanlex.CLOSE_BRACKET},
		{":", scanlex.COLON},
	}
	for _, c := range cases {
		toks := meaningful(tokenize(c.src))
		if len(toks) == 0 {
			t.Errorf("no token for %q", c.src)
			continue
		}
		assertKind(t, toks[0], c.kind)
	}
}

// ---------------------------------------------------------------------------
// Range operators
// ---------------------------------------------------------------------------

func TestTokenize_DotDot(t *testing.T) {
	toks := meaningful(tokenize("1..10"))
	_, found := findKind(toks, scanlex.DOT_DOT)
	if !found {
		t.Error("expected DOT_DOT token in '1..10'")
	}
}

func TestTokenize_DotDotExclusive(t *testing.T) {
	toks := meaningful(tokenize("0..<10"))
	_, found := findKind(toks, scanlex.DOT_DOT_LT)
	if !found {
		t.Error("expected DOT_DOT_LT token in '0..<10'")
	}
}

// ---------------------------------------------------------------------------
// Walrus declaration — integration token sequence
// ---------------------------------------------------------------------------

func TestTokenize_WalrusDecl_Sequence(t *testing.T) {
	// "let someVal := 42;" — check sequence: KEYWORD IDENTIFIER WALRUS NUMBER SEMI_COLON
	toks := meaningful(tokenize("let someVal := 42;"))
	if len(toks) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d: %v", len(toks), toks)
	}
	assertKind(t, toks[0], scanlex.KEYWORD)
	if toks[0].Value != "let" {
		t.Errorf("expected 'let', got %q", toks[0].Value)
	}
	assertKind(t, toks[1], scanlex.IDENTIFIER)
	assertKind(t, toks[2], scanlex.WALRUS)
	assertKind(t, toks[3], scanlex.NUMBER)
}

func TestTokenize_VarDecl_Sequence(t *testing.T) {
	// "someInt co.lang.int = 42;" — IDENTIFIER BUILT_IN_TYPE ASSIGNMENT NUMBER SEMI_COLON
	toks := meaningful(tokenize("someInt co.lang.int = 42;"))
	if len(toks) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d: %v", len(toks), toks)
	}
	assertKind(t, toks[0], scanlex.IDENTIFIER)
	assertKind(t, toks[1], scanlex.BUILT_IN_TYPE)
}

func TestTokenize_FunctionArrow_Sequence(t *testing.T) {
	// "(a)->(b)" — should contain ARROW but not MINUS_ARROW_GT
	toks := meaningful(tokenize("(a)->(b)"))
	_, hasArrow := findKind(toks, scanlex.ARROW)
	if !hasArrow {
		t.Error("expected ARROW in '(a)->(b)'")
	}
}

// ---------------------------------------------------------------------------
// Token count sanity
// ---------------------------------------------------------------------------

func TestTokenize_EmptySource(t *testing.T) {
	// An empty source (only newline) produces 0 tokens — the tokenizer
	// strips the newline in cleanupLB and emits no EOF for empty input.
	toks := scanlex.Tokenize("\n", "test")
	for _, tk := range toks {
		if tk.Kind != scanlex.EOF {
			t.Errorf("expected no meaningful tokens for empty source, got kind=%d value=%q", tk.Kind, tk.Value)
		}
	}
}

func TestTokenize_NumberAlone_NoPrefix(t *testing.T) {
	// Numbers don't go through symbolHandler so they don't need a \n prefix.
	toks := scanlex.Tokenize("99", "test")
	found := false
	for _, tk := range toks {
		if tk.Kind == scanlex.NUMBER {
			found = true
		}
	}
	if !found {
		t.Error("expected NUMBER token for '99' without prefix")
	}
}
