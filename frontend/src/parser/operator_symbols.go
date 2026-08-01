package parser

import (
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// operatorDeclaration is the syntax-relevant part of an operator declaration.
// The decoded options seed both scanning and Pratt parsing before any function
// body is parsed.
type operatorDeclaration struct {
	Options map[string]any
}

// operatorCollection is the result of the declaration prepass.
type operatorCollection struct {
	Custom       *scanlex.CustomOperators
	Declarations []operatorDeclaration
}

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
func declaredOperatorsIn(source, basename string, inherited []operatorDeclaration) operatorCollection {
	declarations := append([]operatorDeclaration(nil), inherited...)
	declarations = append(declarations, operatorDeclarationsInSource(source, basename)...)

	symbols := make([]string, 0, len(declarations))
	for _, declaration := range declarations {
		if symbol := operatorOptionText(declaration.Options, "symbol"); symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return operatorCollection{
		Custom:       scanlex.NewCustomOperators(symbols),
		Declarations: declarations,
	}
}

// operatorDeclarationsInSource performs the quiet declaration scan used by both
// a standalone parse and the project catalog builder.
func operatorDeclarationsInSource(source, basename string) []operatorDeclaration {
	if !declaresOperators(source) {
		return nil
	}
	return collectOperatorDeclarations(scanlex.TokenizeQuiet(source, basename))
}

// collectCustomOperators reads the operator symbols declared in a token stream.
func collectCustomOperators(toks []scanlex.Token) *scanlex.CustomOperators {
	declarations := collectOperatorDeclarations(toks)
	symbols := make([]string, 0, len(declarations))
	for _, declaration := range declarations {
		if symbol := operatorOptionText(declaration.Options, "symbol"); symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return scanlex.NewCustomOperators(symbols)
}

// collectOperatorDeclarations decodes every operator option list in toks. It is
// deliberately non-diagnostic; malformed declarations are reported by the real
// parse, while this pass uses only complete scalar values.
func collectOperatorDeclarations(toks []scanlex.Token) []operatorDeclaration {
	declarations := make([]operatorDeclaration, 0)
	for i := 0; i < len(toks); i++ {
		if !isOperatorDeclarationIntroducer(toks[i]) {
			continue
		}
		if options, ok := operatorOptionsOf(toks, i); ok {
			declarations = append(declarations, operatorDeclaration{Options: options})
		}
	}
	return declarations
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
	options, ok := operatorOptionsOf(toks, start)
	if !ok {
		return "", false
	}
	symbol := operatorOptionText(options, "symbol")
	return symbol, symbol != ""
}

// operatorOptionsOf reads the scalar arguments of either declaration spelling:
//
//	@co.dap.operator(...)
//	name co.lang.operator->(...)
//
// Accounting for the second form's ARROW fixes the old false negative in the
// scanner prepass.
func operatorOptionsOf(toks []scanlex.Token, start int) (map[string]any, bool) {
	i := start + 1
	if i < len(toks) && toks[i].Kind == scanlex.ARROW {
		i++
	}
	if i >= len(toks) || toks[i].Kind != scanlex.OPEN_PAREN {
		return nil, false
	}

	options := map[string]any{}
	depth := 0
	for ; i < len(toks); i++ {
		switch toks[i].Kind {
		case scanlex.OPEN_PAREN, scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY:
			depth++
			continue
		case scanlex.CLOSE_PAREN, scanlex.CLOSE_BRACKET, scanlex.CLOSE_CURLY:
			depth--
			if depth == 0 {
				return options, true
			}
			continue
		}

		// Only the declaration's own arguments count, not a nested list's.
		if depth != 1 || i+2 >= len(toks) || toks[i+1].Value != "=" {
			continue
		}
		key := logicalName(toks[i].Value)
		if !operatorOptionKeys[key] {
			continue
		}
		if value, ok := operatorScalarValue(toks[i+2]); ok {
			options[key] = value
		}
	}
	return nil, false
}

var operatorOptionKeys = map[string]bool{
	"symbol": true, "mode": true, "fixity": true, "precedence": true,
	"associativity": true, "arity": true,
}

// operatorScalarValue decodes only metadata that can affect tokenization or
// precedence. Other annotation metadata remains the real parser's responsibility.
func operatorScalarValue(tok scanlex.Token) (any, bool) {
	if value, ok := literalText(tok); ok {
		return value, true
	}
	if tok.Kind == scanlex.NUMBER {
		value, err := strconv.ParseInt(tok.Value, 10, 64)
		return value, err == nil
	}
	if tok.Kind == scanlex.IDENTIFIER || tok.Kind == scanlex.COMPOSITE_IDENTIFER ||
		tok.Kind == scanlex.KEYWORD || tok.Kind == scanlex.CONTEXT_KEYWORD {
		return logicalName(tok.Value), true
	}
	return nil, false
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

// Project operator scope.
//
// The project driver unions declarations discovered across source files before
// tokenizing the requested file. This supplies lexical and precedence knowledge;
// semantic type/name resolution still decides whether an operator is visible and
// applicable at a particular use site.
