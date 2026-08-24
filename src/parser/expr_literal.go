package parser

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// Literals — section 12 of docs/grammar/folang.ebnf.
//
//	literal         = builtin-literal
//	builtin-literal = integer-literal | floating-literal
//	                | string-literal | character-literal
//	                | boolean-literal | none-literal
//
// FoLang takes a selected C++-compatible subset of literal spellings, and the scanner
// stores each literal's COMPLETE original lexeme so a C++ backend can emit it unchanged.
// This file therefore decodes a value for the AST while leaving the lexeme untouched on
// the token.
//
// One string literal is one string. C's adjacent-literal concatenation is NOT part of
// the subset FoLang takes: builtin-literal names string-literal directly, and the
// reference shows no such spelling. Two strings in a row are therefore two expressions
// with nothing joining them, which is a syntax error where it appears rather than a
// silent concatenation.
//
// FoLang's booleans and null are co.const.true, co.const.false and co.const.none;
// `true` and `false` are ordinary names, not literals.

// parseLiteral parses one builtin-literal.
//
// Implements: literal
// Implements: builtin-literal
func (p *parser) parseLiteral() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	switch p.kind() {
	case scanlex.NUMBER:
		return p.parseNumericLiteral()
	case scanlex.STRING:
		return p.parseStringLiteral()
	case scanlex.CHAR:
		return p.parseCharacterLiteral()
	case scanlex.BUILT_IN_CONSTANTS:
		return p.parseBuiltinConstant()
	case scanlex.BOOL:
		return p.parseBooleanToken()
	}
	p.failf(p.cur(), "expected a literal, found %s", describeToken(p.cur()))
	return nil // unreachable: failf panics
}

// parseNumericLiteral parses the integer-literal and floating-literal productions.
//
// The two are told apart by shape rather than by scanner kind. DECISION-LIT-006
// requires a digit on both sides of the point, so a "." inside the lexeme is
// unambiguously a fractional point and never the start of a range operator or a
// member access. That is what makes `1 .. 10` scan as `1 .. 10` and `3.14.to_str()`
// as `3.14 . to_str ( )` with no lookahead at all.
//
// DECISION-LIT-001 and DECISION-LIT-002 admit binary, octal, decimal and
// hexadecimal integers with the standard suffixes, and decimal and hexadecimal
// floats. Digit separators are not adopted (DECISION-LIT-007).
//
// Implements: integer-literal
// Implements: floating-literal
func (p *parser) parseNumericLiteral() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()
	lexeme := tok.Value

	if isFloatingLexeme(lexeme) {
		value, ok := parseFloatLexeme(lexeme)
		if !ok {
			p.reportf(tok, "malformed floating literal %q; a FoLang floating literal needs a digit on both sides of the point, so write 1.0 rather than 1.", lexeme)
		}
		return ast.NumberLiteral{Span: p.spanFrom(spanStart), Value: value,
			Type_:    "co.lang.double",
			ActType_: "co.lang.double",
			Symb:     p.exprSymbol(lexeme),
		}
	}

	value, ok := parseIntegerLexeme(lexeme)
	if !ok {
		// A bad suffix and bad digits are different mistakes, and only the first
		// has a rule a reader can act on, so it is named rather than folded into
		// the general message.
		if _, suffixOK := trimIntegerSuffix(lexeme); !suffixOK {
			p.reportf(tok, "malformed integer literal %q; an integer suffix pairs at most one length marker (\"l\", \"L\", \"ll\", \"LL\", \"z\", \"Z\") with at most one unsigned marker (\"u\", \"U\"), so each may appear only once", lexeme)
		} else {
			p.reportf(tok, "malformed integer literal %q", lexeme)
		}
	}
	return ast.IntegerLiteral{Span: p.spanFrom(spanStart), Value: value,
		Type_:    "co.lang.int",
		ActType_: "co.lang.int",
		Symb:     p.exprSymbol(lexeme),
	}
}

// isFloatingLexeme reports whether a numeric lexeme is a floating literal.
//
// A floating literal either contains a point, or carries an exponent. The exponent
// test excludes hexadecimal integers, where "e" is an ordinary digit: 0x1e is the
// integer 30, whereas 1e5 is a float and 0x1.8p3 is a hexadecimal float whose
// exponent marker is "p".
func isFloatingLexeme(lexeme string) bool {
	if strings.Contains(lexeme, ".") {
		return true
	}
	lower := strings.ToLower(lexeme)
	if strings.HasPrefix(lower, "0x") {
		return strings.Contains(lower, "p")
	}
	return strings.ContainsAny(lower, "e")
}

