package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Nested kind-declaration recovery recognizes a declaration inside a block whose
// kind is a built-in kind token rather than a type.
//
// DECISION-SYN-008 and the reference's "Local and/or Nested types and
// functions" section forbid physically nested independent named types and
// containers. Only an ordinary named local function and anonymous expressions
// are exceptions. The parser still recognizes this prefix so statement.go can
// issue one precise diagnostic and consume the declaration for recovery.
//
// DECISION-KIND-001 still governs the interaction with variable-declaration:
// the predicate claims only a name followed by a built-in KIND, so an ordinary
// declarator such as `z co.lang.int = 1;` remains unaffected.

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
		hasGenerics := false
		if p.at(scanlex.OPEN_PAREN) {
			hasGenerics = p.looksLikeGenericParameterClause()
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.BUILT_IN_KIND) {
			return false
		}
		// `x co.lang.value` and the other overlapping names are variable
		// declarations in an executable block. An explicit generic clause cannot
		// belong to a variable declarator, so that shape remains a forbidden nested
		// kind declaration and receives the dedicated diagnostic.
		return hasGenerics || !isTypeFirstKind(p.lexeme())
	})
}

// parseLocalKindDeclaration consumes a forbidden kind-introduced declaration in
// statement position after the caller has diagnosed its physical nesting.
//
// It reads the same prefix the primary-declaration dispatcher does — name,
// optional generic clause, kind token — and hands off to the shared dispatcher
// only to retain useful recovery and follow-on diagnostics.
func (p *parser) parseLocalKindDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseDeclarationName("as a local declaration name")
	generics := p.parseOptionalGenericParameterClause()
	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a local declaration's kind")

	return p.dispatchKindDeclaration(declName, generics, kindTok, annotations)
}
