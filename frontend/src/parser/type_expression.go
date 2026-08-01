package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Type syntax — section 4 of docs/grammar/folang.ebnf.
//
// A type is parsed into a typeRef rather than straight into an ast.Type because
// declarations and nested type slots lower derivations differently. A variable
// declaration records its OUTERMOST derivation in a specialised statement node:
// `p co.lang.int->(*)` becomes an ast.PointerVariableDeclStmt whose element type
// is co.lang.int. A nested type slot has no declaration statement to carry that
// information, so fullType wraps the element in ast.DerivedType instead. typeRef
// retains both views until the caller selects the representation appropriate to
// its AST slot.

// typeForm names the outermost derivation applied to a type. It is what selects
// the declaration node for a variable of that type.
type typeForm int

const (
	// formPlain is an undecorated type: co.lang.int, Employee, Vector(T).
	formPlain         typeForm = iota
	formPointer                // ->(*), ->(**)
	formArray                  // ->([5]), ->([2,3]), ->([2][3]), ->([...]), ->([.])
	formReference              // ->(&), ->(&&)
	formHeapReference          // ->(~)
	formAddress                // ->(@)
	formThunk                  // ->(^)
	formSlice                  // ->([:])
	formRange                  // ->(..)
	formFunction               // (T, T)->(T)
	formUnion                  // A | B
	formForall                 // forall(T).T
	formWord                   // ->(repr=…), ->(sign=…) and other bare attribute tails
)

// typeRef is a parsed type expression.
//
// Node is always populated and is what goes into the AST wherever an ast.Type is
// required. The remaining fields describe the derivation so that a declaration
// can be lowered to the matching statement node and symbol flags.
type typeRef struct {
	// Node is the lowered AST type. For a derived type it is the underlying
	// element type, because that is what the declaration nodes expect.
	Node ast.Type
	// Form is the outermost derivation.
	Form typeForm

	// PointerCount is the number of "*" in a pointer derivation.
	PointerCount int
	// RefCount is 1 for "&" and 2 for "&&".
	RefCount int

	// Dims holds the array dimensions of the first "[...]" group. A nil element
	// is an elided dimension, which DECISION-TYP-003 permits in any position, so
	// both ->([,]) and ->([]) are well formed.
	Dims []ast.Expr
	// DimGroups counts "[...]" groups: more than one is a jagged array.
	DimGroups int
	// AllDims holds EVERY dimension group in source order, which is what a jagged
	// array needs to describe its shape: ->([2][3]) is {{2}, {3}}. Dims is its first
	// entry and DimGroups its length.
	AllDims [][]ast.Expr
	// VariableLength records the ->([...]) variable-length form.
	VariableLength bool
	// ZeroDim records the ->([.]) zero-dimension form.
	ZeroDim bool

	// Attrs holds a derivation's trailing attribute list, decoded the same way
	// annotation arguments are. DECISION-TYP-001 allows one on every derivation
	// form, which is what admits co.lang.int->(&, meta={type=out}) and
	// co.lang.word->(repr=intptr).
	Attrs map[string]any

	// Params and Results describe a function type.
	Params  []ast.Parameter
	Results []ast.Returns

	// TypeParams holds the quantified variables of a forall type.
	TypeParams []symboltable.GenericTypeParam

	// Tok is the first token of the type, used for diagnostics.
	Tok scanlex.Token

	// arrowTailConsumed records that this syntactic level has already consumed
	// the optional arrow-type-tail. Parenthesized parameter lists are completed
	// inside parseTypeAtom, so without this bit `(T)->(&)->(*)` could incorrectly
	// acquire a second ungrouped tail when control returned to
	// parseArrowTypeExpression.
	arrowTailConsumed bool
}

// actType returns the type's canonical name as used for symbol bookkeeping.
func (t typeRef) actType() string {
	if t.Node == nil {
		return "co.lang.infer"
	}
	return typeNameOf(t.Node)
}

