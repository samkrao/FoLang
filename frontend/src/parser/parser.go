// Package parser implements the FoLang recursive-descent parser.
package parser

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// BuiltIn represents the category of a built-in symbol (kind or type).
type BuiltIn int

const (
	KIND_ BuiltIn = iota
	TYPE_
)

// StmtKind classifies the kind of statement being parsed.
type StmtKind int

const (
	NASTMT      StmtKind = iota //0
	CONDITIONAL                 //1
	PACKAGE                     //4
	CO                          //11
	TYPE
	KIND
	FUNCTION
	STRUCT
	FUN
	FUNTYPE
	SIGNATURE
	INTERFACE
	CSTRUCT
	TYPECLASS
	MODULE
	OBJECT
	INSTANCE
	EXTENSION
	MATCHER
	ENUM
	CLASS
	MACRO
	INDEXER
	TEMPLATE
	GENERIC
	CODE
	VARFIELDMEMBERPROP
	STEX
	THIS
	FOR
	LET
	BUILTIN_STMT_EXPRS //23
	BUILTIN_METHOD     //24
	BUILTIN_TYPE       //25
	BUILTIN_DIRECTIVES //29
	CUSTOM_DAP         //30
	BUILTIN_KIND
	BUILTIN_CONSTANTS
	IDENTIFIERS
	RESERVEDWORDS
	FORALL
	UNION
	LABEL
	NAMED_BLOCK
	LIBRARY
	SOURCE_LIBRARY
)

// StatementType enumerates the concrete declaration or statement forms.

var stmt_to_stmtKind map[string]StmtKind = map[string]StmtKind{
	"for":    FOR,
	"let":    LET,
	"forall": FORALL,
}
var kind_to_StmtKind map[string]StmtKind = map[string]StmtKind{
	"co.lang.struct":         STRUCT,
	"co.lang.cstruct":        STRUCT, // C-like value type for zone boundary safety
	"co.lang.union":          UNION,
	"co.lang.package":        PACKAGE,
	"co.lang.module":         MODULE,
	"co.lang.type":           TYPE,
	"co.lang.kind":           KIND,
	"co.lang.newtype":        TYPE,
	"co.lang.data":           TYPE,
	"co.lang.opaquetype":     TYPE,
	"co.lang.subtype":        TYPE,
	"co.lang.supertype":      TYPE,
	"co.lang.dependentType":  TYPE,
	"co.lang.associatedtype": TYPE,
	"co.lang.hkt":            TYPE, //higher kind type
	"co.lang.hot":            TYPE, //higher orderr type
	"co.lang.hrt":            TYPE, //higher rank type
	"co.lang.enum":           TYPE,
	"co.lang.interface":      TYPE,
	"co.lang.signature":      TYPE,
	"co.lang.word":           TYPE,
	"co.lang.function":       FUNCTION,
	"co.lang.tag":            TYPE, // tagged value type — co.lang.tag(type, value)
	"co.lang.instance":       TYPE,
	"co.lang.object":         TYPE,
}

// Iterator_meth_to_Type maps iterator method names to the collection types that support them.
var Iterator_meth_to_Type map[string][]string = map[string][]string{
	"each":        {"ARR", "LIST", "SET", "TUPLE", "MAP", "DICT", "RANGE", "SLICE"},
	"contains":    {"ARR", "LIST", "SET", "TUPLE", "SLICE", "MAP", "DICT"},
	"containsVal": {"MAP", "DICT"},
}

// IteratorMethods lists the built-in iterator method names.
var IteratorMethods []string = []string{"each"}

// ISIn lists the built-in containment-check method names.
var ISIn []string = []string{"contains", "containsVal"}

// PMM lists the built-in pattern matching method names.
var PMM []string = []string{"match", "case", "default"}

// Fo_Types lists the FoLang built-in type kind strings.
var Fo_Types []string = []string{"co.lang.struct", "co.lang.cstruct", "co.lang.data", "co.lang.type", "co.lang.module", "co.lang.class", "co.lang.interface"}

