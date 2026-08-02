package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

type wantedToken struct {
	kind  scanlex.TokenKind
	value string
}

func assertTokenStream(t *testing.T, source string, want []wantedToken) {
	t.Helper()
	tokens := meaningful(tokenize(source))
	if len(tokens) != len(want) {
		t.Fatalf("Tokenize(%q) produced %d tokens, want %d: %#v", source, len(tokens), len(want), tokens)
	}
	for i, expected := range want {
		if tokens[i].Kind != expected.kind || tokens[i].Value != expected.value {
			t.Errorf(
				"Tokenize(%q)[%d] = %s(%q), want %s(%q)",
				source,
				i,
				scanlex.TokenKindString(tokens[i].Kind),
				tokens[i].Value,
				scanlex.TokenKindString(expected.kind),
				expected.value,
			)
		}
	}
}

func TestDottedMethodCallsHaveUniformTokenShape(t *testing.T) {
	argumentTail := []wantedToken{
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
	}

	tests := []struct {
		name     string
		source   string
		receiver wantedToken
		method   wantedToken
	}{
		{
			name:     "ordinary-on-one-segment-receiver",
			source:   "employee.calculate(value)",
			receiver: wantedToken{scanlex.IDENTIFIER, "employee_fo"},
			method:   wantedToken{scanlex.METHOD_CALL, "calculate_fo"},
		},
		{
			name:     "reserved-on-one-segment-receiver",
			source:   "items.map(value)",
			receiver: wantedToken{scanlex.IDENTIFIER, "items_fo"},
			method:   wantedToken{scanlex.BUILT_IN_METHOD, "map"},
		},
		{
			name:     "ordinary-on-keyword-receiver",
			source:   "this.custom(value)",
			receiver: wantedToken{scanlex.BUIL_IN_STMT_EXPRS, "this"},
			method:   wantedToken{scanlex.METHOD_CALL, "custom_fo"},
		},
		{
			name:     "reserved-on-keyword-receiver",
			source:   "this.map(value)",
			receiver: wantedToken{scanlex.BUIL_IN_STMT_EXPRS, "this"},
			method:   wantedToken{scanlex.BUILT_IN_METHOD, "map"},
		},
		{
			name:     "ordinary-on-builtin-namespace",
			source:   "co.out.render(value)",
			receiver: wantedToken{scanlex.BUIL_IN_STMT_EXPRS, "co.out"},
			method:   wantedToken{scanlex.METHOD_CALL, "render_fo"},
		},
		{
			name:     "reserved-on-builtin-namespace",
			source:   "co.out.println(value)",
			receiver: wantedToken{scanlex.BUIL_IN_STMT_EXPRS, "co.out"},
			method:   wantedToken{scanlex.BUILT_IN_METHOD, "println"},
		},
		{
			name:     "ordinary-on-qualified-receiver",
			source:   "service.worker.render(value)",
			receiver: wantedToken{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker"},
			method:   wantedToken{scanlex.METHOD_CALL, "render_fo"},
		},
		{
			name:     "reserved-on-qualified-receiver",
			source:   "service.worker.map(value)",
			receiver: wantedToken{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker"},
			method:   wantedToken{scanlex.BUILT_IN_METHOD, "map"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := []wantedToken{tc.receiver, {scanlex.DOT, "."}, tc.method}
			want = append(want, argumentTail...)
			assertTokenStream(t, tc.source, want)
		})
	}
}

func TestMethodCallAfterCompletedExpressionUsesMethodToken(t *testing.T) {
	assertTokenStream(t, "factory().render(value)", []wantedToken{
		{scanlex.IDENTIFIER, "factory_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.CLOSE_PAREN, ")"},
		{scanlex.DOT, "."},
		{scanlex.METHOD_CALL, "render_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
	})
	assertTokenStream(t, "factory().map(value)", []wantedToken{
		{scanlex.IDENTIFIER, "factory_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.CLOSE_PAREN, ")"},
		{scanlex.DOT, "."},
		{scanlex.BUILT_IN_METHOD, "map"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
	})
}

func TestNonCallQualifiedAndMemberReferencesKeepTheirFolding(t *testing.T) {
	assertTokenStream(t, "service.worker.render", []wantedToken{
		{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker_fo.render"},
	})
	assertTokenStream(t, "employee.calculate", []wantedToken{
		{scanlex.COMPOSITE_IDENTIFER, "employee_fo.calculate"},
	})
	assertTokenStream(t, "service.worker.map", []wantedToken{
		{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker"},
		{scanlex.DOT, "."},
		{scanlex.BUILT_IN_METHOD, "map"},
	})
	assertTokenStream(t, "co.out.render", []wantedToken{
		{scanlex.BUIL_IN_STMT_EXPRS, "co.out.render"},
	})
}

func TestReturnStatementBuiltinIsNotMistakenForMethodCall(t *testing.T) {
	assertTokenStream(t, "this.return (value);", []wantedToken{
		{scanlex.BUIL_IN_STMT_EXPRS, "this.return"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
		{scanlex.SEMI_COLON, ";"},
	})
}
