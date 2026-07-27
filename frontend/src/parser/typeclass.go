package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// ---------------------------------------------------------------------------
// Common helpers
// ---------------------------------------------------------------------------

// skipBalancedParens advances past all tokens inside a matched pair of parens,
// including the closing ')'. The opening '(' must have already been consumed.
func skipBalancedParens(p *parser) {
	defer p.traceCurrent()()

	depth := 1
	for p.hasTokens() && depth > 0 {
		switch p.currentTokenKind() {
		case scanlex.OPEN_PAREN:
			depth++
		case scanlex.CLOSE_PAREN:
			depth--
		}
		p.advance()
	}
}

// parseSig parses one method signature (without body) from the current position.
// Expects the parser to be positioned at the method name identifier.
// Returns a FunctionDeclarationStmt with IsBody=false.
func parseSig(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ast.FunctionDeclarationStmt {
	defer p.traceCurrent()()

	nameTk, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	name := nameTk.Value

	// set up a nested context for this method signature
	savedCtx := p.Context_
	savedSym := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(savedCtx.Id, string(symboltable.S_SignatureSymbol))
	p.Context_.ContextType_ = string(symboltable.S_SignatureSymbol)
	p.Context_.ParentCtxSymbolTableId = savedSym.Id
	savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	params, returns, _, _, _, errs := parse_fn_params_and_body(p, true, false, "LEXICAL", ddaps)
	p.addErr(errs)
	symb := symboltable.FunctionSymbol{
		IsBody: false,
	}
	fn := ast.FunctionDeclarationStmt{
		Name:       name,
		Parameters: params,
		ReturnType: returns,
		Symb:       &symb,
	}

	p.Context_ = savedCtx
	p.SymbolTable_ = savedSym

	// skip optional trailing semicolon
	if p.currentTokenKind() == scanlex.SEMI_COLON {
		p.advance()
	}
	return fn
}

// parseTypeclassDecl parses the shared structure of any typeclass declaration:
//
//	Name(TypeParam1, TypeParam2, ...) = {
//	    methodSig(params) -> (returns);
//	    ...
//	}
//
// The annotation (@co.dap.functor etc.) has already been consumed into ddaps.
// Returns (name, typeParams, methodSignatures).
func parseTypeclassDecl(p *parser, kind string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ast.TypeclassStmt {
	defer p.traceCurrent()()

	// --- typeclass name ---
	nameTk, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	name := nameTk.Value

	// --- optional type parameters: (F), (F(_), G(_)), (T), ... ---
	typeParams := []string{}
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		p.advance() // consume '('
		for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
			if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {
				tp := p.advance().Value
				// Handle HK param suffix like F(_) — skip the nested parens
				if p.currentTokenKind() == scanlex.OPEN_PAREN {
					p.advance() // consume inner '('
					skipBalancedParens(p)
				}
				typeParams = append(typeParams, tp)
			} else {
				p.advance() // skip unexpected (e.g. COMMA)
			}
			if p.currentTokenKind() == scanlex.COMMA {
				p.advance()
			}
		}
		_, err_ = p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err_)
	}

	// --- = ---
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// --- { method signatures } ---
	_, err_ = p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	methods := []ast.Stmt{}
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_CURLY && p.currentTokenKind() != scanlex.EOF {
		if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {
			sig := parseSig(p, ddaps)
			methods = append(methods, sig)
		} else {
			p.advance() // skip unexpected tokens
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)

	ts := ast.TypeclassStmt{
		Name:       name,
		TypeParams: typeParams,
		Methods:    methods,
		Kind:       kind,
	}
	return ts
}

// parseTypeclassInstanceBody parses the body of a typeclass instance impl block.
// Expects position at the opening '{'.
func parseTypeclassInstanceBody(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []ast.Stmt {
	defer p.traceCurrent()()

	_, err_ := p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	body := []ast.Stmt{}
	// Temporarily set IsFunction so parse_fun_stmt is used inside the block
	saved := p.IsFunction
	p.IsFunction = true
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_CURLY && p.currentTokenKind() != scanlex.EOF {
		prev := p.currentToken()
		tr := parse_type_kind_block(p, TYPECLASS, ddaps)
		if tr.Node != nil {
			if s, ok := tr.Node.(ast.Stmt); ok {
				body = append(body, s)
			}
		}
		p.advanceIfStuck(prev)
	}
	p.IsFunction = saved

	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)
	return body
}

// ---------------------------------------------------------------------------
// Typeclass declaration parsers
// ---------------------------------------------------------------------------

