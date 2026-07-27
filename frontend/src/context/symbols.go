package symboltable

import (
	"fmt"

	"github.com/jinzhu/copier"
)

type ResolveState string

const (
	Unresolved        ResolveState = "UNRESOLVED"
	Resolving                      = "RESOLVING"
	PartiallyResolved              = "PARTIALLY_RESOLVED"
	Resolved                       = "RESOLVED"
	Error                          = "ERROR"
)

type SymbolsToString string

// SymbolInfo defines the interface for querying and mutating symbol metadata.
type SymbolInfo interface {
	GetSymbolType() string
	GetType() string
	GetName() string
	IsInternal() bool
	ResolutionState() ResolveState
	Clone() SymbolInfo
}

type SymbolDetails struct {
	SymbolType_   string
	Name_         string
	State         ResolveState
	IsInternal_   bool
	Type_         string
	SymbolTableId string //symboltableID

}

// SymbolDetails holds the core metadata for a symbol table entry.

func (s *SymbolDetails) Clone() SymbolInfo {
	panic(fmt.Sprintf("Clone() not implemented for symbol type: %s", s.SymbolType_))

}

func (s SymbolDetails) GetType() string {
	return s.Type_
}

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
func (s SymbolDetails) ResolutionState() ResolveState {
	return s.State
}

// VariableDetails extends SymbolDetails for variable symbols with an internal flag.
type VariableDetails struct {
	SymbolDetails
	IsInternal_     bool
	Inferred        bool
	Dynamic         bool
	Auto            bool
	Mutable         bool
	TypeInfo        TypeSymbol
	IsParam         bool
	IsArg           bool
	ReturnVar       bool
	IsSealed        bool
	IsPublic        bool
	IsPrivate       bool
	IsPackageScope  bool
	IsFriend        bool
	IsInternalScope bool
	DuckType        bool
	HasInitValue    bool
	LocalBinding    bool //patternmatching and others where this is bind variable
	ExplicitType    bool
	Optional        bool
	IsInner         bool
	ThunkVar        bool
	IsCompound      bool
	IsListType      bool
	ActType_        string
	SubType_        string
	SubID           string
	VarType         string
	NamedArg        bool
	VariadicParam   bool
	DefaultArgs     bool
	Type_           string
}

func (s *VariableDetails) Clone() SymbolInfo {
	var dest VariableDetails
	if err := copier.Copy(&dest, s); err != nil {
		panic(err) // or log.Fatal
	}
	return &dest

}

// IsInternal reports whether the VariableDetails entry is internal.
func (s *VariableDetails) IsInternal() bool {
	return s.IsInternal_
}

// SetIsInternal sets the internal flag on a VariableDetails.
func (s *VariableDetails) SetIsInternal(f bool) {
	s.IsInternal_ = f

}
func (s *VariableDetails) GetType() string {
	return s.Type_
}

type VarKind string

const (
	VarVar       VarKind = "Var"
	VarParam     VarKind = "param"
	VarField     VarKind = "field"
	VarStatic    VarKind = "static"
	VarConst     VarKind = "const"
	VarPointer   VarKind = "pointer"
	VarArray     VarKind = "array"
	VarRange     VarKind = "range"
	VarReference VarKind = "reference"
	VarAddress   VarKind = "address"
	VarThunk     VarKind = "thunk"
	VarSlice     VarKind = "slice"
	VarReturn    VarKind = "return"
	VarArg       VarKind = "arg"
)

type IVarSymbol interface {
	SymbolInfo
	Kind() string
}

type VarSymbol struct {
	VariableDetails
	Discard      bool
	BindVar      bool
	Extern       bool
	Forward      bool
	ABI          string // "c", "c++", "folang"
	Mangled      bool
	ExternName   string // name= field
	Alias        bool   // alias= field
	Weak         bool   // weak= field
	AutoCreate   bool
	ExistsAssign bool
}

func (s *VarSymbol) Kind() string {
	return string(VarVar)
}

func (s *VarSymbol) Clone() SymbolInfo {
	var dest VarSymbol
	if err := copier.Copy(&dest, s); err != nil {
		panic(err) // or log.Fatal
	}
	return &dest

}

// IsInternal reports whether the VariableDetails entry is internal.
func (s *VarSymbol) IsInternal() bool {
	return s.IsInternal_
}

// SetIsInternal sets the internal flag on a VariableDetails.
func (s *VarSymbol) SetIsInternal(f bool) {
	s.IsInternal_ = f

}
func (s *VarSymbol) GetType() string {
	return s.Type_
}

