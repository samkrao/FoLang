package parser

import (
	"fmt"
	"slices"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Varsubtype classifies the storage subtype of a variable declaration.
type Varsubtype int

const (
	// Normal represents a plain variable with no special storage qualifier.
	Normal Varsubtype = iota
	// Pointer represents a pointer variable (co.lang.ptr).
	Pointer
	// Reference represents a reference variable (co.lang.ref or co.lang.dbl.ref).
	Reference
	// Address represents an address-of variable (co.lang.addr).
	Address
	// Array represents an array variable (co.lang.arr and variants).
	Array
	// Thunk represents a lazily evaluated thunk variable (co.lang.thunk).
	Thunk
	// Slice represents a slice variable (co.lang.slice).
	Slice
	// HeapAllocRef represents a heap-allocated reference variable (co.lang.heapallocref).
	HeapAllocRef
	// Range represents a range variable (co.lang.range).
	Range
)

// parse_prg_stmt parses the top-level FoLang compilation unit.
//
// Feature examples:
//
//	@co.dap.library(type="application")
//
//
//	// application entry / code file
//	name := "Rao";
//	co.out.println(name);
//
// The top-level parser first collects file-scope hints, then decides whether
// the file is:
// 1. a library surface file
// 2. a regular code/application file
func parse_prg_stmt(p *parser, name string, packagePath string) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors:    nil,
		Node:      nil,
		BuildLibs: false,
	}
	prgStmts := make([]ast.Stmt, 0)
	var result ast.Stmt
	buildLib := false
	tddaps := map[scanlex.DirectiveKind][]ast.Stmt{}

	// File-scope pragmas are collected first because they can affect how the
	// rest of the file is interpreted.
	pragmaDap := collect_hints(p, scanlex.PRAGMA, tddaps)
	ddaps := map[scanlex.DirectiveKind][]ast.Stmt{
		scanlex.PRAGMA: pragmaDap,
	}

	savedPos := p.pos
	dirDap := collect_hints(p, scanlex.DIRECTIVE, tddaps)
	ddaps[scanlex.DIRECTIVE] = dirDap

	annoDap := collect_hints(p, scanlex.ANNOTATION, tddaps)
	ddaps[scanlex.ANNOTATION] = annoDap

	if existsPackageFile(p) {
		parse_package_file_stmt(p, PACKAGE, ddaps)
	}

	if isLibrary(p, ddaps) && isPackageSource(p, ddaps) { //source library package file
		p.pos = savedPos
		tr := parse_co_stmts(p, SOURCE_LIBRARY, ddaps)
		prgStmts = append(prgStmts, tr.Node.(ast.Stmt))
		result = ast.Prog{Body: prgStmts, Name: name, Package: packagePath, CodeBlock: false}
		result.SetDap(ddaps)
		assertProgramEOF(p)
		res.Node = result
		return res
	} else if isPackageSource(p, ddaps) { //package source file
		p.pos = savedPos
		tr := parse_co_stmts(p, PACKAGE, ddaps)
		prgStmts = append(prgStmts, tr.Node.(ast.Stmt))
		result = ast.Prog{Body: prgStmts, Name: name, Package: packagePath, CodeBlock: true}
		result.SetDap(ddaps)
		assertProgramEOF(p)
		res.Node = result
		return res
	} else if !isLibrary(p, ddaps) { // application entry file
		p.pos = savedPos
		delete(ddaps, scanlex.ANNOTATION)
		tr := parse_co_stmts(p, CODE, ddaps)
		prgStmts = append(prgStmts, tr.Node.(ast.Stmt))
		result = ast.Prog{Body: prgStmts, Name: name, Package: packagePath, CodeBlock: true}
		result.SetDap(ddaps)
		assertProgramEOF(p)
		res.Node = result
		return res
	} else { // Library surface file
		tr := parse_co_stmts(p, LIBRARY, ddaps)
		var pkg_name string
		if lib, ok := tr.Node.(ast.Library); ok {
			pkg_name = name
			lib.Symb.Name = name
			lib.Symb.Name_ = name
			lib.Symb.SymbolType_ = string(symboltable.S_Program)
			tr.Node = lib
		}

		prgStmts = append(prgStmts, tr.Node.(ast.Stmt))
		prst := ast.Prog{Body: prgStmts, Name: name, Package: pkg_name, CodeBlock: false}
		prst.SetDap(ddaps)
		result = prst
		assertProgramEOF(p)
		buildLib = true
	}

	res.Node = result
	res.BuildLibs = buildLib
	return res
}

