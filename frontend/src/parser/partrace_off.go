//go:build !partrace

package parser

// Parse tracing in an ordinary build.
//
// The shared function-entry sites remain active for DEBUG_TRACE, but no
// snippet-recording state or source-span collection is present. With
// DEBUG_TRACE false the helper returns immediately and emits nothing. See
// partrace_on.go for the additional recording implementation used by docgen.

// traceEnabled keeps the shared function-entry call sites active so the runtime
// DEBUG_TRACE switch can use them.
const traceEnabled = true

// traceMark carries only the optional human-readable debug nesting state.
type traceMark struct{ debug debugTraceMark }

// traceBegin opens an optional human-readable trace entry.
func (p *parser) traceBegin() traceMark {
	return traceMark{debug: p.debugTraceBegin(2)}
}

// traceEnd closes an optional human-readable trace entry.
func (p *parser) traceEnd(mark traceMark) { p.debugTraceEnd(mark.debug) }

// traceBail is a no-op because ordinary builds do not record accepted spans.
func (p *parser) traceBail() {}

// traceSource is a no-op because ordinary builds do not record source spans.
func (p *parser) traceSource(string) {}