type PointerSymbol struct {
	VariableDetails
	IsRaw               bool
	Count               int
	PtrToConstType      bool // if set true pointer reference can be changed but not value at the address
	ConstPtrToType      bool // if set true value at address can be changed
	ConstPtrToConstType bool // if set true nether pointer reference nor value at address can be changed
	ISFatPointer        bool
	MetaData            map[string]any
	IsShared            bool
	IsWeak              bool
	IsUnique            bool
	Kind_               string
}

func (s *PointerSymbol) Kind() string {
	return string(VarPointer)
}

type ArraySymbol struct {
	VariableDetails
	IsJagged        bool
	IsMultiDimesion bool
	IsSlice         bool
	IsZeroLen       bool
	IsZeroDim       bool
	VLA             bool
	IsRawArray      bool
	ElementLenDecl  bool
	IsDynamic       bool
	SizeFromInit    bool
}

func (s *ArraySymbol) Kind() string {
	return string(VarArray)
}

type RangeSymbol struct {
	VariableDetails
}

func (s *RangeSymbol) Kind() string {
	return string(VarRange)
}

type ThunkSymbol struct {
	VariableDetails
}

func (s *ThunkSymbol) Kind() string {
	return string(VarThunk)
}

type ReferenceSymbol struct {
	VariableDetails
	Lref  bool
	Ref   bool
	Heap  bool
	Count int
}

func (s *ReferenceSymbol) Kind() string {
	return string(VarReference)
}

type AddressSymbol struct {
	VariableDetails
	Addressop bool
	Wordtype  bool
}

func (s *AddressSymbol) Kind() string {
	return string(VarAddress)
}

type FunctionSymbol struct {
	SymbolDetails
	Curried              bool
	IsGeneric            bool
	Closure              bool
	Anonymous            bool
	FunctionObject       bool
	NamedParams          bool
	Variadic             bool
	VariantTypeArgs      bool
	OptionalArgs         bool
	DefaultParams        bool
	FunctionExpression   bool
	Associated           bool
	AssociatedNode       StructSymbol
	Delegate             bool
	InnerFunction        bool
	FWRF                 bool //returns functions
	FWPF                 bool // parameterrs function
	IsMethod             bool
	ClassMethod          bool
	StaticMethod         bool
	InstanceMethod       bool
	ObjectMethod         bool
	FunctionChain        bool
	Lazy                 bool
	Eager                bool
	LexicalStaticScope   bool
	DynamicScope         bool
	MixedScope           bool
	CContinuation        bool
	SRContinuation       bool //shif reduce
	PCContinuation       bool //promptContrl
	TrampolineCC         bool
	YeildCC              bool
	Fiber                bool
	Thread               bool
	Task                 bool
	Parallel             bool
	Concurrent           bool
	HScale               bool
	VScale               bool
	Fork                 bool
	Spawn                bool
	Exec                 bool
	Process              bool
	Async                bool
	Channel              bool
	Promise              bool
	Future               bool
	Defer                bool
	CPS                  bool
	Coroutine            bool
	Goroutine            bool
	Actor                bool
	Event                bool
	Generator            bool
	Iterator             bool
	Inline               bool
	Lambda               bool
	Overrloaded          bool
	Overrridden          bool
	Abstract             bool
	Virtual              bool
	Native               bool
	ISMachincode         bool
	ISAsm                bool
	ISNaked              bool
	Issealed             bool
	IsPublic             bool
	IsPrivate            bool
	IsProtected          bool
	IsPackageScope       bool
	IsBody               bool
	IsTailCall           bool
	OnlyParamTypes       bool
	AsFunctionParam      bool
	ParentIsFunc         bool
	FunTypeParRet        bool
	RestrictedToOverload bool
	IsExportable         bool
	WhatisIt             []string
	IsMeth               bool
	IsRestricted         bool
	IsInner              bool
	Type_                string
	Callback             bool
	IsOperator           bool
}

func (f *FunctionSymbol) GetType() string {
	return f.Type_
}

type DelegateSymbol struct {
	SymbolDetails
	Funcs []FunctionSymbol
	Name  string
}

type LambdaSymbol struct {
	SymbolDetails
}

type Indexer struct {
	SymbolDetails
}

type TypeclassSymbol struct {
	SymbolDetails
	ISFunctor     bool
	ISApplicative bool
	ISMonad       bool
	ISMonoid      bool
	ISTransormer  bool
	IsMonod       bool
	ISFoldeable   bool
}

type TypeConstructor struct {
	SymbolDetails
}
type VariantConstructor struct {
	SymbolDetails
}

type LetBindings struct {
	SymbolDetails
}

type LabelSymbol struct {
	SymbolDetails
}

