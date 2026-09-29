package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// TypeHeader links source type syntax to its resolved semantic type symbol.
type TypeHeader struct {
	Span
	Symbol symboltable.SymbolID
	Kind_  NodeKind
}

// NamedType is a direct type reference.
type NamedType struct {
	TypeHeader
}

func (NamedType) NodeKind() string { return "NamedType" }
func (NamedType) _type()           {}

// TypeApplication applies type or dependent-value arguments to a type.
type TypeApplication struct {
	TypeHeader
	Callee    Type
	TypeArgs  []Type
	ValueArgs []Expr
}

func (TypeApplication) NodeKind() string { return "TypeApplication" }
func (TypeApplication) _type()           {}

// FunctionType preserves callable type syntax while Symbol identifies the
// canonical FunctionType/DelegateType record.
type FunctionType struct {
	TypeHeader
	Parameters []Type
	Results    []Type
}

func (FunctionType) NodeKind() string { return "FunctionType" }
func (FunctionType) _type()           {}

type TupleType struct {
	TypeHeader
	Elements []Type
}

func (TupleType) NodeKind() string { return "TupleType" }
func (TupleType) _type()           {}

// DerivedType preserves derivation operands and dependent dimensions. The
// concrete symbol identifies array, pointer, reference, slice, range, etc.
type DerivedType struct {
	TypeHeader
	Element    Type
	Dimensions []Expr
}

func (DerivedType) NodeKind() string { return "DerivedType" }
func (DerivedType) _type()           {}

type RefinementTypeExpr struct {
	TypeHeader
	Base      Type
	Predicate Expr
}

func (RefinementTypeExpr) NodeKind() string { return "RefinementTypeExpr" }
func (RefinementTypeExpr) _type()           {}

type PredicateTypeExpr struct {
	TypeHeader
	Binder    symboltable.SymbolID
	Predicate Expr
}

func (PredicateTypeExpr) NodeKind() string { return "PredicateTypeExpr" }
func (PredicateTypeExpr) _type()           {}
