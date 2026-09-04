package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// Binding a declaration's name into its symbol table.
//
// scope.go builds the contexts and the visibility segments; this file fills them.
// A segment's SymbolsByName index is what a name lookup searches, so a declaration
// that is never bound is invisible to every later phase however complete its
// symbol record is.
//
// # The table is chosen by the symbol, not by the cursor
//
// Every record carries the id of the segment active where it was minted
// (symbolfactory.go), and binding always targets THAT segment rather than the one
// the cursor happens to be in. The two differ wherever a declaration opens a
// context of its own:
//
//   - a function's name is minted before pushContext, so it binds into the
//     segment that declares it, while its parameters are minted after and bind
//     into the function's own context — which is what B.1 draws;
//   - a struct, a class or a module mints its name AFTER its body has been parsed
//     and its context closed, and still binds into the declaring segment, because
//     the anchor was taken when the record was made.
//
// Nothing here therefore has to reason about which table is meant.
//
// # What is not bound
//
// A name that does not denote a declaration is not a binding: the underscore
// wildcard discards its declarator rather than naming it, an unnamed parameter or
// result has nothing to bind, and an anonymous function or class introduces no
// name into its enclosing scope. Identifier REFERENCES also mint symbol records
// (names.go), and those are uses, not declarations; they are never routed here.
//
// # Speculation
//
// A binding made inside a speculative parse is journalled with its inverse, so a
// branch that is thrown away leaves no name behind. Without that, a rewound parse
// would leave the segment claiming a declaration the accepted parse never made,
// and the next real declaration of that name would be reported as a duplicate.

// declarable is a symbol record that can be bound: it knows its own kind and name
// and the segment it was minted in.
//
// It is declared here rather than added to symboltable.SymbolInfo because binding
// is the parser's concern; every symbol record satisfies it already through the
// SymbolDetails it embeds.
type declarable interface {
	symboltable.SymbolInfo
	Anchor() string
}

// declareGenericAnnotationTypes installs the type names owned by one generic
// declaration in that declaration's scope. Generic markers and aliases are
// ordinary type-name bindings for lookup purposes; Alias and IsGenericType keep
// their different meanings explicit to later compiler stages.
func (p *parser) declareGenericAnnotationTypes(annotations annotationSet) {
	if !annotations.has("@co.dap.generic") {
		return
	}
	if raw, ok := annotations.option("@co.dap.generic", "types"); ok {
		if entries, ok := raw.([]any); ok {
			for _, entry := range entries {
				record, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				marker, ok := record["name"].(string)
				if !ok || marker == "" {
					continue
				}
				sym := p.typeSymbol(marker)
				sym.IsGenericType = true
				p.declareAs(p.cur(), marker, sym)
			}
		}
	}
	for _, alias := range annotations.genericAliases {
		sym := p.typeSymbol(alias.Name)
		sym.Alias = true
		sym.SymbolDetails.Type_ = alias.Written
		p.declareAs(alias.Tok, alias.Name, sym)
	}
}

// declare binds sym under key into the segment sym was minted in, and reports a
// redeclaration when the key is already taken there.
//
// The collision is reported rather than resolved: neither silently keeping the
// first binding nor overwriting it with the second can be right, because one of
// the two declarations would then be missing from the model with nothing said
// about it.
func (p *parser) declare(tok scanlex.Token, key string, sym declarable) {
	table := p.fs.GetSymbolTable(sym.Anchor())
	if table == nil {
		return
	}

	if _, bound := p.fs.Declare(table.Id, key, sym); !bound {
		p.reportNamedf(tok, helpers.DiagnosticDuplicateDeclaration, "Duplicate Declaration", "%s is already declared in this scope", logicalName(sym.GetName()))
		return
	}
	p.journal(func() { p.fs.Undeclare(table.Id, key) })
}

// declareNamed binds a named declaration under its own symbol kind.
//
// A file-backed primary declaration writes its declarator as "_" and takes its
// name from the filename, so the underscore there is a PLACEHOLDER rather than a
// discard and the declaration it introduces is as named as any other. Skipping it
// would leave `Employee.fol` declaring nothing.
func (p *parser) declareNamed(n name, sym declarable) {
	if n.isWildcard() && !n.FromFilename {
		return
	}
	p.declareAs(n.Tok, n.Scanned, sym)
}

