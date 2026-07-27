package helpers

import (
	"errors"
	"fmt"
)

// ErrorType enumerates the categories of compiler errors.
type ErrorType int

const (
	InvalidSyntax ErrorType = iota
	IllegalChar
	ExpectedChar
	ExpectedToken
	IllegalString
	IllegalVariableAssignment
	AlreadyDeclared
	NotFound
	VariableNotDecl
	UnSupported
	RuntTime
	ReservedKeyword
)

// ErrorInterface defines the contract for all compiler error types.
type ErrorInterface interface {
	AsString() string
	error
}

// Error_ is the base error type holding source positions and error details.
type Error_ struct {
	posStart  Position
	posEnd    Position
	errorName string
	details   string
}

// IsKind reports whether err1 matches the target ErrorInterface value.
func IsKind(err1 error, err2 ErrorInterface) bool {
	return errors.Is(err1, err2)
}

// NewError creates a new base Error_ with the given positions, name, and details.
func NewError(posStart, posEnd Position, errorName, details string) *Error_ {
	return &Error_{posStart: posStart, posEnd: posEnd, errorName: errorName, details: details}
}

// AsString returns a formatted string representation of the error with source location.
func (e *Error_) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.errorName, e.details)
	result += fmt.Sprintf("File %s, line %d", e.posStart.Fn, e.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.posStart.Ftxt, e.posStart, e.posEnd)
	return result
}

// Error implements the error interface for Error_.
func (err Error_) Error() string {

	return err.AsString()

}

// String returns the string representation of the error.
func (e *Error_) String() string {
	return e.AsString()
}

// IllegalCharError represents an illegal character encountered during lexing.
type IllegalCharError struct {
	*Error_
}

