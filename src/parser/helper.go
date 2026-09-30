package parser

import (
	"github.com/samkrao/fo-lang/src/scanlex"
)

func (p *Parser) expect(tok scanlex.TokenKind, context string) (bool, *scanlex.Token) {
	current := p.Peek(0)

	if current != nil && current.Kind == tok {
		return true, p.Next()
	}

	return false, scanlex.NewInvalidToken(current, context)

}
