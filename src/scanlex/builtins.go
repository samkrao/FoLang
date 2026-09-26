package scanlex

import (
	"slices"
	"strings"

	"github.com/samkrao/fo-lang/src/builtins"
)

var LIB_KINDS = []string{"application", "dynamicvmrt", "native"}

// Built_in_stmt_exprs maps namespace prefixes to their valid sub-methods and statement expressions.
var Built_in_stmt_exprs map[string][]string = map[string][]string{
	"co.native": {"load", "register", "asm", "inline", "emit", "ffi", "spawnon", "arch"},
	//## turbo pascal like machine code (__asm(".byte ....."))
	////#pascal emit($5B/$59/$0E/$E8/$00/$00/$58/$05/$08/$00/$50/$51/$53/$CB); to
	// //# c  asm (".byte 0x5B, 0x59, 0x0E, 0xE8, 0x00, 0x00, 0x58, 0x05, 0x08, 0x00, 0x50, 0x51, 0x53, 0xCB\n\t")
	"co":         KeyWords_me["co"],
	"co.dynamic": {},
	"co.http":    {},
	"co.tcp":     {},
	"co.udp":     {},
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

	*/
	"co.runtime":     {},
	"co.compiletime": {},
	"co.hokrlt":      {},
	"co.cpca":        {},
	"co.utils":       {"makeImmutable", "makeShared", "copyOnWrite", "toSnapshot"},
	"co.sys":         {"file", "concurrent", "parallel", "goto", "invoke", "bind", "call", "apply", "settimeout", "setinterval", "scheduler", "cron", "event"},
	"co.os":          {"signal", "cmd", "execute", "run", "env", "getenv", "setenv", "sleep", "exit", "cwd", "chdir", "fork", "wait", "pipe", "dup", "dup2", "close", "readfd", "writefd", "random"},
	"co.out":         {"println", "print"},
	"co.in":          {"read", "readln"},
	"co.sys.file":    {},
	"co.encoding":    {"base64Encode", "base64Decode", "json", "yml", "bson"},
	"co.dap":         {},
	"co.crypto":      {"hash", "md5", "aes", "rsa", "ssl", "tls", "uuid", "rand"},
	"co.ddap":        {},
	"co.pdap":        {},
	"co.regex":       {"pattern", "match", "search"},
	"co.const":       {"true", "false", "none"},
	"co.pattern":     {},
	"co.control":     {},
	"co.macro":       {},
	// The co.operator namespace supplies the qualified operator property values
	// of DECISION-OPDECL-006. The leaf spellings must match operator-fixity,
	// operator-associativity and operator-arity exactly; see
	// Operator_source_constants, which is the authority the folder consults.
	"co.hw":                     {"cpu", "memory"},
	"co.stex":                   {},
	"co.operator":               {"arity", "fixity", "associativity"},
	"co.operator.fixity":        {"prefix", "infix", "postfix"},
	"co.operator.arity":         {"unary", "binary"},
	"co.operator.associativity": {"left", "right", "none"},
}

// Builtin_types lists the direct co.* built-in data types and type constructors
// defined by the current language reference.
var Builtin_types []string = []string{
	"co.string", "co.int", "co.bit", "co.double", "co.float", "co.long",
	"co.byte", "co.char", "co.any", "co.bool", "co.void", "co.value",
	"co.untyped", "co.word", "co.MatchBindings", "co.number", "co.uninit",
	"co.error", "co.AbstractError", "co.literal", "co.delegate", "co.condition",
	"co.variants", "co.tag", "co.hokrlt", "co.newtype", "co.opaquetype",
	"co.subtype", "co.supertype", "co.dependentType", "co.polymorphic",
	"co.refinementType", "co.associatedType", "co.predicateType", "co.data",
	"co.type", "co.generic", "co.shape",
}

// Builtin_Kinds lists the direct co.* declaration kinds.
var Builtin_Kinds []string = []string{
	"co.struct", "co.cstruct", "co.class", "co.interface", "co.union",
	"co.object", "co.instance", "co.matcher", "co.loader", "co.trait",
	"co.mixin", "co.extension", "co.typeclass", "co.module", "co.unit",
	"co.block", "co.kind", "co.signature", "co.function", "co.callable",
	"co.boundcallable", "co.enum", "co.symbol", "co.component",
}

var Built_In_Collections = []string{
	"co.List", "co.Set", "co.Map", "co.Tree", "co.Trie", "co.Array",
	"co.Tuple", "co.Comparable", "co.Stack", "co.Queue", "co.StructObject",
	"co.ClassObject", "co.ModuleObject", "co.InstanceObject", "co.ObjectObject",
	"co.Matrix",
}

