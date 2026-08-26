package parser

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/foerrors"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// bailout is the sentinel panicked by fail/failf after a diagnostic has been
// recorded. It unwinds to the nearest recovery point (see recover.go).
//
// Using a panic rather than an error return is what lets every parse function
// have the shape `func (p *parser) parseX() ast.X`: a callee that cannot make
// progress never hands back a half-built node, so no caller needs to test for
// one. The panic is always caught inside this package; it never escapes Parse.
type bailout struct{}

// fail records a diagnostic at tok and aborts the current parse.
func (p *parser) fail(tok scanlex.Token, msg string) {
	p.report(tok, msg)
	if traceEnabled || DEBUG_TRACE {
		p.traceBail()
	}
	panic(bailout{})
}

// failf records a formatted diagnostic at tok and aborts the current parse.
func (p *parser) failf(tok scanlex.Token, format string, args ...any) {
	p.report(tok, fmt.Sprintf(format, args...))
	if traceEnabled || DEBUG_TRACE {
		p.traceBail()
	}
	panic(bailout{})
}

// failExpected records the stable ExpectedToken diagnostic and aborts parsing.
func (p *parser) failExpected(tok scanlex.Token, msg string) {
	start, end := tokenSpan(p.locate(tok))
	p.record(helpers.NewExpectedTokenError(start, end, msg))
	if traceEnabled || DEBUG_TRACE {
		p.traceBail()
	}
	panic(bailout{})
}

// report records a diagnostic at tok without aborting. Use it when the parser
// can describe a problem and still produce a usable node, so that one file can
// yield several diagnostics in a single run.
func (p *parser) report(tok scanlex.Token, msg string) {
	p.reportNamed(tok, helpers.DiagnosticInvalidSyntax, "Invalid Syntax", msg)
}

func (p *parser) reportNamed(tok scanlex.Token, name helpers.DiagnosticName, heading, msg string) {
	start, end := tokenSpan(p.locate(tok))
	p.record(helpers.NewNamedDiagnostic(start, end, name, heading, msg))
}

func (p *parser) failNamed(tok scanlex.Token, name helpers.DiagnosticName, heading, msg string) {
	p.reportNamed(tok, name, heading, msg)
	if traceEnabled || DEBUG_TRACE {
		p.traceBail()
	}
	panic(bailout{})
}

func (p *parser) reportNamedf(tok scanlex.Token, name helpers.DiagnosticName, heading, format string, args ...any) {
	p.reportNamed(tok, name, heading, fmt.Sprintf(format, args...))
}

func (p *parser) failNamedf(tok scanlex.Token, name helpers.DiagnosticName, heading, format string, args ...any) {
	p.failNamed(tok, name, heading, fmt.Sprintf(format, args...))
}

// record appends a diagnostic, enforcing the MaxParseErrors cap.
//
// Recovery resynchronises at item boundaries, so one mistake early in a large
// file can cascade into a diagnostic per surviving item. Past the first few
// dozen, the remainder tell a reader nothing they cannot get by fixing the ones
// above — and for an editor consumer they are pure cost, since the file is
// re-parsed on every keystroke. The cap is recorded rather than silent:
// diagsTruncated is what lets a caller distinguish a complete list from a
// truncated one.
//
// Speculation rewinds p.diags by slicing (see guards.go), so the flag is
// recomputed from the length rather than latched, or a rolled-back speculative
// overflow would leave the file permanently marked as truncated.
func (p *parser) record(diagnostic helpers.ErrorInterface) {
	if !helpers.IsRegisteredDiagnosticName(diagnostic.DiagnosticName()) {
		panic(fmt.Sprintf("unregistered diagnostic name %q", diagnostic.DiagnosticName()))
	}
	if len(p.diags) >= foerrors.MaxParseErrors {
		p.diagsTruncated = true
		return
	}
	p.diags = append(p.diags, diagnostic)
	p.diagsTruncated = false
}

// locate substitutes a positioned token for one that has no position.
//
// The scanner's EOF token carries a placeholder position, so a diagnostic anchored to it would
// print as line 0 of no file. Since the usual reason to report at EOF is something missing from
// the end of the file — an unterminated statement, an unclosed body — the last real token is
// where the reader needs to look.
func (p *parser) locate(tok scanlex.Token) scanlex.Token {
	if tok.Kind != scanlex.EOF && tok.StartPos != nil && tok.StartPos.Ln > 0 {
		return tok
	}
	for i := len(p.toks) - 1; i >= 0; i-- {
		candidate := p.toks[i]
		if candidate.Kind != scanlex.EOF && candidate.StartPos != nil && candidate.StartPos.Ln > 0 {
			return candidate
		}
	}
	return tok
}

// reportf records a formatted diagnostic at tok without aborting.
func (p *parser) reportf(tok scanlex.Token, format string, args ...any) {
	p.report(tok, fmt.Sprintf(format, args...))
}

// reportUnsupported records that a construct is recognised but deliberately
// rejected, which is the required treatment for the reserved operators of
// DECISION-OP-005: the scanner produces them as single tokens and the parser
// refuses them, so a user-defined operator cannot claim a spelling before the
// language assigns it meaning.
func (p *parser) reportUnsupported(tok scanlex.Token, msg string) {
	start, end := tokenSpan(tok)
	p.record(helpers.NewUnSupportedException(start, end, msg))
}

