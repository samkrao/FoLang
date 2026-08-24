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
		return ast.GroupingExpr{Span: p.spanFrom(spanStart), Expr_: first, Symb: p.exprSymbol("group")}
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
	return ast.GroupingExpr{Span: p.spanFrom(spanStart), Expr_: p.foldComma(elements),
		Symb: p.exprSymbol("tuple"),
	}
}

// parseArrayLiteral parses the array-literal production:
//
//	array-literal = "[", [ expression, { ",", expression }, [ "," ] ], "]"
//
// Elements are evaluated left to right in source order (docs/language-ref.md,
// "Collection Literals"), and a trailing comma is permitted.
//
// Implements: array-literal
func (p *parser) parseArrayLiteral() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_BRACKET, "to open an array literal")

	contents := []ast.Expr{}
	for !p.at(scanlex.CLOSE_BRACKET) && !p.atEOF() {
		contents = append(contents, p.parseExpression())
		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_BRACKET, "to close an array literal")

	return ast.ArrayLiteral{Span: p.spanFrom(spanStart), Contents: contents, Symb: p.exprSymbol("array")}
}

// There is no parseMapLiteral, because map-literal is not a primary-expression
// alternative. A braced `{ … }` map body is an object-literal representation, so
// it is a collection BODY and never a value in its own right: the reference states
// plainly that "an untyped `{ ... }` map literal is not a FoLang value"
// (docs/language-ref.md, "Canonical Object and Collection Construction"), and the
// grammar reaches a map body only through typed-collection-literal, behind a type
// prefix. parseCollectionBody in expr_collection.go is where that body is read.
//
// The asymmetry with parseArrayLiteral below is that rule rather than an
// oversight: an array literal is NOT an object literal — C.4.1 makes `[ … ]` a
// simple literal in the same sense as a string or an integer — so it stays
// available untyped and carries no type prefix.
//
// A bare braced group in expression position is therefore always the block
// alternative, which is what parsePrimary does.

// parseObjectConstruction parses the object-construction production:
//
//	object-construction      = type-postfix-expression, "{",
//	                           [ object-field-initializer,
//	                             { ",", object-field-initializer }, [ "," ] ], "}"
//	object-field-initializer = identifier, ":", expression
//
// This is how a value of a user-defined type is written. FoLang has no
// user-defined literal token: DECISION-LIT-004 was withdrawn precisely because
// `Employee{name: "Rao", id: 1}` is an ordinary expression rather than a literal.
//
// Fields use colon and comma (DECISION-COL-001), which is what distinguishes an
// object construction from a block.
//
// Implements: object-construction
func (p *parser) parseObjectConstruction() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	typeRef := p.parseTypePostfixExpression()

	p.expect(scanlex.OPEN_CURLY, "to open an object construction")

	var fields []ast.Expr
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		fieldName := p.parseIdentifier("as an object field name")
		p.expect(scanlex.COLON, "between an object field name and its value")
		value := p.parseExpression()

		fields = append(fields, ast.AssignmentExpr{Span: p.spanFrom(spanStart), Assigne: ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: fieldName.Scanned,
			SymbolType_: "field-name",
			Symb:        p.exprSymbol(fieldName.Scanned),
		},
			AssignedValue: value,
			Symb:          p.exprSymbol(fieldName.Scanned),
		})

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close an object construction")

	// Construction is modelled as a call on the type, which is what NewExpr wraps.
	return ast.NewExpr{Span: p.spanFrom(spanStart), Instantiation: ast.CallExpr{Span: p.spanFrom(spanStart), Method: ast.SDTExpr{Span: p.spanFrom(spanStart), Type_: typeRef.fullType(),
		Symb: p.exprSymbol(typeRef.actType()),
	},
		Arguments:   fields,
		SymbolType_: "object-construction",
		Symb:        p.exprSymbol(typeRef.actType()),
	},
		Symb: p.exprSymbol(typeRef.actType()),
	}
}
