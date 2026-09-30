// Package scanlex provides lexical scanning and tokenization for the fo-lang compiler.
// It defines token types, keywords, built-in symbols, and directive metadata used by the parser.
package scanlex

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/helpers"
)

type TokenState uint8

const (
	TokenPending TokenState = iota
	TokenProcessed
	TokenInvalid
	TokenDiscarded
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
	Next           *Token
	Prev           *Token
	RawPrev        *Token
	RawNext        *Token
	Message        string
	State          TokenState
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

func NewInvalidToken(tok *Token, msg string) *Token {
	if tok == nil {
		return nil
	}
	tok.Message = msg
	tok.State = TokenInvalid
	return tok
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