// AsString returns a formatted string representation of the illegal character error.
func (e *IllegalCharError) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf(" %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln+1)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for IllegalCharError.
func (err IllegalCharError) Error() string {

	return err.AsString()

}

// NewIllegalCharError creates a new IllegalCharError with the given positions and details.
func NewIllegalCharError(posStart, posEnd Position, details string) *IllegalCharError {
	return &IllegalCharError{NewError(posStart, posEnd, "Illegal Character", details)}
}

// ExpectedCharError represents a missing expected character.
type ExpectedCharError struct {
	*Error_
}

// AsString returns a formatted string representation of the expected character error.
func (e *ExpectedCharError) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for ExpectedCharError.
func (err ExpectedCharError) Error() string {

	return err.AsString()

}

// NewExpectedCharError creates a new ExpectedCharError with the given positions and details.
func NewExpectedCharError(posStart, posEnd Position, details string) *ExpectedCharError {
	return &ExpectedCharError{NewError(posStart, posEnd, "Expected Character", details)}
}

// ExpectedTokenError represents a missing or invalid token.
type ExpectedTokenError struct {
	*Error_
}

// AsString returns a formatted string representation of the expected token error.
func (e *ExpectedTokenError) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for ExpectedTokenError.
func (err ExpectedTokenError) Error() string {

	return err.AsString()

}

// NewExpectedTokenError creates a new ExpectedTokenError with the given positions and details.
func NewExpectedTokenError(posStart, posEnd Position, details string) *ExpectedTokenError {
	return &ExpectedTokenError{NewError(posStart, posEnd, "Invalid Token", details)}
}

// NewExpectedTokenErrorName creates a new ExpectedTokenError with a custom error name.
func NewExpectedTokenErrorName(posStart, posEnd Position, errorname string, details string) *ExpectedTokenError {
	return &ExpectedTokenError{NewError(posStart, posEnd, errorname, details)}
}

// InvalidSyntaxError represents a syntax error in the source code.
type InvalidSyntaxError struct {
	*Error_
}

// AsString returns a formatted string representation of the invalid syntax error.
func (e *InvalidSyntaxError) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for InvalidSyntaxError.
func (err InvalidSyntaxError) Error() string {

	return err.AsString()

}

// NewInvalidSyntaxError creates a new InvalidSyntaxError with the given positions and details.
func NewInvalidSyntaxError(posStart, posEnd Position, details string) *InvalidSyntaxError {
	return &InvalidSyntaxError{NewError(posStart, posEnd, "Invalid Syntax", details)}
}

// IllegalStringException represents an invalid string literal.
type IllegalStringException struct {
	*Error_
}

// AsString returns a formatted string representation of the illegal string error.
func (e *IllegalStringException) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf(" %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln+1)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for IllegalStringException.
func (err IllegalStringException) Error() string {

	return err.AsString()

}

// NewIllegalStringException creates a new IllegalStringException with the given positions and details.
func NewIllegalStringException(posStart, posEnd Position, details string) *IllegalStringException {
	return &IllegalStringException{NewError(posStart, posEnd, "Illegal String", details)}
}

// IllegalVariableAssignmentException represents an invalid variable assignment.
type IllegalVariableAssignmentException struct {
	*Error_
}

// AsString returns a formatted string representation of the illegal variable assignment error.
func (e *IllegalVariableAssignmentException) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for IllegalVariableAssignmentException.
func (err IllegalVariableAssignmentException) Error() string {

	return err.AsString()

}

// NewIllegalVariableAssignmentException creates a new IllegalVariableAssignmentException.
func NewIllegalVariableAssignmentException(posStart, posEnd Position, details string) *IllegalVariableAssignmentException {
	return &IllegalVariableAssignmentException{NewError(posStart, posEnd, "Illegal Variable Assignment", details)}
}

// AlreadyDeclaredException represents a duplicate declaration error.
type AlreadyDeclaredException struct {
	*Error_
}

// AsString returns a formatted string representation of the already declared error.
func (e *AlreadyDeclaredException) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for AlreadyDeclaredException.
func (err AlreadyDeclaredException) Error() string {

	return err.AsString()

}

// NewAlreadyDeclaredException creates a new AlreadyDeclaredException with the given positions and details.
func NewAlreadyDeclaredException(posStart, posEnd Position, details string) *AlreadyDeclaredException {
	return &AlreadyDeclaredException{NewError(posStart, posEnd, "Already Declared", details)}
}

// UnSupportedException represents usage of an unsupported language feature.
type UnSupportedException struct {
	*Error_
}

// AsString returns a formatted string representation of the unsupported feature error.
func (e *UnSupportedException) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for UnSupportedException.
func (err UnSupportedException) Error() string {

	return err.AsString()

}

// NewUnSupportedException creates a new UnSupportedException with the given positions and details.
func NewUnSupportedException(posStart, posEnd Position, details string) *UnSupportedException {
	return &UnSupportedException{NewError(posStart, posEnd, "UnSupported Reservedword", details)}
}

// NotFoundException represents a symbol or entity that could not be found.
type NotFoundException struct {
	*Error_
}

// AsString returns a formatted string representation of the not found error.
func (e *NotFoundException) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for NotFoundException.
func (err NotFoundException) Error() string {

	return err.AsString()

}

// NewNotFoundException creates a new NotFoundException with the given positions and details.
func NewNotFoundException(posStart, posEnd Position, details string) *NotFoundException {
	return &NotFoundException{NewError(posStart, posEnd, "NotFound", details)}
}

// ReservedKeywordException represents usage of a reserved keyword as an identifier.
type ReservedKeywordException struct {
	*Error_
}

// AsString returns a formatted string representation of the reserved keyword error.
func (e *ReservedKeywordException) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for ReservedKeywordException.
func (err ReservedKeywordException) Error() string {

	return err.AsString()

}

// VariableNotDeclared represents a reference to an undeclared variable.
type VariableNotDeclared struct {
	*Error_
}

// AsString returns a formatted string representation of the variable not declared error.
func (e *VariableNotDeclared) AsString() string {
	result := fmt.Sprintf("%s: %s\n", e.Error_.errorName, e.Error_.details)
	result += fmt.Sprintf("File %s, line %d", e.Error_.posStart.Fn, e.Error_.posStart.Ln)
	result += "\n\n" + stringWithArrows(e.Error_.posStart.Ftxt, e.Error_.posStart, e.Error_.posEnd)
	return result
}

// Error implements the error interface for VariableNotDeclared.
func (err VariableNotDeclared) Error() string {

	return err.AsString()

}

// NewVariableNotDeclared creates a new VariableNotDeclared error with the given positions and details.
func NewVariableNotDeclared(posStart, posEnd Position, details string) *VariableNotDeclared {
	return &VariableNotDeclared{NewError(posStart, posEnd, "Reserved word", details)}
}

// NewReservedKeywordException creates a new ReservedKeywordException with the given positions and details.
func NewReservedKeywordException(posStart, posEnd Position, details string) *ReservedKeywordException {
	return &ReservedKeywordException{NewError(posStart, posEnd, "Reserved word", details)}
}

// RTError represents a runtime error with an associated execution context.
type RTError struct {
	*Error_
	context *Context
}

// Error implements the error interface for RTError.
func (err RTError) Error() string {

	return err.AsString()

}

// NewRTError creates a new RTError with the given positions, details, and execution context.
func NewRTError(posStart, posEnd Position, details string, context *Context) *RTError {
	return &RTError{NewError(posStart, posEnd, "Runtime Error", details), context}
}

// AsString returns a formatted string representation of the runtime error with traceback.
func (e *RTError) AsString() string {
	result := e.GenerateTraceback()
	result += fmt.Sprintf("%s: %s", e.errorName, e.details)
	result += "\n\n" + stringWithArrows(e.posStart.Ftxt, e.posStart, e.posEnd)
	return result
}

// GenerateTraceback builds a traceback string by walking the execution context chain.
func (e *RTError) GenerateTraceback() string {
	result := ""
	pos := e.posStart
	ctx := e.context

	for ctx != nil {
		result = fmt.Sprintf("  File %s, line %d, in %s\n", pos.Fn, pos.Ln, ctx.displayName) + result
		pos = ctx.parentEntryPos
		ctx = ctx.parent
	}

	return "Traceback (most recent call last):\n" + result
}

// Placeholder for the Context and stringWithArrows function
type Context struct {
	displayName    string
	parent         *Context
	parentEntryPos Position
}