type BlockSymbol struct {
	SymbolDetails
	IsNamed bool
}

type FunctionPattern struct {
	SymbolDetails
	IsLetForm bool
}
type MatcherSymbol struct {
	SymbolDetails
}

type MatcherImplSymbol struct {
	SymbolDetails
}

type ExtensionSymbol struct {
	SymbolDetails
}

type ApplicationSymbol struct {
	SymbolDetails
	Name string
}

type LibrarySymbol struct {
	SymbolDetails
	Name string
	Type string
}

type PackageSymbol struct {
	SymbolDetails
	Name        string
	Library     bool
	Zone        string
	Type        string
	Kind        string
	Parent      []PackageSymbol
	ParentKind  string
	IsParent    bool
	IsSealed    bool
	IsFFI       bool
	IsSystem    bool
	AsExpr      bool
	IsCodeBlock bool
}

type ClassSymbol struct {
	SymbolDetails
	Virtual          bool
	Abstract         bool
	CaseClass        bool
	Property         bool
	Extensions       []string
	Extends          []string
	Inherits         []string
	Uses             []string
	Mixin            []string
	Traits           []string
	With             string
	ComposeAssociate []string
	Anonymous        bool
	InnerClass       bool
	Implements       []string
	IsSealed         bool
	IsPublic         bool
	IsPrivate        bool
	IsPackageScope   bool
	IsInternal_      bool
	IsGeneric        bool
}

type ModuleSymbol struct {
	SymbolDetails
	Signature      SignatureSymbol
	IsSealed       bool
	IsPublic       bool
	IsPrivate      bool
	IsPackageScope bool
	IsInternal_    bool
	IsImpl         bool
	Name           string
}

type InterfaceSymbol struct {
	SymbolDetails
	InnerInterface bool
}

type SignatureSymbol struct {
	SymbolDetails
}

type ITypeSymbol interface {
	SymbolInfo
	IsType() bool
}

type EnumSymbol struct {
	SymbolDetails
	InnerEnum bool
}

func (e *EnumSymbol) IsType() bool {
	return true
}

type StructSymbol struct {
	SymbolDetails
	CStruct     bool
	Embedded    bool
	Composed    bool
	Associated  bool
	InnerStruct bool
	IsSealed    bool
	Anonymous   bool
}

func (e *StructSymbol) IsType() bool {
	return true
}

type UnionSymbol struct {
	SymbolDetails
}

func (e *UnionSymbol) IsType() bool {
	return true
}

type KindSymbol struct {
	SymbolDetails
}

type HokrtlSymbol struct {
	SymbolDetails
	HKType bool
	HRType bool
	HOType bool
	HLType bool
}

type TypeSymbol struct {
	SymbolDetails
	Alias          bool
	NewType        bool
	SubType        bool
	SuperType      bool
	DependentType  bool
	OpaqueType     bool
	AssociatedType bool
	FuntionTyoe    bool
	ForallType     bool
	UDT            bool
	ADT            bool
	BDT            bool
	HOKRTyple      HokrtlSymbol
	AsExpr         bool
	UnionType      bool
	TypeType       string
	FunType        bool
	Kind           KindSymbol
	IsSealed       bool
	IsPublic       bool
	IsPrivate      bool
	IsPackageScope bool
	IsInternal_    bool
	ExplicitType   bool
	AsStmt         bool
	No_user_name   bool
	Ephimeral      bool
	Hidden         bool
	RtErased       bool
	IsGenericType  bool
}

func (e *TypeSymbol) IsType() bool {
	return true
}
func (s *TypeSymbol) Clone() SymbolInfo {
	var dest TypeSymbol
	if err := copier.Copy(&dest, s); err != nil {
		panic(err) // or log.Fatal
	}
	return &dest

}

// IsInternal reports whether the VariableDetails entry is internal.
func (s *TypeSymbol) IsInternal() bool {
	return s.IsInternal_
}

// SetIsInternal sets the internal flag on a VariableDetails.
func (s *TypeSymbol) SetIsInternal(f bool) {
	s.IsInternal_ = f

}

type ForComprehension struct {
	SymbolDetails
}
type InstanceSymbol struct {
	SymbolDetails
	TypeClass TypeclassSymbol
}

type ObjectSymbol struct {
	SymbolDetails
	Object    AnnotationSymbol
	ObjectFor string
}

type StaticSymbol struct {
	SymbolDetails
}

type AnnotationSymbol struct {
	SymbolDetails
	Runtime     bool
	Compiletime bool
}

type DecoratorSymbol struct {
	SymbolDetails
}

