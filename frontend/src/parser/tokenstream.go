package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Token-stream normalisation.
//
// The scanner in package scanlex is the normative lexer and is not modified by
// the parser. Two of its behaviours need reconciling with the grammar before the
// token stream can be parsed, and both are handled here so that every other file
// can assume a clean stream.
//
// 1. Compatibility operator fusion (DECISION-LEX-003).
//
//    scanlex.Tokenize applies maximal munch and emits every built-in spelling below as
//    one token. The rewrites remain as a compatibility fallback for token streams built
//    by older clients or assembled manually in parser tests:
//
//        "=>" ">"  -> "=>>"    delegation chain
//        "*" "*"   -> "**"     exponentiation
//        "*" "="   -> "*="     compound assignment
//        "/" "="   -> "/="
//        "%" "="   -> "%="
//        "&" "="   -> "&="
//        "^" "="   -> "^="
//        "|" "="   -> "|="
//        "**" "="  -> "**="
//
//    Fusion only applies when the two fallback tokens are physically adjacent in the
//    source, so `a * *p` and `a ** p` stay distinct and `x = *p` is untouched.
//
// 2. Operator identity by lexeme.
//
//    The scanner maps several distinct spellings onto one TokenKind — "^" and
//    "**" both arrive as POW, and the additional compound assignments arrive as
//    ASSIGNMENT. The parser therefore keys all operator dispatch on the exact
//    lexeme (see precedence.go) and uses TokenKind only for structural tokens.
//    Fused tokens keep a representative kind purely so that code which does look
//    at kinds still sees an operator.

