package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// grouped-expression and tuple-expression — section 11.
//
//	grouped-expression = "(", expression, ")"
//	tuple-expression   = "(", expression, ",", expression,
//	                     { ",", expression }, ")"
//
// The two differ only by whether a comma appears, so they are parsed as one list
// and told apart by its length. A tuple needs at least two elements, which is what
// keeps `(x)` a grouped expression rather than a one-element tuple.
//
// Grouping is significant beyond precedence: it is the first thing that determines
// expression structure (docs/language-ref.md, "Expression Evaluation Order"), so
// the parentheses are preserved in the AST as ast.GroupingExpr rather than being
// dropped.

// parseGroupedOrTupleExpression parses the grouped-expression and
// tuple-expression productions.
//
// Implements: grouped-expression
// Implements: tuple-expression
func (p *parser) parseGroupedOrTupleExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a parenthesized expression")

	// "()" is not an expression. It only appears as an empty parameter list, and
	// reaching here means one was not expected.
	if p.at(scanlex.CLOSE_PAREN) {
		p.fail(p.cur(), "\"()\" is not an expression; it is only valid as an empty parameter list")
	}

	first := p.parseExpression()

	if !p.at(scanlex.COMMA) {
		p.expect(scanlex.CLOSE_PAREN, "to close a parenthesized expression")
		return ast.GroupingExpr{NodeName: "GroupingExpr", Span: p.spanFrom(spanStart), Expr_: first, Symb: p.exprSymbol("group")}
	}

	elements := []ast.Expr{first}
	for p.accept(scanlex.COMMA) {
		if p.at(scanlex.CLOSE_PAREN) {
			if len(elements) == 1 {
				p.fail(p.cur(), "a tuple expression requires at least two elements; `(x,)` is not a one-element tuple")
			}
			p.fail(p.cur(), "a comma in a tuple expression must be followed by another expression; trailing commas are not allowed")
		}
		elements = append(elements, p.parseExpression())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a tuple expression")

	// A tuple is carried as a comma-expression chain wrapped in a grouping, which
	// is how the AST represents a parenthesized multi-value expression.
	return ast.GroupingExpr{NodeName: "GroupingExpr", Span: p.spanFrom(spanStart), Expr_: p.foldComma(elements),
		Symb: p.exprSymbol("tuple"),
	}
}

// parseBracketedDependentValueList parses a bracketed dependent-type value
// list, such as the dimensions in co.core.Array(2, co.lang.int, [2,4]).
//
// This helper is deliberately not a primary-expression alternative. Runtime
// list and array values use the uniform ConcreteType{...} construction syntax;
// brackets here are metadata carried by a type application.
func (p *parser) parseBracketedDependentValueList() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_BRACKET, "to open a dependent-type value list")

	contents := []ast.Expr{}
	for !p.at(scanlex.CLOSE_BRACKET) && !p.atEOF() {
		contents = append(contents, p.parseExpression())
		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_BRACKET, "to close a dependent-type value list")

	return ast.ArrayLiteral{NodeName: "ArrayLiteral", Span: p.spanFrom(spanStart), Contents: contents, Symb: p.exprSymbol("array")}
}

// parseCompositeConstruction parses the uniform composite-construction production:
//
//	composite-construction = type-postfix-expression, "{",
//	                         [ construction-element,
//	                           { ",", construction-element }, [ "," ] ], "}"
//	construction-element   = expression, [ ":", expression ]
//
// The resolved type determines whether entries are elements, map key/value
// pairs, or object fields. A bare braced group is never a composite value.
//
// Implements: composite-construction
func (p *parser) parseCompositeConstruction() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	typeRef := p.parseTypePostfixExpression()

	p.expect(scanlex.OPEN_CURLY, "to open a composite construction")

	var elements []ast.Expr
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		elementStart := p.pos
		keyOrValue := p.parseExpression()
		element := keyOrValue
		if p.accept(scanlex.COLON) {
			value := p.parseExpression()
			element = ast.AssignmentExpr{NodeName: "AssignmentExpr", Span: p.spanFrom(elementStart), Assigne: keyOrValue,
				AssignedValue: value,
				Symb:          p.exprSymbol("construction-entry"),
			}
		}
		elements = append(elements, element)

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close a composite construction")

	// Construction is modelled as a call on the type, which is what NewExpr wraps.
	return ast.NewExpr{NodeName: "NewExpr", Span: p.spanFrom(spanStart), Instantiation: ast.CallExpr{NodeName: "CallExpr", Span: p.spanFrom(spanStart), Method: ast.SDTExpr{NodeName: "SDTExpr", Span: p.spanFrom(spanStart), Type_: typeRef.fullType(),
		Symb: p.exprSymbol(typeRef.actType()),
	},
		Arguments:   elements,
		SymbolType_: "composite-construction",
		Symb:        p.exprSymbol(typeRef.actType()),
	},
		Symb: p.exprSymbol(typeRef.actType()),
	}
}
