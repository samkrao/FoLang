package scanlex

import "testing"

func TestUnknownSymbolicRunIsPreservedWhole(t *testing.T) {
	tokens := Tokenize(`++ -- !== *** +- β`, "symbols.fol")
	want := []string{"++", "--", "!==", "***", "+-", "β"}
	if got := tokenValues(tokens); !equalStrings(got, want) {
		t.Fatalf("symbolic runs = %#v, want %#v", got, want)
	}
	for _, token := range tokens {
		if token.Kind != SYMBOLIC_RUN {
			t.Errorf("%q kind = %s, want symbolic run", token.Value, TokenKindString(token.Kind))
		}
	}
}

func TestCustomOperatorMatchesOnlyTheCompleteRun(t *testing.T) {
	custom := NewCustomOperatorsWithSpecs([]OperatorSpec{{Symbol: "<=>", Fixity: "infix"}})

	registered := TokenizeWith(`left <=> right`, "symbols.fol", custom)
	if token := tokenWithValue(t, registered, "<=>"); token.Kind != CUSTOM_OPERATOR {
		t.Fatalf("registered whole run kind = %s, want custom operator", TokenKindString(token.Kind))
	}

	unknown := TokenizeWith(`left <=>+ right`, "symbols.fol", custom)
	if token := tokenWithValue(t, unknown, "<=>+"); token.Kind != SYMBOLIC_RUN {
		t.Fatalf("extended run kind = %s, want symbolic run", TokenKindString(token.Kind))
	}
}

func TestCustomOperatorFixityChecksOriginalBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		source string
		at     int
		spec   OperatorSpec
		want   scanAction
	}{
		{"infix-spaces", `a +- b`, 2, OperatorSpec{Symbol: "+-", Fixity: "infix"}, actionEmit},
		{"infix-comments", `a/*x*/+-/*y*/b`, 6, OperatorSpec{Symbol: "+-", Fixity: "infix"}, actionEmit},
		{"infix-delimiters", `(a)+-(b)`, 3, OperatorSpec{Symbol: "+-", Fixity: "infix"}, actionEmit},
		{"infix-unbounded", `a+-b`, 1, OperatorSpec{Symbol: "+-", Fixity: "infix"}, actionError},
		{"prefix-delimiter", `%%(value)`, 0, OperatorSpec{Symbol: "%%", Fixity: "prefix"}, actionEmit},
		{"prefix-unbounded", `%%value`, 0, OperatorSpec{Symbol: "%%", Fixity: "prefix"}, actionError},
		{"postfix-space", `value %%`, 6, OperatorSpec{Symbol: "%%", Fixity: "postfix"}, actionEmit},
		{"postfix-unbounded", `value%%`, 5, OperatorSpec{Symbol: "%%", Fixity: "postfix"}, actionError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lex := createLexer(tc.source, "symbols.fol")
			lex.custom = NewCustomOperatorsWithSpecs([]OperatorSpec{tc.spec})
			lex.pos = tc.at
			got, ok := lex.scanToken(lex.source[tc.at:])
			if !ok {
				t.Fatal("scanner did not recognize symbolic run")
			}
			if got.action != tc.want {
				t.Fatalf("action = %v, want %v (message %q)", got.action, tc.want, got.message)
			}
			if tc.want == actionEmit && (got.kind != CUSTOM_OPERATOR || got.length != len(tc.spec.Symbol)) {
				t.Fatalf("emitted (%s, length %d), want CUSTOM_OPERATOR(%q)", TokenKindString(got.kind), got.length, tc.spec.Symbol)
			}
		})
	}
}

