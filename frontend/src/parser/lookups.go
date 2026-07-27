package parser

import (
	"sync"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

var initLookupsOnce sync.Once

type binding_power int

const (
	defalt_bp binding_power = iota
	comma
	assignment
	logical
	relational
	additive
	multiplicative
	unary
	call
	member
	builtindatatype
	builtinconsts
	primary
)

type stmt_handler func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult
type nud_handler func(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult
type led_handler func(p *parser, left ast.Expr, bp binding_power, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult

type stmt_lookup map[StmtKind]stmt_handler
type nud_lookup map[scanlex.TokenKind]nud_handler
type led_lookup map[scanlex.TokenKind]led_handler
type bp_lookup map[scanlex.TokenKind]binding_power
type bp_su_lookup map[StmtKind]binding_power

var bp_lu = bp_lookup{}
var nud_lu = nud_lookup{}
var led_lu = led_lookup{}
var stmt_lu = stmt_lookup{}
var bp_su = bp_su_lookup{}

// led registers a Pratt left denotation handler for an infix/postfix token.
//
// Feature examples:
//
//	a + b      -> PLUS
//	user.name  -> DOT
//	add(1, 2)  -> OPEN_PAREN
func led(kind scanlex.TokenKind, bp binding_power, led_fn led_handler) {
	bp_lu[kind] = bp
	led_lu[kind] = led_fn
}

// nud registers a Pratt null denotation handler for a token that can begin an
// expression.
//
// Feature examples:
//
//	42         -> NUMBER
//	"hello"    -> STRING
//	(a + b)    -> OPEN_PAREN
//	|x| => x   -> PIPE
func nud(kind scanlex.TokenKind, bp binding_power, nud_fn nud_handler) {
	bp_lu[kind] = primary
	nud_lu[kind] = nud_fn
}

// stmt registers a statement-level handler for keyword-style statement entry
// points.
//
// Feature examples:
//
//	let x := 10
//	forall(T).(T)->(T)
func stmt(kind StmtKind, stmt_fn stmt_handler) {
	bp_su[kind] = defalt_bp
	stmt_lu[kind] = stmt_fn
}

// createTokenLookups builds the Pratt operator tables and the small statement
// dispatch table used by parse_stmt.
//
// The registration functions are intentionally grouped by grammar concern so
// the parser reads top-down:
// 1. assignment
// 2. ranges
// 3. type operators
// 4. binary operators
// 5. expression starters
// 6. prefix forms
// 7. call/member continuations
// 8. statement entry points
func createTokenLookups() {
	registerAssignmentOperators()
	registerRangeOperators()
	registerTypeOperators()
	registerBinaryOperators()
	registerPrimaryParsers()
	registerPrefixParsers()
	registerCallAndMemberParsers()
	registerStatementHandlers()
}

// registerAssignmentOperators wires assignment-like infix operators:
//
//	x = y
//	total += 1
//	count -= 1
func registerAssignmentOperators() {
	led(scanlex.ASSIGNMENT, assignment, parse_assignment_expr)
	led(scanlex.PLUS_EQUALS, assignment, parse_assignment_expr)
	led(scanlex.MINUS_EQUALS, assignment, parse_assignment_expr)
}

// registerRangeOperators wires the range family:
//
//	1..10
//	1..<10
//	..10
func registerRangeOperators() {
	nud(scanlex.DOT_DOT, primary, parse_prefix_range_expr)
	led(scanlex.DOT_DOT, logical, parse_range_expr)
	led(scanlex.LT_DOT_DOT, logical, parse_range_expr_variant)
	led(scanlex.DOT_DOT_LT, logical, parse_range_expr_variant)
	led(scanlex.LT_DOT_DOT_LT, logical, parse_range_expr_variant)
}

// registerTypeOperators wires union/intersection-like operators that are
// routed through the expression parser for now:
//
//	A | B
//	A & B
func registerTypeOperators() {
	led(scanlex.AMPS, logical, parse_type_expr_stmt)
	led(scanlex.PIPE, logical, parse_type_expr_stmt)
	led(scanlex.POW, logical, parse_type_expr_stmt)
}

// registerBinaryOperators wires ordinary logical, relational, additive, and
// multiplicative operators.
func registerBinaryOperators() {
	for _, kind := range []scanlex.TokenKind{scanlex.AND, scanlex.OR} {
		led(kind, logical, parse_binary_expr)
	}
	for _, kind := range []scanlex.TokenKind{
		scanlex.LESS,
		scanlex.LESS_EQUALS,
		scanlex.GREATER,
		scanlex.GREATER_EQUALS,
		scanlex.EQUALS,
		scanlex.NOT_EQUALS,
	} {
		led(kind, relational, parse_binary_expr)
	}
	for _, kind := range []scanlex.TokenKind{scanlex.PLUS, scanlex.MINUS} {
		led(kind, additive, parse_binary_expr)
	}
	for _, kind := range []scanlex.TokenKind{scanlex.SLASH, scanlex.STAR, scanlex.PERCENT} {
		led(kind, multiplicative, parse_binary_expr)
	}
	led(scanlex.COMMA, comma, parse_comma_expr)
}

// registerPrimaryParsers wires tokens that can begin a primary expression.
//
// Feature examples:
//
//	10
//	"abc"
//	name
//	co.lang.int
//	(a + b)
func registerPrimaryParsers() {
	for _, kind := range []scanlex.TokenKind{
		scanlex.NUMBER,
		scanlex.STRING,
		scanlex.BOOL,
		scanlex.CHAR,
		scanlex.IDENTIFIER,
		scanlex.COMPOSITE_IDENTIFER,
		scanlex.BUILT_IN_TYPE,
	} {
		nud(kind, primary, parse_primary_expr)
	}
	nud(scanlex.BUILT_IN_CONSTANTS, builtinconsts, parse_builtin_consts)
	nud(scanlex.BUIL_IN_STMT_EXPRS, builtinconsts, parse_built_in_stmt_expr)
	nud(scanlex.OPEN_PAREN, defalt_bp, parse_grouping_expr)
	nud(scanlex.BUILT_IN_KIND, primary, parse_anon_type_expr)
}

// registerPrefixParsers wires prefix expression starters:
//
//	-x
//	!ready
//	[1, 2, 3]
//	|x| => x + 1
func registerPrefixParsers() {
	nud(scanlex.MINUS, unary, parse_prefix_expr)
	nud(scanlex.NOT, unary, parse_prefix_expr)
	nud(scanlex.OPEN_BRACKET, primary, parse_array_literal_expr)
	nud(scanlex.PIPE, primary, parse_lambda_expr)
}

// registerCallAndMemberParsers wires the postfix/infix continuations that grow
// a left expression:
//
//	user.name
//	nums[0]
//	add(1, 2)
func registerCallAndMemberParsers() {
	led(scanlex.DOT, member, parse_member_expr)
	led(scanlex.OPEN_BRACKET, member, parse_member_expr)
	led(scanlex.OPEN_PAREN, call, parse_call_expr)
}

// registerStatementHandlers wires keyword-led statement entry points used by
// parse_stmt.
func registerStatementHandlers() {
	stmt(BUILTIN_STMT_EXPRS, parse_built_in_stmt_expr)
	stmt(RESERVEDWORDS, parse_built_in_stmt_expr)
	stmt(IDENTIFIERS, parse_built_in_stmt_expr)
	stmt(LET, parse_let_binding)
	stmt(FORALL, parse_forall_stmt)
}
