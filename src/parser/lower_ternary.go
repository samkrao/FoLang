package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
)

// Ternary chains — informative-ternary-chain of section 12 of
// docs/grammar/folang.ebnf.
//
//	informative-ternary-chain =
//	    "(", expression, ")", ".then", "(", expression, ")",
//	    { ".otherwise", "(", expression, ")", ".then", "(", expression, ")" },
//	    ".default", "(", expression, ")"
//
// A ternary is the value-producing member of the same selection vocabulary the
// condition chain uses: `.then` selects, `.otherwise` introduces the next
// condition, and `.default` supplies the fallback. What separates the two is the
// ARGUMENT — a ternary's branches are values while a condition's are blocks — and
// the fact that the final `.default` is mandatory here. A ternary is an
// expression that must produce a value on every path, so a chain without it is
// not a ternary (docs/language-ref.md, "Ternary Operator"):
//
//	s = (truth).then(a).default(b);
//	s = (a).then(x).otherwise(b).then(y).default(z);
//
// The condition is evaluated first and exactly one result expression is then evaluated; the
// unselected expression must not be evaluated (docs/language-ref.md, "Conditional Expressions").
// Representing the chain as a TernaryStmt is what lets a backend honour that, rather than
// emitting all the branches as ordinary calls.

// lowerTernaryChain rewrites a ternary chain into an ast.TernaryStmt.
//
// It returns ok=false unless the chain matches the full canonical shape, terminal
// `.default` included.
func (p *parser) lowerTernaryChain(c chain) (ast.Stmt, bool) {
	// The chain must open with `.then(value)`.
	if c.verbAt(0) != verbThen {
		return nil, false
	}
	firstValue, hasValue := singleArgument(c.segments[0])
	if !hasValue {
		return nil, false
	}
	// A block argument means this is the condition chain rather than the ternary;
	// lowerConditionalChain owns that shape and runs after this one.
	if isBlockArgument(firstValue) {
		return nil, false
	}

	root := ast.TernaryStmt{NodeName: "TernaryStmt",
		Span:     c.span,
		Expr_:    p.lowerExpr(c.subject),
		Stmt_:    p.ternaryResult(firstValue),
		SymbolId: p.statementID("TernaryStmt"),
	}

	var elifs []ast.TernaryStmt
	var elseBranch *ast.DefaultConditionalStmt

	i := 1
	for i < len(c.segments) {
		// The mandatory terminal `.default(value)`; nothing may follow it.
		if c.verbAt(i) == verbDefault {
			if i+1 != len(c.segments) {
				return nil, false
			}
			value, hasFallback := singleArgument(c.segments[i])
			if !hasFallback || isBlockArgument(value) {
				return nil, false
			}
			elseBranch = &ast.DefaultConditionalStmt{NodeName: "DefaultConditionalStmt",
				Span:      c.span,
				Default:   true,
				IsTernary: true,
				Expr_:     []ast.Expr{p.lowerExpr(value)},
				SymbolId:  p.statementID("DefaultConditionalStmt"),
			}
			i++
			continue
		}

		if c.verbAt(i) != verbOtherwise || c.verbAt(i+1) != verbThen {
			return nil, false
		}
		otherwise := c.segments[i]
		result := c.segments[i+1]

		value, hasResult := singleArgument(result)
		if !hasResult || isBlockArgument(value) {
			return nil, false
		}
		cond, hasCond := singleArgument(otherwise)
		if !hasCond {
			return nil, false
		}
		elifs = append(elifs, ast.TernaryStmt{NodeName: "TernaryStmt",
			Span:     c.span,
			Expr_:    p.lowerExpr(cond),
			Stmt_:    p.ternaryResult(value),
			SymbolId: p.statementID("TernaryStmt"),
		})
		i += 2
	}

	// A ternary must be total: without the terminal fallback there is a path that yields
	// no value, so this is not a ternary chain and is left as parsed.
	if elseBranch == nil {
		return nil, false
	}

	root.ElifExprStmt = elifs
	root.ElseExprStmt = elseBranch
	return root, true
}

// isBlockArgument reports whether an argument is a braced block rather than a
// value. `.then` and `.default` take either, and which one they took is what
// tells the condition chain from the ternary chain.
func isBlockArgument(e ast.Expr) bool {
	wrapper, isWrapped := e.(ast.StatementExpr)
	return isWrapped && wrapper.Statement != nil
}

// ternaryResult wraps a branch's result expression as the statement the node stores.
//
// ast.TernaryStmt holds each branch as a Stmt while the grammar's branches are expressions, so
// each is carried in an ExpressionStmt.
func (p *parser) ternaryResult(value ast.Expr) ast.Stmt {
	return ast.ExpressionStmt{NodeName: "ExpressionStmt",
		Span:       spanOfNode(value, ast.Span{}),
		Expression: p.lowerExpr(value),
		SymbolId:   p.statementID("ternary-result"),
	}
}
