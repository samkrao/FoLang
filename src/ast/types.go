package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// NamedType refers to a resolved or unresolved named type symbol.
type NamedType struct {
	Span
	Symbol symboltable.SymbolID
}

func (NamedType) NodeKind() string { return "NamedType" }
func (NamedType) _type()           {}

type BuiltinType struct{ SymbolType }

func (BuiltinType) NodeKind() string { return "BuiltinType" }

type UserDefinedType struct{ SymbolType }

func (UserDefinedType) NodeKind() string { return "UserDefinedType" }

type AbstractTypeRef struct{ SymbolType }

func (AbstractTypeRef) NodeKind() string { return "AbstractTypeRef" }

// SymbolType is the common AST shape for a type whose semantic category is
// carried by the resolved symbol. The specialized nodes below preserve the
// source-level category without copying symbol metadata into the AST.
type SymbolType struct {
	Span
	Symbol symboltable.SymbolID
}

func (SymbolType) NodeKind() string { return "SymbolType" }
func (SymbolType) _type()           {}

type AliasType struct{ SymbolType }

func (AliasType) NodeKind() string { return "AliasType" }

type NewType struct{ SymbolType }

func (NewType) NodeKind() string { return "NewType" }

type SuperType struct{ SymbolType }

func (SuperType) NodeKind() string { return "SuperType" }

type SubType struct{ SymbolType }

func (SubType) NodeKind() string { return "SubType" }

type OpaqueType struct{ SymbolType }

func (OpaqueType) NodeKind() string { return "OpaqueType" }

type ADTType struct{ SymbolType }

func (ADTType) NodeKind() string { return "ADTType" }

type PredicateType struct {
	SymbolType
	Predicate Expr
}

func (PredicateType) NodeKind() string { return "PredicateType" }

type AssociatedTypeRef struct{ SymbolType }

func (AssociatedTypeRef) NodeKind() string { return "AssociatedTypeRef" }

type VariantType struct{ SymbolType }

func (VariantType) NodeKind() string { return "VariantType" }

type DependentType struct {
	SymbolType
	Index Expr
}

func (DependentType) NodeKind() string { return "DependentType" }

type RefinementType struct {
	SymbolType
	Base      Type
	Predicate Expr
}

func (RefinementType) NodeKind() string { return "RefinementType" }

type GenericType struct {
	SymbolType
	Parameters []symboltable.SymbolID
}

func (GenericType) NodeKind() string { return "GenericType" }

type HigherKindType struct{ SymbolType }

func (HigherKindType) NodeKind() string { return "HigherKindType" }

type HKTType struct{ SymbolType }

func (HKTType) NodeKind() string { return "HKTType" }

type ShapeType struct{ SymbolType }

func (ShapeType) NodeKind() string { return "ShapeType" }

type KindType struct{ SymbolType }

func (KindType) NodeKind() string { return "KindType" }

type ForAllType struct {
	SymbolType
	Parameters []symboltable.SymbolID
	Body       Type
}

func (ForAllType) NodeKind() string { return "ForAllType" }

type DelegateType struct{ SymbolType }

func (DelegateType) NodeKind() string { return "DelegateType" }

type ParameterizedType struct {
	SymbolType
	Arguments []Type
}

func (ParameterizedType) NodeKind() string { return "ParameterizedType" }

type DataType struct{ SymbolType }

func (DataType) NodeKind() string { return "DataType" }

type TagType struct{ SymbolType }

func (TagType) NodeKind() string { return "TagType" }

type GenericSpecializationType struct {
	SymbolType
	Arguments []Type
}

func (GenericSpecializationType) NodeKind() string { return "GenericSpecializationType" }

// TypeApplication applies type arguments to a named or constructed type.
type TypeApplication struct {
	Span
	Callee Type
	Args   []Type
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

// TupleType describes a tuple's element types.
type TupleType struct {
	Span
	Elements []Type
}

func (TupleType) NodeKind() string { return "TupleType" }
func (TupleType) _type()           {}

// PointerType describes a pointer or other indirection form.
type PointerType struct {
	Span
	Element Type
	Kind    string
}

func (PointerType) NodeKind() string { return "PointerType" }
func (PointerType) _type()           {}

// ReferenceType describes a reference, lvalue reference, heap reference,
// address, thunk, or slice type.
type ReferenceType struct {
	Span
	Element Type
	Kind    string
}

func (ReferenceType) NodeKind() string { return "ReferenceType" }
func (ReferenceType) _type()           {}

// ArrayType describes fixed, inferred, variable-length, or multidimensional
// array shapes. Dimensions are represented as expressions for later checking.
type ArrayType struct {
	Span
	Element    Type
	Dimensions []Expr
}

func (ArrayType) NodeKind() string { return "ArrayType" }
func (ArrayType) _type()           {}

// RangeType describes a range whose element type is Element.
type RangeType struct {
	Span
	Element Type
}

func (RangeType) NodeKind() string { return "RangeType" }
func (RangeType) _type()           {}

type AddressType struct{ SymbolType }

func (AddressType) NodeKind() string { return "AddressType" }

type WordType struct{ SymbolType }

func (WordType) NodeKind() string { return "WordType" }

type ThunkType struct{ SymbolType }

func (ThunkType) NodeKind() string { return "ThunkType" }

type SliceType struct{ SymbolType }

func (SliceType) NodeKind() string { return "SliceType" }

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
