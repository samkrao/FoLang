package scanlex

// Contains reports whether the target TokenKind is present in the given slice.
func Contains(nums []TokenKind, target TokenKind) bool {
	for _, num := range nums {
		if num == target {
			return true
		}
	}
	return false
}

// currentLineText returns the current source line, caching it until the scanner
// advances to another line.
func (lex *lexer) currentLineText() string {
	if lex == nil || lex.pos < 0 || lex.pos > len(lex.source) {
		return ""
	}
	if lex.lineTextOK && lex.lineTextAt == lex.line {
		return lex.lineText
	}
	start, end := lex.pos, lex.pos
	for start > 0 && lex.source[start-1] != '\r' && lex.source[start-1] != '\n' {
		start--
	}
	for end < len(lex.source) && lex.source[end] != '\r' && lex.source[end] != '\n' {
		end++
	}
	lex.lineText = lex.source[start:end]
	lex.lineTextAt = lex.line
	lex.lineTextOK = true
	return lex.lineText
}

func (lex *lexer) invalidateCurrentLineText() {
	lex.lineTextOK = false
}
