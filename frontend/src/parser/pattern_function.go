package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Function-pattern clauses — section 9.
//
//	bare-function-pattern-clause      = identifier, pattern-parameter-list,
//	                                    [ where-clause ], "=>", pattern-result
//	capturing-function-pattern-clause = "let", identifier,
//	                                    pattern-parameter-list,
//	                                    [ where-clause ], "=", pattern-result
//	where-clause                      = ".where", "(", expression, ")"
//	pattern-result                    = block, body-closure-guard
//	                                  | non-block-expression, statement-end
//
// A function-pattern group is several clauses sharing one name, each matching a
// different argument shape (docs/language-ref.md, "Function Pattern"):
//
//	f(Some(x)) => { x + 1 }
//	f(None())  => { 0 }
//
//	fib(0) => 0;
//	fib(n) => fib(n-1) + fib(n-2);
//	classify(n).where(n > 0) => { this.return "positive"; }
//
// Merging the clauses of one name into a single function with a match expression in its
// body is the semantic phase's job; the parser emits one node per clause.
//
// The two forms differ in capture. A bare group cannot capture surrounding runtime
// variables; the `let` group must capture at least one already-initialized binding and
// is the only entry-file construct that may. Both are permitted in an application entry
// file as restricted dispatch helpers, and neither may escape as a function value.
//
// DECISION-SYN-006 gives pattern-result its two terminators: a block-bodied clause ends
// at "}" and takes no ";", while an expression-bodied clause takes one. Before revision
// 10 such a clause had no terminator at all.

// atBareFunctionPatternClause reports whether the cursor begins a
// bare-function-pattern-clause.
//
// The distinguishing shape is `name ( … ) [ .where( … ) ] "=>"`. The "=>" is what
// separates this from a function declaration, whose binding is "=" or a bare block,
// and from a call, which has no binding at all.
func (p *parser) atBareFunctionPatternClause() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)

		// An optional where clause sits between the patterns and the "=>".
		if p.at(scanlex.DOT) && p.isMemberNameToken(p.peek(1)) && logicalName(p.peek(1).Value) == "where" {
			p.advance()
			p.advance()
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			}
		}
		return p.atOp("=>")
	})
}

// parseBareFunctionPatternClause parses the bare-function-pattern-clause production.
func (p *parser) parseBareFunctionPatternClause() ast.Stmt {
	clauseName := p.parseIdentifier("as a function-pattern name")
	patterns := p.parsePatternParameterList()

	guard := p.parseOptionalWhereClause()

	p.expectOp("=>", "between the patterns and the result of a function-pattern clause")

	return p.finishFunctionPatternClause(clauseName, patterns, guard, false)
}

// parseCapturingFunctionPatternClause parses the
// capturing-function-pattern-clause production.
//
// This is the `let` form, which is the only entry-file construct permitted to capture a
// surrounding runtime binding:
//
//	offset := 100;
//	let adjust(0) = offset
//	let adjust(n) = n + offset
func (p *parser) parseCapturingFunctionPatternClause() ast.Stmt {
	p.expectKeyword("let", "to begin a capturing function-pattern clause")

	clauseName := p.parseIdentifier("as a function-pattern name")
	patterns := p.parsePatternParameterList()

	guard := p.parseOptionalWhereClause()

	p.expectOp("=", "between the patterns and the result of a capturing function-pattern clause")

	return p.finishFunctionPatternClause(clauseName, patterns, guard, true)
}

// parseOptionalWhereClause parses the where-clause production:
//
//	where-clause = ".where", "(", expression, ")"
//
// The clause guards the whole pattern: it is tested only after the patterns have
// matched, which is what makes `classify(n).where(n > 0)` well defined.
func (p *parser) parseOptionalWhereClause() ast.Expr {
	if !p.at(scanlex.DOT) || !p.atMemberNameAt(1, "where") {
		return nil
	}

	p.advance() // "."
	p.advance() // "where"

	p.expect(scanlex.OPEN_PAREN, "to open a where clause")
	guard := p.parseExpression()
	p.expect(scanlex.CLOSE_PAREN, "to close a where clause")

	return guard
}

// atMemberNameAt reports whether the token n positions ahead is a member name with the
// given logical spelling.
func (p *parser) atMemberNameAt(n int, want string) bool {
	tok := p.peek(n)
	return p.isMemberNameToken(tok) && logicalName(tok.Value) == want
}

// finishFunctionPatternClause parses the pattern-result and builds the clause node.
//
// letForm records which of the two clause forms was used, because the capture rules
// differ and the semantic phase needs to know which applies.
func (p *parser) finishFunctionPatternClause(clauseName name, patterns []pattern, guard ast.Expr, letForm bool) ast.Stmt {
	patternArgs := patternExprs(patterns)

	// A guard is attached as an extra pattern argument so the clause carries both
	// the shapes it matches and the condition it requires.
	if guard != nil {
		patternArgs = append(patternArgs, ast.ConditionalExpr{
			Left:        guard,
			CondValStmt: guard,
			Type:        "where",
			Symb:        p.exprSymbol("where"),
		})
	}

	clause := ast.FunctionPatternStmt{
		Name:        clauseName.Scanned,
		PatternArgs: patternArgs,
		IsLetForm:   letForm,
		Symb:        p.patternSymbol(clauseName.Scanned, letForm),
	}

	// pattern-result: a block body ends at "}", an expression body at ";".
	if p.at(scanlex.OPEN_CURLY) && p.startsDirectBody() {
		body := p.parseBlock("a function-pattern body")
		p.bodyClosureGuard("a function-pattern body")
		clause.Body = statementsOf(body)
		return clause
	}

	clause.BodyExpr = p.parseExpression()

	// The reference's entry-file examples omit the terminator on a let clause, so a
	// missing ";" there is tolerated rather than reported.
	if !p.accept(scanlex.SEMI_COLON) && !letForm {
		p.statementEnd("an expression-bodied function-pattern clause")
	}
	return clause
}
