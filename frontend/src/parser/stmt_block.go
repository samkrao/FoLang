package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// block, block-statement and labeled-block — section 10.
//
//	block                 = "{", { block-item }, [ block-tail-expression ], "}"
//	block-item            = use-directive | statement
//	block-tail-expression = expression
//	block-statement       = block, body-closure-guard
//	labeled-block         = identifier, ":", block, body-closure-guard
//
// A block has a value: it may end in one unterminated tail expression, and that
// expression is the block's value rather than a statement. The distinction is made by
// the terminator — a trailing ";" makes the last item a statement, its absence makes it
// the tail expression.
//
// use-directive is the ONE file directive a block admits. An instance or extension is
// activated for the scope it is written in, so `@co.ddap.use(from=…)` inside a function
// body scopes the activation to that body; the import, alias and dynamic-runtime
// directives remain file-level and are rejected by parseStatement.
//
// A bare block is a statement in its own right and takes no trailing semicolon, which
// body-closure-guard enforces.

// parseBlock parses the block production as a scope of its own.
//
// A brace that is not a literal expression opens a new context (scope.go), and a
// bare, labeled or argument block is its own scope, so the context is opened here.
// A block that is the BODY of a construct which already owns a scope goes through
// parseScopeBlock instead.
//
// Implements: block
func (p *parser) parseBlock(context string) ast.Stmt {
	defer p.pushContext(symboltable.S_BlockSymbol)()
	return p.parseScopeBlock(context)
}

// parseScopeBlock parses the block production INTO the context already opened by
// the caller.
//
// A function, lambda or pattern clause owns a scope that begins at its parameter
// list rather than at its brace, so its body must not nest a second context inside
// it: docs/language-ref.md B.1 shows a function's body as the function's own
// context, holding its parameters and its locals together.
//
// Statement-level error recovery lives here: each item is parsed inside a recovery
// point, so one malformed statement costs that statement and the rest of the block
// still parses and still reports.
//
// Implements: block
func (p *parser) parseScopeBlock(context string) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	p.expect(scanlex.OPEN_CURLY, "to open "+context)

	var body []ast.Stmt

	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		// An empty statement contributes no AST node, but Appendix B.9 still
		// classifies it as an intervening context-level statement.
		if p.accept(scanlex.SEMI_COLON) {
			p.noteExecutableItem()
			continue
		}

		startPos := p.pos

		// A use directive is no longer a block-item. Activation is file-scoped and
		// `@co.ddap.use` is a directive like any other, so it is written in the
		// source file's metadata region; one inside a block falls through to
		// parseStatement, which reports the placement error.

		// A tail expression is the last thing in the block and carries no ";".
		// It is only tried once the cursor is at something that starts an
		// expression, and it is accepted only if it runs right up to the closing
		// brace.
		if tail, isTail := p.tryBlockTailExpression(); isTail {
			body = append(body, tail)
			break
		}

		var stmt ast.Stmt
		ok := p.recoverItem(startPos, syncStatement, func() {
			stmt = p.parseStatement()
		})
		if ok && stmt != nil {
			body = append(body, stmt)
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close "+context)

	return &ast.BlockStmt{Span: p.spanFrom(spanStart), Body: body,
		Symb: p.blockSymbol("block", false),
	}
}

// tryBlockTailExpression attempts to read the block-tail-expression production.
//
// The tail is only a tail if it is followed immediately by the block's closing
// brace: anything else means the construct was a statement whose terminator is still
// to come. The attempt is speculative, so a candidate that turns out to be a
// statement is rewound with no diagnostic left behind.
//
// Implements: block-tail-expression
func (p *parser) tryBlockTailExpression() (ast.Stmt, bool) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	if !p.startsExpression() {
		return nil, false
	}

	// A declaration or a control chain must not be mistaken for a tail
	// expression, so the shapes that can only be statements are excluded first.
	if p.startsDeclarationOrStatementOnlyForm() {
		return nil, false
	}

	var tail ast.Stmt
	matched := p.speculate(func() bool {
		spanStart := p.pos
		expr := p.parseExpression()
		if !p.at(scanlex.CLOSE_CURLY) {
			return false
		}
		tail = ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: expr,
			Symb: p.stmtSymbol("block-tail-expression"),
		}
		return true
	})

	return tail, matched
}

// startsDeclarationOrStatementOnlyForm reports whether the cursor begins something
// that can only be a statement, never a tail expression.
//
// Without this test a declaration such as `x co.lang.int = 1` with a missing
// semicolon would be silently re-read as the expression `x` applied to a type, and
// the real error — the missing ";" — would be reported somewhere unhelpful.
func (p *parser) startsDeclarationOrStatementOnlyForm() bool {
	switch {
	case p.atKeyword("let"):
		return true
	case p.at(scanlex.BUIL_IN_STMT_EXPRS) && isControlStatementBuiltin(p.lexeme()):
		return true
	case p.atAnnotation():
		return true
	}
	// A typed or inferred declarator: `name type` or `name :=` / `name ?=`.
	if p.atIdentifier() {
		switch p.peek(1).Kind {
		case scanlex.WALRUS, scanlex.QEQ, scanlex.BUILT_IN_TYPE, scanlex.BUILT_IN_KIND:
			return true
		}
	}
	return false
}