// parse_functor_declaration parses:
//
//	@co.dap.Functor
//	Functor(F) = { map(value F(A), f (A)->B) -> (F(B)); }
func parse_functor_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "functor", ddaps)
	ts.Kind = "Functor"
	ts.SetDap(ddaps)
	return ParseResult{Node: ts}
}

// parse_applicative_declaration parses:
//
//	@co.dap.applicative
//	Applicative(F) = { pure(x A) -> (F(A)); apply(fab F(A->B), fa F(A)) -> (F(B)); }
func parse_applicative_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "applicative", ddaps)
	ts.Kind = "Applicative"
	ts.SetDap(ddaps)
	return ParseResult{Node: ts}
}

// parse_monad_declaration parses:
//
//	@co.dap.monad
//	Monad(F) = { pure(x A) -> (F(A)); flatMap(fa F(A), f (A)->F(B)) -> (F(B)); }
func parse_monad_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "monad", ddaps)
	ts.SetDap(ddaps)
	ts.Kind = "Monad"
	return ParseResult{Node: ts}
}

// parse_semigroup_declaration parses:
//
//	@co.dap.semigroup
//	Semigroup(T) = { combine(a T, b T) -> (T); }
func parse_semigroup_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "semigroup", ddaps)
	ts.Kind = "Semigroup"
	return ParseResult{Node: ts}
}

// parse_monoid_declaration parses:
//
//	@co.dap.monoid
//	Monoid(T) = { empty() -> (T); combine(a T, b T) -> (T); }
func parse_monoid_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "monoid", ddaps)
	ts.Kind = "Monoid"
	return ParseResult{Node: ts}
}

// parse_transformer_declaration parses:
//
//	@co.dap.transformer
//	Transformer(F(_), G(_)) = { map(value F(A), f (A)->B) -> (G(B)); }
func parse_transformer_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "transformer", ddaps)
	ts.Kind = "Transformer"
	return ParseResult{Node: ts}
}

// parse_foldable_declaration parses:
//
//	@co.dap.foldable
//	Foldable(F) = { fold(fa F(A), m instance=Monoid(A)) -> (A); foldMap(...) -> (B); }
func parse_foldable_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "foldable", ddaps)
	ts.Kind = "Foldable"
	return ParseResult{Node: ts}
}

// parse_traversable_declaration parses:
//
//	@co.dap.traversable
//	Traversable(F) = { traverse(fa F(A), f (A)->G(B), ap instance=Applicative(G)) -> (G(F(B))); }
func parse_traversable_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "traversable", ddaps)
	ts.Kind = "Traversable"
	return ParseResult{Node: ts}
}

// parse_matcher_declaration parses a custom pattern-matcher declaration:
//
//	@co.dap.matcher
//	Matcher(T) = {
//	    matchCase(value T, pattern co.lang.untyped) -> (co.lang.int, co.lang.MatchBindings);
//	}
func parse_matcher_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	ts := parseTypeclassDecl(p, "matcher", ddaps)

	// Build a FunctionDeclarationStmt from the typeclass pieces so that the
	// MatcherStmt visitor (which expects an embedded FunctionDeclarationStmt)
	// can lower it to a MatcherDeclNode.
	fn := ast.FunctionDeclarationStmt{
		Name: ts.Name,
		Body: ts.Methods,
		Symb: &symboltable.FunctionSymbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_MatcherSymbol),
			},
		},
	}

	typeParam := ""
	if len(ts.TypeParams) > 0 {
		typeParam = ts.TypeParams[0]
	}

	nd := ast.MatcherStmt{
		FunctionDeclarationStmt: fn,
		Type_:                   typeParam,
	}
	nd.SetDap(ddaps)
	return ParseResult{Node: nd}
}

// ---------------------------------------------------------------------------
// Unified typeclass annotation handler
// ---------------------------------------------------------------------------

// parse_typeclass_by_kind handles the unified @co.dap.typeclass(kind=...) annotation.
// It reads the kind parameter and delegates to the appropriate existing parser.
//
//	@co.dap.typeclass(kind=Functor)
//	Functor(F) = { map(value F(A), f (A)->B) -> (F(B)); }
func parse_typeclass_by_kind(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	kind := extractTypeclassKind(ddaps)

	switch strings.ToLower(kind) {
	case "functor":
		return parse_functor_declaration(p, ddaps)
	case "applicative":
		return parse_applicative_declaration(p, ddaps)
	case "monad":
		return parse_monad_declaration(p, ddaps)
	case "semigroup":
		return parse_semigroup_declaration(p, ddaps)
	case "monoid":
		return parse_monoid_declaration(p, ddaps)
	case "transformer":
		return parse_transformer_declaration(p, ddaps)
	case "foldable":
		return parse_foldable_declaration(p, ddaps)
	case "traversable":
		return parse_traversable_declaration(p, ddaps)
	case "matcher":
		return parse_matcher_declaration(p, ddaps)
	default:
		// user-defined typeclass kind — parse as generic typeclass
		ts := parseTypeclassDecl(p, kind, ddaps)
		ts.SetDap(ddaps)
		return ParseResult{Node: ts}
	}
}