// assertProgramEOF records a parse error when the top-level parser has not
// consumed the full file.
func assertProgramEOF(p *parser) {
	if p.currentTokenKind() != scanlex.EOF {
		err_ := p.errorObj_frm_st([]string{"EOF"}, "End of File")
		p.addErr(err_)
	}
}

// parse_co_stmts selects the top-level parse mode for the current compilation
// unit.
//
// Feature examples:
//
//	library surface file   -> LIBRARY mode
//	application/code file  -> CODE mode
func parse_co_stmts(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}

	parse_import_declaration_stmt(p, ddaps)

	if p.hasTokens() && p.currentTokenKind() != scanlex.EOF {

		val := false
		if p.IsFunction {
			val = p.ParentIsFunc
			p.ParentIsFunc = true
		}
		if stmtK == SOURCE_LIBRARY {
			// source library package file
		} else if stmtK == LIBRARY {

			pr := parse_library_stmt(p, stmtK, ddaps)
			pkgSt, ok := pr.Node.(ast.Library)
			isLib := false
			if ok {
				if pkgSt.Symb.Type == "library" {
					isLib = true
				}
			}
			if !isLib {
				err_ := p.errorExpection("Folang source must be either entry file / library package", helpers.InvalidSyntax)
				foerrors.HandleErrors(err_)

			}
			pr.Node.(ast.Stmt).SetDap(ddaps)
			return pr

		} else if stmtK == CODE {
			ddaps = map[scanlex.DirectiveKind][]ast.Stmt{}
			ddaps[scanlex.DIRECTIVE] = collect_hints(p, scanlex.DIRECTIVE, ddaps)
			return parse_code_stmt(p, CODE, ddaps)
		} else if stmtK == PACKAGE {
			ddaps = map[scanlex.DirectiveKind][]ast.Stmt{}
			ddaps[scanlex.DIRECTIVE] = collect_hints(p, scanlex.DIRECTIVE, ddaps)
			return parse_package_stmt(p, PACKAGE, ddaps)
		} else {
			err_ := p.errorExpection("Invalid statement kind", helpers.InvalidSyntax)
			p.addErr(err_)
			pr.Errors = append(pr.Errors, err_)

		}

		if p.IsFunction {
			p.ParentIsFunc = val
		}
	}
	return pr

}

/*
	****** Package file parsing  *******
	// Name of the file package.fol in a give folder
	// packagename co.lang.package;

	// result:
	// if the folder is empl
	// appl/hr/empl where appl contains entry file
	// then hr is a package and empl is a package by default
	// suppose empl folder contains package.fol
	// with content `emp co.lang.package;`
	// then instead of empl as package emp will be the package name
	// the package.fol overrides the name of package from folder name to name specified in the file.
*/

func parse_package_file_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {

	pr := ParseResult{}

	return pr

}

