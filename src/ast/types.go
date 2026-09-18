package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// NamedType is a type use resolved through its symbol. The symbol table owns
// the semantic category: builtin, alias, refinement, dependent, parameterized,
// data, associated, or another user-defined type.
type NamedType struct {
	Span
	Symbol symboltable.SymbolID
}

func (NamedType) NodeKind() string { return "NamedType" }
func (NamedType) _type()           {}

// TypeApplication applies type or value arguments to a type-producing symbol.
// Value arguments support dependent applications such as Vector(3).
type TypeApplication struct {
	Span
	Callee    Type
	TypeArgs  []Type
	ValueArgs []Expr
}

func (TypeApplication) NodeKind() string { return "TypeApplication" }
func (TypeApplication) _type()           {}

// FunctionType describes callable parameter and result types.
type FunctionType struct {
	Span
	Parameters []Type
	Results    []Type
}

func (FunctionType) NodeKind() string { return "FunctionType" }
func (FunctionType) _type()           {}

// TupleType describes a tuple of types.
type TupleType struct {
	Span
	Elements []Type
}

func (TupleType) NodeKind() string { return "TupleType" }
func (TupleType) _type()           {}

// DerivedType describes source-level type constructors such as pointers,
// references, arrays, ranges, slices, thunks, and addresses. Kind is syntax,
// not a context-symbol classification.
type DerivedType struct {
	Span
	Kind       string
	Element    Type
	Dimensions []Expr
}

func (DerivedType) NodeKind() string { return "DerivedType" }
func (DerivedType) _type()           {}

// InferType represents an omitted type inferred from an initializer.
type InferType struct {
	Span
}

func (InferType) NodeKind() string { return "InferType" }
func (InferType) _type()           {}

// DynamicType represents the dynamic declaration form.
type DynamicType struct {
	Span
}

func (DynamicType) NodeKind() string { return "DynamicType" }
func (DynamicType) _type()           {}

// UnitType is the empty result/type shape.
type UnitType struct {
	Span
}

func (UnitType) NodeKind() string { return "UnitType" }
func (UnitType) _type()           {}
