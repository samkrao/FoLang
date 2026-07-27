package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_annotation_declaration parses a function-like declaration that is
// tagged as an annotation, directive, pragma, or decorator declaration.
//
// Feature example:
//
//	myAnnotation co.lang.object->(for=annotation) = {
//	    value co.lang.string;
//	}
//
// The underlying grammar is function-shaped, so this helper reuses normal
// function declaration parsing and then records the requested declaration kind.
func parse_annotation_declaration(p *parser, kind string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := parse_fn_declaration(p, FUNCTION, ddaps)
	if pr.Node == nil {
		return pr
	}
	n := pr.Node.(ast.FunctionDeclarationStmt)
	n.WhatisIt = []string{kind}
	pr.Node = n
	return pr
}