// normalizeLineEndings rewrites CRLF and lone CR line terminators to LF.
//
// The grammar treats all three spellings as the same thing: line-break is
// "\r\n" | "\n" | "\r", and DECISION-LEX-002 makes a line terminator ordinary whitespace with
// no statement-termination meaning. The scanner, however, recognises "\r\n" as one match while
// advancing a single character, so the trailing "\n" is scanned as a second line break and its
// line counter runs ahead of the file's real line count. Normalising here removes the
// difference before tokenizing.
//
// Only discarded tokens are affected: line breaks between tokens are dropped from the stream
// entirely, so no token's kind or lexeme changes. Column and index positions shift by one per
// preceding CRLF, which makes them consistent rather than skewed.
func normalizeLineEndings(source string) string {
	if !strings.ContainsRune(source, '\r') {
		return source
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	return strings.ReplaceAll(source, "\r", "\n")
}

// fusion describes one adjacent-token rewrite.
type fusion struct {
	first  string
	second string
	result string
	kind   scanlex.TokenKind
}

// fusions lists the adjacent-token rewrites in application order. Longer results
// must precede their own prefixes so that "**" becomes "**=" when an "=" follows:
// the table is applied repeatedly to a token and its successor, and "**" is
// produced by an earlier pass than "**=".
var fusions = []fusion{
	// The scanner emits each result directly. These rules only serve compatibility
	// token streams supplied to normalizeTokens by tests or older clients.
	{first: "=>", second: ">", result: "=>>", kind: scanlex.EQGTGT},
	{first: "**", second: "=", result: "**=", kind: scanlex.ASSIGNMENT},
	{first: "*", second: "*", result: "**", kind: scanlex.POW},
	{first: "*", second: "=", result: "*=", kind: scanlex.ASSIGNMENT},
	{first: "/", second: "=", result: "/=", kind: scanlex.ASSIGNMENT},
	{first: "%", second: "=", result: "%=", kind: scanlex.ASSIGNMENT},
	{first: "&", second: "=", result: "&=", kind: scanlex.ASSIGNMENT},
	{first: "^", second: "=", result: "^=", kind: scanlex.ASSIGNMENT},
	{first: "|", second: "=", result: "|=", kind: scanlex.ASSIGNMENT},
}

// normalizeTokens returns toks with the operator fusions of the table above and the
// numeric-literal fusions of fuseNumericLiteral applied. The input slice is not modified.
func normalizeTokens(toks []scanlex.Token) []scanlex.Token {
	out := make([]scanlex.Token, 0, len(toks))
	for i := 0; i < len(toks); i++ {
		cur := toks[i]

		// Fuse repeatedly so that "*" "*" "=" collapses to "**=" in two steps.
		for i+1 < len(toks) {
			next := toks[i+1]
			f, ok := matchFusion(cur, next)
			if !ok {
				break
			}
			cur = scanlex.NewUniqueToken(f.kind, f.result, cur.StartPos, next.EndPos)
			i++
		}

		if cur.Kind == scanlex.NUMBER {
			cur, i = fuseNumericLiteral(cur, toks, i)
		}

		out = append(out, cur)
	}
	return fuseHexFloats(out)
}

// fuseHexFloats rebuilds a hexadecimal floating literal from the three tokens the first pass
// leaves behind.
//
// A hex float needs two passes because its pieces only become recognisable after numeric fusion
// has run: `0x1.8p3` reaches the first pass as NUMBER("0") IDENT("x1") "." NUMBER("8")
// IDENT("p3"), and leaves it as NUMBER("0x1") "." NUMBER("8p3"). Only now is the shape visible.
//
// DECISION-LIT-002 requires a binary-exponent-part on every hexadecimal floating literal, so the
// trailing piece must carry a "p" exponent. That requirement is what makes the fusion safe: it
// cannot capture a member access on a hex integer, and it cannot capture a range, because
// DECISION-LIT-006 already guarantees a numeric literal never ends at a point.
func fuseHexFloats(toks []scanlex.Token) []scanlex.Token {
	out := make([]scanlex.Token, 0, len(toks))

	for i := 0; i < len(toks); i++ {
		if i+2 < len(toks) && isHexFloatTriple(toks[i], toks[i+1], toks[i+2]) {
			fused := toks[i].Value + "." + toks[i+2].Value
			out = append(out, scanlex.NewUniqueToken(scanlex.NUMBER, fused, toks[i].StartPos, toks[i+2].EndPos))
			i += 2
			continue
		}
		out = append(out, toks[i])
	}
	return out
}

// isHexFloatTriple reports whether a, b, c spell one hexadecimal floating literal.
func isHexFloatTriple(a, b, c scanlex.Token) bool {
	if a.Kind != scanlex.NUMBER || b.Kind != scanlex.DOT || c.Kind != scanlex.NUMBER {
		return false
	}
	if !adjacent(a, b) || !adjacent(b, c) {
		return false
	}
	if !hasHexPrefix(a.Value) {
		return false
	}
	// The fractional part must be hex digits followed by a binary exponent.
	return strings.ContainsAny(c.Value, "pP")
}

// hasHexPrefix reports whether a numeric lexeme carries the 0x or 0X prefix.
func hasHexPrefix(lexeme string) bool {
	return len(lexeme) > 2 && lexeme[0] == '0' && (lexeme[1] == 'x' || lexeme[1] == 'X')
}

// fuseNumericLiteral rebuilds the numeric literals of DECISION-LIT-001 and
// DECISION-LIT-002 from the pieces the scanner emits.
//
// The scanner recognises a numeric literal as `digits [ "." digits ]` only, so every other
// admitted spelling arrives split across a NUMBER and the identifier that follows it:
//
//	0xFF     -> NUMBER("0")    IDENTIFIER("xFF")
//	0b1011   -> NUMBER("0")    IDENTIFIER("b1011")
//	1e5      -> NUMBER("1")    IDENTIFIER("e5")
//	1e-5     -> NUMBER("1")    IDENTIFIER("e")   "-"  NUMBER("5")
//	10u      -> NUMBER("10")   IDENTIFIER("u")
//	3.14f    -> NUMBER("3.14") IDENTIFIER("f")
//
// Fusion applies only when the pieces are physically adjacent and the trailing text is a
// valid continuation of a numeric literal, so an expression such as `1 e5` or `count x`
// is untouched. cur is the NUMBER token and i its index; the returned index is the last
// token consumed.
//
// DECISION-LIT-007 is respected: no digit separator is accepted, so an apostrophe never
// takes part in a numeric literal.
func fuseNumericLiteral(cur scanlex.Token, toks []scanlex.Token, i int) (scanlex.Token, int) {
	if i+1 >= len(toks) {
		return cur, i
	}
	next := toks[i+1]
	if !adjacent(cur, next) {
		return cur, i
	}
	// COMPOSITE_IDENTIFER is accepted alongside IDENTIFIER because token folding reclassifies
	// the tail when a reserved member name follows it: in `0xFF.to_str()` the "xFF" arrives as
	// COMPOSITE_IDENTIFER with ".to_str" split off, and it is still the literal's hex digits.
	if !next.IsOneOfMany(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) {
		return cur, i
	}

	tail := logicalName(next.Value)
	lexeme, kind := numericContinuation(cur.Value, tail)
	switch kind {
	case numericNone:
		return cur, i

	case numericComplete:
		return scanlex.NewUniqueToken(scanlex.NUMBER, lexeme, cur.StartPos, next.EndPos), i + 1

	case numericNeedsExponent:
		// A signed exponent: the sign and its digits are separate tokens.
		if i+3 >= len(toks) {
			return cur, i
		}
		sign, digits := toks[i+2], toks[i+3]
		if (sign.Value != "+" && sign.Value != "-") || digits.Kind != scanlex.NUMBER {
			return cur, i
		}
		if !adjacent(next, sign) || !adjacent(sign, digits) {
			return cur, i
		}
		fused := lexeme + sign.Value + digits.Value
		return scanlex.NewUniqueToken(scanlex.NUMBER, fused, cur.StartPos, digits.EndPos), i + 3
	}
	return cur, i
}

// numericKind classifies the outcome of testing a numeric continuation.
type numericKind int

const (
	// numericNone means the trailing text does not continue a numeric literal.
	numericNone numericKind = iota
	// numericComplete means the fused lexeme is a complete literal.
	numericComplete
	// numericNeedsExponent means an exponent marker was found with no digits attached,
	// so a sign and digits must follow.
	numericNeedsExponent
)

// numericContinuation reports whether tail continues the numeric literal head, and returns
// the fused lexeme.
func numericContinuation(head, tail string) (string, numericKind) {
	fused := head + tail

	// Base prefixes only follow a lone "0": 0xFF, 0B1011.
	if head == "0" && len(tail) > 1 {
		switch tail[0] {
		case 'x', 'X':
			if allDigitsIn(tail[1:], isHexDigit) {
				return fused, numericComplete
			}
			return "", numericNone
		case 'b', 'B':
			if allDigitsIn(tail[1:], isBinaryDigit) {
				return fused, numericComplete
			}
			return "", numericNone
		}
	}

	// A decimal exponent: 1e5, 1E5, and the bare marker of a signed exponent.
	if tail == "e" || tail == "E" {
		return fused, numericNeedsExponent
	}
	if (tail[0] == 'e' || tail[0] == 'E') && len(tail) > 1 && allDigitsIn(tail[1:], isDecimalDigit) {
		return fused, numericComplete
	}

	// A hexadecimal float's binary exponent: 0x1.8p3.
	if (tail[0] == 'p' || tail[0] == 'P') && len(tail) > 1 && allDigitsIn(tail[1:], isDecimalDigit) {
		return fused, numericComplete
	}
	if tail == "p" || tail == "P" {
		return fused, numericNeedsExponent
	}

	// An integer or floating suffix.
	if isNumericSuffix(tail) {
		return fused, numericComplete
	}
	return "", numericNone
}

// numericSuffixes is the integer-suffix and floating-point-suffix set of section 12.
var numericSuffixes = map[string]struct{}{
	"u": {}, "U": {},
	"l": {}, "L": {},
	"ll": {}, "LL": {},
	"z": {}, "Z": {},
	"ul": {}, "uL": {}, "Ul": {}, "UL": {},
	"lu": {}, "lU": {}, "Lu": {}, "LU": {},
	"ull": {}, "ULL": {}, "llu": {}, "LLU": {},
	"uz": {}, "UZ": {}, "zu": {}, "ZU": {},
	"f": {}, "F": {},
	"f16": {}, "F16": {},
	"f32": {}, "F32": {},
	"f64": {}, "F64": {},
	"f128": {}, "F128": {},
	"bf16": {}, "BF16": {},
}

// isNumericSuffix reports whether tail is a numeric literal suffix.
func isNumericSuffix(tail string) bool {
	_, ok := numericSuffixes[tail]
	return ok
}

// allDigitsIn reports whether every byte of s satisfies isDigit and s is non-empty.
func allDigitsIn(s string, isDigit func(byte) bool) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

func isDecimalDigit(c byte) bool { return c >= '0' && c <= '9' }
func isBinaryDigit(c byte) bool  { return c == '0' || c == '1' }
func isHexDigit(c byte) bool {
	return isDecimalDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// matchFusion reports the fusion that applies to the adjacent pair a, b.
func matchFusion(a, b scanlex.Token) (fusion, bool) {
	if !adjacent(a, b) {
		return fusion{}, false
	}
	for _, f := range fusions {
		if a.Value == f.first && b.Value == f.second {
			return f, true
		}
	}
	return fusion{}, false
}

// adjacent reports whether b starts exactly where a ends, i.e. whether the two
// tokens are written with no intervening whitespace or comment. This is the same
// adjacency test the grammar relies on for the encoding-prefix rule
// (DECISION-LEX-008).
func adjacent(a, b scanlex.Token) bool {
	if a.EndPos == nil || b.StartPos == nil {
		return false
	}
	return a.EndPos.Idx == b.StartPos.Idx
}

// foLoweringSuffix is the suffix the scanner appends to every resolved
// user-defined identifier as it folds tokens (DECISION-BACKEND-001: a FoLang
// identifier is lowered to C++ by appending "_fo").
const foLoweringSuffix = "_fo"

// logicalName strips the backend lowering suffix from a scanned identifier,
// returning the name as it was written in source.
//
// The parser needs both spellings. The lowered spelling is what the backend
// emits and is therefore what gets stored in the AST, while the logical spelling
// is what must be compared against grammar keywords and against the control-flow
// verbs of section 11a — a source `.do` arrives as the identifier "do_fo" when
// it follows a ")" rather than a name, because token folding only rewrites a dot
// chain that begins at an identifier.
func logicalName(scanned string) string {
	// A folded dotted path lowers every segment: "a_fo.b_fo". Remove
	// internal suffixes first and the final suffix second; returning immediately
	// after trimming the final segment used to leave "a_fo.b" half-lowered.
	logical := strings.ReplaceAll(scanned, foLoweringSuffix+".", ".")
	return strings.TrimSuffix(logical, foLoweringSuffix)
}
