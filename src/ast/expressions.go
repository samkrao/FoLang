package ast

import (
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
	lexer "github.com/samkrao/fo-lang/src/scanlex"
)

// --------------------
// Literal Expressions
// --------------------

// NumberLiteral represents a floating-point numeric literal expression.
type NumberLiteral struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n NumberLiteral) expr() {}

// IntegerLiteral represents an integer numeric literal expression.
type IntegerLiteral struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n IntegerLiteral) expr() {}

// StringLiteral represents a string literal expression.
type StringLiteral struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n StringLiteral) expr() {}

// CharacterLiteral represents a single character literal expression.
type CharacterLiteral struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n CharacterLiteral) expr() {}

// BooleanLiteral represents a boolean literal expression.
type BooleanLiteral struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n BooleanLiteral) expr() {}

// SymbolExpr represents a symbol reference expression. It never represents an
// invocation; function and method calls, including built-in candidates, use
// CallExpr uniformly.
type SymbolExpr struct {
	Span
	NodeName string
	Value    string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
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
	NodeName  string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
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
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n CommaExpr) expr() {}

// BinaryExpr represents a binary operator expression.
type BinaryExpr struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n BinaryExpr) expr() {}

// GroupingExpr represents a parenthesized grouping expression.
type GroupingExpr struct {
	Span
	NodeName string
	Expr_    Expr
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n GroupingExpr) expr() {}

// SDTExpr represents a simple data type expression wrapping a Type node.
type SDTExpr struct {
	Span
	NodeName string
	Type_    Type
	Dapst    Stmt
	Symb     *symboltable.ExpressionSymbol
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n SDTExpr) expr() {}

// Type__ marks this expression as also being a type expression.
func (n SDTExpr) Type__() {}

// ConditionalExpr represents a conditional (ternary-style) expression.
type ConditionalExpr struct {
	Span
	NodeName    string
	Left        Expr
	Operator    lexer.Token
	Right       Expr
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n ConditionalExpr) expr() {}

// AssignmentExpr represents an assignment expression.
type AssignmentExpr struct {
	Span
	NodeName      string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n AssignmentExpr) expr() {}

// PrefixExpr represents a prefix unary operator expression.
type PrefixExpr struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n PrefixExpr) expr() {}

// MemberExpr represents a member access (dot) expression.
type MemberExpr struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}
func (n MemberExpr) expr() {}

// ParentSelectorExpr is the dedicated compile-time selector for a direct class
// parent. It is distinct from MemberExpr/ComputedExpr so later resolution cannot
// mistake parent selection for ordinary runtime member or index dispatch.
type ParentSelectorExpr struct {
	Span
	NodeName         string
	Receiver         string
	Index            int
	ExplicitTypeName bool
	ParentName       string
	Dapst            Stmt
	Symb             *symboltable.ExpressionSymbol
}

func (n ParentSelectorExpr) GetName() string { return n.Symb.GetName() }
func (n ParentSelectorExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}
func (n ParentSelectorExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if n.Dapst == nil {
		(&n).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	n.Dapst.(*DirectveList).SetDap(daps)
}
func (n ParentSelectorExpr) expr() {}

// RelationshipSelectorExpr selects one directly declared @co.dap.oops
// relationship by its compile-time type name. Relationship categories are
// compile-time namespaces, not runtime values or collections.
type RelationshipSelectorExpr struct {
	Span
	NodeName   string
	Receiver   string
	Category   string
	TargetName string
	Dapst      Stmt
	Symb       *symboltable.ExpressionSymbol
}

func (n RelationshipSelectorExpr) GetName() string { return n.Symb.GetName() }
func (n RelationshipSelectorExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}
func (n RelationshipSelectorExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if n.Dapst == nil {
		(&n).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	n.Dapst.(*DirectveList).SetDap(daps)
}
func (n RelationshipSelectorExpr) expr() {}

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
	NodeName    string
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
		(&b).Dapst = DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(DirectveList).SetDap(daps)
}
func (n CallExpr) expr() {}

// LifecycleCallExpr is the lifecycle-call-suffix:
//
//	lifecycle-call-suffix = lifecycle-invocation-marker, lifecycle-invocation-name,
//	                        "(", [ argument-list ], ")",
//	                        lifecycle-call-context-guard
//
// `receiver::new(…)` is NOT an ordinary call whose callee happens to be a member,
// which is why it is a node of its own rather than a CallExpr over a MemberExpr.
// Ordinary member lookup and lifecycle lookup are separate semantic channels: a
// method named `new` reached through `.` and the lifecycle member reached through
// `::` are unrelated declarations, and the reference is explicit that FoLang
// therefore does not reserve `new` or `init` as ordinary method names
// (docs/language-ref.md, "Lifecycle Members"). Lowering `::` onto MemberExpr
// would merge exactly the two channels the language keeps apart.
//
// The call parentheses are part of the production, so there is no node here for a
// bare `Type::new`: a lifecycle member is not a first-class member value.
type LifecycleCallExpr struct {
	Span
	NodeName string
	// Receiver is the expression left of "::" — a type name for `Type::new(…)`,
	// an object for `object::init(…)`, or `self.parent` / `this.parent` for the
	// parent-lifecycle access a lifecycle customization is permitted to make.
	Receiver Expr
	// Name is the lifecycle-invocation-name as written: "new" or "init".
	Name string
	// Declaration is the lifecycle-declaration-name that invocation selects —
	// "@@new" or "@@init". The mapping is fixed by the language, so resolving it
	// once here saves every consumer from re-deriving it, and keeps the two
	// halves of the split spelling together on one node.
	Declaration string
	Arguments   []Expr
	Dapst       any
	Symb        *symboltable.ExpressionSymbol
}

func (n LifecycleCallExpr) GetName() string {
	return n.Symb.GetName()
}
func (n LifecycleCallExpr) GetSymbolType() string {
	return string(symboltable.S_ExpressionSymbol)
}

// SetDap attaches directive annotations to the node.
func (b LifecycleCallExpr) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.Dapst == nil {
		(&b).Dapst = DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(DirectveList).SetDap(daps)
}
func (n LifecycleCallExpr) expr() {}

// ComputedExpr represents a computed (bracket) member access expression.
type ComputedExpr struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)
}

func (n ComputedExpr) expr() {}

// RangeExpr represents a range expression with optional bounds.
type RangeExpr struct {
	Span
	NodeName     string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)

}

// FunctionExpr represents an inline function expression.
type FunctionExpr struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)

}
func (n FunctionExpr) expr() {}

// ArrayLiteral represents an array literal expression.
type ArrayLiteral struct {
	Span
	NodeName string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.Dapst.(*DirectveList).SetDap(daps)

}

// NewExpr represents an object instantiation expression.
type NewExpr struct {
	Span
	NodeName      string
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
	NodeName string
	Stmt_    Stmt
	Expr_    Expr
	Type_    LetType
	Symb     *symboltable.LetBindings
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
	NodeName   string
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
		(&b).Dapst = &DirectveList{NodeName: "DirectveList"}
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
	NodeName string
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
	NodeName string
	Name     string // "$", "$0", "$1", etc.
	Index    int    // -1 for bare $, else the numeric suffix
	Symb     *symboltable.VarSymbol
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
