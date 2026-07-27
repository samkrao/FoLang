package ast

import (
	"strings"

	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// SymbolTypeNode represents a named symbol type reference.
type SymbolTypeNode struct {
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (d SymbolTypeNode) isNonDependent() {}

// BuiltInDataType represents a built-in primitive data type.
type BuiltInDataType struct {
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// CompoundType represents a compound type formed by combining two types with an operator.
type CompoundType struct {
	Left  Type
	Op    string
	Right Type
	Dapst Stmt
	Symb  *symboltable.TypeSymbol
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t CompoundType) _type()          {}
func (d CompoundType) isNonDependent() {}

// ListType represents a list type wrapping an underlying element type.
type ListType struct {
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t ListType) _type()          {}
func (d ListType) isNonDependent() {}

// FunctionType represents a function signature type with parameters and results.
type FunctionType struct {
	Symb    *symboltable.TypeSymbol
	Params  [][]Parameter
	Results []Returns
	Dapst   Stmt
	Parent  FunctionDeclarationStmt
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
		(&b).Dapst = &DirectveList{}
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t GenericType) _type()          {}
func (d GenericType) isNonDependent() {}

// ForAllType represents a rank-2 polymorphic type used in parameter position.
// Example: f forall(T) (T)->(T)
// The type params are locally scoped to the inner type.
type ForAllType struct {
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (d ForAllType) isNonDependent() {}

// DependentType represents something like x.type or singleton types.
type DependentType struct {
	Base  NonDependentType // original non-dependent type, e.g. Int
	Expr  Expr             // expression this type depends on (e.g., a variable or literal)
	Dapst Stmt
	Symb  *symboltable.TypeSymbol
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
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (t DependentType) _type() {}
