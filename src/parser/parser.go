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
	"path/filepath"
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/foerrors"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/importcheck"
	"github.com/samkrao/fo-lang/src/scanlex"
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
	// importContexts is the project header/index pass's canonical target map.
	// Import directives link to these contexts while the preamble is parsed, so
	// the declaration/body pass starts with its imports already addressable.
	importContexts map[string]*symboltable.Context
	// scope makes this parse a member of a scope model that already exists,
	// rather than the owner of a fresh one. A project parse sets it so that every
	// file lands in ONE FolangSymbols under one project context; left zero, the
	// parse creates its own model exactly as a standalone file parse always has.
	scope projectScope
}

// projectScope is the shared scope model a project parse threads through its
// files: the symbol model every file contributes to, and the context each file's
// root becomes a child of.
type projectScope struct {
	symbols *symboltable.FolangSymbols
	parent  *symboltable.Context
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
	// unitComponent is a component-surface-file: a preamble followed by one
	// component-declaration. It is the root of `src/component.fol` and of every
	// `components/<kind>/component.fol`, the operators component included.
	unitComponent
)

// parser holds the whole mutable parse state.
//
// Speculative parsing rewinds the token cursor (toks/pos), the diagnostic list
// and the scope model, in that order of cost; scope.go explains the last.
//
// Parsing builds the SHAPE of the scope model — the contexts and the
// symbol-table segments — but still enters no name into a table: each AST node
// gets its own freshly minted symbol record (see symbolfactory.go), anchored to
// the segment active at its source position. Binding those names into scopes is
// the job of the later semantic pass.
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

	// ctx/symtab/fs are the parse's scope model. ctx and symtab are the context
	// and the visibility segment ACTIVE at the cursor, which scope.go advances as
	// blocks open and as declarations interleave with statements; fs holds every
	// context and segment the file produced, rooted at the one created here.
	ctx               *symboltable.Context
	symtab            *symboltable.SymbolTable
	fs                *symboltable.FolangSymbols
	identity          string
	symbolOccurrences map[string]int

	// sawExecutable reports that a statement or an expression has been read in the
	// current context since its active segment began, which is what makes the next
	// variable declaration an interleaved one. See beginDeclarationSegment.
	sawExecutable bool
	// scopeJournal holds the inverse of each scope mutation made while a
	// speculation is in flight, so a rewound parse leaves no context behind.
	scopeJournal []func()

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
	imports        []importcheck.Import
	importContexts map[string]*symboltable.Context

	// thisReceiverDepth is greater than zero while a callable that supplies a
	// receiver for `this` is being parsed. Control forms such as this.return are
	// parsed separately and do not consult this counter.
	thisReceiverDepth int

	// directRelationships holds the enclosing class's ordered @co.dap.oops
	// relationship lists while its body is parsed. It enables the dedicated
	// base/parent selector primaries before ordinary postfix parsing.
	directRelationships map[string][]string
	classRelationDepth  int

	// kindOptionDepth distinguishes ordinary maps nested in kind options (for
	// example fat-pointer meta={len:Type}) from @co.* metadata maps, whose named
	// fields use only the declarative '=' binder.
	kindOptionDepth int

	// refinementPredicateDepth is greater than zero while the predicate of a
	// co.lang.refinementType declaration is being parsed, which is the one place
	// `_` denotes the candidate value. See refinementCandidateGuard.
	refinementPredicateDepth int

	// anonymousFunctionBinding permits an anonymous-function expression only as
	// the root value of a variable/function-object binding initializer. The
	// permission is consumed when that root primary is parsed, so nested call
	// arguments, returns and arbitrary subexpressions cannot introduce one.
	anonymousFunctionBinding bool

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
	return newParserIn(toks, projectScope{}, "standalone")
}

