package parser_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/parser"
)

// parseConformanceAST parses one accepted fixture and returns its complete AST.
// These tests complement the data-driven acceptance test by checking that
// information required by later passes was not discarded during parsing.
func parseConformanceAST(t *testing.T, name string) ast.Stmt {
	t.Helper()
	path := filepath.Join("examples", "accepted", name)
	source := readFixture(t, path)

	var root ast.Stmt
	mustNotPanic(t, func() {
		root, _, _, _ = parser.Parse(
			source,
			"conformance",
			filepath.Dir(path),
			filepath.Base(path),
			"",
			"program",
			"program",
			true,
		)
	})
	return root
}

func TestFunctionPatternStructureIsLossless(t *testing.T) {
	root := parseConformanceAST(t, "pattern-families.fol")
	application, ok := root.(ast.Application)
	if !ok {
		t.Fatalf("pattern fixture returned %T, want ast.Application", root)
	}
	if len(application.Body) < 7 {
		t.Fatalf("pattern fixture produced %d entry items, want at least 7", len(application.Body))
	}

	annotated, ok := application.Body[0].(ast.FunctionPatternStmt)
	if !ok {
		t.Fatalf("first entry item is %T, want ast.FunctionPatternStmt", application.Body[0])
	}
	if annotated.Dapst == nil {
		t.Error("function-pattern annotation was discarded")
	}

	guarded, ok := application.Body[1].(ast.FunctionPatternStmt)
	if !ok {
		t.Fatalf("second entry item is %T, want ast.FunctionPatternStmt", application.Body[1])
	}
	if len(guarded.PatternArgs) != 1 {
		t.Errorf("guarded clause has arity %d, want 1", len(guarded.PatternArgs))
	}
	if guarded.Guard == nil {
		t.Error("function-pattern where guard was discarded")
	}

	recordClause, ok := application.Body[3].(ast.FunctionPatternStmt)
	if !ok {
		t.Fatalf("record entry item is %T, want ast.FunctionPatternStmt", application.Body[3])
	}
	record, ok := recordClause.PatternArgs[0].(ast.NewExpr)
	if !ok {
		t.Fatalf("record pattern is %T, want ast.NewExpr", recordClause.PatternArgs[0])
	}
	if len(record.Instantiation.Arguments) != 2 {
		t.Fatalf("record pattern retained %d fields, want 2", len(record.Instantiation.Arguments))
	}
	wantLabels := []string{"id", "name"}
	for i, want := range wantLabels {
		field, ok := record.Instantiation.Arguments[i].(ast.AssignmentExpr)
		if !ok {
			t.Fatalf("record field %d is %T, want ast.AssignmentExpr", i, record.Instantiation.Arguments[i])
		}
		label, ok := field.Assigne.(ast.SymbolExpr)
		if !ok {
			t.Fatalf("record field %d label is %T, want ast.SymbolExpr", i, field.Assigne)
		}
		if got := strings.TrimSuffix(label.Value, "_fo"); got != want {
			t.Errorf("record field %d label = %q, want %q", i, got, want)
		}
	}
}

func TestAnonymousFunctionUsesEnclosingGenericParameters(t *testing.T) {
	root := parseConformanceAST(t, "anonymous_forall.unit.fol")
	pkg, ok := root.(ast.PackageStmt)
	if !ok {
		t.Fatalf("anonymous fixture returned %T, want ast.PackageStmt", root)
	}
	unit := pkg.Body[0].(ast.TypeDeclarationStmt)
	holder := unit.Body[0].(ast.GenerricFun).FunctionDeclarationStmt
	identity := holder.Body[0].(ast.VarDeclarationStmt)
	function, ok := identity.AssignedValue.(ast.FunctionExpr)
	if !ok {
		t.Fatalf("identity initializer is %T, want ast.FunctionExpr", identity.AssignedValue)
	}
	if len(function.TypeParams) != 0 {
		t.Fatalf("anonymous function introduced %d type parameters, want none", len(function.TypeParams))
	}
	if len(function.Parameters) != 1 {
		t.Fatalf("anonymous function parameters = %#v, want one enclosing-generic parameter", function.Parameters)
	}
	if got := strings.TrimSuffix(function.Parameters[0].Type_.GetName(), "_fo"); got != "T" {
		t.Errorf("anonymous parameter type = %q, want enclosing T", got)
	}
}

func TestMatchSelectorExpressionIsPreserved(t *testing.T) {
	root := parseConformanceAST(t, "expression-matcher-selector.fol")
	application := root.(ast.Application)
	declaration := application.Body[0].(ast.VarDeclarationStmt)
	wrapper, ok := declaration.AssignedValue.(ast.StatementExpr)
	if !ok {
		t.Fatalf("match initializer is %T, want ast.StatementExpr", declaration.AssignedValue)
	}
	match, ok := wrapper.Statement.(ast.PatternExprStmt)
	if !ok {
		t.Fatalf("match wrapper contains %T, want ast.PatternExprStmt", wrapper.Statement)
	}
	if _, ok := match.Expr_.MatcherExpr.(ast.CallExpr); !ok {
		t.Fatalf("matcher selector is %T, want ast.CallExpr", match.Expr_.MatcherExpr)
	}
	if !match.Matcher || !match.CustomMatcher {
		t.Errorf("expression matcher flags are Matcher=%v CustomMatcher=%v, want true/true", match.Matcher, match.CustomMatcher)
	}
}