// isControlStatementBuiltin reports whether a folded built-in names a control
// statement, which is a statement and never an expression.
func isControlStatementBuiltin(lexeme string) bool {
	switch lexeme {
	case "this.return", "this.break", "this.continue",
		"self.return", "self.break", "self.continue":
		return true
	}
	return false
}

// parseBlockStatement parses the block-statement production:
//
//	block-statement = block, body-closure-guard
//
// A bare block is a statement (DECISION-SYN-005) and takes no trailing semicolon,
// which the guard enforces.
//
// Implements: block-statement
func (p *parser) parseBlockStatement() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	block := p.parseBlock("a block statement")
	p.bodyClosureGuard("a block statement")
	return block
}

// labeled-block and labeled-loop-statement — section 10.
//
//	labeled-block          = label-declaration, ":", block, body-closure-guard
//	labeled-loop-statement = label-declaration, ":", expression-statement,
//	                         labeled-loop-statement-guard
//
// Revision 24 moved labels off the ordinary identifier spelling and onto the
// apostrophe-prefixed label-identifier, so a label is now lexically distinct from
// every other construct that can follow a name with ":" — a map entry, a
// match-case guard, an argument name. That is why neither predicate below needs
// the multi-token lookahead the old identifier form required:
//
//	'outer: { … }                       labeled-block
//	'outer: (condition).loop({ … });     labeled-loop-statement
//
// The two are separated by what follows the ":", which is the only thing that
// differs between them.

// parseLabeledStatement parses whichever of labeled-block and
// labeled-loop-statement the cursor begins.
//
// Implements: labeled-block
// Implements: labeled-loop-statement
func (p *parser) parseLabeledStatement() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	label := p.parseLabelIdentifier("as a control label")
	p.expect(scanlex.COLON, "after a control label")

	// labeled-block.
	if p.at(scanlex.OPEN_CURLY) {
		block := p.parseBlock("a labeled block")
		p.bodyClosureGuard("a labeled block")

		// The label also names the block symbol, which is what an enclosed
		// `this.break 'outer;` resolves against.
		if b, ok := block.(*ast.BlockStmt); ok {
			b.Symb = p.blockSymbol(label.Scanned, true)
		}
		return ast.LabeledStmt{Span: p.spanFrom(spanStart), Label: label.Scanned,
			Body: block,
			Symb: p.stmtSymbol("labeled-block"),
		}
	}

	// labeled-loop-statement. The labeled statement is parsed as the ordinary
	// expression statement it is, then checked against
	// labeled-loop-statement-guard.
	body := p.parseExpressionStatement(annotationSet{})
	isLoop := p.labeledLoopStatementGuard(label, body)

	return ast.LabeledStmt{Span: p.spanFrom(spanStart), Label: label.Scanned,
		Body:   body,
		IsLoop: isLoop,
		Symb:   p.stmtSymbol("labeled-loop-statement"),
	}
}

// labeledLoopStatementGuard applies the labeled-loop-statement-guard:
//
//	? the labeled expression-statement is a current-profile loop form whose outer
//	  control operation is .loop(...); labeling an arbitrary expression statement
//	  does not turn it into a loop ?
//
// The guard is what keeps `'outer: doSomething();` from becoming a labeled loop
// and so a legal `this.continue 'outer;` target. It is checked on the OUTER
// control operation rather than anywhere in the chain, because `(a).loop({…}).then(…)`
// ends in a conditional, not in a loop.
//
// Implements: labeled-loop-statement-guard
func (p *parser) labeledLoopStatementGuard(label name, statement ast.Stmt) bool {
	expression, ok := statement.(ast.ExpressionStmt)
	if !ok {
		p.reportf(p.cur(), "the label %s must precede a block or a loop statement", label.Logical)
		return false
	}
	if !isLoopChainExpression(expression.Expression) {
		p.reportf(p.cur(), "the label %s precedes a statement that is not a loop; a label may name a block or a %q chain, and labeling an ordinary statement does not make it one", label.Logical, ".loop")
		return false
	}
	return true
}

// atLabeledStatement reports whether the cursor begins a labeled-block or a
// labeled-loop-statement.
//
// A label-identifier has no other use in the statement grammar, so its presence
// alone decides; the ":" is still required and is reported by the parse rather
// than silently declining the dispatch here.
func (p *parser) atLabeledStatement() bool {
	return p.atLabelIdentifier()
}