// spanFrom returns the source region covered by the tokens in [start, p.pos),
// which is what the enclosing parse function consumed.
//
// The convention is half-open in the token stream and inclusive in the source:
// the span runs from the first character of the token at start to the last
// character of the token before the cursor. A parse function therefore records
// p.pos on entry and calls this when it builds its node.
//
// Two degenerate cases matter. A function that consumed nothing yields the
// zero-width span at its start token, which is still a usable cursor position.
// A start beyond the stream — reachable only through a bailout that unwound past
// the end — yields the zero Span, which Span.IsZero reports.
func (p *parser) spanFrom(start int) ast.Span {
	if start < 0 || start >= len(p.toks) {
		return ast.Span{}
	}
	startTok := p.toks[start]

	end := p.pos - 1
	if end < start {
		end = start
	}
	if end >= len(p.toks) {
		end = len(p.toks) - 1
	}
	endTok := p.toks[end]

	// A synthetic token carries no position; fall back to the start so the span
	// stays anchored in the file rather than collapsing to line 0.
	from, _ := tokenSpan(p.locate(startTok))
	_, to := tokenSpan(p.locate(endTok))
	if to.Ln == 0 {
		to = from
	}
	return ast.NewSpan(from, to)
}

// spanOf returns the source region of a single token, for a node built from one
// lexeme.
func (p *parser) spanOf(tok scanlex.Token) ast.Span {
	start, end := tokenSpan(p.locate(tok))
	return ast.NewSpan(start, end)
}

// spanOfFile returns the span of the whole token stream.
//
// A compilation-unit root covers the entire file by definition, including the
// preamble its body parser never saw. Measuring from the cursor instead would
// give a file that is nothing but directives a zero-width span at EOF, and an
// editor asking "which node is the cursor in?" would find no root at all.
func (p *parser) spanFrom0() ast.Span {
	saved := p.pos
	p.pos = len(p.toks)
	span := p.spanFrom(0)
	p.pos = saved
	return span
}

// spanOfNode returns an existing node's span, or fallback when it has none.
//
// A node that WRAPS or is DERIVED FROM another — a derived type over its
// element, a synthesized parameter standing for its type, a lowered chain over
// the expression it replaces — must report the source the user wrote, not the
// cursor position at the moment it was built. Lowering in particular runs after
// parsing is complete, where the cursor is at end of file and would give every
// rewritten node the same meaningless span.
func spanOfNode(node any, fallback ast.Span) ast.Span {
	if spanned, ok := node.(ast.Spanned); ok {
		if span := spanned.GetSpan(); !span.IsZero() {
			return span
		}
	}
	return fallback
}

// tokenSpan returns the source span of tok, tolerating the synthetic tokens that
// have no position (EOF fallbacks and fused operators built by tokenstream.go).
func tokenSpan(tok scanlex.Token) (helpers.Position, helpers.Position) {
	var start, end helpers.Position
	if tok.StartPos != nil {
		start = *tok.StartPos
	}
	if tok.EndPos != nil {
		end = *tok.EndPos
	} else {
		end = start
	}
	return start, end
}

// describeToken renders a token for a diagnostic. Quoting the lexeme is far more
// useful to a reader than naming its internal kind, so the kind is only added
// when the lexeme alone would be ambiguous or empty.
func describeToken(tok scanlex.Token) string {
	switch tok.Kind {
	case scanlex.EOF:
		return "end of file"
	case scanlex.STRING, scanlex.CHAR, scanlex.NUMBER:
		return fmt.Sprintf("the literal %s", tok.Value)
	case scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER:
		return fmt.Sprintf("the name %q", logicalName(tok.Value))
	}
	if tok.Value == "" {
		return scanlex.TokenKindString(tok.Kind)
	}
	return fmt.Sprintf("%q", tok.Value)
}

// describeKind renders a token kind in the words a FoLang programmer would use,
// so that expect() produces messages about source syntax rather than about
// scanner internals.
func describeKind(k scanlex.TokenKind) string {
	switch k {
	case scanlex.OPEN_PAREN:
		return `"("`
	case scanlex.CLOSE_PAREN:
		return `")"`
	case scanlex.OPEN_CURLY:
		return `"{"`
	case scanlex.CLOSE_CURLY:
		return `"}"`
	case scanlex.OPEN_BRACKET:
		return `"["`
	case scanlex.CLOSE_BRACKET:
		return `"]"`
	case scanlex.SEMI_COLON:
		return `";"`
	case scanlex.COLON:
		return `":"`
	case scanlex.COMMA:
		return `","`
	case scanlex.DOT:
		return `"."`
	case scanlex.ASSIGNMENT:
		return `"="`
	case scanlex.ARROW:
		return `"->"`
	case scanlex.EQGT:
		return `"=>"`
	case scanlex.IDENTIFIER:
		return "an identifier"
	case scanlex.BUILT_IN_TYPE:
		return "a built-in type"
	case scanlex.BUILT_IN_KIND:
		return "a built-in kind"
	case scanlex.EOF:
		return "end of file"
	default:
		return scanlex.TokenKindString(k)
	}
}
