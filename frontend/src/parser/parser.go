// Package parser implements the FoLang parser.
//
// # Structure
//
// The parser is a hybrid: a Pratt (precedence-climbing) engine drives every
// operator expression, and hand-written recursive descent drives declarations,
// statements and type syntax, where the grammar is keyword/kind-directed rather
// than operator-directed.
//
//   - Pratt is used wherever a construct is "an operand followed by operators":
//     the whole of section 11 of docs/grammar/folang.ebnf (assignment through
//     postfix) is one precedence-climbing loop keyed by operator lexeme. See
//     precedence.go and pratt.go.
//   - Recursive descent is used wherever the next construct is chosen by a
//     leading token rather than by binding power: compilation units, directives,
//     declarations by built-in kind, statements, type expressions and patterns.
//
// Every construct lives in its own file, named after the production it
// implements, so that docs/grammar/folang.ebnf can be read side by side with the
// source. For example enum-declaration is decl_enum.go, let-value-declaration is
// stmt_let.go, and the do/loop chains are ordinary postfix chains handled by
// expr_postfix.go.
//
// # Grammar conformance
//
// The normative sources are docs/language-ref.md and docs/grammar/folang.ebnf.
// Where the EBNF labels a decision (DECISION-SYN-006, DECISION-OP-002, …) the
// implementing function cites that label so the rule can be traced back.
//
// # Error handling
//
// A parse function either returns a node or aborts by panicking with a bailout
// sentinel after recording a diagnostic. Statement-list and member-list loops
// install a recovery point (see recover.go) which swallows the bailout, skips to
// the next plausible synchronisation token, and continues. That keeps a single
// bad statement from discarding the rest of the file while guaranteeing that no
// parse function ever has to check an error return from its callees.
package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/importcheck"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// fileinfo carries the identity of the source file being parsed. The package
// path is derived from the file's folder relative to the project root, because
// in FoLang every subfolder is a package (docs/language-ref.md, "Package
// Identity").
type fileinfo struct {
	Filename    string
	Basename    string
	Basedir     string
	PackagePath string
	// LocationKnown distinguishes an explicitly discovered project location from
	// the legacy Parse API, whose empty packagePath does not say whether the caller
	// means the project root or simply has no project metadata.
	LocationKnown bool
	// AtRoot is meaningful when LocationKnown is true. FoLang's project root is
	// not a package, so this bit participates in compilation-unit classification.
	AtRoot bool
	// Source is the parsed external source-filename (DECISION-FILE-001). It is
	// computed before the source text is read, because the filename — not the
	// body — is what selects the package-source-file alternative and supplies
	// every filename-derived declaration name and companion owner.
	Source sourceFilename
}

// parseConfiguration carries facts that are known by the project driver but are
// intentionally absent from the legacy Parse/ParseInto signatures.
type parseConfiguration struct {
	locationKnown bool
	atRoot        bool
	// operators is the project-wide lexical/precedence catalog. Visibility and
	// overload applicability remain semantic checks; the parser needs the catalog
	// only so a referenced custom spelling is one token with the right binding.
	operators []operatorDeclaration
	// environment is prepared before primary src parsing. Syntax parsing does
	// not resolve it directly yet; semantic/name-resolution phases consume the
	// isolated published contexts carried here.
	environment *PublishedEnvironment
}

// unitKind classifies the compilation unit, per the compilation-unit production.
// The forms share one file-preamble but differ in what may follow it.
type unitKind int

const (
	// unitEntry is an application-entry-file: a preamble followed by a
	// sequence of entry-items (statements, entry type declarations and
	// function-pattern clauses).
	unitEntry unitKind = iota
	// unitPackage is a package-source-file: a preamble followed by exactly
	// one primary-declaration.
	unitPackage
	// unitLibrary is a library-surface-file: a preamble followed by one
	// library-declaration.
	unitLibrary
	// unitComponent is a component-surface-file: a preamble followed by one
	// component-declaration. It is the root of `src/component.fol` and of every
	// `components/<kind>/component.fol`, the operators component included.
	unitComponent
)

