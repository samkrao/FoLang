package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

/*
*
* parse_unit_declaration_stmt parses a co.lang.unit declaration:

*	Math co.lang.unit = {
*	}
*  or
*  _ co.lang.unit ={
*  }
*
*  Valid unit statments
*   Top level
*   annotations, decorators pragmas, directives
*   in co.lang.unit block
*	*   functions
*	*   @co.dap.extension
*	*   co.lang.type
*	*   co.lang.newtype
*	*   co.lang.dependenttype
*	*   co.lang.opaquetype
*	*   co.lang.subtype
*	*   co.lang.supertype
*   *   function types with co.lang.function.
*	*   type constructors @co.dap.hokrtl, @co.dap.hokrt
*	*   associated functions (static, class, inntance)
*	*   closures
*	*   curried
*   *   indexers @co.dap.indexer
*   *   conntinuations (@co.dap.continuation), coroutines(@co.dap.coroutinne), tasks(@co.dap.task), thread (@co.dap.thread), process( @co.dap.process), fibler(@co.dap.fiber), channel (@co.dap.channel), async (@co.dap.async), events (@co.dap.event) etc
*	*   @co.dap.delegate delegates
*
 */
func parse_unit_declaration_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	return pr
}
