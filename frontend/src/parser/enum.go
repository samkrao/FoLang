package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_enum_declaration_stmt parses an enum declaration:
//
//	myEnum co.lang.enum = {
//	    Variant1,
//	    Variant2,
//	    Variant3
//	}
//
// Entry: current token is the enum name IDENTIFIER; next token is co.lang.enum.
func parse_enum_declaration_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	// Consume the enum name.
	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)

	// Consume `co.lang.enum`.
	if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.enum"); ok {
		p.advance()
	} else {
		err_ = p.errorExpection("expected co.lang.enum but found "+p.currentToken().Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}

	actContext := p.Context_
	actSymTab := p.SymbolTable_
	// New context for the enum scope.
	p.Context_, p.SymbolTable_ = CreateNewContext(actContext.Id, string(symboltable.S_EnumSymbol))
	actContext.ChildCtxIds = append(actContext.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = actSymTab.Id
	p.Context_.ContextType_ = string(symboltable.S_EnumSymbol)
	//Registering new context and symbol table in FolangSymbols
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.ASSIGNMENT)

	p.addErr(err_)
	_, err_ = p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	// Parse comma-separated variant identifiers.
	variants := make([]ast.Stmt, 0)
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_CURLY && p.currentTokenKind() != scanlex.EOF {
		varTk, err_ := p.expect(scanlex.IDENTIFIER)
		p.addErr(err_)

		sd := symboltable.SymbolDetails{
			Name_:       varTk.Value,
			SymbolType_: string(symboltable.S_VarSymbol),
		}

		vd := symboltable.VariableDetails{
			SymbolDetails: sd,
			SubType_:      "ENUM_VARIANT",
		}
		vstmt := ast.BasicVarStmt{
			Identifier: varTk.Value,
			VarType:    name.Value,
		}
		symb := symboltable.VarSymbol{
			VariableDetails: vd,
		}

		v := ast.VarDeclarationStmt{
			BasicVarStmt: vstmt,
			Symb:         &symb,
		}
		variants = append(variants, v)

		if p.currentTokenKind() == scanlex.COMMA {
			p.advance()
		}
	}
	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)
	tsymb := symboltable.TypeSymbol{
		ADT:    true,
		UDT:    true,
		AsExpr: true,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_EnumSymbol),
		},
	}

	typde := ast.TypeDeclarationStmt{
		Name: name.Value,
		Body: variants,
		ADT_: "ENUM",
		Symb: &tsymb,
	}
	pr.Node = typde

	//from Enum Context to Parent Context
	p.Context_ = actContext
	p.SymbolTable_ = actSymTab

	updateContext(p, pr.Node, false, false)
	return pr
}
