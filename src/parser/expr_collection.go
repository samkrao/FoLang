package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// typed-collection-literal — section 11 of docs/grammar/folang.ebnf.
//
//	typed-collection-literal =
//	      type-postfix-expression,
//	      ( array-literal | map-literal | call-suffix ),
//	      typed-collection-literal-guard
//	    | type-postfix-expression, "->", parenthesized-type-list,
//	      ( array-literal | map-literal | call-suffix )
//
// A built-in collection VALUE names its collection type and then supplies the
// literal body that type takes (docs/language-ref.md, "Canonical Object and
// Collection Construction"):
//
//	StringList co.lang.type = co.core.List(co.lang.string);
//	IntSet co.lang.type = co.core.Set(co.lang.int);
//	StringIntMap co.lang.type = co.core.Map(co.lang.string, co.lang.int);
//
//	x   := StringList["A","B","C"];
//	y   := IntSet(1,2,3);
//	map := StringIntMap{"A":1,"B":2};
//
// Those are the only two current-alpha forms: either the surrounding typed
// declaration already supplies the generic arguments and the constructor does not
// repeat them, or the declaration is type-deduced and the constructor supplies
// them explicitly. There is no third inference form.
//
// The BODY FORM is fixed by the collection rather than chosen by the writer —
// List takes "[…]", Map takes "{…}", Set takes "(…)" — which is why a mismatched
// body is a diagnostic here and not an alternative reading.
//
// # Why a guard rather than a syntactic distinction
//
// Without an arrow tail the three spellings collide with three ordinary
// expressions on the same token span: `Type[…]` is also an index-suffix,
// `Type(…)` is also a call-suffix, and `Type{…}` is also object-construction.
// Nothing in the token stream separates them, so the reading is chosen by what
// the PREFIX NAMES, which is what typed-collection-literal-guard decides. After
// an explicit arrow tail the overlap is gone and the body is always a collection
// body.
//
// The braced case additionally overlaps with object-construction, and the
// reference resolves it by shape: a non-empty braced body whose every entry has
// the `identifier ":" expression` object-field-initializer shape is object
// construction. A map body's keys are arbitrary expressions — string literals in
// every example the reference writes — so the two do not in practice compete.

// builtinCollectionTypeNames is the reference's Builtin Collections registry: the
// closed set of built-in names that may stand in a typed-collection-literal
// prefix.
//
// It is a NAME registry consulted by the guard, not a parse path. Which body form
// each name takes is collectionBodyForms below; every other collection semantic
// stays outside the parser.
//
// Implements: builtin-collection-type-name
var builtinCollectionTypeNames = map[string]bool{
	"co.core.List":  true,
	"co.core.Set":   true,
	"co.core.Map":   true,
	"co.core.Tree":  true,
	"co.core.Trie":  true,
	"co.core.Array": true,
	"co.core.Tuple": true,
}

// collectionBodyForm is the one literal body a collection takes.
type collectionBodyForm int

const (
	// collectionBodyUnsupported marks a registered name whose constructor body
	// form the current alpha profile does not define.
	collectionBodyUnsupported collectionBodyForm = iota
	// collectionBodyBracket is `Type[ … ]`.
	collectionBodyBracket
	// collectionBodyBrace is `Type{ … }`.
	collectionBodyBrace
	// collectionBodyParen is `Type( … )`.
	collectionBodyParen
)

// collectionBodyForms fixes each built-in collection's body form.
//
// Only List, Set and Map have current-alpha constructor body forms. Tree, Trie,
// Array and Tuple are RESERVED names: the registry keeps their spellings so
// nothing else can claim them, while their constructor reading stays refused
// until the reference defines those forms. A reserved name used as a constructor
// therefore produces the unsupported-feature diagnostic rather than being read as
// an ordinary index or call.
var collectionBodyForms = map[string]collectionBodyForm{
	"co.core.List":  collectionBodyBracket,
	"co.core.Map":   collectionBodyBrace,
	"co.core.Set":   collectionBodyParen,
	"co.core.Tree":  collectionBodyUnsupported,
	"co.core.Trie":  collectionBodyUnsupported,
	"co.core.Array": collectionBodyUnsupported,
	"co.core.Tuple": collectionBodyUnsupported,
}

// atTypedCollectionLiteral reports whether the cursor begins a
// typed-collection-literal.
//
// This is typed-collection-literal-guard's parser-decidable half: the prefix has
// to NAME a supported collection type rather than denote a value. The built-in
// registry is what the parser can settle on its own. The guard's other two
// sources — a file-local alias bound to a collection by an alias-directive, and a
// user-declared collection type — need name resolution, so a prefix that is
// neither a built-in name nor followed by an explicit type application keeps its ordinary
// index/call/construction reading and the collection interpretation is left to
// the semantic phase.
//
// Implements: typed-collection-literal-guard
func (p *parser) atTypedCollectionLiteral() bool {
	name, width, ok := p.collectionPrefixName()
	if !ok || !builtinCollectionTypeNames[name] {
		return false
	}
	return p.lookaheadOnly(func() bool {
		for i := 0; i < width; i++ {
			p.advance()
		}
		// An explicit type application removes the overlap: whatever body
		// follows a completed `Type( … )` application is a collection body.
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			if p.atAny(scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY, scanlex.OPEN_PAREN) {
				return true
			}
			return collectionBodyForms[name] == collectionBodyParen
		}
		if !p.atAny(scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY, scanlex.OPEN_PAREN) {
			return false
		}
		// A braced body whose entries all have the object-field-initializer shape
		// is object construction, whatever the prefix names.
		if p.at(scanlex.OPEN_CURLY) {
			return !p.looksLikeObjectFieldInitializers()
		}
		return true
	})
}

