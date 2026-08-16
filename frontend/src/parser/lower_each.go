package parser

import (
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Iterator chains — informative-each-chain of section 12 of
// docs/grammar/folang.ebnf.
//
//	informative-each-chain =
//	    postfix-expression, ".each", "(",
//	    ( ( identifier | "_" ), ",", identifier, ",",
//	      ( expression | lambda-expression )
//	    | ( expression | lambda-expression ) ), ")"
//
// `.each` OWNS element iteration in this profile: the per-element action is its
// own third argument rather than a `.do`/`.loop` segment chained after it, and an
// `.each(…)` call cannot be followed by `.loop(…)` because iteration is what it
// already is (docs/language-ref.md, "Looping Arrays / Lists / Maps / Ranges"):
//
//	arr.each(idx, val, {
//	    co.out.print(idx);
//	    co.out.println(val);
//	});
//
//	arr.each(_, val, { co.out.println(val); });
//	arr.each(|idx, val| => co.out.println(val));
//	arr.each(handler);
//
// Two forms, one verb. The explicit-binding form names the index/key and the
// value and then supplies the action; the single-argument form supplies a
// callable callback whose parameters semantic analysis matches against the
// receiver's iteration tuple.
//
// "_" is admitted for the key/index slot precisely because an iterator index may
// be ignored. The value binding cannot be discarded; it is the element on which
// the body operates.

// lowerEachChain rewrites an iterator chain into an ast.ForeachStmt.
func (p *parser) lowerEachChain(c chain) (ast.Stmt, bool) {
	if c.verbAt(0) != verbEach {
		return nil, false
	}
	// `.each(…)` is the complete chain. A trailing segment — `.loop(…)` above
	// all — is not this shape, so the expression is left as parsed.
	if len(c.segments) != 1 {
		return nil, false
	}

	each := c.segments[0]
	if !each.called {
		return nil, false
	}

	// Only the explicit-binding form names an index and a value the AST can
	// record. The single-argument callback form binds nothing at the source
	// level: its parameters belong to the callable, and matching them against the
	// receiver's iteration tuple needs the receiver's type. It therefore stays an
	// ordinary call for the semantic phase.
	if len(each.args) != 3 {
		return nil, false
	}

	keyName, keyOk := iteratorKeyName(each.args[0])
	valueName, valueOk := nameOfExpr(each.args[1])
	if !keyOk || !valueOk {
		return nil, false
	}

	body, hasBody := p.eachAction(each.args[2])
	if !hasBody {
		return nil, false
	}

	collection, hasCollection := subjectName(c.subject)
	if !hasCollection {
		// ForeachStmt stores only a collection name. Keeping a complex receiver
		// as its generic call chain prevents a lossy empty-name lowering.
		return nil, false
	}

	return ast.ForeachStmt{
		Span:           c.span,
		VarName:        valueName,
		AccessorKeyIdx: keyName,
		Accessor:       collection,
		Body:           p.lowerStatement(body),
		Method:         verbEach,
		VarDetails:     p.iteratorVarDetails(valueName),
		Symb:           p.stmtSymbol("ForeachStmt"),
	}, true
}

// eachAction converts the per-element action of the explicit-binding form into
// the body statement ForeachStmt holds.
//
// The grammar admits any expression in the action slot, and the reference writes
// both a braced action and a bare call there:
//
//	arr.each(_, val, { co.out.println(val); });
//	arr.each(_, val, co.out.println(val));
//
// A braced action already IS a block and is unwrapped; any other expression
// becomes the single expression statement of the body, which loses nothing.
//
// A lambda is refused. `arr.each(|idx, val| => …)` is the SINGLE-argument
// callback form, whose bindings belong to the callable rather than to the two
// name slots this form fills; lowering it here would record the iterator's
// bindings twice under different names.
func (p *parser) eachAction(action ast.Expr) (ast.Stmt, bool) {
	if wrapper, isWrapped := action.(ast.StatementExpr); isWrapped && wrapper.Statement != nil {
		return wrapper.Statement, true
	}
	if _, isLambda := action.(ast.LambdaExpr); isLambda {
		return nil, false
	}
	return ast.ExpressionStmt{
		Span:       spanOfNode(action, ast.Span{}),
		Expression: p.lowerExpr(action),
		Symb:       p.stmtSymbol("each-action"),
	}, true
}

// iteratorKeyName accepts the one wildcard position granted by the grammar.
func iteratorKeyName(e ast.Expr) (string, bool) {
	if symbol, ok := e.(ast.SymbolExpr); ok && (symbol.SymbolType_ == "wildcard" || logicalName(symbol.Value) == "_") {
		return symbol.Value, true
	}
	return nameOfExpr(e)
}

// iteratorVarDetails describes the value variable an iterator binds.
//
// The element type is not known syntactically — it follows from the collection being walked — so
// it is left to be inferred, which is what co.lang.infer records.
func (p *parser) iteratorVarDetails(valueName string) symboltable.SymbolDetails {
	return p.details(valueName, symboltable.S_VarSymbol, "co.lang.infer")
}
