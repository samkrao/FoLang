package scanlex

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// The scanner: one dispatch on the leading byte, no regular expressions.
//
// This replaces a table of ~80 regexes that was tried linearly at every source
// position. Ordering in that table silently carried meaning — the first pattern that
// matched at the cursor won, so "==>>" had to precede "==", and "[:]" had to precede
// "[" — which made the lexical rules impossible to read off and easy to break by
// inserting a rule in the wrong place. Here each rule is reached from the byte that
// begins it, and the only ordering that remains is the explicit longest-first testing
// inside a single case, which is what DECISION-LEX-003 maximal munch requires.
//
// The token stream is unchanged: the same kinds, the same lexemes, the same spans, and
// the same NEWLINE tokens for cleanupLB to drop and foldTokens to fold afterwards.
// Nothing downstream of Tokenize sees a difference.

// scanAction says what the driver should do with a scanned span.
type scanAction int

const (
	// actionEmit pushes a token of the scanned kind and lexeme.
	actionEmit scanAction = iota
	// actionSkip consumes the span without producing a token: whitespace and
	// comments. lines records how many line breaks it covered.
	actionSkip
	// actionNewline consumes a line break and pushes the NEWLINE token the old
	// scanner pushed, which cleanupLB removes once scanning is complete.
	actionNewline
	// actionError reports a diagnostic for the span and consumes it.
	actionError
)

// scanned is one lexical decision: what was matched, how long it is, and what to do.
type scanned struct {
	action  scanAction
	kind    TokenKind
	length  int
	lines   int
	message string
	errType helpers.ErrorType
}

func emit(kind TokenKind, length int) scanned {
	return scanned{action: actionEmit, kind: kind, length: length}
}

func skip(length int) scanned { return scanned{action: actionSkip, length: length} }

