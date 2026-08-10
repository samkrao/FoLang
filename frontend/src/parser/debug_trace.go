package parser

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// DEBUG_TRACE enables the human-readable hierarchical parser trace. It is off
// by default and must remain off for normal compiler and LSP operation.
// Trace output is written only to stderr.
var DEBUG_TRACE = false

var (
	debugTraceMu     sync.Mutex
	debugTraceOutput io.Writer = os.Stderr
)

type debugTraceMark struct {
	name  string
	depth int
}

func (p *parser) debugTraceBegin(callerSkip int) debugTraceMark {
	if !DEBUG_TRACE {
		return debugTraceMark{}
	}
	name := "unknown"
	if pc, _, _, ok := runtime.Caller(callerSkip); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			name = shortDebugFunctionName(fn.Name())
		}
	}
	mark := debugTraceMark{name: name, depth: p.indentLevel}
	p.writeDebugTrace("ENTER", mark, p.cur())
	p.indentLevel++
	return mark
}

func (p *parser) debugTraceEnd(mark debugTraceMark) {
	if !DEBUG_TRACE || mark.name == "" {
		return
	}
	if p.indentLevel > 0 {
		p.indentLevel--
	}
	p.writeDebugTrace("EXIT ", mark, p.cur())
}

func (p *parser) writeDebugTrace(event string, mark debugTraceMark, tok scanlex.Token) {
	line, column := 0, 0
	if tok.StartPos != nil {
		line, column = tok.StartPos.Ln, tok.StartPos.Col
	}
	debugTraceMu.Lock()
	defer debugTraceMu.Unlock()
	fmt.Fprintf(debugTraceOutput, "[parser] %s%s %s token=%s literal=%q line=%d column=%d\n",
		strings.Repeat("  ", mark.depth), event, mark.name,
		scanlex.TokenKindString(tok.Kind), tok.Value, line, column)
}

func shortDebugFunctionName(full string) string {
	if dot := strings.LastIndexByte(full, '.'); dot >= 0 {
		full = full[dot+1:]
	}
	return strings.TrimSuffix(full, "-fm")
}
