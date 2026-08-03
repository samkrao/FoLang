package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// The custom-operator bootstrap source has an intentional second grammar root.
// It is parsed before ordinary FoLang files so its symbolic declaration names
// can seed both the lexer and Pratt table. Keeping this reader separate prevents
// co.lang.operator, imports, functions, and ordinary library members from
// leaking into the source-only format.
type operatorSourceParser struct {
	toks   []scanlex.Token
	pos    int
	source string
	file   string
	diags  []error
}

// parseOperatorSource parses exactly one operator-source-file. Recovery may
// collect multiple diagnostics internally, but the returned catalog is atomic:
// any grammar or alpha-semantic finding discards every declaration so no caller
// can accidentally install a partially valid operator table.
func parseOperatorSource(source, basename string) ([]operatorDeclaration, []error) {
	normalized := normalizeLineEndings(source)
	r := &operatorSourceParser{
		toks:   normalizeTokens(scanlex.Tokenize(normalized, basename)),
		source: normalized,
		file:   basename,
	}
	declarations := r.parseFile()
	if len(r.diags) > 0 {
		return nil, r.diags
	}
	return declarations, r.diags
}

func (r *operatorSourceParser) parseFile() []operatorDeclaration {
	// operator-library-marker
	if !r.expectValue("@co.dap.library", "the fixed operator-library marker") ||
		!r.expectValue("(", "after @co.dap.library") ||
		!r.expectLogicalValue("type", "as the operator-library marker field") ||
		!r.expectValue("=", "after marker field type") ||
		!r.expectLogicalValue("operator", "as the operator-library marker type") {
		return nil
	}
	if !r.expectValue(")", "to close the operator-library marker") ||
		!r.expectValue("_", "as the fixed operator-library declaration name") ||
		!r.expectValue("co.lang.library", "as the fixed operator-library declaration kind") ||
		!r.expectValue("=", "before the operator-library body") ||
		!r.expectValue("{", "to open the operator-library body") {
		return nil
	}

	var declarations []operatorDeclaration
	seenSymbols := map[string]scanlex.Token{}
	for !r.at("}") && !r.eof() {
		declaration, symbolTok, ok := r.parseDeclaration()
		if !ok {
			return declarations
		}
		symbol := operatorOptionText(declaration.Options, "symbol")
		if previous, duplicate := seenSymbols[symbol]; duplicate {
			r.invalid(symbolTok, "custom operator %q is declared more than once; every symbol has exactly one operator-source declaration (first declaration at line %d)", symbol, tokenLine(previous))
			continue
		}
		seenSymbols[symbol] = symbolTok
		declarations = append(declarations, declaration)
	}

	if !r.expectValue("}", "to close the operator-library body") {
		return declarations
	}
	if r.at(";") {
		r.invalid(r.advance(), "the operator-library body ends at its closing brace and takes no semicolon")
	}
	if !r.eof() {
		r.invalid(r.cur(), "operator source contains %s after its one operator-library declaration", describeToken(r.cur()))
	}
	return declarations
}

func (r *operatorSourceParser) parseDeclaration() (operatorDeclaration, scanlex.Token, bool) {
	symbolTok := r.cur()
	if !scanlex.IsOperatorSpelling(symbolTok.Value) {
		r.invalid(symbolTok, "an operator-library body may contain only symbolic co.lang.operator declarations; found %s", describeToken(symbolTok))
		return operatorDeclaration{}, symbolTok, false
	}
	r.advance()
	symbol := symbolTok.Value
	if strings.Contains(symbol, "//") || strings.Contains(symbol, "/*") {
		r.invalid(symbolTok, "operator symbol %q cannot contain a comment opener", symbol)
	}
	if scanlex.IsHardReservedOperatorSpelling(symbol) {
		r.invalid(symbolTok, "operator symbol %q is hard-reserved and cannot be declared", symbol)
	} else if scanlex.IsLanguageOwnedOperatorSpelling(symbol) {
		r.invalid(symbolTok, "operator symbol %q is language-owned and cannot be redeclared in a project operator source", symbol)
	}

	if !r.expectValue("co.lang.operator", "after the operator symbol") ||
		!r.expectValue("=", "before the operator metadata body") ||
		!r.expectValue("{", "to open the operator metadata body") {
		return operatorDeclaration{}, symbolTok, false
	}

	options := map[string]any{"symbol": symbol}
	seen := map[string]scanlex.Token{}
	propertyCount := 0
	for !r.at("}") && !r.eof() {
		keyTok := r.cur()
		key := logicalName(keyTok.Value)
		r.advance()
		if !operatorSourcePropertyKeys[key] {
			r.invalid(keyTok, "unknown operator property %q", key)
		}
		if previous, duplicate := seen[key]; duplicate {
			r.invalid(keyTok, "operator property %q occurs more than once (first occurrence at line %d)", key, tokenLine(previous))
		} else {
			seen[key] = keyTok
		}
		propertyCount++

		if !r.expectBinder("after operator property " + key) {
			return operatorDeclaration{}, symbolTok, false
		}
		value, ok := r.parsePropertyValue(key)
		if !ok {
			return operatorDeclaration{}, symbolTok, false
		}
		if operatorSourcePropertyKeys[key] {
			options[key] = value
		}

		if r.at(",") {
			r.advance()
			continue // includes the allowed trailing comma
		}
		if !r.at("}") {
			r.invalid(r.cur(), "operator properties must be separated by a comma; found %s", describeToken(r.cur()))
			return operatorDeclaration{}, symbolTok, false
		}
	}

	if propertyCount == 0 {
		r.invalid(r.cur(), "operator %q requires a non-empty metadata body", symbol)
	}
	if !r.expectValue("}", "to close the operator metadata body") {
		return operatorDeclaration{}, symbolTok, false
	}
	if r.at(";") {
		r.invalid(r.advance(), "an operator declaration body ends at its closing brace and takes no semicolon")
	}

	r.validateDeclaration(symbolTok, options, seen)
	return operatorDeclaration{Options: options}, symbolTok, true
}