// scanToken examines the source at the cursor and decides the next lexical unit.
//
// src is the remainder of the source. The returned length is always at least one, so
// the driver always makes progress.
func (lex *lexer) scanToken(src string) (scanned, bool) {
	c := src[0]

	switch {
	// ---- line breaks -----------------------------------------------------
	// One byte at a time, including for a CRLF pair: the previous scanner matched
	// "\r\n" but advanced a single byte, so a CRLF produced two NEWLINE tokens. The
	// parser normalizes line endings before scanning, so nothing reaches here with a
	// CR in practice, and this keeps a direct Tokenize caller's stream unchanged.
	case c == '\r' || c == '\n':
		return scanned{action: actionNewline, length: 1}, true

	// ---- horizontal white space: " " | "\t" | "\f" -----------------------
	case c == ' ' || c == '\t' || c == '\f':
		n := 0
		for n < len(src) && (src[n] == ' ' || src[n] == '\t' || src[n] == '\f') {
			n++
		}
		return skip(n), true

	// ---- comments and the slash operator ---------------------------------
	case c == '/':
		if strings.HasPrefix(src, "//") {
			n := strings.IndexAny(src, "\r\n")
			if n < 0 {
				n = len(src)
			}
			return skip(n), true
		}
		if strings.HasPrefix(src, "/*") {
			// block-comment ends at the FIRST "*/" and may span line breaks.
			if end := strings.Index(src[2:], "*/"); end >= 0 {
				n := 2 + end + 2
				return scanned{action: actionSkip, length: n, lines: strings.Count(src[:n], "\n")}, true
			}
			return scanned{
				action:  actionError,
				length:  len(src),
				lines:   strings.Count(src, "\n"),
				message: "unterminated block comment; a \"/*\" comment must be closed with \"*/\"",
				errType: helpers.InvalidSyntax,
			}, true
		}
		return emit(SLASH, 1), true

	// ---- string literal --------------------------------------------------
	// alpha-basic-s-character excludes CR and LF, so a string never spans a line
	// break. An unterminated one falls through to the bare DOUBL_QUOTE token the
	// old scanner produced.
	case c == '"':
		if n := stringLiteralLength(src); n > 0 {
			return emit(STRING, n), true
		}
		return emit(DOUBL_QUOTE, 1), true

	// ---- character literal -----------------------------------------------
	// alpha-basic-c-character is any character except the apostrophe, the
	// backslash, CR and LF — a space and a tab are ordinary c-characters.
	case c == '\'':
		if len(src) >= 3 && src[2] == '\'' && src[1] != '\'' && src[1] != '\\' && src[1] != '\r' && src[1] != '\n' {
			return emit(CHAR, 3), true
		}
		return emit(SINGLE_QUOTE, 1), true

	// ---- numeric literal -------------------------------------------------
	case isDigit(c):
		return emit(NUMBER, numericLiteralLength(src)), true

	// ---- "__", "_", identifiers -------------------------------------------
	// The old table tried "__" then "_" before the identifier rule, so a name that
	// begins with an underscore is never one identifier. That is preserved here.
	case c == '_':
		if strings.HasPrefix(src, "__") {
			return emit(DBL_UNDERSCORE, 2), true
		}
		return emit(DISCARD_WILD_VAR, 1), true

	case isAlpha(c):
		return emit(IDENTIFIER, identifierLength(src)), true

	// ---- special methods and annotations ---------------------------------
	case c == '@':
		if strings.HasPrefix(src, "@@") {
			// A special method is scanned WHOLE and checked against the closed
			// Special_methods set, the same way a built-in method name is resolved
			// from a table rather than accepted generically. "@@" followed by a name
			// that is not one of them has no meaning, so it is reported here instead
			// of reaching the parser as a bare "@@" plus an ordinary identifier.
			n := 2 + identifierLength(src[2:])
			if n > 2 {
				if slices.Contains(Special_methods, src[:n]) {
					return emit(SPECIAL_METHODS, n), true
				}
				return scanned{
					action: actionError,
					length: n,
					message: fmt.Sprintf("%q is not a FoLang special method; the special methods are %s",
						src[:n], strings.Join(Special_methods, ", ")),
					errType: helpers.InvalidSyntax,
				}, true
			}
			return emit(DOUBLE_AT, 2), true
		}
		if len(src) > 1 && (isAlpha(src[1]) || src[1] == '_') {
			return emit(ATDAP, 1+identifierLength(src[1:])), true
		}
		return emit(AT, 1), true

	// ---- result and self bindings: "$", "$1", "$12" ------------------------
	case c == '$':
		n := 1
		for n < len(src) && isDigit(src[n]) {
			n++
		}
		return emit(BIND_VAR, n), true

	// ---- brackets and braces ----------------------------------------------
	case c == '[':
		if strings.HasPrefix(src, "[:]") {
			return emit(OB_COLON_CB, 3), true
		}
		return emit(OPEN_BRACKET, 1), true
	case c == ']':
		return emit(CLOSE_BRACKET, 1), true
	case c == '{':
		return emit(OPEN_CURLY, 1), true
	case c == '}':
		return emit(CLOSE_CURLY, 1), true
	case c == '(':
		return emit(OPEN_PAREN, 1), true
	case c == ')':
		return emit(CLOSE_PAREN, 1), true

	// ---- operators, longest spelling first --------------------------------
	case c == '=':
		switch {
		case strings.HasPrefix(src, "==>>"):
			return emit(EQEQGTGT, 4), true
		case strings.HasPrefix(src, "=>>"):
			return emit(EQGTGT, 3), true
		case strings.HasPrefix(src, "=="):
			return emit(EQUALS, 2), true
		case strings.HasPrefix(src, "=>"):
			return emit(EQGT, 2), true
		}
		return emit(ASSIGNMENT, 1), true

	case c == '!':
		if strings.HasPrefix(src, "!=") {
			return emit(NOT_EQUALS, 2), true
		}
		return emit(NOT, 1), true

	case c == '<':
		switch {
		case strings.HasPrefix(src, "<..<"):
			return emit(LT_DOT_DOT_LT, 4), true
		case strings.HasPrefix(src, "<.."):
			return emit(LT_DOT_DOT, 3), true
		case strings.HasPrefix(src, "<->"):
			return emit(BIDIR_ARROW, 3), true
		case strings.HasPrefix(src, "<-"):
			return emit(LEFT_ARROW, 2), true
		case strings.HasPrefix(src, "<="):
			return emit(LESS_EQUALS, 2), true
		}
		return emit(LESS, 1), true

	case c == '>':
		if strings.HasPrefix(src, ">=") {
			return emit(GREATER_EQUALS, 2), true
		}
		return emit(GREATER, 1), true

	case c == '|':
		if strings.HasPrefix(src, "||") {
			return emit(OR, 2), true
		}
		return emit(PIPE, 1), true

	case c == '&':
		if strings.HasPrefix(src, "&&") {
			return emit(AND, 2), true
		}
		return emit(AMPS, 1), true

	case c == '.':
		switch {
		case strings.HasPrefix(src, "..."):
			return emit(DOT_DOT_DOT, 3), true
		case strings.HasPrefix(src, "..<"):
			return emit(DOT_DOT_LT, 3), true
		case strings.HasPrefix(src, ".."):
			return emit(DOT_DOT, 2), true
		}
		return emit(DOT, 1), true

	case c == ';':
		return emit(SEMI_COLON, 1), true

	case c == ':':
		switch {
		case strings.HasPrefix(src, "::="):
			return emit(COLON_WALRUS, 3), true
		case strings.HasPrefix(src, ":="):
			return emit(WALRUS, 2), true
		}
		return emit(COLON, 1), true

	case c == '-':
		switch {
		case strings.HasPrefix(src, "->>"):
			return emit(MINUS_ARROW_GT, 3), true
		case strings.HasPrefix(src, "->"):
			return emit(ARROW, 2), true
		case strings.HasPrefix(src, "--"):
			return emit(MINUS_MINUS, 2), true
		case strings.HasPrefix(src, "-="):
			return emit(MINUS_EQUALS, 2), true
		}
		return emit(MINUS, 1), true

	case c == '?':
		switch {
		case strings.HasPrefix(src, "??="):
			return emit(NULLISH_ASSIGNMENT, 3), true
		case strings.HasPrefix(src, "?="):
			return emit(QEQ, 2), true
		}
		return emit(QUESTION, 1), true

	case c == ',':
		return emit(COMMA, 1), true

	case c == '+':
		switch {
		case strings.HasPrefix(src, "++"):
			return emit(PLUS_PLUS, 2), true
		case strings.HasPrefix(src, "+="):
			return emit(PLUS_EQUALS, 2), true
		}
		return emit(PLUS, 1), true

	case c == '*':
		return emit(STAR, 1), true
	case c == '%':
		return emit(PERCENT, 1), true
	case c == '^':
		return emit(POW, 1), true
	case c == '#':
		return emit(HASH, 1), true
	case c == '`':
		// DECISION-OP-005: reserved. The parser refuses it.
		return emit(BACK_TICK, 1), true

	case c == '~':
		if strings.HasPrefix(src, "~~") {
			return emit(TILD_TILD, 2), true
		}
		return emit(TILD, 1), true
	}

	// ---- reserved-future-operator glyphs ----------------------------------
	// Multi-byte, so this is tested after every single-byte case has missed.
	if n := reservedGlyphLength(src); n > 0 {
		return scanned{
			action:  actionError,
			length:  n,
			message: fmt.Sprintf("the glyph %q is reserved for a future FoLang operator and cannot be used yet", src[:n]),
			errType: helpers.ReservedKeyword,
		}, true
	}

	return scanned{}, false
}

