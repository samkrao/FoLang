package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
)

// grammarMap is the docs/grammar-map.json payload.
type grammarMap struct {
	// Productions maps an EBNF production name to the function that declares it
	// with an "Implements: <production>" doc comment.
	Productions map[string]string `json:"productions"`
	// Missing lists grammar productions no function claims.
	Missing []string `json:"missing"`
	// Extra lists parse functions that claim no production.
	Extra []string `json:"extra"`
	// Conflicts lists productions claimed by more than one function. A
	// production has one implementation; two claims mean one is stale.
	Conflicts map[string][]string `json:"conflicts"`
}

var (
	// productionPattern matches an EBNF production definition at line start.
	productionPattern = regexp.MustCompile(`(?m)^([a-z][a-z0-9-]*)\s*=`)
	// implementsPattern matches the doc-comment annotation that ties a function
	// to a production.
	implementsPattern = regexp.MustCompile(`(?m)^\s*Implements:\s*([a-z][a-z0-9-]*)\s*$`)
)

// buildGrammarMap correlates EBNF productions with the functions claiming them.
func buildGrammarMap(pkgDir, grammarPath string) (*grammarMap, error) {
	grammarSrc, err := os.ReadFile(grammarPath)
	if err != nil {
		return nil, err
	}

	productions := map[string]bool{}
	for _, m := range productionPattern.FindAllStringSubmatch(string(grammarSrc), -1) {
		productions[m[1]] = true
	}

	claims, functions, err := collectClaims(pkgDir)
	if err != nil {
		return nil, err
	}

	result := &grammarMap{
		Productions: map[string]string{},
		Missing:     []string{},
		Extra:       []string{},
		Conflicts:   map[string][]string{},
	}

	claimed := map[string]bool{}
	for production, fns := range claims {
		sort.Strings(fns)
		if len(fns) > 1 {
			result.Conflicts[production] = fns
		}
		result.Productions[production] = fns[0]
		for _, fn := range fns {
			claimed[fn] = true
		}
	}

	for production := range productions {
		if _, ok := result.Productions[production]; !ok {
			result.Missing = append(result.Missing, production)
		}
	}
	// A claim naming a production the grammar does not define is itself a
	// mismatch, so surface the function rather than silently accepting it.
	for production, fns := range claims {
		if !productions[production] {
			result.Missing = append(result.Missing, production)
			_ = fns
		}
	}
	for _, fn := range functions {
		if !claimed[fn] {
			result.Extra = append(result.Extra, fn)
		}
	}

	sort.Strings(result.Missing)
	sort.Strings(result.Extra)
	return result, nil
}

// collectClaims returns production -> claiming functions, and the full list of
// parse functions considered.
func collectClaims(pkgDir string) (map[string][]string, []string, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	claims := map[string][]string{}
	var functions []string

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !isParseFunc(fn) {
					continue
				}
				functions = append(functions, fn.Name.Name)
				if fn.Doc == nil {
					continue
				}
				for _, m := range implementsPattern.FindAllStringSubmatch(fn.Doc.Text(), -1) {
					claims[m[1]] = append(claims[m[1]], fn.Name.Name)
				}
			}
		}
	}

	sort.Strings(functions)
	return claims, functions, nil
}

// isParseFunc reports whether fn is a parse function: a method on *parser whose
// name begins with "parse". This is the same set the partrace instrumentation
// covers, so grammar-map.json and trace.json describe the same functions.
func isParseFunc(fn *ast.FuncDecl) bool {
	if !strings.HasPrefix(fn.Name.Name, "parse") || fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "parser"
}
