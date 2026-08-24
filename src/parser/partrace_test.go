//go:build partrace

package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/foerrors"
)

// TestTraceRecordsConsumedSpan checks the core claim: a successful parse
// function reports exactly the source it consumed, spelled as it appears in the
// file rather than reconstructed from tokens.
func TestTraceRecordsConsumedSpan(t *testing.T) {
	snippets := traceRun(t, "Color co.lang.enum = {\n    Red,\n    Green = 2,\n}\n")

	variants := snippets["parseEnumVariant"]
	if len(variants) == 0 {
		t.Fatal("parseEnumVariant recorded no span")
	}
	got := strings.Join(variants, "|")
	for _, want := range []string{"Red", "Green = 2"} {
		if !strings.Contains(got, want) {
			t.Errorf("parseEnumVariant spans %q, want one equal to %q", got, want)
		}
	}
}

// TestTraceSkipsFailedParse is the "on success" requirement. A bailout unwinds
// through the deferred traceEnd, so without the bailout counter the aborted
// function would report the partial, invalid span it had consumed.
func TestTraceSkipsFailedParse(t *testing.T) {
	// The enum body is unterminated, so parsing the declaration aborts.
	snippets := traceRun(t, "Color co.lang.enum = {\n    Red,\n")

	if spans, ok := snippets["parseEnumDeclaration"]; ok {
		t.Fatalf("aborted parseEnumDeclaration recorded %q, want no span", spans)
	}
}

// TestTraceKeepsSuccessesBeforeAFailure confirms the previous test is not
// passing merely because nothing was recorded at all: work completed before the
// bailout is still reported.
func TestTraceKeepsSuccessesBeforeAFailure(t *testing.T) {
	snippets := traceRun(t, "Color co.lang.enum = {\n    Red,\n")

	if len(snippets) == 0 {
		t.Fatal("a file that fails to parse recorded nothing at all")
	}
	if spans, ok := snippets["parseEnumVariant"]; !ok || len(spans) == 0 {
		t.Fatalf("parseEnumVariant completed before the failure but recorded %q", spans)
	}
}

// TestTraceExcludesSpeculativeParses covers the other correctness rule: a
// tentative parse that is rewound describes text the parser did not accept.
// `blockormacro co.lang.kind = block | macro;` is parsed by speculating on a
// type-expression binding, so a leaked speculative span would show up here.
func TestTraceExcludesSpeculativeParses(t *testing.T) {
	snippets := traceRun(t, "blockormacro co.lang.kind = block | macro;\n")

	for name, spans := range snippets {
		for _, span := range spans {
			if strings.HasSuffix(span, ";;") || strings.Contains(span, "= block | macro;\n") {
				t.Errorf("%s recorded a span that looks rewound: %q", name, span)
			}
		}
	}
}

// traceRun parses one source in isolation and returns the recorded spans.
func traceRun(t *testing.T, source string) map[string][]string {
	t.Helper()

	restore := foerrors.GenPanic
	foerrors.GenPanic = true
	defer func() { foerrors.GenPanic = restore }()

	ResetTrace()
	func() {
		// A deliberately malformed source aborts; that is the case under test.
		defer func() { _ = recover() }()
		Parse(source, "trace", ".", "trace.fol", "", "program", "program", true)
	}()
	return TraceSnippets(5)
}
