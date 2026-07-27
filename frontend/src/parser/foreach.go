package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// parse_foreach_stmt parses iterator-style statement chains on collection
// values.
//
// Feature examples:
//
//	items.foreach(item).do({
//	    print(item);
//	});
//
//	people.foreach(idx, person).do({
//	    print(person);
//	});
//
// The collection subject has already been resolved before this helper runs; it
// validates the iterator method against the collection subtype, introduces the
// accessor variables, and parses the trailing `.do({ ... })` block.
func parse_foreach_stmt(p *parser, det symboltable.SymbolInfo, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	body := ast.BlockStmt{}
	vs := det.(*symboltable.VarSymbol)
	subType := vs.SubID
	currTok := p.currentToken().Value
	accessor := ""
	key := ""
	if v, ok := Iterator_meth_to_Type[currTok]; ok {
		if slices.Contains(v, subType) {
			p.advance()
			_, err_ := p.expect(scanlex.OPEN_PAREN)
			p.addErr(err_)
			if p.nextTokenSafe(1).Kind == scanlex.COMMA {
				key = p.currentToken().Value
				kind := p.currentTokenKind()
				if key == "__fo" || key == "_" || kind == scanlex.DISCARD_WILD_VAR {
					key = "FO_INTERNAL_KEY"
				}
				p.advance()
				k := make(map[string]any)
				//here it is mostly maps so need key's data type
				//right now we are just considering type of of array
				helpers.CopyMapJson(det, &k)
				k["VarName"] = key
				parse_adhoc_var_decl(p, key, "co.lang.int", ddaps)
				//pushToSymbolTable(p, k)
				_, err_ = p.expect(scanlex.COMMA)
				p.addErr(err_)
			}
			accessor = p.currentToken().Value
			parse_adhoc_var_decl(p, accessor, vs.ActType_, ddaps)
			if accessor == "_" {
				accessor = "FO_INTERNAL"
			}
			b := make(map[string]any)
			helpers.CopyMapJson(det, &b)
			b["VarName"] = accessor

			//pushToSymbolTable(p, b)
			p.advance()

			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)

			_, err_ = p.expect(scanlex.DOT)
			p.addErr(err_)
			if p.currentToken().Value == "do" {
				p.advance()
				_, err_ = p.expect(scanlex.OPEN_PAREN)
				p.addErr(err_)

				br := parse_block_stmt(p, ddaps)

				body = br.Node.(ast.BlockStmt)

				_, err_ = p.expect(scanlex.CLOSE_PAREN)
				p.addErr(err_)
				_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
				p.addErr(err_)
			} else {
				err_ = p.errorObj_frm_st([]string{"do"}, "expected Do")
				p.addErr(err_)
			}

		} else {

			err_ := p.errorObj(nil, "Un Supported Operation")
			p.addErr(err_)
		}
	} else {
		err_ := p.errorObj(nil, "Un Supported Operation")
		p.addErr(err_)

	}
	varName := vs.Name_

	psymbInfo := (*p.SymbolTable_).GetVarDetails(*p.Fs, varName)
	st := psymbInfo.(*symboltable.SymbolDetails)
	fst := ast.ForeachStmt{
		AccessorKeyIdx: key,
		Accessor:       accessor,
		VarName:        varName,
		VarDetails:     *st,
		Body:           body,
		Method:         currTok,
		Symb: &symboltable.StatmentSymbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_StatmentSymbol),
			},
		},
	}

	pr.Node = fst
	return pr
}
