package scanlex

import "testing"

func TestTokenStreamPeekBuffersOnlyRequestedLookahead(t *testing.T) {
	const source = "alpha beta gamma delta"
	stream := NewTokenStream([]byte(source), "test.fol")

	if got := stream.Peek(0); got.Value != "alpha" {
		t.Fatalf("Peek(0) = %q, want alpha", got.Value)
	}
	if got := len(stream.buffer); got != 1 {
		t.Fatalf("Peek(0) buffered %d tokens, want 1", got)
	}
	if got := stream.lexer.pos; got != len("alpha") {
		t.Fatalf("Peek(0) scanned through byte %d, want %d", got, len("alpha"))
	}

	if got := stream.Peek(2); got.Value != "beta" {
		t.Fatalf("Peek(2) = %q, want beta", got.Value)
	}
	if got := len(stream.buffer); got != 3 {
		t.Fatalf("Peek(2) buffered %d tokens, want 3", got)
	}
	if got := stream.lexer.pos; got >= len(source) {
		t.Fatalf("Peek(2) eagerly scanned the complete source through byte %d", got)
	}
}

func TestTokenStreamNextBuffersOneToken(t *testing.T) {
	stream := NewTokenStream([]byte("alpha beta"), "test.fol")

	if got := stream.Next(); got.Value != "alpha" {
		t.Fatalf("first Next() = %q, want alpha", got.Value)
	}
	if len(stream.buffer) != 0 {
		t.Fatalf("Next() retained %d unrequested tokens", len(stream.buffer))
	}
	if got := stream.lexer.pos; got != len("alpha") {
		t.Fatalf("Next() scanned through byte %d, want %d", got, len("alpha"))
	}

	if got := stream.Next(); got.Kind != SPACE || got.Value != " " {
		t.Fatalf("second Next() = (%v, %q), want SPACE", got.Kind, got.Value)
	}
	if got := stream.Next(); got.Value != "beta" {
		t.Fatalf("third Next() = %q, want beta", got.Value)
	}
	if got := stream.Next(); got.Kind != EOF {
		t.Fatalf("fourth Next() kind = %v, want EOF", got.Kind)
	}
	if got := stream.Next(); got.Kind != EOF {
		t.Fatalf("EOF was not stable: subsequent kind = %v", got.Kind)
	}
	if got := len(stream.history); got != 3 {
		t.Fatalf("EOF calls changed consumed history length to %d, want 3", got)
	}
}

func TestTokenStreamPeekPastEOFStopsScanning(t *testing.T) {
	stream := NewTokenStream([]byte("only"), "test.fol")

	if got := stream.Peek(10); got.Kind != EOF {
		t.Fatalf("Peek past EOF kind = %v, want EOF", got.Kind)
	}
	if !stream.exhausted {
		t.Fatal("stream did not remember reaching EOF")
	}
	if got := len(stream.buffer); got != 1 {
		t.Fatalf("stream buffered %d source tokens, want 1", got)
	}
	if got := stream.Peek(10); got.Kind != EOF {
		t.Fatalf("repeated Peek past EOF kind = %v, want EOF", got.Kind)
	}
}

func TestTokenStreamAtEOFSupportsIterationWithoutConsuming(t *testing.T) {
	stream := NewTokenStream([]byte("alpha beta"), "test.fol")
	var values []string
	for !stream.AtEOF() {
		values = append(values, stream.Next().Value)
	}

	want := []string{"alpha", " ", "beta"}
	if len(values) != len(want) {
		t.Fatalf("iteration returned %d tokens, want %d: %v", len(values), len(want), values)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("token %d = %q, want %q", i, values[i], want[i])
		}
	}
	if len(stream.history) != len(want) {
		t.Fatalf("AtEOF changed history: got %d consumed tokens, want %d", len(stream.history), len(want))
	}
	if !stream.AtEOF() || stream.Next().Kind != EOF {
		t.Fatal("EOF was not stable after iteration")
	}
}

