package scanlex

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// User-defined operator symbols.
//
// Built-in spellings are exact entries in the scanner's whole-run table. A
// project-local operator is supplied through the compilation's operator catalog
// before ordinary source scanning begins.
//
// CustomOperators is the map that closes that gap. The parser builds it from the project
// operator catalog before scanning a compilation unit and consults it at every position
// where an operator could begin. Name resolution still decides whether a catalogued
// operator is semantically visible at a particular use site; this lexical catalog only
// classifies a valid project operator while preserving unknown runs for contextual
// parser classification.
//
// DECISION-LEX-003 makes the symbolic run, rather than the registry, the lexical
// unit. A registered symbol therefore matches only when it equals the complete
// contiguous run. There is no fallback to a shorter custom or built-in spelling.
type CustomOperators struct {
	// symbols carries the fixity needed by DECISION-LEX-010. Boundary checks
	// happen before comments and whitespace are discarded.
	symbols map[string]string
}

// OperatorSpec is the lexical portion of one registered custom operator.
// Precedence, associativity, and arity belong to parsing and resolution; the
// scanner needs only the whole symbol and fixity to enforce operand boundaries.
type OperatorSpec struct {
	Symbol string
	Fixity string
}

// NewCustomOperators builds an operator set from declared symbols.
//
// A symbol that is empty, that contains a character an operator cannot be spelled with,
// or that duplicates a built-in spelling is ignored: an overload of a built-in keeps the
// built-in token, so it needs nothing here.
//
// Deprecated: use NewCustomOperatorsWithSpecs. This legacy convenience assumes
// every supplied symbol is infix, so a multi-symbol operator requires explicit
// source boundaries on both operand-facing sides.
func NewCustomOperators(symbols []string) *CustomOperators {
	specs := make([]OperatorSpec, 0, len(symbols))
	for _, s := range symbols {
		specs = append(specs, OperatorSpec{Symbol: s, Fixity: "infix"})
	}
	return NewCustomOperatorsWithSpecs(specs)
}

// NewCustomOperatorsWithSpecs builds a fixity-aware custom-operator registry.
// Invalid, comment-opening, and language-owned spellings are ignored here and
// diagnosed by the operator-source validation pass.
func NewCustomOperatorsWithSpecs(specs []OperatorSpec) *CustomOperators {
	set := &CustomOperators{symbols: map[string]string{}}
	for _, spec := range specs {
		if spec.Symbol == "" || !IsOperatorSpelling(spec.Symbol) ||
			strings.Contains(spec.Symbol, "//") || strings.Contains(spec.Symbol, "/*") ||
			builtinOperatorSpellings[spec.Symbol] || languagePredeclaredOperatorSpellings[spec.Symbol] {
			continue
		}
		switch spec.Fixity {
		case "infix", "prefix", "postfix":
			set.symbols[spec.Symbol] = spec.Fixity
		}
	}
	return set
}

// Empty reports whether any custom operator is registered, which lets the scanner skip
// the lookup entirely for the overwhelmingly common case.
func (c *CustomOperators) Empty() bool { return c == nil || len(c.symbols) == 0 }

// match returns the registered fixity only when run is an exact whole-symbol
// match. Unknown runs are intentionally not split into shorter registered pieces.
func (c *CustomOperators) match(run string) (string, bool) {
	if c.Empty() {
		return "", false
	}
	fixity, ok := c.symbols[run]
	return fixity, ok
}

// operatorRunLength returns the byte length of the run of operator characters at the
// cursor. It stops at the first character that cannot spell an operator, which is what
// makes whitespace — and a letter, a digit, a bracket or a quote — a lexeme boundary.
func operatorRunLength(src string) int {
	n := 0
	for n < len(src) {
		// A comment opener terminates a preceding symbolic run as well as
		// winning when it occurs at the scanner cursor. Thus **/*comment*/
		// is POW("**") followed by a comment, never SYMBOLIC_RUN("**/*").
		if strings.HasPrefix(src[n:], "//") || strings.HasPrefix(src[n:], "/*") {
			break
		}
		r, size := utf8.DecodeRuneInString(src[n:])
		if size == 0 || r == utf8.RuneError && size == 1 || !isOperatorRune(r) {
			break
		}
		n += size
	}
	return n
}

// IsOperatorSpelling reports whether every character of s can appear in an operator, so
// a declaration can be rejected at its source rather than silently never matching.
func IsOperatorSpelling(s string) bool {
	if s == "" {
		return false
	}
	for offset := 0; offset < len(s); {
		r, size := utf8.DecodeRuneInString(s[offset:])
		if size == 0 || r == utf8.RuneError && size == 1 || !isOperatorRune(r) {
			return false
		}
		offset += size
	}
	return true
}