// parser holds the whole mutable parse state.
//
// The token cursor (toks/pos) is the only state that speculative parsing has to
// rewind, together with the diagnostic list. Parsing deliberately does not
// populate the shared symbol table: each AST node gets its own freshly minted
// symbol record (see symbolfactory.go) so that a speculative parse can be thrown
// away without leaving partial declarations behind. Binding names to scopes is
// the job of the later semantic pass, which owns Context and SymbolTable.
type parser struct {
	id   string
	toks []scanlex.Token
	pos  int

	// diags accumulates diagnostics in source order, capped at
	// foerrors.MaxParseErrors by record().
	diags []helpers.ErrorInterface
	// diagsTruncated reports that the cap was reached and further diagnostics
	// were dropped, so a caller can say so rather than implying the list is
	// complete.
	diagsTruncated bool

	file fileinfo
	unit unitKind

	// ctx/symtab/fs are the root scope handed back to the caller. They are
	// created here so that downstream phases have a root to hang symbols on.
	ctx    *symboltable.Context
	symtab *symboltable.SymbolTable
	fs     *symboltable.FolangSymbols

	// ops is the user-defined operator registry consulted by the Pratt
	// engine for any operator lexeme that is not built in (DECISION-EXT-001).
	ops *operatorTable
	// operatorSignatures detects duplicate normalized operator declarations in
	// the one named class or unit body a package source file may contain.
	operatorSignatures map[string]scanlex.Token

	// buildLibs records whether the file asked the driver to build libraries,
	// which a source-library import implies.
	buildLibs bool

	// imports collects the file's import directives for the importcheck phase. They are
	// captured during parsing because ast.ImportStmt carries no source position and the
	// diagnostics need one.
	imports []importcheck.Import

	// libraryName and libraryType record a library-surface file's declared name and kind,
	// which the restricted-import rules are stated in terms of. They stay empty for an
	// entry file or a package source file.
	libraryName string
	libraryType string

	// selfReceiverDepth is greater than zero while a declaration that supplies a
	// class/type receiver for `self` is being parsed. See selfContextGuard.
	selfReceiverDepth int

	// refinementPredicateDepth is greater than zero while the predicate of a
	// co.lang.refinementType declaration is being parsed, which is the one place
	// `_` denotes the candidate value. See refinementCandidateGuard.
	refinementPredicateDepth int

	// lifecycle describes the enclosing class body's lifecycle-customization
	// capability while its members are being parsed. It is what
	// class-lifecycle-capability-guard and lifecycle-declaration-context-guard
	// test a source-declared @@new or @@init against; outside a class body it is
	// the zero value, which admits no lifecycle declaration.
	lifecycle lifecycleCapability

	// speculating is greater than zero while a tentative parse is running.
	// Diagnostics are still collected, but recover.go must not treat a
	// bailout as recoverable while a speculation is in flight.
	speculating int

	// depth guards against runaway recursion on pathological input.
	depth int
	// indentLevel is the per-parser nesting depth of the optional human-readable
	// debug trace. It is independent of the recursion guard above.
	indentLevel int

	// lambdaParamDepth is greater than zero while a lambda's parameter list is being
	// parsed. A lambda is delimited by "|", which is also the type-union operator, so a
	// parameter's type annotation must not absorb the closing delimiter. See
	// parseUnionTypeExpression.
	lambdaParamDepth int

	// lambdaCallContexts records whether each currently parsed call target is a
	// collection operation that accepts an inline lambda. Permission is passed
	// directly to the lambda parser for one argument; it is deliberately not kept
	// as ambient state, because an outer collection callback must not make a nested
	// non-collection call's lambda legal.
	lambdaCallContexts []bool

	// expressionModes is a stack because an expression may contain grammar
	// productions that start another expression parse (for example a grouped
	// expression or a call argument). Those nested parses must inherit restrictions
	// imposed by their enclosing expression. At present the only restricted mode is
	// constant-expression, which excludes assignment by operator role rather than
	// by binding power so project operators at every declared precedence remain
	// available.
	expressionModes []expressionMode
}

// maxRecursionDepth bounds nesting of recursive productions. Real source never
// approaches this; malformed source with unbalanced brackets can, and a bounded
// depth turns a stack overflow into an ordinary diagnostic.
const maxRecursionDepth = 400

// Parser is the exported alias for the internal parser type, retained because
// other phases refer to the type by name.
type Parser = parser

// newParser builds a parser over an already normalised token stream and creates
// the root context and symbol table.
func newParser(toks []scanlex.Token) (*parser, *symboltable.Context) {

	fs := &symboltable.FolangSymbols{}
	fs.CreateFolangSymbols()

	ctx, symtab := CreateNewContext("", string(symboltable.S_Program))
	fs.AddContext(ctx)
	fs.AddSymbolTable(symtab)

	return &parser{
		id:                 helpers.GenUniqueName("parser"),
		toks:               toks,
		pos:                0,
		ctx:                ctx,
		symtab:             symtab,
		fs:                 fs,
		ops:                newOperatorTable(),
		operatorSignatures: map[string]scanlex.Token{},
	}, ctx
}

