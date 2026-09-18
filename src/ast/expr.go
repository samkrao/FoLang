package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// LiteralExpr is a compile-time literal value. Its concrete value is resolved
// by later semantic passes.
type LiteralExpr struct {
	Span
	Value any
}

func (LiteralExpr) NodeKind() string { return "LiteralExpr" }
func (LiteralExpr) expr()            {}

// TypeValueExpr lifts a type into a value expression. It is used when FoLang
// treats a type as a runtime or compile-time value, such as box(PolyId) or an
// assignment to a type-valued binding. Ordinary type annotations remain Type.
type TypeValueExpr struct {
	Span
	Type Type
}

func (TypeValueExpr) NodeKind() string { return "TypeValueExpr" }
func (TypeValueExpr) expr()            {}

// BindingExpr refers to a special FoLang binding. Index -1 represents the
// recursive binding '$'; positive indexes represent chained-call results such
// as '$1' and '$2'. Index zero is invalid and should be rejected by validation.
type BindingExpr struct {
	Span
	Symbol symboltable.SymbolID
	Index  int
}

func (BindingExpr) NodeKind() string { return "BindingExpr" }
func (BindingExpr) expr()            {}

// VariableAccessExpr reads a value from the symbol identified by Symbol.
type VariableAccessExpr struct {
	Span
	Symbol symboltable.SymbolID
}

func (VariableAccessExpr) NodeKind() string { return "VariableAccessExpr" }
func (VariableAccessExpr) expr()            {}

// MemberAccessExpr reads a member from Receiver.
type MemberAccessExpr struct {
	Span
	Receiver Expr
	Member   symboltable.SymbolID
}

func (MemberAccessExpr) NodeKind() string { return "MemberAccessExpr" }
func (MemberAccessExpr) expr()            {}

// IndexExpr indexes Receiver with Index.
type IndexExpr struct {
	Span
	Receiver Expr
	Index    Expr
}

func (IndexExpr) NodeKind() string { return "IndexExpr" }
func (IndexExpr) expr()            {}

// CallExpr invokes Function with Arguments. Receiver is set for a method call.
// Body carries a direct block argument such as the body passed to loop or then.
type CallExpr struct {
	Span
	Receiver  Expr
	Function  symboltable.SymbolID
	Arguments []Expr
	Body      Block
}

func (CallExpr) NodeKind() string { return "CallExpr" }
func (CallExpr) expr()            {}

// UnaryExpr applies Operator to Operand.
type UnaryExpr struct {
	Span
	Operator string
	Operand  Expr
}

func (UnaryExpr) NodeKind() string { return "UnaryExpr" }
func (UnaryExpr) expr()            {}

// BinaryExpr applies Operator to Left and Right.
type BinaryExpr struct {
	Span
	Left     Expr
	Operator string
	Right    Expr
}

func (BinaryExpr) NodeKind() string { return "BinaryExpr" }
func (BinaryExpr) expr()            {}

// GroupExpr preserves explicit source grouping.
type GroupExpr struct {
	Span
	Expression Expr
}

func (GroupExpr) NodeKind() string { return "GroupExpr" }
func (GroupExpr) expr()            {}

// TupleExpr constructs a tuple value.
type TupleExpr struct {
	Span
	Elements []Expr
}

func (TupleExpr) NodeKind() string { return "TupleExpr" }
func (TupleExpr) expr()            {}

// CollectionExpr constructs an array, list, or map value.
type CollectionExpr struct {
	Span
	Elements []Expr
}

func (CollectionExpr) NodeKind() string { return "CollectionExpr" }
func (CollectionExpr) expr()            {}

// MatchExpr evaluates the subject and selects the first matching case.
type MatchExpr struct {
	Span
	Subject Expr
	Cases   []MatchCase
}

func (MatchExpr) NodeKind() string { return "MatchExpr" }
func (MatchExpr) expr()            {}
