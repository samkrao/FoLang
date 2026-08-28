package entry

import (
	"fmt"
	"os"

	"github.com/akamensky/argparse"
	ll "github.com/samkrao/fo-lang/src/parser"
)

// Run is the entry point for the frontend tool.
// args should be os.Args (program name + flags).
func Run(args []string) {
	if err := configureDebugTracingAtStartup(); err != nil {
		fmt.Fprintf(os.Stderr, "fo-frontend: %v\n", err)
		os.Exit(1)
	}

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

	binary = parser.Flag("b", "Binary", &argparse.Options{Required: false, Help: "Deprecated; artifact wire is selected by backend-conf.json"})
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

	_, artifact, _, _, err := ll.Focmain(*fname, *binary, true, *stopAt, *toast, "")
	if err != nil {

		os.Exit(1)
	}

	// The driver writes the JSON artifact itself, beneath the project's build/
	// domain, which is where "Compiler and Backend" puts the frontend/backend
	// interchange artifact. This used to write a second copy under ./ast/ relative
	// to the CURRENT WORKING DIRECTORY — a location the reference does not define,
	// which moved with the shell rather than with the project and which the backend
	// has no reason to look in. Reporting the real path is what is left to do.
	if !*genast {
		return
	}
	if artifact != "" {
		fmt.Println(artifact)
		return
	}

	// No artifact is not the same as a failed compile, and the reason is worth
	// stating: build/ is a project-ROOT domain, so a file compiled outside a
	// discovered project has nowhere to put one. It goes to stderr because stdout
	// carries the artifact path.
	fmt.Fprintf(os.Stderr, "%s\n", noArtifactReason(*fname))
}

// noArtifactReason explains a compile that produced no build/ artifact.
func noArtifactReason(sourceFile string) string {
	return "no build/ artifact written: " + sourceFile +
		" is not inside a discovered project, and build/ is a project-root domain;" +
		" add a fol-conf.yaml at the project root, or pass the root explicitly, to get one"
}
