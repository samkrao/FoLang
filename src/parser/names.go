package parser

import (
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
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
	// FromFilename marks a name the underscore form derived from the source
	// filename. Only such a name carries an unresolved kind suffix, and only it may
	// be rewritten once the declaration's kind is known.
	FromFilename bool
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
//
// The contextual spelling is included for the reason the grammar gives: it is
// are NOT hard reserved words. `hard-reserved-word` is co, let, this, for and fo;
// `contextual-keyword` is a separate production that the `token` rule does not
// list, so `forall` is an identifier token that the parser reclassifies only
// where its contextual form holds. `self` is now simply an ordinary identifier.
func (p *parser) atIdentifier() bool {
	return p.atAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) ||
		p.atContextualIdentifier()
}

// atContextualIdentifier reports whether the cursor holds a contextual-keyword
// spelling, which is an ordinary identifier wherever its contextual form does not
// apply.
//
// Implements: contextual-keyword
func (p *parser) atContextualIdentifier() bool {
	return p.atAny(scanlex.KEYWORD, scanlex.CONTEXT_KEYWORD) &&
		contextualKeywords[p.lexeme()]
}

// contextualKeywords is the contextual-keyword production: the spellings the
// lexer leaves available as identifiers.
//
// Implements: contextual-keyword
var contextualKeywords = map[string]bool{
	"forall": true,
}

// parseIdentifier consumes one identifier.
//
// DECISION-LEX-001 and DECISION-LEX-006 are enforced by the scanner: an
// identifier starts with an ASCII letter, never contains consecutive
// underscores and never ends in one, and a lone "_" is a separate contextual
// token that is never an identifier.
//
// Implements: identifier
func (p *parser) parseIdentifier(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.atIdentifier() {
		p.failf(p.cur(), "expected an identifier %s, found %s", context, describeToken(p.cur()))
	}

	// A dotted name is folded into one COMPOSITE_IDENTIFER token, which atIdentifier
	// accepts because a qualified name occupies the same grammatical slot as a plain
	// one. This production is the SINGLE identifier, though: a field name, a parameter
	// name and a declaration name are each spelled `identifier`, so `foo.bar` there is
	// a qualified name in a position that has no room for one.
	if p.at(scanlex.COMPOSITE_IDENTIFER) {
		p.failf(p.cur(), "expected a single identifier %s, found the qualified name %q", context, logicalName(p.lexeme()))
	}

	return nameFrom(p.advance())
}

// parseFilenameDerivedName parses the filename-derived-name production:
//
//	filename-derived-name = "_"
//
// Revision 23 removed the identifier alternative this slot used to have. A
// file-backed primary declaration takes its public name from the filename and
// nothing else, so the head must spell "_" (docs/language-ref.md, "File-Backed
// Primary Declarations"):
//
//	// Employee.fol
//	_ co.lang.struct = { id co.lang.int; }
//
// Six declaration forms are stated exceptions and keep an explicit identifier in
// the head, because filename derivation cannot express what they need
// (DECISION-FILE-003). They call parseIdentifier directly and never reach here:
// surface-struct-declaration and surface-cstruct-declaration, because one
// library surface file carries several declarations; data-declaration, because
// the head names the variants; parameterized-type-declaration and
// entry-parameterized-type-declaration, because a filename cannot carry a
// generic parameter list; and entry-simple-type-declaration, because the entry
// file is not file-backed.
//
// Implements: filename-derived-name
func (p *parser) parseFilenameDerivedName(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.at(scanlex.DISCARD_WILD_VAR) {
		// Naming the declaration explicitly is the mistake the reference calls
		// out by name, so it is worth reporting as itself rather than as a
		// missing "_".
		if p.atIdentifier() {
			p.failf(p.cur(),
				"%s takes its name from the filename, so its declaration name must be written \"_\"; found %q",
				context, logicalName(p.lexeme()))
		}
		p.failf(p.cur(), "expected \"_\" as the declaration name of %s, found %s", context, describeToken(p.cur()))
	}

	tok := p.advance()
	return p.filenameDerivedName(tok, context)
}

// filenameDerivedName resolves the name the source filename supplies.
//
// The classification has already stripped the structural suffix and normalized
// the component to UpperCamelCase (DECISION-FILE-002), so all that remains is to
// report the cases in which nothing usable can be derived and to apply the
// backend lowering the scanner performs for ordinary source identifiers.
func (p *parser) filenameDerivedName(tok scanlex.Token, context string) name {
	derived := p.file.Source.DerivedName

	switch {
	case p.file.Source.Class == sourceClassUnknown:
		p.reportf(tok,
			"%s derives its name from the source filename, but %q is not a recognized FoLang source filename",
			context, p.file.Basename)
	case !p.file.Source.Valid:
		p.reportf(tok,
			"filename component %q is not a valid FoLang filename identifier, so %s has no derivable name; "+
				"a component begins with an ASCII letter and contains only ASCII letters, digits and isolated internal underscores",
			p.file.Source.Component, context)
	}

	if derived == "" {
		// Keep the wildcard spelling so downstream nodes still carry a name and
		// one diagnostic is not multiplied into a cascade.
		return name{Scanned: tok.Value, Logical: tok.Value, Tok: tok, FromFilename: true}
	}
	return name{
		Scanned:      derived + foLoweringSuffix,
		Logical:      derived,
		Tok:          tok,
		FromFilename: true,
	}
}

