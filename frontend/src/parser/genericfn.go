package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_generic_function_declaration handles functions annotated with
// @co.dap.generic, e.g.:
//
//	@co.dap.generic(typename=T)
//	identity(x T)->(T) = { this.return x; }
//
//	@co.dap.generic(
//	    at=runtime, refied=true, where=callsite,
//	    type={T:{variance:invariant,bound=Number,Kind:Param}, R:{...}})
//	map(f (T)->(R), xs List(T))->(List(R)) = { this.return xs.each(f); }
//
// It is invoked from callFunction in stmtsup.go when @co.dap.generic is present.
func parse_generic_function_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}

	// 1. Read type parameters and control fields from the annotation.
	typeParams := extractGenericTypeParams(p, ddaps)
	scope, where, reified := genericAnnotationMeta(p, ddaps)

	// 2. Build the type-param name list used for validation.
	typeParamNames := make([]string, len(typeParams))
	for i, tp := range typeParams {
		typeParamNames[i] = tp.Name
	}

	// 3. Set parser state so that every nested param/return type parse goes
	//    through the right branch.
	//
	//    p.GenericType = true
	//        → tells the identifier nud handler to accept unknown names as
	//          GENERIC SymbolTypeNodes instead of raising "type not found".
	//
	//    p.AdditionalInfo["genericfunction"] = true
	//    p.AdditionalInfo["generrictypes"]   = []string{...}
	//        → activates the parse_generic_types() branch in parse_decl_stmt
	//          (variabledecl.go), which wraps each type var in ast.GenericType{}
	//          and validates the name against the declared list.
	//          This is the path that correctly handles function-type parameters
	//          like `f (T)->(R)` where T and R are generic variables, because
	//          p.AdditionalInfo is parser-level state that persists through the
	//          nested parse_fn_gen_declaration calls for function-type params.
	prevGeneric := p.GenericType
	p.GenericType = true

	prevGenericFn, hadGenericFn := p.AdditionalInfo["genericfunction"]
	prevGenricTypes, hadGenricTypes := p.AdditionalInfo["generrictypes"] // double-r matches existing key spelling
	p.AdditionalInfo["genericfunction"] = true
	p.AdditionalInfo["generrictypes"] = typeParamNames

	tr := parse_fn_declaration(p, GENERIC, ddaps)

	// 4. Restore parser state — must happen even if parse failed.
	p.GenericType = prevGeneric
	if hadGenericFn {
		p.AdditionalInfo["genericfunction"] = prevGenericFn
	} else {
		delete(p.AdditionalInfo, "genericfunction")
	}
	if hadGenricTypes {
		p.AdditionalInfo["generrictypes"] = prevGenricTypes
	} else {
		delete(p.AdditionalInfo, "generrictypes")
	}

	if tr.Node == nil {
		return pr
	}

	// 5. Stamp generic-specific flags on the function node.
	fn := *(tr.Node.(*ast.FunctionDeclarationStmt))
	fn.Symb.IsGeneric = true
	fn.Scope = "LEXICAL"

	symb := symboltable.GenericDetails{
		Scope_:           scope,
		Where:            where,
		Reified:          reified,
		GenericTypeParam: typeParams,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_GenericDetails),
		},
	}
	// 6. Wrap in GenerricFun.
	gfn := ast.GenerricFun{
		FunctionDeclarationStmt: fn,
		Type_:                   "generic",
		Generic:                 symb,
	}

	pr.Node = &gfn
	return pr
}

// genericAnnotationMeta extracts the control parameters from a @co.dap.generic
// annotation: `at` (scope), `where`, and `refied`.
//
// Defaults: scope="compiletime", where="callsite", reified=false.
func genericAnnotationMeta(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (scope, where string, reified bool) {
	defer p.traceCurrent()()

	scope = "compiletime"
	where = "callsite"

	ann, ok := getAnn(ddaps, "@co.dap.generic")
	if !ok {
		return
	}
	if at, ok := ann.Parameters["at"].(string); ok && at != "" {
		scope = at
	}
	if w, ok := ann.Parameters["where"].(string); ok && w != "" {
		where = w
	}
	// Spec spells this "refied" (one 'i' — not a typo in the language spec).
	if r, ok := ann.Parameters["refied"].(bool); ok {
		reified = r
	}
	return
}
