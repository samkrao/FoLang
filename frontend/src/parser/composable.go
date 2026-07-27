package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_composable parses a top-level identifier that may either refer to a
// previously declared type-as-value or fall back to a normal variable
// declaration.
//
// Feature examples:
//
//	Employee;
//	x co.lang.int;
//
// When the current identifier names a known type symbol, this produces a
// TypeComposeStmt. Otherwise it delegates to the regular variable declaration
// path so syntax like `x co.lang.int;` still works.
func parse_composable(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {

		if p.SymbolTable_.ExistsType(*p.Fs, p.currentToken().Value, string(symboltable.S_TypeSymbol)) {
			det := p.SymbolTable_.GetDetails(*p.Fs, p.currentToken().Value, string(symboltable.S_TypeSymbol))

			if string(det.GetSymbolType()) != "Type" {
				err_ := p.errorExpection("Need Type but found "+string(det.GetSymbolType()), helpers.InvalidSyntax)
				p.addErr(err_)
			} else {
				t_t, ok := det.(*symboltable.TypeSymbol)
				if !ok {
					err_ := p.errorExpection("Invalid type", helpers.IllegalVariableAssignment)
					p.addErr(err_)
				}
				t := ast.TypeComposeStmt{
					Type_:    t_t,
					TypeName: det.GetName(),
				}
				pr.Node = t
				p.advance()
				_, err_ := p.expect(scanlex.SEMI_COLON)
				p.addErr(err_)
				return pr
			}
		} else {
			err_ := p.errorExpection("Type doesnot defined ", helpers.InvalidSyntax)
			p.addErr(err_)
		}

	}

	pr = parse_var_decl_stmt(p, ddaps)
	return pr
}