// newParserIn is newParser for a file that belongs to a scope model that already
// exists.
//
// A project is one scope model, not one per file: a package spans its folder, so
// two files in it must be able to see each other's declarations, and that is only
// true if their contexts live in the same FolangSymbols. When scope carries a
// parent, this file's root context becomes a child of it and records the parent's
// active segment as its branch point, exactly as any nested context does.
func newParserIn(toks []scanlex.Token, scope projectScope, identity string) (*parser, *symboltable.Context) {

	fs := scope.symbols
	if fs == nil {
		fs = &symboltable.FolangSymbols{}
		fs.CreateFolangSymbols()
	}

	parentId := ""
	if scope.parent != nil {
		parentId = scope.parent.Id
	}

	ctx, symtab := CreateNewContext(parentId, symboltable.S_Program, identity)
	if scope.parent != nil {
		ctx.ParentCtxSymbolTableId = scope.parent.SymbolTable_
		scope.parent.ChildCtxIds = append(scope.parent.ChildCtxIds, ctx.Id)
	}
	fs.AddContext(ctx)
	fs.AddSymbolTable(symtab)
	if fs.RootContextId == "" {
		fs.RootContextId = ctx.Id
		for root := scope.parent; root != nil; root = fs.GetContext(root.ParentId) {
			fs.RootContextId = root.Id
			if root.ParentId == "" {
				break
			}
		}
	}

	return &parser{
		id:                 helpers.GenUniqueName("parser"),
		toks:               toks,
		pos:                0,
		ctx:                ctx,
		symtab:             symtab,
		fs:                 fs,
		ops:                newOperatorTable(),
		operatorSignatures: map[string]scanlex.Token{},
		identity:           identity, symbolOccurrences: map[string]int{},
	}, ctx
}

// CreateNewContext creates a child context and its symbol table under the given
// parent context id. contextType selects the default resolution policy for the
// new scope.
func CreateNewContext(parentCtxID string, contextType symboltable.SymbolsToString, discriminator ...string) (*symboltable.Context, *symboltable.SymbolTable) {
	identity := strings.Join(discriminator, "\x00")
	contextID := helpers.StableID("ctx", parentCtxID, string(contextType), identity)

	ctx := &symboltable.Context{
		Id:                        contextID,
		ParentId:                  parentCtxID,
		ContextType_:              contextType,
		RestrictedSymbolNameReuse: []string{},
		ImportedContextIds:        map[string]string{},
		ChildCtxIds:               []string{},
		ResolutionPolicy:          resolutionPolicyFor(contextType),
	}
	symtab := &symboltable.SymbolTable{
		Id:            helpers.StableID("symtab", contextID, "segment", "0"),
		ContextId:     ctx.Id,
		Prefix:        ctx.Id,
		SymbolsByName: map[string][]string{},
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
		return Result{Root: ast.DummyStmt{NodeName: "DummyStmt"}, Tokens: toks, Diagnostics: lexical}
	}

	fileIdentity := helpers.CanonicalIdentityPath(filepath.Join(dir, basename))
	p, ctx := newParserIn(toks, configuration.scope, fileIdentity)
	p.importContexts = configuration.importContexts
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
// IsLibrarySurface, LibraryName, LibraryType and LibraryPath are deliberately left
// unset. They were filled from the withdrawn `library.fol` surface, whose parse root
// this revision removed along with the rest of that form; the reference's surviving
// standalone-library surface is `src/component.fol` carrying `@co.dap.library`.
//
// So importcheck's library-boundary rules — direction.go and the surface half of
// restricted.go — currently receive no library at all. That is not a regression:
// the fields have been unreachable for as long as `library.fol` has been, so the
// checks were already inert. Re-pointing them at the component surface is a
// separate change, because it restores enforcement rather than removing dead code,
// and it is recorded in docs/parser-conformance-audit.md.
func (p *parser) importFile() importcheck.File {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return importcheck.File{
		Name:        p.file.Basename,
		PackagePath: p.file.PackagePath,
		Imports:     p.imports,
	}
}

// validateImports either checks this file's imports or contributes its edges to the caller's
// graph, per the ownership rule documented on ParseInto.
func (p *parser) validateImports(graph *importcheck.Graph) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

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
		root = ast.DummyStmt{NodeName: "DummyStmt", Span: p.spanFrom(spanStart)}
	}()

	return p.parseCompilationUnit()
}

