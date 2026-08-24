package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
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

// Control-flow verbs of the current alpha profile, spelled as the reference
// writes them (docs/grammar/folang.ebnf, section 12).
//
// The vocabulary is closed and each verb has exactly one role:
//
//	.then(X)             conditionally evaluates X once; X is a block or a value.
//	.loop(block)         single-condition repetition.
//	.otherwise(cond)     ALWAYS introduces another selection condition.
//	.default(X)          the optional terminal fallback of a selection chain.
//	.each(…)             element iteration, which owns its own action argument.
//	.contains(v)         a containment test, which then selects with .then.
//
// Three shapes the earlier vocabulary had are gone rather than renamed, and the
// matchers below reject them instead of accepting a second spelling:
//
//   - `.do` is not a control verb in this profile at all. `.then` is the
//     one-shot selection verb.
//   - there is no conditionless `.otherwise`. A chain's fallback is `.default(X)`,
//     which is why the else branch is matched from a verb of its own rather than
//     from an uncalled `.otherwise` segment.
//   - `.loop` takes exactly one condition, so it can be followed by neither
//     `.otherwise(cond)` nor `.default(X)`, and `.each` cannot be followed by
//     `.loop`: iteration is what `.each` already is.
const (
	verbThen      = "then"
	verbLoop      = "loop"
	verbOtherwise = "otherwise"
	verbDefault   = "default"
	verbEach      = "each"
	verbContains  = "contains"
)

// chainSegment is one `.verb` or `.verb(args)` link of a postfix chain.
type chainSegment struct {
	// verb is the member's logical name, with the backend lowering suffix removed.
	verb string
	// args holds the call arguments. It is nil when the member was not called, which is how
	// the argument-less `.match` of a matcher chain presents.
	args []ast.Expr
	// called distinguishes `.match(expr)` from a bare `.match`, since a call with no
	// arguments also yields an empty args slice. Every control verb of this
	// profile is called, so a control matcher requires it.
	called bool
}

// chain is a decomposed postfix chain: the operand it starts from, and its links in source order.
type chain struct {
	// span is the source region of the ORIGINAL expression the chain was
	// decomposed from. Lowering runs after parsing, when the cursor sits at end
	// of file, so every node it builds takes its span from here rather than
	// from p.pos — otherwise a rewritten chain would report the end of the file
	// as its location.
	span     ast.Span
	subject  ast.Expr
	segments []chainSegment
}

// decomposeChain flattens a postfix chain into its subject and segments.
//
// The parser builds `(c).then({…}).default({…})` as a left-nested tree, with the outermost
// node being the last call. Walking inward and prepending therefore recovers source order.
//
// ok is false when the expression is not a member/call chain at all, so a caller can leave it
// alone.
func decomposeChain(e ast.Expr) (chain, bool) {
	var segments []chainSegment
	cur := e

	for {
		switch n := cur.(type) {
		case ast.CallExpr:
			// Parentheses around a member callee are transparent. Keeping the
			// GroupingExpr in the original AST still preserves source structure,
			// while decomposition sees `(items.each)(...)` as the same canonical
			// chain as `items.each(...)`.
			//
			// The grouped spelling also arrives in a different NODE shape: the
			// scanner splits a dotted path into receiver, DOT and member only when
			// "(" follows the member, so the grouped form reaches here as one
			// folded qualified name. splitCallReceiver reads both shapes, which is
			// what keeps the two spellings equivalent rather than lowering only the
			// ungrouped one.
			receiver, verb, isMember := splitCallReceiver(n.Method)
			if !isMember {
				// A call on something that is not a member access ends the chain; the
				// callee becomes the subject, as in `getValue().match…`.
				return chain{span: spanOfNode(e, ast.Span{}), subject: cur, segments: segments}, len(segments) > 0
			}
			segments = append([]chainSegment{{
				verb:   verb,
				args:   n.Arguments,
				called: true,
			}}, segments...)
			cur = receiver

		case ast.MemberExpr:
			segments = append([]chainSegment{{
				verb:   logicalName(n.Property),
				called: false,
			}}, segments...)
			cur = n.Member

		default:
			return chain{span: spanOfNode(e, ast.Span{}), subject: cur, segments: segments}, len(segments) > 0
		}
	}
}

// splitCallReceiver separates a receiver-qualified callee into its receiver and
// the member being invoked.
//
// It reads the two shapes a qualified callee can take. `items.each(…)` is a
// MemberExpr, because the scanner split the path when it saw the "(". The
// grouped `(items.each)(…)` is one folded qualified name, because ")" came
// between the member and the "(" — the parenthesised part is a qualified-name
// primary expression, which is exactly what the grammar makes it.
//
// The reconstructed receiver keeps the whole node's span. Only the receiver's
// NAME is read by the chain lowerings, and a folded name has no separate span
// for its prefix to take.
func splitCallReceiver(callee ast.Expr) (ast.Expr, string, bool) {
	switch target := transparentCallTarget(callee).(type) {
	case ast.MemberExpr:
		return target.Member, logicalName(target.Property), true

	case ast.SymbolExpr:
		dot := strings.LastIndexByte(target.Value, '.')
		if dot < 0 {
			return nil, "", false
		}
		receiver := target
		receiver.Value = target.Value[:dot]
		return receiver, logicalName(target.Value[dot+1:]), true

	default:
		return nil, "", false
	}
}

// verbAt returns the verb of segment i, or "" when i is out of range.
func (c chain) verbAt(i int) string {
	if i < 0 || i >= len(c.segments) {
		return ""
	}
	return c.segments[i].verb
}

// isBranchVerb reports whether a verb takes a branch body: ".then" or ".loop".
//
// They are NOT interchangeable and the two chains they head are separate
// productions. A `.then` chain may continue with `.otherwise(cond).then(block)`
// links and end in `.default(block)`; a `.loop` chain has exactly one condition
// and ends there. lowerConditionalChain enforces that split, so this predicate
// only says "a body follows".
func isBranchVerb(verb string) bool {
	return verb == verbThen || verb == verbLoop
}

// isLoopChainExpression reports whether an expression is a loop statement of the
// current profile — a chain whose OUTER control operation is `.loop(…)`:
//
//	informative-loop-chain = "(", expression, ")", ".loop", "(", block, ")"
//
// labeled-loop-statement-guard needs this to tell `'outer: (c).loop({…});` from a
// label written in front of an arbitrary expression statement. The LAST segment
// decides, not any segment: `.loop` is terminal in this profile, so a chain that
// continues past it is not a loop chain and the guard must not treat it as one.
//
// Implements: informative-loop-chain
// Implements: informative-labeled-loop
func isLoopChainExpression(e ast.Expr) bool {
	if e == nil {
		return false
	}
	decomposed, ok := decomposeChain(e)
	if !ok || len(decomposed.segments) == 0 {
		return false
	}
	last := decomposed.segments[len(decomposed.segments)-1]
	return last.called && last.verb == verbLoop
}

// blockArgument extracts the single block argument of a `.then({…})`,
// `.loop({…})` or `.default({…})` segment.
//
// The parser wraps a block argument in ast.StatementExpr, which is the AST's
// statement-as-expression carrier, so the block is unwrapped back to a Stmt here.
func blockArgument(seg chainSegment) (ast.Stmt, bool) {
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
	return nameOfExpr(e)
}
