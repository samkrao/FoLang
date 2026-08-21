// Command refblocks re-extracts the reference-block corpora from
// docs/language-ref.md.
//
// The corpora under testdata/refblocks/ are every ```folang block in the
// reference, split three ways:
//
//	parsing/   the block parses; the parser must keep it parsing
//	invalid/   the block demonstrates an error and must be rejected
//	excluded/  the block is not a parseable compilation unit as written,
//	           or is one the parser cannot yet accept (a tracked gap)
//
// Each file is named L<line>.fol after the line the block opens on, so a corpus
// entry can be found in the reference. That naming is also why the corpora rot:
// editing the reference renumbers every block below the edit, and nothing
// re-extracts them, so the files drift until a third of the "must parse" corpus
// no longer parses.
//
// # Preserving judgement across a re-extraction
//
// The excluded/invalid split encodes decisions a person made — in particular
// whether a rejected block is wrong AS WRITTEN or is a parser gap worth fixing.
// Re-extraction must not silently discard that. Classification is therefore
// carried forward by BLOCK CONTENT rather than by filename, which survives the
// renumbering that breaks everything else.
//
// A block whose content is not in any existing corpus is new. If it parses it
// joins parsing/; if it does not, it is reported by name and left for a person
// to classify rather than being guessed into a bucket.
//
// Usage, from frontend/:
//
//	go run ./cmd/refblocks           report what would change
//	go run ./cmd/refblocks -write     rewrite the corpora
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/parser"
)

const (
	referencePath = "../docs/language-ref.md"
	corpusRoot    = "testdata/refblocks"
	fence         = "```"
	folangFence   = "```folang"
)

// category is one of the three corpus directories.
type category string

const (
	catParsing  category = "parsing"
	catInvalid  category = "invalid"
	catExcluded category = "excluded"
)

// block is one fenced folang block from the reference.
type block struct {
	line    int      // 1-based line the opening fence sits on
	content string   // the block body, verbatim
	files   []string // source filenames the block names in comments
}

// filename returns the source filename the block should be parsed under.
//
// FoLang classifies a source file BY ITS NAME — `package.fol` is package
// metadata, `x.unit.fol` is a unit, `X.fol` is a file-backed primary — so
// parsing a block under the wrong name misclassifies it and reports errors the
// compiler never would. The reference names each block's file in a leading
// comment for exactly this reason, and honouring it is what makes extraction
// faithful rather than approximate.
//
// A block that names no file is an ordinary primary named after its line.
func (b block) filename() string {
	if len(b.files) == 1 {
		return b.files[0]
	}
	return fmt.Sprintf("L%04d.fol", b.line)
}

// directory is the per-block folder a corpus entry lives in.
//
// One folder per block is what lets the entry keep the reference's own filename:
// `package.fol`, `appl.fol` and `library.fol` are reserved exact spellings that cannot be
// prefixed with a line number without ceasing to be themselves.
func (b block) directory() string {
	return fmt.Sprintf("L%04d", b.line)
}

// sourceFilePattern matches a leading comment that names a FoLang source file,
// as in `// Employee.fol` or `//Functor.fol`.
var sourceFilePattern = regexp.MustCompile(`(?m)^\s*//\s*/?([A-Za-z0-9_./-]+\.fol)\b`)

