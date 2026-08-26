package importcheck

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/src/helpers"
)

// Restricted imports.
//
// A library surface is a boundary, and the spec constrains what may cross it in the inward
// direction. Two rules from docs/language-ref.md are enforced here.
//
// 1. The surface may reach its OWN internal packages and nothing else's.
//
//    "Surface-to-Internal Dependency Direction" states the source-level dependency is
//    one-way: the library surface imports and invokes internal packages, and internal
//    packages do not import the surface. "Library Compilation Order" gives the reason —
//    internal packages are compiled without depending on surface types — and
//    "Surface-to-Internal Dependency Direction" notes this "prevents a surface/internal
//    compilation cycle".
//
//    So an ordinary package import from a surface is legitimate when it names a package
//    inside that library's own subtree, and is a boundary violation otherwise. Reaching
//    outside means reaching either an application package or another library's internals.
//
// 2. Another library is reachable only through its projected surface.
//
//    "Application-Workspace Source Library Surface" states that "only the projected surface
//    API is importable", that "internal packages remain hidden even though their source files
//    are physically available", and that "once a source tree is treated as a library, its
//    subpackages cannot be imported as ordinary packages by consumers". A packaged library is
//    identical in this respect: "Only the packaged library's projected surface API is visible
//    to the consumer."
//
//    So an import that names another library's projected surface, via `library=`,
//    is allowed, while an ordinary `package=` import that lands inside another
//    library is not.
//
// Both reduce to the same test, which is why one function decides them: from a library
// surface, an ordinary package import must stay within the surface's own package subtree.

// ValidateRestrictedImports checks a file's imports against the library-boundary rules and
// returns one diagnostic per violation.
//
// Only a library-surface file is constrained. An application entry file and an ordinary
// package source file may import any package they can see, subject to the visibility rules
// the semantic phase enforces separately.
//
// Whether a file that is itself INSIDE a source library obeys rule 1 cannot be decided here:
// that needs the project layout to know which files belong to which library. This function
// therefore covers the surface, which is where the boundary is declared, and a whole-project
// driver is required for the internal side.
func ValidateRestrictedImports(f File) []error {
	if !f.IsLibrarySurface {
		return nil
	}

	// A packaged library project's surface sits at the root and every subfolder is internal
	// to it, so no ordinary package import can leave its boundary.
	if f.LibraryPath == WholeProject {
		return nil
	}

	owned := f.LibraryPath
	if owned == "" {
		// Without a discovered layout the surface's own package path is the best
		// available approximation of its subtree.
		owned = f.PackagePath
	}

	var findings []error
	for _, imp := range f.Imports {
		// A library-surface import reaches another library's public surface, which is
		// always permitted at this level; the dependency-direction rules that govern
		// WHICH libraries may be reached are checked separately.
		if imp.isLibrarySurfaceImport() {
			continue
		}

		target := imp.Package
		if target == "" {
			continue // nothing named; the parser has already reported the malformed field
		}

		if withinSubtree(target, owned) {
			continue // the surface reaching its own internals, which is the allowed direction
		}

		findings = append(findings, finding(imp, helpers.DiagnosticInvalidImport, "Restricted Import", fmt.Sprintf(
			"library surface %q cannot import the package %q: a library surface may import only its own internal packages (those under %q), "+
				"and must reach any other library through its projected surface using library=. "+
				"Importing an application package or another library's subpackages would reverse the one-way surface dependency direction",
			libraryLabel(f), target, subtreeLabel(owned))))
	}
	return findings
}

