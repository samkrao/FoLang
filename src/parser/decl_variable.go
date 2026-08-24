package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// variable-declaration and its declarators — section 5.
//
//	variable-declaration      = annotations, typed-variable-declarator,
//	                            { ",", typed-variable-declarator }, statement-end
//	typed-variable-declarator = identifier, type-expression, [ "=", expression ]
//
// DECISION-SYN-002 makes a comma-separated list of declarators ONE statement, so a
// single ";" terminates them all:
//
//	x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true;
//
// The type's derivation decides which AST node a declarator becomes. This compiler
// has no derived-type node: a pointer declaration produces an
// ast.PointerVariableDeclStmt whose element type is the base type, an array
// declaration produces an ast.ArrayVariableDeclStmt carrying its dimensions, and so
// on. lowerDeclarator below is that mapping, and it is the reason the type parser
// returns a typeRef rather than a bare ast.Type.

// atTypedVariableDeclaration reports whether the cursor begins a
// typed-variable-declarator.
//
// The shape is `identifier type-expression`, which needs care because a bare
// identifier also begins an expression statement and a function declaration. The
// decision is made on the token after the name: a built-in type or a name that could
// itself start a type means this is a declaration, while a "(" means a function or a
// call and an operator means an expression.
func (p *parser) atTypedVariableDeclaration() bool {
	if !p.atIdentifier() {
		return false
	}

	next := p.peek(1)
	switch next.Kind {
	case scanlex.OPEN_PAREN:
		// type-atom admits a parenthesized type list, so a declarator's type may be a
		// bare function type: `functionType (co.lang.int)->(co.lang.string);`. That
		// shares its prefix with a call, `compute(x);`, and only the "->" after the
		// balanced group tells them apart. A local function declaration has the same
		// prefix again but ends in a block, and parseStatement tests for it first.
		return p.lookaheadOnly(func() bool {
			p.advance() // the name
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			return p.at(scanlex.ARROW)
		})
	case scanlex.BUILT_IN_TYPE:
		return true
	case scanlex.BUILT_IN_KIND:
		// A few co.lang names intentionally inhabit both levels. In executable
		// declaration position the type reading wins: a binding that receives a
		// co.lang.dependentType result is itself declared with that type, and
		// co.lang.value/nothing/data are likewise usable type names.
		return isTypeFirstKind(next.Value)
	case scanlex.BUIL_IN_STMT_EXPRS:
		// A co.* type that is not in the built-in type table, such as co.lang.map,
		// folds down to its namespace and is still a valid type name.
		return true
	case scanlex.KEYWORD, scanlex.RESERVEDWORD:
		return next.Value == "forall"
	case scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER:
		// `name Type` is a declaration with a user-defined type. Two juxtaposed names
		// have no other reading in FoLang: there is no application-by-juxtaposition, so
		// the second name can only be a type.
		//
		// A "(" after the type does NOT make it a call. It is a type-argument list, so
		// `items Vector(co.lang.int) = …` declares a variable of an applied generic type
		// (type-postfix-expression, section 4). Every `name (` form that really is a
		// function — a local function declaration and a bare function-pattern clause —
		// is dispatched by parseStatement BEFORE this predicate is reached, so nothing
		// is left here for a "(" to disambiguate. A closure declaration puts its "="
		// before the parameter lists, so it never reaches this predicate either.
		return true
	}
	return false
}

// isTypeFirstKind reports whether an overlapping BUILT_IN_KIND token must be read as
// a type when it appears in a declarator. Dedicated type-declaration kinds are type
// level by definition; the scanner's built-in type registry supplies the remaining
// overlaps without duplicating that list in the parser.
func isTypeFirstKind(kind string) bool {
	if _, ok := typeDeclarationKinds[kind]; ok {
		return true
	}
	for _, builtin := range scanlex.Builtin_types {
		if builtin == kind {
			return true
		}
	}
	return false
}

// parseVariableDeclaration parses the variable-declaration production.
//
// Implements: variable-declaration
func (p *parser) parseVariableDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declarators := []ast.Stmt{p.parseTypedVariableDeclarator(annotations)}

	for p.accept(scanlex.COMMA) {
		if !p.atTypedVariableDeclaration() {
			p.fail(p.cur(), "a typed declaration list may contain only typed declarators; move assignments or inferred declarations to a separate statement")
		}
		declarators = append(declarators, p.parseTypedVariableDeclarator(annotations))
	}

	p.statementEnd("a variable declaration")

	return p.oneOrGrouped(declarators, "variable-declaration")
}

