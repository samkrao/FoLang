package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/importcheck"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Built-in directives — section 2 of docs/grammar/folang.ebnf.
//
//	file-directive             = import-directive | alias-directive | use-directive
//	                           | dynamic-runtime-directive | pragma-directive
//	                           | generic-directive
//	import-directive           = "@co.ddap.import", "(", import-field,
//	                             { ",", import-field }, [ "," ], ")"
//	import-field               = ( "package" | "library" | "src-library" | "expect"
//	                             | "as" | "realm" | "parent-realm" ), "=",
//	                             annotation-value
//	alias-directive            = "@co.ddap.alias", "(", co-path, ",",
//	                             "as", "=", string-literal, [ "," ], ")"
//	use-directive              = "@co.ddap.use", "(", use-field,
//	                             { ",", use-field }, [ "," ], ")"
//	use-field                  = ( "from" | "methods" ), "=", annotation-value
//	dynamic-runtime-directive  = "@co.ddap.dynamicruntime",
//	                             [ "(", [ annotation-argument-list ], ")" ]
//	pragma-directive           = ( "@co.pdap.compiler" | "@co.pdap.scale" ),
//	                             [ "(", [ annotation-argument-list ], ")" ]
//	generic-directive          = "@co.ddap.", identifier,
//	                             [ "(", [ annotation-argument-list ], ")" ]
//
// DECISION-DIR-001: a built-in directive is SELF-DELIMITING. It ends at its complete
// directive form — the closing argument parenthesis when it has arguments — and no semicolon
// is accepted or required. That makes directives the deliberate exception to the mandatory
// statement terminator of DECISION-SYN-001.
//
// The import, alias and use directives get their own parse functions because their fields
// are fixed and worth checking; the rest share the general annotation-argument machinery.

