// Package scanlex provides lexical scanning and tokenization for the fo-lang compiler.
// It defines token types, keywords, built-in symbols, and directive metadata used by the parser.
package scanlex

import (
	"fmt"
	"slices"
	"strings"

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
	// OPERATOR_SOURCE_KIND preserves co.lang.operator for the dedicated
	// operator-source grammar without admitting it as an ordinary BUILT_IN_KIND.
	OPERATOR_SOURCE_KIND //111
	// OPERATOR_SOURCE_CONSTANT preserves a co.operator.* property value for the
	// dedicated operator-source grammar, for the same reason OPERATOR_SOURCE_KIND
	// exists: these spellings are meaningful only inside an operator declaration,
	// so classifying them as ordinary constants would admit them as literals in
	// every expression (DECISION-OPDECL-006).
	OPERATOR_SOURCE_CONSTANT //112
)

// SpecialBuiltins lists built-in identifiers that receive special treatment during token folding.
//
// "return" is in Reserved_me, so without an entry here the fold would split the path into
// a receiver, a DOT and a BUILT_IN_METHOD, and the parser would never see the single
// BUIL_IN_STMT_EXPRS token a return statement is dispatched on. The grammar spells the
// statement as ( "this" | "self" ), ".return", so BOTH receivers need an entry.
var SpecialBuiltins []string = []string{"this.return", "self.return"}

// Built_in_constants maps co.const constant names to their literal values.
var Built_in_constants map[string]string = map[string]string{
	"co.const.true":  "true",
	"co.const.false": "false",
	"co.const.none":  "none",
}