// oneOrGrouped returns a single statement directly, or groups several into a block so
// that a comma list remains one statement as DECISION-SYN-002 requires.
func (p *parser) oneOrGrouped(items []ast.Stmt, label string) ast.Stmt {
	spanStart := p.pos
	if len(items) == 1 {
		return items[0]
	}
	return &ast.BlockStmt{Span: p.spanFrom(spanStart), Body: items, Symb: p.blockSymbol(label, false)}
}

// parseTypedVariableDeclarator parses the typed-variable-declarator production:
//
//	typed-variable-declarator = identifier, type-expression, [ "=", expression ]
//
// Implements: typed-variable-declarator
func (p *parser) parseTypedVariableDeclarator(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseIdentifier("as a variable name")
	t := p.parseTypeExpression()

	var value ast.Expr
	if p.acceptOp("=") {
		value = p.parseVariableInitializer()
	}

	return p.lowerDeclarator(declName, t, value, annotations)
}

// parseVariableInitializer parses the expression on the right of "=" in a declarator.
//
// A braced group here is an expression, not a body: DECISION-SYN-006's
// expression-brace rule means `cfg co.lang.map = { "a": 1 };` still needs its ";".
// The one exception is an anonymous function used as a direct inline body, which
// ends at its own brace, and that is what a function-kind declaration relies on.
func (p *parser) parseVariableInitializer() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseExpression()
}

// lowerDeclarator maps a declarator onto the AST statement node its type derivation
// selects, and binds the declared name into the symbol table.
//
// Every typed variable form reaches this function, fields included, so binding here
// rather than in each branch is what makes "a declarator declares its name" true by
// construction: a derivation added to declaratorNode cannot forget to bind.
func (p *parser) lowerDeclarator(declName name, t typeRef, value ast.Expr, annotations annotationSet) ast.Stmt {
	decl, symb := p.declaratorNode(declName, t, value, annotations)
	p.declareNamed(declName, symb)
	return decl
}

// declaratorNode builds the node for a declarator and returns it with the symbol
// record its caller must bind.
//
// This is the single place that knows the correspondence between a derivation and a
// declaration node, so a new derivation form needs a change here and nowhere else.
func (p *parser) declaratorNode(declName name, t typeRef, value ast.Expr, annotations annotationSet) (ast.Stmt, declarable) {
	spanStart := p.pos
	basic := ast.BasicVarStmt{
		Identifier:    declName.Scanned,
		AssignedValue: value,
		Type_:         t.Node,
		VarType:       t.actType(),
		SDapst:        annotations.list(),
	}
	initialized := value != nil

	switch t.Form {
	case formPointer:
		symb := p.pointerSymbol(declName.Scanned, t.actType())
		symb.Count = t.PointerCount
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		applyPointerAttributes(symb, t.Attrs)
		return ast.PointerVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Kind_: pointerKindOf(t.Attrs), Symb: symb}, symb

	case formArray:
		symb := p.arraySymbol(declName.Scanned, t.actType())
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		symb.IsMultiDimesion = len(t.Dims) > 1
		symb.IsJagged = t.DimGroups > 1
		symb.VLA = t.VariableLength
		symb.IsZeroDim = t.ZeroDim
		symb.IsZeroLen = isZeroLengthArray(t)
		symb.SizeFromInit = allDimensionsElided(t) && initialized
		return ast.ArrayVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic,
			Dimensions: len(t.Dims),
			Sizes:      t.Dims,
			Symb:       symb,
		}, symb

	case formSlice:
		symb := p.arraySymbol(declName.Scanned, t.actType())
		symb.IsSlice = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.SliceVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	case formReference:
		symb := p.referenceSymbol(declName.Scanned, t.actType())
		symb.Count = t.RefCount
		symb.Ref = t.RefCount == 1
		symb.Lref = t.RefCount == 2
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.RefVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	case formHeapReference:
		symb := p.referenceSymbol(declName.Scanned, t.actType())
		symb.Heap = true
		symb.Count = t.RefCount
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.HeapAllocatedRefStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	case formAddress:
		symb := p.addressSymbol(declName.Scanned, t.actType())
		symb.Addressop = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.AddressVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	case formThunk:
		symb := p.thunkSymbol(declName.Scanned, t.actType())
		symb.ThunkVar = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.ThunkVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	case formRange:
		symb := p.rangeSymbol(declName.Scanned, t.actType())
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.RangeVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	case formWord:
		// An attribute-only derivation on co.lang.word is the address-manipulation
		// form: co.lang.word->(repr=intptr).
		symb := p.addressSymbol(declName.Scanned, t.actType())
		symb.Wordtype = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.AddressVariableDeclStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb

	default:
		symb := p.varSymbol(declName.Scanned, t.actType())
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		symb.IsCompound = t.Form == formUnion
		symb.Discard = declName.isWildcard()
		return ast.VarDeclarationStmt{Span: p.spanFrom(spanStart), BasicVarStmt: basic, Symb: symb}, symb
	}
}

