package symboltable

import (
	"reflect"
)

type SymbolsToString string

func (s SymbolDetails) Anchor() SymbolTableID { return s.SymbolTableId }

// SymbolInfo defines the interface for querying and mutating symbol metadata.
type SymbolInfo interface {
	GetSymbolID() SymbolID
	GetSymbolType() SymbolID
	GetType() SymbolID
	GetName() SymbolName
	IsInternal() bool
	GetContextID() ContextID
	SetOwnedContextID(ContextID)
	GetSymbolTableID() SymbolTableID
	Anchor() SymbolTableID
	GetMappedName() SymbolName
	SymbolTypeKind() string
}

type ResolutionState string

const (
	Unresolved         ResolutionState = "UnResolved"
	Resolving          ResolutionState = "Resolving"
	Resolved           ResolutionState = "Resolved"
	Ambiguous          ResolutionState = "Ambiguous"
	Invalid            ResolutionState = "Invalid"
	Partially_resolved ResolutionState = "Partially_Resolved"
)

type SymbolDetails struct {
	SymbolId_        SymbolID
	OwnedContextId   ContextID // context owned by this symbol, if any
	SymbolType_      SymbolID
	Name_            SymbolName
	MappedName_      SymbolName
	IsInternal_      bool
	Type_            SymbolID
	SymbolTableId    SymbolTableID   //symboltableID where this symbol is defined
	ResolutionState_ ResolutionState // "resolved" | "unresolved" | "partially_resolved"

}

func (s SymbolDetails) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s SymbolDetails) GetMappedName() SymbolName {
	return s.MappedName_
}

func (s SymbolDetails) GetSymbolTableID() SymbolTableID {
	return s.SymbolTableId
}

func (s SymbolDetails) GetType() SymbolID {
	return s.Type_
}

// GetSymbolID returns the stable identity used by AST and symbol-table artifacts.
func (s SymbolDetails) GetSymbolID() SymbolID { return s.SymbolId_ }

// GetContextID returns the identity of context which it owns if owns or empty or

func (s SymbolDetails) GetContextID() ContextID { return s.OwnedContextId }

// SetOwnedContextID links a scope-owning symbol to its context.
func (s *SymbolDetails) SetOwnedContextID(id ContextID) { s.OwnedContextId = id }

// IsInternal reports whether the SymbolDetails entry is internal.
func (s SymbolDetails) IsInternal() bool {
	return s.IsInternal_
}

// GetName returns the name of a SymbolDetails entry.
func (s SymbolDetails) GetName() SymbolName {
	return s.Name_
}

// GetSymbolType returns the symbolID of type Symbol for a SymbolDetails.
func (s SymbolDetails) GetSymbolType() SymbolID {
	return s.SymbolType_
}

type ProgramSymbol struct {
	SymbolDetails
	Kind_           string // application, library, packaged_export
	LibKind_        string // application, dynamicvmrt, native, na
	DynamicDispatch bool
}

func (s ProgramSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ProgramSymbol) Kind() string {
	return s.Kind_
}

func (s ProgramSymbol) LibKind() string {
	return s.LibKind_
}

type ApplicationSymbol struct {
	SymbolDetails
}

func (s ApplicationSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
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
	Error         BDTKind = "co.error"
	AbstractError BDTKind = "co.AbstractError"
	Value         BDTKind = "co.value"
	Untyped       BDTKind = "co.untyped"
	Uninit        BDTKind = "co.uninit"
)

type BDTtype struct {
	AbstractType
	Kind_       BDTKind
	Isinferred  bool
	IsRedeclare bool
	IsDynamic   bool
}

func (s BDTtype) IsType() bool {
	return true
}
func (s BDTtype) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s BDTtype) Kind() string {
	return string(s.Kind_)
}

type UDTtype struct {
	AbstractType
	Isinferred  bool
	IsRedeclare bool
}

func (s UDTtype) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s UDTtype) IsType() bool {
	return true
}

type PredefinedObjects string