// parseFloatLexeme decodes a floating literal, stripping the floating-point suffix
// of DECISION-LIT-002 (f, F, l, L and the extended f16/f32/f64/f128/bf16 forms)
// before conversion.
//
// NumberLiteral.Value is a float64 convenience slot, exactly as IntegerLiteral.Value
// is an int64 one, and it is held to the same invariant: it carries the decoded value
// when the literal fits, and zero when it does not, while the exact, authoritative
// lexeme remains on the expression symbol for later typing and backend emission.
//
// Overflowing the slot must not become an internal error. strconv.ParseFloat reports
// an out-of-range literal such as 1e9999 as ±Inf with strconv.ErrRange, and a
// non-finite float64 has no JSON encoding, so storing it would make the driver's AST
// emission fail with "json: unsupported value: +Inf" — a message with no source
// position, for a literal the grammar accepts. Underflow needs no special case: it
// already yields a finite zero.
func parseFloatLexeme(lexeme string) (float64, bool) {
	trimmed := trimFloatSuffix(lexeme)
	v, err := strconv.ParseFloat(trimmed, 64)
	// Range is a typing/representation concern, not malformed syntax.
	if !(err == nil || errors.Is(err, strconv.ErrRange)) {
		return 0, false
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, true
	}
	return v, true
}

// floatSuffixes lists the floating-point suffixes in longest-first order, so that
// "f128" is stripped before "f".
var floatSuffixes = []string{
	"bf16", "BF16",
	"f128", "F128",
	"f64", "F64",
	"f32", "F32",
	"f16", "F16",
	"f", "F", "l", "L",
}

// trimFloatSuffix removes a trailing floating-point suffix from a lexeme.
func trimFloatSuffix(lexeme string) string {
	for _, s := range floatSuffixes {
		if strings.HasSuffix(lexeme, s) {
			return strings.TrimSuffix(lexeme, s)
		}
	}
	return lexeme
}

// parseIntegerLexeme decodes an integer literal in any of the four bases of
// DECISION-LIT-001, stripping the integer suffix first.
//
// Syntax is validated with math/big rather than strconv.ParseInt. The grammar
// does not impose a signed-64-bit limit, and C++-compatible unsigned or
// implementation-sized literals can legitimately exceed int64. IntegerLiteral
// retains its historical int64 convenience value when the literal fits; for a
// wider value it stores zero while the exact, authoritative lexeme remains on
// the expression symbol for later typing and backend emission.
func parseIntegerLexeme(lexeme string) (int64, bool) {
	trimmed, suffixOK := trimIntegerSuffix(lexeme)
	if !suffixOK || trimmed == "" {
		return 0, false
	}

	value, ok := new(big.Int).SetString(trimmed, 0)
	if !ok {
		return 0, false
	}
	if value.IsInt64() {
		return value.Int64(), true
	}
	return 0, true
}

// trimIntegerSuffix removes a trailing integer-suffix and reports whether what it
// removed is one:
//
//	integer-suffix = unsigned-suffix, [ long-suffix | long-long-suffix | size-suffix ]
//	               | long-suffix,     [ unsigned-suffix ]
//	               | long-long-suffix, [ unsigned-suffix ]
//	               | size-suffix,     [ unsigned-suffix ]
//
// Every alternative pairs AT MOST ONE length marker with AT MOST ONE unsigned
// marker, so each marker appears once and `1uu`, `1zz`, `1lll` and `1lul` are none
// of them. A blanket TrimRight over "uUlLzZ" accepted all four, because it asks
// only which characters are suffix characters and never how many of each the
// production admits.
//
// "ll" and "LL" are read before a single "l"/"L" so the long-long marker is not
// mistaken for two long markers, and the two halves of it must agree in case:
// long-long-suffix is spelled "ll" or "LL", never "lL".
func trimIntegerSuffix(lexeme string) (string, bool) {
	end := len(lexeme)

	unsigned := func() bool {
		if end > 0 && (lexeme[end-1] == 'u' || lexeme[end-1] == 'U') {
			end--
			return true
		}
		return false
	}
	length := func() bool {
		if end >= 2 {
			switch lexeme[end-2 : end] {
			case "ll", "LL":
				end -= 2
				return true
			}
		}
		if end > 0 {
			switch lexeme[end-1] {
			case 'l', 'L', 'z', 'Z':
				end--
				return true
			}
		}
		return false
	}

	// The trailing marker decides which alternative applies; the other marker is
	// then optional. Reading from the right mirrors the way the suffix is written.
	if unsigned() {
		length()
	} else if length() {
		unsigned()
	}

	// Anything still suffix-shaped at the end is a marker the production has no
	// room for, which is exactly the repetition this check exists to catch.
	if end > 0 && strings.ContainsRune("uUlLzZ", rune(lexeme[end-1])) {
		// A hexadecimal literal's digits legitimately end in these letters, so
		// only a decimal/binary/octal head is judged by its last character.
		if lower := strings.ToLower(lexeme); !strings.HasPrefix(lower, "0x") {
			return lexeme[:end], false
		}
	}
	return lexeme[:end], true
}

// parseStringLiteral parses one string-literal.
//
// It consumes exactly one token. A second string immediately after it is left for the
// caller, which has no expression rule that continues with a literal and so reports it
// — adjacent literals do not concatenate in FoLang.
//
// Implements: string-literal
func (p *parser) parseStringLiteral() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	first := p.advance()
	value := unquote(first.Value)

	return ast.StringLiteral{Span: p.spanFrom(spanStart), Value: value,
		ActType_: "co.lang.string",
		Symb:     p.exprSymbol(first.Value),
	}
}

