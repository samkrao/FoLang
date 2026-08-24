package parser

import (
	"bytes"
	"strings"
	"testing"
)

// TestDebugTraceEntryAndExitNestBalanced asserts the property the whole trace
// depends on: every ENTER is matched by an EXIT at the same depth, and the trace
// returns to depth zero.
//
// The depth is carried on the parser and adjusted by a deferred call, so the two
// ways it can break are silent. A parse that bails out unwinds through the
// deferred traceEnd, and a speculative parse that is thrown away still has to
// close the frames it opened; if either failed to decrement, every later line
// would be indented one level too deep for the rest of the file rather than
// producing an error anyone would notice.
func TestDebugTraceEntryAndExitNestBalanced(t *testing.T) {
	var buf bytes.Buffer
	previousOutput := debugTraceOutput
	debugTraceOutput = &buf
	DEBUG_TRACE = true
	defer func() { DEBUG_TRACE = false; debugTraceOutput = previousOutput }()

	// A unit with a body exercises speculation (the type/expression readings a
	// binding is tried under) as well as ordinary nesting.
	ParseFile("_ co.lang.unit = {\n"+
		"  Fn co.lang.type = (co.lang.int)->(co.lang.int);\n"+
		"  f()->(co.lang.int) = { this.return 1; }\n"+
		"}\n",
		"t", ".", "probe.unit.fol", "")

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 10 {
		t.Fatalf("trace too short: %d lines\n%s", len(lines), buf.String())
	}
	depth := 0
	for i, ln := range lines {
		body := strings.TrimPrefix(ln, "[parser] ")
		indent := len(body) - len(strings.TrimLeft(body, " "))
		if indent%2 != 0 {
			t.Fatalf("line %d has odd indent: %q", i, ln)
		}
		switch {
		case strings.HasPrefix(strings.TrimLeft(body, " "), "ENTER"):
			if indent/2 != depth {
				t.Fatalf("line %d ENTER at indent %d, want %d\n%s", i, indent/2, depth, ln)
			}
			depth++
		case strings.HasPrefix(strings.TrimLeft(body, " "), "EXIT"):
			depth--
			if indent/2 != depth {
				t.Fatalf("line %d EXIT at indent %d, want %d\n%s", i, indent/2, depth, ln)
			}
		}
	}
	if depth != 0 {
		t.Fatalf("trace ended at depth %d, want 0", depth)
	}
	t.Logf("%d balanced trace lines; first 6:\n%s", len(lines), strings.Join(lines[:6], "\n"))
}