// declareQuietly binds a name and says nothing when it is already bound. It is for
// the two cases where a second declaration of one name is not this function's to
// report:
//
//   - a form that is DEFINED to accept an existing name. `?=` defines the name when
//     no visible one exists and reassigns it otherwise, and a function-pattern
//     clause is one of several clauses that together make a single declaration.
//   - a construct whose duplicate rule is owned by a dedicated check that can say
//     more. A repeated variant constructor is reported against the type that owns
//     it, naming both the variant and the type, which is worth more to a reader
//     than a bare collision.
//
// The first binding keeps the segment either way: the record that owns the name is
// the one that introduced it.
func (p *parser) declareQuietly(name string, sym declarable) {
	if name == "" || name == "_" {
		return
	}

	table := p.fs.GetSymbolTable(sym.Anchor())
	if table == nil {
		return
	}

	key := symboltable.SymbolKey(name, sym.GetSymbolType())
	if _, bound := p.fs.Declare(table.Id, key, sym); !bound {
		return
	}
	p.journal(func() { p.fs.Undeclare(table.Id, key) })
}

// declareAs binds a declaration whose name the caller holds as a string, which is
// the shape of the names a parse derives rather than scans: a receiver's binding,
// an embedded field's composed type, a lifecycle member.
func (p *parser) declareAs(tok scanlex.Token, name string, sym declarable) {
	if name == "" || name == "_" {
		return
	}
	p.declare(tok, symboltable.SymbolKey(name, sym.GetSymbolType()), sym)
}

// declareFunction binds a function-like declaration under its signature.
//
// It is called once the parameter lists and the result clause have been read,
// because a function's key includes both; the declaration still lands in the
// segment that declared the NAME, since that is the anchor the record took when
// it was minted before the function's context opened.
func (p *parser) declareFunction(tok scanlex.Token, decl *ast.FunctionDeclarationStmt) {
	if decl.Symb == nil || decl.Name == "" || decl.Name == "_" || decl.Symb.Anonymous {
		return
	}

	table := p.fs.GetSymbolTable(decl.Symb.Anchor())
	if table == nil {
		return
	}

	params := make([]string, 0, len(decl.Parameters))
	for _, list := range decl.Parameters {
		for _, param := range list {
			params = append(params, writtenType(param.SymbolDeclStmt))
		}
	}

	results := make([]string, 0, len(decl.ReturnType))
	for _, result := range decl.ReturnType {
		results = append(results, writtenType(result.SymbolDeclStmt))
	}
	decl.Symb.ReturnSignature = strings.Join(results, ",")
	decl.Symb.OverloadRestriction = overloadRestriction(decl)

	category := callableCategory(decl)
	key := symboltable.FunctionKey(decl.Name, category, params)

	// A matcher's protocol function is counted by the matcher's own rule, which
	// names the matcher and says how many times matchCase was written. Reporting
	// the collision here as well would put the poorer message first.
	if p.isMatcherProtocol(table, decl) {
		if _, bound := p.fs.Declare(table.Id, key, decl.Symb); bound {
			p.journal(func() { p.fs.Undeclare(table.Id, key) })
		}
		p.declareSignatureNames(tok, decl)
		return
	}

	p.checkOverloadFamily(tok, table, symboltable.FunctionFamily(decl.Name, category), decl)

	if _, bound := p.fs.Declare(table.Id, key, decl.Symb); !bound {
		p.reportNamedf(tok, helpers.DiagnosticDuplicateCallableSignature, "Duplicate Callable Signature", "%s is already declared in this scope with the same parameter signature", logicalName(decl.Name))
		return
	}
	p.journal(func() { p.fs.Undeclare(table.Id, key) })

	p.declareSignatureNames(tok, decl)
}

// isMatcherProtocol reports whether a declaration is the matchCase a matcher body
// declares. The owning context is read from the table the name binds into, since
// the function's own context is already open by the time binding happens.
func (p *parser) isMatcherProtocol(table *symboltable.SymbolTable, decl *ast.FunctionDeclarationStmt) bool {
	if logicalName(decl.Name) != matcherProtocolFunction {
		return false
	}
	owner := p.fs.GetContext(table.ContextId)
	return owner != nil && owner.ContextType_ == symboltable.S_MatcherImplSymbol
}