// stringLiteralLength returns the length of a complete string literal at the cursor,
// or 0 when the closing quote is missing before the line ends.
func stringLiteralLength(src string) int {
	for i := 1; i < len(src); i++ {
		switch src[i] {
		case '"':
			return i + 1
		case '\r', '\n':
			return 0
		}
	}
	return 0
}

// identifierLength returns the length of the identifier at the cursor. The span is
// deliberately wider than the grammar's identifier so a malformed name is consumed and
// reported whole; emitIdentifier applies DECISION-LEX-001/006.
func identifierLength(src string) int {
	n := 0
	for n < len(src) && (isAlpha(src[n]) || isDigit(src[n]) || src[n] == '_') {
		n++
	}
	return n
}

// numericLiteralLength returns the length of the COMPLETE integer-literal or
// floating-literal at the cursor, suffix included.
//
// The grammar requires the scanner to do this in one pass: with the abbreviated float
// forms gone a numeric literal can never end at a point, so "the scanner needs no
// numeric lookahead and the parser never re-lexes" (the note withdrawing
// DECISION-LEX-005).
func numericLiteralLength(src string) int {
	// Hexadecimal and binary both start "0x"/"0X" or "0b"/"0B".
	if len(src) > 1 && src[0] == '0' {
		switch src[1] {
		case 'x', 'X':
			n := 2
			d := digitRun(src, n, isHexDigit)
			if d == n { // "0x" with no digits: just the zero
				return 1
			}
			n = d
			// hexadecimal-fractional-constant, then a mandatory binary exponent
			if n < len(src) && src[n] == '.' {
				if f := digitRun(src, n+1, isHexDigit); f > n+1 {
					if e := binaryExponentLength(src, f); e > 0 {
						return floatSuffixEnd(src, f+e)
					}
				}
				return n // "0xF." is the integer then a member access
			}
			if e := binaryExponentLength(src, n); e > 0 {
				return floatSuffixEnd(src, n+e)
			}
			return intSuffixEnd(src, n)
		case 'b', 'B':
			n := digitRun(src, 2, isBinDigit)
			if n == 2 {
				return 1
			}
			return intSuffixEnd(src, n)
		}
	}

	// decimal-integer-literal and octal-integer-literal share this shape.
	n := digitRun(src, 0, isDigit)

	// fractional-constant needs a digit on BOTH sides of the point
	// (DECISION-LIT-006), so "1." stays an integer followed by a member access.
	if n < len(src) && src[n] == '.' {
		if f := digitRun(src, n+1, isDigit); f > n+1 {
			if e := exponentLength(src, f); e > 0 {
				return floatSuffixEnd(src, f+e)
			}
			return floatSuffixEnd(src, f)
		}
		return n
	}

	// decimal-digit-sequence with a mandatory exponent.
	if e := exponentLength(src, n); e > 0 {
		return floatSuffixEnd(src, n+e)
	}
	return intSuffixEnd(src, n)
}

