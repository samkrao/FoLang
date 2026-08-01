package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Collecting the operator symbols a file declares, before it is parsed.
//
// DECISION-EXT-001 lets a declaration introduce a NEW operator symbol:
//
//	@co.dap.operator(symbol='∪', mode=define, fixity=infix, precedence=60, …)
//	union(left Vector, right Vector)->(Vector) = { … }
//
// The scanner has to know that spelling to produce one token for it, but the spelling is
// introduced by a declaration the scanner has not reached. The way out is that the
// declaration writes the symbol inside a LITERAL — `'∪'` or `"<+>"` — and a literal is
// something an ordinary scan already reads correctly. So the file is scanned once with
// no custom operators, the declared spellings are read out of that stream, and the file
// is scanned again with them in scope.
//
// This pass deliberately does not parse. It matches the shape
//
//	ATDAP("@co.dap.operator") "(" … "symbol" "=" <literal> … ")"
//
// and reads nothing else, so a malformed declaration costs a missing operator and a
// diagnostic from the real parse, never a failure here.

// declaresOperators reports whether source could contain an operator declaration.
//
// The collecting scan exists only for files that declare one, and almost none do — two
// of the reference corpus's 140. A substring test costs a fraction of a scan and skips
// the whole extra pass for every other file, so the common case is scanned once as it
// was before custom operators existed.
//
// A false positive — the name inside a comment or a string — costs one wasted scan and
// nothing else, because the collector then finds no declaration. A false negative is
// impossible: the declaration cannot be written without one of these names.
func declaresOperators(source string) bool {
	return strings.Contains(source, "@co.dap.operator") ||
		strings.Contains(source, "co.lang.operator")
}

// declaredOperatorsIn returns the operator symbols source declares.
//
// The collecting scan is SILENT: it runs with the operators still unknown, so a declared
// "∪" still reads to it as a reserved glyph. Reporting there would fail the file for an
// error the real scan is about to resolve, so every diagnostic comes from the scan that
// has the operators in scope.
func declaredOperatorsIn(source, basename string) *scanlex.CustomOperators {
	if !declaresOperators(source) {
		return scanlex.NewCustomOperators(nil)
	}
	return collectCustomOperators(scanlex.TokenizeQuiet(source, basename))
}

// collectCustomOperators reads the operator symbols declared in a token stream.
func collectCustomOperators(toks []scanlex.Token) *scanlex.CustomOperators {
	var symbols []string

	for i := 0; i < len(toks); i++ {
		if !isOperatorDeclarationIntroducer(toks[i]) {
			continue
		}
		if symbol, ok := operatorSymbolOf(toks, i); ok {
			symbols = append(symbols, symbol)
		}
	}

	return scanlex.NewCustomOperators(symbols)
}

// isOperatorDeclarationIntroducer reports whether a token opens an operator declaration.
//
// Both spellings the reference uses are accepted: the @co.dap.operator annotation and
// the co.lang.operator declaration kind.
func isOperatorDeclarationIntroducer(tok scanlex.Token) bool {
	switch tok.Value {
	case "@co.dap.operator", "co.lang.operator":
		return true
	}
	return false
}

// operatorSymbolOf reads the `symbol=` argument of the declaration beginning at start.
//
// The search is bounded by the declaration's own parenthesised argument list, so it can
// never run past the declaration and pick up an unrelated `symbol` elsewhere in the file.
func operatorSymbolOf(toks []scanlex.Token, start int) (string, bool) {
	i := start + 1
	if i >= len(toks) || toks[i].Kind != scanlex.OPEN_PAREN {
		return "", false
	}

	depth := 0
	for ; i < len(toks); i++ {
		switch toks[i].Kind {
		case scanlex.OPEN_PAREN, scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY:
			depth++
			continue
		case scanlex.CLOSE_PAREN, scanlex.CLOSE_BRACKET, scanlex.CLOSE_CURLY:
			depth--
			if depth == 0 {
				return "", false // the argument list ended without a symbol
			}
			continue
		}

		// Only the declaration's own arguments count, not a nested list's.
		if depth != 1 || logicalName(toks[i].Value) != "symbol" {
			continue
		}
		if i+2 >= len(toks) || toks[i+1].Value != "=" {
			continue
		}
		if symbol, ok := literalText(toks[i+2]); ok {
			return symbol, true
		}
	}
	return "", false
}

// literalText unquotes a character or string literal, which is how an operator symbol is
// written. Anything else is not a spelling and is refused.
func literalText(tok scanlex.Token) (string, bool) {
	switch tok.Kind {
	case scanlex.CHAR, scanlex.STRING:
		if len(tok.Value) < 2 {
			return "", false
		}
		return tok.Value[1 : len(tok.Value)-1], true
	}
	return "", false
}

// Scope: an operator is NOT imported.
//
// An operator function is declared in the companion unit of the struct it belongs to,
// or inside a class — never at package scope and never in an unrelated unit. That is
// what validateOperatorOwnership enforces, and it is why this pass reads one compilation
// unit and stops there: a symbol declared elsewhere is not in scope here, so there is
// nothing for a cross-file collector to contribute.
//
// If importing operators is ever added, the seam is the CustomOperators argument to
// scanlex.TokenizeWith — a driver that knows the import graph would union the imported
// symbols into the set built here. Nothing else in the scanner or the parser would
// change.