// typeNameOf returns the NAME a type is known by.
//
// GetActType returns a pair whose two halves mean different things per node:
// SymbolTypeNode reports (name, symbol-category), BuiltInDataType reports (name, name).
// Reading the second half — as this function's caller used to — collapsed every
// user-defined type to the category literal "Type" and every applied generic to "CDT",
// which is what then went into symbol metadata and declaration nodes. The first half is
// the name, so that is what is read, and the cases where even that is a placeholder are
// unwrapped to the type actually being named.
func typeNameOf(node ast.Type) string {
	switch node := node.(type) {
	case nil:
		return ""
	case ast.SymbolTypeNode:
		return node.Value
	case ast.BuiltInDataType:
		return node.Value
	case ast.DerivedType:
		// A derivation is named by the type it derives from; Form is what
		// distinguishes them, and it is recorded on the node itself.
		return typeNameOf(node.Underlying)
	case ast.CompoundType:
		// A type application is named by its constructor, so Vector(co.lang.int) is
		// a Vector. A union has no single name and keeps the compound placeholder.
		if node.Op == "apply" {
			return typeNameOf(node.Left)
		}
	case ast.GenericType:
		return typeNameOf(node.Type_)
	}

	actual, declared := node.GetActType()
	if actual != "" {
		return actual
	}
	if declared != "" {
		return declared
	}
	return node.GetName()
}

// derivationForms maps a parsed derivation onto the AST's DerivationForm.
//
// formPlain, formFunction, formUnion and formForall are absent on purpose: each already
// has a node of its own that carries everything about it, so wrapping them would add a
// layer without adding information.
var derivationForms = map[typeForm]ast.DerivationForm{
	formPointer:       ast.DerivePointer,
	formArray:         ast.DeriveArray,
	formReference:     ast.DeriveReference,
	formHeapReference: ast.DeriveHeapReference,
	formAddress:       ast.DeriveAddress,
	formThunk:         ast.DeriveThunk,
	formSlice:         ast.DeriveSlice,
	formRange:         ast.DeriveRange,
	formWord:          ast.DeriveWord,
}

// fullType returns the type as an AST node INCLUDING its derivation.
//
// Node alone is the element type, which is what the declaration lowering wants: a
// pointer variable becomes a PointerVariableDeclStmt whose type is the pointee. Every
// other position that admits a type — a parameter, a result, a type alias, a function
// type's components — has no statement node to record the derivation on, so it must
// travel with the type itself or be lost.
//
// An undecorated type is returned unwrapped, so nothing changes for the overwhelmingly
// common case and only a genuinely derived type gains a level.
func (t typeRef) fullType() ast.Type {
	form, derived := derivationForms[t.Form]
	if !derived || t.Node == nil {
		return t.Node
	}

	return ast.DerivedType{
		Underlying:     t.Node,
		Form:           form,
		PointerCount:   t.PointerCount,
		RefCount:       t.RefCount,
		DimGroups:      t.AllDims,
		VariableLength: t.VariableLength,
		ZeroDim:        t.ZeroDim,
		Attrs:          t.Attrs,
	}
}

// parseTypeExpression parses the type-expression production:
//
//	type-expression = forall-type | union-type-expression
func (p *parser) parseTypeExpression() typeRef {
	defer p.enter()()

	if p.atKeyword("forall") {
		return p.parseForallType()
	}
	return p.parseUnionTypeExpression()
}

// parseForallType parses the forall-type production:
//
//	forall-type = "forall", "(", type-parameter-list, ")", ".", type-expression
//
// This is the rank-N polymorphic type, used both as a declaration prefix and in
// parameter position, as in `f forall(T) (T)->(T)`.
func (p *parser) parseForallType() typeRef {
	start := p.expectKeyword("forall", "to begin a forall type")
	p.expect(scanlex.OPEN_PAREN, "to open the type-parameter list of a forall type")
	params := p.parseTypeParameterList()
	p.expect(scanlex.CLOSE_PAREN, "to close the type-parameter list of a forall type")
	p.expect(scanlex.DOT, "after the type-parameter list of a forall type")

	inner := p.parseTypeExpression()

	return typeRef{
		Node: ast.ForAllType{
			TypeParams: params,
			// The quantified body is a nested type slot. It has no declaration
			// statement on which a derivation could be recorded, so preserve the
			// complete type rather than only its element node.
			Inner: inner.fullType(),
			Symb:  p.typeSymbol("forall"),
		},
		Form:       formForall,
		TypeParams: params,
		Tok:        start,
	}
}