// isOperatorRune reports whether r may appear in an operator spelling.
//
// The operator-source grammar excludes ASCII letters/digits/underscore rather
// than every Unicode letter. Consequently non-ASCII mathematical modifier
// letters such as â, Ť and Ṡ are symbolic characters alongside ∪ and ⊗.
func isOperatorRune(r rune) bool {
	if r < utf8.RuneSelf {
		return strings.ContainsRune(asciiOperatorChars, r)
	}
	return !unicode.IsSpace(r)
}

const asciiOperatorChars = "+-*/%<>=!&|^~?:.@#$`\\"

// builtinSymbolKinds is the exact whole-run classification table. Entries that
// are structural or hard-reserved remain here because they are scanner-known
// spellings even though they are not expression operators.
var builtinSymbolKinds = map[string]TokenKind{
	"+": PLUS, "-": MINUS, "*": STAR, "/": SLASH, "%": PERCENT, "**": POW,
	"+=": PLUS_EQUALS, "-=": MINUS_EQUALS, "*=": ASSIGNMENT, "/=": ASSIGNMENT,
	"%=": ASSIGNMENT, "**=": ASSIGNMENT,
	"=": ASSIGNMENT, "==": EQUALS, "!=": NOT_EQUALS, "!": NOT,
	"<": LESS, ">": GREATER, "<=": LESS_EQUALS, ">=": GREATER_EQUALS,
	"&&": AND, "||": OR, "&": AMPS, "|": PIPE,
	"&=": ASSIGNMENT, "^=": ASSIGNMENT, "|=": ASSIGNMENT,
	"^": POW, "~": TILD, "#": HASH, "@": AT, "@@": DOUBLE_AT,
	".": DOT, "..": DOT_DOT, "...": DOT_DOT_DOT, "..<": DOT_DOT_LT,
	"<..": LT_DOT_DOT, "<..<": LT_DOT_DOT_LT,
	":": COLON, ":=": WALRUS, "::=": COLON_WALRUS,
	"->": ARROW, "->>": MINUS_ARROW_GT, "<-": LEFT_ARROW, "<->": BIDIR_ARROW,
	"=>": EQGT, "=>>": EQGTGT, "==>>": EQEQGTGT,
	"?": QUESTION, "?=": QEQ, "$": BIND_VAR, "`": BACK_TICK, "\\": BACK_SLASH,
}

var builtinOperatorSpellings = func() map[string]bool {
	spellings := make(map[string]bool, len(builtinSymbolKinds))
	for spelling := range builtinSymbolKinds {
		spellings[spelling] = true
	}
	return spellings
}()

// languagePredeclaredOperatorSpellings are reserved by the language rather than
// declarable by a project. Each is a valid whole symbolic run, so the lexer
// recognizes it and does not fail merely because its operator semantics are
// unimplemented; the PARSER is what rejects its use as an expression operator
// (docs/language-ref.md, C.10).
var languagePredeclaredOperatorSpellings = map[string]bool{
	"λ": true, "⒪": true, "â": true, "Ť": true, "∀": true, "∃": true,
	"○": true, "ö": true, "∪": true, "Ṡ": true, "Ŝ": true, "ṁ": true,
	"𝚷": true, "⇛": true, "𝑓": true, "𝒯": true, "𝘷": true, "𝓕": true,
	"↓": true, "∂": true, "⊥": true, "↧": true, "⇓": true,
}

// IsLanguageOwnedOperatorSpelling reports whether spelling is registered or
// reserved by the language and therefore cannot be declared in a project-local
// operator source.
//
// The pre-declared glyph set is language-reserved: a project may neither declare
// one with co.lang.operator nor supply an overload implementation for one, because
// the language has not yet enabled the operator each stands for. They tokenize
// like any other complete symbolic run so that the diagnostic can come from the
// parser rather than from a lexical failure.
func IsLanguageOwnedOperatorSpelling(spelling string) bool {
	return builtinOperatorSpellings[spelling] || languagePredeclaredOperatorSpellings[spelling]
}

// IsPredeclaredOperatorSpelling reports whether spelling is one of the
// language-predeclared operator glyphs. Unlike IsLanguageOwnedOperatorSpelling,
// it deliberately excludes built-in and structural operator spellings.
func IsPredeclaredOperatorSpelling(spelling string) bool {
	return languagePredeclaredOperatorSpellings[spelling]
}

// IsHardReservedOperatorSpelling distinguishes spellings that cannot receive
// overload implementations from ordinary built-in/pre-declared operators.
func IsHardReservedOperatorSpelling(spelling string) bool {
	switch spelling {
	case "::=", "->>", "<->", "`", "\\", "#", "//", "/*":
		return true
	default:
		return false
	}
}
