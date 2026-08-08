package ast

import (
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	lexer "github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// --------------------
// Literal Expressions
// --------------------

// PlaceHolderExpr represents a placeholder expression node.
type PlaceHolderExpr struct {
	Span
	Symb *symboltable.ExpressionSymbol
}

func (n PlaceHolderExpr) GetName() string {
	return n.Symb.GetName()
}
func (n PlaceHolderExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b PlaceHolderExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}
func (p PlaceHolderExpr) expr() {}

// NumberLiteral represents a floating-point numeric literal expression.
type NumberLiteral struct {
	Span
	Value    float64
	Type_    string
	ActType_ string
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n NumberLiteral) GetName() string {
	return n.Symb.GetName()
}
func (n NumberLiteral) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b NumberLiteral) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n NumberLiteral) expr() {}

// IntegerLiteral represents an integer numeric literal expression.
type IntegerLiteral struct {
	Span
	Value    int64
	Type_    string
	ActType_ string
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n IntegerLiteral) GetName() string {
	return n.Symb.GetName()
}
func (n IntegerLiteral) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b IntegerLiteral) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n IntegerLiteral) expr() {}

// StringLiteral represents a string literal expression.
type StringLiteral struct {
	Span
	Value    string
	ActType_ string
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n StringLiteral) GetName() string {
	return n.Symb.GetName()
}
func (n StringLiteral) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b StringLiteral) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n StringLiteral) expr() {}

// CharacterLiteral represents a single character literal expression.
type CharacterLiteral struct {
	Span
	Value    rune
	ActType_ string
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n CharacterLiteral) GetName() string {
	return n.Symb.GetName()
}
func (n CharacterLiteral) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b CharacterLiteral) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n CharacterLiteral) expr() {}

// BooleanLiteral represents a boolean literal expression.
type BooleanLiteral struct {
	Span
	Value    bool
	ActType_ string
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n BooleanLiteral) GetName() string {
	return n.Symb.GetName()
}
func (n BooleanLiteral) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b BooleanLiteral) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n BooleanLiteral) expr() {}

// SymbolExpr represents a symbol reference expression. It never represents an
// invocation; function and method calls, including built-in candidates, use
// CallExpr uniformly.
type SymbolExpr struct {
	Span
	Value string
	// IsMethodCall is retained for serialized-AST compatibility only.
	// Deprecated: parser-produced invocations always use CallExpr and leave
	// this field false.
	IsMethodCall bool
	SymbolType_  string
	Dapst        Stmt
	Symb         *symboltable.ExpressionSymbol
}

func (n SymbolExpr) GetName() string {
	return n.Symb.GetName()
}
func (n SymbolExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b SymbolExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n SymbolExpr) expr() {}

// TypeExpr is an expression that also represents a type.
type TypeExpr interface {
	Expr
	Type__()
}

// StatementExpr wraps a statement as an expression.
type StatementExpr struct {
	Span
	Statement Stmt
	Dapst     Stmt
	Symb      *symboltable.ExpressionSymbol
}

func (n StatementExpr) GetName() string {
	return n.Symb.GetName()
}
func (n StatementExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b StatementExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n StatementExpr) expr() {}

// --------------------
// Complex Expressions
// --------------------

// CommaExpr represents a comma-separated binary expression.
type CommaExpr struct {
	Span
	Left     Expr
	Operator lexer.Token
	Right    Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n CommaExpr) GetName() string {
	return n.Symb.GetName()
}
func (n CommaExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b CommaExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n CommaExpr) expr() {}

// BinaryExpr represents a binary operator expression.
type BinaryExpr struct {
	Span
	Left     Expr
	Operator lexer.Token
	Right    Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n BinaryExpr) GetName() string {
	return n.Symb.GetName()
}
func (n BinaryExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b BinaryExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n BinaryExpr) expr() {}

// GroupingExpr represents a parenthesized grouping expression.
type GroupingExpr struct {
	Span
	Expr_ Expr
	Dapst Stmt
	Symb  *symboltable.ExpressionSymbol
}

func (n GroupingExpr) GetName() string {
	return n.Symb.GetName()
}
func (n GroupingExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b GroupingExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n GroupingExpr) expr() {}

// ADTExpr represents an algebraic data type expression.
type ADTExpr struct {
	Span
	Left     Expr
	Operator lexer.Token
	Right    Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n ADTExpr) GetName() string {
	return n.Symb.GetName()
}
func (n ADTExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}
func (a ADTExpr) expr() {}

// Type__ marks this expression as also being a type expression.
func (a ADTExpr) Type__() {}

