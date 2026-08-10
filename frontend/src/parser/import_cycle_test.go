package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/importcheck"
)

// A source library is identified by its fixed srclib/ SLOT, and imported by naming that
// slot with `library=` plus `src-library=true`. Two slots that import each other still
// close a cycle, and the edges may be written inside the surface's body rather than in
// its preamble — which is the scan this test guards.
func TestScannedLibraryBodyImportsParticipateInSourceLibraryCycles(t *testing.T) {
	ffi := ScanImportSurface(`_ co.lang.library = {
    @co.ddap.import(library="system", src-library=true, as="sys")
}`, "library.fol", "library", "", true, "ffi")
	system := ScanImportSurface(`_ co.lang.library = {
    @co.ddap.import(library="ffi", src-library=true, as="ffilib")
}`, "library.fol", "library", "", true, "system")

	findings := importcheck.ValidateProject([]importcheck.File{ffi, system})
	if len(findings) == 0 {
		t.Fatal("two source libraries importing each other produced no finding")
	}
	joined := ""
	for _, finding := range findings {
		joined += finding.Error() + "\n"
	}
	if !strings.Contains(joined, "Package Import Cycle") ||
		!strings.Contains(joined, "ffi -> system -> ffi") {
		t.Fatalf("unexpected findings:\n%s", joined)
	}
}

func TestScanLibraryBodyImportsSkipsNestedDirectiveLikeTokens(t *testing.T) {
	surface := ScanImportSurface(`_ co.lang.library = {
    @co.ddap.import(package="api.internal", as="internal")
    run ()->() = {
        @co.ddap.import(package="must.not.scan", as="nested")
    }
}`, "library.fol", "library", "", true, "ffi")

	if len(surface.Imports) != 1 {
		t.Fatalf("imports = %d, want only the surface-level body import", len(surface.Imports))
	}
	if got := surface.Imports[0].Package; got != "api.internal" {
		t.Fatalf("scanned package = %q, want api.internal", got)
	}
}
