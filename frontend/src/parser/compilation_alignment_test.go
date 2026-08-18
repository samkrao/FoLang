package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// The compilation-unit form is selected by the FILENAME, not by the body and not by
// where in the tree the file sits. The two reserved names decide outright, and every
// other `.fol` file is a package source file.
func TestCompilationUnitClassificationFollowsTheReservedFilenames(t *testing.T) {

	tests := []struct {
		name     string
		source   string
		basename string
		atRoot   bool
		want     unitKind
	}{
		// appl.fol is the one entry file, and it is the only way to get one.
		{"appl.fol is the entry file", `value := 1;`, "appl.fol", true, unitEntry},
		// package.fol and library.fol have no structural meaning in the current
		// model; both are ordinary identifier-derived primary filenames.
		{"library.fol is an ordinary package file", `_ co.lang.struct = {}`, "library.fol", true, unitPackage},
		{"nested library.fol is an ordinary package file", `_ co.lang.struct = {}`, "library.fol", false, unitPackage},
		// An ordinary name is a package source file wherever it sits. A struct at
		// the top of src/ used to be read as an entry file; it is a file-backed
		// primary, and only appl.fol is an entry.
		{"ordinary name at the domain root is a package", `_ co.lang.struct = { id co.lang.int; }`, "Employee.fol", true, unitPackage},
		{"ordinary name below the domain root is a package", `_ co.lang.struct = { id co.lang.int; }`, "Employee.fol", false, unitPackage},
		// A library declaration under an ordinary name does not make a surface: the
		// name is what selects the root, so this is a package file whose body the
		// primary parser then rejects.
		{"library body under an ordinary name is not a surface", `_ co.lang.library = {}`, "Api.fol", true, unitPackage},
		{"unit filename is a package source file", `_ co.lang.unit = {}`, "arithmetic.unit.fol", true, unitPackage},
		{"companion filename is a package source file", `_ co.lang.unit = {}`, "Employee.comp.unit.fol", true, unitPackage},
		{"package.fol is an ordinary package file", `_ co.lang.struct = {}`, "package.fol", true, unitPackage},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toks := normalizeTokens(scanlex.Tokenize(test.source, test.basename))
			p, _ := newParser(toks)
			p.file = fileinfo{
				Basename:      test.basename,
				LocationKnown: true,
				AtRoot:        test.atRoot,
				Source:        classifySourceFilename(test.basename),
			}
			if got := p.classifyCompilationUnit(); got != test.want {
				t.Fatalf("classification = %v, want %v", got, test.want)
			}
		})
	}
}

// An entry file admits parameterized co.lang.type constructors. It could already USE a
// polymorphic type, and `Option(T) co.lang.type = …` is a type declaration like any
// other in that family, so refusing only its parameter clause drew a line the reference
// does not draw.
func TestEntryFileAdmitsParameterizedTypeConstructor(t *testing.T) {
	root, p := parseEntrySource(t, `Option(T) co.lang.type = Some(T) | None(); value Option(co.lang.int);`)

	if _, ok := root.(ast.Application); !ok {
		t.Fatalf("root = %T, want ast.Application", root)
	}
	if len(p.diags) != 0 {
		t.Fatalf("parameterized entry type constructor produced diagnostics: %v", p.diags)
	}
}

func TestEntryFileRejectsParameterizedNonTypeKind(t *testing.T) {
	_, p := parseEntrySource(t, `Alias(F(_)) co.lang.newtype = co.lang.int;`)

	if len(p.diags) != 1 {
		t.Fatalf("diagnostics = %d, want exactly one parameterized-kind diagnostic", len(p.diags))
	}
	if got := p.diags[0].Error(); !strings.Contains(got, "only a co.lang.type declaration may be parameterized") {
		t.Fatalf("diagnostic = %q, want the parameterized-kind restriction", got)
	}
}

func TestEntryFileDeclarationStillAllowsForallTypeAlias(t *testing.T) {
	root, p := parseEntrySource(t, `PolyId co.lang.type = forall(T).(T)->(T); value PolyId;`)

	if _, ok := root.(ast.Application); !ok {
		t.Fatalf("root = %T, want ast.Application", root)
	}
	if len(p.diags) != 0 {
		t.Fatalf("forall entry-file alias produced diagnostics: %v", p.diags)
	}
}

