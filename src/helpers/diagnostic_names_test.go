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

	if len(documented) != len(implemented) {
		t.Fatalf("diagnostic registry count: reference=%d implementation=%d\nreference=%v\nimplementation=%v", len(documented), len(implemented), documented, implemented)
	}
	for i := range documented {
		if documented[i] != implemented[i] {
			t.Fatalf("diagnostic registry differs at %d: reference=%q implementation=%q", i, documented[i], implemented[i])
		}
	}
}
