package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/scanlex"
)

// TestCanonicalFileKeyFoldsCaseAndDropsUnderscores covers DECISION-FILE-004. The
// key is what makes two spellings the SAME declaration, so every spelling that
// derives one name under DECISION-FILE-002 must also share one key.
func TestCanonicalFileKeyFoldsCaseAndDropsUnderscores(t *testing.T) {
	group := []string{"employee_service", "EmployeeService", "employeeService", "EMPLOYEE_SERVICE"}
	want := canonicalFileKey(group[0])
	for _, component := range group[1:] {
		if got := canonicalFileKey(component); got != want {
			t.Errorf("canonicalFileKey(%q) = %q, want %q", component, got, want)
		}
	}

	// Distinct declarations must not collide.
	if canonicalFileKey("Employee") == canonicalFileKey("Employees") {
		t.Error("Employee and Employees share a canonical key")
	}
}

// TestFilenameDerivationAgreesWithCanonicalKey pins the invariant that ties the
// two filename decisions together: spellings that share a key must derive the
// same name, or the compiler would report a conflict between files that declare
// genuinely different things.
func TestFilenameDerivationAgreesWithCanonicalKey(t *testing.T) {
	for _, component := range []string{"employee_service", "EmployeeService", "employeeService"} {
		if got := upperCamelFilenameName(component); got != "EmployeeService" {
			t.Errorf("upperCamelFilenameName(%q) = %q, want EmployeeService", component, got)
		}
	}
}

func TestDuplicateSourceFilenamesDetectsCaseOnlyCollisions(t *testing.T) {
	tests := []struct {
		name     string
		basename string
		siblings []string
		want     []string
	}{
		{
			// The case-sensitive-filesystem collision this check exists for.
			name:     "case-only primary collision",
			basename: "Employee.fol",
			siblings: []string{"Employee.fol", "employee.fol", "Other.fol"},
			want:     []string{"employee.fol"},
		},
		{
			name:     "underscore and case spellings all collide",
			basename: "EmployeeService.fol",
			siblings: []string{"EmployeeService.fol", "employee_service.fol", "employeeService.fol"},
			want:     []string{"employeeService.fol", "employee_service.fol"},
		},
		{
			// The intended pairing: a companion shares its owner's key by design.
			name:     "companion does not collide with its owner",
			basename: "Employee.fol",
			siblings: []string{"Employee.fol", "Employee.comp.unit.fol"},
			want:     nil,
		},
		{
			name:     "companions collide with each other",
			basename: "Employee.comp.unit.fol",
			siblings: []string{"Employee.comp.unit.fol", "employee.comp.unit.fol"},
			want:     []string{"employee.comp.unit.fol"},
		},
		{
			name:     "unit fragments collide with each other",
			basename: "arithmetic.unit.fol",
			siblings: []string{"arithmetic.unit.fol", "Arithmetic.unit.fol"},
			want:     []string{"Arithmetic.unit.fol"},
		},
		{
			// A primary and a unit fragment are different source forms.
			name:     "unit does not collide with a primary",
			basename: "Employee.fol",
			siblings: []string{"Employee.fol", "employee.unit.fol"},
			want:     nil,
		},
		{
			// A component that derives no name cannot collide with one.
			name:     "invalid components are skipped",
			basename: "struct-body.fol",
			siblings: []string{"struct-body.fol", "Struct-Body.fol"},
			want:     nil,
		},
		{
			name:     "distinct names do not collide",
			basename: "Employee.fol",
			siblings: []string{"Employee.fol", "Employees.fol", "Manager.fol"},
			want:     nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			got := duplicateSourceFilenames(test.basename, test.siblings)
			if strings.Join(got, ",") != strings.Join(test.want, ",") {
				t.Fatalf("duplicateSourceFilenames = %v, want %v", got, test.want)
			}
		})
	}
}

