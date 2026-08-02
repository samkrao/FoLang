package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// postfix-expression — the highest-binding level of section 11.
//
//	postfix-expression = primary-expression, { postfix-suffix | postfix-operator }
//	postfix-suffix     = call-suffix | index-suffix | member-suffix | match-suffix
//	postfix-operator   = "!" | "++" | "--"
//
// This loop is where FoLang's control flow lives. The language has no if, else,
// for, while or foreach keyword: every branch, loop, iterator and ternary is an
// associated-function chain, so it already parses as a postfix chain of member and
// call suffixes where an argument may be a block (section 11a).
//
//	(pp > 10).do({ … }).otherwise(pp == 11).do({ … }).otherwise.do({ … });
//	arr.each(idx, val).do({ … });
//	arr.contains(k).do({ … }).otherwise.do({ … });
//	s = (truth).return(a).otherwise.return(b);
//
// The grammar is explicit that these shapes must NOT be a second parse path:
// recognising them here would need unbounded lookahead to tell a control chain
// from any other method chain, so the canonical-shape check belongs to semantic
// analysis. This file therefore parses them as what they are — ordinary chains.

// parsePostfix parses the { postfix-suffix | postfix-operator } tail of a
// postfix-expression, given its already-parsed primary.
func (p *parser) parsePostfix(left ast.Expr) ast.Expr {
	defer p.enter()()

	for {
		switch {
		case p.at(scanlex.DOT):
			left = p.parseMemberOrMatchSuffix(left)

		case p.at(scanlex.OPEN_PAREN):
			left = p.parseCallSuffix(left)

		case p.at(scanlex.OPEN_BRACKET):
			left = p.parseIndexSuffix(left)

		case p.isBuiltinPostfixOperator() && p.postfixOperatorApplies():
			opTok := p.advance()
			left = ast.PrefixExpr{
				Operator: opTok,
				Right:    left,
				Symb:     p.exprSymbol("postfix" + opTok.Value),
			}

		default:
			return left
		}
	}
}

// postfixOperatorApplies guards the postfix reading of a spelling that is also
// prefix or infix.
//
// "++" and "--" are unambiguous in this position, since an operand has just been
// completed. "!" is the interesting one: it is postfix here, but a "!" followed by
// "=" would already have been fused into "!=" by the scanner, so no further check
// is needed for it either. What must be excluded is a "-" or "+", which are infix
// in this position and belong to the Pratt loop rather than to this one.
func (p *parser) postfixOperatorApplies() bool {
	// The three built-in postfix spellings are each ALSO a prefix or infix operator,
	// so seeing one after an operand is not on its own enough to make it postfix; this
	// guard is what settles that overlap.
	switch p.lexeme() {
	case "!", "++", "--":
		return true
	}

	return false
}

// parseMemberOrMatchSuffix parses the member-suffix and match-suffix productions:
//
//	member-suffix = ".", ( identifier | lifecycle-name )
//	match-suffix  = ".match", [ "(", [ expression ], ")" ],
//	                { match-case }, [ match-default ]
//
// ".match" is checked first because it is a member name that begins a construct of
// its own rather than a plain access.
func (p *parser) parseMemberOrMatchSuffix(left ast.Expr) ast.Expr {
	// Look past the "." to see which suffix this is.
	member := p.peek(1)

	if p.isMemberNameToken(member) {
		switch logicalName(member.Value) {
		case "match":
			return p.parseMatchSuffix(left)
		case "where":
			// The postfix let form: (x + 1).where(x = 10). Recorded as a let
			// expression so both let spellings reach the semantic phase alike.
			p.advance() // "."
			p.advance() // "where"
			return p.parseWhereSuffix(left)
		}
	}

	p.advance() // "."

	if p.atLifecycleName() {
		lifecycle := p.parseLifecycleName()
		return ast.MemberExpr{
			Member:   left,
			Property: lifecycle.Scanned,
			Type_:    scanlex.SPECIAL_METHODS,
			Symb:     p.exprSymbol(lifecycle.Scanned),
		}
	}

	if !p.isMemberNameToken(p.cur()) {
		p.failf(p.cur(), "expected a member name after %q, found %s", ".", describeToken(p.cur()))
	}
	nameTok := p.advance()

	return ast.MemberExpr{
		Member:   left,
		Property: nameTok.Value,
		Type_:    nameTok.Kind,
		Symb:     p.exprSymbol(nameTok.Value),
	}
}

