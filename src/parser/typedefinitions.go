package parser

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/builtins"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseTypeDefinitions() ast.SET {

	ident := parser.nextWOS()

	typeTok := parser.nextWOS()
	parser.expect(scanlex.EQUALS, "Assignment is expected for type definitions")

	if typeTok.Value == "co.type" {
		if _, ok := builtins.BuilinTypes[parser.PeekWOS(0).Value]; ok {
			if parser.PeekWOS(1).Kind == scanlex.SEMI_COLON || parser.PeekWOS(1).Kind == scanlex.NEWLINE {
				alias := true
				fmt.Sprint(alias)
				fmt.Sprint(ident)
			}
		}

	} else if typeTok.Value == "co.newtype" {

	} else if typeTok.Value == "co.opaquetype" {

	} else if typeTok.Value == "co.predicatetype" {

	} else if typeTok.Value == "co.refinementtype" {

	} else if typeTok.Value == "co.supertype" {

	} else if typeTok.Value == "co.subtype" {

	} else if typeTok.Value == "co.polymorphic" {

	} else if typeTok.Value == "co.variants" {

	} else if typeTok.Value == "co.generic" {

	}

	return ast.TypeDefinition{}
}

func (parser Parser) parseType() ast.SET {
	return ast.TypeValueExpr{}
}
