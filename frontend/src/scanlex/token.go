// Package scanlex provides lexical scanning and tokenization for the fo-lang compiler.
// It defines token types, keywords, built-in symbols, and directive metadata used by the parser.
package scanlex

import (
	"fmt"
	"slices"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

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
	NUM_TOKENS    // 53
	HASH          // 54
	DOLLAR        // 55
	TILD          // 56
	FORWARD_SLASH // 57
	BACK_TICK     // 58
	PIPE          // 59
	SINGLE_QUOTE  // 60
	DOUBL_QUOTE   // 61

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
	COLON_WALRUS           // 99 ::=	used for type declarations and macro signatures
	QEQ                    // 100 ?=	used for conditional assignments like in if statements and pattern matching
	LEFT_ARROW             // 101 <- comprehension generator / channel receive
	MINUS_ARROW_GT         // 102 ->> pipeline / reverse chaining operator
	BIDIR_ARROW            // 103 <-> bidirectional channel / swap operator
	DOUBLE_AT              // 104 @@ special method prefix (@@new, @@init)
)

// SpecialBuiltins lists built-in identifiers that receive special treatment during token folding.
var SpecialBuiltins []string = []string{"this.return"}

// Built_in_constants maps co.const constant names to their literal values.
var Built_in_constants map[string]string = map[string]string{
	"co.const.true":  "true",
	"co.const.false": "false",
	"co.const.none":  "none",
}

// Reserved_lu maps reserved language keywords to their TokenKind.
var Reserved_lu map[string]TokenKind = map[string]TokenKind{
	"co":     KEYWORD,         // holds everything
	"this":   KEYWORD,         // refers this/self
	"self":   CONTEXT_KEYWORD, // refers to the current context
	"for":    KEYWORD,         // for comprehensions and for.all
	"let":    KEYWORD,         //let bindings and let recursions
	"forall": KEYWORD,         //haskell kind exactly
	"fo":     RESERVEDWORD,    // fo-lang reserved word
}

// UnsupportedObjects lists keywords whose dot-member access is currently unsupported.
var UnsupportedObjects []string = []string{"let", "forall"}

// KeyWords_me maps each keyword to its valid dot-accessible sub-identifiers.
var KeyWords_me map[string][]string = map[string][]string{
	"let":    {"where"},
	"forall": {},
	"self":   {"parent"},
	"co":     {"dynamic", "macro", "hokrt", "hokrtl", "encoding", "net", "crypto", "nop", "lang", "dap", "ddap", "out", "const", "native", "meta", "core", "sys", "os", "in", "pattern", "control", "runtime", "comptime"},
	"this":   {"prototype", "base", "super", "proto", "object", "class", "module", "kind", "type", "struct", "instance", "callee", "args", "caller", "continue", "break", "fallthrough", "yield", "parent", "return"},
	"fo":     {},
	"for":    {},
}

// Reserved_me lists method and keyword names reserved for built-in object operations.
var Reserved_me []string = []string{
	"to_str",
	"to_int",
	"to_float",
	"to_double",
	"classof",
	"typeof",
	"new",
	"prototype",
	"proto",
	"make",
	"objectof",
	"instanceof",
	"is",
	"as",
	"iskindof",
	"has",
	"hasown",
	"uses",
	"match",
	"matchall",
	"matchany",
	"matchnone",
	"matchtype",
	"case",
	"with",
	"print",
	"println",
	"printsp",
	"echo",
	"contains",
	"cast",
	"to",
	"dummy",
	"clone",
	"of",
	"for",
	"when",
	"where",
	"then",
	"callback",
	"getAttr",
	"inject",
	"isinstance",
	"cast_to",
	"cast_from",
	"do",
	"map",
	"flatMap",
	"orElse",
	"filter",
	"fold",
	"recover",
	"peek",
	"loop",
	"istrue",
	"isfalse",
	"if",
	"elif",
	"else",
	"return",
	"otherwise",
	"each",
	"containsVal",
	"in",
	"decltype", //deduce the type at compile time
	"replace",
	"send",
	"receive",
}