/*
		****** Package source fileParsing *******

		// parse_package_stmt parses a sequence of top-level executable or declaration
		// statements in PACKAGE mode.

	 	// Top level statements in a package source file can be:
		// class
		// struct
		// unit
		// enum
		// union
		// interface
		// signature
		// module
		// cstruct
		// instance
		// object
		// matcher
		// template
		// annotations
		// decorators
		// macros
		// @co.dap.typeclass annotations
		// @co.dap.matcher annotations
		// @co.dap.decorator and @co.dap.annotation annotations
*/
func parse_package_src_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()
	pr := ParseResult{}
	ddaps = map[scanlex.DirectiveKind][]ast.Stmt{}
	ddaps[scanlex.DIRECTIVE] = collect_hints(p, scanlex.DIRECTIVE, ddaps)
	name := p.currentToken().Value
	if p.currentToken().Value == "_" {
		name = p.fileInfo.Filename
	}
	fmt.Println(name)
	if p.nextToken(1).Kind == scanlex.BUILT_IN_KIND {
		if p.nextToken(1).Value == "co.lang.package" {
		} else if p.nextToken(1).Value == "co.lang.module" {
			return parse_module_stmt(p, stmtK, false, ddaps)
		} else if p.nextToken(1).Value == "co.lang.block" {
			return parse_block_stmt(p, ddaps)
		} else if p.nextToken(1).Value == "co.lang.delegate" {
			return parse_delegate_stmt(p, ddaps)
		} else if p.nextToken(1).Value == "co.lang.type" {

		} else if p.nextToken(1).Value == "co.lang.struct" {
			return parse_type_kind_block(p, PACKAGE, ddaps)
		} else if p.nextToken(1).Value == "co.lang.class" {
			return parse_type_kind_block(p, PACKAGE, ddaps)
		} else if p.nextToken(1).Value == "co.lang.interface" {
			return parse_type_kind_block(p, PACKAGE, ddaps)
		} else if p.nextToken(1).Value == "co.lang.enum" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.union" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.instance" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.object" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.cstruct" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.newtype" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.opaquetype" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.subtype" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.supertype" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.dependenttype" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.Matcher" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.template" || isTemplateDecl(p, -1, ddaps) {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.macro" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.signature" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if p.nextToken(1).Value == "co.lang.unit" {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if isTypeClass(p, -1, ddaps) {
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else if isLibrary(p, ddaps) { //soruce libraries
			return parse_type_kind_block(p, PACKAGE, ddaps)

		} else {
			err_ := p.errorExpection("Invalid package declaration", helpers.InvalidSyntax)
			p.addErr(err_)
			pr.Errors = append(pr.Errors, err_)
		}

	}

	return pr
}

/*   ****** Entry File or Application File Parsing *******
// parse_code_stmt parses a sequence of top-level executable or declaration
// statements in CODE mode.

// built-in directives annotations, pragmas and decorators that are valid for an entry file
// package and library imports
// import aliases declared with as=
// file-local aliases for co.* paths declared with @co.ddap.alias
// type aliases declared with co.lang.type
// new types declared with co.lang.newtype
// opaque types declared with co.lang.opaquetype
// dependent-type aliases and dependent-type usages that do not declare a type-constructor function
// dependent type declared with co.lang.dependenttype
// subtype declarations
// supertype declarations
// non-capturing entry-local function-pattern groups
// capturing entry-local let function-pattern groups
// variable declarations, initialization, assignment, and mutation
// calls to built-in methods and functions
// calls to imported package and library APIs
// ordinary expressions
// comprehensions
// ternary expressions
// conditions and loops
// containment operations such as contains
// iterators such as each
// pattern matching and destructuring
*/

// Feature examples:
//
//	name := "Rao";
//	age  co.lang.int = 30;
//	co.out.println(name);
func parse_code_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	// Validate @co.ddap.alias directives collected at the top of the file.
	for _, err := range validate_alias_directives(p, ddaps) {
		p.addErr(err)
	}

	var pr ParseResult
	body := []ast.Stmt{}
	p.IsPrevDeclSymbol = false
	dddaps := map[scanlex.DirectiveKind][]ast.Stmt{}
	dddaps[scanlex.ANNOTATION] = collect_hints(p, scanlex.ANNOTATION, ddaps)
	for p.hasTokens() && p.currentTokenKind() != scanlex.EOF && !p.errorLimitReached() {

		pr = parse_entry_stmt(p, CODE, dddaps)

		if !appendParsedStatement(p, pr, &body, dddaps) {
			if p.errorLimitReached() {
				break
			}
			continue
		}
		dddaps[scanlex.ANNOTATION] = collect_hints(p, scanlex.ANNOTATION, ddaps)
	}

	cst := ast.Application{
		Body: body,
	}
	cst.SetDap(ddaps)
	pr.Node = cst

	return pr
}

// the function which actually parses entry file statements