// parseCallSuffix parses the call-suffix production:
//
//	call-suffix   = "(", [ argument-list ], ")"
//	argument-list = argument, { ",", argument }, [ "," ]
func (p *parser) parseCallSuffix(left ast.Expr) ast.Expr {
	p.expect(scanlex.OPEN_PAREN, "to open an argument list")
	target := callTargetName(left)
	popLambdaContext := p.pushLambdaCallContext(isLambdaCollectionOperation(target))
	defer popLambdaContext()

	var args []ast.Expr
	if !p.at(scanlex.CLOSE_PAREN) {
		args = p.parseArgumentList(target)
	}

	p.expect(scanlex.CLOSE_PAREN, "to close an argument list")

	return ast.CallExpr{
		Method:      left,
		Arguments:   args,
		CallKind:    p.classifyCall(left),
		SymbolType_: "call",
		Symb:        p.exprSymbol(target),
	}
}

// classifyCall records what can be known from syntax without attempting name
// resolution. Every invocation remains an ast.CallExpr; this classification is
// metadata only and may be replaced by the resolver once the receiver type and
// visible class, companion, extension, or instance methods are known.
func (p *parser) classifyCall(callee ast.Expr) ast.CallKind {
	switch target := callee.(type) {
	case ast.MemberExpr:
		// BUILT_IN_METHOD is assigned by the tokenizer from Reserved_me. Keep
		// the spelling check as a defensive fallback for ASTs produced from an
		// unfused or externally supplied token stream.
		if target.Type_ == scanlex.BUILT_IN_METHOD || scanlex.IsReservedMethod(logicalName(target.Property)) {
			return ast.CallBuiltInMethod
		}
		return ast.CallMethod

	case ast.SymbolExpr:
		return ast.CallFunction

	default:
		return ast.CallUnresolved
	}
}

// parseArgumentList parses the argument-list production, allowing the trailing
// comma of DECISION-COL-001.
func (p *parser) parseArgumentList(target string) []ast.Expr {
	var args []ast.Expr
	for {
		args = append(args, p.parseArgument(target, len(args)))
		if !p.accept(scanlex.COMMA) {
			return args
		}
		if p.at(scanlex.CLOSE_PAREN) {
			return args // trailing comma
		}
	}
}

// parseArgument parses the argument production:
//
//	argument = ( [ identifier, "=" ], expression )
//	         | block
//	         | lambda-expression
//
// The block alternative is what carries every control-flow body: the `{ … }` in
// `.do({ … })` is an argument, not a nested statement. The named form
// `identifier "="` is the named-argument syntax of docs/language-ref.md, "Named
// Parameters".
func (p *parser) parseArgument(target string, index int) ast.Expr {
	// The contextual underscore is not a general primary expression. Its only
	// call-argument position is the first (key/index) binding of each. The value
	// binding of each and the searched value of contains/containsVal must be real
	// expressions; discarding either would make the operation meaningless.
	// Handling it before parseExpression also makes the permission direct: `_ + 1`
	// and a wildcard nested in another call are still rejected.
	if p.at(scanlex.DISCARD_WILD_VAR) && wildcardCallArgumentAllowed(target, index) {
		return p.parseWildcard()
	}

	// A block argument. A braced group here is a block unless it is a map
	// literal, and the guard gives the block reading priority.
	if p.at(scanlex.OPEN_CURLY) && !p.looksLikeMapLiteral() {
		block := p.parseBlock("a block argument")
		return ast.StatementExpr{Statement: block, Symb: p.exprSymbol("block-argument")}
	}

	// A lambda argument: |x| => x * x
	if p.atOp("|") {
		return p.parseDirectLambdaArgument()
	}

	// A named argument: name = value. The "=" must not be mistaken for an
	// assignment expression, so the name is only taken as a label when the
	// following token is "=" and the one after it starts a value.
	if p.atIdentifier() && p.peek(1).Value == "=" {
		label := p.parseIdentifier("as an argument name")
		opTok := p.advance() // "="
		var value ast.Expr
		if p.atOp("|") {
			// A named callback is still a direct call argument. Route it through
			// the same placement check as the positional lambda alternative.
			value = p.parseDirectLambdaArgument()
		} else {
			value = p.parseExpression()
		}
		return ast.AssignmentExpr{
			Assigne: ast.SymbolExpr{
				Value:       label.Scanned,
				SymbolType_: "argument-name",
				Symb:        p.exprSymbol(label.Scanned),
			},
			Operator:      opTok,
			AssignedValue: value,
			Symb:          p.exprSymbol(label.Scanned),
		}
	}

	return p.parseExpression()
}

