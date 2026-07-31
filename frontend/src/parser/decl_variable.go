package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
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
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}

	next := p.peek(1)
	switch next.Kind {
	case scanlex.BUILT_IN_TYPE:
		return true
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

// parseVariableDeclaration parses the variable-declaration production.
func (p *parser) parseVariableDeclaration(annotations annotationSet) ast.Stmt {
	declarators := []ast.Stmt{p.parseTypedVariableDeclarator(annotations)}

	for p.accept(scanlex.COMMA) {
		// A comma list may mix declarators with plain assignments to names
		// already in scope, which the reference corpus relies on:
		//
		//	c co.lang.string, a = 20, b = 30;
		if p.atTypedVariableDeclaration() {
			declarators = append(declarators, p.parseTypedVariableDeclarator(annotations))
			continue
		}
		declarators = append(declarators, ast.ExpressionStmt{
			Expression: p.parseExpression(),
			Symb:       p.stmtSymbol("assignment"),
		})
	}

	p.statementEnd("a variable declaration")

	return p.oneOrGrouped(declarators, "variable-declaration")
}

// parseCommaStatementTail continues a comma-separated statement whose first item has
// already been parsed as an expression.
//
// This is reached when a statement opens with an assignment rather than with a
// declarator, as in `a = 20, b = 30;`.
func (p *parser) parseCommaStatementTail(first ast.Expr, annotations annotationSet) ast.Stmt {
	items := []ast.Stmt{
		ast.ExpressionStmt{Expression: first, Symb: p.stmtSymbol("assignment")},
	}

	for p.accept(scanlex.COMMA) {
		if p.atTypedVariableDeclaration() {
			items = append(items, p.parseTypedVariableDeclarator(annotations))
			continue
		}
		items = append(items, ast.ExpressionStmt{
			Expression: p.parseExpression(),
			Symb:       p.stmtSymbol("assignment"),
		})
	}

	p.statementEnd("a comma-separated statement")
	return p.oneOrGrouped(items, "comma-statement")
}

// oneOrGrouped returns a single statement directly, or groups several into a block so
// that a comma list remains one statement as DECISION-SYN-002 requires.
func (p *parser) oneOrGrouped(items []ast.Stmt, label string) ast.Stmt {
	if len(items) == 1 {
		return items[0]
	}
	return &ast.BlockStmt{Body: items, Symb: p.blockSymbol(label, false)}
}

// parseTypedVariableDeclarator parses the typed-variable-declarator production:
//
//	typed-variable-declarator = identifier, type-expression, [ "=", expression ]
func (p *parser) parseTypedVariableDeclarator(annotations annotationSet) ast.Stmt {
	declName := p.parseDeclarationName("as a variable name")
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
	return p.parseExpression()
}

// lowerDeclarator maps a declarator onto the AST statement node its type derivation
// selects.
//
// This is the single place that knows the correspondence between a derivation and a
// declaration node, so a new derivation form needs a change here and nowhere else.
func (p *parser) lowerDeclarator(declName name, t typeRef, value ast.Expr, annotations annotationSet) ast.Stmt {
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
		return ast.PointerVariableDeclStmt{BasicVarStmt: basic, Kind_: pointerKindOf(t.Attrs), Symb: symb}

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
		return ast.ArrayVariableDeclStmt{
			BasicVarStmt: basic,
			Dimensions:   len(t.Dims),
			Sizes:        t.Dims,
			Symb:         symb,
		}

	case formSlice:
		symb := p.arraySymbol(declName.Scanned, t.actType())
		symb.IsSlice = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.SliceVariableDeclStmt{BasicVarStmt: basic, Symb: symb}

	case formReference:
		symb := p.referenceSymbol(declName.Scanned, t.actType())
		symb.Count = t.RefCount
		symb.Ref = t.RefCount == 1
		symb.Lref = t.RefCount == 2
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.RefVariableDeclStmt{BasicVarStmt: basic, Symb: symb}

	case formHeapReference:
		symb := p.referenceSymbol(declName.Scanned, t.actType())
		symb.Heap = true
		symb.Count = t.RefCount
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.HeapAllocatedRefStmt{BasicVarStmt: basic, Symb: symb}

	case formAddress:
		symb := p.addressSymbol(declName.Scanned, t.actType())
		symb.Addressop = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.AddressVariableDeclStmt{BasicVarStmt: basic, Symb: symb}

	case formThunk:
		symb := p.thunkSymbol(declName.Scanned, t.actType())
		symb.ThunkVar = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.ThunkVariableDeclStmt{BasicVarStmt: basic, Symb: symb}

	case formRange:
		symb := p.rangeSymbol(declName.Scanned, t.actType())
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.RangeVariableDeclStmt{BasicVarStmt: basic, Symb: symb}

	case formWord:
		// An attribute-only derivation on co.lang.word is the address-manipulation
		// form: co.lang.word->(repr=intptr).
		symb := p.addressSymbol(declName.Scanned, t.actType())
		symb.Wordtype = true
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		return ast.AddressVariableDeclStmt{BasicVarStmt: basic, Symb: symb}

	default:
		symb := p.varSymbol(declName.Scanned, t.actType())
		symb.HasInitValue = initialized
		symb.ExplicitType = true
		symb.IsCompound = t.Form == formUnion
		symb.Discard = declName.isWildcard()
		return ast.VarDeclarationStmt{BasicVarStmt: basic, Symb: symb}
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
