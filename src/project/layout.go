package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/src/helpers"
)

// Layout rules for the standardized project-root domains
// (docs/language-ref.md, "Project Layout" and "Project-Domain Presence Invariant").
//
// FoLang distinguishes ABSENCE from an EMPTY DECLARED DOMAIN. For the developer-owned
// input domains, physically creating the directory expresses intent to participate in
// the project, so `lib/` must hold valid content once it exists. `build/`
// is exempt: it is compiler-owned generated state and may be absent, empty or full.
//
// These are project-wide facts that no single file can establish, which is why they are
// checked here rather than in the parser. A file is still parsed on its own terms when
// the layout around it is wrong; the findings are reported alongside its diagnostics.

// ProjectKind is what the presence of a fixed structural surface classifies the project
// as.
type ProjectKind int

const (
	// KindUnknown is a tree whose layout was never validated, which is the legacy
	// single-file path.
	KindUnknown ProjectKind = iota
	// KindApplication has src/appl.fol.
	KindApplication
	// KindStandaloneLibrary has src/component.fol, under either exposure model.
	KindStandaloneLibrary
)

// Layout is the validated shape of a project's standardized domains.
type Layout struct {
	// Kind is what src/'s structural surface classifies the project as.
	Kind ProjectKind
	// EntrySurface is the absolute path of src/appl.fol, when present.
	EntrySurface string
	// LibrarySurface is the absolute path of src/component.fol, when present. That
	// surface makes the project a standalone library under either exposure model:
	// projected when it carries @co.dap.library, packaged when it does not
	// (docs/language-ref.md, "Form Exclusivity").
	LibrarySurface string
	// Findings are the layout violations found, in a stable order.
	Findings []error
}

// ValidateLayout checks a project root against the domain rules and reports every
// violation it finds rather than stopping at the first, so one run names everything that
// has to change.
func ValidateLayout(root string) Layout {
	layout := Layout{}

	layout.validateSourceDomain(root)
	layout.validateComponentDomain(root)
	layout.validatePackagedLibraryDomain(root)

	return layout
}

// The file kinds a project domain may hold.
const (
	// sourceFileExtension is FoLang source, which lives under src/ and components/.
	sourceFileExtension = ".fol"
	// artifactFileExtension is a compiled library artifact, which lives in lib/.
	artifactFileExtension = ".folenc"
	// standardArtifactFilename is the reserved standard-package identity. It is
	// loaded from <install-root>/stdlib/ and a project copy of it is an error:
	// "project content cannot shadow or replace the installed co.* packages"
	// (docs/language-ref.md, "Standard Package Location").
	standardArtifactFilename = "co.folenc"
	// operatorsComponentKind is the one component kind a projected application
	// library may keep, because custom operator syntax must be registered while
	// compiling that library's own source.
	operatorsComponentKind = "operators"
)

// validateSourceDomain checks src/, which is the one mandatory domain and the one that
// classifies the project.
func (l *Layout) validateSourceDomain(root string) {
	dir := filepath.Join(root, SourceDomain)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		l.report("%s/ is missing; every FoLang project has a src/ domain holding either %s or %s",
			SourceDomain, ApplicationEntryFilename, ComponentSurfaceFilename)
		return
	}
	if err != nil {
		l.report("reading %s/: %v", SourceDomain, err)
		return
	}

	hasEntry := containsFile(entries, ApplicationEntryFilename)
	hasLibrary := containsFile(entries, ComponentSurfaceFilename)

	switch {
	case hasEntry && hasLibrary:
		l.report("%s/ contains both %s and %s; a project is an application OR a standalone library, so exactly one structural surface may be present",
			SourceDomain, ApplicationEntryFilename, ComponentSurfaceFilename)
	case hasEntry:
		l.Kind = KindApplication
		l.EntrySurface = filepath.Join(dir, ApplicationEntryFilename)
	case hasLibrary:
		l.Kind = KindStandaloneLibrary
		l.LibrarySurface = filepath.Join(dir, ComponentSurfaceFilename)
	default:
		l.report("%s/ has no structural surface; add %s for an application or %s for a standalone packaged library",
			SourceDomain, ApplicationEntryFilename, ComponentSurfaceFilename)
	}

	// "No other file may occur directly in src/. Every other direct entry under
	// src/ must be a non-empty package directory containing valid FoLang package
	// source" (docs/language-ref.md, "src/ — Primary Project Source").
	//
	// An empty package directory is checked here rather than during discovery
	// because discovery only ever sees FILES: a directory holding none is
	// invisible to it, which is exactly the case this rule is about.
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			if name == ApplicationEntryFilename || name == ComponentSurfaceFilename {
				continue
			}
			l.report("%s/%s occurs directly in %s/, which holds one structural surface and package directories; move it into a package directory",
				SourceDomain, name, SourceDomain)
			continue
		}
		l.validateNonEmptyPackage(filepath.Join(dir, name), SourceDomain+"/"+name)
	}
}