// SetDap attaches directive annotations to the node.
func (b ADTExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

// SDTExpr represents a simple data type expression wrapping a Type node.
type SDTExpr struct {
	Span
	Type_ Type
	Dapst Stmt
	Symb  *symboltable.ExpressionSymbol
}

func (n SDTExpr) GetName() string {
	return n.Symb.GetName()
}
func (n SDTExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b SDTExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n SDTExpr) expr() {}

// Type__ marks this expression as also being a type expression.
func (n SDTExpr) Type__() {}

// ConditionalExpr represents a conditional (ternary-style) expression.
type ConditionalExpr struct {
	Span
	Left        Expr
	Operator    lexer.Token
	Right       Expr
	BoolStmt    BuiltInConstantStmt
	BoolVarStmt SymbolRefExpr
	ArrayVar    Stmt
	CondVarStmt Expr
	CondValStmt Expr
	Type        string
	ValOrVar    string
	Dapst       Stmt
	Symb        *symboltable.ExpressionSymbol
}

func (n ConditionalExpr) GetName() string {
	return n.Symb.GetName()
}
func (n ConditionalExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b ConditionalExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n ConditionalExpr) expr() {}

// DefaultExpr represents a default value expression.
type DefaultExpr struct {
	Span
	Default bool
	Dapst   Stmt
	Symb    *symboltable.ExpressionSymbol
}

func (n DefaultExpr) GetName() string {
	return n.Symb.GetName()
}
func (n DefaultExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b DefaultExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n DefaultExpr) expr() {}

// AssignmentExpr represents an assignment expression.
type AssignmentExpr struct {
	Span
	Assigne       Expr
	Operator      lexer.Token
	AssignedValue Expr
	Dapst         Stmt
	Symb          *symboltable.ExpressionSymbol
}

func (n AssignmentExpr) GetName() string {
	return n.Symb.GetName()
}
func (n AssignmentExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b AssignmentExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n AssignmentExpr) expr() {}

// PrefixExpr represents a prefix unary operator expression.
type PrefixExpr struct {
	Span
	Operator lexer.Token
	Right    Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n PrefixExpr) GetName() string {
	return n.Symb.GetName()
}
func (n PrefixExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b PrefixExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n PrefixExpr) expr() {}

// MemberExpr represents a member access (dot) expression.
type MemberExpr struct {
	Span
	Member   Expr
	Property string
	Type_    lexer.TokenKind
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n MemberExpr) GetName() string {
	return n.Symb.GetName()
}
func (n MemberExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b MemberExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n MemberExpr) expr() {}

// CallKind records the parser's provisional syntactic classification of an
// invocation. Every value assigned during parsing may be refined by name and
// type resolution; none is a final dispatch decision.
//
// CallBuiltInMethod is deliberately only a candidate classification: the
// parser assigns it when the member name is in scanlex.Reserved_me. Name and
// type resolution may replace it with CallMethod when a class method,
// companion function, extension, or another user declaration wins lookup.
type CallKind uint8

const (
	// CallUnresolved is used when syntax alone cannot determine the call form,
	// for example when the callee is another expression.
	CallUnresolved CallKind = iota
	// CallFunction is a direct call whose callee is a name expression.
	CallFunction
	// CallMethod is an ordinary member invocation candidate.
	CallMethod
	// CallBuiltInMethod is a reserved built-in method candidate awaiting
	// receiver-aware name resolution.
	CallBuiltInMethod
)

// CallExpr represents every syntactic invocation, including function calls,
// ordinary method calls, and built-in method candidates. The Method expression
// retains the callee structure; CallKind supplies the provisional distinction
// needed by later resolution without changing the AST shape.
type CallExpr struct {
	Span
	Method      Expr
	Arguments   []Expr
	CallKind    CallKind
	SymbolType_ string
	Dapst       any
	Symb        *symboltable.ExpressionSymbol
}

func (n CallExpr) GetName() string {
	return n.Symb.GetName()
}
func (n CallExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b CallExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = DirectveList{}
	}
	b.Dapst.(DirectveList).SetDap(daps)
}
func (n CallExpr) expr() {}

