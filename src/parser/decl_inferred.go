package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// inferred-variable-declaration — section 5.
//
//	inferred-variable-declaration = annotations, inferred-variable-declarator,
//	                                { ",", inferred-variable-declarator },
//	                                statement-end
//	inferred-variable-declarator  = identifier, ( ":=" | "?=" ), expression
//
// The two operators differ in what they do when the name already exists
// (docs/language-ref.md, "Variables"):
//
//	name := "Rao";     define and infer; an error if name is already declared
//	name ?= "Kumar";   define and infer if undefined, otherwise reassign
//
// DECISION-OP-003 is important here: ":=" and "?=" are statement-level DEFINITION
// operators, not expression operators. They cannot be chained, so `a := b := c` is
// not a declaration of two names, and "::=" remains reserved and is rejected. That is
// why they are handled in this file rather than in the Pratt table.

// atInferredVariableDeclaration reports whether the cursor begins an
// inferred-variable-declarator.
// The declarator is spelled `identifier, ( ":=" | "?=" ), expression`, so "??=" is not
// one of its operators: it is the nullish compound ASSIGNMENT, which updates a name
// that already exists rather than declaring one. Accepting it here made `x ??= 5;` an
// undocumented third way to declare a variable. "::=" stays in the set so the reserved
// operator keeps its own diagnostic (DECISION-OP-003) instead of falling through to a
// generic one.
func (p *parser) atInferredVariableDeclaration() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.peek(1).IsOneOfMany(scanlex.WALRUS, scanlex.QEQ, scanlex.COLON_WALRUS)
}

// parseInferredVariableDeclaration parses the inferred-variable-declaration
// production.
//
// Implements: inferred-variable-declaration
func (p *parser) parseInferredVariableDeclaration(annotations annotationSet) ast.Stmt {
	p.rejectEffectsPlacement(annotations, "an inferred variable declaration")
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declarators := []ast.Stmt{p.parseInferredVariableDeclarator(annotations)}

	for p.accept(scanlex.COMMA) {
		if !p.atInferredVariableDeclaration() {
			p.fail(p.cur(), "an inferred declaration list may contain only inferred declarators; move typed declarations or assignments to a separate statement")
		}
		declarators = append(declarators, p.parseInferredVariableDeclarator(annotations))
	}

	p.statementEnd("an inferred variable declaration")
	return p.oneOrGrouped(declarators, "inferred-variable-declaration")
}

// parseInferredVariableDeclarator parses one inferred-variable-declarator.
//
// Implements: inferred-variable-declarator
func (p *parser) parseInferredVariableDeclarator(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseIdentifier("as an inferred variable name")
	opTok := p.advance()
	if opTok.Kind == scanlex.WALRUS || opTok.Kind == scanlex.QEQ {
		p.requireDefinitionOperatorBoundaries(opTok)
	}

	// "::=" is reserved and must be refused rather than treated as a definition
	// operator (DECISION-OP-003 with DECISION-OP-005).
	if opTok.Kind == scanlex.COLON_WALRUS {
		p.reportUnsupported(opTok, "the operator ::= is reserved for a future feature; use := to define and infer, or ?= to define or reassign")
		panic(bailout{})
	}

	value := p.parseExpression()
	inferredType := p.firstPassExpressionType(value)

	// A second definition operator on the same line would be a chain, which
	// DECISION-OP-003 forbids.
	if p.atAny(scanlex.WALRUS, scanlex.QEQ) {
		p.reportf(p.cur(), "%q is a statement-level definition operator and cannot be chained; write the declarations separately", p.lexeme())
	}

	var symb *symboltable.VarSymbol
	if opTok.Kind == scanlex.QEQ {
		// ?= searches the complete visible chain from the segment active at the
		// occurrence. If it finds a local variable declaration, this AST node
		// refers to that declaration rather than minting a second binding.
		if visible := p.symtab.GetVarDetails(*p.fs, declName.Scanned); visible != nil && visible.GetSymbolID() != "" {
			symb, _ = visible.(*symboltable.VarSymbol)
		}
	}
	if symb == nil {
		declaredType := inferredType
		if declaredType == "" {
			declaredType = "co.lang.infer"
		}
		symb = p.varSymbol(declName.Scanned, declaredType)
		symb.Inferred = true
		symb.HasInitValue = true
		symb.ExplicitType = false
		symb.Discard = declName.isWildcard()
		symb.ExistsAssign = opTok.Kind == scanlex.QEQ
		symb.AutoCreate = opTok.Kind == scanlex.QEQ
		p.declareNamed(declName, symb)
	}
	varType := symb.GetType()
	if varType == "" {
		varType = "co.lang.infer"
	}

	return ast.VarDeclarationStmt{NodeName: "VarDeclarationStmt", Span: p.spanFrom(spanStart), BasicVarStmt: ast.BasicVarStmt{
		Identifier:         declName.Scanned,
		AssignedValue:      value,
		VarType:            varType,
		DefinitionOperator: opTok.Value,
		SDapst:             annotations.list(),
	},
		Symb: symb,
	}
}

