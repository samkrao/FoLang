package symboltable

import (
	"fmt"
)

type SymbolsToString string

func (s SymbolDetails) Anchor() string { return s.SymbolTableId }

// SymbolInfo defines the interface for querying and mutating symbol metadata.
type SymbolInfo interface {
	GetSymbolID() string
	GetSymbolType() string
	GetType() string
	GetName() string
	IsInternal() bool
	Clone() SymbolInfo
	GetContextID() string
	SetOwnedContextID(string)
}

type SymbolDetails struct {
	SymbolId_       string
	OwnedContextId  string // context owned by this symbol, if any
	SymbolType_     string
	Name_           string
	IsInternal_     bool
	Type_           string
	SymbolTableId   string //symboltableID where this symbol is defined
	ResolutionState string // "resolved" | "unresolved" | "partially_resolved"

}

// SymbolDetails holds the core metadata for a symbol table entry.

func (s *SymbolDetails) Clone() SymbolInfo {
	panic(fmt.Sprintf("Clone() not implemented for symbol type: %s", s.SymbolType_))

}

func (s SymbolDetails) GetType() string {
	return s.Type_
}

// GetSymbolID returns the stable identity used by AST and symbol-table artifacts.
func (s SymbolDetails) GetSymbolID() string { return s.SymbolId_ }

// GetContextID returns the identity of context which it owns if owns or empty or

func (s SymbolDetails) GetContextID() string { return s.OwnedContextId }

// SetOwnedContextID links a scope-owning symbol to its context.
func (s *SymbolDetails) SetOwnedContextID(id string) { s.OwnedContextId = id }

// IsInternal reports whether the SymbolDetails entry is internal.
func (s SymbolDetails) IsInternal() bool {
	return s.IsInternal_
}

// GetName returns the name of a SymbolDetails entry.
func (s SymbolDetails) GetName() string {
	return s.Name_
}

// GetSymbolType returns the symbol type string for a SymbolDetails.
func (s SymbolDetails) GetSymbolType() string {
	return s.SymbolType_
}

type ProgramSymbol struct {
	SymbolDetails
	Kind_           string // application, library, packaged_export
	LibKind_        string // application, dynamicvmrt, native, na
	DynamicDispatch bool
}

func (s ProgramSymbol) Kind() string {
	return s.Kind_
}

func (s ProgramSymbol) LibKind() string {
	return s.LibKind_
}

type ITypeSymbol interface {
	SymbolInfo
	IsType() bool
}

type AbstractType struct {
	SymbolDetails
}

type BDTKind string

const (
	Int           BDTKind = "co.int"
	String        BDTKind = "co.string"
	Void          BDTKind = "co.void"
	Double        BDTKind = "co.double"
	Float         BDTKind = "co.float"
	Char          BDTKind = "co.char"
	Bit           BDTKind = "co.bit"
	Long          BDTKind = "co.long"
	Bool          BDTKind = "co.bool"
	Any           BDTKind = "co.any"
	Byte          BDTKind = "co.byte"
	Number        BDTKind = "co.number"
	Error         BDTKind = "co.erro"
	AbstractError BDTKind = "co.AbstractError"
	Value         BDTKind = "co.value"
	MatchBindings BDTKind = "co.MatchBindings"
	Untyped       BDTKind = "co.untyped"
	Uninit        BDTKind = "co.uninit"
)

type BDTtype struct {
	AbstractType
	Kind_ BDTKind
}

func (s BDTtype) IsType() bool {
	return true
}

func (s BDTtype) Kind() string {
	return string(s.Kind_)
}

type UDTtype struct {
	AbstractType
}

func (s UDTtype) IsType() bool {
	return true
}

type AliasType struct {
	AbstractType
}

func (s AliasType) IsType() bool {
	return true
}

type NewType struct {
	AbstractType
}

func (s NewType) IsType() bool {
	return true
}

