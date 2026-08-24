package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

// TestMethodCallsUseOneASTShape verifies that every dotted invocation is one
// CallExpr with a MemberExpr callee, so argument and receiver evaluation are
// represented uniformly whatever the member is named.
//
// Every case here classifies as CallMethod. The current profile's reserved
// built-in method registry (scanlex.Reserved_me) is EMPTY: the reference removed
// its Builtin Methods section, and member-suffix now admits any identifier other
// than `match`, which match-suffix owns. `map`, `println` and `to_str` are
// therefore ordinary members reached through ordinary member syntax —
// `println` is a member of the `co.out` object, not a lexically reserved method
// spelling. ast.CallBuiltInMethod stays in the AST as the candidate
// classification a future registry entry would produce; nothing in the current
// profile produces it, which is what the CallMethod expectations below record.
func TestMethodCallsUseOneASTShape(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantKind ast.CallKind
	}{
		{
			name:     "collection-operation-member",
			source:   "items.map(transform);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "built-in-namespace-method",
			source:   "co.out.println(value);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "longest-built-in-namespace-receiver",
			source:   "co.sys.file.open(value);",
			wantKind: ast.CallMethod,
		},
		{
			name:     "built-in-constant-receiver",
			source:   "co.const.true.to_str();",
			wantKind: ast.CallMethod,
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
			name:     "collection-operation-on-call-result",
			source:   "factory().map(transform);",
			wantKind: ast.CallMethod,
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

// TestGroupedMemberCallRetainsMethodClassification checks that transparent
// grouping around a member does not change what the call is: `(items.map)(f)`
// classifies exactly as `items.map(f)` does.
func TestGroupedMemberCallRetainsMethodClassification(t *testing.T) {
	call := parsedExpressionCall(t, "(items.map)(transform);")
	if call.CallKind != ast.CallMethod {
		t.Fatalf("grouped member call kind = %v, want CallMethod", call.CallKind)
	}
	group, ok := call.Method.(ast.GroupingExpr)
	if !ok {
		t.Fatalf("grouped callee is %T, want ast.GroupingExpr", call.Method)
	}
	// Inside the parentheses the path is a qualified-name primary expression,
	// which is a SymbolExpr: the scanner splits a dotted path into receiver, DOT
	// and member only when "(" follows the member, and here ")" does. The
	// classification above is what makes the two spellings equivalent; the node
	// shape follows the grammar alternative each spelling actually matches.
	symbol, ok := group.Expr_.(ast.SymbolExpr)
	if !ok {
		t.Fatalf("grouped expression is %T, want the qualified-name ast.SymbolExpr", group.Expr_)
	}
	if got := strings.ReplaceAll(symbol.Value, "_fo", ""); got != "items.map" {
		t.Fatalf("grouped qualified name = %q, want %q", got, "items.map")
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

// TestThisAndSelfCallReceiversRemainSelfReferences fixes the receiver shape of a
// `this.` / `self.` call: a MemberExpr over a SymbolExpr, never a folded name.
//
// The two differ in what that symbol MEANS. `this` is hard-reserved and is always
// the receiver. `self` is contextual: it denotes the class/type receiver only
// inside a co.lang.class method or an @co.dap.class method of a target-bound
// co.lang.extension, and the source here is an entry-file statement in neither
// context — so it is an ordinary identifier spelling, which is what
// self-context-guard decides.
func TestThisAndSelfCallReceiversRemainSelfReferences(t *testing.T) {
	tests := []struct {
		receiverName string
		wantSymbol   string
	}{
		{"this", "self-reference"},
		{"self", "identifier"},
	}
	for _, tc := range tests {
		t.Run(tc.receiverName, func(t *testing.T) {
			call := parsedExpressionCall(t, tc.receiverName+".custom(value);")
			member, ok := call.Method.(ast.MemberExpr)
			if !ok {
				t.Fatalf("callee is %T, want ast.MemberExpr", call.Method)
			}
			receiver, ok := member.Member.(ast.SymbolExpr)
			if !ok {
				t.Fatalf("receiver is %T, want ast.SymbolExpr", member.Member)
			}
			if receiver.SymbolType_ != tc.wantSymbol {
				t.Fatalf("receiver symbol type = %q, want %q", receiver.SymbolType_, tc.wantSymbol)
			}
		})
	}
}

// TestSelfIsTheClassReceiverInsideAClassMethod is the other half of
// self-context-guard: inside a class body `self` IS the class/type receiver.
func TestSelfIsTheClassReceiverInsideAClassMethod(t *testing.T) {
	for _, source := range []struct {
		name     string
		basename string
		body     string
	}{
		{
			name:     "class-method",
			basename: "Worker.fol",
			body:     "_ co.lang.class = { run()->() = { self.custom(value); } }",
		},
		{
			name:     "extension-method",
			basename: "WorkerExtension.fol",
			body:     "_ co.lang.extension->(fortype=Worker) = { @co.dap.class run()->() = { self.custom(value); } }",
		},
	} {
		source := source
		t.Run(source.name, func(t *testing.T) {
			if !containsSelfReference(parseRegressionFile(t, source.body, source.basename)) {
				t.Fatal("self did not resolve to the class/type receiver inside its declared context")
			}
		})
	}
}

// containsSelfReference reports whether any node in the tree is a `self` symbol
// carrying the class/type receiver classification.
//
// The search is by reflection because `self` can sit at any depth a statement can
// reach, and the point of the test is the CLASSIFICATION rather than the path
// through the tree.
func containsSelfReference(root any) bool {
	return findSelfReference(reflect.ValueOf(root), map[uintptr]bool{})
}

func findSelfReference(v reflect.Value, seen map[uintptr]bool) bool {
	if !v.IsValid() {
		return false
	}

	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return false
		}
		if v.Kind() == reflect.Ptr {
			if seen[v.Pointer()] {
				return false
			}
			seen[v.Pointer()] = true
		}
		return findSelfReference(v.Elem(), seen)

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if findSelfReference(v.Index(i), seen) {
				return true
			}
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			if findSelfReference(v.MapIndex(key), seen) {
				return true
			}
		}

	case reflect.Struct:
		if symbol, ok := v.Interface().(ast.SymbolExpr); ok {
			if strings.TrimSuffix(symbol.Value, "_fo") == "self" && symbol.SymbolType_ == "self-reference" {
				return true
			}
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			if findSelfReference(v.Field(i), seen) {
				return true
			}
		}
	}
	return false
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