// parseTypeParameterList parses the type-parameter-list production:
//
//	type-parameter-list = identifier, { ",", identifier }
func (p *parser) parseTypeParameterList() []symboltable.GenericTypeParam {
	params := []symboltable.GenericTypeParam{
		{Name: p.parseIdentifier("as a type parameter").Scanned},
	}
	for p.accept(scanlex.COMMA) {
		params = append(params, symboltable.GenericTypeParam{
			Name: p.parseIdentifier("as a type parameter").Scanned,
		})
	}
	return params
}

// parseUnionTypeExpression parses the union-type-expression production:
//
//	union-type-expression = arrow-type-expression, { "|", arrow-type-expression }
//
// This is the algebraic sum type of docs/language-ref.md, "Type Declarations":
// `y co.lang.type = co.lang.int | co.lang.char`. The "|" here is the union
// operator, not bitwise OR; the two are distinguished by position, since the
// value-expression Pratt loop is never entered while a type is being parsed.
func (p *parser) parseUnionTypeExpression() typeRef {
	left := p.parseArrowTypeExpression()

	// Inside a lambda's parameter list the "|" is the lambda's closing delimiter, not the
	// union operator, so `|x co.lang.int| => x*x` must not read the "|" as continuing the
	// parameter's type.
	if p.lambdaParamDepth > 0 {
		return left
	}

	if !p.atOp("|") {
		return left
	}

	// Each union arm is a complete type-expression. A derivation belongs to that
	// arm, not to the union declaration itself, so it must be embedded in the arm.
	node := left.fullType()
	for p.atOp("|") {
		opTok := p.advance()
		right := p.parseArrowTypeExpression()
		node = ast.CompoundType{
			Left:  node,
			Op:    opTok.Value,
			Right: right.fullType(),
			Symb:  p.typeSymbol("union"),
		}
	}
	return typeRef{Node: node, Form: formUnion, Tok: left.Tok}
}

// parseArrowTypeExpression parses the arrow-type-expression production:
//
//	arrow-type-expression = type-postfix-expression, [ "->", arrow-type-tail ]
//
// The "->" carries two unrelated meanings that only the tail distinguishes:
//
//	co.lang.int->(*)                 a derivation applied to co.lang.int
//	(co.lang.int)->(co.lang.int)     a function type from int to int
//
// parseArrowTypeTail resolves which. The EBNF admits one direct arrow tail. A
// nested derivation remains expressible by grouping its base, for example
// `(co.lang.int->(*))->(&)`, while the ungrouped chain
// `co.lang.int->(*)->(&)` is rejected rather than silently losing one layer.
func (p *parser) parseArrowTypeExpression() typeRef {
	head := p.parseTypePostfixExpression()

	if !p.at(scanlex.ARROW) {
		return head
	}
	if head.arrowTailConsumed {
		p.fail(p.cur(), "a type expression admits only one direct arrow tail; parenthesize the completed type before applying another derivation")
	}
	p.advance()
	head = p.parseArrowTypeTail(head)
	head.arrowTailConsumed = true

	// `arrow-type-expression` has an optional SINGLE arrow tail. A bare
	// type-expression tail may recursively consume its own arrow, but a second
	// arrow left here is an ungrouped chain outside the production.
	if p.at(scanlex.ARROW) {
		p.fail(p.cur(), "a type expression admits only one direct arrow tail; parenthesize the derived base, as in `(T->(*))->(&)`, to apply another derivation")
	}
	return head
}

