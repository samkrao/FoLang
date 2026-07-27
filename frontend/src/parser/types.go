package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	lexer "github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// CompoundTypes lists the token kinds that form compound (union/intersection) types.
var CompoundTypes []lexer.TokenKind = []lexer.TokenKind{lexer.AMPS, lexer.PIPE, lexer.POW}

type type_nud_handler func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface)
type type_led_handler func(p *parser, left ast.Type, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ast.Type

type type_nud_lookup map[lexer.TokenKind]type_nud_handler
type type_led_lookup map[lexer.TokenKind]type_led_handler
type type_bp_lookup map[lexer.TokenKind]binding_power

var type_bp_lu = type_bp_lookup{}
var type_nud_lu = type_nud_lookup{}
var type_led_lu = type_led_lookup{}

func type_led(kind lexer.TokenKind, bp binding_power, led_fn type_led_handler) {
	type_bp_lu[kind] = bp
	type_led_lu[kind] = led_fn
}

// type_nud registers a prefix parser for a token that can begin a type.
//
// Feature examples:
//
//	co.lang.int
//	[]co.lang.string
//	(co.lang.int)->(co.lang.string)
//	forall(T).(T)->(T)
func type_nud(kind lexer.TokenKind, bp binding_power, nud_fn type_nud_handler) {
	type_bp_lu[kind] = primary
	type_nud_lu[kind] = nud_fn
}

// createTypeTokenLookups wires the Pratt-style type parser tables.
//
// Expression parsing and type parsing share the same overall Pratt idea, but
// types have their own token universe and precedence rules. This setup keeps
// type-specific entry points together in one readable place.
func createTypeTokenLookups() {
	// forall(T).(T)->(T) — rank-2/3 type in parameter or return position
	type_nud(lexer.KEYWORD, primary, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
		if p.currentToken().Value == "forall" {
			return parse_forall_type(p, ddaps)
		}
		err_ := p.errorObj(nil, "unexpected keyword in type position: "+p.currentToken().Value)
		p.addErr(err_)
		p.advance()
		return ast.SymbolTypeNode{}, nil
	})

	// (T)->(R) or (forall(T).(T,T)->(T)) -> (co.lang.int) — function type in type position.
	// Required for Rank-3 params/returns where the entire function signature is wrapped in (...).
	type_nud(lexer.OPEN_PAREN, primary, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
		return parse_fn_type(p, defalt_bp, ddaps)
	})

	type_nud(lexer.IDENTIFIER, primary, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
		var tok scanlex.Token
		if p.Adhoc {
			tok = p.AdhocToken
		} else {
			tok = p.advance()
		}
		symb_ := p.SymbolTable_

		if !symb_.ExistsType(*p.Fs, tok.Value, string(symboltable.S_TypeSymbol)) && !p.GenericType {

			v := strings.TrimSuffix(tok.Value, "_fo")
			if slices.Contains(lexer.Builtin_types, v) {
				bdt := ast.BuiltInDataType{
					Value:      v,
					Type:       "data",
					SymbolType: "BDT",
					Symb: &symboltable.TypeSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_TypeSymbol),
						},
					},
				}

				return bdt, nil
			} else if slices.Contains(lexer.Builtin_Kinds, v) {
				bdt := ast.BuiltInDataType{
					Value:      v,
					Type:       "composable",
					SymbolType: "CONTAINER",
					Symb: &symboltable.TypeSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_TypeSymbol),
						},
					},
				}

				return bdt, nil

			}

			err := p.errorExpection("Type "+tok.Value+"  not found ", helpers.NotFound)
			p.addErr(err)
		}

		symbType_ := ast.SymbolTypeNode{
			Value:      tok.Value,
			SymbolType: "UDT",
			Symb: &symboltable.TypeSymbol{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_TypeSymbol),
				},
			},
		}

		if p.GenericType {
			symbType_.SymbolType = "GENERIC"
		}
		return symbType_, nil
	})
	type_nud(lexer.COMPOSITE_IDENTIFER, primary, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
		var tok scanlex.Token
		if p.Adhoc {
			tok = p.AdhocToken
		} else {

			tok = p.advance()
		}
		symb_ := p.SymbolTable_
		if !symb_.ExistsType(*p.Fs, tok.Value, string(symboltable.S_TypeSymbol)) {
			v := strings.TrimSuffix(tok.Value, "_fo")
			if slices.Contains(lexer.Builtin_types, v) {
				bdt := ast.BuiltInDataType{
					Value:      v,
					Type:       "data",
					SymbolType: "BDT",
					Symb: &symboltable.TypeSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_TypeSymbol),
						},
					},
				}

				return bdt, nil
			} else if slices.Contains(lexer.Builtin_Kinds, v) {
				bdt := ast.BuiltInDataType{
					Value:      v,
					Type:       "composable",
					SymbolType: "CONTAINER",
					Symb: &symboltable.TypeSymbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_TypeSymbol),
						},
					},
				}

				return bdt, nil

			}

			err := p.errorExpection("Type "+tok.Value+" not Found ", helpers.NotFound)
			p.addErr(err)
		}
		symbType_ := ast.SymbolTypeNode{
			Value:      tok.Value,
			SymbolType: "UDT",
			Symb: &symboltable.TypeSymbol{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_TypeSymbol),
				},
			},
		}

		return symbType_, nil
	})
	// []number
	type_nud(lexer.OPEN_BRACKET, member, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
		if !p.Adhoc {
			p.advance()
		}
		_, err_ := p.expect(lexer.CLOSE_BRACKET)
		p.addErr(err_)
		insideType, errs := parse_type(p, defalt_bp, ddaps)

		lsty := ast.ListType{
			Underlying: insideType,
		}

		return lsty, errs
	})

	type_nud(lexer.BUILT_IN_TYPE, primary, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
		var tok scanlex.Token
		if p.Adhoc {
			tok = p.AdhocToken
		} else {
			tok = p.advance()
		}
		bdt := ast.BuiltInDataType{
			Value:      tok.Value,
			Type:       "data",
			SymbolType: "BDT",
			Symb: &symboltable.TypeSymbol{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_TypeSymbol),
				},
			},
		}

		return bdt, nil
	})

	type_nud(lexer.BUILT_IN_KIND, primary, func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {

		var tok scanlex.Token
		if p.Adhoc {
			tok = p.AdhocToken
		} else {
			tok = p.advance()
		}

		val := tok.Value
		typ := "composable"
		if slices.Contains(lexer.Builtin_Containers, val) {
			typ = "container"
		}
		bdt := ast.BuiltInDataType{
			Value:      val,
			Type:       typ,
			SymbolType: "CONTAINER",
			Symb: &symboltable.TypeSymbol{
				SymbolDetails: symboltable.SymbolDetails{
					SymbolType_: string(symboltable.S_TypeSymbol),
				},
			},
		}

		return bdt, nil
	})

}

