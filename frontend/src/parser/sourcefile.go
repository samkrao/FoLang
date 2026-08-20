package parser

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/project"
)

// Source filenames — section 1 of docs/grammar/folang.ebnf.
//
//	source-filename            = companion-unit-filename
//	                           | ordinary-unit-filename
//	                           | application-entry-filename
//	                           | component-surface-filename
//	                           | ordinary-source-filename
//	companion-unit-filename     = filename-identifier, ".comp.unit.fol"
//	ordinary-unit-filename      = filename-identifier, ".unit.fol"
//	application-entry-filename  = "appl.fol"
//	component-surface-filename  = "component.fol"
//	ordinary-source-filename    = filename-identifier, ".fol"
//	filename-identifier         = identifier-head, { "_", identifier-segment }
//
// This is an EXTERNAL metadata grammar. It is never concatenated with the source
// text: the compiler classifies the filename FIRST and the classification is what
// selects the source-text root, because `_ co.lang.unit` alone cannot say whether
// a unit merges into the package namespace or attaches to a struct.
//
// Classification uses the longest recognized suffix first, so
// `Employee.comp.unit.fol` is never read as an ordinary `.unit.fol` file
// (docs/language-ref.md, "Package Source Files").
//
// TWO exact filenames are reserved and never read as an identifier-derived
// component: `appl.fol` and `component.fol`, the fixed structural surfaces. Neither
// contributes a name — `appl.fol` never implies an application called `appl`
// (docs/language-ref.md, "Structural Source Filenames").
//
// `package.fol` is NOT among them. The reference states that FoLang "defines no
// co.lang.package declaration kind and no reserved package.fol metadata form", and
// that such a file "is classified by the ordinary <Name>.fol filename rule", so it
// declares an ordinary primary named `Package` like any other stem.
//
// The identifier-derived component reuses the ordinary ASCII identifier shape, but a
// filename component is not a source token: it undergoes no reserved-word
// classification, and its canonical UpperCamelCase spelling is produced by
// normalization here rather than by the scanner.

// sourceFileClass is the source-filename alternative a file was classified as.
type sourceFileClass int

const (
	// sourceClassUnknown is the legacy Parse case: no usable basename, so no
	// alternative of source-filename applies and no name can be derived.
	sourceClassUnknown sourceFileClass = iota
	// sourceClassPrimary is ordinary-source-filename, `<Name>.fol`, which
	// carries one file-backed primary-declaration.
	sourceClassPrimary
	// sourceClassOrdinaryUnit is ordinary-unit-filename, `<Fragment>.unit.fol`.
	// The fragment is organizational only; its members merge into the package
	// namespace.
	sourceClassOrdinaryUnit
	// sourceClassCompanionUnit is companion-unit-filename,
	// `<StructName>.comp.unit.fol`. The filename supplies the companion owner.
	sourceClassCompanionUnit
	// sourceClassApplicationEntry is the reserved application-entry-filename,
	// `appl.fol`. It selects application-entry-file as the source-text root.
	sourceClassApplicationEntry
	// sourceClassComponentSurface is the reserved component-surface-filename,
	// `component.fol`. It selects component-surface-file. Like the two above it
	// contributes no name; unlike them its DIRECTORY additionally supplies the
	// component kind, because `src/component.fol` and every
	// `components/<kind>/component.fol` share this one filename.
	sourceClassComponentSurface
)

// sourceFilename is the parsed form of one external source filename.
type sourceFilename struct {
	// Class is the alternative of source-filename that matched.
	Class sourceFileClass
	// Component is the identifier-derived part, with the recognized structural
	// suffix removed. It is empty for the two reserved exact filenames.
	Component string
	// Valid reports whether Component is a well-formed filename-identifier. A
	// reserved exact filename is always valid.
	Valid bool
	// DerivedName is Component normalized to the canonical UpperCamelCase
	// declaration spelling (DECISION-FILE-002). It is the public name of a
	// file-backed primary and the owner of a companion unit. It is empty when
	// Component is not a filename-identifier.
	DerivedName string
}

// The reserved exact filenames, which keep their context-defined classification rather
// than being read as identifier-derived components.
const (
	applicationEntryFilename   = project.ApplicationEntryFilename
	componentSurfaceFilename   = "component.fol"
	componentDomain            = "components"
	componentKindApplication   = "application"
	componentKindNative        = "native"
	componentKindDynamicVMRT   = "dynamicvmrt"
	componentKindPackaged      = "packaged"
	componentKindOperators     = "operators"
	componentKindStandaloneSrc = ""
)

// componentKinds is the closed set of project-local component kinds, each fixed
// by the immediate `components/<kind>/` directory the surface sits in
// (docs/language-ref.md, "components/ — Project-Owned Components").
var componentKinds = map[string]bool{
	componentKindApplication: true,
	componentKindNative:      true,
	componentKindDynamicVMRT: true,
	componentKindPackaged:    true,
	componentKindOperators:   true,
}

