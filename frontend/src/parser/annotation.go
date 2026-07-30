package parser

import (
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Annotations and metadata — section 2 of docs/grammar/folang.ebnf.
//
//	annotations = { annotation }
//	annotation  = "@", qualified-name, [ "(", [ annotation-argument-list ], ")" ]
//
// An annotation is self-delimiting: it takes no terminator of its own
// (DECISION-DIR-001), so a run of them simply precedes whatever they decorate.
//
// Annotation values are decoded to plain Go values rather than to AST nodes,
// because that is the shape the consumer expects: ast.DirectiveStmt carries
// Parameters as map[string]any and the semantic phase reads options such as
// `for=Functor` or `kind=length` straight out of it.

// annotationSet is a parsed run of annotations, grouped the way the AST wants
// them.
//
// FoLang distinguishes pragmas, directives, annotations and decorators
// (scanlex.DirectiveKind), and each AST node stores them grouped by kind, so the
// grouping is done once here.
type annotationSet struct {
	byKind map[scanlex.DirectiveKind][]ast.Stmt
	all    []ast.DirectiveStmt
}

// empty reports whether no annotation was present.
func (a annotationSet) empty() bool { return len(a.all) == 0 }

// list returns the annotations as the ast.Stmt a node's directive field holds.
//
// The value is stored directly rather than through SetDap because every SetDap
// implementation in package ast takes a value receiver, so it mutates a copy and
// cannot reach the node being built.
func (a annotationSet) list() ast.Stmt {
	return &ast.DirectveList{Dapst: a.byKind}
}

// has reports whether the set contains the named annotation, for example
// "@co.dap.hokrt".
func (a annotationSet) has(name string) bool {
	for _, d := range a.all {
		if d.Name == name {
			return true
		}
	}
	return false
}

// option returns the value of a named argument on a named annotation, as in the
// `fortype` of `@co.dap.extension(fortype=Employee)`.
func (a annotationSet) option(annotation, key string) (any, bool) {
	for _, d := range a.all {
		if d.Name != annotation {
			continue
		}
		if v, ok := d.Parameters[key]; ok {
			return v, true
		}
	}
	return nil, false
}

// optionString returns a named annotation argument as a string, or "" when it is
// absent.
func (a annotationSet) optionString(annotation, key string) string {
	if v, ok := a.option(annotation, key); ok {
		if s, isString := v.(string); isString {
			return s
		}
	}
	return ""
}

// atAnnotation reports whether the cursor begins an annotation.
//
// The scanner folds an annotation's dotted path into one token and classifies it
// as a built-in directive, a custom directive, or the unfolded ATDAP form when the
// path is a single segment.
func (p *parser) atAnnotation() bool {
	return p.atAny(scanlex.BUILT_IN_DIRECTIVES, scanlex.CUSTOM_DIRECTIVES, scanlex.ATDAP)
}

// parseAnnotations parses the annotations production, a possibly empty run.
func (p *parser) parseAnnotations() annotationSet {
	set := annotationSet{byKind: map[scanlex.DirectiveKind][]ast.Stmt{}}
	for p.atAnnotation() {
		d := p.parseAnnotation()
		set.all = append(set.all, d)
		kind := directiveKindOf(d.Name)
		set.byKind[kind] = append(set.byKind[kind], d)
	}
	return set
}

// parseOneOrMoreAnnotations parses the one-or-more-annotations production, used by
// the declaration forms that are selected by the presence of an annotation —
// annotated-contract-declaration and annotated-function-primary.
func (p *parser) parseOneOrMoreAnnotations() annotationSet {
	if !p.atAnnotation() {
		p.failf(p.cur(), "expected an annotation, found %s", describeToken(p.cur()))
	}
	return p.parseAnnotations()
}

// parseAnnotation parses one annotation:
//
//	annotation = "@", qualified-name, [ "(", [ annotation-argument-list ], ")" ]
//
// Positional arguments are keyed by their index so that Parameters can stay a
// single map, and a bare flag key is recorded with the boolean value true, as
// DECISION-ANN-001 specifies.
func (p *parser) parseAnnotation() ast.DirectiveStmt {
	tok := p.advance()
	annotationName := tok.Value

	params := map[string]any{}
	if p.at(scanlex.OPEN_PAREN) {
		p.advance()
		if !p.at(scanlex.CLOSE_PAREN) {
			for i, arg := range p.parseAnnotationArgumentList() {
				if arg.Key == "" {
					params[strconv.Itoa(i)] = arg.Value
					continue
				}
				params[arg.Key] = arg.Value
			}
		}
		p.expect(scanlex.CLOSE_PAREN, "to close an annotation argument list")
	}

	kind := directiveKindOf(annotationName)
	return ast.DirectiveStmt{
		Name:            annotationName,
		Parameters:      params,
		DirectiveType:   scanlex.KindToString[kind],
		DirectiveKind_:  scanlex.KindToPhase[kind],
		DirectiveScope_: scanlex.KindToScope[kind],
		Symb:            p.directiveSymbol(annotationName, kind == scanlex.PRAGMA),
	}
}

// directiveKindOf classifies an annotation name as a pragma, directive,
// annotation or decorator. An unregistered name is treated as an annotation,
// which is the kind a user-defined decorator declares itself into.
func directiveKindOf(annotationName string) scanlex.DirectiveKind {
	if kind, ok := scanlex.Built_in_directive_kind(annotationName); ok {
		return kind
	}
	return scanlex.ANNOTATION
}

// annotationArg is one parsed annotation argument. Key is empty for a positional
// argument, and Flag marks a bare key whose value defaults to true.
type annotationArg struct {
	Key   string
	Value any
	Flag  bool
	Tok   scanlex.Token
}

// parseAnnotationArgumentList parses the annotation-argument-list production:
//
//	annotation-argument-list = annotation-argument,
//	                           { ",", annotation-argument }, [ "," ]
//
// A trailing comma is permitted (DECISION-COL-001).
func (p *parser) parseAnnotationArgumentList() []annotationArg {
	var args []annotationArg
	for {
		args = append(args, p.parseAnnotationArgument())
		if !p.accept(scanlex.COMMA) {
			return args
		}
		if p.at(scanlex.CLOSE_PAREN) {
			return args // trailing comma
		}
	}
}

// parseAnnotationArgument parses the annotation-argument production:
//
//	annotation-argument = [ annotation-key, annotation-binder ], annotation-value
//	annotation-binder   = "=" | ":"
//
// DECISION-ANN-001 makes "=" and ":" interchangeable binders, which is what admits
// the reference forms `@co.dap.oops(A: { inherit:true })` and
// `@co.dap.generic(type={T:{typename}, R:{variance:invariant, bound=Number}})`
// that mix both spellings freely.
//
// The optional key group is decided by lookahead rather than by backtracking: a
// key is present only when a binder follows it. For a bare value such as
// `co.lang.int` there is no binder, so the whole group is skipped and the value is
// matched by annotation-value.
func (p *parser) parseAnnotationArgument() annotationArg {
	start := p.cur()

	if p.atAnnotationKeyWithBinder() {
		key := p.parseAnnotationKey("as an annotation argument name")
		p.advance() // the binder, "=" or ":"
		return annotationArg{Key: key, Value: p.parseAnnotationValue(), Tok: start}
	}

	return annotationArg{Value: p.parseAnnotationValue(), Tok: start}
}

// atAnnotationKeyWithBinder reports whether the cursor begins an annotation key
// followed by a binder.
//
// A key may itself contain hyphens — `parent-realm` is one key, per
// annotation-key — so the scan walks the `identifier { "-" identifier }` shape
// before testing for the binder.
func (p *parser) atAnnotationKeyWithBinder() bool {
	if !p.atIdentifier() && !p.at(scanlex.KEYWORD) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance()
		for p.atOp("-") && p.isMemberNameToken(p.peek(1)) {
			p.advance()
			p.advance()
		}
		return p.atOp("=") || p.at(scanlex.COLON)
	})
}

