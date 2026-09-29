package preparser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSerializeDeserializeProtobufArtifact(t *testing.T) {
	symbols := &symboltable.FolangSymbols{}
	symbols.CreateFolangSymbols()
	symbols.BackendConf = symboltable.BackendConfig{
		Protocol:          "folang-plugin/1.0",
		HIRSchema:         "folang-hir/1",
		Wire:              "protobuf/1.0",
		RuntimeOperations: "folang-runtime-operations/1",
	}

	variable := &symboltable.Variable{SymbolDetails: symboltable.SymbolDetails{
		SymbolId_:     "symbol:value",
		Name_:         "value",
		Type_:         "co.int",
		SymbolTableId: "table:root",
	}}
	literal := &symboltable.Literal{SymbolDetails: symboltable.SymbolDetails{
		SymbolId_: "literal:42",
		Name_:     "42",
		Type_:     "co.int",
	}}
	symbols.RegisterSymbol(variable)
	symbols.RegisterSymbol(literal)
	symbols.AddSymbolTable(&symboltable.SymbolTable{
		Id:            "table:root",
		ContextId:     "context:root",
		SymbolIds:     []symboltable.SymbolID{"symbol:value"},
		SymbolsByName: map[symboltable.SymbolName][]symboltable.SymbolID{"value": {"symbol:value"}},
	})
	symbols.AddContext(&symboltable.Context{
		Id:            "context:root",
		SymbolTables_: []symboltable.SymbolTableID{"table:root"},
	})
	symbols.AddFolContext(&symboltable.FolContext{
		Id:            "project:root",
		Context_:      "context:root",
		SymbolTables_: []symboltable.SymbolTableID{"table:root"},
	})
	if err := symbols.Operators.Register("%%", &symboltable.OperatorSymbol{
		SymbolDetails: symboltable.SymbolDetails{SymbolId_: "operator:%%", Name_: "%%"},
		Fixity:        symboltable.OperatorInfix,
		Precedence:    60,
		Associativity: symboltable.OperatorLeft,
		Arity:         symboltable.OperatorBinary,
		Foldable:      true,
	}); err != nil {
		t.Fatal(err)
	}

	surface := &symboltable.FolangSymbols{}
	surface.CreateFolangSymbols()
	span := ast.NewSpan(
		*helpers.NewPosition(0, 1, 1, 0, "test.fol", "value = 42", false),
		*helpers.NewPosition(10, 1, 11, 10, "test.fol", "value = 42", false),
	)
	root := ast.SourceFile{
		Span:     span,
		Filename: "test.fol",
		Items: []ast.SET{ast.ValueDefinition{
			DefinitionHeader: ast.DefinitionHeader{Span: span, Symbol: "symbol:value"},
			Initializer:      ast.LiteralExpr{Span: span, Symbol: "literal:42"},
		}},
	}

	filename := filepath.Join(t.TempDir(), "nested", "library.folenc")
	if status := Serialize(symbols, root, surface, filename); status != "success" {
		t.Fatalf("Serialize status = %q, want success", status)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	wire := &structpb.Struct{}
	if err := proto.Unmarshal(data, wire); err != nil {
		t.Fatalf("artifact is not protobuf: %v", err)
	}
	backend := wire.Fields["folang_symbols"].GetStructValue().Fields["backend_conf"].GetStructValue()
	if got := backend.Fields["protocol"].GetStringValue(); got != "folang-plugin/1.0" {
		t.Fatalf("serialized backend protocol = %q", got)
	}
	if got := backend.Fields["hir_schema"].GetStringValue(); got != "folang-hir/1" {
		t.Fatalf("serialized HIR schema = %q", got)
	}
	if got := backend.Fields["wire"].GetStringValue(); got != "protobuf/1.0" {
		t.Fatalf("serialized wire = %q", got)
	}
	if got := backend.Fields["runtime_operations"].GetStringValue(); got != "folang-runtime-operations/1" {
		t.Fatalf("serialized runtime operations = %q", got)
	}

	decoded, status := Deserialize(filename)
	if status != "success" {
		t.Fatalf("Deserialize status = %q, want success", status)
	}
	if decoded.FolangSymbols == nil || decoded.SurfaceSymbols == nil {
		t.Fatal("Deserialize lost a symbol graph")
	}
	if decoded.FolangSymbols.BackendConf != symbols.BackendConf {
		t.Fatalf("backend config = %#v, want %#v", decoded.FolangSymbols.BackendConf, symbols.BackendConf)
	}
	gotVariable, ok := decoded.FolangSymbols.SymbolsById["symbol:value"].(*symboltable.Variable)
	if !ok || gotVariable.Name_ != "value" || gotVariable.Type_ != "co.int" {
		t.Fatalf("variable symbol = %#v", decoded.FolangSymbols.SymbolsById["symbol:value"])
	}
	gotContext, ok := decoded.FolangSymbols.ContextMap["project:root"].(*symboltable.FolContext)
	if !ok || gotContext.Context_ != "context:root" {
		t.Fatalf("root context = %#v", decoded.FolangSymbols.ContextMap["project:root"])
	}
	gotOperator := decoded.FolangSymbols.Operators.BySpelling["%%"]
	if gotOperator == nil || gotOperator.Fixity != symboltable.OperatorInfix || gotOperator.Precedence != 60 {
		t.Fatalf("operator = %#v", gotOperator)
	}
	gotRoot, ok := decoded.Ast.(ast.SourceFile)
	if !ok {
		t.Fatalf("AST type = %T, want ast.SourceFile", decoded.Ast)
	}
	if len(gotRoot.Items) != 1 {
		t.Fatalf("AST items = %d, want 1", len(gotRoot.Items))
	}
	gotDefinition, ok := gotRoot.Items[0].(ast.ValueDefinition)
	if !ok {
		t.Fatalf("AST item type = %T, want ast.ValueDefinition", gotRoot.Items[0])
	}
	gotLiteral, ok := gotDefinition.Initializer.(ast.LiteralExpr)
	if !ok || gotLiteral.Symbol != "literal:42" {
		t.Fatalf("literal = %#v", gotDefinition.Initializer)
	}
}

func TestDeserializeRejectsInvalidProtobuf(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "invalid.folenc")
	if err := os.WriteFile(filename, []byte("not protobuf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, status := Deserialize(filename); status != "error" {
		t.Fatalf("Deserialize status = %q, want error", status)
	}
}