// parseQualifiedName parses the qualified-name production:
//
//	qualified-name = ( identifier | "co" ), { ".", identifier }
//
// Folded tokens already contain most dots, so the loop only runs for segments
// folding left separate. METHOD_CALL and BUILT_IN_METHOD are accepted here
// because non-expression contexts such as qualified constructor patterns and
// declaration references still use the qualified-name grammar even when their
// final segment happens to precede "(".
//
// Implements: qualified-name
func (p *parser) parseQualifiedName(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseQualifiedNameWith(context, p.isMemberNameToken)
}

// parseExpressionQualifiedName parses the non-call prefix of a name expression.
// Invoked members deliberately stop before METHOD_CALL or BUILT_IN_METHOD so the
// postfix parser can preserve the receiver/member boundary in ast.MemberExpr.
func (p *parser) parseExpressionQualifiedName(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseQualifiedNameWith(context, isNameSegmentToken)
}

// parseQualifiedTypeName documents a qualified-name use in type position. Type
// and other non-expression contexts use the broad qualified-name parser because
// they have no postfix method-call boundary to preserve. For example,
// `co.lang.map` may arrive as BUIL_IN_STMT_EXPRS("co.lang"), DOT,
// BUILT_IN_METHOD("map") and must be rejoined as one type name.
func (p *parser) parseQualifiedTypeName(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseQualifiedName(context)
}

// parseQualifiedNameWith parses a qualified-name whose continuation segments are accepted by
// extends.
func (p *parser) parseQualifiedNameWith(context string, extends func(scanlex.Token) bool) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	head := p.cur()
	switch {
	case p.atIdentifier(),
		p.at(scanlex.BUILT_IN_TYPE),
		p.at(scanlex.BUILT_IN_KIND),
		p.at(scanlex.BUILT_IN_COLLECTIONS),
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
// This is deliberately narrower than isMemberNameToken. METHOD_CALL and a reserved member
// name must not be absorbed into a name: METHOD_CALL marks the final ordinary member before
// an argument list, while reserved names begin constructs rather than continue paths:
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
// names their own kinds, ordinary invoked members use METHOD_CALL so they cannot
// be reabsorbed into a qualified SymbolExpr, and because the control-flow verbs
// of section 11a (`.do`, `.loop`, `.otherwise`, `.each`, `.contains`, `.match`,
// `.return`) reach the parser as whichever kind folding chose for them.
func (p *parser) isMemberNameToken(tok scanlex.Token) bool {
	return tok.IsOneOfMany(
		scanlex.IDENTIFIER,
		scanlex.COMPOSITE_IDENTIFER,
		scanlex.METHOD_CALL,
		scanlex.BUILT_IN_METHOD,
		scanlex.KEYWORD,
		scanlex.CONTEXT_KEYWORD,
		scanlex.RESERVEDWORD,
	)
}

// parseLifecycleName parses the lifecycle-declaration-name production:
//
//	lifecycle-declaration-name = "@@new" | "@@init"
//
// These are the DECLARATION spellings of the compiler-owned class lifecycle
// family (docs/language-ref.md, "Special lifecycle members"). Revision 24 split
// the former single `lifecycle-name` in two, because a lifecycle member is now
// declared and invoked under different spellings:
//
//	@@new   declaration          new   invocation, through "::"
//	@@init  declaration          init  invocation, through "::"
//
// lifecycleInvocationNames below is the other half of that split.
//
// The scanner resolves the whole spelling against the closed Special_methods set and
// emits ONE SPECIAL_METHODS token, the same way it resolves a built-in method name from
// a table. So there is nothing to assemble here: a name that is not a special method
// never reaches the parser as "@@" plus an identifier.
//
// Implements: lifecycle-declaration-name
func (p *parser) parseLifecycleName() name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.expect(scanlex.SPECIAL_METHODS, "as a special method name")
	return nameFrom(tok)
}

// atLifecycleName reports whether the cursor begins a lifecycle-declaration-name.
func (p *parser) atLifecycleName() bool {
	return p.at(scanlex.SPECIAL_METHODS)
}