// CreateNewContext creates a child context and its symbol table under the given
// parent context id. contextType selects the default resolution policy for the
// new scope.
func CreateNewContext(parentCtxID string, contextType string) (*symboltable.Context, *symboltable.SymbolTable) {

	ctx := &symboltable.Context{
		Id:                        helpers.NewContextId(),
		ParentId:                  parentCtxID,
		ContextType_:              contextType,
		RestrictedSymbolNameReuse: []string{},
		ImportedContextIds:        map[string]string{},
		ChildCtxIds:               []string{},
		ResolutionPolicy:          resolutionPolicyFor(contextType),
	}
	symtab := &symboltable.SymbolTable{
		Id:            helpers.NewSymbolTableId(),
		ContextId:     ctx.Id,
		Prefix:        ctx.Id,
		Symboldetails: map[string]symboltable.SymbolInfo{},
	}
	ctx.SymbolTable_ = symtab.Id
	return ctx, symtab
}

// Parse tokenizes and parses source, returning the AST root, the token stream,
// the root context, and whether the file requires libraries to be built.
//
// When parse is false the function stops after tokenizing; this is the mode the
// driver uses for `--stopAt Tokens` and that the tests use to assert on the
// token stream alone.
func Parse(source string, name string, dir string, basename string, packagePath string, contextid string, symbolid string, parse bool) (ast.Stmt, []scanlex.Token, *symboltable.Context, bool) {
	return ParseInto(nil, source, name, dir, basename, packagePath, contextid, symbolid, parse)
}

// ParseInto is Parse with a hook for whole-program import checking.
//
// The graph argument decides who OWNS import checking:
//
//   - nil means this is a standalone parse. The parser runs the import checks it can decide
//     from one file — a package importing itself, and a library surface reaching outside its
//     own boundary — and reports them itself.
//
//   - non-nil transfers ownership to the caller. The file's import edges are recorded
//     into the graph and no findings are raised here, because a driver that has scanned the
//     whole project can check strictly more: cycles need every file's edges, and the
//     library-boundary rules need the project layout to know which library owns a file. Running
//     both would report the same problem twice.
func ParseInto(graph *importcheck.Graph, source string, name string, dir string, basename string, packagePath string, contextid string, symbolid string, parse bool) (ast.Stmt, []scanlex.Token, *symboltable.Context, bool) {
	// A non-empty package path proves that the file is below the project root.
	// An empty path is ambiguous for compatibility callers, so only the project
	// driver uses the explicitly located entry point below.
	configuration := parseConfiguration{locationKnown: packagePath != ""}
	return parseIntoConfigured(graph, source, name, dir, basename, packagePath, contextid, symbolid, parse, configuration)
}

// parseIntoConfigured is the shared parser entry point. The project driver uses
// it to enforce the root-versus-package-folder rules; public compatibility APIs
// continue to work when their callers do not have layout metadata.
//
// This is the BATCH path and its behaviour is unchanged: a file with any
// diagnostic ends the process through foerrors.HandleErrors and this function
// does not return. Callers that must survive a malformed file use ParseFile.
func parseIntoConfigured(graph *importcheck.Graph, source string, name string, dir string, basename string, packagePath string, contextid string, symbolid string, parse bool, configuration parseConfiguration) (ast.Stmt, []scanlex.Token, *symboltable.Context, bool) {
	result := parseCollecting(graph, source, name, dir, basename, packagePath, parse, configuration)
	if len(result.Diagnostics) > 0 {
		foerrors.HandleErrors(result.Diagnostics...)
	}
	return result.Root, result.Tokens, result.Context, result.BuildLibraries
}