// exponentLength returns the length of an exponent-part at i, or 0.
func exponentLength(src string, i int) int {
	return signedExponent(src, i, func(c byte) bool { return c == 'e' || c == 'E' })
}

// binaryExponentLength returns the length of a binary-exponent-part at i, or 0.
func binaryExponentLength(src string, i int) int {
	return signedExponent(src, i, func(c byte) bool { return c == 'p' || c == 'P' })
}

// signedExponent matches marker, optional sign, then a digit sequence.
func signedExponent(src string, i int, isMarker func(byte) bool) int {
	if i >= len(src) || !isMarker(src[i]) {
		return 0
	}
	n := i + 1
	if n < len(src) && (src[n] == '+' || src[n] == '-') {
		n++
	}
	d := digitRun(src, n, isDigit)
	if d == n {
		return 0
	}
	return d - i
}

// floatSuffixes and integerSuffixes are longest-first so "f128" is preferred over "f"
// and "ll" over "l".
var floatSuffixes = []string{
	"bf16", "BF16", "f128", "F128",
	"f16", "F16", "f32", "F32", "f64", "F64",
	"f", "F", "l", "L",
}

// floatSuffixEnd returns the index past an optional floating-point-suffix at i.
func floatSuffixEnd(src string, i int) int {
	for _, s := range floatSuffixes {
		if strings.HasPrefix(src[i:], s) {
			return i + len(s)
		}
	}
	return i
}

