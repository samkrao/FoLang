package preparser

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

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

func importStdLibraries(symbols *symboltable.FolangSymbols, installDir string) {
	path := filepath.Join(installDir, "atdlib", "co.folenc")
	source, status := Deserialize(path)
	if status == "error" {
		return
	}
	updateFolangSymbols(source, "co", symbols)

}

func importLibraries(symbols *symboltable.FolangSymbols, projectDir string) {

	path := filepath.Join(projectDir, "lib")
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	filenames := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".folenc" {

			fullPath := filepath.Join(path, entry.Name())
			filenames = append(filenames, fullPath)
		}
	}
	for _, path := range filenames {
		source, status := Deserialize(path)
		name := filepath.Base(path)
		ext := filepath.Ext(name)
		primary := strings.TrimSuffix(name, ext)
		if status == "error" {
			continue
		}
		updateFolangSymbols(source, primary, symbols)

	}
}

type Kind string

const (
	Packaged    Kind = "packaged"
	Native      Kind = "native"
	Dynamicvmrt Kind = "dynamicvmrt"
	Operators   Kind = "operators"
	Application Kind = "application"
)

func parseAndImportComponent(symbols *symboltable.FolangSymbols, projectDir string, installDir string, kind_ Kind) {

	path := filepath.Join(projectDir, "components", string(kind_), "component.fol")
	source, status := fetchSourceByFile(path)
	if status == "error" {
		return
	}
	stream := scanlex.NewTokenStream(source, path, &symbols.Operators)

	switch kind_ {
	case Operators:
		parser := Init(projectDir, installDir, symbols, stream)
		parser.ParseOperators()
	case Packaged:
		parser := Init(projectDir, installDir, symbols, stream)
		parser.ParsePackaged()

	default:
		parser := Init(projectDir, installDir, symbols, stream)
		parser.ParseComponents()

	}
}

/*
	    importStd Library installDir/stdlib/co.folenc

		import project-dir/lib/.folenc symbols

		parse project-dir/components/<kind>/ symbols
*/
func PreParse(installDir string, projectDir string) *symboltable.FolangSymbols {
	symbols := &symboltable.FolangSymbols{}
	symbols.CreateFolangSymbols()

	importStdLibraries(symbols, installDir)
	importLibraries(symbols, projectDir)
	parseAndImportComponent(symbols, projectDir, installDir, Operators)
	parseAndImportComponent(symbols, projectDir, installDir, Packaged)
	parseAndImportComponent(symbols, projectDir, installDir, Application)
	parseAndImportComponent(symbols, projectDir, installDir, Native)
	parseAndImportComponent(symbols, projectDir, installDir, Dynamicvmrt)
	return symbols
}

func validateFolder(path string) (error, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return err, false
	}

	if !info.IsDir() {
		return nil, false
	}

	return nil, true
}

func fetchSourceByFile(filePath string) ([]byte, string) {
	sourceBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "error"
	}
	return normalizeSourceBytes(sourceBytes), "success"
}
func FetchSource(projectRoot string) (string, []byte, string) {

	if _, k := validateFolder(projectRoot); !k {
		return "", nil, "projectrootfolder"
	}

	folderPath := filepath.Join(projectRoot, "src")

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
	return filenames[0], normalizeSourceBytes(sourceBytes), "success"
}
