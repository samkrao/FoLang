// Package scanlex provides lexical scanning and tokenization for the fo-lang compiler.
// It defines token types, keywords, built-in symbols, and directive metadata used by the parser.
package scanlex

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/builtins"
	"github.com/samkrao/fo-lang/src/helpers"
)

// Token represents a single lexical token with its kind, string value, and source positions.
type Token struct {
	Kind     builtins.TokenKind
	Value    string
	StartPos *helpers.Position
	EndPos   *helpers.Position
	// BoundaryBefore and BoundaryAfter retain whether the original source had
	// whitespace, a comment, or a delimiter immediately on that side. The parser
	// uses these flags only when a multi-symbol token is an expression operator;
	// structural uses of the same spelling remain exempt (DECISION-LEX-010).
	BoundaryBefore bool
	BoundaryAfter  bool
}

// Println prints the token value, kind, and position range to stdout.
func (tk Token) Println() {
	fmt.Print(tk.Value + " == " + fmt.Sprint(tk.Kind) + " == ")
	tk.StartPos.Print()
	tk.EndPos.Print()
	fmt.Println("")
}

// IsOneOfMany reports whether the token kind matches any of the given expected kinds.
func (tk Token) IsOneOfMany(expectedTokens ...builtins.TokenKind) bool {
	for _, expected := range expectedTokens {
		if expected == tk.Kind {
			return true
		}
	}

	return false
}

// DummyNode is a sentinel Token with INVALID kind used as a placeholder.
var DummyNode Token = Token{
	Kind: builtins.INVALID, Value: "Invalid", StartPos: helpers.NilPosition, EndPos: helpers.NilPosition,
}

// NewUniqueToken creates a new Token with the given kind, value, and position range.
func NewUniqueToken(kind builtins.TokenKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return newUniqueToken(kind, value, startPos, endPos)
}
func newUniqueToken(kind builtins.TokenKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		Kind: kind, Value: value, StartPos: startPos, EndPos: endPos,
	}
}

func newDummyToken(value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		Kind: builtins.INVALID, Value: value, StartPos: startPos, EndPos: endPos,
	}
}

// Debug prints the token value and kind to stdout for debugging.
func (token Token) Debug() {
	fmt.Printf("%s => (%s)\n", token.Value, builtins.TokenKindString(token.Kind))
}
