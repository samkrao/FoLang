package preparser

import (
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/parser"
)

func Init(projectDir string) parser.Parser {

	return parser.Parser{}
}

func importStdLibraries(installDir string) symboltable.FolangSymbols {
	return symboltable.FolangSymbols{}
}

func importLibraries(projectDir string) symboltable.FolangSymbols {
	return symboltable.FolangSymbols{}
}

type Kind string

const (
	Packaged    Kind = "pakcaged"
	Native           = "native"
	Dynamicvmrt      = "dynamicvmrt"
	Operators        = "operators"
	Application      = "application"
)

func parseAndImportComponent(projectDir string, kind_ Kind) symboltable.FolangSymbols {

	return symboltable.FolangSymbols{}
}

/*
	    importStd Library installDir/stdlib/co.folenc

		import project-dir/lib/.folenc symbols

		parse project-dir/components/<kind>/ symbols
*/
func PreParse(installDir string, projectDir string) symboltable.FolangSymbols {

	return symboltable.FolangSymbols{}
}
