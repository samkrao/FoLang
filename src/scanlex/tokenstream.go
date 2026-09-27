package scanlex

import (
	"strings"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/src/helpers"
)

// lexer scans source bytes on demand. nextToken returns one lexical token at a
// time. Spaces and source line endings are tokens; line comments consume their
// terminating line ending, and all comments are consumed without being returned.
//
// The source bytes are copied into the scanner's immutable string storage. The
// caller may therefore reuse or modify source after NewTokenStream returns.
func newLexer(source []byte, fn string) *lexer {
	text := string(source)
	// A leading BOM is source-file metadata and does not participate in token
	// positions.
	text = strings.TrimPrefix(text, utf8BOM)
	return createLexer(text, fn)
}

// nextToken scans just far enough to return the next token. EOF is stable: all
// calls after the source is exhausted return the same EOF token.
func (lexer *lexer) nextToken() Token {
	if lexer == nil {
		return Token{Kind: EOF, SubKind: NA, Value: "EOF"}
	}
	if lexer.finished {
		return lexer.eof
	}

	core := lexer
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
			return core.emitNewline(src[:result.length])
		case actionSkip:
			core.advanceN(result.length)
			if result.lines > 0 {
				core.advanceline(result.lines)
				core.col = result.endColumn
			}
		case actionUnknown:
			return lexer.emitUnknown(result.length, result.lines, result.endColumn)
		case actionEmit:
			return core.emitToken(result.kind, result.subKind, src[:result.length])
		}
	}

	lexer.finished = true
	end := helpers.NewPosition(core.pos, core.line, core.col, core.pos, core.fn, core.currentLineText(), false)
	lexer.eof = newUniqueToken(EOF, NA, "EOF", end.Copy(), end)
	return lexer.eof
}

func (lexer *lexer) emitUnknown(length, lines, endColumn int) Token {
	core := lexer
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
	return token
}

// TokenStream provides lazy parser-style lookahead over source text. UNKNOWN
// tokens pass through unchanged, including their original lexeme.
type TokenStream struct {
	lexer        *lexer
	rawBuffer    []Token
	buffer       []Token
	history      []Token
	eof          Token
	rawExhausted bool
	exhausted    bool
}

// NewTokenStream creates the internal lexer and its lazy parser-facing buffer.
// Constructing a stream copies source but does not scan it.
func NewTokenStream(source []byte, fn string) *TokenStream {
	return NewTokenStreamWithOperators(source, fn, nil)
}