// isOperatorBootstrapPath reports whether a named source file is the fixed operator
// bootstrap surface, however much of its path the reference wrote out.
func isOperatorBootstrapPath(named string) bool {
	slashed := strings.ReplaceAll(named, `\`, "/")
	return strings.HasSuffix(slashed, "operators/library.fol")
}

// classified is a block with the bucket it belongs in.
type classified struct {
	block
	category category
	reason   string
	isNew    bool
}

func main() {
	write := flag.Bool("write", false, "rewrite the corpora instead of reporting")
	flag.Parse()

	blocks, err := extractBlocks(referencePath)
	if err != nil {
		fail(err)
	}

	known, reasons, err := loadExistingClassifications()
	if err != nil {
		fail(err)
	}

	results := make([]classified, 0, len(blocks))
	unclassified := make([]classified, 0)
	for _, b := range blocks {
		key := hashOf(b.content)
		if cat, ok := known[key]; ok {
			// A block whose text is unchanged keeps the bucket it was put in, but
			// the two buckets that make a claim about parsing are re-checked: the
			// whole point of the corpus is that the claims stay true.
			switch {
			case cat == catParsing && !parses(b):
				if reason, byDesign := inferByDesignExclusion(b); byDesign {
					results = append(results, classified{block: b, category: catExcluded, reason: reason})
					continue
				}
				// It was expected to parse and no longer does. That is a
				// regression or a deliberate grammar change, and either way a
				// person decides.
				c := classified{block: b, category: catExcluded, isNew: true,
					reason: "was in parsing/ and no longer parses; reclassify or fix"}
				results = append(results, c)
				unclassified = append(unclassified, c)

			case cat == catExcluded && parses(b):
				// It was excluded and now parses. Promotion needs no judgement
				// and is applied automatically: excluded/ means "not a parseable
				// compilation unit", so a block that parses has outgrown it, and
				// the assertion it gains — keep parsing — is one this tool can
				// check on every run. Demotion is the direction that needs a
				// person, which is why only this way round is automatic.
				//
				// The usual cause is not a grammar change but a better-read
				// block: a filename recovered from the comment above the fence
				// changes how the file is classified, and with it what the body
				// is allowed to be.
				results = append(results, classified{block: b, category: catParsing})

			default:
				results = append(results, classified{block: b, category: cat, reason: reasons[key]})
			}
			continue
		}

		if parses(b) {
			results = append(results, classified{block: b, category: catParsing})
			continue
		}
		// A block that names several source files is showing a layout, not a
		// compilation unit, so its rejection is expected and needs no judgement.
		if len(b.files) > 1 {
			results = append(results, classified{block: b, category: catExcluded,
				reason: fmt.Sprintf("by-design\tblock shows %d source files (%s) and is not one compilation unit",
					len(b.files), strings.Join(b.files, ", "))})
			continue
		}
		// The operator bootstrap source has its own grammar root and its own
		// reader, so the ordinary parser cannot accept it by construction. It is
		// recognised by its fixed PATH: `library.fol` is the surface filename of
		// every srclib slot, and only the enclosing `operators/` directory says
		// that this one is the operator bootstrap.
		if isOperatorBootstrapPath(b.filename()) {
			results = append(results, classified{block: b, category: catExcluded,
				reason: "by-design\toperator-source file; parsed by the dedicated operator-source grammar, not the ordinary parser"})
			continue
		}
		// A block containing a "..." elision is prose: the reference is showing a
		// shape with its body omitted. It was never meant to compile.
		if hasElision(b.content) {
			results = append(results, classified{block: b, category: catExcluded,
				reason: "by-design\tblock elides code with \"...\" and is illustrative"})
			continue
		}
		if reason, byDesign := inferByDesignExclusion(b); byDesign {
			results = append(results, classified{block: b, category: catExcluded, reason: reason})
			continue
		}
		c := classified{block: b, category: catExcluded, isNew: true,
			reason: "new block that does not parse; classify by-design or gap"}
		results = append(results, c)
		unclassified = append(unclassified, c)
	}

	report(results, unclassified)

	if !*write {
		fmt.Println("\nreport only; pass -write to rewrite the corpora")
		return
	}
	if err := writeCorpora(results); err != nil {
		fail(err)
	}
	fmt.Println("\ncorpora rewritten")
}

var bareFunctionSnippet = regexp.MustCompile(`(?m)^\s*[A-Za-z][A-Za-z0-9_]*\s*\([^\r\n]*\)\s*->`)

// inferByDesignExclusion recognizes two recurring forms whose surrounding prose
// makes them examples rather than standalone compilation units. Keeping these
// rules in the generator makes regeneration reproducible instead of requiring a
// hand edit to its generated manifest after every reference renumbering.
func inferByDesignExclusion(b block) (string, bool) {
	result := parser.ParseFile(b.content, "refblocks", filepath.Join(corpusRoot, string(catParsing), b.directory()), b.filename(), "")
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.AsString(), "already declared in this scope") {
			return "by-design\tblock presents alternative declarations or constructions and intentionally reuses example names", true
		}
	}
	if len(b.files) == 0 && bareFunctionSnippet.MatchString(b.content) {
		return "by-design\tbare function is an Appendix B scope-model example shown outside its required unit container", true
	}
	return "", false
}

// extractBlocks returns every ```folang block with the line its fence opens on.
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
		// The trailing newline is normalized so that a block's identity does not
		// depend on whether the reference had a blank line before its closing
		// fence.
		content := strings.TrimRight(strings.Join(body, "\n"), "\n") + "\n"
		files := namedFiles(content)
		if len(files) == 0 {
			files = namedFiles(headingComment(lines, start-1))
		}
		blocks = append(blocks, block{
			line:    start,
			content: content,
			files:   files,
		})
	}
	return blocks, nil
}