type SuperType struct {
	AbstractType
}

func (s SuperType) IsType() bool {
	return true
}

type SubType struct {
	AbstractType
}

func (s SubType) IsType() bool {
	return true
}

type OpaqueType struct {
	AbstractType
}

func (s OpaqueType) IsType() bool {
	return true
}

type ADTtype struct {
	AbstractType
}

func (s ADTtype) IsType() bool {
	return true
}

type PredicateType struct {
	AbstractType
}

func (s PredicateType) IsType() bool {
	return true
}

type AssociatedType struct {
	AbstractType
}

func (s AssociatedType) IsType() bool {
	return true
}

type VariantType struct {
	AbstractType
}

func (s VariantType) IsType() bool {
	return true
}

type DependentType struct {
	AbstractType
}

func (s DependentType) IsType() bool {
	return true
}

type RefinementType struct {
	AbstractType
}

func (s RefinementType) IsType() bool {
	return true
}

type GenericType struct {
	AbstractType
}

func (s GenericType) IsType() bool {
	return true
}

type Hokrltype struct {
	AbstractType
}

func (s Hokrltype) IsType() bool {
	return true
}

type ShapeType struct {
	AbstractType
}

func (s ShapeType) IsType() bool {
	return true
}

type KindType struct {
	AbstractType
}

func (s KindType) IsType() bool {
	return true
}

type FunctionType struct {
	AbstractType
}

func (s FunctionType) IsType() bool {
	return true
}

type ForAllType struct {
	AbstractType
}

func (s ForAllType) IsType() bool {
	return true
}

type DelegateType struct {
	AbstractType
}

func (s DelegateType) IsType() bool {
	return true
}

type ParameterizedType struct {
	AbstractType
}

func (s ParameterizedType) IsType() bool {
	return true
}

type DataType struct {
	AbstractType
}

func (s DataType) IsType() bool {
	return true
}

type TagType struct {
	AbstractType
}

func (s TagType) IsType() bool {
	return true
}

type DerivedType interface {
	ITypeSymbol
	IsDerived() bool
}

type ArrayType struct {
	AbstractType
}

func (s ArrayType) IsType() bool {
	return true
}
func (s ArrayType) IsDerived() bool {
	return true
}

type PointerType struct {
	AbstractType
}

func (s PointerType) IsType() bool {
	return true
}
func (s PointerType) IsDerived() bool {
	return true
}

type ReferenceType struct {
	AbstractType
}

func (s ReferenceType) IsType() bool {
	return true
}

func (s ReferenceType) IsDerived() bool {
	return true
}

type AdressType struct {
	AbstractType
}

func (s AdressType) IsType() bool {
	return true
}

func (s AdressType) IsDerived() bool {
	return true
}

type WordType struct {
	AbstractType
}

func (s WordType) IsType() bool {
	return true
}

func (s WordType) IsDerived() bool {
	return true
}

type RangeType struct {
	AbstractType
}

func (s RangeType) IsType() bool {
	return true
}

func (s RangeType) IsDerived() bool {
	return true
}

type ThunkType struct {
	AbstractType
}

func (s ThunkType) IsType() bool {
	return true
}

func (s ThunkType) IsDerived() bool {
	return true
}

type SliceType struct {
	AbstractType
}

func (s SliceType) IsType() bool {
	return true
}

func (s SliceType) IsDerived() bool {
	return true
}

type GenericSpecializationType struct {
	AbstractType
}

func (s GenericSpecializationType) IsType() bool {
	return true
}

func (s GenericSpecializationType) IsDerived() bool {
	return true
}

type IKindSymbol interface {
	SymbolInfo
	Kind() string
}

type KindSymbol struct {
	SymbolDetails
}
type StructSymbol struct {
	KindSymbol
	HasCompanionUnit    bool
	IsTypeLevelFunction bool
}

func (s KindSymbol) Kind() string {
	return "struct"
}

