package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// check_for_if_expr_or_ternary converts a conditional expression into the
// chained `.return(...).otherwise(...)` form used for ternary-style statements.
//
// Feature examples:
//
//	x > 0.return(x).otherwise(0)
//	flag.return({ this.return 1; }).otherwise({ this.return 0; })
func check_for_if_expr_or_ternary(p *parser, res *ParseResult, inner bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	if ce, ok := res.Node.(ast.ConditionalExpr); ok {
		pr := handle_ifexpr_ternary(p, &ce, inner, ddaps)

		return pr
	} else if gr, ok := res.Node.(ast.GroupingExpr); ok {
		if ce, ok := gr.Expr_.(ast.ConditionalExpr); ok {
			return handle_ifexpr_ternary(p, &ce, inner, ddaps)
		} else {
			err_ := p.errorObj(nil, "Conditinal expression required")
			p.addErr(err_)

		}
	} else {
		stex, oky := res.Node.(ast.StatementExpr)
		bst, oka := stex.Statement.(ast.BuiltInConstantStmt)
		bst1, oka1 := stex.Statement.(ast.SymbolRefExpr)
		var ce ast.ConditionalExpr
		if oka {
			ce = ast.ConditionalExpr{
				BoolStmt: bst,
				Type:     "TERNARY",
				ValOrVar: "VAL",
				Symb: &symboltable.ExpressionSymbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_ExpressionSymbol),
					},
				},
			}
		} else if oka1 {
			ce = ast.ConditionalExpr{
				BoolVarStmt: bst1,
				Type:        "TERNARY",
				ValOrVar:    "VAR",

				Symb: &symboltable.ExpressionSymbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_ExpressionSymbol),
					},
				},
			}
		}
		if oky && (oka || oka1) {
			return handle_ifexpr_ternary(p, &ce, inner, ddaps)
		} else {
			err_ := p.errorObj(nil, "Conditinal expression required")
			p.addErr(err_)
		}
	}

	return ParseResult{}
}

// handle_ifexpr_ternary parses the ternary return chain attached to a
// ConditionalExpr.
//
// Feature examples:
//
//	x > 0.return(x).otherwise(0)
//	x > 0.return({ this.return x; }).otherwise({ this.return 0; })
//
// The first `.return(...)` captures the "then" branch, and repeated
// `.otherwise(...)` segments are folded into elif/default slots.
func handle_ifexpr_ternary(p *parser, ce *ast.ConditionalExpr, inner bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}
	cs := &ast.TernaryStmt{}
	if p.currentTokenKind() == scanlex.DOT {
		if p.nextTokenSafe(1).Value == "return" {
			// Feature example:
			//   cond.return(expr)
			//
			// The body inside `return(...)` is parsed as a statement so it can be
			// either a simple expression statement or a richer nested construct.
			p.advance()
			p.advance() //for eating up return
			_, err_ := p.expect(scanlex.OPEN_PAREN)
			p.addErr(err_)
			p.SKIP_SEMI_COMMA = true
			bl := parse_stmt(p, CODE, ddaps)
			p.SKIP_SEMI_COMMA = false
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)

			cs.Expr_ = *ce
			cs.Stmt_ = bl.Node.(ast.Stmt)
		}
		if p.pos < len(p.tokens)-1 {
			for scanlex.DOT == p.currentToken().Kind {
				p.advance()
				if p.currentToken().Value == "otherwise" {
					p.advance()
					if p.currentToken().Kind == scanlex.OPEN_PAREN {
						// Feature example:
						//   cond.return(a).otherwise(otherCond.return(b).otherwise(c))
						elifpr := parse_grouping_expr_st(p, ddaps, true)
						elifSt := elifpr.Node.(ast.StatementExpr).Statement.(ast.TernaryStmt)
						nT := ast.TernaryStmt{
							Expr_: elifSt.Expr_,
							Stmt_: elifSt.Stmt_,
							Symb: &symboltable.StatmentSymbol{
								SymbolDetails: symboltable.SymbolDetails{
									SymbolType_: string(symboltable.S_StatmentSymbol),
								},
							},
						}
						cs.ElifExprStmt = append(cs.ElifExprStmt, nT)
						cs.ElifExprStmt = append(cs.ElifExprStmt, elifSt.ElifExprStmt...)
						def := *elifSt.ElseExprStmt
						cs.ElseExprStmt = &def

					} else {
						// Feature example:
						//   cond.return(a).otherwise.return(b)
						//
						// Ungrouped chaining recurses into the next ternary segment
						// and stores it as the default branch.

						elseS := handle_ifexpr_ternary(p, ce, false, ddaps)
						cond := elseS.Node.(ast.StatementExpr).Statement.(ast.TernaryStmt)
						defCond := ast.DefaultConditionalStmt{
							Stmt_:     cond.Stmt_,
							IsTernary: true,
							Symb: &symboltable.StatmentSymbol{
								SymbolDetails: symboltable.SymbolDetails{
									SymbolType_: string(symboltable.S_StatmentSymbol),
								},
							},
						}
						cs.ElseExprStmt = &defCond

						break

					}

				} else {
					err_ := p.errorObj(nil, "Invalid syntax expected return/otherwise")
					p.addErr(err_)

				}
			}
		}

	}
	if cs.ElseExprStmt != nil {
		cs.ElseExprStmt.Expr_ = append(cs.ElseExprStmt.Expr_, cs.Expr_)
		for _, nd := range cs.ElifExprStmt {
			cs.ElseExprStmt.Expr_ = append(cs.ElseExprStmt.Expr_, nd.Expr_)
		}
	}
	pr.Node = ast.StatementExpr{
		Statement: *cs,
	}

	return pr
}
