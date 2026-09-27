package parser

import (
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

	return ast.Definition{}
}
