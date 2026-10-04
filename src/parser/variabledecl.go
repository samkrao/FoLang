package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (parser Parser) parseVariableDeclaration() ast.SET {

	ident_symb := symboltable.Variable{}
	tok := parser.nextWOS()
	nTok := parser.PeekWOS(0)
	redeclare := false
	initrequired := false
	if nTok.Kind == scanlex.WALRUS {
		initrequired = true
		ident_symb.IsAutoInferred = true
		parser.nextWOS()
	} else if nTok.Kind == scanlex.QEQ {
		ident_symb.IsAutoRedecl = true
		redeclare = false
		initrequired = true
		parser.nextWOS()
	} else if nTok.Kind == scanlex.COLON_WALRUS {
		ident_symb.IsDynamicAny = true
		redeclare = false
		initrequired = true
		parser.nextWOS()
	} else if nTok.Kind == scanlex.BUILT_INS_FOL || nTok.Kind == scanlex.COMPOSITE_IDENTIFIER {

		parser.parseType()
		if parser.PeekWOS(0).Kind == scanlex.EQUALS {
			initrequired = true
		}
	}

	if initrequired {
		parser.parseExpressions()
	}
	if !parser.Exists(tok, "var") {
		ident_symb.Name_ = symboltable.SymbolName(tok.Value)
	} else if redeclare {

	} else {

	}
	ident_symb.SymbolId_ = symboltable.SymbolID(tok.Value)
	parser.Symbols.SymbolsById[ident_symb.SymbolId_] = &ident_symb
	return ast.Declaration{
		DefinitionHeader: ast.DefinitionHeader{Symbol: ident_symb.SymbolId_}}
}

func (parser Parser) Exists(tok *scanlex.Token, symbolType string) bool {
	symbId := parser.SymoblTableID
	smbTable := parser.Symbols.GetSymbolTable(symbId)
	return smbTable.Exists(*parser.Symbols, tok.Value, symbolType)

}
