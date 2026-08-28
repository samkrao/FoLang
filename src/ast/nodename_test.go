package ast

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"strings"
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

// The source is the authority for the roster. Every concrete AST node declares
// one of the private marker methods used by Stmt, Expr and Type, so comparing
// those receivers with astNodes detects both a newly omitted node and a stale
// roster entry. A fixed expected count cannot detect an omitted new type.
func TestNodeRosterCoversEveryNodeType(t *testing.T) {
	want := declaredNodeTypes(t)
	got := make(map[string]bool, len(astNodes))
	for _, node := range astNodes {
		name := reflect.TypeOf(node).Name()
		if got[name] {
			t.Errorf("%s occurs more than once in astNodes", name)
		}
		got[name] = true
	}

	for name := range want {
		if !got[name] {
			t.Errorf("AST node %s is missing from astNodes", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("astNodes contains %s, which declares no AST marker method", name)
		}
	}
}

func declaredNodeTypes(t *testing.T) map[string]bool {
	t.Helper()

	packages, err := parser.ParseDir(token.NewFileSet(), ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse ast package: %v", err)
	}
	pkg, ok := packages["ast"]
	if !ok {
		t.Fatal("parsed source contains no ast package")
	}

	nodes := map[string]bool{}
	embeddedTypes := map[string][]string{}
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			if generic, ok := declaration.(*goast.GenDecl); ok {
				for _, specification := range generic.Specs {
					typeSpec, ok := specification.(*goast.TypeSpec)
					if !ok {
						continue
					}
					structType, ok := typeSpec.Type.(*goast.StructType)
					if !ok {
						continue
					}
					for _, field := range structType.Fields.List {
						if len(field.Names) == 0 {
							if name := receiverTypeName(field.Type); name != "" {
								embeddedTypes[typeSpec.Name.Name] = append(embeddedTypes[typeSpec.Name.Name], name)
							}
						}
					}
				}
			}

			function, ok := declaration.(*goast.FuncDecl)
			if !ok || function.Recv == nil || len(function.Recv.List) == 0 {
				continue
			}
			switch function.Name.Name {
			case "stmt", "expr", "_type":
			default:
				continue
			}
			if name := receiverTypeName(function.Recv.List[0].Type); name != "" {
				nodes[name] = true
			}
		}
	}

	// A struct embedding a node receives its private marker method through method
	// promotion and therefore also implements the corresponding AST interface.
	for changed := true; changed; {
		changed = false
		for outer, embedded := range embeddedTypes {
			if nodes[outer] {
				continue
			}
			for _, inner := range embedded {
				if nodes[inner] {
					nodes[outer] = true
					changed = true
					break
				}
			}
		}
	}
	return nodes
}

func receiverTypeName(expression goast.Expr) string {
	switch expression := expression.(type) {
	case *goast.Ident:
		return expression.Name
	case *goast.StarExpr:
		return receiverTypeName(expression.X)
	default:
		return ""
	}
}
