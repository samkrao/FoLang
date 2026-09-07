package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

func TestTypeclassShapeComesFromBuiltinAnnotation(t *testing.T) {
	result := ParseFile(`@co.dap.typeclass(kind=Functor, shape=(F(_)), aliases=[
    {name=Mapper, type=(A)->(B)},
    {name=InputContainer, type=F(A)},
    {name=ResultContainer, type=F(B)}
])
_ co.lang.typeclass = {
    map(value InputContainer, f Mapper) -> (ResultContainer);
}`, "test", ".", "Functor.fol", "tc")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("annotated typeclass produced diagnostics: %v", result.Diagnostics)
	}
	packageRoot, ok := result.Root.(ast.PackageStmt)
	if !ok || len(packageRoot.Body) != 1 {
		t.Fatalf("root = %#v, want one-declaration package", result.Root)
	}
	declaration, ok := packageRoot.Body[0].(ast.TypeclassStmt)
	if !ok {
		t.Fatalf("root = %T, want ast.TypeclassStmt", result.Root)
	}
	if len(declaration.TypeParams) != 1 || logicalName(declaration.TypeParams[0].Name) != "F" || declaration.TypeParams[0].Types != "_" {
		t.Fatalf("typeclass shape = %#v, want unary F(_)", declaration.TypeParams)
	}
}

func TestTypeclassDeclarationHeadShapeIsRejected(t *testing.T) {
	result := ParseFile(`@co.dap.typeclass(kind=Functor, shape=(F(_)))
_ (F(_)) co.lang.typeclass = {
    map(value F(A), f Mapper) -> (F(B));
}`, "test", ".", "Functor.fol", "tc")
	if len(result.Diagnostics) == 0 || !strings.Contains(result.Diagnostics[0].Error(), "declaration-head type parameters are not allowed") {
		t.Fatalf("obsolete declaration head diagnostics = %v", result.Diagnostics)
	}
}
