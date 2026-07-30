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
//	closure(f int) => (x int) = x * f;  closure-declaration (arrow form)
//	curry(f int)(v int) = f * v;        closure-declaration (curried form)
//	a, b = b, a;                        multiple-assignment-statement
//	outer:{ … }                         labeled-block
//	x = add(1, 2);                      expression-statement
//
// Each identifier-led form gets its own predicate that inspects the tokens after the
// name, so the dispatch stays a single pass with no backtracking except where a
// predicate cannot decide alone.

// parseStatement parses one statement.
func (p *parser) parseStatement() ast.Stmt {
	defer p.enter()()

	// empty-statement: a bare ";".
	if p.at(scanlex.SEMI_COLON) {
		p.advance()
		return nil
	}

	// A run of annotations may prefix a declaration or an expression statement
	// (DECISION-SYN-004), so they are read first and passed on.
	annotations := p.parseAnnotations()

	switch {
	// A built-in directive standing alone. Directives are self-delimiting
	// (DECISION-DIR-001), so one appearing here is a statement of its own.
	case !annotations.empty() && p.startsNothingAfterAnnotations():
		return p.annotationOnlyStatement(annotations)

	// block-statement: a bare block.
	case p.at(scanlex.OPEN_CURLY):
		return p.parseBlockStatement()

	// labeled-block: identifier ":" "{".
	case p.atLabeledBlock():
		return p.parseLabeledBlock()

	// grouped-variable-declaration: "(" typed-declarator { "," … } ")" ";".
	case p.atGroupedVariableDeclaration():
		return p.parseGroupedVariableDeclaration(annotations)

	// let-value-declaration and the capturing let function-pattern clause both
	// begin with "let".
	case p.atKeyword("let"):
		return p.parseLetLedStatement(annotations)

	// return-statement: "this.return" / "self.return", and the break and
	// continue statements that fold the same way.
	case p.at(scanlex.BUIL_IN_STMT_EXPRS) && isControlStatementBuiltin(p.lexeme()):
		return p.parseControlStatement()

	// inferred-variable-declaration: name ":=" or name "?=".
	case p.atInferredVariableDeclaration():
		return p.parseInferredVariableDeclaration(annotations)

	// local-function-declaration: name "(" … ")" "->" "(" … ")" block.
	case p.atLocalFunctionDeclaration():
		return p.parseLocalFunctionDeclaration(annotations)

	// closure-declaration, both the curried and the arrow form.
	case p.atClosureDeclaration():
		return p.parseClosureDeclaration(annotations)

	// A bare function-pattern clause: name "(" pattern … ")" "=>" result.
	case p.atBareFunctionPatternClause():
		return p.parseBareFunctionPatternClause()

	// A declaration introduced by a built-in KIND rather than by a type, such as a local
	// lambda, function object or nested type. DECISION-KIND-001 prefers
	// variable-declaration in statement position, which is why this case is tested by
	// the presence of a kind token and so cannot capture an ordinary declarator.
	case p.atLocalKindDeclaration():
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

// annotationOnlyStatement wraps a run of standalone directives as a statement.
//
// A built-in directive ends at its complete directive form and takes no semicolon
// (DECISION-DIR-001), so one is consumed here without a terminator. A stray ";"
// after it is reported rather than silently accepted.
func (p *parser) annotationOnlyStatement(annotations annotationSet) ast.Stmt {
	if p.at(scanlex.SEMI_COLON) {
		p.reportf(p.cur(), "unexpected %q after a built-in directive; directives are self-delimiting and take no terminator", ";")
		p.advance()
	}

	body := make([]ast.Stmt, 0, len(annotations.all))
	for _, d := range annotations.all {
		body = append(body, d)
	}

	// A single directive is returned directly; several are grouped.
	if len(body) == 1 {
		return body[0]
	}
	return &ast.BlockStmt{Body: body, Symb: p.blockSymbol("directives", false)}
}

// parseLetLedStatement parses the two statements that begin with "let".
//
//	let-value-declaration             = "let", identifier, [ type-expression ],
//	                                    "=", expression, statement-end
//	capturing-function-pattern-clause = "let", identifier,
//	                                    pattern-parameter-list,
//	                                    [ where-clause ], "=", pattern-result
//
// The two are told apart by what follows the name: a "(" makes it a pattern clause,
// anything else a value declaration.
func (p *parser) parseLetLedStatement(annotations annotationSet) ast.Stmt {
	if p.peek(2).Kind == scanlex.OPEN_PAREN {
		return p.parseCapturingFunctionPatternClause()
	}
	return p.parseLetValueDeclaration(annotations)
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
	expr := p.parseExpression()

	// A comma here is a comma-separated declaration or assignment list rather
	// than an expression: `c co.lang.string, a = 20, b = 30;`.
	if p.at(scanlex.COMMA) {
		return p.parseCommaStatementTail(expr, annotations)
	}

	p.statementEnd("an expression statement")

	return ast.ExpressionStmt{
		Expression: expr,
		Symb:       p.stmtSymbol("expression-statement"),
	}
}

// parseControlStatement parses the control statements the scanner folds into a
// single built-in token: this.return, this.break and this.continue.
func (p *parser) parseControlStatement() ast.Stmt {
	switch logicalControlVerb(p.lexeme()) {
	case "return":
		return p.parseReturnStatement()
	case "break":
		return p.parseBreakStatement()
	case "continue":
		return p.parseContinueStatement()
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
