//go:build partrace

package parser

import (
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Parse tracing — recording build (`-tags partrace`).
//
// Every instrumented parse function opens with
//
//	if traceEnabled { defer p.traceEnd(p.traceBegin()) }
//
// traceBegin runs when the defer statement executes, so it captures the cursor
// on entry; traceEnd runs on return, when the cursor sits just past whatever the
// function consumed. The pair therefore yields the exact source span, and the
// function's own name comes from the call stack rather than a string literal —
// there is no name to keep in sync with a rename.
//
// Three cases are deliberately NOT recorded:
//
//	bailout      A failed parse panics (see diagnostics.go) and unwinds straight
//	             through the deferred traceEnd. fail/failf bump a counter, so a
//	             span whose counter moved between begin and end is discarded:
//	             only successful parses are recorded, as required.
//	speculation  speculate() rewinds the cursor when a tentative parse is thrown
//	             away, and lookahead never keeps its work at all. A span produced
//	             while p.speculating > 0 describes text the parser did not
//	             actually accept, so it is dropped.
//	empty span   A function that returns without consuming a token has no source
//	             to show.
//
// State is held here rather than on the parser struct so that no field exists in
// an ordinary build; partrace_off.go supplies the same method set as no-ops.

// traceEnabled reports whether parse tracing is compiled in.
const traceEnabled = true

// maxSnippetsPerFunction bounds how many distinct spans are retained per
// function. Selection down to the reported few happens afterwards, but a long
// corpus would otherwise accumulate one entry per call site.
const maxSnippetsPerFunction = 512

// traceMark is the cursor and bailout counter captured on entry to a function.
type traceMark struct {
	pos  int
	bail uint64
	name string
}

// traceState is the per-parser recording context.
type traceState struct {
	source string
	bail   uint64
}

var (
	traceMu       sync.Mutex
	traceStates   = map[*parser]*traceState{}
	traceSnippets = map[string]map[string]struct{}{}
)

// traceSource registers the source text a parser is about to read. The span
// offsets carried by tokens index into exactly this string.
func (p *parser) traceSource(source string) {
	traceMu.Lock()
	defer traceMu.Unlock()
	traceStates[p] = &traceState{source: source}
}

// traceBail records that a parse is aborting, so any span still open on the
// stack is discarded rather than reported as a successful parse.
func (p *parser) traceBail() {
	traceMu.Lock()
	defer traceMu.Unlock()
	if st := traceStates[p]; st != nil {
		st.bail++
	}
}

// traceBegin captures the entry cursor and the calling function's name.
func (p *parser) traceBegin() traceMark {
	mark := traceMark{pos: p.pos}

	// Skip 1 frame: the caller of traceBegin is the parse function being traced.
	if pc, _, _, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			mark.name = shortFunctionName(fn.Name())
		}
	}

	traceMu.Lock()
	defer traceMu.Unlock()
	if st := traceStates[p]; st != nil {
		mark.bail = st.bail
	}
	return mark
}

// traceEnd records the span consumed since mark, unless the parse did not
// succeed, was speculative, or consumed nothing.
func (p *parser) traceEnd(mark traceMark) {
	if mark.name == "" || p.speculating > 0 || p.pos <= mark.pos {
		return
	}

	traceMu.Lock()
	defer traceMu.Unlock()

	st := traceStates[p]
	if st == nil || st.bail != mark.bail {
		return
	}

	snippet := strings.TrimSpace(p.traceSpanText(st.source, mark.pos, p.pos))
	if snippet == "" {
		return
	}

	set := traceSnippets[mark.name]
	if set == nil {
		set = map[string]struct{}{}
		traceSnippets[mark.name] = set
	}
	if _, seen := set[snippet]; !seen && len(set) < maxSnippetsPerFunction {
		set[snippet] = struct{}{}
	}
}

// traceSpanText slices the source covered by tokens [from, to). Offsets come
// from the tokens themselves, so the snippet keeps its original spelling and
// spacing rather than being reconstructed from token values.
func (p *parser) traceSpanText(source string, from, to int) string {
	if from < 0 || to > len(p.toks) || from >= to {
		return ""
	}
	start, startOK := traceOffset(p.toks[from], true)
	end, endOK := traceOffset(p.toks[to-1], false)
	if !startOK || !endOK || start < 0 || end > len(source) || start >= end {
		return ""
	}
	return source[start:end]
}

// traceOffset reads a token's start or end byte offset.
func traceOffset(tok scanlex.Token, start bool) (int, bool) {
	if start {
		if tok.StartPos == nil {
			return 0, false
		}
		return tok.StartPos.Idx, true
	}
	if tok.EndPos == nil {
		return 0, false
	}
	return tok.EndPos.Idx, true
}

// shortFunctionName reduces a fully qualified name such as
// ".../src/parser.(*parser).parseBlock" to "parseBlock".
func shortFunctionName(full string) string {
	if dot := strings.LastIndexByte(full, '.'); dot >= 0 {
		full = full[dot+1:]
	}
	return strings.TrimSuffix(full, "-fm")
}

// TraceSnippets returns the recorded spans per function, deduplicated, shortest
// first, capped at limit per function. It is the input cmd/docgen writes to
// docs/trace.json, and is available only in a partrace build.
func TraceSnippets(limit int) map[string][]string {
	traceMu.Lock()
	defer traceMu.Unlock()

	out := make(map[string][]string, len(traceSnippets))
	for name, set := range traceSnippets {
		snippets := make([]string, 0, len(set))
		for snippet := range set {
			snippets = append(snippets, snippet)
		}
		// Shortest first; equal lengths ordered lexically so a rerun over the
		// same corpus produces byte-identical output.
		sort.Slice(snippets, func(i, j int) bool {
			if len(snippets[i]) != len(snippets[j]) {
				return len(snippets[i]) < len(snippets[j])
			}
			return snippets[i] < snippets[j]
		})
		if limit > 0 && len(snippets) > limit {
			snippets = snippets[:limit]
		}
		out[name] = snippets
	}
	return out
}

// ResetTrace clears recorded state. docgen calls it between corpus runs.
func ResetTrace() {
	traceMu.Lock()
	defer traceMu.Unlock()
	traceStates = map[*parser]*traceState{}
	traceSnippets = map[string]map[string]struct{}{}
}
