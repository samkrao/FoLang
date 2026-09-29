package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

type ExpresshonHeader struct {
	Kind_ NodeKind
}

// LiteralExpr refers to the canonical Literal symbol containing the lexeme and
// resolved literal type.
type LiteralExpr struct {
	Span
	ExpresshonHeader
	Symbol symboltable.SymbolID
}

func (LiteralExpr) NodeKind() string { return "LiteralExpr" }
func (LiteralExpr) expr()            {}

// TypeValueExpr lifts a resolved type symbol into value position.
type TypeValueExpr struct {
	Span
	ExpresshonHeader
	Type symboltable.SymbolID
}

func (TypeValueExpr) NodeKind() string { return "TypeValueExpr" }
func (TypeValueExpr) expr()            {}

// BindingExpr refers to $, $1, and related compiler bindings. Index -1 is the
// recursive binding; positive indexes select chained-call results.
type BindingExpr struct {
	Span
	ExpresshonHeader
	Symbol symboltable.SymbolID
	Index  int
}

func (BindingExpr) NodeKind() string { return "BindingExpr" }
func (BindingExpr) expr()            {}

type VariableAccessExpr struct {
	Span
	ExpresshonHeader
	Symbol symboltable.SymbolID
}

func (VariableAccessExpr) NodeKind() string { return "VariableAccessExpr" }
func (VariableAccessExpr) expr()            {}

type MemberAccessExpr struct {
	Span
	ExpresshonHeader
	Receiver Expr
	Member   symboltable.SymbolID
}

func (MemberAccessExpr) NodeKind() string { return "MemberAccessExpr" }
func (MemberAccessExpr) expr()            {}

type IndexExpr struct {
	Span
	ExpresshonHeader
	Receiver Expr
	Indices  []Expr
}

func (IndexExpr) NodeKind() string { return "IndexExpr" }
func (IndexExpr) expr()            {}

// CallArgument supports resolved named parameters and positional values. Value
// may be an expression, block, lambda, or WildcardArgument.
type CallArgument struct {
	Span
	ExpresshonHeader
	Parameter symboltable.SymbolID
	Value     SET
}

func (CallArgument) NodeKind() string { return "CallArgument" }

type CallExpr struct {
	Span
	ExpresshonHeader
	Receiver  Expr
	Function  symboltable.SymbolID
	Metadata  []MetadataRef
	Arguments []CallArgument
}

func (CallExpr) NodeKind() string { return "CallExpr" }
func (CallExpr) expr()            {}

type UnaryExpr struct {
	Span
	ExpresshonHeader
	Operator symboltable.SymbolID
	Operand  Expr
}

func (UnaryExpr) NodeKind() string { return "UnaryExpr" }
func (UnaryExpr) expr()            {}

type BinaryExpr struct {
	Span
	ExpresshonHeader
	Left     Expr
	Operator symboltable.SymbolID
	Right    Expr
}

func (BinaryExpr) NodeKind() string { return "BinaryExpr" }
func (BinaryExpr) expr()            {}

type AssignmentExpr struct {
	Span
	ExpresshonHeader
	Target Expr
	Value  Expr
}

func (AssignmentExpr) NodeKind() string { return "AssignmentExpr" }
func (AssignmentExpr) expr()            {}

type CompoundAssignmentExpr struct {
	Span
	ExpresshonHeader
	Target   Expr
	Operator symboltable.SymbolID
	Value    Expr
}

func (CompoundAssignmentExpr) NodeKind() string { return "CompoundAssignmentExpr" }
func (CompoundAssignmentExpr) expr()            {}

type TupleExpr struct {
	Span
	ExpresshonHeader
	Elements []Expr
}

func (TupleExpr) NodeKind() string { return "TupleExpr" }
func (TupleExpr) expr()            {}

type ConstructionElement struct {
	Span
	ExpresshonHeader
	Key   Expr
	Value Expr
}

func (ConstructionElement) NodeKind() string { return "ConstructionElement" }

// ConstructionExpr is the only aggregate construction form. Type must resolve
// to a concrete constructible type.
type ConstructionExpr struct {
	Span
	ExpresshonHeader
	Type     symboltable.SymbolID
	Elements []ConstructionElement
}

func (ConstructionExpr) NodeKind() string { return "ConstructionExpr" }
func (ConstructionExpr) expr()            {}

type MatchCase struct {
	Span
	ExpresshonHeader
	Pattern Pattern
	Guard   Expr
	Body    SET
}

func (MatchCase) NodeKind() string { return "MatchCase" }

type MatchExpr struct {
	Span
	ExpresshonHeader
	Subject Expr
	Matcher Expr
	Cases   []MatchCase
	Default SET
}

func (MatchExpr) NodeKind() string { return "MatchExpr" }
func (MatchExpr) expr()            {}

type AnonymousFunctionExpr struct {
	Span
	ExpresshonHeader
	Function symboltable.SymbolID
	Body     Block
}

func (AnonymousFunctionExpr) NodeKind() string { return "AnonymousFunctionExpr" }
func (AnonymousFunctionExpr) expr()            {}

type LambdaExpr struct {
	Span
	ExpresshonHeader
	Function symboltable.SymbolID
	Body     SET
}

func (LambdaExpr) NodeKind() string { return "LambdaExpr" }
func (LambdaExpr) expr()            {}

type AnonymousClassExpr struct {
	Span
	ExpresshonHeader
	Class symboltable.SymbolID
	Body  Block
}

func (AnonymousClassExpr) NodeKind() string { return "AnonymousClassExpr" }
func (AnonymousClassExpr) expr()            {}

type ComprehensionExpr struct {
	Span
	ExpresshonHeader
	Binding Pattern
	Source  Expr
	Results []Expr
}

func (ComprehensionExpr) NodeKind() string { return "ComprehensionExpr" }
func (ComprehensionExpr) expr()            {}

// WildcardArgument is deliberately not a general Expr. It is admitted only by
// call/comprehension positions that explicitly accept the wildcard token.
type WildcardArgument struct {
	ExpresshonHeader
	Span
}

func (WildcardArgument) NodeKind() string { return "WildcardArgument" }

// SelectionExpr is the normalized HIR for then/otherwise/default chains. Each
// branch uses Block so both block and value-producing source arms normalize to
// one representation.
type SelectionBranch struct {
	ExpresshonHeader
	Span
	Condition Expr
	Body      Block
}

func (SelectionBranch) NodeKind() string { return "SelectionBranch" }

type SelectionExpr struct {
	ExpresshonHeader
	Span
	Branches []SelectionBranch
	Default  *Block
}

func (SelectionExpr) NodeKind() string { return "SelectionExpr" }
func (SelectionExpr) expr()            {}
