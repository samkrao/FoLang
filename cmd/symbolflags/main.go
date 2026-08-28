// Command symbolflags generates the normative symbol flag layout table.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

func main() {
	if err := os.WriteFile("docs/symbol-flag-layout.md", render(), 0o644); err != nil {
		panic(err)
	}
}

func render() []byte {
	applicable := declaringTypes()
	var out strings.Builder
	out.WriteString("# Symbol flag layout\n\n")
	out.WriteString("This is the normative backend-neutral layout for serialized FoLang symbols. Format version: `1`. ")
	out.WriteString("Position `P` is byte `P / 8`, bit `P % 8`; bit zero is the least-significant bit of byte zero. ")
	out.WriteString("JSON writes the canonical bytes as a lowercase hexadecimal `symbolFlags` string. Missing trailing bytes mean false. ")
	out.WriteString("Positions are append-only; removed positions become reserved and are never reused. Readers reject unsupported format versions. ")
	out.WriteString("Unknown set bits are preserved by transport and ignored semantically until a newer registry defines them. Backends may use any in-memory bitset, but must decode these bytes without native-word, alignment, ABI, or endianness assumptions.\n\n")
	out.WriteString("Primary symbol category remains the stable `symbolType` string in version 1; relationship data such as subtype/supertype targets remains in symbol fields and is not reduced to a flag.\n\n")
	out.WriteString("The final column names the Go record type that declares the property; embedded records inherit those flags. The names and positions are wire contracts, while these Go type names are implementation-location guidance.\n\n")
	out.WriteString("| Position | Byte | Bit | Flag name | Meaning | Applicable symbol kinds |\n")
	out.WriteString("|---:|---:|---:|---|---|---|\n")
	for _, flag := range symboltable.SymbolFlagRegistry {
		kinds := strings.Join(applicable[flag.Name], ", ")
		fmt.Fprintf(&out, "| %d | %d | %d | `%s` | Existing `%s` symbol property | %s |\n", flag.Position, flag.Position/8, flag.Position%8, flag.Name, flag.Name, kinds)
	}
	return []byte(out.String())
}

func declaringTypes() map[string][]string {
	_, file, _, _ := runtime.Caller(0)
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(file), "..", "..", "src", "context", "symbols.go"), nil, 0)
	if err != nil {
		panic(err)
	}
	result := map[string][]string{}
	for _, declaration := range parsed.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range generic.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structure, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				ident, isBool := field.Type.(*ast.Ident)
				if !isBool || ident.Name != "bool" {
					continue
				}
				for _, name := range field.Names {
					flagName := name.Name
					if flagName == "IsInternal_" {
						flagName = "IsInternal"
					}
					if flagName == "FuntionTyoe" {
						flagName = "FunctionType"
					}
					result[flagName] = append(result[flagName], "`"+typeSpec.Name.Name+"`")
				}
			}
		}
	}
	for _, types := range result {
		sort.Strings(types)
	}
	return result
}
