package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseEntry() ast.SET {

	parser.collectPragmas()

	return ast.SourceFile{}
}

func (parser Parser) collectPragmas() ast.SET {

	for !parser.Stream.AtEOF() {

		if parser.Stream.Peek(0).Kind == scanlex.BUILT_INS_FOL && parser.Stream.Peek(0).SubKind == scanlex.DIRECTIVES {
			state := 0

			metaData := symboltable.MetaDataApplication{}
			tok := parser.Stream.Next()
			if strings.HasPrefix(tok.Value, "@co.pdap") && (state == 0 || state == 1) {
				metaData.Kind_ = symboltable.Pragma
				metaData.Attributes = []symboltable.MetaDataValue{}

			} else if strings.HasPrefix(tok.Value, "@co.ddap") && (state == 0 || state == 1) {

				if state == 0 {
					state = 1
				}
			} else if strings.HasPrefix(tok.Value, "@co.dap") && (state == 1 || state == 2) {

				if state == 1 {
					state = 2
				}
			} else {
				//error consume token but add to errors

			}
			continue
		}

		break
	}
	return ast.SourceFile{}
}
