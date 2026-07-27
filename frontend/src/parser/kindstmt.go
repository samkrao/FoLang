package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_kind_stmt parses a kind definition:
//
//	myKind co.lang.kind = <type expression>;
//
// A kind definition introduces a new kind (type-of-types).
// The RHS is a type expression or ADT expression.
func parse_kind_stmt(p *parser) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)

	// consume co.lang.kind or co.lang.kindkind
	if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.kind"); ok {
		p.advance()
	} else if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.kindkind"); ok {
		p.advance()
	} else {
		err_ = p.errorExpection("co.lang.kind expected but found "+p.currentToken().Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}

	actCtxId := p.Context_
	actSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_KindSymbol))
	actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ContextType_ = string(symboltable.S_KindSymbol)
	p.Context_.ParentCtxSymbolTableId = actSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// Parse the RHS as a type alias expression (type or ADT).
	flag, typeExpr, explicitType, typetype, funType, newTypeName := parse_type_alias_newtype(p, false, true, false, true, nil)
	if !flag {
		err_ = p.errorExpection("expected kind definition body", helpers.InvalidSyntax)
		p.addErr(err_)
	}
	symb := symboltable.TypeSymbol{
		Alias:   true,
		FunType: funType,
		UDT:     true,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_TypeSymbol),
		},
	}
	typde := ast.TypeDeclarationStmt{
		Name:        name.Value,
		Type_:       explicitType,
		TypeExpr:    typeExpr,
		Kind:        "kind",
		Typetype:    typetype,
		NewTypeName: newTypeName,
		Symb:        &symb,
	}

	if explicitType != nil {
		if bdt, ok := explicitType.(ast.BuiltInDataType); ok {
			typde.SubType_ = bdt.Value
		}
	}

	pr.Node = typde

	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	updateContext(p, pr.Node, false, false)
	return pr
}

func parse_kind_decl_stmt(p *parser, stmtK StmtKind) ParseResult {
	defer p.traceCurrent()()

	return parse_kind_stmt(p)
}
