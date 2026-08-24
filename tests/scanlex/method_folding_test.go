package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/src/scanlex"
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

// TestDottedMethodCallsHaveUniformTokenShape fixes the shape of a dotted
// invocation: receiver, DOT, method, whatever the receiver is and whatever the
// member is named.
//
// Every method here folds to METHOD_CALL. scanlex.Reserved_me — the lexically
// reserved built-in method registry — is EMPTY in the current profile: the
// reference removed its Builtin Methods section, and member-suffix now admits
// any identifier other than `match`, which match-suffix owns. `map`, `println`
// and `to_str` are ordinary members; `println` in particular is a member of the
// `co.out` object rather than a reserved spelling, which is why only its
// RECEIVER is a built-in token here. BUILT_IN_METHOD remains in the token kinds
// for a future registry entry, and nothing in this profile emits it.
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
			name:     "collection-operation-on-one-segment-receiver",
			source:   "items.map(value)",
			receiver: wantedToken{scanlex.IDENTIFIER, "items_fo"},
			method:   wantedToken{scanlex.METHOD_CALL, "map_fo"},
		},
		{
			name:     "ordinary-on-keyword-receiver",
			source:   "this.custom(value)",
			receiver: wantedToken{scanlex.KEYWORD, "this"},
			method:   wantedToken{scanlex.METHOD_CALL, "custom_fo"},
		},
		{
			name:     "collection-operation-on-keyword-receiver",
			source:   "this.map(value)",
			receiver: wantedToken{scanlex.KEYWORD, "this"},
			method:   wantedToken{scanlex.METHOD_CALL, "map_fo"},
		},
		{
			name:     "ordinary-on-contextual-self-receiver",
			source:   "self.custom(value)",
			receiver: wantedToken{scanlex.CONTEXT_KEYWORD, "self"},
			method:   wantedToken{scanlex.METHOD_CALL, "custom_fo"},
		},
		{
			name:     "ordinary-on-builtin-namespace",
			source:   "co.out.render(value)",
			receiver: wantedToken{scanlex.BUIL_IN_STMT_EXPRS, "co.out"},
			method:   wantedToken{scanlex.METHOD_CALL, "render_fo"},
		},
		{
			name:     "namespace-member-on-builtin-namespace",
			source:   "co.out.println(value)",
			receiver: wantedToken{scanlex.BUIL_IN_STMT_EXPRS, "co.out"},
			method:   wantedToken{scanlex.METHOD_CALL, "println_fo"},
		},
		{
			name:     "ordinary-on-qualified-receiver",
			source:   "service.worker.render(value)",
			receiver: wantedToken{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker"},
			method:   wantedToken{scanlex.METHOD_CALL, "render_fo"},
		},
		{
			name:     "collection-operation-on-qualified-receiver",
			source:   "service.worker.map(value)",
			receiver: wantedToken{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker"},
			method:   wantedToken{scanlex.METHOD_CALL, "map_fo"},
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
		{scanlex.METHOD_CALL, "map_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
	})
}

func TestLongestRegisteredBuiltinReceiverIsPreserved(t *testing.T) {
	assertTokenStream(t, "co.sys.file.open(value)", []wantedToken{
		{scanlex.BUIL_IN_STMT_EXPRS, "co.sys.file"},
		{scanlex.DOT, "."},
		{scanlex.METHOD_CALL, "open_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
	})
	assertTokenStream(t, "co.const.true.to_str()", []wantedToken{
		{scanlex.BUILT_IN_CONSTANTS, "co.const.true"},
		{scanlex.DOT, "."},
		{scanlex.METHOD_CALL, "to_str_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.CLOSE_PAREN, ")"},
	})
	assertTokenStream(t, "co.sys.file.handle.open(value)", []wantedToken{
		{scanlex.BUIL_IN_STMT_EXPRS, "co.sys.file"},
		{scanlex.DOT, "."},
		{scanlex.IDENTIFIER, "handle_fo"},
		{scanlex.DOT, "."},
		{scanlex.METHOD_CALL, "open_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
	})
	assertTokenStream(t, "co.sys.file", []wantedToken{
		{scanlex.BUIL_IN_STMT_EXPRS, "co.sys.file"},
	})
}

func TestCompletedReceiverKeepsEveryMemberBoundary(t *testing.T) {
	assertTokenStream(t, "factory().service.worker.calculate(value)", []wantedToken{
		{scanlex.IDENTIFIER, "factory_fo"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.CLOSE_PAREN, ")"},
		{scanlex.DOT, "."},
		{scanlex.IDENTIFIER, "service_fo"},
		{scanlex.DOT, "."},
		{scanlex.IDENTIFIER, "worker_fo"},
		{scanlex.DOT, "."},
		{scanlex.METHOD_CALL, "calculate_fo"},
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
		{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker_fo.map"},
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
	assertTokenStream(t, "self.return (value);", []wantedToken{
		{scanlex.BUIL_IN_STMT_EXPRS, "self.return"},
		{scanlex.OPEN_PAREN, "("},
		{scanlex.IDENTIFIER, "value_fo"},
		{scanlex.CLOSE_PAREN, ")"},
		{scanlex.SEMI_COLON, ";"},
	})
}
