package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_object_declaration_stmt parses a co.lang.object declaration:
//
//	myAnnotation co.lang.object->(for=annotation) = {
//	    value   co.lang.string;
//	    enabled co.lang.bool;
//	}
//
// The ->(for=...) clause specifies what the object is: annotation, directive, or pragma.
// Entry: current token is the name IDENTIFIER; next token is co.lang.object.
func parse_object_declaration_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)

	p.advance() // consume co.lang.object

	// Parse optional ->(for=annotation|directive|pragma)
	objectFor := ""
	if p.currentTokenKind() == scanlex.ARROW {
		p.advance() // consume ->
		_, err_ = p.expect(scanlex.OPEN_PAREN)
		p.addErr(err_)
		for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
			keyTk := p.advance()
			key := strings.TrimSuffix(keyTk.Value, "_fo")
			_, err_ = p.expect(scanlex.ASSIGNMENT)
			p.addErr(err_)
			valTk := p.advance()
			val := strings.TrimSuffix(valTk.Value, "_fo")
			val = strings.Trim(val, "\"")
			if key == "for" {
				objectFor = val
			} else {
				err_ = p.errorExpection("unknown object metadata key '"+key+"', expected 'for'", helpers.InvalidSyntax)
				p.addErr(err_)
			}
			if p.currentTokenKind() == scanlex.COMMA {
				p.advance()
			}
		}
		_, err_ = p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err_)
	}

	actCtxId := p.Context_
	actSymbId := p.SymbolTable_

	p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_ObjectSymbol))
	actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ContextType_ = string(symboltable.S_ObjectSymbol)
	p.Context_.ParentCtxSymbolTableId = actSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// Parse body like a struct: { field declarations }
	_, body := parse_type_definition(p, name, ddaps)

	typde := ast.ObjectDeclStmt{
		Name:      name.Value,
		Body:      body,
		Kind:      "using",
		ObjectFor: objectFor,
		Symb: &symboltable.ObjectSymbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_ObjectSymbol),
			},
		},
	}
	pr.Node = typde

	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	updateContext(p, pr.Node, false, false)

	return pr
}