// parseArrowTypeTail parses the arrow-type-tail production:
//
//	arrow-type-tail = type-derivation
//	                | parenthesized-type-list
//	                | type-expression
//
// base is the type-postfix-expression already parsed to the left of the "->".
//
// The three alternatives are separated without backtracking:
//
//   - A "(" whose first token starts a derivation-specification — "*", "&",
//     "&&", "~", "@", "^", "[:]", "..", "[" or an `attribute=` pair — is a
//     type-derivation.
//   - Any other "(" is a parenthesized-type-list, which makes this a function
//     type whose results it lists.
//   - Anything else is a bare type-expression tail.
func (p *parser) parseArrowTypeTail(base typeRef) typeRef {
	if p.at(scanlex.OPEN_PAREN) {
		if p.startsDerivationSpecification() {
			return p.parseTypeDerivation(base)
		}
		results := p.parseParenthesizedReturnList()
		baseType := base.fullType()
		return typeRef{
			Node:    p.functionTypeNode(baseType.GetName(), []ast.Type{baseType}, results),
			Form:    formFunction,
			Params:  parametersFromTypes(p, []ast.Type{baseType}),
			Results: results,
			Tok:     base.Tok,
		}
	}

	// A bare tail: the arrow's right side is itself a type expression.
	tail := p.parseTypeExpression()
	baseType := base.fullType()
	tailType := tail.fullType()
	results := p.returnsFromTypes([]ast.Type{tailType})
	return typeRef{
		Node:    p.functionTypeNode(baseType.GetName(), []ast.Type{baseType}, results),
		Form:    formFunction,
		Params:  parametersFromTypes(p, []ast.Type{baseType}),
		Results: results,
		Tok:     base.Tok,
	}
}

// parseTypePostfixExpression parses the type-postfix-expression production:
//
//	type-postfix-expression = type-atom, { type-argument-list }
//
// A type-argument-list applies a type constructor to arguments, as in
// `Vector(co.lang.int)` or the higher-kinded `F(A)`. Repeated lists apply
// left-to-right, so `F(A)(B)` is `(F applied to A) applied to B`.
func (p *parser) parseTypePostfixExpression() typeRef {
	atom := p.parseTypeAtom()

	for p.at(scanlex.OPEN_PAREN) && !p.startsDerivationSpecification() {
		if atom.arrowTailConsumed {
			p.fail(p.cur(), "a type-argument list cannot follow a completed arrow type without grouping it")
		}
		args := p.parseTypeArgumentList()
		for _, arg := range args {
			// Type application is recorded as a compound type with the "apply"
			// operator, because the AST has no dedicated application node.
			atom.Node = ast.CompoundType{
				Left:  atom.Node,
				Op:    "apply",
				Right: arg,
				Symb:  p.typeSymbol(atom.Node.GetName()),
			}
		}
	}
	return atom
}

// parseTypeAtom parses the type-atom production:
//
//	type-atom = qualified-name
//	          | "(", type-expression, ")"
//	          | "(", type-list, ")"
//
// The two parenthesised alternatives differ only by whether a comma appears, so
// they are parsed as one list. A single-element list is a grouped type; a longer
// one is the parameter list of a function type and is kept as such so that a
// following "->" can consume it.
func (p *parser) parseTypeAtom() typeRef {
	start := p.cur()

	if p.at(scanlex.OPEN_PAREN) {
		p.advance()

		// "()" is an empty parameter list, which only makes sense before "->".
		if p.at(scanlex.CLOSE_PAREN) {
			p.advance()
			return p.finishParenthesizedTypeAtom(nil, start)
		}

		items := p.parseParenthesizedTypeItems()
		p.expect(scanlex.CLOSE_PAREN, "to close a parenthesized type")
		return p.finishParenthesizedTypeAtom(items, start)
	}

	return p.parseNamedTypeAtom()
}

