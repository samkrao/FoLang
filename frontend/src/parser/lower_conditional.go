package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Conditional and loop chains — informative-condition-chain, informative-loop-chain and
// informative-mixed-chain of section 11a.
//
//	informative-condition-chain =
//	    "(", expression, ")", ".do", "(", block, ")",
//	    { ".otherwise", "(", expression, ")", ".do", "(", block, ")" },
//	    [ ".otherwise", ".do", "(", block, ")" ]
//
// The loop chain is the same shape with ".loop", and the mixed chain allows the two branch verbs
// to alternate, so all three are recognised by one matcher over informative-branch-verb
// (docs/language-ref.md, "Conditions, Loops and Iterators"):
//
//	(pp > 10).do({ … }).otherwise(pp == 11).do({ … }).otherwise.do({ … });
//	(pp > 10).loop({ … });
//	(a).do({ … }).otherwise(b).loop({ … }).otherwise.loop({ … });
//
// Conditions are evaluated left to right and only the selected branch runs
// (docs/language-ref.md, "Conditions and Branches"), which is exactly the semantics the
// resulting ConditionalStmt chain expresses.

// lowerConditionalChain rewrites a condition, loop or mixed chain into an ast.ConditionalStmt.
//
// It returns ok=false when the chain does not match the canonical shape, in which case the caller
// leaves the expression as parsed.
func (p *parser) lowerConditionalChain(c chain) (ast.Stmt, bool) {
	// The chain must open with a branch verb applied to a block: `(cond).do({…})`.
	if len(c.segments) < 1 || !isBranchVerb(c.verbAt(0)) {
		return nil, false
	}
	firstBody, hasBody := blockArgument(c.segments[0])
	if !hasBody {
		return nil, false
	}

	root := ast.ConditionalStmt{
		IfExpr: c.subject,
		IfStmt: p.lowerStatement(firstBody),
		Loop:   c.verbAt(0) == verbLoop,
		Symb:   p.stmtSymbol("ConditionalStmt"),
	}

	// Every branch verb seen, so the aggregate loop flags can be set afterwards.
	verbs := []string{c.verbAt(0)}

	var elifs []ast.ConditionalStmt
	var elseBranch *ast.DefaultConditionalStmt

	// Walk the `.otherwise …` tail in pairs.
	i := 1
	for i < len(c.segments) {
		if c.verbAt(i) != verbOtherwise {
			// Something other than an otherwise continues the chain, which means this is
			// not a control chain after all — for example a pipeline applied to the
			// result. Leaving it unlowered is safer than guessing.
			return nil, false
		}
		otherwise := c.segments[i]

		// The branch verb that follows the otherwise.
		if !isBranchVerb(c.verbAt(i + 1)) {
			return nil, false
		}
		branch := c.segments[i+1]
		body, ok := blockArgument(branch)
		if !ok {
			return nil, false
		}
		verbs = append(verbs, branch.verb)

		if otherwise.called {
			// `.otherwise(cond).do({…})` is a further guarded branch.
			cond, hasCond := singleArgument(otherwise)
			if !hasCond {
				return nil, false
			}
			elifs = append(elifs, ast.ConditionalStmt{
				IfExpr: cond,
				IfStmt: p.lowerStatement(body),
				Loop:   branch.verb == verbLoop,
				Symb:   p.stmtSymbol("ConditionalStmt"),
			})
		} else {
			// A bare `.otherwise.do({…})` is the final unguarded branch, and nothing may
			// follow it.
			if i+2 != len(c.segments) {
				return nil, false
			}
			elseBranch = &ast.DefaultConditionalStmt{
				Stmt_:   p.lowerStatement(body),
				Default: true,
				Loop:    branch.verb == verbLoop,
				Symb:    p.stmtSymbol("DefaultConditionalStmt"),
			}
		}
		i += 2
	}

	root.ElifExprStmt = elifs
	root.ElseExprStmt = elseBranch
	applyLoopFlags(&root, verbs)

	// A condition that is itself a containment call is decomposed like the dedicated contains
	// chain, so `(arr.contains(k)).do({…})` and `arr.contains(k).do({…})` yield the same tree.
	if replaced, isContains := p.containsCondition(root.IfExpr); isContains {
		root.IfExpr = replaced
		root.ISParentArrCont = true
	}
	return root, true
}

// applyLoopFlags sets the aggregate loop flags from the branch verbs the chain used.
//
// ContainsLoop records that at least one branch loops, and OnlyLoop that every branch does. A
// mixed chain sets the former and not the latter, which is what lets a consumer tell
// informative-mixed-chain from a pure loop chain.
func applyLoopFlags(cond *ast.ConditionalStmt, verbs []string) {
	anyLoop, allLoop := false, len(verbs) > 0
	for _, v := range verbs {
		if v == verbLoop {
			anyLoop = true
		} else {
			allLoop = false
		}
	}
	cond.ContainsLoop = anyLoop
	cond.OnlyLoop = allLoop

	// The same flags are carried on the else branch, whose consumer sees it independently of
	// the conditional that owns it.
	if cond.ElseExprStmt != nil {
		cond.ElseExprStmt.ContainsLoop = anyLoop
		cond.ElseExprStmt.OnlyLoop = allLoop
	}
	for i := range cond.ElifExprStmt {
		cond.ElifExprStmt[i].ContainsLoop = anyLoop
		cond.ElifExprStmt[i].OnlyLoop = allLoop
	}
}
