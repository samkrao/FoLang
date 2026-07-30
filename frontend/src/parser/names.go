package parser

import (
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Names and references — section 3 of docs/grammar/folang.ebnf.
//
// One scanner behaviour shapes every function here: token folding collapses a
// dotted name into a single token before the parser sees it. `co.lang.int` arrives
// as one BUILT_IN_TYPE, `co.lang.struct` as one BUILT_IN_KIND, `co.out` as one
// BUIL_IN_STMT_EXPRS, and a user-written `pkg.Type` as one COMPOSITE_IDENTIFER.
//
// So `qualified-name = ( identifier | "co" ), { ".", identifier }` is usually
// already one token, and the loop over "." segments exists for the cases folding
// leaves alone — chiefly a dot that follows ")" or "}" rather than a name, which
// folding never rewrites.

// name is a parsed name together with the token it came from.
//
// Two spellings are kept deliberately. Scanned is what the scanner produced,
// carrying the "_fo" backend lowering suffix (DECISION-BACKEND-001), and is what
// goes into the AST because it is what the backend emits. Logical is the spelling
// as written in source and is what grammar comparisons use.
type name struct {
	Scanned string
	Logical string
	Tok     scanlex.Token
}

// isWildcard reports whether this name is the contextual underscore token.
func (n name) isWildcard() bool { return n.Tok.Kind == scanlex.DISCARD_WILD_VAR }

// nameFrom builds a name from a token.
func nameFrom(tok scanlex.Token) name {
	return name{Scanned: tok.Value, Logical: logicalName(tok.Value), Tok: tok}
}

// atIdentifier reports whether the cursor holds something usable as an ordinary
// identifier.
//
// COMPOSITE_IDENTIFER is included because a folded dotted user name occupies the
// same grammatical slot as a plain one. BUILT_IN_METHOD is included because
// folding classifies a reserved member name that way — `arr.each` yields
// COMPOSITE_IDENTIFER("arr"), DOT, BUILT_IN_METHOD("each") — and the member is
// still just a name at this level.
func (p *parser) atIdentifier() bool {
	return p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
}

// parseIdentifier consumes one identifier.
//
// DECISION-LEX-001 and DECISION-LEX-006 are enforced by the scanner: an
// identifier starts with an ASCII letter, never contains consecutive
// underscores and never ends in one, and a lone "_" is a separate contextual
// token that is never an identifier.
func (p *parser) parseIdentifier(context string) name {
	if !p.atIdentifier() {
		p.failf(p.cur(), "expected an identifier %s, found %s", context, describeToken(p.cur()))
	}
	return nameFrom(p.advance())
}

// parseDeclarationName parses the declaration-name production:
//
//	declaration-name = identifier | "_"
//
// The underscore form exists so a primary declaration can take its name from the
// filename instead of from the source text (docs/language-ref.md, "Primary
// Declaration Names and Filename Inference"). When it is used, the filename's
// base is substituted so the node still carries a usable name.
func (p *parser) parseDeclarationName(context string) name {
	if p.at(scanlex.DISCARD_WILD_VAR) {
		tok := p.advance()
		inferred := p.file.Basename
		if inferred == "" {
			inferred = "_"
		}
		return name{Scanned: inferred, Logical: inferred, Tok: tok}
	}
	return p.parseIdentifier(context)
}

// parseQualifiedName parses the qualified-name production:
//
//	qualified-name = ( identifier | "co" ), { ".", identifier }
//
// Folded tokens already contain their dots, so the loop only runs for the
// segments folding left separate.
func (p *parser) parseQualifiedName(context string) name {
	return p.parseQualifiedNameWith(context, isNameSegmentToken)
}

// parseQualifiedTypeName parses a qualified-name in TYPE position.
//
// It differs from parseQualifiedName in one respect: a "." always continues the name. A type
// has no method chain, so there is no construct for a reserved member name to begin, and a
// co.* path that is not in the built-in type table arrives split across its namespace and its
// last segment — `co.lang.map` presents as BUIL_IN_STMT_EXPRS("co.lang") plus
// BUILT_IN_METHOD("map") — which has to be rejoined here to name the type.
func (p *parser) parseQualifiedTypeName(context string) name {
	return p.parseQualifiedNameWith(context, func(tok scanlex.Token) bool {
		return p.isMemberNameToken(tok)
	})
}

// parseQualifiedNameWith parses a qualified-name whose continuation segments are accepted by
// extends.
func (p *parser) parseQualifiedNameWith(context string, extends func(scanlex.Token) bool) name {
	head := p.cur()
	switch {
	case p.atIdentifier(),
		p.at(scanlex.BUILT_IN_TYPE),
		p.at(scanlex.BUILT_IN_KIND),
		p.at(scanlex.BUIL_IN_STMT_EXPRS),
		p.at(scanlex.BUILT_IN_CONSTANTS),
		p.at(scanlex.KEYWORD),
		p.at(scanlex.CONTEXT_KEYWORD):
		p.advance()
	default:
		p.failf(head, "expected a qualified name %s, found %s", context, describeToken(head))
	}

	scanned := head.Value
	last := head
	for p.at(scanlex.DOT) && extends(p.peek(1)) {
		p.advance() // "."
		seg := p.advance()
		scanned += "." + seg.Value
		last = seg
	}

	tok := scanlex.NewUniqueToken(head.Kind, scanned, head.StartPos, last.EndPos)
	return nameFrom(tok)
}

// isNameSegmentToken reports whether tok may EXTEND a qualified name after a ".".
//
// This is deliberately narrower than isMemberNameToken. A reserved member name must not be
// absorbed into a name, because those names begin constructs rather than continue paths:
// absorbing the ".match" of `x.match(co.pattern.Type)` would hide the match chain from the
// postfix parser, and the same applies to `.do`, `.loop`, `.otherwise`, `.each`, `.contains`
// and the other chain verbs of section 11a.
//
// In practice this loop rarely runs at all, because token folding has already collapsed a
// dotted path into one token before the parser sees it. A "." that survives folding is a
// member access, and member accesses belong to the postfix chain.
func isNameSegmentToken(tok scanlex.Token) bool {
	return tok.IsOneOfMany(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
}

// isMemberNameToken reports whether tok may appear after "." as a member name.
//
// The set is wider than "identifier" because folding assigns reserved member
// names their own kinds, and because the control-flow verbs of section 11a
// (`.do`, `.loop`, `.otherwise`, `.each`, `.contains`, `.match`, `.return`) reach
// the parser as whichever kind folding chose for them.
func (p *parser) isMemberNameToken(tok scanlex.Token) bool {
	return tok.IsOneOfMany(
		scanlex.IDENTIFIER,
		scanlex.COMPOSITE_IDENTIFER,
		scanlex.BUILT_IN_METHOD,
		scanlex.KEYWORD,
		scanlex.CONTEXT_KEYWORD,
		scanlex.RESERVEDWORD,
	)
}

// parseLifecycleName parses the lifecycle-name production:
//
//	lifecycle-name = "@@", identifier
//
// These are the class lifecycle methods, `@@new` and `@@init`
// (docs/language-ref.md, "The @@new and @@init Methods").
func (p *parser) parseLifecycleName() name {
	at := p.expect(scanlex.DOUBLE_AT, "to begin a lifecycle method name")
	id := p.parseIdentifier("after \"@@\"")
	tok := scanlex.NewUniqueToken(scanlex.DOUBLE_AT, "@@"+id.Scanned, at.StartPos, id.Tok.EndPos)
	return nameFrom(tok)
}

// atLifecycleName reports whether the cursor begins a lifecycle-name.
func (p *parser) atLifecycleName() bool {
	return p.at(scanlex.DOUBLE_AT) && p.isMemberNameToken(p.peek(1))
}

// parseFunctionName parses the function-name production:
//
//	function-name = identifier | lifecycle-name
func (p *parser) parseFunctionName(context string) name {
	if p.atLifecycleName() {
		return p.parseLifecycleName()
	}
	return p.parseIdentifier(context)
}

// parseSpecialBinding parses the special-binding production:
//
//	special-binding = result-binding | self-binding
//	self-binding    = "$"
//	result-binding  = "$", digit, { digit }
//
// A bare "$" is the self-referential let binding; "$1", "$2", … capture the
// previous result in a "=>>" delegation chain. The scanner emits either form as a
// single BIND_VAR token, so the numeric suffix is decoded here.
func (p *parser) parseSpecialBinding() ast.Expr {
	tok := p.expect(scanlex.BIND_VAR, "to begin a bind variable")
	index := -1 // -1 marks the bare "$" self-binding.
	if digits := strings.TrimPrefix(tok.Value, "$"); digits != "" {
		n, err := strconv.Atoi(digits)
		if err != nil {
			p.failf(tok, "malformed bind variable %q; expected \"$\" or \"$\" followed by digits", tok.Value)
		}
		index = n
	}
	return ast.BindVariableExpr{
		Name:  tok.Value,
		Index: index,
		Symb:  p.varSymbol(tok.Value, "co.lang.infer"),
	}
}

// parseWildcard parses the wildcard production, the contextual "_" token.
//
// It is admitted only where a production spells "_" directly: pattern matching,
// containment tests and iterator bindings (docs/language-ref.md, "Discard /
// Wildcard Variable").
func (p *parser) parseWildcard() ast.Expr {
	tok := p.expect(scanlex.DISCARD_WILD_VAR, "as a wildcard")
	return ast.SymbolExpr{
		Value:       tok.Value,
		SymbolType_: "wildcard",
		Symb:        p.exprSymbol(tok.Value),
	}
}

// parseDeclarationReference parses the declaration-reference production:
//
//	declaration-reference        = qualified-function-reference | qualified-name
//	qualified-function-reference = qualified-name, "(", [ type-list ], ")",
//	                               return-type-clause
//
// The function form is what lets an annotation name a specific overload, as in
// the target sets of docs/language-ref.md, "Declaration-Reference Syntax". It is
// tried speculatively because only the return-type clause after the parameter
// types distinguishes it from a plain name followed by an unrelated group.
func (p *parser) parseDeclarationReference(context string) ast.Expr {
	qn := p.parseQualifiedName(context)

	ref := ast.SymbolExpr{
		Value:       qn.Scanned,
		SymbolType_: "declaration-reference",
		Symb:        p.exprSymbol(qn.Scanned),
	}

	if !p.at(scanlex.OPEN_PAREN) {
		return ref
	}

	var signature ast.Type
	matched := p.speculate(func() bool {
		p.advance() // "("
		var params []ast.Type
		if !p.at(scanlex.CLOSE_PAREN) {
			params = p.parseTypeList()
		}
		if !p.accept(scanlex.CLOSE_PAREN) {
			return false
		}
		if !p.at(scanlex.ARROW) {
			return false
		}
		results := p.parseReturnTypeClause()
		signature = p.functionTypeNode(qn.Scanned, params, results)
		return true
	})
	if !matched {
		return ref
	}
	return ast.SDTExpr{Type_: signature, Symb: p.exprSymbol(qn.Scanned)}
}