const (
	List       PredefinedObjects = "co.List"
	Set        PredefinedObjects = "co.Set"
	Map        PredefinedObjects = "co.Map"
	Tree       PredefinedObjects = "co.Tree"
	Trie       PredefinedObjects = "co.Trie"
	Array      PredefinedObjects = "co.Array"
	Tuple      PredefinedObjects = "co.Tuple"
	Comparable PredefinedObjects = "co.Comparable"
	Stack      PredefinedObjects = "co.Stack"
	Queue      PredefinedObjects = "co.Queue"
	Matrix     PredefinedObjects = "co.Matrix"
)

// ValueList co.type = co.List(co.int);
// co.List, co.Set, co.Map,  co.Tuple, co.
type PredefinedCollections struct {
	AbstractType
	Kind_ PredefinedObjects
}

func (s PredefinedCollections) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s PredefinedCollections) IsType() bool {
	return true
}

func (s PredefinedCollections) Kind() string {
	return string(s.Kind_)
}

// x co.type = co.int;
type AliasType struct {
	AbstractType
}

func (s AliasType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s AliasType) IsType() bool {
	return true
}

// x co.newtype =  co.int;
type NewType struct {
	AbstractType
}

func (s NewType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s NewType) IsType() bool {
	return true
}

// x co.supertype = sompackage.Employee
type SuperType struct {
	AbstractType
}

func (s SuperType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s SuperType) IsType() bool {
	return true
}

// x co.subtype =  somepackage.Employee
type SubType struct {
	AbstractType
}

func (s SubType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s SubType) IsType() bool {
	return true
}

// x co.opaqueType =  co.int
type OpaqueType struct {
	AbstractType
}

func (s OpaqueType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s OpaqueType) IsType() bool {
	return true
}

// x co.type = co.int | co.string
type ADTtype struct {
	AbstractType
}

func (s ADTtype) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ADTtype) IsType() bool {
	return true
}

// someType co.predicateType = (co.type).where( candidate => candidate == co.int || candidate == co.string );
type PredicateType struct {
	AbstractType
}

func (s PredicateType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s PredicateType) IsType() bool {
	return true
}

// T co.associatedType; // in signature
// T co.associatedType =  co.int; in module
type AssociatedType struct {
	AbstractType
}

func (s AssociatedType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s AssociatedType) IsType() bool {
	return true
}

// Vector(n) co.type = co.dependentType( co.int->([n]) );
type DependentType struct {
	AbstractType
}

func (s DependentType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s DependentType) IsType() bool {
	return true
}

// positiveInt co.refinementType = (co.int).where(_ > 0);
type RefinementType struct {
	AbstractType
}

func (s RefinementType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s RefinementType) IsType() bool {
	return true
}

// x co.type = co.generic(T)
type GenericType struct {
	AbstractType
}

func (s GenericType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s GenericType) IsType() bool {
	return true
}

// co.hokrlt type as value
type Hokrltype struct {
	AbstractType
}

func (s Hokrltype) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s Hokrltype) IsType() bool {
	return true
}

// co.shape
type ShapeType struct {
	AbstractType
}

func (s ShapeType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ShapeType) IsType() bool {
	return true
}

// co.uninit
type UninitType struct {
	AbstractType
}

func (s UninitType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s UninitType) IsType() bool {
	return true
}

// blockormacro co.kind = block | macro
type KindType struct {
	AbstractType
}

func (s KindType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s KindType) IsType() bool {
	return true
}

// x co.type=(co.int, co.int)->(co.bool,co.int)

type FunctionType struct {
	AbstractType
}

func (s FunctionType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s FunctionType) IsType() bool {
	return true
}

// pathIdentity co.type =  (x co.int)->(x.type);

type PathDependentType struct {
	AbstractType
}

func (s PathDependentType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s PathDependentType) IsType() bool {
	return true
}

// someDelegate co.delegate = (a co.int, b co.int)->(co.int, co.int);
type DelegateType struct {
	AbstractType
}

func (s DelegateType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s DelegateType) IsType() bool {
	return true
}

// Option(T) co.type =  co.variants(Some(T), None);
type ParameterizedType struct {
	AbstractType
}

func (s ParameterizedType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ParameterizedType) IsType() bool {
	return true
}

