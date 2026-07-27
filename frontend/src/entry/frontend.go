package entry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/akamensky/argparse"
	ll "github.com/samkrao/fo-lang/frontend/src/parser"
)

// Run is the entry point for the frontend tool.
// args should be os.Args (program name + flags).
func Run(args []string) {
	parser := argparse.NewParser("fo-frontend", "fo-lang frontend")
	help := parser.Flag("h", "help", &argparse.Options{Required: false, Help: "Show help"})

	var genast *bool = new(bool)
	*genast = true

	var binary *bool = new(bool)
	*binary = false

	var fname *string = new(string)
	*fname = ""

	var stopAt *string = new(string)
	*stopAt = "none"

	binary = parser.Flag("b", "Binary", &argparse.Options{Required: false, Help: "AST as protobuf binary (.pb)"})
	fname = parser.String("f", "filename", &argparse.Options{Required: false, Help: "File name"})
	stopAt = parser.String("p", "stopAt", &argparse.Options{Required: false, Help: "StopAt"})
	toast := parser.Flag("t", "toast", &argparse.Options{Required: false, Help: "Round-trip protobuf deserialization test"})

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

	filename, _, ast, _, err := ll.Focmain(*fname, *binary, true, *stopAt, *toast, "")
	if err != nil {

		os.Exit(1)
	}

	if *genast {
		dir, _ := os.Getwd()
		fmt.Println(dir)
		ext := ".json"
		if *binary {
			ext = ".pb"
		}
		writeToFile(filepath.Join(dir, "ast"), ast, filename+ext)
	}
}

func writeToFile(dir string, s string, filename string) {
	file := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		panic(err)
	}
	err := os.WriteFile(file, []byte(s), 0744)
	if err != nil {
		panic(err)
	}
}
