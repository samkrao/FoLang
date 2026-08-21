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
//
// co.lang.block is the one kind this guard must NOT claim. DECISION-DECL-003
// makes a named block a statement, so parseStatement dispatches the
// identifier-headed spelling before reaching here; what still arrives is the "_"
// head, which dispatchKindDeclaration rejects with the naming rule.

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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseLocalDeclarationName()
	generics := p.parseOptionalGenericParameterClause()
	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a local declaration's kind")

	return p.dispatchKindDeclaration(declName, generics, kindTok, annotations)
}

// parseLocalDeclarationName reads the name of an already-diagnosed nested
// declaration. Both spellings are accepted here on purpose: this is a recovery
// path, and the declaration has been rejected for its position rather than for
// how it names itself, so re-reporting the head would bury the real diagnostic.
func (p *parser) parseLocalDeclarationName() name {
	if p.at(scanlex.DISCARD_WILD_VAR) {
		return nameFrom(p.advance())
	}
	return p.parseIdentifier("as a local declaration name")
}

// Physical nesting inside a DECLARATION body — docs/language-ref.md, "Struct
// Declaration Relationships", "Class Declaration Relationships" and "Invalid
// Physical Nesting".
//
// atLocalKindDeclaration above answers the same question for an executable
// block. The containers below need it too, and for the same reason: the
// reference gives physical nesting a rule of its own and names the replacement
// for it, so a member that breaks that rule should be told which rule it broke.
//
// Without this the nested declaration fell through to the container's ordinary
// member grammar and was reported by whatever failed first, which named
// something the author had not written:
//
//	_ co.lang.struct = { Address co.lang.struct = { … } }
//	    -> "a struct field cannot have a default value"
//	_ co.lang.class  = { Address co.lang.struct = { … } }
//	    -> "expected \";\" after a field declaration, found \"}\""
//
// It applies to every container whose member grammar admits NO kind-introduced
// declaration at all, which is all of them but three:
//
//	struct   cstruct  union     enum       class
//	trait    mixin    interface typeclass  object
//	matcher  instance extension                     -> guarded
//
//	unit     signature module                       -> exempt
//
// The three exemptions are the only member grammars that name a built-in kind:
// `unit-member` admits data-declaration, type-declaration,
// function-object-declaration and delegate-declaration; `signature-member` and
// `module-member` admit signature-type-component and the associated-type forms.
// Guarding those bodies would reject the reference's own examples.
//
// Everything else is guarded, `instance-body` included — it is
// `{ function-declaration | variable-declaration }`, and a variable declarator's
// type is a type-expression, which atLocalKindDeclaration already separates from
// a kind token through isTypeFirstKind. `x co.lang.int = 1;` is therefore
// untouched while `Inner co.lang.struct = { … }` is not.
//
// What the guard claims is a nested DEFINITION, never a forward declaration that
// shares its shape. `Dept co.lang.struct;` is the extern form the reference
// defines as a legal class and unit member, and it stays legal here; only a
// binding makes the declaration a physically nested one.

// rejectNestedKindDeclaration reports a named kind-introduced declaration written
// directly inside a declaration body that cannot hold one.
//
// container names the enclosing body for the diagnostic. The check is a pure
// lookahead: it consumes nothing, so a body that does not begin a nested
// declaration proceeds to its ordinary member grammar untouched.
//
// It looks PAST any annotations first. Every declaration production begins
// `annotations, ...`, and a nested declaration is annotated more often than not —
// `@co.dap.local` is the very thing the diagnostic recommends, so a reader who
// half-remembers the rule writes it in the wrong place. Testing at the raw cursor
// saw the "@" instead of the name and declined, which sent exactly that reader
// back to the unrelated member-grammar error this guard exists to replace.
func (p *parser) rejectNestedKindDeclaration(container string) {
	if !p.atNestedKindDefinition() {
		return
	}
	p.failf(p.cur(), "a named %s declaration cannot be physically nested in %s; declare it in its own package source file and restrict it to this declaration with %s",
		p.nestedKindName(), container, "@co.dap.local")
}

// atNestedKindDefinition reports whether the cursor begins a nested kind
// DEFINITION, as opposed to a forward declaration of the same shape.
//
// The two are told apart by the binding, which is the only thing that separates
// them:
//
//	Dept co.lang.struct;          forward/extern declaration — a legal member
//	Dept co.lang.struct = { … }   a definition — physically nested, forbidden
//
// The reference gives the first its own section and states that
// "@co.dap.declare is optional" for functions and types, so the annotation cannot
// be the discriminator and the ANNOTATED spelling is not the only one to admit.
// The absence of a body is what makes a forward declaration one, and the
// nesting rule is about definitions: a forward declaration introduces no nested
// body and no nested scope, which is exactly what "physical nesting" means.
func (p *parser) atNestedKindDefinition() bool {
	return p.lookaheadOnly(func() bool {
		p.skipAnnotationApplications()
		if !p.atLocalKindDeclaration() {
			return false
		}
		p.advance() // the name
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		p.advance() // the kind token
		// kind-options may precede the binding: `co.lang.module->( … ) = { … }`.
		if p.at(scanlex.ARROW) {
			p.advance()
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			}
		}
		return p.atOp("=")
	})
}

// nestedKindName returns the built-in kind spelling the nested declaration uses,
// so the diagnostic can name what was written rather than describing it.
//
// It skips the same annotations the guard does, so an annotated declaration is
// named by its kind rather than falling back to the generic "kind".
func (p *parser) nestedKindName() string {
	kind := ""
	p.lookaheadOnly(func() bool {
		p.skipAnnotationApplications()
		p.advance() // the name
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if p.at(scanlex.BUILT_IN_KIND) {
			kind = p.lexeme()
		}
		return false
	})
	if kind == "" {
		return "kind"
	}
	return kind
}

// skipAnnotationApplications advances the cursor past a run of metadata
// applications, `"@" qualified-name [ "(" … ")" ]` each.
//
// It is lookahead machinery rather than a parse: no node is built and no
// diagnostic is raised, so it must only ever be called inside lookaheadOnly or
// speculate.
func (p *parser) skipAnnotationApplications() {
	for p.atAnnotation() {
		p.advance()
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
	}
}