var relational_logical_ops []scanlex.TokenKind = []scanlex.TokenKind{
	scanlex.LESS,
	scanlex.GREATER,
	scanlex.GREATER_EQUALS,
	scanlex.LESS_EQUALS,
	scanlex.NOT,
	scanlex.NOT_EQUALS,
	scanlex.AND,
	scanlex.OR,
	scanlex.EQUALS,
}
var adt_operators []scanlex.TokenKind = []scanlex.TokenKind{
	scanlex.PIPE,
	scanlex.AMPS,
	scanlex.POW,
}

var sp_adt []scanlex.TokenKind = []scanlex.TokenKind{scanlex.PIPE}

// Vartype indicates the role of a variable (parameter, argument, etc.).
type Vartype int

const (
	VAR Vartype = iota
	PARAM
	ARG
	RESULT
	FUNC
)

// GetVarType returns the string representation of a Vartype value.
func GetVarType(v Vartype) string {
	switch v {
	case VAR:
		return "var"
	case PARAM:
		return "param"
	case ARG:
		return "argument"
	case RESULT:
		return "result"
	default:
		return "UnKnown"
	}
}

var SymbolTypeToResolpolicy map[string]map[string]string = map[string]map[string]string{

	string(symboltable.S_MacroSymbol):            {"default": "macro_definition_site", "unhygene": "macro_expansion_site"},
	string(symboltable.S_PackageSymbol):          {"default": "lexical_static_complete_container"},
	string(symboltable.S_ClassSymbol):            {"default": "lexical_static_complete_container"},
	string(symboltable.S_ModuleSymbol):           {"default": "lexical_static_complete_container"},
	string(symboltable.S_InterfaceSymbol):        {"default": "lexical_static_complete_container"},
	string(symboltable.S_SignatureSymbol):        {"default": "lexical_static_complete_container"},
	string(symboltable.S_BlockSymbol):            {"default": "lexical_static_ordered"},
	string(symboltable.S_EnumSymbol):             {"default": "lexical_static_complete_container"},
	string(symboltable.S_DelegateSymbol):         {"default": "lexical_static_ordered"},
	string(symboltable.S_ExtensionSymbol):        {"default": "lexical_static_ordered"},
	string(symboltable.S_InstanceSymbol):         {"default": "lexical_static_ordered"},
	string(symboltable.S_ObjectSymbol):           {"default": "lexical_static_ordered"},
	string(symboltable.S_MatcherSymbol):          {"default": "lexical_static_ordered"},
	string(symboltable.S_MatcherImplSymbol):      {"default": "lexical_static_ordered"},
	string(symboltable.S_FunctionSymbol):         {"default": "lexical_static_ordered", "closure": "late_lexical_formation_site", "inner": "late_lexical_call_site", "dynamicscope": "dynamic_call_site", "curry": "late_lexical_formation_site", "inline": "dynamic_call_site"},
	string(symboltable.S_FunctionPattern):        {"default": "lexical_static_ordered"},
	string(symboltable.S_VarSymbol):              {"extern": "runtime_bound", "foreign": "runtime_bound", "forward": "late_lexical_static"},
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
	string(symboltable.S_DymanicRuntime):         {"default": "runtime_lexical_static", "dynamicscope": "runtime_dynamic_callsite"},
	string(symboltable.S_GenericDetails):         {"default": "lexical_static_ordered"},
	string(symboltable.S_Program):                {"default": "lexical_static_ordered"},
}

type fileinfo struct {
	Filename             string
	Basename             string
	Basedir              string
	Entry_or_surface_dir string
	PackagePath          string
}

