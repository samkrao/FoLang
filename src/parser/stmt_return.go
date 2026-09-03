package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// return-statement — section 10.
//
//	return-statement = "this", ".return",
//	                   [ expression-list ], statement-end
//
// A FoLang function may return several values, so the statement takes an expression
// LIST rather than a single expression:
//
//	this.return a + b;
//	this.return x, y;
//	this.return;
//
// The scanner folds `this.return` into one BUIL_IN_STMT_EXPRS token, so there is
// no "." to consume here. `this.break` and `this.continue` fold the same way and are
// routed by the same dispatcher; see parseLoopControlStatement.

// parseReturnStatement parses the return-statement production.
//
// Implements: return-statement
func (p *parser) parseReturnStatement() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.advance() // the folded "this.return"

	var values []ast.Expr
	if p.startsExpression() {
		values = p.parseExpressionList()
	}

	p.statementEnd("a return statement")

	return ast.ReturnStmt{NodeName: "ReturnStmt", Span: p.spanFrom(spanStart), StmtExpr_: p.returnPayload(values),
		MultiReturns: len(values) > 1,
		Symb:         p.stmtSymbol("this.return"),
	}
}

// returnPayload packages a return statement's values into the single node the AST
// carries.
//
// No value is an empty return, one value is carried directly, and several are folded
// into a comma expression, which is the AST's multi-value expression carrier. In
// every case the payload is wrapped as a statement, because ReturnStmt.StmtExpr_ is
// typed as the shared ast.SET interface.
func (p *parser) returnPayload(values []ast.Expr) ast.SET {
	spanStart := p.pos
	switch len(values) {
	case 0:
		return ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), SymbolId: p.statementID("empty-return")}
	case 1:
		return ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: values[0],
			SymbolId: p.statementID("return-value"),
		}
	default:
		return ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: p.foldComma(values),
			SymbolId: p.statementID("return-values"),
		}
	}
}

// multiple-assignment-statement — section 10.
//
//	multiple-assignment-statement = assignment-target, ",", assignment-target,
//	                                { ",", assignment-target }, "=",
//	                                expression-list, statement-end
//	assignment-target             = postfix-expression | tuple-assignment-target
//	tuple-assignment-target       = "(", assignment-target, ",", assignment-target,
//	                                { ",", assignment-target }, ")"
//
// Multiple assignment is a STATEMENT rather than an expression because it has
// several destinations. Every right-hand side is evaluated completely before any
// target receives its value (docs/language-ref.md, "Multiple Assignment"), which is
// what permits an exchange with no temporary:
//
//	a, b = b, a;

// atMultipleAssignment reports whether the cursor begins a
// multiple-assignment-statement.
//
// The distinguishing shape is a comma-separated target list followed by a single "=".
// The probe walks to the "=" at bracket depth zero, which is what separates
// `a, b = b, a;` from a comma-separated declaration list like `a = 20, b = 30;`,
// where each item has its own "=".
func (p *parser) atMultipleAssignment() bool {
	return p.lookaheadOnly(func() bool {
		sawComma := false
		depth := 0

		for !p.atEOF() {
			switch p.kind() {
			case scanlex.OPEN_PAREN, scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY:
				depth++
			case scanlex.CLOSE_PAREN, scanlex.CLOSE_BRACKET, scanlex.CLOSE_CURLY:
				depth--
			case scanlex.COMMA:
				if depth == 0 {
					sawComma = true
				}
			case scanlex.SEMI_COLON:
				return false
			case scanlex.ASSIGNMENT:
				// Only a plain "=" separates targets from values; a compound
				// assignment has a single target.
				if depth == 0 && p.lexeme() == "=" {
					return sawComma
				}
			}
			p.advance()
		}
		return false
	})
}

// parseMultipleAssignmentStatement parses the multiple-assignment-statement
// production.
//
// Implements: multiple-assignment-statement
func (p *parser) parseMultipleAssignmentStatement() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	targets := []ast.Expr{p.parseAssignmentTarget()}
	for p.accept(scanlex.COMMA) {
		targets = append(targets, p.parseAssignmentTarget())
	}

	opTok := p.expectOp("=", "between the targets and the values of a multiple assignment")
	values := p.parseExpressionList()

	if len(values) != len(targets) {
		p.reportf(opTok, "a multiple assignment needs one value per target, but found %d target(s) and %d value(s)", len(targets), len(values))
	}

	p.statementEnd("a multiple assignment")

	return ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: ast.AssignmentExpr{NodeName: "AssignmentExpr", Span: p.spanFrom(spanStart), Assigne: p.foldComma(targets),
		Operator:      opTok,
		AssignedValue: p.foldComma(values),
		Symb:          p.exprSymbol("multiple-assignment"),
	},
		SymbolId: p.statementID("multiple-assignment"),
	}
}

// parseAssignmentTarget parses the assignment-target production.
//
// A target is a postfix expression — a name, a member access or an index — or a
// parenthesised nested target list, which is what allows a nested destructuring.
//
// Implements: assignment-target
func (p *parser) parseAssignmentTarget() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.at(scanlex.OPEN_PAREN) && p.looksLikeTupleAssignmentTarget() {
		return p.parseTupleAssignmentTarget()
	}
	return p.parsePostfix(p.parsePrimary())
}

// looksLikeTupleAssignmentTarget reports whether the "(" at the cursor opens a
// tuple-assignment-target, which needs at least two comma-separated members.
func (p *parser) looksLikeTupleAssignmentTarget() bool {
	return p.lookaheadOnly(func() bool {
		p.advance() // "("
		depth := 0
		for !p.atEOF() {
			switch p.kind() {
			case scanlex.OPEN_PAREN, scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY:
				depth++
			case scanlex.CLOSE_BRACKET, scanlex.CLOSE_CURLY:
				depth--
			case scanlex.CLOSE_PAREN:
				if depth == 0 {
					return false
				}
				depth--
			case scanlex.COMMA:
				if depth == 0 {
					return true
				}
			case scanlex.SEMI_COLON:
				return false
			}
			p.advance()
		}
		return false
	})
}

// parseTupleAssignmentTarget parses the tuple-assignment-target production.
//
// Implements: tuple-assignment-target
func (p *parser) parseTupleAssignmentTarget() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a tuple assignment target")

	targets := []ast.Expr{p.parseAssignmentTarget()}
	for p.accept(scanlex.COMMA) {
		targets = append(targets, p.parseAssignmentTarget())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a tuple assignment target")

	return ast.GroupingExpr{NodeName: "GroupingExpr", Span: p.spanFrom(spanStart), Expr_: p.foldComma(targets),
		Symb: p.exprSymbol("tuple-target"),
	}
}
