package helpers

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

func TestDiagnosticRegistryMatchesLanguageReference(t *testing.T) {
	raw, err := os.ReadFile("../../docs/language-ref.md")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^\| ` + "`" + `([A-Za-z][A-Za-z0-9]*)` + "`" + ` \| Error \|`)
	matches := re.FindAllSubmatch(raw, -1)
	if len(matches) == 0 {
		t.Fatal("diagnostic registry table not found in docs/language-ref.md; expected rows shaped as | `Name` | Error |")
	}
	documented := make([]string, 0, len(matches))
	for _, match := range matches {
		documented = append(documented, string(match[1]))
	}
	sort.Strings(documented)

	implemented := make([]string, 0, len(registeredDiagnosticNames))
	for name := range registeredDiagnosticNames {
		implemented = append(implemented, string(name))
	}
	sort.Strings(implemented)

	missing, extra := setDifference(documented, implemented), setDifference(implemented, documented)
	if len(missing) != 0 || len(extra) != 0 {
		t.Fatalf("diagnostic registry differs: missing from implementation=%v; extra in implementation=%v", missing, extra)
	}
}

func TestGenericErrorsRemainVisiblyUnclassified(t *testing.T) {
	err := NewError(Position{}, Position{}, "Unknown Error", "legacy diagnostic")
	if got := err.DiagnosticName(); got != string(DiagnosticUnclassified) {
		t.Fatalf("NewError diagnostic name = %q, want %q", got, DiagnosticUnclassified)
	}
	if !IsRegisteredDiagnosticName(err.DiagnosticName()) {
		t.Fatalf("fallback diagnostic name %q is not registered", err.DiagnosticName())
	}
}

func setDifference(left, right []string) []string {
	present := make(map[string]struct{}, len(right))
	for _, value := range right {
		present[value] = struct{}{}
	}
	var difference []string
	for _, value := range left {
		if _, ok := present[value]; !ok {
			difference = append(difference, value)
		}
	}
	return difference
}