// classifySourceFilename parses one source-filename.
//
// Implements: source-filename
// Implements: companion-unit-filename
// Implements: ordinary-unit-filename
// Implements: application-entry-filename
// Implements: component-surface-filename
// Implements: ordinary-source-filename
func classifySourceFilename(basename string) sourceFilename {
	if basename == "" {
		return sourceFilename{Class: sourceClassUnknown}
	}

	// The reserved exact filenames are recognized before any suffix rule, so none of
	// them is ever read as an identifier-derived component.
	switch basename {
	case applicationEntryFilename:
		return sourceFilename{Class: sourceClassApplicationEntry, Valid: true}
	case componentSurfaceFilename:
		return sourceFilename{Class: sourceClassComponentSurface, Valid: true}
	}

	// Longest recognized suffix first: `.comp.unit.fol` before `.unit.fol`
	// before `.fol`.
	class := sourceClassUnknown
	component := ""
	switch {
	case strings.HasSuffix(basename, ".comp.unit.fol"):
		class, component = sourceClassCompanionUnit, strings.TrimSuffix(basename, ".comp.unit.fol")
	case strings.HasSuffix(basename, ".unit.fol"):
		class, component = sourceClassOrdinaryUnit, strings.TrimSuffix(basename, ".unit.fol")
	case strings.HasSuffix(basename, ".fol"):
		class, component = sourceClassPrimary, strings.TrimSuffix(basename, ".fol")
	default:
		return sourceFilename{Class: sourceClassUnknown}
	}

	parsed := sourceFilename{Class: class, Component: component}
	// filename-identifier reuses the ordinary ASCII identifier shape: an ASCII
	// letter head, then ASCII letters, digits and isolated internal
	// underscores, with no Unicode and no leading, trailing or doubled "_".
	if isFilenameIdentifier(component) {
		parsed.Valid = true
		parsed.DerivedName = upperCamelFilenameName(component)
	}
	return parsed
}

// isFilenameIdentifier reports whether component is a filename-identifier.
//
// The shape is identical to the source identifier's, which is why the check is
// shared: DECISION-FILE-002 states the reuse explicitly, and duplicating it
// here would let the two definitions drift.
//
// Implements: filename-identifier
func isFilenameIdentifier(component string) bool {
	return isFoLangIdentifier(component)
}

// upperCamelFilenameName renders a filename-identifier in the canonical
// UpperCamelCase declaration spelling (DECISION-FILE-002).
//
// Underscores are explicit word boundaries. Within one segment, an existing
// lower-to-upper boundary is preserved, so a mixed-case segment keeps its
// internal words; a single-case segment carries no boundary information at all
// and is case-folded and capitalized as ONE word. That is why a multiword name
// written wholly in one case must use underscores to keep its boundaries:
//
//	employee        -> Employee
//	EMPLOYEE        -> Employee
//	employeeRecord  -> EmployeeRecord      (boundary preserved)
//	employee_record -> EmployeeRecord      (explicit boundary)
//	employeerecord  -> Employeerecord      (no boundary to recover)
func upperCamelFilenameName(component string) string {
	var out strings.Builder
	for _, segment := range strings.Split(component, "_") {
		if segment == "" {
			continue
		}
		if isSingleCaseSegment(segment) {
			out.WriteString(capitalizeASCII(strings.ToLower(segment)))
			continue
		}
		out.WriteString(capitalizeASCII(segment))
	}
	return out.String()
}

// isSingleCaseSegment reports whether a segment's letters are all one case, in
// which case it carries no lower-to-upper boundary to preserve.
func isSingleCaseSegment(segment string) bool {
	hasUpper, hasLower := false, false
	for i := 0; i < len(segment); i++ {
		switch c := segment[i]; {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		}
	}
	return !hasUpper || !hasLower
}

// capitalizeASCII upper-cases the first byte, which is an ASCII letter in every
// filename-identifier.
func capitalizeASCII(text string) string {
	if text == "" {
		return ""
	}
	if c := text[0]; c >= 'a' && c <= 'z' {
		return string(c-'a'+'A') + text[1:]
	}
	return text
}