// TestParseReportsFilenameCollision exercises the check end to end against a
// real folder on disk.
//
// The colliding pair is spelled Employee.fol/Employe_e.fol rather than
// Employee.fol/employee.fol on purpose: a case-only pair cannot be created on a
// case-insensitive filesystem, so a test written that way would silently pass on
// Windows and macOS by never constructing the situation. Both spellings coexist
// everywhere and share the same canonical key, so this exercises the identical
// code path portably. The case-only pair itself is covered by
// TestDuplicateSourceFilenamesDetectsCaseOnlyCollisions, which needs no
// filesystem at all.
func TestParseReportsFilenameCollision(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Employee.fol", "_ co.lang.struct = { id co.lang.int; }\n")
	// Underscores are dropped before folding, so this is the same canonical key.
	write("Employe_e.fol", "_ co.lang.struct = { id co.lang.int; }\n")

	toks := normalizeTokens(scanlex.Tokenize("_ co.lang.struct = { id co.lang.int; }\n", "Employee.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{
		Filename:      filepath.Join(dir, "Employee.fol"),
		Basename:      "Employee.fol",
		Basedir:       dir,
		PackagePath:   "people",
		LocationKnown: true,
		Source:        classifySourceFilename("Employee.fol"),
	}
	p.parseCompilationUnit()

	if len(p.diags) != 1 {
		t.Fatalf("diagnostics = %d, want exactly one filename collision: %v", len(p.diags), p.diags)
	}
	message := p.diags[0].Error()
	for _, want := range []string{"Employee.fol", "Employe_e.fol", "denotes the same declaration"} {
		if !strings.Contains(message, want) {
			t.Errorf("diagnostic %q does not mention %q", message, want)
		}
	}
}

// TestParseAcceptsOwnerAndCompanionInOneFolder guards the false positive that
// would matter most: the owner/companion pairing is the documented layout.
func TestParseAcceptsOwnerAndCompanionInOneFolder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Employee.fol", "Employee.comp.unit.fol", "arithmetic.unit.fol"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("_ co.lang.unit = {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	source := "_ co.lang.struct = { id co.lang.int; }\n"
	toks := normalizeTokens(scanlex.Tokenize(source, "Employee.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{
		Basename:      "Employee.fol",
		Basedir:       dir,
		PackagePath:   "people",
		LocationKnown: true,
		Source:        classifySourceFilename("Employee.fol"),
	}
	p.parseCompilationUnit()

	if len(p.diags) != 0 {
		t.Fatalf("owner/companion/unit layout produced diagnostics: %v", p.diags)
	}
}

// The canonical file key exists to detect duplicate DECLARATIONS, so it has one
// invariant: two stems share a key exactly when they are case variants of one
// derived declaration name (docs/language-ref.md, "Filename Canonicalization" —
// "Case variants and canonically equivalent spellings produce the same
// package-index key").
//
// The key is therefore defined as the case fold of the derived name rather than
// by a second, independent rule over the raw stem. This test is what stops the
// two from drifting: it asserts the equivalence directly rather than restating
// the implementation.
func TestCanonicalFileKeyIsTheCaseFoldOfTheDerivedName(t *testing.T) {
	stems := []string{
		"employee", "Employee", "EMPLOYEE",
		"employee_service", "EmployeeService", "employeeService", "employeeservice",
		"v1_hr", "V1Hr", "vendor", "Vendor2", "a_b_c",
	}

	for _, stem := range stems {
		want := strings.ToLower(upperCamelFilenameName(stem))
		if got := canonicalFileKey(stem); got != want {
			t.Errorf("canonicalFileKey(%q) = %q, want the case fold of %q, which is %q",
				stem, got, upperCamelFilenameName(stem), want)
		}
	}

	// Two stems collide exactly when their derived names are case variants.
	for _, a := range stems {
		for _, b := range stems {
			sameKey := canonicalFileKey(a) == canonicalFileKey(b)
			sameName := strings.EqualFold(upperCamelFilenameName(a), upperCamelFilenameName(b))
			if sameKey != sameName {
				t.Errorf("%q and %q: sameKey=%v but sameDerivedName=%v (%q vs %q)",
					a, b, sameKey, sameName,
					upperCamelFilenameName(a), upperCamelFilenameName(b))
			}
		}
	}
}

// The fold must not consult a locale. A Turkish-locale fold maps "I" to "ı",
// which would index the same filename differently on different machines.
func TestCanonicalFileKeyFoldIsLocaleInvariant(t *testing.T) {
	if got := canonicalFileKey("INDEX"); got != "index" {
		t.Errorf(`canonicalFileKey("INDEX") = %q, want "index"`, got)
	}
}