// parseCharacterLiteral parses the character-literal production.
//
// The lexeme keeps its quotes and any escape sequence, so the delimiters are
// stripped and a leading encoding prefix (DECISION-LEX-008: u8, u, U or L) removed
// before decoding. A character literal admits any translation character except the
// apostrophe, the backslash, CR and LF, so a space or a comma is an ordinary
// character (DECISION-LIT-007).
//
// Implements: character-literal
func (p *parser) parseCharacterLiteral() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()

	value, ok := decodeCharacterLexeme(tok.Value)
	if !ok {
		p.reportf(tok, "malformed character literal %s", tok.Value)
	}

	return ast.CharacterLiteral{Span: p.spanFrom(spanStart), Value: value,
		ActType_: "co.lang.char",
		Symb:     p.exprSymbol(tok.Value),
	}
}

// decodeCharacterLexeme extracts the code point from a character literal's lexeme.
func decodeCharacterLexeme(lexeme string) (rune, bool) {
	body := stripEncodingPrefix(lexeme)
	if len(body) < 2 || body[0] != '\'' || body[len(body)-1] != '\'' {
		return 0, false
	}
	body = body[1 : len(body)-1]
	if body == "" {
		return 0, false
	}

	if body[0] == '\\' {
		return decodeEscape(body)
	}
	r, size := utf8.DecodeRuneInString(body)
	return r, size == len(body)
}

// simpleEscapes is the simple-escape-sequence table of section 12.
var simpleEscapes = map[byte]rune{
	'\'': '\'', '"': '"', '?': '?', '\\': '\\',
	'a': 7, 'b': 8, 'f': 12, 'n': 10, 'r': 13, 't': 9, 'v': 11,
}

// decodeEscape decodes one escape-sequence, covering the simple escapes, the
// numeric octal and hexadecimal forms, and the \u/\U universal character names.
func decodeEscape(body string) (rune, bool) {
	if len(body) < 2 {
		return 0, false
	}
	switch c := body[1]; {
	case simpleEscapes[c] != 0:
		return simpleEscapes[c], len(body) == 2
	case c == 'x':
		v, err := strconv.ParseUint(strings.Trim(body[2:], "{}"), 16, 32)
		return rune(v), err == nil
	case c == 'u', c == 'U':
		v, err := strconv.ParseUint(strings.Trim(body[2:], "{}"), 16, 32)
		return rune(v), err == nil
	case c >= '0' && c <= '7':
		v, err := strconv.ParseUint(strings.Trim(body[1:], "o{}"), 8, 32)
		return rune(v), err == nil
	default:
		// conditional-escape-sequence: the character stands for itself.
		return rune(c), len(body) == 2
	}
}

// encodingPrefixes is the encoding-prefix set of DECISION-LEX-008.
var encodingPrefixes = []string{"u8", "u", "U", "L"}

// stripEncodingPrefix removes a literal's encoding prefix.
//
// A letter sequence is a prefix only when it is exactly u8, u, U or L and is
// immediately followed by a quote, which is what keeps u8, u, U and L usable as
// ordinary variable names.
func stripEncodingPrefix(lexeme string) string {
	for _, prefix := range encodingPrefixes {
		if !strings.HasPrefix(lexeme, prefix) {
			continue
		}
		rest := lexeme[len(prefix):]
		if strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, `'`) {
			return rest
		}
	}
	return lexeme
}

// parseBuiltinConstant parses the boolean-literal and none-literal productions:
//
//	boolean-literal = "co.const.true" | "co.const.false"
//	none-literal    = "co.const.none"
//
// These spellings also match qualified-name; DECISION-LIT-005 selects the literal
// reading, which the scanner has already applied by classifying them as
// BUILT_IN_CONSTANTS.
//
// Implements: boolean-literal
// Implements: none-literal
func (p *parser) parseBuiltinConstant() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()

	switch tok.Value {
	case "co.const.true", "co.const.false":
		return ast.BooleanLiteral{Span: p.spanFrom(spanStart), Value: tok.Value == "co.const.true",
			ActType_: "co.lang.bool",
			Symb:     p.exprSymbol(tok.Value),
		}
	case "co.const.none":
		// The none literal has no value node of its own; it is represented as
		// the built-in constant statement wrapped for expression position.
		return ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: tok.Value,
			SymbolType_: "none",
			Symb:        p.exprSymbol(tok.Value),
		}
	}

	p.failf(tok, "unknown built-in constant %q", tok.Value)
	return nil // unreachable: failf panics
}

// parseBooleanToken parses a BOOL token.
//
// The scanner reserves this kind but never emits it for source text, since FoLang
// booleans are the co.const spellings. It is handled for completeness so a
// synthesised token from a macro expansion still parses.
func (p *parser) parseBooleanToken() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()
	return ast.BooleanLiteral{Span: p.spanFrom(spanStart), Value: tok.Value == "true" || tok.Value == "co.const.true",
		ActType_: "co.lang.bool",
		Symb:     p.exprSymbol(tok.Value),
	}
}
