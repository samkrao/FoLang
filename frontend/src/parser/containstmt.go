package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// parse_contains_stmt parses statement-form containment checks that continue
// into `.do(...)`, `.loop(...)`, or `.return(...)` chains.
//
// Feature examples:
//
//	items.contains(value).do({ process(value); });
//	items.contains(value).loop({ process(value); });
//	items.contains(value).return(found).otherwise(notFound);
//
// These forms are handled here because `a.contains(k)` is used directly as a
// statement/control-flow head rather than as a grouped expression.
func parse_contains_stmt(p *parser, curToken scanlex.Token, det symboltable.SymbolInfo, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	tk, ok := det.(*symboltable.VarSymbol)
	if !ok {
		err_ := p.errorObj(nil, "Un Supported Operation")
		p.addErr(err_)
		return pr
	}
	subType := tk.SubID
	currTok := p.currentToken().Value

	if v, ok := Iterator_meth_to_Type[currTok]; ok {
		if slices.Contains(v, subType) && slices.Contains(ISIn, currTok) {

			p.advance()
			_, err_ := p.expect(scanlex.OPEN_PAREN)
			p.addErr(err_)
			tr := parse_primary_expr(p, ddaps)
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
			if p.currentToken().Kind == scanlex.DOT {
				if p.nextTokenSafe(1).Value == "do" || p.nextTokenSafe(1).Value == "loop" {
					ce := ast.ConditionalExpr{
						ArrayVar: ast.SymbolRefExpr{
							Identifier: ast.SymbolExpr{
								Value:        curToken.Value,
								IsMethodCall: false,
								Symb: &symboltable.ExpressionSymbol{
									SymbolDetails: symboltable.SymbolDetails{
										SymbolType_: string(symboltable.S_ExpressionSymbol),
									},
								},
							},
							AdditionalInfo: fetchDetails(p, curToken, ddaps),
							ExprType:       "GEN",
							Symb: &symboltable.Symbol{
								SymbolDetails: symboltable.SymbolDetails{
									SymbolType_: string(symboltable.S_SymbolRefExpr),
								},
							},
						},
						CondVarStmt: tr.Node.(ast.Expr),
						Type:        "ARR_CONTAINS",
						ValOrVar:    "VAR",
						Symb: &symboltable.ExpressionSymbol{
							SymbolDetails: symboltable.SymbolDetails{
								SymbolType_: string(symboltable.S_ExpressionSymbol),
							},
						},
					}
					pr.Node = ast.ExpressionStmt{
						Expression: handle_do_or_loop(p, &ce, false, ddaps).Node.(ast.Expr),
						Symb: &symboltable.StatmentSymbol{
							SymbolDetails: symboltable.SymbolDetails{
								SymbolType_: string(symboltable.S_StatmentSymbol),
							},
						},
					}
					_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
					p.addErr(err_)
					return pr

				} else if p.nextTokenSafe(1).Value == "return" {
					ce := ast.ConditionalExpr{
						ArrayVar: ast.SymbolRefExpr{
							Identifier: ast.SymbolExpr{
								Value:        curToken.Value,
								IsMethodCall: false,
								Symb: &symboltable.ExpressionSymbol{
									SymbolDetails: symboltable.SymbolDetails{
										SymbolType_: string(symboltable.S_ExpressionSymbol),
									},
								},
							},
							AdditionalInfo: fetchDetails(p, curToken, ddaps),
							ExprType:       "GEN",
							Symb: &symboltable.Symbol{
								SymbolDetails: symboltable.SymbolDetails{
									SymbolType_: string(symboltable.S_SymbolRefExpr),
								},
							},
						},
						CondVarStmt: tr.Node.(ast.Expr),
						Type:        "ARR_CONTAINS",
						ValOrVar:    "VAR",
						Symb: &symboltable.ExpressionSymbol{
							SymbolDetails: symboltable.SymbolDetails{
								SymbolType_: string(symboltable.S_ExpressionSymbol),
							},
						},
					}
					pr.Node = ast.ExpressionStmt{
						Expression: handle_ifexpr_ternary(p, &ce, false, ddaps).Node.(ast.Expr),
						Symb: &symboltable.StatmentSymbol{
							SymbolDetails: symboltable.SymbolDetails{
								SymbolType_: string(symboltable.S_StatmentSymbol),
							},
						},
					}
					_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
					p.addErr(err_)
					return pr

				}
			}
		} else {
			err_ := p.errorObj(nil, "Un Supported Operation")
			p.addErr(err_)

		}
	} else {
		err_ := p.errorObj(nil, "Un Supported Operation")
		p.addErr(err_)

	}
	return pr

}
