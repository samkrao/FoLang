package parser

import (
	"fmt"
	"math"
	"strconv"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// makeNumberLiteral converts a numeric token value into the most specific
// literal node used by the current AST.
//
// Feature examples:
//
//	42     -> ast.IntegerLiteral
//	3.14   -> ast.NumberLiteral
//
// This helper is intentionally small and side-effect free so primary-expression
// parsing can stay focused on grammar decisions rather than literal conversion.
func makeNumberLiteral(val string) (ast.Expr, error) {

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, err
	}
	if math.Trunc(f) == f {
		return ast.IntegerLiteral{Value: int64(f), Type_: "Integer", ActType_: "co.lang.int"}, nil
	}
	return ast.NumberLiteral{Value: f, Type_: "Float", ActType_: "co.lang.float"}, nil
}

// makeCharLiteral converts a quoted character token into a CharacterLiteral.
//
// Feature example:
//
//	'a' -> ast.CharacterLiteral{Value: 'a'}
func makeCharLiteral(val string) (ast.Expr, error) {
	r := []rune(val)
	if len(r) < 2 {
		return nil, fmt.Errorf("invalid char literal")
	}
	return ast.CharacterLiteral{Value: r[1], ActType_: "co.lang.char"}, nil
}

// parse_expr is the Pratt parser entry point for FoLang expressions.
//
// Feature examples:
//
//	a + b * c
//	nums.each(|x| => x + 1)
//	value == 10
//	1..10
//
// Pratt parsing works in two phases:
//  1. parse a null denotation (nud) for the current token
//  2. keep folding left denotations (led) while the next operator has higher
//     binding power than the current expression context
//
// Recursive descent is still used elsewhere for declarations, statements,
// and declaration-specific bodies. Expressions are where Pratt parsing gives
// the clearest precedence handling with the least branching.
func parse_expr(p *parser, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	if !enterRec(p) {
		p.addErr(p.errorObj(nil, "recursion depth exceeded in expression"))
		return ParseResult{}
	}
	defer leaveRec(p)
	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}

	left := parse_nud_expr(p, ddaps)
	if left.Node == nil {
		return res
	}

	for should_continue_expr(p, bp) {
		left = parse_led_expr(p, left.Node.(ast.Expr), bp, ddaps)
		if left.Node == nil {
			return res
		}
	}

	res.Node = left.Node.(ast.Expr)
	return res
}

// parse_nud_expr resolves the current token through the Pratt nud table.
//
// Feature examples:
//
//	42             -> parse_primary_expr
//	-x             -> parse_prefix_expr
//	(a + b)        -> parse_grouping_expr
//	|x| => x + 1   -> parse_lambda_expr
//
// A nud parser is responsible for tokens that can *start* an expression.
func parse_nud_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	tokenKind := p.currentTokenKind()
	nudFn, exists := nud_lu[tokenKind]
	if !exists {
		err_ := p.errorObj(nil, fmt.Sprintf("Invalid token %s\n", scanlex.TokenKindString(tokenKind)))
		p.addErr(err_)
		return ParseResult{}
	}
	return nudFn(p, ddaps)
}

// parse_led_expr resolves the current token through the Pratt led table.
//
// Feature examples:
//
//	a + b          -> parse_binary_expr
//	user.name      -> parse_member_expr
//	add(1, 2)      -> parse_call_expr
//	x = y          -> parse_assignment_expr
//
// An led parser is responsible for tokens that extend an already-parsed left
// expression.
func parse_led_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	tokenKind := p.currentTokenKind()
	ledFn, exists := led_lu[tokenKind]
	if !exists {
		err_ := p.errorObj(nil, fmt.Sprintf("Invalid token %s\n", scanlex.TokenKindString(tokenKind)))
		p.addErr(err_)
		return ParseResult{}
	}
	return ledFn(p, left, bp, ddaps)
}

// should_continue_expr answers the core Pratt loop question:
// does the upcoming operator bind more tightly than the caller's binding power?
//
// If yes, the loop consumes that operator through its led handler and grows the
// expression tree. If not, control returns to the surrounding parse context.
func should_continue_expr(p *parser, bp binding_power) bool {
	tokenBp, exists := bp_lu[p.currentTokenKind()]
	return exists && tokenBp > bp
}

