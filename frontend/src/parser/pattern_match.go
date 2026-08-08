package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// match-suffix, match-case and match-default — section 9.
//
//	match-suffix    = ".match", [ "(", [ expression ], ")" ],
//	                  { match-case }, [ match-default ]
//	match-case      = ".case", "(", match-case-body, ")"
//	match-case-body = pattern, [ ":", expression ], "=>",
//	                  ( expression | block )
//	match-default   = ".default", "(", ( expression | block ), ")"
//
// The optional argument to ".match" selects the matcher rather than the subject —
// the subject is the expression to the left of the chain
// (docs/language-ref.md, "Pattern Matching"):
//
//	x.match.case(n: n > 10 => { n = n+100; "GT" }).case(n: n < 10 => "LT").default("EQ");
//	x.match(co.pattern.Type).case(co.lang.int => …).case(co.lang.float => …);
//	x.match(co.pattern.Value).case(0 => …).case(1 => …);
//	x.match(co.pattern.Shape).case(Point{x, y} => …).default(…);
//	x.match(PositiveEvenMatcher).case(0 => "…").default(…);
//
// The optional `":" expression` inside a case body is the guard, and the pattern to
// its left binds the name the guard tests: in `case(n: n > 10 => …)` the pattern
// binds n and the guard uses it. A guard is evaluated only after its structural
// pattern has matched.
//
// The grammar accepts a chain with zero cases so that a malformed chain produces a
// semantic diagnostic rather than a parse error; at least one case is required
// semantically.

// parseMatchSuffix parses the match-suffix production. subject is the expression to
// the left of ".match".
//
// Implements: match-suffix
func (p *parser) parseMatchSuffix(subject ast.Expr) ast.Expr {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.DOT, "before \"match\"")
	p.expectMemberName("match", "to begin a match chain")

	// The grammar permits any expression here. Preserve the complete selector;
	// derive a compatibility name only when it is statically name-shaped.
	var matcher ast.Expr
	if p.at(scanlex.OPEN_PAREN) {
		p.advance()
		if !p.at(scanlex.CLOSE_PAREN) {
			matcher = p.parseExpression()
		}
		p.expect(scanlex.CLOSE_PAREN, "to close the matcher of a match chain")
	}

	return p.parseMatchChain(subject, matcher, matcherExpressionName(matcher))
}

// atFoldedMatchSubject reports whether the cursor holds a name into which the scanner has
// folded a trailing ".match".
//
// Token folding rewrites a dot chain that begins at an identifier, and it splits off only
// the LAST segment when that segment is a reserved member name. So match chains can reach
// the parser with the subject and `.match` in a single identifier token:
//
//	x.match(co.pattern.Type)  ->  "x"  "."  "match"  "("  …     the "." survives
//	x.match.case(0 => 1)      ->  "x.match"  "."  "case"  "("   the "." is folded away
//	x.match                   ->  "x.match"                         zero-case error
//
// The first form is handled by parseMatchSuffix through the ordinary postfix chain. This
// predicate recognises every folded `.match`, including the zero-case form that
// must reach finishMatch to receive its required-case diagnostic.
func (p *parser) atFoldedMatchSubject() bool {
	if !p.atIdentifier() {
		return false
	}
	return strings.HasSuffix(logicalName(p.lexeme()), ".match")
}

// parseFoldedMatchChain parses a match chain whose subject and ".match" were folded into one
// token, as described on atFoldedMatchSubject.
//
// The subject is recovered by stripping the ".match" suffix from the folded lexeme.
func (p *parser) parseFoldedMatchChain() ast.Expr {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()

	subjectName := stripMatchSuffix(tok.Value)
	subject := ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: subjectName,
		SymbolType_: "reference",
		Symb:        p.exprSymbol(subjectName),
	}

	return p.parseMatchChain(subject, nil, "")
}

