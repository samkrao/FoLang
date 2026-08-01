package parser

import (
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Iterator chains — informative-each-chain of section 11a.
//
//	informative-each-chain =
//	    postfix-expression, ".each", "(", ( identifier | "_" ), ",", identifier,
//	    ")", ".do", "(", block, ")"
//
// The two arguments bind the key/index and the value, and the index may be discarded with the
// contextual "_" (docs/language-ref.md, "Looping Arrays / Lists / Maps / Ranges"):
//
//	arr.each(idx, val).do({
//	    co.out.print(idx);
//	    co.out.println(val);
//	});
//
//	arr.each(_, val).do({
//	    co.out.println(val);
//	});
//
// "_" is admitted here precisely because an iterator construct is one of the three contexts that
// spell the wildcard directly, alongside pattern matching and containment.

// lowerEachChain rewrites an iterator chain into an ast.ForeachStmt.
func (p *parser) lowerEachChain(c chain) (ast.Stmt, bool) {
	if c.verbAt(0) != verbEach || c.verbAt(1) != verbDo {
		return nil, false
	}
	// The chain is exactly `.each(k, v).do({…})`; anything trailing it is not this shape.
	if len(c.segments) != 2 {
		return nil, false
	}

	each := c.segments[0]
	if !each.called || len(each.args) != 2 {
		return nil, false
	}

	keyName, keyOk := iteratorKeyName(each.args[0])
	valueName, valueOk := nameOfExpr(each.args[1])
	if !keyOk || !valueOk {
		return nil, false
	}

	body, hasBody := blockArgument(c.segments[1])
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
		VarName:        valueName,
		AccessorKeyIdx: keyName,
		Accessor:       collection,
		Body:           p.lowerStatement(body),
		Method:         verbEach,
		VarDetails:     p.iteratorVarDetails(valueName),
		Symb:           p.stmtSymbol("ForeachStmt"),
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
