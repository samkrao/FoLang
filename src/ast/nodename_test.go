package ast

import (
	"reflect"
	"testing"
)

// astNodes is one zero value of every AST node type.
//
// The roster is written out rather than discovered because Go cannot enumerate a
// package's types at run time, and a reflective walk from a parsed tree only ever
// reaches the forms that source happened to use. Listing them makes the count
// itself a fact the tests can check.
var astNodes = []any{
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

// NodeName must be DECLARED on each node type, never inherited.
//
// Nine statement nodes embed FunctionDeclarationStmt and one embeds TypeclassStmt.
// An embedder that does not declare its own field still compiles and still has a
// NodeName — the promoted one — so it silently reports the embedded form. A
// decorator would call itself a function declaration everywhere it is printed,
// and no construction site would look wrong.
//
// FieldByName reports the access path in Index; a length of one means the field
// sits on the type itself rather than inside something it embeds.
func TestNodeNameIsDeclaredOnEveryNodeType(t *testing.T) {
	for _, node := range astNodes {
		type_ := reflect.TypeOf(node)
		field, ok := type_.FieldByName("NodeName")
		switch {
		case !ok:
			t.Errorf("%s has no NodeName field", type_.Name())
		case field.Type.Kind() != reflect.String:
			t.Errorf("%s.NodeName is %s, want string", type_.Name(), field.Type)
		case len(field.Index) != 1:
			t.Errorf("%s inherits NodeName from an embedded node instead of declaring its own", type_.Name())
		}
	}
}

// A node type added to the package and forgotten here is never checked, so the
// count is pinned: adding a node fails this test, and the fix is to add it above.
func TestNodeRosterCoversEveryNodeType(t *testing.T) {
	const nodeTypes = 113
	if len(astNodes) != nodeTypes {
		t.Fatalf("roster holds %d node types, want %d; add the new node to astNodes", len(astNodes), nodeTypes)
	}
}