// ComputedExpr represents a computed (bracket) member access expression.
type ComputedExpr struct {
	Span
	Member   Expr
	Property Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n ComputedExpr) GetName() string {
	return n.Symb.GetName()
}
func (n ComputedExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b ComputedExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

func (n ComputedExpr) expr() {}

// RangeExpr represents a range expression with optional bounds.
type RangeExpr struct {
	Span
	Lower        Expr // nil means open lower bound (e.g. ..100)
	Upper        Expr // nil means open upper bound (e.g. 1..)
	ExcludeStart bool // true for <.. operators (lower bound excluded)
	ExcludeEnd   bool // true for ..< operators (upper bound excluded)
	Dapst        Stmt
	Symb         *symboltable.ExpressionSymbol
}

func (n RangeExpr) GetName() string {
	return n.Symb.GetName()
}
func (n RangeExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

func (n RangeExpr) expr() {}

// SetDap attaches directive annotations to the node.
func (b RangeExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)

}

// FunctionExpr represents an inline function expression.
type FunctionExpr struct {
	Span
	// TypeParams preserves the optional forall parameters of a polymorphic
	// anonymous function.  Keeping only FunctionSymbol.IsGeneric loses their
	// names, constraints and higher-kinded arity before semantic analysis.
	TypeParams []symboltable.GenericTypeParam
	Parameters []Parameter
	Body       []Stmt
	ReturnType []Returns
	AsExpr     bool
	Dapst      Stmt
	Symb       *symboltable.FunctionSymbol
}

func (n FunctionExpr) GetName() string {
	return n.Symb.GetName()
}
func (n FunctionExpr) GetSymbolType() string {
	return string(symboltable.S_FunctionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b FunctionExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)

}
func (n FunctionExpr) expr() {}

// ArrayLiteral represents an array literal expression.
type ArrayLiteral struct {
	Span
	Contents []Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
}

func (n ArrayLiteral) GetName() string {
	return n.Symb.GetName()
}
func (n ArrayLiteral) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

func (n ArrayLiteral) expr() {}

// SetDap attaches directive annotations to the node.
func (b ArrayLiteral) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)

}

// NewExpr represents an object instantiation expression.
type NewExpr struct {
	Span
	Instantiation CallExpr
	Symb          *symboltable.ExpressionSymbol
}

func (n NewExpr) GetName() string {
	return n.Symb.GetName()
}
func (n NewExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b NewExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}
func (n NewExpr) expr() {}

// LetType distinguishes between let-in and let-where binding forms.
type LetType string

const (
	IN    LetType = "IN"
	WHERE LetType = "WHERE"
)

// LetExpr represents a let-in or let-where binding expression.
type LetExpr struct {
	Span
	Stmt_ Stmt
	Expr_ Expr
	Type_ LetType
	Symb  *symboltable.LetBindings
}

func (n LetExpr) GetName() string {
	return n.Symb.GetName()
}
func (n LetExpr) GetSymbolType() string {
	return string(symboltable.S_LetBindings)
}

// SetDap attaches directive annotations to the node.
func (b LetExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {

}
func (n LetExpr) expr() {}

// LambdaExpr represents an inline lambda: |x, y| => x + y
// Only allowed as callback arguments to collection operations (map, filter, etc.)
type LambdaExpr struct {
	Span
	Parameters []Parameter
	Body       Expr // single expression body (after =>)
	Dapst      Stmt
	Symb       *symboltable.LambdaSymbol
}

func (n LambdaExpr) GetName() string {
	return n.Symb.GetName()
}
func (n LambdaExpr) GetSymbolType() string {
	return string(symboltable.S_LambdaSymbol)
}

// SetDap attaches directive annotations to the node.
func (b LambdaExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = &DirectveList{}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n LambdaExpr) expr() {}

// ForBinding represents one variable binding in a for-comprehension generator,
// e.g. the `x` in `x <- collection` or `name` / `age` in `(name, age) <- pairs`.
type ForBinding struct {
	Name string
	Symb *symboltable.ExpressionSymbol
}

func (n ForBinding) GetName() string {
	return n.Symb.GetName()
}
func (n ForBinding) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// ForComprehensionExpr represents a for-comprehension expression:
//
//	for (x <- collection).yield(projection)
//	for ((name, age) <- pairs).yield(name.toUpperCase, age)
type ForComprehensionExpr struct {
	Span
	symboltable.ForComprehension
	Bindings []ForBinding // one or more bound variable names
	Source   Expr         // the generator/collection expression
	Yield    Expr         // the projection expression after .yield(...)
	Symb     *symboltable.ForComprehension
}

func (n ForComprehensionExpr) GetName() string {
	return n.Symb.GetName()
}
func (n ForComprehensionExpr) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b ForComprehensionExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {}
func (n ForComprehensionExpr) expr()                                        {}

// BindVariableExpr represents a bind/capture variable: $, $0, $1, $2, …
// These appear in function-chaining (=>>) results and let-binding recursion.
type BindVariableExpr struct {
	Span
	Name  string // "$", "$0", "$1", etc.
	Index int    // -1 for bare $, else the numeric suffix
	Symb  *symboltable.VarSymbol
}

func (n BindVariableExpr) GetName() string {
	return n.Symb.GetName()
}
func (n BindVariableExpr) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

// SetDap attaches directive annotations to the node.
func (b BindVariableExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {}
func (n BindVariableExpr) expr()                                        {}
