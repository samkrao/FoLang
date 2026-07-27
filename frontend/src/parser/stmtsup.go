package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// callFunction dispatches function-like declarations based on their
// annotations.
//
// Feature examples:
//
//	@co.dap.generic
//	identity(x T)->(T) = { this.return x; }
//
//	@co.dap.template
//	max(a co.lang.int, b co.lang.int)->(co.lang.int) = { ... }
//
//	@co.dap.indexer(symbol="[]")
//	get(index co.lang.int)->(co.lang.int) = { ... }
//
// The writing style intentionally stays close to the rest of this codebase:
// read annotations from ddaps, select the matching parser, then delegate to
// the feature-specific parse_* routine.
func callFunction(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	//p.advanceN(index)
	pr := ParseResult{}
	if _, ok := isAnnDec(ddaps, "@co.dap.macro"); ok {
		pr = parse_macro_declaration(p, ddaps)
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.operator"); ok {
		pr = parse_operator_function(p, ddaps)
		return pr
	} else if name, ok := isAnnDec(ddaps, "@co.dap.decorator"); ok {
		pr = parse_annotation_declaration(p, name, ddaps)
		return pr
	} else if name, ok := isAnnDec(ddaps, "@co.dap.template", "@co.dap.inline"); ok {
		pr = parse_template_declaration(p, name, ddaps)
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.generic"); ok {
		pr = parse_generic_function_declaration(p, ddaps)
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.typeclass"); ok {
		return parse_typeclass_by_kind(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.functor"); ok {
		return parse_functor_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.applicative"); ok {
		return parse_applicative_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.monad", "@co.dap.monod"); ok {
		return parse_monad_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.semigroup"); ok {
		return parse_semigroup_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.monoid"); ok {
		return parse_monoid_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.transformer"); ok {
		return parse_transformer_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.foldable"); ok {
		return parse_foldable_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.traversable"); ok {
		return parse_traversable_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.matcher"); ok {
		return parse_matcher_declaration(p, ddaps)
	} else if _, ok := isAnnDec(ddaps, "@co.dap.typefromvalue"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.typefromvalue"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.native"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.native"}
		pr.Node = n
		n.Symb.RestrictedToOverload = true
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.constructor"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.constructor"}
		n.Symb.IsMeth = true
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.method.static", "@co.dap.method.instance", "@co.dap.method.class", "@co.dap.method.object"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		n.Symb.IsMeth = true
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.override", "@co.dap.overload", "@co.dap.virtual", "@co.dap.abstract"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		n.Symb.IsMeth = true
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.continuation"); ok {
		//cps
		//shift reset
		//prompt control
		//trampolining
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.continuation"}
		pr.Node = n
		return pr

	} else if _, ok := isAnnDec(ddaps, "@co.dap.event"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.event"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.subroutine"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.subroutine"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.generator"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.generator"}
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.coroutine", "@co.dap.goroutine"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.async", "@co.dap.promise", "@co.dap.future"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.thread"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.task"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.task"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.fiber"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.fiber"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.process"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.process"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.exec"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.exec"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.spawn"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.spawn"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.fork"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.fork"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.callback"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.callback"}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.defer"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.defer"}
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.csp", "@co.dap.actor"); ok {
		//channels communication sequential processes
		// actors
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.eager"); ok {
		if _, ok := isAnnDec(ddaps, "@co.dap.comptime"); ok {
			pr = parse_fn_declaration(p, FUNCTION, ddaps)
			n := pr.Node.(ast.FunctionDeclarationStmt)
			n.WhatisIt = []string{"@co.dap.eager", "@co.dap.comptime"}
			pr.Node = n
			return pr
		} else if _, ok := isAnnDec(ddaps, "@co.dap.eval"); ok {
			pr = parse_fn_declaration(p, FUNCTION, ddaps)
			n := pr.Node.(ast.FunctionDeclarationStmt)
			n.WhatisIt = []string{"@co.dap.eager", "@co.dap.eval"}
			pr.Node = n
			return pr
		} else {
			pr = parse_fn_declaration(p, FUNCTION, ddaps)
			n := pr.Node.(ast.FunctionDeclarationStmt)
			n.WhatisIt = []string{"@co.dap.eager"}
			pr.Node = n
			return pr
		}

	} else if _, ok := isAnnDec(ddaps, "@co.dap.lazy"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{"@co.dap.lazy"}
		pr.Node = n
		return pr
	} else if nam, ok := isAnnDec(ddaps, "@co.dap.lambda", "@co.dap.anonymous"); ok {
		pr = parse_fn_declaration(p, FUNCTION, ddaps)
		n := pr.Node.(ast.FunctionDeclarationStmt)
		n.WhatisIt = []string{nam}
		pr.Node = n
		return pr
	} else if _, ok := isAnnDec(ddaps, "@co.dap.indexer"); ok {
		pr = parse_indexer_declaration(p, ddaps)
		return pr
	} else if p.nextToken(index).Kind == scanlex.CLOSE_BRACKET && p.nextToken(index-1).Kind == scanlex.OPEN_BRACKET {
		pr = parse_indexer_declaration(p, ddaps)
		return pr
	} else {

		pr = parse_fn_declaration(p, FUNCTION, ddaps)
	}
	return pr

}