// stripMatchSuffix removes a trailing ".match" from a folded name, tolerating the "_fo"
// backend suffix the scanner appends to each segment.
func stripMatchSuffix(scanned string) string {
	for _, suffix := range []string{".match" + foLoweringSuffix, ".match"} {
		if strings.HasSuffix(scanned, suffix) {
			return strings.TrimSuffix(scanned, suffix)
		}
	}
	return scanned
}

// parseMatchChain parses the `{ match-case } [ match-default ]` tail of a match chain, given
// its subject and the matcher it selects.
func (p *parser) parseMatchChain(subject ast.Expr, matcher ast.Expr, matcherName string) ast.Expr {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	match := ast.MatchExprStmt{Span: p.spanFrom(spanStart), Expr_: subject,
		MatcherExpr: matcher,
		Name:        matcherName,
		Symb:        p.matcherSymbol(matcherName),
	}

	var cases []ast.CaseStmt
	var defaultCase *ast.CaseStmt

	for p.at(scanlex.DOT) && p.isMemberNameToken(p.peek(1)) {
		switch logicalName(p.peek(1).Value) {
		case "case":
			cases = append(cases, p.parseMatchCase())
		case "default":
			d := p.parseMatchDefault()
			defaultCase = &d
			p.rejectMatchArmAfterDefault()
			// A default ends the match arms. An unrelated postfix suffix may still
			// operate on the resulting match value.
			return p.finishMatch(match, cases, defaultCase, matcher != nil, matcherName)
		case "otherwise":
			p.fail(p.peek(1), `a match uses .case(...) and .default(...); .otherwise belongs only to conditional and ternary return chains`)
		default:
			return p.finishMatch(match, cases, defaultCase, matcher != nil, matcherName)
		}
	}

	return p.finishMatch(match, cases, defaultCase, matcher != nil, matcherName)
}

// rejectMatchArmAfterDefault prevents a match arm from being reinterpreted as an
// ordinary member call after the match parser has reached its terminal default arm.
func (p *parser) rejectMatchArmAfterDefault() {
	if !p.at(scanlex.DOT) || !p.isMemberNameToken(p.peek(1)) {
		return
	}
	switch logicalName(p.peek(1).Value) {
	case "case":
		p.fail(p.peek(1), "a .case(...) arm cannot follow the terminal .default(...) arm of a match")
	case "default":
		p.fail(p.peek(1), "a match may contain at most one .default(...) arm")
	case "otherwise":
		p.fail(p.peek(1), `a match uses .default(...), not .otherwise`)
	}
}

// finishMatch assembles the parsed cases into the pattern-expression node and wraps
// it for expression position.
//
// A match is a statement node in this AST, so it is wrapped in ast.StatementExpr to
// be usable as an operand — which it must be, since a match's value is routinely
// assigned or returned.
func (p *parser) finishMatch(match ast.MatchExprStmt, cases []ast.CaseStmt, defaultCase *ast.CaseStmt, hasMatcher bool, matcherName string) ast.Expr {
	spanStart := p.pos
	if len(cases) == 0 {
		p.reportf(p.cur(), "a match expression requires at least one .case(...) arm")
	}

	patternExpr := ast.PatternExprStmt{Span: p.spanFrom(spanStart), Expr_: match,
		CaseExprStmt:    cases,
		DefaultExprStmt: defaultCase,
		CustomMatcher:   hasMatcher && (matcherName == "" || isCustomMatcher(matcherName)),
		Matcher:         hasMatcher,
		MatcherType:     matcherName,
		Symb:            p.matcherImplSymbol(matcherName),
	}

	return ast.StatementExpr{Span: p.spanFrom(spanStart), Statement: patternExpr,
		Symb: p.exprSymbol("match"),
	}
}

