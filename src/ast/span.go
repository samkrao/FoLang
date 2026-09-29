package ast

import "github.com/samkrao/fo-lang/src/helpers"

// Position is the compact source coordinate serialized with AST nodes. The
// source filename and text are owned once by SourceFile, not repeated at every
// node boundary.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Span is a half-open source region: Start is included and End is excluded.
type Span struct {
	Start Position
	End   Position
}

func (s Span) GetSpan() Span { return s }

func (s Span) IsZero() bool {
	return s.Start == (Position{}) && s.End == (Position{})
}

func (s Span) Contains(line, column int) bool {
	if line < s.Start.Line || line > s.End.Line {
		return false
	}
	if line == s.Start.Line && column < s.Start.Column {
		return false
	}
	if line == s.End.Line && column >= s.End.Column {
		return false
	}
	return true
}

type Spanned interface {
	GetSpan() Span
}

// NewSpan converts scanner positions without retaining their filename or full
// source-text fields on every AST node.
func NewSpan(start, end helpers.Position) Span {
	return Span{
		Start: Position{Offset: start.Idx, Line: start.Ln, Column: start.Col},
		End:   Position{Offset: end.Idx, Line: end.Ln, Column: end.Col},
	}
}