var operatorSourcePropertyKeys = map[string]bool{
	"fixity":           true,
	"precedence":       true,
	"associativity":    true,
	"arity":            true,
	"commutative":      true,
	"idempotent":       true,
	"identity":         true,
	"foldable":         true,
	"vectorizable":     true,
	"distributes_over": true,
	"desugar":          true,
}

func (r *operatorSourceParser) parsePropertyValue(key string) (any, bool) {
	switch key {
	case "fixity", "associativity":
		if r.eof() || r.at(",") || r.at("}") {
			r.invalid(r.cur(), "operator property %q requires a name value", key)
			return nil, false
		}
		return logicalName(r.advance().Value), true

	case "precedence":
		tok := r.cur()
		if tok.Kind != scanlex.NUMBER || !isPrecedenceDecimal(tok.Value) {
			r.invalid(tok, "operator precedence must be a decimal integer from 0 through 100")
			return nil, false
		}
		r.advance()
		value, err := strconv.ParseInt(tok.Value, 10, 64)
		if err != nil {
			r.invalid(tok, "operator precedence must be a decimal integer from 0 through 100")
			return nil, false
		}
		return value, true

	case "arity":
		tok := r.cur()
		if tok.Kind == scanlex.NUMBER {
			if !isPositiveDecimal(tok.Value) {
				r.invalid(tok, "numeric operator arity must be a positive decimal integer")
				return nil, false
			}
			r.advance()
			value, err := strconv.ParseInt(tok.Value, 10, 64)
			if err != nil {
				r.invalid(tok, "numeric operator arity must be a positive decimal integer")
				return nil, false
			}
			return value, true
		}
		if r.eof() || r.at(",") || r.at("}") {
			r.invalid(tok, "operator arity requires unary, binary, ternary, or a positive decimal integer")
			return nil, false
		}
		return logicalName(r.advance().Value), true

	case "commutative", "idempotent", "foldable", "vectorizable":
		tok := r.cur()
		if tok.Value != "co.const.true" && tok.Value != "co.const.false" {
			r.invalid(tok, "operator property %q accepts only co.const.true or co.const.false", key)
			return nil, false
		}
		r.advance()
		return tok.Value == "co.const.true", true

	case "identity":
		tok := r.cur()
		if tok.Kind == scanlex.STRING {
			var value strings.Builder
			for r.cur().Kind == scanlex.STRING {
				part, _ := literalText(r.advance())
				value.WriteString(part)
			}
			return value.String(), true
		}
		if !isOperatorIdentityLiteral(tok) {
			r.invalid(tok, "operator identity must be one FoLang literal, including co.const.none")
			return nil, false
		}
		r.advance()
		return tok.Value, true

	case "distributes_over":
		return r.parseOperatorSymbolList()

	case "desugar":
		tok := r.cur()
		value, ok := literalText(tok)
		if !ok || tok.Kind != scanlex.STRING {
			r.invalid(tok, "operator desugar metadata must be a string literal")
			return nil, false
		}
		r.advance()
		return value, true

	default:
		// Retain parser position after an unknown scalar so the next declaration
		// can still be diagnosed. Nested values are intentionally not part of the
		// source-only grammar.
		if r.eof() || r.at(",") || r.at("}") {
			r.invalid(r.cur(), "unknown operator property %q requires a value before it can be skipped", key)
			return nil, false
		}
		return r.advance().Value, true
	}
}

func (r *operatorSourceParser) parseOperatorSymbolList() (any, bool) {
	if !r.expectValue("[", "to open distributes_over") {
		return nil, false
	}
	var symbols []string
	for !r.at("]") && !r.eof() {
		tok := r.cur()
		symbol, ok := literalText(tok)
		if !ok || !scanlex.IsOperatorSpelling(symbol) {
			r.invalid(tok, "distributes_over entries must be quoted operator symbols")
			return nil, false
		}
		r.advance()
		symbols = append(symbols, symbol)
		if !r.at(",") {
			break
		}
		r.advance()
	}
	if !r.expectValue("]", "to close distributes_over") {
		return nil, false
	}
	return symbols, true
}

