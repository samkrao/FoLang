package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// grouped-expression and tuple-expression — section 11.
//
//	grouped-expression = "(", expression, ")"
//	tuple-expression   = "(", expression, ",", expression,
//	                     { ",", expression }, [ "," ], ")"
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
func (p *parser) parseGroupedOrTupleExpression() ast.Expr {
	p.expect(scanlex.OPEN_PAREN, "to open a parenthesized expression")

	// "()" is not an expression. It only appears as an empty parameter list, and
	// reaching here means one was not expected.
	if p.at(scanlex.CLOSE_PAREN) {
		p.fail(p.cur(), "\"()\" is not an expression; it is only valid as an empty parameter list")
	}

	first := p.parseExpression()

	if !p.at(scanlex.COMMA) {
		p.expect(scanlex.CLOSE_PAREN, "to close a parenthesized expression")
		return ast.GroupingExpr{Expr_: first, Symb: p.exprSymbol("group")}
	}

	elements := []ast.Expr{first}
	for p.accept(scanlex.COMMA) {
		if p.at(scanlex.CLOSE_PAREN) {
			break // trailing comma
		}
		elements = append(elements, p.parseExpression())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a tuple expression")

	// A tuple is carried as a comma-expression chain wrapped in a grouping, which
	// is how the AST represents a parenthesized multi-value expression.
	return ast.GroupingExpr{
		Expr_: p.foldComma(elements),
		Symb:  p.exprSymbol("tuple"),
	}
}

// parseArrayLiteral parses the array-literal production:
//
//	array-literal = "[", [ expression, { ",", expression }, [ "," ] ], "]"
//
// Elements are evaluated left to right in source order (docs/language-ref.md,
// "Collection Literals"), and a trailing comma is permitted.
func (p *parser) parseArrayLiteral() ast.Expr {
	p.expect(scanlex.OPEN_BRACKET, "to open an array literal")

	contents := []ast.Expr{}
	for !p.at(scanlex.CLOSE_BRACKET) && !p.atEOF() {
		contents = append(contents, p.parseExpression())
		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_BRACKET, "to close an array literal")

	return ast.ArrayLiteral{Contents: contents, Symb: p.exprSymbol("array")}
}

// parseMapLiteral parses the map-literal production:
//
//	map-literal = "{", [ map-entry, { ",", map-entry }, [ "," ] ], "}"
//	map-entry   = expression, ":", expression
//
// The caller must have established via looksLikeMapLiteral that this braced group
// is a map and not a block. An empty "{}" is always a block, never an empty map, so
// this function is never entered for one.
//
// A map literal is an EXPRESSION, so its closing brace ends only the expression:
// the enclosing simple statement still needs its ";" (DECISION-SYN-006, the
// expression-brace rule).
//
//	cfg co.lang.map = { "a": 1, "b": 2 };
func (p *parser) parseMapLiteral() ast.Expr {
	p.expect(scanlex.OPEN_CURLY, "to open a map literal")

	entries := []ast.Expr{}
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		key := p.parseExpression()
		p.expect(scanlex.COLON, "between a map key and its value")
		value := p.parseExpression()

		entries = append(entries, ast.CommaExpr{
			Left:  key,
			Right: value,
			Symb:  p.exprSymbol("map-entry"),
		})

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close a map literal")

	// The entry list is carried as an array of key/value pairs, which is the
	// AST's only aggregate expression node.
	return ast.ArrayLiteral{Contents: entries, Symb: p.exprSymbol("map")}
}

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
func (p *parser) parseObjectConstruction() ast.Expr {
	typeRef := p.parseTypePostfixExpression()

	p.expect(scanlex.OPEN_CURLY, "to open an object construction")

	var fields []ast.Expr
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		fieldName := p.parseIdentifier("as an object field name")
		p.expect(scanlex.COLON, "between an object field name and its value")
		value := p.parseExpression()

		fields = append(fields, ast.AssignmentExpr{
			Assigne: ast.SymbolExpr{
				Value:       fieldName.Scanned,
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
	return ast.NewExpr{
		Instantiation: ast.CallExpr{
			Method: ast.SDTExpr{
				Type_: typeRef.Node,
				Symb:  p.exprSymbol(typeRef.actType()),
			},
			Arguments:   fields,
			SymbolType_: "object-construction",
			Symb:        p.exprSymbol(typeRef.actType()),
		},
		Symb: p.exprSymbol(typeRef.actType()),
	}
}
