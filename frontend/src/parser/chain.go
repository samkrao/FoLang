package parser

import (
	"fmt"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// Control-flow chain decomposition — section 11a of docs/grammar/folang.ebnf.
//
// FoLang has no if, else, for, while or foreach keyword. Every branch, loop, iterator and
// ternary is an associated-function chain, so it PARSES as an ordinary postfix expression of
// member and call suffixes where an argument may be a block. The grammar states this plainly and
// then forbids a second parse path:
//
//	"The productions below are INFORMATIVE. They document the canonical shapes the semantic
//	 analyzer enforces; they are not a second parse path and must not be entered from
//	 expression. Enforcing these shapes in the parser instead would require unbounded lookahead
//	 to distinguish a control chain from any other method chain, so the check belongs to
//	 semantic analysis."
//
// So the recognition happens HERE, after parsing, over the tree the postfix parser already
// built. That respects the grammar — expression parsing stays uniform and needs no lookahead —
// while still producing the dedicated AST nodes a backend needs. A chain that does not match a
// canonical shape is left exactly as parsed.
//
// This file does the shape-independent half: turning a left-nested MemberExpr/CallExpr tree back
// into a flat subject-plus-segments form that the individual lowerings can pattern-match.

// Control-flow verbs, spelled as the reference writes them.
const (
	verbDo        = "do"
	verbLoop      = "loop"
	verbOtherwise = "otherwise"
	verbReturn    = "return"
	verbEach      = "each"
	verbContains  = "contains"
)

// chainSegment is one `.verb` or `.verb(args)` link of a postfix chain.
type chainSegment struct {
	// verb is the member's logical name, with the backend lowering suffix removed.
	verb string
	// args holds the call arguments. It is nil when the member was not called, which is how
	// the bare `.otherwise` of a final else branch presents.
	args []ast.Expr
	// called distinguishes `.otherwise(cond)` from a bare `.otherwise`, since a call with no
	// arguments also yields an empty args slice.
	called bool
}

// chain is a decomposed postfix chain: the operand it starts from, and its links in source order.
type chain struct {
	subject  ast.Expr
	segments []chainSegment
}

// decomposeChain flattens a postfix chain into its subject and segments.
//
// The parser builds `(c).do({…}).otherwise.do({…})` as a left-nested tree, with the outermost
// node being the last call. Walking inward and prepending therefore recovers source order.
//
// ok is false when the expression is not a member/call chain at all, so a caller can leave it
// alone.
func decomposeChain(e ast.Expr) (chain, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.decomposeChain -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	var segments []chainSegment
	cur := e

	for {
		switch n := cur.(type) {
		case ast.CallExpr:
			// Parentheses around a member callee are transparent. Keeping the
			// GroupingExpr in the original AST still preserves source structure,
			// while decomposition sees `(items.each)(...).do(...)` as the same
			// canonical chain as `items.each(...).do(...)`.
			member, isMember := transparentCallTarget(n.Method).(ast.MemberExpr)
			if !isMember {
				// A call on something that is not a member access ends the chain; the
				// callee becomes the subject, as in `getValue().match…`.
				return chain{subject: cur, segments: segments}, len(segments) > 0
			}
			segments = append([]chainSegment{{
				verb:   logicalName(member.Property),
				args:   n.Arguments,
				called: true,
			}}, segments...)
			cur = member.Member

		case ast.MemberExpr:
			segments = append([]chainSegment{{
				verb:   logicalName(n.Property),
				called: false,
			}}, segments...)
			cur = n.Member

		default:
			return chain{subject: cur, segments: segments}, len(segments) > 0
		}
	}
}

// verbAt returns the verb of segment i, or "" when i is out of range.
func (c chain) verbAt(i int) string {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.verbAt -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if i < 0 || i >= len(c.segments) {
		return ""
	}
	return c.segments[i].verb
}

// isBranchVerb reports whether a verb is one of the two branch verbs of
// informative-branch-verb: ".do" or ".loop". A chain may mix them freely
// (informative-mixed-chain).
func isBranchVerb(verb string) bool {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.isBranchVerb -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return verb == verbDo || verb == verbLoop
}

// blockArgument extracts the single block argument of a `.do({…})` or `.loop({…})` segment.
//
// The parser wraps a block argument in ast.StatementExpr, which is the AST's
// statement-as-expression carrier, so the block is unwrapped back to a Stmt here.
func blockArgument(seg chainSegment) (ast.Stmt, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.blockArgument -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if !seg.called || len(seg.args) != 1 {
		return nil, false
	}
	wrapper, isWrapped := seg.args[0].(ast.StatementExpr)
	if !isWrapped || wrapper.Statement == nil {
		return nil, false
	}
	return wrapper.Statement, true
}

// singleArgument extracts the single expression argument of a segment such as
// `.otherwise(cond)`, `.contains(v)` or `.return(v)`.
func singleArgument(seg chainSegment) (ast.Expr, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.singleArgument -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	if !seg.called || len(seg.args) != 1 {
		return nil, false
	}
	return seg.args[0], true
}

// nameOfExpr returns the source name an expression denotes, for the chain forms that bind names
// rather than evaluate operands — the iterator variables of `.each(idx, val)`.
//
// The scanned spelling is returned, since that is what the rest of the AST stores.
func nameOfExpr(e ast.Expr) (string, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.nameOfExpr -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	switch n := e.(type) {
	case ast.SymbolExpr:
		// The wildcard is accepted for the key/index slot of `.each`, but it
		// never denotes a value that another construct can bind or search.
		if n.SymbolType_ == "wildcard" || logicalName(n.Value) == "_" {
			return "", false
		}
		return n.Value, true
	case ast.GroupingExpr:
		return nameOfExpr(n.Expr_)
	}
	return "", false
}

// subjectName renders a chain's subject as a name when it is one, which the iterator and
// containment nodes record as the collection being walked or searched.
func subjectName(e ast.Expr) (string, bool) {
	file, line, funname := helpers.Trace()
	fmt.Println("--- in chain.subjectName -----")
	fmt.Printf("The file %s is and the line is %d, and the function calling is %s\n", file, line, funname)
	fmt.Println("--------------------------------")

	return nameOfExpr(e)
}
