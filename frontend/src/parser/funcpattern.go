package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// isLetFuncPattern detects `let fib(n) = expr` when cursor is at the identifier
// (i.e. after 'let' has been consumed). Returns true when the token sequence is:
//
//	IDENTIFIER OPEN_PAREN ... CLOSE_PAREN ASSIGNMENT
func isLetFuncPattern(p *parser) bool {
	defer p.traceCurrent()()

	if p.currentTokenKind() != scanlex.IDENTIFIER {
		return false
	}
	if p.nextTokenSafe(1).Kind != scanlex.OPEN_PAREN {
		return false
	}
	// Scan forward to find the matching close paren (ignoring nested parens).
	depth := 0
	i := 1
	for {
		tk := p.nextTokenSafe(i).Kind
		if tk == scanlex.EOF {
			return false
		}
		if tk == scanlex.OPEN_PAREN {
			depth++
		} else if tk == scanlex.CLOSE_PAREN {
			if depth <= 1 {
				// i is at the matching close paren; check what follows
				return p.nextTokenSafe(i+1).Kind == scanlex.ASSIGNMENT
			}
			depth--
		}
		i++
	}
}

// parse_func_pattern_stmt parses a single function pattern arm of the form:
//
//	f (pattern1, pattern2, ...) => { body }
//
// The cursor must be at the function name identifier on entry.
// Multiple arms for the same function name are left for the compiler to merge.
func parse_func_pattern_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	nameTok := p.advance() // consume function name

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	// Parse comma-separated pattern expressions
	patterns := make([]ast.Expr, 0)
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
		pat := parse_expr(p, logical, ddaps)
		patterns = append(patterns, pat.Node.(ast.Expr))
		if p.currentTokenKind() == scanlex.COMMA {
			p.advance()
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)

	// Consume =>
	_, err_ = p.expect(scanlex.EQGT)
	p.addErr(err_)

	// Parse block body
	bodyPR := parse_block_stmt(p, ddaps)
	body := bodyPR.Node.(ast.BlockStmt).Body

	fn := ast.FunctionPatternStmt{
		Name:        nameTok.Value,
		PatternArgs: patterns,
		Body:        body,
		IsLetForm:   false,
		Symb: &symboltable.FunctionPattern{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_FunctionPattern),
			},
		},
	}
	return ParseResult{Node: fn}
}

// parse_let_func_pattern_stmt parses a let-style function pattern arm:
//
//	let fib(0) = 1
//	let fib(n) = fib(n-1) + fib(n-2)
//
// The cursor must be at the function name identifier on entry (after 'let' consumed).
func parse_let_func_pattern_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	nameTok := p.advance() // consume function name

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	// Parse comma-separated pattern expressions
	patterns := make([]ast.Expr, 0)
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
		pat := parse_expr(p, logical, ddaps)
		patterns = append(patterns, pat.Node.(ast.Expr))
		if p.currentTokenKind() == scanlex.COMMA {
			p.advance()
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)

	// Consume =
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// Parse single-expression body
	bodyExpr := parse_expr(p, assignment, ddaps)

	fn := ast.FunctionPatternStmt{
		Name:        nameTok.Value,
		PatternArgs: patterns,
		BodyExpr:    bodyExpr.Node.(ast.Expr),
		IsLetForm:   true,
		Symb: &symboltable.FunctionPattern{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_FunctionPattern),
			},
		},
	}

	_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)
	return ParseResult{Node: fn}
}
