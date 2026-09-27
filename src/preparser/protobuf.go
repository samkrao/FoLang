package preparser

import symboltable "github.com/samkrao/fo-lang/src/context"

func Deserialize(filename string) (symboltable.FolangSymbols, string) {

	return symboltable.FolangSymbols{}, "error"
}

func Serialize(symbols *symboltable.FolangSymbols, filename string) {

}
