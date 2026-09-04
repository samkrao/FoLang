package parser

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/src/ast"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
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
	// genericAliases are the declaration-local co.lang.type aliases introduced
	// by @co.dap.generic(aliases=[...]). They are kept out of DirectiveStmt's
	// plain-data payload: the public metadata retains the source spelling while
	// this side channel retains the parsed type tree needed for scope binding.
	genericAliases []genericContextAlias
	// at is the source position the run was read from. It is recorded even when
	// the run is EMPTY, because every declaration still carries a DirectveList
	// node and a node in the tree with no position is one an editor cannot place.
	// An empty run yields the zero-width span where an annotation would have
	// gone, which is both truthful and usable as a cursor target.
	at ast.Span
}

// empty reports whether no annotation was present.
func (a annotationSet) empty() bool {
	return len(a.all) == 0
}

// list returns the annotations as the ast.Stmt a node's directive field holds.
//
// The value is stored directly rather than through SetDap because every SetDap
// implementation in package ast takes a value receiver, so it mutates a copy and
// cannot reach the node being built.
func (a annotationSet) list() ast.Stmt {
	// The list covers the run of annotations it holds, from the first "@" to the
	// end of the last one. There is no parser cursor here, so the span is taken
	// from the annotations themselves.
	return &ast.DirectveList{NodeName: "DirectveList", Span: a.span(), Dapst: a.byKind}
}

// span returns the source region covered by the whole run of annotations, or
// the zero-width position the run was read from when it is empty.
func (a annotationSet) span() ast.Span {
	if len(a.all) == 0 {
		return a.at
	}
	return ast.NewSpan(a.all[0].GetSpan().Start, a.all[len(a.all)-1].GetSpan().End)
}

// has reports whether the set contains the named annotation, for example
// "@co.dap.hokrlt".
func (a annotationSet) has(name string) bool {
	for _, d := range a.all {
		if d.Name == name {
			return true
		}
	}
	return false
}