// calledMethodName reduces a call target to the method being invoked.
//
// A target reaches these predicates in one of two shapes, decided by the scanner rather
// than by the source. A member name the scanner knows — "map" and "filter" are both in
// Reserved_me — is split off as its own token, so the callee is a MemberExpr and the
// target is already just "map". Any other member name stays an ordinary identifier, and
// parseQualifiedName then absorbs it, so `nums.reduce(…)` arrives as the single name
// "nums.reduce".
//
// Comparing the whole target therefore recognised only the operations that happen to be
// in Reserved_me: map and filter worked while reduce, forEach, sortBy and groupBy did
// not. The method is the LAST segment in both shapes, so that is what is compared.
func calledMethodName(target string) string {
	logical := logicalName(target)
	if dot := strings.LastIndexByte(logical, '.'); dot >= 0 {
		return logical[dot+1:]
	}
	return logical
}

// wildcardCallArgumentAllowed is the closed set of call positions in which `_`
// is syntax rather than an ordinary identifier.
func wildcardCallArgumentAllowed(target string, index int) bool {
	return index == 0 && calledMethodName(target) == "each"
}

// isLambdaCollectionOperation is the closed call-site set from the reference's
// Lambda section: a lambda is admitted only as a direct argument of one of these.
func isLambdaCollectionOperation(name string) bool {
	switch calledMethodName(name) {
	case "map", "filter", "reduce", "forEach", "sortBy", "groupBy":
		return true
	default:
		return false
	}
}

// parseIndexSuffix parses the index-suffix production:
//
//	index-suffix = "[", [ expression-list ], "]"
//
// A list of indices is a multi-dimensional access, as in `grid[row, col]`. An empty
// "[]" is admitted by the grammar and left for the semantic phase to reject or
// interpret.
func (p *parser) parseIndexSuffix(left ast.Expr) ast.Expr {
	p.expect(scanlex.OPEN_BRACKET, "to open an index")

	var indices []ast.Expr
	if !p.at(scanlex.CLOSE_BRACKET) {
		indices = p.parseExpressionList()
	}

	p.expect(scanlex.CLOSE_BRACKET, "to close an index")

	// A single index is the common case and maps directly onto ComputedExpr.
	// Several indices are folded into a comma expression, which is the AST's
	// only multi-value expression carrier.
	var property ast.Expr
	switch len(indices) {
	case 0:
		property = nil
	case 1:
		property = indices[0]
	default:
		property = p.foldComma(indices)
	}

	return ast.ComputedExpr{
		Member:   left,
		Property: property,
		Symb:     p.exprSymbol("index"),
	}
}

// foldComma folds several expressions into a left-leaning ast.CommaExpr chain.
func (p *parser) foldComma(exprs []ast.Expr) ast.Expr {
	out := exprs[0]
	for _, e := range exprs[1:] {
		out = ast.CommaExpr{
			Left:  out,
			Right: e,
			Symb:  p.exprSymbol(","),
		}
	}
	return out
}

// callTargetName renders a call's callee for the symbol record, so a diagnostic or
// a dump names the function rather than saying "call".
func callTargetName(callee ast.Expr) string {
	switch c := callee.(type) {
	case ast.SymbolExpr:
		return c.Value
	case ast.MemberExpr:
		return c.Property
	default:
		return "call"
	}
}
