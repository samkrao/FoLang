package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/importcheck"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Built-in directives — section 2 of docs/grammar/folang.ebnf.
//
//	file-directive             = import-directive | alias-directive | use-directive
//	                           | dynamic-runtime-directive
//	                           | dynamic-dispatch-directive
//	                           | other-file-metadata-directive
//	import-directive           = "@co.ddap.import", "(", import-field,
//	                             { ",", import-field }, ")"
//	import-field               = "package", "=", string-literal
//	                           | "library", "=", string-literal
//	                           | "src-library", "=", "true"
//	                           | "as", "=", string-literal
//	alias-directive            = "@co.ddap.alias", "(", co-path, ",",
//	                             "as", "=", string-literal, ")"
//	use-directive              = "@co.ddap.use", "(", use-field,
//	                             { ",", use-field }, ")"
//	use-field                  = "from", "=", string-literal
//	                           | "methods", "=", "[", [ identifier,
//	                                                    { ",", identifier } ], "]"
//	dynamic-runtime-directive  = "@co.ddap.dynamicruntime",
//	                             [ "(", [ annotation-argument-list ], ")" ]
//
// A built-in directive is SELF-DELIMITING. It ends at its complete directive form — the
// closing argument parenthesis when it has arguments — and no semicolon is accepted or
// required. That makes directives the deliberate exception to the mandatory statement
// terminator.
//
// file-directive is closed by the built-in metadata REGISTRY rather than by a
// namespace prefix. other-file-metadata-directive admits any name the registry
// lists as a DIRECTIVE or a PRAGMA, and its zero-width guard is exactly that
// lookup; an unregistered `@co.*` spelling is a parse error rather than an
// ordinary user annotation (docs/grammar/folang.ebnf, file-directive-category-guard
// and builtin-metadata-name-check). A `@co.dap.*` annotation is NOT a file
// directive: it decorates the declaration that follows it.
//
// The registry closes form NAMES, not fields. Every field of a recognized form is
// parsed and preserved; the frontend validates the fields it knows and leaves the
// rest available to later stages.
//
// The import, alias and use directives get their own parse functions because their fields
// are fixed and typed; the remaining registered forms share the general annotation-argument
// machinery.

// parseFileDirective parses one file-directive and returns the statement it produces.
//
// Implements: file-directive
func (p *parser) parseFileDirective() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	directiveName := p.lexeme()

	switch directiveName {
	case "@co.ddap.import":
		return p.parseImportDirective()
	case "@co.ddap.alias":
		return p.parseAliasDirective()
	case "@co.ddap.use":
		return p.parseUseDirective()
	}

	// dynamic-runtime-directive, dynamic-dispatch-directive and every other
	// registered directive/pragma share the `name [ "(" args ")" ]` shape, which
	// parseAnnotation already handles. Their fields are open by design: the
	// registry closes the form's NAME, and unknown fields of a known form stay
	// collected and preserved for later stages.
	directive := p.parseAnnotation()
	p.rejectDirectiveTerminator(directiveName)
	return directive
}

// rejectDirectiveTerminator reports a semicolon written after a self-delimiting directive.
//
// The ";" is consumed after reporting, so the enclosing item loop stays in step rather than
// stalling on a token nothing will accept.
func (p *parser) rejectDirectiveTerminator(directiveName string) {
	if !p.at(scanlex.SEMI_COLON) {
		return
	}
	p.reportf(p.cur(), "unexpected %q after %s; a built-in directive is self-delimiting and takes no terminator", ";", directiveName)
	p.advance()
}

// atFileDirective reports whether the cursor begins a file-directive.
//
// Only the built-in directive namespaces qualify. A @co.dap.* annotation decorates a
// declaration rather than standing alone, so it is not a file directive.
func (p *parser) atFileDirective() bool {
	if !p.atAny(scanlex.BUILT_IN_DIRECTIVES, scanlex.CUSTOM_DIRECTIVES, scanlex.ATDAP) {
		return false
	}
	return isFileDirectiveName(p.lexeme())
}

