package symboltable

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
}

type ResolutionState string

const (
	Unresolved         ResolutionState = "UnResolved"
	Resolving                          = "Resolving"
	Resolved                           = "Resolved"
	Ambiguous                          = "Ambiguous"
	Invalid                            = "Invalid"
	Partially_resolved                 = "Partially_Resolved"
)

type SymbolDetails struct {
	SymbolId_        SymbolID
	OwnedContextId   ContextID // context owned by this symbol, if any
	SymbolType_      SymbolID
	Name_            SymbolName
	IsInternal_      bool
	Type_            SymbolID
	SymbolTableId    SymbolTableID   //symboltableID where this symbol is defined
	ResolutionState_ ResolutionState // "resolved" | "unresolved" | "partially_resolved"

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

func (s ProgramSymbol) Kind() string {
	return s.Kind_
}

func (s ProgramSymbol) LibKind() string {
	return s.LibKind_
}

type ApplicationSymbol struct {
	SymbolDetails
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

func (s BDTtype) Kind() string {
	return string(s.Kind_)
}

type UDTtype struct {
	AbstractType
	Isinferred  bool
	IsRedeclare bool
}

func (s UDTtype) IsType() bool {
	return true
}

type PredefinedObjects string

const (
	List       PredefinedObjects = "co.List"
	Set                          = "co.Set"
	Map                          = "co.Map"
	Tree                         = "co.Tree"
	Trie                         = "co.Trie"
	Array                        = "co.Array"
	Tuple                        = "co.Tuple"
	Comparable                   = "co.Comparable"
	Stack                        = "co.Stack"
	Queue                        = "co.Queue"
	Matrix                       = "co.Matrix"
)

// ValueList co.type = co.List(co.lang.int);
// co.List, co.Set, co.Map,  co.Tuple, co.
type PredefinedCollections struct {
	AbstractType
	Kind_ PredefinedObjects
}

func (s PredefinedCollections) IsType() bool {
	return true
}

func (s PredefinedCollections) Kind() string {
	return string(s.Kind_)
}

// x co.type = co.lang.int;
type AliasType struct {
	AbstractType
}

func (s AliasType) IsType() bool {
	return true
}

// x co.newtype =  co.lang.int;
type NewType struct {
	AbstractType
}

func (s NewType) IsType() bool {
	return true
}

// x co.supertype = sompackage.Employee
type SuperType struct {
	AbstractType
}

func (s SuperType) IsType() bool {
	return true
}

// x co.subtype =  somepackage.Employee
type SubType struct {
	AbstractType
}

func (s SubType) IsType() bool {
	return true
}

// x co.opaqueType =  co.int
type OpaqueType struct {
	AbstractType
}

func (s OpaqueType) IsType() bool {
	return true
}

// x co.type = co.int | co.string
type ADTtype struct {
	AbstractType
}

func (s ADTtype) IsType() bool {
	return true
}

// someType co.predicateType = (co.type).where( candidate => candidate == co.int || candidate == co.string );
type PredicateType struct {
	AbstractType
}

func (s PredicateType) IsType() bool {
	return true
}

// T co.associatedType; // in signature
// T co.associatedType =  co.int; in module
type AssociatedType struct {
	AbstractType
}

func (s AssociatedType) IsType() bool {
	return true
}

// Vector(n) co.type = co.dependentType( co.int->([n]) );
type DependentType struct {
	AbstractType
}

func (s DependentType) IsType() bool {
	return true
}

// positiveInt co.refinementType = (co.int).where(_ > 0);
type RefinementType struct {
	AbstractType
}

func (s RefinementType) IsType() bool {
	return true
}

// x co.type = co.generic(T)
type GenericType struct {
	AbstractType
}

func (s GenericType) IsType() bool {
	return true
}

// co.hokrlt type as value
type Hokrltype struct {
	AbstractType
}

func (s Hokrltype) IsType() bool {
	return true
}

// co.shape
type ShapeType struct {
	AbstractType
}

func (s ShapeType) IsType() bool {
	return true
}

// co.uninit
type UninitType struct {
	AbstractType
}

func (s UninitType) IsType() bool {
	return true
}

// blockormacro co.kind = block | macro
type KindType struct {
	AbstractType
}

func (s KindType) IsType() bool {
	return true
}

// x co.type=(co.int, co.int)->(co.bool,co.int)

type FunctionType struct {
	AbstractType
}

func (s FunctionType) IsType() bool {
	return true
}

// someDelegate co.delegate = (a co.int, b co.int)->(co.int, co.int);
type DelegateType struct {
	AbstractType
}

func (s DelegateType) IsType() bool {
	return true
}

// Option(T) co.type =  co.variants(Some(T), None);
type ParameterizedType struct {
	AbstractType
}

func (s ParameterizedType) IsType() bool {
	return true
}

// SelectedValue co.type = co.data(StringValue(co.string), BoolValue(co.bool));
type DataType struct {
	AbstractType
}

func (s DataType) IsType() bool {
	return true
}

// x(T,n) co.type= co.tag(T,n);
type TagType struct {
	AbstractType
}

func (s TagType) IsType() bool {
	return true
}

// x co.type = co.polymorphic({U}, (U,U)->(U));
// similar to saying forall(T).(T, T)->(T) in other languages
type PolymorphicType struct {
	AbstractType
}

func (s PolymorphicType) IsType() bool {
	return true
}

// Given xx co.type = co.polymorphic({U},(U)->(U))
// identity xx(vaule) =  value; parse this line
type Impredicativetypes struct {
	AbstractType
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

func (s ReferenceType) IsType() bool {
	return true
}

func (s ReferenceType) IsDerived() bool {
	return true
}

// x co.type = co.int(@);
type AdressType struct {
	AbstractType
}

func (s AdressType) IsType() bool {
	return true
}

func (s AdressType) IsDerived() bool {
	return true
}

// x co.type = co.word->(repr=intptr);
type WordType struct {
	AbstractType
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

// _ co.cstruct = {}
type CStructSymbol struct {
	KindSymbol
}

func (s CStructSymbol) Kind() string {
	return "cstruct"
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

// _ co.module={}
type ModuleSymbol struct {
	KindSymbol
}

func (s ModuleSymbol) Kind() string {
	return "module"
}

// _ co.signature={}
type SignatureSymbol struct {
	KindSymbol
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

// _ co.class = {}
type ClassSymbol struct {
	KindSymbol
	Anonnymous bool
}

func (s ClassSymbol) Kind() string {
	return "class"
}

// _ co.typeclass={}
type TypeClassSymbol struct {
	KindSymbol
}

func (s TypeClassSymbol) Kind() string {
	return "typeclass"
}

// _ co.instance= {}
type InstanceSymbol struct {
	KindSymbol
}

func (s InstanceSymbol) Kind() string {
	return "instance"
}

// _ co.trait = {}
type TraitSymbol struct {
	KindSymbol
}

func (s TraitSymbol) Kind() string {
	return "trait"
}

// _ co.mixin ={}
type MixinSymbol struct {
	KindSymbol
}

func (s MixinSymbol) Kind() string {
	return "mixin"
}

// _ co.componnent={}
type ComponentSymbol struct {
	KindSymbol
	Kind_ string // application, native, dynamicvmrt, packaged, operators
}

func (s ComponentSymbol) Kind() string {
	return "component"
}

// _ co.unit = {}
type UnitSymbol struct {
	KindSymbol
}

func (s UnitSymbol) Kind() string {
	return "package"
}

// _ co.extension={}
type ExtensionSymbol struct {
	KindSymbol
}

func (s ExtensionSymbol) Kind() string {
	return "extension"
}

// _ co.object->()={}
type ObjectSymbol struct {
	KindSymbol
}

func (s ObjectSymbol) Kind() string {
	return "object"
}

// @co.dap.annotation
// _ co.object-={}
type AnnotationSymbol struct {
	ObjectSymbol
}

func (s AnnotationSymbol) Kind() string {
	return "Annotation"
}

// _ co.matcher->()= {}
type MatcherSymbol struct {
	KindSymbol
}

func (s MatcherSymbol) Kind() string {
	return "matcher"
}

// _ co.union ={}
type UnionSymbol struct {
	KindSymbol
}

func (s UnionSymbol) Kind() string {
	return "union"
}

// _ co.block ={}
type BlockSymbol struct {
	KindSymbol
	IsAnonymous bool
}

func (s BlockSymbol) Kind() string {
	return "block"
}

// _ co.symbol={}
type SymbolSymbol struct {
	KindSymbol
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
	Dynamic               = "dynamic"
	Mixed                 = "mixed"
)

// x ()->()={}
type FunctionSymbol struct {
	SymbolDetails
	IsClosure    bool
	Inner        bool
	OverLoadable bool
	Overridable  bool
	IsAnonymous  bool
	Scope        FunctionScope //lexical, dynamic, mixed
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
	SymbolDetails
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
	SymbolDetails
}

func (s DecoratorSymbol) FunctionShape() string {
	return "Decorator"
}

// @co.dap.extension(fortype=co.string, what=extends)
// upperCase()->(co.string) = { this => this.upper(); }
type ExensionMethodSymbol struct {
	SymbolDetails
}

func (s ExensionMethodSymbol) FunctionShape() string {
	return "extension_method"
}

// @co.dap.native
// x ()->()={}
type NativeFunctionSymbol struct {
	SymbolDetails
}

func (s NativeFunctionSymbol) FunctionShape() string {
	return "native_function"
}

// @co.dap.macro()
// if(condition expr, body block)->()={}
type MacroSymbol struct {
	SymbolDetails
}

func (s MacroSymbol) FunctionShape() string {

	return "macro"
}

// @co.dap.template
// x ()->(untyped)={}

type TemplateSymbol struct {
	SymbolDetails
}

func (s TemplateSymbol) FunctionShape() string {
	return "template"
}

// @co.dap.executionmodel()
// fu ()->()={}
type ExecutionModelSymbol struct {
	SymbolDetails
}

func (s ExecutionModelSymbol) FunctionShape() string {
	return "execution_mode"
}

// f (..)(..)->()={}
type CurryingFunctionSymbol struct {
	FunctionSymbol
}

func (s CurryingFunctionSymbol) FunctionShape() string {
	return "currying_function"
}

// @co.dap.defer
// ff()->()={}();
type DeferredFunctionSymbol struct {
	FunctionSymbol
}

func (s DeferredFunctionSymbol) FunctionShape() string {
	return "deferred_function"
}

// ff( x ..co.int)->()={}
type VariadicFunctionSymbol struct {
	FunctionSymbol
}

func (s VariadicFunctionSymbol) FunctionShape() string {
	return "variadic_function"
}

// fun1(~k co.int, ~v co.int)->()={}
type NamedParameterFunctionSymbol struct {
	FunctionSymbol
}

func (s NamedParameterFunctionSymbol) FunctionShape() string {
	return "named_parameter_function"
}

// fun1(k? co.int)->()={}
type OptionalParameterFunctionSymbol struct {
	FunctionSymbol
}

func (s OptionalParameterFunctionSymbol) FunctionShape() string {
	return "Optional_parameter_function"
}

// fun1(k co.int, b co.char = 'A')->(co.int, co.char)={ }
type DefaultParameterFunctionSymbol struct {
	FunctionSymbol
}

func (s DefaultParameterFunctionSymbol) FunctionShape() string {
	return "Default_parameter_function"
}

// @co.dap.indexer(symbol="[]")
// (g MyList) get(index co.int)->(co.int) ={ this => g.eles[index]; }
type IndexerSymbol struct {
	FunctionSymbol
}

func (s IndexerSymbol) FunctionShape() string {
	return "Indexer"
}

// @co.dap.operator(symbol='∩', mode=overload)
// @co.dap.extension(fortype=co.Set, what=extends)intersection(...)
type OperatorFunctionSymbol struct {
	FunctionSymbol
}

func (s OperatorFunctionSymbol) FunctionShape() string {
	return "Operator_Function"
}

// @co.dap.inline
// x ()->()={}
type InlineFunctionSymbol struct {
	SymbolDetails
}

func (s InlineFunctionSymbol) FunctionShape() string {
	return "inline_function"
}

// @co.dap.local
// x ()->()={}
type LocalFunctionSymbol struct {
	SymbolDetails
}

func (s LocalFunctionSymbol) FunctionShape() string {
	return "local_function"
}

// @co.dap.nested
// x ()->()={}
type NestedFunctionSymbol struct {
	SymbolDetails
}

func (s NestedFunctionSymbol) FunctionShape() string {
	return "nested_function"
}

// @co.dap.inner
// x ()->()={}
type InnerFunctionSymbol struct {
	SymbolDetails
}

func (s InnerFunctionSymbol) FunctionShape() string {
	return "inner_function"
}

// @@new() @@init() ...
type LifecycleSymbol struct {
	SymbolDetails
}

func (s LifecycleSymbol) FunctionShape() string {
	return "LifecycleMethods"
}

// (g MyList) get(index co.int)->(co.int) ={ this => g.eles[index]; }
type AssociatedFunction struct {
	FunctionSymbol
}

func (s AssociatedFunction) FunctionShape() string {
	return "Associated_Function"
}

type IIdentifier interface {
	IdentifierType() string
}

// someVariable
type Variable struct {
	SymbolDetails
	IsAdhoc       bool
	IsInternalVar bool
	IsDiscard     bool
	IsBindVar     bool
}

func (a Variable) IdentifierType() string {
	return "Variable"
}

// SomeParameter
type Parameter struct {
	SymbolDetails
}

func (a Parameter) IdentifierType() string {
	return "Parameter"
}

// SomeReturntype
type Return struct {
	SymbolDetails
}

func (a Return) IdentifierType() string {
	return "Return"
}

// co.struct
type KindIdentifier struct {
	SymbolDetails
}

func (a KindIdentifier) IdentifierType() string {
	return "KindIdentifier"
}

// co.int
type TypeIdentifier struct {
	SymbolDetails
}

func (a TypeIdentifier) IdentifierType() string {
	return "TypeIdentifier"
}

// ()->()={}
type FunctionShapeIdentifier struct {
	SymbolDetails
}

func (a FunctionShapeIdentifier) IdentifierType() string {
	return "FunctionShape"
}

// @co.
type PDADSymbol struct {
	SymbolDetails
	Kind_ string // annotation. pragma, directive, decorator
}

func (a PDADSymbol) Kind() string {
	return a.Kind_
}

// result := (x <- IntList{1,2,3}).yield(x * 2);
type ForExprSymbol struct {
	SymbolDetails
}

func (a ForExprSymbol) Kind() string {
	return "ForExpr"
}

// println(...)
type CallExpr struct {
	SymbolDetails
}

func (s CallExpr) Kind() string {
	return "CallExpr"
}

// x co.operator={}
type OperatorSymbol struct {
	SymbolDetails
}

func (s OperatorSymbol) Kind() string {
	return "Operator_Symbol"
}

// isNone(), sameRef() ....
type BuiltInProtoTypalProp struct {
	SymbolDetails
}

func (s BuiltInProtoTypalProp) Kind() string {
	return "BuiltIn_Proto_Typal"
}

// 10, 'X', "AB"
type Literal struct {
	SymbolDetails
}

func (s Literal) Kind() string {
	return "Literal"
}

type KeywordKind string

const (
	This KeywordKind = "this"
	Co               = "co"
	FΦλ              = "fΦλ"
)

// this->parent, this->parents etc.,
type ThisProperties struct {
	SymbolDetails
}

func (s ThisProperties) Kind() string {
	return "this_property"
}

// co.out, co.in etc.,
type CoProperties struct {
	SymbolDetails
}

func (s CoProperties) Kind() string {
	return "Co_property"
}

// FΦλ. etc.,
type FΦλProperties struct {
	SymbolDetails
}

func (s FΦλProperties) Kind() string {
	return "FΦλ_property"
}

// this ->>
type ContinueSymbol struct {
	SymbolDetails
}

func (s ContinueSymbol) Kind() string {
	return "continue"
}

// this => <value(s)>
type ReturnSymbol struct {
	SymbolDetails
}

func (s ReturnSymbol) Kind() string {
	return "Return"
}

// this ->|
type BreakSymbol struct {
	SymbolDetails
}

func (s BreakSymbol) Kind() string {
	return "Break"
}

// this ^=> <values>
type ReturnEscapeSymbol struct {
	SymbolDetails
}

func (s ReturnEscapeSymbol) Kind() string {
	return "escape_return"
}

// this co
type ReservedWord struct {
	SymbolDetails
	Kind_ KeywordKind
}

func (s ReservedWord) Kind() string {
	return string(s.Kind_)
}

// 'identifer:
type LabelSymbol struct {
	SymbolDetails
	Kind_ string
}

func (s LabelSymbol) Kind() string {
	return s.Kind_
}

// a()->()=>>b()
type ChainedMethodSymbol struct {
	SymbolDetails
}

func (s ChainedMethodSymbol) Kind() string {
	return "ChainedMethod"
}

// co.MatchBindings
// contains co.tagged values wrapped in MatchBindings object
type MatchBindings struct {
	SymbolDetails
}

func (s MatchBindings) Kind() string {
	return "MatchBindings"
}

// (x > 10)
// co.const.true  even though it is boolean but when participates in an expression like
// true.then() it will be wrapped into conditionobject
// which contains boolean truth and boolean executed
// so when the next in chain  otherwise , default methods are available on condition object
// then method is available on both boolean and conndition object
type ConditionObject struct {
	SymbolDetails
}

func (s ConditionObject) Kind() string {
	return "ConditionObject"
}

// x.match() match generate pattern object on that we have methods like case and default
type PatternObject struct {
	SymbolDetails
}

func (s PatternObject) Kind() string {
	return "PatternObject"
}

// |idx, val| => co.out.println(val)
type LambdaExpression struct {
	SymbolDetails
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
var _ SymbolInfo = (*AdressType)(nil)
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
var _ SymbolInfo = (*ExensionMethodSymbol)(nil)
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
var _ SymbolInfo = (*KindIdentifier)(nil)
var _ SymbolInfo = (*TypeIdentifier)(nil)
var _ SymbolInfo = (*FunctionShapeIdentifier)(nil)
var _ SymbolInfo = (*PDADSymbol)(nil)
var _ SymbolInfo = (*ForExprSymbol)(nil)
var _ SymbolInfo = (*CallExpr)(nil)
var _ SymbolInfo = (*OperatorSymbol)(nil)
var _ SymbolInfo = (*BuiltInProtoTypalProp)(nil)
var _ SymbolInfo = (*Literal)(nil)
var _ SymbolInfo = (*ReservedWord)(nil)
var _ SymbolInfo = (*LabelSymbol)(nil)
var _ SymbolInfo = (*ChainedMethodSymbol)(nil)
