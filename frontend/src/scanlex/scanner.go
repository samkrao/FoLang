package scanlex

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// The scanner: one dispatch on the leading byte, no regular expressions.
//
// Closed composites, comments, and literals receive their specified priority.
// Every remaining sequence of symbol characters is then consumed as one complete
// run and classified by exact match; no shorter operator fallback exists.
//
// This switch is the lexical source of truth: direct Tokenize callers receive the same
// whole-run operator spellings that the parser consumes. NEWLINE tokens are still
// emitted here for cleanupLB to discard before the final stream is returned.

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
	action    scanAction
	kind      TokenKind
	length    int
	lines     int
	endColumn int
	message   string
	errType   helpers.ErrorType
}

func emit(kind TokenKind, length int) scanned {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.emit -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return scanned{action: actionEmit, kind: kind, length: length}
}

func skip(length int) scanned {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.skip -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return scanned{action: actionSkip, length: length}
}

// scanToken examines the source at the cursor and decides the next lexical unit.
//
// src is the remainder of the source. The returned length is always at least one, so
// the driver always makes progress.
func (lex *lexer) scanToken(src string) (scanned, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.scanToken -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")
	// Custom operators are classified inside scanBuiltin only after comments,
	// literals, and closed composite spellings receive their required priority.
	return lex.scanBuiltin(src)
}

// scanBuiltin decides the next lexical unit using the language's own spellings.
func (lex *lexer) scanBuiltin(src string) (scanned, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.scanBuiltin -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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
				lines, endColumn := multilineMetrics(src[:n])
				return scanned{action: actionSkip, length: n, lines: lines, endColumn: endColumn}, true
			}
			lines, endColumn := multilineMetrics(src)
			return scanned{
				action:    actionError,
				length:    len(src),
				lines:     lines,
				endColumn: endColumn,
				message:   "unterminated block comment; a \"/*\" comment must be closed with \"*/\"",
				errType:   helpers.InvalidSyntax,
			}, true
		}
		return lex.scanSymbolicRun(src)

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
		if n := characterLiteralLength(src); n > 0 {
			return emit(CHAR, n), true
		}
		return emit(SINGLE_QUOTE, 1), true

	// ---- numeric literal -------------------------------------------------
	case isDigit(c):
		if length, message := malformedNumericLiteral(src); length > 0 {
			return scanned{
				action:  actionError,
				length:  length,
				message: message,
				errType: helpers.InvalidSyntax,
			}, true
		}
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
			// DECISION-LEX-009: a special method is one complete spelling from a
			// closed scanner-known set. "@@" is NOT a prefix operator over an
			// arbitrary identifier, so the whole run is scanned and checked against
			// Special_methods, the same way a built-in method name is resolved from
			// a table rather than accepted generically. An unrecognized spelling is
			// a lexical error that names the admissible set, rather than reaching
			// the parser as a bare "@@" plus an ordinary identifier.
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
			return lex.scanSymbolicRun(src)
		}
		if len(src) > 1 && (isAlpha(src[1]) || src[1] == '_') {
			return emit(ATDAP, 1+identifierLength(src[1:])), true
		}
		return lex.scanSymbolicRun(src)

	// ---- result and self bindings: "$", "$1", "$12" ------------------------
	case c == '$':
		n := 1
		for n < len(src) && isDigit(src[n]) {
			n++
		}
		if n > 1 || operatorRunLength(src) == 1 {
			return emit(BIND_VAR, n), true
		}
		return lex.scanSymbolicRun(src)

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
	case c == ',':
		return emit(COMMA, 1), true
	case c == ';':
		return emit(SEMI_COLON, 1), true

	// ---- complete symbolic run -------------------------------------------
	case operatorRunLength(src) > 0:
		return lex.scanSymbolicRun(src)
	}

	return scanned{}, false
}