// parseEntrySource parses source as src/appl.fol, the one application entry file.
func parseEntrySource(t *testing.T, source string) (ast.Stmt, *parser) {
	t.Helper()

	toks := normalizeTokens(scanlex.Tokenize(source, "appl.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{
		Basename:      "appl.fol",
		LocationKnown: true,
		AtRoot:        true,
		Source:        classifySourceFilename("appl.fol"),
	}
	return p.parseCompilationUnit(), p
}

func TestComponentSurfaceAndComponentImportUseCurrentGrammar(t *testing.T) {
	toks := normalizeTokens(scanlex.Tokenize(`_ co.lang.component = {
    @co.ddap.import(component="native", as="native")
    ping()->() = {}
}`, "component.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{
		Basename:      "component.fol",
		Basedir:       "components/application",
		LocationKnown: true,
		Source:        classifySourceFilename("component.fol"),
	}

	root := p.parseCompilationUnit()
	component, ok := root.(ast.ComponentDeclarationStmt)
	if !ok {
		t.Fatalf("root = %T, want ast.ComponentDeclarationStmt", root)
	}
	if len(p.diags) != 0 {
		t.Fatalf("component surface produced diagnostics: %v", p.diags)
	}
	if len(component.Body) != 2 {
		t.Fatalf("component members = %d, want 2", len(component.Body))
	}
	imported, ok := component.Body[0].(ast.ImportStmt)
	if !ok || imported.Component != "native" || imported.Name != "native" {
		t.Fatalf("component import = %#v", component.Body[0])
	}
}

func TestKnownImportPreservesUnhandledFields(t *testing.T) {
	root, p := parseEntrySource(t, `@co.ddap.import(package="hr", future={mode=true})
value := 1;`)
	if len(p.diags) != 0 {
		t.Fatalf("known import with future field produced diagnostics: %v", p.diags)
	}
	application := root.(ast.Application)
	imported := application.Body[0].(ast.ImportStmt)
	future, ok := imported.ExtraFields["future"].(map[string]any)
	if !ok || future["mode"] != "true" {
		t.Fatalf("preserved future field = %#v", imported.ExtraFields["future"])
	}
}

func TestSpecializedFunctionNodesPreserveCompleteMetadata(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
	}{
		{"macro", `@co.dap.macro(future={tag=true})`},
		{"template", `@co.dap.template(future={tag=true})`},
		{"decorator", `@co.dap.decorator(future={tag=true})`},
		{"native", `@co.dap.native(future={tag=true})`},
		{"execution model", `@co.dap.executionmodel(type=concurrent, future={tag=true})`},
		{"extension", `@co.dap.extension(fortype=co.lang.string, future={tag=true})`},
		{"generic", `@co.dap.generic(future={tag=true})`},
		{"indexer", `@co.dap.indexer(symbol="[]", future={tag=true})`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := "_ co.lang.unit = {\n" + test.annotation + "\nf()->() = {}\n}"
			toks := normalizeTokens(scanlex.Tokenize(source, "metadata.unit.fol"))
			p, _ := newParser(toks)
			p.file = fileinfo{Basename: "metadata.unit.fol", LocationKnown: true,
				Source: classifySourceFilename("metadata.unit.fol")}

			root := p.parseCompilationUnit().(ast.PackageStmt)
			unit := root.Body[0].(ast.TypeDeclarationStmt)
			fn := embeddedFunctionDeclaration(t, unit.Body[0])
			list := fn.Dapst.(*ast.DirectveList)
			var preserved bool
			for _, statements := range list.Dapst {
				for _, statement := range statements {
					directive := statement.(ast.DirectiveStmt)
					if _, ok := directive.Parameters["future"]; ok {
						preserved = true
					}
				}
			}
			if !preserved {
				t.Fatalf("%T discarded the unhandled metadata field", unit.Body[0])
			}
		})
	}
}

func embeddedFunctionDeclaration(t *testing.T, statement ast.Stmt) ast.FunctionDeclarationStmt {
	t.Helper()
	switch node := statement.(type) {
	case ast.MacroStmt:
		return node.FunctionDeclarationStmt
	case ast.TemplateStmt:
		return node.FunctionDeclarationStmt
	case ast.DecoratorStmt:
		return node.FunctionDeclarationStmt
	case ast.NativeFunctionStmt:
		return node.FunctionDeclarationStmt
	case ast.ExecutionModelFunctionStmt:
		return node.FunctionDeclarationStmt
	case ast.ExtensionStmt:
		return node.FunctionDeclarationStmt
	case ast.GenerricFun:
		return node.FunctionDeclarationStmt
	case ast.IndexerStmt:
		return node.FunctionDeclarationStmt
	default:
		t.Fatalf("specialized declaration = %T", statement)
		return ast.FunctionDeclarationStmt{}
	}
}

