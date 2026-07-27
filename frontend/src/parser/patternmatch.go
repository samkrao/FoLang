package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_match_case_def_stmt is the statement-form entry point for chained
// pattern matching.
//
// Feature example:
//
//	value.match(co.pattern.Type)
//	    .case(num: co.lang.int => { this.return num; })
//	    .default({ this.return 0; });
//
// Cursor is at the method name (`match`, `case`, or `default`) on entry.
func parse_match_case_def_stmt(p *parser, curToken scanlex.Token, det symboltable.SymbolInfo, isval bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	_ = isval
	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	// Build the subject statement from the variable token
	subjectStmt := ast.SymbolRefExpr{
		Identifier: ast.SymbolExpr{
			Value:        curToken.Value,
			IsMethodCall: false,
			Symb: &symboltable.ExpressionSymbol{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_ExpressionSymbol),
				},
			},
			SymbolType_: string(symboltable.S_ExpressionSymbol),
		},
		AdditionalInfo: det,
		ExprType:       "GEN",
		Symb: &symboltable.Symbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_SymbolRefExpr),
			},
		},
	}

	pme := ast.PatternExprStmt{}
	pme.Stmt_ = subjectStmt

	// Current token should be "match" — consume it
	p.advance() // eat "match"

	// Optional matcher type: .match(co.pattern.XXX)
	//
	// Feature examples:
	//   .match(co.pattern.Type)
	//   .match(co.pattern.Value)
	//   .match(CustomMatcher)
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		p.advance() // eat '('
		if p.currentToken().Value == "co.pattern.Type" {
			pme.Matcher = true
			pme.MatcherType = "Type"
		} else if p.currentToken().Value == "co.pattern.Shape" {
			pme.Matcher = true
			pme.MatcherType = "Shape"
		} else if p.currentToken().Value == "co.pattern.Value" {
			pme.Matcher = true
			pme.MatcherType = "Value"
		} else if p.currentToken().Value == "co.pattern.Instance" {
			pme.Matcher = true
			pme.MatcherType = "Instance"
		} else if p.currentToken().Value == "co.pattern.Object" {
			pme.Matcher = true
			pme.MatcherType = "Object"
		} else if p.currentToken().Value == "co.pattern.Any" {
			pme.Matcher = true
			pme.MatcherType = "Any"
		} else if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {
			pme.Matcher = true
			pme.CustomMatcher = true
			pme.MatcherType = p.currentToken().Value
		}
		p.advance() // eat matcher value
		p.expect(scanlex.CLOSE_PAREN)
	}
	if !pme.Matcher {
		pme.MatcherType = "Any"
	}

	result := handle_match_expr(p, pme, false, ddaps)
	_, err_ := p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)
	pr.Node = result.Node
	return pr
}
