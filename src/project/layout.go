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
// the project, so `srclib/` and `lib/` must hold valid content once they exist. `build/`
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
	// KindStandaloneLibrary has src/library.fol.
	KindStandaloneLibrary
)

// Layout is the validated shape of a project's standardized domains.
type Layout struct {
	// Kind is what src/'s structural surface classifies the project as.
	Kind ProjectKind
	// EntrySurface is the absolute path of src/appl.fol, when present.
	EntrySurface string
	// LibrarySurface is the absolute path of src/library.fol, when present.
	LibrarySurface string
	// SourceLibraries maps each present srclib/ slot to the absolute path of its
	// fixed library.fol surface. A slot whose surface is missing is still listed, with
	// an empty path, because the missing file is itself reported.
	SourceLibraries map[string]string
	// Findings are the layout violations found, in a stable order.
	Findings []error
}

// OperatorBootstrap returns the path of srclib/operators/library.fol, or "" when the
// project declares no project-local operators.
func (l Layout) OperatorBootstrap() string {
	return l.SourceLibraries[OperatorsLibrarySlot]
}

// ValidateLayout checks a project root against the domain rules and reports every
// violation it finds rather than stopping at the first, so one run names everything that
// has to change.
func ValidateLayout(root string) Layout {
	layout := Layout{SourceLibraries: map[string]string{}}

	layout.validateSourceDomain(root)
	layout.validateSourceLibraryDomain(root)
	layout.validatePackagedLibraryDomain(root)

	return layout
}

// validateSourceDomain checks src/, which is the one mandatory domain and the one that
// classifies the project.
func (l *Layout) validateSourceDomain(root string) {
	dir := filepath.Join(root, SourceDomain)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		l.report("%s/ is missing; every FoLang project has a src/ domain holding either %s or %s",
			SourceDomain, ApplicationEntryFilename, LibrarySurfaceFilename)
		return
	}
	if err != nil {
		l.report("reading %s/: %v", SourceDomain, err)
		return
	}

	hasEntry := containsFile(entries, ApplicationEntryFilename)
	hasLibrary := containsFile(entries, LibrarySurfaceFilename)

	switch {
	case hasEntry && hasLibrary:
		l.report("%s/ contains both %s and %s; a project is an application OR a standalone library, so exactly one structural surface may be present",
			SourceDomain, ApplicationEntryFilename, LibrarySurfaceFilename)
	case hasEntry:
		l.Kind = KindApplication
		l.EntrySurface = filepath.Join(dir, ApplicationEntryFilename)
	case hasLibrary:
		l.Kind = KindStandaloneLibrary
		l.LibrarySurface = filepath.Join(dir, LibrarySurfaceFilename)
	default:
		l.report("%s/ has no structural surface; add %s for an application or %s for a standalone packaged library",
			SourceDomain, ApplicationEntryFilename, LibrarySurfaceFilename)
	}
}

// validateSourceLibraryDomain checks srclib/, which is optional but must be valid once
// it physically exists.
func (l *Layout) validateSourceLibraryDomain(root string) {
	dir := filepath.Join(root, SourceLibraryDomain)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return // absent is valid: the project uses no project-local source libraries
	}
	if err != nil {
		l.report("reading %s/: %v", SourceLibraryDomain, err)
		return
	}
	if len(entries) == 0 {
		l.report("%s/ is present but empty; omit the directory when the project has no project-local source libraries or operator bootstrap", SourceLibraryDomain)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			l.report("%s/%s is not one of the standardized source-library directories; %s/ holds only %s",
				SourceLibraryDomain, name, SourceLibraryDomain, sourceLibrarySlotList())
			continue
		}
		if !sourceLibrarySlots[name] {
			l.report("%s/%s/ is not a standardized source-library slot; only %s are permitted",
				SourceLibraryDomain, name, sourceLibrarySlotList())
			continue
		}
		l.validateSourceLibrarySlot(dir, name)
	}
}

// validateSourceLibrarySlot checks one standardized srclib/ child.
//
// Every slot holds exactly one direct `library.fol`. They differ in what else may sit
// beside it: an ordinary slot may hold internal package directories, while operators/
// holds that one file and nothing at all besides.
func (l *Layout) validateSourceLibrarySlot(srclibDir, slot string) {
	dir := filepath.Join(srclibDir, slot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		l.report("reading %s/%s/: %v", SourceLibraryDomain, slot, err)
		return
	}

	if !containsFile(entries, LibrarySurfaceFilename) {
		l.report("%s/%s/ has no %s; every source-library slot has exactly one fixed surface file of that name",
			SourceLibraryDomain, slot, LibrarySurfaceFilename)
		l.SourceLibraries[slot] = ""
	} else {
		l.SourceLibraries[slot] = filepath.Join(dir, LibrarySurfaceFilename)
	}

	for _, entry := range entries {
		name := entry.Name()
		switch {
		case !entry.IsDir() && name == LibrarySurfaceFilename:
			// The fixed surface itself.

		case slot == OperatorsLibrarySlot:
			// The operator bootstrap slot holds one file and nothing else, because
			// its surface is a parser bootstrap rather than an importable API with
			// an implementation behind it.
			l.report("%s/%s/ contains %q; the operator bootstrap slot holds exactly one file named %s and no subdirectories",
				SourceLibraryDomain, slot, name, LibrarySurfaceFilename)

		case !entry.IsDir():
			l.report("%s/%s/%s is a loose file; a source-library root holds its one %s, and every other entry must be an internal package directory",
				SourceLibraryDomain, slot, name, LibrarySurfaceFilename)

		default:
			l.validateNoNestedSurface(dir, filepath.Join(SourceLibraryDomain, slot), name)
		}
	}
}

// validateNoNestedSurface reports a `library.fol` below a source-library root.
//
// Nested source-library boundaries are forbidden: a slot's own root is the boundary, so
// a second surface deeper in the tree would claim a library inside a library.
func (l *Layout) validateNoNestedSurface(slotDir, slotLabel, child string) {
	_ = filepath.WalkDir(filepath.Join(slotDir, child), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Name() != LibrarySurfaceFilename {
			return nil
		}
		rel, err := filepath.Rel(slotDir, path)
		if err != nil {
			rel = path
		}
		l.report("%s/%s is a nested %s; a source library's root is its only boundary, so no implementation package may declare its own surface",
			slotLabel, filepath.ToSlash(rel), LibrarySurfaceFilename)
		return nil
	})
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

	artifacts := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".folenc":
			artifacts++
			if strings.EqualFold(entry.Name(), "co.folenc") {
				l.report("%s/co.folenc cannot shadow the installed standard package; co.* is loaded only from <install-root>/stdlib/co.folenc", PackagedLibraryDomain)
			}
		case ".fol":
			l.report("%s/%s is FoLang source; %s/ holds compiled .folenc artifacts and never participates in source discovery",
				PackagedLibraryDomain, entry.Name(), PackagedLibraryDomain)
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

// sourceLibrarySlotList renders the standardized slot names for a diagnostic.
func sourceLibrarySlotList() string {
	names := make([]string, 0, len(sourceLibrarySlots))
	for name := range sourceLibrarySlots {
		names = append(names, name+"/")
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
