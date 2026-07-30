package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// return-statement — section 10, plus the break and continue statements that share
// its spelling.
//
//	return-statement = ( "this" | "self" ), ".return",
//	                   [ expression-list ], statement-end
//
// A FoLang function may return several values, so the statement takes an expression
// LIST rather than a single expression:
//
//	this.return a + b;
//	this.return x, y;
//	this.return;
//
// The scanner folds `this.return`, `this.break` and `this.continue` into one
// BUIL_IN_STMT_EXPRS token each, so there is no "." to consume here — the verb
// arrives already attached to its receiver.

// parseReturnStatement parses the return-statement production.
func (p *parser) parseReturnStatement() ast.Stmt {
	p.advance() // the folded "this.return"

	var values []ast.Expr
	if p.startsExpression() {
		values = p.parseExpressionList()
	}

	// A returned anonymous function with a block body is self-terminating.
	//
	// DECISION-SYN-006's expression-brace rule normally means a braced expression leaves
	// its statement needing a ";", as in `this.return Employee{ id: 1 };`. A direct
	// anonymous function is the exception DECISION-SYN-007 carves out: its body ends at
	// its own closing brace. The reference's closure example returns one with no
	// terminator (docs/language-ref.md, "Closure"):
	//
	//	this.return (x co.lang.int) -> (co.lang.int) = {
	//	    sum += x;
	//	    this.return sum;
	//	}
	//
	// so a ";" is optional after such a return and required after every other.
	if endsWithAnonymousFunctionBody(values) {
		p.accept(scanlex.SEMI_COLON)
	} else {
		p.statementEnd("a return statement")
	}

	return ast.ReturnStmt{
		StmtExpr_:    p.returnPayload(values),
		MultiReturns: len(values) > 1,
		Symb:         p.stmtSymbol("this.return"),
	}
}

// endsWithAnonymousFunctionBody reports whether the last returned value is an anonymous
// function, whose block body is self-terminating.
func endsWithAnonymousFunctionBody(values []ast.Expr) bool {
	if len(values) == 0 {
		return false
	}
	_, isAnonymousFunction := values[len(values)-1].(ast.FunctionExpr)
	return isAnonymousFunction
}

// returnPayload packages a return statement's values into the single node the AST
// carries.
//
// No value is an empty return, one value is carried directly, and several are folded
// into a comma expression, which is the AST's multi-value expression carrier. In
// every case the payload is wrapped as a statement, because ReturnStmt.StmtExpr_ is
// typed as the shared ast.SET interface.
func (p *parser) returnPayload(values []ast.Expr) ast.SET {
	switch len(values) {
	case 0:
		return ast.ExpressionStmt{Symb: p.stmtSymbol("empty-return")}
	case 1:
		return ast.ExpressionStmt{
			Expression: values[0],
			Symb:       p.stmtSymbol("return-value"),
		}
	default:
		return ast.ExpressionStmt{
			Expression: p.foldComma(values),
			Symb:       p.stmtSymbol("return-values"),
		}
	}
}

// parseBreakStatement parses the `this.break` statement.
//
// FoLang has no break keyword: the statement is a built-in member of `this`, which is
// why it folds into a single token and is dispatched here rather than by a keyword.
// An optional argument names the enclosing label to break out of.
func (p *parser) parseBreakStatement() ast.Stmt {
	p.advance() // the folded "this.break"

	label := p.parseOptionalJumpLabel()
	p.statementEnd("a break statement")

	return ast.BreakStmt{Args: label, Symb: p.stmtSymbol("this.break")}
}

// parseContinueStatement parses the `this.continue` statement.
func (p *parser) parseContinueStatement() ast.Stmt {
	p.advance() // the folded "this.continue"

	label := p.parseOptionalJumpLabel()
	p.statementEnd("a continue statement")

	return ast.ContinueStmt{Args: label, Symb: p.stmtSymbol("this.continue")}
}

// parseOptionalJumpLabel parses the optional label of a break or continue statement,
// which targets a labeled block:
//
//	this.break(outer);
//	this.break outer;
func (p *parser) parseOptionalJumpLabel() string {
	if p.accept(scanlex.OPEN_PAREN) {
		if p.at(scanlex.CLOSE_PAREN) {
			p.advance()
			return ""
		}
		label := p.parseIdentifier("as a jump label").Scanned
		p.expect(scanlex.CLOSE_PAREN, "to close a jump label")
		return label
	}

	if p.atIdentifier() {
		return p.parseIdentifier("as a jump label").Scanned
	}
	return ""
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
func (p *parser) parseMultipleAssignmentStatement() ast.Stmt {
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

	return ast.ExpressionStmt{
		Expression: ast.AssignmentExpr{
			Assigne:       p.foldComma(targets),
			Operator:      opTok,
			AssignedValue: p.foldComma(values),
			Symb:          p.exprSymbol("multiple-assignment"),
		},
		Symb: p.stmtSymbol("multiple-assignment"),
	}
}

// parseAssignmentTarget parses the assignment-target production.
//
// A target is a postfix expression — a name, a member access or an index — or a
// parenthesised nested target list, which is what allows a nested destructuring.
func (p *parser) parseAssignmentTarget() ast.Expr {
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
func (p *parser) parseTupleAssignmentTarget() ast.Expr {
	p.expect(scanlex.OPEN_PAREN, "to open a tuple assignment target")

	targets := []ast.Expr{p.parseAssignmentTarget()}
	for p.accept(scanlex.COMMA) {
		targets = append(targets, p.parseAssignmentTarget())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a tuple assignment target")

	return ast.GroupingExpr{
		Expr_: p.foldComma(targets),
		Symb:  p.exprSymbol("tuple-target"),
	}
}
