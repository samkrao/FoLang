package preparser

import (
	"log"
	"os"
	"path/filepath"
	"slices"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/parser"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func Init(projectDir string, installDir string, symbols *symboltable.FolangSymbols, stream *scanlex.TokenStream) parser.Parser {

	return parser.Parser{
		ProjectDir: projectDir,
		Stream:     stream,
		InstallDir: installDir,
		Symbols:    symbols,
	}
}

func importStdLibraries(installDir string) symboltable.FolangSymbols {
	return symboltable.FolangSymbols{}
}

func importLibraries(projectDir string) symboltable.FolangSymbols {
	return symboltable.FolangSymbols{}
}

type Kind string

const (
	Packaged    Kind = "packaged"
	Native      Kind = "native"
	Dynamicvmrt Kind = "dynamicvmrt"
	Operators   Kind = "operators"
	Application Kind = "application"
)

func parseAndImportComponent(projectDir string, kind_ Kind) symboltable.FolangSymbols {

	return symboltable.FolangSymbols{}
}

/*
	    importStd Library installDir/stdlib/co.folenc

		import project-dir/lib/.folenc symbols

		parse project-dir/components/<kind>/ symbols
*/
func PreParse(installDir string, projectDir string) *symboltable.FolangSymbols {
	importStdLibraries(installDir)
	importLibraries(projectDir)
	parseAndImportComponent(projectDir, Operators)
	parseAndImportComponent(projectDir, Packaged)
	parseAndImportComponent(projectDir, Application)
	parseAndImportComponent(projectDir, Native)
	parseAndImportComponent(projectDir, Dynamicvmrt)
	return &symboltable.FolangSymbols{}
}

func FetchSource(folderPath string) (string, []byte, string) {

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		log.Fatal(err)
	}
	filenames := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".fol" {
			fullPath := filepath.Join(folderPath, entry.Name())
			filenames = append(filenames, fullPath)
		}
	}
	if len(filenames) > 1 {
		return "", nil, "more"
	}
	if !slices.Contains(filenames, "appl.fol") && !slices.Contains(filenames, "component.fol") {
		return "", nil, "both"
	}
	sourceBytes, err := os.ReadFile(filenames[0])
	if err != nil {
		return "", nil, "fileread"
	}
	return filenames[0], sourceBytes, "success"
}