// parseAnnotationKey parses the annotation-key production:
//
//	annotation-key = identifier, { "-", identifier }
//
// The hyphenated form is what lets an import field be spelled `parent-realm` or
// `src-library`.
func (p *parser) parseAnnotationKey(context string) string {
	var sb strings.Builder

	if !p.atIdentifier() && !p.at(scanlex.KEYWORD) {
		p.failf(p.cur(), "expected a name %s, found %s", context, describeToken(p.cur()))
	}
	sb.WriteString(logicalName(p.advance().Value))

	for p.atOp("-") && p.isMemberNameToken(p.peek(1)) {
		p.advance() // "-"
		sb.WriteString("-")
		sb.WriteString(logicalName(p.advance().Value))
	}
	return sb.String()
}

// parseAnnotationValue parses the annotation-value production:
//
//	annotation-value = literal
//	                 | type-expression
//	                 | qualified-name
//	                 | declaration-reference
//	                 | annotation-list
//	                 | annotation-map
//	                 | annotation-arrow-pair
//
// Values decode to plain Go values: a string, a number, a bool, a []any for a
// list, or a map[string]any for a map. A name — including a type name or a
// declaration reference — decodes to its source spelling, which is what the
// semantic phase resolves later.
//
// DECISION-LIT-005 matters here: a bare `true`, `false` or `True` inside an
// annotation argument is an ordinary name and not a boolean literal, because
// FoLang's booleans are spelled co.const.true and co.const.false.
func (p *parser) parseAnnotationValue() any {
	switch {
	case p.at(scanlex.OPEN_BRACKET):
		return p.parseAnnotationList()
	case p.at(scanlex.OPEN_CURLY):
		return p.parseAnnotationMap()
	case p.at(scanlex.STRING):
		return p.parseAnnotationStringOrArrowPair()
	case p.at(scanlex.NUMBER):
		return numericValue(p.advance().Value)
	case p.at(scanlex.CHAR):
		return p.advance().Value
	case p.at(scanlex.BUILT_IN_CONSTANTS):
		return builtinConstantValue(p.advance().Value)
	case p.at(scanlex.DISCARD_WILD_VAR):
		return p.advance().Value
	case p.atOp("-"), p.atOp("+"):
		// A signed number, as in a negative precedence or offset.
		sign := p.advance().Value
		if p.at(scanlex.NUMBER) {
			return numericValue(sign + p.advance().Value)
		}
		p.failf(p.cur(), "expected a number after %q in an annotation value, found %s", sign, describeToken(p.cur()))
	}

	// Anything else is a name: a qualified name, a type expression, or a
	// declaration reference. All three decode to their source spelling.
	return p.parseAnnotationNameValue()
}