type parser struct {
	Id               string
	tokens           []scanlex.Token
	pos              int
	module           bool
	Context_         *symboltable.Context
	SymbolTable_     *symboltable.SymbolTable
	IsPrevDeclSymbol bool
	Fs               *symboltable.FolangSymbols
	ChangeCtx        bool
	list_ids         []string
	SKIP_SEMI_COMMA  bool
	Stack            []interface{} // Shift-reduce stack
	IsFunction       bool
	IsADT            bool
	ParentIsFunc     bool
	Errors           []helpers.ErrorInterface
	Scope            string
	Let              bool
	GenericType      bool
	AdditionalInfo   map[string]any
	Adhoc            bool
	AdhocToken       scanlex.Token
	LetBinding       bool
	recDepth         int // recursion depth counter (thread-safe: per-instance)
	debugTrace       bool
	traceDepth       int
	fileInfo         fileinfo
	AllLets          bool
	Entry            bool
	LibType          string
	LibSurface       bool
}

// Parser is the exported alias for the internal parser type.
type Parser = parser

// 1. TopDown  (LL) -> Recursive Descent and Pratt Parsing
// 2. BottomUp (LR) -> Recursive Ascent Parser
func createParser(tokens []scanlex.Token) (*parser, *symboltable.Context) {
	fs := &symboltable.FolangSymbols{}
	fs.CreateFolangSymbols()

	ctx, symb := CreateNewContext("", string(symboltable.S_Program))
	fs.AddContext(ctx)
	fs.AddSymbolTable(symb)

	initLookupsOnce.Do(func() {
		createTokenLookups()
		createTypeTokenLookups()
	})

	p := &parser{
		Id:               helpers.GenUniqueName("parser"),
		tokens:           tokens,
		pos:              0,
		module:           false,
		Fs:               fs,
		IsPrevDeclSymbol: true,
		Context_:         ctx,
		SymbolTable_:     symb,
		ChangeCtx:        false,
		SKIP_SEMI_COMMA:  false,
		IsFunction:       false,
		IsADT:            false,
		ParentIsFunc:     false,
		Let:              false,
		GenericType:      false,
		LetBinding:       false,
		Adhoc:            false,
		AdhocToken:       scanlex.DummyNode,
		recDepth:         50,
		AdditionalInfo:   map[string]any{},
		debugTrace:       true,
		traceDepth:       1,
		AllLets:          true,
		Entry:            false,
		LibType:          "application",
		LibSurface:       false,
	}

	return p, ctx
}

// CreateNewContext creates a child context and symbol table under the current parser scope.
func CreateNewContext(ctxid string, contextType string) (*symboltable.Context, *symboltable.SymbolTable) {
	cctxid := helpers.NewContextId()

	var cctx *symboltable.Context = &symboltable.Context{
		Id:                        cctxid,
		ParentId:                  ctxid,
		RestrictedSymbolNameReuse: []string{},
		SymbolTable_:              "",
		Prefix:                    "",
		ImportedContextIds:        map[string]string{},
		ChildCtxIds:               []string{},
		ResolutionPolicy:          SymbolTypeToResolpolicy[contextType]["default"],
	}
	CurrentSymTblId := helpers.NewSymbolTableId()
	symT := &symboltable.SymbolTable{
		Id:            CurrentSymTblId,
		ContextId:     cctx.Id,
		Prev:          "",
		Prefix:        cctx.Id,
		Symboldetails: map[string]symboltable.SymbolInfo{},
		Next:          "",
	}
	cctx.SymbolTable_ = symT.Id
	return cctx, symT
}
func createNewSymbolTable(ctxid string, parent_symbid string) *symboltable.SymbolTable {

	CurrentSymTblId := helpers.NewSymbolTableId()

	symT := &symboltable.SymbolTable{
		Id:            CurrentSymTblId,
		ContextId:     ctxid,
		Prev:          parent_symbid,
		Prefix:        ctxid,
		Symboldetails: map[string]symboltable.SymbolInfo{},
	}
	return symT
}
func updateContext(p *parser, symbSt ast.SET, childSymbolTable bool, redeclareable bool) *symboltable.Context {
	ctx := p.Context_
	symbolName := ""
	restricOverload := false
	fun := false
	type_ := ""

	var symbTab symboltable.SymbolTable = symboltable.SymbolTable{}
	//var currSymbTab symboltable.SymbolTable = symboltable.SymbolTable{}

	if childSymbolTable {
		symT := createNewSymbolTable(ctx.Id, p.SymbolTable_.Id)
		p.SymbolTable_.Next = symT.Id
		p.SymbolTable_ = symT
		p.Fs.AddSymbolTable(symT)
		ctx.SymbolTable_ = symT.Id

	}
	symbTab = *p.SymbolTable_

	symbolInfoMap1 := symbTab.Symboldetails
	sArr := strings.Split(symbolName, "=")
	keyName := ""
	if len(sArr) > 1 {
		keyName = sArr[0] + "_" + "type_" + sArr[1]
	} else {
		keyName = symbolName + "_" + type_
	}

	if fun && restricOverload {

		(p.Context_.RestrictedSymbolNameReuse) = append((p.Context_.RestrictedSymbolNameReuse), keyName)
	}
	if _, ok := symbolInfoMap1[keyName]; ok && !redeclareable {
		err_ := p.errorExpection(helpers.RemoveAfterUnderscore(symbolName, "_")+" of "+type_+" type already Declared", helpers.AlreadyDeclared)
		p.addErr(err_)
	}
	var symdet symboltable.SymbolInfo

	symbolInfoMap1[keyName] = symdet

	return ctx
}