// KeyWords_me maps each keyword to its language-owned qualified roots. `this`
// member access and fΦλ private-package access remain ordinary dotted syntax;
// keeping their lists empty prevents those chains from being collapsed into a
// built-in token.
var KeyWords_me map[string][]string = map[string][]string{
	"co":   {"http", "tcp", "udp", "sys", "os", "meta", "native", "in", "out", "regex", "crypto", "dap", "ddap", "pdap", "const", "encoding", "utils", "dynamic", "runtime", "compiletime", "macro", "pattern", "control", "cpca", "hokrlt", "operator", "hw", "stex"},
	"this": {},
	"fΦλ":  {},
}

var Special_methods []string = []string{
	"@@new",
	"@@init",
}

// Reserved_me lists method and keyword names reserved for built-in object operations.
var Reserved_me []string = []string{}

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
var PDADs map[builtins.DirectiveKind][]string = map[builtins.DirectiveKind][]string{
	builtins.PRAGMA: []string{"@co.pdap.threadpool", "@co.pdap.schedularpool"},
	builtins.DIRECTIVE: []string{"@co.ddap.import", "@co.ddap.dynamicruntime",
		"@co.ddap.use", "@co.ddap.alias", "@co.ddap.dynamicdispatch",
		"@co.ddap.overload"},
	builtins.ANNOTATION: []string{"@co.dap.template", "@co.dap.macro",
		"@co.dap.extend", "@co.dap.operator", "@co.dap.annotation", "@co.dap.library",
		"@co.dap.native", "@co.dap.class", "@co.dap.static",
		"@co.dap.object", "@co.dap.inline",
		"@co.dap.ctfe", "@co.dap.friend", "@co.dap.sealed", "@co.dap.extension",
		"@co.dap.override", "@co.dap.implement", "@co.dap.virtual", "@co.dap.abstract",
		"@co.dap.delegate", "@co.dap.scope", "@co.dap.typeclass", "@co.dap.matcher", "@co.dap.constructor", "@co.dap.oops",
		"@co.dap.extends", "@co.dap.hokrlt", "@co.dap.indexer", "@co.dap.generic", "@co.dap.comptime", "@co.dap.typefromvalue",
		"@co.dap.local", "@co.dap.private", "@co.dap.public", "@co.dap.compose", "@co.dap.guard", "@co.dap.package",
		"@co.dap.protected", "@co.dap.internal", "@co.dap.export",
		"@co.dap.eager", "@co.dap.lazy", "@co.dap.packed", "@co.dap.declare",
		"@co.dap.implementation", "@co.dap.simd", "@co.dap.reflection", "@co.dap.mop", "@co.dap.nested",
		"@co.dap.inner", "@co.dap.final", "@co.dap.const", "@co.dap.decorator",
		"@co.dap.specialize", "@co.dap.symbol", "@co.dap.with",
	},
	//mop => meta object programming
	builtins.DECORATOR: []string{"@co.dap.before", "@co.dap.after",
		"@co.dap.around", "@co.dap.effects", "@co.dap.onEffect", "@co.dap.defer", "@co.dap.callable",
		"@co.dap.executionmodel"},
}

// IsReservedMethod reports whether name is one of FoLang's lexically reserved
// built-in method candidates. It is the shared query for token folding and for
// parser checks that must accept a reserved method regardless of how its receiver
// was folded; callers still need semantic resolution to confirm applicability to
// a particular receiver type.
func IsReservedMethod(name string) bool {

	return slices.Contains(Reserved_me, name)
}

var KindToString map[builtins.DirectiveKind]string = map[builtins.DirectiveKind]string{
	builtins.PRAGMA:     "PRAGMA",
	builtins.DIRECTIVE:  "DIRECTIVE",
	builtins.ANNOTATION: "ANNOTATION",
	builtins.DECORATOR:  "DECORATOR",
	builtins.Invalid:    "INVALID",
}
var KindToPhase map[builtins.DirectiveKind]string = map[builtins.DirectiveKind]string{
	builtins.PRAGMA:     "COMPILE",
	builtins.DIRECTIVE:  "COMPILE",
	builtins.ANNOTATION: "RUNTIME",
	builtins.DECORATOR:  "RUNTIME",
	builtins.Invalid:    "INVALID",
}
var KindToScope map[builtins.DirectiveKind]string = map[builtins.DirectiveKind]string{
	builtins.PRAGMA:     "ENTRY_OR_LIB",
	builtins.DIRECTIVE:  "PACKAGE",
	builtins.ANNOTATION: "ANY",
	builtins.DECORATOR:  "FUN_OR_METH",
	builtins.Invalid:    "INVALID",
}

