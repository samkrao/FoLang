package parser

import (
	"unicode/utf8"

	"github.com/samkrao/fo-lang/src/scanlex"
)

// Operand-facing boundaries for multi-symbol expression operators.
//
// DECISION-LEX-010 keeps structural uses of spellings such as ->, := and ranges
// context-sensitive. The scanner records source boundaries without deciding
// those roles; the parser applies them only after a token has been selected as
// an expression/definition operator.
func (p *parser) requireInfixOperatorBoundaries(tok scanlex.Token) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if utf8.RuneCountInString(tok.Value) <= 1 {
		return
	}
	p.requireOperatorBoundaryBefore(tok, "multi-symbol infix operator")
	p.requireOperatorBoundaryAfter(tok, "multi-symbol infix operator")
}

func (p *parser) requirePrefixOperatorBoundary(tok scanlex.Token) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if utf8.RuneCountInString(tok.Value) > 1 {
		p.requireOperatorBoundaryAfter(tok, "multi-symbol prefix operator")
	}
}

func (p *parser) requirePostfixOperatorBoundary(tok scanlex.Token) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if utf8.RuneCountInString(tok.Value) > 1 {
		p.requireOperatorBoundaryBefore(tok, "multi-symbol postfix operator")
	}
}

func (p *parser) requireDefinitionOperatorBoundaries(tok scanlex.Token) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.requireOperatorBoundaryBefore(tok, "definition operator")
	p.requireOperatorBoundaryAfter(tok, "definition operator")
}

func (p *parser) requireOperatorBoundaryBefore(tok scanlex.Token, role string) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !tok.BoundaryBefore {
		p.reportf(tok, "%s %q requires whitespace, a comment, or a delimiter on its operand-facing left side", role, tok.Value)
	}
}

func (p *parser) requireOperatorBoundaryAfter(tok scanlex.Token, role string) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !tok.BoundaryAfter {
		p.reportf(tok, "%s %q requires whitespace, a comment, or a delimiter on its operand-facing right side", role, tok.Value)
	}
}