func parse_entry_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()
	defer func() {
		p.Entry = false
		p.AllLets = true

	}()

	var pr ParseResult
	ok, _ := checkSpecialMethods(p, ddaps)
	stmtKind, ok1 := stmt_to_stmtKind[p.currentToken().Value]
	if (p.currentTokenKind() == scanlex.KEYWORD || p.currentTokenKind() == scanlex.RESERVEDWORD) && ok1 {
		if p.currentToken().Value == "let" || p.currentToken().Value == "for" {

		} else {
			err_ := p.errorObj(nil, fmt.Sprintf("Invalid keword %s\n", scanlex.TokenKindString(p.currentTokenKind())))
			p.addErr(err_)
			return pr
		}
		p.AllLets = false
		stmt_fn, exists := stmt_lu[stmtKind]
		if !exists {
			err_ := p.errorObj(nil, fmt.Sprintf("Invalid token %s\n", scanlex.TokenKindString(p.currentTokenKind())))
			p.addErr(err_)
		}

		pr := stmt_fn(p, ddaps)
		return pr

	} else if (p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER) && ok {
		// Feature examples:
		//   nums.each(i).do({ ... })
		//   nums.contains(10).do({ ... })
		//
		// Special built-in statement expressions start with a normal identifier
		// token, but their method names put them on a statement-only path.
		return parse_built_in_stmt_expr(p, ddaps)
	} else if p.currentTokenKind() == scanlex.BUIL_IN_STMT_EXPRS {
		return parse_built_in_stmt_expr(p, ddaps)
	} else if p.currentTokenKind() == scanlex.IDENTIFIER {
		p.Entry = true
		if isLabel(p, -1) {
			return parse_label_stmt(p, ddaps)
		} else if p.nextTokenSafe(1).Value == "co.lang.auto" {
			return parse_let_decl_stmt(p, ddaps)

		} else if p.nextToken(1).Value == "co.lang.dynamic" {
			return parse_auto_decl_stmt(p, ddaps)
		} else if p.nextToken(1).Kind == scanlex.QEQ {
			if isVarDefn(p, -1, ddaps) {
				return parse_redecl_decl_stmt(p, ddaps)
			} else if isRangeDecl(p, -1, ddaps) {
				return parse_range_decl_stmt(p, ddaps, true)
			}

		} else if p.nextToken(1).Kind == scanlex.WALRUS {
			// Feature examples:
			//   x := co.const.true
			//   add := (a co.lang.int, b co.lang.int)->(co.lang.int) = { ... } not allowed
			//
			// Walrus declarations infer their static type from the right-hand side.
			if isVarDefn(p, -1, ddaps) {
				return parse_walrus_decl_stmt(p, ddaps)
			} else if isRangeDecl(p, -1, ddaps) {
				return parse_range_decl_stmt(p, ddaps, false)
			}
			//return parse_fo_kindtype_decl(p, ddaps, VARFIELDMEMBERPROP)
		}
		return parse_var_decl_stmt(p, ddaps)
	} else {
		pr = parse_expression_stmt(p, ddaps)
	}
	return pr

}

// appendParsedStatement centralizes statement append and recovery behavior for
// code blocks and top-level CODE mode.
func appendParsedStatement(p *parser, pr ParseResult, body *[]ast.Stmt, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	if pr.Node == nil {
		err_ := p.errorExpection("Parse error: unable to parse statement", helpers.InvalidSyntax)
		p.addErr(err_)
		p.sync(scanlex.SEMI_COLON, scanlex.CLOSE_CURLY)
		if p.currentTokenKind() == scanlex.SEMI_COLON {
			p.advance()
		}
		return false
	}

	stmtNode := pr.Node.(ast.Stmt)
	stmtNode.SetDap(ddaps)
	_, p.IsPrevDeclSymbol = pr.Node.(ast.SymbolDeclStmt)
	*body = append(*body, stmtNode)
	return true
}