func TestOperatorNodePreservesUnhandledMetadataFields(t *testing.T) {
	source := `_ co.lang.class = {
    @co.dap.operator(symbol='+', mode=overload, future={tag=true})
    (value Staff) add(other Staff)->(Staff) = { this.return value; }
}`
	toks := normalizeTokens(scanlex.Tokenize(source, "Staff.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{Basename: "Staff.fol", LocationKnown: true,
		Source: classifySourceFilename("Staff.fol")}

	root := p.parseCompilationUnit().(ast.PackageStmt)
	class := root.Body[0].(ast.ClassDeclarationStmt)
	operator := class.Body[0].(ast.OperatorStmt)
	list := operator.Dapst.(*ast.DirectveList)
	directive := list.Dapst[scanlex.ANNOTATION][0].(ast.DirectiveStmt)
	if _, ok := directive.Parameters["future"]; !ok {
		t.Fatal("operator node discarded the unhandled metadata field")
	}
}

// TestIndexerDeclarationIsItsOwnDeclarationKind covers the classification-table
// row `@co.dap.indexer -> IndexerDecl`. The reference's own indexer pair names
// its declarations `get` and `set`, so nothing but the `symbol=` field
// distinguishes the read accessor from the write accessor — which is why the
// node lifts it rather than leaving it in Dapst
// (docs/language-ref.md, "Indexer").
func TestIndexerDeclarationIsItsOwnDeclarationKind(t *testing.T) {
	source := `_ co.lang.unit = {
    @co.dap.indexer(symbol="[]")
    (g MyList) get(index co.lang.int)->(co.lang.int) = { this.return g.eles[index]; }

    @co.dap.indexer(symbol="[]=")
    (g MyList) set(index co.lang.int, value co.lang.int)->() = { g.eles[index] = value; }
}`
	root, p := parsePackageSource(t, source, "MyList.comp.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("indexer companion unit produced diagnostics: %v", p.diags)
	}

	unit := root.(ast.PackageStmt).Body[0].(ast.TypeDeclarationStmt)
	for index, want := range []string{"[]", "[]="} {
		indexer, ok := unit.Body[index].(ast.IndexerStmt)
		if !ok {
			t.Fatalf("member %d = %T, want ast.IndexerStmt", index, unit.Body[index])
		}
		if indexer.Symbol != want {
			t.Fatalf("indexer %d symbol = %q, want %q", index, indexer.Symbol, want)
		}
		// The specialization does not remove the callable interface: the
		// receiver and signature stay reachable through the embedded declaration.
		if indexer.AssociatedReceiver == nil {
			t.Fatalf("indexer %d lost its receiver clause", index)
		}
	}
}

func TestFunctionShapeClassifiersAreMutuallyExclusive(t *testing.T) {
	classifiers := []struct {
		name       string
		annotation string
	}{
		{"generic", "@co.dap.generic(types=[{name=T}])"},
		{"decorator", "@co.dap.decorator"},
		{"extension", "@co.dap.extension(fortype=co.lang.string, what=extends)"},
		{"macro", "@co.dap.macro"},
		{"template", "@co.dap.template"},
		{"native", "@co.dap.native"},
		{"execution model", "@co.dap.executionmodel(type=concurrent)"},
		{"indexer", `@co.dap.indexer(symbol="[]")`},
	}

	for left := 0; left < len(classifiers); left++ {
		for right := left + 1; right < len(classifiers); right++ {
			first, second := classifiers[left], classifiers[right]
			t.Run(first.name+" and "+second.name, func(t *testing.T) {
				annotations := first.annotation + "\n" + second.annotation
				_, p := parsePackageSource(t, "_ co.lang.unit = {\n"+annotations+"\nf()->() = {}\n}", "classification.unit.fol")
				if len(p.diags) == 0 {
					t.Fatal("two function-shape classifiers were accepted on one declaration")
				}
				var found bool
				for _, diagnostic := range p.diags {
					if strings.Contains(diagnostic.Error(), "mutually exclusive") {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("diagnostics = %v, want the function-shape mutual-exclusion rule", p.diags)
				}
			})
		}
	}
}

func TestVariantDefinitionRejectsInvalidConstructorSets(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"empty", "_ co.lang.unit = { Option(T) co.lang.type = co.lang.variants(); }", "requires at least one"},
		{"duplicate constructor", "_ co.lang.unit = { Option(T) co.lang.type = co.lang.variants(Some(T), Some()); }", "more than once"},
		{"missing constructor parentheses", "_ co.lang.unit = { Option(T) co.lang.type = co.lang.variants(Some); }", "variant constructor payload"},
		{"trailing constructor comma", "_ co.lang.unit = { Option(T) co.lang.type = co.lang.variants(Some(T),); }", "trailing comma"},
		{"trailing payload comma", "_ co.lang.unit = { Option(T) co.lang.type = co.lang.variants(Some(T,)); }", "trailing comma"},
		{"non-type declaration", "_ co.lang.unit = { Option co.lang.newtype = co.lang.variants(Some()); }", "variant-definition right-hand side"},
		{"ordinary expression", "_ co.lang.unit = { f()->() = { value := co.lang.variants(Some()); } }", "only as"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, p := parsePackageSource(t, test.source, "variants.unit.fol")
			if len(p.diags) == 0 {
				t.Fatal("invalid co.lang.variants definition was accepted")
			}
			if got := p.diags[0].Error(); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostic = %q, want text containing %q", got, test.want)
			}
		})
	}
}

