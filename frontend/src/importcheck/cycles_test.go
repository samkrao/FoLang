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

// Only `package=` contributes a package-graph edge. `library=` names the projected
// surface of a prebuilt lib/<name>.folenc artifact and `component=` a same-owner
// projected component, so neither closes a package cycle even when it is spelled
// like the importing package itself.
func TestNonPackageImportsContributeNoPackageCycleEdge(t *testing.T) {
	for _, imp := range []Import{
		{Library: "shared"},
		{Component: "shared"},
	} {
		findings := ValidateProject([]File{
			{Name: "Alpha.fol", PackagePath: "shared", Imports: []Import{imp}},
		})
		if len(findings) != 0 {
			t.Errorf("%#v produced package-cycle findings: %v", imp, findings)
		}
	}
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
