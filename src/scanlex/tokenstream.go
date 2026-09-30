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
	lexer      *lexer
	foldBuffer []Token
	head       *Token
	current    *Token
}

// NewTokenStream creates the internal lexer and its lazy parser-facing buffer.
// A nil operator registry is treated as an empty registry.
// Constructing a stream copies source but does not scan it.
func NewTokenStream(source []byte, fn string, operators OperatorLookup) *TokenStream {
	lexer := newLexer(source, fn)
	lexer.operators = operators
	return &TokenStream{lexer: lexer}
}

// Current returns the next parser-visible token without consuming it. The
// first call lazily folds and links the first token.
func (stream *TokenStream) Current() *Token {
	if stream == nil {
		return nil
	}
	if stream.current != nil {
		return stream.current
	}
	if stream.head != nil {
		stream.current = stream.head
		return stream.current
	}

	node := stream.nextFoldedNode()
	stream.head = node
	stream.current = node
	return node
}

// RawFirst returns the first folded FoLang token in the immutable source view.
// Parser discards do not affect this traversal root.
func (stream *TokenStream) RawFirst() *Token {
	node := stream.Current()
	for node != nil && node.RawPrev != nil {
		node = node.RawPrev
	}
	return node
}

// Peek returns the token n positions ahead without consuming it. Peek(0) is
// Current(). A negative lookahead is a programmer error and panics.
func (stream *TokenStream) Peek(n int) *Token {
	if n < 0 {
		panic("scanlex.TokenStream.Peek: negative lookahead")
	}

	node := stream.Current()
	for i := 0; i < n && node != nil; i++ {
		if node.Kind == EOF {
			return node
		}
		node = stream.ensureNext(node)
	}
	return node
}

// AtEOF reports whether EOF is the current parser-facing token. It may lazily
// fold the first token, but it never advances the parser cursor.
func (stream *TokenStream) AtEOF() bool {
	token := stream.Current()
	return token == nil || token.Kind == EOF
}

// Next returns and consumes Current(). The token remains in both linked views,
// so it is available through Previous. EOF is stable and is never advanced.
func (stream *TokenStream) Next() *Token {
	token := stream.Current()
	if token == nil || token.Kind == EOF {
		return token
	}

	if token.State == TokenPending {
		token.State = TokenProcessed
	}
	stream.current = stream.ensureNext(token)
	return token
}

// DiscardCurrent unlinks Current() from the parser-visible chain and advances
// to its successor. Raw links are never changed, and EOF is never discarded.
func (stream *TokenStream) DiscardCurrent() *Token {
	discarded := stream.Current()
	if discarded == nil || discarded.Kind == EOF {
		return discarded
	}

	next := stream.ensureNext(discarded)
	previous := discarded.Prev
	if previous == nil {
		stream.head = next
	} else {
		previous.Next = next
	}
	if next != nil {
		next.Prev = previous
	}

	discarded.Prev = nil
	discarded.Next = nil
	discarded.State = TokenDiscarded
	stream.current = next
	return discarded
}

// Previous returns a parser-visible token before Current() without moving the
// cursor. Previous(1) is the most recently consumed retained token.
func (stream *TokenStream) Previous(n int) (*Token, bool) {
	if n <= 0 {
		panic("scanlex.TokenStream.Previous: lookbehind must be positive")
	}

	node := stream.Current()
	for i := 0; i < n; i++ {
		if node == nil || node.Prev == nil {
			return nil, false
		}
		node = node.Prev
	}
	return node, true
}

// MovePrevious moves Current() backward through the parser-visible chain.
func (stream *TokenStream) MovePrevious() (*Token, bool) {
	previous, ok := stream.Previous(1)
	if !ok {
		return nil, false
	}
	stream.current = previous
	return previous, true
}

func (stream *TokenStream) nextFoldedNode() *Token {
	token := foldTokens(stream)
	return &token
}

// ensureNext returns the parser-visible successor of node, lazily generating
// and linking one final folded token when node is currently the list tail.
func (stream *TokenStream) ensureNext(node *Token) *Token {
	if node == nil || node.Kind == EOF {
		return node
	}
	if node.Next != nil {
		return node.Next
	}

	next := stream.nextFoldedNode()
	node.Next = next
	next.Prev = node

	// Parser discards never change raw links. Usually node is also the raw tail;
	// walking to the raw tail keeps appending correct after prior discards.
	rawTail := node
	for rawTail.RawNext != nil {
		rawTail = rawTail.RawNext
	}
	rawTail.RawNext = next
	next.RawPrev = rawTail
	return next
}

// ensureRaw buffers scanner tokens only for folding lookahead. The buffer is
// temporary and is not either of the linked parser-facing token views.
func (stream *TokenStream) ensureRaw(count int) {
	if count <= len(stream.foldBuffer) {
		return
	}
	if stream.lexer == nil {
		return
	}

	for len(stream.foldBuffer) < count {
		token := stream.lexer.nextToken()
		if token.Kind == EOF {
			break
		}
		stream.foldBuffer = append(stream.foldBuffer, token)
	}
}

func (stream *TokenStream) peekRaw(n int) Token {
	stream.ensureRaw(n + 1)
	if n < len(stream.foldBuffer) {
		return stream.foldBuffer[n]
	}
	return stream.rawEOF()
}

func (stream *TokenStream) nextRaw() Token {
	stream.ensureRaw(1)
	if len(stream.foldBuffer) == 0 {
		return stream.rawEOF()
	}

	token := stream.foldBuffer[0]
	stream.foldBuffer[0] = Token{}
	stream.foldBuffer = stream.foldBuffer[1:]
	if len(stream.foldBuffer) == 0 {
		stream.foldBuffer = nil
	}
	return token
}

func (stream *TokenStream) rawEOF() Token {
	if stream == nil || stream.lexer == nil {
		return Token{Kind: EOF, SubKind: NA, Value: "EOF"}
	}
	return stream.lexer.nextToken()
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
			stream.foldBuffer = append([]Token{dot, segment}, stream.foldBuffer...)
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
