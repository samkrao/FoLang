// Package ast defines the high-level intermediate representation (HIR) AST node types
// used by the fo-lang frontend parser.
package ast

import (
	"maps"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// SubType represents the dimension classification of an array type.
type SubType int

const (
	ZEROD SubType = iota
	SINGLED
	MULTID
	JAGGED
	VARIABLELEN
)

// VariableType represents the storage category of a variable.
type VariableType int

const (
	NORMAL VariableType = iota
	ARRAY
	REFERENCE
	DBL_REFERENCE
	POINTER
	ADDRESS
	OBJECT
	Thunk
	LET
)

// SET is the base interface for all AST nodes that can be visited and annotated.
type SET interface {
	SetDap(d map[scanlex.DirectiveKind][]Stmt)
	Visit(s any) SET
	GetName() string
	GetSymbolType() string
}

// Stmt is the interface for all statement AST nodes.
type Stmt interface {
	stmt()
	SET
}

// Expr is the interface for all expression AST nodes.
type Expr interface {
	expr()
	SET
}

// SymbType provides type metadata accessors for symbol table entries.
type SymbType interface {
	GetName() string
	GetActType() (string, string)
	GetSubType() string
}

// Type is the interface for all type AST nodes.
type Type interface {
	_type()
	SET
	SymbType
}

// NonDependentType is a Type that does not depend on a runtime expression.
type NonDependentType interface {
	Type
	isNonDependent() // unexported marker method
}

// ExpectExpr casts an Expr to the specified concrete expression type.
func ExpectExpr[T Expr](expr Expr) T {
	return helpers.ExpectType[T](expr)
}

// ExpectStmt casts a Stmt to the specified concrete statement type.
func ExpectStmt[T Stmt](expr Stmt) T {
	return helpers.ExpectType[T](expr)
}

// DirectveList holds directive annotation mappings for an AST node.
type DirectveList struct {
	Span
	Dapst map[scanlex.DirectiveKind][]Stmt
}

// SetDap attaches directive annotations to the node.
func (b DirectveList) SetDap(d map[scanlex.DirectiveKind][]Stmt) {
	(&b).SetDdap(d)
}

// SetDdap copies directive annotations into this node's map.
func (b *DirectveList) SetDdap(daps map[scanlex.DirectiveKind][]Stmt) {
	b.Dapst = make(map[scanlex.DirectiveKind][]Stmt)
	maps.Copy(b.Dapst, daps)
}

// GetDap returns the directive annotations attached to this node.
func (b *DirectveList) GetDap() map[scanlex.DirectiveKind][]Stmt {
	return b.Dapst
}

func (n DirectveList) stmt()                 {}
func (n DirectveList) GetName() string       { return "" }
func (n DirectveList) GetSymbolType() string { return "" }
