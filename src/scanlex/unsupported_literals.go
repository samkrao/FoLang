package scanlex

import (
	"strings"

	"github.com/samkrao/fo-lang/src/helpers"
)

// detectUnsupportedAlphaLiteral recognizes complete literal spellings reserved
// for a post-alpha release.
//
// This check runs before the ordinary regex table. That ordering is essential:
// without it, `u8"hello"` becomes an identifier followed by a string, while
// `"a\"b"` is split at the escaped quote and causes several unrelated parser
// errors. Recognizing the whole future spelling gives the user one intentional
// unsupported-feature diagnostic.
func detectUnsupportedAlphaLiteral(source string) (length int, message string, ok bool) {
	if length, ok := scanRawStringLiteral(source); ok {
		return length, "raw string literals are not supported in the alpha release", true
	}

	if prefixLength := encodedLiteralPrefixLength(source); prefixLength > 0 {
		quote := source[prefixLength]
		length, _ := scanQuotedLiteral(source, prefixLength, quote)
		if length == 0 {
			length = len(source)
		}
		if quote == '\'' {
			return length, "encoded character literal prefixes are not supported in the alpha release", true
		}
		return length, "encoded string literal prefixes are not supported in the alpha release", true
	}

	if source == "" || (source[0] != '"' && source[0] != '\'') {
		return 0, "", false
	}

	length, hasEscape := scanQuotedLiteral(source, 0, source[0])
	if length == 0 || !hasEscape {
		return 0, "", false
	}

	if source[0] == '\'' {
		if strings.Contains(source[:length], `\N{`) {
			return length, "named universal character literals are not supported in the alpha release", true
		}
		return length, "escaped character literals are not supported in the alpha release", true
	}
	return length, "escaped string characters are not supported in the alpha release", true
}

// encodedLiteralPrefixLength returns the length of an encoding prefix only when
// it is immediately followed by a quote. With whitespace, u8/u/U/L remain
// ordinary identifiers.
func encodedLiteralPrefixLength(source string) int {
	for _, prefix := range []string{"u8", "u", "U", "L"} {
		if len(source) > len(prefix) && strings.HasPrefix(source, prefix) {
			next := source[len(prefix)]
			if next == '"' || next == '\'' {
				return len(prefix)
			}
		}
	}
	return 0
}

// scanQuotedLiteral consumes through the matching quote while treating a
// backslash and its following byte as one future escape. Zero means the source
// does not contain a complete quoted literal.
func scanQuotedLiteral(source string, quoteIndex int, quote byte) (length int, hasEscape bool) {
	for i := quoteIndex + 1; i < len(source); i++ {
		switch source[i] {
		case '\r', '\n':
			return 0, hasEscape
		case '\\':
			hasEscape = true
			if i+1 < len(source) {
				i++
			}
		default:
			if source[i] == quote {
				return i + 1, hasEscape
			}
		}
	}
	return 0, hasEscape
}

// scanRawStringLiteral recognizes a complete C++-shaped raw string, including
// an optional encoding prefix and matching custom delimiter. The spelling is
// reserved so it is recognized only to produce an alpha diagnostic.
func scanRawStringLiteral(source string) (int, bool) {
	openLength := 0
	for _, opening := range []string{`u8R"`, `uR"`, `UR"`, `LR"`, `R"`} {
		if strings.HasPrefix(source, opening) {
			openLength = len(opening)
			break
		}
	}
	if openLength == 0 {
		return 0, false
	}

	openParen := strings.IndexByte(source[openLength:], '(')
	if openParen < 0 {
		return len(source), true
	}
	openParen += openLength
	delimiter := source[openLength:openParen]
	closing := ")" + delimiter + `"`
	closeOffset := strings.Index(source[openParen+1:], closing)
	if closeOffset < 0 {
		return len(source), true
	}
	return openParen + 1 + closeOffset + len(closing), true
}

// rejectUnsupportedAlphaLiteral advances over the complete recognized spelling
// before raising one unsupported-feature error.
func rejectUnsupportedAlphaLiteral(lex *lexer, length int, message string) {
	start := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.advanceN(length)
	end := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.report(lex.errorException(message, helpers.UnSupported, *start, *end))
}