// parse_fn_type parses a function type in pure type position.
//
// Feature examples:
//
//	(co.lang.int)->(co.lang.int)
//	(forall(T).(T)->(T))->(co.lang.int)
//
// Internally this reuses function declaration parsing in a no-body mode so the
// parameter/result grammar stays identical between declarations and type uses.
func parse_fn_type(p *parser, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {

	name := "__co_internal_fun_" + helpers.GenUnique(4)
	pr := parse_fn_gen_declaration(p, FUNTYPE, true, true, true, name, ddaps)
	st := pr.Node.(ast.FunctionDeclarationStmt)
	sd := symboltable.SymbolDetails{
		Name_:       name,
		SymbolType_: string(symboltable.S_FunctionType),
	}
	symb := symboltable.TypeSymbol{

		SymbolDetails: sd,
	}
	fnType := ast.FunctionType{
		Params:  st.Parameters,
		Results: st.ReturnType,
		Symb:    &symb,
		Parent:  st,
	}

	//updateContext(p, fnType, symboltable.Function, false)
	return fnType, pr.Errors
}

// parse_compound_type parses chained union/intersection-like type operators.
//
// Feature examples:
//
//	Person | Company
//	Result | Error
//
// Today `|` is the main supported branch; `&` and `^` are recognized but still
// guarded as unsupported where the current frontend has not finished them.
func parse_compound_type(p *parser, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
	defer p.traceCurrent()()

	var cmpd ast.CompoundType
	var errs []helpers.ErrorInterface
	if slices.Contains(CompoundTypes, p.nextTokenSafe(1).Kind) {
		cmpd = ast.CompoundType{}

		l, err_ := parse_type(p, bp, ddaps)
		p.addErr(err_)
		cmpd.Left = l
		op := p.advance()
		if op.Value == "&" || op.Value == "^" {
			err_ := p.errorExpection("Currently unsupported", helpers.UnSupported)
			p.addErr(err_)
		}
		cmpd.Op = op.Value
		l, err_ = parse_compound_type(p, bp, ddaps)
		p.addErr(err_)
		cmpd.Right = l
	} else {
		return parse_type(p, bp, ddaps)
	}
	return cmpd, errs
}

// parse_generic_types parses a generic type placeholder or generic type head.
//
// Feature examples:
//
//	T
//	Result(T)
//
// This wraps the underlying type node in ast.GenericType so later phases can
// distinguish a generic placeholder from a resolved concrete type.
func parse_generic_types(p *parser, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
	defer p.traceCurrent()()

	var left ast.GenericType = ast.GenericType{}
	var errs []helpers.ErrorInterface

	tokenKind := p.currentTokenKind()

	nud_fn, exists := type_nud_lu[tokenKind]
	if !exists {
		err_ := p.errorObj(nil, fmt.Sprintf("type: Un Supported %s\n", lexer.TokenKindString(tokenKind)))
		p.addErr(err_)
		return left, errs
	}
	left.Type_, errs = nud_fn(p, ddaps)
	name, _ := left.Type_.GetActType()
	symb := symboltable.TypeSymbol{
		SymbolDetails: symboltable.SymbolDetails{
			Name_:       name,
			SymbolType_: string(symboltable.S_TypeSymbol),
		},
	}
	left.Symb = &symb
	return left, errs
}

// parse_adhoc_type parses a type starting from a synthetic or already-read
// token instead of consuming directly from the token stream.
//
// Feature example:
//
//	Used when declaration parsing has already consumed the identifier but still
//	needs to reinterpret it as a type head without rewinding the parser.
func parse_adhoc_type(p *parser, bp binding_power, currentToken scanlex.Token, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
	defer p.traceCurrent()()

	var left ast.Type
	var errs []helpers.ErrorInterface

	tokenKind := p.currentTokenKind()
	if currentToken != scanlex.DummyNode {
		tokenKind = currentToken.Kind
		p.Adhoc = true
		p.AdhocToken = currentToken
	}

	nud_fn, exists := type_nud_lu[tokenKind]
	if !exists {
		err_ := p.errorObj(nil, fmt.Sprintf("type: Un Supported %s\n", lexer.TokenKindString(tokenKind)))
		p.addErr(err_)
		p.Adhoc = false
		p.AdhocToken = scanlex.DummyNode
		return left, errs
	}
	left, errs = nud_fn(p, ddaps)

	p.Adhoc = false
	p.AdhocToken = scanlex.DummyNode
	return left, errs
}

// parse_type is the Pratt entry point for FoLang type parsing.
//
// Feature examples:
//
//	co.lang.int
//	[]co.lang.string
//	(co.lang.int)->(co.lang.string)
//	Person | Company
//
// The loop starts with a type prefix parser (nud) and then folds any trailing
// type operators whose binding power is higher than the caller's threshold.
func parse_type(p *parser, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
	defer p.traceCurrent()()

	if !enterRec(p) {
		p.addErr(p.errorObj(nil, "recursion depth exceeded in type"))
		return ast.SymbolTypeNode{}, nil
	}
	defer leaveRec(p)
	var left ast.Type
	var errs []helpers.ErrorInterface

	tokenKind := p.currentTokenKind()
	nud_fn, exists := type_nud_lu[tokenKind]
	if !exists {
		err_ := p.errorObj(nil, fmt.Sprintf("type: Un Supported %s\n", lexer.TokenKindString(tokenKind)))
		p.addErr(err_)
		return left, errs
	}
	left, errs = nud_fn(p, ddaps)

	for nextBP, ok := type_bp_lu[p.currentTokenKind()]; ok && nextBP > bp; {
		tokenKind = p.currentTokenKind()
		led_fn, exists := type_led_lu[tokenKind]
		if !exists {
			err_ := p.errorObj(nil, fmt.Sprintf("type: Un supported %s\n", lexer.TokenKindString(tokenKind)))
			p.addErr(err_)
			break
		}
		left = led_fn(p, left, bp, ddaps)
		errs = nil
	}

	return left, errs
}
