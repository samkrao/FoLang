package parser

import (
	"slices"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

type Parser struct {
	ProjectDir    string
	InstallDir    string
	Stream        *scanlex.TokenStream
	ContextID     symboltable.ContextID
	SymoblTableID symboltable.SymbolTableID
	Symbols       *symboltable.FolangSymbols
}

func (parser Parser) Parse() ast.SET {

	parser.parseEntry()
	return ast.SourceFile{}
}

func (parser Parser) ParseOperators() *symboltable.FolangSymbols {
	return &symboltable.FolangSymbols{}
}

func (parser Parser) ParsePackaged() ast.SET {
	return ast.SourceFile{}
}

func (parser Parser) ParseComponents() ast.SET {
	return ast.SourceFile{}
}

var NonIgnoreSpaceOps []scanlex.TokenKind = []scanlex.TokenKind{scanlex.DOUBL_QUOTE, scanlex.SINGLE_QUOTE}
var ALLOWEDSPACES []scanlex.TokenKind = []scanlex.TokenKind{scanlex.SPACE, scanlex.NEWLINE}
var AllowedOPS []scanlex.SubKind = []scanlex.SubKind{scanlex.OPERATORS}

func (parser Parser) ignoreWP(tok *scanlex.Token) bool {
	if tok == nil {
		return false
	}

	if slices.Contains(ALLOWEDSPACES, tok.Kind) {
		return true
	} else if slices.Contains(AllowedOPS, tok.SubKind) {
		return true
	}
	return false
}

func (parser Parser) ignoreWSLB(tok *scanlex.Token) bool {
	if tok == nil {
		return false
	}

	if slices.Contains(ALLOWEDSPACES, tok.Kind) {
		return true
	} else if slices.Contains(AllowedOPS, tok.SubKind) && !slices.Contains(NonIgnoreSpaceOps, tok.Kind) {
		return true
	}
	return false
}

func (parser Parser) ignoreLB(tok *scanlex.Token) bool {
	if tok == nil {
		return false
	}

	if slices.Contains(ALLOWEDSPACES, tok.Kind) {
		return true
	} else if slices.Contains(AllowedOPS, tok.SubKind) && !slices.Contains(NonIgnoreSpaceOps, tok.Kind) {
		return true
	}
	return false
}

func (parser Parser) nextWOS() *scanlex.Token {
	for parser.Stream.Peek(0).Kind == scanlex.SPACE {
		parser.Stream.NextWH()
	}
	return parser.Stream.Next()
}

func (parser Parser) nextWOLB() *scanlex.Token {
	for parser.Stream.Peek(0).Kind == scanlex.NEWLINE {
		parser.Stream.NextWH()
	}
	return parser.Stream.Next()
}

func (parser Parser) nextWOSPLB() *scanlex.Token {
	for {
		tok := parser.Stream.Peek(0)
		if tok.Kind != scanlex.SPACE && tok.Kind != scanlex.NEWLINE {
			return parser.Stream.Next()
		}
		parser.Stream.NextWH()
	}
}

func (parser Parser) NextValid() *scanlex.Token {
	if parser.ignoreWSLB(parser.Peek(0)) {
		return parser.nextWOSPLB()
	} else if parser.ignoreWP(parser.Peek(0)) {
		return parser.nextWOS()
	} else if parser.ignoreLB(parser.Peek(0)) {
		return parser.nextWOLB()
	} else {
		return parser.Stream.Next()
	}

}
func (parser Parser) Next() *scanlex.Token {
	return parser.Stream.Next()
}

func (parser Parser) IsEOF() bool {
	return parser.Stream.AtEOF()
}

func (parser Parser) Previous(n int) (*scanlex.Token, bool) {
	return parser.Stream.Previous(n)
}

func (parser Parser) Peek(n int) *scanlex.Token {
	return parser.Stream.Peek(n)
}

func (parser Parser) PeekValid(n int) *scanlex.Token {
	return parser.Stream.Peek(n)
}
