package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_block_stmt parses a standard curly-brace statement block.
//
// Feature examples:
//
//	{
//	    x := 10;
//	    this.return x;
//	}
func parse_block_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_block_stmt_generic(p, false, true, ddaps)
}

// parse_block_stmt_generic parses either a normal `{ ... }` block or a grouped
// statement body used in function/conditional helper paths.
//
// Feature examples:
//
//	{
//	    x := 10;
//	}
//
//	.do({
//	    this.return 1;
//	})
//
// When `p.IsFunction` is true, statements are parsed through parse_fun_stmt so
// return-aware function-body rules stay active inside nested blocks.
func parse_block_stmt_generic(p *parser, funHasResults bool, checkCurly bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	if checkCurly {
		_, err_ := p.expect(scanlex.OPEN_CURLY)
		p.addErr(err_)
	}
	kind := scanlex.CLOSE_CURLY
	if !checkCurly {
		kind = scanlex.CLOSE_PAREN
	}
	body := []ast.Stmt{}
	isFun := p.IsFunction
	if isFun {
		// Feature example:
		//   fun add(a co.lang.int)->(co.lang.int) = {
		//       this = a + 1;
		//   }
		//
		// Function blocks route through parse_fun_stmt so `this`, `this.return`,
		// and trailing-return rules are handled consistently.
		for p.hasTokens() && p.currentTokenKind() != kind {
			tr := parse_fun_stmt(p, funHasResults, true, ddaps)

			body = append(body, tr.Node.(ast.Stmt))
		}
		return pr

	} else {

		// Feature example:
		//   {
		//       x := 10;
		//       y = x + 1;
		//   }
		for p.hasTokens() && p.currentTokenKind() != kind {

			tr := parse_stmt(p, CODE, ddaps)
			body = append(body, tr.Node.(ast.Stmt))
		}
	}
	if checkCurly {
		_, err_ := p.expect(scanlex.CLOSE_CURLY)
		p.addErr(err_)
	}
	pr.Node = ast.BlockStmt{
		Body: body,
	}
	return pr
}

// parse_label_stmt parses a label followed by a block:
//
//	outer: {
//	    // statements
//	}
//
// The identifier is the current token; the colon and block follow.
func parse_label_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Node: nil, Errors: []helpers.ErrorInterface{}}

	labelTok, err_ := p.expectError(scanlex.IDENTIFIER, "expected label name")
	p.addErr(err_)

	_, err_ = p.expect(scanlex.COLON)
	p.addErr(err_)

	inner := parse_block_stmt(p, ddaps)
	blk := inner.Node.(ast.BlockStmt)
	blk.Symb.Name_ = labelTok.Value
	blk.Symb.IsNamed = true

	pr.Node = blk
	return pr
}

// parse_named_block_stmt parses a named block variable:
//
//	labelBlock co.lang.block = {
//	    // statements
//	}
//
// The identifier is the current token; co.lang.block and the block follow.
func parse_named_block_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Node: nil, Errors: []helpers.ErrorInterface{}}

	nameTok, err_ := p.expectError(scanlex.IDENTIFIER, "expected block name")
	p.addErr(err_)

	// consume co.lang.block
	p.advance()

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	inner := parse_block_stmt(p, ddaps)
	blk := inner.Node.(ast.BlockStmt)
	blk.Symb.Name_ = nameTok.Value
	blk.Symb.SymbolType_ = string(symboltable.S_BlockSymbol)
	blk.Symb.IsNamed = true

	_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)

	pr.Node = blk
	return pr
}
