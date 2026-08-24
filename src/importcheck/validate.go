package importcheck

// Whole-project validation.
//
// The individual rules live in their own files; this is the single entry point a driver calls
// once it has scanned every file in the project. Keeping the composition here means a driver
// does not have to know which checks exist or in what order they must run.

// ValidateProject runs every import-relationship check over a scanned project and returns all
// findings.
//
// files must contain one record per source file in the project, as produced by the scan pass.
// The function assigns library ownership first, because the internal-side rules depend on it,
// then runs the per-file checks, then the cross-file cycle check.
//
// files is modified in place: ownership is recorded on each record.
func ValidateProject(files []File) []error {
	AssignOwners(files)
	libraries := LibraryIndex(files)

	graph := NewGraph()
	var findings []error

	for _, f := range files {
		findings = append(findings, ValidateSelfImports(f)...)
		findings = append(findings, ValidateRestrictedImports(f)...)
		findings = append(findings, ValidateLibraryInternals(f)...)
		findings = append(findings, ValidateDependencyDirection(f, libraries)...)
		graph.Add(f)
	}

	// Cycles come last: they are the only check that needs every file's edges, so reporting
	// them after the local problems means a genuine cycle is not buried under the noise of a
	// misplaced import.
	findings = append(findings, graph.Validate()...)
	return findings
}
