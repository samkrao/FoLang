package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// TestDiscardWildcardCallPositions fixes the contextual meaning of `_` at call
// sites. It may discard only the key/index binding of each; the each value and a
// containment search value must remain usable expressions.
func TestDiscardWildcardCallPositions(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, "items.each(_, value).do({});")
	})
	mustNotPanic(t, func() {
		parseRegressionBody(t, "recordPattern(Employee{id: _}) => 0;")
	})

	rejected := []struct {
		name   string
		source string
	}{
		{"each-value", "items.each(index, _).do({});"},
		{"contains-value", "items.contains(_).do({});"},
		{"containsVal-value", "items.containsVal(_).do({});"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseRegressionBody(t, tc.source)
			})
		})
	}
}

// Collection callback and discard permissions belong to member calls, not to a
// spelling in isolation. Parentheses around a member access are transparent.
func TestContextualCollectionArgumentsRequireMemberCalls(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, "(items.map)(|value| => value);")
	})
	mustNotPanic(t, func() {
		body := parseRegressionBody(t, "(items.each)(_, value).do({});")
		if len(body) != 1 {
			t.Fatalf("grouped each produced %d statements, want 1", len(body))
		}
		if _, ok := body[0].(ast.ForeachStmt); !ok {
			t.Fatalf("grouped each lowered to %T, want ast.ForeachStmt", body[0])
		}
	})

	rejected := []struct {
		name   string
		source string
	}{
		{"bare-map-lambda", "map(|value| => value);"},
		{"bare-each-wildcard", "each(_, value);"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseRegressionBody(t, tc.source)
			})
		})
	}
}

// TestNonCollectionGroupsRejectTrailingCommas requires every separator in a
// record pattern, grouped declaration, or expression tuple to introduce a real
// following item. Collection literals have separate grammar and are unaffected.
func TestNonCollectionGroupsRejectTrailingCommas(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, "result := (1, 2);")
	})
	mustNotPanic(t, func() {
		parseRegressionBody(t, "recordPattern(Employee{id: value, name: _}) => value;")
	})
	mustNotPanic(t, func() {
		parseRegressionBody(t, "(x co.lang.int = 1, y co.lang.int = 2);")
	})

	rejected := []struct {
		name   string
		source string
	}{
		{"record-pattern", "recordPattern(Employee{id: value,}) => value;"},
		{"grouped-declaration", "(x co.lang.int = 1,);"},
		{"one-element-tuple", "result := (1,);"},
		{"tuple", "result := (1, 2,);"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseRegressionBody(t, tc.source)
			})
		})
	}
}

func TestMatchUsesCaseAndDefaultRatherThanOtherwise(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, `result := value.match.case(1 => "one").default("other");`)
	})

	rejected := []struct {
		name   string
		source string
	}{
		{"otherwise-arm", `result := value.match.case(1 => "one").otherwise("other");`},
		{"case-after-default", `result := value.match.case(1 => "one").default("other").case(2 => "two");`},
		{"duplicate-default", `result := value.match.case(1 => "one").default("other").default("again");`},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseRegressionBody(t, tc.source)
			})
		})
	}
}
