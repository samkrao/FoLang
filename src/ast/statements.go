package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// MatchCase is a match arm. The pattern may implement Binder; its bound names
// are visible only while evaluating Body.
type MatchCase struct {
	Span
	Pattern Pattern
	Body    FunctionBody
}

func (MatchCase) NodeKind() string { return "MatchCase" }

// EmptyStatement represents an intentionally empty statement.
type EmptyStatement struct {
	Span
}

func (EmptyStatement) NodeKind() string { return "EmptyStatement" }
func (EmptyStatement) stmt()            {}

// ExpressionStatement evaluates an expression for its effects and discards its
// result. A standalone call such as foo(); is represented this way.
type ExpressionStatement struct {
	Span
	Expression Expr
}

func (ExpressionStatement) NodeKind() string { return "ExpressionStatement" }
func (ExpressionStatement) stmt()            {}

// AssignmentStatement assigns the value of Value to Target.
type AssignmentStatement struct {
	Span
	Target Expr
	Value  Expr
}

func (AssignmentStatement) NodeKind() string { return "AssignmentStatement" }
func (AssignmentStatement) stmt()            {}

// CompoundAssignmentStatement represents operators such as += and ?=.
type CompoundAssignmentStatement struct {
	Span
	Target   Expr
	Operator string
	Value    Expr
}

func (CompoundAssignmentStatement) NodeKind() string { return "CompoundAssignmentStatement" }
func (CompoundAssignmentStatement) stmt()            {}

// ReturnStatement returns an optional value from the enclosing callable.
type ReturnStatement struct {
	Span
	Value Expr
}

func (ReturnStatement) NodeKind() string { return "ReturnStatement" }
func (ReturnStatement) stmt()            {}

// LoopStatement is the HIR form of condition.loop({ ... }). The surface syntax
// is a postfix call, but the reference defines loop as a control statement.
type LoopStatement struct {
	Span
	Condition Expr
	Body      FunctionBody
	Label     symboltable.SymbolID
}

func (LoopStatement) NodeKind() string { return "LoopStatement" }
func (LoopStatement) stmt()            {}

// ConditionalBranch is one branch of a then/otherwise/default selection.
type ConditionalBranch struct {
	Span
	Condition Expr
	Body      FunctionBody
	IsDefault bool
}

func (ConditionalBranch) NodeKind() string { return "ConditionalBranch" }

// SelectionStatement is the HIR form of a then/otherwise/default chain.
type SelectionStatement struct {
	Span
	Branches []ConditionalBranch
}

func (SelectionStatement) NodeKind() string { return "SelectionStatement" }
func (SelectionStatement) stmt()            {}

// BreakStatement exits the nearest or named loop.
type BreakStatement struct {
	Span
	Label symboltable.SymbolID
}

func (BreakStatement) NodeKind() string { return "BreakStatement" }
func (BreakStatement) stmt()            {}

// ContinueStatement advances the nearest or named loop.
type ContinueStatement struct {
	Span
	Label symboltable.SymbolID
}

func (ContinueStatement) NodeKind() string { return "ContinueStatement" }
func (ContinueStatement) stmt()            {}
