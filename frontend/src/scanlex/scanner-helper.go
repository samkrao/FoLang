package scanlex

import (
	"fmt"
	"runtime"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
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
// WhoCalledMe prints the caller's file, line, and function name to stdout for debugging.
func WhoCalledMe() {
	// 0 = this function, 1 = its caller
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		fmt.Println("Could not get caller info")
		return
	}
	fn := runtime.FuncForPC(pc)
	fmt.Printf("Called from: %s:%d (%s)\n", file, line, fn.Name())
}
func (lex *lexer) errorObj(expectedKind *Token, str string) helpers.ErrorInterface {
	err := ""
	token := lex.currentToken()
	fmt.Println(expectedKind)
	if expectedKind != nil {
		expKind := *expectedKind
		err = fmt.Sprintf("Expected %s but recieved %s instead\n", TokenKindString(expKind.Kind), TokenKindString(lex.currentToken().Kind))
		return helpers.NewExpectedTokenError(*token.StartPos, *token.EndPos, err)
	} else {

		err = fmt.Sprintf("Found %s Token\n", TokenKindString(lex.currentToken().Kind))
		return helpers.NewExpectedTokenErrorName(*token.StartPos, *token.EndPos, str, err)

	}
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
	default:
		fmt.Println("Unknow " + str)
	}
	return helpers.NewError(startPos, endPos, "Unknown Error", str)

}