func (r *operatorSourceParser) validateDeclaration(symbolTok scanlex.Token, options map[string]any, seen map[string]scanlex.Token) {
	symbol := operatorOptionText(options, "symbol")
	for _, required := range []string{"fixity", "precedence", "associativity", "arity"} {
		if _, present := seen[required]; !present {
			r.invalid(symbolTok, "operator %q requires property %q exactly once", symbol, required)
		}
	}

	fixity := operatorOptionText(options, "fixity")
	switch fixity {
	case "infix", "prefix", "postfix":
	case "circumfix", "postcircumfix", "precircumfix", "mixfix", "ternary", "distfix":
		r.invalid(symbolTok, "operator %q uses future fixity %q, which the alpha parser does not implement", symbol, fixity)
	default:
		if fixity != "" {
			r.invalid(symbolTok, "operator %q has unknown fixity %q", symbol, fixity)
		}
	}

	if precedence, ok := operatorOptionInteger(options, "precedence"); ok && (precedence < 0 || precedence > 100) {
		r.invalid(symbolTok, "operator %q has precedence %d; precedence must be from 0 through 100", symbol, precedence)
	}

	associativity := operatorOptionText(options, "associativity")
	if associativity != "" && associativity != "left" && associativity != "right" && associativity != "none" {
		r.invalid(symbolTok, "operator %q has unknown associativity %q", symbol, associativity)
	}

	if value, present := options["arity"]; present {
		arity, ok := operatorArityCount(value)
		if !ok {
			r.invalid(symbolTok, "operator %q has invalid arity %q", symbol, describeOperatorArity(value))
			return
		}
		switch fixity {
		case "prefix", "postfix":
			if arity != 1 {
				r.invalid(symbolTok, "alpha %s operator %q requires unary arity (1), found %d", fixity, symbol, arity)
			}
		case "infix":
			if arity != 2 {
				r.invalid(symbolTok, "alpha infix operator %q requires binary arity (2), found %d", symbol, arity)
			}
		}
	}
}

func isDecimalDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isPrecedenceDecimal(value string) bool {
	return value == "0" || isPositiveDecimal(value)
}

func isPositiveDecimal(value string) bool {
	return isDecimalDigits(value) && value[0] >= '1' && value[0] <= '9'
}

func isOperatorIdentityLiteral(tok scanlex.Token) bool {
	switch tok.Kind {
	case scanlex.NUMBER, scanlex.STRING, scanlex.CHAR, scanlex.BOOL, scanlex.BUILT_IN_CONSTANTS:
		return true
	}
	return tok.Value == "co.const.true" || tok.Value == "co.const.false" || tok.Value == "co.const.none"
}

func (r *operatorSourceParser) cur() scanlex.Token {
	if r.pos >= len(r.toks) {
		return scanlex.Token{Kind: scanlex.EOF, Value: "EOF"}
	}
	return r.toks[r.pos]
}

func (r *operatorSourceParser) advance() scanlex.Token {
	tok := r.cur()
	if r.pos < len(r.toks) {
		r.pos++
	}
	return tok
}

func (r *operatorSourceParser) at(value string) bool { return r.cur().Value == value }
func (r *operatorSourceParser) eof() bool            { return r.cur().Kind == scanlex.EOF }

func (r *operatorSourceParser) expectValue(value, context string) bool {
	if r.at(value) {
		r.advance()
		return true
	}
	r.invalid(r.cur(), "expected %q %s, found %s", value, context, describeToken(r.cur()))
	return false
}

func (r *operatorSourceParser) expectLogicalValue(value, context string) bool {
	if logicalName(r.cur().Value) == value {
		r.advance()
		return true
	}
	r.invalid(r.cur(), "expected %q %s, found %s", value, context, describeToken(r.cur()))
	return false
}

func (r *operatorSourceParser) expectBinder(context string) bool {
	if r.at("=") || r.at(":") {
		r.advance()
		return true
	}
	r.invalid(r.cur(), "expected '=' or ':' %s, found %s", context, describeToken(r.cur()))
	return false
}

func (r *operatorSourceParser) invalid(tok scanlex.Token, format string, args ...any) {
	start, end := tokenSpan(tok)
	if start.Ln == 0 {
		start = helpers.Position{Ln: 1, Col: 1, Fn: r.file, Ftxt: r.source}
		end = start
	}
	r.diags = append(r.diags, helpers.NewInvalidSyntaxError(start, end, fmt.Sprintf(format, args...)))
}

func tokenLine(tok scanlex.Token) int {
	if tok.StartPos != nil {
		return tok.StartPos.Ln
	}
	return 0
}
