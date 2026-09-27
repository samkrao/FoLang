package scanlex

import (
	"strings"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/src/helpers"
)

// tokenLexer scans source bytes on demand. nextToken returns one lexical token
// at a time; whitespace, comments, and line breaks are consumed without being
// returned.
//
// The source bytes are copied into the scanner's immutable string storage. The
// caller may therefore reuse or modify source after NewTokenStream returns.
type tokenLexer struct {
	inner    *lexer
	finished bool
	eof      Token
}

func newLexer(source []byte, fn string) *tokenLexer {
	text := string(source)
	// A leading BOM is source-file metadata and does not participate in token
	// positions, matching the established Tokenize API.
	text = strings.TrimPrefix(text, utf8BOM)
	core := createLexer(text, fn)
	return &tokenLexer{
		inner: core,
		eof:   newUniqueToken(EOF, NA, "EOF", helpers.NewPosition(1, 0, 1, 0, "", "", false), helpers.NewPosition(1, 0, 1, 0, "", "", false)),
	}
}

// nextToken scans just far enough to return the next token. EOF is stable: all
// calls after the source is exhausted return the same EOF token.
func (lexer *tokenLexer) nextToken() Token {
	if lexer == nil || lexer.inner == nil {
		return Token{Kind: EOF, SubKind: NA, Value: "EOF"}
	}
	if lexer.finished {
		return lexer.eof
	}

	core := lexer.inner
	for !core.at_eof() {
		src := core.remainder()
		if r, size := utf8.DecodeRuneInString(src); r == utf8.RuneError && size == 1 {
			return lexer.emitUnknown(1, 0, 0)
		}
		if length, _, unsupported := detectUnsupportedLiteral(src); unsupported {
			lines, endColumn := multilineMetrics(src[:length])
			return lexer.emitUnknown(length, lines, endColumn)
		}
		if strings.HasPrefix(src, utf8BOM) {
			return lexer.emitUnknown(len(utf8BOM), 0, 0)
		}

		result, ok := core.scanToken(src)
		if !ok {
			r, size := utf8.DecodeRuneInString(src)
			if r == utf8.RuneError && size == 0 {
				size = 1
			}
			return lexer.emitUnknown(size, 0, 0)
		}

		switch result.action {
		case actionNewline:
			core.advanceN(result.length)
			core.advanceline(1)
		case actionSkip:
			core.advanceN(result.length)
			if result.lines > 0 {
				core.advanceline(result.lines)
				core.col = result.endColumn
			}
		case actionUnknown:
			return lexer.emitUnknown(result.length, result.lines, result.endColumn)
		case actionEmit:
			core.emitToken(result.kind, src[:result.length])
			token := core.Tokens[len(core.Tokens)-1]
			// Keep only the most recent emitted token for diagnostic context.
			core.Tokens = []Token{token}
			core.currentPos = 0
			return token
		}
	}

	lexer.finished = true
	core.Tokens = []Token{lexer.eof}
	core.currentPos = 0
	return lexer.eof
}

func (lexer *tokenLexer) emitUnknown(length, lines, endColumn int) Token {
	core := lexer.inner
	src := core.remainder()
	if length <= 0 || length > len(src) {
		length = 1
	}
	lexeme := src[:length]
	boundaryBefore := explicitSymbolBoundaryBefore(core.source, core.pos)
	boundaryAfter := explicitSymbolBoundaryAfter(core.source, core.pos+length)
	start := helpers.NewPosition(core.pos, core.line, core.col, core.pos, core.fn, core.currentLineText(), false)
	core.advanceN(length)
	if lines > 0 {
		core.advanceline(lines)
		core.col = endColumn
	}
	end := helpers.NewPosition(core.pos, core.line, core.col, core.pos, core.fn, core.currentLineText(), false)
	token := newUniqueToken(UNKNOWN, NA, lexeme, start.Copy(), end)
	token.BoundaryBefore = boundaryBefore
	token.BoundaryAfter = boundaryAfter
	core.Tokens = []Token{token}
	core.currentPos = 0
	return token
}

// TokenStream provides lazy parser-style lookahead over source text. UNKNOWN
// tokens pass through unchanged, including their original lexeme.
type TokenStream struct {
	lexer     *tokenLexer
	buffer    []Token
	history   []Token
	eof       Token
	exhausted bool
}

// NewTokenStream creates the internal lexer and its lazy parser-facing buffer.
// Constructing a stream copies source but does not scan it.
func NewTokenStream(source []byte, fn string) *TokenStream {
	return &TokenStream{
		lexer: newLexer(source, fn),
		eof:   Token{Kind: EOF, SubKind: NA, Value: "EOF"},
	}
}

// Peek returns the token n positions ahead without consuming it. Peek(0) is
// the next token. A negative lookahead is a programmer error and panics.
func (stream *TokenStream) Peek(n int) Token {
	if n < 0 {
		panic("scanlex.TokenStream.Peek: negative lookahead")
	}
	stream.ensure(n + 1)
	if n < len(stream.buffer) {
		return stream.buffer[n]
	}
	return stream.eof
}

// Next returns and consumes the next token.
func (stream *TokenStream) Next() Token {
	stream.ensure(1)
	if len(stream.buffer) == 0 {
		return stream.eof
	}

	token := stream.buffer[0]
	stream.history = append(stream.history, token)
	stream.buffer[0] = Token{}
	stream.buffer = stream.buffer[1:]
	if len(stream.buffer) == 0 {
		stream.buffer = nil
	}
	return token
}

// Previous returns a previously consumed token without changing the stream.
// Previous(1) is the most recently consumed token, Previous(2) is the token
// consumed before that, and so on. The result is false when the requested
// history does not exist. A non-positive lookbehind is a programmer error.
func (stream *TokenStream) Previous(n int) (Token, bool) {
	if n <= 0 {
		panic("scanlex.TokenStream.Previous: lookbehind must be positive")
	}

	index := len(stream.history) - n
	if index < 0 {
		return Token{}, false
	}
	return stream.history[index], true
}

// ensure lazily scans until count tokens are buffered or EOF is reached.
// Callers request parser-facing counts: Next requests one and Peek(n) requests
// n+1 because Peek(0) addresses the first buffered token.
func (stream *TokenStream) ensure(count int) {
	if count <= len(stream.buffer) || stream.exhausted {
		return
	}

	if stream.lexer == nil || stream.lexer.inner == nil {
		stream.exhausted = true
		return
	}

	for len(stream.buffer) < count {
		token := stream.lexer.nextToken()
		if token.Kind == EOF {
			stream.eof = token
			stream.exhausted = true
			break
		}
		stream.buffer = append(stream.buffer, token)
	}
}
