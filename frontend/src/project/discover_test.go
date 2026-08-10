package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverRejectsInvalidExplicitProjectRelationships(t *testing.T) {
	base := t.TempDir()
	target := writeProjectFile(t, filepath.Join(base, "outside", "main.fol"))

	t.Run("missing root", func(t *testing.T) {
		_, err := Discover(target, filepath.Join(base, "missing"))
		if err == nil || !strings.Contains(err.Error(), "checking project root") {
			t.Fatalf("Discover missing root error = %v", err)
		}
	})

	t.Run("root is a file", func(t *testing.T) {
		rootFile := writeProjectFile(t, filepath.Join(base, "not-a-directory"))
		_, err := Discover(target, rootFile)
		if err == nil || !strings.Contains(err.Error(), "is not a directory") {
			t.Fatalf("Discover file root error = %v", err)
		}
	})

	t.Run("target outside root", func(t *testing.T) {
		root := filepath.Join(base, "project")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := Discover(target, root)
		if err == nil || !strings.Contains(err.Error(), "outside project root") {
			t.Fatalf("Discover outside target error = %v", err)
		}
	})
}

func TestCollectSourceFilesPropagatesRootWalkFailure(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, err := collectSourceFiles(missing, nil)
	if err == nil || !strings.Contains(err.Error(), "walking project root") {
		t.Fatalf("collectSourceFiles missing root error = %v", err)
	}
}

func TestDiscoverUsesDefaultRootOutputFoldersOnly(t *testing.T) {
	root := t.TempDir()
	writeProjectFile(t, filepath.Join(root, projectMarker))
	target := writeProjectFile(t, filepath.Join(root, "main.fol"))

	for _, path := range []string{
		filepath.Join(root, "out", "generated.fol"),
		filepath.Join(root, "lib", "generated.fol"),
		filepath.Join(root, "build", "generated.fol"),
		filepath.Join(root, "src", "node_modules", "generated.fol"),
	} {
		writeProjectFile(t, path)
	}
	for _, path := range []string{
		filepath.Join(root, "src", "out", "source.fol"),
		filepath.Join(root, "src", "lib", "source.fol"),
		filepath.Join(root, "src", "build", "source.fol"),
		filepath.Join(root, "src", "ast", "source.fol"),
		filepath.Join(root, "src", "libs", "source.fol"),
	} {
		writeProjectFile(t, path)
	}

	project, err := Discover(target, root)
	if err != nil {
		t.Fatal(err)
	}
	got := discoveredRelativePaths(t, project)
	want := []string{
		"main.fol",
		filepath.Join("src", "ast", "source.fol"),
		filepath.Join("src", "build", "source.fol"),
		filepath.Join("src", "lib", "source.fol"),
		filepath.Join("src", "libs", "source.fol"),
		filepath.Join("src", "out", "source.fol"),
	}
	assertStringSlices(t, got, want)
}

func TestDiscoverUsesConfiguredRootRelativeOutputFolders(t *testing.T) {
	root := t.TempDir()
	config := `fol-lang:
  output_folder: generated/out
  lib_folder: artifacts/libs # compiled libraries
  exe_folder: bin
`
	writeProjectFileWithContent(t, filepath.Join(root, projectMarker), config)
	target := writeProjectFile(t, filepath.Join(root, "main.fol"))

	for _, path := range []string{
		filepath.Join(root, "generated", "out", "generated.fol"),
		filepath.Join(root, "artifacts", "libs", "generated.fol"),
		filepath.Join(root, "bin", "generated.fol"),
	} {
		writeProjectFile(t, path)
	}
	for _, path := range []string{
		filepath.Join(root, "out", "source.fol"),
		filepath.Join(root, "lib", "source.fol"),
		filepath.Join(root, "build", "source.fol"),
		filepath.Join(root, "src", "lib", "source.fol"),
		filepath.Join(root, "src", "build", "source.fol"),
	} {
		writeProjectFile(t, path)
	}

	project, err := Discover(target, root)
	if err != nil {
		t.Fatal(err)
	}
	got := discoveredRelativePaths(t, project)
	// Root-level lib/ and build/ are the standardized non-source domains and are
	// skipped whatever the configured output folders are: lib/ holds compiled
	// artifacts and build/ is compiler-managed output, so a .fol file in either is a
	// layout error rather than something to compile. The identically named packages
	// under src/ are ordinary source and stay discoverable.
	want := []string{
		"main.fol",
		filepath.Join("out", "source.fol"),
		filepath.Join("src", "build", "source.fol"),
		filepath.Join("src", "lib", "source.fol"),
	}
	assertStringSlices(t, got, want)
}

func TestConfiguredOutputFolderScalarKeepsLiteralHash(t *testing.T) {
	root := t.TempDir()
	writeProjectFileWithContent(t, filepath.Join(root, projectMarker), "output_folder: generated#cache\n")
	target := writeProjectFile(t, filepath.Join(root, "main.fol"))
	writeProjectFile(t, filepath.Join(root, "generated#cache", "generated.fol"))

	project, err := Discover(target, root)
	if err != nil {
		t.Fatal(err)
	}
	assertStringSlices(t, discoveredRelativePaths(t, project), []string{"main.fol"})
}

func writeProjectFile(t *testing.T, path string) string {
	t.Helper()
	return writeProjectFileWithContent(t, path, "\n")
}

func writeProjectFileWithContent(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func discoveredRelativePaths(t *testing.T, project *Project) []string {
	t.Helper()
	paths := make([]string, len(project.Files))
	for index, file := range project.Files {
		rel, err := filepath.Rel(project.Root, file.Path)
		if err != nil {
			t.Fatal(err)
		}
		paths[index] = rel
	}
	return paths
}

func assertStringSlices(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("paths = %q, want %q", got, want)
		}
	}
}
