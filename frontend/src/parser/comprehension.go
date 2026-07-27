package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// isForComprehension reports whether the current `for` token begins a
// for-comprehension of the form:
//
//	for (x <- collection).yield(projection)
//	for ((name, age) <- collection).yield(projection)
//
// It scans ahead (up to 30 tokens) inside the opening paren looking for a
// LEFT_ARROW token.  If found before EOF, it's a comprehension.
func isForComprehension(p *parser) bool {
	defer p.traceCurrent()()

	if p.currentToken().Value != "for" {
		return false
	}
	if p.nextTokenSafe(1).Kind != scanlex.OPEN_PAREN {
		return false
	}
	for i := 2; i < 30; i++ {
		tk := p.nextTokenSafe(i)
		if tk.Kind == scanlex.LEFT_ARROW {
			return true
		}
		if tk.Kind == scanlex.EOF {
			return false
		}
	}
	return false
}

// parse_for_comprehension_expr parses a for-comprehension expression:
//
//	for (x <- collection).yield(projection)
//	for ((name, age) <- collection).yield(projection)
//
// The `for` keyword must be the current token when called.
// Returns a ParseResult whose Node is ast.ForComprehensionExpr.
func parse_for_comprehension_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Node: nil, Errors: []helpers.ErrorInterface{}}

	p.advance() // eat `for`

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	// Parse the binding pattern.
	//   Simple:  x <- collection
	//   Tuple:   (name, age) <- collection
	var bindings []ast.ForBinding
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		// Tuple destructuring: (name, age)
		p.advance() // eat (
		for p.hasTokens() &&
			p.currentTokenKind() != scanlex.CLOSE_PAREN &&
			p.currentTokenKind() != scanlex.EOF {
			nameTk, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
			p.addErr(err_)
			fb := ast.ForBinding{Name: nameTk.Value}
			bindings = append(bindings, fb)
			if p.currentTokenKind() == scanlex.COMMA {
				p.advance()
			}
		}
		_, err_ = p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err_)
	} else {
		// Simple binding: x
		nameTk, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
		p.addErr(err_)
		fb := ast.ForBinding{Name: nameTk.Value}
		bindings = append(bindings, fb)
	}

	// Consume `<-`.
	_, err_ = p.expect(scanlex.LEFT_ARROW)
	p.addErr(err_)

	// Parse the source/generator expression (everything up to the closing paren).
	sourcePR := parse_expr(p, assignment, ddaps)
	var sourceExpr ast.Expr
	if sourcePR.Node != nil {
		sourceExpr = sourcePR.Node.(ast.Expr)
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)

	// Parse `.yield(projection)`.
	_, err_ = p.expect(scanlex.DOT)
	p.addErr(err_)

	// `yield` is an identifier; consume it.
	_, err_ = p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
	p.addErr(err_)

	_, err_ = p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	yieldPR := parse_expr(p, defalt_bp, ddaps)
	var yieldExpr ast.Expr
	if yieldPR.Node != nil {
		yieldExpr = yieldPR.Node.(ast.Expr)
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)

	fc := ast.ForComprehensionExpr{
		Bindings: bindings,
		Source:   sourceExpr,
		Yield:    yieldExpr,
		Symb: &symboltable.ForComprehension{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_ForComprehension),
			},
		},
	}
	pr.Node = fc
	return pr
}