// looksLikeObjectFieldInitializers reports whether the braced body at the cursor
// has the `identifier ":" …` shape of object-field-initializer.
//
// An EMPTY braced body is not object construction on a recognized braced
// collection — the reference makes it an empty typed collection there, and object
// construction only for every other type — so an empty body reports false and the
// collection reading wins.
func (p *parser) looksLikeObjectFieldInitializers() bool {
	return p.lookaheadOnly(func() bool {
		p.advance() // "{"
		if p.at(scanlex.CLOSE_CURLY) {
			return false
		}
		return p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) &&
			p.peek(1).Kind == scanlex.COLON
	})
}

// looksLikeAliasedMapConstruction recognizes the non-object braced constructor
// shape after a concrete alias. Object fields start with `identifier :`; map
// entries may use any expression key. Alias/category validation remains semantic.
func (p *parser) looksLikeAliasedMapConstruction() bool {
	return p.peek(1).Kind == scanlex.OPEN_CURLY && p.lookaheadOnly(func() bool {
		p.advance()
		if !p.at(scanlex.OPEN_CURLY) {
			return false
		}
		p.advance()
		if p.at(scanlex.CLOSE_CURLY) {
			return false
		}
		if p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) &&
			(p.peek(1).Kind == scanlex.COLON || p.peek(1).Value == "=") {
			return false
		}
		return true
	})
}

func (p *parser) parseAliasedMapConstruction() ast.Expr {
	spanStart := p.pos
	name := p.lexeme()
	t := p.parseNamedTypeAtom()
	elements := p.parseCollectionBody(collectionBodyBrace)
	return ast.NewExpr{NodeName: "NewExpr", Span: p.spanFrom(spanStart), Instantiation: ast.CallExpr{
		NodeName: "CallExpr", Span: p.spanFrom(spanStart),
		Method:    ast.SDTExpr{NodeName: "SDTExpr", Span: p.spanFrom(spanStart), Type_: t.fullType(), Symb: p.exprSymbol(name)},
		Arguments: elements, SymbolType_: "typed-collection-literal", Symb: p.exprSymbol(name),
	}, Symb: p.exprSymbol(name)}
}

// collectionPrefixName returns the logical collection name at the cursor and how
// many tokens it spans.
//
// The name arrives in two token shapes, because the scanner only splits a dotted
// built-in path when a "(" follows the member. `co.core.List[` keeps the whole
// path in one token, while `co.core.Set(` becomes `co.core`, a DOT and the
// member — so the Set constructor and the List constructor look nothing alike in
// the token stream despite being the same production.
func (p *parser) collectionPrefixName() (string, int, bool) {
	if !p.atAny(scanlex.BUIL_IN_STMT_EXPRS, scanlex.BUILT_IN_COLLECTIONS, scanlex.BUILT_IN_TYPE) {
		return "", 0, false
	}
	whole := p.lexeme()
	if builtinCollectionTypeNames[whole] {
		return whole, 1, true
	}
	if p.peek(1).Kind == scanlex.DOT && p.isMemberNameToken(p.peek(2)) {
		split := whole + "." + logicalName(p.peek(2).Value)
		if builtinCollectionTypeNames[split] {
			return split, 3, true
		}
	}
	return "", 0, false
}

// parseTypedCollectionLiteral parses the typed-collection-literal production.
//
// Implements: typed-collection-literal
func (p *parser) parseTypedCollectionLiteral() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	name, width, _ := p.collectionPrefixName()
	prefixTok := p.cur()
	if collectionBodyForms[name] == collectionBodyUnsupported {
		p.failNamedf(prefixTok, helpers.DiagnosticUnsupportedFeature, "Unsupported Feature", "%s is a reserved built-in collection name whose constructor body form is not defined in the current alpha profile", name)
	}
	p.failf(prefixTok, "%s is an unspecialized generic collection type and cannot construct a value directly; declare a concrete co.lang.type alias and construct through that alias", name)
	for i := 0; i < width; i++ {
		p.advance()
	}

	// The optional parenthesized application carries the collection's generic arguments.
	var typeArgs []ast.Returns
	if p.at(scanlex.OPEN_PAREN) && p.lookaheadOnly(func() bool {
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		return p.atAny(scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY, scanlex.OPEN_PAREN)
	}) {
		typeArgs = p.parseCollectionTypeArguments()
	}

	form := collectionBodyForms[name]
	if form == collectionBodyUnsupported {
		p.reportUnsupported(prefixTok, name+" is a reserved built-in collection name whose constructor body form is not defined in the current alpha profile")
		panic(bailout{})
	}
	p.expectCollectionBody(prefixTok, name, form)

	elements := p.parseCollectionBody(form)

	return ast.NewExpr{NodeName: "NewExpr", Span: p.spanFrom(spanStart), Instantiation: ast.CallExpr{NodeName: "CallExpr", Span: p.spanFrom(spanStart),
		Method: ast.SDTExpr{NodeName: "SDTExpr", Span: p.spanFrom(spanStart), Type_: p.collectionType(name, typeArgs, spanStart),
			Symb: p.exprSymbol(name),
		},
		Arguments:   elements,
		SymbolType_: "typed-collection-literal",
		Symb:        p.exprSymbol(name),
	},
		Symb: p.exprSymbol(name),
	}
}