// SelectedValue co.type = co.data(StringValue(co.string), BoolValue(co.bool));
type DataType struct {
	AbstractType
}

func (s DataType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s DataType) IsType() bool {
	return true
}

// x(T,n) co.type= co.tag(T,n);
type TagType struct {
	AbstractType
}

func (s TagType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s TagType) IsType() bool {
	return true
}

// x co.type = co.polymorphic({U}, (U,U)->(U));
// similar to saying forall(T).(T, T)->(T) in other languages
type PolymorphicType struct {
	AbstractType
}

func (s PolymorphicType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s PolymorphicType) IsType() bool {
	return true
}

// Given xx co.type = co.polymorphic({U},(U)->(U))
// identity xx(vaule) =  value; parse this line
type Impredicativetypes struct {
	AbstractType
}

func (s Impredicativetypes) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s Impredicativetypes) IsType() bool {
	return true
}

type DerivedType interface {
	ITypeSymbol
	IsDerived() bool
}

// x  co.type =  co.int->([]);
type ArrayType struct {
	IsZeroLength         bool
	IsZeroDimension      bool
	IsJagged             bool
	IsMultiDimension     bool
	IsVariableLength     bool
	Dimensions           int
	LengthInferredOnInit bool

	AbstractType
}

func (s ArrayType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ArrayType) IsType() bool {
	return true
}
func (s ArrayType) IsDerived() bool {
	return true
}

// x co.type =  co.int->(*);
// DefaultFatIntPtr co.type = co.int->(*, kind="", meta={});
type PointerType struct {
	Degree int
	AbstractType
}

func (s PointerType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s PointerType) IsType() bool {
	return true
}
func (s PointerType) IsDerived() bool {
	return true
}

// x co.type = co.int(&)
// x co.type = co.int(~);
type ReferenceType struct {
	IsLValue        bool
	IsRValue        bool
	IsHeapReference bool
	AbstractType
}

func (s ReferenceType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ReferenceType) IsType() bool {
	return true
}

func (s ReferenceType) IsDerived() bool {
	return true
}

// x co.type = co.int(@);
type AddressType struct {
	AbstractType
}

func (s AddressType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s AddressType) IsType() bool {
	return true
}

func (s AddressType) IsDerived() bool {
	return true
}

// x co.type = co.word->(repr=intptr);
type WordType struct {
	AbstractType
}

func (s WordType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s WordType) IsType() bool {
	return true
}

func (s WordType) IsDerived() bool {
	return true
}

// x co.type = co.int->(..)
type RangeType struct {
	AbstractType
}

func (s RangeType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s RangeType) IsType() bool {
	return true
}

func (s RangeType) IsDerived() bool {
	return true
}

// x co.type = co.int->(^);
type ThunkType struct {
	AbstractType
}

func (s ThunkType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ThunkType) IsType() bool {
	return true
}

func (s ThunkType) IsDerived() bool {
	return true
}

// x co.type = co.int->(:);
type SliceType struct {
	AbstractType
}

func (s SliceType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s SliceType) IsType() bool {
	return true
}

func (s SliceType) IsDerived() bool {
	return true
}

// @co.dap.generic and/or @co.dap.specialize
type GenericSpecializationType struct {
	AbstractType
	IsSpecialize bool
}

func (s GenericSpecializationType) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
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

// _ co.struct = {}
type StructSymbol struct {
	KindSymbol
	HasCompanionUnit    bool
	IsTypeLevelFunction bool
	IsEmbedded          bool
}

