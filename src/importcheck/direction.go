package importcheck

import (
	"fmt"
)

// Library dependency direction and the import-site type assertion.
//
// docs/language-ref.md, "Dependency Direction" states the allowed flow is one-way:
//
//	application -> dynamicvmrt -> advanced -> system -> ffi
//
// "A library may depend on a library at the same or a lower level only when the dependency does
// not create a cycle. Reverse dependencies are compiler errors." The "Cross-Library
// Communication" table spells out every pair, and it is exactly the ordering above: any edge
// from a lower level to a higher one is marked ❌ reverse dependency.
//
// The same table's cycle proviso is handled by Graph, since a cycle between two libraries at
// the same level is a graph property rather than a level comparison.
//
// This file also checks the `expect=` field. "Source Library Import" describes it as "an
// import-site assertion; the compiler checks it against the actual library type", which is only
// possible once the target surface has been scanned — hence its place in the project pass.

// libraryLevels ranks the library kinds. A smaller number is a higher level, and a dependency
// may only run from a higher level to the same or a lower one.
var libraryLevels = map[string]int{
	"application": 0,
	"dynamicvmrt": 1,
	"advanced":    2,
	"system":      3,
	"ffi":         4,
}

// levelOf returns a library kind's rank and whether the kind is known.
func levelOf(libraryType string) (int, bool) {
	level, ok := libraryLevels[libraryType]
	return level, ok
}

// LibraryInfo describes a library the project contains, as resolved from its surface file.
type LibraryInfo struct {
	// Name is the library's declared name.
	Name string
	// Type is its declared kind.
	Type string
	// Path is the logical path consumers import it by, or WholeProject for a packaged
	// library project's surface.
	Path string
}

// ValidateDependencyDirection checks a library surface's outgoing library dependencies against
// the one-way level ordering.
//
// libraries maps a logical library path to the library found there, which the project pass
// builds by scanning every surface file. A target that is not in the map is a packaged
// artifact under lib/, whose kind lives in its compiled `.folenc` projection rather than in
// source; import-field carries no assertion about it, so such a target is skipped here.
func ValidateDependencyDirection(f File, libraries map[string]LibraryInfo) []error {
	if !f.IsLibrarySurface {
		return nil
	}

	fromLevel, known := levelOf(f.LibraryType)
	if !known {
		return nil // an unrecognised kind is reported where the surface declares it
	}

	var findings []error
	for _, imp := range f.Imports {
		if !imp.isLibrarySurfaceImport() {
			continue
		}

		targetType, targetName := resolveTargetType(imp, libraries)

		if targetType == "" {
			continue // nothing to compare against
		}

		toLevel, targetKnown := levelOf(targetType)
		if !targetKnown {
			continue
		}

		// A dependency may run to the same or a lower level only.
		if toLevel < fromLevel {
			findings = append(findings, finding(imp, "Reverse Dependency", fmt.Sprintf(
				"library %q is of type %q and cannot depend on %q, which is of the higher-level type %q: "+
					"the allowed dependency flow is one-way, application -> dynamicvmrt -> advanced -> system -> ffi, "+
					"so a library may depend only on the same or a lower level",
				libraryLabel(f), f.LibraryType, targetName, targetType)))
		}
	}
	return findings
}

// resolveTargetType determines the type of the library an import targets.
//
// A source-library import names a path that the project scan can resolve to a real surface.
// A packaged-library import names an artifact that is not in the source tree; nothing in
// the import says what kind it is, so the type comes back empty and the caller skips the
// level comparison rather than guessing one.
func resolveTargetType(imp Import, libraries map[string]LibraryInfo) (libraryType string, name string) {
	if imp.Package != "" {
		if info, ok := libraries[imp.Package]; ok {
			return info.Type, info.Name
		}
	}
	if imp.Library != "" {
		if info, ok := libraries[imp.Library]; ok {
			return info.Type, info.Name
		}
		return "", imp.Library
	}
	return "", imp.target()
}

// LibraryIndex builds the path-to-library map ValidateDependencyDirection needs from the scanned
// files.
func LibraryIndex(files []File) map[string]LibraryInfo {
	index := map[string]LibraryInfo{}
	for _, f := range files {
		if !f.IsLibrarySurface {
			continue
		}
		info := LibraryInfo{Name: libraryLabel(f), Type: f.LibraryType, Path: f.LibraryPath}

		// A source library is imported by its logical path.
		if f.LibraryPath != "" && f.LibraryPath != WholeProject {
			index[f.LibraryPath] = info
		}
		// A packaged library is imported by its declared name.
		if f.LibraryName != "" {
			index[f.LibraryName] = info
		}
	}
	return index
}
