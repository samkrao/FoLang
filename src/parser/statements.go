package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseStatemtnsAndOrExpressions() ast.SET {
	for !parser.Stream.AtEOF() {

		parser.parseDecoratorAndorAnnotation()
		kind := parser.Stream.Peek(1)
		if kind.Kind == scanlex.BUILT_INS_FOL && kind.SubKind == scanlex.STATEMENT_EXPR {

			kindTok := parser.Stream.Peek(1)
			if symb, ok := parser.Symbols.SystemSymbols[symboltable.QualifiedName(kindTok.Value)]; !ok {
				kind_ := parser.Symbols.SymbolsById[symb]
				type_ := kind_.SymbolTypeKind()
				if type_ == "BDTtype" {
					parser.parseVariableDeclaration()
				} else if _, ok := kind_.(symboltable.IKindSymbol); ok {
					// key does not exist
				} else if _, ok := kind_.(symboltable.TypeDef); ok {
					parser.parseTypeDefinitions()

				}
			}
		} else if kind.Kind == scanlex.COMPOSITE_IDENTIFIER {
			parser.parseVariableDeclaration()

		}
		parser.parseExpressions()

	}
	return ast.BlockStatement{}
}
