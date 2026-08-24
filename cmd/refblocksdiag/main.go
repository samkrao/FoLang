// Command refblocksdiag is a throwaway diagnostic harness: it extracts the
// reference's folang blocks exactly as cmd/refblocks does and prints the parser
// diagnostics for the ones that fail, so a failure can be attributed to a rule
// rather than merely counted.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/src/parser"
)

const (
	referencePath = "docs/language-ref.md"
	corpusRoot    = "testdata/refblocks"
	fence         = "```"
	folangFence   = "```folang"
)

type block struct {
	line    int
	content string
	files   []string
}

func (b block) filename() string {
	if len(b.files) == 1 {
		return b.files[0]
	}
	return fmt.Sprintf("L%04d.fol", b.line)
}

func (b block) directory() string { return fmt.Sprintf("L%04d", b.line) }

var sourceFilePattern = regexp.MustCompile(`(?m)^\s*//\s*/?([A-Za-z0-9_./-]+\.fol)\b`)
var elisionPattern = regexp.MustCompile(`(^|[\s{])\.\.\.($|[\s}])`)

func namedFiles(content string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range sourceFilePattern.FindAllStringSubmatch(content, -1) {
		name := filepath.Base(strings.ReplaceAll(m[1], `\`, "/"))
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

func headingComment(lines []string, fenceIndex int) string {
	index := fenceIndex - 1
	for index >= 0 && strings.TrimSpace(lines[index]) == "" {
		index--
	}
	end := index
	for index >= 0 && strings.HasPrefix(strings.TrimSpace(lines[index]), "//") {
		index--
	}
	if index == end {
		return ""
	}
	return strings.Join(lines[index+1:end+1], "\n")
}

func extractBlocks(path string) ([]block, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	var blocks []block
	for i := 0; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") != folangFence {
			continue
		}
		start := i + 1
		var body []string
		for i++; i < len(lines) && !strings.HasPrefix(strings.TrimRight(lines[i], " \t"), fence); i++ {
			body = append(body, lines[i])
		}
		content := strings.TrimRight(strings.Join(body, "\n"), "\n") + "\n"
		files := namedFiles(content)
		if len(files) == 0 {
			files = namedFiles(headingComment(lines, start-1))
		}
		blocks = append(blocks, block{line: start, content: content, files: files})
	}
	return blocks, nil
}

func main() {
	only := flag.String("lines", "", "comma-separated opening line numbers to report")
	showSource := flag.Bool("src", false, "print the block source too")
	flag.Parse()

	want := map[int]bool{}
	if *only != "" {
		for _, f := range strings.Split(*only, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(f))
			if err == nil {
				want[n] = true
			}
		}
	}

	blocks, err := extractBlocks(referencePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, b := range blocks {
		if len(want) > 0 && !want[b.line] {
			continue
		}
		dir := filepath.Join(corpusRoot, "parsing", b.directory())
		result := parser.ParseFile(b.content, "refblocks", dir, b.filename(), "")
		if len(result.Diagnostics) == 0 {
			if len(want) > 0 {
				fmt.Printf("L%d %s: PARSES\n", b.line, b.filename())
			}
			continue
		}
		if len(want) == 0 && !elisionPattern.MatchString(b.content) {
			// unfiltered sweep: only the non-elided failures matter
		} else if len(want) == 0 {
			continue
		}
		fmt.Printf("=== L%d  %s  (elision=%v)\n", b.line, b.filename(),
			elisionPattern.MatchString(b.content))
		if *showSource {
			for i, l := range strings.Split(strings.TrimRight(b.content, "\n"), "\n") {
				fmt.Printf("   %3d | %s\n", i+1, l)
			}
		}
		for i, d := range result.Diagnostics {
			if i >= 4 {
				fmt.Printf("    ... +%d more\n", len(result.Diagnostics)-4)
				break
			}
			fmt.Printf("    -> %s\n", d.AsString())
		}
		fmt.Println()
	}
}