// extractTypeclassKind extracts the kind from @co.dap.typeclass(kind=...).
func extractTypeclassKind(ddaps map[scanlex.DirectiveKind][]ast.Stmt) string {
	dir, ok := getAnn(ddaps, "@co.dap.typeclass")
	if !ok {
		return ""
	}
	if v, ok := dir.Parameters["kind"].([]any); ok && len(v) > 0 {
		return fmt.Sprint(v[0])
	}
	// try positional: first value in the parameters map
	for _, v := range dir.Parameters {
		if vals, ok := v.([]any); ok && len(vals) > 0 {
			return fmt.Sprint(vals[0])
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Instance declaration via co.lang.instance kind (spec syntax)
// ---------------------------------------------------------------------------

// parse_instance_declaration parses the spec-syntax typeclass instance:
//
//	ListFunctor co.lang.instance->(for=Functor, type=List) = {
//	    map(value List(A), f (A)->B) -> (List(B)) = { ... }
//	}
//
// Entry: current token is the instance name IDENTIFIER; next token is co.lang.instance.
func parse_instance_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	p.advance() // consume co.lang.instance

	typeclassName, forType := parseInstanceArrowClause(p)

	// --- = ---
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// --- { method implementations } ---
	body := parseTypeclassInstanceBody(p, ddaps)
	symb := symboltable.InstanceSymbol{
		SymbolDetails: symboltable.SymbolDetails{Name_: name.Value,
			SymbolType_: string(symboltable.S_TypeclassSymbol)},
	}
	nd := ast.TypeclassInstanceStmt{
		TypeclassName: typeclassName,
		ForType:       forType,
		Body:          body,
		Symb:          &symb,
	}
	nd.SetDap(ddaps)

	return ParseResult{Node: nd}
}

// ---------------------------------------------------------------------------
// Matcher instance declaration via co.lang.matcher kind (spec syntax)
// ---------------------------------------------------------------------------

// parse_matcher_instance_declaration parses the spec-syntax matcher instance:
//
//	PositiveEvenMatcher co.lang.matcher->(for=Matcher, type=co.lang.int) = {
//	    matchCase(value co.lang.int, pat co.lang.untyped)->(co.lang.int, co.lang.MatchBindings) = { ... }
//	}
//
// Entry: current token is the instance name IDENTIFIER; next token is co.lang.matcher.
func parse_matcher_instance_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	p.advance() // consume co.lang.matcher

	matcherName, forType := parseInstanceArrowClause(p)

	// --- = ---
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// --- { method implementations } ---
	body := parseTypeclassInstanceBody(p, ddaps)
	symb := symboltable.MatcherImplSymbol{
		SymbolDetails: symboltable.SymbolDetails{Name_: name.Value,
			SymbolType_: string(symboltable.S_MatcherImplSymbol)},
	}
	nd := ast.MatcherInstanceStmt{
		MatcherName: matcherName,
		ForType:     forType,
		Body:        body,
		Symb:        &symb,
	}
	nd.SetDap(ddaps)

	return ParseResult{Node: nd}
}

// parseInstanceArrowClause parses the shared ->(for=X, type=Y) clause
// used by both co.lang.instance and co.lang.matcher declarations.
func parseInstanceArrowClause(p *parser) (forName string, forType string) {
	defer p.traceCurrent()()

	if p.currentTokenKind() == scanlex.ARROW {
		p.advance() // eat '->'
		if p.currentTokenKind() == scanlex.OPEN_PAREN {
			p.advance() // eat '('
			for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
				tk := p.advance()
				key := strings.TrimSuffix(tk.Value, "_fo")
				if p.currentTokenKind() == scanlex.ASSIGNMENT {
					p.advance() // eat '='
					valTk := p.advance()
					val := strings.TrimSuffix(valTk.Value, "_fo")
					switch key {
					case "for":
						forName = val
					case "type", "types":
						forType = val
					}
				}
				if p.currentTokenKind() == scanlex.COMMA {
					p.advance()
				}
			}
			_, err_ := p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
		}
	}
	return
}
