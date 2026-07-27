package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// parse_library_stmt parses a library surface file.
//
// A library surface file is annotated with @co.dap.library and contains
// top-level declarations (structs, interfaces, functions, enums, type aliases,
// etc.) without wrapping curly braces:
//
//	 @co.dap.library(type="application")
//	 lbiname co.lang.library={
//			// structs
//			// cstructs
//			// functions
//	 }

func parse_library_stmt(p *parser, stmtK StmtKind, daps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	// Merge pre-collected hints from parse_prg_stmt (which already consumed
	// @co.dap.library from the token stream) with any remaining stream hints.
	// parse_prg_stmt passes the annotation list in daps[ANNOTATION] so we don't
	// need to re-collect it; we only collect additional PRAGMA/DIRECTIVE hints
	// that might follow.
	ldaps := map[scanlex.DirectiveKind][]ast.Stmt{}
	ldaps[scanlex.PRAGMA] = append(daps[scanlex.PRAGMA], collect_hints(p, scanlex.PRAGMA, ldaps)...)
	ldaps[scanlex.ANNOTATION] = append(daps[scanlex.ANNOTATION], collect_hints(p, scanlex.ANNOTATION, ldaps)...)
	ldaps[scanlex.DIRECTIVE] = append(daps[scanlex.DIRECTIVE], collect_hints(p, scanlex.DIRECTIVE, ldaps)...)

	// Extract the library kind from @co.dap.library(type="...").
	libKind := ""
	for _, stmt := range ldaps[scanlex.ANNOTATION] {
		dir, ok := stmt.(ast.DirectiveStmt)
		if !ok || dir.Name != "@co.dap.library" {
			continue
		}
		if t, ok2 := dir.Parameters["type"].([]any); ok2 && len(t) > 0 {
			libKind = strings.TrimSuffix(strings.Trim(fmt.Sprint(t[0]), "\""), "_fo")
		}
	}

	validKinds := []string{"ffi", "system", "advanced", "application", "dynamicvmrt"}
	if libKind != "" && !slices.Contains(validKinds, libKind) {
		err_ := p.errorExpection(
			"invalid library kind '"+libKind+"'; expected ffi, system, advanced, dynamicvmrt or application",
			helpers.InvalidSyntax,
		)
		p.addErr(err_)
	}

	actCtxId := p.Context_
	actSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_PackageSymbol))
	actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ContextType_ = string(symboltable.S_PackageSymbol)
	p.Context_.ParentCtxSymbolTableId = actSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	// Parse body declarations until EOF — same pattern as parse_pkg_stmt body loop.
	body := make([]ast.Stmt, 0)
	bdaps := map[scanlex.DirectiveKind][]ast.Stmt{
		scanlex.ANNOTATION: collect_hints(p, scanlex.ANNOTATION, ldaps),
		scanlex.DECORATOR:  collect_hints(p, scanlex.DECORATOR, ldaps),
	}
	for p.hasTokens() && p.currentTokenKind() != scanlex.EOF && !p.errorLimitReached() {
		prev := p.currentToken()
		tr := parse_type_kind_block(p, LIBRARY, bdaps)
		if tr.Node != nil {
			tr.Node.(ast.Stmt).SetDap(bdaps)
			body = append(body, tr.Node.(ast.Stmt))
		} else if !p.madeProgress(prev) {
			p.advance()
		}
		bdaps = map[scanlex.DirectiveKind][]ast.Stmt{
			scanlex.ANNOTATION: collect_hints(p, scanlex.ANNOTATION, ldaps),
			scanlex.DECORATOR:  collect_hints(p, scanlex.DECORATOR, ldaps),
		}
	}

	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	// Name is left empty here; parse_prg_stmt fills it in from the file's
	// basename so that a library "hrlib.fol" gets Package/Name = "hrlib".
	symb := symboltable.LibrarySymbol{
		Name: "",
		Type: "library",
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_PackageSymbol),
		},
	}

	lib := ast.Library{
		Body: body,
		Symb: &symb,
	}
	lib.SetDap(ldaps)
	pr.Node = lib
	return pr
}