// checkOverloadFamily applies the two rules that hold ACROSS the siblings a name
// already has here, both of which need the family rather than one declaration.
//
// A declaration in one of the non-overloadable categories has no family at all, so
// a second declaration of that identity is invalid however its parameters differ.
// A family that does admit siblings still has one invariant result contract: the
// return type is not an overload discriminator, so a sibling declaring a different
// one is invalid rather than a further overload (docs/language-ref.md,
// "Non-overloadable Function Forms" and "Overload-Family Parameter and Return
// Rules").
//
// Operators are excluded from both. They resolve by normalized operand signature
// under their own rule, which lets distinct operand signatures carry distinct
// result types and which admits the derived-type operands the ordinary categories
// exclude.
func (p *parser) checkOverloadFamily(tok scanlex.Token, table *symboltable.SymbolTable, family string, decl *ast.FunctionDeclarationStmt) {
	if decl.Symb.IsOperator {
		return
	}

	for key, bound := range p.fs.Bindings(table.Id) {
		sibling, isFunction := bound.(*symboltable.FunctionSymbol)
		if !isFunction || !strings.HasPrefix(key, family+"(") {
			continue
		}

		switch {
		case sibling.OverloadRestriction != "":
			p.reportNamedf(tok, helpers.DiagnosticOverloadNotAllowed, "Overload Not Allowed", "%s cannot be overloaded: the declaration already in this scope has %s", logicalName(decl.Name), sibling.OverloadRestriction)
		case decl.Symb.OverloadRestriction != "":
			p.reportNamedf(tok, helpers.DiagnosticOverloadNotAllowed, "Overload Not Allowed", "%s cannot be overloaded: this declaration has %s", logicalName(decl.Name), decl.Symb.OverloadRestriction)
		case sibling.ReturnSignature != decl.Symb.ReturnSignature:
			p.reportNamedf(tok, helpers.DiagnosticDuplicateCallableSignature, "Duplicate Callable Signature", "every declaration of %s must declare the same return signature; a return type never distinguishes two overloads, so this is a conflicting declaration rather than a further one", logicalName(decl.Name))
		default:
			continue
		}
		return
	}
}

// callableCategory is the callable/receiver dimension of a declaration's canonical
// identity: the class-method categories the reference names, and the receiver forms
// a struct companion unit distinguishes. A declaration in another category is not a
// sibling overload merely because its name and its parameters match.
func callableCategory(decl *ast.FunctionDeclarationStmt) string {
	switch symb := decl.Symb; {
	case symb.StaticMethod:
		return "static"
	case symb.ClassMethod:
		return "class"
	case symb.ObjectMethod:
		return "object"
	case symb.InstanceMethod:
		return "instance"
	}

	if decl.AssociatedReceiver == nil {
		return ""
	}
	// A named receiver binds the instance; the bare form gives only the type, which
	// is the type-associated form.
	if symb := declaratorSymbol(decl.AssociatedReceiver.SymbolStmt); symb != nil && symb.Name_ != "" {
		return "value-receiver"
	}
	return "type-receiver"
}

// overloadRestriction names the category that makes a declaration
// non-overloadable, and is empty for one that may have siblings.
//
// The ten categories of docs/language-ref.md, "Non-overloadable Function Forms",
// are all decidable from the signature the parse has just read. They are a
// different set from the callback and execution-model restrictions that
// RestrictedToOverload records, despite that field's name: that set includes
// dynamic and mixed scope, and excludes the return-position and derived-type
// categories here.
func overloadRestriction(decl *ast.FunctionDeclarationStmt) string {
	switch symb := decl.Symb; {
	case symb.NamedParams:
		return "named parameters"
	case symb.OptionalArgs:
		return "optional parameters"
	case symb.DefaultParams:
		return "default parameters"
	case symb.Variadic:
		return "a variadic parameter"
	case symb.Curried:
		return "curried parameter lists"
	case symb.FWPF:
		return "a function type in a parameter position"
	case symb.FWRF:
		return "a function type in a return position"
	case len(decl.ReturnType) > 1:
		return "multiple returns"
	}

	for _, result := range decl.ReturnType {
		if result.IsNamed {
			return "a named return"
		}
	}
	if form := derivedSignatureForm(decl); form != "" {
		return "a " + form + " in its signature"
	}
	return ""
}

