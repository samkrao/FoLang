package parser

import (
	"fmt"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
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
	if traceEnabled {
		p.traceBail()
	}
	panic(bailout{})
}

// failf records a formatted diagnostic at tok and aborts the current parse.
func (p *parser) failf(tok scanlex.Token, format string, args ...any) {
	p.report(tok, fmt.Sprintf(format, args...))
	if traceEnabled {
		p.traceBail()
	}
	panic(bailout{})
}

// report records a diagnostic at tok without aborting. Use it when the parser
// can describe a problem and still produce a usable node, so that one file can
// yield several diagnostics in a single run.
func (p *parser) report(tok scanlex.Token, msg string) {
	start, end := tokenSpan(p.locate(tok))
	p.diags = append(p.diags, helpers.NewInvalidSyntaxError(start, end, msg))
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
	p.diags = append(p.diags, helpers.NewUnSupportedException(start, end, msg))
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
