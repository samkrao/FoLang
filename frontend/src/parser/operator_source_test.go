package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

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
	source := `@co.dap.library(type=operator)
_ co.lang.library = {
    <+> co.lang.operator = {
        fixity=infix,
        fixity=infix,
        fixity=infix,
        precedence=60,
        associativity=left,
        arity=binary
    }
}`
	_, findings := parseOperatorSource(source, operatorSourceFileName)
	diagnostic := joinFindings(findings)
	if count := strings.Count(diagnostic, "first occurrence at line 4"); count != 2 {
		t.Fatalf("duplicate-property diagnostics should both retain the original location; count=%d\n%s", count, diagnostic)
	}
}

func TestProjectOperatorBootstrapLoadsOnlyFixedConfiguredSource(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, projectConfigFile), "fol-lang:\n  operator_library_folder: operators # source-only\n")
	operatorArea := filepath.Join(root, "operators")
	if err := os.MkdirAll(operatorArea, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join("..", "..", "tests", "parser", "examples", "operator-source", "accepted", "canonical-metadata.fol")
	writeTestFile(t, filepath.Join(operatorArea, operatorSourceFileName), readOperatorFixture(t, fixture))
	writeTestFile(t, filepath.Join(operatorArea, "ignored.fol"), "this is not parsed")

	bootstrap := loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 {
		t.Fatalf("bootstrap findings:\n%s", joinFindings(bootstrap.Findings))
	}
	if len(bootstrap.Declarations) != 2 {
		t.Fatalf("declarations = %d, want 2", len(bootstrap.Declarations))
	}
	if !pathWithin(filepath.Join(operatorArea, "nested", "anything.fol"), bootstrap.Area) {
		t.Fatal("configured descendants were not classified inside the excluded operator area")
	}
	if pathWithin(filepath.Join(root, "operators-old", "ordinary.fol"), bootstrap.Area) {
		t.Fatal("path prefix outside configured area was incorrectly excluded")
	}
}

func TestOperatorBootstrapConfigPreservesLiteralHashFolder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, projectConfigFile), "operator_library_folder: operators#cache # source-only\n")
	operatorArea := filepath.Join(root, "operators#cache")
	if err := os.MkdirAll(operatorArea, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(operatorArea, operatorSourceFileName), `@co.dap.library(type=operator)
_ co.lang.library = {
    <+> co.lang.operator = { fixity=infix, precedence=60, associativity=left, arity=binary }
}`)

	bootstrap := loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 {
		t.Fatalf("bootstrap findings:\n%s", joinFindings(bootstrap.Findings))
	}
	if len(bootstrap.Declarations) != 1 || bootstrap.Area != operatorArea {
		t.Fatalf("bootstrap area/declarations = %q/%d, want %q/1", bootstrap.Area, len(bootstrap.Declarations), operatorArea)
	}
}

func TestStripYAMLCommentRespectsQuotedHashesAndEscapedQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`operator_library_folder: "operators#cache" # note`, `operator_library_folder: "operators#cache" `},
		{`operator_library_folder: "operators\"#cache" # note`, `operator_library_folder: "operators\"#cache" `},
		{`operator_library_folder: 'operators''#cache' # note`, `operator_library_folder: 'operators''#cache' `},
	}
	for _, test := range tests {
		if got := stripYAMLComment(test.input); got != test.want {
			t.Errorf("stripYAMLComment(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestOperatorBootstrapMissingFolderAndFileMeanNoLocalOperators(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, projectConfigFile), "operator_library_folder: missing\n")
	bootstrap := loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 || len(bootstrap.Declarations) != 0 {
		t.Fatalf("missing folder result = declarations:%d findings:%d", len(bootstrap.Declarations), len(bootstrap.Findings))
	}
	if bootstrap.Area == "" {
		t.Fatal("configured missing area was not retained for discovery exclusion")
	}

	if err := os.MkdirAll(bootstrap.Area, 0o755); err != nil {
		t.Fatal(err)
	}
	bootstrap = loadProjectOperatorBootstrap(root)
	if len(bootstrap.Findings) != 0 || len(bootstrap.Declarations) != 0 {
		t.Fatalf("missing fixed file result = declarations:%d findings:%d", len(bootstrap.Declarations), len(bootstrap.Findings))
	}
}

func TestOperatorBootstrapRejectsUnsafeOrAmbiguousConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"project root", "operator_library_folder: .\n"},
		{"parent traversal", "operator_library_folder: ../operators\n"},
		{"duplicate key", "operator_library_folder: operators\noperator_library_folder: other\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, projectConfigFile), test.config)
			bootstrap := loadProjectOperatorBootstrap(root)
			if len(bootstrap.Findings) == 0 {
				t.Fatalf("configuration %q was accepted", test.config)
			}
		})
	}
}

func TestTokenOnlyDriverUsesBootstrappedCustomOperators(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, projectConfigFile), "operator_library_folder: operators\n")
	area := filepath.Join(root, "operators")
	if err := os.MkdirAll(area, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(area, operatorSourceFileName), `@co.dap.library(type=operator)
_ co.lang.library = {
    <+> co.lang.operator = { fixity=infix, precedence=60, associativity=left, arity=binary }
}`)
	mainFile := filepath.Join(root, "main.fol")
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
		t.Fatalf("token-only stream did not classify configured operator: %s", encoded)
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
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, projectConfigFile), "operator_library_folder: operators\n")

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
	projectRoot := t.TempDir()
	writeTestFile(t, filepath.Join(projectRoot, projectConfigFile), "operator_library_folder: operators\n")

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
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, projectConfigFile), "operator_library_folder: operators\n")
	area := filepath.Join(root, "operators")
	if err := os.MkdirAll(filepath.Join(area, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	operatorFile := filepath.Join(area, operatorSourceFileName)
	writeTestFile(t, operatorFile, `@co.dap.library(type=operator)
_ co.lang.library = {
    <+> co.lang.operator = { fixity=infix, precedence=60, associativity=left, arity=binary }
}`)
	nestedFile := filepath.Join(area, "nested", "ignored.fol")
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

	accepted := `Vector co.lang.unit = {
    @co.dap.operator(symbol="<+>", mode=overload)
    merge(left Vector, right Vector)->(Vector) = { this.return left; }
}`
	if findings := parseWithOperatorCatalog(accepted, declaration); len(findings) != 0 {
		t.Fatalf("valid implementation findings:\n%s", joinParserFindings(findings))
	}

	rejected := `Vector co.lang.unit = {
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
		rejected := `Vector co.lang.unit = {
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
		accepted := `Vector co.lang.unit = {
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
	return `Vector co.lang.unit = {
    use(a Vector, b Vector, c Vector)->(Vector) = {
        result := ` + expression + `;
        this.return result;
    }
}`
}

func parseWithOperatorCatalog(source string, declarations ...operatorDeclaration) []string {
	collection := declaredOperatorsIn(source, "Vector.unit.fol", declarations)
	raw := scanlex.TokenizeWith(source, "Vector.unit.fol", collection.Custom)
	p, _ := newParser(normalizeTokens(raw))
	p.preRegisterOperatorDeclarations(collection.Declarations)
	p.file = fileinfo{Basename: "Vector.unit.fol", PackagePath: "vectors", LocationKnown: true}
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

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
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
