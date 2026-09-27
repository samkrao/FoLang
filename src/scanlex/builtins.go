package scanlex

import (
	"fmt"
	"strings"
)

// TokenKind represents the type of a lexical token.
type TokenKind int

const (
	EOF                  TokenKind = iota // 0
	NUMBER                                // 1
	CHAR                                  // 2
	BOOL                                  // 3
	STRING                                // 4
	IDENTIFIER                            // 5
	COMPOSITE_IDENTIFIER                  // 6

	// Keywords
	KEYWORD // 7

	BUILT_INS_FOL      // 8
	CUSTOM_ANNOT_DECOR // 9

	// Grouping & Braces
	OPEN_BRACKET  // 10 [
	CLOSE_BRACKET // 11 ]
	OPEN_CURLY    // 12 {
	CLOSE_CURLY   // 13 }
	OPEN_PAREN    // 14 (
	CLOSE_PAREN   // 15 )

	// Equivilance
	ASSIGNMENT // 16 =
	EQUALS     // 17 ==
	NOT_EQUALS // 18 !=
	NOT        // 19 !

	// Conditional
	LESS           // 20 <
	LESS_EQUALS    // 21 <=
	GREATER        // 22 >
	GREATER_EQUALS // 23 >=

	// Logical
	OR  // 24  ||
	AND // 25  &&

	// Symbols
	DOT         // 26 .
	DOT_DOT     // 27 ..
	DOT_DOT_DOT // 28 ...
	SEMI_COLON  // 29 ;
	COLON       // 30 :
	QUESTION    // 31 ?
	COMMA       // 32 ,

	// Shorthand
	PLUS_EQUALS  // 33 +=
	MINUS_EQUALS // 34 -=
	STAR_EQUALS  // 35 *=
	SLASH_EQUALS // 36 /=

	//Maths
	PLUS    // 37 +
	MINUS   // 38 -
	SLASH   // 39 /
	STAR    // 40 *
	PERCENT // 41 %
	POW     // 42 **

	// object ops
	AT    // 43 @
	AMPS  // 44 &
	ARROW // 45 ->
	EQGT  // 46 =>

	// Misc
	HASH          // 47 #
	DOLLAR        // 48 $
	TILD          // 49 ~
	FORWARD_SLASH // 50 \
	CARET         // 51 ^
	BACK_TICK     // 52 `
	PIPE          // 53 |
	SINGLE_QUOTE  // 54 '
	DOUBL_QUOTE   // 55 ""

	EQGTGT            //56 =>>
	NEWLINE           //57 \n\r
	SPACE             //58
	DBL_COLON         // 59 ::
	POW_EQUALS        // 60 **=
	UNDERSCORE        // 61 _
	PERCENTILE_EQUALS //62 %=
	DOT_DOT_LT        //63 ..<
	LT_DOT_DOT        //64 <..
	LT_DOT_DOT_LT     //65 <..<
	OB_COLON_CB       //66 [:]
	WALRUS            // 67 :=
	COLON_WALRUS      // 68 ::=	dynamic variable binding operator, used for dynamic variable binding in pattern matching and comprehensions
	QEQ               // 69 ?=	used for conditional assignments like in if statements and pattern matching
	LEFT_ARROW        // 70 <- comprehension generator / channel receive
	ARROW_GT          // 71 ->> continue marker after `this`; pipeline/reverse chaining elsewhere
	BIDIR_ARROW       // 72 <-> bidirectional channel / swap operator
	DOUBLE_AT         // 73 @@ special method prefix (@@new, @@init)
	EQEQGTGT          // 74  ==>>
	METHOD_CALL       // 75
	ARROW_PIPE        //76 ->|
	UNKNOWN           //77

	UDT // 78
	CUSTOM_OPERATOR
)

type SubKind int

const (
	SIGILS SubKind = iota
	DIRECTIVES
	STATEMENT_EXPR
	METHOD
	SPECIAL_METHOD
	LABEL_LITERAL
	DISCARD_WILD_CHAR
	BINDVAR
	OPERATORS
	OPERATOR_SOURCE
	NA
)

// DirectiveKind distinguishes between pragmas and annotation decorators.
type DirectiveKind int

const (
	PRAGMA DirectiveKind = iota
	DIRECTIVE
	ANNOTATION
	DECORATOR
	Invalid
)

