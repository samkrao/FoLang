package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_extension_declaration parses a function annotated with @co.dap.extension.
//
//	@co.dap.extension(fortype=co.lang.string, what=extends)
//	upperCase()->(string) = {
//	    return this.upper()
//	}
//
// Entry: current token is the function name IDENTIFIER (or receiver '('.
func parse_extension_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	tr := parse_fn_declaration(p, FUNCTION, ddaps)
	if tr.Node == nil {
		return ParseResult{}
	}
	fn := *(tr.Node.(*ast.FunctionDeclarationStmt))
	fn.Symb.SymbolDetails.SymbolType_ = string(symboltable.S_ExtensionSymbol)
	est := ast.ExtensionStmt{}
	est.FunctionDeclarationStmt = fn

	// Extract fortype= and what= from the @co.dap.extension annotation.
	if dir, ok := getAnn(ddaps, "@co.dap.extension"); ok {
		est.ForType = extractExtField(dir.Parameters, "fortype")
		est.What = extractExtField(dir.Parameters, "what")
	}

	return ParseResult{Node: &est, Errors: tr.Errors}
}

// extractExtField pulls the first string value from an extension annotation
// parameter, trimming surrounding quotes and the internal "_fo" suffix.
func extractExtField(params map[string]any, key string) string {
	arr, ok := params[key].([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	s := strings.TrimSuffix(strings.Trim(strings.TrimSpace(strings.ReplaceAll(
		strings.Trim(arr[0].(string), "\""), "[", "")), "]"), "_fo")
	return s
}