// resolutionPolicyFor returns the default symbol resolution policy for a scope
// of the given context type, falling back to lexical ordered resolution.
func resolutionPolicyFor(contextType symboltable.SymbolsToString) symboltable.ResolutionPolicy {

	if byName, ok := symbolTypeToResolutionPolicy[string(contextType)]; ok {
		if pol, ok := byName["default"]; ok {
			return pol
		}
	}
	return symboltable.LexicalOrdered
}

// symbolTypeToResolutionPolicy maps a symbol/scope kind to the resolution
// policies it supports. "default" is the policy used unless a declaration opts
// into another one (for example a dynamically scoped function).
var symbolTypeToResolutionPolicy = map[string]map[string]symboltable.ResolutionPolicy{
	string(symboltable.S_Program):                {"default": symboltable.LexicalOrdered},
	string(symboltable.S_PackageSymbol):          {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_ClassSymbol):            {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_ModuleSymbol):           {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_InterfaceSymbol):        {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_SignatureSymbol):        {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_EnumSymbol):             {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_StructSymbol):           {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_UnionSymbol):            {"default": symboltable.LexicalCompleteContainer},
	string(symboltable.S_BlockSymbol):            {"default": symboltable.LexicalOrdered},
	string(symboltable.S_DelegateSymbol):         {"default": symboltable.LexicalOrdered},
	string(symboltable.S_ExtensionSymbol):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_InstanceSymbol):         {"default": symboltable.LexicalOrdered},
	string(symboltable.S_ObjectSymbol):           {"default": symboltable.LexicalOrdered},
	string(symboltable.S_MatcherSymbol):          {"default": symboltable.LexicalOrdered},
	string(symboltable.S_MatcherImplSymbol):      {"default": symboltable.LexicalOrdered},
	string(symboltable.S_FunctionPattern):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_OperatorDetails):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_TemplateDetails):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_AnnotationSymbol):       {"default": symboltable.LexicalOrdered},
	string(symboltable.S_TypeConstructor):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_TypeclassSymbol):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_LabelSymbol):            {"default": symboltable.LexicalOrdered},
	string(symboltable.S_LambdaSymbol):           {"default": symboltable.LexicalOrdered},
	string(symboltable.S_ForAllSymbol):           {"default": symboltable.LexicalOrdered},
	string(symboltable.S_DirectivePragmaDetails): {"default": symboltable.LexicalOrdered},
	string(symboltable.S_DecoratorSymbol):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_ForComprehension):       {"default": symboltable.LexicalOrdered},
	string(symboltable.S_HokrtlSymbol):           {"default": symboltable.LexicalOrdered},
	string(symboltable.S_Indexer):                {"default": symboltable.LexicalOrdered},
	string(symboltable.S_KindSymbol):             {"default": symboltable.LexicalOrdered},
	string(symboltable.S_TypeSymbol):             {"default": symboltable.LexicalOrdered},
	string(symboltable.S_AddressSymbol):          {"default": symboltable.LexicalOrdered},
	string(symboltable.S_PointerSymbol):          {"default": symboltable.LexicalOrdered},
	string(symboltable.S_RangeSymbol):            {"default": symboltable.LexicalOrdered},
	string(symboltable.S_ReferenceSymbol):        {"default": symboltable.LexicalOrdered},
	string(symboltable.S_GenericDetails):         {"default": symboltable.LexicalOrdered},
	string(symboltable.S_MacroSymbol): {
		"default":  symboltable.MacroDefinitionSite,
		"unhygene": symboltable.MacroExpansionSite,
	},
	string(symboltable.S_FunctionSymbol): {
		"default":      symboltable.LexicalOrdered,
		"closure":      symboltable.LateLexicalFormationSite,
		"inner":        symboltable.LateLexicalCallSite,
		"curry":        symboltable.LateLexicalFormationSite,
		"dynamicscope": symboltable.DynamicCallSite,
		"inline":       symboltable.DynamicCallSite,
	},
	string(symboltable.S_VarSymbol): {
		"default": symboltable.LexicalOrdered,
		"extern":  symboltable.RuntimeBound,
		"foreign": symboltable.RuntimeBound,
		"forward": symboltable.LateLexicalCallSite,
	},
	string(symboltable.S_DymanicRuntime): {
		"default":      symboltable.RuntimeBound,
		"dynamicscope": symboltable.DynamicCallSite,
	},
}
