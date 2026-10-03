package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/builtins"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseStatemtnsAndOrExpressions() ast.SET {
	for !parser.Stream.AtEOF() {

		parser.parseDecoratorAndorAnnotation()
		kind := parser.Stream.Peek(1)
		if kind.Kind == scanlex.BUILT_INS_FOL {

			if _, ok := builtins.BuilinTypes[kind.Value]; ok {
				parser.parseVariableDeclaration()
				continue
			} else if _, ok := builtins.BuiltinKinds[kind.Value]; ok {
				parser.ParserDefinitions()
				continue
			} else if _, ok := builtins.TypeForms[kind.Value]; ok {
				parser.parseTypeDefinitions()
				continue
			}
		} else if kind.Kind == scanlex.COMPOSITE_IDENTIFIER {
			parser.parseVariableDeclaration()
			continue
		}
		parser.parseExpressions()

	}
	return ast.BlockStatement{}
}