// lifecycleInvocationNames is the lifecycle-invocation-name production:
//
//	lifecycle-invocation-name = "new" | "init"
//
// It maps each invocation spelling to the declaration spelling it selects, which
// is the correspondence the reference tabulates:
//
//	new  -> @@new
//	init -> @@init
//
// The set is CLOSED, and closing it here is what makes `value::whatever(…)` a
// parse error naming the lifecycle family rather than a silently accepted call
// to a member that cannot exist. A future language-defined lifecycle name
// extends both this table and the declaration spellings together.
//
// These are ordinary identifiers everywhere else: a method named `init` reached
// through `.` is unrelated to the lifecycle member reached through `::`, which is
// exactly why the invocation needs its own marker.
//
// Implements: lifecycle-invocation-name
var lifecycleInvocationNames = map[string]string{
	"new":  "@@new",
	"init": "@@init",
}

// parseLabelIdentifier parses the label-identifier production in either of its
// two roles:
//
//	label-identifier  = single-quote, identifier, label-identifier-guard
//	label-declaration = label-identifier
//	label-reference   = label-identifier
//
// The two roles share one lexical form and differ only by position — a
// declaration is followed by ":" — so one function serves both and the CALLER
// records which it read. The scanner has already applied label-identifier-guard
// by preferring a complete character literal, so `'c'` never arrives here.
//
// The returned name keeps the leading apostrophe: labels occupy their own
// namespace, and `'outer` must not be confusable with the ordinary identifier
// `outer` in anything that later compares the two.
//
// Implements: label-identifier
// Implements: label-declaration
// Implements: label-reference
// Implements: label-identifier-guard
func (p *parser) parseLabelIdentifier(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.expect(scanlex.LABEL_IDENTIFIER, context)
	return nameFrom(tok)
}

// atLabelIdentifier reports whether the cursor is a label-identifier.
func (p *parser) atLabelIdentifier() bool {
	return p.at(scanlex.LABEL_IDENTIFIER)
}

// parseFunctionName parses an ordinary declared function name.
//
// Lifecycle spellings are intentionally not accepted here. `@@new` and
// `@@init` are compiler-owned class lifecycle methods and are consumed only by
// parseLifecycleMethodDeclaration after parseClassMember has established the
// class-body context. Letting this shared helper consume them would also admit
// package, unit, local, interface and general-kind functions with lifecycle
// names.
//
// Implements: function-name
func (p *parser) parseFunctionName(context string) name {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.atLifecycleName() {
		p.failNamedf(p.cur(), helpers.DiagnosticInvalidLifecycleDeclaration, "Invalid Lifecycle Declaration", "%q is a class lifecycle method and cannot be declared %s", p.lexeme(), context)
	}
	return p.parseIdentifier(context)
}

// parseSpecialBinding parses the special-binding production:
//
//	special-binding = result-binding | recursive-binding
//	recursive-binding = "$"
//	result-binding  = "$", digit, { digit }
//
// A bare "$" is the self-referential let binding; "$1", "$2", … capture the
// previous result in a "=>>" delegation chain. The scanner emits either form as a
// single BIND_VAR token, so the numeric suffix is decoded here.
//
// Implements: special-binding
// Implements: recursive-binding
// Implements: result-binding
func (p *parser) parseSpecialBinding() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.expect(scanlex.BIND_VAR, "to begin a bind variable")
	index := -1 // -1 marks the bare "$" self-binding.
	if digits := strings.TrimPrefix(tok.Value, "$"); digits != "" {
		n, err := strconv.Atoi(digits)
		if err != nil {
			p.failf(tok, "malformed bind variable %q; expected \"$\" or \"$\" followed by digits", tok.Value)
		}
		index = n
	}
	return ast.BindVariableExpr{NodeName: "BindVariableExpr", Span: p.spanFrom(spanStart), Name: tok.Value,
		Index: index,
		Symb:  p.varSymbol(tok.Value, "co.lang.infer"),
	}
}

// parseWildcard parses the wildcard production, the contextual "_" token.
//
// It is admitted only where a production spells "_" directly: pattern matching,
// containment tests and iterator bindings (docs/language-ref.md, "Discard /
// Wildcard Variable").
//
// Implements: wildcard
func (p *parser) parseWildcard() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.expect(scanlex.DISCARD_WILD_VAR, "as a wildcard")
	return ast.SymbolExpr{NodeName: "SymbolExpr", Span: p.spanFrom(spanStart), Value: tok.Value,
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
//
// Implements: declaration-reference
// Implements: qualified-function-reference
func (p *parser) parseDeclarationReference(context string) ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	qn := p.parseQualifiedName(context)

	ref := ast.SymbolExpr{NodeName: "SymbolExpr", Span: p.spanFrom(spanStart), Value: qn.Scanned,
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
	return ast.SDTExpr{NodeName: "SDTExpr", Span: p.spanFrom(spanStart), Type_: signature, Symb: p.exprSymbol(qn.Scanned)}
}
