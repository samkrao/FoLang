package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Error recovery.
//
// A parse function aborts by panicking with bailout{} (see diagnostics.go). This
// file provides the only places that catch it: the loops that read a sequence of
// independent items — statements in a block, members in a declaration body,
// items in an entry file. Recovering there and nowhere else gives a useful
// property: a malformed statement costs exactly that statement, and the rest of
// the file is still parsed and still diagnosed.
//
// Recovery must guarantee forward progress. If the cursor has not moved by the
// time the bailout is caught, skipping is unconditional, otherwise a statement
// that fails on its very first token would spin forever.

// syncStatement is the set of tokens at which statement-level recovery stops.
// A ";" ends the broken statement; the braces are structural boundaries that
// almost certainly belong to an enclosing construct.
var syncStatement = []scanlex.TokenKind{
	scanlex.SEMI_COLON,
	scanlex.CLOSE_CURLY,
	scanlex.OPEN_CURLY,
}

// syncMember is the set of tokens at which member-level recovery stops, used
// inside declaration bodies where members are ";"-terminated.
var syncMember = []scanlex.TokenKind{
	scanlex.SEMI_COLON,
	scanlex.CLOSE_CURLY,
}

// recoverItem runs body and reports whether it completed without aborting.
//
// On a bailout it consumes tokens up to and including the next synchronisation
// point drawn from sync, leaving the cursor ready for the next item. startPos is
// the cursor value from before body ran and is used to force progress when body
// failed without consuming anything.
//
// Any panic that is not a bailout is re-panicked: a nil dereference in the
// parser is a bug, not malformed input, and must not be silently swallowed.
func (p *parser) recoverItem(startPos int, sync []scanlex.TokenKind, body func()) (ok bool) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if _, isBailout := r.(bailout); !isBailout {
			panic(r)
		}
		ok = false
		p.synchronize(startPos, sync)
	}()
	body()
	return true
}

// synchronize skips forward to a point where parsing can plausibly resume.
//
// It stops at the first synchronisation token. A ";" is consumed, because it
// belongs to the statement that failed. A "}" is left in place, because it
// belongs to the enclosing body and that body's own loop must see it to
// terminate.
//
// It ALSO stops at a line boundary, which is what keeps recovery proportionate.
// Terminator-only synchronisation costs far more than the broken construct: a
// garbled line with no ";" of its own runs on to the next terminator, and since
// a brace group is skipped whole on the way, the next well-formed declaration is
// swallowed with it. In
//
//	first()->(co.lang.int) = { this.return 1; }
//	&&& broken &&&
//	second()->(co.lang.int) = { this.return 2; }
//
// the first ";" at this level is inside second's body, so second used to
// disappear entirely and the file reported one diagnostic instead of one bad
// line. That is the ordinary state of a buffer being typed into, so an editor
// consumer lost its outline on almost every keystroke.
//
// Stopping at the next line that begins a new construct costs the broken line
// and nothing more. The cost is a source construct deliberately spread over
// several lines, which may now report more than one diagnostic; that is the
// right trade, and MaxParseErrors bounds it.
func (p *parser) synchronize(startPos int, sync []scanlex.TokenKind) {
	// Force progress when the failure happened on the very first token.
	if p.pos == startPos && !p.atEOF() {
		p.pos++
	}
	resumeLine := p.lineAt(p.pos - 1)

	for !p.atEOF() {
		if p.atAny(sync...) {
			if p.at(scanlex.SEMI_COLON) {
				p.pos++
			}
			return
		}
		if line := p.lineAt(p.pos); line > resumeLine && p.startsRecoveryUnit() {
			return
		}
		// Skipping over a balanced brace group keeps a broken statement that
		// contains a block from stranding the cursor inside that block.
		if p.at(scanlex.OPEN_CURLY) {
			p.skipBalanced(scanlex.OPEN_CURLY, scanlex.CLOSE_CURLY)
			continue
		}
		p.pos++
	}
}

// lineAt returns the source line of the token at index i, or 0 when it has none.
func (p *parser) lineAt(i int) int {
	if i < 0 || i >= len(p.toks) {
		return 0
	}
	if pos := p.toks[i].StartPos; pos != nil {
		return pos.Ln
	}
	return 0
}

// startsRecoveryUnit reports whether the cursor holds a token that could begin a
// new declaration, member or statement.
//
// The set is deliberately permissive — recovery only needs a PLAUSIBLE resume
// point, and a wrong guess costs one extra diagnostic rather than a wrong parse.
// What it excludes is the tokens that can only CONTINUE something: an operator,
// a separator, or a closing delimiter. Resuming on one of those would re-enter
// the middle of the construct that just failed.
func (p *parser) startsRecoveryUnit() bool {
	switch p.kind() {
	case scanlex.COMMA, scanlex.DOT, scanlex.COLON, scanlex.SEMI_COLON,
		scanlex.CLOSE_PAREN, scanlex.CLOSE_CURLY, scanlex.CLOSE_BRACKET,
		scanlex.ARROW, scanlex.EQGT, scanlex.EQGTGT, scanlex.ASSIGNMENT,
		scanlex.PIPE, scanlex.SYMBOLIC_RUN, scanlex.CUSTOM_OPERATOR:
		return false
	}
	return true
}

// skipBalanced consumes a balanced open/close group starting at the cursor,
// which must be positioned on the opening token. Nested groups of the same pair
// are counted, so the cursor lands just past the matching close token.
func (p *parser) skipBalanced(open, close scanlex.TokenKind) {
	if !p.at(open) {
		return
	}
	depth := 0
	for !p.atEOF() {
		switch {
		case p.at(open):
			depth++
		case p.at(close):
			depth--
		}
		p.pos++
		if depth == 0 {
			return
		}
	}
}
