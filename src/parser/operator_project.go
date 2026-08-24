package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/project"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// declarationSurface is the declaration-local information required by the
// project-level companion check. It deliberately does not build an AST.
type declarationSurface struct {
	PackagePath string
	Name        string
	Kind        string
	HasOperator bool
	// HasCompanionOperator excludes operator functions that are explicitly
	// owned by an existing type through @co.dap.extension. Only companion-owned
	// operators require a same-name struct in the project pass.
	HasCompanionOperator bool
	OperatorTok          scanlex.Token
	File                 string
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
	for index, tok := range toks {
		if tok.Value == "@co.dap.operator" {
			surface.HasOperator = true
			if !annotationGroupContains(toks, index, "@co.dap.extension") && !surface.HasCompanionOperator {
				surface.HasCompanionOperator = true
				surface.OperatorTok = tok
			}
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

// annotationGroupContains reports whether the contiguous annotation run that
// contains toks[index] also contains target. It understands optional balanced
// annotation argument lists in both directions so annotation order does not
// change operator ownership classification.
func annotationGroupContains(toks []scanlex.Token, index int, target string) bool {
	start := index
	for {
		previous, ok := previousAnnotationStart(toks, start)
		if !ok {
			break
		}
		start = previous
	}

	for start < len(toks) && isSurfaceAnnotation(toks[start]) {
		if toks[start].Value == target {
			return true
		}
		start = afterSurfaceAnnotation(toks, start)
	}
	return false
}

func previousAnnotationStart(toks []scanlex.Token, before int) (int, bool) {
	if before <= 0 {
		return 0, false
	}
	previous := before - 1
	if isSurfaceAnnotation(toks[previous]) {
		return previous, true
	}
	if toks[previous].Kind != scanlex.CLOSE_PAREN {
		return 0, false
	}

	depth := 1
	for index := previous - 1; index >= 0; index-- {
		switch toks[index].Kind {
		case scanlex.CLOSE_PAREN:
			depth++
		case scanlex.OPEN_PAREN:
			depth--
			if depth == 0 {
				if index > 0 && isSurfaceAnnotation(toks[index-1]) {
					return index - 1, true
				}
				return 0, false
			}
		}
	}
	return 0, false
}

func afterSurfaceAnnotation(toks []scanlex.Token, start int) int {
	next := start + 1
	if next >= len(toks) || toks[next].Kind != scanlex.OPEN_PAREN {
		return next
	}
	depth := 0
	for ; next < len(toks); next++ {
		switch toks[next].Kind {
		case scanlex.OPEN_PAREN:
			depth++
		case scanlex.CLOSE_PAREN:
			depth--
			if depth == 0 {
				return next + 1
			}
		}
	}
	return next
}

func isSurfaceAnnotation(tok scanlex.Token) bool {
	return tok.Kind == scanlex.BUILT_IN_DIRECTIVES ||
		tok.Kind == scanlex.CUSTOM_DIRECTIVES || tok.Kind == scanlex.ATDAP
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
// a unit containing a companion-owned operator must be the one unit paired with
// exactly one same-name struct in the same package. Counting,
// rather than recording mere presence, prevents duplicate primary declarations
// from making an otherwise ambiguous owner appear valid. Built-in extension
// operators carry their owner explicitly and do not participate in this check.
func validateOperatorCompanions(surfaces []declarationSurface) []error {
	type companionCounts struct {
		structs int
		units   int
	}
	counts := map[string]companionCounts{}
	for _, surface := range surfaces {
		key := companionKey(surface.PackagePath, surface.Name)
		count := counts[key]
		switch {
		case surface.Kind == "co.lang.struct":
			count.structs++
		case surface.Kind == "co.lang.unit":
			count.units++
		}
		counts[key] = count
	}

	var findings []error
	reported := map[string]bool{}
	for _, surface := range surfaces {
		if surface.Kind != "co.lang.unit" || !surface.HasCompanionOperator {
			continue
		}
		key := companionKey(surface.PackagePath, surface.Name)
		count := counts[key]
		if count.structs == 1 && count.units == 1 {
			continue
		}
		if reported[key] {
			continue
		}
		reported[key] = true
		start, end := tokenSpan(surface.OperatorTok)
		findings = append(findings, helpers.NewExpectedTokenErrorName(
			start,
			end,
			"Invalid Operator Companion",
			fmt.Sprintf(
				"unit %q in package %q declares an operator but companion ownership requires exactly one same-name co.lang.struct and one co.lang.unit; found %d struct declarations and %d unit declarations",
				surface.Name,
				surface.PackagePath,
				count.structs,
				count.units,
			),
		))
	}
	return findings
}

func companionKey(packagePath, name string) string { return packagePath + "\x00" + name }
