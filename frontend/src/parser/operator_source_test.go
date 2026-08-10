package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/project"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestOperatorSourceFullFileFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "tests", "parser", "examples", "operator-source")

	accepted, err := filepath.Glob(filepath.Join(root, "accepted", "*.fol"))
	if err != nil || len(accepted) == 0 {
		t.Fatalf("discover accepted operator-source fixtures: paths=%v err=%v", accepted, err)
	}
	sort.Strings(accepted)
	for _, path := range accepted {
		path := path
		t.Run("accepted/"+filepath.Base(path), func(t *testing.T) {
			source := readOperatorFixture(t, path)
			declarations, findings := parseOperatorSource(source, filepath.Base(path))
			if len(findings) != 0 {
				t.Fatalf("accepted source produced findings:\n%s", joinFindings(findings))
			}
			if len(declarations) != 2 {
				t.Fatalf("declarations = %d, want 2", len(declarations))
			}
			if got := declarations[0].Options["desugar"]; got != "intrinsic:combine" {
				t.Fatalf("desugar metadata = %#v", got)
			}
		})
	}

	rejected, err := filepath.Glob(filepath.Join(root, "rejected", "*.fol"))
	if err != nil || len(rejected) == 0 {
		t.Fatalf("discover rejected operator-source fixtures: paths=%v err=%v", rejected, err)
	}
	sort.Strings(rejected)
	for _, path := range rejected {
		path := path
		t.Run("rejected/"+filepath.Base(path), func(t *testing.T) {
			declarations, findings := parseOperatorSource(readOperatorFixture(t, path), filepath.Base(path))
			if len(findings) == 0 {
				t.Fatal("rejected source produced no diagnostic")
			}
			if len(declarations) != 0 {
				t.Fatalf("rejected source leaked %d declarations into its atomic catalog", len(declarations))
			}
		})
	}
}

func TestOperatorSourceDuplicatePropertyKeepsFirstLocation(t *testing.T) {
	source := `_ co.lang.library = {
    <+> co.lang.operator = {
        fixity: co.operator.fixity.infix,
        fixity: co.operator.fixity.infix,
        fixity: co.operator.fixity.infix,
        precedence: 60,
        associativity: co.operator.associativity.left,
        arity: co.operator.arity.binary
    };
}`
	_, findings := parseOperatorSource(source, project.LibrarySurfaceFilename)
	diagnostic := joinFindings(findings)
	if count := strings.Count(diagnostic, "first occurrence at line 3"); count != 2 {
		t.Fatalf("duplicate-property diagnostics should both retain the original location; count=%d\n%s", count, diagnostic)
	}
}

// writeOperatorProject lays out a minimal but VALID project around a fixed operator
// bootstrap surface, and returns the project root.
//
// The layout is the real one: src/appl.fol is what makes the tree an application, and
// srclib/operators/library.fol is the one place a project-local operator may be
// declared. A fixture that skipped either would be testing a tree the compiler rejects.
func writeOperatorProject(t *testing.T, operatorSource string) string {
	t.Helper()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, project.MarkerFilename), "fol-lang:\n  name: fixture\n")
	writeTestFile(t, filepath.Join(root, project.SourceDomain, project.ApplicationEntryFilename), "value := 1;\n")
	if operatorSource != "" {
		writeTestFile(t, operatorBootstrapPath(root), operatorSource)
	}
	return root
}

// operatorBootstrapPath is the fixed location of a project's operator bootstrap surface.
func operatorBootstrapPath(root string) string {
	return filepath.Join(root, project.SourceLibraryDomain, project.OperatorsLibrarySlot, project.LibrarySurfaceFilename)
}

// canonicalOperatorSource is one well-formed declaration in the current grammar: no kind
// annotation, ":" property binders, and no trailing comma.
const canonicalOperatorSource = `_ co.lang.library = {
    <+> co.lang.operator = { fixity: co.operator.fixity.infix, precedence: 60, associativity: co.operator.associativity.left, arity: co.operator.arity.binary };
}`

