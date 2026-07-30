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
func (p *parser) synchronize(startPos int, sync []scanlex.TokenKind) {
	// Force progress when the failure happened on the very first token.
	if p.pos == startPos && !p.atEOF() {
		p.pos++
	}
	for !p.atEOF() {
		if p.atAny(sync...) {
			if p.at(scanlex.SEMI_COLON) {
				p.pos++
			}
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
