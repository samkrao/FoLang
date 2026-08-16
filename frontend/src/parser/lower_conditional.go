package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Condition and loop chains — informative-condition-chain and
// informative-loop-chain of section 12 of docs/grammar/folang.ebnf.
//
//	informative-condition-chain =
//	    "(", expression, ")", ".then", "(", block, ")",
//	    { ".otherwise", "(", expression, ")", ".then", "(", block, ")" },
//	    [ ".default", "(", block, ")" ]
//
//	informative-loop-chain =
//	    "(", expression, ")", ".loop", "(", block, ")"
//
// FoLang has no if, else, for or while keyword: a selection is a chain of
// associated-function calls, so it PARSES as an ordinary postfix expression and
// is recognised here, after parsing, over the tree the postfix parser built
// (docs/language-ref.md, "Conditions, Loops and Iterators"):
//
//	(pp > 10).then({ … }).otherwise(pp == 11).then({ … }).default({ … });
//	(pp > 10).loop({ … });
//
// The two chains are separate productions rather than one shape with two branch
// verbs, and this matcher keeps them apart:
//
//   - `.loop` carries exactly ONE condition. It cannot be followed by
//     `.otherwise(condition)` or by `.default(X)`, because neither belongs to a
//     repetition — `.otherwise` introduces another SELECTION condition and
//     `.default` is a selection chain's terminal fallback.
//   - `.otherwise` is never conditionless. A selection's fallback is spelled
//     `.default(X)`, so an uncalled `.otherwise` segment matches nothing.
//
// Conditions are evaluated left to right and only the selected branch runs
// (docs/language-ref.md, "Conditions and Branches"), which is exactly the
// semantics the resulting ConditionalStmt chain expresses.

// lowerConditionalChain rewrites a condition or loop chain into an
// ast.ConditionalStmt.
//
// It returns ok=false when the chain does not match a canonical shape, in which
// case the caller leaves the expression as parsed.
func (p *parser) lowerConditionalChain(c chain) (ast.Stmt, bool) {
	// The chain must open with a branch verb applied to a block:
	// `(cond).then({…})` or `(cond).loop({…})`.
	if len(c.segments) < 1 || !isBranchVerb(c.verbAt(0)) {
		return nil, false
	}
	if !c.segments[0].called {
		return nil, false
	}
	firstBody, hasBody := blockArgument(c.segments[0])
	if !hasBody {
		return nil, false
	}

	isLoop := c.verbAt(0) == verbLoop

	// A loop is a complete chain on its own. Anything after its single condition
	// is not loop syntax, so the whole expression is left as parsed rather than
	// lowered into a selection it does not describe.
	if isLoop && len(c.segments) != 1 {
		return nil, false
	}

	root := ast.ConditionalStmt{
		Span:   c.span,
		IfExpr: p.lowerExpr(c.subject),
		IfStmt: p.lowerStatement(firstBody),
		Loop:   isLoop,
		Symb:   p.stmtSymbol("ConditionalStmt"),
	}

	var elifs []ast.ConditionalStmt
	var elseBranch *ast.DefaultConditionalStmt

	// Walk the `.otherwise(cond).then(block)` tail in pairs, then the optional
	// terminal `.default(block)`.
	i := 1
	for i < len(c.segments) {
		// `.default(block)` closes the chain and takes no partner segment.
		if c.verbAt(i) == verbDefault {
			if i+1 != len(c.segments) {
				return nil, false
			}
			body, hasDefault := blockArgument(c.segments[i])
			if !hasDefault {
				return nil, false
			}
			elseBranch = &ast.DefaultConditionalStmt{
				Span:    c.span,
				Stmt_:   p.lowerStatement(body),
				Default: true,
				Symb:    p.stmtSymbol("DefaultConditionalStmt"),
			}
			i++
			continue
		}

		if c.verbAt(i) != verbOtherwise {
			// Something other than an otherwise continues the chain, which means this is
			// not a control chain after all — for example a pipeline applied to the
			// result. Leaving it unlowered is safer than guessing.
			return nil, false
		}
		otherwise := c.segments[i]

		// `.otherwise` always carries the next selection condition.
		cond, hasCond := singleArgument(otherwise)
		if !hasCond {
			return nil, false
		}

		// Only `.then` continues a selection chain; `.loop` cannot appear in one.
		if c.verbAt(i+1) != verbThen {
			return nil, false
		}
		body, ok := blockArgument(c.segments[i+1])
		if !ok {
			return nil, false
		}

		elifs = append(elifs, ast.ConditionalStmt{
			Span:   c.span,
			IfExpr: p.lowerExpr(cond),
			IfStmt: p.lowerStatement(body),
			Symb:   p.stmtSymbol("ConditionalStmt"),
		})
		i += 2
	}

	root.ElifExprStmt = elifs
	root.ElseExprStmt = elseBranch
	applyLoopFlags(&root, isLoop)

	// A condition that is itself a containment call is decomposed like the dedicated contains
	// chain, so `(arr.contains(k)).then({…})` and `arr.contains(k).then({…})` yield the same tree.
	if replaced, isContains := p.containsCondition(root.IfExpr); isContains {
		root.IfExpr = replaced
		root.ISParentArrCont = true
	}
	return root, true
}

// applyLoopFlags sets the aggregate loop flags.
//
// A chain is either one `.loop` condition or a `.then`/`.otherwise` selection;
// this profile has no mixed chain, so ContainsLoop and OnlyLoop always agree and
// both simply record which of the two productions matched. They are kept as two
// fields because consumers read them independently.
func applyLoopFlags(cond *ast.ConditionalStmt, isLoop bool) {
	cond.ContainsLoop = isLoop
	cond.OnlyLoop = isLoop

	// The same flags are carried on the else branch, whose consumer sees it independently of
	// the conditional that owns it.
	if cond.ElseExprStmt != nil {
		cond.ElseExprStmt.ContainsLoop = isLoop
		cond.ElseExprStmt.OnlyLoop = isLoop
	}
	for i := range cond.ElifExprStmt {
		cond.ElifExprStmt[i].ContainsLoop = isLoop
		cond.ElifExprStmt[i].OnlyLoop = isLoop
	}
}