// matcherExpressionName extracts the stable selector name retained by the
// legacy matcher fields. Arbitrary selector expressions intentionally return
// an empty name; their lossless representation is MatchExprStmt.MatcherExpr.
func matcherExpressionName(expr ast.Expr) string {
	switch n := expr.(type) {
	case ast.SymbolExpr:
		return logicalName(n.Value)
	case ast.MemberExpr:
		prefix := matcherExpressionName(n.Member)
		if prefix == "" {
			return ""
		}
		return prefix + "." + logicalName(n.Property)
	case ast.GroupingExpr:
		return matcherExpressionName(n.Expr_)
	case ast.SDTExpr:
		if n.Symb != nil {
			return logicalName(n.Symb.GetName())
		}
	}
	return ""
}

// isCustomMatcher reports whether a matcher name refers to a user-defined matcher
// rather than to one of the built-in co.pattern.* selectors.
func isCustomMatcher(matcherName string) bool {
	if matcherName == "" {
		return false
	}
	const builtinPrefix = "co.pattern."
	return len(matcherName) < len(builtinPrefix) || matcherName[:len(builtinPrefix)] != builtinPrefix
}

// parseMatchCase parses the match-case and match-case-body productions.
//
// Implements: match-case
// Implements: match-case-body
func (p *parser) parseMatchCase() ast.CaseStmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.DOT, "before \"case\"")
	p.expectMemberName("case", "to begin a match case")
	p.expect(scanlex.OPEN_PAREN, "to open a match case")

	pat := p.parsePattern()

	// The optional guard. A guard is only meaningful when the pattern binds a
	// name for it to test, which is the `n: n > 10` shape.
	var guard ast.Expr
	if p.accept(scanlex.COLON) {
		guard = p.parseExpression()
	}

	p.expectOp("=>", "between a match pattern and its result")
	result := p.parseCaseResult("a match case")

	p.expect(scanlex.CLOSE_PAREN, "to close a match case")

	binding := ""
	if names := p.bindingNamesOf(pat); len(names) > 0 {
		binding = names[0]
	}

	return ast.CaseStmt{Span: p.spanFrom(spanStart), Expr_: p.caseSubject(pat, guard),
		Stmt_:   result,
		Binding: binding,
		Symb:    p.stmtSymbol("case"),
	}
}

// caseSubject combines a case's pattern with its guard into the single expression
// the AST node carries.
//
// With no guard the pattern stands alone. With one, the two are joined so the
// semantic phase can see both and enforce the ordering rule: the structural pattern
// is tested first and the guard only afterwards.
func (p *parser) caseSubject(pat pattern, guard ast.Expr) ast.Expr {
	spanStart := p.pos
	if guard == nil {
		return pat.Expr
	}
	return ast.ConditionalExpr{Span: p.spanFrom(spanStart), Left: pat.Expr,
		Right:       guard,
		CondVarStmt: pat.Expr,
		CondValStmt: guard,
		Type:        "guard",
		Symb:        p.exprSymbol("guard"),
	}
}

// parseMatchDefault parses the match-default production:
//
//	match-default = ".default", "(", ( expression | block ), ")"
//
// Implements: match-default
func (p *parser) parseMatchDefault() ast.CaseStmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.DOT, "before \"default\"")
	p.expectMemberName("default", "to begin a match default")
	p.expect(scanlex.OPEN_PAREN, "to open a match default")

	result := p.parseCaseResult("a match default")

	p.expect(scanlex.CLOSE_PAREN, "to close a match default")

	return ast.CaseStmt{Span: p.spanFrom(spanStart), Stmt_: result,
		Default: true,
		Symb:    p.stmtSymbol("default"),
	}
}

// parseCaseResult parses the `( expression | block )` result shared by match-case
// and match-default.
//
// A block result is a value: DECISION-BLK-001 makes its final unterminated
// expression the block's value, which is what
// `case(n: n > 10 => { n = n+100; "GT" })` relies on to yield "GT".
func (p *parser) parseCaseResult(context string) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if p.at(scanlex.OPEN_CURLY) && !p.looksLikeMapLiteral() {
		return p.parseBlock(context)
	}
	return ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: p.parseExpression(),
		Symb: p.stmtSymbol("case-result"),
	}
}
