package parser

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDebugTraceIsHierarchicalAndStderrSafe(t *testing.T) {
	if debugTraceOutput != os.Stderr {
		t.Fatal("production debug trace sink is not stderr")
	}
	var output bytes.Buffer
	oldOutput, oldEnabled := debugTraceOutput, DEBUG_TRACE
	debugTraceOutput, DEBUG_TRACE = &output, true
	t.Cleanup(func() {
		debugTraceOutput, DEBUG_TRACE = oldOutput, oldEnabled
	})

	result := ParseFile("value := 1 + 2;", "trace.fol", "", "trace.fol", "")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("trace fixture failed to parse: %v", result.Diagnostics)
	}

	trace := output.String()
	for _, want := range []string{
		"ENTER parseExpression",
		"ENTER nud",
		"ENTER led",
		"token=number",
		"literal=\"1\"",
		"line=1 column=",
	} {
		if !strings.Contains(trace, want) {
			t.Errorf("trace does not contain %q:\n%s", want, trace)
		}
	}
	if !strings.Contains(trace, "[parser]   ENTER") {
		t.Errorf("trace has no hierarchical indentation:\n%s", trace)
	}
}
