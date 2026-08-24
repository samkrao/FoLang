package scanlex

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/helpers"
)

// Contains reports whether the target TokenKind is present in the given slice.
func Contains(nums []TokenKind, target TokenKind) bool {
	for _, num := range nums {
		if num == target {
			return true
		}
	}
	return false
}

// WhoCalledMe was an unused debugging helper that printed to stdout. It is
// removed rather than gated: nothing called it, and a library that a language
// server embeds must not hold a stdout writer at all — anything on that stream
// corrupts a JSON-RPC transport.

// errorObj builds a scanner diagnostic positioned at the current token.
//
// DummyNode is returned by currentToken when no token has been pushed yet, and its
// positions are helpers.NilPosition, which is a nil pointer. Dereferencing those crashed
// the scanner on any diagnostic raised at the FIRST token of a file — the error path
// failed instead of reporting the error. The span is resolved defensively so a
// diagnostic always has somewhere to point.
func (lex *lexer) errorObj(expectedKind *Token, str string) helpers.ErrorInterface {
	token := lex.currentToken()
	start, end := lex.diagnosticSpan(token)

	if expectedKind != nil {
		err := fmt.Sprintf("Expected %s but recieved %s instead\n", TokenKindString(expectedKind.Kind), TokenKindString(token.Kind))
		return helpers.NewExpectedTokenError(start, end, err)
	}
	err := fmt.Sprintf("Found %s Token\n", TokenKindString(token.Kind))
	return helpers.NewExpectedTokenErrorName(start, end, str, err)
}

// diagnosticSpan returns a usable start and end position for a token, falling back to
// the scanner's current source position when the token carries none.
func (lex *lexer) diagnosticSpan(token Token) (helpers.Position, helpers.Position) {
	here := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)

	start := here
	if token.StartPos != nil {
		start = token.StartPos
	}
	end := here
	if token.EndPos != nil {
		end = token.EndPos
	}
	return *start, *end
}

// currentLineText returns the source text of the line the scanner is on, or "" when the
// line is out of range.
func (lex *lexer) currentLineText() string {
	if lex.line-1 < 0 || lex.line-1 >= len(lex.sourcearr) {
		return ""
	}
	return lex.sourcearr[lex.line-1]
}

func (lex *lexer) errorException(str string, errType helpers.ErrorType, startPos helpers.Position, endPos helpers.Position) helpers.ErrorInterface {
	switch errType {
	case helpers.IllegalChar:
		return helpers.NewIllegalCharError(startPos, endPos, str)
	case helpers.IllegalString:
		return helpers.NewIllegalStringException(startPos, endPos, str)
	case helpers.AlreadyDeclared:
		return helpers.NewAlreadyDeclaredException(startPos, endPos, str)
	case helpers.ExpectedChar:
		return helpers.NewExpectedCharError(startPos, endPos, str)
	case helpers.ExpectedToken:
		return helpers.NewExpectedTokenError(startPos, endPos, str)
	case helpers.InvalidSyntax:
		return helpers.NewInvalidSyntaxError(startPos, endPos, str)
	case helpers.IllegalVariableAssignment:
		return helpers.NewIllegalVariableAssignmentException(startPos, endPos, str)
	case helpers.VariableNotDecl:
		return helpers.NewVariableNotDeclared(startPos, endPos, str)
	case helpers.RuntTime:
		return helpers.NewRTError(startPos, endPos, str, nil)
	case helpers.UnSupported:
		return helpers.NewUnSupportedException(startPos, endPos, str)
	case helpers.NotFound:
		return helpers.NewNotFoundException(startPos, endPos, str)
	case helpers.ReservedKeyword:
		return helpers.NewReservedKeywordException(startPos, endPos, str)
	}
	return helpers.NewError(startPos, endPos, "Unknown Error", str)

}
