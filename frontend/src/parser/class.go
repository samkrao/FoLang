package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// extractGenericTypeParams reads type parameters from a @co.dap.generic annotation.
//
// Supported annotation forms:
//
//	@co.dap.generic(typename=T)
//	@co.dap.generic(type={T:{variance:invariant,bound=Number,Kind:Param}, R:{...}})
func extractGenericTypeParams(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []symboltable.GenericTypeParam {
	defer p.traceCurrent()()

	ann, ok := getAnn(ddaps, "@co.dap.generic")
	if !ok {
		return nil
	}

	params := make([]symboltable.GenericTypeParam, 0)

	// Single typename form: @co.dap.generic(typename=T)
	if rawName, exists := ann.Parameters["typename"]; exists && rawName != nil {
		if name, ok := rawName.(string); ok && name != "" {
			params = append(params, symboltable.GenericTypeParam{Name: name})
			return params
		}
	}

	// Multi-type form: @co.dap.generic(type={T:{...}, R:{...}})
	if rawType, exists := ann.Parameters["type"]; exists && rawType != nil {
		if typeMap, ok := rawType.(map[string]interface{}); ok {
			for paramName, paramSpec := range typeMap {
				gtp := symboltable.GenericTypeParam{Name: paramName}
				if spec, ok := paramSpec.(map[string]interface{}); ok {
					if variance, ok := spec["variance"].(string); ok {
						gtp.Variance = variance
					}
					if bound, ok := spec["bound"].(string); ok {
						gtp.Constraint = bound
					}
					if kind, ok := spec["kind"].(string); ok {
						gtp.Kind_ = kind
					} else if kind, ok := spec["Kind"].(string); ok {
						gtp.Kind_ = kind
					}
					if def, ok := spec["default"].(string); ok {
						gtp.Default = def
					}
					if nullable, ok := spec["nullable"].(bool); ok {
						gtp.Nullable = nullable
					}
					if inclusive, ok := spec["inclusive"].(bool); ok {
						gtp.Inclusive = inclusive
					}
					if impredicative, ok := spec["impredicative"].(bool); ok {
						gtp.Impredicative = impredicative
					}
					if typekind, ok := spec["typekind"].(string); ok {
						gtp.TypeKind = typekind
					}
					if types, ok := spec["types"].(string); ok {
						gtp.Types = types
					}
				} else if specStr, ok := paramSpec.(string); ok {
					// e.g. T:{typename}
					gtp.Constraint = specStr
				}
				params = append(params, gtp)
			}
		}
	}

	return params
}

// parseClassArrowClause parses the optional `->( key=val, key=[a,b], ... )`
// clause that follows `co.lang.class`.
//
// Feature example:
//
//	Employee co.lang.class ->(implements=[IEmployee], extends=[BaseEmployee]) = { ... }
//
// The arrow clause carries relationship metadata that shapes the class symbol
// but does not itself enter the class body scope.
func parseClassArrowClause(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (
	implements, extends, inherits, uses, composes []string, with_ string,
) {
	if p.currentTokenKind() != scanlex.ARROW {
		return
	}
	p.advance() // eat ->

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
		key_t, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
		p.addErr(err_)
		key := key_t.Value

		_, err_ = p.expect(scanlex.ASSIGNMENT)
		p.addErr(err_)

		var vals []string
		if p.currentTokenKind() == scanlex.OPEN_BRACKET {
			p.advance() // eat [
			for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_BRACKET && p.currentTokenKind() != scanlex.EOF {
				id_t, _ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
				if id_t.Value != "" {
					vals = append(vals, id_t.Value)
				}
				if p.currentTokenKind() == scanlex.COMMA {
					p.advance()
				}
			}
			_, err_ = p.expect(scanlex.CLOSE_BRACKET)
			p.addErr(err_)
		} else {
			id_t, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
			p.addErr(err_)
			vals = []string{id_t.Value}
		}

		switch key {
		case "implements":
			implements = vals
		case "extends":
			extends = vals
		case "inherits":
			inherits = vals
		case "uses":
			uses = vals
		case "composes":
			composes = vals
		case "with":
			if len(vals) > 0 {
				with_ = vals[0]
			}
		}

		if p.currentTokenKind() == scanlex.COMMA {
			p.advance()
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	return
}

// parseOopsAnnotation extracts class relationship lists from @co.dap.oops(...).
//
// Feature example:
//
//	@co.dap.oops(implements=[IEmployee], inherits=[BaseEmployee])
func parseOopsAnnotation(ddaps map[scanlex.DirectiveKind][]ast.Stmt) (
	inherits, implements, traits, mixins, uses, composes, extends []string,
) {
	ann, ok := getAnn(ddaps, "@co.dap.oops")
	if !ok {
		return
	}

	extractList := func(key string) []string {
		rawVal, exists := ann.Parameters[key]
		if !exists || rawVal == nil {
			return nil
		}
		switch v := rawVal.(type) {
		case []interface{}:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		case []string:
			return v
		case string:
			if v != "" {
				return []string{v}
			}
		}
		return nil
	}

	inherits = extractList("inherits")
	implements = extractList("implements")
	traits = extractList("traits")
	mixins = extractList("mixins")
	uses = extractList("uses")
	composes = extractList("composes")
	extends = extractList("extends")
	return
}

// parseClassBody parses the `{ ... }` body of a class.
//
// Feature examples:
//
//	Employee co.lang.class = {
//	    Id co.lang.int;
//
//	    @co.dap.method.instance
//	    getId()->(co.lang.int) = { this.return self.Id; }
//
//	    @@init(id co.lang.int) = { self.Id = id; }
//	}
//
// The body reuses parse_type_kind_block so fields, methods, inner types, and
// special lifecycle methods all flow through the same declaration dispatcher.
func parseClassBody(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []ast.Stmt {
	body := make([]ast.Stmt, 0)

	_, err_ := p.expect(scanlex.OPEN_CURLY)
	p.addErr(err_)

	for p.hasTokens() && p.currentTokenKind() != scanlex.EOF && p.currentTokenKind() != scanlex.CLOSE_CURLY {
		// Collect per-member annotations/directives so each class member sees only
		// the hints that syntactically belong to it.
		dddaps := map[scanlex.DirectiveKind][]ast.Stmt{
			scanlex.ANNOTATION: collect_hints(p, scanlex.ANNOTATION, ddaps),
			scanlex.DECORATOR:  collect_hints(p, scanlex.DECORATOR, ddaps),
		}

		prev := p.currentToken()
		var tr ParseResult
		if p.currentTokenKind() == scanlex.DOUBLE_AT {
			tr = parse_special_method_stmt(p, dddaps)
		} else {
			tr = parse_type_kind_block(p, CLASS, dddaps)
		}
		p.advanceIfStuck(prev)

		if tr.Node == nil {
			continue
		}
		if st, ok := tr.Node.(ast.Stmt); ok {
			body = append(body, st)
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err_)
	return body
}

// parse_class_declaration_stmt parses a full class declaration.
//
//	Employee co.lang.class ={ ... }
//	Employee co.lang.class ->(implements=[IEmployee]) ={ ... }
//
//	@co.dap.generic(type={T:{typename},R:{typename}})
//	Employee co.lang.class ={ id T; name R; ... }
func parse_class_declaration_stmt(p *parser, stmtK StmtKind, alias bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	// 1. Read class name.
	className_t, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	className := className_t.Value

	// 2. Consume `co.lang.class`.
	if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.class"); ok {
		p.advance()
	} else {
		err_ = p.errorExpection("Expected co.lang.class but found "+p.currentToken().Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}

	// Feature example:
	//   @co.dap.generic(type={T:{typename}})
	//   Box co.lang.class = { value T; }
	//
	// Generic metadata is attached before we parse the body so member types can
	// resolve type parameters consistently.
	typeParams := extractGenericTypeParams(p, ddaps)

	// 4. Extract class relationships from @co.dap.oops annotation.
	oopsInherits, oopsImplements, oopsTraits, oopsMixins, oopsUses, oopsComposes, oopsExtensions :=
		parseOopsAnnotation(ddaps)

	// 5. Parse optional `->( ... )` clause for inline relationship declarations.
	arrowImpl, arrowExt, arrowInh, arrowUses, arrowComposes, with_ :=
		parseClassArrowClause(p, ddaps)

	// Merge annotation-sourced and arrow-clause-sourced relationship lists.
	inherits := append(oopsInherits, arrowInh...)
	implements := append(oopsImplements, arrowImpl...)
	extensions := append(oopsExtensions, arrowExt...)
	uses := append(oopsUses, arrowUses...)
	composes := append(oopsComposes, arrowComposes...)

	// 6. Expect `=`.
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// 7. Create a new context / symbol table for the class scope.
	actCtxId := p.Context_
	actSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_ClassSymbol))
	p.Context_.ChildCtxIds = append(p.Context_.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = actSymbId.Id
	p.Context_.ContextType_ = string(symboltable.S_ClassSymbol)
	//Registering new context and symbol table in FolangSymbols
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	// 8. Parse class body.
	body := parseClassBody(p, ddaps)

	// 9. Restore outer context.
	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	symb := symboltable.ClassSymbol{
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_ClassSymbol),
		},
		Inherits:         inherits,
		Extensions:       extensions,
		Traits:           oopsTraits,
		Mixin:            oopsMixins,
		Uses:             uses,
		ComposeAssociate: composes,
		Implements:       implements,
		With:             with_,
		IsGeneric:        len(typeParams) > 0,
	}

	cls := ast.ClassDeclarationStmt{
		Name:       className,
		Body:       body,
		TypeParams: typeParams,
		Symb:       &symb,
	}
	pr.Node = cls

	updateContext(p, pr.Node, false, false)
	return pr
}
