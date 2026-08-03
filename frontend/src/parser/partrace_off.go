//go:build !partrace

package parser

// Parse tracing — disabled build.
//
// This is the variant compiled into every ordinary build. traceEnabled is an
// untyped constant false, so each `if traceEnabled { … }` guard in the parser is
// dead code the compiler removes outright: no call, no defer record, no field
// access. The instrumentation therefore costs nothing and cannot change parser
// behaviour unless the partrace tag is supplied.
//
// The declarations below exist only so the guarded call sites type-check. See
// partrace_on.go for the recording implementation and the trace file format.

// traceEnabled reports whether parse tracing is compiled in.
const traceEnabled = false

// traceMark is the empty stand-in for a recorded span start.
type traceMark struct{}

// traceBegin is never called: every call site sits behind `if traceEnabled`.
func (p *parser) traceBegin() traceMark { return traceMark{} }

// traceEnd is never called: every call site sits behind `if traceEnabled`.
func (p *parser) traceEnd(traceMark) {}

// traceBail is never called: its call site sits behind `if traceEnabled`.
func (p *parser) traceBail() {}

// traceSource is never called: its call site sits behind `if traceEnabled`.
func (p *parser) traceSource(string) {}