// Result is the outcome of a non-fatal parse.
//
// Every field is populated even when Diagnostics is non-empty: that is the whole
// point of the type. A malformed file still yields its token stream and whatever
// tree the parser's recovery could build, which is what lets an editor keep
// highlighting, folding and navigating a buffer the user is halfway through
// typing.
type Result struct {
	// Root is the parsed tree. It is ast.DummyStmt only when recovery could not
	// produce anything at all.
	Root ast.Stmt
	// Tokens is always the complete stream, including for a file with lexical
	// errors.
	Tokens []scanlex.Token
	// Context is the root scope, or nil when parsing was not requested.
	Context *symboltable.Context
	// RootSymbolTable and Symbols are the isolated scope graph owned by this
	// parse. Declaration binding is populated by the semantic pass, but project
	// preparation must retain these objects rather than merging component scopes.
	RootSymbolTable *symboltable.SymbolTable
	Symbols         *symboltable.FolangSymbols
	// Diagnostics holds the lexical and syntactic findings in source order.
	Diagnostics []helpers.ErrorInterface
	// Truncated reports that MaxParseErrors was reached and further diagnostics
	// were dropped, so a caller can say "the first N problems" rather than
	// implying the list is complete.
	Truncated bool
	// BuildLibraries mirrors the batch API's build-libraries flag.
	BuildLibraries bool
}

// ParseFile parses one source file without ever terminating the process.
//
// It is the entry point for an embedding consumer — a language server above all.
// Where Parse stops the process at the first diagnostic, this returns everything
// it has: the token stream, the recovered tree, and the diagnostics themselves,
// each carrying the source range an editor needs.
//
// It also does not panic on malformed input. A panic that is NOT the parser's
// own recovery sentinel is still a bug and is deliberately left to propagate;
// see parseTopLevel.
func ParseFile(source, name, dir, basename, packagePath string) Result {
	configuration := parseConfiguration{locationKnown: packagePath != ""}
	return parseCollecting(nil, source, name, dir, basename, packagePath, true, configuration)
}

// ParseFileWithOperators is ParseFile for a project whose custom operator
// catalog has already been loaded.
//
// A custom operator cannot be recognised from one file alone, so a server that
// has read the project's components/operators/component.fol passes the catalog here; without it a
// registered spelling scans as an unknown symbolic run and the file reports
// errors the compiler would not.
func ParseFileWithOperators(source, name, dir, basename, packagePath string, operators Operators) Result {
	configuration := parseConfiguration{
		locationKnown: packagePath != "",
		operators:     operators.declarations,
	}
	return parseCollecting(nil, source, name, dir, basename, packagePath, true, configuration)
}

// Operators is an opaque handle to a project's custom operator catalog, so that
// a consumer can carry one between parses without depending on the parser's
// internal declaration representation.
type Operators struct {
	declarations []operatorDeclaration
}

// LoadOperators reads the project's configured operator bootstrap source.
// Findings are returned rather than reported, so a missing or malformed
// operator bootstrap surface does not stop a server from parsing the rest of the project.
func LoadOperators(rootDir string) (Operators, []error) {
	bootstrap := loadProjectOperatorBootstrap(rootDir)
	return Operators{declarations: bootstrap.Declarations}, bootstrap.Findings
}

// parseCollecting is the shared implementation. It NEVER calls HandleErrors;
// the batch wrapper above decides whether findings are fatal.
func parseCollecting(graph *importcheck.Graph, source, name, dir, basename, packagePath string, parse bool, configuration parseConfiguration) Result {
	normalized := normalizeLineEndings(source)

	// The configured operator source is parsed before ordinary files. Its immutable
	// catalog classifies complete custom symbolic runs and seeds the Pratt table;
	// this source file may implement those symbols but cannot register new ones.
	collection := declaredOperatorsIn(normalized, basename, configuration.operators)

	// Lexical diagnostics are collected rather than reported, so a bad byte costs
	// that byte instead of the whole token stream.
	var custom *scanlex.CustomOperators
	if !collection.Custom.Empty() {
		custom = collection.Custom
	}
	raw, lexical := scanlex.TokenizeCollecting(normalized, basename, custom)
	toks := normalizeTokens(raw)

	if !parse {
		return Result{Root: ast.DummyStmt{}, Tokens: toks, Diagnostics: lexical}
	}

	p, ctx := newParser(toks)
	if traceEnabled || DEBUG_TRACE {
		// Span offsets carried by tokens index into this exact string.
		p.traceSource(normalized)
	}
	p.preRegisterOperatorDeclarations(collection.Declarations)
	p.file = fileinfo{
		Filename:      name,
		Basename:      basename,
		Basedir:       dir,
		PackagePath:   packagePath,
		LocationKnown: configuration.locationKnown,
		AtRoot:        configuration.atRoot,
		Source:        classifySourceFilename(basename),
	}

	root := p.parseTopLevel()

	// Control-flow chains are recognised after parsing, not during it. The grammar requires
	// expression parsing to stay uniform (section 11a), so this pass narrows the canonical
	// chain shapes into their dedicated nodes over the finished tree.
	root = p.lowerControlFlow(root)

	// The import-relationship checks run after parsing, because they need the file's unit
	// kind and, for a library surface, its declared type — neither of which is known until
	// the whole file has been read.
	p.validateImports(graph)

	// Lexical findings come first: they are earlier in the pipeline, and a
	// consumer that shows only the first diagnostic should show the cause rather
	// than a syntactic consequence of it.
	diagnostics := append(append([]helpers.ErrorInterface{}, lexical...), p.diags...)

	return Result{
		Root:            root,
		Tokens:          toks,
		Context:         ctx,
		RootSymbolTable: p.symtab,
		Symbols:         p.fs,
		Diagnostics:     diagnostics,
		Truncated:       p.diagsTruncated,
		BuildLibraries:  p.buildLibs,
	}
}