// checkSpecialMethods detects built-in object-style statement expressions such
// as collection iteration and containment checks.
//
// Feature examples:
//
//	nums.each(i).do({ ... })
//	nums.contains(10).do({ ... })
func checkSpecialMethods(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (bool, int) {
	defer p.traceCurrent()()

	if p.nextTokenSafe(1).Kind == scanlex.DOT && slices.Contains(ISIn, p.nextToken(2).Value) {
		return true, 1
	} else if p.nextTokenSafe(1).Kind == scanlex.DOT && slices.Contains(IteratorMethods, p.nextToken(2).Value) {
		return true, 2
	}
	return false, 0
}

func unused_parse_package_src_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	var pr ParseResult
	stmtKind, ok1 := stmt_to_stmtKind[p.currentToken().Value]
	if (p.currentTokenKind() == scanlex.KEYWORD || p.currentTokenKind() == scanlex.RESERVEDWORD) && ok1 {
		// Feature examples:
		//   let x := 10;
		//   forall(T).(T)->(T)
		//
		// Keyword-led statements are routed through the small statement lookup
		// table built in lookups.go.
		stmt_fn, exists := stmt_lu[stmtKind]
		if !exists {
			err_ := p.errorObj(nil, fmt.Sprintf("Invalid token %s\n", scanlex.TokenKindString(p.currentTokenKind())))
			p.addErr(err_)
		}

		return stmt_fn(p, ddaps)

	} else if isDelegateDecl(p, -1, ddaps) {
		// Feature example:
		//   @co.dap.delegate
		//   someDelegate co.lang.delegate = (a co.lang.int)->(co.lang.int);
		pr := parse_delegate_stmt(p, ddaps)
		return pr
	}
	return pr

}

// parse_stmt is the main statement dispatcher for CODE mode.
//
// Feature examples:
//
//	let x := 10;
//	value.match(...);
//	someDelegate co.lang.delegate = (a co.lang.int)->(co.lang.int);
//	label: { ... }
//	blockName co.lang.block = { ... }
//
// The dispatcher keeps the existing codebase style:
// 1. check keyword-led statement forms
// 2. check built-in statement expressions
// 3. check declaration-shaped special cases
// 4. fall back to expression statement parsing
func parse_stmt(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	var pr ParseResult
	ok, _ := checkSpecialMethods(p, ddaps)
	stmtKind, ok1 := stmt_to_stmtKind[p.currentToken().Value]
	if (p.currentTokenKind() == scanlex.KEYWORD || p.currentTokenKind() == scanlex.RESERVEDWORD) && ok1 {
		// Feature examples:
		//   let x := 10;
		//   forall(T).(T)->(T)
		//
		// Keyword-led statements are routed through the small statement lookup
		// table built in lookups.go.
		stmt_fn, exists := stmt_lu[stmtKind]
		if !exists {
			err_ := p.errorObj(nil, fmt.Sprintf("Invalid token %s\n", scanlex.TokenKindString(p.currentTokenKind())))
			p.addErr(err_)
		}

		return stmt_fn(p, ddaps)

	} else if (p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER) && ok {
		// Feature examples:
		//   nums.each(i).do({ ... })
		//   nums.contains(10).do({ ... })
		//
		// Special built-in statement expressions start with a normal identifier
		// token, but their method names put them on a statement-only path.
		return parse_built_in_stmt_expr(p, ddaps)
	} else if p.currentTokenKind() == scanlex.BUIL_IN_STMT_EXPRS {
		return parse_built_in_stmt_expr(p, ddaps)
	} else if isDelegateDecl(p, -1, ddaps) {
		// Feature example:
		//   @co.dap.delegate
		//   someDelegate co.lang.delegate = (a co.lang.int)->(co.lang.int);
		pr := parse_delegate_stmt(p, ddaps)
		return pr
	} else if p.currentTokenKind() == scanlex.IDENTIFIER {
		if isLabel(p, -1) {
			return parse_label_stmt(p, ddaps)
		} else if isNamedBlockStmt(p, -1) {
			return parse_named_block_stmt(p, ddaps)
		} else if isTypeConstructorDecl(p, -1, ddaps) {
			return parse_type_constructor_stmt(p, ddaps)
		} else if p.nextTokenSafe(1).Value == "co.lang.auto" {
			return parse_let_decl_stmt(p, ddaps)

		} else if p.nextToken(1).Value == "co.lang.dynamic" {
			return parse_auto_decl_stmt(p, ddaps)
		} else if p.nextToken(1).Kind == scanlex.QEQ {
			if isVarDefn(p, -1, ddaps) {
				return parse_redecl_decl_stmt(p, ddaps)
			} else if isRangeDecl(p, -1, ddaps) {
				return parse_range_decl_stmt(p, ddaps, true)
			}

		} else if p.nextToken(1).Kind == scanlex.WALRUS {
			// Feature examples:
			//   x := co.const.true
			//   add := (a co.lang.int, b co.lang.int)->(co.lang.int) = { ... }
			//
			// Walrus declarations infer their static type from the right-hand side.
			if isVarDefn(p, -1, ddaps) {
				return parse_walrus_decl_stmt(p, ddaps)
			} else if isRangeDecl(p, -1, ddaps) {
				return parse_range_decl_stmt(p, ddaps, false)
			}
			//return parse_fo_kindtype_decl(p, ddaps, VARFIELDMEMBERPROP)
		}
		return parse_var_decl_stmt(p, ddaps)
	} else {
		pr = parse_expression_stmt(p, ddaps)
	}
	return pr
}