func parse_type_kind_block(p *parser, parentKind StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	// @@new / @@init are only valid inside class bodies; error if seen elsewhere
	if p.currentTokenKind() == scanlex.DOUBLE_AT {
		p.addErr(p.errorObj(nil, "@@new / @@init special methods are only valid inside class declarations"))
		p.advance() // skip @@
		if p.currentTokenKind() == scanlex.IDENTIFIER && slices.Contains([]string{"new", "init"}, p.currentToken().Value) {
			p.advance() // skip method name
		} else {
			err_ := p.errorExpection("Expected @@new or @@init but found "+p.currentToken().Value, helpers.InvalidSyntax)
			p.addErr(err_)
		}
		return ParseResult{}
	}

	if isTypeConstructorDecl(p, -1, ddaps) {
		return parse_type_constructor_stmt(p, ddaps)
	}

	// Feature example:
	//   @co.dap.extension(fortype=co.lang.string, what=extends)
	//   upperCase()->(string) = { this.return this.upper(); }
	//
	// Extensions are parsed before the generic function/statement fallback
	// because they share the outer function declaration shape but carry
	// declaration-level metadata that changes the AST node kind.
	if _, ok := isAnnDec(ddaps, "@co.dap.extension"); ok {
		return parse_extension_declaration(p, ddaps)
	}

	// Feature example:
	//   @co.ddap.use(extensions=[equals, upperCase])
	//
	// This is a directive-shaped statement that does not start with a normal
	// declaration keyword, so we dispatch it before the type/function recognizers.
	if _, ok := isAnnDec(ddaps, "@co.dap.use"); ok {
		return parse_use_directive_stmt(p, ddaps)
	}

	if isStructDecl(p, -1) {
		return parse_type_decl_stmt(p, STRUCT, "type", ddaps)
	} else if isUnionDecl(p, -1) {
		return parse_type_decl_stmt(p, UNION, "type", ddaps)
	} else if isClassDecl(p, -1) {
		return parse_class_declaration_stmt(p, CLASS, false, ddaps)
	} else if isDependentTypeDecl(p, -1) {
		return parse_dependent_type_decl(p, ddaps)
	} else if isTypeDecl(p, -1, ddaps) {
		return parse_type_decl_stmt(p, TYPE, "type", ddaps)
	} else if isModuleDecl(p, -1, ddaps) {
		return parse_module_stmt(p, MODULE, false, ddaps)
	} else if isEnumDecl(p, -1) {
		return parse_enum_declaration_stmt(p, NASTMT, ddaps)
	} else if isSigDecl(p, -1, ddaps) {
		return parse_signature_declaration_stmt(p, ddaps)
	} else if isInterfaceDecl(p, -1) {
		return parse_interface_declaration_stmt(p, ddaps)
	} else if isInstanceDecl(p, -1) {
		return parse_instance_declaration(p, ddaps)
	} else if isMatcherInstanceDecl(p, -1) {
		return parse_matcher_instance_declaration(p, ddaps)
	} else if isObjectDecl(p, -1) {
		//"@co.dap.annotation",  "@co.dap.pragma", "@co.dap.directive"
		return parse_object_declaration_stmt(p, ddaps)
	} else if isKindDefn(p, -1, ddaps) {
		return parse_kind_decl_stmt(p, KIND)
	} else if isKindDecl(p, -1, ddaps) {
		return parse_type_decl_stmt(p, KIND, "kind", ddaps)
	} else if isFunctionObject(p, -1, ddaps) {
		return parse_function_object_declaration_stmt(p, ddaps)
	} else if isClosureDecl(p) {
		return parse_closure_or_curry_stmt(p, true, ddaps)
	} else if isCurryDecl(p) {
		return parse_closure_or_curry_stmt(p, false, ddaps)
	} else if isFuncPattern(p, -1, ddaps) {
		return parse_func_pattern_stmt(p, ddaps)
	} else if isFuncDecl(p, -1, ddaps) {
		return callFunction(p, -1, ddaps)
	} else {

		tr := parse_stmt(p, CODE, ddaps)

		if _, ok := tr.Node.(ast.SymbolDeclStmt); ok {
			p.IsPrevDeclSymbol = true
		} else {
			p.IsPrevDeclSymbol = false
		}
		return tr
	}

}