// isFileDirectiveName reports whether a directive name is a file-directive.
//
// This is file-directive-category-guard: the name has to be registered as a
// built-in DIRECTIVE or PRAGMA. The set is closed by the REGISTRY, not by a
// namespace prefix — a `@co.ddap.` prefix test used to stand in for the withdrawn
// generic-directive and admitted any spelling the reference never defines.
//
// Implements: file-directive-category-guard
// Implements: other-file-metadata-directive
func isFileDirectiveName(directiveName string) bool {
	return scanlex.IsBuiltinDirectiveMetadataName(directiveName) ||
		scanlex.IsBuiltinPragmaMetadataName(directiveName)
}

// isImportDirectiveName reports whether a name is the import directive.
//
// Implements: import-directive-guard
func isImportDirectiveName(directiveName string) bool {
	return directiveName == "@co.ddap.import"
}

// isAliasDirectiveName reports whether a name is the alias directive.
//
// Implements: alias-directive-guard
func isAliasDirectiveName(directiveName string) bool {
	return directiveName == "@co.ddap.alias"
}

// isUseDirectiveName reports whether a name is the use directive.
//
// Implements: use-directive-guard
func isUseDirectiveName(directiveName string) bool {
	return directiveName == "@co.ddap.use"
}

// isDynamicRuntimeDirectiveName reports whether a name is the dynamic-runtime
// directive.
//
// Implements: dynamic-runtime-directive-guard
func isDynamicRuntimeDirectiveName(directiveName string) bool {
	return directiveName == "@co.ddap.dynamicruntime"
}

// isDynamicDispatchDirectiveName reports whether a name is the dynamic-dispatch
// directive.
//
// Implements: dynamic-dispatch-directive
// Implements: dynamic-dispatch-directive-guard
func isDynamicDispatchDirectiveName(directiveName string) bool {
	return directiveName == "@co.ddap.dynamicdispatch"
}

// validateCompilationUnitDirectives enforces the one file-directive rule that
// cannot be decided while parsing the common preamble.  The unit kind is known
// only after the preamble has been consumed, so validation scans the retained
// token stream immediately after classification.
func (p *parser) validateCompilationUnitDirectives() {
	if p.unit == unitEntry {
		return
	}
	for _, tok := range p.toks {
		if tok.Value == "@co.ddap.dynamicruntime" {
			p.reportf(tok, "%s is allowed only in an application entry file", tok.Value)
		}
	}
}

// hasPrefix reports whether s starts with prefix.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// parseImportDirective parses the import-directive production.
//
// An import is how a package or library becomes visible: user packages and libraries are not
// automatically available, unlike the `co.*` paths
// (docs/language-ref.md, "Built-in and Imported Names"):
//
//	@co.ddap.import(package="hr.employee", as="emp")
//	@co.ddap.import(library="ffi", src-library=true, as="ffilib")
//
// When `as=` is present the API is reached through that alias; when it is omitted the
// complete imported path must be used.
//
// Every field is TYPED by the grammar rather than taking a general annotation-value:
// package, library and as each take a string-literal, and src-library takes the literal
// `true` and nothing else (docs/language-ref.md, "Import Directive Fields").
//
// Implements: import-directive
// Implements: import-field
func (p *parser) parseImportDirective() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	directiveTok := p.advance()

	p.expect(scanlex.OPEN_PAREN, "to open an import directive")

	stmt := ast.ImportStmt{Span: p.spanFrom(spanStart), ExtraFields: map[string]any{}, Symb: p.directiveSymbol(directiveTok.Value, false)}

	sawSrcLibrary := false
	for {
		fieldTok := p.cur()
		field := p.parseAnnotationKey("as an import field name")
		p.expectOp("=", "after an import field name")

		p.parseImportField(&stmt, field, fieldTok, directiveTok, &sawSrcLibrary)

		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_PAREN) {
			p.fail(p.cur(), "a comma in an import directive must be followed by another import field; trailing commas are not allowed")
		}
	}

	closing := p.expect(scanlex.CLOSE_PAREN, "to close an import directive")
	p.rejectDirectiveTerminator("@co.ddap.import")

	// An import names exactly one subject. Neither leaves nothing to import, and both
	// leaves the resolver with two competing subjects, so each is reported here rather
	// than becoming a confusing downstream failure.
	switch {
	case importTargetCount(stmt) == 0:
		p.reportf(directiveTok, "an import directive must name what it imports; add %q, %q or %q", "package", "library", "component")
	case importTargetCount(stmt) > 1:
		p.reportf(directiveTok, "an import directive names exactly one of %q, %q or %q; write a separate directive for each target", "package", "library", "component")
	}

	// src-library modifies how library= resolves, so it is meaningless without one
	// (docs/language-ref.md, "Import Directive Fields").
	if sawSrcLibrary && stmt.From == "" {
		p.reportf(directiveTok, "%q is valid only together with %q; it selects the project-local srclib/ source library that %s names", "src-library", "library", "library=")
	}

	// A source-library import means the library has to be built from source before its
	// consumers, which the driver needs to know.
	if stmt.SrcLibrary {
		p.buildLibs = true
	}

	// Record the edge for the import-relationship checks, which need the directive's
	// position and so must capture it here while the tokens are in hand.
	p.recordImport(stmt, directiveTok, closing)

	return stmt
}

