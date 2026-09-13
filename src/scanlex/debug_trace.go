package scanlex

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// DEBUG_TRACE enables the human-readable hierarchical lexer trace. It is off
// by default. Output is always directed to stderr, never stdout, so enabling it
// cannot corrupt an LSP JSON-RPC stream.
var DEBUG_TRACE = false

var (
	debugTraceMu     sync.Mutex
	debugTraceOutput io.Writer = os.Stderr
)

type debugTraceMark struct {
	name   string
	depth  int
	kind   TokenKind
	value  string
	line   int
	column int
}

func (lex *lexer) debugTraceBegin(name string, kind TokenKind, value string) debugTraceMark {
	if !DEBUG_TRACE {
		return debugTraceMark{}
	}
	mark := debugTraceMark{name: name, depth: lex.indentLevel, kind: kind, value: value, line: lex.line, column: lex.col}
	lex.writeDebugTrace("ENTER", mark)
	lex.indentLevel++
	return mark
}

func (lex *lexer) debugTraceEnd(mark debugTraceMark) {
	if !DEBUG_TRACE || mark.name == "" {
		return
	}
	if lex.indentLevel > 0 {
		lex.indentLevel--
	}
	lex.writeDebugTrace("EXIT ", mark)
}

func (lex *lexer) writeDebugTrace(event string, mark debugTraceMark) {
	debugTraceMu.Lock()
	defer debugTraceMu.Unlock()
	fmt.Fprintf(debugTraceOutput, "[lexer] %s%s %s token=%s literal=%q line=%d column=%d\n",
		strings.Repeat("  ", mark.depth), event, mark.name,
		TokenKindString(mark.kind), mark.value, mark.line, mark.column)
}