func (s KindSymbol) Kind() string {
	return "struct"
}
func (s StructSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

// _ co.cstruct = {}
type CStructSymbol struct {
	KindSymbol
}

func (s CStructSymbol) Kind() string {
	return "cstruct"
}

func (s CStructSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

// _ co.enum= {}
type EnumSymbol struct {
	KindSymbol
	State         bool
	StateFunction bool
}

func (s EnumSymbol) Kind() string {
	return "enum"
}

func (s EnumSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

// _ co.module={}
type ModuleSymbol struct {
	KindSymbol
	Signature []SymbolID
}

func (s ModuleSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ModuleSymbol) Kind() string {
	return "module"
}

// _ co.signature={}
type SignatureSymbol struct {
	KindSymbol
}

func (s SignatureSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s SignatureSymbol) Kind() string {
	return "signature"
}

// _ co.interface = {}
type InterfaceSymbol struct {
	KindSymbol
}

func (s InterfaceSymbol) Kind() string {
	return "interface"
}

func (s InterfaceSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

// _ co.class = {}
type ClassSymbol struct {
	KindSymbol
	Anonymous  bool
	Traits     []SymbolID
	Mixins     []SymbolID
	Extensions []SymbolID
	Interfaces []SymbolID
	Classes    []SymbolID
}

func (s ClassSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ClassSymbol) Kind() string {
	return "class"
}

// _ co.typeclass={}
type TypeClassSymbol struct {
	KindSymbol
}

func (s TypeClassSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s TypeClassSymbol) Kind() string {
	return "typeclass"
}

// _ co.instance= {}
type InstanceSymbol struct {
	KindSymbol
	TypeClass SymbolID
}

func (s InstanceSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s InstanceSymbol) Kind() string {
	return "instance"
}

// _ co.trait = {}
type TraitSymbol struct {
	KindSymbol
}

func (s TraitSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s TraitSymbol) Kind() string {
	return "trait"
}

// _ co.mixin ={}
type MixinSymbol struct {
	KindSymbol
}

func (s MixinSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s MixinSymbol) Kind() string {
	return "mixin"
}

// _ co.componnent={}
type ComponentSymbol struct {
	KindSymbol
	Kind_ string // application, native, dynamicvmrt, packaged, operators
}

func (s ComponentSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ComponentSymbol) Kind() string {
	return "component"
}

// _ co.unit = {}
type UnitSymbol struct {
	KindSymbol
}

func (s UnitSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s UnitSymbol) Kind() string {
	return "package"
}

// _ co.extension={}
type ExtensionSymbol struct {
	KindSymbol
}

func (s ExtensionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ExtensionSymbol) Kind() string {
	return "extension"
}

// _ co.object->()={}
type ObjectSymbol struct {
	KindSymbol
}

func (s ObjectSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ObjectSymbol) Kind() string {
	return "object"
}

// @co.dap.annotation
// _ co.object-={}
type AnnotationSymbol struct {
	ObjectSymbol
}

func (s AnnotationSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s AnnotationSymbol) Kind() string {
	return "Annotation"
}

// _ co.matcher->()= {}
type MatcherSymbol struct {
	KindSymbol
}

func (s MatcherSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s MatcherSymbol) Kind() string {
	return "matcher"
}

// _ co.union ={}
type UnionSymbol struct {
	KindSymbol
}

func (s UnionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s UnionSymbol) Kind() string {
	return "union"
}

// _ co.block ={}
type BlockSymbol struct {
	KindSymbol
	IsAnonymous bool
}

func (s BlockSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s BlockSymbol) Kind() string {
	return "block"
}

// _ co.symbol={}
type SymbolSymbol struct {
	KindSymbol
}

func (s SymbolSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s SymbolSymbol) Kind() string {
	return "symbol"
}

type IFunctionShape interface {
	SymbolInfo
	FunctionShape() string
}

type FunctionScope string

const (
	Lexical FunctionScope = "lexical"
	Dynamic FunctionScope = "dynamic"
	Mixed   FunctionScope = "mixed"
)

type AbstractFunctionShape struct {
	SymbolDetails
	SignatureTypeId SymbolID
	Parameters      []SymbolID
	Results         []SymbolID
}

// x ()->()={}
type FunctionSymbol struct {
	AbstractFunctionShape
	IsClosure    bool
	Inner        bool
	OverLoadable bool
	Overridable  bool
	IsAnonymous  bool
	IsMethod     bool
	Scope        FunctionScope //lexical, dynamic, mixed

}

func (s FunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s FunctionSymbol) FunctionShape() string {
	return "function"
}

// expression-bodied named function
// classify(n co.int)->(co.string) = n.match() .case(v: v > 0 => "positive") .case(v: v < 0 => "negative") .default("zero");
// IntBinary co.type = (co.int, co.int)->(co.int);add IntBinary(a, b) = a + b;
//
// folang doesn't support x:= (a co.int, b co.int)->(co.int) ==> a + b;

type ExpressionBodiedFunction struct {
	AbstractFunctionShape
}

func (s ExpressionBodiedFunction) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (a ExpressionBodiedFunction) Kind() string {
	return "Function_Expression"
}

func (s ExpressionBodiedFunction) FunctionShape() string {
	return "Function_Expression"
}

// @co.dap.decorator
// myDecorator(target co.function)->(co.function) = { }
type DecoratorSymbol struct {
	AbstractFunctionShape
}

func (s DecoratorSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s DecoratorSymbol) FunctionShape() string {
	return "Decorator"
}

// @co.dap.extension(fortype=co.string, what=extends)
// upperCase()->(co.string) = { this => this.upper(); }
type ExtensionMethodSymbol struct {
	AbstractFunctionShape
}

func (s ExtensionMethodSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ExtensionMethodSymbol) FunctionShape() string {
	return "extension_method"
}

// @co.dap.native
// x ()->()={}
type NativeFunctionSymbol struct {
	AbstractFunctionShape
}

func (s NativeFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s NativeFunctionSymbol) FunctionShape() string {
	return "native_function"
}

// @co.dap.macro()
// if(condition expr, body block)->()={}
type MacroSymbol struct {
	AbstractFunctionShape
}

func (s MacroSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s MacroSymbol) FunctionShape() string {

	return "macro"
}

// @co.dap.template
// x ()->(untyped)={}

type TemplateSymbol struct {
	AbstractFunctionShape
}

func (s TemplateSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s TemplateSymbol) FunctionShape() string {
	return "template"
}

// @co.dap.executionmodel()
// fu ()->()={}
type ExecutionModelSymbol struct {
	AbstractFunctionShape
}

func (s ExecutionModelSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ExecutionModelSymbol) FunctionShape() string {
	return "execution_mode"
}

// f (..)(..)->()={}
type CurryingFunctionSymbol struct {
	AbstractFunctionShape
}

func (s CurryingFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s CurryingFunctionSymbol) FunctionShape() string {
	return "currying_function"
}

// @co.dap.defer
// ff()->()={}();
type DeferredFunctionSymbol struct {
	AbstractFunctionShape
}

func (s DeferredFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s DeferredFunctionSymbol) FunctionShape() string {
	return "deferred_function"
}

// someFun(a co.int, b co.int)->(co.int)={}
// k :=  someFun.bind(10, 20);
// res := k.invoke()
type BindCallableSymbol struct {
	AbstractFunctionShape
}

func (s BindCallableSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s BindCallableSymbol) FunctionShape() string {
	return "BindCallable_Signature"
}

// ff( x ..co.int)->()={}
type VariadicFunctionSymbol struct {
	AbstractFunctionShape
}

func (s VariadicFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s VariadicFunctionSymbol) FunctionShape() string {
	return "variadic_function"
}

// fun1(~k co.int, ~v co.int)->()={}
type NamedParameterFunctionSymbol struct {
	AbstractFunctionShape
}

func (s NamedParameterFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s NamedParameterFunctionSymbol) FunctionShape() string {
	return "named_parameter_function"
}

// fun1(k? co.int)->()={}
type OptionalParameterFunctionSymbol struct {
	AbstractFunctionShape
}

func (s OptionalParameterFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s OptionalParameterFunctionSymbol) FunctionShape() string {
	return "Optional_parameter_function"
}

// fun1(k co.int, b co.char = 'A')->(co.int, co.char)={ }
type DefaultParameterFunctionSymbol struct {
	AbstractFunctionShape
}

func (s DefaultParameterFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s DefaultParameterFunctionSymbol) FunctionShape() string {
	return "Default_parameter_function"
}

// @co.dap.indexer(symbol="[]")
// (g MyList) get(index co.int)->(co.int) ={ this => g.eles[index]; }
type IndexerSymbol struct {
	AbstractFunctionShape
}

func (s IndexerSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s IndexerSymbol) FunctionShape() string {
	return "Indexer"
}

// @co.dap.operator(symbol='∩', mode=overload)
// @co.dap.extension(fortype=co.Set, what=extends)intersection(...)
type OperatorFunctionSymbol struct {
	AbstractFunctionShape
}

func (s OperatorFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s OperatorFunctionSymbol) FunctionShape() string {
	return "Operator_Function"
}

// @co.dap.inline
// x ()->()={}
type InlineFunctionSymbol struct {
	AbstractFunctionShape
}

func (s InlineFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s InlineFunctionSymbol) FunctionShape() string {
	return "inline_function"
}

// @co.dap.local
// x ()->()={}
type LocalFunctionSymbol struct {
	AbstractFunctionShape
}

func (s LocalFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s LocalFunctionSymbol) FunctionShape() string {
	return "local_function"
}

// @co.dap.nested
// x ()->()={}
type NestedFunctionSymbol struct {
	AbstractFunctionShape
}

func (s NestedFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s NestedFunctionSymbol) FunctionShape() string {
	return "nested_function"
}

// @co.dap.inner
// x ()->()={}
type InnerFunctionSymbol struct {
	AbstractFunctionShape
}

func (s InnerFunctionSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s InnerFunctionSymbol) FunctionShape() string {
	return "inner_function"
}

// @@new() @@init() ...
type LifecycleSymbol struct {
	AbstractFunctionShape
}

func (s LifecycleSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s LifecycleSymbol) FunctionShape() string {
	return "LifecycleMethods"
}

// (g MyList) get(index co.int)->(co.int) ={ this => g.eles[index]; }
type AssociatedFunction struct {
	AbstractFunctionShape
}

func (s AssociatedFunction) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s AssociatedFunction) FunctionShape() string {
	return "Associated_Function"
}

// add(a co.int, b co.int)->(co.int)={}
// x co.function = ()->(){}
// x co.function = add;

type FunctionObject struct {
	AbstractFunctionShape
}

func (s FunctionObject) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s FunctionObject) Kind() string {
	return "Function Object"
}

type Callable struct {
	AbstractFunctionShape
}

func (s Callable) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s Callable) Kind() string {
	return "Callable"
}

type IIdentifier interface {
	IdentifierType() string
}

// someVariable
type Variable struct {
	SymbolDetails
	IsAdhoc         bool
	IsInternalVar   bool
	IsDiscard       bool
	IsBindVar       bool
	IsPathDependent bool // x somevar.type; kinds
}

func (s Variable) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (a Variable) IdentifierType() string {
	return "Variable"
}

// SomeParameter
type Parameter struct {
	SymbolDetails
	Position int
}

func (s Parameter) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (a Parameter) IdentifierType() string {
	return "Parameter"
}

// SomeReturntype
type Return struct {
	SymbolDetails
	Position int
}

func (s Return) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (a Return) IdentifierType() string {
	return "Return"
}

type MetadataKind string

const (
	Directive  MetadataKind = "Directive"
	Pragma     MetadataKind = "Pragma"
	Annotation MetadataKind = "Annotation"
	Decorator  MetadataKind = "Decorator"
)

type MetaDataValue struct {
	Key         string
	Value       any // later this can become a narrower metadata-value interface
	SourceIndex int
}

// @co.
type MetaDataApplication struct {
	SymbolDetails
	Kind_        MetadataKind
	DefinitionId SymbolID
	Attributes   []MetaDataValue
}

func (s MetaDataApplication) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (a MetaDataApplication) Kind() string {
	return string(a.Kind_)
}

type OperatorFixity string
type OperatorAssociativity string
type OperatorArity string

const (
	OperatorPrefix  OperatorFixity = "prefix"
	OperatorInfix   OperatorFixity = "infix"
	OperatorPostfix OperatorFixity = "postfix"

	OperatorLeft  OperatorAssociativity = "left"
	OperatorRight OperatorAssociativity = "right"
	OperatorNone  OperatorAssociativity = "none"

	OperatorUnary  OperatorArity = "unary"
	OperatorBinary OperatorArity = "binary"
)

// OperatorSymbol is one completely parsed application-global co.operator
// declaration. The registry key carries its symbolic spelling.
type OperatorSymbol struct {
	SymbolDetails
	Fixity          OperatorFixity
	Precedence      int
	Associativity   OperatorAssociativity
	Arity           OperatorArity
	Commutative     bool
	Idempotent      bool
	Identity        string
	Foldable        bool
	Vectorizable    bool
	DistributesOver []string
	Desugar         string
	Spelling        string
}

func (s OperatorSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s OperatorSymbol) Kind() string {
	return "Operator_Symbol"
}

// isNone(), sameRef() ....
type BuiltInProtoTypalProp struct {
	SymbolDetails
}

func (s BuiltInProtoTypalProp) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s BuiltInProtoTypalProp) Kind() string {
	return "BuiltIn_Proto_Typal"
}