// parse_interface_declaration_stmt parses an interface declaration:
//
//	IEmployee co.lang.interface = {
//	    storeEmployee(emp Employee)->(Employee);
//	}
//
// Entry: current token is the interface name IDENTIFIER; next token is co.lang.interface.
func parse_interface_declaration_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	p.advance() // skip co.lang.interface

	currCtxId := p.Context_
	currSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(currCtxId.Id, string(symboltable.S_InterfaceSymbol))
	p.Context_.ContextType_ = string(symboltable.S_InterfaceSymbol)
	currCtxId.ChildCtxIds = append(currCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = currSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)
	_, err_ = p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	body := make([]ast.Stmt, 0)
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_CURLY && p.currentTokenKind() != scanlex.EOF {
		if p.currentTokenKind() == scanlex.IDENTIFIER {
			sig := parseSig(p, ddaps)
			body = append(body, sig)
		} else {
			p.advance()
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)

	symb := symboltable.TypeSymbol{
		UDT:    true,
		AsExpr: true,
		SymbolDetails: symboltable.SymbolDetails{Name_: name.Value,
			SymbolType_: string(symboltable.S_TypeSymbol),
		},
	}

	typde := ast.TypeDeclarationStmt{
		Name:     name.Value,
		Body:     body,
		SubType_: "INTERFACE",
		Symb:     &symb,
	}
	pr.Node = typde

	p.Context_ = currCtxId
	p.SymbolTable_ = currSymbId

	updateContext(p, pr.Node, false, false)
	return pr
}

// parse_signature_declaration_stmt parses a signature declaration:
//
//	SEmployee co.lang.signature = {
//
//	    storeEmployee(emp Employee)->(Employee);
//	}
//
// A signature is a module type: it can contain both type declarations and
// method signatures. Entry: current token is the signature name IDENTIFIER;
// next token is co.lang.signature.
func parse_signature_declaration_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	p.advance() // skip co.lang.signature

	currCtxId := p.Context_
	currSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(currCtxId.Id, string(symboltable.S_SignatureSymbol))
	p.Context_.ContextType_ = string(symboltable.S_SignatureSymbol)
	currCtxId.ChildCtxIds = append(currCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = currSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)
	_, err_ = p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	body := make([]ast.Stmt, 0)
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_CURLY && p.currentTokenKind() != scanlex.EOF && !p.errorLimitReached() {
		prev := p.currentToken()
		// Signature body may contain type declarations (e.g. `Employee co.lang.struct;`)
		// or method signatures (e.g. `save(emp Employee)->(Employee);`).
		tr := parse_type_kind_block(p, SIGNATURE, ddaps)
		if tr.Node != nil {
			body = append(body, tr.Node.(ast.Stmt))
		} else if !p.madeProgress(prev) {
			p.advance()
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)
	symb := symboltable.TypeSymbol{
		SymbolDetails: symboltable.SymbolDetails{
			Name_:       name.Value,
			SymbolType_: string(symboltable.S_TypeSymbol),
		},
		UDT:    true,
		AsExpr: true,
	}

	typde := ast.TypeDeclarationStmt{
		Name:     name.Value,
		Body:     body,
		SubType_: "SIGNATURE",
		Symb:     &symb,
	}
	pr.Node = typde

	p.Context_ = currCtxId
	p.SymbolTable_ = currSymbId

	updateContext(p, pr.Node, false, false)
	return pr
}