// intSuffixEnd returns the index past an optional integer-suffix at i.
//
// integer-suffix pairs an unsigned marker with at most one length marker, in either
// order: 42u, 100LL, 7uL, 12z, 9ull.
func intSuffixEnd(src string, i int) int {
	n := i
	unsigned := func() bool {
		if n < len(src) && (src[n] == 'u' || src[n] == 'U') {
			n++
			return true
		}
		return false
	}
	length := func() bool {
		if strings.HasPrefix(src[n:], "ll") || strings.HasPrefix(src[n:], "LL") {
			n += 2
			return true
		}
		if n < len(src) {
			switch src[n] {
			case 'l', 'L', 'z', 'Z':
				n++
				return true
			}
		}
		return false
	}

	if unsigned() {
		length()
	} else if length() {
		unsigned()
	}
	return n
}

// reservedGlyphs is the reserved-future-operator set of DECISION-OP-005.
var reservedGlyphs = []string{
	"λ", "⒪", "â", "Ť", "∀", "∃", "○", "ö", "∪", "Ṡ", "Ŝ", "ṁ",
	"𝚷", "⇛", "𝑓", "𝒯", "𝘷", "𝓕", "↓", "∂", "⊥", "↧", "⇓",
}

// reservedGlyphLength returns the byte length of a reserved glyph at the cursor, or 0.
func reservedGlyphLength(src string) int {
	for _, g := range reservedGlyphs {
		if strings.HasPrefix(src, g) {
			return len(g)
		}
	}
	return 0
}

func digitRun(src string, i int, isDigitFn func(byte) bool) int {
	for i < len(src) && isDigitFn(src[i]) {
		i++
	}
	return i
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isBinDigit(c byte) bool {
	return c == '0' || c == '1'
}
func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// emitNewline pushes the NEWLINE token that marks a line break.
//
// cleanupLB drops these once scanning finishes; they exist so line bookkeeping happens
// in one place. The lexeme is the literal "LSP" the previous scanner used, so a caller
// inspecting the raw stream sees exactly what it saw before.
func (lex *lexer) emitNewline(_ string) {
	start := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
	lex.posi = start
	lex.advanceN(1)
	end := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
	lex.push(newUniqueToken(NEWLINE, "LSP", start.Copy(), end))
	lex.posi = end
	lex.advanceline(1)
}

// emitToken pushes the token a scan decided on, keeping the position bookkeeping the
// handlers used: the span runs from the cursor before the lexeme to the cursor after
// it, and lex.posi is left at the end so the next token starts from there.
func (lex *lexer) emitToken(kind TokenKind, lexeme string) {
	start := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
	lex.posi = start
	lex.advanceN(len(lexeme))
	end := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)

	switch kind {
	case IDENTIFIER:
		lex.emitIdentifier(lexeme, start, end)
	default:
		lex.push(newUniqueToken(kind, lexeme, start.Copy(), end))
	}
	lex.posi = end
}

// emitIdentifier applies the identifier rules and the reserved-word classification.
//
// DECISION-LEX-001/006: an identifier carries single underscores between nonempty
// alphanumeric segments and never ends in one. Reserved_lu then gives a reserved word
// its KEYWORD or RESERVEDWORD kind instead of letting it pass as an IDENTIFIER.
func (lex *lexer) emitIdentifier(lexeme string, start, end *helpers.Position) {
	if strings.Contains(lexeme, "__") {
		foerrors.HandleErrors(lex.errorException(
			fmt.Sprintf("identifier %q contains consecutive underscores; FoLang identifiers use single underscores between alphanumeric segments", lexeme),
			helpers.InvalidSyntax, *start, *end))
	}
	if strings.HasSuffix(lexeme, "_") {
		foerrors.HandleErrors(lex.errorException(
			fmt.Sprintf("identifier %q ends in an underscore; a FoLang identifier must end in a letter or digit", lexeme),
			helpers.InvalidSyntax, *start, *end))
	}

	kind := IDENTIFIER
	if k, ok := Reserved_lu[lexeme]; ok {
		kind = k
	}
	lex.push(newUniqueToken(kind, lexeme, start.Copy(), end))
}
