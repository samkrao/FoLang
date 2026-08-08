package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Ternary chains — informative-ternary-chain of section 11a.
//
//	informative-ternary-chain =
//	    "(", expression, ")", ".return", "(", expression, ")",
//	    { ".otherwise", "(", expression, ")", ".return", "(", expression, ")" },
//	    ".otherwise", ".return", "(", expression, ")"
//
// Note the final `.otherwise.return(…)` is NOT optional here, unlike the else branch of a
// condition chain. A ternary is an expression that must produce a value on every path, so a
// chain without the final branch is not a ternary (docs/language-ref.md, "Ternary Operator"):
//
//	s = (truth).return(a).otherwise.return(b);
//	s = (a).return(x).otherwise(b).return(y).otherwise.return(z);
//
// The condition is evaluated first and exactly one result expression is then evaluated; the
// unselected expression must not be evaluated (docs/language-ref.md, "Conditional Expressions").
// Representing the chain as a TernaryStmt is what lets a backend honour that, rather than
// emitting all the branches as ordinary calls.

// lowerTernaryChain rewrites a ternary chain into an ast.TernaryStmt.
//
// It returns ok=false unless the chain matches the full canonical shape, final branch included.
func (p *parser) lowerTernaryChain(c chain) (ast.Stmt, bool) {
	// The chain must open with `.return(value)`.
	if c.verbAt(0) != verbReturn {
		return nil, false
	}
	firstValue, hasValue := singleArgument(c.segments[0])
	if !hasValue {
		return nil, false
	}

	root := ast.TernaryStmt{
		Span:  c.span,
		Expr_: p.lowerExpr(c.subject),
		Stmt_: p.ternaryResult(firstValue),
		Symb:  p.stmtSymbol("TernaryStmt"),
	}

	var elifs []ast.TernaryStmt
	var elseBranch *ast.DefaultConditionalStmt

	i := 1
	for i < len(c.segments) {
		if c.verbAt(i) != verbOtherwise || c.verbAt(i+1) != verbReturn {
			return nil, false
		}
		otherwise := c.segments[i]
		result := c.segments[i+1]

		value, hasResult := singleArgument(result)
		if !hasResult {
			return nil, false
		}

		if otherwise.called {
			cond, hasCond := singleArgument(otherwise)
			if !hasCond {
				return nil, false
			}
			elifs = append(elifs, ast.TernaryStmt{
				Span:  c.span,
				Expr_: p.lowerExpr(cond),
				Stmt_: p.ternaryResult(value),
				Symb:  p.stmtSymbol("TernaryStmt"),
			})
		} else {
			// The mandatory final branch; nothing may follow it.
			if i+2 != len(c.segments) {
				return nil, false
			}
			elseBranch = &ast.DefaultConditionalStmt{
				Span:      c.span,
				Default:   true,
				IsTernary: true,
				Expr_:     []ast.Expr{p.lowerExpr(value)},
				Symb:      p.stmtSymbol("DefaultConditionalStmt"),
			}
		}
		i += 2
	}

	// A ternary must be total: without the final unguarded branch there is a path that yields
	// no value, so this is not a ternary chain and is left as parsed.
	if elseBranch == nil {
		return nil, false
	}

	root.ElifExprStmt = elifs
	root.ElseExprStmt = elseBranch
	return root, true
}

// ternaryResult wraps a branch's result expression as the statement the node stores.
//
// ast.TernaryStmt holds each branch as a Stmt while the grammar's branches are expressions, so
// each is carried in an ExpressionStmt.
func (p *parser) ternaryResult(value ast.Expr) ast.Stmt {
	return ast.ExpressionStmt{
		Span:       spanOfNode(value, ast.Span{}),
		Expression: p.lowerExpr(value),
		Symb:       p.stmtSymbol("ternary-result"),
	}
}