// rejectEffectsPlacement enforces the declaration half of the split effect
// syntax wherever the parser has established that the target is not callable.
func (p *parser) rejectEffectsPlacement(a annotationSet, target string) {
	if a.has("@co.dap.effects") {
		p.reportNamedf(p.cur(), helpers.DiagnosticInvalidEffectDeclaration, "Invalid Effect Declaration", "@co.dap.effects may decorate only a callable declaration or definition, not %s", target)
	}
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

// parseAnnotations parses the annotations production, a possibly empty run:
//
//	annotations          = { declaration-metadata }
//	declaration-metadata = annotation, declaration-metadata-category-guard
//
// Every element is a declaration-metadata, so each one passes the category guard
// before it is read. Sharing the `@qualified.name(...)` surface syntax with a
// file directive grants no placement permission: the guard is what decides
// whether the metadata application is legal in this position, which is why a
// directive cannot be smuggled into a nested scope by writing it where an
// annotation goes.
//
// Implements: annotations
// Implements: declaration-metadata
func (p *parser) parseAnnotations() annotationSet {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	set := annotationSet{
		byKind: map[scanlex.DirectiveKind][]ast.Stmt{},
		at:     p.spanOf(p.cur()),
	}
	seenOperator := false
	for p.atAnnotation() {
		annotationToken := p.cur()
		p.rejectMisplacedFileMetadata(annotationToken)
		d, aliases := p.parseAnnotation()
		if d.Name == "@co.dap.onEffect" {
			p.reportNamed(annotationToken, helpers.DiagnosticInvalidMetadataPlacement, "Invalid Metadata Placement", "@co.dap.onEffect is call-site metadata and may appear only immediately before a call expression")
		}
		if d.Name == "@co.dap.operator" {
			if seenOperator {
				p.reportNamed(annotationToken, helpers.DiagnosticConflictingMetadata, "Conflicting Metadata", "@co.dap.operator may appear at most once on a declaration")
			}
			seenOperator = true
		}
		set.all = append(set.all, d)
		set.genericAliases = append(set.genericAliases, aliases...)
		kind := directiveKindOf(d.Name)
		set.byKind[kind] = append(set.byKind[kind], d)
	}
	return set
}

// parseAnnotation parses one annotation:
//
//	annotation = "@", qualified-name, [ "(", [ annotation-argument-list ], ")" ]
//
// Positional arguments are keyed by their index so that Parameters can stay a
// single map, and a bare flag key is recorded with the boolean value true, as
// DECISION-ANN-001 specifies.
//
// The argument list is optional, and function-declaration puts an optional
// receiver-clause directly after the annotations, so an annotation with no arguments
// on a method is followed by a "(" that belongs to the RECEIVER:
//
//	@co.dap.public (emp Employee) fullLabel()->(co.lang.string) = { … }
//
// Claiming that group here would consume the receiver and then fail on the function
// name, so a group shaped like a receiver is left for the declaration to parse.
//
// Implements: annotation
func (p *parser) parseAnnotation() (ast.DirectiveStmt, []genericContextAlias) {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()
	annotationName := tok.Value
	if !isValidAnnotationName(annotationName) {
		p.reportf(tok, "invalid annotation name %q; every segment after @ must be a FoLang identifier", annotationName)
	}
	p.checkBuiltinMetadataName(tok, annotationName)

	params := map[string]any{}
	var parsedArgs []annotationArg
	var aliases []genericContextAlias
	if p.at(scanlex.OPEN_PAREN) && !p.atReceiverClause() {
		p.advance()
		if !p.at(scanlex.CLOSE_PAREN) {
			if annotationName == "@co.dap.generic" {
				parsedArgs, aliases = p.parseGenericAnnotationArgumentList()
			} else {
				parsedArgs = p.parseAnnotationArgumentList()
			}
			for i, arg := range parsedArgs {
				if arg.Key == "" {
					params[strconv.Itoa(i)] = arg.Value
					continue
				}
				params[arg.Key] = arg.Value
			}
		}
		p.expect(scanlex.CLOSE_PAREN, "to close an annotation argument list")
	}
	p.validateOperatorSymbolAnnotation(tok, annotationName, params, parsedArgs)
	p.validateEffectMetadata(tok, annotationName, params, parsedArgs)

	kind := directiveKindOf(annotationName)
	return ast.DirectiveStmt{NodeName: "DirectiveStmt", Span: p.spanFrom(spanStart), Name: annotationName,
		Parameters:      params,
		DirectiveType:   scanlex.KindToString[kind],
		DirectiveKind_:  scanlex.KindToPhase[kind],
		DirectiveScope_: scanlex.KindToScope[kind],
		Symb:            p.directiveSymbol(annotationName, kind == scanlex.PRAGMA),
	}, aliases
}

// genericContextAlias is the parser-only representation of one aliases= entry.
// Written is serialized in annotation metadata; Type is used exactly as the RHS
// of an ordinary co.lang.type alias when the decorated declaration scope opens.
type genericContextAlias struct {
	Name    string
	Written string
	Type    ast.Type
	Tok     scanlex.Token
}

// parseGenericAnnotationArgumentList is the sole annotation-value exception in
// the language: @co.dap.generic aliases records may carry a type expression.
// Every other argument continues through the ordinary annotation-value grammar.
func (p *parser) parseGenericAnnotationArgumentList() ([]annotationArg, []genericContextAlias) {
	var args []annotationArg
	var aliases []genericContextAlias
	for {
		start := p.cur()
		if p.atAnnotationKeyWithBinder() {
			key := p.parseAnnotationKey("as an annotation argument name")
			p.advance() // "="
			valueTok := p.cur()
			if key == "aliases" {
				value, parsed := p.parseGenericAliasList()
				args = append(args, annotationArg{Key: key, Value: value, KeyTok: start, ValueTok: valueTok})
				aliases = append(aliases, parsed...)
			} else {
				args = append(args, annotationArg{Key: key, Value: p.parseAnnotationValue(), KeyTok: start, ValueTok: valueTok})
			}
		} else {
			args = append(args, annotationArg{Value: p.parseAnnotationValue(), KeyTok: start, ValueTok: start})
		}
		if !p.accept(scanlex.COMMA) || p.at(scanlex.CLOSE_PAREN) {
			return args, aliases
		}
	}
}

func (p *parser) parseGenericAliasList() ([]any, []genericContextAlias) {
	p.expect(scanlex.OPEN_BRACKET, "to open @co.dap.generic aliases")
	var values []any
	var aliases []genericContextAlias
	seen := map[string]bool{}
	for !p.at(scanlex.CLOSE_BRACKET) {
		recordTok := p.cur()
		p.expect(scanlex.OPEN_CURLY, "to open a generic alias record")
		record := map[string]any{}
		alias := genericContextAlias{Tok: recordTok}
		fields := map[string]bool{}
		for !p.at(scanlex.CLOSE_CURLY) {
			fieldTok := p.cur()
			field := p.parseAnnotationKey("in a generic alias record")
			p.expectOp("=", "after a generic alias field")
			if fields[field] {
				p.reportf(fieldTok, "generic alias field %q occurs more than once", field)
			}
			fields[field] = true
			switch field {
			case "name":
				nameTok := p.cur()
				name := p.parseQualifiedName("as a generic alias name")
				if strings.Contains(name.Scanned, ".") {
					p.report(nameTok, "a generic-context alias name must be a local identifier")
				}
				alias.Name, alias.Tok, record[field] = name.Logical, nameTok, name.Logical
			case "type":
				if p.atKeyword("forall") && p.forallContextGuard() {
					p.fail(p.cur(), "a generic-context alias cannot introduce a forall binder; declare the polymorphic type with co.lang.type and refer to that alias")
				}
				start := p.pos
				alias.Type = p.parseTypeExpression().fullType()
				alias.Written = p.spellingOf(start, p.pos)
				record[field] = alias.Written
			default:
				p.reportf(fieldTok, "a generic alias record has no %q field; expected exactly name and type", field)
				record[field] = p.parseAnnotationValue()
			}
			if !p.accept(scanlex.COMMA) {
				break
			}
		}
		p.expect(scanlex.CLOSE_CURLY, "to close a generic alias record")
		if alias.Name == "" || alias.Type == nil {
			p.report(alias.Tok, "each generic alias record requires exactly name and type fields")
		} else {
			if seen[alias.Name] {
				p.reportf(alias.Tok, "generic alias %q occurs more than once", alias.Name)
			}
			seen[alias.Name] = true
			aliases = append(aliases, alias)
		}
		values = append(values, record)
		if !p.accept(scanlex.COMMA) {
			break
		}
	}
	p.expect(scanlex.CLOSE_BRACKET, "to close @co.dap.generic aliases")
	return values, aliases
}

// validateGenericAliasSignatureNames enforces the single declaration-signature
// namespace promised by the reference. Type lookup and value lookup remain
// separate generally; aliases= deliberately forbids shadowing a parameter or a
// named result because the alias is owned by this exact signature.
func (p *parser) validateGenericAliasSignatureNames(annotations annotationSet, params [][]ast.Parameter, results []ast.Returns) {
	if len(annotations.genericAliases) == 0 {
		return
	}
	names := map[string]bool{}
	for _, list := range params {
		for _, parameter := range list {
			if parameter.Symb != nil {
				names[logicalName(parameter.GetName())] = true
			}
		}
	}
	for _, result := range results {
		if result.IsNamed && result.Symb != nil {
			names[logicalName(result.GetName())] = true
		}
	}
	for _, alias := range annotations.genericAliases {
		if names[alias.Name] {
			p.reportf(alias.Tok, "generic alias %q duplicates a parameter or named result in the decorated declaration", alias.Name)
		}
	}
}

func (p *parser) validateEffectMetadata(tok scanlex.Token, name string, params map[string]any, args []annotationArg) {
	if name != "@co.dap.effects" && name != "@co.dap.onEffect" {
		return
	}
	seen := map[string]bool{}
	for _, arg := range args {
		if arg.Key == "" {
			p.reportf(arg.ValueTok, "%s accepts only named arguments", name)
			continue
		}
		if seen[arg.Key] {
			p.reportf(arg.KeyTok, "%s argument %q occurs more than once", name, arg.Key)
		}
		seen[arg.Key] = true
	}

	if name == "@co.dap.effects" {
		emits, ok := params["emits"].([]any)
		if !ok || len(emits) == 0 || len(params) != 1 {
			p.reportNamed(tok, helpers.DiagnosticInvalidEffectDeclaration, "Invalid Effect Declaration", "@co.dap.effects requires exactly one non-empty emits=[...] argument")
		}
		return
	}

	for effect, raw := range params {
		if !strings.Contains(effect, ".") {
			p.reportNamedf(tok, helpers.DiagnosticInvalidEffectPolicy, "Invalid Effect Policy", "@co.dap.onEffect key %q must be a qualified effect type name", effect)
			continue
		}
		record, ok := raw.(map[string]any)
		if !ok {
			p.reportNamedf(tok, helpers.DiagnosticInvalidEffectPolicy, "Invalid Effect Policy", "@co.dap.onEffect entry %q must be a record", effect)
			continue
		}
		resolution, hasResolution := record["resolution"].(string)
		handlers, hasHandlers := record["handlers"]
		if !hasResolution {
			list, listOK := handlers.([]any)
			if !hasHandlers || !listOK || len(list) == 0 {
				p.reportNamedf(tok, helpers.DiagnosticInvalidEffectResolution, "Invalid Effect Resolution", "@co.dap.onEffect entry %q requires either an ordinary-call resolution= value or a non-empty handlers list for execution-model call resolution", effect)
			}
			if _, hasRetry := record["retry"]; hasRetry {
				p.reportNamedf(tok, helpers.DiagnosticInvalidRetryPolicy, "Invalid Retry Policy", "@co.dap.onEffect entry %q cannot use retry without resolution=retry", effect)
			}
			for field := range record {
				if field != "handlers" {
					p.reportf(tok, "an execution-model @co.dap.onEffect candidate for %q has unsupported field %q", effect, field)
				}
			}
			continue
		}
		switch resolution {
		case "continue", "return", "return_with_error", "propagate":
			if _, hasRetry := record["retry"]; hasRetry {
				p.reportNamedf(tok, helpers.DiagnosticInvalidRetryPolicy, "Invalid Retry Policy", "@co.dap.onEffect entry %q may use retry= only with resolution=retry", effect)
			}
		case "retry":
			retry, retryOK := record["retry"].(map[string]any)
			attempts, attemptsOK := retry["max_attempts"].(int64)
			exhausted, exhaustedOK := retry["on_exhausted"].(string)
			if !retryOK || !attemptsOK || attempts <= 0 || !exhaustedOK || exhausted == "retry" ||
				(exhausted != "continue" && exhausted != "return" && exhausted != "return_with_error" && exhausted != "propagate") {
				p.reportNamedf(tok, helpers.DiagnosticInvalidRetryPolicy, "Invalid Retry Policy", "@co.dap.onEffect retry for %q requires retry={max_attempts=<positive integer>, on_exhausted=<continue|return|return_with_error|propagate>}", effect)
			}
		default:
			p.reportNamedf(tok, helpers.DiagnosticInvalidEffectResolution, "Invalid Effect Resolution", "@co.dap.onEffect entry %q has invalid resolution %q", effect, resolution)
		}
		if hasHandlers {
			list, listOK := handlers.([]any)
			if !listOK || len(list) == 0 {
				p.reportNamedf(tok, helpers.DiagnosticInvalidEffectHandler, "Invalid Effect Handler", "@co.dap.onEffect handlers for %q must be a non-empty list", effect)
			}
		}
		for field := range record {
			if field != "resolution" && field != "handlers" && field != "retry" {
				p.reportf(tok, "@co.dap.onEffect entry %q has unsupported field %q", effect, field)
			}
		}
	}
}

// checkBuiltinMetadataName classifies a metadata name the moment it has been
// read, which is what builtin-metadata-name-check requires.
//
// The classification is by NAMESPACE first and by registry second:
//
//   - a `@co.*` name must match the predefined built-in metadata registry
//     completely. The language owns that namespace, so an unregistered spelling
//     names nothing the language defines and there is no later phase that could
//     resolve it. It is a parse error rather than an accepted unknown.
//   - any other name is collected as a custom annotation or decorator
//     application and resolved later through the ordinary symbol table. FoLang
//     provides declaration constructs for user-defined annotations and decorators
//     only, so a custom directive or pragma has nothing to resolve to — but that
//     is a resolution rule, not one this parse can decide.
//
// The registry closes NAMES, not fields: a recognized form's complete field
// payload is parsed and preserved whatever it contains.
//
// Implements: builtin-metadata-name-check
func (p *parser) checkBuiltinMetadataName(tok scanlex.Token, annotationName string) {
	if !scanlex.IsLanguageOwnedMetadataName(annotationName) {
		return
	}
	if scanlex.IsBuiltinMetadataName(annotationName) {
		return
	}
	p.reportf(tok, "%q is not a built-in FoLang metadata name; the @co.* namespace is language-owned, so a name outside the predefined metadata registry cannot be applied", annotationName)
}

// isValidAnnotationName applies qualified-name's identifier rules to a name the
// scanner has already folded into one ATDAP/directive token. Without this check,
// single-segment spellings such as @_, @name_ and @a__b bypassed emitIdentifier.
func isValidAnnotationName(annotationName string) bool {
	if !strings.HasPrefix(annotationName, "@") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(annotationName, "@"), ".")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !isFoLangIdentifier(part) {
			return false
		}
	}
	return true
}

