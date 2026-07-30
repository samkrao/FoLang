package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// The Pratt engine.
//
// Section 11 of docs/grammar/folang.ebnf spells the expression grammar out as a
// cascade of one production per precedence level — logical-or-expression calls
// logical-and-expression calls bitwise-or-expression and so on down to
// primary-expression. That cascade is exactly what precedence climbing collapses:
// this file replaces thirteen mutually recursive functions with one loop driven by
// the table in precedence.go, so adding an operator is a table edit and the
// declared precedence cannot drift out of step with the code that applies it.
//
// The layering below the infix loop is kept as explicit recursive descent,
// because the grammar's shape there is not "operand then operators":
//
//	unary-expression = { prefix-operator }, power-expression
//	power-expression = postfix-expression, [ "**", unary-expression ]
//	postfix-expression = primary-expression, { postfix-suffix | postfix-operator }
//
// Writing those three as written keeps "**" binding tighter than a prefix
// operator on its left but looser than one on its right, which is what makes
// `-a ** b` parse as `-(a ** b)`.

// parseExpression parses a complete expression production.
//
// This is the entry point every other file uses when the grammar says
// "expression". It starts at the lowest binding power, so assignment and every
// operator above it are in scope.
func (p *parser) parseExpression() ast.Expr {
	return p.parseExpr(bpNone)
}

// parseConstantExpression parses the constant-expression production, which the
// grammar defines as logical-or-expression: everything except assignment.
//
// It is used where an expression must not have an effect, such as an enum
// variant's value or an array dimension.
func (p *parser) parseConstantExpression() ast.Expr {
	return p.parseExpr(bpAssignment)
}

// parseExpr is the precedence-climbing loop.
//
// It parses one unary expression as the left operand, then repeatedly absorbs an
// infix operator whose binding power is at least minBP together with its right
// operand. Recursing with nextMinBindingPower is what implements associativity:
// see the comment on that function.
//
// minBP is the lowest precedence this call is willing to absorb. Passing bpNone
// means "absorb everything"; passing bpAssignment+1 would mean "stop before an
// assignment".
func (p *parser) parseExpr(minBP bindingPower) ast.Expr {
	defer p.enter()()

	left := p.parseUnary()

	for {
		op, ok := p.infixOperator()
		if !ok || op.bp < minBP {
			return left
		}
		// "|" is bitwise OR in a value expression but the union operator in a
		// type expression. The type parser never enters this loop, so reaching
		// here means the value reading is correct and no special case is needed.
		opTok := p.advance()

		switch op.role {
		case roleAssignment:
			left = p.finishAssignment(left, opTok, op)
		case roleRange:
			left = p.finishRange(left, opTok, op)
		default:
			right := p.parseExpr(nextMinBindingPower(op))
			left = ast.BinaryExpr{
				Left:     left,
				Operator: opTok,
				Right:    right,
				Symb:     p.exprSymbol(opTok.Value),
			}
		}
	}
}

// parseUnary parses the unary-expression production:
//
//	unary-expression = { prefix-operator }, power-expression
//
// Prefix operators are right-associative, which the right-recursive call gives
// for free, so `--count` and `!!flag` both parse.
func (p *parser) parseUnary() ast.Expr {
	defer p.enter()()

	if p.isPrefixOperator() && p.canStartPrefixOperator() {
		opTok := p.advance()
		return ast.PrefixExpr{
			Operator: opTok,
			Right:    p.parseUnary(),
			Symb:     p.exprSymbol(opTok.Value),
		}
	}
	return p.parsePower()
}

// canStartPrefixOperator guards against reading a token that merely looks like a
// prefix operator.
//
// Several prefix spellings are also infix or structural: "@" begins an annotation
// or a built-in directive when a name follows it, and "^" is bitwise XOR in infix
// position. Reaching parseUnary already means an operand is expected, so the only
// case left to exclude is the annotation form, where "@" is followed immediately
// by a name.
func (p *parser) canStartPrefixOperator() bool {
	if p.atOp("@") {
		return !p.atAny(scanlex.ATDAP, scanlex.BUILT_IN_DIRECTIVES, scanlex.CUSTOM_DIRECTIVES)
	}
	return true
}

// parsePower parses the power-expression production:
//
//	power-expression = postfix-expression, [ "**", unary-expression ]
//
// The right operand is a unary-expression, not a power-expression, so `2 ** -3`
// parses and the right-recursion through parseUnary makes `2 ** 3 ** 2` group as
// `2 ** (3 ** 2)`.
func (p *parser) parsePower() ast.Expr {
	left := p.parsePostfix(p.parsePrimary())

	if p.atOp("**") {
		opTok := p.advance()
		return ast.BinaryExpr{
			Left:     left,
			Operator: opTok,
			Right:    p.parseUnary(),
			Symb:     p.exprSymbol(opTok.Value),
		}
	}
	return left
}

// finishAssignment builds the assignment-expression production after the operator
// has been consumed.
//
// DECISION-OP-002: runtime assignment operators are right-associative, so
// `a = b = c` parses as `a = (b = c)`, and an assignment expression yields the
// assigned value. Recursing at the operator's own binding power is what produces
// the right-leaning tree.
//
// DECISION-OP-003 is enforced here too: ":=" and "?=" are statement-level
// definition operators, and "::=" is reserved. None of them is an
// assignment-expression operator, so meeting one in this position is a
// diagnostic rather than a parse.
func (p *parser) finishAssignment(target ast.Expr, opTok scanlex.Token, op infixOp) ast.Expr {
	value := p.parseExpr(nextMinBindingPower(op))
	return ast.AssignmentExpr{
		Assigne:       target,
		Operator:      opTok,
		AssignedValue: value,
		Symb:          p.exprSymbol(opTok.Value),
	}
}

// parseParenthesizedExpression parses "(" expression ")" and returns the inner
// expression, used where the grammar spells a parenthesised operand out rather
// than reaching it through primary-expression.
func (p *parser) parseParenthesizedExpression(context string) ast.Expr {
	p.expect(scanlex.OPEN_PAREN, "to open "+context)
	e := p.parseExpression()
	p.expect(scanlex.CLOSE_PAREN, "to close "+context)
	return e
}

// parseExpressionList parses the expression-list production:
//
//	expression-list = expression, { ",", expression }
func (p *parser) parseExpressionList() []ast.Expr {
	list := []ast.Expr{p.parseExpression()}
	for p.accept(scanlex.COMMA) {
		list = append(list, p.parseExpression())
	}
	return list
}