// Parse tokenizes and parses the given source code, returning the AST root, tokens, context, and whether libraries should be built.
// packagePath is the dot-separated package identity derived from the file's folder relative to the project root
// (e.g. "hr.employee" for a file in hr/employee/). Pass "" for entry files at the project root.
func Parse(source string, name string, dir string, basename string, packagePath string, contextid string, symbolid string, parse bool) (ast.Stmt, []scanlex.Token, *symboltable.Context, bool) {
	tokens := scanlex.Tokenize(source, basename)
	if !parse {
		fmt.Println("No Parseing Just Tokenizing")
		return ast.DummyStmt{}, tokens, nil, false
	}
	p, ctx := createParser(tokens)
	fileInfo := fileinfo{
		Filename:             name,
		Basename:             basename,
		Basedir:              dir,
		Entry_or_surface_dir: dir,
		PackagePath:          packagePath,
	}
	p.fileInfo = fileInfo

	pr := parse_prg_stmt(p, name, packagePath)

	if pr.Errors == nil {
		return pr.Node.(ast.Stmt), tokens, ctx, pr.BuildLibs
	} else {
		foerrors.HandleErrors(pr.Errors...)

	}

	return ast.DummyStmt{}, tokens, nil, pr.BuildLibs
}

func (p *Parser) trace(name string) func() {
	if !p.debugTrace {
		return func() {}
	}

	indent := strings.Repeat("  ", p.traceDepth)
	fmt.Printf("[%s] %sENTER %s\n", p.Id, indent, name)
	p.traceDepth++

	return func() {
		p.traceDepth--
		indent := strings.Repeat("  ", p.traceDepth)
		fmt.Printf("[%s] %sEXIT  %s\n", p.Id, indent, name)
	}
}

func (p *Parser) traceCurrent() func() {
	if !p.debugTrace {
		return func() {}
	}

	name := "unknown"

	// Stack at this point:
	// 0 => runtime.Caller
	// 1 => (*Parser).traceCurrent
	// 2 => actual parser method, e.g. parseExpr
	pc, _, _, ok := runtime.Caller(2)
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			name = shortFuncName(fn.Name())
		}
	}

	return p.trace(name)
}

func shortFuncName(full string) string {
	// Example:
	// "github.com/me/folang/parser.(*Parser).parseExpr"
	// -> "parseExpr"

	if i := strings.LastIndex(full, "."); i >= 0 && i+1 < len(full) {
		return full[i+1:]
	}
	return full
}
