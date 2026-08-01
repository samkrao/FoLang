package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

// TestControlChainsLowerOnlyWhenTheirCanonicalShapeFits protects the HIR from
// lossy rewrites. ForeachStmt and ContainsStmt retain only a receiver name, and
// the reference gives each/contains a .do branch—not a .loop branch.
func TestControlChainsLowerOnlyWhenTheirCanonicalShapeFits(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantNode string
	}{
		{"each-do", "items.each(_, value).do({});", "foreach"},
		{"each-loop", "items.each(index, value).loop({});", "generic"},
		{"each-wildcard-value", "items.each(index, _).do({});", "generic"},
		{"each-complex-subject", "makeItems().each(index, value).do({});", "generic"},
		{"contains-do", "items.contains(value).do({});", "conditional"},
		{"contains-loop", "items.contains(value).loop({});", "generic"},
		{"contains-complex-subject", "makeItems().contains(value).do({});", "generic"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := parseRegressionBody(t, tc.source)
			if len(body) != 1 {
				t.Fatalf("parsed %d statements, want 1", len(body))
			}

			switch tc.wantNode {
			case "foreach":
				if _, ok := body[0].(ast.ForeachStmt); !ok {
					t.Fatalf("canonical each chain is %T, want ast.ForeachStmt", body[0])
				}
			case "conditional":
				if _, ok := body[0].(ast.ConditionalStmt); !ok {
					t.Fatalf("canonical contains chain is %T, want ast.ConditionalStmt", body[0])
				}
			case "generic":
				if _, ok := body[0].(ast.ExpressionStmt); !ok {
					t.Fatalf("non-canonical or lossy chain lowered to %T", body[0])
				}
			}
		})
	}
}

func TestNestedObjectFieldIsLoweredRecursively(t *testing.T) {
	body := parseRegressionBody(t, "result := Box{value: (truth).return(1).otherwise.return(2)};")
	decl := body[0].(ast.VarDeclarationStmt)
	if !hasLoweredTernary(decl.AssignedValue) {
		t.Fatalf("object construction did not lower its field value: %T", decl.AssignedValue)
	}
}

func TestReturnPayloadIsLoweredRecursively(t *testing.T) {
	body := parseRegressionBody(t, "Box co.lang.class = { run()->() = { this.return (truth).return(1).otherwise.return(2); } }")
	class, ok := body[0].(ast.ClassDeclarationStmt)
	if !ok || len(class.Body) != 1 {
		t.Fatalf("class declaration is %T with unexpected body", body[0])
	}
	function, ok := class.Body[0].(ast.FunctionDeclarationStmt)
	if !ok || len(function.Body) != 1 {
		t.Fatalf("class member is %T with unexpected body", class.Body[0])
	}
	returnStmt, ok := function.Body[0].(ast.ReturnStmt)
	if !ok {
		t.Fatalf("function body is %T, want ast.ReturnStmt", function.Body[0])
	}
	if _, ok := returnStmt.StmtExpr_.(ast.TernaryStmt); !ok {
		t.Fatalf("return payload is %T, want lowered ast.TernaryStmt", returnStmt.StmtExpr_)
	}
}

func TestNestedRangeOperandIsLoweredRecursively(t *testing.T) {
	body := parseRegressionBody(t, "result := 0..((truth).return(1).otherwise.return(2));")
	decl, ok := body[0].(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("statement is %T, want ast.VarDeclarationStmt", body[0])
	}
	rangeExpr, ok := decl.AssignedValue.(ast.RangeExpr)
	if !ok {
		t.Fatalf("initializer is %T, want ast.RangeExpr", decl.AssignedValue)
	}
	if !hasLoweredTernary(rangeExpr.Upper) {
		t.Fatalf("range upper operand did not retain a lowered ternary: %T", rangeExpr.Upper)
	}
}

func TestMatchSelectorAndCaseResultsAreLoweredRecursively(t *testing.T) {
	body := parseRegressionBody(t, `result := value.match((truth).return(MatcherA).otherwise.return(MatcherB)).case(x => (truth).return(1).otherwise.return(2)).default(0);`)
	decl := body[0].(ast.VarDeclarationStmt)
	wrapper, ok := decl.AssignedValue.(ast.StatementExpr)
	if !ok {
		t.Fatalf("match initializer is %T, want ast.StatementExpr", decl.AssignedValue)
	}
	match, ok := wrapper.Statement.(ast.PatternExprStmt)
	if !ok || len(match.CaseExprStmt) != 1 {
		t.Fatalf("match statement is %T with unexpected cases", wrapper.Statement)
	}
	if !hasLoweredTernary(match.Expr_.MatcherExpr) {
		t.Fatalf("match selector did not retain a lowered ternary: %T", match.Expr_.MatcherExpr)
	}
	caseResult, ok := match.CaseExprStmt[0].Stmt_.(ast.ExpressionStmt)
	if !ok || !hasLoweredTernary(caseResult.Expression) {
		t.Fatalf("match case result did not retain a lowered ternary: %T", match.CaseExprStmt[0].Stmt_)
	}
}

func TestBuiltInTypeCanDispatchToObjectConstruction(t *testing.T) {
	body := parseRegressionBody(t, "value := co.lang.int{};")
	decl, ok := body[0].(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("statement is %T, want ast.VarDeclarationStmt", body[0])
	}
	if _, ok := decl.AssignedValue.(ast.NewExpr); !ok {
		t.Fatalf("built-in construction parsed as %T, want ast.NewExpr", decl.AssignedValue)
	}
}

func parseRegressionBody(t *testing.T, source string) []ast.Stmt {
	t.Helper()

	root, _, _, _ := parser.Parse(source, "regression", ".", "regression.fol", "", "program", "program", true)
	switch n := root.(type) {
	case ast.Application:
		return n.Body
	case ast.PackageStmt:
		return n.Body
	default:
		t.Fatalf("root is %T, want an application or package", root)
		return nil
	}
}

func hasLoweredTernary(expr ast.Expr) bool {
	switch n := expr.(type) {
	case ast.StatementExpr:
		switch statement := n.Statement.(type) {
		case ast.TernaryStmt:
			return true
		case ast.ExpressionStmt:
			return hasLoweredTernary(statement.Expression)
		}
	case ast.GroupingExpr:
		return hasLoweredTernary(n.Expr_)
	case ast.RangeExpr:
		return hasLoweredTernary(n.Lower) || hasLoweredTernary(n.Upper)
	case ast.NewExpr:
		for _, argument := range n.Instantiation.Arguments {
			if hasLoweredTernary(argument) {
				return true
			}
		}
	case ast.AssignmentExpr:
		return hasLoweredTernary(n.Assigne) || hasLoweredTernary(n.AssignedValue)
	case ast.CallExpr:
		if hasLoweredTernary(n.Method) {
			return true
		}
		for _, argument := range n.Arguments {
			if hasLoweredTernary(argument) {
				return true
			}
		}
	}
	return false
}