// validateNonEmptyPackage reports a package directory that holds no source at all.
//
// "Every other direct entry under src/ must be a NON-EMPTY package directory
// containing valid FoLang package source" (docs/language-ref.md, "src/ — Primary
// Project Source").
//
// Source ANYWHERE beneath the directory satisfies that. An intermediate package
// that only groups subpackages is ordinary — `src/hr/` holding just
// `src/hr/employee/` is the reference's own Package Identity example — so the
// violation is a directory with nothing under it at any depth, which names a
// package whose every dot path resolves to nothing.
func (l *Layout) validateNonEmptyPackage(dir, label string) {
	if l.holdsSource(dir, label) {
		return
	}
	l.report("%s/ is a package directory holding no FoLang source at any depth; omit it or add the source it names", label)
}

// holdsSource reports whether dir contains a .fol file at any depth.
func (l *Layout) holdsSource(dir, label string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		l.report("reading %s/: %v", label, err)
		// The directory could not be read, so its contents are unknown. Reporting
		// it empty as well would name a second violation from one failure.
		return true
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if l.holdsSource(filepath.Join(dir, entry.Name()), label+"/"+entry.Name()) {
				return true
			}
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), sourceFileExtension) {
			return true
		}
	}
	return false
}

// validateComponentDomain checks components/, which is optional but must be valid
// once it physically exists.
//
// It is checked HERE rather than during discovery because every rule it carries is
// about DIRECTORIES. An empty components/, an unknown immediate child, and a
// component kind holding no surface are all invisible to a walk that only sees
// .fol files, which is why the reference states them as layout rules
// (docs/language-ref.md, "Project Layout" and "components/ — Project-Owned
// Components").
//
// WHICH kinds a project may own is not decided here. That rule turns on the
// exposure model written inside src/component.fol — only a projected APPLICATION
// library keeps the components/operators/ exception, and a packaged, native or
// dynamicvmrt library may hold no components/ tree at all — and a filesystem walk
// cannot read an annotation. The parser applies it once the surface is parsed.
func (l *Layout) validateComponentDomain(root string) {
	dir := filepath.Join(root, ComponentDomain)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		l.report("reading %s/: %v", ComponentDomain, err)
		return
	}
	if len(entries) == 0 {
		l.report("%s/ is present but empty; omit the directory when the project owns no components", ComponentDomain)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			l.report("%s/%s occurs directly in %s/, which holds only the standardized component-kind directories %s",
				ComponentDomain, name, ComponentDomain, componentKindList())
			continue
		}
		if !ComponentKinds[name] {
			l.report("%s/%s is not a standardized component kind; %s/ admits only %s",
				ComponentDomain, name, ComponentDomain, componentKindList())
			continue
		}
		l.validateComponentKind(filepath.Join(dir, name), name)
	}
}

