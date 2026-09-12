package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// statement — section 10 of docs/grammar/folang.ebnf.
//
//	statement = named-block-declaration
//	          | variable-declaration
//	          | inferred-variable-declaration
//	          | grouped-variable-declaration
//	          | let-value-declaration
//	          | local-function-declaration
//	          | closure-declaration
//	          | multiple-assignment-statement
//	          | return-statement
//	          | break-statement
//	          | continue-statement
//	          | labeled-block
//	          | labeled-loop-statement
//	          | expression-statement
//	          | block-statement
//	          | empty-statement
//
// This is the statement dispatcher. The alternatives are ordered so that the more
// specific shape is always tried first, because several of them begin with a bare
// identifier and are only distinguished by what follows it:
//
//	labelBlock co.lang.block={ … }      named-block-declaration
//	x co.lang.int = 1;                  variable-declaration
//	x := 1;                             inferred-variable-declaration
//	someother()->()={ … }               local-function-declaration
//	closure = (f int, x int) ==>> x*f;  closure-declaration
//	curry = (f int)(v int) ==>> f * v;  closure-declaration (curried)
//	a, b = b, a;                        multiple-assignment-statement
//	this ->| 'outer;                    break-statement
//	'outer: { … }                       labeled-block
//	'outer: (c).loop({ … });            labeled-loop-statement
//	x = add(1, 2);                      expression-statement
//
// Each identifier-led form gets its own predicate that inspects the tokens after the
// name, so the dispatch stays a single pass with no backtracking except where a
// predicate cannot decide alone.