type MacroSymbol struct {
	SymbolDetails
	Hygeine bool
	GenSym  bool
	Escape  bool
	Depends *MacroSymbol
}

type TemplateDetails struct {
	SymbolDetails
	Untyped bool
	Typed   bool
}

type OperatorDetails struct {
	SymbolDetails
	Define   bool
	Override bool
	Overload bool
	Default  bool
	Provided bool
}

type GenericDetails struct {
	SymbolDetails
	TypeSymbols      []string
	Scope_           string // "runtime" | "compiletime"
	Where            string // "callsite" | "usesite"
	Reified          bool   // type info available at runtime
	GenericTypeParam []GenericTypeParam
}

// GenericTypeParam represents a single type parameter in a generic declaration.
// e.g. T, T: Orderable, T (variance=covariant, bound=Number)
type GenericTypeParam struct {
	Name          string // e.g. "T"
	Constraint    string // bound type name, empty if unconstrained
	Variance      string // "covariant" | "contravariant" | "invariant" | ""
	Kind_         string // "param" | "result" | "var" | "arg" | ""
	Default       string // default type, empty if none
	Nullable      bool   // whether T can be nullable
	Inclusive     bool   // whether bound is inclusive
	Impredicative bool   // when true, allows T to be instantiated with a forall type (v2)
	TypeKind      string // "type" | "class" | "function" | "module" | "package" | ""
	Types         string // comma-separated list of allowed types for constraints
}
type DirectivePragmaDetails struct {
	SymbolDetails
	IsPragma bool
}

type DymanicRuntime struct {
	SymbolDetails
}
type UseSymbol struct {
	SymbolDetails
}

type ExpressionSymbol struct {
	SymbolDetails
	Type_ string
}

type StatmentSymbol struct {
	SymbolDetails
	Type_ string
}

type Symbol struct {
	SymbolDetails
	SymbolDet     SymbolInfo
	ResolveState_ ResolveState
}

// ================== SymbolsToSring ==================

const (
	S_VarSymbol              SymbolsToString = "Var"
	S_FunctionSymbol         SymbolsToString = "Fun"
	S_PackageSymbol          SymbolsToString = "Package"
	S_MacroSymbol            SymbolsToString = "Macro"
	S_ExpressionSymbol       SymbolsToString = "Expr"
	S_TemplateDetails        SymbolsToString = "TemplateDetails"
	S_FunctionPattern        SymbolsToString = "FunctionPattern"
	S_PointerSymbol          SymbolsToString = "Pointer"
	S_ArraySymbol            SymbolsToString = "Array"
	S_RangeSymbol            SymbolsToString = "Range"
	S_TypeSymbol             SymbolsToString = "Type"
	S_DelegateSymbol         SymbolsToString = "Delegate"
	S_LambdaSymbol           SymbolsToString = "Lambda"
	S_Indexer                SymbolsToString = "Indexer"
	S_TypeclassSymbol        SymbolsToString = "Typeclass"
	S_TypeConstructor        SymbolsToString = "TypeConstructor"
	S_VariantConstructor     SymbolsToString = "VariantConstructor"
	S_LetBindings            SymbolsToString = "LetBindings"
	S_LabelSymbol            SymbolsToString = "LabelSymbol"
	S_BlockSymbol            SymbolsToString = "BlockSymbol"
	S_ForComprehension       SymbolsToString = "ForComprehension"
	S_MatcherSymbol          SymbolsToString = "MatcherSymbol"
	S_MatcherImplSymbol      SymbolsToString = "MatcherImplSymbol"
	S_ExtensionSymbol        SymbolsToString = "ExtensionSymbol"
	S_ClassSymbol            SymbolsToString = "Class"
	S_ModuleSymbol           SymbolsToString = "Module"
	S_InterfaceSymbol        SymbolsToString = "Interface"
	S_SignatureSymbol        SymbolsToString = "Signature"
	S_EnumSymbol             SymbolsToString = "Enum"
	S_StructSymbol           SymbolsToString = "Struct"
	S_UnionSymbol            SymbolsToString = "Union"
	S_KindSymbol             SymbolsToString = "Kind"
	S_HokrtlSymbol           SymbolsToString = "Hokrtl"
	S_UseSymbol              SymbolsToString = "Use"
	S_ObjectSymbol           SymbolsToString = "Object"
	S_StaticSymbol           SymbolsToString = "Static"
	S_AnnotationSymbol       SymbolsToString = "Annotation"
	S_DecoratorSymbol        SymbolsToString = "Decorator"
	S_InstanceSymbol         SymbolsToString = "Instance"
	S_ReferenceSymbol        SymbolsToString = "Reference"
	S_AddressSymbol          SymbolsToString = "Address"
	S_ThunkSymbol            SymbolsToString = "Thunk"
	S_DymanicRuntime         SymbolsToString = "DynamicRuntime"
	S_DirectivePragmaDetails SymbolsToString = "DirectivePragmaDetails"
	S_SymbolDetails          SymbolsToString = "SymbolDetails"
	S_GenericDetails         SymbolsToString = "GenericDetails"
	S_OperatorDetails        SymbolsToString = "OperatorDetails"
	S_VariableDetails        SymbolsToString = "Var"
	S_StatmentSymbol         SymbolsToString = "Statement"
	S_SymbolRefExpr          SymbolsToString = "SymbolRef"
	S_ForAllSymbol           SymbolsToString = "ForAll"
	S_FunctionType           SymbolsToString = "FuntionType"
	S_Program                SymbolsToString = "Program"
)