// validateComponentKind checks one standardized components/ child. Each holds
// exactly one direct component.fol surface: "Every component-kind directory
// contains exactly one direct structural source file named component.fol, and
// every such file contains exactly one _ co.lang.component declaration."
func (l *Layout) validateComponentKind(dir, kind string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		l.report("reading %s/%s/: %v", ComponentDomain, kind, err)
		return
	}
	if !containsFile(entries, ComponentSurfaceFilename) {
		l.report("%s/%s/ has no %s; every component kind holds exactly one direct structural surface",
			ComponentDomain, kind, ComponentSurfaceFilename)
	}
	// "Every component-kind directory contains exactly one direct structural
	// source file named component.fol… No alternative direct component-surface
	// filename is valid" (docs/language-ref.md, "components/ — Project-Owned
	// Components"). Implementation source belongs in a descendant package
	// directory, where it is component-private; a second source file sitting
	// beside the surface belongs to no package and would be read as a second
	// surface.
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == ComponentSurfaceFilename {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), sourceFileExtension) {
			l.report("%s/%s/%s sits beside %s; a component kind holds one direct surface, and its implementation source belongs in a descendant package directory",
				ComponentDomain, kind, entry.Name(), ComponentSurfaceFilename)
		}
	}
	// The operator component contributes syntax metadata only and "permits no
	// descendant package directories".
	if kind != operatorsComponentKind {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			l.report("%s/%s/%s is a package directory; the operator component permits none and contributes syntax metadata only",
				ComponentDomain, kind, entry.Name())
		}
	}
}

// componentKindList renders the standardized kind names for a diagnostic.
func componentKindList() string {
	names := make([]string, 0, len(ComponentKinds))
	for name := range ComponentKinds {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// validatePackagedLibraryDomain checks lib/, which holds compiled artifacts only.
func (l *Layout) validatePackagedLibraryDomain(root string) {
	dir := filepath.Join(root, PackagedLibraryDomain)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return // absent is valid: the project has no packaged dependencies
	}
	if err != nil {
		l.report("reading %s/: %v", PackagedLibraryDomain, err)
		return
	}
	if len(entries) == 0 {
		l.report("%s/ is present but empty; omit the directory when the project has no packaged .folenc dependencies", PackagedLibraryDomain)
		return
	}

	// lib/ admits DIRECT compiled artifacts and nothing else: the reference lists
	// "FoLang source files, or non-.folenc project content" among what it may not
	// hold. A directory is neither a direct artifact nor discoverable source, so
	// it would otherwise sit there unreported and unread.
	artifacts := 0
	for _, entry := range entries {
		if entry.IsDir() {
			l.report("%s/%s is a directory; %s/ holds direct compiled .folenc artifacts and has no nested structure",
				PackagedLibraryDomain, entry.Name(), PackagedLibraryDomain)
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case artifactFileExtension:
			artifacts++
			if strings.EqualFold(entry.Name(), standardArtifactFilename) {
				l.report("%s/%s cannot shadow the installed standard package; co.* is loaded only from <install-root>/stdlib/co.folenc",
					PackagedLibraryDomain, standardArtifactFilename)
			}
		case sourceFileExtension:
			l.report("%s/%s is FoLang source; %s/ holds compiled .folenc artifacts and never participates in source discovery",
				PackagedLibraryDomain, entry.Name(), PackagedLibraryDomain)
		default:
			l.report("%s/%s is not a compiled %s artifact; %s/ holds no other project content",
				PackagedLibraryDomain, entry.Name(), artifactFileExtension, PackagedLibraryDomain)
		}
	}
	if artifacts == 0 {
		l.report("%s/ contains no .folenc artifact; a present %s/ must hold one or more compiled FoLang library artifacts",
			PackagedLibraryDomain, PackagedLibraryDomain)
	}
}

// report records one layout violation.
//
// A layout finding is about the project TREE rather than about a place in a file, so it
// carries no meaningful source position. It is still built as an ordinary diagnostic so
// that it travels the same reporting path as every other finding instead of being
// silently dropped by a consumer that only understands diagnostics.
func (l *Layout) report(format string, args ...any) {
	var nowhere helpers.Position
	l.Findings = append(l.Findings, helpers.NewInvalidSyntaxError(nowhere, nowhere,
		"project layout: "+fmt.Sprintf(format, args...)))
}

// containsFile reports whether entries holds a regular file of the given name.
func containsFile(entries []os.DirEntry, name string) bool {
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == name {
			return true
		}
	}
	return false
}
