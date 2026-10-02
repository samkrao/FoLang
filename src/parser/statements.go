package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseStatemtnsAndOrExpressions() ast.SET {
	for !parser.Stream.AtEOF() {

		parser.parseDecoratorAndorAnnotation()
		kind := parser.Stream.Peek(1)
		if kind.Kind == scanlex.BUILT_INS_FOL && kind.SubKind == scanlex.STATEMENT_EXPR {
			parser.parseVariableDeclaration()
			parser.parseTypeDefinitions()
		}
		parser.parseExpressions()

	}
	return ast.BlockStatement{}
}