// recordImport captures an import directive for the importcheck phase.
//
// ast.ImportStmt carries no source position, so the span is taken from the directive's own
// tokens as they are consumed. Without this the cycle and restricted-import diagnostics would
// have nowhere to point.
func (p *parser) recordImport(stmt ast.ImportStmt, directiveTok, closing scanlex.Token) {
	start, _ := tokenSpan(directiveTok)
	_, end := tokenSpan(closing)

	p.imports = append(p.imports, importcheck.Import{
		Package:    stmt.Package,
		Library:    stmt.From,
		Component:  stmt.Component,
		SrcLibrary: stmt.SrcLibrary,
		Alias:      stmt.Name,
		Start:      start,
		End:        end,
	})
}

// isFoLangIdentifier reports whether text is spellable as a FoLang identifier.
//
// This is the grammar's identifier: an ASCII letter first, then letters, digits and
// single underscores, never two in a row and never one at the end
// (DECISION-LEX-001/006). The scanner enforces it for names written in source; a name
// arriving as a directive's string argument bypasses the scanner, so it is checked here.
func isFoLangIdentifier(text string) bool {
	if text == "" {
		return false
	}
	if !isASCIILetter(text[0]) {
		return false
	}
	if text[len(text)-1] == '_' {
		return false
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case isASCIILetter(c) || (c >= '0' && c <= '9'):
		case c == '_':
			if i+1 < len(text) && text[i+1] == '_' {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// parseImportField parses one import-field's value and records it on the import
// statement.
//
// Each alternative of import-field carries its own value grammar, so the value is
// parsed per field rather than as a general annotation-value. That is what makes
// `src-library="finance"` a syntax error at the point of the value instead of a
// well-formed directive that silently imports something it never named.
//
// The field set is closed, so an unrecognised name is reported rather than ignored.
func (p *parser) parseImportField(stmt *ast.ImportStmt, field string, fieldTok, directiveTok scanlex.Token, sawSrcLibrary *bool) {
	switch field {
	case "package":
		stmt.Package = p.parseImportStringField("package")
	case "library":
		stmt.From = p.parseImportStringField("library")
	case "component":
		stmt.Component = p.parseImportStringField("component")
	case "src-library":
		// The grammar admits the single literal `true`. The field is a flag that
		// switches library= resolution from lib/ to srclib/, so `false` says
		// nothing the directive does not already say by omitting it, and a string
		// would name a library the directive never imports.
		// The scanner lowers an ordinary identifier, so the bare `true` arrives as a
		// lowered name and is compared by its logical spelling.
		*sawSrcLibrary = true
		if logicalName(p.lexeme()) != "true" {
			p.reportf(p.cur(), "the %q field takes the literal %s and nothing else; omit the field to resolve %s from the packaged-library domain",
				"src-library", "true", "library=")
			p.advance()
			return
		}
		p.advance()
		stmt.SrcLibrary = true
	case "as":
		// The alias becomes a name written in ordinary code — `emp.Employee` — so it
		// has to be spellable as one. Accepting any string let an import introduce a
		// name no source file could ever refer to.
		alias := p.parseImportStringField("as")
		if !isFoLangIdentifier(alias) {
			p.reportf(directiveTok, "the import alias %q is not a valid FoLang identifier; an alias is used as a name in code, so it starts with a letter and contains letters, digits and single underscores", alias)
			return
		}
		stmt.Name = alias
	default:
		// Known metadata forms preserve fields this frontend does not yet handle.
		stmt.ExtraFields[field] = p.parseAnnotationValue()
	}
}

func importTargetCount(stmt ast.ImportStmt) int {
	count := 0
	for _, target := range []string{stmt.Package, stmt.From, stmt.Component} {
		if target != "" {
			count++
		}
	}
	return count
}

// parseImportStringField reads the string-literal an import-field alternative takes.
func (p *parser) parseImportStringField(field string) string {
	return unquote(p.expect(scanlex.STRING, "as the value of the import field \""+field+"\"").Value)
}

// parseAliasDirective parses the alias-directive production.
//
// An alias gives a `co.*` path a file-local short name
// (docs/language-ref.md, "@co.ddap.alias"):
//
//	@co.ddap.alias(co.out, as="out")
//	@co.ddap.alias(co.core.List, as="list")
//
// Creating an alias does not hide the complete name: both forms stay valid in that file. The
// alias target must be a `co.*` path, which the grammar spells with co-path directly, so a
// target that is not one is reported.
//
// Implements: alias-directive
func (p *parser) parseAliasDirective() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	directiveTok := p.advance()

	p.expect(scanlex.OPEN_PAREN, "to open an alias directive")

	target := p.parseCoPath()
	p.expect(scanlex.COMMA, "after the target of an alias directive")

	aliasKey := p.parseAnnotationKey("as the \"as\" field of an alias directive")
	if aliasKey != "as" {
		p.reportf(directiveTok, "an alias directive's second field must be %q, found %q", "as", aliasKey)
	}
	p.expectOp("=", "after \"as\" in an alias directive")
	aliasName := unquote(p.expect(scanlex.STRING, "as the alias name").Value)
	// The alias is referenced as an ordinary source name, so a string that the
	// scanner could never tokenize as one would create an unusable binding.
	if !isFoLangIdentifier(aliasName) {
		p.reportf(directiveTok, "the alias name %q is not a valid FoLang identifier; it must start with a letter and contain letters, digits and single underscores", aliasName)
	}

	if p.at(scanlex.COMMA) {
		p.fail(p.cur(), "an alias directive has exactly two fields; trailing commas are not allowed")
	}
	p.expect(scanlex.CLOSE_PAREN, "to close an alias directive")
	p.rejectDirectiveTerminator("@co.ddap.alias")

	return ast.DirectiveStmt{Span: p.spanFrom(spanStart), Name: directiveTok.Value,
		Parameters: map[string]any{
			"target": target,
			"as":     aliasName,
		},
		DirectiveType:   scanlex.KindToString[scanlex.DIRECTIVE],
		DirectiveKind_:  scanlex.KindToPhase[scanlex.DIRECTIVE],
		DirectiveScope_: scanlex.KindToScope[scanlex.DIRECTIVE],
		Symb:            p.directiveSymbol(directiveTok.Value, false),
	}
}

// parseCoPath parses the co-path production:
//
//	co-path = "co", ".", identifier, { ".", identifier }
//
// The scanner folds a `co.*` path into a single token, so this normally consumes one token
// and then verifies the prefix.
//
// Implements: co-path
func (p *parser) parseCoPath() string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	start := p.cur()
	path := p.parseQualifiedName("as a co.* path").Logical

	if !hasPrefix(path, "co.") {
		p.reportf(start, "an alias target must be a co.* path, found %q", path)
	}
	return path
}

// parseUseDirective parses the use-directive production:
//
//	use-directive = "@co.ddap.use", "(", use-field, { ",", use-field }, ")"
//	use-field     = "from", "=", string-literal
//	              | "methods", "=", "[", [ identifier, { ",", identifier } ], "]"
//
// A use directive ACTIVATES an extension unit or a typeclass instance,
// making its functions callable as methods on the receiver
// (docs/language-ref.md, "Using an Instance"):
//
//	@co.ddap.use(from="tu.stringextension", methods=[upperCase])   extension unit
//	@co.ddap.use(from="tc.ListFunctor", methods=[map, reduce])     typeclass instance
//
// The field list is CLOSED, matching import-field, so a mistyped key such as "method" is
// a parse error rather than an argument that is silently ignored. "from" names a
// declaration rather than a package and takes a string-literal; "methods" takes a
// BRACKETED list of identifiers and nothing else, so a bare `methods=upperCase` is a
// syntax error rather than a silently accepted one-element shorthand. Omitting "methods"
// activates everything the source provides.
//
// Which of the two "from" resolves to, the resolution order for a method call, and the
// one-activation-per-receiver rule are all semantic, so this production accepts any
// combination of the two fields.
//
// Implements: use-directive
// Implements: use-field
func (p *parser) parseUseDirective() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	directiveTok := p.advance()

	p.expect(scanlex.OPEN_PAREN, "to open a use directive")

	// Grouped by field so the semantic phase can tell the activation source from the
	// method list it selects.
	used := map[string][]string{}
	name := ""

	for {
		fieldTok := p.cur()
		field := p.parseAnnotationKey("as a use field name")
		p.expectOp("=", "after a use field name")

		p.parseUseField(used, &name, field, fieldTok)

		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_PAREN) {
			p.fail(p.cur(), "a comma in a use directive must be followed by another use field; trailing commas are not allowed")
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a use directive")
	p.rejectDirectiveTerminator("@co.ddap.use")

	// A use directive carries no annotations of its own, but the node still holds
	// a directive list, so the empty one is anchored at the directive rather than
	// left position-less.
	empty := annotationSet{
		byKind: map[scanlex.DirectiveKind][]ast.Stmt{},
		at:     p.spanFrom(spanStart),
	}
	return ast.UseStmtDirective{Span: p.spanFrom(spanStart), Name: name,
		Type:   used,
		SDapst: empty.list(),
		Symb:   p.useSymbol(directiveTok.Value),
	}
}

// parseUseField parses one use-field's value, rejecting a key outside the closed set.
//
// "from" is a single declaration name and also becomes the directive's name, which is what
// the semantic phase resolves. "methods" is a bracketed identifier list; the brackets are
// part of the grammar rather than an optional flourish around a single name.
func (p *parser) parseUseField(used map[string][]string, name *string, field string, fieldTok scanlex.Token) {
	switch field {
	case "from":
		text := unquote(p.expect(scanlex.STRING, "as the value of the use field \"from\"").Value)
		used["from"] = append(used["from"], text)
		if *name == "" {
			*name = text
		}
	case "methods":
		used["methods"] = append(used["methods"], p.parseUseMethodList()...)
	default:
		p.reportf(fieldTok, "unknown use field %q; a use directive accepts from and methods", field)
		p.parseAnnotationValue()
	}
}

// parseUseMethodList parses the bracketed identifier list of a "methods" field.
//
// The list may be empty — `methods=[]` activates nothing — but it is always bracketed,
// and like every other list in a directive it takes no trailing comma.
func (p *parser) parseUseMethodList() []string {
	p.expect(scanlex.OPEN_BRACKET, "to open the method list of a use directive")

	var names []string
	for !p.at(scanlex.CLOSE_BRACKET) && !p.atEOF() {
		names = append(names, p.parseIdentifier("as an activated method name").Logical)
		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_BRACKET) {
			p.fail(p.cur(), "a comma in a use directive's method list must be followed by another method name; trailing commas are not allowed")
		}
	}

	p.expect(scanlex.CLOSE_BRACKET, "to close the method list of a use directive")
	return names
}

// parseFilePreamble parses the file-preamble production:
//
//	file-preamble  = { file-directive }
//
// Every compilation unit begins with the same preamble, so this runs before the unit's form
// is decided.
//
// Implements: file-preamble
func (p *parser) parseFilePreamble() []ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var directives []ast.Stmt

	for p.atFileDirective() {
		startPos := p.pos

		var directive ast.Stmt
		ok := p.recoverItem(startPos, syncStatement, func() {
			directive = p.parseFileDirective()
		})
		if ok && directive != nil {
			directives = append(directives, directive)
		}
	}
	return directives
}
