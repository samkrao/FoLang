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
	if name, ok := tokenKindNames[kind]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_TOKEN_KIND(%d)", kind)
}

var tokenKindNames = map[TokenKind]string{
	EOF: "EOF", NUMBER: "NUMBER", CHAR: "CHAR", BOOL: "BOOL", STRING: "STRING",
	IDENTIFIER: "IDENTIFIER", COMPOSITE_IDENTIFIER: "COMPOSITE_IDENTIFIER", KEYWORD: "KEYWORD",
	BUILT_INS_FOL: "BUILT_INS_FOL", CUSTOM_ANNOT_DECOR: "CUSTOM_ANNOT_DECOR",
	OPEN_BRACKET: "OPEN_BRACKET", CLOSE_BRACKET: "CLOSE_BRACKET", OPEN_CURLY: "OPEN_CURLY",
	CLOSE_CURLY: "CLOSE_CURLY", OPEN_PAREN: "OPEN_PAREN", CLOSE_PAREN: "CLOSE_PAREN",
	ASSIGNMENT: "ASSIGNMENT", EQUALS: "EQUALS", NOT_EQUALS: "NOT_EQUALS", NOT: "NOT",
	LESS: "LESS", LESS_EQUALS: "LESS_EQUALS", GREATER: "GREATER", GREATER_EQUALS: "GREATER_EQUALS",
	OR: "OR", AND: "AND", DOT: "DOT", DOT_DOT: "DOT_DOT", DOT_DOT_DOT: "DOT_DOT_DOT",
	SEMI_COLON: "SEMI_COLON", COLON: "COLON", QUESTION: "QUESTION", COMMA: "COMMA",
	PLUS_EQUALS: "PLUS_EQUALS", MINUS_EQUALS: "MINUS_EQUALS", STAR_EQUALS: "STAR_EQUALS",
	SLASH_EQUALS: "SLASH_EQUALS", PLUS: "PLUS", MINUS: "MINUS", SLASH: "SLASH", STAR: "STAR",
	PERCENT: "PERCENT", POW: "POW", AT: "AT", AMPS: "AMPS", ARROW: "ARROW", EQGT: "EQGT",
	HASH: "HASH", DOLLAR: "DOLLAR", TILD: "TILD", FORWARD_SLASH: "FORWARD_SLASH", CARET: "CARET",
	BACK_TICK: "BACK_TICK", PIPE: "PIPE", SINGLE_QUOTE: "SINGLE_QUOTE", DOUBL_QUOTE: "DOUBL_QUOTE",
	EQGTGT: "EQGTGT", NEWLINE: "NEWLINE", SPACE: "SPACE", DBL_COLON: "DBL_COLON",
	POW_EQUALS: "POW_EQUALS", UNDERSCORE: "UNDERSCORE", PERCENTILE_EQUALS: "PERCENTILE_EQUALS",
	DOT_DOT_LT: "DOT_DOT_LT", LT_DOT_DOT: "LT_DOT_DOT", LT_DOT_DOT_LT: "LT_DOT_DOT_LT",
	OB_COLON_CB: "OB_COLON_CB", WALRUS: "WALRUS", COLON_WALRUS: "COLON_WALRUS", QEQ: "QEQ",
	LEFT_ARROW: "LEFT_ARROW", ARROW_GT: "ARROW_GT", BIDIR_ARROW: "BIDIR_ARROW", DOUBLE_AT: "DOUBLE_AT",
	EQEQGTGT: "EQEQGTGT", METHOD_CALL: "METHOD_CALL", ARROW_PIPE: "ARROW_PIPE", UNKNOWN: "UNKNOWN",
	UDT: "UDT", CUSTOM_OPERATOR: "CUSTOM_OPERATOR",
}

// SubKindString returns the stable diagnostic name of a parser-facing subkind.
func SubKindString(subKind SubKind) string {
	switch subKind {
	case SIGILS:
		return "SIGILS"
	case DIRECTIVES:
		return "DIRECTIVES"
	case STATEMENT_EXPR:
		return "STATEMENT_EXPR"
	case METHOD:
		return "METHOD"
	case SPECIAL_METHOD:
		return "SPECIAL_METHOD"
	case LABEL_LITERAL:
		return "LABEL_LITERAL"
	case DISCARD_WILD_CHAR:
		return "DISCARD_WILD_CHAR"
	case BINDVAR:
		return "BINDVAR"
	case OPERATORS:
		return "OPERATORS"
	case OPERATOR_SOURCE:
		return "OPERATOR_SOURCE"
	case NA:
		return "NA"
	default:
		return fmt.Sprintf("UNKNOWN_SUBKIND(%d)", subKind)
	}
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