// Built_in_stmt_exprs maps namespace prefixes to their valid sub-methods and statement expressions.
var Built_in_stmt_exprs map[string][]string = map[string][]string{
	"co.native": {"load", "register", "asm", "inline", "emit", "ffi"},
	//## turbo pascal like machine code (__asm(".byte ....."))
	////#pascal emit($5B/$59/$0E/$E8/$00/$00/$58/$05/$08/$00/$50/$51/$53/$CB); to
	// //# c  asm (".byte 0x5B, 0x59, 0x0E, 0xE8, 0x00, 0x00, 0x58, 0x05, 0x08, 0x00, 0x50, 0x51, 0x53, 0xCB\n\t")
	"co":         KeyWords_me["co"],
	"co.dynamic": {},
	"co.meta":    {"ast", "instrument", "transform", "augment", "reflect", "introspect", "patch", "inject", "create", "runtime", "realm"},
	/*
			     patch      :  For patching exiting types, methods/functions, blocks etc
		         instrument :  Add observability/monitoring hooks
				 ast        :  Adding to AST mainly using macros of folang
				 reflect    :  Reflections reading metadata and allowing modification about anything
				 introspect :  Read only Reflection
				 transform  :  Run structural transformations over larger graphs
				 inject     :  Attach behavior or data from the outside
				 create     :  Creating new things
				 augment    :  Extend capabilities in a non-destructive way.
				 runtime    :  Which has eval the evil function like javascript evaluates any string (must be valid folang code ) at runtime without AST changes
				 realm : kind of namespace for adding/accessing same/different class/type/struct name in different realms

	*/
	"co.runtime":        {},
	"co.comptime":       {},
	"co.hokrtl":         {},
	"co.hokrt":          {"Option", "Int", "Char", "Result", "Ordering", "Maybe", "Either"},
	"co.hokrt.Option":   {"Some", "None"},
	"co.hokrt.Maybe":    {"Just", "Nothing"},
	"co.hokrt.Result":   {"Ok", "Err"},
	"co.hokrt.Either":   {"Left", "Right"},
	"co.hokrt.Ordering": {"Less", "Equal", "Greater"},
	"co.core":           {"list", "set", "map", "tree", "trie", "sort", "search", "array", "pointer", "ref", "address", "ptr", "matrix"},
	"co.lang":           {},
	"co.sys":            {"file", "concurrent", "parallel", "goto", "event", "invoke", "bind", "call", "apply", "settimeout", "setinterval", "schedular", "cron", "event", "random", "timer", "date", "time"},
	"co.os":             {"signal", "cmd", "execute", "run", "env", "getenv", "setenv", "unsetenv", "sleep", "exit", "cwd", "chdir", "fork", "wait", "pipe", "dup", "dup2", "close", "readfd", "writefd", "random"},
	"co.out":            {"println", "printsp", "print", "echo"},
	"co.in":             {"read", " readln", "input"},
	"co.sys.file":       {"write", "read", "open", "close", "append", "delete", "copy", "move", "exists"},
	"co.encoding":       {"json", "bson", "base64encode", "base64decode", "yml"},
	"co.dap":            {},
	"co.crypto":         {"hash", "md5", "aes", "rsa", "ssl", "tls", "uuid", "rand"},
	"co.ddap":           {},
	"co.pdap":           {},
	"co.net":            {"tcp", "udp", "http"},
	"co.const":          {"true", "false", "none"},
	"co.pattern":        {"match", "case", "default", "regex", "stex", "Type", "Value", "Shape", "Object", "Instance", "Any"},
	"co.control":        {"do", "if", "else", "otherwise", "default", "return", "shift", "resume"},
	"co.macro":          {"quote", "esc", "gensym", "unquote"},
	"let":               KeyWords_me["let"],
	"for":               KeyWords_me["for"],
	"this":              KeyWords_me["this"],
	"_object":           Reserved_me,
	"_instance":         Reserved_me,
	"_class":            {},
	"_type":             {},
	"_kind":             {},
	"_module":           {},
	"_package":          {},
	"_function":         {},
}

// Builtin_types lists the recognized built-in data type identifiers (co.lang.int, co.lang.string, etc.).
var Builtin_types []string = []string{
	"co.lang.string",
	"co.lang.int",
	"co.lang.bit",
	"co.lang.double",
	"co.lang.float",
	"co.lang.long",
	"co.lang.byte",
	"co.lang.char",
	"co.lang.any",
	"co.lang.dynamic",
	"co.lang.auto",
	"co.lang.infer",
	"co.lang.bool",
	"co.lang.void",
	"co.lang.data",
	"co.lang.value",
	"co.lang.typed",
	"co.lang.untyped", //emulating templates in nim
	"co.mem.region",
	"co.lang.nothing",
	"co.lang.word",
	"co.lang.MatchBindings",
	"co.lang.tag",
	"co.lang.typevalue",
}

