package parser

import (
	"github.com/samkrao/fo-lang/src/scanlex"
)

// Contextual guards and speculative parsing.
//
// The grammar resolves its two hardest ambiguities with zero-width contextual
// guards rather than with ordered choice alone (DECISION-SYN-007). Both are
// implemented here, together with the bounded speculation they need.
//
// The ambiguities are:
//
//   - Is a "}" the end of a declaration BODY or the end of a braced EXPRESSION?
//     A body takes no trailing ";"; an expression leaves its enclosing statement
//     still needing one (DECISION-SYN-006, the expression-brace rule).
//
//         Employee co.lang.struct = { id co.lang.int; }     body, no ";"
//         emp := Employee{ id: 1 };                         expression, needs ";"
//
//   - Is a braced or parenthesised span a direct body or an expression that
//     merely begins the same way?
//
//         classify(n) => { this.return "positive"; }         direct block body
//         someFArg co.lang.function = (a co.lang.int)->(co.lang.int) = { … }
//                                                            direct anon-fn body
//         oObj co.lang.function = add;                       expression binding
//
// The guards below decide these by looking past the balanced group and asking
// whether an expression continuation follows it. That is exactly the property
// the EBNF describes: a grouped block, a block with a postfix suffix, or an
// operator expression containing a block is still an expression, while a bare
// braced group in a body position is a body.

// speculate runs try with the cursor and diagnostic list snapshotted.
//
// If try returns true the parse is kept and the cursor stays where try left it.
// If try returns false, or aborts with a bailout, everything is rolled back —
// cursor and diagnostics both — as if try had never run.
//
// Rolling back diagnostics is what makes speculation usable for disambiguation:
// a failed attempt must not leave a complaint behind about a reading the parser
// then abandoned. The scope model is rolled back for the same reason: a block the
// abandoned reading entered would otherwise be recorded twice, once per reading.
func (p *parser) speculate(try func() bool) (ok bool) {
	startPos := p.pos
	startDiags := len(p.diags)
	startScope := scopeFrame{ctx: p.ctx, symtab: p.symtab, sawExecutable: p.sawExecutable}
	scopeMark := len(p.scopeJournal)
	p.speculating++

	defer func() {
		p.speculating--
		r := recover()
		if r != nil {
			if _, isBailout := r.(bailout); !isBailout {
				panic(r)
			}
			ok = false
		}
		switch {
		case !ok:
			p.pos = startPos
			p.diags = p.diags[:startDiags]
			p.rollbackScopes(scopeMark)
			p.ctx, p.symtab, p.sawExecutable = startScope.ctx, startScope.symtab, startScope.sawExecutable
		case p.speculating == 0:
			// The kept parse is final, so its inverses are unreachable. An enclosing
			// speculation would still need them, which is why they are only discarded
			// at the outermost level.
			p.scopeJournal = p.scopeJournal[:scopeMark]
		}
	}()

	return try()
}

// lookaheadOnly runs probe with the cursor snapshotted and always rewinds,
// returning whatever probe reported. Use it for pure lookahead questions such as
// "does a built-in kind token follow this declarator?", where no node is built
// and nothing should be kept.
func (p *parser) lookaheadOnly(probe func() bool) bool {
	startPos := p.pos
	startDiags := len(p.diags)
	startScope := scopeFrame{ctx: p.ctx, symtab: p.symtab, sawExecutable: p.sawExecutable}
	scopeMark := len(p.scopeJournal)
	p.speculating++
	defer func() {
		p.speculating--
		p.pos = startPos
		p.diags = p.diags[:startDiags]
		p.rollbackScopes(scopeMark)
		p.ctx, p.symtab, p.sawExecutable = startScope.ctx, startScope.symtab, startScope.sawExecutable
		if r := recover(); r != nil {
			if _, isBailout := r.(bailout); !isBailout {
				panic(r)
			}
		}
	}()
	return probe()
}

// bodyClosureGuard implements the body-closure-guard production: a zero-width
// assertion that the next significant token is not ";", or that there is no next
// token.
//
// It is applied immediately after a body-selecting "}" — a declaration body, a
// function body, a function-pattern body or a standalone block statement — and
// consumes nothing. A ";" there means the author terminated a body as if it were
// a simple statement, which DECISION-SYN-006 forbids, so it is reported and
// consumed to keep the enclosing list in step.
func (p *parser) bodyClosureGuard(context string) {
	if !p.at(scanlex.SEMI_COLON) {
		return
	}
	p.reportf(p.cur(), "unexpected %q after the closing %q of %s; a body ends at its brace and takes no terminator", ";", "}", context)
	p.advance()
}

