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
