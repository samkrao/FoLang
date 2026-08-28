package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
)

// Containment chains — informative-contains-chain of section 12 of
// docs/grammar/folang.ebnf.
//
//	informative-contains-chain =
//	    postfix-expression, ".contains", "(", expression, ")",
//	    ".then", "(", block, ")",
//	    [ ".default", "(", block, ")" ]
//
// A containment test branches on whether a collection holds a value
// (docs/language-ref.md, "Array / List / Map / Range — Contains Element"):
//
//	arr.contains(k).then({
//	    co.out.println(k);
//	}).default({
//	    co.out.println("Not Found");
//	});
//
// The test selects with the ordinary selection vocabulary, so the fallback is
// `.default(block)` — the terminal fallback of a selection chain — and not a
// conditionless `.otherwise`, which this profile does not have.
//
// The result is an ast.ConditionalStmt whose ISParentArrCont flag is set — that flag exists for
// exactly this case — with the test itself decomposed into an ast.ContainsStmt so a consumer gets
// the collection, the method and the searched value without re-walking a call chain.
//
// ContainsStmt is a Stmt while a conditional's test is an Expr, so it travels in an
// ast.StatementExpr, the AST's statement-as-expression carrier.
//
// `.containsVal` is accepted alongside `.contains`: the built-in method table lists it as the
// map/dictionary form of the same operation, and the chain shape is identical.

// containsVerbs are the containment methods that take this chain shape.
var containsVerbs = map[string]bool{
	verbContains:    true,
	verbContainsVal: true,
}

// lowerContainsChain rewrites a containment chain into an ast.ConditionalStmt over an
// ast.ContainsStmt.
func (p *parser) lowerContainsChain(c chain) (ast.Stmt, bool) {
	if !containsVerbs[c.verbAt(0)] || c.verbAt(1) != verbThen {
		return nil, false
	}

	test := c.segments[0]
	searched, hasSearched := singleArgument(test)
	if !hasSearched {
		return nil, false
	}

	thenBody, hasThen := blockArgument(c.segments[1])
	if !hasThen {
		return nil, false
	}

	// The optional terminal `.default({…})`.
	var elseBranch *ast.DefaultConditionalStmt
	switch len(c.segments) {
	case 2:
		// No fallback branch.
	case 3:
		if c.verbAt(2) != verbDefault {
			return nil, false
		}
		elseBody, hasElse := blockArgument(c.segments[2])
		if !hasElse {
			return nil, false
		}
		elseBranch = &ast.DefaultConditionalStmt{
			Span:     c.span,
			Stmt_:    p.lowerStatement(elseBody),
			Default:  true,
			Loop:     false,
			SymbolId: p.statementID("DefaultConditionalStmt"),
		}
	default:
		return nil, false
	}

	collection, hasCollection := subjectName(c.subject)
	if !hasCollection {
		// ContainsStmt has no expression-valued receiver field. Preserve a
		// complex receiver as the generic chain instead of dropping it.
		return nil, false
	}

	containment := ast.ContainsStmt{
		Span:     c.span,
		VarName:  collection,
		VarType:  "co.lang.infer",
		Method:   test.verb,
		Accessor: p.containsSearchedValue(searched),
		SymbolId: p.statementID("ContainsStmt"),
	}

	return ast.ConditionalStmt{
		Span: c.span,
		IfExpr: ast.StatementExpr{
			Span:      c.span,
			Statement: containment,
			Symb:      p.exprSymbol(test.verb),
		},
		IfStmt:          p.lowerStatement(thenBody),
		ElseExprStmt:    elseBranch,
		Loop:            false,
		ContainsLoop:    false,
		ISParentArrCont: true,
		SymbolId:        p.statementID("ConditionalStmt"),
	}, true
}

// containsCondition recognises a containment call used as an ordinary condition and decomposes it
// the same way lowerContainsChain does.
//
// The corpus writes the containment test both ways:
//
//	arr.contains(k).then({ … })      the dedicated contains chain
//	(arr.contains(k)).then({ … })    a condition chain whose test is a containment call
//
// Both are well formed — the second matches informative-condition-chain, whose condition is any
// expression — but a consumer should not have to tell them apart. This recognises the second form
// so it produces the same ContainsStmt and the same ISParentArrCont flag as the first.
//
// The returned expression replaces the condition; ok is false when the condition is not a
// containment call, leaving it untouched.
func (p *parser) containsCondition(cond ast.Expr) (ast.Expr, bool) {
	call, isCall := unwrapGrouping(cond).(ast.CallExpr)
	if !isCall {
		return nil, false
	}
	member, isMember := call.Method.(ast.MemberExpr)
	if !isMember {
		return nil, false
	}
	verb := logicalName(member.Property)
	if !containsVerbs[verb] || len(call.Arguments) != 1 {
		return nil, false
	}
	collection, hasCollection := subjectName(member.Member)
	if !hasCollection {
		return nil, false
	}

	containment := ast.ContainsStmt{
		Span:     spanOfNode(cond, ast.Span{}),
		VarName:  collection,
		VarType:  "co.lang.infer",
		Method:   verb,
		Accessor: p.containsSearchedValue(call.Arguments[0]),
		SymbolId: p.statementID("ContainsStmt"),
	}
	return ast.StatementExpr{
		Span:      spanOfNode(cond, ast.Span{}),
		Statement: containment,
		Symb:      p.exprSymbol(verb),
	}, true
}

// unwrapGrouping strips redundant parentheses so a condition can be inspected for its shape.
func unwrapGrouping(e ast.Expr) ast.Expr {
	for {
		group, isGroup := e.(ast.GroupingExpr)
		if !isGroup {
			return e
		}
		e = group.Expr_
	}
}

// containsSearchedValue wraps the value being searched for as the statement ContainsStmt.Accessor
// holds.
func (p *parser) containsSearchedValue(searched ast.Expr) ast.Stmt {
	return ast.ExpressionStmt{
		Span:       spanOfNode(searched, ast.Span{}),
		Expression: p.lowerExpr(searched),
		SymbolId:   p.statementID("contains-value"),
	}
}
