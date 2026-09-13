package ast

import "github.com/samkrao/fo-lang/src/helpers"

// Source spans on AST nodes.
//
// Every node carries the region of source it was parsed from. Diagnostics have
// always had ranges, but until a node has one there is no way to answer the
// question every editor feature starts from: which declaration is the cursor
// in? Without it go-to-definition, hover, document symbols, rename, folding and
// selection ranges are all unimplementable, and only publishDiagnostics and
// lexical highlighting work.
//
// The span is a FIELD rather than an interface method with a setter. Node types
// are used as values throughout the parser and satisfy Stmt/Expr/Type with value
// receivers, so a pointer-receiver setter would stop those values implementing
// the interfaces at all. Populating the field in the composite literal that
// builds the node keeps value semantics and puts the span next to the tokens it
// came from.

// Span is the half-open source region a node was parsed from.
//
// Start is the first character of the node and End is one past its last, which
// is the convention helpers.Position already uses for token spans and the one an
// LSP Range expects.
type Span struct {
	Start helpers.Position
	End   helpers.Position
}

// GetSpan returns the node's source region.
//
// It is promoted onto every node that embeds Span, which is what lets a
// consumer ask any Stmt, Expr or Type where it came from without a type switch
// over 104 node types.
func (s Span) GetSpan() Span { return s }

// IsZero reports whether no span was recorded.
//
// A node built by the parser always has one. Zero means either a synthetic node
// — lowering builds a few that correspond to no source region — or a
// construction site that was missed, which is what TestEveryNodeCarriesASpan
// exists to catch.
func (s Span) IsZero() bool {
	return s.Start.Ln == 0 && s.End.Ln == 0
}

// Contains reports whether the span covers a line/column position.
//
// This is the primitive an editor feature is built from: walking the tree and
// keeping the innermost node whose span contains the cursor yields the node to
// hover, rename or navigate from.
func (s Span) Contains(line, column int) bool {
	if line < s.Start.Ln || line > s.End.Ln {
		return false
	}
	if line == s.Start.Ln && column < s.Start.Col {
		return false
	}
	if line == s.End.Ln && column > s.End.Col {
		return false
	}
	return true
}

// Spanned is implemented by every AST node, through the embedded Span.
type Spanned interface {
	GetSpan() Span
}

// NewSpan builds a span from a start and end position.
func NewSpan(start, end helpers.Position) Span {
	return Span{Start: start, End: end}
}
