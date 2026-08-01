package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/importcheck"
)

func TestScannedLibraryBodyImportsParticipateInSourceLibraryCycles(t *testing.T) {
	alpha := ScanImportSurface(`Alpha co.lang.library = {
    @co.ddap.import(package="libs.beta", src-library=co.const.true, as="beta")
}`, "alpha.fol", "alpha", "libs", false)
	beta := ScanImportSurface(`Beta co.lang.library = {
    @co.ddap.import(package="libs.alpha", src-library=co.const.true, as="alpha")
}`, "beta.fol", "beta", "libs", false)

	findings := importcheck.ValidateProject([]importcheck.File{alpha, beta})
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want one source-library cycle: %v", len(findings), findings)
	}
	message := findings[0].Error()
	if !strings.Contains(message, "Package Import Cycle") ||
		!strings.Contains(message, "libs.alpha -> libs.beta -> libs.alpha") {
		t.Fatalf("unexpected finding: %s", message)
	}
}

func TestScanLibraryBodyImportsSkipsNestedDirectiveLikeTokens(t *testing.T) {
	surface := ScanImportSurface(`Api co.lang.library = {
    @co.ddap.import(package="api.internal", as="internal")
    run ()->() = {
        @co.ddap.import(package="must.not.scan", as="nested")
    }
}`, "api.fol", "api", "libs", false)

	if len(surface.Imports) != 1 {
		t.Fatalf("imports = %d, want only the surface-level body import", len(surface.Imports))
	}
	if got := surface.Imports[0].Package; got != "api.internal" {
		t.Fatalf("scanned package = %q, want api.internal", got)
	}
}