// parse_prefix_expr parses prefix operators such as:
//
//	-x
//	!done
func parse_prefix_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	operatorToken := p.advance()
	expr := parse_expr(p, unary, ddaps)

	pref := ast.PrefixExpr{
		Operator: operatorToken,
		Right:    expr.Node.(ast.Expr),
	}
	res.Node = pref
	return res
}

// makeConditionalBinary constructs the AST node used for relational and
// logical operators.
//
// Feature examples:
//
//	a == b
//	age >= 18
//	isReady && hasValue
func makeConditionalBinary(left ast.Expr, operator scanlex.Token, right ast.Expr) ast.ConditionalExpr {
	return ast.ConditionalExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
		Type:     "CONDITIONAL_OP",
		ValOrVar: "NA",
	}
}

// makeBinary constructs the generic binary-expression node used for arithmetic
// and other non-conditional infix operators.
func makeBinary(left ast.Expr, operator scanlex.Token, right ast.Expr) ast.BinaryExpr {
	return ast.BinaryExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
	}
}

// parse_assignment_expr parses assignment-shaped infix expressions:
//
//	x = y
//	total += 1
//	count -= 1
func parse_assignment_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	operator := p.advance()
	rhs := parse_expr(p, bp, ddaps)
	aEx := ast.AssignmentExpr{
		Assigne:       left,
		Operator:      operator,
		AssignedValue: rhs.Node.(ast.Expr),
	}
	res.Node = aEx
	return res
}

// parse_range_expr parses infix range expressions:
//
//	1..10
//	start..end
//	1..
func parse_range_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}
	p.advance() // consume ..
	rEx := ast.RangeExpr{Lower: left}
	// Support open upper bound: `1..` (terminated by ; , ) EOF)
	tk := p.currentTokenKind()
	if tk != scanlex.SEMI_COLON && tk != scanlex.COMMA &&
		tk != scanlex.CLOSE_PAREN && tk != scanlex.EOF {
		upp := parse_expr(p, bp, ddaps)
		rEx.Upper = upp.Node.(ast.Expr)
	}
	res.Node = rEx
	return res
}

// parse_range_expr_variant parses exclusive-bound range operators:
//
//	1..<10
//	1<..<10
//	1<..10
func parse_range_expr_variant(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}
	opTok := p.advance()
	excludeStart := opTok.Kind == scanlex.LT_DOT_DOT || opTok.Kind == scanlex.LT_DOT_DOT_LT
	excludeEnd := opTok.Kind == scanlex.DOT_DOT_LT || opTok.Kind == scanlex.LT_DOT_DOT_LT
	rEx := ast.RangeExpr{Lower: left, ExcludeStart: excludeStart, ExcludeEnd: excludeEnd}
	tk := p.currentTokenKind()
	if tk != scanlex.SEMI_COLON && tk != scanlex.COMMA &&
		tk != scanlex.CLOSE_PAREN && tk != scanlex.EOF {
		upp := parse_expr(p, bp, ddaps)
		rEx.Upper = upp.Node.(ast.Expr)
	}
	res.Node = rEx
	return res
}

// parse_prefix_range_expr parses open-lower-bound ranges:
//
//	..10
func parse_prefix_range_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}
	p.advance()            // consume ..
	rEx := ast.RangeExpr{} // Lower is nil — open lower bound (e.g. ..100)
	tk := p.currentTokenKind()
	if tk != scanlex.SEMI_COLON && tk != scanlex.COMMA &&
		tk != scanlex.CLOSE_PAREN && tk != scanlex.EOF {
		upp := parse_expr(p, logical, ddaps)
		rEx.Upper = upp.Node.(ast.Expr)
	}
	res.Node = rEx
	return res
}

// parse_comma_expr parses comma-chained expressions:
//
//	a, b
//	x := 1, y := 2
func parse_comma_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	operator := p.advance()
	rhs := parse_expr(p, bp, ddaps)
	cEx := ast.CommaExpr{
		Left:     left,
		Operator: operator,
		Right:    rhs.Node.(ast.Expr),
	}
	res.Node = cEx
	return res
}