// headingComment returns the run of "//" comment lines that sits directly above a
// block's opening fence, joined as if it were block content.
//
// The reference names most blocks' source file in a comment ABOVE the fence rather
// than inside it:
//
//	### Inner Function
//	//someInnerFun.unit.fol
//	```folang
//	_ co.lang.unit = { … }
//	```
//
// Reading only the block body loses that name, and losing it is not cosmetic: FoLang
// classifies a source file BY ITS NAME, so a unit body extracted as `L6711.fol` is
// parsed as a file-backed primary and rejected for holding `co.lang.unit`. Dozens of
// valid blocks sat in excluded/ for exactly that reason.
//
// One run of blank lines between the comment and the fence is tolerated, because the
// reference writes it both ways. The search stops at the first line that is neither
// blank nor a comment, so a heading or a paragraph never leaks a name into a block that
// has none.
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

// elisionPattern matches a "..." placeholder standing in for omitted code.
//
// The reference writes that where a body is beside the point — the shape of the
// declaration is the subject, not its contents — both on a line of its own and
// inline as `{ ... }`. Such a block cannot compile and was never intended to,
// which makes it a by-design exclusion rather than a parser gap.
//
// The ellipsis must be delimited by whitespace or a brace, so a range such as
// `0...9` and a variadic parameter are not mistaken for one.
var elisionPattern = regexp.MustCompile(`(^|[\s{])\.\.\.($|[\s}])`)

// hasElision reports whether the block elides code with "...".
func hasElision(content string) bool {
	return elisionPattern.MatchString(content)
}

// manifestKeys returns the spellings under which a corpus file may appear in the
// manifest, most specific first.
//
// A nested entry is keyed "L<line>/<file>" and a flat one just "<file>". Both are
// tried so that a manifest written under either layout still resolves.
func manifestKeys(path string) []string {
	base := filepath.Base(path)
	parent := filepath.Base(filepath.Dir(path))
	if strings.HasPrefix(parent, "L") {
		return []string{parent + "/" + base, base}
	}
	return []string{base}
}

// namedFiles returns the source filenames a block names in its comments, in
// order and without duplicates.
//
// One name identifies the block's own file. SEVERAL mean the block shows a
// layout across files — an owner beside its companion unit, a typeclass beside
// its instance — which is not a compilation unit and cannot parse as one. That
// is a by-design exclusion the extractor can recognise on its own rather than
// leaving to a person.
func namedFiles(content string) []string {
	var files []string
	seen := map[string]bool{}
	for _, match := range sourceFilePattern.FindAllStringSubmatch(content, -1) {
		name := filepath.Base(match[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		files = append(files, name)
	}
	return files
}

// loadExistingClassifications maps each existing corpus file's content to its
// bucket, so judgement survives renumbering.
func loadExistingClassifications() (map[string]category, map[string]string, error) {
	known := map[string]category{}
	reasons := map[string]string{}

	// Manifest reasons are keyed by filename, so read them before the files.
	manifestReasons, err := readManifest(filepath.Join(corpusRoot, string(catExcluded), "MANIFEST.tsv"))
	if err != nil {
		return nil, nil, err
	}

	for _, cat := range []category{catParsing, catInvalid, catExcluded} {
		// Both layouts are read: the flat one this tool replaces, and the
		// per-block folders it writes. Without the flat glob the first
		// re-extraction would see no prior classifications at all and would put
		// every rejected block back in front of a person.
		flat, err := filepath.Glob(filepath.Join(corpusRoot, string(cat), "*.fol"))
		if err != nil {
			return nil, nil, err
		}
		nested, err := filepath.Glob(filepath.Join(corpusRoot, string(cat), "L*", "*.fol"))
		if err != nil {
			return nil, nil, err
		}
		for _, path := range append(flat, nested...) {
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, nil, err
			}
			content := strings.ReplaceAll(string(raw), "\r\n", "\n")
			key := hashOf(content)
			known[key] = cat
			// The manifest keys a nested entry as "L<line>/<file>" and a flat one
			// as "<file>", so both spellings are tried. Looking up only the base
			// name loses every reason on the first regeneration after the layout
			// changed — silently, because an absent reason simply reads as
			// "unsorted".
			for _, id := range manifestKeys(path) {
				if reason, ok := manifestReasons[id]; ok {
					reasons[key] = reason
					break
				}
			}
		}
	}
	return known, reasons, nil
}

// readManifest reads the excluded corpus's per-file reasons.
func readManifest(path string) (map[string]string, error) {
	reasons := map[string]string{}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return reasons, nil
	}
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 4 {
			reasons[fields[0]] = fields[2] + "\t" + fields[3]
		}
	}
	return reasons, nil
}