// builtinMetadataNames indexes PDADs for the per-name lookup the parser makes on
// every metadata application.
var builtinMetadataNames = func() map[string]builtins.DirectiveKind {
	index := map[string]builtins.DirectiveKind{}
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
//
// The bare root counts. `co` is a hard-reserved word and the built-in package
// root, so `@co` can never resolve to a user-defined annotation or decorator
// through the symbol table; leaving it to that path would defer a name that has
// no possible resolution, and would accept `@co` while rejecting `@co.dap`.
func IsLanguageOwnedMetadataName(name string) bool {
	return name == "@co" || strings.HasPrefix(name, "@co.")
}

// IsBuiltinDirectiveMetadataName reports whether name is registered as a built-in
// DIRECTIVE.
//
// Implements: builtin-directive-name
func IsBuiltinDirectiveMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == builtins.DIRECTIVE
}

// IsBuiltinPragmaMetadataName reports whether name is registered as a built-in
// PRAGMA.
//
// Implements: builtin-pragma-name
func IsBuiltinPragmaMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == builtins.PRAGMA
}

// IsBuiltinAnnotationMetadataName reports whether name is registered as a
// built-in ANNOTATION.
//
// Implements: builtin-annotation-name
func IsBuiltinAnnotationMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == builtins.ANNOTATION
}

// IsBuiltinDecoratorMetadataName reports whether name is registered as a built-in
// DECORATOR.
//
// Implements: builtin-decorator-name
func IsBuiltinDecoratorMetadataName(name string) bool {
	kind, ok := builtinMetadataNames[name]
	return ok && kind == builtins.DECORATOR
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
func Built_in_directive_kind(dirname string) (builtins.DirectiveKind, bool) {

	for k, v := range PDADs {
		if slices.Contains(v, dirname) {
			return k, true
		}
	}
	return builtins.Invalid, false
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
// The set is closed and comes from the current operator parse table and operator
// declaration examples. Removed future fixities are deliberately absent: the
// 1.0 reference says a spelling is not implicitly reserved merely because an
// older design or derived grammar mentioned it.
var Operator_source_constants map[string]string = map[string]string{
	"co.operator.fixity.infix":        "infix",
	"co.operator.fixity.postfix":      "postfix",
	"co.operator.fixity.prefix":       "prefix",
	"co.operator.associativity.left":  "left",
	"co.operator.associativity.right": "right",
	"co.operator.associativity.none":  "none",
	"co.operator.arity.unary":         "unary",
	"co.operator.arity.binary":        "binary",
}

// SpecialBuiltins lists dotted built-in identifiers that require indivisible
// token folding. Symbolic control statements no longer need such an exception.
var SpecialBuiltins []string

// Built_in_constants maps co.const constant names to their literal values.
var Built_in_constants map[string]string = map[string]string{
	"co.const.true":  "true",
	"co.const.false": "false",
	"co.const.none":  "none",
}

// Reserved_lu maps reserved language keywords to their TokenKind.
var Reserved_lu map[string]builtins.TokenKind = map[string]builtins.TokenKind{
	"co":   builtins.KEYWORD,      // holds everything
	"this": builtins.KEYWORD,      // refers this/self
	"fΦλ":  builtins.RESERVEDWORD, // fo-lang reserved word
}

// UnsupportedObjects lists reserved spellings whose dotted forms must remain
// visible to the parser as one UNKNOWN lexeme instead of being folded. The
// current reference defines no such spelling.
var UnsupportedObjects = []string{}

// isSpecialBuiltin compares source and internally lowered identifier spellings.
// Folding may already have appended _fo to one or more path segments when this
// decision is made; that implementation suffix must not turn a statement form
// such as a language-owned dotted built-in into an ordinary method call.
func IsSpecialBuiltin(name string) bool {
	logical := strings.ReplaceAll(name, "_fo.", ".")
	logical = strings.TrimSuffix(logical, "_fo")
	return slices.Contains(SpecialBuiltins, logical)
}

// AllowedOps lists the token kinds that may legally precede a keyword or reserved word.
var AllowedOps = []builtins.TokenKind{builtins.OPEN_BRACKET, builtins.OPEN_CURLY, builtins.OPEN_PAREN, builtins.DOT, builtins.ASSIGNMENT, builtins.CLOSE_CURLY, builtins.CLOSE_BRACKET, builtins.CLOSE_PAREN, builtins.PIPE, builtins.AMPS, builtins.POW}