// parse_type_expr_stmt parses ADT/type-expression operators that are still
// represented through the expression machinery.
//
// Feature examples:
//
//	A | B
//	A & B
func parse_type_expr_stmt(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	operatorToken := p.advance()
	right := parse_expr(p, defalt_bp, ddaps)

	if slices.Contains(adt_operators, operatorToken.Kind) {
		if !slices.Contains(sp_adt, operatorToken.Kind) {
			err := p.errorExpection("Unsupported token "+operatorToken.Value, helpers.UnSupported)
			p.addErr(err)
		}
		adt := ast.ADTExpr{
			Left:     left,
			Operator: operatorToken,
			Right:    right.Node.(ast.Expr),
		}
		res.Node = adt
	} else {
		err_ := p.errorExpection("Invalid Token "+operatorToken.Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}
	return res
}

// parse_binary_expr parses standard infix operators.
//
// Feature examples:
//
//	a + b
//	total * factor
//	ready && ok
//	count == 10
//
// Conditional/logical operators produce ast.ConditionalExpr because later
// stages use that node family for control-flow related features. Pure
// arithmetic operators produce ast.BinaryExpr.
func parse_binary_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	operatorToken := p.advance()

	right := parse_expr(p, bp, ddaps)

	if slices.Contains(relational_logical_ops, operatorToken.Kind) {
		res.Node = makeConditionalBinary(left, operatorToken, right.Node.(ast.Expr))
		return res
	}
	res.Node = makeBinary(left, operatorToken, right.Node.(ast.Expr))

	return res
}

func parse_builtin_consts(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	var leftExpr ast.Expr = ast.PlaceHolderExpr{}
	v := p.advance().Value
	stmt := ast.BuiltInConstantStmt{
		Identifier: ast.SymbolExpr{
			Value:        v,
			IsMethodCall: false,
		},
		Type_: "BUILT_IN_CONSTANTS",
	}
	leftExpr = ast.StatementExpr{
		Statement: stmt,
	}
	res.Node = leftExpr
	return res
}

