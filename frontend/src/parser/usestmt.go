package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_use_directive_stmt parses an @co.dap.use annotation directive:
//
//	@co.dap.use(extensions=[equals, upperCase])
//	k.upperCase();  // explicitly activated
//
// The annotation parameters are extracted from ddaps and stored
// in UseStmtDirective.Type as a map of key -> []string values.
// This node does not consume any code tokens; the annotation
// has already been collected by the annotation collector.
func parse_use_directive_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}

	dir, ok := getAnn(ddaps, "@co.dap.use")
	if !ok {
		pr.Node = ast.UseStmtDirective{}
		return pr
	}

	// Extract all parameters into a string map.
	// Parameters come as map[string]any where values are []any from parse_ddaps.
	// Array brackets (e.g. "[equals,upperCase]") are stripped.
	typeMap := make(map[string][]string)
	for key, val := range dir.Parameters {
		switch arr := val.(type) {
		case []any:
			strs := make([]string, 0, len(arr))
			for _, v := range arr {
				s := strings.Trim(fmt.Sprint(v), "[]")
				if s != "" {
					strs = append(strs, s)
				}
			}
			typeMap[key] = strs
		case []string:
			typeMap[key] = arr
		}
	}

	nd := ast.UseStmtDirective{
		Name: "use",
		Type: typeMap,
		Symb: &symboltable.UseSymbol{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_UseSymbol),
			},
		},
	}
	nd.SetDap(ddaps)
	pr.Node = nd

	return pr
}
