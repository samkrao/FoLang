// Throwaway: parse every .fol under examples/ and report failures.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/src/parser"
)

// exampleParseContext supplies the virtual filename required by the source form
// demonstrated by one pedagogical example. The examples tree is a catalogue,
// not one compilable project: several independent examples in the same chapter
// can each demonstrate an application entry file, so their on-disk catalogue
// names cannot all literally be appl.fol.
func exampleParseContext(path, source string) (string, string) {
	base := filepath.Base(path)
	switch {
	case strings.Contains(source, "_ co.lang.component"):
		base = "component.fol"
	case strings.Contains(source, "_ co.lang.unit"):
		if strings.Contains(strings.ToLower(source), "companion unit") {
			owner := strings.TrimSuffix(filepath.Base(path), ".fol")
			base = owner + ".comp.unit.fol"
		} else if !strings.HasSuffix(strings.ToLower(base), ".unit.fol") {
			base = strings.TrimSuffix(base, ".fol") + ".unit.fol"
		}
	case strings.Contains(strings.ToLower(source), "application entry file"),
		strings.EqualFold(base, "app.fol"):
		base = "appl.fol"
	}
	for _, legacy := range []string{".signature.fol", ".enum.fol"} {
		if strings.HasSuffix(strings.ToLower(base), legacy) {
			base = strings.TrimSuffix(base, legacy) + ".fol"
		}
	}
	// No catalogue sibling participates in compilation; an empty directory also
	// prevents the parser from applying project-layout rules to this snippet.
	return "", base
}

func main() {
	root := "examples"
	var paths []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".fol") {
			paths = append(paths, p)
		}
		return nil
	})
	sort.Strings(paths)

	ok, bad := 0, 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		dir, base := exampleParseContext(p, string(src))
		result := parser.ParseFile(string(src), "examples", dir, base, "")
		if len(result.Diagnostics) == 0 {
			ok++
			continue
		}
		bad++
		fmt.Printf("%s\n    %s\n", p, firstLine(result.Diagnostics[0].Error()))
	}
	fmt.Printf("\nparsed %d  failed %d  total %d\n", ok, bad, len(paths))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