// parseCollectionTypeArguments parses a collection constructor's parenthesized
// type application.
//
// Generic arguments are positional and bind in declaration order.
//
//	co.core.Map(co.lang.string, co.lang.int)             positional
//
// Named and space-bound argument forms are rejected by the general type parser.
//
// The positional spelling uses parenthesized-type-list; `name=value` remains a
// metadata or derivation-attribute spelling and is not a generic argument.
func (p *parser) parseCollectionTypeArguments() []ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseParenthesizedReturnList()
}

// expectCollectionBody reports a body form the collection does not take.
//
// The form is fixed by the collection, so writing `co.core.List{…}` is not a
// second way to build a list; it is a different production applied to the wrong
// type. Naming both the collection and the form it does take is what makes the
// diagnostic actionable.
func (p *parser) expectCollectionBody(prefixTok scanlex.Token, name string, form collectionBodyForm) {
	got := collectionBodyUnsupported
	switch {
	case p.at(scanlex.OPEN_BRACKET):
		got = collectionBodyBracket
	case p.at(scanlex.OPEN_CURLY):
		got = collectionBodyBrace
	case p.at(scanlex.OPEN_PAREN):
		got = collectionBodyParen
	}
	if got == form {
		return
	}
	p.failf(prefixTok, "%s takes the %s constructor body; the body form is fixed by the collection", name, collectionBodyDescriptions[form])
}

// collectionBodyDescriptions renders a body form for a diagnostic.
var collectionBodyDescriptions = map[collectionBodyForm]string{
	collectionBodyBracket: `"[ … ]"`,
	collectionBodyBrace:   `"{ … }"`,
	collectionBodyParen:   `"( … )"`,
}

// parseCollectionBody reads the elements of one collection body.
//
// A map body's entries are `key ":" value` pairs and are carried as assignment
// expressions, which is the same shape object-construction uses for its fields;
// the two are told apart by the node's SymbolType_ rather than by their element
// representation.
func (p *parser) parseCollectionBody(form collectionBodyForm) []ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	open, close := scanlex.OPEN_BRACKET, scanlex.CLOSE_BRACKET
	switch form {
	case collectionBodyBrace:
		open, close = scanlex.OPEN_CURLY, scanlex.CLOSE_CURLY
	case collectionBodyParen:
		open, close = scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN
	}

	p.expect(open, "to open a collection body")

	var elements []ast.Expr
	for !p.at(close) && !p.atEOF() {
		spanStart := p.pos
		element := p.parseExpression()

		if form == collectionBodyBrace {
			p.expect(scanlex.COLON, "between a map entry's key and value")
			value := p.parseExpression()
			element = ast.AssignmentExpr{NodeName: "AssignmentExpr", Span: p.spanFrom(spanStart), Assigne: element,
				AssignedValue: value,
				Symb:          p.exprSymbol("map-entry"),
			}
		}
		elements = append(elements, element)

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(close, "to close a collection body")
	return elements
}

// collectionType renders the collection's type.
//
// The generic arguments the constructor supplied are recorded on the node as its
// constraint, which is where GenericType carries an applied argument. They are
// OPTIONAL by design: the reference's two forms differ in whether the surrounding
// typed declaration already supplied them, so a constructor without a tail is not
// an untyped collection but one whose arguments live on the declaration.
func (p *parser) collectionType(name string, typeArgs []ast.Returns, spanStart int) ast.Type {
	base := ast.SymbolTypeNode{NodeName: "SymbolTypeNode", Span: p.spanFrom(spanStart), Value: name,
		SymbolType: string(symboltable.S_TypeSymbol),
		Symb:       p.typeSymbol(name),
	}
	if len(typeArgs) == 0 {
		return base
	}

	applied := ast.Type(base)
	for _, arg := range typeArgs {
		applied = ast.GenericType{NodeName: "GenericType", Span: p.spanFrom(spanStart), Type_: applied,
			Constraint: arg.Type_,
			Symb:       p.typeSymbol(name),
		}
	}
	return applied
}
