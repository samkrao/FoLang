package builtins

import "fmt"

// TokenKind represents the type of a lexical token.
type TokenKind int

const (
	EOF                 TokenKind = iota // 0
	NUMBER                               // 1
	STRING                               // 2
	IDENTIFIER                           // 3
	COMPOSITE_IDENTIFER                  // 4

	// Grouping & Braces
	OPEN_BRACKET  // 5
	CLOSE_BRACKET // 6
	OPEN_CURLY    // 7
	CLOSE_CURLY   // 8
	OPEN_PAREN    // 9
	CLOSE_PAREN   // 10

	// Equivilance
	ASSIGNMENT // 11
	EQUALS     // 12
	NOT_EQUALS // 13
	NOT        // 14

	// Conditional
	LESS           // 15
	LESS_EQUALS    // 16
	GREATER        // 17
	GREATER_EQUALS // 18

	// Logical
	OR  // 19
	AND // 20

	// Symbols
	DOT         // 21
	DOT_DOT     // 22
	DOT_DOT_DOT // 23
	SEMI_COLON  // 24
	COLON       // 25
	QUESTION    // 26
	COMMA       // 27

	// Shorthand
	PLUS_PLUS          // 28
	MINUS_MINUS        // 29
	PLUS_EQUALS        // 30
	MINUS_EQUALS       // 31
	NULLISH_ASSIGNMENT // ??=	// 32

	//Maths
	PLUS    // 33
	MINUS   // 34
	SLASH   // 35
	MUL     // 36
	PERCENT // 37
	POW     // 38

	// object ops
	STAR  // 39
	AT    // 40
	AMPS  // 41
	ARROW // 42
	EQGT  // 43

	// Keywords
	KEYWORD // 44
	//REserved Words
	RESERVEDWORD // 45

	CONTEXT_KEYWORD // 46

	BUILT_IN_METHOD     // 47
	BUILT_IN_TYPE       // 48
	BUILT_IN_KIND       // 48
	BUILT_IN_DIRECTIVES // 50
	BUIL_IN_STMT_EXPRS  // 51
	BUILT_IN_CONSTANTS  // 52

	// Misc
	NUM_TOKENS           // 53
	HASH                 // 54
	CONTEXT_SIGIL_DOLLAR // 55
	TILD                 // 56
	FORWARD_SLASH        // 57
	BACK_TICK            // 58
	PIPE                 // 59
	SINGLE_QUOTE         // 60
	DOUBL_QUOTE          // 61

	//OTHER_OPERTORS
	TT_OP_LAMBDA          // λ	// 62 ⒪
	TT_OP_ANONYMOUS       // â	// 63
	TT_OP_TURING          // Ť	// 64
	TT_OP_FOREACH         // ∀	// 65
	TT_OP_THEXISTS        // ∃	// 66
	TT_OP_ALG_OF          // ○	// 67 ö
	TT_OP_ALG_UNION       // ∪	// 68
	TT_OP_S_EXPR          // Ṡ	// 69 Ŝ
	TT_OP_M_EXPR          // ṁ	// 70
	TT_OP_PI              // 𝚷	// 71
	TT_OP_TRPL_ARROW      // ⇛ // 72
	TT_OP_FUNCTION        // 𝑓 // 73
	TT_OP_TYPE            // 𝒯 // 74
	TT_OP_VARIABLE        // 𝘷 // 75
	TT_OP_MATH_CAL_F_DELC // 𝓕 //76  ↓, λ, ∂, or ⊥ ↧ or ⇓
	NOT_SUPPORTED         // 77
	NATOKN                //78
	CHAR                  //79
	BOOL                  // 80
	INVALID               //81

	EQGTGT                 //82
	BUILT_IN_SPECIAL_TYPES //83
	ATDAP                  //84
	NONKEYRESERVEDWORD     //85
	TK_UNIT                //86
	CUSTOM_DIRECTIVES      //87
	NEWLINE                //88
	SPACE                  //89
	TILD_TILD              //90
	DBL_UNDERSCORE         //91
	BIND_VAR               //92
	DISCARD_WILD_VAR       //93
	DOT_DOT_LT             //94
	LT_DOT_DOT             //95
	LT_DOT_DOT_LT          //96
	OB_COLON_CB            //97
	WALRUS                 // 98 :=
	COLON_WALRUS           // 99 ::=	ynamic variable binding operator, used for dynamic variable binding in pattern matching and comprehensions
	QEQ                    // 100 ?=	used for conditional assignments like in if statements and pattern matching
	LEFT_ARROW             // 101 <- comprehension generator / channel receive
	ARROW_GT               // 102 ->> continue marker after `this`; pipeline/reverse chaining elsewhere
	BIDIR_ARROW            // 103 <-> bidirectional channel / swap operator
	DOUBLE_AT              // 104 @@ special method prefix (@@new, @@init)
	EQEQGTGT               // 105  ==>>
	SPECIAL_METHODS        //106
	CUSTOM_OPERATOR        // 107 a user-defined operator symbol (DECISION-EXT-001)
	BACK_SLASH             // 108 reserved backslash operator (DECISION-OP-005)
	// METHOD_CALL marks an ordinary dotted member immediately followed by an
	// argument list. Unlike BUILT_IN_METHOD it carries no built-in candidacy;
	// the distinction prevents qualified-name folding from hiding the member
	// boundary needed by the parser's uniform CallExpr shape.
	METHOD_CALL //109
	// SYMBOLIC_RUN preserves a complete contiguous spelling that has no fixed or
	// registered lexical classification. Grammar context may still accept it as
	// metadata (for example *** as pointer degree); otherwise the parser rejects
	// the whole run without fallback splitting (DECISION-LEX-003).
	SYMBOLIC_RUN //110
	// OPERATOR_SOURCE_KIND preserves co.operator for the dedicated
	// operator-source grammar without admitting it as an ordinary BUILT_IN_KIND.
	OPERATOR_SOURCE_KIND //111
	// OPERATOR_SOURCE_CONSTANT preserves a co.operator.* property value for the
	// dedicated operator-source grammar, for the same reason OPERATOR_SOURCE_KIND
	// exists: these spellings are meaningful only inside an operator declaration,
	// so classifying them as ordinary constants would admit them as literals in
	// every expression (DECISION-OPDECL-006).
	OPERATOR_SOURCE_CONSTANT //112
	// LABEL_IDENTIFIER is a structured-control label: an apostrophe followed by
	// an ordinary identifier and NOT closed by a second apostrophe.
	//
	//	label-identifier = single-quote, identifier, label-identifier-guard
	//
	// The guard is what keeps `'c'` a CHAR: a complete character literal is
	// recognized first, so the label rule only ever sees an apostrophe run that
	// no closing apostrophe terminates. Labels live in their own lexical and
	// control namespace, so this is a token kind of its own rather than an
	// apostrophe operator applied to an identifier
	// (docs/language-ref.md, "Label Lexing and Character Literals").
	LABEL_IDENTIFIER //113
	// LIFECYCLE_MARKER is "::", the structural marker of lifecycle-call-suffix.
	//
	// It is deliberately absent from reserved-operator and from the Pratt tables:
	// `::` is consumed only by the postfix lifecycle-call form `receiver::new(…)`
	// and is never an ordinary infix operator. The longer "::=" keeps its own
	// COLON_WALRUS classification, which the symbolic-run scan resolves first
	// because it matches the longest complete spelling.
	LIFECYCLE_MARKER //114

	BUILT_IN_COLLECTIONS //115
	// CARET_EQGT is retained as a numeric compatibility value. The ^=> spelling
	// is not part of the current language and is never emitted by the lexer.
	CARET_EQGT //116
	ARROW_PIPE //117
	EQ_GT      //118
	UNKNOWN    //119

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
	case PLUS_PLUS:
		return "plus_plus"
	case MINUS_MINUS:
		return "minus_minus"
	case PLUS_EQUALS:
		return "plus_equals"
	case MINUS_EQUALS:
		return "minus_equals"
	case NULLISH_ASSIGNMENT:
		return "nullish_assignment"
	case PLUS:
		return "plus"
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
	case RESERVEDWORD:
		return "reservedword"
	case BUILT_IN_METHOD:
		return "builtinmethod"
	case METHOD_CALL:
		return "methodcall"
	case SPECIAL_METHODS:
		return "specialmethod"
	case CUSTOM_OPERATOR:
		return "custom operator"
	case SYMBOLIC_RUN:
		return "symbolic run"
	case OPERATOR_SOURCE_KIND:
		return "operator source kind"
	case BACK_TICK:
		return "backtick"
	case BACK_SLASH:
		return "backslash"
	case BUILT_IN_TYPE:
		return "builtindatatype"
	case COMPOSITE_IDENTIFER:
		return "compositeidentifier"
	case ARROW:
		return "arrow"
	case LEFT_ARROW:
		return "left_arrow"
	case NONKEYRESERVEDWORD:
		return "Non KeyWord/Reserved Word"
	case INVALID:
		return "InValid "
	case UNKNOWN:
		return "UNKNOWN"

	// The remaining kinds were missing a name, so every one of them printed as
	// `unknown(N)`. That is only cosmetic in a diagnostic, but the debug trace
	// labels each step with this string, and two thirds of a trace reading
	// `token=unknown(93)` is a trace nobody can follow. TestTokenKindStringNamesEveryKind
	// keeps the set complete.
	case AT:
		return "at"
	case ATDAP:
		return "at_dap"
	case DOUBLE_AT:
		return "double_at"
	case AMPS:
		return "amps"
	case PIPE:
		return "pipe"
	case MUL:
		return "mul"
	case POW:
		return "pow"
	case HASH:
		return "hash"
	case CONTEXT_SIGIL_DOLLAR:
		return "dollar"
	case TILD:
		return "tilde"
	case TILD_TILD:
		return "tilde_tilde"
	case FORWARD_SLASH:
		return "forward_slash"
	case SINGLE_QUOTE:
		return "single_quote"
	case DOUBL_QUOTE:
		return "double_quote"
	case DBL_UNDERSCORE:
		return "double_underscore"
	case CHAR:
		return "character"
	case LABEL_IDENTIFIER:
		return "label_identifier"
	case LIFECYCLE_MARKER:
		return "lifecycle_marker"
	case BOOL:
		return "boolean"

	case CONTEXT_KEYWORD:
		return "context_keyword"
	case BUILT_IN_KIND:
		return "builtin_kind"
	case BUILT_IN_DIRECTIVES:
		return "builtin_directive"
	case BUIL_IN_STMT_EXPRS:
		return "builtin_statement_expression"
	case BUILT_IN_CONSTANTS:
		return "builtin_constant"
	case BUILT_IN_SPECIAL_TYPES:
		return "builtin_special_type"
	case BUILT_IN_COLLECTIONS:
		return "builtin_collection"
	case CUSTOM_DIRECTIVES:
		return "custom_directive"
	case OPERATOR_SOURCE_CONSTANT:
		return "operator_source_constant"
	case TK_UNIT:
		return "unit"

	case DOT_DOT_DOT:
		return "dot_dot_dot"
	case DOT_DOT_LT:
		return "dot_dot_lt"
	case LT_DOT_DOT:
		return "lt_dot_dot"
	case LT_DOT_DOT_LT:
		return "lt_dot_dot_lt"
	case OB_COLON_CB:
		return "open_bracket_colon_close_bracket"
	case WALRUS:
		return "walrus"
	case COLON_WALRUS:
		return "colon_walrus"
	case QEQ:
		return "question_equals"
	case EQGT:
		return "eq_gt"
	case EQGTGT:
		return "eq_gt_gt"
	case EQEQGTGT:
		return "eq_eq_gt_gt"
	case BIDIR_ARROW:
		return "bidirectional_arrow"
	case BIND_VAR:
		return "bind_var"
	case DISCARD_WILD_VAR:
		return "discard_wildcard"

	case NEWLINE:
		return "newline"
	case SPACE:
		return "space"
	case NOT_SUPPORTED:
		return "not_supported"
	case NATOKN:
		return "not_a_token"
	case NUM_TOKENS:
		return "num_tokens"

	// The pre-declared operator glyphs. Each is language-reserved and rejected
	// by the parser, so a trace showing one is showing why a file failed.
	case TT_OP_LAMBDA:
		return "glyph_lambda"
	case TT_OP_ANONYMOUS:
		return "glyph_anonymous"
	case TT_OP_TURING:
		return "glyph_turing"
	case TT_OP_FOREACH:
		return "glyph_forall"
	case TT_OP_THEXISTS:
		return "glyph_exists"
	case TT_OP_ALG_OF:
		return "glyph_algebra_of"
	case TT_OP_ALG_UNION:
		return "glyph_algebra_union"
	case TT_OP_S_EXPR:
		return "glyph_s_expression"
	case TT_OP_M_EXPR:
		return "glyph_m_expression"
	case TT_OP_PI:
		return "glyph_pi"
	case TT_OP_TRPL_ARROW:
		return "glyph_triple_arrow"
	case TT_OP_FUNCTION:
		return "glyph_function"
	case TT_OP_TYPE:
		return "glyph_type"
	case TT_OP_VARIABLE:
		return "glyph_variable"
	case TT_OP_MATH_CAL_F_DELC:
		return "glyph_calligraphic_f"

	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

var BuiltInDirectives map[DirectiveKind][]string = map[DirectiveKind][]string{}

var Operator_source_constants map[string]string = map[string]string{}

var Built_in_stmt_exprs map[string][]string = map[string][]string{}

var Builtin_types []string = []string{}

var Built_In_Collections = []string{}

var Builtin_Kinds []string = []string{}

var KeyWords_me map[string][]string = map[string][]string{}

var Special_methods []string = []string{}

var Reserved_me []string = []string{}