// Builtin_Containers lists the recognized container identifiers (package, namespace).
var Builtin_Containers []string = []string{
	"package",
	"namespace",
}

// Builtin_Kinds lists the recognized co.lang kind identifiers (type, struct, class, etc.).
var Builtin_Kinds []string = []string{
	"co.lang.type",
	"co.lang.struct",
	"co.lang.cstruct",
	"co.lang.unit",
	"co.lang.realm",  // similar to loader where symbols reside
	"co.lang.loader", // class, type, functions loader where objects reside loader takes realm as parameter
	"co.lang.class",
	"co.lang.interface",
	"co.lang.union",
	"co.lang.role",
	"co.lang.record",
	"co.lang.property",
	"co.lang.indexer",
	"co.lang.object",
	"co.lang.instance",
	"co.lang.matcher",
	"co.lang.trait",
	"co.lang.mixin",
	"co.lang.extension",
	"co.lang.delegate",
	"co.lang.typeclass",
	"co.lang.concept",
	"co.lang.typealias",
	"co.lang.assoicatedtype",
	"co.lang.module",
	"co.lang.macro",
	"co.lang.template",
	"co.lang.lambda",
	"co.lang.block",
	"co.lang.behavior",
	"co.lang.package",
	"co.lang.library",
	"co.lang.signature",
	"co.lang.function",
	"co.lang.method",
	"co.lang.operator",
	"co.lang.namespace",
	"co.lang.stex",
	"co.lang.kind",
	"co.lang.level",
	"co.lang.order",
	"co.lang.rank",
	"co.lang.newtype",
	"co.lang.opaquetype",
	"co.lang.subtype",
	"co.lang.supertype",
	"co.lang.dependentType",
	"co.lang.refinementType",
	"co.lang.associatedtype",
	"co.lang.hokrtl",
	"co.lang.data",
	"co.lang.enum",
	"co.lang.typetype",
	"co.lang.typekind",
	"co.lang.alias",
	"co.lang.value",
	"co.lang.just",
	"co.lang.nothing",
}

var LIB_KINDS = []string{"application", "advanced", "dynamicvmrt", "ffi", "system"}

// DirectiveKind distinguishes between pragmas and annotation decorators.
type DirectiveKind int

const (
	PRAGMA DirectiveKind = iota
	DIRECTIVE
	ANNOTATION
	DECORATOR
	Invalid
)

var KindToString map[DirectiveKind]string = map[DirectiveKind]string{
	PRAGMA:     "PRAGMA",
	DIRECTIVE:  "DIRECTIVE",
	ANNOTATION: "ANNOTATION",
	DECORATOR:  "DECORATOR",
	Invalid:    "INVALID",
}
var KindToPhase map[DirectiveKind]string = map[DirectiveKind]string{
	PRAGMA:     "COMPILE",
	DIRECTIVE:  "COMPILE",
	ANNOTATION: "RUNTIME",
	DECORATOR:  "RUNTIME",
	Invalid:    "INVALID",
}
var KindToScope map[DirectiveKind]string = map[DirectiveKind]string{
	PRAGMA:     "ENTRY_OR_LIB",
	DIRECTIVE:  "PACKAGE",
	ANNOTATION: "ANY",
	DECORATOR:  "FUN_OR_METH",
	Invalid:    "INVALID",
}
var PDADs map[DirectiveKind][]string = map[DirectiveKind][]string{
	PRAGMA: []string{"@co.pdap.compiler", "@co.pdap.scale"},
	DIRECTIVE: []string{"@co.ddap.movetotop", "@co.ddap.import",
		"@co.ddap.dynamicruntime", "@co.ddap.use", "@co.ddap.alias", "@co.dap.library"},
	ANNOTATION: []string{"@co.dap.template", "@co.dap.macro",
		"@co.dap.operator", "@co.dap.annotation", "@co.dap.library",
		"@co.dap.module", "@co.dap.pragma", "@co.dap.directive",
		"@co.dap.native", "@co.dap.class", "@co.dap.static",
		"@co.dap.instance", "@co.dap.object", "@co.dap.inline",
		"@co.dap.ctfe", "@co.dap.friend", "@co.dap.sealed", "@co.dap.extension",
		"@co.dap.override", "@co.dap.virtual", "@co.dap.abstract",
		"@co.dap.delegate", "@co.dap.dynamicscope", "@co.dap.lexicalscope", "@co.dap.staticscope",
		"@co.dap.mixedscope", "@co.dap.typeclass", "@co.dap.matcher",
		"@co.dap.constructor", "@co.dap.oops", "@co.dap.hokrt",
		"@co.dap.hokrlt", "@co.dap.indexer", "@co.dap.generic", "@co.dap.declare",
		"@co.dap.comptime", "@co.dap.typefromvalue", "@co.dap.export",
		"@co.dap.private", "@co.dap.public", "@co.dap.package", "@co.dap.protected",
		"@co.dap.internal", "@co.dap.eager", "@co.dap.lazy", "@co.dap.packed", "@co.dap.declare",
		"@co.dap.simd", "@co.dap.reflection", "@co.dap.mop", "@co.dap.local", "@co.dap.nested", "@co.dap.inner",
	},
	//mop => meta object programming
	DECORATOR: []string{"@co.dap.before", "@co.dap.after",
		"@co.dap.around", "@co.fx.onErrExcept", "@co.fx.InvokeAlways",
		"@co.fx.HandleEffect", "@co.dap.callback", "@co.dap.defer",
		"@co.dap.continuation", "@co.dap.event", "@co.dap.scale", "@co.dap.distributed",
		"@co.dap.concurrent", "@co.dap.parallel", "@co.dap.subroutine",
		"@co.dap.generator", "@co.dap.goroutine", "@co.dap.coroutine",
		"@co.dap.async", "@co.dap.promise", "@co.dap.future",
		"@co.dap.thread", "@co.dap.task", "@co.dap.fiber", "@co.dap.process",
		"@co.dap.spawn", "@co.dap.exec", "@co.dap.fork", "@co.dap.csp",
		"@co.dap.actor", "@co.dap.synthetic", "@co.dap.bridge",
		"@co.dap.greenlet", "@co.dap.channel", "@co.dap.callable", "@co.dap.iterator"},
}

