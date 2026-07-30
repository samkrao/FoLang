package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// comprehension-expression — section 11.
//
//	comprehension-expression = "for", "(", comprehension-binding, ")",
//	                           ".yield", "(", expression-list, ")"
//	comprehension-binding    = pattern, "<-", expression
//
// A comprehension draws values from a generator and projects them
// (docs/language-ref.md, "Comprehensions"):
//
//	result := for (x <- List(1,2,3)).yield(x * 2)          // List(2, 4, 6)
//	result := for (x <- Some(5)).yield(x * 2)              // Some(10)
//	upper  := for ((name, age) <- ages).yield(name.toUpperCase, age)
//
// The binding is a full pattern, not just a name, which is what makes the tuple
// destructuring in the last example work. The result's container follows the
// generator's, and that is a semantic rule rather than a syntactic one.

// parseComprehensionExpression parses the comprehension-expression production.
func (p *parser) parseComprehensionExpression() ast.Expr {
	p.expectKeyword("for", "to begin a comprehension")

	p.expect(scanlex.OPEN_PAREN, "to open a comprehension binding")
	bindings, source := p.parseComprehensionBinding()
	p.expect(scanlex.CLOSE_PAREN, "to close a comprehension binding")

	// ".yield" is spelled as a member access.
	p.expect(scanlex.DOT, "before \"yield\" in a comprehension")
	p.expectMemberName("yield", "to introduce the projection of a comprehension")

	p.expect(scanlex.OPEN_PAREN, "to open a comprehension projection")
	projection := p.parseExpressionList()
	p.expect(scanlex.CLOSE_PAREN, "to close a comprehension projection")

	// Several projected expressions form one tuple-valued yield.
	var yield ast.Expr
	if len(projection) == 1 {
		yield = projection[0]
	} else {
		yield = p.foldComma(projection)
	}

	return ast.ForComprehensionExpr{
		Bindings: bindings,
		Source:   source,
		Yield:    yield,
		Symb:     p.comprehensionSymbol("for"),
	}
}

// parseComprehensionBinding parses the comprehension-binding production:
//
//	comprehension-binding = pattern, "<-", expression
//
// The pattern's bound names are collected into ForBinding entries, because that is
// what the AST node carries: a comprehension needs the names it introduces, and the
// pattern's structure is only used to derive them.
func (p *parser) parseComprehensionBinding() ([]ast.ForBinding, ast.Expr) {
	pat := p.parsePattern()

	p.expect(scanlex.LEFT_ARROW, "between a comprehension binding and its generator")
	source := p.parseExpression()

	names := p.bindingNamesOf(pat)
	bindings := make([]ast.ForBinding, 0, len(names))
	for _, n := range names {
		bindings = append(bindings, ast.ForBinding{
			Name: n,
			Symb: p.exprSymbol(n),
		})
	}

	return bindings, source
}

// bindingNamesOf collects the names a pattern binds, in source order.
//
// Only the binding forms contribute: a binding-pattern contributes its own name, a
// tuple or constructor pattern contributes its sub-patterns' names, and a literal or
// wildcard contributes nothing.
func (p *parser) bindingNamesOf(pat pattern) []string {
	var names []string

	switch pat.Form {
	case patternBinding:
		names = append(names, pat.Name)
	case patternTuple, patternConstructor, patternRecord:
		for _, sub := range pat.Elements {
			names = append(names, p.bindingNamesOf(sub)...)
		}
	}
	return names
}