type CStructSymbol struct {
	KindSymbol
}

func (s CStructSymbol) Kind() string {
	return "cstruct"
}

type EnumSymbol struct {
	KindSymbol
}

func (s EnumSymbol) Kind() string {
	return "enum"
}

type ModuleSymbol struct {
	KindSymbol
}

func (s ModuleSymbol) Kind() string {
	return "module"
}

type SignatureSymbol struct {
	KindSymbol
}

func (s SignatureSymbol) Kind() string {
	return "signature"
}

type InterfaceSymbol struct {
	KindSymbol
}

func (s InterfaceSymbol) Kind() string {
	return "interface"
}

type ClassSymbol struct {
	KindSymbol
}

func (s ClassSymbol) Kind() string {
	return "class"
}

type TypeClassSymbol struct {
	KindSymbol
}

func (s TypeClassSymbol) Kind() string {
	return "typeclass"
}

type InstanceSymbol struct {
	KindSymbol
}

func (s InstanceSymbol) Kind() string {
	return "instance"
}

type TraitSymbol struct {
	KindSymbol
}

func (s TraitSymbol) Kind() string {
	return "trait"
}

type MixinSymbol struct {
	KindSymbol
}

func (s MixinSymbol) Kind() string {
	return "mixin"
}

type ComponentSymbol struct {
	KindSymbol
	Kind_ string // application, native, dynamicvmrt, packaged, operators
}

func (s ComponentSymbol) Kind() string {
	return "component"
}

type UnitSymbol struct {
	KindSymbol
}

func (s UnitSymbol) Kind() string {
	return "package"
}

type ExtensionSymbol struct {
	KindSymbol
}

func (s ExtensionSymbol) Kind() string {
	return "extension"
}

type ObjectSymbol struct {
	KindSymbol
}

func (s ObjectSymbol) Kind() string {
	return "object"
}

type AnnotationSymbol struct {
	ObjectSymbol
}

func (s AnnotationSymbol) Kind() string {
	return "Annotation"
}

type MatcherSymbol struct {
	KindSymbol
}

func (s MatcherSymbol) Kind() string {
	return "matcher"
}

type UnionSymbol struct {
	KindSymbol
}

func (s UnionSymbol) Kind() string {
	return "union"
}

type BlockSymbol struct {
	KindSymbol
}

func (s BlockSymbol) Kind() string {
	return "block"
}

type SymbolSymbol struct {
	KindSymbol
}

func (s SymbolSymbol) Kind() string {
	return "symbol"
}

type ExpressionSymbol struct {
	KindSymbol
}

func (s ExpressionSymbol) Kind() string {
	return "expression"
}

type StatementSymbol struct {
	KindSymbol
}

func (s StatementSymbol) Kind() string {
	return "statement"
}

type IFunctionShape interface {
	SymbolInfo
	FunctionShape() string
}

type FunctionSymbol struct {
	SymbolDetails
	IsClosure    bool
	IsInner      bool
	IsNested     bool
	IsLocal      bool
	Inner        bool
	OverLoadable bool
	IsAnonymous  bool
}

func (s FunctionSymbol) FunctionShape() string {
	return "function"
}

type DecoratorSymbol struct {
	SymbolDetails
}

func (s DecoratorSymbol) FunctionShape() string {
	return "Decorator"
}

type ExensionMethodSymbol struct {
	SymbolDetails
}

func (s ExensionMethodSymbol) FunctionShape() string {
	return "extension_method"
}

type NativeFunctionSymbol struct {
	SymbolDetails
}

func (s NativeFunctionSymbol) FunctionShape() string {
	return "native_function"
}

type MacroSymbol struct {
	SymbolDetails
}

func (s MacroSymbol) FunctionShape() string {

	return "macro"
}

type TemplateSymbol struct {
	SymbolDetails
}

func (s TemplateSymbol) FunctionShape() string {
	return "template"
}