// startsDirectBody reports whether the cursor begins a direct block body rather
// than a braced expression.
//
// This is the affirmative form of the non-block-expression guard. The cursor must
// be on "{", and the balanced group it opens must not be followed by anything
// that would continue an expression. When both readings are possible — the empty
// "{}" being the canonical case — the block reading wins, as DECISION-SYN-007
// requires.
func (p *parser) startsDirectBody() bool {
	// Bare braced values and postfix operations on bare blocks are intentionally
	// absent from the grammar. Supporting `{ ... }.member` or `{ ... }(args)`
	// would make this position ambiguous again and require looking beyond the
	// matching brace. Typed object/collection expressions are unaffected because
	// their explicit type prefix is consumed before their brace.
	return p.at(scanlex.OPEN_CURLY)
}

// startsAnonymousFunction reports whether the cursor begins an
// anonymous-function-expression used as a direct inline body.
//
// The empty spelling commits on `() ->`; a non-empty spelling commits on its
// first typed parameter. The selected production parses the complete signature.
func (p *parser) startsAnonymousFunction() bool {
	if !p.at(scanlex.OPEN_PAREN) {
		return false
	}
	if p.peek(1).Kind == scanlex.CLOSE_PAREN {
		return p.peek(2).Kind == scanlex.ARROW
	}
	// `(name Type` commits to an anonymous function. We do not scan to its
	// closing parenthesis or body: the selected production consumes those once
	// and reports any later error. Anonymous functions cannot own forall binders.
	return p.startsTypedParameterPrefix(1)
}

// There is no looksLikeMapLiteral guard, because an unprefixed "{" never opens a
// map literal. map-literal is not a primary-expression alternative — a braced
// map body is a collection BODY reachable only behind a type prefix. A bare
// braced group in expression position is rejected; it is not a block value.
//
// The guards that DO remain are the ones separating a braced body from a typed
// braced construction, where a type prefix has already been read:
// looksLikeObjectConstruction and looksLikeObjectFieldInitializers below.

// looksLikeObjectConstruction reports whether the cursor begins an
// object-construction expression, `type-postfix-expression "{" … "}"`.
//
// The distinguishing shape is a name — possibly dotted, possibly with type
// arguments — immediately followed by "{" whose contents are either empty or
// `identifier ":"` field initialisers. Requiring the identifier-colon shape is
// what keeps `x.match { … }`-style chains and bare blocks from being captured
// here.
func (p *parser) looksLikeObjectConstruction() bool {
	if !p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER, scanlex.BUILT_IN_TYPE) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance()
		// Optional type-argument lists: Vector(co.lang.int){ … }
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.OPEN_CURLY) {
			return false
		}
		p.advance()
		if p.at(scanlex.CLOSE_CURLY) {
			return true // Employee{} is a well-formed empty construction.
		}
		return p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) &&
			p.peek(1).Kind == scanlex.COLON
	})
}

// rejectEqualsObjectFieldBinder reports a braced construction whose fields bind
// with "=" instead of ":".
//
// The reference states the binder outright — "Object field initializers use `:`
// between the field name and value and `,` between fields. `=` is not an
// object-field initializer binder" (docs/language-ref.md, "Canonical Object and
// Collection Construction") — but looksLikeObjectConstruction simply declines the
// shape, so `Employee{id = 1}` fell through to the type-as-value reading and was
// reported as a missing ";" before a block. That names neither the construct the
// author wrote nor the rule it breaks.
//
// It runs only after the construction guard has declined, so a well-formed
// construction never reaches it, and a bare block never does either: the shape
// required here begins with a type name.
func (p *parser) rejectEqualsObjectFieldBinder() {
	if !p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER, scanlex.BUILT_IN_TYPE) {
		return
	}
	found := p.lookaheadOnly(func() bool {
		p.advance()
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.OPEN_CURLY) {
			return false
		}
		p.advance()
		return p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) &&
			p.peek(1).Value == "="
	})
	if found {
		p.failf(p.cur(), "an object field initializer binds its value with \":\", as in %q; \"=\" is not an object-field initializer binder", "Employee{id: 1}")
	}
}
