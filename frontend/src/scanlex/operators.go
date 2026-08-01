package scanlex

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// User-defined operator symbols.
//
// A built-in operator is recognised by the byte switch in scanner.go, where each
// spelling is reached from the character that begins it. That works because the set is
// closed and known while the scanner is being written. A user-defined operator is
// neither: `@co.dap.operator(symbol='∪', mode=define, …)` introduces a spelling the
// scanner has never seen, and without help it is split into whatever built-in pieces it
// happens to contain — "<+>" becomes "<", "+", ">" — so the operator can be declared but
// never used.
//
// CustomOperators is the map that closes that gap. It is filled before scanning with the
// symbols the compilation unit itself declares, and consulted at every position where an
// operator could begin. An operator is not imported: it is declared in the companion
// unit of the struct it belongs to, or inside a class, so the declaring unit is the whole
// of what is in scope.
//
// Two rules keep it safe:
//
//   - LONGEST MATCH WINS, across the custom set and the built-in spellings together. A
//     custom symbol may extend a built-in one — declaring "<=>" must not stop "<=" from
//     lexing — so the decision is made on length, never on which table was consulted
//     first.
//   - WHITESPACE ENDS A LEXEME. Only operator characters are accumulated, and never
//     across a space, so `a < b` can never be read as one symbol and an operator can
//     never absorb the operand beside it.
type CustomOperators struct {
	// symbols holds each declared spelling. A set rather than a list: the scanner
	// only asks whether a candidate span is one.
	symbols map[string]struct{}
	// maxLen is the longest declared symbol in bytes, which bounds the candidate
	// span so a long run of operator characters costs no more than that.
	maxLen int
}

// NewCustomOperators builds an operator set from declared symbols.
//
// A symbol that is empty, that contains a character an operator cannot be spelled with,
// or that duplicates a built-in spelling is ignored: an overload of a built-in keeps the
// built-in token, so it needs nothing here.
func NewCustomOperators(symbols []string) *CustomOperators {
	set := &CustomOperators{symbols: map[string]struct{}{}}
	for _, s := range symbols {
		if s == "" || !IsOperatorSpelling(s) || builtinOperatorSpellings[s] {
			continue
		}
		set.symbols[s] = struct{}{}
		if len(s) > set.maxLen {
			set.maxLen = len(s)
		}
	}
	return set
}

// Empty reports whether any custom operator is registered, which lets the scanner skip
// the lookup entirely for the overwhelmingly common case.
func (c *CustomOperators) Empty() bool { return c == nil || len(c.symbols) == 0 }

// match returns the length of the longest declared symbol at the cursor, or 0.
//
// The candidate span runs to the first character that cannot appear in an operator —
// whitespace included — so the search never crosses a lexeme boundary.
func (c *CustomOperators) match(src string) int {
	if c.Empty() {
		return 0
	}

	limit := operatorRunLength(src)
	if limit > c.maxLen {
		limit = c.maxLen
	}

	// Longest first, so a declared "<+>" is preferred over a declared "<+".
	for n := limit; n > 0; n-- {
		if _, ok := c.symbols[src[:n]]; ok {
			return n
		}
	}
	return 0
}

// operatorRunLength returns the byte length of the run of operator characters at the
// cursor. It stops at the first character that cannot spell an operator, which is what
// makes whitespace — and a letter, a digit, a bracket or a quote — a lexeme boundary.
func operatorRunLength(src string) int {
	n := 0
	for n < len(src) {
		r, size := utf8.DecodeRuneInString(src[n:])
		if size == 0 || !isOperatorRune(r) {
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
	for _, r := range s {
		if !isOperatorRune(r) {
			return false
		}
	}
	return true
}

// isOperatorRune reports whether r may appear in an operator spelling.
//
// The ASCII set is the punctuation the built-in operators are drawn from, minus the
// characters that delimit or group — brackets, braces, parens, quotes, the comma and the
// semicolon — because those end a lexeme rather than continue one. Beyond ASCII, the
// Unicode symbol and punctuation categories are admitted, which is what lets the
// reference's own `∪` and `∩` be operators.
func isOperatorRune(r rune) bool {
	if r < utf8.RuneSelf {
		return strings.ContainsRune(asciiOperatorChars, r)
	}
	return unicode.IsSymbol(r) || unicode.IsPunct(r)
}

const asciiOperatorChars = `+-*/%<>=!&|^~?:.@#$`

// builtinOperatorSpellings is every spelling the byte switch already produces.
//
// A custom symbol identical to one of these is not registered: DECISION-EXT-001 says an
// overload of a built-in symbol keeps the built-in binding, so it must keep the built-in
// token too. Listing them here also lets a longest-match comparison prefer a built-in
// when a custom symbol is the same length.
var builtinOperatorSpellings = map[string]bool{
	"+": true, "-": true, "*": true, "/": true, "%": true,
	"++": true, "--": true, "+=": true, "-=": true,
	"=": true, "==": true, "!=": true, "!": true,
	"<": true, ">": true, "<=": true, ">=": true,
	"&&": true, "||": true, "&": true, "|": true,
	"^": true, "~": true, "~~": true, "#": true, "@": true, "@@": true,
	".": true, "..": true, "...": true, "..<": true,
	"<..": true, "<..<": true, ":": true, ":=": true, "::=": true,
	"->": true, "->>": true, "<-": true, "<->": true,
	"=>": true, "=>>": true, "==>>": true,
	"?": true, "?=": true, "??=": true, "$": true, "`": true,
}
