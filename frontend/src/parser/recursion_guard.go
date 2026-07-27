package parser

const maxRecursionDepth = 2048

func enterRec(p *parser) bool {
	p.recDepth++
	if p.recDepth > maxRecursionDepth {
		return false
	}
	return true
}

func leaveRec(p *parser) {
	if p.recDepth > 0 {
		p.recDepth--
	}
}
