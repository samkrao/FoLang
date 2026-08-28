package ast

import (
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// TypeclassStmt is the shared body for every typeclass declaration.
// Specific wrappers (FunctorStmt, MonadStmt, etc.) embed this type.
//
// Parsed from syntax like:
//
//	@co.dap.Functor
//	Functor(F) = {
//	    map(value F(A), f (A)->B) -> (F(B));
//	}
type TypeclassStmt struct {
	Span
	NodeName string
	Name     string // typeclass name, e.g. "Functor"
	// TypeParams keeps each complete generic parameter, including higher-kinded
	// arity such as F(_), rather than retaining only its display name.
	TypeParams []symboltable.GenericTypeParam
	Methods    []Stmt // method signatures (FunctionDeclarationStmt with IsBody=false)
	Kind       string // annotation kind: "functor", "monad", "applicative", etc.
	SDapst     Stmt
	Symb       *symboltable.TypeclassSymbol
}

func (n TypeclassStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TypeclassStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}
func (t TypeclassStmt) stmt() {}

// SetDap attaches directive annotations to the node.
func (b TypeclassStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// Visit converts the AST node to a MIR node.
func (n TypeclassStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// TypeclassInstanceStmt represents a typeclass instance.
//
// Parsed from syntax like:
//
//	ListFunctor co.lang.instance->(for=Functor, type=List) = {
//	    map(value List(A), f (A)->B) -> (List(B)) = { ... }
//	}
type TypeclassInstanceStmt struct {
	Span
	NodeName      string
	TypeclassName string   // e.g. "Functor" (from for=...)
	ForType       string   // e.g. "List" (from type=...)
	TypeArgs      []string // optional extra type args (e.g. ["E"] for Result(A,E))
	TypeParams    []symboltable.GenericTypeParam
	Body          []Stmt // method implementations
	SDapst        Stmt
	Symb          *symboltable.InstanceSymbol
}

func (n TypeclassInstanceStmt) GetName() string {
	return n.Symb.GetName()
}
func (n TypeclassInstanceStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (t TypeclassInstanceStmt) stmt() {}

// Visit converts the AST node to a MIR node.
func (n TypeclassInstanceStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// SetDap attaches directive annotations to the node.
func (b TypeclassInstanceStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}

// MatcherInstanceStmt represents a matcher instance declaration.
//
// Parsed from syntax like:
//
//	PositiveEvenMatcher co.lang.matcher->(for=Matcher, type=co.lang.int) = {
//	    matchCase(value co.lang.int, pat co.lang.untyped)->(co.lang.int, co.lang.MatchBindings) = { ... }
//	}
type MatcherInstanceStmt struct {
	Span
	NodeName    string
	MatcherName string // e.g. "Matcher" (from for=...)
	ForType     string // e.g. "co.lang.int" (from type=...)
	TypeParams  []symboltable.GenericTypeParam
	Body        []Stmt // method implementations (matchCase etc.)
	SDapst      Stmt
	Symb        *symboltable.MatcherImplSymbol
}

func (n MatcherInstanceStmt) GetName() string {
	return n.Symb.GetName()
}
func (n MatcherInstanceStmt) GetSymbolType() string {
	return string(n.Symb.GetSymbolType())
}

func (m MatcherInstanceStmt) stmt() {}

// Visit converts the AST node to a MIR node.
// Produces a MatcherDeclNode wrapping a FunctionDeclarationNode built from the body.
func (m MatcherInstanceStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// SetDap attaches directive annotations to the node.
func (b MatcherInstanceStmt) SetDap(daps map[scanlex.DirectiveKind][]Stmt) {
	if b.SDapst == nil {
		(&b).SDapst = &DirectveList{NodeName: "DirectveList"}
	}
	b.SDapst.(*DirectveList).SetDap(daps)
}