// canonicalFileKey returns the package-index key of a filename component
// (DECISION-FILE-004):
//
//	canonical file key = asciiCaseFold(normalize(filename stem))
//
// normalize is upperCamelFilenameName, the SAME derivation that produces the
// declaration name. Stating the key that way is what keeps the index honest:
// the key exists to detect duplicate declarations, so two stems must share a key
// exactly when they are case variants of one derived declaration name. Deriving
// the key by an independent rule leaves two definitions of one relationship free
// to drift, and the reference gives only one — "Case variants and canonically
// equivalent spellings produce the same package-index key".
//
// The partition this produces is worth spelling out, because it looks surprising
// until the transitivity is followed through:
//
//	employee_service.fol  -> EmployeeService  //	EmployeeService.fol   -> EmployeeService   |  one key, one declaration
//	employeeService.fol   -> EmployeeService   |
//	employeeservice.fol   -> Employeeservice  /
//
// The last stem derives a DIFFERENT name, yet shares the key, and must: it is a
// case variant of EmployeeService.fol, which the reference requires to conflict,
// and conflict has to be transitive for a key-based index to mean anything. A
// single-case segment carries no word boundary to recover, which is why the name
// differs while the identity does not.
//
// A component is ASCII by construction (DECISION-FILE-002), so no Unicode
// normalization applies. The fold is written out here rather than taken from
// strings.ToLower's locale-free guarantee being assumed: the decision requires a
// LOCALE-INVARIANT fold, because a Turkish-locale fold maps "I" to "ı" and would
// index the same filename differently on different machines.
func canonicalFileKey(component string) string {
	return asciiCaseFold(upperCamelFilenameName(component))
}

// asciiCaseFold lower-cases the ASCII letters of text without consulting a locale.
func asciiCaseFold(text string) string {
	var folded strings.Builder
	folded.Grow(len(text))
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c >= 'A' && c <= 'Z' {
			c = c - 'A' + 'a'
		}
		folded.WriteByte(c)
	}
	return folded.String()
}

// duplicateSourceFilenames returns the sibling filenames that denote the same
// declaration as basename.
//
// This is the check a case-sensitive filesystem makes necessary. `Employee.fol`
// and `employee.fol` are two distinct files there, yet both derive the
// declaration `Employee`, so the package would silently gain two declarations of
// one name; on a case-insensitive filesystem the same pair cannot coexist at
// all. The reference therefore requires the conflict to be reported from the
// canonical key and states that the compiler must not rely on operating-system
// filename comparison (docs/language-ref.md, "Filename Canonicalization").
//
// Only files of the same classification are compared. `Employee.fol` and
// `Employee.comp.unit.fol` share a key by design — that pairing is how a
// companion finds its owner — so comparing across classes would report the
// intended layout as an error.
//
// A component that is not a valid filename-identifier is skipped: it derives no
// name, so it cannot collide with one.
func duplicateSourceFilenames(basename string, siblings []string) []string {
	self := classifySourceFilename(basename)
	if !self.Valid || self.Component == "" {
		return nil
	}

	key := canonicalFileKey(self.Component)
	var conflicts []string
	for _, sibling := range siblings {
		if sibling == basename {
			continue
		}
		other := classifySourceFilename(sibling)
		if other.Class != self.Class || !other.Valid || other.Component == "" {
			continue
		}
		if canonicalFileKey(other.Component) == key {
			conflicts = append(conflicts, sibling)
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

// componentKindOf returns the project-local component kind a component.fol
// surface belongs to, or "" when the surface is the standalone src/component.fol.
//
// The fixed `component.fol` name is shared by every component surface, so the
// ENCLOSING directory is the only thing that says which kind this one is. That
// is exactly what the reference means by "Its immediate folder determines its
// component kind before component.fol is parsed" — the kind is known before a
// token is read, which is why it is derived here rather than from the body.
//
// A directory that is neither `components/<kind>/` nor anything recognizable
// yields "" as well: the standalone surface decides its own exposure model from
// its annotations, and validating the placement itself belongs to the project
// layout rather than to a single file's parse.
func componentKindOf(basedir string) string {
	if basedir == "" {
		return componentKindStandaloneSrc
	}
	clean := strings.TrimRight(filepath.ToSlash(basedir), "/")
	slash := strings.LastIndexByte(clean, '/')
	if slash < 0 {
		return componentKindStandaloneSrc
	}
	kind := clean[slash+1:]
	if filepath.Base(clean[:slash]) != componentDomain || !componentKinds[kind] {
		return componentKindStandaloneSrc
	}
	return kind
}

// isUnitSourceFile reports whether the filename classifies this file as one of
// the two unit source forms, which share the unit-declaration grammar and
// differ only in how their members are indexed.
func (f sourceFilename) isUnitSourceFile() bool {
	return f.Class == sourceClassOrdinaryUnit || f.Class == sourceClassCompanionUnit
}

// describeClass names the source form for a diagnostic.
func (f sourceFilename) describeClass() string {
	switch f.Class {
	case sourceClassPrimary:
		return "a file-backed primary declaration file"
	case sourceClassOrdinaryUnit:
		return "an ordinary package unit file"
	case sourceClassCompanionUnit:
		return "a companion unit file"
	case sourceClassApplicationEntry:
		return "the fixed src/appl.fol application entry file"
	case sourceClassComponentSurface:
		return "a fixed component.fol surface file"
	default:
		return "a source file with no recognized FoLang filename form"
	}
}
