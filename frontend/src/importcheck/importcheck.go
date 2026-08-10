// Package importcheck validates FoLang import relationships.
//
// It holds the rules that cannot be expressed in the grammar because they are about how
// declarations RELATE rather than how they are spelled. Two families live here:
//
//   - Restricted imports: what a library surface is allowed to reach for
//     (docs/language-ref.md, "Surface-to-Internal Dependency Direction" and
//     "Application-Workspace Source Library Surface"). These are decidable from one file,
//     because a file knows its own package identity and its own import list.
//
//   - Cycles: the compiler must reject a cycle through package imports
//     (docs/language-ref.md, "Cycles"). A cycle is a property of the
//     whole program, so it needs every file's edges before it can be decided. Graph
//     accumulates them and Validate walks the result.
//
// The package deliberately does not depend on the ast or parser packages. It takes plain
// records describing what was imported, so the checks can be driven by the parser today and
// by a whole-project driver later without either owning the other.
package importcheck

import (
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// Import is one parsed import directive, reduced to the fields the checks need.
//
// Exactly one of Package or Library is set: Package names a logical package path, Library
// names a library. Which DOMAIN the library comes from is what SrcLibrary selects
// (docs/language-ref.md, "Import Directive Fields").
type Import struct {
	// Package is the `package=` field: a logical package path such as "hr.employee".
	Package string
	// Library is the `library=` field. It names a packaged library under lib/, or —
	// when SrcLibrary is set — one of the standardized srclib/ slots.
	Library string
	// SrcLibrary is the `src-library=` flag. It switches Library's resolution from the
	// packaged-library domain lib/ to the project-local source-library domain srclib/,
	// and is valid only alongside Library.
	SrcLibrary bool
	// Alias is the `as=` field. When empty the full imported path must be used.
	Alias string

	// Start and End locate the directive, so a finding can point at it.
	Start helpers.Position
	End   helpers.Position
}

// target returns the path this import refers to, whichever field carried it.
func (i Import) target() string {
	if i.Package != "" {
		return i.Package
	}
	return i.Library
}

// isLibrarySurfaceImport reports whether the import targets a library's projected surface
// rather than an ordinary package.
//
// Both packaged libraries (`library=`) and source libraries (`src-library=true`) expose only
// their surface, and the two use exactly the same public-surface and API-projection rules.
func (i Import) isLibrarySurfaceImport() bool {
	return i.SrcLibrary || i.Library != ""
}

// File describes one parsed compilation unit's import surface.
type File struct {
	// Name is the file's base name, used in diagnostics.
	Name string
	// PackagePath is the file's own package identity, derived from its folder relative to
	// the project root. It is empty for a file at the root, which is not a package
	// (docs/language-ref.md, "Package Identity").
	PackagePath string

	// IsLibrarySurface reports whether this file is a library-surface file.
	IsLibrarySurface bool
	// LibraryName and LibraryType are the surface's declared name and kind, the latter
	// being one of application, dynamicvmrt, advanced, system or ffi.
	LibraryName string
	LibraryType string

	// LibraryPath is the package subtree this surface owns, set only on a surface file.
	//
	// It is WholeProject for a surface at the project root, which is a packaged library
	// project whose every subfolder is internal to it. For a source library below the root
	// it is the surface's own logical path, so com/abc/ffi.fol owns "com.abc.ffi".
	LibraryPath string

	// Owning describes the library this file is INSIDE, set only on a file that is one of
	// some library's internal package files. It is nil for an application file and for a
	// surface itself.
	Owning *Owner

	// Imports lists the file's import directives in source order.
	Imports []Import
}

// WholeProject is the LibraryPath of a surface at the project root, whose library encompasses
// every package in the project.
const WholeProject = "*"

// Owner identifies the library that contains an internal package file.
type Owner struct {
	// Name is the owning library's declared name.
	Name string
	// Type is the owning library's kind.
	Type string
	// Path is the package subtree the library owns, or WholeProject.
	Path string
	// SurfacePath is the logical path of the library's surface file, which its own internal
	// packages must not import.
	SurfacePath string
}

// owns reports whether pkg falls inside the owned subtree.
func (o Owner) owns(pkg string) bool {
	if o.Path == WholeProject {
		return true
	}
	return withinSubtree(pkg, o.Path)
}

// finding builds a diagnostic anchored at an import directive.
//
// A dedicated error name is used rather than "Invalid Syntax", because nothing here is a
// spelling problem: the directive parsed correctly and is disallowed by a relationship rule.
func finding(imp Import, name string, detail string) helpers.ErrorInterface {
	return helpers.NewExpectedTokenErrorName(imp.Start, imp.End, name, detail)
}