// 10, 'X', "AB"
type Literal struct {
	SymbolDetails
}

func (s Literal) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s Literal) Kind() string {
	return "Literal"
}

type KeywordKind string

const (
	This KeywordKind = "this"
	Co   KeywordKind = "co"
	Fol  KeywordKind = "fΦλ"
)

// this->parent, this->parents etc.,
type ThisProperties struct {
	SymbolDetails
}

func (s ThisProperties) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s ThisProperties) Kind() string {
	return "this_property"
}

// co.out, co.in etc.,
type CoProperties struct {
	SymbolDetails
}

func (s CoProperties) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s CoProperties) Kind() string {
	return "Co_property"
}

// FΦλ. etc.,
type FΦλProperties struct {
	SymbolDetails
}

func (s FΦλProperties) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s FΦλProperties) Kind() string {
	return "FΦλ_property"
}

type SigileSymbol struct {
	SymbolDetails
}

func (s SigileSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s SigileSymbol) Kind() string {
	return "Context_Sigil"
}

// this co
type ReservedWord struct {
	SymbolDetails
	Kind_ KeywordKind
}

func (s ReservedWord) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ReservedWord) Kind() string {
	return string(s.Kind_)
}

// 'identifer:
type LabelSymbol struct {
	SymbolDetails
	Kind_ string
}

