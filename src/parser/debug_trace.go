package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/samkrao/fo-lang/src/scanlex"
)

// DEBUG_TRACE enables the human-readable hierarchical parser trace. It is off
// by default and must remain off for normal compiler and LSP operation.
// Trace output is written only to stderr.
var DEBUG_TRACE = false

var (
	debugTraceMu     sync.Mutex
	debugTraceOutput io.Writer = os.Stderr
	debugTraceEvents []debugTraceEvent
)

// debugTraceEvent is the machine-readable counterpart of one human trace line.
// Sequence preserves the exact call flow; matching depth/function ENTER and EXIT
// events make the resulting JSON readable as either a flat timeline or a tree.
type debugTraceEvent struct {
	Sequence int    `json:"sequence"`
	Event    string `json:"event"`
	Function string `json:"function"`
	Depth    int    `json:"depth"`
	Token    string `json:"token"`
	Literal  string `json:"literal,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type serializedDebugTrace struct {
	Kind   string            `json:"kind"`
	Events []debugTraceEvent `json:"events"`
}

const debugTraceArtifactExtension = ".trace.json"

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
	line, column, file := 0, 0, ""
	if tok.StartPos != nil {
		line, column = tok.StartPos.Ln, tok.StartPos.Col
		file = tok.StartPos.Fn
	}
	debugTraceMu.Lock()
	defer debugTraceMu.Unlock()
	debugTraceEvents = append(debugTraceEvents, debugTraceEvent{
		Sequence: len(debugTraceEvents) + 1,
		Event:    strings.TrimSpace(event),
		Function: mark.name,
		Depth:    mark.depth,
		Token:    scanlex.TokenKindString(tok.Kind),
		Literal:  tok.Value,
		File:     file,
		Line:     line,
		Column:   column,
	})
	fmt.Fprintf(debugTraceOutput, "[parser] %s%s %s token=%s literal=%q line=%d column=%d\n",
		strings.Repeat("  ", mark.depth), event, mark.name,
		scanlex.TokenKindString(tok.Kind), tok.Value, line, column)
}

func resetDebugTraceEvents() {
	debugTraceMu.Lock()
	defer debugTraceMu.Unlock()
	debugTraceEvents = nil
}

// writeDebugTraceArtifact writes the parser call flow beside the AST artifact.
// Tracing remains opt-in through folang-debug.json; normal parser/LSP calls do
// not allocate an artifact or touch the project tree.
func writeDebugTraceArtifact(artifact astArtifact) (string, error) {
	if !DEBUG_TRACE || artifact.Root == "" || artifact.Stem == "" {
		return "", nil
	}

	debugTraceMu.Lock()
	events := append([]debugTraceEvent(nil), debugTraceEvents...)
	debugTraceMu.Unlock()

	encoded, err := json.MarshalIndent(serializedDebugTrace{Kind: "parser-function-flow", Events: events}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding parser debug trace: %w", err)
	}
	directory := filepath.Join(artifact.Root, "build")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("creating build domain for parser debug trace: %w", err)
	}
	path := filepath.Join(directory, artifact.Stem+debugTraceArtifactExtension)
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return "", fmt.Errorf("writing parser debug trace: %w", err)
	}
	return path, nil
}

func shortDebugFunctionName(full string) string {
	if dot := strings.LastIndexByte(full, '.'); dot >= 0 {
		full = full[dot+1:]
	}
	return strings.TrimSuffix(full, "-fm")
}
