package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/parser"
)

// TestControlChainsLowerOnlyWhenTheirCanonicalShapeFits protects the HIR from
// lossy rewrites. ForeachStmt and ContainsStmt retain only a receiver name, and
// the current profile gives `.each` its own action argument while `.contains`
// selects with `.then` — neither may be followed by `.loop`.
func TestControlChainsLowerOnlyWhenTheirCanonicalShapeFits(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantNode string
	}{
		{"each-action", "items.each(_, value, {});", "foreach"},
		{"each-callback", "items.each(handler);", "generic"},
		{"each-complex-subject", "makeItems().each(index, value, {});", "generic"},
		{"contains-then", "items.contains(value).then({});", "conditional"},
		{"contains-complex-subject", "makeItems().contains(value).then({});", "generic"},
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

func TestMalformedControlVocabularyIsRejected(t *testing.T) {
	for _, source := range []string{
		`(a).loop({}).otherwise(b);`,
		`(a).loop({}).default({});`,
		`(a).then(1).otherwise();`,
		`(a).otherwise(b);`,
		`(a).default(1);`,
		`(a).otherwise({});`,
		`(a).default({});`,
		`(a).then({}).then({});`,
		`(a).then({}).default({}).otherwise(c);`,
		`(a).loop({}).loop({});`,
		`(a).then();`,
		`(a).loop();`,
		`items.each(handler).loop({});`,
		`items.each(value, {});`,
		`items.each(index, value, extra, {});`,
		`items.contains(value).then({}).otherwise(c).then({});`,
		`(a).do({});`,
	} {
		source := source
		mustPanic(t, func() { parseRegressionBody(t, source) })
	}
}

func TestControlVerbSpellingsRemainAvailableToOrdinaryMethods(t *testing.T) {
	for _, source := range []string{
		`y = config.default(1);`,
		`y = grid.each(1, 2);`,
		`y = engine.loop(1, 2);`,
		`y = worker.otherwise(1, 2);`,
		`y = worker.otherwise(1);`,
		`y = task.do();`,
		`y = a.b().do(1).c();`,
		`y = items.each();`,
	} {
		source := source
		mustNotPanic(t, func() { parseRegressionBody(t, source) })
	}
}

func TestLegacyBaseGuardIsClassContextual(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, `self Thing; y = self.base;`)
	})
}

func TestLegacyBaseLockFieldAndPositionalExternAreRejected(t *testing.T) {
	for _, tc := range []struct {
		source   string
		basename string
	}{
		{`_ co.lang.class = { run()->() = { self.base.classes[Base].run(); } }`, "Employee.fol"},
		{`_ co.lang.class = { queueLock co.lang.lock; }`, "Employee.fol"},
		{`_ co.lang.unit = { @co.dap.declare(extern) someBool co.lang.bool; }`, "Employee.comp.unit.fol"},
	} {
		tc := tc
		mustPanic(t, func() { parseRegressionFile(t, tc.source, tc.basename) })
	}
}

// TestLoweredControlChainsRetainTheirUnresolvedCalls protects the hand-off to
// receiver-aware method resolution. A control verb is only a candidate at parse
// time: a class/companion method or activated extension named `then` or `each`
// can still win, so lowering must not discard the uniform CallExpr/MemberExpr
// tree it rewrote.
func TestLoweredControlChainsRetainTheirUnresolvedCalls(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"each", "items.each(_, value, {});"},
		{"contains", "items.contains(value).then({});"},
		{"conditional", "(truth).then({});"},
		{"loop", "(truth).loop({});"},
		{"ternary", "(truth).then(1).default(2);"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := parseRegressionBody(t, tc.source)
			if len(body) != 1 {
				t.Fatalf("parsed %d statements, want 1", len(body))
			}

			var original ast.Expr
			switch node := body[0].(type) {
			case ast.ForeachStmt:
				original = node.OriginalChain
			case ast.ConditionalStmt:
				original = node.OriginalChain
			case ast.TernaryStmt:
				original = node.OriginalChain
			default:
				t.Fatalf("lowered statement is %T", body[0])
			}
			if original == nil {
				t.Fatal("lowered control node discarded its original call chain")
			}
			if !hasCallKind(original, ast.CallMethod) {
				t.Fatal("original chain does not retain its parsed member-call tree")
			}
		})
	}
}