func TestProjectOperatorBootstrapLoadsOnlyTheFixedSurface(t *testing.T) {
	fixture := filepath.Join("..", "..", "tests", "parser", "examples", "operator-source", "accepted", "canonical-metadata.fol")
	root := writeOperatorProject(t, readOperatorFixture(t, fixture))
	area := filepath.Dir(operatorBootstrapPath(root))

	bootstrap := loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 {
		t.Fatalf("bootstrap findings:\n%s", joinFindings(bootstrap.Findings))
	}
	if len(bootstrap.Declarations) != 2 {
		t.Fatalf("declarations = %d, want 2", len(bootstrap.Declarations))
	}
	if bootstrap.Area != area {
		t.Fatalf("bootstrap area = %q, want the fixed %q", bootstrap.Area, area)
	}
	if !pathWithin(filepath.Join(area, "nested", "anything.fol"), bootstrap.Area) {
		t.Fatal("descendants of the operator slot were not classified inside the excluded area")
	}
	if pathWithin(filepath.Join(root, project.SourceLibraryDomain, "operators-old", "ordinary.fol"), bootstrap.Area) {
		t.Fatal("a path merely sharing the slot's prefix was incorrectly excluded")
	}
}

func TestOperatorBootstrapMissingSlotAndFileMeanNoLocalOperators(t *testing.T) {
	root := writeOperatorProject(t, "")
	bootstrap := loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 || len(bootstrap.Declarations) != 0 {
		t.Fatalf("missing slot result = declarations:%d findings:%d", len(bootstrap.Declarations), len(bootstrap.Findings))
	}
	// The area is retained even when absent, because discovery must exclude it either
	// way rather than treating srclib/operators/ as an ordinary package once created.
	if bootstrap.Area == "" {
		t.Fatal("the fixed operator area was not retained for discovery exclusion")
	}

	if err := os.MkdirAll(bootstrap.Area, 0o755); err != nil {
		t.Fatal(err)
	}
	bootstrap = loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 || len(bootstrap.Declarations) != 0 {
		t.Fatalf("missing fixed file result = declarations:%d findings:%d", len(bootstrap.Declarations), len(bootstrap.Findings))
	}
}

