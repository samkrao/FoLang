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

			} else if strings.HasPrefix(tok.Value, "@co.ddap") && (state == 0 || state == 1) {
				metaData.Kind_ = symboltable.Directive
				if state == 0 {
					state = 1
				}
			} else if strings.HasPrefix(tok.Value, "@co.dap") || tok.Kind == scanlex.CUSTOM_ANNOT_DECOR && (state == 1 || state == 2) {
				if parser.IsAnnotation(tok) {
					metaData.Kind_ = symboltable.Annotation
				} else {
					metaData.Kind_ = symboltable.Decorator
				}
				if state == 1 {
					state = 2
				}
			} else {
				//error consume token but add to errors

			}
			parser.expect(scanlex.OPEN_PAREN, "Missing opening parenthesis after directive")
			metaData.Attributes = parser.parseMetaData()
			parser.expect(scanlex.CLOSE_PAREN, "Missing closing parenthesis after directive")
			parser.expect(scanlex.NEWLINE, "Missing newline after directive")
			continue
		}

		break
	}
	return ast.SourceFile{}
}

func (parser Parser) parseMetaData() []symboltable.MetaDataValue {
	var metaDatas []symboltable.MetaDataValue = []symboltable.MetaDataValue{}
	for !parser.Stream.AtEOF() && parser.Stream.Peek(0).Kind != scanlex.CLOSE_PAREN {
		idx := 1

		for {
			if parser.Stream.Peek(1).Kind == scanlex.EQUALS {
				ok, tok := parser.expect(scanlex.IDENTIFIER, "Missing Key or Value in metadata")
				var value any
				parser.Stream.Next() // consume the equals sign
				value = parser.ParserMetaDataAttributeValue()
				if !ok {
					continue
				}
				metaDatas = append(metaDatas, symboltable.MetaDataValue{Key: tok.Value, Value: value, SourceIndex: idx})
			} else {
				value := parser.ParserMetaDataAttributeValue()
				metaDatas = append(metaDatas, symboltable.MetaDataValue{Value: value, OnlyValue: true, SourceIndex: idx})

			}
			if parser.Stream.Peek(0).Kind == scanlex.COMMA {
				parser.Stream.Next() // consume the comma
			} else {
				break
			}
			idx = idx + 1
		}

	}
	return metaDatas
}

func (parser Parser) ParserMetaDataAttributeValue() any {
	if parser.Stream.Peek(0).Kind == scanlex.STRING {
		return parser.Stream.Next().Value
	} else if parser.Stream.Peek(0).Kind == scanlex.CHAR {
		return parser.Stream.Next().Value
	} else if parser.Stream.Peek(0).Kind == scanlex.IDENTIFIER || parser.Stream.Peek(0).Kind == scanlex.COMPOSITE_IDENTIFIER || parser.Stream.Peek(0).Kind == scanlex.BUILT_INS_FOL || parser.Stream.Peek(0).Kind == scanlex.KEYWORD {
		return parser.Stream.Next().Value
	} else if parser.Stream.Peek(0).Kind == scanlex.NUMBER {
		return parser.Stream.Next().Value
	} else if parser.Stream.Peek(0).Kind == scanlex.BOOL {
		return parser.Stream.Next().Value
	} else {
		if parser.Stream.Peek(0).Kind == scanlex.OPEN_PAREN {
			parser.Stream.Next() // consume the open paren
			value := parser.parseMetaData()
			parser.expect(scanlex.CLOSE_PAREN, "Missing closing parenthesis after directive")
			return value
		} else if parser.Stream.Peek(0).Kind == scanlex.OPEN_BRACKET {
			parser.Stream.Next() // consume the open bracket
			value := parser.parseMetaData()
			parser.expect(scanlex.CLOSE_BRACKET, "Missing closing bracket after directive")
			return value
		} else if parser.Stream.Peek(0).Kind == scanlex.OPEN_CURLY {
			parser.Stream.Next() // consume the open curly
			value := parser.parseMetaData()
			parser.expect(scanlex.CLOSE_CURLY, "Missing closing curly after directive")
			return value
		} else {
			return nil
		}
	}

}
func (parser Parser) IsAnnotation(tok *scanlex.Token) bool {

	return false
}