// ValidateLibraryInternals checks the imports of a file that lies INSIDE a library.
//
// This is the other half of the boundary rule, and it needs the project layout to know which
// library owns the file — which is why it is driven by the project walk rather than by a
// single-file parse.
//
// Two rules apply, both from docs/language-ref.md,
// "Surface-to-Internal Dependency Direction":
//
//  1. Internal packages "do not import the library surface". The dependency is one-way, and
//     "Library Compilation Order" explains why: internal packages are compiled before the
//     surface's adapter bodies, so an internal file depending on the surface would be a
//     compilation cycle.
//
//  2. An internal package may not reach outside its own library for an ordinary package.
//     Reaching sideways lands in another library's internals, which are hidden even when
//     physically present; reaching upward lands in an application package, which reverses the
//     dependency direction. Another library remains reachable through its projected surface.
func ValidateLibraryInternals(f File) []error {
	if f.Owning == nil || f.IsLibrarySurface {
		return nil
	}
	owner := *f.Owning

	var findings []error
	for _, imp := range f.Imports {
		if imp.isLibrarySurfaceImport() {
			continue
		}

		target := imp.Package
		if target == "" {
			continue
		}

		// A package importing itself is reported by ValidateSelfImports. It reaches here
		// too, because a source library's surface path and the package path of the files
		// directly inside it are the same string — com/abc/ffi.fol is imported as
		// "com.abc.ffi" and com/abc/ffi/Guts.fol lives in package "com.abc.ffi". The
		// self-import message is the accurate one, so this rule stands aside.
		if target == f.PackagePath {
			continue
		}

		// Rule 1: an internal package must not import its own library's surface.
		if owner.SurfacePath != "" && target == owner.SurfacePath {
			findings = append(findings, finding(imp, helpers.DiagnosticInvalidImport, "Restricted Import", fmt.Sprintf(
				"package %q is internal to library %q and cannot import that library's surface %q: "+
					"the surface-to-internal dependency is one-way, and internal packages are compiled before the surface's boundary adapters",
				f.PackagePath, owner.Name, owner.SurfacePath)))
			continue
		}

		// Rule 2: an internal package must not reach outside its own library.
		if !owner.owns(target) {
			findings = append(findings, finding(imp, helpers.DiagnosticInvalidImport, "Restricted Import", fmt.Sprintf(
				"package %q is internal to library %q and cannot import the package %q: an internal package may import only packages under %q, "+
					"and must reach any other library through its projected surface using library=",
				f.PackagePath, owner.Name, target, subtreeLabel(owner.Path))))
		}
	}
	return findings
}

// withinSubtree reports whether pkg is root or lies beneath it in the dot-separated package
// hierarchy.
//
// Comparison is segment-wise so that "com.abc.ffi" contains "com.abc.ffi.internal" but not
// "com.abc.ffixture", which a plain string-prefix test would wrongly accept.
func withinSubtree(pkg, root string) bool {
	if root == "" {
		// A surface at the project root owns no package subtree, so it has no internal
		// packages to reach and every ordinary package import leaves its boundary. The
		// spec requires a source-library surface to sit below the application root, so
		// this case only arises for a malformed layout.
		return false
	}
	if pkg == root {
		return true
	}
	return strings.HasPrefix(pkg, root+".")
}

// libraryLabel names the library in a diagnostic, preferring its declared name.
func libraryLabel(f File) string {
	if f.LibraryName != "" {
		return f.LibraryName
	}
	if f.Name != "" {
		return f.Name
	}
	return "<unnamed>"
}

// subtreeLabel renders the package subtree a surface owns, for a diagnostic.
func subtreeLabel(root string) string {
	switch root {
	case "":
		return "<none: the surface owns no package subtree>"
	case WholeProject:
		return "the whole project"
	default:
		return root
	}
}

// AssignOwners fills in the Owning field of every file that lies inside a library.
//
// Ownership is by longest matching subtree, so a source library nested inside another is
// attributed to the innermost one. A surface file is never owned by its own library: it IS the
// boundary, and its rules are ValidateRestrictedImports's business.
//
// files is modified in place.
func AssignOwners(files []File) {
	type library struct {
		owner Owner
		depth int
	}

	var libraries []library
	for _, f := range files {
		if !f.IsLibrarySurface {
			continue
		}
		libraries = append(libraries, library{
			owner: Owner{
				Name:        libraryLabel(f),
				Type:        f.LibraryType,
				Path:        f.LibraryPath,
				SurfacePath: surfacePathOf(f),
			},
			depth: subtreeDepth(f.LibraryPath),
		})
	}
	if len(libraries) == 0 {
		return
	}

	for i := range files {
		if files[i].IsLibrarySurface || files[i].PackagePath == "" {
			continue
		}

		best := -1
		for j, lib := range libraries {
			if !lib.owner.owns(files[i].PackagePath) {
				continue
			}
			if best == -1 || lib.depth > libraries[best].depth {
				best = j
			}
		}
		if best >= 0 {
			owner := libraries[best].owner
			files[i].Owning = &owner
		}
	}
}

// surfacePathOf returns the logical path of a surface file, which its internals must not import.
//
// For a source library that is its LibraryPath. For a packaged library project the surface owns
// the whole project and has no meaningful package path, so there is no importable surface path
// to guard against.
func surfacePathOf(f File) string {
	if f.LibraryPath == WholeProject {
		return ""
	}
	return f.LibraryPath
}

// subtreeDepth measures how specific an owned subtree is, so the innermost library wins when
// libraries nest. WholeProject is the least specific.
func subtreeDepth(path string) int {
	if path == WholeProject || path == "" {
		return 0
	}
	return strings.Count(path, ".") + 1
}
