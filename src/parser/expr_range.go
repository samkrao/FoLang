package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// range-expression — section 11 of docs/grammar/folang.ebnf.
//
//	range-expression = additive-expression,
//	                   [ range-operator, [ additive-expression ] ]
//	                 | range-operator, additive-expression
//	range-operator   = ".." | "<.." | "..<" | "<..<"
//
// Both bounds are optional and each operator independently decides whether its end
// is inclusive (docs/language-ref.md, "Range Declaration"):
//
//	rangeI := 1 .. 10;      [1, 10]    both inclusive
//	rangeS := 0 <.. 5;      (0, 5]     lower excluded
//	rangeL := 0 ..< 100;    [0, 100)   upper excluded
//	rangeB := 0 <..< 100;   (0, 100)   both excluded
//	rangeE := .. 100;       (_, 100]   open lower bound
//	rangeF := 1 ..;         [1, _)     open upper bound
//
// A range contains at most one range operator, so the operators are
// non-associative in the precedence table and `1 .. 5 .. 9` is a diagnostic rather
// than a nested range.

// finishRange builds the range-expression production after the operator has been
// consumed by the Pratt loop.
//
// lower is the left operand, which is nil only for the prefix form. The upper bound
// is absent when the operator is followed by something that cannot begin an
// expression, which is what allows `1 ..` to stand as an open-ended range.
//
// Implements: range-expression
func (p *parser) finishRange(lower ast.Expr, opTok scanlex.Token, op infixOp) ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	spanStart := p.pos
	excludeStart, excludeEnd := rangeBounds(opTok.Value)
	p.requireOperatorBoundaryBefore(opTok, "range operator")
	if _, alreadyRange := lower.(ast.RangeExpr); alreadyRange {
		p.reportf(opTok, "a range expression may contain at most one range operator; %q follows an existing range", opTok.Value)
	}

	var upper ast.Expr
	if p.startsExpression() {
		p.requireOperatorBoundaryAfter(opTok, "range operator")
		// The right operand is parsed above the range level so that a second
		// range operator is not absorbed here; the non-associativity check below
		// then reports it.
		upper = p.parseExpr(bpRange + 1)
	}

	return ast.RangeExpr{NodeName: "RangeExpr", Span: p.spanFrom(spanStart), Lower: lower,
		Upper:        upper,
		ExcludeStart: excludeStart,
		ExcludeEnd:   excludeEnd,
		Symb:         p.exprSymbol(opTok.Value),
	}
}

// parsePrefixRange parses the second alternative of range-expression, where the
// range operator opens the expression and the lower bound is absent:
//
//	range-expression = range-operator, additive-expression
//
// This is the `.. 100` form, an open lower bound.
func (p *parser) parsePrefixRange() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	opTok := p.advance()
	p.requireOperatorBoundaryAfter(opTok, "open-lower range operator")
	excludeStart, excludeEnd := rangeBounds(opTok.Value)

	upper := p.parseExpr(bpRange + 1)

	return ast.RangeExpr{NodeName: "RangeExpr", Span: p.spanFrom(spanStart), Lower: nil,
		Upper:        upper,
		ExcludeStart: excludeStart,
		ExcludeEnd:   excludeEnd,
		Symb:         p.exprSymbol(opTok.Value),
	}
}

// rangeBounds decodes which ends a range operator excludes. A leading "<" excludes
// the lower bound and a trailing "<" excludes the upper one.
func rangeBounds(lexeme string) (excludeStart bool, excludeEnd bool) {
	switch lexeme {
	case "..":
		return false, false
	case "<..":
		return true, false
	case "..<":
		return false, true
	case "<..<":
		return true, true
	}
	return false, false
}

// startsExpression reports whether the token at the cursor could begin an
// expression.
//
// It is the test for every optional-expression position in the grammar: a range's
// upper bound, a return statement's value list, an index, and a match subject.
// Only tokens that can open a primary-expression or a prefix operator qualify.
func (p *parser) startsExpression() bool {
	switch p.kind() {
	case scanlex.NUMBER, scanlex.STRING, scanlex.CHAR, scanlex.BOOL,
		scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER,
		scanlex.BUILT_IN_TYPE, scanlex.BUILT_IN_KIND,
		scanlex.BUILT_IN_CONSTANTS, scanlex.BUIL_IN_STMT_EXPRS,
		scanlex.BIND_VAR, scanlex.DISCARD_WILD_VAR,
		scanlex.OPEN_PAREN, scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY, scanlex.PIPE,
		scanlex.SPECIAL_METHODS:
		return true
	case scanlex.KEYWORD, scanlex.RESERVEDWORD, scanlex.CONTEXT_KEYWORD:
		switch p.lexeme() {
		case "this", "let", "for", "forall":
			return true
		}
		return false
	}
	// A prefix operator can also open an expression, but a reserved spelling
	// cannot.
	return p.isPrefixOperator() && !p.atReservedOperator()
}