func (s LabelSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s LabelSymbol) Kind() string {
	return s.Kind_
}

// a()->()=>>b()
type ChainedMethodSymbol struct {
	SymbolDetails
}

func (s ChainedMethodSymbol) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}

func (s ChainedMethodSymbol) Kind() string {
	return "ChainedMethod"
}

// co.MatchBindings
// contains co.tagged values wrapped in MatchBindings object
type MatchBindings struct {
	SymbolDetails
}

func (s MatchBindings) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s MatchBindings) Kind() string {
	return "MatchBindings"
}

// |idx, val| => co.out.println(val)
type LambdaExpression struct {
	SymbolDetails
}

func (s LambdaExpression) SymbolTypeKind() string {
	return string(reflect.TypeOf(s).Name())
}
func (s LambdaExpression) Kind() string {
	return "lambda_expression"
}

var _ SymbolInfo = (*SymbolDetails)(nil)
var _ SymbolInfo = (*ProgramSymbol)(nil)
var _ SymbolInfo = (*ApplicationSymbol)(nil)

var _ SymbolInfo = (*AbstractType)(nil)
var _ SymbolInfo = (*BDTtype)(nil)
var _ SymbolInfo = (*UDTtype)(nil)
var _ SymbolInfo = (*AliasType)(nil)
var _ SymbolInfo = (*NewType)(nil)
var _ SymbolInfo = (*SuperType)(nil)
var _ SymbolInfo = (*SubType)(nil)
var _ SymbolInfo = (*OpaqueType)(nil)
var _ SymbolInfo = (*ADTtype)(nil)
var _ SymbolInfo = (*PredicateType)(nil)
var _ SymbolInfo = (*AssociatedType)(nil)