// importFile assembles the record the importcheck package works from.
//
// For a library surface the owned package subtree is derived the same way the project scan
// derives it: a surface with no package path sits at the project root, which makes it a packaged
// library project owning every package; one below the root is a source library owning the subtree
// its own path names.
func (p *parser) importFile() importcheck.File {

	file := importcheck.File{
		Name:             p.file.Basename,
		PackagePath:      p.file.PackagePath,
		IsLibrarySurface: p.unit == unitLibrary,
		LibraryName:      p.libraryName,
		LibraryType:      p.libraryType,
		Imports:          p.imports,
	}

	if file.IsLibrarySurface {
		if p.file.PackagePath == "" {
			file.LibraryPath = importcheck.WholeProject
		} else {
			file.LibraryPath = logicalPathOf(p.file.PackagePath, stemOf(p.file.Basename))
		}
	}
	return file
}

// stemOf strips a file name's extension, giving the last segment of a surface's logical path.
func stemOf(basename string) string {
	if dot := strings.LastIndexByte(basename, '.'); dot > 0 {
		return basename[:dot]
	}
	return basename
}

// validateImports either checks this file's imports or contributes its edges to the caller's
// graph, per the ownership rule documented on ParseInto.
func (p *parser) validateImports(graph *importcheck.Graph) {
	if len(p.imports) == 0 {
		return
	}
	file := p.importFile()

	if graph != nil {
		graph.Add(file)
		return
	}

	// Standalone parse: run the checks decidable from one file. Cycles across files and the
	// library-internals rules need a project pass and are the driver's responsibility.
	p.appendFindings(importcheck.ValidateSelfImports(file))
	p.appendFindings(importcheck.ValidateRestrictedImports(file))
}

// appendFindings records diagnostics produced by another phase.
func (p *parser) appendFindings(findings []error) {

	for _, f := range findings {
		if diag, ok := f.(helpers.ErrorInterface); ok {
			p.diags = append(p.diags, diag)
		}
	}
}

// parseTopLevel parses the compilation unit under a final recovery point.
//
// Recovery normally happens at the item loops (see recover.go), which is where a malformed
// statement or member is contained. A few positions have no enclosing loop — chiefly a
// package source file's single declaration — so a failure there would unwind all the way out.
// This guard makes sure that becomes a reported diagnostic and a partial tree rather than a
// panic escaping Parse, which no caller is prepared for.
func (p *parser) parseTopLevel() (root ast.Stmt) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer func() {
		spanStart := p.pos
		r := recover()
		if r == nil {
			return
		}
		if _, isBailout := r.(bailout); !isBailout {
			panic(r)
		}
		// The diagnostic that triggered the bailout has already been recorded; all
		// that is left is to hand back a node so the caller has something to return.
		if len(p.diags) == 0 {
			p.report(p.cur(), "the file could not be parsed")
		}
		root = ast.DummyStmt{Span: p.spanFrom(spanStart)}
	}()

	return p.parseCompilationUnit()
}

// resolutionPolicyFor returns the default symbol resolution policy for a scope
// of the given context type, falling back to lexical ordered resolution.
func resolutionPolicyFor(contextType string) string {

	if byName, ok := symbolTypeToResolutionPolicy[contextType]; ok {
		if pol, ok := byName["default"]; ok {
			return pol
		}
	}
	return "lexical_static_ordered"
}