func TestTokenStreamPreviousTracksOnlyConsumedTokens(t *testing.T) {
	stream := NewTokenStream([]byte("alpha beta gamma"), "test.fol")

	if _, ok := stream.Previous(1); ok {
		t.Fatal("Previous(1) succeeded before a token was consumed")
	}
	if got := stream.Peek(2); got.Value != "beta" {
		t.Fatalf("Peek(2) = %q, want beta", got.Value)
	}
	if _, ok := stream.Previous(1); ok {
		t.Fatal("Peek unexpectedly added a token to consumed history")
	}

	if got := stream.Next(); got.Value != "alpha" {
		t.Fatalf("first Next() = %q, want alpha", got.Value)
	}
	if got, ok := stream.Previous(1); !ok || got.Value != "alpha" {
		t.Fatalf("Previous(1) = (%q, %v), want (alpha, true)", got.Value, ok)
	}
	if _, ok := stream.Previous(2); ok {
		t.Fatal("Previous(2) succeeded with only one consumed token")
	}

	if got := stream.Next(); got.Kind != SPACE {
		t.Fatalf("second Next() kind = %v, want SPACE", got.Kind)
	}
	if got := stream.Next(); got.Value != "beta" {
		t.Fatalf("third Next() = %q, want beta", got.Value)
	}
	if got, ok := stream.Previous(1); !ok || got.Value != "beta" {
		t.Fatalf("Previous(1) = (%q, %v), want (beta, true)", got.Value, ok)
	}
	if got, ok := stream.Previous(2); !ok || got.Kind != SPACE {
		t.Fatalf("Previous(2) = (%v, %v), want (SPACE, true)", got.Kind, ok)
	}
	if got, ok := stream.Previous(3); !ok || got.Value != "alpha" {
		t.Fatalf("Previous(3) = (%q, %v), want (alpha, true)", got.Value, ok)
	}
}

func TestTokenStreamPreviousRejectsNonPositiveLookbehind(t *testing.T) {
	stream := NewTokenStream(nil, "test.fol")
	defer func() {
		if recover() == nil {
			t.Fatal("Previous(0) did not panic")
		}
	}()
	stream.Previous(0)
}

func TestScannerPreservesTokenSubKinds(t *testing.T) {
	stream := NewTokenStream([]byte("_ @@new $ $2 +"), "test.fol")
	want := []struct {
		kind    TokenKind
		subKind SubKind
		value   string
	}{
		{BUILT_INS_FOL, DISCARD_WILD_CHAR, "_"},
		{BUILT_INS_FOL, SPECIAL_METHOD, "@@new"},
		{BUILT_INS_FOL, SIGILS, "$"},
		{BUILT_INS_FOL, BINDVAR, "$2"},
		{BUILT_INS_FOL, OPERATORS, "+"},
	}

	for i, expected := range want {
		got := stream.Next()
		for got.Kind == SPACE {
			got = stream.Next()
		}
		if got.Kind != expected.kind || got.SubKind != expected.subKind || got.Value != expected.value {
			t.Fatalf("token %d = (%v, %v, %q), want (%v, %v, %q)",
				i, got.Kind, got.SubKind, got.Value,
				expected.kind, expected.subKind, expected.value)
		}
	}
}