// parseParenthesizedTypeItems parses the contents of a parenthesised type group.
//
// The grammar spells this as a type-list, whose entries are bare types, but the reference
// writes names in the same position when the group is a function type's parameter list
// (docs/language-ref.md, "Other ways to declare closures/function objects"):
//
//	someFArg co.lang.type = (co.lang.int, co.lang.int)->(co.lang.int)   unnamed
//	funtype  co.lang.type = (a co.lang.int, b co.lang.int)->(co.lang.int)   named
//
// Both are accepted, so each item is `[ identifier ] type-expression` and the name is kept
// when present. A name is only taken as a name when a type follows it, which is the same
// test parseReturnItem uses.
func (p *parser) parseParenthesizedTypeItems() []ast.Parameter {
	items := []ast.Parameter{p.parseFunctionTypeParameter()}
	for p.accept(scanlex.COMMA) {
		if p.at(scanlex.CLOSE_PAREN) {
			break // trailing comma
		}
		items = append(items, p.parseFunctionTypeParameter())
	}
	return items
}

// finishParenthesizedTypeAtom decides what a parenthesised type group means.
//
// When a "->" follows, the group is the parameter list of a function type and the return
// clause completes it. Otherwise a single unnamed type is a grouped type, and anything else
// has no meaning on its own and is reported.
//
// The return clause has the two shapes arrow-type-tail allows after a parameter list:
//
//	f (A)->(B)     parenthesized-type-list, one or more results
//	f (A)->B       type-expression, a single unparenthesized result
//
// The second is what the typeclass signatures in the reference use — `map(value F(A),
// f (A)->B)->(F(B))` — where parenthesising the inner result would only add noise.
func (p *parser) finishParenthesizedTypeAtom(items []ast.Parameter, start scanlex.Token) typeRef {
	if p.at(scanlex.ARROW) {
		p.advance()

		// Ordered choice in arrow-type-tail puts type-derivation before a
		// parenthesized result list. Consequently `(T)->(&)` derives a reference
		// from the grouped T; it is not a function with an invalid `&` result.
		// Grouping is also the EBNF-compliant way to apply another derivation to
		// an already-derived type: `(T->(*))->(&)`.
		if p.startsDerivationSpecification() {
			if len(items) != 1 || items[0].Name_ != "" {
				p.fail(start, "a type derivation after a parenthesized type requires exactly one unnamed base type")
			}
			base := typeRef{Node: items[0].Type_, Form: formPlain, Tok: start}
			derived := p.parseTypeDerivation(base)
			derived.arrowTailConsumed = true
			return derived
		}

		results := p.parseArrowTypeResults()
		return typeRef{
			Node: ast.FunctionType{
				Params:  [][]ast.Parameter{items},
				Results: results,
				Symb:    p.typeSymbol("co.lang.function"),
			},
			Form:              formFunction,
			Params:            items,
			Results:           results,
			Tok:               start,
			arrowTailConsumed: true,
		}
	}

	if len(items) == 1 && items[0].Name_ == "" {
		return typeRef{Node: items[0].Type_, Form: formPlain, Tok: start}
	}
	if len(items) == 0 {
		p.fail(start, "an empty type list \"()\" is only valid as the parameter list of a function type, so it must be followed by \"->\"")
	}
	p.fail(start, "a parenthesized parameter list is only valid as part of a function type, so it must be followed by \"->\"")
	return typeRef{}
}

// parseArrowTypeResults parses the results of a function type after its "->".
//
//	arrow-type-tail = type-derivation
//	                | parenthesized-type-list
//	                | type-expression
//
// A "(" opens the parenthesized-type-list, which is the multi-result form and the one
// every declaration uses. Anything else is the bare type-expression alternative, a
// single unparenthesized result.
func (p *parser) parseArrowTypeResults() []ast.Returns {
	if p.at(scanlex.OPEN_PAREN) {
		return p.parseParenthesizedReturnList()
	}
	tail := p.parseTypeExpression()
	return p.returnsFromTypes([]ast.Type{tail.fullType()})
}

