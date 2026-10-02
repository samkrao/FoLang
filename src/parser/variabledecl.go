package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
)

func (parser Parser) parseVariableDeclaration() ast.SET {

	kindTok := parser.Stream.Peek(1)

	if _, ok := parser.Symbols.SystemSymbols[symboltable.QualifiedName(kindTok.Value)]; !ok {
		// key does not exist
	}
	return ast.Declaration{}
}