// NewTokenStreamWithOperators creates a stream whose scanner recognizes the
// supplied project-local operator spellings. Lexer construction remains an
// internal implementation detail.
func NewTokenStreamWithOperators(source []byte, fn string, custom *CustomOperators) *TokenStream {
	lexer := newLexer(source, fn)
	lexer.custom = custom
	return &TokenStream{
		lexer: lexer,
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

// AtEOF reports whether EOF is the next parser-facing token. It may scan and
// buffer the next token, but it never consumes it or changes token history.
func (stream *TokenStream) AtEOF() bool {
	return stream.Peek(0).Kind == EOF
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

// NextWH returns and consumes the next token without recording it in history.
func (stream *TokenStream) NextWH() Token {
	stream.ensure(1)
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

	if stream.lexer == nil {
		stream.exhausted = true
		return
	}

	for len(stream.buffer) < count {
		token := foldTokens(stream)
		if token.Kind == EOF {
			stream.eof = token
			stream.exhausted = true
			break
		}
		stream.buffer = append(stream.buffer, token)
	}
}

// ensureRaw buffers scanner tokens only for folding lookahead. These tokens are
// never exposed directly through Peek, Next, or Previous.
func (stream *TokenStream) ensureRaw(count int) {
	if count <= len(stream.rawBuffer) || stream.rawExhausted {
		return
	}
	if stream.lexer == nil {
		stream.rawExhausted = true
		return
	}

	for len(stream.rawBuffer) < count {
		token := stream.lexer.nextToken()
		if token.Kind == EOF {
			stream.eof = token
			stream.rawExhausted = true
			break
		}
		stream.rawBuffer = append(stream.rawBuffer, token)
	}
}

func (stream *TokenStream) peekRaw(n int) Token {
	stream.ensureRaw(n + 1)
	if n < len(stream.rawBuffer) {
		return stream.rawBuffer[n]
	}
	return stream.eof
}

func (stream *TokenStream) nextRaw() Token {
	stream.ensureRaw(1)
	if len(stream.rawBuffer) == 0 {
		return stream.eof
	}

	token := stream.rawBuffer[0]
	stream.rawBuffer[0] = Token{}
	stream.rawBuffer = stream.rawBuffer[1:]
	if len(stream.rawBuffer) == 0 {
		stream.rawBuffer = nil
	}
	return token
}

func (stream *TokenStream) foldDotted(first Token) (Token, string) {
	last := first
	var value strings.Builder
	value.WriteString(first.Value)

	for stream.rawStartsDottedSegment(last) {
		dot := stream.nextRaw()
		segment := stream.nextRaw()
		if dot.Kind != DOT || !tokensAdjacent(last, dot) ||
			(segment.Kind != IDENTIFIER && segment.Kind != KEYWORD) || !tokensAdjacent(dot, segment) {
			// rawStartsDottedSegment guarantees this shape; retain a defensive
			// fallback without discarding an unexpected scanner token.
			stream.rawBuffer = append([]Token{dot, segment}, stream.rawBuffer...)
			break
		}
		value.WriteString(dot.Value)
		value.WriteString(segment.Value)
		last = segment
	}
	return last, value.String()
}

func (stream *TokenStream) foldThisPath(first Token) (Token, string, bool) {
	if stream.rawStartsDottedSegment(first) {
		last, value := stream.foldDotted(first)
		return last, value, true
	}
	if !stream.rawStartsArrowSegment(first) {
		return first, first.Value, false
	}

	arrow := stream.nextRaw()
	segment := stream.nextRaw()
	value := first.Value + arrow.Value + segment.Value
	last := segment
	if stream.rawStartsDottedSegment(last) {
		tailLast, tailValue := stream.foldDotted(last)
		value += tailValue[len(last.Value):]
		last = tailLast
	}
	return last, value, true
}

func (stream *TokenStream) rawStartsDottedSegment(after Token) bool {
	return stream.rawStartsSeparatedSegment(after, ".", DOT)
}

func (stream *TokenStream) rawStartsArrowSegment(after Token) bool {
	return stream.rawStartsSeparatedSegment(after, "->", ARROW)
}

func (stream *TokenStream) rawStartsSeparatedSegment(after Token, separator string, kind TokenKind) bool {
	if stream == nil || stream.lexer == nil || after.EndPos == nil {
		return false
	}
	end := after.EndPos.Idx
	segmentStart := end + len(separator)
	if end < 0 || segmentStart >= len(stream.lexer.source) ||
		!strings.HasPrefix(stream.lexer.source[end:], separator) ||
		!isAlpha(stream.lexer.source[segmentStart]) {
		return false
	}

	separatorToken := stream.peekRaw(0)
	segment := stream.peekRaw(1)
	return separatorToken.Kind == kind && tokensAdjacent(after, separatorToken) &&
		(segment.Kind == IDENTIFIER || segment.Kind == KEYWORD) && tokensAdjacent(separatorToken, segment)
}

func (stream *TokenStream) rawByteAtEnd(token Token) byte {
	if stream == nil || stream.lexer == nil || token.EndPos == nil {
		return 0
	}
	end := token.EndPos.Idx
	if end < 0 || end >= len(stream.lexer.source) {
		return 0
	}
	return stream.lexer.source[end]
}

func (stream *TokenStream) followedByCall(token Token) bool {
	if stream == nil || stream.lexer == nil || token.EndPos == nil {
		return false
	}
	source := stream.lexer.source
	for i := token.EndPos.Idx; i < len(source); {
		switch {
		case source[i] == ' ' || source[i] == '\t' || source[i] == '\f' || source[i] == '\r' || source[i] == '\n':
			i++
		case strings.HasPrefix(source[i:], "//"):
			if end := strings.IndexAny(source[i+2:], "\r\n"); end >= 0 {
				i += 2 + end
			} else {
				return false
			}
		case strings.HasPrefix(source[i:], "/*"):
			if end := strings.Index(source[i+2:], "*/"); end >= 0 {
				i += 2 + end + 2
			} else {
				return false
			}
		default:
			return source[i] == '('
		}
	}
	return false
}

func tokensAdjacent(left, right Token) bool {
	return left.EndPos != nil && right.StartPos != nil && left.EndPos.Idx == right.StartPos.Idx
}
