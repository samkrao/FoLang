// Package scanlex provides lexical scanning and tokenization for the fo-lang compiler.
// It defines token types, keywords, built-in symbols, and directive metadata used by the parser.
package scanlex

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/helpers"
)

// Token represents a single lexical token with its kind, string value, and source positions.
type Token struct {
	Kind     TokenKind
	SubKind  SubKind
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

// Println prints the quoted token value, kind, subkind, and position range to
// stdout. Quoting keeps whitespace and newline tokens visible on one line.
func (tk Token) Println() {
	fmt.Printf("%q == kind %s == subkind %s == ", tk.Value, TokenKindString(tk.Kind), SubKindString(tk.SubKind))
	tk.StartPos.Print()
	tk.EndPos.Print()
	fmt.Println()
}

// IsOneOfMany reports whether the token kind matches any of the given expected kinds.
func (tk Token) IsOneOfMany(expectedTokens ...TokenKind) bool {
	for _, expected := range expectedTokens {
		if expected == tk.Kind {
			return true
		}
	}

	return false
}

// DummyNode is a sentinel Token with INVALID kind used as a placeholder.
var DummyNode Token = Token{
	Kind: UNKNOWN, SubKind: NA, Value: "Invalid", StartPos: helpers.NilPosition, EndPos: helpers.NilPosition,
}

// NewUniqueToken creates a new Token with the given kind, value, and position range.
func NewUniqueToken(kind TokenKind, subKind SubKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return newUniqueToken(kind, subKind, value, startPos, endPos)
}
func newUniqueToken(kind TokenKind, subKind SubKind, value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		Kind: kind, SubKind: subKind, Value: value, StartPos: startPos, EndPos: endPos,
	}
}

func newDummyToken(value string, startPos *helpers.Position, endPos *helpers.Position) Token {
	return Token{
		Kind: UNKNOWN, SubKind: NA, Value: value, StartPos: startPos, EndPos: endPos,
	}
}

// Debug prints the token value and kind to stdout for debugging.
func (token Token) Debug() {
	fmt.Printf("%s => (%s)\n", token.Value, TokenKindString(token.Kind))
}

var Special_methods []string = []string{
	"@@new",
	"@@init",
}

// Reserved_lu maps reserved language keywords to their TokenKind.
var Reserved_lu map[string]TokenKind = map[string]TokenKind{
	"co":   KEYWORD, // holds everything
	"this": KEYWORD, // refers this/self
	"fΦλ":  KEYWORD, // fo-lang reserved word
}
