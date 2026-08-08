package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// block, block-statement and labeled-block — section 10.
//
//	block                 = "{", { block-item }, [ block-tail-expression ], "}"
//	block-item            = statement
//	block-tail-expression = expression
//	block-statement       = block, body-closure-guard
//	labeled-block         = identifier, ":", block, body-closure-guard
//
// DECISION-BLK-001 gives a block a value: it may end in one unterminated tail
// expression, and that expression is the block's value rather than a statement. The
// distinction is made by the terminator — a trailing ";" makes the last item a
// statement, its absence makes it the tail expression.
//
// DECISION-SYN-005 makes a bare block a statement in its own right, and
// DECISION-SYN-001 says it takes no trailing semicolon, which body-closure-guard
// enforces.

// parseBlock parses the block production.
//
// Statement-level error recovery lives here: each item is parsed inside a recovery
// point, so one malformed statement costs that statement and the rest of the block
// still parses and still reports.
//
// Implements: block
func (p *parser) parseBlock(context string) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	p.expect(scanlex.OPEN_CURLY, "to open "+context)

	var body []ast.Stmt

	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		// An empty statement is a bare ";" and contributes nothing.
		if p.accept(scanlex.SEMI_COLON) {
			continue
		}

		startPos := p.pos

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
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	block := p.parseBlock("a block statement")
	p.bodyClosureGuard("a block statement")
	return block
}

// parseLabeledBlock parses the labeled-block production:
//
//	labeled-block = identifier, ":", block, body-closure-guard
//
// This is the label form of docs/language-ref.md, "Labels and Named Blocks":
//
//	outer:{
//	    // statements
//	}
//
// Implements: labeled-block
func (p *parser) parseLabeledBlock() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	label := p.parseIdentifier("as a block label")
	p.expect(scanlex.COLON, "after a block label")

	block := p.parseBlock("a labeled block")
	p.bodyClosureGuard("a labeled block")

	// The label names the block, which is what a jump to it resolves against.
	if b, ok := block.(*ast.BlockStmt); ok {
		b.Symb = p.blockSymbol(label.Scanned, true)
	}
	return block
}

// atLabeledBlock reports whether the cursor begins a labeled-block.
//
// The shape is `identifier ":" "{"`, which is checked in full because a bare
// `identifier ":"` also begins a map entry and a match-case guard.
func (p *parser) atLabeledBlock() bool {
	return p.atIdentifier() &&
		p.peek(1).Kind == scanlex.COLON &&
		p.peek(2).Kind == scanlex.OPEN_CURLY
}
