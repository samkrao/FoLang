package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

type StatementHeader struct {
	Kind_ NodeKind
}
type EmptyStatement struct {
	Span
	StatementHeader
}

func (EmptyStatement) NodeKind() string { return "EmptyStatement" }
func (EmptyStatement) stmt()            {}

type ExpressionStatement struct {
	Span
	StatementHeader
	Expression Expr
}

func (ExpressionStatement) NodeKind() string { return "ExpressionStatement" }
func (ExpressionStatement) stmt()            {}

// MultipleAssignmentStatement is the structural multi-target assignment form.
// Ordinary and compound assignments are expressions wrapped by
// ExpressionStatement when used as statements.
type MultipleAssignmentStatement struct {
	Span
	StatementHeader
	Targets []Expr
	Values  []Expr
}

func (MultipleAssignmentStatement) NodeKind() string { return "MultipleAssignmentStatement" }
func (MultipleAssignmentStatement) stmt()            {}

type ReturnStatement struct {
	Span
	StatementHeader
	Values []Expr
}

func (ReturnStatement) NodeKind() string { return "ReturnStatement" }
func (ReturnStatement) stmt()            {}

// EnclosingCallableReturnStatement represents this ^=> from an anonymous call
// argument block. It is intentionally distinct from an ordinary return.
type EnclosingCallableReturnStatement struct {
	Span
	StatementHeader
	Values []Expr
}

func (EnclosingCallableReturnStatement) NodeKind() string {
	return "EnclosingCallableReturnStatement"
}
func (EnclosingCallableReturnStatement) stmt() {}

// LoopStatement is the normalized HIR for condition.loop(block).
type LoopStatement struct {
	Span
	StatementHeader
	Condition Expr
	Body      Block
	Label     symboltable.SymbolID
}

func (LoopStatement) NodeKind() string { return "LoopStatement" }
func (LoopStatement) stmt()            {}

type LockStatement struct {
	Span
	StatementHeader
	Lock Expr
	Body Block
}

func (LockStatement) NodeKind() string { return "LockStatement" }
func (LockStatement) stmt()            {}

type BlockStatement struct {
	Span
	StatementHeader
	Body Block
}

func (BlockStatement) NodeKind() string { return "BlockStatement" }
func (BlockStatement) stmt()            {}

type LabeledBlockStatement struct {
	Span
	StatementHeader
	Label symboltable.SymbolID
	Body  Block
}

func (LabeledBlockStatement) NodeKind() string { return "LabeledBlockStatement" }
func (LabeledBlockStatement) stmt()            {}

type BreakStatement struct {
	Span
	StatementHeader
	Label symboltable.SymbolID
}

func (BreakStatement) NodeKind() string { return "BreakStatement" }
func (BreakStatement) stmt()            {}

type ContinueStatement struct {
	Span
	StatementHeader
	Label symboltable.SymbolID
}

func (ContinueStatement) NodeKind() string { return "ContinueStatement" }
func (ContinueStatement) stmt()            {}