// Operator_source_constants maps each co.operator.* property value to the bare
// name the operator source parser records for it.
//
// DECISION-OPDECL-006 replaced the bare keyword spellings — infix, left, binary —
// with these qualified constants, so an operator property value is now spelled
// exactly like the co.const.true and co.const.none that appear in the same
// property list. The withdrawal is what removes the contextual keywords: `infix`
// is once again nothing but an ordinary identifier and can never collide with
// one.
//
// The set is closed and comes from operator-fixity, operator-associativity and
// operator-arity. A fixity the alpha profile has not implemented is still listed
// here, because the parser must recognize the spelling in order to report it as
// reserved rather than as unknown.
var Operator_source_constants map[string]string = map[string]string{
	"co.operator.fixity.infix":         "infix",
	"co.operator.fixity.postfix":       "postfix",
	"co.operator.fixity.prefix":        "prefix",
	"co.operator.fixity.circumfix":     "circumfix",
	"co.operator.fixity.postcircumfix": "postcircumfix",
	"co.operator.fixity.precircumfix":  "precircumfix",
	"co.operator.fixity.mixfix":        "mixfix",
	"co.operator.fixity.ternary":       "ternary",
	"co.operator.fixity.distfix":       "distfix",
	"co.operator.associativity.left":   "left",
	"co.operator.associativity.right":  "right",
	"co.operator.associativity.none":   "none",
	"co.operator.arity.unary":          "unary",
	"co.operator.arity.binary":         "binary",
	"co.operator.arity.ternary":        "ternary",
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
var Reserved_me []string = []string{}

// IsReservedMethod reports whether name is one of FoLang's lexically reserved
// built-in method candidates. It is the shared query for token folding and for
// parser checks that must accept a reserved method regardless of how its receiver
// was folded; callers still need semantic resolution to confirm applicability to
// a particular receiver type.
func IsReservedMethod(name string) bool {
	return slices.Contains(Reserved_me, name)
}

var Special_methods []string = []string{
	"@@new",
	"@@init",
}

// Built_in_stmt_exprs maps namespace prefixes to their valid sub-methods and statement expressions.
var Built_in_stmt_exprs map[string][]string = map[string][]string{
	"co.native": {"load", "register", "asm", "inline", "emit", "ffi"},
	//## turbo pascal like machine code (__asm(".byte ....."))
	////#pascal emit($5B/$59/$0E/$E8/$00/$00/$58/$05/$08/$00/$50/$51/$53/$CB); to
	// //# c  asm (".byte 0x5B, 0x59, 0x0E, 0xE8, 0x00, 0x00, 0x58, 0x05, 0x08, 0x00, 0x50, 0x51, 0x53, 0xCB\n\t")
	"co":         KeyWords_me["co"],
	"co.dynamic": {},
	"co.meta":    {"ast", "instrument", "transform", "augment", "reflect", "introspect", "patch", "inject", "create", "runtime"},
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
	// Capitalized per the reference's co.core row: the member is co.core.List, not co.core.list.
	"co.core":     {"List", "Set", "Map", "Tree", "Trie", "Sort", "Search", "Array", "Pointer", "Ref", "Address", "Ptr", "Matrix", "Word"},
	"co.lang":     {},
	"co.sys":      {"file", "concurrent", "parallel", "goto", "event", "invoke", "bind", "call", "apply", "settimeout", "setinterval", "schedular", "cron", "event", "random", "timer", "date", "time"},
	"co.os":       {"signal", "cmd", "execute", "run", "env", "getenv", "setenv", "unsetenv", "sleep", "exit", "cwd", "chdir", "fork", "wait", "pipe", "dup", "dup2", "close", "readfd", "writefd", "random"},
	"co.out":      {"println", "printsp", "print", "echo"},
	"co.in":       {"read", " readln", "input"},
	"co.sys.file": {"write", "read", "open", "close", "append", "delete", "copy", "move", "exists"},
	"co.encoding": {"json", "bson", "base64encode", "base64decode", "yml"},
	"co.dap":      {},
	"co.crypto":   {"hash", "md5", "aes", "rsa", "ssl", "tls", "uuid", "rand"},
	"co.ddap":     {},
	"co.pdap":     {},
	"co.net":      {"tcp", "udp", "http"},
	"co.const":    {"true", "false", "none"},
	"co.pattern":  {"match", "case", "default", "regex", "stex", "Type", "Value", "Shape", "Object", "Instance", "Any"},
	"co.control":  {"do", "if", "else", "otherwise", "default", "return", "shift", "resume"},
	"co.macro":    {"quote", "esc", "gensym", "unquote"},
	// The co.operator namespace supplies the qualified operator property values
	// of DECISION-OPDECL-006. The leaf spellings must match operator-fixity,
	// operator-associativity and operator-arity exactly; see
	// Operator_source_constants, which is the authority the folder consults.
	"co.operator":               {"arity", "fixity", "associativity"},
	"co.operator.fixity":        {"prefix", "infix", "postfix", "circumfix", "postcircumfix", "precircumfix", "mixfix", "ternary", "distfix"},
	"co.operator.arity":         {"unary", "binary", "ternary"},
	"co.operator.associativity": {"left", "right", "none"},
	"let":                       KeyWords_me["let"],
	"for":                       KeyWords_me["for"],
	"this":                      KeyWords_me["this"],
	// return-statement is ( "this" | "self" ), ".return", so the two receivers fold
	// identically. Without this key the fold never enters the built-in branch for a
	// self.* path and splits it into a receiver, a DOT and a method instead.
	"self":      KeyWords_me["this"],
	"_object":   Reserved_me,
	"_instance": Reserved_me,
	"_class":    {},
	"_type":     {},
	"_kind":     {},
	"_module":   {},
	"_package":  {},
	"_function": {},
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
	"co.lang.just",
	"co.lang.typed",
	"co.lang.untyped", //emulating templates in nim
	"co.mem.region",
	"co.lang.nothing",
	// A dependent type is both a declaration/result kind and a usable type. The
	// scanner still emits the overlapping BUILT_IN_KIND token; the parser resolves
	// that token contextually when it follows a variable name.
	"co.lang.dependentType",
	"co.lang.word",
	"co.lang.MatchBindings",
	"co.lang.tag",
	"co.lang.typevalue",
	"co.lang.pointer",
	"co.lang.address",
	"co.lang.reference",
	"co.lang.thunk",
	"co.lang.array",
	"co.lang.literal",
	"co.lang.uninit",
	"co.lang.range",
	"co.lang.slice",
	// co.lang.operator belongs exclusively to the dedicated operator-source
	// grammar. Ordinary token folding must not route it as a declarable kind.
	"co.lang.operator",
	"co.lang.typeclass",
	"co.lang.typeconstructor",
	"co.lang.typefunction",
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
	// co.lang.realm was WITHDRAWN by DECISION-PKG-005. It never had a production,
	// a decision, or a headed example in language-ref.md; it appeared only in a
	// summary table, which is not normative. It is absent from this table too, so
	// the spelling is now an ordinary identifier rather than a reserved kind.
	"co.lang.loader",
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
	"co.lang.module",
	"co.lang.macro",
	"co.lang.template",
	"co.lang.lambda",
	"co.lang.block",
	"co.lang.behavior",
	"co.lang.component",
	"co.lang.signature",
	"co.lang.function",
	"co.lang.method",
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
	"co.lang.associatedType",
	"co.lang.hokrlt",
	"co.lang.data",
	"co.lang.enum",
	"co.lang.typetype",
	"co.lang.typekind",
	"co.lang.alias",
	"co.lang.symbol",
	"co.lang.reservedkeyword",
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

// PDADs is the predefined built-in metadata registry: the complete, CLOSED set of
// language-owned `@co.*` metadata names, grouped by category.
//
// It is the reference's "Built-in Metadata Registry" table verbatim, which the
// consolidated grammar mirrors as builtin-pragma-name, builtin-directive-name,
// builtin-annotation-name and builtin-decorator-name. Keeping it verbatim is the
// point: after reading a metadata name the parser must match the COMPLETE name
// against this registry, and an unregistered `@co.*` name is a parse error rather
// than a user annotation the symbol table might later resolve
// (docs/language-ref.md, "Built-in Metadata Parsing").
//
// The registry closes form NAMES, not fields. Every field of a recognized form is
// parsed and preserved; the frontend validates the fields it knows and leaves the
// rest for later stages.
//
// The execution kinds a previous revision listed here as decorators — async,
// thread, task, fiber, process, coroutine, goroutine and the rest — are gone
// rather than renamed. FoLang now expresses every non-default execution model
// through one decorator, `@co.dap.executionmodel(type=…, kind=…)`, so a separate
// per-kind spelling would be a second way to say the same thing.
var PDADs map[DirectiveKind][]string = map[DirectiveKind][]string{
	PRAGMA: []string{"@co.pdap.threadpool", "@co.pdap.schedularpool"},
	DIRECTIVE: []string{"@co.ddap.import", "@co.ddap.dynamicruntime",
		"@co.ddap.use", "@co.ddap.alias", "@co.ddap.dynamicdispatch",
		"@co.ddap.overload"},
	ANNOTATION: []string{"@co.dap.template", "@co.dap.macro",
		"@co.dap.operator", "@co.dap.annotation", "@co.dap.library",
		"@co.dap.module", "@co.dap.native", "@co.dap.class", "@co.dap.static",
		"@co.dap.instance", "@co.dap.object", "@co.dap.inline",
		"@co.dap.ctfe", "@co.dap.friend", "@co.dap.sealed", "@co.dap.extension",
		"@co.dap.override", "@co.dap.virtual", "@co.dap.abstract",
		"@co.dap.delegate", "@co.dap.dynamicscope", "@co.dap.lexicalscope",
		"@co.dap.staticscope", "@co.dap.mixedscope", "@co.dap.typeclass",
		"@co.dap.matcher", "@co.dap.constructor", "@co.dap.oops",
		"@co.dap.extends", "@co.dap.hokrlt", "@co.dap.indexer",
		"@co.dap.generic", "@co.dap.comptime", "@co.dap.typefromvalue",
		"@co.dap.local", "@co.dap.private", "@co.dap.public", "@co.dap.package",
		"@co.dap.protected", "@co.dap.internal", "@co.dap.export",
		"@co.dap.eager", "@co.dap.lazy", "@co.dap.packed", "@co.dap.declare",
		"@co.dap.simd", "@co.dap.reflection", "@co.dap.mop", "@co.dap.nested",
		"@co.dap.inner", "@co.dap.final", "@co.dap.const", "@co.dap.decorator",
		"@co.dap.specialize",
	},
	//mop => meta object programming
	DECORATOR: []string{"@co.dap.before", "@co.dap.after",
		"@co.dap.around", "@co.dap.onErrExcept", "@co.dap.InvokeAlways",
		"@co.dap.HandleEffect", "@co.dap.defer", "@co.dap.callable",
		"@co.dap.executionmodel"},
}

// builtinMetadataNames indexes PDADs for the per-name lookup the parser makes on
// every metadata application.
var builtinMetadataNames = func() map[string]DirectiveKind {
	index := map[string]DirectiveKind{}
	for kind, names := range PDADs {
		for _, name := range names {
			index[name] = kind
		}
	}
	return index
}()

// IsBuiltinMetadataName reports whether name is registered in the predefined
// built-in metadata registry.
//
// Implements: builtin-metadata-name
func IsBuiltinMetadataName(name string) bool {
	_, ok := builtinMetadataNames[name]
	return ok
}

// IsLanguageOwnedMetadataName reports whether name is spelled in the
// language-owned `@co.` namespace, whether or not it is registered.
//
// The distinction is what makes an unregistered name an error rather than a
// custom annotation: a non-`co.*` name is collected as custom metadata and
// resolved later through the symbol table, while a `co.*` name that is not in the
// registry names nothing the language defines.
func IsLanguageOwnedMetadataName(name string) bool {
	return strings.HasPrefix(name, "@co.")
}

// IsBuiltinDirectiveMetadataName reports whether name is registered as a built-in
// DIRECTIVE.
//
// Implements: builtin-directive-name
func IsBuiltinDirectiveMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == DIRECTIVE
}

// IsBuiltinPragmaMetadataName reports whether name is registered as a built-in
// PRAGMA.
//
// Implements: builtin-pragma-name
func IsBuiltinPragmaMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == PRAGMA
}

