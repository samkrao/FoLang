package helpers

import (
	"fmt"
	"strconv"
)

// Position represents a source location with index, line, column, and file information.
type Position struct {
	Idx  int
	Ln   int
	Col  int
	Fn   string
	Ftxt string
	Pos  int
}

// NilPosition is a nil-valued Position used as a sentinel for missing locations.
var NilPosition *Position = NewPosition(0, 0, 0, 0, "", "", true)

// NewPosition creates a new Position, or returns nil if returnnil is true.
func NewPosition(idx, ln, col, pos int, fn, ftxt string, returnnil bool) *Position {
	if returnnil {
		return nil
	}
	return &Position{Idx: idx, Ln: ln, Col: col, Pos: pos, Fn: fn, Ftxt: ftxt}
}

// Advance moves the position forward by one character, updating line and column on newlines.
func (p *Position) Advance(currentChar rune) *Position {
	p.Idx++
	p.Col++
	p.Pos++

	if currentChar == '\n' {
		p.Ln++
		p.Col = 0
	}

	return p
}

// Retreat moves the position backward by one character, updating line and column on newlines.
func (p *Position) Retreat(currentChar rune) *Position {
	p.Idx--
	p.Col--
	p.Pos--

	if currentChar == '\n' {
		p.Ln--
		p.Col = 0
	}

	return p
}

// NextPos returns a new Position one character ahead of the current position.
func (p *Position) NextPos() *Position {
	return NewPosition(p.Idx+1, p.Ln, p.Col+1, p.Pos+1, p.Fn, p.Ftxt, false)
}

// Copy returns a deep copy of the Position.
func (p *Position) Copy() *Position {
	return NewPosition(p.Idx, p.Ln, p.Col, p.Pos, p.Fn, p.Ftxt, false)
}
// Print writes the position fields to standard output.
func (p *Position) Print() {
	if p != nil {
		fmt.Print(strconv.Itoa(p.Idx) + " :: " + strconv.Itoa(p.Ln) + " :: " + strconv.Itoa(p.Col) + " :: " + strconv.Itoa(p.Pos) + " :: ")
	}
}