func TestComponentContextRejectsWrongMemberKinds(t *testing.T) {
	tests := []struct {
		name    string
		basedir string
		source  string
		want    string
	}{
		{
			name:    "operator in application component",
			basedir: "components/application",
			source: `_ co.lang.component = {
    <+> co.lang.operator = { fixity: co.operator.fixity.infix, precedence: 60, associativity: co.operator.associativity.left, arity: co.operator.arity.binary };
}`,
			want: "components/operators/component.fol",
		},
		{
			name:    "function in operator component",
			basedir: "components/operators",
			source:  `_ co.lang.component = { f()->() = {} }`,
			want:    "only co.lang.operator",
		},
		{
			name:    "import in operator component",
			basedir: "components/operators",
			source: `_ co.lang.component = {
    @co.ddap.import(package="hr")
}`,
			want: "only co.lang.operator",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toks := normalizeTokens(scanlex.Tokenize(test.source, "component.fol"))
			p, _ := newParser(toks)
			p.file = fileinfo{Basename: "component.fol", Basedir: test.basedir, LocationKnown: true,
				Source: classifySourceFilename("component.fol")}
			p.parseCompilationUnit()
			if len(p.diags) == 0 {
				t.Fatal("context-invalid component member was accepted")
			}
			if got := p.diags[0].Error(); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostic = %q, want text containing %q", got, test.want)
			}
		})
	}
}

func parsePackageSource(t *testing.T, source, basename string) (ast.Stmt, *parser) {
	t.Helper()
	toks := normalizeTokens(scanlex.Tokenize(source, basename))
	p, _ := newParser(toks)
	p.file = fileinfo{Basename: basename, LocationKnown: true, Source: classifySourceFilename(basename)}
	return p.parseCompilationUnit(), p
}

func TestOperatorComponentUsesTheCommonComponentRoot(t *testing.T) {
	toks := normalizeTokens(scanlex.Tokenize(`_ co.lang.component = {
    <+> co.lang.operator = {
        fixity: co.operator.fixity.infix,
        precedence: 60,
        associativity: co.operator.associativity.left,
        arity: co.operator.arity.binary
    };
}`, "component.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{
		Basename:      "component.fol",
		Basedir:       "components/operators",
		LocationKnown: true,
		Source:        classifySourceFilename("component.fol"),
	}

	root := p.parseCompilationUnit()
	component, ok := root.(ast.ComponentDeclarationStmt)
	if !ok || len(component.Body) != 1 {
		t.Fatalf("operator component = %T with %d members; diagnostics: %v", root, len(component.Body), p.diags)
	}
	if len(p.diags) != 0 {
		t.Fatalf("operator component produced diagnostics: %v", p.diags)
	}
}

func TestFilenameDerivedNameIsValidatedAndLowered(t *testing.T) {
	root, _, _, _ := parseIntoConfigured(nil,
		`_ co.lang.struct = { id co.lang.int; }`,
		"Employee", ".", "Employee.fol", "people", "program", "program", true,
		parseConfiguration{locationKnown: true, atRoot: false},
	)
	pkg, ok := root.(ast.PackageStmt)
	if !ok || len(pkg.Body) != 1 {
		t.Fatalf("root = %T with unexpected body", root)
	}
	decl, ok := pkg.Body[0].(ast.TypeDeclarationStmt)
	if !ok {
		t.Fatalf("declaration = %T", pkg.Body[0])
	}
	if decl.Name != "Employee_fo" {
		t.Fatalf("name = %q, want Employee_fo", decl.Name)
	}
}

func TestPackageIdentityDistinguishesLegacyCallersFromKnownProjectRoot(t *testing.T) {
	tests := []struct {
		name string
		file fileinfo
		want string
	}{
		{"legacy API keeps basename fallback", fileinfo{Basename: "people"}, "people"},
		{"known project root is not a package", fileinfo{Basename: "people", LocationKnown: true, AtRoot: true}, ""},
		{"known subfolder uses package path", fileinfo{Basename: "Employee", PackagePath: "people", LocationKnown: true}, "people"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := parser{file: test.file}
			if got := p.packageIdentity(); got != test.want {
				t.Fatalf("package identity = %q, want %q", got, test.want)
			}
		})
	}
}