func fetchDetails(p *parser, curToken scanlex.Token, ddaps map[scanlex.DirectiveKind][]ast.Stmt) symboltable.SymbolInfo {
	defer p.traceCurrent()()

	symbTab := *p.SymbolTable_
	v := curToken.Value
	if symbTab.ExistsVar(*p.Fs, v) && validScopeForExists(p) {
		t := symbTab.GetVarDetails(*p.Fs, v)
		return t
	} else {
		err_ := p.errorObj(nil, "Variable not declared")
		p.addErr(err_)

	}

	return nil
}

func parse_expression_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	expression := parse_expr(p, defalt_bp, ddaps)
	if !p.SKIP_SEMI_COMMA {
		_, err_ := p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
		p.addErr(err_)
	}
	if _, ok := expression.Node.(ast.StatementExpr); ok {
		pr.Node = expression.Node.(ast.StatementExpr).Statement
		return pr
	}
	exprst := ast.ExpressionStmt{
		Expression: expression.Node.(ast.Expr),
	}
	pr.Node = exprst

	return pr
}

func parse_var_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := parse_decl_stmt(p, "var", false, true, false, VAR, "", false, ddaps)
	// Comma-separated declarations: x co.lang.int = 10, y co.lang.string;
	// parse_decl_stmt consumes the trailing SEMI_COLON or COMMA.
	// If it consumed a COMMA, parse the next declaration and chain via CommaExpr.
	for p.previousToken(1).Kind == scanlex.COMMA && p.hasTokens() {
		right := parse_decl_stmt(p, "var", false, true, false, VAR, "", false, ddaps)
		leftExpr, lok := pr.Node.(ast.Expr)
		rightExpr, rok := right.Node.(ast.Expr)
		if !lok {
			leftExpr = ast.StatementExpr{Statement: pr.Node.(ast.Stmt)}
		}
		if !rok {
			rightExpr = ast.StatementExpr{Statement: right.Node.(ast.Stmt)}
		}
		pr.Node = ast.CommaExpr{
			Left:  leftExpr,
			Right: rightExpr,
			Symb: &symboltable.ExpressionSymbol{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_ExpressionSymbol),
				},
			},
		}
	}
	return pr
}
func parse_auto_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_decl_stmt(p, "auto", false, true, false, VAR, "", false, ddaps)
}
func parse_redecl_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_walrus_decl_impl(p, ddaps, true)
}

func parse_walrus_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_walrus_decl_impl(p, ddaps, false)
}
func parse_range_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt, redeclare bool) ParseResult {
	defer p.traceCurrent()()

	return parse_range_decl_impl(p, ddaps, redeclare)
}
func parse_let_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_decl_stmt(p, "let", false, true, false, VAR, "", false, ddaps)
}
func parse_adhoc_var_decl(p *parser, varName string, type_ string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	var keyVal map[string]string = map[string]string{
		"Varname": varName,
		"Type":    type_,
	}
	return parse_decl_stmt(p, "adhoc", false, false, false, VAR, keyVal, false, ddaps)
}

func parse_fo_var_decl_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_decl_stmt(p, "fovar", false, true, false, VAR, "", false, ddaps)
}

func parse_let_binding(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	p.advance() //eat let
	// Check for let function pattern: `let fib(n) = expr`
	if isLetFuncPattern(p) {
		return parse_let_func_pattern_stmt(p, ddaps)
	}
	pr := ParseResult{}
	if p.AllLets {
		tLet := p.Let
		p.Let = true
		pr = parse_expr(p, defalt_bp, ddaps)
		p.Let = tLet
	}
	return pr
}