func hasCallKind(expr ast.Expr, kind ast.CallKind) bool {
	switch node := expr.(type) {
	case ast.CallExpr:
		if node.CallKind == kind {
			return true
		}
		if hasCallKind(node.Method, kind) {
			return true
		}
		for _, argument := range node.Arguments {
			if hasCallKind(argument, kind) {
				return true
			}
		}
	case ast.MemberExpr:
		return hasCallKind(node.Member, kind)
	case ast.GroupingExpr:
		return hasCallKind(node.Expr_, kind)
	case ast.StatementExpr:
		if expression, ok := node.Statement.(ast.ExpressionStmt); ok {
			return hasCallKind(expression.Expression, kind)
		}
	}
	return false
}

func TestNestedObjectFieldIsLoweredRecursively(t *testing.T) {
	body := parseRegressionBody(t, "result := Box{value: (truth).then(1).default(2)};")
	decl := body[0].(ast.VarDeclarationStmt)
	if !hasLoweredTernary(decl.AssignedValue) {
		t.Fatalf("object construction did not lower its field value: %T", decl.AssignedValue)
	}
}

func TestReturnPayloadIsLoweredRecursively(t *testing.T) {
	body := parseRegressionFile(t,
		"_ co.lang.class = { run()->() = { this.return (truth).then(1).default(2); } }",
		"Box.fol")
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
	body := parseRegressionBody(t, "result := 0 .. ((truth).then(1).default(2));")
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

func TestMatchSelectorCaseAndDefaultResultsAreLoweredRecursively(t *testing.T) {
	body := parseRegressionBody(t, `result := value.match((truth).then(MatcherA).default(MatcherB)).case(x => (truth).then(1).default(2)).default((fallback).then(3).default(4));`)
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
	if _, ok := match.CaseExprStmt[0].Stmt_.(ast.TernaryStmt); !ok {
		t.Fatalf("match case result is %T, want directly lowered ast.TernaryStmt", match.CaseExprStmt[0].Stmt_)
	}
	if match.DefaultExprStmt == nil {
		t.Fatal("match default result is missing")
	}
	if _, ok := match.DefaultExprStmt.Stmt_.(ast.TernaryStmt); !ok {
		t.Fatalf("match default result is %T, want directly lowered ast.TernaryStmt", match.DefaultExprStmt.Stmt_)
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

	return parseRegressionFile(t, source, "regression.fol")
}

// parseRegressionFile parses source under an explicit source filename. A
// declaration-shaped regression needs one, because the filename is what selects
// the package-source-file root and supplies every filename-derived name
// (DECISION-FILE-001).
func parseRegressionFile(t *testing.T, source, basename string) []ast.Stmt {
	t.Helper()

	root, _, _, _ := parser.Parse(source, "regression", ".", basename, "", "program", "program", true)
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

// TestLoopControlStatementsAreParsed covers this.break and this.continue.
//
// Both were rejected as "not part of the current FoLang statement grammar" until the
// reference gained the loopsEg2 example that uses them inside a `.loop({ … })` chain.
// The Disclaimer makes an example the thing that implements a feature, so the example
// is what moved them from reserved to parsed.
//
// Neither takes an operand: a loop chain is left, not left with a value.
func TestLoopControlStatementsAreParsed(t *testing.T) {
	body := `x := co.const.true;
        v := 0;
        x.loop({
            (v == 10).then({
                this.break;
            });
            v += 1;
        });
        (co.const.true).loop({
            (v == 30).then({
                this.continue;
            });
            v += 5;
        });`
	mustNotPanic(t, func() { parseRegressionBody(t, body) })

	// A value after either verb is a syntax error rather than a silently dropped
	// operand, so `this.break x;` cannot read as a break followed by nothing.
	for _, source := range []string{`this.break 1;`, `this.continue v;`} {
		source := source
		mustPanic(t, func() { parseRegressionBody(t, source) })
	}
}

// TestMatchChainRequiresAtLeastOneCase pins the rule the reference states directly:
// "A match chain contains one or more `.case(...)` arms followed by at most one
// terminal `.default(...)` arm."
//
// folang.ebnf spelled match-suffix with `{ match-case }` — zero or more — which
// admitted `x.match;` as a complete expression. The reference governs, and the
// grammar has been corrected to `match-case, { match-case }`.
func TestMatchChainRequiresAtLeastOneCase(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, `x.match.case(n: n > 10 => "GT").default("EQ");`)
	})
	for _, source := range []string{
		`x.match;`,
		`x.match();`,
		`x.match.default("only-default");`,
	} {
		source := source
		mustPanic(t, func() { parseRegressionBody(t, source) })
	}
}