// derivedSignatureForm reports the first pointer, address, reference, thunk or
// slice derivation written anywhere in a signature. Array and range derivations are
// deliberately absent: the reference's category names the five indirection forms
// only.
func derivedSignatureForm(decl *ast.FunctionDeclarationStmt) string {
	for _, list := range decl.Parameters {
		for _, param := range list {
			if form := indirectionForm(param.Type_); form != "" {
				return form
			}
		}
	}
	for _, result := range decl.ReturnType {
		if form := indirectionForm(result.Type_); form != "" {
			return form
		}
	}
	return ""
}

// indirectionForm walks a type's derivations, since they nest: a pointer to a slice
// carries the slice as its underlying type.
func indirectionForm(t ast.Type) string {
	derived, isDerived := t.(ast.DerivedType)
	if !isDerived {
		return ""
	}

	switch derived.Form {
	case ast.DerivePointer:
		return "pointer"
	case ast.DeriveAddress:
		return "address"
	case ast.DeriveReference, ast.DeriveHeapReference:
		return "reference"
	case ast.DeriveThunk:
		return "thunk"
	case ast.DeriveSlice:
		return "slice"
	}
	return indirectionForm(derived.Underlying)
}

// declareSignatureNames binds the names a function's signature introduces: its
// parameters and any named result.
//
// A name binds only when it was minted in a segment OTHER than the one holding the
// function's own name, which is precisely the test for the function having opened a
// context. A function-specification has no body and therefore no context, so its
// parameter names are part of a signature rather than declarations; binding them
// would put them in the interface or typeclass body that holds the specification,
// where two specifications sharing a parameter name would collide.
func (p *parser) declareSignatureNames(tok scanlex.Token, decl *ast.FunctionDeclarationStmt) {
	own := decl.Symb.Anchor()

	for _, list := range decl.Parameters {
		for _, param := range list {
			p.declareSignatureName(tok, own, param.Name_, param.SymbolDeclStmt)
		}
	}
	for _, result := range decl.ReturnType {
		if result.IsNamed {
			p.declareSignatureName(tok, own, result.GetName(), result.SymbolDeclStmt)
		}
	}
}

// declareSignatureName binds one signature name, unless it shares the segment that
// holds the function's own name.
func (p *parser) declareSignatureName(tok scanlex.Token, own string, name string, decl ast.SymbolDeclStmt) {
	symb := declaratorSymbol(decl)
	if symb == nil || symb.Anchor() == own {
		return
	}
	p.declareAs(tok, name, symb)
}

// declareDeclarator binds a parameter or a result under the declarator symbol
// declFor minted for it.
//
// That record, rather than the plain Symbol the node also carries, is what an
// ordinary variable lookup expects to find, because it is the one minted as a
// variable.
func (p *parser) declareDeclarator(n name, decl ast.SymbolDeclStmt) {
	if symb := declaratorSymbol(decl); symb != nil {
		p.declareNamed(n, symb)
	}
}

// declareReceiver binds a receiver's name in the function's own context.
//
// A receiver clause is READ before that context opens, because it precedes the name
// the declaration is dispatched on, so its symbol was minted against the enclosing
// segment. The name belongs to the function's scope all the same — the body is what
// uses it — so the anchor is corrected here and the binding follows it. Two methods
// on one type can then each call their receiver `g` without colliding, while a
// parameter that repeats the receiver's name still does.
//
// A function-specification has no context of its own and never reaches this: it has
// no body, so its receiver declares nothing.
func (p *parser) declareReceiver(receiver *ast.FunctionReceiver) {
	if receiver == nil {
		return
	}

	symb := declaratorSymbol(receiver.SymbolStmt)
	if symb == nil {
		return
	}

	symb.SymbolTableId = p.symtab.Id
	p.declareAs(p.cur(), symb.Name_, symb)
}

// declaratorSymbol reaches the symbol record inside a parameter's or a result's
// synthesized declarator, and reports nil for anything else.
func declaratorSymbol(decl ast.SymbolDeclStmt) *symboltable.VarSymbol {
	item, isDeclarator := decl.(ast.VarDeclarationStmt)
	if !isDeclarator {
		return nil
	}
	return item.Symb
}

// writtenType is the type spelling a parameter or a result carries, which is what
// a parse-time signature can be keyed on. GetActType returns the resolved type
// first and the written one second; only the second is populated during the parse.
func writtenType(decl ast.SymbolDeclStmt) string {
	if decl == nil {
		return ""
	}
	resolved, written := decl.GetActType()
	if written != "" {
		return written
	}
	return resolved
}
