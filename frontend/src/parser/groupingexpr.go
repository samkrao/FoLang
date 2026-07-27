package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_grouping_expr parses a parenthesized expression or grouped
// statement-like construct.
//
// Feature examples:
//
//	(a + b)
//	(x > 0).do({ this.return x; })
//	(value).match(co.pattern.Any).default(0)
func parse_grouping_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_grouping_expr_st(p, ddaps, false)
}

// postGrouping dispatches suffix chains that follow a grouped expression.
//
// Feature examples:
//
//	(x > 0).do({ this.return x; })
//	(x > 0).return(x).otherwise(0)
//	(value).match(co.pattern.Type).default(0)
//	(let x := 10).where(x > 0)
func postGrouping(p *parser, inner bool, expr ParseResult, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	asCond := func(e any) (ast.ConditionalExpr, bool) {
		if ce, ok := e.(ast.ConditionalExpr); ok {
			return ce, true
		}
		if gr, ok := e.(ast.GroupingExpr); ok {
			if ce, ok := gr.Expr_.(ast.ConditionalExpr); ok {
				return ce, true
			}
		}
		return ast.ConditionalExpr{}, false
	}
	if _, ok := asCond(expr.Node); ok {
		switch p.nextTokenSafe(1).Value {
		case "do", "loop":
			return check_for_loops_or_conditions(p, &expr, inner, ddaps)
		case "return":
			return check_for_if_expr_or_ternary(p, &expr, inner, ddaps)
		default:
			p.addErr(p.errorObj(nil, "Unexpected token after grouping"))
			return expr
		}
	} else if p.nextTokenSafe(1).Value == "match" {
		return check_pattern_match(p, &expr, ddaps)
	} else if expr.LetBindings {
		p.advance() //eat dot
		p.advance() //eat where or in
		letBind := p.LetBinding
		p.LetBinding = true
		ex := parse_expr(p, defalt_bp, ddaps)
		p.LetBinding = letBind
		typ := ast.WHERE
		var stmt_ ast.Stmt

		var expr_ ast.Expr

		if p.Let {
			typ = ast.IN
			stmt_ = expr.Node.(ast.StatementExpr).Statement
			expr_ = ex.Node.(ast.Expr)
		} else {
			typ = ast.WHERE
			expr_ = expr.Node.(ast.Expr)
			stmt_ = ex.Node.(ast.StatementExpr).Statement
		}
		res := ast.LetExpr{
			Stmt_: stmt_,
			Expr_: expr_,
			Type_: typ,
			Symb: &symboltable.LetBindings{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_LetBindings),
				},
			},
		}
		expr.LetBindings = false
		return ParseResult{Node: res}

	}
	if se, ok := expr.Node.(ast.StatementExpr); ok {
		if bst, ok := se.Statement.(ast.BuiltInConstantStmt); ok {
			ce := ast.ConditionalExpr{BoolStmt: bst, Type: "TERNARY", ValOrVar: "VAL"}
			_ = ce
			if p.nextTokenSafe(1).Value == "return" {
				return check_for_if_expr_or_ternary(p, &expr, inner, ddaps)
			}
		}
		if vr, ok := se.Statement.(ast.SymbolRefExpr); ok {
			if vr.ExprType == "BOOL" {
				ce := ast.ConditionalExpr{BoolVarStmt: vr, Type: "TERNARY", ValOrVar: "VAR"}
				_ = ce
				switch p.nextTokenSafe(1).Value {
				case "do", "loop":
					return check_for_loops_or_conditions(p, &expr, inner, ddaps)
				case "return":
					return check_for_if_expr_or_ternary(p, &expr, inner, ddaps)
				case "match":
					return check_pattern_match(p, &expr, ddaps)
				}
			} else if p.nextTokenSafe(1).Value == "match" {
				return check_pattern_match(p, &expr, ddaps)

			}
		}
	}
	p.addErr(p.errorObj(nil, "Conditinal expression required"))
	return ParseResult{}
}

// parse_grouping_expr_st is the shared grouping parser used by both expression
// and statement-like grouped forms.
//
// Feature examples:
//
//	(a + b)
//	((a co.lang.int)->(co.lang.int) = { this.return a + 1; })
//	({ x := 10; }).where(x > 0)
//
// Besides ordinary parenthesized expressions, this helper also detects
// anonymous function syntax and let-binding style grouped forms.
func parse_grouping_expr_st(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt, b_inner ...bool) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}

	inner := false
	if len(b_inner) > 0 {
		inner = b_inner[0]
	}
	if !enterRec(p) {
		p.addErr(p.errorObj(nil, "recursion depth exceeded in grouping"))
		return res
	}
	defer leaveRec(p)

	// Detect anonymous function: (params) -> (returns) = { body }
	// Guard: skip if next token is also ( — that's IIFE grouping, not direct params.
	if p.nextTokenSafe(1).Kind != scanlex.OPEN_PAREN && checkFunctionType(p, 0) {
		return parse_anon_fn_expr(p, ddaps)
	}

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)
	i := 1
	var expr ParseResult
	checkCurly := false
	if p.nextToken(1).Kind == scanlex.OPEN_CURLY {
		checkCurly = true
	}
	for p.nextToken(i).Kind != scanlex.CLOSE_PAREN {

		i = i + 1
	}
	if p.LetBinding || p.nextToken(i+2).Value == "where" || (p.nextToken(i+2).Value == "in" && p.Let) {

		expr = parse_block_stmt_generic(p, false, checkCurly, ddaps)
		expr.Node = ast.StatementExpr{
			Statement: expr.Node.(ast.Stmt),
		}
		expr.LetBindings = true
	} else {
		expr = parse_expr(p, defalt_bp, ddaps)
	}
	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	if p.currentTokenKind() == scanlex.DOT {
		return postGrouping(p, inner, expr, ddaps)
	}
	res.Node = ast.GroupingExpr{Expr_: expr.Node.(ast.Expr)}
	return res
}
