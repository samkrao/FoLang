// Package project discovers the layout of a FoLang project on disk.
//
// The import-relationship rules are stated in terms of layout, so they cannot be checked
// without it. Two facts in particular are only knowable from the file tree:
//
//   - A subfolder containing source files IS a package, and the root itself is not
//     (docs/language-ref.md, "Package Identity"). A file's package path is therefore its
//     folder relative to the root, with separators turned into dots.
//
//   - Whether a library surface sits AT the root or BELOW it decides which packages it owns.
//     A packaged library project has "exactly one @co.dap.library surface file" and "the
//     surface file is located at the project root", so every subfolder is internal to it
//     (docs/language-ref.md, "Packaged Library Project Surface"). A source library in an
//     application workspace "must be below the application root, never at the application
//     root", so it owns only the subtree named by its own path
//     (docs/language-ref.md, "Application-Workspace Source Library Surface").
//
// This package answers both. It reads no source content beyond locating files; classifying a
// file as an entry, a package source file or a library surface is the parser's job.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// sourceExtensions are the file extensions treated as FoLang source.
//
// ".fol" is the extension used throughout the reference.
var sourceExtensions = map[string]bool{
	".fol": true,
}

// projectMarker is the per-project configuration file. Its presence marks the project root.
const projectMarker = "fol-conf.yaml"

// outputFolders are directory names excluded from the source walk. They correspond to the
// output, library and executable folders named in fol-conf.yaml, plus the usual tooling
// directories.
var outputFolders = map[string]bool{
	"out":          true,
	"lib":          true,
	"libs":         true,
	"build":        true,
	"ast":          true,
	".git":         true,
	".vscode":      true,
	"node_modules": true,
}

// File is one discovered source file.
type File struct {
	// Path is the absolute path on disk.
	Path string
	// Base is the file name with its extension, as diagnostics report it.
	Base string
	// Stem is the file name without its extension. For a library surface this is the last
	// segment of the library's logical path.
	Stem string
	// PackagePath is the dot-separated path of the file's FOLDER relative to the project
	// root. It is empty for a file at the root, which is not a package.
	PackagePath string
	// AtRoot reports whether the file sits directly in the project root.
	AtRoot bool
}

// LogicalPath returns the dot path that names this file itself, rather than its folder.
//
// It is the path a source-library import uses: a surface at com/abc/ffi.fol is imported as
// `package="com.abc.ffi"`, which is its folder path followed by its own stem
// (docs/language-ref.md, "Source Library Import").
func (f File) LogicalPath() string {
	if f.PackagePath == "" {
		return f.Stem
	}
	return f.PackagePath + "." + f.Stem
}

// Project is a discovered project tree.
type Project struct {
	// Root is the absolute project root directory.
	Root string
	// Files lists every source file found under Root, in stable path order.
	Files []File
	// MarkerFound reports whether the root was located by finding fol-conf.yaml rather
	// than by falling back to the target file's own directory.
	MarkerFound bool
}

// Discover locates the project containing target and enumerates its source files.
//
// The root is chosen in order of preference:
//
//  1. rootOverride, when the caller supplies one.
//  2. The nearest ancestor directory of target containing fol-conf.yaml.
//
// In both of those cases the whole tree below the root is enumerated. Failing both, the target
// file's own directory becomes the root and ONLY the target file is enumerated: with no marker
// there is no evidence of the project's extent, so a whole-directory walk would be a guess. A
// caller that wants the full project should pass rootOverride or add a marker file.
func Discover(target string, rootOverride string) (*Project, error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %w", target, err)
	}

	root, markerFound, err := resolveRoot(absTarget, rootOverride)
	if err != nil {
		return nil, err
	}

	// Without a marker or an explicit root there is no evidence of how far the project
	// extends, so only the target file is compiled. Walking its directory would be a guess,
	// and a wrong guess is actively harmful: a loose file in a directory that happens to
	// contain other FoLang trees would pull them in and compute package paths relative to a
	// root none of them share, producing cross-project findings about unrelated code.
	if !markerFound && rootOverride == "" {
		file, describeErr := describeFile(root, absTarget)
		if describeErr != nil {
			return nil, describeErr
		}
		return &Project{Root: root, Files: []File{file}, MarkerFound: false}, nil
	}

	files, err := collectSourceFiles(root)
	if err != nil {
		return nil, err
	}

	return &Project{Root: root, Files: files, MarkerFound: markerFound}, nil
}

// resolveRoot applies the root-selection order documented on Discover.
func resolveRoot(absTarget string, rootOverride string) (root string, markerFound bool, err error) {
	if rootOverride != "" {
		abs, absErr := filepath.Abs(rootOverride)
		if absErr != nil {
			return "", false, fmt.Errorf("resolving project root %s: %w", rootOverride, absErr)
		}
		return abs, false, nil
	}

	dir := filepath.Dir(absTarget)
	if marker := findMarkerUpward(dir); marker != "" {
		return marker, true, nil
	}
	return dir, false, nil
}

// findMarkerUpward walks from dir toward the filesystem root looking for the project marker,
// returning the first directory that contains it or "" if none does.
func findMarkerUpward(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, projectMarker)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // reached the filesystem root
		}
		dir = parent
	}
}

// collectSourceFiles walks root and returns every source file, skipping output and tooling
// directories.
func collectSourceFiles(root string) ([]File, error) {
	var files []File

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// An unreadable directory should not abort discovery of the rest.
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() {
			if path != root && outputFolders[strings.ToLower(entry.Name())] {
				return filepath.SkipDir
			}
			return nil
		}

		if !sourceExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			return nil
		}

		file, buildErr := describeFile(root, path)
		if buildErr != nil {
			return nil // a path that will not relativize is not part of this project
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking project root %s: %w", root, err)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// describeFile builds a File record, computing the package path from the file's folder.
func describeFile(root, path string) (File, error) {
	rel, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return File{}, err
	}

	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	// A file at the root has no package path, because the root is not a package.
	packagePath := ""
	atRoot := rel == "."
	if !atRoot {
		packagePath = strings.ReplaceAll(rel, string(filepath.Separator), ".")
		packagePath = strings.TrimPrefix(packagePath, ".")
	}

	return File{
		Path:        path,
		Base:        base,
		Stem:        stem,
		PackagePath: packagePath,
		AtRoot:      atRoot,
	}, nil
}

// PackagePathOf returns the package path a file at the given path would have in this project.
func (p *Project) PackagePathOf(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	for _, f := range p.Files {
		if f.Path == abs {
			return f.PackagePath
		}
	}
	file, err := describeFile(p.Root, abs)
	if err != nil {
		return ""
	}
	return file.PackagePath
}