// Built_in_directives is an alias for the DDAPS directive registry.
func Built_in_directives(dirname string) (string, bool) {
	var flag = false
	var type_ string
	for k, v := range PDADs {
		if slices.Contains(v, dirname) {
			flag = true
			type_ = KindToString[k]
			break
		}
	}
	return type_, flag
}

// Built_in_directive_kind resolves the directive/decorator/pragma/annotation
// name to its actual DirectiveKind, rather than its string form. This lets
// callers that expect one kind still recognize (and correctly parse) a
// built-in directive that turns out to belong to a different kind.
func Built_in_directive_kind(dirname string) (DirectiveKind, bool) {
	for k, v := range PDADs {
		if slices.Contains(v, dirname) {
			return k, true
		}
	}
	return Invalid, false
}

// Token represents a single lexical token with its kind, string value, and source positions.
type Token struct {
	Kind     TokenKind
	Value    string
	StartPos *helpers.Position
	EndPos   *helpers.Position
}

// Println prints the token value, kind, and position range to stdout.
func (tk Token) Println() {
	fmt.Print(tk.Value + " == " + fmt.Sprint(tk.Kind) + " == ")
	tk.StartPos.Print()
	tk.EndPos.Print()
	fmt.Println("")
}

// IsOneOfMany reports whether the token kind matches any of the given expected kinds.
func (tk Token) IsOneOfMany(expectedTokens ...TokenKind) bool {
	for _, expected := range expectedTokens {
		if expected == tk.Kind {
			return true
		}
	}

	return false
}

// DummyNode is a sentinel Token with INVALID kind used as a placeholder.
var DummyNode Token = Token{
	INVALID, "Invalid", helpers.NilPosition, helpers.NilPosition,
}

// NewUniqueToken creates a new Token with the given kind, value, and position range.
func NewUniqueToken(kind TokenKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return newUniqueToken(kind, value, startPos, endPos)
}
func newUniqueToken(kind TokenKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		kind, value, startPos, endPos,
	}
}

func newDummyToken(value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		INVALID, value, startPos, endPos,
	}
}

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
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

// Debug prints the token value and kind to stdout for debugging.
func (token Token) Debug() {
	fmt.Printf("%s => (%s)\n", token.Value, TokenKindString(token.Kind))
}

// AllowedOps lists the token kinds that may legally precede a keyword or reserved word.
var AllowedOps = []TokenKind{OPEN_BRACKET, OPEN_CURLY, OPEN_PAREN, DOT, ASSIGNMENT, CLOSE_CURLY, CLOSE_BRACKET, CLOSE_PAREN, PIPE, AMPS, POW}
