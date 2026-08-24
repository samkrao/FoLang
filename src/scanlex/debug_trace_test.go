package scanlex

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDebugTraceIncludesTokenMetadataAndUsesStderr(t *testing.T) {
	if debugTraceOutput != os.Stderr {
		t.Fatal("production debug trace sink is not stderr")
	}
	var output bytes.Buffer
	oldOutput, oldEnabled := debugTraceOutput, DEBUG_TRACE
	debugTraceOutput, DEBUG_TRACE = &output, true
	t.Cleanup(func() {
		debugTraceOutput, DEBUG_TRACE = oldOutput, oldEnabled
	})

	tokens, diagnostics := TokenizeCollecting("name := 1;", "trace.fol", nil)
	if len(diagnostics) != 0 || len(tokens) == 0 {
		t.Fatalf("trace fixture failed to tokenize: %v", diagnostics)
	}

	trace := output.String()
	for _, want := range []string{
		"ENTER tokenize",
		"  ENTER scanToken",
		"ENTER emitToken token=identifier literal=\"name\" line=1 column=1",
	} {
		if !strings.Contains(trace, want) {
			t.Errorf("trace does not contain %q:\n%s", want, trace)
		}
	}
}