// parseAnnotationNameValue parses the name-shaped annotation values and returns
// the spelling.
//
// A declaration reference carries a signature — `find(co.lang.int)->(Employee)` —
// so the parenthesised part is consumed when present, and the whole reference is
// rendered back to text.
func (p *parser) parseAnnotationNameValue() any {
	qn := p.parseQualifiedName("as an annotation value")
	spelling := qn.Logical

	// A declaration reference or a type application: consume balanced groups so
	// the value keeps its full spelling.
	for p.at(scanlex.OPEN_PAREN) {
		start := p.pos
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		spelling += p.spellingOf(start, p.pos)
	}
	if p.at(scanlex.ARROW) {
		start := p.pos
		p.advance()
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		spelling += p.spellingOf(start, p.pos)
	}
	return spelling
}

// spellingOf renders tokens in [from, to) back to a source-like string. It is used
// only for annotation values, where a reference is kept as text for the semantic
// phase to resolve.
func (p *parser) spellingOf(from, to int) string {
	var sb strings.Builder
	for i := from; i < to && i < len(p.toks); i++ {
		sb.WriteString(logicalName(p.toks[i].Value))
	}
	return sb.String()
}

// parseAnnotationStringOrArrowPair parses a string value, or the
// annotation-arrow-pair production when a "=>" follows it:
//
//	annotation-arrow-pair = string-literal, "=>", string-literal
//
// The pair is decoded as a single-entry map so that it fits the same map[string]any
// shape as every other structured value.
func (p *parser) parseAnnotationStringOrArrowPair() any {
	left := unquote(p.advance().Value)

	if p.atOp("=>") {
		p.advance()
		right := unquote(p.expect(scanlex.STRING, "as the right side of an annotation arrow pair").Value)
		return map[string]any{left: right}
	}
	return left
}

// parseAnnotationList parses the annotation-list production:
//
//	annotation-list = "[", [ annotation-value,
//	                         { ",", annotation-value }, [ "," ] ], "]"
func (p *parser) parseAnnotationList() []any {
	p.expect(scanlex.OPEN_BRACKET, "to open an annotation list")

	values := []any{}
	for !p.at(scanlex.CLOSE_BRACKET) && !p.atEOF() {
		values = append(values, p.parseAnnotationValue())
		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_BRACKET, "to close an annotation list")
	return values
}

// parseAnnotationMap parses the annotation-map production:
//
//	annotation-map       = "{", [ annotation-map-entry,
//	                              { ",", annotation-map-entry }, [ "," ] ], "}"
//	annotation-map-entry = annotation-key, annotation-binder, annotation-value
//	                     | annotation-key
//
// A bare key is a flag whose value is the boolean true (DECISION-COL-001 with
// DECISION-ANN-001), which is what makes `{typename}` and
// `{variance:invariant, bound=Number}` and `{type=out}` all well formed.
func (p *parser) parseAnnotationMap() map[string]any {
	p.expect(scanlex.OPEN_CURLY, "to open an annotation map")

	entries := map[string]any{}
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		key := p.parseAnnotationKey("as an annotation map key")

		if p.atOp("=") || p.at(scanlex.COLON) {
			p.advance()
			entries[key] = p.parseAnnotationValue()
		} else {
			entries[key] = true // a bare key is a flag
		}

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close an annotation map")
	return entries
}

// numericValue decodes a numeric literal's lexeme.
//
// An integer decodes to int64 and a real to float64, so that a consumer reading an
// annotation option gets the type it expects. DECISION-LIT-000 keeps the original
// lexeme available on the token for anything that needs it verbatim.
func numericValue(lexeme string) any {
	if i, err := strconv.ParseInt(lexeme, 0, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(lexeme, 64); err == nil {
		return f
	}
	return lexeme
}

// builtinConstantValue decodes co.const.true, co.const.false and co.const.none
// (DECISION-LIT-005).
func builtinConstantValue(lexeme string) any {
	switch lexeme {
	case "co.const.true":
		return true
	case "co.const.false":
		return false
	default:
		return nil // co.const.none
	}
}

// unquote strips the surrounding double quotes from a string literal's lexeme. The
// scanner stores the complete lexeme including its delimiters
// (DECISION-LIT-000), so annotation values that are used as identifiers or paths
// need them removed.
func unquote(lexeme string) string {
	if len(lexeme) >= 2 && strings.HasPrefix(lexeme, `"`) && strings.HasSuffix(lexeme, `"`) {
		return lexeme[1 : len(lexeme)-1]
	}
	return lexeme
}
