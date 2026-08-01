package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/project"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// declarationSurface is the declaration-local information required by the
// project-level companion check. It deliberately does not build an AST.
type declarationSurface struct {
	PackagePath string
	Name        string
	Kind        string
	HasOperator bool
	OperatorTok scanlex.Token
	File        string
}

// scanDeclarationSurface reads the single primary declaration header and notes
// whether its body contains an operator annotation. Package discovery already
// supplies the physical identity, so this inexpensive scan is enough to connect
// a unit to a same-package struct without parsing every body in the project.
func scanDeclarationSurface(source string, file project.File) declarationSurface {
	toks := normalizeTokens(scanlex.TokenizeQuiet(normalizeLineEndings(source), file.Base))
	surface := declarationSurface{PackagePath: file.PackagePath, File: file.Base}

	parenDepth, bracketDepth, curlyDepth := 0, 0, 0
	var lastName scanlex.Token
	for _, tok := range toks {
		if tok.Value == "@co.dap.operator" && !surface.HasOperator {
			surface.HasOperator = true
			surface.OperatorTok = tok
		}

		atTop := parenDepth == 0 && bracketDepth == 0 && curlyDepth == 0
		if atTop {
			switch tok.Kind {
			case scanlex.IDENTIFIER, scanlex.DISCARD_WILD_VAR:
				lastName = tok
			case scanlex.BUILT_IN_KIND:
				if surface.Kind == "" && lastName.Value != "" {
					surface.Kind = tok.Value
					surface.Name = declarationSurfaceName(lastName, file.Stem, tok.Value)
				}
			}
		}

		switch tok.Kind {
		case scanlex.OPEN_PAREN:
			parenDepth++
		case scanlex.CLOSE_PAREN:
			if parenDepth > 0 {
				parenDepth--
			}
		case scanlex.OPEN_BRACKET:
			bracketDepth++
		case scanlex.CLOSE_BRACKET:
			if bracketDepth > 0 {
				bracketDepth--
			}
		case scanlex.OPEN_CURLY:
			curlyDepth++
		case scanlex.CLOSE_CURLY:
			if curlyDepth > 0 {
				curlyDepth--
			}
		}
	}
	return surface
}

func declarationSurfaceName(tok scanlex.Token, fileStem, kind string) string {
	if tok.Kind != scanlex.DISCARD_WILD_VAR {
		return logicalName(tok.Value)
	}
	suffix := kind
	if dot := strings.LastIndexByte(suffix, '.'); dot >= 0 {
		suffix = suffix[dot+1:]
	}
	qualified := "." + suffix
	if strings.HasSuffix(strings.ToLower(fileStem), strings.ToLower(qualified)) {
		return fileStem[:len(fileStem)-len(qualified)]
	}
	return fileStem
}

// validateOperatorCompanions performs the cross-file half of operator ownership:
// a unit containing an operator must have a same-name struct in the same package.
func validateOperatorCompanions(surfaces []declarationSurface) []error {
	structs := map[string]bool{}
	for _, surface := range surfaces {
		if surface.Kind == "co.lang.struct" {
			structs[companionKey(surface.PackagePath, surface.Name)] = true
		}
	}

	var findings []error
	for _, surface := range surfaces {
		if surface.Kind != "co.lang.unit" || !surface.HasOperator {
			continue
		}
		if structs[companionKey(surface.PackagePath, surface.Name)] {
			continue
		}
		start, end := tokenSpan(surface.OperatorTok)
		findings = append(findings, helpers.NewExpectedTokenErrorName(
			start,
			end,
			"Invalid Operator Companion",
			fmt.Sprintf("unit %q in package %q declares an operator but has no same-name co.lang.struct in that package", surface.Name, surface.PackagePath),
		))
	}
	return findings
}

func companionKey(packagePath, name string) string { return packagePath + "\x00" + name }