// scanSymbolicRun classifies exactly one complete contiguous symbol run.
// Unknown runs remain whole as SYMBOLIC_RUN so grammar context can accept
// contextual metadata such as *** in T->(***), or reject the complete spelling
// everywhere else. No shorter-token fallback is attempted (DECISION-LEX-003).
func (lex *lexer) scanSymbolicRun(src string) (scanned, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.scanSymbolicRun -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	length := operatorRunLength(src)
	if length == 0 {
		return scanned{}, false
	}
	run := src[:length]

	if kind, ok := builtinSymbolKinds[run]; ok {
		return emit(kind, length), true
	}
	if languagePredeclaredOperatorSpellings[run] {
		return emit(CUSTOM_OPERATOR, length), true
	}
	if fixity, ok := lex.custom.match(run); ok {
		before := explicitSymbolBoundaryBefore(lex.source, lex.pos)
		after := explicitSymbolBoundaryAfter(lex.source, lex.pos+length)
		if utf8.RuneCountInString(run) > 1 && !boundariesSatisfyFixity(fixity, before, after) {
			return scanned{
				action: actionError,
				length: length,
				message: fmt.Sprintf(
					"multi-symbol %s operator %q requires an explicit boundary on every operand-facing side",
					fixity, run,
				),
				errType: helpers.InvalidSyntax,
			}, true
		}
		return emit(CUSTOM_OPERATOR, length), true
	}
	return emit(SYMBOLIC_RUN, length), true
}

func boundariesSatisfyFixity(fixity string, before, after bool) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.boundariesSatisfyFixity -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	switch fixity {
	case "infix":
		return before && after
	case "prefix":
		return after
	case "postfix":
		return before
	default:
		return false
	}
}

// Explicit source boundaries are retained before separators are discarded.
// A comment itself supplies a boundary even when no whitespace surrounds it.
func explicitSymbolBoundaryBefore(source string, at int) bool {

	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.explicitSymbolBoundaryBefore -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if at <= 0 || at > len(source) {
		return false
	}
	if at >= 2 && source[at-2:at] == "*/" {
		return true
	}
	return isExplicitBoundaryByte(source[at-1])
}

func explicitSymbolBoundaryAfter(source string, at int) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.explicitSymbolBoundaryAfter -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if at < 0 || at >= len(source) {
		return false
	}
	if strings.HasPrefix(source[at:], "//") || strings.HasPrefix(source[at:], "/*") {
		return true
	}
	return isExplicitBoundaryByte(source[at])
}

func isExplicitBoundaryByte(c byte) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.isExplicitBoundaryByte -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	switch c {
	case ' ', '\t', '\f', '\r', '\n', '(', ')', '[', ']', '{', '}', ',', ';', '\'', '"':
		return true
	default:
		return false
	}
}

// multilineMetrics reports both the number of logical line breaks and the byte
// column after the final one. The scanner needs both values because advanceN first
// counts the entire comment on the old line; merely resetting the column loses the
// text after the last newline.
func multilineMetrics(span string) (lines, endColumn int) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.multilineMetrics -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	lastLineStart := 0
	for i := 0; i < len(span); i++ {
		switch span[i] {
		case '\r':
			if i+1 < len(span) && span[i+1] == '\n' {
				i++
			}
			lines++
			lastLineStart = i + 1
		case '\n':
			lines++
			lastLineStart = i + 1
		}
	}
	if lines > 0 {
		endColumn = len(span) - lastLineStart
	}
	return lines, endColumn
}

// characterLiteralLength returns the byte length of a complete character literal at the
// cursor, or 0 when the span is not one.
//
// alpha-basic-c-character is any translation CHARACTER except the apostrophe, the
// backslash, CR and LF — a character, not a byte. Assuming the literal was three bytes
// wide rejected every non-ASCII one, so `'∪'` did not lex, which in turn made the
// reference's own custom-operator declaration unparseable.
func characterLiteralLength(src string) int {

	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.characterLiteralLength -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if len(src) < 2 {
		return 0
	}
	r, size := utf8.DecodeRuneInString(src[1:])
	if size == 0 || r == utf8.RuneError && size == 1 {
		return 0
	}
	switch r {
	case '\'', '\\', '\r', '\n':
		return 0
	}
	if 1+size >= len(src) || src[1+size] != '\'' {
		return 0
	}
	return 1 + size + 1
}

