package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_module_stmt parses module declarations in two forms:
//
// Spec form (preferred):
//
//	@co.dap.module(signature=EmployeeModule)
//	EmployeeModImpl co.lang.module ->(signature=EmployeeModule,matches=EmployeeModule) = {
//	    ...
//	}
//
// Entry: current token is the module name IDENTIFIER.
func parse_module_stmt(p *parser, stmtK StmtKind, alias bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	body := make([]ast.Stmt, 0)
	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	def := p.currentToken()
	if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.module"); ok {
		p.advance()
	} else {
		err_ = p.errorExpection("co.lang.module expected but found "+def.Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}

	p.module = true

	// Feature example:
	//   EmployeeModImpl co.lang.module->(signature=EmployeeModule, matches=EmployeeModule) = { ... }
	//
	// The arrow metadata binds a module implementation to the signature it
	// matches and, when present, also repeats the public signature name. We
	// parse these fields here before entering the module body so the surrounding
	// symbol and context setup has the complete declaration metadata available.
	matchesSig := ""
	signatureName := ""
	if p.currentTokenKind() == scanlex.ARROW {
		p.advance() // eat '->'
		if p.currentTokenKind() == scanlex.OPEN_PAREN {
			p.advance() // eat '('
			for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
				tk := p.advance()
				key := strings.TrimSuffix(tk.Value, "_fo")
				if p.currentTokenKind() == scanlex.ASSIGNMENT {
					p.advance() // eat '='
					valTk := p.advance()
					val := strings.TrimSuffix(valTk.Value, "_fo")
					if key == "matches" {
						matchesSig = val
					} else if key == "signature" {
						signatureName = val
					}
				}
				if p.currentTokenKind() == scanlex.COMMA {
					p.advance()
				}
			}
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
		}
	}

	// extract signature from @co.dap.module(signature=X) annotation
	if signatureName == "" {
		signatureName = extractModuleAnnotationParam(ddaps)
	}

	// also try extracting sig name from @co.dap.modulesig annotation (legacy)
	if matchesSig == "" {
		matchesSig = extractModuleSigParam(ddaps)
	}

	// if matches is still empty but we have a signature name, use that
	if matchesSig == "" && signatureName != "" {
		matchesSig = signatureName
	}

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	currCtxId := p.Context_
	currSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(currCtxId.Id, string(symboltable.S_ModuleSymbol))
	currCtxId.ChildCtxIds = append(currCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ContextType_ = string(symboltable.S_ModuleSymbol)
	p.Context_.ParentCtxSymbolTableId = currSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	for p.hasTokens() && p.currentTokenKind() != scanlex.EOF && p.currentTokenKind() != scanlex.CLOSE_CURLY && !p.errorLimitReached() {
		prev := p.currentToken()
		tr := parse_type_kind_block(p, MODULE, ddaps)
		if tr.Node != nil {
			body = append(body, tr.Node.(ast.Stmt))
		} else if !p.madeProgress(prev) {
			p.advance()
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)

	msym := symboltable.ModuleSymbol{
		Name: name.Value,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_ModuleSymbol),
		},
	}
	mst := ast.ModuleStmt{
		Body:          body,
		MatchesSig:    matchesSig,
		SignatureName: signatureName,
		Symb:          &msym,
	}
	mst.SetDap(ddaps)
	pr.Node = mst

	p.Context_ = currCtxId
	p.SymbolTable_ = currSymbId

	updateContext(p, pr.Node, false, false)
	return pr
}

// extractModuleAnnotationParam extracts the signature name from
// @co.dap.module(signature=X) stored in ddaps.
func extractModuleAnnotationParam(ddaps map[scanlex.DirectiveKind][]ast.Stmt) string {
	dir, ok := getAnn(ddaps, "@co.dap.module")
	if !ok {
		return ""
	}
	if v, ok := dir.Parameters["signature"].([]any); ok && len(v) > 0 {
		return fmt.Sprint(v[0])
	}
	// try positional: first value in the parameters map
	for _, v := range dir.Parameters {
		if vals, ok := v.([]any); ok && len(vals) > 0 {
			return fmt.Sprint(vals[0])
		}
	}
	return ""
}

// extractModuleSigParam extracts the signature name from
// @co.dap.modulesig(sig=name) stored in ddaps.
func extractModuleSigParam(ddaps map[scanlex.DirectiveKind][]ast.Stmt) string {
	dir, ok := getAnn(ddaps, "@co.dap.modulesig")
	if !ok {
		return ""
	}
	if v, ok := dir.Parameters["sig"].([]any); ok && len(v) > 0 {
		return fmt.Sprint(v[0])
	}
	// try positional: first value in the parameters map
	for _, v := range dir.Parameters {
		if vals, ok := v.([]any); ok && len(vals) > 0 {
			return fmt.Sprint(vals[0])
		}
	}
	return ""
}
