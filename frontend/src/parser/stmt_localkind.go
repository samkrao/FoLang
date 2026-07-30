package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Local kind declarations — a declaration inside a block whose kind is a built-in kind token
// rather than a type.
//
// The statement production of section 10 lists variable-declaration,
// inferred-variable-declaration, local-function-declaration and closure-declaration, but a
// block may also declare something by KIND. The reference documents local and nested
// declarations directly (docs/language-ref.md, "Local and/or Nested types and functions"), and
// the corpus uses the shape for a local lambda:
//
//	add (x co.lang.int)->(co.lang.int)={
//	    z co.lang.lambda = (y co.lang.int)->(co.lang.int)=>x + y;
//	    this.return z(5);
//	}
//
// DECISION-KIND-001 governs the interaction with variable-declaration: in statement position
// variable-declaration is preferred, so this production must only claim a declaration that
// actually carries a kind token. That is exactly what atLocalKindDeclaration tests, which is
// why an ordinary declarator such as `z co.lang.int = 1;` is unaffected — co.lang.int is a
// built-in TYPE, not a kind.

// atLocalKindDeclaration reports whether the cursor begins a declaration whose kind is a
// built-in kind token.
//
// The shape is `name [ generic-clause ] built-in-kind`, so the probe skips the name and any
// parenthesised clause and requires a BUILT_IN_KIND token to follow.
func (p *parser) atLocalKindDeclaration() bool {
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		return p.at(scanlex.BUILT_IN_KIND)
	})
}

// parseLocalKindDeclaration parses a kind-introduced declaration in statement position.
//
// It reads the same prefix the primary-declaration dispatcher does — name, optional generic
// clause, kind token — and then hands off to the shared dispatcher, so a local declaration is
// parsed by exactly the same production as the file-level one.
func (p *parser) parseLocalKindDeclaration(annotations annotationSet) ast.Stmt {
	declName := p.parseDeclarationName("as a local declaration name")
	generics := p.parseOptionalGenericParameterClause()
	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a local declaration's kind")

	return p.dispatchKindDeclaration(declName, generics, kindTok, annotations)
}
