package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

const (
	projectConfigFile      = "fol-conf.yaml"
	operatorSourceFileName = "operators.fol"
)

// projectOperatorBootstrap is the result of DECISION-OPBOOT-001. Area is kept
// even when the configured folder or fixed file does not exist, because the
// configured namespace is excluded from ordinary project discovery either way.
type projectOperatorBootstrap struct {
	Area         string
	File         string
	Declarations []operatorDeclaration
	Findings     []error
}

// loadProjectOperatorBootstrap reads only the one configuration key needed by
// the parser bootstrap, resolves it relative to root, and parses only the fixed
// <area>/operators.fol source. Missing configuration, folder, or file means an
// empty project-local catalog rather than an error.
//
// This is step 1-4 of DECISION-OPBOOT-003's bootstrap order. DECISION-OPBOOT-002
// is why the fixed file is handed to parseOperatorSource rather than the ordinary
// parser: it is read FIRST, by the dedicated operator-source lexer and parser,
// before any ordinary source has been tokenized. Steps 5-6 — combining these
// registrations with the language-owned ones into the immutable symbol and
// precedence tables — belong to precedence.go.
func loadProjectOperatorBootstrap(root string) projectOperatorBootstrap {
	configPath := filepath.Join(root, projectConfigFile)
	content, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return projectOperatorBootstrap{}
	}
	if err != nil {
		return bootstrapFinding(configPath, "reading project configuration: %v", err)
	}

	folder, line, present, findings := operatorLibraryFolderValue(string(content), configPath)
	result := projectOperatorBootstrap{Findings: findings}
	if !present || len(findings) > 0 {
		return result
	}
	if filepath.IsAbs(folder) {
		result.Findings = append(result.Findings, configFinding(configPath, string(content), line,
			"operator_library_folder must be a path relative to the project root"))
		return result
	}

	rootAbs, rootErr := filepath.Abs(root)
	if rootErr != nil {
		return bootstrapFinding(configPath, "resolving project root: %v", rootErr)
	}
	areaAbs, areaErr := filepath.Abs(filepath.Join(rootAbs, filepath.Clean(folder)))
	if areaErr != nil {
		return bootstrapFinding(configPath, "resolving operator_library_folder: %v", areaErr)
	}
	rel, relErr := filepath.Rel(rootAbs, areaAbs)
	if relErr != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		result.Findings = append(result.Findings, configFinding(configPath, string(content), line,
			"operator_library_folder must name a folder strictly below the project root"))
		return result
	}

	result.Area = filepath.Clean(areaAbs)
	result.File = filepath.Join(result.Area, operatorSourceFileName)
	info, statErr := os.Stat(result.Area)
	if os.IsNotExist(statErr) {
		return result
	}
	if statErr != nil {
		return bootstrapFindingWithArea(result, configPath, "checking operator source folder: %v", statErr)
	}
	if !info.IsDir() {
		result.Findings = append(result.Findings, configFinding(configPath, string(content), line,
			"operator_library_folder resolves to a file; it must name a folder"))
		return result
	}

	source, readErr := os.ReadFile(result.File)
	if os.IsNotExist(readErr) {
		return result
	}
	if readErr != nil {
		return bootstrapFindingWithArea(result, result.File, "reading operator source: %v", readErr)
	}
	result.Declarations, result.Findings = parseOperatorSource(string(source), operatorSourceFileName)
	return result
}

// operatorLibraryFolderValue is a deliberately small YAML scalar reader. It
// accepts the key at any indentation (matching existing fol-conf layouts),
// quoted or unquoted scalar paths, and comments outside quotes. It rejects
// duplicates and empty/non-scalar values instead of silently choosing one.
func operatorLibraryFolderValue(source, filename string) (string, int, bool, []error) {
	var value string
	var foundLine int
	var findings []error
	for index, raw := range strings.Split(normalizeLineEndings(source), "\n") {
		lineNumber := index + 1
		line := strings.TrimSpace(stripYAMLComment(raw))
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 || strings.TrimSpace(line[:colon]) != "operator_library_folder" {
			continue
		}
		if foundLine != 0 {
			findings = append(findings, configFinding(filename, source, lineNumber,
				fmt.Sprintf("operator_library_folder occurs more than once (first occurrence at line %d)", foundLine)))
			continue
		}
		foundLine = lineNumber
		rawValue := strings.TrimSpace(line[colon+1:])
		if rawValue == "" {
			findings = append(findings, configFinding(filename, source, lineNumber,
				"operator_library_folder requires a non-empty scalar path"))
			continue
		}
		decoded, ok := decodeYAMLPathScalar(rawValue)
		if !ok || strings.TrimSpace(decoded) == "" {
			findings = append(findings, configFinding(filename, source, lineNumber,
				"operator_library_folder must be one quoted or unquoted scalar path"))
			continue
		}
		value = decoded
	}
	return value, foundLine, foundLine != 0, findings
}

func stripYAMLComment(line string) string {
	var quote byte
	escaped := false
	for index := 0; index < len(line); index++ {
		ch := line[index]
		if quote == '"' {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if quote == '\'' {
			if ch == '\'' {
				if index+1 < len(line) && line[index+1] == '\'' {
					index++
					continue
				}
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case '#':
			if index == 0 || line[index-1] == ' ' || line[index-1] == '\t' {
				return line[:index]
			}
		}
	}
	return line
}

func decodeYAMLPathScalar(value string) (string, bool) {
	if strings.HasPrefix(value, `"`) {
		decoded, err := strconv.Unquote(value)
		return decoded, err == nil
	}
	if strings.HasPrefix(value, "'") {
		if len(value) < 2 || !strings.HasSuffix(value, "'") {
			return "", false
		}
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), true
	}
	if strings.ContainsAny(value, "[]{}\"'") {
		return "", false
	}
	return strings.TrimSpace(value), true
}

// pathWithin reports whether path is the configured area itself or a child of
// it. Rel-based comparison avoids prefix bugs such as operators versus
// operators-old and works with platform path separators.
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
