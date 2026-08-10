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

// A source library lives in its own namespace. It is named by its fixed srclib/ slot,
// while packages are named by path, so a package that happens to be spelled like a slot
// is a different node entirely and importing it closes no cycle.
func TestSourceLibrarySurfaceDistinguishesItsSlotFromASamelyNamedPackage(t *testing.T) {
	surface := File{
		Name:             "library.fol",
		IsLibrarySurface: true,
		LibraryPath:      "ffi",
	}

	ordinaryPackage := surface
	ordinaryPackage.Imports = []Import{{Package: "ffi"}}
	if findings := ValidateProject([]File{ordinaryPackage}); len(findings) != 0 {
		t.Fatalf("a package merely named like a source-library slot produced findings: %v", findings)
	}

	selfSurface := surface
	selfSurface.Imports = []Import{{Library: "ffi", SrcLibrary: true}}
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