// stringLiteralLength returns the length of a complete string literal at the cursor,
// or 0 when the closing quote is missing before the line ends.
func stringLiteralLength(src string) int {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.stringLiteralLength -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.identifierLength -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")
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
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.numericLiteralLength -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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

// malformedNumericLiteral recognises abbreviated floating forms that otherwise
// split into several individually valid tokens. Alpha FoLang requires digits on
// both sides of a decimal point, and hexadecimal fractions additionally require a
// binary exponent.
func malformedNumericLiteral(src string) (int, string) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.malformedNumericLiteral -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if len(src) >= 2 && src[0] == '0' && (src[1] == 'x' || src[1] == 'X') {
		before := digitRun(src, 2, isHexDigit)
		if before < len(src) && src[before] == '.' {
			after := digitRun(src, before+1, isHexDigit)
			if before == 2 || after == before+1 && after < len(src) && (src[after] == 'p' || src[after] == 'P') {
				end := malformedExponentEnd(src, after, func(c byte) bool { return c == 'p' || c == 'P' })
				if end == after {
					end = after
				}
				return end, fmt.Sprintf("malformed hexadecimal floating literal %q; digits are required on both sides of the point", src[:end])
			}
		}
	}

	digits := digitRun(src, 0, isDigit)
	if digits < len(src) && src[digits] == '.' && (digits+1 == len(src) || decimalPointCannotStartPostfix(src[digits+1])) {
		return digits + 1, fmt.Sprintf("malformed floating literal %q; write a digit after the decimal point (for example, 1.0)", src[:digits+1])
	}
	return 0, ""
}

func malformedExponentEnd(src string, i int, marker func(byte) bool) int {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.malformedExponentEnd -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if i >= len(src) || !marker(src[i]) {
		return i
	}
	n := i + 1
	if n < len(src) && (src[n] == '+' || src[n] == '-') {
		n++
	}
	n = digitRun(src, n, isDigit)
	return floatSuffixEnd(src, n)
}

// A point followed by another point begins a range, and a point followed by an
// identifier begins member access. Every other continuation after `1.` is an
// unambiguous attempt at the unsupported abbreviated float spelling.
func decimalPointCannotStartPostfix(next byte) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.decimalPointCannotStartPostfix -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return next != '.' && !isDigit(next) && !isAlpha(next) && next != '_'
}

// exponentLength returns the length of an exponent-part at i, or 0.
func exponentLength(src string, i int) int {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.exponentLength -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return signedExponent(src, i, func(c byte) bool { return c == 'e' || c == 'E' })
}

// binaryExponentLength returns the length of a binary-exponent-part at i, or 0.
func binaryExponentLength(src string, i int) int {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.binaryExponentLength -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return signedExponent(src, i, func(c byte) bool { return c == 'p' || c == 'P' })
}

// signedExponent matches marker, optional sign, then a digit sequence.
func signedExponent(src string, i int, isMarker func(byte) bool) int {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.signedExponent -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.floatSuffixEnd -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.intSuffixEnd -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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

func digitRun(src string, i int, isDigitFn func(byte) bool) int {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.digitRun -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	for i < len(src) && isDigitFn(src[i]) {
		i++
	}
	return i
}

func isDigit(c byte) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in parser.isDigit -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return c >= '0' && c <= '9'
}
func isBinDigit(c byte) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in parser.isBinDigit -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return c == '0' || c == '1'
}
func isHexDigit(c byte) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in parser.isHexDigit -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
func isAlpha(c byte) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in parser.isAlpha -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// emitNewline pushes the NEWLINE token that marks a line break.
//
// cleanupLB drops these once scanning finishes; they exist so line bookkeeping happens
// in one place. The lexeme is the literal "LSP" the previous scanner used, so a caller
// inspecting the raw stream sees exactly what it saw before.
func (lex *lexer) emitNewline(_ string) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.emitNewline -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

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
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.emitToken -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	boundaryBefore := explicitSymbolBoundaryBefore(lex.source, lex.pos)
	boundaryAfter := explicitSymbolBoundaryAfter(lex.source, lex.pos+len(lexeme))
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
	last := len(lex.Tokens) - 1
	if last >= 0 {
		lex.Tokens[last].BoundaryBefore = boundaryBefore
		lex.Tokens[last].BoundaryAfter = boundaryAfter
	}
	lex.posi = end
}

// emitIdentifier applies the identifier rules and the reserved-word classification.
//
// DECISION-LEX-001/006: an identifier carries single underscores between nonempty
// alphanumeric segments and never ends in one. Reserved_lu then gives a reserved word
// its KEYWORD or RESERVEDWORD kind instead of letting it pass as an IDENTIFIER.
func (lex *lexer) emitIdentifier(lexeme string, start, end *helpers.Position) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in scanner.emitIdentifier -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	// The symbol-collecting pass reports nothing; the real scan that follows it
	// reports everything, so a malformed name is still caught exactly once.
	if !lex.quiet {
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
	}

	kind := IDENTIFIER
	if k, ok := Reserved_lu[lexeme]; ok {
		kind = k
	}
	lex.push(newUniqueToken(kind, lexeme, start.Copy(), end))
}
