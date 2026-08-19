package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
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
// then abandoned.
func (p *parser) speculate(try func() bool) (ok bool) {
	startPos := p.pos
	startDiags := len(p.diags)
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
		if !ok {
			p.pos = startPos
			p.diags = p.diags[:startDiags]
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
	p.speculating++
	defer func() {
		p.speculating--
		p.pos = startPos
		p.diags = p.diags[:startDiags]
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
	if !p.at(scanlex.OPEN_CURLY) {
		return false
	}
	return !p.lookaheadOnly(func() bool {
		p.skipBalanced(scanlex.OPEN_CURLY, scanlex.CLOSE_CURLY)
		return p.continuesExpression()
	})
}

// startsAnonymousFunction reports whether the cursor begins an
// anonymous-function-expression used as a direct inline body.
//
// It recognises the shape
//
//	[ "forall" "(" type-parameter-list ")" "." ] "(" … ")" "->" "(" … ")" "{"
//
// by skipping the balanced parameter list and requiring a return-type clause and
// then a brace. The trailing brace is what distinguishes an inline body from a
// bare function type such as `(co.lang.int)->(co.lang.int)`, which is a type
// expression and not an expression at all. No "=" sits between the signature and
// the brace: an anonymous function juxtaposes the two, and the "=" spelling is the
// named function-definition.
func (p *parser) startsAnonymousFunction() bool {
	return p.lookaheadOnly(func() bool {
		if p.atOp("forall") {
			p.advance()
			if !p.at(scanlex.OPEN_PAREN) {
				return false
			}
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			if !p.accept(scanlex.DOT) {
				return false
			}
		}
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		if !p.at(scanlex.ARROW) {
			return false
		}
		p.advance()
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		return p.at(scanlex.OPEN_CURLY)
	})
}

// continuesExpression reports whether the token at the cursor could continue an
// expression whose operand has just been completed.
//
// It covers the postfix suffixes (call, index, member, match) and any infix
// operator known to the built-in precedence table or to the user-defined operator
// registry. It is the test that separates `{ … }` used as a body from `{ … }.f()`
// or `{ … } + x`, which are expressions.
func (p *parser) continuesExpression() bool {
	switch p.kind() {
	case scanlex.OPEN_PAREN:
		// A "(" is normally a call suffix, but a receiver clause also opens with one,
		// and a member that carries a receiver is the next DECLARATION rather than a
		// continuation of the body just closed:
		//
		//	plain()->(S) = { … }
		//	(emp Employee) labelled()->(S) = { … }
		//
		// Without this the "}" above would be read as a braced expression called with
		// "(emp Employee)", and the whole member would misparse. atReceiverClause is
		// the precise test: it requires a name and a parameter list after the group,
		// which no call argument list is followed by.
		return !p.atReceiverClause()
	case scanlex.DOT, scanlex.OPEN_BRACKET:
		return true
	}
	if _, ok := p.infixOperator(); ok {
		return true
	}
	if _, ok := postfixOperators[p.lexeme()]; ok {
		return true
	}
	return false
}

<<<<<<< Updated upstream
// There is no looksLikeMapLiteral guard, because an unprefixed "{" never opens a
// map literal. map-literal is not a primary-expression alternative — a braced
// map body is a collection BODY reachable only behind a type prefix — so every
// braced group in expression position is unambiguously the block reading and
// needs no lookahead to establish it (docs/grammar/folang.ebnf,
// primary-expression and non-block-expression-guard).
//
// The guards that DO remain are the ones separating a braced body from a typed
// braced construction, where a type prefix has already been read:
// looksLikeObjectConstruction and looksLikeObjectFieldInitializers below.

=======
>>>>>>> Stashed changes
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
