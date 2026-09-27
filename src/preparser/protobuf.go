package preparser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
)

type SerializableLibComponents struct {
	SurfaceSymbols *symboltable.FolangSymbols
	Ast            ast.SET
	FolangSymbols  *symboltable.FolangSymbols
}

func Deserialize(filename string) (SerializableLibComponents, string) {

	return SerializableLibComponents{}, "error"
}

func Serialize(symbols *symboltable.FolangSymbols, ast ast.SET, surfaceSymbols *symboltable.FolangSymbols, filename string) {

}