func TestFoldTokensClassifiesParserFacingLexemes(t *testing.T) {
	custom := NewCustomOperatorsWithSpecs([]OperatorSpec{{Symbol: "%%", Fixity: "infix"}})
	stream := NewTokenStreamWithOperators([]byte(
		"@co.dap.generic co.int co.out.println () 'outer: $ $12 @@custom _ + \u222a %% alpha.beta @pkg.meta",
	), "test.fol", custom)

	want := []struct {
		kind    TokenKind
		subKind SubKind
		value   string
	}{
		{BUILT_INS_FOL, DIRECTIVES, "@co.dap.generic"},
		{BUILT_INS_FOL, STATEMENT_EXPR, "co.int"},
		{BUILT_INS_FOL, METHOD, "co.out.println"},
		{OPEN_PAREN, NA, "("},
		{CLOSE_PAREN, NA, ")"},
		{BUILT_INS_FOL, LABEL_LITERAL, "'outer"},
		{BUILT_INS_FOL, OPERATORS, ":"},
		{BUILT_INS_FOL, SIGILS, "$"},
		{BUILT_INS_FOL, BINDVAR, "$12"},
		{BUILT_INS_FOL, SPECIAL_METHOD, "@@custom"},
		{BUILT_INS_FOL, DISCARD_WILD_CHAR, "_"},
		{BUILT_INS_FOL, OPERATORS, "+"},
		{BUILT_INS_FOL, OPERATORS, "\u222a"},
		{CUSTOM_OPERATOR, NA, "%%"},
		{COMPOSITE_IDENTIFIER, NA, "alpha.beta"},
		{CUSTOM_ANNOT_DECOR, NA, "@pkg.meta"},
		{EOF, NA, "EOF"},
	}

	for i, expected := range want {
		got := nextNonWhitespace(stream)
		if got.Kind != expected.kind || got.SubKind != expected.subKind || got.Value != expected.value {
			t.Fatalf("token %d = (%v, %v, %q), want (%v, %v, %q)",
				i, got.Kind, got.SubKind, got.Value,
				expected.kind, expected.subKind, expected.value)
		}
	}
}

func TestFoldTokensRecognizesOperatorSourceAndPreservesFullSpan(t *testing.T) {
	stream := NewTokenStream([]byte("co.operator.fixity.infix"), "test.fol")
	got := stream.Next()
	if got.Kind != BUILT_INS_FOL || got.SubKind != OPERATOR_SOURCE || got.Value != "co.operator.fixity.infix" {
		t.Fatalf("operator source = (%v, %v, %q), want BUILT_INS_FOL/OPERATOR_SOURCE",
			got.Kind, got.SubKind, got.Value)
	}
	if got.StartPos == nil || got.EndPos == nil || got.StartPos.Idx != 0 || got.EndPos.Idx != len(got.Value) {
		t.Fatalf("folded span = %#v..%#v, want byte range 0..%d", got.StartPos, got.EndPos, len(got.Value))
	}
}

func TestFoldTokensClassifiesThisReceiverPaths(t *testing.T) {
	stream := NewTokenStream([]byte("this.field this.kind () this->parents this->classes[0] this->parent::new() this -> parents"), "test.fol")
	want := []struct {
		kind    TokenKind
		subKind SubKind
		value   string
	}{
		{BUILT_INS_FOL, STATEMENT_EXPR, "this.field"},
		{BUILT_INS_FOL, METHOD, "this.kind"},
		{OPEN_PAREN, NA, "("},
		{CLOSE_PAREN, NA, ")"},
		{BUILT_INS_FOL, STATEMENT_EXPR, "this->parents"},
		{BUILT_INS_FOL, STATEMENT_EXPR, "this->classes"},
		{OPEN_BRACKET, NA, "["},
		{NUMBER, NA, "0"},
		{CLOSE_BRACKET, NA, "]"},
		{BUILT_INS_FOL, STATEMENT_EXPR, "this->parent"},
		{BUILT_INS_FOL, OPERATORS, "::"},
		{IDENTIFIER, NA, "new"},
		{OPEN_PAREN, NA, "("},
		{CLOSE_PAREN, NA, ")"},
		{KEYWORD, NA, "this"},
		{BUILT_INS_FOL, OPERATORS, "->"},
		{IDENTIFIER, NA, "parents"},
		{EOF, NA, "EOF"},
	}

	for i, expected := range want {
		got := nextNonWhitespace(stream)
		if got.Kind != expected.kind || got.SubKind != expected.subKind || got.Value != expected.value {
			t.Fatalf("token %d = (%v, %v, %q), want (%v, %v, %q)",
				i, got.Kind, got.SubKind, got.Value,
				expected.kind, expected.subKind, expected.value)
		}
	}
}

