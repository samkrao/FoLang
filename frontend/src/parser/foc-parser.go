package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	paast "github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// go get -u github.com/sanity-io/litter

// Focmain is the main entry point that reads, parses, and optionally serializes a FoLang source file.
func Focmain(fname string, binary bool, singleton bool, stopAt string, toast bool, rootDir string) (string, string, string, bool, error) {
	sourceFile := fname

	fmt.Printf("source %s\n", sourceFile)
	sourceFile, _ = filepath.Abs(sourceFile)
	sourceBytes, errr := os.ReadFile(sourceFile)
	basename := filepath.Base(sourceFile)
	filename := strings.TrimSuffix(basename, filepath.Ext(basename))

	//fileDir := filepath.Dir(sourceFile)

	if errr != nil {
		fmt.Println(errr)
		return filename, "", "", false, errr
	}
	source := string(sourceBytes)
	start := time.Now()
	parse := true
	if stopAt == "Tokens" {
		parse = false
	}

	// Derive the package path from the file's directory relative to the project
	// root (current working directory). Per the spec every subfolder is a package;
	// the root itself is not a package so root files get an empty package path.
	packagePath := ""
	absFile, absErr := filepath.Abs(sourceFile)
	baseDir := ""
	if absErr == nil {
		cwd, cwdErr := os.Getwd()
		baseDir = cwd
		// compiler can be invoked from any directory,
		// so we need to check if the file directory is different from the current working directory. If it is, we update cwd to the file's directory.
		if cwd != rootDir && rootDir != "" {
			baseDir = rootDir
		}
		if cwdErr == nil {
			dir := filepath.Dir(absFile)
			leafDir := filepath.Base(dir)
			rel, relErr := filepath.Rel(baseDir, dir)
			if rootDir != "" {
				if relErr == nil && rel != "." {
					// convert OS path separators to dots: hr/employee → hr.employee
					packagePath = strings.ReplaceAll(rel, string(filepath.Separator), ".")
					// strip any leading dots that arise from paths above cwd
					packagePath = strings.TrimPrefix(packagePath, ".")
				}
			} else {
				rootDir = leafDir
			}
		}
	}

	context := "program"
	symbol := "program"
	fmt.Println("========Printing path info started========")
	fmt.Println(rootDir + ":::" + strings.Split(filename, ".")[0] + ":::" + helpers.RsplitOnce(basename, ".")[0] + ":::" + packagePath)
	fmt.Println("========Printed path info completed=====")
	os.Exit(0)
	ast, tokens, ctx, buildLib := Parse(source, rootDir, strings.Split(filename, ".")[0], helpers.RsplitOnce(basename, ".")[0], packagePath, context, symbol, parse)
	duration := time.Since(start)

	if stopAt == "Tokens" {
		t, err := json.Marshal(tokens)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(t[:]))
		//litter.Dump(ast)
		os.Exit(0)
	}
	fmt.Printf("Duration: %v\n", duration)
	//a, _ := json.Marshal(ast)
	//fmt.Println(string(a[:]))
	nd := paast.Treevistor(ast)
	type finalAST_ struct {
		Context *symboltable.Context `json:"SymbolTable"`
		Paast   paast.SET            `json:"AST"`
	}
	finalAST := finalAST_{
		Context: ctx,
		Paast:   nd,
	}
	//b, _ := json.Marshal(nd)
	s := ""
	if binary {

		s = ""
	} else {
		b, _ := helpers.Marshal(finalAST)

		s = string(b[:])
	}

	return filename, "", s, buildLib, nil
}
