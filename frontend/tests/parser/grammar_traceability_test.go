package parser_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestGrammarProductionsHaveImplementationTrace verifies that every normative
// EBNF production is traceable either to parser source or to an explicitly
// classified scanner/Pratt/zero-width implementation. This is a traceability
// guard, not a substitute for behavioral conformance fixtures.
func TestGrammarProductionsHaveImplementationTrace(t *testing.T) {
	grammar := readAuditFile(t, filepath.Join("..", "..", "..", "docs", "grammar", "folang.ebnf"))
	parserSource := readGoDirectory(t, filepath.Join("..", "..", "src", "parser"))

	productionPattern := regexp.MustCompile(`(?m)^([a-z][a-z0-9-]*)\s*=`)
	missing := make([]string, 0)
	for _, match := range productionPattern.FindAllStringSubmatch(grammar, -1) {
		name := match[1]
		if strings.Contains(parserSource, name) || nonParserProduction[name] != "" {
			continue
		}
		missing = append(missing, name)
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("EBNF productions without an implementation trace:\n%s", strings.Join(missing, "\n"))
	}
}

// These productions are implemented below recursive-descent production level.
// Keeping the classification explicit prevents a newly added grammar rule from
// being silently treated as covered merely because it resembles a token.
var nonParserProduction = map[string]string{
	"additive-operator":                       "Pratt operator table",
	"alpha-basic-c-character":                 "scanner literal token",
	"alpha-basic-s-character":                 "scanner literal token",
	"ascii-alphanumeric":                      "scanner identifier token",
	"ascii-letter":                            "scanner identifier token",
	"binary-digit":                            "scanner numeric token",
	"binary-digit-sequence":                   "scanner numeric token",
	"binary-integer-literal":                  "scanner numeric token",
	"bitwise-and-expression":                  "Pratt precedence table",
	"bitwise-xor-expression":                  "Pratt precedence table",
	"block-comment":                           "scanner trivia",
	"block-comment-character":                 "scanner trivia",
	"compound-assignment-operator":            "Pratt operator table",
	"contextual-keyword":                      "scanner token classification",
	"decimal-digit":                           "scanner numeric token",
	"decimal-digit-sequence":                  "scanner numeric token",
	"decimal-floating-literal":                "scanner numeric token",
	"decimal-integer-literal":                 "scanner numeric token",
	"double-quote":                            "scanner literal delimiter",
	"equality-expression":                     "Pratt precedence table",
	"equality-operator":                       "Pratt operator table",
	"extended-operator-expression":            "registered Pratt operator table",
	"fractional-constant":                     "scanner numeric token",
	"hard-reserved-word":                      "scanner token classification",
	"hexadecimal-digit":                       "scanner numeric token",
	"hexadecimal-digit-sequence":              "scanner numeric token",
	"hexadecimal-floating-literal":            "scanner numeric token",
	"hexadecimal-fractional-constant":         "scanner numeric token",
	"hexadecimal-integer-literal":             "scanner numeric token",
	"hexadecimal-prefix":                      "scanner numeric token",
	"horizontal-white-space":                  "scanner trivia",
	"identifier-head":                         "scanner identifier token",
	"identifier-segment":                      "scanner identifier token",
	"identifier-trailing-guard":               "scanner identifier token",
	"informative-pipeline-chain":              "informative semantic shape",
	"keyword-token":                           "scanner token classification",
	"line-comment":                            "scanner trivia",
	"long-long-suffix":                        "scanner numeric token",
	"long-suffix":                             "scanner numeric token",
	"multiplicative-expression":               "Pratt precedence table",
	"multiplicative-operator":                 "Pratt operator table",
	"non-anonymous-function-expression-guard": "zero-width parser guard",
	"non-block-expression-guard":              "zero-width parser guard",
	"nonzero-digit":                           "scanner numeric token",
	"octal-digit":                             "scanner numeric token",
	"octal-digit-sequence":                    "scanner numeric token",
	"octal-integer-literal":                   "scanner numeric token",
	"relational-expression":                   "Pratt precedence table",
	"relational-operator":                     "Pratt operator table",
	"reserved-future-operator":                "scanner/reserved operator table",
	"reserved-operator":                       "scanner/reserved operator table",
	"runtime-assignment-operator":             "Pratt operator table",
	"single-quote":                            "scanner literal delimiter",
	"size-suffix":                             "scanner numeric token",
	"token-separator":                         "scanner trivia boundary",
	"unsigned-suffix":                         "scanner numeric token",
	"white-space":                             "scanner trivia",
}

func readAuditFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit input %s: %v", path, err)
	}
	return string(data)
}

func readGoDirectory(t *testing.T, dir string) string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("list parser source: %v", err)
	}
	var source strings.Builder
	for _, path := range paths {
		source.WriteString(readAuditFile(t, path))
		source.WriteByte('\n')
	}
	return source.String()
}
