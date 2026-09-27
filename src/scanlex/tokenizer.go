package scanlex

// lexer owns only source-scanning state. TokenStream owns parser-facing token
// buffering, consumption, lookahead, and history.
type lexer struct {
	fn        string
	operators OperatorLookup
	source    string
	pos       int
	line      int
	col       int

	finished bool
	eof      Token

	lineText   string
	lineTextAt int
	lineTextOK bool

	// indentLevel is the per-lexer nesting depth of the optional debug trace.
	indentLevel int
}

// utf8BOM is the U+FEFF byte-order mark in its UTF-8 encoding.
var utf8BOM = string(rune(0xFEFF))

func (lex *lexer) advanceN(n int) {
	lex.pos += n
	lex.col += n
}

func (lex *lexer) advanceline(n int) {
	lex.line += n
	lex.invalidateCurrentLineText()
	lex.col = 0
}

func (lex *lexer) remainder() string {
	return lex.source[lex.pos:]
}

func (lex *lexer) at_eof() bool {
	return lex.pos >= len(lex.source)
}

func createLexer(source string, fn string) *lexer {
	return &lexer{
		line:   1,
		source: source,
		fn:     fn,
		col:    1,
	}
}