// ============ COMPILE-TIME CHECKS ============

var _ SymbolInfo = (*VarSymbol)(nil)
var _ SymbolInfo = (*PackageSymbol)(nil)
var _ SymbolInfo = (*FunctionSymbol)(nil)
var _ SymbolInfo = (*MacroSymbol)(nil)
var _ SymbolInfo = (*ExpressionSymbol)(nil)
var _ SymbolInfo = (*TemplateDetails)(nil)
var _ SymbolInfo = (*FunctionPattern)(nil)
var _ SymbolInfo = (*PointerSymbol)(nil)
var _ SymbolInfo = (*ArraySymbol)(nil)
var _ SymbolInfo = (*RangeSymbol)(nil)
var _ SymbolInfo = (*TypeSymbol)(nil)
var _ SymbolInfo = (*DelegateSymbol)(nil)
var _ SymbolInfo = (*LambdaSymbol)(nil)
var _ SymbolInfo = (*Indexer)(nil)
var _ SymbolInfo = (*TypeclassSymbol)(nil)
var _ SymbolInfo = (*TypeConstructor)(nil)
var _ SymbolInfo = (*VariantConstructor)(nil)
var _ SymbolInfo = (*LetBindings)(nil)
var _ SymbolInfo = (*LabelSymbol)(nil)
var _ SymbolInfo = (*BlockSymbol)(nil)
var _ SymbolInfo = (*ForComprehension)(nil)
var _ SymbolInfo = (*MatcherSymbol)(nil)
var _ SymbolInfo = (*MatcherImplSymbol)(nil)
var _ SymbolInfo = (*ExtensionSymbol)(nil)
var _ SymbolInfo = (*ClassSymbol)(nil)
var _ SymbolInfo = (*ModuleSymbol)(nil)
var _ SymbolInfo = (*InterfaceSymbol)(nil)
var _ SymbolInfo = (*SignatureSymbol)(nil)
var _ SymbolInfo = (*EnumSymbol)(nil)
var _ SymbolInfo = (*StructSymbol)(nil)
var _ SymbolInfo = (*UnionSymbol)(nil)
var _ SymbolInfo = (*KindSymbol)(nil)
var _ SymbolInfo = (*HokrtlSymbol)(nil)
var _ SymbolInfo = (*UseSymbol)(nil)
var _ SymbolInfo = (*ObjectSymbol)(nil)
var _ SymbolInfo = (*StaticSymbol)(nil)
var _ SymbolInfo = (*AnnotationSymbol)(nil)
var _ SymbolInfo = (*DecoratorSymbol)(nil)
var _ SymbolInfo = (*VariableDetails)(nil)
var _ SymbolInfo = (*InstanceSymbol)(nil)
var _ SymbolInfo = (*ReferenceSymbol)(nil)
var _ SymbolInfo = (*AddressSymbol)(nil)
var _ SymbolInfo = (*ThunkSymbol)(nil)
var _ SymbolInfo = (*DymanicRuntime)(nil)
var _ SymbolInfo = (*DirectivePragmaDetails)(nil)
var _ SymbolInfo = (*SymbolDetails)(nil)
var _ SymbolInfo = (*GenericDetails)(nil)
var _ SymbolInfo = (*OperatorDetails)(nil)
var _ SymbolInfo = (*SymbolDetails)(nil)
var _ SymbolInfo = (*Symbol)(nil)

// ============ COMPILE-TIME CHECKS FOR ITypeSymbol ============

var _ ITypeSymbol = (*EnumSymbol)(nil)
var _ ITypeSymbol = (*StructSymbol)(nil)
var _ ITypeSymbol = (*UnionSymbol)(nil)
var _ ITypeSymbol = (*TypeSymbol)(nil)