type ExecutionModelSymbol struct {
	SymbolDetails
}

func (s ExecutionModelSymbol) FunctionShape() string {
	return "execution_mode"
}

type CurryingFunctionSymbol struct {
	FunctionSymbol
}

func (s CurryingFunctionSymbol) FunctionShape() string {
	return "currying_function"
}

type DeferredFunctionSymbol struct {
	FunctionSymbol
}

func (s DeferredFunctionSymbol) FunctionShape() string {
	return "deferred_function"
}

type VariadicFunctionSymbol struct {
	FunctionSymbol
}

func (s VariadicFunctionSymbol) FunctionShape() string {
	return "variadic_function"
}

type NamedParameterFunctionSymbol struct {
	FunctionSymbol
}

func (s NamedParameterFunctionSymbol) FunctionShape() string {
	return "named_parameter_function"
}

type OptionalParameterFunctionSymbol struct {
	FunctionSymbol
}

func (s OptionalParameterFunctionSymbol) FunctionShape() string {
	return "Optional_parameter_function"
}

type DefaultParameterFunctionSymbol struct {
	FunctionSymbol
}

func (s DefaultParameterFunctionSymbol) FunctionShape() string {
	return "Default_parameter_function"
}

type ExtensionMethodSymbol struct {
	FunctionSymbol
}

func (s ExtensionMethodSymbol) FunctionShape() string {
	return "extension_method"
}

type IndexerSymbol struct {
	FunctionSymbol
}

func (s IndexerSymbol) FunctionShape() string {
	return "Indexer"
}

type OperatorFunctionSymbol struct {
	FunctionSymbol
}

func (s OperatorFunctionSymbol) FunctionShape() string {
	return "Operator_Function"
}

type AssociatedFunction struct {
	FunctionSymbol
}

func (s AssociatedFunction) FunctionShape() string {
	return "Associated_Function"
}

type IIdentifier interface {
	IdentifierType() string
}

type Variable struct {
	SymbolDetails
}

func (a Variable) IdentifierType() string {
	return "Variable"
}

type Parameter struct {
	SymbolDetails
}

func (a Parameter) IdentifierType() string {
	return "Parameter"
}

type Return struct {
	SymbolDetails
}

func (a Return) IdentifierType() string {
	return "Return"
}

type KindIdentifier struct {
	SymbolDetails
}

func (a KindIdentifier) IdentifierType() string {
	return "KindIdentifier"
}

type TypeIdentifier struct {
	SymbolDetails
}

func (a TypeIdentifier) IdentifierType() string {
	return "TypeIdentifier"
}

type FunctionShapeIdentifier struct {
	SymbolDetails
}

func (a FunctionShapeIdentifier) IdentifierType() string {
	return "FunctionShape"
}

type PDADSymbol struct {
	SymbolDetails
	Kind_ string // annotation. pragma, directive, decorator
}

func (a PDADSymbol) Kind() string {
	return a.Kind_
}

type LetVarSymbol struct {
	SymbolDetails
}

func (a LetVarSymbol) Kind() string {
	return "letvar"
}

type LetfunSymbol struct {
	SymbolDetails
}

func (a LetfunSymbol) Kind() string {
	return "letfun"
}

type ForExprSymbol struct {
	SymbolDetails
}

func (a ForExprSymbol) Kind() string {
	return "ForExpr"
}

type CallExpr struct {
	SymbolDetails
}

func (s CallExpr) Kind() string {
	return "CallExpr"
}

type OperatorSymbol struct {
	SymbolDetails
}

func (s OperatorSymbol) Kind() string {
	return "Operator_Symbol"
}

type BuiltInProtoTypal struct {
	SymbolDetails
}

func (s BuiltInProtoTypal) Kind() string {
	return "BuiltIn_Proto_Typal"
}

type Literal struct {
	SymbolDetails
}

func (s Literal) Kind() string {
	return "Literal"
}