// TokenKindString returns the human-readable string name for a TokenKind.
func TokenKindString(kind TokenKind) string {
	return fmt.Sprintf("(%d)", kind)

}

// foldTokens consumes one raw scanner token, plus any immediately-adjacent
// tokens that form the same parser-facing lexeme. Whitespace and newlines are
// deliberately not folded away; they remain ordinary stream tokens.
func foldTokens(stream *TokenStream) Token {
	first := stream.nextRaw()
	if first.Kind == EOF {
		return first
	}

	// Any syntactically valid @@name is kept whole for the parser. The parser,
	// rather than the scanner, decides whether that special method is allowed.
	if strings.HasPrefix(first.Value, "@@") && isValidIdentifierLexeme(first.Value[2:]) {
		return refold(first, first, BUILT_INS_FOL, SPECIAL_METHOD, first.Value)
	}

	// Metadata names are folded through their last adjacent dotted segment.
	// The co namespace is language owned; all other names are resolved later as
	// user annotations/decorators.
	if first.Kind == AT && len(first.Value) > 1 {
		last, value := stream.foldDotted(first)
		if value == "@co" || strings.HasPrefix(value, "@co.") {
			return refold(first, last, BUILT_INS_FOL, DIRECTIVES, value)
		}
		return refold(first, last, CUSTOM_ANNOT_DECOR, NA, value)
	}

	// co.* is one language-owned lexeme. A following call parenthesis makes it
	// a method; co.operator and its property constants are operator-source
	// spellings; all other paths are statement/type/expression built-ins.
	if first.Kind == KEYWORD && first.Value == "co" && stream.rawStartsDottedSegment(first) {
		last, value := stream.foldDotted(first)
		subKind := STATEMENT_EXPR
		if stream.followedByCall(last) {
			subKind = METHOD
		} else if value == "co.operator" || strings.HasPrefix(value, "co.operator.") {
			subKind = OPERATOR_SOURCE
		}
		return refold(first, last, BUILT_INS_FOL, subKind, value)
	}

	// this.member and compiler-owned this->member receiver paths use the same
	// parser-facing classification as co paths. The parser still validates which
	// arrow attributes are legal in the current declaration context.
	if first.Kind == KEYWORD && first.Value == "this" {
		if last, value, ok := stream.foldThisPath(first); ok {
			subKind := STATEMENT_EXPR
			if stream.followedByCall(last) {
				subKind = METHOD
			}
			return refold(first, last, BUILT_INS_FOL, subKind, value)
		}
	}

	// A label declaration is an apostrophe-prefixed identifier immediately
	// followed by a colon. The colon remains a separate parser-facing token.
	if first.Kind == SINGLE_QUOTE && stream.rawByteAtEnd(first) == ':' {
		return refold(first, first, BUILT_INS_FOL, LABEL_LITERAL, first.Value)
	}

	if first.Kind == DOLLAR {
		subKind := SIGILS
		if len(first.Value) > 1 {
			subKind = BINDVAR
		}
		return refold(first, first, BUILT_INS_FOL, subKind, first.Value)
	}

	if first.Kind == UNDERSCORE && first.Value == "_" {
		return refold(first, first, BUILT_INS_FOL, DISCARD_WILD_CHAR, first.Value)
	}

	if first.SubKind == OPERATORS || IsPredeclaredOperatorSpelling(first.Value) {
		return refold(first, first, BUILT_INS_FOL, OPERATORS, first.Value)
	}

	if first.Kind == CUSTOM_OPERATOR || first.Kind == UDT {
		return refold(first, first, CUSTOM_OPERATOR, NA, first.Value)
	}

	if first.Kind == IDENTIFIER && stream.rawStartsDottedSegment(first) {
		last, value := stream.foldDotted(first)
		return refold(first, last, COMPOSITE_IDENTIFIER, NA, value)
	}

	// Subkinds are meaningful only for language built-ins. Everything else is
	// normalized to NA before it reaches the parser.
	first.SubKind = NA
	return first
}

func refold(first, last Token, kind TokenKind, subKind SubKind, value string) Token {
	token := newUniqueToken(kind, subKind, value, first.StartPos, last.EndPos)
	token.BoundaryBefore = first.BoundaryBefore
	token.BoundaryAfter = last.BoundaryAfter
	return token
}
