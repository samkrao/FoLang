package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// TestMethodCallsUseOneASTShape verifies that built-in candidates and ordinary
// methods differ only in metadata. Both invocations must remain CallExpr nodes
// with a MemberExpr callee so argument and receiver evaluation are represented
// uniformly.
func TestMethodCallsUseOneASTShape(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantKind ast.CallKind
	}{
		{
			name:     "reserved-built-in-candidate",
			source:   "items.map(transform);",
			wantKind: ast.CallBuiltInMethod,
		},
		{
			name:     "built-in-namespace-method",
			source:   "co.out.println(value);",
			wantKind: ast.CallBuiltInMethod,
		},
		{
			name:     "longest-built-in-namespace-receiver",
			source:   "co.sys.file.open(value);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "built-in-constant-receiver",
			source:   "co.const.true.to_str();",
			wantKind: ast.CallBuiltInMethod,
		},
		{
			name:     "ordinary-method",
			source:   "employee.calculate(value);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "ordinary-method-on-qualified-receiver",
			source:   "service.worker.calculate(value);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "ordinary-method-on-call-result",
			source:   "factory().calculate(value);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "reserved-candidate-on-call-result",
			source:   "factory().map(transform);",
			wantKind: ast.CallBuiltInMethod,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			call := parsedExpressionCall(t, tc.source)
			if call.CallKind != tc.wantKind {
				t.Fatalf("call kind = %v, want %v", call.CallKind, tc.wantKind)
			}
			if _, ok := call.Method.(ast.MemberExpr); !ok {
				t.Fatalf("method callee is %T, want ast.MemberExpr", call.Method)
			}
		})
	}
}

// TestNonMethodCallsReceiveSyntaxOnlyClassification documents the remaining
// parse-time categories. Resolution may refine an unresolved higher-order call
// later, but it must not require changing the CallExpr node shape.
func TestNonMethodCallsReceiveSyntaxOnlyClassification(t *testing.T) {
	direct := parsedExpressionCall(t, "calculate(value);")
	if direct.CallKind != ast.CallFunction {
		t.Fatalf("direct call kind = %v, want CallFunction", direct.CallKind)
	}

	higherOrder := parsedExpressionCall(t, "(factory())(value);")
	if higherOrder.CallKind != ast.CallUnresolved {
		t.Fatalf("higher-order call kind = %v, want CallUnresolved", higherOrder.CallKind)
	}
}

func TestMethodCallRetainsCompletedReceiverExpression(t *testing.T) {
	outer := parsedExpressionCall(t, "factory().calculate(value);")
	member, ok := outer.Method.(ast.MemberExpr)
	if !ok {
		t.Fatalf("outer callee is %T, want ast.MemberExpr", outer.Method)
	}
	inner, ok := member.Member.(ast.CallExpr)
	if !ok {
		t.Fatalf("method receiver is %T, want the factory ast.CallExpr", member.Member)
	}
	if inner.CallKind != ast.CallFunction {
		t.Fatalf("factory call kind = %v, want CallFunction", inner.CallKind)
	}
}

func TestGroupedMemberCallRetainsMethodClassification(t *testing.T) {
	call := parsedExpressionCall(t, "(items.map)(transform);")
	if call.CallKind != ast.CallBuiltInMethod {
		t.Fatalf("grouped member call kind = %v, want CallBuiltInMethod", call.CallKind)
	}
	group, ok := call.Method.(ast.GroupingExpr)
	if !ok {
		t.Fatalf("grouped callee is %T, want ast.GroupingExpr", call.Method)
	}
	if _, ok := group.Expr_.(ast.MemberExpr); !ok {
		t.Fatalf("grouped expression is %T, want ast.MemberExpr", group.Expr_)
	}
}

func TestCompletedReceiverRetainsEveryMemberBoundary(t *testing.T) {
	call := parsedExpressionCall(t, "factory().service.worker.calculate(value);")
	calculate, ok := call.Method.(ast.MemberExpr)
	if !ok {
		t.Fatalf("calculate callee is %T, want ast.MemberExpr", call.Method)
	}
	worker, ok := calculate.Member.(ast.MemberExpr)
	if !ok {
		t.Fatalf("calculate receiver is %T, want worker ast.MemberExpr", calculate.Member)
	}
	service, ok := worker.Member.(ast.MemberExpr)
	if !ok {
		t.Fatalf("worker receiver is %T, want service ast.MemberExpr", worker.Member)
	}
	if _, ok := service.Member.(ast.CallExpr); !ok {
		t.Fatalf("service receiver is %T, want factory ast.CallExpr", service.Member)
	}
}

func TestThisAndSelfCallReceiversRemainSelfReferences(t *testing.T) {
	for _, receiverName := range []string{"this", "self"} {
		t.Run(receiverName, func(t *testing.T) {
			call := parsedExpressionCall(t, receiverName+".custom(value);")
			member, ok := call.Method.(ast.MemberExpr)
			if !ok {
				t.Fatalf("callee is %T, want ast.MemberExpr", call.Method)
			}
			receiver, ok := member.Member.(ast.SymbolExpr)
			if !ok {
				t.Fatalf("receiver is %T, want ast.SymbolExpr", member.Member)
			}
			if receiver.SymbolType_ != "self-reference" {
				t.Fatalf("receiver symbol type = %q, want self-reference", receiver.SymbolType_)
			}
		})
	}
}

// TestMethodTokenRemainsContextual ensures METHOD_CALL prevents name folding
// only in expression-call position. A qualified constructor followed by "(" is
// still one qualified name in pattern and data-declaration contexts.
func TestMethodTokenRemainsContextual(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, "constructorPattern(pkg.Some(value)) => value;")
	})
	mustNotPanic(t, func() {
		// A data declaration is a unit member, so it is exercised in a unit file.
		parseRegressionFile(t,
			"_ co.lang.unit = { Maybe co.lang.data = pkg.Some(co.lang.int); }",
			"maybe.unit.fol")
	})
}

func parsedExpressionCall(t *testing.T, source string) ast.CallExpr {
	t.Helper()

	body := parseRegressionBody(t, source)
	if len(body) != 1 {
		t.Fatalf("parsed %d statements, want 1", len(body))
	}
	expression, ok := body[0].(ast.ExpressionStmt)
	if !ok {
		t.Fatalf("statement is %T, want ast.ExpressionStmt", body[0])
	}
	call, ok := expression.Expression.(ast.CallExpr)
	if !ok {
		t.Fatalf("expression is %T, want ast.CallExpr", expression.Expression)
	}
	return call
}