func TestTokenOnlyDriverUsesBootstrappedCustomOperators(t *testing.T) {
	root := writeOperatorProject(t, canonicalOperatorSource)
	// The entry file is where a bare statement belongs; every other name under src/ is
	// a package source file and holds a declaration instead.
	mainFile := filepath.Join(root, project.SourceDomain, project.ApplicationEntryFilename)
	writeTestFile(t, mainFile, "result := left <+> right;\n")

	_, _, encoded, _, err := Focmain(mainFile, false, false, "Tokens", false, root)
	if err != nil {
		t.Fatalf("token-only driver: %v", err)
	}
	var tokens []scanlex.Token
	if err := json.Unmarshal([]byte(encoded), &tokens); err != nil {
		t.Fatalf("decode token stream: %v", err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Value == "<+>" && tok.Kind == scanlex.CUSTOM_OPERATOR {
			found = true
		}
	}
	if !found {
		t.Fatalf("token-only stream did not classify the bootstrapped operator: %s", encoded)
	}

	_, _, serialized, _, err := Focmain(mainFile, false, false, "", false, root)
	if err != nil {
		t.Fatalf("full bootstrap parse: %v", err)
	}
	if strings.TrimSpace(serialized) == "" {
		t.Fatal("full bootstrap parse returned an empty serialized AST")
	}
}

func TestTokenOnlyDriverRejectsExplicitRootOutsideTarget(t *testing.T) {
	projectRoot := writeOperatorProject(t, canonicalOperatorSource)

	outside := t.TempDir()
	target := filepath.Join(outside, "main.fol")
	writeTestFile(t, target, "value := left <+> right;\n")

	_, _, encoded, _, err := Focmain(target, false, false, "Tokens", false, projectRoot)
	if err == nil {
		t.Fatal("token-only driver silently accepted a target outside its explicit project root")
	}
	if encoded != "" {
		t.Fatalf("token-only driver returned tokens after failed discovery: %s", encoded)
	}
	if !strings.Contains(err.Error(), "project root") {
		t.Fatalf("discovery error = %q, want project-root context", err)
	}
}

func TestFullDriverRejectsExplicitRootOutsideTarget(t *testing.T) {
	projectRoot := writeOperatorProject(t, canonicalOperatorSource)

	outside := t.TempDir()
	target := filepath.Join(outside, "main.fol")
	writeTestFile(t, target, "value := 1;\n")

	_, _, serialized, _, err := Focmain(target, false, false, "", false, projectRoot)
	if err == nil {
		t.Fatal("full driver silently accepted a target outside its explicit project root")
	}
	if serialized != "" {
		t.Fatalf("full driver returned an AST after failed discovery: %s", serialized)
	}
	if !strings.Contains(err.Error(), "project root") {
		t.Fatalf("discovery error = %q, want project-root context", err)
	}
}

func TestDriverRejectsDirectOperatorAreaTargets(t *testing.T) {
	root := writeOperatorProject(t, canonicalOperatorSource)
	operatorFile := operatorBootstrapPath(root)
	nestedFile := filepath.Join(filepath.Dir(operatorFile), "nested", "ignored.fol")
	writeTestFile(t, nestedFile, "value := 1;\n")

	for _, target := range []string{operatorFile, nestedFile} {
		for _, stopAt := range []string{"Tokens", ""} {
			name := filepath.Base(target) + "/full"
			if stopAt == "Tokens" {
				name = filepath.Base(target) + "/tokens"
			}
			t.Run(name, func(t *testing.T) {
				_, _, artifact, _, err := Focmain(target, false, false, stopAt, false, root)
				if err == nil || !strings.Contains(err.Error(), "source-only bootstrap files") {
					t.Fatalf("direct operator-area target error = %v, want source-only diagnostic", err)
				}
				if artifact != "" {
					t.Fatalf("direct operator-area target returned an artifact: %s", artifact)
				}
			})
		}
	}
}

func TestRegisteredCustomOperatorImplementationArity(t *testing.T) {
	declaration := operatorDeclaration{Options: map[string]any{
		"symbol": "<+>", "fixity": "infix", "precedence": int64(60),
		"associativity": "left", "arity": "binary",
	}}

	accepted := `_ co.lang.unit = {
    @co.dap.operator(symbol="<+>", mode=overload)
    merge(left Vector, right Vector)->(Vector) = { this.return left; }
}`
	if findings := parseWithOperatorCatalog(accepted, declaration); len(findings) != 0 {
		t.Fatalf("valid implementation findings:\n%s", joinParserFindings(findings))
	}

	rejected := `_ co.lang.unit = {
    @co.dap.operator(symbol="<+>", mode=overload)
    merge(left Vector)->(Vector) = { this.return left; }
}`
	findings := parseWithOperatorCatalog(rejected, declaration)
	if len(findings) == 0 || !strings.Contains(joinParserFindings(findings), "registered callable arity") {
		t.Fatalf("wrong-arity implementation findings:\n%s", joinParserFindings(findings))
	}
}

func TestCustomNonAssociativeOperatorRejectsUnparenthesizedChain(t *testing.T) {
	declaration := operatorDeclaration{Options: map[string]any{
		"symbol": "<+>", "fixity": "infix", "precedence": int64(60),
		"associativity": "none", "arity": "binary",
	}}

	for _, expression := range []string{
		"a <+> b <+> c",
		"a <+> b + c",
		"a + b <+> c",
	} {
		rejected := `_ co.lang.unit = {
    use(a Vector, b Vector, c Vector)->(Vector) = {
        result := ` + expression + `;
        this.return result;
    }
}`
		findings := parseWithOperatorCatalog(rejected, declaration)
		if len(findings) == 0 || !strings.Contains(joinParserFindings(findings), "non-associative") {
			t.Fatalf("non-associative chain %q findings:\n%s", expression, joinParserFindings(findings))
		}
	}

	for _, expression := range []string{"(a <+> b) + c", "a + (b <+> c)"} {
		accepted := `_ co.lang.unit = {
    use(a Vector, b Vector, c Vector)->(Vector) = {
        result := ` + expression + `;
        this.return result;
    }
}`
		if findings := parseWithOperatorCatalog(accepted, declaration); len(findings) != 0 {
			t.Fatalf("parenthesized non-associative expression %q findings:\n%s", expression, joinParserFindings(findings))
		}
	}
}

func TestCustomNonAssociativeOperatorCrossesRightAssociativeRecursion(t *testing.T) {
	tests := []struct {
		name       string
		precedence int64
		rejected   string
		accepted   string
	}{
		{"assignment", 10, "a = b <+> c", "a = (b <+> c)"},
		{"power", 90, "a ** b <+> c", "a ** (b <+> c)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declaration := testOperatorDeclaration("<+>", "infix", test.precedence, "none", "binary")
			findings := parseWithOperatorCatalog(operatorUseUnit(test.rejected), declaration)
			if len(findings) == 0 || !strings.Contains(joinParserFindings(findings), "non-associative") {
				t.Fatalf("right-recursive non-associative expression %q findings:\n%s", test.rejected, joinParserFindings(findings))
			}
			if findings := parseWithOperatorCatalog(operatorUseUnit(test.accepted), declaration); len(findings) != 0 {
				t.Fatalf("parenthesized right-recursive expression %q findings:\n%s", test.accepted, joinParserFindings(findings))
			}
		})
	}
}

