package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// check_for_loops_or_conditions converts a parsed conditional expression into a
// chained `.do(...)`, `.loop(...)`, `.otherwise(...)` statement form.
//
// Feature examples:
//
//	x > 0.do({ this.return x; })
//	flag.loop({ work(); }).otherwise({ stop(); })
func check_for_loops_or_conditions(p *parser, res *ParseResult, inner bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	//var errs []helpers.ErrorInterface
	if ce, ok := res.Node.(ast.ConditionalExpr); ok {
		pr := handle_do_or_loop(p, &ce, inner, ddaps)

		return pr
	} else if gr, ok := res.Node.(ast.GroupingExpr); ok {
		if ce, ok := gr.Expr_.(ast.ConditionalExpr); ok {
			return handle_do_or_loop(p, &ce, inner, ddaps)
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
				Type:     "LOOP_COND",
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
				Type:        "LOOP_COND",
				ValOrVar:    "VAR",
				Symb: &symboltable.ExpressionSymbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_ExpressionSymbol),
					},
				},
			}
		}
		if oky && (oka || oka1) {
			return handle_do_or_loop(p, &ce, inner, ddaps)
		} else {
			err_ := p.errorObj(nil, "Conditinal expression required")
			p.addErr(err_)
		}
	}
	return ParseResult{}
}

// handle_do_or_loop parses the control-flow suffixes attached to a
// ConditionalExpr.
//
// Feature examples:
//
//	x > 0.do({ this.return x; })
//	items.contains(v).loop({ process(v); }).otherwise({ stop(); })
//
// The first suffix chooses between a one-shot `.do(...)` and a looping
// `.loop(...)`, and any trailing `.otherwise(...)` chain is folded into elif or
// default branches.
func handle_do_or_loop(p *parser, ce *ast.ConditionalExpr, inner bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}
	cs := &ast.ConditionalStmt{}
	onlyLoop := true
	if p.currentTokenKind() == scanlex.DOT {
		if p.nextTokenSafe(1).Value == "do" || p.nextTokenSafe(1).Value == "loop" {
			// Feature examples:
			//   cond.do({ ... })
			//   cond.loop({ ... })
			//
			// Both forms share the same core structure and differ mainly in the
			// looping metadata stored on ast.ConditionalStmt.
			loop := false
			p.advance()
			current_token := p.advance()

			_, err_ := p.expect(scanlex.OPEN_PAREN)
			p.addErr(err_)

			bl := parse_block_stmt(p, ddaps)
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
			if current_token.Value == "loop" {
				loop = true
				cs.ContainsLoop = true
			} else {
				onlyLoop = false
			}
			cs.IfExpr = *ce
			cs.IfStmt = bl.Node.(ast.Stmt)
			cs.Loop = loop
			cs.OnlyLoop = onlyLoop
			cs.ISParentArrCont = false
			if ce.Type == "ARR_CONTAINS" {
				cs.ISParentArrCont = true
			}

		}
		if p.pos < len(p.tokens)-1 {
			for scanlex.DOT == p.currentToken().Kind {
				p.advance()
				if p.currentToken().Value == "otherwise" {
					p.advance()
					if p.currentToken().Kind == scanlex.OPEN_PAREN {
						// Feature example:
						//   cond.do({ ... }).otherwise(otherCond.do({ ... }))
						//
						// Grouped `otherwise(...)` is parsed as another conditional
						// chain and flattened into elif/default slots.
						elifpr := parse_grouping_expr_st(p, ddaps, true)
						elifSt := elifpr.Node.(ast.StatementExpr).Statement.(ast.ConditionalStmt)
						//cs.ElifExprStmt = append(cs.ElifExprStmt, elifSt)
						if !cs.ContainsLoop {
							cs.ContainsLoop = elifSt.ContainsLoop
						}
						if cs.OnlyLoop {
							cs.OnlyLoop = elifSt.OnlyLoop
						}
						arrCont := false
						if elifSt.ISParentArrCont {
							arrCont = true
						}
						nT := ast.ConditionalStmt{
							IfExpr:          elifSt.IfExpr,
							IfStmt:          elifSt.IfStmt,
							Loop:            elifSt.Loop,
							ContainsLoop:    elifSt.ContainsLoop,
							OnlyLoop:        elifSt.OnlyLoop,
							ISParentArrCont: arrCont,
							Symb: &symboltable.StatmentSymbol{
								SymbolDetails: symboltable.SymbolDetails{
									SymbolType_: string(symboltable.S_StatmentSymbol),
								},
							},
						}
						cs.ElifExprStmt = append(cs.ElifExprStmt, nT)
						cs.ElifExprStmt = append(cs.ElifExprStmt, elifSt.ElifExprStmt...)
						if elifSt.ElseExprStmt != nil {
							def := *elifSt.ElseExprStmt
							cs.ElseExprStmt = &def
						}

					} else {
						// Feature example:
						//   cond.do({ ... }).otherwise.do({ ... })
						//
						// Ungrouped chaining recursively parses the next conditional
						// suffix and stores it as the default branch.

						elseS := handle_do_or_loop(p, ce, true, ddaps)
						cond := elseS.Node.(ast.StatementExpr).Statement.(ast.ConditionalStmt)
						defCond := ast.DefaultConditionalStmt{
							Stmt_:        cond.IfStmt,
							ContainsLoop: cond.ContainsLoop,
							OnlyLoop:     cond.OnlyLoop,
							Loop:         cond.Loop,
							IsTernary:    false,
							Symb: &symboltable.StatmentSymbol{
								SymbolDetails: symboltable.SymbolDetails{
									SymbolType_: string(symboltable.S_StatmentSymbol),
								},
							},
						}
						cs.ElseExprStmt = &defCond
						if !cs.ContainsLoop {
							cs.ContainsLoop = cond.ContainsLoop
						}
						if cs.OnlyLoop {
							cs.OnlyLoop = cond.OnlyLoop
						}

						break

					}

				} else {
					err_ := p.errorObj(nil, "Invalid syntax expected do /loop /otherwise")
					p.addErr(err_)
				}
			}
		}

	}
	if cs.ElseExprStmt != nil && !inner {
		cs.ElseExprStmt.Expr_ = append(cs.ElseExprStmt.Expr_, cs.IfExpr)
		for _, nd := range cs.ElifExprStmt {
			cs.ElseExprStmt.Expr_ = append(cs.ElseExprStmt.Expr_, nd.IfExpr)
		}
	}
	pr.Node = ast.StatementExpr{
		Statement: *cs,
	}

	return pr
}
