package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CompilationProjectKind uint8

const (
	CompilationApplication CompilationProjectKind = iota + 1
	CompilationStandaloneComponent
)

// ValidateCompilationRoot applies the current structural source rule without
// relying on the withdrawn library.fol/srclib layout model, which no longer exists.
func ValidateCompilationRoot(root string) (CompilationProjectKind, []error) {
	entries, err := os.ReadDir(filepath.Join(root, SourceDomain))
	if err != nil {
		return 0, []error{fmt.Errorf("reading src/: %w", err)}
	}
	hasApplication, hasComponent := false, false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch entry.Name() {
		case ApplicationEntryFilename:
			hasApplication = true
		case ComponentSurfaceFilename:
			hasComponent = true
		default:
			return 0, []error{fmt.Errorf("src/%s is a loose file; only appl.fol or component.fol may occur directly in src/", entry.Name())}
		}
	}
	switch {
	case hasApplication && hasComponent:
		return 0, []error{fmt.Errorf("src/ contains both appl.fol and component.fol; exactly one structural surface is required")}
	case hasApplication:
		return CompilationApplication, nil
	case hasComponent:
		return CompilationStandaloneComponent, nil
	default:
		return 0, []error{fmt.Errorf("src/ has no structural surface; add appl.fol or component.fol")}
	}
}

// CompilationStage is the mandatory project preparation order. Components are
// source inputs that establish local APIs and syntax; compiled libraries are
// loaded next; primary src is parsed only after both environments exist.
type CompilationStage uint8

const (
	StageLibraries CompilationStage = iota + 1
	StageComponents
	StagePrimarySource
)

const (
	ComponentDomain          = "components"
	ComponentSurfaceFilename = "component.fol"
)

var ComponentKinds = map[string]bool{
	"application": true,
	"native":      true,
	"dynamicvmrt": true,
	"packaged":    true,
	"operators":   true,
}

// CompilationInput describes one input without conflating its filesystem
// location with a package name.
type CompilationInput struct {
	Path          string
	Stage         CompilationStage
	ComponentKind string
	Surface       bool
	PackagePath   string
}

// CompilationInputs returns inputs in dependency order: independent compiled
// libraries first, then project-owned components in their fixed dependency
// order, and finally primary src.
func CompilationInputs(root string, files []File) ([]CompilationInput, error) {
	var inputs []CompilationInput
	seenSurfaces := map[string]bool{}
	seenComponents := map[string]bool{}
	sourceFiles := map[string]bool{}
	for _, file := range files {
		sourceFiles[filepath.Clean(file.Path)] = true
	}

	for _, file := range files {
		rel, err := filepath.Rel(root, file.Path)
		if err != nil {
			return nil, fmt.Errorf("classifying project input %s: %w", file.Path, err)
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		switch {
		case len(parts) >= 3 && parts[0] == ComponentDomain:
			kind := parts[1]
			seenComponents[kind] = true
			if !ComponentKinds[kind] {
				return nil, fmt.Errorf("%s/%s is not a standardized component kind", ComponentDomain, kind)
			}
			isSurface := len(parts) == 3 && parts[2] == ComponentSurfaceFilename
			if isSurface {
				if seenSurfaces[kind] {
					return nil, fmt.Errorf("component %s has more than one %s", kind, ComponentSurfaceFilename)
				}
				seenSurfaces[kind] = true
			}
			if kind == "operators" && !isSurface {
				return nil, fmt.Errorf("components/operators contains only component.fol and no implementation packages")
			}
			packagePath := ""
			if len(parts) > 3 {
				packagePath = strings.Join(parts[2:len(parts)-1], ".")
			}
			inputs = append(inputs, CompilationInput{Path: file.Path, Stage: StageComponents,
				ComponentKind: kind, Surface: isSurface, PackagePath: packagePath})

		case len(parts) >= 2 && parts[0] == SourceDomain:
			if strings.HasSuffix(parts[len(parts)-1], ".comp.unit.fol") {
				owner := strings.TrimSuffix(parts[len(parts)-1], ".comp.unit.fol") + ".fol"
				ownerPath := filepath.Join(filepath.Dir(file.Path), owner)
				if !sourceFiles[filepath.Clean(ownerPath)] {
					return nil, fmt.Errorf("companion unit %s requires owner type file %s", file.Path, ownerPath)
				}
			}
			inputs = append(inputs, CompilationInput{Path: file.Path, Stage: StagePrimarySource,
				Surface:     len(parts) == 2 && (parts[1] == ApplicationEntryFilename || parts[1] == ComponentSurfaceFilename),
				PackagePath: file.PackagePath})
		}
	}
	for kind := range seenComponents {
		if !seenSurfaces[kind] {
			return nil, fmt.Errorf("components/%s has no component.fol surface", kind)
		}
	}

	artifacts, err := filepath.Glob(filepath.Join(root, PackagedLibraryDomain, "*.folenc"))
	if err != nil {
		return nil, fmt.Errorf("discovering compiled libraries: %w", err)
	}
	for _, path := range artifacts {
		inputs = append(inputs, CompilationInput{Path: path, Stage: StageLibraries, Surface: true})
	}

	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i].Stage != inputs[j].Stage {
			return inputs[i].Stage < inputs[j].Stage
		}
		if inputs[i].ComponentKind != inputs[j].ComponentKind {
			left, right := componentPreparationRank(inputs[i].ComponentKind), componentPreparationRank(inputs[j].ComponentKind)
			if left != right {
				return left < right
			}
			return inputs[i].ComponentKind < inputs[j].ComponentKind
		}
		if inputs[i].Stage == StagePrimarySource {
			left, right := primarySourceRank(inputs[i].Path), primarySourceRank(inputs[j].Path)
			if left != right {
				return left < right
			}
		}
		// A component surface establishes the boundary before its private
		// implementation packages are parsed.
		if inputs[i].Surface != inputs[j].Surface {
			return inputs[i].Surface
		}
		return inputs[i].Path < inputs[j].Path
	})
	return inputs, nil
}

// componentPreparationRank encodes the one-way dependency flow. Operators are
// an independent tokenizer bootstrap; the remaining components may depend only
// on kinds appearing before them here.
func componentPreparationRank(kind string) int {
	switch kind {
	case "operators":
		return 0
	case "native":
		return 1
	case "application":
		return 2
	case "dynamicvmrt":
		return 3
	case "packaged":
		return 4
	default:
		return 5
	}
}

// primarySourceRank makes declaration availability deterministic: file-backed
// types are parsed first, then their companion units, then ordinary units, and
// finally the root application/component surface.
func primarySourceRank(path string) int {
	name := filepath.Base(path)
	switch {
	case strings.HasSuffix(name, ".comp.unit.fol"):
		return 1
	case strings.HasSuffix(name, ".unit.fol"):
		return 2
	case name == ApplicationEntryFilename || name == ComponentSurfaceFilename:
		return 3
	default:
		return 0
	}
}
