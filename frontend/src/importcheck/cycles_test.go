package importcheck

import (
	"strings"
	"testing"
)

func TestValidateProjectRejectsDirectPackageImportCycle(t *testing.T) {
	findings := ValidateProject([]File{
		{Name: "Alpha.fol", PackagePath: "alpha", Imports: []Import{{Package: "beta"}}},
		{Name: "Beta.fol", PackagePath: "beta", Imports: []Import{{Package: "alpha"}}},
	})

	requireSingleFinding(t, findings, "Package Import Cycle", "alpha -> beta -> alpha")
}

func TestValidateProjectRejectsIndirectPackageImportCycle(t *testing.T) {
	findings := ValidateProject([]File{
		{Name: "Alpha.fol", PackagePath: "alpha", Imports: []Import{{Package: "beta"}}},
		{Name: "Beta.fol", PackagePath: "beta", Imports: []Import{{Package: "gamma"}}},
		{Name: "Gamma.fol", PackagePath: "gamma", Imports: []Import{{Package: "alpha"}}},
	})

	requireSingleFinding(t, findings, "Package Import Cycle", "alpha -> beta -> gamma -> alpha")
}

func TestValidateProjectRejectsSelfImportOnce(t *testing.T) {
	findings := ValidateProject([]File{
		{Name: "Employee.fol", PackagePath: "hr.employee", Imports: []Import{{Package: "hr.employee"}}},
	})

	requireSingleFinding(t, findings, "Package Import Cycle", "imports itself")
}

func TestSourceLibrarySurfaceDistinguishesItsSurfaceFromSamePathPackage(t *testing.T) {
	surface := File{
		Name:             "ffi.fol",
		PackagePath:      "com.abc",
		IsLibrarySurface: true,
		LibraryPath:      "com.abc.ffi",
	}

	ordinaryInternal := surface
	ordinaryInternal.Imports = []Import{{Package: "com.abc.ffi"}}
	if findings := ValidateProject([]File{ordinaryInternal}); len(findings) != 0 {
		t.Fatalf("ordinary same-path package import produced findings: %v", findings)
	}

	selfSurface := surface
	selfSurface.Imports = []Import{{Package: "com.abc.ffi", SrcLibrary: true}}
	requireSingleFinding(t, ValidateProject([]File{selfSurface}), "Package Import Cycle", "imports itself")
}

func TestValidateProjectAcceptsAcyclicPackageGraph(t *testing.T) {
	findings := ValidateProject([]File{
		{Name: "Alpha.fol", PackagePath: "alpha", Imports: []Import{{Package: "shared"}}},
		{Name: "Beta.fol", PackagePath: "beta", Imports: []Import{{Package: "shared"}}},
		{Name: "Shared.fol", PackagePath: "shared"},
	})

	if len(findings) != 0 {
		t.Fatalf("acyclic project produced %d findings: %v", len(findings), findings)
	}
}

func requireSingleFinding(t *testing.T, findings []error, fragments ...string) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1: %v", len(findings), findings)
	}
	message := findings[0].Error()
	for _, fragment := range fragments {
		if !strings.Contains(message, fragment) {
			t.Errorf("finding %q does not contain %q", message, fragment)
		}
	}
}
