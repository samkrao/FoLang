package ast

import (
	"strings"

	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// DummyStmt represents a no-op dummy statement.
type DummyStmt struct {
}

func (n DummyStmt) GetName() string       { return "DummyStmt" }
func (n DummyStmt) GetSymbolType() string { return "Dummy" }

// SetDap attaches directive annotations to the node.
func (b DummyStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

func (d DummyStmt) stmt() {}

// DefaultConditionalStmt represents the default (else) branch of a conditional.
type DefaultConditionalStmt struct {
	Stmt_        Stmt
	Default      bool
	Loop         bool
	ContainsLoop bool
	OnlyLoop     bool
	IsTernary    bool
	Expr_        []Expr
	Dapst        Stmt
	Symb         *symboltable.StatmentSymbol
}

// SetDap attaches directive annotations to the node.
func (b DefaultConditionalStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n DefaultConditionalStmt) getStmt() {}
func (n DefaultConditionalStmt) stmt()    {}
func (n DefaultConditionalStmt) GetName() string {
	return "DefaultConditionalStmt"
}
func (n DefaultConditionalStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// ConditionalStmt represents an if/elif/else conditional statement.
type ConditionalStmt struct {
	IfExpr          Expr
	IfStmt          Stmt
	ElifExprStmt    []ConditionalStmt
	ElseExprStmt    *DefaultConditionalStmt
	Loop            bool
	ContainsLoop    bool
	OnlyLoop        bool
	ISParentArrCont bool
	Dapst           Stmt
	Symb            *symboltable.StatmentSymbol
}

func (n ConditionalStmt) GetName() string {
	return "ConditionalStmt"
}
func (n ConditionalStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b ConditionalStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n ConditionalStmt) getStmt() {}
func (n ConditionalStmt) stmt()    {}

// TernaryStmt represents a ternary conditional statement.
type TernaryStmt struct {
	Expr_        Expr
	Stmt_        Stmt
	ElifExprStmt []TernaryStmt
	ElseExprStmt *DefaultConditionalStmt
	Symb         *symboltable.StatmentSymbol
}

func (n TernaryStmt) GetName() string {
	return "TernaryStmt"
}
func (n TernaryStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

func (t TernaryStmt) getStmt() {}
func (t TernaryStmt) stmt()    {}

// SetDap attaches directive annotations to the node.
func (b TernaryStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

// BreakStmt represents a loop break statement.
type BreakStmt struct {
	Args string
	Symb *symboltable.StatmentSymbol
}

func (n BreakStmt) GetName() string {
	return "BreakStmt"
}
func (n BreakStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}
func (t BreakStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b BreakStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

// ContinueStmt represents a loop continue statement.
type ContinueStmt struct {
	Args string
	Symb *symboltable.StatmentSymbol
}

func (n ContinueStmt) GetName() string {
	return "ContinueStmt"
}
func (n ContinueStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b ContinueStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

func (t ContinueStmt) stmt() {}

// ReturnStmt represents a function return statement.
type ReturnStmt struct {
	StmtExpr_    SET
	MultiReturns bool
	Symb         symboltable.SymbolInfo
}

func (n ReturnStmt) GetName() string {
	return "ReturnStmt"
}
func (n ReturnStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b ReturnStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

func (t ReturnStmt) stmt() {}

// Prog represents the top-level program node.
type Prog struct {
	Body      []Stmt
	Name      string
	Package   string
	CodeBlock bool
	GDapst    Stmt //pragmas
	Symb      *symboltable.StatmentSymbol
}

func (n Prog) GetName() string {
	return n.Name
}
func (n Prog) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b Prog) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.GDapst == nil {
		(&b).GDapst = &DirectveList{}
	}
	b.GDapst.(*DirectveList).SetDap(daps)
}

// GetBlockType returns the block type of this statement.
func (b Prog) stmt() {}

// CodeStmt represents a block of code statements.
type CodeStmt struct {
	Body  []Stmt
	DDaps Stmt //import directives
	Symb  *symboltable.StatmentSymbol
}

func (n CodeStmt) GetName() string {
	return "CodeStmt"
}
func (n CodeStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b CodeStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.DDaps == nil {
		(&b).DDaps = &DirectveList{}
	}
	b.DDaps.(*DirectveList).SetDap(daps)
}

type Application struct {
	Body   []Stmt
	IDapst Stmt
	PDapst Stmt
	ODapst Stmt
	Symb   *symboltable.ApplicationSymbol
}

func (n Application) GetName() string {
	return n.Symb.Name_
}
func (n Application) GetSymbolType() string {
	return string(symboltable.S_PackageSymbol)
}

// SetDap attaches directive annotations to the node.
func (b Application) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.IDapst == nil {
		(&b).IDapst = &DirectveList{}
	}
	if b.ODapst == nil {
		(&b).ODapst = &DirectveList{}
	}
	idapa := daps[scanlex.DIRECTIVE]
	var first Stmt
	rest := idapa
	parent := false
	if len(idapa) > 0 {
		tf := idapa[0]
		if tf.(DirectiveStmt).Name == "@co.ddap.parent" {
			first = tf
			parent = true
		}

	}
	othD := []Stmt{}
	impD := []Stmt{}
	if parent && len(idapa) > 1 {
		rest = idapa[1:]
	}

	for _, sto := range rest {
		if stoo, ok := sto.(DirectiveStmt); ok {
			if stoo.Name == "@co.ddap.import" {
				impD = append(impD, stoo)
			} else {
				othD = append(othD, stoo)
			}
		}
	}
	daps[scanlex.DIRECTIVE] = impD
	b.PDapst = first
	otherD := map[scanlex.DirectiveKind][]Stmt{scanlex.DIRECTIVE: othD}
	b.ODapst.(*DirectveList).SetDap(otherD)
	b.IDapst.(*DirectveList).SetDap(daps)
}

// GetBlockType returns the block type of this statement.
func (p Application) stmt() {}

type Library struct {
	Body   []Stmt
	IDapst Stmt
	PDapst Stmt
	ODapst Stmt
	Symb   *symboltable.LibrarySymbol
}

func (n Library) GetName() string {
	return n.Symb.Name_
}
func (n Library) GetSymbolType() string {
	return string(symboltable.S_PackageSymbol)
}

// SetDap attaches directive annotations to the node.
func (b Library) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.IDapst == nil {
		(&b).IDapst = &DirectveList{}
	}
	if b.ODapst == nil {
		(&b).ODapst = &DirectveList{}
	}
	idapa := daps[scanlex.DIRECTIVE]
	var first Stmt
	rest := idapa
	parent := false
	if len(idapa) > 0 {
		tf := idapa[0]
		if tf.(DirectiveStmt).Name == "@co.ddap.parent" {
			first = tf
			parent = true
		}

	}
	othD := []Stmt{}
	impD := []Stmt{}
	if parent && len(idapa) > 1 {
		rest = idapa[1:]
	}

	for _, sto := range rest {
		if stoo, ok := sto.(DirectiveStmt); ok {
			if stoo.Name == "@co.ddap.import" {
				impD = append(impD, stoo)
			} else {
				othD = append(othD, stoo)
			}
		}
	}
	daps[scanlex.DIRECTIVE] = impD
	b.PDapst = first
	otherD := map[scanlex.DirectiveKind][]Stmt{scanlex.DIRECTIVE: othD}
	b.ODapst.(*DirectveList).SetDap(otherD)
	b.IDapst.(*DirectveList).SetDap(daps)
}

// GetBlockType returns the block type of this statement.
func (p Library) stmt() {}

// PackageStmt represents a package declaration statement.
type PackageStmt struct {
	Body   []Stmt
	IDapst Stmt
	PDapst Stmt
	ODapst Stmt
	Symb   *symboltable.PackageSymbol
}

func (n PackageStmt) GetName() string {
	return n.Symb.Name_
}
func (n PackageStmt) GetSymbolType() string {
	return string(symboltable.S_PackageSymbol)
}

// SetDap attaches directive annotations to the node.
func (b PackageStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.IDapst == nil {
		(&b).IDapst = &DirectveList{}
	}
	if b.ODapst == nil {
		(&b).ODapst = &DirectveList{}
	}
	idapa := daps[scanlex.DIRECTIVE]
	var first Stmt
	rest := idapa
	parent := false
	if len(idapa) > 0 {
		tf := idapa[0]
		if tf.(DirectiveStmt).Name == "@co.ddap.parent" {
			first = tf
			parent = true
		}

	}
	othD := []Stmt{}
	impD := []Stmt{}
	if parent && len(idapa) > 1 {
		rest = idapa[1:]
	}

	for _, sto := range rest {
		if stoo, ok := sto.(DirectiveStmt); ok {
			if stoo.Name == "@co.ddap.import" {
				impD = append(impD, stoo)
			} else {
				othD = append(othD, stoo)
			}
		}
	}
	daps[scanlex.DIRECTIVE] = impD
	b.PDapst = first
	otherD := map[scanlex.DirectiveKind][]Stmt{scanlex.DIRECTIVE: othD}
	b.ODapst.(*DirectveList).SetDap(otherD)
	b.IDapst.(*DirectveList).SetDap(daps)
}

// GetBlockType returns the block type of this statement.
func (p PackageStmt) stmt() {}

func (b CodeStmt) stmt() {}

// ModuleStmt represents a module declaration statement.
type ModuleStmt struct {
	Body          []Stmt
	Extensions    []string
	Uses          []string
	MatchesSig    string // signature name from ->(matches=sigName)
	SignatureName string // signature name from ->(signature=X) or @co.dap.module(signature=X)
	SDapst        Stmt
	Symb          *symboltable.ModuleSymbol
}

func (n ModuleStmt) GetName() string {
	return n.Symb.Name_
}
func (n ModuleStmt) GetSymbolType() string {
	return string(symboltable.S_ModuleSymbol)
}
func (m ModuleStmt) stmt() {

}

// SetDap attaches directive annotations to the node.
func (b ModuleStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// BlockStmt represents a named or anonymous block of statements.
type BlockStmt struct {
	Body  []Stmt
	Dapst Stmt
	Symb  *symboltable.BlockSymbol
}

func (n BlockStmt) GetName() string {
	return n.Symb.Name_
}
func (n BlockStmt) GetSymbolType() string {
	return string(symboltable.S_BlockSymbol)
}

// SetDap attaches directive annotations to the node.
func (b BlockStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

func (b BlockStmt) stmt() {}

// ContainsStmt represents a containment relationship for a variable in a type.
type ContainsStmt struct {
	VarName        string
	VarType        string
	VarType_       string
	VarSubType     string
	VarActType     string
	Accessor       Stmt
	AdditionalInfo any
	Method         string
	Symb           *symboltable.StatmentSymbol
}

func (n ContainsStmt) GetName() string {
	return n.Symb.Name_
}
func (n ContainsStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b ContainsStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

// BuiltInStmt represents a built-in language construct statement.
type BuiltInStmt struct {
	Body  []Stmt
	Value string
	Dapst Stmt
	Symb  *symboltable.StatmentSymbol
}

func (n BuiltInStmt) GetName() string {
	return n.Symb.Name_
}
func (n BuiltInStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b BuiltInStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (b BuiltInStmt) stmt() {}

// BuiltInConstantStmt represents a built-in constant reference statement.
type BuiltInConstantStmt struct {
	Identifier SymbolExpr
	Type_      string
	Dapst      Stmt
	Symb       *symboltable.StatmentSymbol
}

func (n BuiltInConstantStmt) GetName() string {
	return n.Symb.Name_
}
func (n BuiltInConstantStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

func (n BuiltInConstantStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b BuiltInConstantStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// VarAccessStmt represents a variable access statement.
type SymbolRefExpr struct {
	Identifier       SymbolExpr
	ExprType         string // type of variable Bool,GEN, TERNARY, etc
	SymbolKind_      string // function, variable etc
	MetaNode         Stmt   //annotations and/or decorators
	ResolutionState  string
	ResolutionPolicy string
	/*
		     *  lexical_static_ordered,
			 *  lexical_static_complete_container
			 *  late_lexical_call_site
			 *  late_lexical_formation_site
			 *  macro_definition_site
			 *  macro_expansion_site
			 *  runtime_bound
			 *  dynamic_call_site
	*/
	AdditionalInfo symboltable.SymbolInfo
	Symb           *symboltable.Symbol
}

func (n SymbolRefExpr) GetName() string {
	return n.Symb.Name_
}
func (n SymbolRefExpr) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b SymbolRefExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.MetaNode == nil {
		(&b).MetaNode = &DirectveList{}
	}
	b.MetaNode.(*DirectveList).SetDap(daps)
}
func (n SymbolRefExpr) stmt()  {}
func (n SymbolRefExpr) expr()  {}
func (t SymbolRefExpr) _type() {}

// SymbolDeclStmt is the interface for symbol declaration statements.
type SymbolDeclStmt interface {
	SET
	IsInitialize() bool
	SetInner(bool)
	GetName() string
	GetActType() (string, string)
	GetSubType() string
	GetSubId() string
	IsThunk() bool
	IsOptional() bool
}

//Basic VarStatment

type BasicVarStmt struct {
	Identifier    string //is Name_ in varsymbol needs to assign
	AssignedValue Expr
	Type_         Type
	VarType       string
	SDapst        Stmt
}

// VarDeclarationStmt represents a variable declaration statement.
type VarDeclarationStmt struct {
	BasicVarStmt
	Symb *symboltable.VarSymbol
}

func (n VarDeclarationStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetActType returns the actual type pair for this node.
func (n VarDeclarationStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.VarType
}

// GetSubType returns the sub-type classifier string.
func (n VarDeclarationStmt) GetSubType() string {
	return n.Symb.SubType_
}

// GetSubId returns the sub-identifier string.
func (n VarDeclarationStmt) GetSubId() string {
	return n.Symb.SubID
}

// IsInitialize reports whether the variable has an initial value.
func (n VarDeclarationStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}

// IsOptional reports whether the parameter or variable is optional.
func (n VarDeclarationStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n VarDeclarationStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// GetName returns the name of the node.
func (n VarDeclarationStmt) GetName() string {
	return n.Symb.GetName()
}

// SetInner marks the node as an inner (nested) declaration.
func (n VarDeclarationStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}
func (n VarDeclarationStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b VarDeclarationStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// ArrayVariableDeclStmt represents an array variable declaration.
type ArrayVariableDeclStmt struct {
	BasicVarStmt
	Dimensions int
	Sizes      []Expr
	Symb       *symboltable.ArraySymbol
}

func (n ArrayVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n ArrayVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n ArrayVariableDeclStmt) GetActType() (string, string) {
	return n.Type_.GetActType()
}

// GetSubType returns the sub-type classifier string.
func (n ArrayVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n ArrayVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}

// GetName returns the name of the node.
func (n ArrayVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ArrayVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n ArrayVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b ArrayVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n ArrayVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n ArrayVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// PointerVariableDeclStmt represents a pointer variable declaration.
type PointerVariableDeclStmt struct {
	BasicVarStmt
	Kind_ string
	Symb  *symboltable.PointerSymbol
}

func (n PointerVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n PointerVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n PointerVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n PointerVariableDeclStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n PointerVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n PointerVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n PointerVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n PointerVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b PointerVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n PointerVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n PointerVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// RefVariableDeclStmt represents a reference variable declaration.
type RefVariableDeclStmt struct {
	BasicVarStmt
	Symb *symboltable.ReferenceSymbol
}

func (n RefVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n RefVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n RefVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n RefVariableDeclStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n RefVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n RefVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n RefVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n RefVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b RefVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n RefVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n RefVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// AddressVariableDeclStmt represents an address-of variable declaration.
type AddressVariableDeclStmt struct {
	BasicVarStmt
	Symb *symboltable.AddressSymbol
}

func (n AddressVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n AddressVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n AddressVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n AddressVariableDeclStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n AddressVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n AddressVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n AddressVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n AddressVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b AddressVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n AddressVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n AddressVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// ThunkVariableDeclStmt represents a thunk (lazy) variable declaration.
type ThunkVariableDeclStmt struct {
	BasicVarStmt
	Symb *symboltable.ThunkSymbol
}

func (n ThunkVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ThunkVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n ThunkVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n ThunkVariableDeclStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n ThunkVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n ThunkVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n ThunkVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n ThunkVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b ThunkVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n ThunkVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n ThunkVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// HeapAllocatedRefStmt represents a heap-allocated reference variable declaration.
type HeapAllocatedRefStmt struct {
	BasicVarStmt
	Symb *symboltable.ReferenceSymbol
}

func (n HeapAllocatedRefStmt) GetName() string {
	return n.Symb.GetName()
}
func (n HeapAllocatedRefStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n HeapAllocatedRefStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n HeapAllocatedRefStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n HeapAllocatedRefStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n HeapAllocatedRefStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n HeapAllocatedRefStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n HeapAllocatedRefStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b HeapAllocatedRefStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n HeapAllocatedRefStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n HeapAllocatedRefStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// SliceVariableDeclStmt represents a slice variable declaration.
type SliceVariableDeclStmt struct {
	BasicVarStmt
	Symb *symboltable.ArraySymbol
}

func (n SliceVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n SliceVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n SliceVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n SliceVariableDeclStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n SliceVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n SliceVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n SliceVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n SliceVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b SliceVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n SliceVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n SliceVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// RangeVariableDeclStmt represents a range variable declaration.
type RangeVariableDeclStmt struct {
	BasicVarStmt
	Symb *symboltable.RangeSymbol
}

func (n RangeVariableDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n RangeVariableDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n RangeVariableDeclStmt) GetSubId() string {
	return n.Symb.SubID
}

// GetActType returns the actual type pair for this node.
func (n RangeVariableDeclStmt) GetActType() (string, string) {
	return n.Symb.ActType_, n.Symb.ActType_
}

// GetSubType returns the sub-type classifier string.
func (n RangeVariableDeclStmt) GetSubType() string {
	return n.Symb.SubType_
}

// IsInitialize reports whether the variable has an initial value.
func (n RangeVariableDeclStmt) IsInitialize() bool {
	return n.Symb.HasInitValue
}
func (n RangeVariableDeclStmt) stmt() {}

// SetInner marks the node as an inner (nested) declaration.
func (n RangeVariableDeclStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

// SetDap attaches directive annotations to the node.
func (b RangeVariableDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n RangeVariableDeclStmt) IsOptional() bool {
	return n.Symb.Optional
}

// IsThunk reports whether the variable is lazily evaluated.
func (n RangeVariableDeclStmt) IsThunk() bool {
	return n.Symb.ThunkVar
}

// TypeStmt wraps a Type node as a statement.
type TypeStmt struct {
	Type_     Type
	ContextId string
	Symb      *symboltable.TypeSymbol
}

func (n TypeStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TypeStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b TypeStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

func (n TypeStmt) stmt() {}

// ExpressionStmt wraps an expression as a statement.
type ExpressionStmt struct {
	Expression Expr
	Symb       *symboltable.StatmentSymbol
}

func (n ExpressionStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ExpressionStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b ExpressionStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

func (n ExpressionStmt) stmt() {}

// Parameter represents a function parameter declaration.
type Parameter struct {
	SymbolDeclStmt
	Scope        string
	VarArgs      bool
	Optional     bool
	Argument_    *Argument
	Default      Expr
	HasDefault   bool
	OnlyType     bool
	Type_        Type
	Name_        string
	UseVarDecl   bool
	MappedParam  string
	WhatType     string
	SDapst       Stmt
	DefaultArgs  bool
	NamedArgs    bool
	OptionalArgs bool
	ThunkArgs    bool
	Variadic     bool
	Symb         *symboltable.Symbol
}

func (n Parameter) GetName() string {
	return n.Symb.GetName()
}
func (n Parameter) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b Parameter) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

func (n Parameter) stmt() {}

// Argument represents a function call argument.
type Argument struct {
	Value    Expr
	VarArgs  bool
	Optional bool
	SDapst   Stmt
}

func (n Argument) stmt() {}

// SetDap attaches directive annotations to the node.
func (b Argument) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// Returns represents a function return type declaration.
type Returns struct {
	SymbolDeclStmt
	Value    Expr
	IsNamed  bool
	Type_    Type
	OnlyType bool
	WhatType string
	SDapst   Stmt
	Symb     symboltable.SymbolInfo
}

func (n Returns) GetName() string {
	return n.Symb.GetName()
}
func (n Returns) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b Returns) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

func (n Returns) stmt() {}

// DelegateStmt represents a delegate (function pointer) declaration.
type DelegateStmt struct {
	SDapst Stmt
	Type_  Stmt
	Symb   *symboltable.DelegateSymbol
}

func (n DelegateStmt) GetName() string {
	return n.Symb.GetName()
}
func (n DelegateStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n DelegateStmt) GetSubId() string {
	return "DELEGATE"
}

// GetActType returns the actual type pair for this node.
func (n DelegateStmt) GetActType() (string, string) {
	return "DELEGATE", "DELEGATE"
}

// GetSubType returns the sub-type classifier string.
func (n DelegateStmt) GetSubType() string {
	return "DELEGATE"
}

// IsInitialize reports whether the variable has an initial value.
func (n DelegateStmt) IsInitialize() bool {
	return false
}

// SetInner marks the node as an inner (nested) declaration.
func (n DelegateStmt) SetInner(b bool) {

}

func (n DelegateStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b DelegateStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n DelegateStmt) IsOptional() bool {
	return false
}

// IsThunk reports whether the variable is lazily evaluated.
func (n DelegateStmt) IsThunk() bool {
	return false
}

// FunctionReceiver represents the receiver parameter of a method.
type FunctionReceiver struct {
	SymbolStmt SymbolDeclStmt
	What       VariableType
}

// SetDap attaches directive annotations to the node.
func (b FunctionReceiver) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

func (t FunctionReceiver) stmt() {}

// FunctionDeclarationStmt represents a function or method declaration.
type FunctionDeclarationStmt struct {
	Parameters         [][]Parameter
	Name               string
	Body               []Stmt `json:"-"`
	ReturnType         []Returns
	Scope              string
	Parent             *FunctionDeclarationStmt
	ParamNames         string
	ResultNames        string
	IntOverLdParam     string
	IntOverLdParamName string
	AssociatedReceiver *FunctionReceiver
	Dapst              Stmt
	Symb               *symboltable.FunctionSymbol
	AsExpr             bool
	WhatisIt           []string
}

func (n FunctionDeclarationStmt) GetName() string {
	return n.Symb.GetName()
}
func (n FunctionDeclarationStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GetSubId returns the sub-identifier string.
func (n FunctionDeclarationStmt) GetSubId() string {
	return "FUNMETH"
}

// IsRestrictedToOverload reports whether the function is overload-only.
func (p FunctionDeclarationStmt) IsRestrictedToOverload() bool {
	return p.Symb.RestrictedToOverload
}

// GetActType returns the actual type pair for this node.
func (n FunctionDeclarationStmt) GetActType() (string, string) {
	actType := "["
	if helpers.HasElements(n.Parameters) {
		for _, row := range n.Parameters {

			for _, val := range row {
				sutyp, _ := val.GetActType()
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
	if len(n.ReturnType) > 0 {
		for _, ret := range n.ReturnType {
			sutyp := ""

			if ret.OnlyType {
				_, sutyp = ret.Type_.GetActType()
			} else {
				_, sutyp = ret.GetActType()
			}
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

// GetSubType returns the sub-type classifier string.
func (n FunctionDeclarationStmt) GetSubType() string {
	return "FUN"
}

// IsInitialize reports whether the variable has an initial value.
func (n FunctionDeclarationStmt) IsInitialize() bool {
	return false
}

// SetInner marks the node as an inner (nested) declaration.
func (n FunctionDeclarationStmt) SetInner(b bool) {
	(&n).Symb.IsInner = b
}

func (n FunctionDeclarationStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b FunctionDeclarationStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// IsOptional reports whether the parameter or variable is optional.
func (n FunctionDeclarationStmt) IsOptional() bool {
	return false
}

// IsThunk reports whether the variable is lazily evaluated.
func (n FunctionDeclarationStmt) IsThunk() bool {
	return false
}

// IfStmt represents a simple if/else statement.
type IfStmt struct {
	Condition  Expr
	Consequent Stmt
	Alternate  Stmt
	Symb       *symboltable.StatmentSymbol
}

func (n IfStmt) GetName() string {
	return n.Symb.GetName()
}
func (n IfStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// SetDap attaches directive annotations to the node.
func (b IfStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}
func (n IfStmt) stmt() {}

// ImportStmt represents an import declaration.
type ImportStmt struct {
	Name        string // as
	From        string // library
	Package     string // package
	Parent      string // parent — composite "as.package" for nesting
	Realm       string // realm
	ParentRealm string // parent-realm
	SrcLibrary  bool   // src-library
	Expect      string // expect

	Symb *symboltable.DirectivePragmaDetails
}

func (n ImportStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ImportStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (n ImportStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b ImportStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

// DirectiveStmt represents a compiler directive statement.
type DirectiveStmt struct {
	Name            string
	Parameters      map[string]any
	DirectiveType   string
	DirectiveKind_  string
	DirectiveScope_ string
	Symb            *symboltable.DirectivePragmaDetails
}

func (n DirectiveStmt) GetName() string {
	return n.Symb.GetName()
}
func (n DirectiveStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (n DirectiveStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b DirectiveStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

// ForeachStmt represents a foreach iteration statement.
type ForeachStmt struct {
	VarName        string
	AccessorKeyIdx string
	Accessor       string
	Body           Stmt
	VarDetails     symboltable.SymbolDetails
	Method         string
	Dapst          Stmt
	Symb           *symboltable.StatmentSymbol
}

func (n ForeachStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ForeachStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}
func (n ForeachStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b ForeachStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// TypeDeclarationStmt represents a type declaration statement.
type TypeDeclarationStmt struct {
	Name          string
	Body          []Stmt
	Extensions    []string
	SubType_      string
	Type_         Type
	TypeExpr      TypeExpr
	Typetype      string
	Kind          string
	NewTypeName   string
	ADT_          string
	SDapst        Stmt
	KDapst        Stmt
	DependentKind string // e.g. "length" from co.lang.dependentType->(kind=length)
	ObjectFor     string // "annotation", "directive", "pragma" — from co.lang.object->(for=...)
	Symb          symboltable.ITypeSymbol
}

func (n TypeDeclarationStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TypeDeclarationStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b TypeDeclarationStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

func (n TypeDeclarationStmt) stmt() {}

type ObjectDeclStmt struct {
	Name      string
	Body      []Stmt
	Kind      string
	SDapst    Stmt
	KDapst    Stmt
	ObjectFor string // "annotation", "directive", "pragma" — from co.lang.object->(for=...)
	Symb      *symboltable.ObjectSymbol
}

func (n ObjectDeclStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ObjectDeclStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b ObjectDeclStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

func (n ObjectDeclStmt) stmt() {}

// ForAllStmt wraps a generic declaration (struct, class, function, var) introduced
// with the `forall` keyword.
//
//	forall(T) LinkedList co.lang.struct = { value T; next LinkedList; }
//	forall(T: Orderable) sort(list T->([...]))->(T->([...])) = {}
type ForAllStmt struct {
	TypeParams []symboltable.GenericTypeParam
	Body       Stmt
	Symb       *symboltable.GenericDetails
}

func (n ForAllStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ForAllStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (n ForAllStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b ForAllStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Body != nil {
		b.Body.SetDap(daps)
	}
}

// ClassDeclarationStmt represents a class declaration statement.
type ClassDeclarationStmt struct {
	Name       string
	Body       []Stmt
	TypeParams []symboltable.GenericTypeParam
	SDapst     Stmt
	Symb       *symboltable.ClassSymbol
}

func (n ClassDeclarationStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ClassDeclarationStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (n ClassDeclarationStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b ClassDeclarationStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// AST Node Types
type ArrowFunction struct {
	Params []Parameter
	Body   SET
	Dapst  Stmt
	Symb   *symboltable.FunctionSymbol
}

func (n ArrowFunction) GetName() string {
	return n.Symb.GetName()
}
func (n ArrowFunction) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (b ArrowFunction) stmt() {}

// SetDap attaches directive annotations to the node.
func (b ArrowFunction) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// MacroStmt represents a macro function declaration.
type MacroStmt struct {
	FunctionDeclarationStmt
	Type_        string
	IsExportable bool
}

func (n MacroStmt) GetName() string {
	return n.Symb.GetName()
}
func (n MacroStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// TemplateStmt represents a template function declaration.
type TemplateStmt struct {
	FunctionDeclarationStmt
	Type_ string
}

func (n TemplateStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TemplateStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// OperatorStmt represents an operator function declaration.
type OperatorStmt struct {
	FunctionDeclarationStmt
	Type_ string
}

func (n OperatorStmt) GetName() string {
	return n.Symb.GetName()
}
func (n OperatorStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// GenerricFun represents a generic function declaration with type parameters.
type GenerricFun struct {
	FunctionDeclarationStmt
	Type_ string

	Generic symboltable.GenericDetails
}

func (n GenerricFun) GetName() string {
	return n.Symb.GetName()
}
func (n GenerricFun) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// DDapStmt represents a directive/annotation/pragma/decorator declaration.
type DDapStmt struct {
	FunctionDeclarationStmt
	Type_ string
}

func (n DDapStmt) GetName() string {
	return n.Symb.GetName()
}
func (n DDapStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// IndexerStmt represents an indexer function declaration.
type IndexerStmt struct {
	FunctionDeclarationStmt
	Type_ string
}

func (n IndexerStmt) GetName() string {
	return n.Symb.GetName()
}
func (n IndexerStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// ExtensionStmt represents an extension method declaration.
type ExtensionStmt struct {
	FunctionDeclarationStmt
	Type_   string // unused legacy field kept for backward compatibility
	ForType string // target type from @co.dap.extension(fortype=...)
	What    string // relationship from @co.dap.extension(what=extends|overrides)
}

func (n ExtensionStmt) GetName() string {
	return n.Symb.GetName()
}
func (n ExtensionStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// MatcherStmt wraps a custom pattern-matcher declaration (@co.dap.matcher).
type MatcherStmt struct {
	FunctionDeclarationStmt
	Type_ string
}

func (n MatcherStmt) GetName() string {
	return n.Symb.GetName()
}
func (n MatcherStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// CaseStmt represents a single case arm in a pattern match.
type CaseStmt struct {
	Stmt_   Stmt
	Expr_   Expr
	Default bool
	Binding string // optional binding name (e.g. "n" in `case(n: n > 10 => ...)`)
	Symb    *symboltable.StatmentSymbol
}

func (n CaseStmt) GetName() string {
	return n.Symb.GetName()
}
func (n CaseStmt) GetSymbolType() string {
	return string(symboltable.S_StatmentSymbol)
}

// FunctionPatternStmt represents one pattern arm of a pattern-matched function.
//
// Syntax forms:
//   - f (pattern) => { body }   — regular function pattern
//   - let f(pattern) = expr     — let-style function pattern (IsLetForm=true)
//
// Multiple arms for the same function name are left for the compiler to merge
// into a single function with a match expression in its body.
type FunctionPatternStmt struct {
	Name        string
	PatternArgs []Expr // pattern expressions (one per argument position)
	Body        []Stmt // block body (used when IsLetForm=false)
	BodyExpr    Expr   // expression body (used when IsLetForm=true)
	IsLetForm   bool   // true for `let f(p) = expr` form
	Symb        *symboltable.FunctionPattern
}

func (n FunctionPatternStmt) GetName() string {
	return n.Symb.GetName()
}
func (n FunctionPatternStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (t CaseStmt) getStmt() {}
func (t CaseStmt) stmt()    {}

// SetDap attaches directive annotations to the node.
func (b CaseStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {}

func (n FunctionPatternStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b FunctionPatternStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {}

// MatchExprStmt represents a match expression statement.
type MatchExprStmt struct {
	Stmt_   Stmt
	Expr_   Expr
	Name    string
	Default bool
	Symb    *symboltable.MatcherSymbol
}

func (n MatchExprStmt) GetName() string {
	return n.Symb.GetName()
}
func (n MatchExprStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (t MatchExprStmt) getStmt() {}
func (t MatchExprStmt) stmt()    {}

// SetDap attaches directive annotations to the node.
func (b MatchExprStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {}

// PatternExprStmt represents a pattern matching expression with case arms.
type PatternExprStmt struct {
	Expr_           MatchExprStmt
	Stmt_           Stmt
	CaseExprStmt    []CaseStmt
	DefaultExprStmt *CaseStmt
	CustomMatcher   bool
	Matcher         bool
	MatcherType     string
	Symb            *symboltable.MatcherImplSymbol
}

func (n PatternExprStmt) GetName() string {
	return n.Symb.GetName()
}
func (n PatternExprStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (t PatternExprStmt) getStmt() {}
func (t PatternExprStmt) stmt()    {}

// SetDap attaches directive annotations to the node.
func (b PatternExprStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}

// VariantConstructor is a single constructor arm of a type constructor declaration.
// E.g., Some(T) or None() in: Option(T) co.lang.type = Some(T) | None()
type VariantConstructor struct {
	Name     string
	TypeArgs []string // type parameter names carried by this variant (e.g. ["T"])
	Symb     *symboltable.VariantConstructor
}

func (n VariantConstructor) GetName() string {
	return n.Symb.GetName()
}
func (n VariantConstructor) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// TypeConstructorStmt represents a higher-kinded / algebraic data type constructor.
//
// Syntax (decorated with @co.dap.hokrt):
//
//	@co.dap.hokrt
//	Option(T) co.lang.type = Some(T) | None();
type TypeConstructorStmt struct {
	Name       string
	TypeParams []string // e.g. ["T"]
	Variants   []VariantConstructor
	SDapst     Stmt
	Symb       *symboltable.TypeConstructor
}

func (n TypeConstructorStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TypeConstructorStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (n TypeConstructorStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b TypeConstructorStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// UseStmtDirective represents a use-directive statement for trait/mixin usage.
type UseStmtDirective struct {
	Name   string
	SDapst Stmt
	Type   map[string][]string
	Symb   *symboltable.UseSymbol
}

func (n UseStmtDirective) GetName() string {
	return n.Symb.GetName()
}
func (n UseStmtDirective) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (n UseStmtDirective) stmt() {}

// SetDap attaches directive annotations to the node.
func (b UseStmtDirective) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

type TypeComposeStmt struct {
	Symb     *symboltable.TypeSymbol
	SDapst   Stmt
	TypeName string
	Type_    *symboltable.TypeSymbol
}

func (n TypeComposeStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TypeComposeStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b TypeComposeStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

func (n TypeComposeStmt) stmt() {}
