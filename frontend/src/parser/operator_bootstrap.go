package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/project"
)

// projectOperatorBootstrap is the project's custom-operator catalog, read before any
// ordinary source. Area is kept even when the folder or the fixed file does not exist,
// because it is excluded from ordinary project discovery either way.
type projectOperatorBootstrap struct {
	Area         string
	File         string
	Declarations []operatorDeclaration
	Findings     []error
}

// loadProjectOperatorBootstrap parses the project's one operator bootstrap surface.
//
// The location is FIXED by the project layout rather than configured:
//
//	<project-root>/srclib/operators/library.fol
//
// `srclib/operators/` is the one standardized operator slot, and its presence is what
// says the project introduces project-local custom operator symbols. An absent slot or
// an absent file means an empty catalog rather than an error
// (docs/language-ref.md, "Project-Local Custom Operator Source").
//
// The fixed file is handed to parseOperatorSource rather than to the ordinary parser
// because it is read FIRST, by the dedicated operator-source lexer and parser, before
// any ordinary source has been tokenized: its symbols have to exist before the ordinary
// lexer can fuse them into single tokens. Combining these registrations with the
// language-owned ones into the immutable symbol and precedence tables belongs to
// precedence.go.
func loadProjectOperatorBootstrap(root string) projectOperatorBootstrap {
	rootAbs, rootErr := filepath.Abs(root)
	if rootErr != nil {
		return bootstrapFinding(root, "resolving project root: %v", rootErr)
	}

	result := projectOperatorBootstrap{
		Area: filepath.Join(rootAbs, project.SourceLibraryDomain, project.OperatorsLibrarySlot),
	}
	result.File = filepath.Join(result.Area, project.LibrarySurfaceFilename)

	source, readErr := os.ReadFile(result.File)
	if os.IsNotExist(readErr) {
		return result
	}
	if readErr != nil {
		return bootstrapFindingWithArea(result, result.File, "reading operator source: %v", readErr)
	}

	result.Declarations, result.Findings = parseOperatorSource(string(source), project.LibrarySurfaceFilename)
	return result
}

// pathWithin reports whether path is the operator area itself or a child of it.
// Rel-based comparison avoids prefix bugs such as operators versus operators-old and
// works with platform path separators.
func pathWithin(path, area string) bool {
	if area == "" {
		return false
	}
	pathAbs, pathErr := filepath.Abs(path)
	areaAbs, areaErr := filepath.Abs(area)
	if pathErr != nil || areaErr != nil {
		return false
	}
	rel, err := filepath.Rel(areaAbs, pathAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func configFinding(filename, source string, line int, message string) error {
	if line < 1 {
		line = 1
	}
	lineText := ""
	lines := strings.Split(normalizeLineEndings(source), "\n")
	if line <= len(lines) {
		lineText = lines[line-1]
	}
	start := helpers.Position{Ln: line, Col: 1, Fn: filename, Ftxt: lineText}
	end := start
	end.Col += len(lineText)
	return helpers.NewInvalidSyntaxError(start, end, message)
}

func bootstrapFinding(filename, format string, args ...any) projectOperatorBootstrap {
	return projectOperatorBootstrap{Findings: []error{configFinding(filename, "", 1, fmt.Sprintf(format, args...))}}
}

func bootstrapFindingWithArea(result projectOperatorBootstrap, filename, format string, args ...any) projectOperatorBootstrap {
	result.Findings = append(result.Findings, configFinding(filename, "", 1, fmt.Sprintf(format, args...)))
	return result
}