// parseStatement parses one statement.
//
// The dispatch is also where the two visibility rules of docs/language-ref.md B.2
// are applied, because this is the point at which the parser knows which of the
// two a statement is. A variable declaration calls beginDeclarationSegment, which
// opens a new symbol-table segment when another context-level item has intervened.
// Non-variable declarations and statements call noteExecutableItem, which makes
// the NEXT variable declaration interleaved; the intervening item itself remains
// anchored to the already-open frontier. See scope.go.
//
// Implements: statement
func (p *parser) parseStatement() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	// empty-statement: a bare ";".
	if p.at(scanlex.SEMI_COLON) {
		p.noteExecutableItem()
		p.advance()
		return nil
	}

	// Every directive is admitted by file-preamble alone, so one here is inside a
	// body and must not be reinterpreted as an ordinary annotation-only
	// statement. `@co.ddap.use` used to be the exception, admitted by block-item
	// for a block-scoped activation; activation is file-scoped now and the
	// exception is gone with it.
	//
	// The directive is reported and then parsed rather than bailing out, so a
	// misplaced one costs a single diagnostic and the rest of the block is still
	// read.
	if p.atFileDirective() {
		p.noteExecutableItem()
		p.rejectMisplacedFileMetadata(p.cur())
		return p.parseFileDirective()
	}

	// @co.dap.onEffect is call-site metadata and belongs to the expression parser.
	// Every other metadata application here is declaration metadata.
	annotations := annotationSet{byKind: map[scanlex.DirectiveKind][]ast.Stmt{}, at: p.spanOf(p.cur())}
	if !(p.atAnnotation() && p.cur().Value == "@co.dap.onEffect") {
		annotations = p.parseAnnotations()
	}
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
		p.noteExecutableItem()
		return p.parseBlockStatement()

	// labeled-block and labeled-loop-statement, both introduced by a
	// label-declaration.
	case p.atLabeledStatement():
		p.noteExecutableItem()
		return p.parseLabeledStatement()

	// grouped-variable-declaration: "(" typed-declarator { "," … } ")" ";".
	case p.atGroupedVariableDeclaration():
		p.beginDeclarationSegment()
		return p.parseGroupedVariableDeclaration(annotations)

	// let-value-declaration. The capturing pattern form is dispatched only by
	// parseEntryItem, because it is not a general statement.
	case p.atKeyword("let"):
		p.beginDeclarationSegment()
		return p.parseLetValueDeclaration(annotations)

	// return-statement, break-statement and continue-statement, which the scanner
	// folds into one built-in token each.
	case p.atValueReturnStatement():
		p.noteExecutableItem()
		return p.parseValueReturnStatement()

	case p.atLegacyControlStatement():
		p.noteExecutableItem()
		p.failf(p.cur(), "dotted control statement %q is not valid; use this =>, this ->, or this ->|", p.legacyControlStatementName())
		return nil // unreachable: failf panics

	case p.atControlStatement():
		p.noteExecutableItem()
		return p.parseControlStatement()

	// lock-statement: contextual lock(target) { ... } syntax. "lock" remains an
	// ordinary identifier outside this complete statement shape.
	case p.atLockStatement():
		p.noteExecutableItem()
		return p.parseLockStatement()

	// inferred-variable-declaration: name ":=" or name "?=".
	case p.atInferredVariableDeclaration():
		p.beginDeclarationSegment()
		return p.parseInferredVariableDeclaration(annotations)

	// local-function-declaration: name "(" … ")" "->" "(" … ")" "=" block.
	//
	// An entry file's top level admits neither this nor a closure declaration, but
	// that is entry-statement's rule and parseEntryStatement reports it; a nested
	// block inside an entry file still reaches this dispatcher normally.
	case p.atLocalFunctionDeclaration():
		p.noteExecutableItem()
		return p.parseLocalFunctionDeclaration(annotations)

	// closure-declaration: name "=" parameter-list { parameter-list } "==>>".
	case p.atClosureDeclaration():
		p.noteExecutableItem()
		return p.parseClosureDeclaration(annotations)

	// named-block-declaration: name "co.lang.block" "=" block. It is a statement, so
	// it is dispatched BEFORE the nested-kind guard that rejects every other
	// kind-introduced declaration in a block. A block is the one construct the
	// reference requires to live inside a function or method.
	case p.atNamedBlockDeclaration():
		p.noteExecutableItem()
		return p.parseNamedBlockDeclaration(annotations)

	case p.atTypeDeclarationMember():
		p.beginDeclarationSegment()
		return p.parseLocalKindDeclaration(annotations)

	// Ordinary non-UDT type declarations were dispatched above and are legal in
	// executable blocks. Any other built-in-kind declaration reaching this point
	// would create a forbidden file-backed/container declaration, so consume it for
	// recovery after reporting the placement error.
	case p.atLocalKindDeclaration():
		p.noteExecutableItem()
		p.reportNamed(p.cur(), helpers.DiagnosticInvalidDeclarationForm, "Invalid Declaration Form", "a named kind declaration cannot be physically nested in a function or executable block; declare it in its own package source file or use an anonymous expression")
		return p.parseLocalKindDeclaration(annotations)

	// variable-declaration: name type [ "=" expression ].
	case p.atTypedVariableDeclaration():
		p.beginDeclarationSegment()
		return p.parseVariableDeclaration(annotations)

	// multiple-assignment-statement: target "," target { "," target } "=" values.
	case p.atMultipleAssignment():
		p.noteExecutableItem()
		return p.parseMultipleAssignmentStatement()
	}

	// expression-statement is the fallback.
	p.noteExecutableItem()
	return p.parseExpressionStatement(annotations)
}

// startsNothingAfterAnnotations reports whether a run of annotations is followed by
// something that cannot be decorated, which means the annotations were standalone
// directives rather than a prefix.
func (p *parser) startsNothingAfterAnnotations() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.at(scanlex.SEMI_COLON) || p.at(scanlex.CLOSE_CURLY) || p.atEOF() || p.atAnnotation()
}

// parseExpressionStatement parses the expression-statement production:
//
// The normative reference still permits declaration-style metadata such as
// @co.dap.lazy on an expression statement. @co.dap.onEffect is the narrower
// exception parsed as part of its call expression.
//
// The non-block-expression guard means a bare braced group here is a block statement
// rather than an expression, and that case has already been taken by the dispatcher.
//
// Implements: expression-statement
func (p *parser) parseExpressionStatement(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
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

	return ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: expr,
		SDapst:   annotations.list(),
		SymbolId: p.statementID("expression-statement"),
	}
}

