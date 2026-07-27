package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_forall_type parses `forall(T, ...).<inner-type>` as a type (rank-2/3 polymorphism).
// Used when `forall` appears in type position only — as a parameter type, return type,
// or co.lang.type alias.  Examples:
//
//	someFunction(f forall(T).(T)->(T)) -> (co.lang.int) = {}
//	someFArg co.lang.type = forall(T).(T, T)->(T)
//	makeIdentity() -> (forall(T).(T)->(T)) = {}
//
// The dot after the type-parameter list is mandatory per spec.
// The `forall` keyword must be the current token when this is called.
func parse_forall_type(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (ast.Type, []helpers.ErrorInterface) {
	defer p.traceCurrent()()
	// Consume `forall`.
	p.advance()

	// Parse the type parameter list forall(T, R: Constraint, ...).
	typeParams := parseForAllTypeParams(p, ddaps)

	// Activate generic mode so identifiers matching a type param are accepted
	// as generic type variables inside the inner type.
	prevGeneric := p.GenericType
	p.GenericType = true

	// The dot is mandatory: forall(T).(T)->(R).
	// It is the syntactic signal that what follows is a type body, not a declaration name.
	if p.currentTokenKind() == scanlex.DOT {
		p.advance()
	} else {
		err_ := p.errorExpection("forall type expression requires '.' after type parameters: use forall(T).(T)->(T)", helpers.InvalidSyntax)
		p.addErr(err_)
		// continue with error recovery — still attempt to parse the inner type
	}

	// Parse the inner type.  The most common form is a function type (T)->(R).
	var innerType ast.Type
	var errs []helpers.ErrorInterface
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		innerType, errs = parse_fn_type(p, defalt_bp, ddaps)
	} else {
		innerType, errs = parse_type(p, defalt_bp, ddaps)
	}

	p.GenericType = prevGeneric

	ft := ast.ForAllType{
		TypeParams: typeParams,
		Inner:      innerType,
		Symb: &symboltable.TypeSymbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_ForAllSymbol),
			},
		},
	}
	return ft, errs
}

// parseForAllTypeParams parses the type parameter list `(T, R, T: Orderable, ...)` that
// follows the `forall` keyword.  The opening paren must be the current token.
func parseForAllTypeParams(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []symboltable.GenericTypeParam {
	defer p.traceCurrent()()

	params := make([]symboltable.GenericTypeParam, 0)

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
		nameTk, err_ := p.expect(scanlex.IDENTIFIER)
		p.addErr(err_)

		gtp := symboltable.GenericTypeParam{Name: nameTk.Value}

		// Optional constraint: T: ConstraintType
		if p.currentTokenKind() == scanlex.COLON {
			p.advance() // eat :
			constraintTk, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
			p.addErr(err_)
			gtp.Constraint = constraintTk.Value
		}

		params = append(params, gtp)

		if p.currentTokenKind() == scanlex.COMMA {
			p.advance() // eat ,
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	return params
}

// parse_forall_stmt is the handler for `forall` at statement/declaration level.
// Per spec, `forall` at declaration level is a compiler error.
// Use `@co.dap.generic` to declare generic functions, structs, and classes instead.
//
//	❌  forall(T) identity(x T)->(T) = {}
//	✅  @co.dap.generic(type={T:{variance:invariant}})
//	    identity(x T)->(T) = {}
func parse_forall_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	err_ := p.errorExpection(
		"'forall' at declaration level is a compiler error; use @co.dap.generic instead",
		helpers.InvalidSyntax,
	)
	p.addErr(err_)
	// Consume `forall` and sync to the next safe recovery point so parsing can continue.
	p.advance()
	p.sync(scanlex.SEMI_COLON, scanlex.CLOSE_CURLY, scanlex.EOF)
	if p.currentTokenKind() == scanlex.SEMI_COLON {
		p.advance()
	}
	return pr
}
