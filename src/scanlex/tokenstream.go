package scanlex

import (
	"strings"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/src/helpers"
)

// Lexer scans source bytes on demand. nextToken returns one lexical token at a
// time; whitespace, comments, and line breaks are consumed without being
// returned. These are lexical tokens: the legacy Tokenize functions additionally
// apply the historical dotted-name and built-in folding passes.
//
// The source bytes are copied into the scanner's immutable string storage. The
// caller may therefore reuse or modify source after NewLexer returns.
type Lexer struct {
	inner    *lexer
	finished bool
	eof      Token
}

// NewLexer creates a lazy lexer. Invalid or unsupported lexemes are returned as
// UNKNOWN tokens; the lexer does not report diagnostics.
func NewLexer(source []byte, fn string, custom *CustomOperators) *Lexer {
	return newLexer(source, fn, custom)
}

// NewLexerCollecting is kept as a compatibility alias for NewLexer. Lexical
// errors are represented as UNKNOWN tokens rather than collected diagnostics.
func NewLexerCollecting(source []byte, fn string, custom *CustomOperators) *Lexer {
	return NewLexer(source, fn, custom)
}

func newLexer(source []byte, fn string, custom *CustomOperators) *Lexer {
	text := string(source)
	// A leading BOM is source-file metadata and does not participate in token
	// positions, matching the established Tokenize API.
	text = strings.TrimPrefix(text, utf8BOM)
	core := createLexer(text, fn)
	core.custom = custom
	return &Lexer{
		inner: core,
		eof:   newUniqueToken(EOF, "EOF", helpers.NewPosition(1, 0, 1, 0, "", "", false), helpers.NewPosition(1, 0, 1, 0, "", "", false)),
	}
}

// nextToken scans just far enough to return the next token. EOF is stable: all
// calls after the source is exhausted return the same EOF token.
func (lexer *Lexer) nextToken() Token {
	if lexer == nil || lexer.inner == nil {
		return Token{Kind: EOF, Value: "EOF"}
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

func (lexer *Lexer) emitUnknown(length, lines, endColumn int) Token {
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
	token := newUniqueToken(UNKNOWN, lexeme, start.Copy(), end)
	token.BoundaryBefore = boundaryBefore
	token.BoundaryAfter = boundaryAfter
	core.Tokens = []Token{token}
	core.currentPos = 0
	return token
}

// Diagnostics remains for source compatibility. Lexical problems are tokens,
// so this always returns nil.
func (lexer *Lexer) Diagnostics() []helpers.ErrorInterface {
	return nil
}

// TokenStream provides parser-style lookahead over a Lexer. Before exposing a
// token it applies the same whole-stream folding contract as Tokenize: built-in
// names and types, composite identifiers, directives, and method calls retain
// their established parser-facing token kinds and values. UNKNOWN tokens pass
// through unchanged, including their original lexeme.
type TokenStream struct {
	lexer    *Lexer
	buffer   []Token
	eof      Token
	prepared bool
}

// NewTokenStream wraps lexer with a parser-facing folding buffer. The buffer is
// populated on first use, so constructing a stream does not scan source.
func NewTokenStream(lexer *Lexer) *TokenStream {
	return &TokenStream{lexer: lexer, eof: Token{Kind: EOF, Value: "EOF"}}
}

// Peek returns the token n positions ahead without consuming it. Peek(0) is
// the next token. A negative lookahead is a programmer error and panics.
func (stream *TokenStream) Peek(n int) Token {
	if n < 0 {
		panic("scanlex.TokenStream.Peek: negative lookahead")
	}
	stream.prepare()
	if n < len(stream.buffer) {
		return stream.buffer[n]
	}
	return stream.eof
}

// Next returns and consumes the next token.
func (stream *TokenStream) Next() Token {
	stream.prepare()
	if len(stream.buffer) == 0 {
		return stream.eof
	}

	token := stream.buffer[0]
	stream.buffer[0] = Token{}
	stream.buffer = stream.buffer[1:]
	if len(stream.buffer) == 0 {
		stream.buffer = nil
	}
	return token
}

// Diagnostics returns nil because lexical errors are returned as UNKNOWN
// tokens instead of diagnostics.
func (stream *TokenStream) Diagnostics() []helpers.ErrorInterface {
	return nil
}

// prepare materializes and folds the lexical stream once. Folding dotted names
// requires arbitrary forward context (for example, distinguishing a composite
// identifier from a method call), so exposing raw tokens incrementally would
// give TokenStream a different contract from Tokenize.
func (stream *TokenStream) prepare() {
	if stream.prepared {
		return
	}
	stream.prepared = true

	if stream.lexer == nil || stream.lexer.inner == nil {
		return
	}

	raw := make([]Token, 0)
	for {
		token := stream.lexer.nextToken()
		if token.Kind == EOF {
			stream.eof = token
			break
		}
		raw = append(raw, token)
	}

	core := stream.lexer.inner
	core.Tokens = raw
	core.currentPos = 0
	cleanupLB(core)
	foldTokens(core)
	foldSpecialStatementBuiltins(core)
	stream.buffer = append(stream.buffer, core.Tokens...)
}