// parseControlStatement parses the three symbolic controls headed by `this`.
func (p *parser) parseControlStatement() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	switch {
	case p.atValueReturnStatement():
		return p.parseReturnStatement()
	case p.atBreakStatement():
		return p.parseBreakStatement()
	case p.atContinueStatement():
		return p.parseContinueStatement()
	}
	p.failf(p.cur(), "unsupported control statement %q", p.lexeme())
	return nil // unreachable: failf panics
}

// atControlStatement recognizes only complete symbolic `this` control heads.
func (p *parser) atControlStatement() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.atValueReturnStatement() || p.atBreakStatement() || p.atContinueStatement()
}

func (p *parser) atLegacyControlStatement() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	return p.legacyControlStatementName() != ""
}

func (p *parser) legacyControlStatementName() string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if name := logicalName(p.lexeme()); name == "this.return" || name == "this.break" || name == "this.continue" {
		return name
	}
	if logicalName(p.lexeme()) != "this" || p.peek(1).Kind != scanlex.DOT {
		return ""
	}
	name := "this." + logicalName(p.peek(2).Value)
	if name == "this.return" || name == "this.break" || name == "this.continue" {
		return name
	}
	return ""
}

func (p *parser) atBreakStatement() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return logicalName(p.lexeme()) == "this" && (p.peek(1).Value == "->|" ||
		p.peek(1).Kind == scanlex.ARROW && p.peek(2).Kind == scanlex.PIPE)
}

func (p *parser) atContinueStatement() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return logicalName(p.lexeme()) == "this" && p.peek(1).Kind == scanlex.ARROW && p.peek(2).Kind != scanlex.PIPE
}

// break-statement and continue-statement — section 10.
//
//	break-statement    = "this", "->", "|", [ label-reference ], statement-end,
//	                     break-target-guard
//	continue-statement = "this", "->", [ label-reference ], statement-end,
//	                     continue-target-guard
//
// Both are structured exits, not jumps: the optional label-reference selects
// WHICH enclosing region is left, and nothing else about the transfer changes
// (docs/language-ref.md, "Label Resolution").
//
// Both target guards are semantic. Whether an enclosing region with the named
// label is active, whether that region is a loop — which is what separates a
// legal `this -> 'outer;` from an illegal one — and which of several
// same-spelled labels is innermost are all questions about the enclosing
// declaration's control regions, not about the token stream, so the parse
// records the reference and leaves resolution to the phase that has that scope.
//
// The scanner emits `->` as ARROW and `|` as PIPE. Parser context gives that
// pair control meaning only after the hard-reserved `this` head.

// parseBreakStatement parses the break-statement production.
//
// Implements: break-statement
// Implements: break-target-guard
func (p *parser) parseBreakStatement() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.advance() // this
	if p.atOp("->|") {
		p.advance()
	} else {
		p.expect(scanlex.ARROW, "after this in a break statement")
		p.expect(scanlex.PIPE, "after this -> in a break statement")
	}
	label := p.parseOptionalLabelReference()
	p.statementEnd("a break statement")

	return ast.BreakStmt{NodeName: "BreakStmt", Span: p.spanFrom(spanStart), Label: label,
		SymbolId: p.statementID("this ->|"),
	}
}

// parseContinueStatement parses the continue-statement production.
//
// Implements: continue-statement
// Implements: continue-target-guard
func (p *parser) parseContinueStatement() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.advance() // this
	p.expect(scanlex.ARROW, "after this in a continue statement")
	label := p.parseOptionalLabelReference()
	p.statementEnd("a continue statement")

	return ast.ContinueStmt{NodeName: "ContinueStmt", Span: p.spanFrom(spanStart), Label: label,
		SymbolId: p.statementID("this ->"),
	}
}

// parseOptionalLabelReference reads the `[ label-reference ]` both control
// statements share, returning "" when the statement is unlabeled.
func (p *parser) parseOptionalLabelReference() string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	if !p.atLabelIdentifier() {
		return ""
	}
	return p.parseLabelIdentifier("as a control label reference").Scanned
}
