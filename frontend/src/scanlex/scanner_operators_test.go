package scanlex

import (
	"reflect"
	"testing"
)

func TestTokenizeBuiltInOperatorsUsesMaximalMunch(t *testing.T) {
	tokens := Tokenize(`**= ** *= /= %= &= ^= |=`, "operators.fol")

	wantValues := []string{"**=", "**", "*=", "/=", "%=", "&=", "^=", "|="}
	wantKinds := []TokenKind{
		ASSIGNMENT,
		POW,
		ASSIGNMENT,
		ASSIGNMENT,
		ASSIGNMENT,
		ASSIGNMENT,
		ASSIGNMENT,
		ASSIGNMENT,
	}

	if got := tokenValues(tokens); !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("Tokenize values = %#v, want %#v", got, wantValues)
	}
	if got := tokenKinds(tokens); !reflect.DeepEqual(got, wantKinds) {
		t.Fatalf("Tokenize kinds = %#v, want %#v", got, wantKinds)
	}
}

func TestTokenizeOperatorsDoesNotFuseAcrossWhitespace(t *testing.T) {
	tokens := Tokenize(`* * ** = * = / = % = & = ^ = | =`, "operators.fol")

	wantValues := []string{"*", "*", "**", "=", "*", "=", "/", "=", "%", "=", "&", "=", "^", "=", "|", "="}
	if got := tokenValues(tokens); !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("Tokenize values = %#v, want %#v", got, wantValues)
	}
}

func TestTokenizeCommentsWinOverSlashOperators(t *testing.T) {
	tokens := Tokenize("// /= **=\n/= /* **= /= */ **=", "operators.fol")

	wantValues := []string{"/=", "**="}
	if got := tokenValues(tokens); !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("Tokenize values = %#v, want %#v", got, wantValues)
	}
}

func TestBuiltinOperatorRegistryIncludesMaximalMunchSpellings(t *testing.T) {
	for _, spelling := range []string{"**", "**=", "*=", "/=", "%=", "&=", "^=", "|="} {
		if !builtinOperatorSpellings[spelling] {
			t.Errorf("builtinOperatorSpellings is missing %q", spelling)
		}
	}
}

func TestTokenizeBackslashKeepsDistinctReservedKind(t *testing.T) {
	tokens := Tokenize(`\`, "operators.fol")
	if len(tokens) != 1 || tokens[0].Kind != BACK_SLASH || tokens[0].Value != `\` {
		t.Fatalf("Tokenize backslash = %#v, want one BACK_SLASH token", tokens)
	}
}

func tokenValues(tokens []Token) []string {
	values := make([]string, len(tokens))
	for i, token := range tokens {
		values[i] = token.Value
	}
	return values
}

func tokenKinds(tokens []Token) []TokenKind {
	kinds := make([]TokenKind, len(tokens))
	for i, token := range tokens {
		kinds[i] = token.Kind
	}
	return kinds
}
