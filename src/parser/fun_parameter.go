package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// parameter-list and parameter — section 8.
//
//	parameter-list = "(", [ parameter, { ",", parameter } ], ")"
//	parameter      = [ "..." ], [ "~" ], identifier, [ "?" ],
//	                 [ type-expression ], [ "=", expression ]
//
// The four optional markers each turn on one calling convention
// (docs/language-ref.md, "Functions"):
//
//	fun1(k co.lang.int, b co.lang.char = 10)     default parameter
//	fun1(k co.lang.int, ...b co.lang.char)       variadic
//	fun1(k? co.lang.int)                         optional
//	fun1(~k co.lang.int)                         named
//
// The reference records a restriction the parser does not enforce, because it is a
// property of the whole declaration rather than of one parameter: a curried function
// may not be variadic, and vice versa. Special functions of this kind also cannot be
// overloaded or used as callbacks, all of which the semantic phase checks.

// parseParameterList parses the parameter-list production.
//
// A parameter list takes no trailing comma. The collection literals keep theirs
// — array-literal and the annotation lists still spell `[ "," ]` —
// but parameter-list does not, so a comma here must be followed by another
// parameter rather than closing the list.
//
// Implements: parameter-list
func (p *parser) parseParameterList(allowUntyped bool) []ast.Parameter {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a parameter list")

	var params []ast.Parameter
	for !p.at(scanlex.CLOSE_PAREN) && !p.atEOF() {
		params = append(params, p.parseParameter(allowUntyped))
		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_PAREN) {
			p.fail(p.cur(), "a comma in a parameter list must be followed by another parameter; trailing commas are not allowed")
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a parameter list")
	return params
}

// parseParameterLists parses one or more consecutive parameter lists.
//
// Several lists in a row make the function curried
// (docs/language-ref.md, "Curried"):
//
//	add(first co.lang.int)(second co.lang.int)->(co.lang.int) = { … }
func (p *parser) parseParameterLists(allowUntyped bool) [][]ast.Parameter {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	lists := [][]ast.Parameter{p.parseParameterList(allowUntyped)}
	for p.at(scanlex.OPEN_PAREN) {
		lists = append(lists, p.parseParameterList(allowUntyped))
	}
	return lists
}

// parseParameter parses the parameter production.
//
// Implements: parameter
// Implements: typed-parameter
// Implements: untyped-template-parameter
// Implements: template-parameter-context-guard
func (p *parser) parseParameter(allowUntyped bool) ast.Parameter {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// "..." marks a variadic parameter, which collects the remaining arguments.
	variadic := p.accept(scanlex.DOT_DOT_DOT)

	// "~" marks a named parameter, which callers must pass by name.
	named := p.acceptOp("~")

	paramName := p.parseIdentifier("as a parameter name")

	// "?" marks an optional parameter, which callers may omit.
	optional := p.accept(scanlex.QUESTION)

	// The type is optional in the grammar; it is inferred when absent, which is
	// what the abbreviated closure declarations rely on.
	declaredType := typeRef{Form: formPlain}
	hasType := false
	if (p.startsTypeUse(p.cur()) || p.at(scanlex.OPEN_PAREN) || p.atKeyword("forall")) && !p.atOp("=") {
		declaredType = p.parseTypeUse("as a parameter type")
		hasType = true
	}
	if !hasType && !allowUntyped {
		p.failf(paramName.Tok, "parameter %q requires an explicit type; only a declaration classified by built-in @co.dap.template may omit parameter types", paramName.Logical)
	}

	// A default value makes the parameter a default parameter.
	var defaultValue ast.Expr
	if p.acceptOp("=") {
		defaultValue = p.parseExpression()
	}

	actType := "co.lang.infer"
	if hasType {
		actType = declaredType.actType()
	}

	symb := p.genericSymbol(paramName.Scanned, symboltable.S_VariableDetails, actType)

	// Type_ carries the derivation: a parameter has no statement node to record it
	// on, so `p co.lang.int->(**)` would otherwise arrive as a plain co.lang.int.
	return ast.Parameter{NodeName: "Parameter", Span: p.spanFrom(spanStart), SymbolDeclStmt: p.declFor(paramName.Scanned, actType, declaredType.fullType()),
		Name_:        paramName.Scanned,
		Type_:        declaredType.fullType(),
		Default:      defaultValue,
		HasDefault:   defaultValue != nil,
		DefaultArgs:  defaultValue != nil,
		Optional:     optional,
		OptionalArgs: optional,
		VarArgs:      variadic,
		Variadic:     variadic,
		NamedArgs:    named,
		ThunkArgs:    declaredType.Form == formThunk,
		OnlyType:     false,
		WhatType:     "param",
		Scope:        "param",
		Symb:         symb,
	}
}

// receiver-clause — section 8.
//
//	receiver-clause = "(", ( type-use | identifier, type-use ), ")"
//
// A receiver clause turns a function into a method on the named type. It precedes the
// function name, which is what distinguishes it from the parameter list that follows
// it.

// atReceiverClause reports whether the cursor begins a receiver-clause.
//
// This is a bounded prefix decision. Folang has no `(Type)value` cast form, so
// in a declaration position no competing production has either receiver shape:
//
//	@co.dap.public (emp Employee) label()->(S)    receiver, after an annotation
//	@co.dap.inline(level=2) compute()->(S)        the annotation's own arguments
//	someFn()->(S) = { … } (x)                     a call suffix on a body
//
// Qualified names are folded into one scanner token. Recognition therefore reads
// at most five tokens beyond the current `(`; it neither scans a balanced group
// nor moves and rewinds the parser cursor.
func (p *parser) atReceiverClause() bool {
	if !p.at(scanlex.OPEN_PAREN) {
		return false
	}
	first, second := p.peek(1), p.peek(2)
	if p.startsTypeUse(first) && second.Kind == scanlex.CLOSE_PAREN {
		return receiverFunctionNameToken(p.peek(3)) && p.peek(4).Kind == scanlex.OPEN_PAREN
	}
	if receiverFunctionNameToken(first) && p.startsTypeUse(second) && p.peek(3).Kind == scanlex.CLOSE_PAREN {
		return receiverFunctionNameToken(p.peek(4)) && p.peek(5).Kind == scanlex.OPEN_PAREN
	}
	return false
}

// receiverFunctionNameToken is the cursor-free equivalent of the declaration-name
// check. It keeps receiver recognition a fixed token lookup with no speculation.
func receiverFunctionNameToken(tok scanlex.Token) bool {
	return tok.IsOneOfMany(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER, scanlex.SPECIAL_METHODS) ||
		(tok.IsOneOfMany(scanlex.KEYWORD, scanlex.CONTEXT_KEYWORD) && contextualKeywords[tok.Value])
}

// parseReceiverClause parses the receiver-clause production.
//
// Implements: receiver-clause
func (p *parser) parseReceiverClause() *ast.FunctionReceiver {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a receiver clause")

	// The named form binds the receiver to a name; the bare form gives only its
	// type.
	receiverName := ""
	if p.atIdentifier() && p.namePrecedesType() {
		receiverName = p.parseIdentifier("as a receiver name").Scanned
	}

	t := p.parseTypeUse("as a receiver type")
	p.expect(scanlex.CLOSE_PAREN, "to close a receiver clause")

	symb := p.varSymbol(receiverName, t.actType())
	symb.IsParam = true

	return &ast.FunctionReceiver{NodeName: "FunctionReceiver", Span: p.spanFrom(spanStart), SymbolStmt: ast.VarDeclarationStmt{NodeName: "VarDeclarationStmt", Span: p.spanFrom(spanStart), BasicVarStmt: ast.BasicVarStmt{
		Identifier: receiverName,
		Type_:      t.fullType(),
		VarType:    t.actType(),
	},
		Symb: symb,
	},
		What: receiverVariableType(t),
	}
}

// receiverVariableType maps a receiver's type derivation onto the storage category
// the AST records, so a method on a pointer receiver is distinguishable from one on a
// value receiver.
func receiverVariableType(t typeRef) ast.VariableType {
	switch t.Form {
	case formPointer:
		return ast.POINTER
	case formReference:
		if t.RefCount == 2 {
			return ast.DBL_REFERENCE
		}
		return ast.REFERENCE
	case formHeapReference:
		return ast.REFERENCE
	case formAddress:
		return ast.ADDRESS
	case formThunk:
		return ast.Thunk
	case formArray, formSlice:
		return ast.ARRAY
	default:
		return ast.NORMAL
	}
}
