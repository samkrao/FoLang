// Package main is the entry point for the fo-lang compiler frontend.
package main

import (
	"fmt"
	"os"

	"github.com/akamensky/argparse"
	"github.com/samkrao/fo-lang/src/preparser"
	"github.com/samkrao/fo-lang/src/scanlex"
)

func main() {

	parser := argparse.NewParser("fo-frontend", "fo-lang frontend")
	help := parser.Flag("h", "help", &argparse.Options{Required: false, Help: "Show help"})

	var binary *bool = new(bool)
	*binary = false

	var fname *string = new(string)
	*fname = ""

	var tokenizeonly *bool = new(bool)
	*binary = false

	binary = parser.Flag("b", "Binary", &argparse.Options{Required: false, Help: "Generate wire format or not"})
	fname = parser.String("f", "filename", &argparse.Options{Required: false, Help: "File name"})
	tokenizeonly = parser.Flag("t", "tokonly", &argparse.Options{Required: false, Help: "tokenize only"})

	args := os.Args
	err := parser.Parse(args)

	if err != nil || len(args) < 2 {
		fmt.Print(parser.Usage(err))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *help {
		fmt.Print(parser.Usage("usage"))
		os.Exit(1)

	}

	file, source, status := preparser.FetchSource(*fname)

	if status == "success" {

		installDir, err := preparser.InstalledStandardArtifactPath()
		if err != nil {
			fmt.Println("Install folder not found!")
			os.Exit(1)
		}

		symbols := preparser.PreParse(installDir, *fname)

		stream := scanlex.NewTokenStream(source, file)

		if *tokenizeonly {
			for {
				token := stream.Next()
				token.Println()
				if token.Kind == scanlex.EOF {
					break
				}
			}
			return
		}
		parser := preparser.Init(*fname, installDir, symbols, stream)
		ast := parser.Parse()

		fmt.Println(ast)
	} else {
		if status == "more" {
			fmt.Print("More than on fol file present ")
		} else if status == "both" {
			fmt.Print("Project should be either Library or Application cannot be both ")
		} else if status == "fileread" {
			fmt.Print("FileRead Error")
		} else {
			fmt.Print("Unknown Error")
		}
	}
}