func TestFoldTokensDoesNotJoinAcrossComment(t *testing.T) {
	stream := NewTokenStream([]byte("alpha/* separator */.beta"), "test.fol")
	first := stream.Next()
	if first.Kind != IDENTIFIER || first.SubKind != NA || first.Value != "alpha" {
		t.Fatalf("first token = (%v, %v, %q), want IDENTIFIER/NA alpha", first.Kind, first.SubKind, first.Value)
	}
	second := stream.Next()
	if second.Kind != BUILT_INS_FOL || second.SubKind != OPERATORS || second.Value != "." {
		t.Fatalf("second token = (%v, %v, %q), want BUILT_INS_FOL/OPERATORS dot", second.Kind, second.SubKind, second.Value)
	}
}

func TestFoldTokensUsesNAForNonBuiltins(t *testing.T) {
	stream := NewTokenStream([]byte("name 'label 42\n"), "test.fol")
	for {
		token := stream.Next()
		if token.Kind == EOF {
			break
		}
		if token.Kind != BUILT_INS_FOL && token.SubKind != NA {
			t.Fatalf("non-builtin token (%v, %q) has subkind %v, want NA", token.Kind, token.Value, token.SubKind)
		}
	}
}

func TestFoldTokensKeepsInvalidLexemesUnknownAndWhole(t *testing.T) {
	stream := NewTokenStream([]byte("$0 @@bad_ @co..thing bad__name"), "test.fol")
	want := []string{"$0", "@@bad_", "@co..thing", "bad__name"}
	for i, value := range want {
		token := nextNonWhitespace(stream)
		if token.Kind != UNKNOWN || token.SubKind != NA || token.Value != value {
			t.Fatalf("invalid token %d = (%v, %v, %q), want UNKNOWN/NA %q",
				i, token.Kind, token.SubKind, token.Value, value)
		}
	}
}

func nextNonWhitespace(stream *TokenStream) Token {
	for {
		token := stream.Next()
		if token.Kind != SPACE && token.Kind != NEWLINE {
			return token
		}
	}
}

func TestScannerCollectsSpacesAndNewlinesButSkipsComments(t *testing.T) {
	stream := NewTokenStream([]byte("a \t// ignored\r\n\f b"), "test.fol")
	want := []struct {
		kind  TokenKind
		value string
	}{
		{IDENTIFIER, "a"},
		{SPACE, " \t"},
		{NEWLINE, "\r\n"},
		{SPACE, "\f "},
		{IDENTIFIER, "b"},
		{EOF, "EOF"},
	}

	for i, expected := range want {
		got := stream.Next()
		if got.Kind != expected.kind || got.Value != expected.value {
			t.Fatalf("token %d = (%v, %q), want (%v, %q)",
				i, got.Kind, got.Value, expected.kind, expected.value)
		}
		if got.Kind == NEWLINE && got.EndPos.Ln != 2 {
			t.Fatalf("newline end line = %d, want 2", got.EndPos.Ln)
		}
	}
}

func TestScannerEOFUsesActualSourcePosition(t *testing.T) {
	const source = "alpha\nbeta"
	stream := NewTokenStream([]byte(source), "test.fol")

	for stream.Next().Kind != EOF {
	}
	eof := stream.Next()
	if eof.StartPos == nil || eof.EndPos == nil {
		t.Fatal("EOF has nil source positions")
	}
	if eof.StartPos.Idx != len(source) || eof.EndPos.Idx != len(source) {
		t.Fatalf("EOF indexes = (%d, %d), want (%d, %d)",
			eof.StartPos.Idx, eof.EndPos.Idx, len(source), len(source))
	}
	if eof.StartPos.Ln != 2 || eof.EndPos.Ln != 2 {
		t.Fatalf("EOF lines = (%d, %d), want (2, 2)", eof.StartPos.Ln, eof.EndPos.Ln)
	}
	if eof.StartPos.Fn != "test.fol" || eof.EndPos.Fn != "test.fol" {
		t.Fatalf("EOF filenames = (%q, %q), want test.fol", eof.StartPos.Fn, eof.EndPos.Fn)
	}
}
