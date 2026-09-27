package preparser

import (
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
	info, err := os.Stat(folderPath)
	if err != nil {
		return "", nil, "fileread"
	}
	if !info.IsDir() {
		if filepath.Ext(folderPath) != ".fol" {
			return "", nil, "fileread"
		}
		sourceBytes, err := os.ReadFile(folderPath)
		if err != nil {
			return "", nil, "fileread"
		}
		return folderPath, sourceBytes, "success"
	}

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return "", nil, "fileread"
	}
	filenames := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".fol" {
			fullPath := filepath.Join(folderPath, entry.Name())
			filenames = append(filenames, fullPath)
		}
	}
	if len(filenames) == 0 {
		return "", nil, "fileread"
	}
	if len(filenames) > 1 {
		return "", nil, "more"
	}
	base := filepath.Base(filenames[0])
	if !slices.Contains([]string{"appl.fol", "component.fol"}, base) {
		return "", nil, "both"
	}
	sourceBytes, err := os.ReadFile(filenames[0])
	if err != nil {
		return "", nil, "fileread"
	}
	return filenames[0], sourceBytes, "success"
}
