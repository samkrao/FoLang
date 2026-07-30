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
//	use-directive              = "@co.ddap.use", "(", annotation-argument-list, ")"
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
// The import and alias directives get their own parse functions because their fields are
// fixed and worth checking; the rest share the general annotation-argument machinery.

// parseFileDirective parses one file-directive and returns the statement it produces.
func (p *parser) parseFileDirective() ast.Stmt {
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
	const (
		ddap = "@co.ddap."
		pdap = "@co.pdap."
	)
	return hasPrefix(directiveName, ddap) || hasPrefix(directiveName, pdap)
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
		if flag, isBool := value.(bool); isBool {
			stmt.SrcLibrary = flag
		} else {
			stmt.SrcLibrary = true
			stmt.From = text
		}
	case "expect":
		stmt.Expect = text
	case "as":
		stmt.Name = text
	case "realm":
		stmt.Realm = text
	case "parent-realm":
		stmt.ParentRealm = text
	case "parent":
		stmt.Parent = text
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
	start := p.cur()
	path := p.parseQualifiedName("as a co.* path").Logical

	if !hasPrefix(path, "co.") {
		p.reportf(start, "an alias target must be a co.* path, found %q", path)
	}
	return path
}

// parseUseDirective parses the use-directive production.
//
// A use directive brings a trait's or mixin's members into the current scope.
func (p *parser) parseUseDirective() ast.Stmt {
	directiveTok := p.advance()

	p.expect(scanlex.OPEN_PAREN, "to open a use directive")
	args := p.parseAnnotationArgumentList()
	p.expect(scanlex.CLOSE_PAREN, "to close a use directive")
	p.rejectDirectiveTerminator("@co.ddap.use")

	// The used names are grouped by their field so the semantic phase can tell a trait
	// from a mixin.
	used := map[string][]string{}
	name := ""
	for i, arg := range args {
		key := arg.Key
		if key == "" {
			key = "uses"
		}
		if text, isString := arg.Value.(string); isString {
			used[key] = append(used[key], text)
			if i == 0 {
				name = text
			}
			continue
		}
		if list, isList := arg.Value.([]any); isList {
			for _, item := range list {
				if text, isString := item.(string); isString {
					used[key] = append(used[key], text)
				}
			}
		}
	}

	return ast.UseStmtDirective{
		Name:   name,
		Type:   used,
		SDapst: (&annotationSet{byKind: map[scanlex.DirectiveKind][]ast.Stmt{}}).list(),
		Symb:   p.useSymbol(directiveTok.Value),
	}
}

// parseFilePreamble parses the file-preamble production:
//
//	file-preamble  = { file-directive }
//
// Every compilation unit begins with the same preamble, so this runs before the unit's form
// is decided.
func (p *parser) parseFilePreamble() []ast.Stmt {
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
