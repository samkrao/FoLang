package parser

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// validAliasIdentifier implements the "Valid `as` Values" rule from the
// language reference: a FoLang identifier with no dots, starting with a
// letter or underscore.
var validAliasIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validExpectKinds lists the library kinds accepted by @co.ddap.import's
// expect= assertion.
var validExpectKinds = []string{"ffi", "system", "advanced", "dynamicvmrt", "application"}

// parse_import_declaration_stmt converts every @co.ddap.import directive
// already collected in ddaps[scanlex.DIRECTIVE] into a validated
// ast.ImportStmt.
//
// Feature examples:
//
//	@co.ddap.import(package="hr.employee", as="emp")
//	@co.ddap.import(package="com.abc.ffi", src-library=true, expect="ffi", as="ffilib")
//	@co.ddap.import(library="hrlib", as="hr")
//
// Only the rules a single directive (or the set of import directives in this
// file) can decide are checked here — required package/library, "as" shape,
// src-library/expect values, and same-realm alias collisions. Cross-file
// rules such as import cycles need whole-program visibility and belong to
// the semantic analyzer.
func parse_import_declaration_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	imports := make([]ast.Stmt, 0)
	// realmAlias tracks the package/library bound to a given (realm, alias)
	// pair. Re-declaring the same alias in the same realm for the same
	// package/library is valid aggregation; a different package/library is
	// an invalid collision. A different realm is coexistence or shadowing,
	// neither of which collides here.
	realmAlias := map[string]string{}

	for _, stmt := range ddaps[scanlex.DIRECTIVE] {
		dir, ok := stmt.(ast.DirectiveStmt)
		if !ok || dir.Name != "@co.ddap.import" {
			continue
		}
		imports = append(imports, build_import_stmt(p, dir, realmAlias))
	}

	pr.Node = ast.CodeStmt{Body: imports}
	return pr
}

// build_import_stmt validates one @co.ddap.import directive and converts it
// into an ast.ImportStmt. Errors are reported directly through p.addErr so
// callers that discard the ParseResult (as parse_co_stmts does) still see
// them.
func build_import_stmt(p *parser, dir ast.DirectiveStmt, realmAlias map[string]string) ast.ImportStmt {
	pkg, hasPkg := importDirectiveParam(dir, "package")
	lib, hasLib := importDirectiveParam(dir, "library")
	if hasPkg == hasLib {
		p.addErr(p.errorExpection(
			"@co.ddap.import requires exactly one of package= or library=",
			helpers.InvalidSyntax,
		))
	}

	as, hasAs := importDirectiveParam(dir, "as")
	if hasAs && !validAliasIdentifier.MatchString(as) {
		p.addErr(p.errorExpection(
			"invalid alias as=\""+as+"\"; alias must be a valid FoLang identifier without dots",
			helpers.InvalidSyntax,
		))
	}

	srcLibrary := false
	if raw, hasSrcLib := importDirectiveParam(dir, "src-library"); hasSrcLib {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			p.addErr(p.errorExpection(
				"invalid src-library=\""+raw+"\"; expected true or false",
				helpers.InvalidSyntax,
			))
		} else {
			srcLibrary = b
		}
		if srcLibrary && !hasPkg {
			p.addErr(p.errorExpection(
				"src-library=true is only valid together with package=",
				helpers.InvalidSyntax,
			))
		}
	}

	expect, hasExpect := importDirectiveParam(dir, "expect")
	if hasExpect && !slices.Contains(validExpectKinds, expect) {
		p.addErr(p.errorExpection(
			"invalid expect=\""+expect+"\"; expected one of ffi, system, advanced, dynamicvmrt or application",
			helpers.InvalidSyntax,
		))
	}

	realm, hasRealm := importDirectiveParam(dir, "realm")
	if !hasRealm {
		realm = "app"
	}
	parentRealm, hasParentRealm := importDirectiveParam(dir, "parent-realm")

	if (hasRealm || hasParentRealm) && p.LibType != "dynamicvmrt" {
		p.addErr(p.errorExpection(
			"invalid realm/parent-realm attribute use only libraries with dynamicvmrt should have these attributes",
			helpers.InvalidSyntax,
		))

	}

	if hasParentRealm && (!hasRealm || realm == "app") {
		p.addErr(p.errorExpection(
			"parent-realm is only meaningful when realm is explicitly provided and is not \"app\"",
			helpers.InvalidSyntax,
		))
	}
	if !hasParentRealm {
		parentRealm = "app"
	}

	target := pkg
	if hasLib {
		target = lib
	}
	if hasAs {
		key := realm + "\x00" + as
		if prev, seen := realmAlias[key]; seen && prev != target {
			p.addErr(p.errorExpection(
				"alias \""+as+"\" is already imported in realm \""+realm+"\" from a different package/library",
				helpers.AlreadyDeclared,
			))
		} else {
			realmAlias[key] = target
		}
	}

	name := target
	if hasAs {
		name = as
	}

	parent := ""
	if hasAs {
		parent = as + "." + target
	}

	return ast.ImportStmt{
		Name:        as,
		From:        lib,
		Package:     pkg,
		Parent:      parent,
		Realm:       realm,
		ParentRealm: parentRealm,
		SrcLibrary:  srcLibrary,
		Expect:      expect,
		Symb: &symboltable.DirectivePragmaDetails{
			SymbolDetails: symboltable.SymbolDetails{
				Name_:       name,
				SymbolType_: string(symboltable.S_DirectivePragmaDetails),
			},
		},
	}
}

// importDirectiveParam reads a single-valued directive parameter collected
// by parse_ddaps. Parameter values arrive as []any (one entry per
// comma-separated value); import fields are always scalar so only the first
// entry is used. A value normalized to "_" means the source text was empty.
func importDirectiveParam(dir ast.DirectiveStmt, key string) (string, bool) {
	raw, ok := dir.Parameters[key]
	if !ok {
		return "", false
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return "", false
	}
	val := strings.TrimSpace(fmt.Sprint(arr[0]))
	if val == "" || val == "_" {
		return "", false
	}
	return val, true
}

// validate_alias_directives validates every @co.ddap.alias directive
// collected in ddaps[scanlex.DIRECTIVE]: the "as" value must be a valid
// FoLang identifier, and it must not collide with another alias already
// declared in the same file.
//
// Feature examples:
//
//	@co.ddap.alias(co.out, as="out")
//	@co.ddap.alias(co.core.list, as="list")
func validate_alias_directives(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []helpers.ErrorInterface {
	errors := []helpers.ErrorInterface{}
	seen := map[string]bool{}

	for _, stmt := range ddaps[scanlex.DIRECTIVE] {
		dir, ok := stmt.(ast.DirectiveStmt)
		if !ok || dir.Name != "@co.ddap.alias" {
			continue
		}

		as, hasAs := importDirectiveParam(dir, "as")
		if !hasAs {
			errors = append(errors, p.errorExpection(
				"@co.ddap.alias requires as=\"...\"",
				helpers.InvalidSyntax,
			))
			continue
		}
		if !validAliasIdentifier.MatchString(as) {
			errors = append(errors, p.errorExpection(
				"invalid alias as=\""+as+"\"; alias must be a valid FoLang identifier without dots",
				helpers.InvalidSyntax,
			))
			continue
		}
		if seen[as] {
			errors = append(errors, p.errorExpection(
				"duplicate alias \""+as+"\" declared more than once in this file",
				helpers.AlreadyDeclared,
			))
			continue
		}
		seen[as] = true
	}

	return errors
}