// symbolTypeToResolutionPolicy maps a symbol/scope kind to the resolution
// policies it supports. "default" is the policy used unless a declaration opts
// into another one (for example a dynamically scoped function).
var symbolTypeToResolutionPolicy = map[string]map[string]string{
	string(symboltable.S_Program):                {"default": "lexical_static_ordered"},
	string(symboltable.S_PackageSymbol):          {"default": "lexical_static_complete_container"},
	string(symboltable.S_ClassSymbol):            {"default": "lexical_static_complete_container"},
	string(symboltable.S_ModuleSymbol):           {"default": "lexical_static_complete_container"},
	string(symboltable.S_InterfaceSymbol):        {"default": "lexical_static_complete_container"},
	string(symboltable.S_SignatureSymbol):        {"default": "lexical_static_complete_container"},
	string(symboltable.S_EnumSymbol):             {"default": "lexical_static_complete_container"},
	string(symboltable.S_StructSymbol):           {"default": "lexical_static_complete_container"},
	string(symboltable.S_UnionSymbol):            {"default": "lexical_static_complete_container"},
	string(symboltable.S_BlockSymbol):            {"default": "lexical_static_ordered"},
	string(symboltable.S_DelegateSymbol):         {"default": "lexical_static_ordered"},
	string(symboltable.S_ExtensionSymbol):        {"default": "lexical_static_ordered"},
	string(symboltable.S_InstanceSymbol):         {"default": "lexical_static_ordered"},
	string(symboltable.S_ObjectSymbol):           {"default": "lexical_static_ordered"},
	string(symboltable.S_MatcherSymbol):          {"default": "lexical_static_ordered"},
	string(symboltable.S_MatcherImplSymbol):      {"default": "lexical_static_ordered"},
	string(symboltable.S_FunctionPattern):        {"default": "lexical_static_ordered"},
	string(symboltable.S_OperatorDetails):        {"default": "lexical_static_ordered"},
	string(symboltable.S_TemplateDetails):        {"default": "lexical_static_ordered"},
	string(symboltable.S_AnnotationSymbol):       {"default": "lexical_static_ordered"},
	string(symboltable.S_TypeConstructor):        {"default": "lexical_static_ordered"},
	string(symboltable.S_TypeclassSymbol):        {"default": "lexical_static_ordered"},
	string(symboltable.S_LabelSymbol):            {"default": "lexical_static_ordered"},
	string(symboltable.S_LambdaSymbol):           {"default": "lexical_static_ordered"},
	string(symboltable.S_ForAllSymbol):           {"default": "lexical_static_ordered"},
	string(symboltable.S_DirectivePragmaDetails): {"default": "lexical_static_ordered"},
	string(symboltable.S_DecoratorSymbol):        {"default": "lexical_static_ordered"},
	string(symboltable.S_ForComprehension):       {"default": "lexical_static_ordered"},
	string(symboltable.S_HokrtlSymbol):           {"default": "lexical_static_ordered"},
	string(symboltable.S_Indexer):                {"default": "lexical_static_ordered"},
	string(symboltable.S_KindSymbol):             {"default": "lexical_static_ordered"},
	string(symboltable.S_TypeSymbol):             {"default": "lexical_static_ordered"},
	string(symboltable.S_AddressSymbol):          {"default": "lexical_static_ordered"},
	string(symboltable.S_PointerSymbol):          {"default": "lexical_static_ordered"},
	string(symboltable.S_RangeSymbol):            {"default": "lexical_static_ordered"},
	string(symboltable.S_ReferenceSymbol):        {"default": "lexical_static_ordered"},
	string(symboltable.S_GenericDetails):         {"default": "lexical_static_ordered"},
	string(symboltable.S_MacroSymbol): {
		"default":  "macro_definition_site",
		"unhygene": "macro_expansion_site",
	},
	string(symboltable.S_FunctionSymbol): {
		"default":      "lexical_static_ordered",
		"closure":      "late_lexical_formation_site",
		"inner":        "late_lexical_call_site",
		"curry":        "late_lexical_formation_site",
		"dynamicscope": "dynamic_call_site",
		"inline":       "dynamic_call_site",
	},
	string(symboltable.S_VarSymbol): {
		"default": "lexical_static_ordered",
		"extern":  "runtime_bound",
		"foreign": "runtime_bound",
		"forward": "late_lexical_static",
	},
	string(symboltable.S_DymanicRuntime): {
		"default":      "runtime_lexical_static",
		"dynamicscope": "runtime_dynamic_callsite",
	},
}