// parseNamedTypeAtom parses the qualified-name alternative of type-atom.
//
// Built-in data types arrive from the scanner as a single BUILT_IN_TYPE token and
// become ast.BuiltInDataType; every other name becomes ast.SymbolTypeNode, which
// the semantic phase resolves to a user-defined type or a type parameter.
func (p *parser) parseNamedTypeAtom() typeRef {
	tok := p.cur()

	if p.at(scanlex.BUILT_IN_TYPE) {
		p.advance()
		return typeRef{
			Node: ast.BuiltInDataType{
				Value:      tok.Value,
				Type:       tok.Value,
				SymbolType: string(symboltable.S_TypeSymbol),
				Symb:       p.typeSymbol(tok.Value),
			},
			Form: formPlain,
			Tok:  tok,
		}
	}

	qn := p.parseQualifiedTypeName("as a type")
	return typeRef{
		Node: ast.SymbolTypeNode{
			Value:      qn.Scanned,
			SymbolType: string(symboltable.S_TypeSymbol),
			Symb:       p.typeSymbol(qn.Scanned),
		},
		Form: formPlain,
		Tok:  qn.Tok,
	}
}

// parseTypeArgumentList parses the type-argument-list production:
//
//	type-argument-list     = "(", [ type-or-value-argument,
//	                              { ",", type-or-value-argument } ], ")"
//	type-or-value-argument = type-expression | dependent-index
//
// DECISION-TYP-002: where a token sequence satisfies both readings, the
// type-expression reading is selected. The value reading exists for dependent
// types, where an argument is a length or other index, as in
// `co.lang.int->([n])`.
func (p *parser) parseTypeArgumentList() []ast.Type {
	p.expect(scanlex.OPEN_PAREN, "to open a type-argument list")

	var args []ast.Type
	if !p.at(scanlex.CLOSE_PAREN) {
		args = append(args, p.parseTypeOrValueArgument())
		for p.accept(scanlex.COMMA) {
			args = append(args, p.parseTypeOrValueArgument())
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a type-argument list")
	return args
}

// parseTypeOrValueArgument parses one type-or-value-argument, preferring the type
// reading per DECISION-TYP-002 and falling back to a dependent index.
func (p *parser) parseTypeOrValueArgument() ast.Type {
	// A "_" here is a generic-arity slot, the same one generic-parameter-clause
	// admits in declaration position. It stands for an unnamed slot of a
	// higher-kinded constructor, so `Transformer(F(_), G(_))` names the shape of F
	// and G rather than passing them a type. It is not an index, and the strict
	// dependent-index reading below must not claim it: the arity slot has no value
	// and cannot be compared, so it never reaches DECISION-TYP-005.
	if p.at(scanlex.DISCARD_WILD_VAR) {
		p.advance()
		return ast.BuiltInDataType{
			Value:      "co.lang.infer",
			Type:       "co.lang.infer",
			SymbolType: string(symboltable.S_TypeSymbol),
			Symb:       p.typeSymbol("co.lang.infer"),
		}
	}

	var result ast.Type
	if p.speculate(func() bool {
		t := p.parseTypeExpression()
		// The type reading is only accepted if it consumed the whole argument.
		if !p.atAny(scanlex.COMMA, scanlex.CLOSE_PAREN) {
			return false
		}
		// A type ARGUMENT is another slot with no declaration to record a derivation
		// on, so the argument of Vector(co.lang.int->(*)) keeps its pointer here.
		result = t.fullType()
		return true
	}) {
		return result
	}

	// A value argument: wrap the index as a dependent type, which is precisely
	// what a type parameterised by a value is.
	value := p.parseDependentIndex("a dependent-type argument", scanlex.COMMA, scanlex.CLOSE_PAREN)
	return ast.DependentType{
		Base: ast.BuiltInDataType{
			Value:      "co.lang.dependentType",
			Type:       "co.lang.dependentType",
			SymbolType: string(symboltable.S_TypeSymbol),
			Symb:       p.typeSymbol("co.lang.dependentType"),
		},
		Expr: value,
		Symb: p.typeSymbol("co.lang.dependentType"),
	}
}

// parseDependentIndex parses the dependent-index production:
//
//	dependent-index = integer-literal | qualified-name
//
// DECISION-TYP-004: an index is an INDEX position, not a general expression. Only an
// integer literal or a name is admissible; arithmetic, calls and index expressions are
// rejected here. That restriction is what keeps dependent-type equality decidable by
// inspection rather than by a solver (DECISION-TYP-005), and it is why the same
// production serves both dependent-type arguments and array dimensions — admitting
// arithmetic in a dimension would reintroduce it behind the dependent type.
//
//	v Vector(3);                    literal
//	v Vector(SIZE);                 name, resolved by the checker
//	buf co.lang.int->([SIZE]);      the same rule for array sizes
//
//	v Vector(n + 1);                rejected: arithmetic
//	buf co.lang.int->([n * 2]);     rejected: arithmetic
//
// No prefix operator is reachable from this production, which is what makes a LITERAL
// index non-negative by construction: "-1" is a parse error positioned at the "-".
// A NAMED index that resolves to a negative constant is a semantic error, because it
// can only be detected after @co.dap.const substitution.
//
// The restriction applies to the SIZE of an array, never to element access: `buf[i + 1]`
// is an ordinary index expression and goes through parseExpression as usual.
//
// terminators are the tokens that legitimately end the index in the calling context —
// "," and ")" in a type-argument list, "," and "]" in an array dimension. Anything else
// is an operator the author tried to use, and naming it is what makes the diagnostic
// actionable.
func (p *parser) parseDependentIndex(context string, terminators ...scanlex.TokenKind) ast.Expr {
	// A prefix operator here is almost always a negative literal, which is the one
	// case worth naming outright.
	if _, isPrefix := prefixOperators[p.lexeme()]; isPrefix {
		if p.lexeme() == "-" {
			p.failf(p.cur(), "a dependent index may not be negative, so %q cannot appear in %s; an index is a non-negative integer literal or a name", "-", context)
		}
		p.failf(p.cur(), "the prefix operator %q is not permitted in %s; an index is an integer literal or a name", p.lexeme(), context)
	}

	var index ast.Expr
	switch {
	case p.at(scanlex.NUMBER):
		tok := p.cur()
		// dependent-index admits integer-literal only. A float is not an index.
		if isFloatingLexeme(tok.Value) {
			p.failf(tok, "%q is not a valid index in %s; an index is an integer literal or a name", tok.Value, context)
		}
		index = p.parseNumericLiteral()

	case p.atIdentifier(), p.at(scanlex.BUIL_IN_STMT_EXPRS), p.at(scanlex.BUILT_IN_TYPE):
		qn := p.parseQualifiedTypeName("as an index")
		index = ast.SymbolExpr{
			Value:       qn.Scanned,
			SymbolType_: "reference",
			Symb:        p.exprSymbol(qn.Scanned),
		}

	default:
		p.failf(p.cur(), "expected an integer literal or a name as %s, found %s", context, describeToken(p.cur()))
	}

	// Anything other than a terminator means the author wrote an expression. The
	// grammar rejects that syntactically, so the diagnostic can name the operator.
	if !p.atAny(terminators...) {
		p.rejectDependentIndexTail(context)
	}
	return index
}

// rejectDependentIndexTail reports the operator that continued a dependent index and
// aborts. It is only reached once the index atom itself has parsed, so the cursor is
// on whatever the author tried to combine it with.
func (p *parser) rejectDependentIndexTail(context string) {
	tok := p.cur()
	switch {
	case tok.Kind == scanlex.OPEN_PAREN:
		p.failf(tok, "a call is not permitted in %s; an index is an integer literal or a name", context)
	case tok.Kind == scanlex.OPEN_BRACKET:
		p.failf(tok, "an index expression is not permitted in %s; an index is an integer literal or a name", context)
	case tok.Kind == scanlex.DOT:
		p.failf(tok, "a member access is not permitted in %s; an index is an integer literal or a name", context)
	}
	if _, isPostfix := postfixOperators[tok.Value]; isPostfix {
		p.failf(tok, "the operator %q is not permitted in %s; an index is an integer literal or a name", tok.Value, context)
	}
	if _, isInfix := p.infixOperator(); isInfix {
		p.failf(tok, "arithmetic is not permitted in %s; an index is an integer literal or a name, so write a @co.dap.const constant instead of %q", context, tok.Value)
	}
	p.failf(tok, "unexpected %s in %s; an index is an integer literal or a name", describeToken(tok), context)
}

// parseTypeList parses the type-list production:
//
//	type-list = type-expression, { ",", type-expression }
func (p *parser) parseTypeList() []ast.Type {
	// A type-list is used as a nested payload/signature slot; no declaration
	// statement exists there to carry a derivation, so every item is complete.
	list := []ast.Type{p.parseTypeExpression().fullType()}
	for p.accept(scanlex.COMMA) {
		list = append(list, p.parseTypeExpression().fullType())
	}
	return list
}

// functionTypeNode builds the ast.FunctionType for a function type with the given
// parameter types and results.
func (p *parser) functionTypeNode(name string, params []ast.Type, results []ast.Returns) ast.Type {
	return ast.FunctionType{
		Params:  [][]ast.Parameter{parametersFromTypes(p, params)},
		Results: results,
		Symb:    p.typeSymbol(name),
	}
}

// parametersFromTypes turns a bare type list into positional parameters. A
// function type names no parameters, so each carries only its type and is marked
// OnlyType.
func parametersFromTypes(p *parser, types []ast.Type) []ast.Parameter {
	params := make([]ast.Parameter, 0, len(types))
	for _, t := range types {
		params = append(params, ast.Parameter{
			SymbolDeclStmt: p.declFor("", actTypeOf(t), t),
			Type_:          t,
			OnlyType:       true,
			WhatType:       "param",
			Symb:           p.genericSymbol("", symboltable.S_VariableDetails, actTypeOf(t)),
		})
	}
	return params
}

// returnsFromTypes turns a bare type list into unnamed results.
func (p *parser) returnsFromTypes(types []ast.Type) []ast.Returns {
	results := make([]ast.Returns, 0, len(types))
	for _, t := range types {
		results = append(results, ast.Returns{
			SymbolDeclStmt: p.declFor("", actTypeOf(t), t),
			Type_:          t,
			OnlyType:       true,
			WhatType:       "result",
			Symb:           p.genericSymbol("", symboltable.S_VariableDetails, actTypeOf(t)),
		})
	}
	return results
}

// actTypeOf returns a type node's canonical type name, tolerating a nil node.
func actTypeOf(t ast.Type) string {
	if t == nil {
		return "co.lang.infer"
	}
	// GetActType's second value is a broad category for several user-defined
	// nodes ("Type", "CDT"), not the source type's canonical name.
	return typeNameOf(t)
}

// atKeyword reports whether the cursor holds the given hard reserved word.
// Reserved words arrive as KEYWORD or RESERVEDWORD tokens, so the comparison is
// on the lexeme.
func (p *parser) atKeyword(word string) bool {
	return p.atAny(scanlex.KEYWORD, scanlex.RESERVEDWORD, scanlex.CONTEXT_KEYWORD) &&
		p.lexeme() == word
}

// expectKeyword consumes the given hard reserved word or reports a diagnostic.
func (p *parser) expectKeyword(word string, context string) scanlex.Token {
	if p.atKeyword(word) {
		return p.advance()
	}
	p.failf(p.cur(), "expected %q %s, found %s", word, context, describeToken(p.cur()))
	return eofToken // unreachable: failf panics
}
