package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// The compiler's own installation, located from the running executable.
//
// The location is derived from the REAL executable and from nothing else: not
// the working directory, not argv[0] as text, not a project file, a manifest, or
// an environment variable (docs/language-ref.md, "Installed Standard-Package
// Artifact"). A compiler that found its installation any other way could be made
// to load a standard package or a backend contract the toolchain did not ship.
//
// Two files live there and both are located this way:
//
//	<install-root>/bin/folcc                the running executable
//	<install-root>/bin/backend-conf.json    the backend interchange contract
//	<install-root>/stdlib/co.folenc         the standard package
var (
	// executablePath and evaluateSymlinks are seams so a test can stand an
	// installation up in a temporary directory.
	executablePath   = os.Executable
	evaluateSymlinks = filepath.EvalSymlinks
)

// ExecutableDirectory returns the directory holding the running compiler, with
// symbolic links resolved. It is where the backend interchange contract sits.
func ExecutableDirectory() (string, error) {
	executable, err := executablePath()
	if err != nil {
		return "", fmt.Errorf("locating the running compiler executable: %w", err)
	}
	real, err := evaluateSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("resolving compiler executable %s: %w", executable, err)
	}
	return filepath.Dir(real), nil
}

// InstallRoot returns the installation root, which is the parent of the
// executable's bin/ directory.
func InstallRoot() (string, error) {
	binDirectory, err := ExecutableDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Dir(binDirectory), nil
}

// UseInstallationForTest points the installation derivation at executable for
// the duration of one test, restoring it afterwards.
//
// It lives beside the derivation rather than in a test file because two packages
// stand installations up: the standard package is located from src/parser and
// the backend contract from here, and both must be able to place a real one in a
// temporary directory without each re-deriving the seams.
func UseInstallationForTest(t interface{ Cleanup(func()) }, executable string) {
	previousExecutable, previousSymlinks := executablePath, evaluateSymlinks
	t.Cleanup(func() {
		executablePath, evaluateSymlinks = previousExecutable, previousSymlinks
	})
	executablePath = func() (string, error) { return executable, nil }
	evaluateSymlinks = func(path string) (string, error) { return path, nil }
}
