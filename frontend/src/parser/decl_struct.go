package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
)

// struct-declaration and cstruct-declaration — section 6.
//
//	struct-declaration  = annotations, declaration-name,
//	                      [ generic-parameter-clause ], "co.lang.struct", "=",
//	                      struct-body
//	struct-body         = "{", { struct-member }, body-close
//	cstruct-declaration = annotations, declaration-name,
//	                      [ generic-parameter-clause ], "co.lang.cstruct", "=",
//	                      cstruct-body
//	cstruct-body        = "{", { field-declaration }, body-close
//
// A struct is FoLang's ordinary record type and may embed other types. A cstruct is a
// C-layout value type used at a zone boundary, so its body admits only named fields —
// no embedding, which would change the layout.
//
// Both bodies end at their closing brace and take NO trailing semicolon
// (DECISION-SYN-006, the body-brace rule):
//
//	Employee co.lang.struct = { id co.lang.int; }

// parseStructDeclaration parses the struct-declaration production.
//
// declName, generics and annotations have already been read by the primary-declaration
// dispatcher, which needed them to identify the kind.
func (p *parser) parseStructDeclaration(declName name, generics []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a struct body")
	members := p.parseBracedBody("a struct body", p.parseStructMember)

	symb := p.structSymbol(declName.Scanned)
	symb.Embedded = hasEmbeddedField(members)
	applyTypeVisibility(&symb.SymbolDetails, annotations)
	symb.IsSealed = annotations.has("@co.dap.sealed")

	return ast.TypeDeclarationStmt{
		Name:       declName.Scanned,
		TypeParams: generics,
		Body:       members,
		Kind:       "co.lang.struct",
		SubType_:   "STRUCT",
		Typetype:   "UDT",
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
}

// parseCStructDeclaration parses the cstruct-declaration production.
//
// Only named fields are admitted, so an embedded field here is reported rather than
// silently accepted: a C-layout type cannot compose another type without changing its
// layout.
func (p *parser) parseCStructDeclaration(declName name, generics []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a cstruct body")

	members := p.parseBracedBody("a cstruct body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		p.rejectOperatorPlacement(memberAnnotations, "a cstruct field")
		if p.atEmbeddedField() {
			p.report(p.cur(), "a cstruct body admits only named fields; an embedded type would change the C layout")
		}
		return p.parsePureFieldDeclaration(memberAnnotations, "cstruct")
	})

	symb := p.structSymbol(declName.Scanned)
	symb.CStruct = true
	applyTypeVisibility(&symb.SymbolDetails, annotations)
	symb.IsSealed = annotations.has("@co.dap.sealed")

	return ast.TypeDeclarationStmt{
		TypeParams: generics,
		Name:       declName.Scanned,
		Body:       members,
		Kind:       "co.lang.cstruct",
		SubType_:   "CSTRUCT",
		Typetype:   "UDT",
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
}

// hasEmbeddedField reports whether any member is an embedded field, which is what makes
// a struct composed rather than plain.
//
// An embedded field is recorded with its type's name as its identifier and no
// initializer, which is how it is recognised here.
func hasEmbeddedField(members []ast.Stmt) bool {
	for _, m := range members {
		if d, ok := m.(ast.VarDeclarationStmt); ok {
			if d.AssignedValue == nil && d.Identifier == d.VarType {
				return true
			}
		}
	}
	return false
}

// applyTypeVisibility records a declaration's visibility annotations on its symbol.
//
// FoLang spells visibility with annotations rather than with keywords, so
// @co.dap.public, @co.dap.private and @co.dap.package are what set it.
func applyTypeVisibility(details *symboltable.SymbolDetails, annotations annotationSet) {
	// SymbolDetails itself carries no visibility fields; recording the annotation
	// names keeps them available to the semantic phase, which owns the visibility
	// domain rules.
	if annotations.has("@co.dap.internal") {
		details.IsInternal_ = true
	}
}