// parseFileDirective parses one file-directive and returns the statement it produces.
func (p *parser) parseFileDirective() ast.Stmt {
	if traceEnabled {
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

	// dynamic-runtime-directive, pragma-directive and generic-directive all share the
	// shape `name [ "(" args ")" ]`, which parseAnnotation already handles.
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

// isFileDirectiveName reports whether a directive name belongs to a file-directive
// namespace.
func isFileDirectiveName(directiveName string) bool {
	switch directiveName {
	case "@co.ddap.import", "@co.ddap.alias", "@co.ddap.use",
		"@co.ddap.dynamicruntime", "@co.pdap.compiler", "@co.pdap.scale":
		return true
	}

	// generic-directive has exactly one identifier after @co.ddap.  Accepting a
	// prefix alone also admitted undocumented pragma names and deeper paths such
	// as @co.pdap.unknown and @co.ddap.foo.bar.
	const ddap = "@co.ddap."
	if !hasPrefix(directiveName, ddap) {
		return false
	}
	return isFoLangIdentifier(directiveName[len(ddap):])
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
//	@co.ddap.import(library="finance", src-library=co.const.true)
//
// When `as=` is present the API is reached through that alias; when it is omitted the
// complete imported path must be used.
func (p *parser) parseImportDirective() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	directiveTok := p.advance()

	p.expect(scanlex.OPEN_PAREN, "to open an import directive")

	stmt := ast.ImportStmt{Symb: p.directiveSymbol(directiveTok.Value, false)}

	for {
		field := p.parseAnnotationKey("as an import field name")
		p.expectOp("=", "after an import field name")
		value := p.parseAnnotationValue()

		p.assignImportField(&stmt, field, value, directiveTok)

		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_PAREN) {
			break // trailing comma
		}
	}

	closing := p.expect(scanlex.CLOSE_PAREN, "to close an import directive")
	p.rejectDirectiveTerminator("@co.ddap.import")

	// An import names exactly one subject. Neither leaves nothing to import, and both
	// leaves the resolver with two competing subjects, so each is reported here rather
	// than becoming a confusing downstream failure.
	switch {
	case stmt.Package == "" && stmt.From == "":
		p.reportf(directiveTok, "an import directive must name what it imports; add %q or %q", "package", "library")
	case stmt.Package != "" && stmt.From != "":
		p.reportf(directiveTok, "an import directive names one subject, but this one has both %q and %q; write a separate directive for each", "package", "library")
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
		Package:     stmt.Package,
		Library:     stmt.From,
		SrcLibrary:  stmt.SrcLibrary,
		Alias:       stmt.Name,
		Expect:      stmt.Expect,
		Realm:       stmt.Realm,
		ParentRealm: stmt.ParentRealm,
		Start:       start,
		End:         end,
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

// isBooleanValue reports whether an annotation value spells a boolean.
//
// Three spellings reach here: a real bool from a parsed boolean token, the
// co.const.true/false literals of DECISION-LIT-005, and a bare true/false, which the
// grammar treats as an ordinary annotation-value NAME and so delivers as a string.
func isBooleanValue(value any) bool {
	if _, isBool := value.(bool); isBool {
		return true
	}
	switch text, _ := value.(string); text {
	case "true", "false", "co.const.true", "co.const.false":
		return true
	}
	return false
}

// booleanValue decodes a value isBooleanValue has accepted.
func booleanValue(value any) bool {
	if flag, isBool := value.(bool); isBool {
		return flag
	}
	text, _ := value.(string)
	return text == "true" || text == "co.const.true"
}

// assignImportField records one import-field on the import statement.
//
// The field set is closed by the grammar, so an unrecognised name is reported rather than
// silently ignored.
func (p *parser) assignImportField(stmt *ast.ImportStmt, field string, value any, directiveTok scanlex.Token) {
	text, _ := value.(string)

	switch field {
	case "package":
		stmt.Package = text
	case "library":
		stmt.From = text
	case "src-library":
		// The flag may be written as a boolean constant or as a library name.
		//
		// DECISION-LIT-005 makes co.const.true/false the boolean literals, so a bare
		// true/false is an ordinary annotation-value NAME and arrives as a string.
		// Treating every non-boolean as a library name meant `src-library=false` both
		// set the flag to true and overwrote an already-parsed `library=` with the
		// text "false", so a documented spelling silently corrupted the import.
		// The reference defines src-library as a BOOLEAN flag that modifies how
		// package= resolves — "when true, package= resolves to a source library
		// surface file" — not as a subject of its own. Treating any non-boolean as a
		// library name let `src-library="finance"` import something the directive
		// never named, and overwrote an already-parsed library= with it.
		if !isBooleanValue(value) {
			p.reportf(directiveTok, "the %q field is a true/false flag that modifies %q, not a name; write %s to import a library",
				"src-library", "package", "library=\"…\"")
			return
		}
		stmt.SrcLibrary = booleanValue(value)
	case "expect":
		stmt.Expect = text
	case "as":
		// The alias becomes a name written in ordinary code — `emp.Employee` — so it
		// has to be spellable as one. Accepting any string let an import introduce a
		// name no source file could ever refer to.
		if !isFoLangIdentifier(text) {
			p.reportf(directiveTok, "the import alias %q is not a valid FoLang identifier; an alias is used as a name in code, so it starts with a letter and contains letters, digits and single underscores", text)
			return
		}
		stmt.Name = text
	case "realm":
		stmt.Realm = text
	case "parent-realm":
		stmt.ParentRealm = text
	default:
		p.reportf(directiveTok, "unknown import field %q; an import accepts package, library, src-library, expect, as, realm and parent-realm", field)
	}
}

// parseAliasDirective parses the alias-directive production.
//
// An alias gives a `co.*` path a file-local short name
// (docs/language-ref.md, "@co.ddap.alias"):
//
//	@co.ddap.alias(co.out, as="out")
//	@co.ddap.alias(co.core.list, as="list")
//
// Creating an alias does not hide the complete name: both forms stay valid in that file. The
// alias target must be a `co.*` path, which the grammar spells with co-path directly, so a
// target that is not one is reported.
func (p *parser) parseAliasDirective() ast.Stmt {
	if traceEnabled {
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

	p.accept(scanlex.COMMA) // optional trailing comma
	p.expect(scanlex.CLOSE_PAREN, "to close an alias directive")
	p.rejectDirectiveTerminator("@co.ddap.alias")

	return ast.DirectiveStmt{
		Name: directiveTok.Value,
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
func (p *parser) parseCoPath() string {
	if traceEnabled {
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
//	use-directive = "@co.ddap.use", "(", use-field, { ",", use-field }, [ "," ], ")"
//	use-field     = ( "from" | "methods" ), "=", annotation-value
//
// DECISION-SEM-002: a use directive ACTIVATES an extension unit or a typeclass instance,
// making its functions callable as methods on the receiver
// (docs/language-ref.md, "Using an Instance"):
//
//	@co.ddap.use(from="tu.stringextension", methods=[upperCase])   extension unit
//	@co.ddap.use(from="tc.ListFunctor", methods=[map, reduce])     typeclass instance
//
// The field list is CLOSED, matching import-field, so a mistyped key such as "method" is
// a parse error rather than an argument that is silently ignored. "from" names a
// declaration rather than a package, and "methods" is the only list attribute; omitting
// it activates everything the source provides.
//
// Which of the two "from" resolves to, the resolution order for a method call, and the
// one-activation-per-receiver rule are all semantic (DECISION-SEM-002), so this
// production accepts any combination of the two fields.
func (p *parser) parseUseDirective() ast.Stmt {
	if traceEnabled {
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
		value := p.parseAnnotationValue()

		p.assignUseField(used, &name, field, value, fieldTok)

		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_PAREN) {
			break // trailing comma
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a use directive")
	p.rejectDirectiveTerminator("@co.ddap.use")

	return ast.UseStmtDirective{
		Name:   name,
		Type:   used,
		SDapst: (&annotationSet{byKind: map[scanlex.DirectiveKind][]ast.Stmt{}}).list(),
		Symb:   p.useSymbol(directiveTok.Value),
	}
}

// assignUseField records one use-field, rejecting a key outside the closed set.
//
// "from" is a single declaration name and also becomes the directive's name, which is what
// the semantic phase resolves. "methods" is a list, but a lone name is accepted too so that
// activating one method needs no brackets.
func (p *parser) assignUseField(used map[string][]string, name *string, field string, value any, fieldTok scanlex.Token) {
	switch field {
	case "from":
		text, isString := value.(string)
		if !isString {
			p.reportf(fieldTok, "the %q field of a use directive names one declaration, so it takes a single name", "from")
			return
		}
		used["from"] = append(used["from"], text)
		if *name == "" {
			*name = text
		}
	case "methods":
		used["methods"] = append(used["methods"], useMethodNames(value)...)
	default:
		p.reportf(fieldTok, "unknown use field %q; a use directive accepts from and methods", field)
	}
}

// useMethodNames flattens the value of a "methods" field to the names it activates.
func useMethodNames(value any) []string {
	if text, isString := value.(string); isString {
		return []string{text}
	}
	list, isList := value.([]any)
	if !isList {
		return nil
	}
	names := make([]string, 0, len(list))
	for _, item := range list {
		if text, isString := item.(string); isString {
			names = append(names, text)
		}
	}
	return names
}

// parseFilePreamble parses the file-preamble production:
//
//	file-preamble  = { file-directive }
//
// Every compilation unit begins with the same preamble, so this runs before the unit's form
// is decided.
func (p *parser) parseFilePreamble() []ast.Stmt {
	if traceEnabled {
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