func parse_primary_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	return parse_primary_expr_inner(p, map[string]bool{}, ddaps)

}
func validScopeForExists(p *parser) bool {
	defer p.traceCurrent()()

	if p.Scope == "LEXICAL" || p.Scope == "STATIC" {
		return true
	}
	return false
}
func parse_primary_expr_inner(p *parser, special map[string]bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	symbTab := p.SymbolTable_

	iterators := false
	methodexpr := false
	if p.nextTokenSafe(1).Kind == scanlex.DOT {
		if p.nextToken(2).Kind == scanlex.BUILT_IN_METHOD || p.nextToken(2).Kind == scanlex.IDENTIFIER {

			if p.nextToken(3).Kind == scanlex.OPEN_PAREN {
				methodexpr = true
				if slices.Contains(ISIn, p.nextToken(2).Value) {
					/*
					* This is for handling (arr.contains(k)).do expression parsing (through grouping)
					 */
					iterators = true
				}

			}
		}
	}
	var leftExpr ast.Expr = ast.PlaceHolderExpr{}
	switch p.currentTokenKind() {
	case scanlex.NUMBER:
		lit, err := makeNumberLiteral(p.advance().Value)
		if err != nil {
			p.addErr(p.errorObj(nil, err.Error()))
		}
		res.Node = lit
		return res

	case scanlex.CHAR:
		lit, err := makeCharLiteral(p.advance().Value)
		if err != nil {
			p.addErr(p.errorObj(nil, err.Error()))
		}
		res.Node = lit
		return res

	case scanlex.STRING:
		res.Node = ast.StringLiteral{
			Value:    p.advance().Value,
			ActType_: "co.lang.string",
		}
		return res
	case scanlex.BUILT_IN_TYPE, scanlex.BUILT_IN_KIND:
		type_, err_ := parse_type(p, defalt_bp, ddaps)
		p.addErr(err_)
		leftExpr = ast.SDTExpr{
			Type_: type_,
		}

	case scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER:

		varTok := p.advance()
		v := varTok.Value
		if v == "_" || v == "__fo" {
			err_ := p.errorObj(nil, "Cannot use _")
			p.addErr(err_)
		}
		typ_ := string(symboltable.S_VarSymbol)
		isMethodCall := false
		isADT := p.IsADT
		if p.currentTokenKind() == scanlex.OPEN_PAREN {
			typ_ = string(symboltable.S_FunctionSymbol)
			isMethodCall = true
		}

		if isADT && !symbTab.ExistsType(*p.Fs, v, string(symboltable.S_TypeSymbol)) && validScopeForExists(p) {
			err_ := p.errorObj(nil, "Type "+v+" not declared")
			p.addErr(err_)
		} else if !isADT && !symbTab.Exists(*p.Fs, v, typ_) && validScopeForExists(p) {

			err_ := p.errorObj(nil, "Variable "+v+" not declared")
			p.addErr(err_)

		}
		if isADT {
			p.backtrack()
			type_, errs_ := parse_type(p, defalt_bp, ddaps)
			p.addErr(errs_)
			leftExpr = ast.SDTExpr{
				Type_: type_,
			}

		} else {
			temp := symbTab.GetDetails(*p.Fs, v, typ_)
			vartype := "GEN"

			if temp.GetType() == "co.lang.bool" {
				vartype = "BOOL"
			}
			stmt := ast.SymbolRefExpr{
				Identifier: ast.SymbolExpr{
					Value:        v,
					IsMethodCall: isMethodCall,
					SymbolType_:  string(temp.GetSymbolType()),
				},
				ExprType:       vartype,
				SymbolKind_:    string(temp.GetSymbolType()),
				AdditionalInfo: temp,
			}
			leftExpr = ast.StatementExpr{
				Statement: stmt,
			}
		}
	case scanlex.DISCARD_WILD_VAR:
	case scanlex.BIND_VAR:
		name := p.currentToken().Value // "$", "$0", "$1", etc.
		idx := -1
		if len(name) > 1 {
			if n, err := strconv.Atoi(name[1:]); err == nil {
				idx = n
			}
		}
		p.advance()
		res.Node = ast.BindVariableExpr{
			Name:  name,
			Index: idx,
		}
		return res
	default:
		err_ := p.errorObj(nil, fmt.Sprintf("Cannot create primary_expr from %s\n", scanlex.TokenKindString(p.currentTokenKind())))
		p.addErr(err_)

	}
	if methodexpr {
		if iterators {
			_, err_ := p.expect(scanlex.DOT)
			p.addErr(err_)
			p.advance()
			_, err_ = p.expect(scanlex.OPEN_PAREN)
			p.addErr(err_)
			valExpr := parse_primary_expr_inner(p, map[string]bool{"methexpr": true}, ddaps)
			if nd, ok := valExpr.Node.(ast.StatementExpr); ok {
				res.Node = ast.ConditionalExpr{
					ArrayVar:    leftExpr.(ast.StatementExpr).Statement,
					CondVarStmt: nd,
					Type:        "ARR_CONTAINS",
					ValOrVar:    "VAR",
				}

			} else {
				res.Node = ast.ConditionalExpr{
					ArrayVar:    leftExpr.(ast.StatementExpr).Statement,
					CondValStmt: valExpr.Node.(ast.Expr),
					Type:        "ARR_CONTAINS",
					ValOrVar:    "VAL",
				}
			}

			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
			return res

		} else {
			return parse_call_expr(p, leftExpr, defalt_bp, ddaps)
		}
	} else {
		res.Node = leftExpr
		return res
	}
}

func parse_member_expr(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   ast.PlaceHolderExpr{},
	}

	tk0 := p.advance()
	isComputed := tk0.Kind == scanlex.OPEN_BRACKET
	if isComputed {
		rhs := parse_expr(p, bp, ddaps)
		_, err_ := p.expect(scanlex.CLOSE_BRACKET)
		p.addErr(err_)
		res.Node = ast.ComputedExpr{
			Member:   left,
			Property: rhs.Node.(ast.Expr),
		}
		return res
	}

	tk := p.currentToken().Kind
	prop_p, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.BUILT_IN_METHOD)
	p.addErr(err_)
	prop := prop_p.Value
	res.Node = ast.MemberExpr{
		Member:   left,
		Property: prop,
		Type_:    tk,
	}
	return res
}

func parse_array_literal_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	_, err_ := p.expect(scanlex.OPEN_BRACKET)
	p.addErr(err_)
	arrayContents := make([]ast.Expr, 0)

	if p.currentTokenKind() != scanlex.CLOSE_BRACKET {
		for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_BRACKET {
			tmp := parse_expr(p, logical, ddaps)
			arrayContents = append(arrayContents, tmp.Node.(ast.Expr))

			if !p.currentToken().IsOneOfMany(scanlex.EOF, scanlex.CLOSE_BRACKET) {
				_, err_ := p.expect(scanlex.COMMA)
				p.addErr(err_)
			}
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_BRACKET)
	p.addErr(err_)

	res.Node = ast.ArrayLiteral{
		Contents: arrayContents,
	}
	return res
}