// IsBuiltinAnnotationMetadataName reports whether name is registered as a
// built-in ANNOTATION.
//
// Implements: builtin-annotation-name
func IsBuiltinAnnotationMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == ANNOTATION
}

// IsBuiltinDecoratorMetadataName reports whether name is registered as a built-in
// DECORATOR.
//
// Implements: builtin-decorator-name
func IsBuiltinDecoratorMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == DECORATOR
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
	// BoundaryBefore and BoundaryAfter retain whether the original source had
	// whitespace, a comment, or a delimiter immediately on that side. The parser
	// uses these flags only when a multi-symbol token is an expression operator;
	// structural uses of the same spelling remain exempt (DECISION-LEX-010).
	BoundaryBefore bool
	BoundaryAfter  bool
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
	Kind: INVALID, Value: "Invalid", StartPos: helpers.NilPosition, EndPos: helpers.NilPosition,
}

// NewUniqueToken creates a new Token with the given kind, value, and position range.
func NewUniqueToken(kind TokenKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return newUniqueToken(kind, value, startPos, endPos)
}
func newUniqueToken(kind TokenKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		Kind: kind, Value: value, StartPos: startPos, EndPos: endPos,
	}
}

func newDummyToken(value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		Kind: INVALID, Value: value, StartPos: startPos, EndPos: endPos,
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
	case DOLLAR:
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
	case MINUS_ARROW_GT:
		return "minus_arrow_gt"
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

// Debug prints the token value and kind to stdout for debugging.
func (token Token) Debug() {
	fmt.Printf("%s => (%s)\n", token.Value, TokenKindString(token.Kind))
}

// AllowedOps lists the token kinds that may legally precede a keyword or reserved word.
var AllowedOps = []TokenKind{OPEN_BRACKET, OPEN_CURLY, OPEN_PAREN, DOT, ASSIGNMENT, CLOSE_CURLY, CLOSE_BRACKET, CLOSE_PAREN, PIPE, AMPS, POW}
