package project

import (
	"fmt"
	"os"
	"path/filepath"
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
	// KindStandaloneLibrary has src/library.fol.
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