// validateOperatorSymbolAnnotation checks the lexical half of an operator
// declaration without changing the Pratt registry. A spelling containing a
// letter, digit, delimiter, or whitespace can never be emitted as an operator
// token, so accepting it would create an unusable declaration.
func (p *parser) validateOperatorSymbolAnnotation(tok scanlex.Token, annotationName string, params map[string]any, args []annotationArg) {
	if annotationName != "@co.dap.operator" {
		return
	}
	seen := map[string]bool{}
	for _, arg := range args {
		if arg.Key == "" {
			p.reportf(arg.ValueTok, "@co.dap.operator accepts only named symbol and mode arguments")
			continue
		}
		// The metadata registry closes form names, not their fields. Only fields
		// understood by this frontend are validated below; every other field stays
		// in DirectiveStmt.Parameters for later stages.
		if seen[arg.Key] {
			p.reportf(arg.KeyTok, "@co.dap.operator argument %q occurs more than once", arg.Key)
		}
		seen[arg.Key] = true
	}
	value, present := params["symbol"]
	if !present {
		return // the declaration validator reports the missing required option
	}
	symbol, isText := value.(string)
	if !isText || !scanlex.IsOperatorSpelling(symbol) {
		p.reportf(tok, "operator symbol %q is not a valid operator spelling", value)
		return
	}
	for _, arg := range args {
		if arg.Key != "symbol" {
			continue
		}
		if utf8.RuneCountInString(symbol) == 1 &&
			arg.ValueTok.Kind != scanlex.CHAR && arg.ValueTok.Kind != scanlex.STRING {
			p.reportf(arg.ValueTok, "one-character operator symbol %q must use a character or string literal", symbol)
		}
		if utf8.RuneCountInString(symbol) > 1 && arg.ValueTok.Kind != scanlex.STRING {
			p.reportf(arg.ValueTok, "multi-character operator symbol %q must use a string literal", symbol)
		}
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
	Key      string
	Value    any
	Flag     bool
	KeyTok   scanlex.Token
	ValueTok scanlex.Token
}

// parseAnnotationArgumentList parses the annotation-argument-list production:
//
//	annotation-argument-list = annotation-argument,
//	                           { ",", annotation-argument }, [ "," ]
//
// A trailing comma is permitted (DECISION-COL-001).
//
// Implements: annotation-argument-list
func (p *parser) parseAnnotationArgumentList() []annotationArg {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

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
//	annotation-binder   = "="
//
// The optional key group is decided by lookahead rather than by backtracking: a
// key is present only when a binder follows it. For a bare value such as
// `co.lang.int` there is no binder, so the whole group is skipped and the value is
// matched by annotation-value.
//
// Implements: annotation-argument
// Implements: declarative-attribute-binder
func (p *parser) parseAnnotationArgument() annotationArg {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	start := p.cur()

	if p.atAnnotationKeyWithBinder() {
		key := p.parseAnnotationKey("as an annotation argument name")
		p.advance() // "="
		valueTok := p.cur()
		return annotationArg{Key: key, Value: p.parseAnnotationValue(), KeyTok: start, ValueTok: valueTok}
	}

	return annotationArg{Value: p.parseAnnotationValue(), KeyTok: start, ValueTok: start}
}

// atAnnotationKeyWithBinder reports whether the cursor begins an annotation key
// followed by a binder.
//
// A key may itself contain hyphens, since annotation-key is
// `annotation-key-segment, { "-", annotation-key-segment }`, so the scan walks the
// whole `identifier { "-" identifier }` shape before testing for the binder.
func (p *parser) atAnnotationKeyWithBinder() bool {
	if !p.atIdentifier() && !p.at(scanlex.KEYWORD) && !strings.Contains(p.lexeme(), ".") {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance()
		for p.atOp("-") && p.isMemberNameToken(p.peek(1)) {
			p.advance()
			p.advance()
		}
		for p.at(scanlex.DOT) && p.isMemberNameToken(p.peek(1)) {
			p.advance()
			p.advance()
		}
		return p.atOp("=")
	})
}

// parseAnnotationKey parses the annotation-key production:
//
//	annotation-key         = annotation-key-segment, { "-", annotation-key-segment }
//	                       | qualified-name
//	annotation-key-segment = identifier | "for"
//
// A segment is an identifier or the keyword "for". "for" is admitted by name because
// several kind options are spelled with it — `co.lang.instance->(for=Functor)` — and it
// is a reserved word, so it never arrives as an identifier token.
//
// Implements: annotation-key
func (p *parser) parseAnnotationKey(context string) string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var sb strings.Builder
	if strings.Contains(p.lexeme(), ".") {
		return logicalName(p.advance().Value)
	}
	sb.WriteString(p.parseAnnotationKeySegment(context))

	for p.atOp("-") && p.atAnnotationKeySegment(p.peek(1)) {
		p.advance() // "-"
		sb.WriteString("-")
		sb.WriteString(p.parseAnnotationKeySegment(context))
	}
	for p.at(scanlex.DOT) && p.isMemberNameToken(p.peek(1)) {
		p.advance()
		sb.WriteString(".")
		sb.WriteString(logicalName(p.advance().Value))
	}
	return sb.String()
}

