package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parameter-list and parameter — section 8.
//
//	parameter-list = "(", [ parameter, { ",", parameter }, [ "," ] ], ")"
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

// parseParameterList parses the parameter-list production, allowing the trailing
// comma of DECISION-COL-001.
//
// Implements: parameter-list
func (p *parser) parseParameterList() []ast.Parameter {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a parameter list")

	var params []ast.Parameter
	for !p.at(scanlex.CLOSE_PAREN) && !p.atEOF() {
		params = append(params, p.parseParameter())
		if !p.accept(scanlex.COMMA) {
			break
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
func (p *parser) parseParameterLists() [][]ast.Parameter {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	lists := [][]ast.Parameter{p.parseParameterList()}
	for p.at(scanlex.OPEN_PAREN) {
		lists = append(lists, p.parseParameterList())
	}
	return lists
}

// parseParameter parses the parameter production.
//
// Implements: parameter
func (p *parser) parseParameter() ast.Parameter {
	if traceEnabled {
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
	if p.startsTypeExpression(p.cur()) && !p.atOp("=") {
		declaredType = p.parseTypeExpression()
		hasType = true
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
	return ast.Parameter{
		SymbolDeclStmt: p.declFor(paramName.Scanned, actType, declaredType.fullType()),
		Name_:          paramName.Scanned,
		Type_:          declaredType.fullType(),
		Default:        defaultValue,
		HasDefault:     defaultValue != nil,
		DefaultArgs:    defaultValue != nil,
		Optional:       optional,
		OptionalArgs:   optional,
		VarArgs:        variadic,
		Variadic:       variadic,
		NamedArgs:      named,
		ThunkArgs:      declaredType.Form == formThunk,
		OnlyType:       false,
		WhatType:       "param",
		Scope:          "param",
		Symb:           symb,
	}
}

// receiver-clause — section 8.
//
//	receiver-clause = "(", ( type-expression | identifier, type-expression ), ")"
//
// A receiver clause turns a function into a method on the named type. It precedes the
// function name, which is what distinguishes it from the parameter list that follows
// it.

// atReceiverClause reports whether the cursor begins a receiver-clause.
//
// The shape is a parenthesised type, optionally named, followed by a function name.
// The trailing name is what separates a receiver from an ordinary parameter list, so
// the probe checks for it.
//
// The CONTENTS are checked too, because the same "(" opens three other things that a
// trailing `name (` does not rule out:
//
//	@co.dap.public (emp Employee) label()->(S)    receiver, after an annotation
//	@co.dap.inline(level=2) compute()->(S)        the annotation's own arguments
//	someFn()->(S) = { … } (x)                     a call suffix on a body
//
// A receiver holds `[ identifier ] type-expression` and so never contains a top-level
// "=" and never begins with a literal, while an annotation argument list is built from
// `key = value` pairs and bare values. Testing that keeps the annotation form and the
// receiver form apart without whitespace ever mattering.
func (p *parser) atReceiverClause() bool {
	if !p.at(scanlex.OPEN_PAREN) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		if !p.receiverGroupShape() {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		// A receiver is followed by the function's name and then its parameter
		// list.
		if !p.atIdentifier() && !p.atLifecycleName() {
			return false
		}
		p.advance()
		return p.at(scanlex.OPEN_PAREN)
	})
}

// receiverGroupShape reports whether the balanced group at the cursor could be a
// receiver's `[ identifier ] type-expression` rather than an argument list.
//
// It consumes nothing of its own — callers run it inside a lookahead — and only reads
// far enough to reject the two shapes a receiver cannot have: an empty group, and a
// group holding a binder or a literal.
func (p *parser) receiverGroupShape() bool {
	return p.lookaheadOnly(func() bool {
		p.advance() // "("
		if p.at(scanlex.CLOSE_PAREN) {
			return false // "()" is an empty argument list, never a receiver
		}
		// A receiver names a type, so it opens with a name or a built-in type.
		if !p.atIdentifier() && !p.at(scanlex.BUILT_IN_TYPE) && !p.at(scanlex.BUIL_IN_STMT_EXPRS) {
			return false
		}
		depth := 0
		for !p.atEOF() {
			switch p.kind() {
			case scanlex.OPEN_PAREN, scanlex.OPEN_BRACKET, scanlex.OPEN_CURLY:
				depth++
			case scanlex.CLOSE_BRACKET, scanlex.CLOSE_CURLY:
				depth--
			case scanlex.CLOSE_PAREN:
				if depth == 0 {
					return true
				}
				depth--
			case scanlex.STRING, scanlex.NUMBER, scanlex.CHAR:
				if depth == 0 {
					return false // a literal is an annotation value, not a type
				}
			case scanlex.COMMA:
				if depth == 0 {
					return false // a receiver declares exactly one type
				}
			default:
				// A binder at the top level makes this `key = value`.
				if depth == 0 && p.atAnyOp("=", ":") {
					return false
				}
			}
			p.advance()
		}
		return false
	})
}

// parseReceiverClause parses the receiver-clause production.
//
// Implements: receiver-clause
func (p *parser) parseReceiverClause() *ast.FunctionReceiver {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a receiver clause")

	// The named form binds the receiver to a name; the bare form gives only its
	// type.
	receiverName := ""
	if p.atIdentifier() && p.namePrecedesType() {
		receiverName = p.parseIdentifier("as a receiver name").Scanned
	}

	t := p.parseTypeExpression()
	p.expect(scanlex.CLOSE_PAREN, "to close a receiver clause")

	symb := p.varSymbol(receiverName, t.actType())
	symb.IsParam = true

	return &ast.FunctionReceiver{
		SymbolStmt: ast.VarDeclarationStmt{
			BasicVarStmt: ast.BasicVarStmt{
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
