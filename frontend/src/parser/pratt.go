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
// Fixed postfix suffixes remain an explicit recursive-descent layer at binding
// power 100. Prefix operators and exponentiation participate in the same Pratt
// binding-power mechanism as project operators:
//
//	prefix operator: parse its operand at the prefix's binding power
//	"**": right-associative infix entry at binding power 90
//	postfix suffix: parsed directly from a primary at binding power 100
//
// This preserves `-a ** b` as `-(a ** b)` and admits `2 ** -3`, while allowing
// a declared operator at (for example) 95 to bind more tightly than power.
// The EBNF power-expression production is the right-associative "**" entry in
// builtinInfixOperators; unary-expression is implemented by parseUnary below.

// parseExpression parses a complete expression production.
//
// This is the entry point every other file uses when the grammar says
// "expression". It starts at the lowest binding power, so assignment and every
// operator above it are in scope.
//
// Implements: expression
func (p *parser) parseExpression() ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseExpr(bpNone)
}

// parseConstantExpression parses the constant-expression production, which the
// grammar defines as logical-or-expression: everything except assignment.
//
// DECISION-OP-007 cannot be expressed by a binding-power cutoff: project
// operators may legally use precedence 0 through 100, including values below
// assignment's precedence. Instead this entry point installs an
// assignment-forbidden mode. Nested grouped expressions and call arguments
// inherit the mode through parseExpression.
//
// Implements: constant-expression
func (p *parser) parseConstantExpression() ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expressionModes = append(p.expressionModes, expressionModeNoAssignment)
	defer func() {
		p.expressionModes = p.expressionModes[:len(p.expressionModes)-1]
	}()
	return p.parseExpr(bpNone)
}

// expressionMode carries restrictions that are independent of precedence.
// Keeping assignment permission separate from minBP is essential because a
// custom operator at precedence 0 is still part of a constant-expression.
type expressionMode uint8

const (
	expressionModeNormal expressionMode = iota
	expressionModeNoAssignment
)

// currentExpressionMode returns the restriction inherited by a nested
// expression parse. Ordinary expression entry points run in normal mode.
func (p *parser) currentExpressionMode() expressionMode {
	if len(p.expressionModes) == 0 {
		return expressionModeNormal
	}
	return p.expressionModes[len(p.expressionModes)-1]
}

// parseExpr is the precedence-climbing loop.
//
// It parses one unary expression as the left operand, then repeatedly absorbs an
// infix operator whose binding power is at least minBP together with its right
// operand. Recursing with nextMinBindingPower is what implements associativity:
// see the comment on that function.
//
// minBP is the lowest precedence this call is willing to absorb. Passing bpNone
// means "absorb everything". Grammar restrictions that are not precedence rules,
// such as constant-expression excluding assignment, are represented by
// expressionMode rather than by an artificial binding-power threshold.
func (p *parser) parseExpr(minBP bindingPower) ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseExprWithContext(minBP, nil)
}

// parseExprWithContext carries the right-associative infix whose operand this
// invocation is parsing. A right-associative operator admits another operator
// at equal binding power into that recursive operand; retaining the parent is
// therefore necessary to reject an equal-precedence non-associative operator
// there. Explicit grouping calls parseExpression and starts with nil again.
func (p *parser) parseExprWithContext(minBP bindingPower, enclosingEqual *infixOp) ast.Expr {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	left := p.nud(enclosingEqual)
	var previousInfix infixOp
	hasPreviousInfix := false
	// An open-lower range is parsed in operand position by parsePrefixRange,
	// before this infix loop starts. Seed the same-precedence history that an
	// ordinary range operator would have established in the loop, so
	// `.. upper OP value` cannot evade range non-associativity when OP also has
	// precedence 55. A grouped range is an ast.GroupingExpr and intentionally
	// does not seed this history.
	if _, startsWithOpenLowerRange := left.(ast.RangeExpr); startsWithOpenLowerRange {
		previousInfix = infixOp{
			lexeme: "open-lower range",
			bp:     bpRange,
			assoc:  nonAssoc,
			role:   roleRange,
		}
		hasPreviousInfix = true
	}

	for {
		// A declared postfix operator participates in the same dynamic binding
		// table as custom infix operators. Built-in postfix syntax remains in
		// parsePostfix because its precedence is fixed by the grammar.
		if bp, ok := p.ops.postfix[p.lexeme()]; ok && bp >= minBP {
			opTok := p.advance()
			p.requirePostfixOperatorBoundary(opTok)
			left = ast.PrefixExpr{Span: p.spanFrom(spanStart), Operator: opTok,
				Right: left,
				Symb:  p.exprSymbol("postfix" + opTok.Value),
			}
			// A custom postfix result is an ordinary expression value. Therefore
			// the grammar's fixed, highest-binding suffixes may immediately
			// continue from it: `value %% .field`, `value %% ()`, and
			// `value %% [0]` (assuming %% is the registered postfix symbol).
			// Feeding the completed node back through the suffix loop preserves
			// their source order without assigning the custom operator the fixed
			// bpPostfix binding used only by built-ins.
			left = p.parsePostfix(left)
			continue
		}

		// A pre-declared glyph is a complete token the scanner recognises, but the
		// current alpha profile gives it no expression semantics. Reporting it here
		// — where an operator would otherwise be looked up — is what turns it into
		// the unsupported-operator error the reference requires, rather than the
		// missing-terminator error a silently unrecognised infix produces
		// (docs/language-ref.md, C.10).
		if scanlex.IsPredeclaredOperatorSpelling(p.lexeme()) {
			p.reportPredeclaredOperatorGlyph()
		}

		op, ok := p.infixOperator()
		if !ok || op.bp < minBP {
			return left
		}
		if op.role == roleAssignment && p.currentExpressionMode() == expressionModeNoAssignment {
			p.failf(p.cur(), "assignment operator %q is not allowed in a constant expression", p.lexeme())
		}
		if enclosingEqual != nil && enclosingEqual.bp == op.bp &&
			(enclosingEqual.assoc == nonAssoc || op.assoc == nonAssoc) {
			p.failf(p.cur(), "non-associative operator %q cannot share precedence %d with the unparenthesized right operand of %q; parenthesize the intended grouping", p.lexeme(), op.bp, enclosingEqual.lexeme)
		}
		// Non-associativity is symmetric at one unparenthesized precedence
		// level. Reject both `a OP b + c` and `a + b OP c` when OP and + have
		// equal binding power. A parenthesized subexpression runs a fresh
		// parseExpr invocation, intentionally resetting this history.
		if hasPreviousInfix && previousInfix.bp == op.bp &&
			(previousInfix.assoc == nonAssoc || op.assoc == nonAssoc) {
			p.failf(p.cur(), "non-associative operator at precedence %d cannot be chained with %q; parenthesize the intended grouping", op.bp, p.lexeme())
		}
		// "|" is bitwise OR in a value expression but the union operator in a
		// type expression. The type parser never enters this loop, so reaching
		// here means the value reading is correct and no special case is needed.
		opTok := p.advance()
		if op.role != roleRange {
			p.requireInfixOperatorBoundaries(opTok)
		}

		left = p.led(left, opTok, op, spanStart)

		previousInfix = op
		hasPreviousInfix = true
	}
}

