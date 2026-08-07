package parser_test

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestGrammarDecisionsHaveImplementationTrace is the decision-register companion
// to TestGrammarProductionsHaveImplementationTrace.
//
// The production guard proves every EBNF rule is reachable from parser source.
// It cannot prove that the RULES ABOUT those productions were applied: a
// decision such as DECISION-GEN-001 removes a clause from a production whose
// name never changes, so a parser that ignored it would still pass the
// production guard. This test closes that gap.
//
// Every ACTIVE decision in docs/grammar/grammar-decisions.md must be either
// named in frontend/src, or classified below as belonging to a phase the
// frontend does not implement. A withdrawn decision is skipped: it exists in the
// register as history and has nothing to implement.
func TestGrammarDecisionsHaveImplementationTrace(t *testing.T) {
	register := readAuditFile(t, filepath.Join("..", "..", "..", "docs", "grammar", "grammar-decisions.md"))
	source := readGoDirectory(t, filepath.Join("..", "..", "src", "parser")) +
		readGoDirectory(t, filepath.Join("..", "..", "src", "scanlex")) +
		readGoDirectory(t, filepath.Join("..", "..", "src", "importcheck"))

	// A register row is `| `DECISION-X-N` | <status> | <text> |`.
	rowPattern := regexp.MustCompile("(?m)^\\|\\s*`(DECISION-[A-Z]+-[0-9]+)`\\s*\\|\\s*([^|]+?)\\s*\\|")

	untraced := make([]string, 0)
	staleClassification := make([]string, 0)
	seen := map[string]bool{}

	for _, row := range rowPattern.FindAllStringSubmatch(register, -1) {
		id, status := row[1], row[2]
		seen[id] = true

		if strings.HasPrefix(status, "Withdrawn") {
			// A withdrawn decision must not be classified as deferred work.
			if _, classified := nonFrontendDecision[id]; classified {
				staleClassification = append(staleClassification, id+" is withdrawn but is still classified as non-frontend")
			}
			continue
		}
		if strings.Contains(source, id) {
			if _, classified := nonFrontendDecision[id]; classified {
				staleClassification = append(staleClassification, id+" is implemented in the frontend but is still classified as non-frontend")
			}
			continue
		}
		if nonFrontendDecision[id] != "" {
			continue
		}
		untraced = append(untraced, id)
	}

	if len(seen) == 0 {
		t.Fatal("no decisions parsed from the register; the row format changed")
	}

	// A classification for a decision the register no longer lists is dead weight
	// and would silently excuse a future decision that reused the identifier.
	for id := range nonFrontendDecision {
		if !seen[id] {
			staleClassification = append(staleClassification, id+" is classified but is not in the register")
		}
	}

	sort.Strings(untraced)
	sort.Strings(staleClassification)
	if len(untraced) != 0 {
		t.Errorf("active decisions with no implementation trace and no classification:\n%s", strings.Join(untraced, "\n"))
	}
	if len(staleClassification) != 0 {
		t.Errorf("stale classifications:\n%s", strings.Join(staleClassification, "\n"))
	}
}

// nonFrontendDecision records the active decisions the parser and scanner
// deliberately do not implement, with the phase that owns each.
//
// These are all decisions the grammar itself states are decided elsewhere. The
// classification is explicit so that a NEW decision cannot be silently treated
// as covered merely because nobody looked at it.
var nonFrontendDecision = map[string]string{
	"DECISION-SEM-001": "name resolution: instance selection is by name and instance placement is checked " +
		"during resolution. The grammar notes that a misplaced instance parses successfully.",
	"DECISION-SYN-009": "name resolution: annotated-function-primary is only a syntactic envelope, and " +
		"whether an annotation really defines a primary declaration kind is decided after annotation resolution.",
	"DECISION-SCOPE-001": "name resolution: the lexical declaration environment of a local function.",
	"DECISION-SCOPE-002": "name resolution: @co.dap.inner call-site lexical-context resolution.",
	"DECISION-TYP-006": "type checking: dependent types are checked against written signatures, never inferred.",
}
