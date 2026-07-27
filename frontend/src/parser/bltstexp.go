package parser

import (
	"reflect"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

func parse_built_in_stmt_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	var val = p.currentToken().Value
	currTok := p.currentToken()
	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	body := make([]ast.Stmt, 0)
	if p.nextTokenSafe(1).Kind == scanlex.DOT {
		p.advance()

		symbTab := *p.SymbolTable_
		temp := symbTab.GetVarDetails(*p.Fs, val)
		//fmt.Println(temp)
		// handling a.do or a.loop when a is boolean variable. and ternary operations
		if (p.nextTokenSafe(1).Value == "do" || p.nextTokenSafe(1).Value == "loop") && temp.GetType() == "co.lang.bool" {
			ce := ast.ConditionalExpr{
				BoolVarStmt: ast.SymbolRefExpr{
					Identifier: ast.SymbolExpr{
						Value:        val,
						IsMethodCall: false,
						Symb: &symboltable.ExpressionSymbol{
							SymbolDetails: symboltable.SymbolDetails{
								SymbolType_: string(symboltable.S_ExpressionSymbol),
							},
						},
					},
					AdditionalInfo: fetchDetails(p, currTok, ddaps),
					ExprType:       "BOOL",
				},
				Type:     "LOOP_COND",
				ValOrVar: "VAR",
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
			return pr

		} else if (p.nextTokenSafe(1).Value == "return") && temp.GetType() == "co.lang.bool" {
			ce := ast.ConditionalExpr{
				BoolVarStmt: ast.SymbolRefExpr{
					Identifier: ast.SymbolExpr{
						Value:        val,
						IsMethodCall: false,
					},
					AdditionalInfo: fetchDetails(p, currTok, ddaps),
					ExprType:       "BOOL",
					Symb: &symboltable.Symbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_SymbolRefExpr),
						},
					},
				},
				Type:     "TERNARY",
				ValOrVar: "VAR",
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
			return pr

		} else if p.nextTokenSafe(1).Kind == scanlex.BUILT_IN_METHOD {
			p.advance()
			methName := p.currentToken().Value
			if slices.Contains(IteratorMethods, methName) {
				det := fetchDetails(p, currTok, ddaps)
				return parse_foreach_stmt(p, det, ddaps)
			} else if slices.Contains(ISIn, methName) {
				det := fetchDetails(p, currTok, ddaps)
				return parse_contains_stmt(p, currTok, det, ddaps)
			} else if slices.Contains(PMM, methName) {
				det := fetchDetails(p, currTok, ddaps)
				return parse_match_case_def_stmt(p, currTok, det, false, ddaps)
			} else {
				p.advance()
				meth := ast.SymbolExpr{
					Value:        methName,
					IsMethodCall: true,
					Symb: &symboltable.ExpressionSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_ExpressionSymbol),
						},
					},
				}
				expression := parse_call_expr(p, meth, defalt_bp, ddaps)
				body = append(body, ast.ExpressionStmt{
					Expression: expression.Node.(ast.Expr),
					Symb: &symboltable.StatmentSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_StatmentSymbol),
						},
					},
				})
				_, err_ := p.expectAny(scanlex.COMMA, scanlex.SEMI_COLON)
				p.addErr(err_)
				pr.Node = ast.BuiltInStmt{
					Body:  body,
					Value: val,
					Symb: &symboltable.StatmentSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_StatmentSymbol),
						},
					},
				}

				return pr
			}

		}
		p.backtrack()
	} else if _, ok := scanlex.Built_in_stmt_exprs[val]; ok {

		p.advance()
		if v, ok := ast.Stmt_to_Type[val]; ok {
			stm := reflect.New(v).Elem().Interface()
			switch br := stm.(type) {
			case ast.BreakStmt:
				if p.currentToken().Kind != scanlex.SEMI_COLON && p.currentTokenKind() != scanlex.COMMA {
					br.Args = p.currentToken().Value
				}
				pr.Node = br

			}

		} else {
			pr.Node = ast.BuiltInStmt{
				Body:  nil,
				Value: val,
				Symb: &symboltable.StatmentSymbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_StatmentSymbol),
					},
				},
			}
		}
		_, err_ := p.expectAny(scanlex.COMMA, scanlex.SEMI_COLON)
		p.addErr(err_)
		return pr
	}
	return ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
}
