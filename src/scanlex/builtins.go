package scanlex

import (
	"fmt"
)

// TokenKind represents the type of a lexical token.
type TokenKind int

const (
	EOF                 TokenKind = iota // 0
	NUMBER                               // 1
	CHAR                                 // 2
	BOOL                                 // 3
	STRING                               // 4
	IDENTIFIER                           // 5
	COMPOSITE_IDENTIFER                  // 6

	// Keywords
	KEYWORD // 7

	BUILT_INS_FOL     // 8
	CUSTOM_DIRECTIVES // 9

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
	CARET_EQGT        // 60 ^=>
	UNDERSCORE        // 61 _
	EQ_GT             //62 =>
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
	POW_EQUALS        // 78 **=
	PERCENTILE_EQUALS //79 %=
	UDT               // 80
	COMMENTS          // 81

)

type SubKind int

const (
	TYPE SubKind = iota
	COLLECTION
	KIND
	SIGILS
	TWIGILS
	CONSTANTS
	DIRECTIVES
	STATMENT_EXPR
	METHOD
	SPECIAL_METHOD
	LABEL_LITERAL
	DISCARD_WILD_CJAR
	BINDVAR
	CONTEXT_SIGIL
	CONTEXT_TWIGIL
	CUSTOM_OPERATOR
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
	switch any(kind) {
	case EOF:
		return "eof"
	case NUMBER:
		return "number"
	case STRING:
		return "string"
	case IDENTIFIER:
		return "identifier"
	case OPEN_BRACKET:
		return "open_bracket"
	case CLOSE_BRACKET:
		return "close_bracket"
	case OPEN_CURLY:
		return "open_curly"
	case CLOSE_CURLY:
		return "close_curly"
	case OPEN_PAREN:
		return "open_paren"
	case CLOSE_PAREN:
		return "close_paren"
	case ASSIGNMENT:
		return "assignment"
	case EQUALS:
		return "equals"
	case NOT_EQUALS:
		return "not_equals"
	case NOT:
		return "not"
	case LESS:
		return "less"
	case LESS_EQUALS:
		return "less_equals"
	case GREATER:
		return "greater"
	case GREATER_EQUALS:
		return "greater_equals"
	case OR:
		return "or"
	case AND:
		return "and"
	case DOT:
		return "dot"
	case DOT_DOT:
		return "dot_dot"
	case SEMI_COLON:
		return "semi colon"
	case COLON:
		return "colon"
	case QUESTION:
		return "question"
	case COMMA:
		return "comma"
	case MINUS:
		return "dash"
	case SLASH:
		return "slash"
	case STAR:
		return "star"
	case PERCENT:
		return "percent"
	case KEYWORD:
		return "keyword"
	case UNKNOWN:
		return "UNKNOWN"

	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

func foldTokens(lex *lexer) []Token {
	nTokens := make([]Token, 0)
	for {
		if lex.isEof() {
			break
		}

		Token_ := lex.currentToken()
		if Token_.Kind == IDENTIFIER || Token_.Kind == KEYWORD || Token_.Kind == AT {

			return nTokens
		}
	}
	return nTokens
}
