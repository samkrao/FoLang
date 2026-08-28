package ast

import (
	"reflect"
	"testing"
)

// nodeNamer is NodeName alone, so the roster below can hold every node type
// without also depending on the rest of SET.
type nodeNamer interface {
	NodeName() string
}

// astNodes is one zero value of every AST node type.
//
// The roster is written out rather than discovered because Go cannot enumerate a
// package's types at run time, and a reflective walk from a parsed tree would
// only ever reach the forms that source happened to use. Listing them makes the
// count itself a fact the tests can check.
var astNodes = []nodeNamer{
	AddressVariableDeclStmt{},
	Application{},
	Argument{},
	ArrayVariableDeclStmt{},
	ArrowFunction{},
	BlockStmt{},
	BreakStmt{},
	BuiltInConstantStmt{},
	BuiltInStmt{},
	CaseStmt{},
	ClassDeclarationStmt{},
	CodeStmt{},
	ComponentDeclarationStmt{},
	ConditionalStmt{},
	ContainsStmt{},
	ContinueStmt{},
	DecoratorStmt{},
	DefaultConditionalStmt{},
	DelegateStmt{},
	DependentTypeDeclarationStmt{},
	DirectiveStmt{},
	DirectveList{},
	DummyStmt{},
	ExecutionModelFunctionStmt{},
	ExpressionStmt{},
	ExtensionDeclarationStmt{},
	ExtensionStmt{},
	ForAllStmt{},
	ForeachStmt{},
	FunctionDeclarationStmt{},
	FunctionPatternStmt{},
	FunctionReceiver{},
	GenerricFun{},
	HeapAllocatedRefStmt{},
	IfStmt{},
	ImportStmt{},
	IndexerStmt{},
	LabeledStmt{},
	Library{},
	LockStmt{},
	MacroStmt{},
	MatchExprStmt{},
	MatcherInstanceStmt{},
	ModuleStmt{},
	NativeFunctionStmt{},
	ObjectDeclStmt{},
	OperatorStmt{},
	PackageStmt{},
	Parameter{},
	PatternExprStmt{},
	PointerVariableDeclStmt{},
	PredicateTypeDeclarationStmt{},
	Prog{},
	ProjectStmt{},
	RangeVariableDeclStmt{},
	RefVariableDeclStmt{},
	RefinementTypeDeclarationStmt{},
	ReturnStmt{},
	Returns{},
	SliceVariableDeclStmt{},
	TemplateStmt{},
	TernaryStmt{},
	ThunkVariableDeclStmt{},
	TraversableStmt{},
	TypeComposeStmt{},
	TypeConstructorStmt{},
	TypeDeclarationStmt{},
	TypeStmt{},
	TypeclassInstanceStmt{},
	TypeclassStmt{},
	UseStmtDirective{},
	VarDeclarationStmt{},
	ADTExpr{},
	ArrayLiteral{},
	AssignmentExpr{},
	BinaryExpr{},
	BindVariableExpr{},
	BooleanLiteral{},
	CallExpr{},
	CharacterLiteral{},
	CommaExpr{},
	ComputedExpr{},
	ConditionalExpr{},
	DefaultExpr{},
	ForComprehensionExpr{},
	FunctionExpr{},
	GroupingExpr{},
	IntegerLiteral{},
	LambdaExpr{},
	LetExpr{},
	LifecycleCallExpr{},
	MemberExpr{},
	NewExpr{},
	NumberLiteral{},
	ParentSelectorExpr{},
	PlaceHolderExpr{},
	PrefixExpr{},
	RangeExpr{},
	RelationshipSelectorExpr{},
	SDTExpr{},
	StatementExpr{},
	StringLiteral{},
	SymbolExpr{},
	BuiltInDataType{},
	CompoundType{},
	DependentType{},
	DerivedType{},
	ForAllType{},
	FunctionType{},
	GenericType{},
	ListType{},
	SymbolRefExpr{},
	SymbolTypeNode{},
}

// NodeName must be the node's own Go type name: no package qualifier, no
// decoration, and above all not an inherited one.
//
// The last is the case worth testing. Nine statement nodes embed
// FunctionDeclarationStmt and would otherwise inherit its NodeName, so a missing
// method does not fail to compile — it silently answers "FunctionDeclarationStmt"
// for a decorator, an operator and seven other forms.
func TestNodeNameMatchesTheGoTypeName(t *testing.T) {
	for _, node := range astNodes {
		want := reflect.TypeOf(node).Name()
		if got := node.NodeName(); got != want {
			t.Errorf("%T.NodeName() = %q, want %q", node, got, want)
		}
	}
}

// Two nodes answering the same name is the copy-paste failure the roster exists
// to catch: the tree still compiles, and every reader downstream conflates the
// two forms.
func TestNodeNameIsUniquePerNodeType(t *testing.T) {
	claimed := map[string]string{}
	for _, node := range astNodes {
		name := node.NodeName()
		type_ := reflect.TypeOf(node).Name()
		if previous, taken := claimed[name]; taken {
			t.Errorf("%s and %s both answer NodeName() = %q", previous, type_, name)
			continue
		}
		claimed[name] = type_
	}
}

// A node type added to the package without a NodeName does not compile, because
// SET requires it. A node type added without a ROSTER entry compiles fine and is
// simply never checked, so the count is pinned here: adding a node to the package
// and forgetting this file fails, and the fix is to add the node below.
func TestNodeRosterCoversEveryNodeType(t *testing.T) {
	const nodeTypes = 113
	if len(astNodes) != nodeTypes {
		t.Fatalf("roster holds %d node types, want %d; add the new node to astNodes", len(astNodes), nodeTypes)
	}
}
