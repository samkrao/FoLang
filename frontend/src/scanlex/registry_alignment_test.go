package scanlex

import (
	"slices"
	"testing"
)

func TestReferencePackageRegistrySpellings(t *testing.T) {
	for _, name := range []string{"co.hokrlt", "co.cpca", "co.utils", "co.compiletime", "co.operator", "co.pdap"} {
		if kind, ok := classifyBuiltInName(name); !ok || kind != BUIL_IN_STMT_EXPRS {
			t.Errorf("%s classification = (%v, %v), want BUIL_IN_STMT_EXPRS", name, kind, ok)
		}
	}
	for _, withdrawn := range []string{"co.hokrtl", "co.hokrt", "co.nop", "co.comptime"} {
		if _, ok := classifyBuiltInName(withdrawn); ok {
			t.Errorf("non-reference package spelling %s is still registered", withdrawn)
		}
	}
}

func TestErrorIsABuiltinDataType(t *testing.T) {
	if kind, ok := classifyBuiltInName("co.lang.error"); !ok || kind != BUILT_IN_TYPE {
		t.Fatalf("co.lang.error classification = (%v, %v), want BUILT_IN_TYPE", kind, ok)
	}
}

func TestPredicateTypeRegistryAdditions(t *testing.T) {
	tests := []struct {
		name string
		kind TokenKind
	}{
		{"co.lang.predicateType", BUILT_IN_KIND},
		{"co.lang.number", BUILT_IN_TYPE},
	}
	for _, test := range tests {
		if kind, ok := classifyBuiltInName(test.name); !ok || kind != test.kind {
			t.Errorf("%s classification = (%v, %v), want %v", test.name, kind, ok, test.kind)
		}
	}
	if members := Built_in_stmt_exprs["co.core"]; !slices.Contains(members, "Comparable") {
		t.Errorf("co.core registry members = %v, want Comparable", members)
	}
}
