package ast

import (
	"strings"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// SymbolTypeNode represents a named symbol type reference.
type SymbolTypeNode struct {
	Span
	NodeName   string
	Value      string
	SymbolType string
	Dapst      Stmt
	Symb       *symboltable.TypeSymbol
}

func (n SymbolTypeNode) GetName() string {
	return n.Symb.GetName()
}
func (n SymbolTypeNode) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}
func (t SymbolTypeNode) _type() {}

// GetSubType returns the sub-type classifier string.
func (t SymbolTypeNode) GetSubType() string { return "Type" }

// GetActType returns the actual type pair for this node.
func (n SymbolTypeNode) GetActType() (string, string) {
	return n.Value, n.SymbolType
}

// SetDap attaches directive annotations to the node.
func (b SymbolTypeNode) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (d SymbolTypeNode) isNonDependent() {}

// BuiltInDataType represents a built-in primitive data type.
type BuiltInDataType struct {
	Span
	NodeName   string
	Value      string
	Type       string
	SymbolType string
	Dapst      Stmt
	Symb       *symboltable.TypeSymbol
}

func (n BuiltInDataType) GetName() string {
	return n.Symb.GetName()
}
func (n BuiltInDataType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubType returns the sub-type classifier string.
func (t BuiltInDataType) GetSubType() string { return "BDT" }

// GetActType returns the actual type pair for this node.
func (n BuiltInDataType) GetActType() (string, string) {
	return n.Type, n.Value
}
func (d BuiltInDataType) isNonDependent() {}
func (t BuiltInDataType) _type()          {}

// SetDap attaches directive annotations to the node.
func (b BuiltInDataType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// CompoundType represents a compound type formed by combining two types with an operator.
type CompoundType struct {
	Span
	NodeName string
	Left     Type
	Op       string
	Right    Type
	Dapst    Stmt
	Symb     *symboltable.TypeSymbol
}

func (n CompoundType) GetName() string {
	return n.Symb.GetName()
}
func (n CompoundType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubType returns the sub-type classifier string.
func (t CompoundType) GetSubType() string { return "BDT" }

// GetActType returns the actual type pair for this node.
func (n CompoundType) GetActType() (string, string) {
	return "CDT", "CDT"
}

// SetDap attaches directive annotations to the node.
func (b CompoundType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t CompoundType) _type()          {}
func (d CompoundType) isNonDependent() {}

// ListType represents a list type wrapping an underlying element type.
type ListType struct {
	Span
	NodeName   string
	Underlying Type
	Dapst      Stmt
	Symb       *symboltable.TypeSymbol
}

func (n ListType) GetName() string {
	return n.Symb.GetName()
}
func (n ListType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubType returns the sub-type classifier string.
func (t ListType) GetSubType() string { return "BDT" }

// GetActType returns the actual type pair for this node.
func (n ListType) GetActType() (string, string) {
	return "list", "list"
}

// SetDap attaches directive annotations to the node.
func (b ListType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t ListType) _type()          {}
func (d ListType) isNonDependent() {}

// FunctionType represents a function signature type with parameters and results.
type FunctionType struct {
	Span
	NodeName string
	Symb     *symboltable.TypeSymbol
	Params   [][]Parameter
	Results  []Returns
	Dapst    Stmt
	Parent   FunctionDeclarationStmt
}

func (n FunctionType) GetName() string {
	return n.Symb.GetName()
}
func (n FunctionType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b FunctionType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// GetSubType returns the sub-type classifier string.
func (t FunctionType) GetSubType() string { return "FUN" }

// GetActType returns the actual type pair for this node.
func (n FunctionType) GetActType() (string, string) {
	actType := "["

	if helpers.HasElements(n.Params) {
		for _, row := range n.Params {

			for _, val := range row {
				_, sutyp := val.GetActType()

				actType = actType + "_" + sutyp

			}

		}
	} else {
		actType = actType + "_" + "co.lang.void"
	}
	if len(actType) > 0 && actType[0] == '_' {
		actType = actType[1:]
	}

	actType = actType + "::"
	if len(n.Results) > 0 {
		for _, ret := range n.Results {

			_, sutyp := ret.Type_.GetActType()
			actType = actType + "_" + sutyp

		}
	} else {
		actType = actType + "_" + "co.lang.void"
	}
	actType = actType + "]"
	actType = strings.ReplaceAll(actType, "::_", "::")
	actType = strings.ReplaceAll(actType, "[_", "[")
	return "co.lang.fun", actType
}
func (t FunctionType) _type()          {}
func (d FunctionType) isNonDependent() {}

// GenericType represents a generic type parameter with an optional constraint.
type GenericType struct {
	Span
	NodeName   string
	Type_      Type
	Constraint Type
	Dapst      Stmt
	Symb       *symboltable.TypeSymbol
}

func (n GenericType) GetName() string {
	return n.Symb.GetName()
}
func (n GenericType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubType returns the sub-type classifier string.
func (t GenericType) GetSubType() string { return "BDT" }

// GetActType returns the actual type pair for this node.
func (n GenericType) GetActType() (string, string) {
	return "Generic", "GDT"
}

// SetDap attaches directive annotations to the node.
func (b GenericType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t GenericType) _type()          {}
func (d GenericType) isNonDependent() {}

// ForAllType represents a rank-2 polymorphic type used in parameter position.
// Example: f forall(T) (T)->(T)
// The type params are locally scoped to the inner type.
type ForAllType struct {
	Span
	NodeName   string
	TypeParams []symboltable.GenericTypeParam
	Inner      Type
	Dapst      Stmt
	Symb       *symboltable.TypeSymbol
}

func (n ForAllType) GetName() string {
	return n.Symb.GetName()
}
func (n ForAllType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}
func (t ForAllType) _type() {}

// GetSubType returns the sub-type classifier string.
func (t ForAllType) GetSubType() string { return "FORALL" }

// GetActType returns the actual type pair for this node.
func (n ForAllType) GetActType() (string, string) {
	return "forall", "FORALL_TYPE"
}

// SetDap attaches directive annotations to the node.
func (b ForAllType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (d ForAllType) isNonDependent() {}

// DependentType represents something like x.type or singleton types.
type DependentType struct {
	Span
	NodeName string
	Base     NonDependentType // original non-dependent type, e.g. Int
	Expr     Expr             // expression this type depends on (e.g., a variable or literal)
	Dapst    Stmt
	Symb     *symboltable.TypeSymbol
}

func (n DependentType) GetName() string {
	return n.Symb.GetName()
}
func (n DependentType) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubType returns the sub-type classifier string.
func (t DependentType) GetSubType() string { return "BDT" }

// GetActType returns the actual type pair for this node.
func (n DependentType) GetActType() (string, string) {
	return "Dependet", "DDT"
}

// SetDap attaches directive annotations to the node.
func (b DependentType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t DependentType) _type() {}

// DerivationForm names the derivation a DerivedType applies to its element type.
//
// These are the derivation-specification alternatives of the grammar's section 4,
// spelled as a type is written: co.lang.int->(*), ->([5]), ->(&), ->(~), ->(@),
// ->(^), ->([:]) and ->(..).
type DerivationForm string

const (
	DerivePointer       DerivationForm = "pointer"       // ->(*), ->(**)
	DeriveArray         DerivationForm = "array"         // ->([5]), ->([2,3]), ->([2][3])
	DeriveReference     DerivationForm = "reference"     // ->(&), ->(&&)
	DeriveHeapReference DerivationForm = "heapReference" // ->(~)
	DeriveAddress       DerivationForm = "address"       // ->(@)
	DeriveThunk         DerivationForm = "thunk"         // ->(^)
	DeriveSlice         DerivationForm = "slice"         // ->([:])
	DeriveRange         DerivationForm = "range"         // ->(..)
	DeriveWord          DerivationForm = "word"          // ->(repr=intptr) and other bare attribute tails
)

// DerivedType is a type with a derivation applied to it.
//
// A DECLARATION records its derivation on the statement node it lowers to — a pointer
// variable becomes a PointerVariableDeclStmt — because the declaration nodes want the
// element type. Every OTHER position that admits a type has no such statement to carry
// it: a parameter, a result, a type alias and a function type's components are all
// plain ast.Type slots. Without this node the derivation was parsed and then dropped,
// so `f(p co.lang.int->(**))` reached the AST as an ordinary co.lang.int parameter and
// nothing downstream could tell a pointer from a value.
//
// Underlying is the element type. The remaining fields describe the derivation, and
// which of them are meaningful depends on Form.
type DerivedType struct {
	Span
	NodeName   string
	Underlying Type
	Form       DerivationForm

	// PointerCount is the star count of a pointer derivation: ->(*) is 1, ->(**) is 2.
	PointerCount int
	// RefCount is 1 for ->(&) and 2 for ->(&&).
	RefCount int

	// DimGroups holds EVERY array dimension group in source order, so a jagged
	// array keeps all of them: ->([2][3]) is {{2}, {3}}. A nil entry inside a group
	// is an elided dimension, which DECISION-TYP-003 permits, so ->([,]) is one
	// group of two nil entries.
	DimGroups [][]Expr
	// VariableLength records ->([...]) and ZeroDim records ->([.]).
	VariableLength bool
	ZeroDim        bool

	// Attrs is the derivation's trailing attribute list, which DECISION-TYP-001
	// allows on every form: ->(*, kind=region, meta={}) and ->(repr=intptr).
	Attrs map[string]any

	Dapst Stmt
	Symb  *symboltable.TypeSymbol
}

// Dims returns the first dimension group, which is the whole shape of a
// non-jagged array.
func (n DerivedType) Dims() []Expr {
	if len(n.DimGroups) == 0 {
		return nil
	}
	return n.DimGroups[0]
}

// IsJagged reports whether the array carries more than one dimension group.
func (n DerivedType) IsJagged() bool { return len(n.DimGroups) > 1 }

func (n DerivedType) GetName() string {
	if n.Symb != nil {
		return n.Symb.GetName()
	}
	return string(n.Form)
}

func (n DerivedType) GetSymbolType() string {
	if n.Symb != nil {
		return string(n.Symb.GetSymbolType())
	}
	return ""
}

// GetSubType returns the sub-type classifier string.
func (n DerivedType) GetSubType() string { return "DRV" }

// GetActType returns the actual type pair for this node.
//
// The element type's pair is reported so that a consumer keyed on the underlying type
// keeps working; Form is what distinguishes the derived type from it.
func (n DerivedType) GetActType() (string, string) {
	if n.Underlying == nil {
		return string(n.Form), string(n.Form)
	}
	return n.Underlying.GetActType()
}

// SetDap attaches directive annotations to the node.
func (b DerivedType) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

func (n DerivedType) Visit(t any) SET {
	node := t.(SET)
	return node
}

func (t DerivedType) _type() {}
