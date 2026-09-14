package symboltable

import (
	"fmt"

	"github.com/jinzhu/copier"
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

type ITypeSymbol interface {
	SymbolInfo
	IsType() bool
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
	Alias         bool
	NewType       bool
	SubType       bool
	SuperType     bool
	DependentType bool
	// RefinementType marks a co.lang.refinementType declaration: a base type
	// narrowed by a predicate over its candidate value.
	RefinementType bool
	// PredicateType marks a type-valued predicate over a dedicated immutable
	// co.lang.typevalue binder.
	PredicateType bool
	// AssociatedType marks a co.lang.associatedType component. ExplicitType
	// separates the two forms: a signature's requirement has no binding, while a
	// matching module's binding does.
	AssociatedType bool
	OpaqueType     bool
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
	ExplicitType   bool
	AsStmt         bool
	No_user_name   bool
	Ephimeral      bool
	Hidden         bool
	RtErased       bool
	IsGenericType  bool
	AccessorType   string //public, private, protected, package
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

// VariableDetails extends SymbolDetails for variable symbols with an internal flag.
type VariableDetails struct {
	SymbolDetails
	IsInternal_  bool
	Inferred     bool
	Dynamic      bool
	Auto         bool
	Adhoc        bool
	Mutable      bool
	Const        bool
	TypeInfo     TypeSymbol
	IsSealed     bool
	DuckType     bool
	HasInitValue bool
	LocalBinding bool //patternmatching and others where this is bind variable
	ExplicitType bool
	ActType_     string
	SubType_     string
	SubID        string
	VarType      string
	Type_        string
}

type MemberField struct {
	VariableDetails
	AccessorType string //public, private, protected, internal
	IsFriend     bool
}

type ParameterDetails struct {
	VariableDetails
	ParamType string
	Variadic  bool
	Optional  bool
	Named     bool
	Default   bool
}

type ReturnDetails struct {
	VariableDetails
	ReturnType string
}
