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
		result := parser.ParseFile(string(src), "examples", filepath.Dir(p), filepath.Base(p), "")
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
