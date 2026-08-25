package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (p *parser) atLockStatement() bool {
	if !p.atIdentifier() || logicalName(p.lexeme()) != "lock" || p.peek(1).Kind != scanlex.OPEN_PAREN {
		return false
	}
	depth := 0
	for offset := 1; ; offset++ {
		tok := p.peek(offset)
		switch tok.Kind {
		case scanlex.OPEN_PAREN:
			depth++
		case scanlex.CLOSE_PAREN:
			depth--
			if depth == 0 {
				return p.peek(offset+1).Kind == scanlex.OPEN_CURLY
			}
		case scanlex.EOF:
			return false
		}
	}
}

// parseLockStatement parses the scoped locking form used for independently
// shared state owned by a named co.lang.object.
//
// Implements: lock-statement
func (p *parser) parseLockStatement() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.advance() // contextual "lock"
	p.expect(scanlex.OPEN_PAREN, "to open a lock target")
	target := p.parseExpression()
	p.expect(scanlex.CLOSE_PAREN, "to close a lock target")
	body := p.parseBlock("a lock body")
	p.bodyClosureGuard("a lock statement")
	return ast.LockStmt{Span: p.spanFrom(spanStart), Target: target, Body: body,
		Symb: p.exprSymbol("lock")}
}