// isZeroLengthArray reports whether an array declaration is the ->([0]) zero-length
// form.
func isZeroLengthArray(t typeRef) bool {
	if len(t.Dims) != 1 || t.Dims[0] == nil {
		return false
	}
	lit, ok := t.Dims[0].(ast.IntegerLiteral)
	return ok && lit.Value == 0
}

// allDimensionsElided reports whether every dimension was omitted, as in ->([]) and
// ->([,]). Such a declaration takes its sizes from its initializer.
func allDimensionsElided(t typeRef) bool {
	if len(t.Dims) == 0 {
		return false
	}
	for _, d := range t.Dims {
		if d != nil {
			return false
		}
	}
	return true
}

// pointerKindOf reads the `kind=` attribute of a fat-pointer derivation, as in
// `co.lang.int->(*, kind=region, meta={})` (docs/language-ref.md, "Fat Pointers").
func pointerKindOf(attrs map[string]any) string {
	if kind, ok := attrs["kind"]; ok {
		if s, isString := kind.(string); isString {
			return s
		}
	}
	return ""
}

// applyPointerAttributes records a fat pointer's attribute list on its symbol.
//
// The attributes carry the pointer's kind and metadata — region, len, cap, vtab and
// so on — which is what distinguishes a fat pointer from a thin one
// (docs/language-ref.md, "Fat Pointers"). The shared-, weak- and unique-ownership
// kinds are also recognised here, since they are spelled as kinds rather than as
// separate derivations.
func applyPointerAttributes(symb *symboltable.PointerSymbol, attrs map[string]any) {
	if len(attrs) == 0 {
		symb.IsRaw = true
		return
	}
	symb.MetaData = attrs
	symb.ISFatPointer = true

	switch pointerKindOf(attrs) {
	case "sptr":
		symb.IsShared = true
	case "uptr":
		symb.IsUnique = true
	case "weak":
		symb.IsWeak = true
	}
}

// extern-variable-declaration — section 5.
//
//	extern-variable-declaration = "@co.dap.declare", "(", "extern", ")",
//	                              identifier, type-expression, statement-end
//
// This declares that a variable of the named type exists but is defined elsewhere
// (docs/language-ref.md, "Variables extern declaration"):
//
//	_ co.lang.unit = {
//	    @co.dap.declare(extern)
//	    someBool co.lang.bool;
//	}
//
// The annotation is part of the SYNTAX here rather than decoration on an ordinary
// declaration, which is what the reference means by "for functions and types
// @co.dap.declare is optional; for variables it is required". Without it the same
// tokens are a variable-declaration, and a unit body admits no such member — so the
// annotation is what makes this shape a unit member at all.
//
// The declarator takes no initializer. An extern name is bound outside this unit, so
// a value written here would have nothing to initialise.

// externDeclareAnnotation is the annotation that introduces an
// extern-variable-declaration, together with the one argument it takes.
const (
	externDeclareAnnotation = "@co.dap.declare"
	externDeclareArgument   = "extern"
)

// atExternVariableDeclaration reports whether the already-parsed annotations and the
// cursor together begin an extern-variable-declaration.
//
// The annotation alone is not enough: `@co.dap.declare(extern)` also introduces the
// external TYPE declaration of a class body, which is a field-shaped member. What
// selects this production is the annotation plus a typed declarator.
func (p *parser) atExternVariableDeclaration(annotations annotationSet) bool {
	if annotations.optionString(externDeclareAnnotation, "0") != externDeclareArgument {
		return false
	}
	return p.atTypedVariableDeclaration()
}

// parseExternVariableDeclaration parses the extern-variable-declaration production.
//
// Implements: extern-variable-declaration
func (p *parser) parseExternVariableDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseIdentifier("as an extern variable name")
	t := p.parseTypeExpression()

	// An extern declaration names a binding defined elsewhere, so it has no
	// initializer of its own.
	if p.atOp("=") {
		p.fail(p.cur(), "an extern variable is defined elsewhere and takes no initializer; drop the \"=\" or drop @co.dap.declare(extern)")
	}
	p.statementEnd("an extern variable declaration")

	decl := p.lowerDeclarator(declName, t, nil, annotations)
	if plain, ok := decl.(ast.VarDeclarationStmt); ok {
		plain.Symb.Extern = true
		return plain
	}
	return decl
}
