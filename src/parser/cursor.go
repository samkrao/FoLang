package parser

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/scanlex"
)

// This file is the parser's only window onto the token stream. Every other file
// navigates tokens through these helpers, which keeps two invariants in one
// place:
//
//  1. Reading past the end of the stream yields the EOF token rather than
//     panicking, so no caller needs a bounds check.
//  2. The cursor position is a single integer, which is what makes the cheap
//     speculative parsing in guards.go possible.

// eofToken is returned for any read beyond the end of the stream. The scanner
// always appends a real EOF token, so this is a defensive fallback only.
var eofToken = scanlex.Token{Kind: scanlex.EOF, Value: "EOF"}

// cur returns the token at the cursor without consuming it.
func (p *parser) cur() scanlex.Token {
	if p.pos >= len(p.toks) {
		return eofToken
	}
	return p.toks[p.pos]
}

// peek returns the token n positions ahead of the cursor. peek(0) is cur.
func (p *parser) peek(n int) scanlex.Token {
	i := p.pos + n
	if i < 0 || i >= len(p.toks) {
		return eofToken
	}
	return p.toks[i]
}

// matchingParenOffset returns the offset of the close parenthesis matching the
// open parenthesis at openOffset. It is a read-only token-index probe: it never
// changes p.pos, creates parser state, or requires rollback.
func (p *parser) matchingParenOffset(openOffset int) (int, bool) {
	if p.peek(openOffset).Kind != scanlex.OPEN_PAREN {
		return 0, false
	}
	depth := 0
	for i := openOffset; ; i++ {
		switch p.peek(i).Kind {
		case scanlex.EOF:
			return 0, false
		case scanlex.OPEN_PAREN:
			depth++
		case scanlex.CLOSE_PAREN:
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
}

func (p *parser) tokenAfterMatchingParen(openOffset int) (scanlex.Token, bool) {
	closeOffset, ok := p.matchingParenOffset(openOffset)
	if !ok {
		return eofToken, false
	}
	return p.peek(closeOffset + 1), true
}

// kind returns the kind of the token at the cursor.
func (p *parser) kind() scanlex.TokenKind { return p.cur().Kind }

// lexeme returns the exact source text of the token at the cursor. Operator
// dispatch is keyed by lexeme rather than by token kind, because the scanner
// reuses a single kind for several spellings (see tokenstream.go).
func (p *parser) lexeme() string { return p.cur().Value }

// atEOF reports whether the cursor has reached the end of the stream.
func (p *parser) atEOF() bool { return p.cur().Kind == scanlex.EOF }

// advance consumes and returns the token at the cursor.
func (p *parser) advance() scanlex.Token {
	t := p.cur()
	if p.pos < len(p.toks) {
		p.pos++
	}
	return t
}

// at reports whether the token at the cursor has the given kind.
func (p *parser) at(k scanlex.TokenKind) bool { return p.cur().Kind == k }

// atAny reports whether the token at the cursor has any of the given kinds.
func (p *parser) atAny(kinds ...scanlex.TokenKind) bool {
	return p.cur().IsOneOfMany(kinds...)
}

// atDeclarationKindToken bridges the scanner's historical type/kind buckets.
// The language reference now classifies several type-producing declaration
// markers as types, so declaration dispatch must use the known lexeme rather
// than require every marker to carry BUILT_IN_KIND.
func (p *parser) atDeclarationKindToken() bool {
	// Exact declaration-marker spelling is authoritative. The scanner's broad
	// built-in categories are an implementation detail and some historical names
	// (notably co.lang.kind) do not currently land in the type bucket.
	if _, ok := typeDeclarationKinds[p.lexeme()]; ok {
		return true
	}
	if unitMemberKinds[p.lexeme()] {
		return true
	}
	if p.at(scanlex.BUILT_IN_KIND) {
		return true
	}
	return false
}

func (p *parser) expectDeclarationKind(context string) scanlex.Token {
	if p.atDeclarationKindToken() {
		return p.advance()
	}
	p.failExpected(p.cur(), fmt.Sprintf("expected a built-in declaration kind %s, found %s", context, describeToken(p.cur())))
	return eofToken
}

// atOp reports whether the token at the cursor is the operator spelled lex.
// Structural tokens are matched with at/atAny; operators are matched here so
// that fused spellings such as "**" and "=>>" are recognised by text.
func (p *parser) atOp(lex string) bool { return p.cur().Value == lex }

// atAnyOp reports whether the cursor holds any of the given operator spellings.
func (p *parser) atAnyOp(lexemes ...string) bool {
	v := p.cur().Value
	for _, l := range lexemes {
		if v == l {
			return true
		}
	}
	return false
}

// accept consumes the token at the cursor if it has the given kind and reports
// whether it did.
func (p *parser) accept(k scanlex.TokenKind) bool {
	if p.at(k) {
		p.pos++
		return true
	}
	return false
}

// acceptOp consumes the token at the cursor if it is the operator spelled lex
// and reports whether it did.
func (p *parser) acceptOp(lex string) bool {
	if p.atOp(lex) {
		p.pos++
		return true
	}
	return false
}

// expect consumes a token of the given kind, or reports a diagnostic naming
// what the enclosing production wanted and aborts the current parse.
//
// context should name the construct being parsed, so the message reads
// "expected ')' to close a parameter list" rather than just "expected ')'".
func (p *parser) expect(k scanlex.TokenKind, context string) scanlex.Token {
	if p.at(k) {
		return p.advance()
	}
	p.failExpected(p.cur(), fmt.Sprintf("expected %s %s, found %s", describeKind(k), context, describeToken(p.cur())))
	return eofToken // unreachable: failf panics
}

// expectOp consumes the operator spelled lex, or reports a diagnostic and
// aborts the current parse.
func (p *parser) expectOp(lex string, context string) scanlex.Token {
	if p.atOp(lex) {
		return p.advance()
	}
	p.failExpected(p.cur(), fmt.Sprintf("expected %q %s, found %s", lex, context, describeToken(p.cur())))
	return eofToken // unreachable: failf panics
}

// statementEnd consumes the mandatory ";" that closes a simple statement.
//
// DECISION-SYN-001: a semicolon is mandatory after every production that uses
// statement-end. Newlines are ordinary whitespace and never terminate a
// statement, and there is no semicolon insertion.
func (p *parser) statementEnd(context string) {
	if p.accept(scanlex.SEMI_COLON) {
		return
	}
	// An unterminated statement at end of file is worth a clearer message than
	// the generic "expected ;", because it is usually a missing terminator on
	// the last line rather than a stray token.
	if p.atEOF() {
		p.failExpected(p.cur(), fmt.Sprintf("missing %q at end of %s; FoLang never terminates a statement with a newline", ";", context))
	}
	p.failExpected(p.cur(), fmt.Sprintf("expected %q after %s, found %s", ";", context, describeToken(p.cur())))
}

// mark captures the cursor so that a tentative parse can be rewound. Pair every
// mark with either reset or a commit; see guards.go for the wrappers that make
// this safe.
func (p *parser) mark() int { return p.pos }

// reset rewinds the cursor to a position previously returned by mark.
func (p *parser) reset(m int) { p.pos = m }

// skipTo advances until the cursor is at one of the given kinds or at EOF. It is
// used by error recovery, never by a successful parse.
func (p *parser) skipTo(kinds ...scanlex.TokenKind) {
	for !p.atEOF() && !p.atAny(kinds...) {
		p.pos++
	}
}

// enter increments the recursion guard and returns the matching exit function.
// Recursive productions call it as `defer p.enter()()` so that pathological
// input becomes a diagnostic instead of a stack overflow.
func (p *parser) enter() func() {
	p.depth++
	if p.depth > maxRecursionDepth {
		p.failf(p.cur(), "expression or type nests too deeply (limit %d); check for unbalanced brackets", maxRecursionDepth)
	}
	return func() { p.depth-- }
}