func TestOpenLowerRangeParticipatesInSymmetricNonAssociativity(t *testing.T) {
	declaration := testOperatorDeclaration("<+>", "infix", 55, "left", "binary")
	rejected := ".. a <+> b"
	findings := parseWithOperatorCatalog(operatorUseUnit(rejected), declaration)
	if len(findings) == 0 || !strings.Contains(joinParserFindings(findings), "non-associative") {
		t.Fatalf("open-lower range chain %q findings:\n%s", rejected, joinParserFindings(findings))
	}

	accepted := "(.. a) <+> b"
	if findings := parseWithOperatorCatalog(operatorUseUnit(accepted), declaration); len(findings) != 0 {
		t.Fatalf("grouped open-lower range %q findings:\n%s", accepted, joinParserFindings(findings))
	}
}

func operatorUseUnit(expression string) string {
	return `_ co.lang.unit = {
    use(a Vector, b Vector, c Vector)->(Vector) = {
        result := ` + expression + `;
        this.return result;
    }
}`
}

// parseWithOperatorCatalog parses source as the companion unit of Vector, which
// is where DECISION-COMP-001 puts a struct's operator implementations.
func parseWithOperatorCatalog(source string, declarations ...operatorDeclaration) []string {
	return parseFileWithOperatorCatalog(source, "Vector.comp.unit.fol", declarations...)
}

// parseFileWithOperatorCatalog parses source under an explicit source filename,
// which selects the package-source-file root and supplies every filename-derived
// name.
func parseFileWithOperatorCatalog(source, basename string, declarations ...operatorDeclaration) []string {
	collection := declaredOperatorsIn(source, basename, declarations)
	raw := scanlex.TokenizeWith(source, basename, collection.Custom)
	p, _ := newParser(normalizeTokens(raw))
	p.preRegisterOperatorDeclarations(collection.Declarations)
	p.file = fileinfo{
		Basename:      basename,
		PackagePath:   "vectors",
		LocationKnown: true,
		Source:        classifySourceFilename(basename),
	}
	p.parseTopLevel()
	out := make([]string, len(p.diags))
	for index, finding := range p.diags {
		out[index] = finding.Error()
	}
	return out
}

func readOperatorFixture(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// writeTestFile writes one fixture file, creating the domain directories above it. The
// standardized layout is several levels deep, so a fixture that had to mkdir each level
// itself would bury what it is actually testing.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func joinFindings(findings []error) string {
	parts := make([]string, len(findings))
	for index, finding := range findings {
		parts[index] = finding.Error()
	}
	return strings.Join(parts, "\n")
}

func joinParserFindings(findings []string) string { return strings.Join(findings, "\n") }