// parseAnnotationKeySegment consumes one annotation-key-segment.
//
// Implements: annotation-key-segment
func (p *parser) parseAnnotationKeySegment(context string) string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	if !p.atAnnotationKeySegment(p.cur()) {
		p.failf(p.cur(), "expected a name %s, found %s", context, describeToken(p.cur()))
	}
	return logicalName(p.advance().Value)
}

// atAnnotationKeySegment reports whether tok may spell an annotation-key-segment.
func (p *parser) atAnnotationKeySegment(tok scanlex.Token) bool {
	return tok.IsOneOfMany(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) ||
		logicalName(tok.Value) == "for"
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
//
// Implements: annotation-value
func (p *parser) parseAnnotationValue() any {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

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
		return annotationCharacterValue(p.advance().Value)
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
	if p.kindOptionDepth > 0 && (p.startsTypeExpression(p.cur()) || p.at(scanlex.OPEN_PAREN) || p.atKeyword("forall")) {
		start := p.pos
		p.parseTypeExpression()
		return p.spellingOf(start, p.pos)
	}

	// Anything else is either a qualified name/type-alias reference or an
	// explicitly signed declaration reference. Both decode to source spelling.
	return p.parseAnnotationNameValue()
}

// annotationCharacterValue converts a character literal into the textual value
// metadata consumers expect.
//
// Operator declarations in the normative reference use character syntax, for
// example `@co.dap.operator(symbol='+', mode=overload)`. Keeping the quotes in
// the annotation map would register the spelling "'+'" instead of "+", which
// can never match an operator token. Character-valued metadata is therefore
// represented as its unquoted source character, just like string-valued
// metadata is represented without double quotes.
func annotationCharacterValue(lexeme string) string {
	body := stripEncodingPrefix(lexeme)
	if len(body) >= 2 && strings.HasPrefix(body, "'") && strings.HasSuffix(body, "'") {
		return body[1 : len(body)-1]
	}
	return body
}

// parseAnnotationNameValue parses either a plain name/type-alias reference or
// the explicit overloaded-declaration reference
// `find(co.lang.int)->(Employee)`. Annotation values never contain anonymous
// type expressions: derived, function, forall and parameterized types must be
// named first with co.lang.type.
//
// Implements: annotation-declaration-reference
// Implements: annotation-reference-type
func (p *parser) parseAnnotationNameValue() any {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	start := p.pos
	if p.at(scanlex.OPEN_PAREN) || p.atKeyword("forall") {
		p.fail(p.cur(), "an inline type expression is not permitted as an annotation value; declare it with co.lang.type and use the alias")
	}

	p.parseQualifiedName("as an annotation value")
	if !p.at(scanlex.OPEN_PAREN) {
		if p.at(scanlex.ARROW) {
			p.fail(p.cur(), "an inline derived type is not permitted as an annotation value; declare it with co.lang.type and use the alias")
		}
		return p.spellingOf(start, p.pos)
	}

	// A parenthesized suffix is reserved here for an overload signature. Each
	// signature entry is one named type use; the following arrow makes the form
	// explicit and prevents it from being confused with a parameterized type.
	p.advance()
	if !p.at(scanlex.CLOSE_PAREN) {
		p.parseNamedTypeAtom()
		for p.accept(scanlex.COMMA) {
			p.parseNamedTypeAtom()
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a declaration-reference parameter list")
	if !p.accept(scanlex.ARROW) {
		p.fail(p.cur(), "a parameterized type expression is not permitted as an annotation value; use a co.lang.type alias, or add '->(...)' when identifying an overloaded declaration")
	}
	p.expect(scanlex.OPEN_PAREN, "to open a declaration-reference result list")
	if !p.at(scanlex.CLOSE_PAREN) {
		p.parseNamedTypeAtom()
		for p.accept(scanlex.COMMA) {
			p.parseNamedTypeAtom()
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a declaration-reference result list")
	return p.spellingOf(start, p.pos)
}

// spellingOf renders tokens in [from, to) back to a source-like string. It is used
// only for annotation values, where a reference is kept as text for the semantic
// phase to resolve.
func (p *parser) spellingOf(from, to int) string {
	var sb strings.Builder
	for i := from; i < to && i < len(p.toks); i++ {
		part := logicalName(p.toks[i].Value)
		if annotationSpellingNeedsSpace(sb.String(), part) {
			sb.WriteByte(' ')
		}
		sb.WriteString(part)
	}
	return sb.String()
}

// annotationSpellingNeedsSpace keeps adjacent name tokens distinct when an
// annotation value is reconstructed from the whitespace-free token stream.
// Punctuation already separates every other token pair, but `name Type` would
// otherwise collapse to `nameType` and change the function/type signature
// delivered to the semantic phase.
func annotationSpellingNeedsSpace(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	last, _ := utf8.DecodeLastRuneInString(left)
	first, _ := utf8.DecodeRuneInString(right)
	return annotationWordRune(last) && annotationWordRune(first)
}

func annotationWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$'
}

// parseAnnotationStringOrArrowPair parses a string value, or the
// annotation-arrow-pair production when a "=>" follows it:
//
//	annotation-arrow-pair = string-literal, "=>", string-literal
//
// The pair is decoded as a single-entry map so that it fits the same map[string]any
// shape as every other structured value.
//
// Implements: annotation-arrow-pair
func (p *parser) parseAnnotationStringOrArrowPair() any {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
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
//
// Implements: annotation-list
func (p *parser) parseAnnotationList() []any {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

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
//	annotation-map-entry = annotation-map-key, annotation-binder, annotation-value
//	annotation-map-key   = annotation-key | qualified-name
//
// A map key is wider than an argument key: annotation-map-key admits a QUALIFIED
// NAME as well, which is what lets the packaged export selector name package
// paths as its keys — `@co.dap.export(packages={hr.employee={recurse=true}})`.
// An argument key never needs that, because an argument names a field of the
// metadata form rather than something in the program.
//
// Implements: annotation-map
func (p *parser) parseAnnotationMap() map[string]any {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_CURLY, "to open an annotation map")

	entries := map[string]any{}
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		keyTok := p.cur()
		key := p.parseAnnotationMapKey()

		if p.kindOptionDepth > 0 && p.at(scanlex.COLON) {
			p.advance() // ordinary kind-option map entry separator
		} else {
			p.expectOp("=", "between an annotation map key and value")
		}
		value := p.parseAnnotationValue()
		if _, duplicate := entries[key]; duplicate {
			p.reportf(keyTok, "annotation map key %q occurs more than once", key)
		}
		entries[key] = value

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close an annotation map")
	return entries
}

// parseAnnotationMapKey parses the annotation-map-key production:
//
//	annotation-map-key = annotation-key | qualified-name
//
// The two overlap on their first segment, so the hyphenated annotation-key shape
// is read first and a "." tail is then absorbed. A dotted key is one qualified
// name and not an entry per segment, which is what keeps
// `packages={hr.employee={recurse=true}}` a single selector entry.
//
// Implements: annotation-map-key
func (p *parser) parseAnnotationMapKey() string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	key := p.parseAnnotationKey("as an annotation map key")
	for p.at(scanlex.DOT) && p.isMemberNameToken(p.peek(1)) {
		p.advance() // "."
		key += "." + logicalName(p.advance().Value)
	}
	return key
}

// numericValue decodes a numeric literal's lexeme.
//
// An integer decodes to int64 and a real to float64, so that a consumer reading an
// annotation option gets the type it expects. DECISION-LIT-000 keeps the original
// lexeme available on the token for anything that needs it verbatim.
func numericValue(lexeme string) any {
	if isFloatingLexeme(lexeme) {
		if f, ok := parseFloatLexeme(lexeme); ok {
			return f
		}
		return lexeme
	}

	// strconv does not accept C++ integer suffixes. The scanner has already
	// validated their shape, so remove them before decoding metadata such as an
	// operator's `precedence=65u`.
	trimmed := strings.TrimRight(lexeme, "uUlLzZ")
	if i, err := strconv.ParseInt(trimmed, 0, 64); err == nil {
		return i
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