// firstPassExpressionType returns a type only when parsing has enough local,
// deterministic information to do so. Calls, overloads, dynamic values,
// imports, and generic substitution remain work for semantic resolution.
func (p *parser) firstPassExpressionType(expression ast.Expr) string {
	switch value := expression.(type) {
	case ast.IntegerLiteral:
		return "co.lang.int"
	case ast.NumberLiteral:
		return "co.lang.double"
	case ast.StringLiteral:
		return "co.lang.string"
	case ast.CharacterLiteral:
		return "co.lang.char"
	case ast.BooleanLiteral:
		return "co.lang.bool"
	case ast.GroupingExpr:
		return p.firstPassExpressionType(value.Expr_)
	case ast.SymbolExpr:
		visible := p.symtab.GetVarDetails(*p.fs, value.Value)
		if visible == nil || visible.GetSymbolID() == "" || deferredType(visible.GetType()) {
			return ""
		}
		return visible.GetType()
	case ast.BinaryExpr:
		left := p.firstPassExpressionType(value.Left)
		right := p.firstPassExpressionType(value.Right)
		switch value.Operator.Value {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
			if left != "" && right != "" {
				return "co.lang.bool"
			}
			return ""
		}
		if left == right {
			return left
		}
		if (left == "co.lang.int" && right == "co.lang.double") ||
			(left == "co.lang.double" && right == "co.lang.int") {
			return "co.lang.double"
		}
	}
	return ""
}

func deferredType(typeName string) bool {
	normalized := strings.ToLower(typeName)
	return normalized == "" || normalized == "co.lang.infer" ||
		strings.Contains(normalized, "co.lang.any") ||
		strings.Contains(normalized, "co.lang.dynamic") ||
		strings.Contains(normalized, "generic")
}

// grouped-variable-declaration — section 10.
//
//	grouped-variable-declaration = "(", typed-variable-declarator,
//	                               { ",", typed-variable-declarator }, ")",
//	                               statement-end
//
// This is the parenthesised declaration group of docs/language-ref.md, "Comma and
// Grouping":
//
//	(x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true);

// atGroupedVariableDeclaration reports whether the cursor begins a
// grouped-variable-declaration.
//
// A "(" also begins a grouped expression, a tuple and an anonymous function, so the
// check looks inside for the `identifier type` shape that only a declarator has.
func (p *parser) atGroupedVariableDeclaration() bool {
	if !p.at(scanlex.OPEN_PAREN) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // "("
		if !p.atIdentifier() {
			return false
		}
		p.advance()
		return p.at(scanlex.BUILT_IN_TYPE) ||
			(p.at(scanlex.BUILT_IN_KIND) && isTypeFirstKind(p.lexeme())) ||
			(p.atIdentifier() && !p.peek(1).IsOneOfMany(scanlex.OPEN_PAREN))
	})
}

// parseGroupedVariableDeclaration parses the grouped-variable-declaration
// production.
//
// Implements: grouped-variable-declaration
func (p *parser) parseGroupedVariableDeclaration(annotations annotationSet) ast.Stmt {
	p.rejectEffectsPlacement(annotations, "a grouped variable declaration")
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a grouped variable declaration")

	declarators := []ast.Stmt{p.parseTypedVariableDeclarator(annotations)}
	for p.accept(scanlex.COMMA) {
		if p.at(scanlex.CLOSE_PAREN) {
			p.fail(p.cur(), "a comma in a grouped variable declaration must be followed by another declarator; trailing commas are not allowed")
		}
		declarators = append(declarators, p.parseTypedVariableDeclarator(annotations))
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a grouped variable declaration")
	p.statementEnd("a grouped variable declaration")

	return p.oneOrGrouped(declarators, "grouped-variable-declaration")
}

// let-value-declaration — section 10.
//
//	let-value-declaration = "let", identifier, [ type-expression ], "=",
//	                        expression, statement-end
//
// This is the ordinary let value binding. It is distinct from the let EXPRESSION of
// expr_let.go, which spells its bindings inside `let({…}).in({…})`, and from the
// capturing let function-pattern clause of pattern_function.go.
//
// The reference notes that ordinary let value bindings are forbidden in an
// application entry file, where "let" is reserved for a capturing function-pattern
// group. That is a context restriction rather than a syntactic one, so it is
// reported here only when the unit is known to be an entry file.

// parseLetValueDeclaration parses the let-value-declaration production.
//
// Implements: let-value-declaration
func (p *parser) parseLetValueDeclaration(annotations annotationSet) ast.Stmt {
	p.rejectEffectsPlacement(annotations, "a let value declaration")
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	letTok := p.expectKeyword("let", "to begin a let declaration")

	if p.unit == unitEntry {
		p.report(letTok, "an ordinary let value binding is not allowed in an application entry file; there, \"let\" introduces a capturing function-pattern group")
	}

	declName := p.parseIdentifier("as a let binding name")

	// The type annotation is optional; without it the type comes from the value.
	declaredType := typeRef{Form: formPlain}
	hasType := false
	if p.startsTypeUse(p.cur()) && !p.atOp("=") {
		declaredType = p.parseTypeUse("as a declared binding type")
		hasType = true
	}

	p.expectOp("=", "in a let declaration")
	value := p.parseExpression()
	p.statementEnd("a let declaration")

	actType := "co.lang.infer"
	if hasType {
		actType = declaredType.actType()
	}

	symb := p.varSymbol(declName.Scanned, actType)
	symb.LocalBinding = true
	symb.HasInitValue = true
	symb.Inferred = !hasType
	symb.ExplicitType = hasType
	p.declareNamed(declName, symb)

	return ast.VarDeclarationStmt{NodeName: "VarDeclarationStmt", Span: p.spanFrom(spanStart), BasicVarStmt: ast.BasicVarStmt{
		Identifier:    declName.Scanned,
		AssignedValue: value,
		// Unlike an ordinary typed declarator, a let declaration always uses
		// VarDeclarationStmt and therefore has no derivation-specific statement
		// node. Preserve a written derived type in the type slot itself.
		Type_:   declaredType.fullType(),
		VarType: "let",
		SDapst:  annotations.list(),
	},
		Symb: symb,
	}
}