// parses reports whether a block parses cleanly on its own.
//
// It uses the non-fatal entry point, so a block that cannot be parsed reports
// diagnostics instead of ending this process — which is exactly why extraction
// can be automated at all.
func parses(b block) bool {
	// The directory is the one the block will be written to, so that parsing here
	// and re-parsing the corpus entry later agree — the filename-uniqueness check
	// reads the folder, and a block sharing a folder with unrelated blocks would
	// report collisions the reference does not have.
	dir := filepath.Join(corpusRoot, string(catParsing), b.directory())
	result := parser.ParseFile(b.content, "refblocks", dir, b.filename(), "")
	return len(result.Diagnostics) == 0
}

func hashOf(content string) string {
	sum := sha256.Sum256([]byte(strings.ReplaceAll(content, "\r\n", "\n")))
	return hex.EncodeToString(sum[:])
}

func report(results []classified, unclassified []classified) {
	counts := map[category]int{}
	for _, r := range results {
		counts[r.category]++
	}
	fmt.Printf("blocks extracted: %d\n", len(results))
	for _, cat := range []category{catParsing, catInvalid, catExcluded} {
		fmt.Printf("  %-9s %d\n", cat, counts[cat])
	}

	if len(unclassified) == 0 {
		fmt.Println("\nevery block carried a known classification")
		return
	}
	fmt.Printf("\n%d block(s) need a person to classify:\n", len(unclassified))
	for _, c := range unclassified {
		fmt.Printf("  %s/%s (language-ref.md line %d)\n      %s\n", c.directory(), c.filename(), c.line, c.reason)
	}
}

// writeCorpora replaces the three directories with the freshly extracted files.
func writeCorpora(results []classified) error {
	for _, cat := range []category{catParsing, catInvalid, catExcluded} {
		dir := filepath.Join(corpusRoot, string(cat))
		// Both the old flat *.fol entries and the current per-block folders are
		// cleared, so a re-extraction leaves no stale file behind under either
		// layout.
		entries, err := filepath.Glob(filepath.Join(dir, "L*"))
		if err != nil {
			return err
		}
		for _, path := range entries {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	for _, r := range results {
		dir := filepath.Join(corpusRoot, string(r.category), r.directory())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, r.filename()), []byte(r.content), 0o644); err != nil {
			return err
		}
	}
	return writeManifest(results)
}

// writeManifest records the reason each excluded block is excluded.
func writeManifest(results []classified) error {
	var b strings.Builder
	b.WriteString("# Excluded reference blocks from docs/language-ref.md.\n#\n")
	b.WriteString("# Regenerate with: go run ./cmd/refblocks -write\n#\n")
	b.WriteString("# Columns: file, first line of the block in language-ref.md, category, reason.\n#\n")
	b.WriteString("#   by-design  not a parseable compilation unit as written\n")
	b.WriteString("#   ref-bug    valid-looking, but the reference violates a current rule;\n")
	b.WriteString("#              see docs/UNSORTED-TRIAGE.md for the correction\n")
	b.WriteString("#   gap        should parse but is rejected; see docs/REFBLOCK-GAPS.md\n")
	b.WriteString("#   unsorted   newly extracted and not yet classified\n#\n")
	b.WriteString("# file\tref_line\tcategory\treason\n")

	excluded := make([]classified, 0)
	for _, r := range results {
		if r.category == catExcluded {
			excluded = append(excluded, r)
		}
	}
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].line < excluded[j].line })

	for _, r := range excluded {
		category, reason := "unsorted", r.reason
		if parts := strings.SplitN(r.reason, "\t", 2); len(parts) == 2 {
			category, reason = parts[0], parts[1]
		}
		fmt.Fprintf(&b, "%s/%s\t%d\t%s\t%s\n", r.directory(), r.filename(), r.line, category, reason)
	}
	return os.WriteFile(filepath.Join(corpusRoot, string(catExcluded), "MANIFEST.tsv"), []byte(b.String()), 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
