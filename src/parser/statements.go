package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/builtins"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseStatemtnsAndOrExpressions() ast.SET {
	for !parser.Stream.AtEOF() {

		parser.parseDecoratorAndorAnnotation()
		kind := parser.Peek(1)
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
		} else if kind.Kind == scanlex.WALRUS || kind.Kind == scanlex.QEQ || kind.Kind == scanlex.COLON_WALRUS {
			parser.parseVariableDeclaration()
		}
		parser.parseExpressions()
		parser.NewScope = true
	}
	return ast.BlockStatement{}
}