var _ SymbolInfo = (*DependentType)(nil)
var _ SymbolInfo = (*RefinementType)(nil)
var _ SymbolInfo = (*GenericType)(nil)
var _ SymbolInfo = (*Hokrltype)(nil)
var _ SymbolInfo = (*ShapeType)(nil)
var _ SymbolInfo = (*KindType)(nil)
var _ SymbolInfo = (*FunctionType)(nil)

var _ SymbolInfo = (*DelegateType)(nil)
var _ SymbolInfo = (*ParameterizedType)(nil)
var _ SymbolInfo = (*DataType)(nil)
var _ SymbolInfo = (*TagType)(nil)
var _ SymbolInfo = (*ArrayType)(nil)
var _ SymbolInfo = (*PointerType)(nil)
var _ SymbolInfo = (*ReferenceType)(nil)
var _ SymbolInfo = (*AddressType)(nil)
var _ SymbolInfo = (*WordType)(nil)
var _ SymbolInfo = (*RangeType)(nil)
var _ SymbolInfo = (*ThunkType)(nil)
var _ SymbolInfo = (*SliceType)(nil)
var _ SymbolInfo = (*GenericSpecializationType)(nil)

var _ SymbolInfo = (*KindSymbol)(nil)
var _ SymbolInfo = (*StructSymbol)(nil)
var _ SymbolInfo = (*CStructSymbol)(nil)
var _ SymbolInfo = (*EnumSymbol)(nil)
var _ SymbolInfo = (*ModuleSymbol)(nil)
var _ SymbolInfo = (*SignatureSymbol)(nil)
var _ SymbolInfo = (*InterfaceSymbol)(nil)
var _ SymbolInfo = (*ClassSymbol)(nil)
var _ SymbolInfo = (*TypeClassSymbol)(nil)
var _ SymbolInfo = (*InstanceSymbol)(nil)
var _ SymbolInfo = (*TraitSymbol)(nil)
var _ SymbolInfo = (*MixinSymbol)(nil)
var _ SymbolInfo = (*ComponentSymbol)(nil)
var _ SymbolInfo = (*UnitSymbol)(nil)
var _ SymbolInfo = (*ExtensionSymbol)(nil)
var _ SymbolInfo = (*ObjectSymbol)(nil)
var _ SymbolInfo = (*AnnotationSymbol)(nil)
var _ SymbolInfo = (*MatcherSymbol)(nil)
var _ SymbolInfo = (*UnionSymbol)(nil)
var _ SymbolInfo = (*BlockSymbol)(nil)
var _ SymbolInfo = (*SymbolSymbol)(nil)

