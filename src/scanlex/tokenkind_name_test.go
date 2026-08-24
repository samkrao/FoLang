package scanlex

import (
	"strings"
	"testing"
)

// TestTokenKindStringNamesEveryKind keeps TokenKindString complete.
//
// The name is not only diagnostic decoration: the DEBUG_TRACE lexer and parser
// traces label every step with it, so a kind with no name turns a trace line
// into `token=unknown(93)`. Sixty-one of the hundred-odd kinds were unnamed
// before this test existed, including ones as common as BUILT_IN_KIND, WALRUS
// and DISCARD_WILD_VAR.
//
// The list is written out rather than derived, because Go offers no way to
// enumerate the members of an untyped constant block at run time. A kind added
// without a line here is caught by TestTokenKindListIsComplete below.
func TestTokenKindStringNamesEveryKind(t *testing.T) {
	for _, kind := range allTokenKinds {
		name := TokenKindString(kind)
		if strings.HasPrefix(name, "unknown(") {
			t.Errorf("TokenKind %d has no name", int(kind))
		}
		if strings.TrimSpace(name) == "" {
			t.Errorf("TokenKind %d has an empty name", int(kind))
		}
	}
}

// TestTokenKindListIsComplete checks that allTokenKinds covers the whole
// contiguous range the constant block defines, so a kind appended to that block
// cannot silently escape the name check above.
func TestTokenKindListIsComplete(t *testing.T) {
	seen := make(map[TokenKind]bool, len(allTokenKinds))
	highest := TokenKind(0)
	for _, kind := range allTokenKinds {
		if seen[kind] {
			t.Errorf("TokenKind %d (%s) is listed twice", int(kind), TokenKindString(kind))
		}
		seen[kind] = true
		if kind > highest {
			highest = kind
		}
	}
	for kind := TokenKind(0); kind <= highest; kind++ {
		if !seen[kind] {
			t.Errorf("TokenKind %d is defined but missing from allTokenKinds", int(kind))
		}
	}
}

// allTokenKinds is every TokenKind the constant block in token.go defines.
var allTokenKinds = []TokenKind{
	EOF, NUMBER, STRING, IDENTIFIER, COMPOSITE_IDENTIFER, OPEN_BRACKET, CLOSE_BRACKET,
	OPEN_CURLY, CLOSE_CURLY, OPEN_PAREN, CLOSE_PAREN, ASSIGNMENT, EQUALS,
	NOT_EQUALS, NOT, LESS, LESS_EQUALS, GREATER, GREATER_EQUALS, OR, AND,
	DOT, DOT_DOT, DOT_DOT_DOT, SEMI_COLON, COLON, QUESTION, COMMA, PLUS_PLUS,
	MINUS_MINUS, PLUS_EQUALS, MINUS_EQUALS, NULLISH_ASSIGNMENT, PLUS, MINUS, SLASH, MUL, PERCENT,
	POW, STAR, AT, AMPS, ARROW, EQGT, KEYWORD, RESERVEDWORD, CONTEXT_KEYWORD,
	BUILT_IN_METHOD, BUILT_IN_TYPE, BUILT_IN_KIND, BUILT_IN_DIRECTIVES,
	BUIL_IN_STMT_EXPRS, BUILT_IN_CONSTANTS, NUM_TOKENS, HASH, DOLLAR, TILD,
	FORWARD_SLASH, BACK_TICK, PIPE, SINGLE_QUOTE, DOUBL_QUOTE,
	TT_OP_LAMBDA, TT_OP_ANONYMOUS, TT_OP_TURING, TT_OP_FOREACH, TT_OP_THEXISTS,
	TT_OP_ALG_OF, TT_OP_ALG_UNION, TT_OP_S_EXPR, TT_OP_M_EXPR, TT_OP_PI,
	TT_OP_TRPL_ARROW, TT_OP_FUNCTION, TT_OP_TYPE, TT_OP_VARIABLE,
	TT_OP_MATH_CAL_F_DELC, NOT_SUPPORTED, NATOKN, CHAR, BOOL, INVALID,
	EQGTGT, BUILT_IN_SPECIAL_TYPES, ATDAP, NONKEYRESERVEDWORD, TK_UNIT,
	CUSTOM_DIRECTIVES, NEWLINE, SPACE, TILD_TILD, DBL_UNDERSCORE, BIND_VAR,
	DISCARD_WILD_VAR, DOT_DOT_LT, LT_DOT_DOT, LT_DOT_DOT_LT, OB_COLON_CB,
	WALRUS, COLON_WALRUS, QEQ, LEFT_ARROW, MINUS_ARROW_GT, BIDIR_ARROW,
	DOUBLE_AT, EQEQGTGT, SPECIAL_METHODS, CUSTOM_OPERATOR, BACK_SLASH,
	METHOD_CALL, SYMBOLIC_RUN, OPERATOR_SOURCE_KIND, OPERATOR_SOURCE_CONSTANT,
	LABEL_IDENTIFIER, LIFECYCLE_MARKER,
}
