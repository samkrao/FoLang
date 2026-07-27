package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func parse_call_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	p.advance()

	arguments := make([]ast.Expr, 0)

	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
		tmp := parse_expr(p, assignment, ddaps)
		arguments = append(arguments, tmp.Node.(ast.Expr))

		if !p.currentToken().IsOneOfMany(scanlex.EOF, scanlex.CLOSE_PAREN) {
			_, err_ := p.expect(scanlex.COMMA)
			p.addErr(err_)
		}
	}

	_, err_ := p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	res.Node = ast.CallExpr{
		Method:      left,
		Arguments:   arguments,
		SymbolType_: "Call",
		Symb: &symboltable.ExpressionSymbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_ExpressionSymbol),
			},
		},
	}
	return res
}