var _ SymbolInfo = (*FunctionSymbol)(nil)
var _ SymbolInfo = (*DecoratorSymbol)(nil)
var _ SymbolInfo = (*ExtensionMethodSymbol)(nil)
var _ SymbolInfo = (*NativeFunctionSymbol)(nil)
var _ SymbolInfo = (*MacroSymbol)(nil)
var _ SymbolInfo = (*TemplateSymbol)(nil)
var _ SymbolInfo = (*ExecutionModelSymbol)(nil)
var _ SymbolInfo = (*CurryingFunctionSymbol)(nil)
var _ SymbolInfo = (*DeferredFunctionSymbol)(nil)
var _ SymbolInfo = (*VariadicFunctionSymbol)(nil)
var _ SymbolInfo = (*NamedParameterFunctionSymbol)(nil)
var _ SymbolInfo = (*OptionalParameterFunctionSymbol)(nil)
var _ SymbolInfo = (*DefaultParameterFunctionSymbol)(nil)
var _ SymbolInfo = (*IndexerSymbol)(nil)
var _ SymbolInfo = (*OperatorFunctionSymbol)(nil)
var _ SymbolInfo = (*LifecycleSymbol)(nil)
var _ SymbolInfo = (*AssociatedFunction)(nil)

var _ SymbolInfo = (*Variable)(nil)
var _ SymbolInfo = (*Parameter)(nil)
var _ SymbolInfo = (*Return)(nil)

var _ SymbolInfo = (*MetaDataApplication)(nil)

var _ SymbolInfo = (*OperatorSymbol)(nil)
var _ SymbolInfo = (*BuiltInProtoTypalProp)(nil)
var _ SymbolInfo = (*Literal)(nil)
var _ SymbolInfo = (*ReservedWord)(nil)
var _ SymbolInfo = (*LabelSymbol)(nil)
var _ SymbolInfo = (*ChainedMethodSymbol)(nil)