// nud is the Pratt null-denotation stage. Keeping it explicit makes the debug
// trace show the operand-producing phase independently from the expression loop.
func (p *parser) nud(enclosingEqual *infixOp) ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}
	return p.parseUnary(enclosingEqual)
}

// led is the Pratt left-denotation stage for one consumed infix operator.
func (p *parser) led(left ast.Expr, opTok scanlex.Token, op infixOp, spanStart int) ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}
	switch op.role {
	case roleAssignment:
		return p.finishAssignment(left, opTok, op)
	case roleRange:
		return p.finishRange(left, opTok, op)
	default:
		right := p.parseInfixRightOperand(op)
		return ast.BinaryExpr{Span: p.spanFrom(spanStart), Left: left,
			Operator: opTok,
			Right:    right,
			Symb:     p.exprSymbol(opTok.Value),
		}
	}
}

// parseUnary is the Pratt null-denotation for prefix operators. Each prefix
// parses its operand at its own binding power, so built-in prefix precedence 80
// and a custom prefix precedence remain comparable with every infix entry.
// Consequently `- -count` and `! !flag` parse, while a contiguous unknown run
// such as `--` is never split into two prefix operators.
func (p *parser) parseUnary(enclosingEqual *infixOp) ast.Expr {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	if bp, custom := p.ops.prefix[p.lexeme()]; custom && p.canStartPrefixOperator() {
		opTok := p.advance()
		p.requirePrefixOperatorBoundary(opTok)
		return ast.PrefixExpr{Span: p.spanFrom(spanStart), Operator: opTok,
			Right: p.parseExprWithContext(bp, enclosingEqual),
			Symb:  p.exprSymbol(opTok.Value),
		}
	}
	if p.isPrefixOperator() && p.canStartPrefixOperator() {
		opTok := p.advance()
		return ast.PrefixExpr{Span: p.spanFrom(spanStart), Operator: opTok,
			Right: p.parseExprWithContext(bpPrefix, enclosingEqual),
			Symb:  p.exprSymbol(opTok.Value),
		}
	}
	return p.parsePostfix(p.parsePrimary())
}

// parseInfixRightOperand preserves an equal-precedence parent only for a
// right-associative operator. Left and non-associative operators raise the
// recursive minimum above their own binding power, so their operand cannot
// absorb an equal-precedence infix operator.
func (p *parser) parseInfixRightOperand(op infixOp) ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	var enclosing *infixOp
	if op.assoc == rightAssoc {
		copy := op
		enclosing = &copy
	}
	return p.parseExprWithContext(nextMinBindingPower(op), enclosing)
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
//
// Implements: assignment-expression
func (p *parser) finishAssignment(target ast.Expr, opTok scanlex.Token, op infixOp) ast.Expr {
	spanStart := p.pos
	value := p.parseInfixRightOperand(op)
	return ast.AssignmentExpr{Span: p.spanFrom(spanStart), Assigne: target,
		Operator:      opTok,
		AssignedValue: value,
		Symb:          p.exprSymbol(opTok.Value),
	}
}

// parseParenthesizedExpression parses "(" expression ")" and returns the inner
// expression, used where the grammar spells a parenthesised operand out rather
// than reaching it through primary-expression.
func (p *parser) parseParenthesizedExpression(context string) ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open "+context)
	e := p.parseExpression()
	p.expect(scanlex.CLOSE_PAREN, "to close "+context)
	return e
}

// parseExpressionList parses the expression-list production:
//
//	expression-list = expression, { ",", expression }
//
// Implements: expression-list
func (p *parser) parseExpressionList() []ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	list := []ast.Expr{p.parseExpression()}
	for p.accept(scanlex.COMMA) {
		list = append(list, p.parseExpression())
	}
	return list
}