func TestSymbolTokensRetainOriginalBoundaryMetadata(t *testing.T) {
	tokens := Tokenize(`left ** (right)`, "symbols.fol")
	power := tokenWithValue(t, tokens, "**")
	if !power.BoundaryBefore || !power.BoundaryAfter {
		t.Fatalf("power boundaries = (%v, %v), want both true", power.BoundaryBefore, power.BoundaryAfter)
	}
	commentPower := tokenWithValue(t, Tokenize(`left/*a*/**/*b*/right`, "symbols.fol"), "**")
	if !commentPower.BoundaryBefore || !commentPower.BoundaryAfter {
		t.Fatalf("comment-delimited power boundaries = (%v, %v), want both true", commentPower.BoundaryBefore, commentPower.BoundaryAfter)
	}

	structural := Tokenize(`T->(***)`, "symbols.fol")
	arrow := tokenWithValue(t, structural, "->")
	if arrow.BoundaryBefore || !arrow.BoundaryAfter {
		t.Fatalf("structural arrow boundaries = (%v, %v), want (false, true)", arrow.BoundaryBefore, arrow.BoundaryAfter)
	}
	stars := tokenWithValue(t, structural, "***")
	if stars.Kind != SYMBOLIC_RUN {
		t.Fatalf("pointer-star run kind = %s, want symbolic run", TokenKindString(stars.Kind))
	}
	if !stars.BoundaryBefore || !stars.BoundaryAfter {
		t.Fatalf("pointer-star boundaries = (%v, %v), want both true", stars.BoundaryBefore, stars.BoundaryAfter)
	}
}

func TestPredeclaredGlyphIsAnExpressionOperatorCandidate(t *testing.T) {
	for _, glyph := range []string{"∪", "â", "Ť", "Ṡ", "𝒯"} {
		token := tokenWithValue(t, Tokenize(glyph, "symbols.fol"), glyph)
		if token.Kind != CUSTOM_OPERATOR {
			t.Errorf("predeclared glyph %q kind = %s, want custom operator", glyph, TokenKindString(token.Kind))
		}
	}
}

func TestOperatorOwnershipQueriesDistinguishRegistrations(t *testing.T) {
	for _, spelling := range []string{"+", "**", "∪", "Ť"} {
		if !IsLanguageOwnedOperatorSpelling(spelling) {
			t.Errorf("%q should be language-owned", spelling)
		}
	}
	for _, spelling := range []string{"++", "+-", "β"} {
		if IsLanguageOwnedOperatorSpelling(spelling) {
			t.Errorf("%q should remain project-declarable", spelling)
		}
	}
	for _, spelling := range []string{"::=", "->>", "<->", "`", `\`, "#", "//", "/*"} {
		if !IsHardReservedOperatorSpelling(spelling) {
			t.Errorf("%q should be hard-reserved", spelling)
		}
	}
}

func TestPredeclaredOperatorSpellingQueryIsGlyphOnly(t *testing.T) {
	for _, spelling := range []string{"∪", "Ť"} {
		if !IsPredeclaredOperatorSpelling(spelling) {
			t.Errorf("%q should be predeclared", spelling)
		}
	}
	for _, spelling := range []string{"+", "**", "::=", "++"} {
		if IsPredeclaredOperatorSpelling(spelling) {
			t.Errorf("%q should not be classified as a predeclared glyph", spelling)
		}
	}
}

func TestOrdinaryScanDoesNotClassifyOperatorSourceKind(t *testing.T) {
	tokens := Tokenize(`co.lang.operator`, "symbols.fol")
	if len(tokens) != 1 {
		t.Fatalf("co.lang.operator tokens = %#v, want one source-kind token", tokens)
	}
	if tokens[0].Kind != OPERATOR_SOURCE_KIND || tokens[0].Value != "co.lang.operator" {
		t.Fatalf("co.lang.operator token = %#v, want exact OPERATOR_SOURCE_KIND", tokens[0])
	}
}

func tokenWithValue(t *testing.T, tokens []Token, value string) Token {
	t.Helper()
	for _, token := range tokens {
		if token.Value == value {
			return token
		}
	}
	t.Fatalf("token stream %#v has no %q", tokens, value)
	return Token{}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
