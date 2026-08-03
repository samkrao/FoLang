package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// statement — section 10 of docs/grammar/folang.ebnf.
//
//	statement = variable-declaration
//	          | inferred-variable-declaration
//	          | grouped-variable-declaration
//	          | let-value-declaration
//	          | local-function-declaration
//	          | closure-declaration
//	          | multiple-assignment-statement
//	          | return-statement
//	          | expression-statement
//	          | labeled-block
//	          | block-statement
//	          | empty-statement
//
// This is the statement dispatcher. The alternatives are ordered so that the more
// specific shape is always tried first, because several of them begin with a bare
// identifier and are only distinguished by what follows it:
//
//	x co.lang.int = 1;                  variable-declaration
//	x := 1;                             inferred-variable-declaration
//	someother()->()={ … }               local-function-declaration
//	closure = (f int, x int) ==>> x*f;  closure-declaration
//	curry = (f int)(v int) ==>> f * v;  closure-declaration (curried)
//	a, b = b, a;                        multiple-assignment-statement
//	outer:{ … }                         labeled-block
//	x = add(1, 2);                      expression-statement
//
// Each identifier-led form gets its own predicate that inspects the tokens after the
// name, so the dispatch stays a single pass with no backtracking except where a
// predicate cannot decide alone.

// parseStatement parses one statement.
func (p *parser) parseStatement() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	// empty-statement: a bare ";".
	if p.at(scanlex.SEMI_COLON) {
		p.advance()
		return nil
	}

	// File directives are admitted only by file-preamble and entry-item. An
	// import or pragma inside a function/block must not be reinterpreted as an
	// ordinary annotation-only statement.
	if p.atFileDirective() {
		p.failf(p.cur(), "%s is a file directive and cannot appear inside a statement block", p.lexeme())
	}

	// A run of annotations may prefix a declaration or an expression statement
	// (DECISION-SYN-004), so they are read first and passed on.
	annotations := p.parseAnnotations()
	p.rejectOperatorPlacement(annotations, "an executable statement or local declaration")

	switch {
	// An ordinary annotation cannot stand alone. Annotations decorate the
	// declaration or expression that follows; standalone directives have already
	// been dispatched by file-preamble or entry-item before reaching this parser.
	case !annotations.empty() && p.startsNothingAfterAnnotations():
		p.failf(p.cur(), "an annotation must decorate a declaration or expression; it cannot stand alone in a statement block")
		return nil // unreachable: failf panics

	// block-statement: a bare block.
	case p.at(scanlex.OPEN_CURLY):
		return p.parseBlockStatement()

	// labeled-block: identifier ":" "{".
	case p.atLabeledBlock():
		return p.parseLabeledBlock()

	// grouped-variable-declaration: "(" typed-declarator { "," … } ")" ";".
	case p.atGroupedVariableDeclaration():
		return p.parseGroupedVariableDeclaration(annotations)

	// let-value-declaration. The capturing pattern form is dispatched only by
	// parseEntryItem, because it is not a general statement.
	case p.atKeyword("let"):
		return p.parseLetValueDeclaration(annotations)

	// return-statement, plus explicit rejection of scanner-folded control names
	// that are not in the current statement grammar.
	case p.at(scanlex.BUIL_IN_STMT_EXPRS) && isControlStatementBuiltin(p.lexeme()):
		return p.parseControlStatement()

	// inferred-variable-declaration: name ":=" or name "?=".
	case p.atInferredVariableDeclaration():
		return p.parseInferredVariableDeclaration(annotations)

	// local-function-declaration: name "(" … ")" "->" "(" … ")" block.
	case p.atLocalFunctionDeclaration():
		if p.unit == unitEntry {
			p.reportf(p.cur(), "ordinary function declarations are not allowed in an application entry file; use an entry-local function-pattern group")
		}
		return p.parseLocalFunctionDeclaration(annotations)

	// closure-declaration: name "=" parameter-list { parameter-list } "==>>".
	case p.atClosureDeclaration():
		if p.unit == unitEntry {
			p.reportf(p.cur(), "closure declarations are not allowed in an application entry file")
		}
		return p.parseClosureDeclaration(annotations)

	// A declaration introduced by a built-in KIND would create a physically nested
	// named declaration. DECISION-SYN-008 permits only named local functions and
	// anonymous expressions in a block, so consume this shape for recovery but
	// diagnose it rather than silently constructing a legal local type/container.
	case p.atLocalKindDeclaration():
		p.reportf(p.cur(), "a named kind declaration cannot be physically nested in a function or executable block; declare it in its own package source file or use an anonymous expression")
		return p.parseLocalKindDeclaration(annotations)

	// variable-declaration: name type [ "=" expression ].
	case p.atTypedVariableDeclaration():
		return p.parseVariableDeclaration(annotations)

	// multiple-assignment-statement: target "," target { "," target } "=" values.
	case p.atMultipleAssignment():
		return p.parseMultipleAssignmentStatement()
	}

	// expression-statement is the fallback.
	return p.parseExpressionStatement(annotations)
}

// startsNothingAfterAnnotations reports whether a run of annotations is followed by
// something that cannot be decorated, which means the annotations were standalone
// directives rather than a prefix.
func (p *parser) startsNothingAfterAnnotations() bool {
	return p.at(scanlex.SEMI_COLON) || p.at(scanlex.CLOSE_CURLY) || p.atEOF() || p.atAnnotation()
}

// parseExpressionStatement parses the expression-statement production:
//
//	expression-statement = annotations, non-block-expression, statement-end
//
// DECISION-SYN-004 allows annotations on an expression statement, which is what
// admits `@co.dap.lazy` applied to `x = add(1, 2);`.
//
// The non-block-expression guard means a bare braced group here is a block statement
// rather than an expression, and that case has already been taken by the dispatcher.
func (p *parser) parseExpressionStatement(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	expr := p.parseExpression()

	// Commas do not form a general statement list. The dispatcher has already
	// selected homogeneous typed/inferred declarations and the dedicated
	// multiple-assignment production, so a comma here is an arbitrary or mixed
	// statement form and must not be accepted as an extension of expression syntax.
	if p.at(scanlex.COMMA) {
		p.fail(p.cur(), "a comma-separated statement must be a homogeneous typed declaration, a homogeneous inferred declaration, or a multiple assignment")
	}

	p.statementEnd("an expression statement")

	// DECISION-SYN-004: an expression statement may carry annotations, so they are
	// attached rather than dropped once the statement is complete.
	return ast.ExpressionStmt{
		Expression: expr,
		SDapst:     annotations.list(),
		Symb:       p.stmtSymbol("expression-statement"),
	}
}

// parseControlStatement parses the control statements the scanner folds into a
// single built-in token: this.return, this.break and this.continue.
func (p *parser) parseControlStatement() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	switch logicalControlVerb(p.lexeme()) {
	case "return":
		return p.parseReturnStatement()
	case "break", "continue":
		verb := logicalControlVerb(p.lexeme())
		p.reportUnsupported(p.cur(), "this."+verb+" is not part of the current FoLang statement grammar")
		panic(bailout{})
	}
	p.failf(p.cur(), "unsupported control statement %q", p.lexeme())
	return nil // unreachable: failf panics
}

// logicalControlVerb extracts the verb from a folded control built-in, so that both
// "this.return" and "self.return" yield "return".
func logicalControlVerb(lexeme string) string {
	for i := len(lexeme) - 1; i >= 0; i-- {
		if lexeme[i] == '.' {
			return lexeme[i+1:]
		}
	}
	return lexeme
}
