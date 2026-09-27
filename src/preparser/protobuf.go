package preparser

import (
	"os"
	"path/filepath"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type SerializableLibComponents struct {
	SurfaceSymbols *symboltable.FolangSymbols
	Ast            ast.SET
	FolangSymbols  *symboltable.FolangSymbols
}

func Deserialize(filename string) (SerializableLibComponents, string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return SerializableLibComponents{}, "error"
	}

	artifact := &structpb.Struct{}
	if err := proto.Unmarshal(data, artifact); err != nil {
		return SerializableLibComponents{}, "error"
	}

	var components SerializableLibComponents
	if err := decodeArtifact(artifact, &components); err != nil {
		return SerializableLibComponents{}, "error"
	}
	return components, "success"
}

// Serialize writes the provisional protobuf Value/Struct/List representation
// selected by the backend contract. It returns "success" only after the
// complete artifact has been written.
func Serialize(symbols *symboltable.FolangSymbols, root ast.SET, surfaceSymbols *symboltable.FolangSymbols, filename string) string {
	artifact, err := encodeArtifact(SerializableLibComponents{
		SurfaceSymbols: surfaceSymbols,
		Ast:            root,
		FolangSymbols:  symbols,
	})
	if err != nil {
		return "error"
	}

	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(artifact)
	if err != nil {
		return "error"
	}
	if dir := filepath.Dir(filename); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "error"
		}
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return "error"
	}
	return "success"
}

func encodeArtifact(components SerializableLibComponents) (*structpb.Struct, error) {
	surface, err := encodePortable(components.SurfaceSymbols)
	if err != nil {
		return nil, err
	}
	root, err := encodePortable(components.Ast)
	if err != nil {
		return nil, err
	}
	symbols, err := encodePortable(components.FolangSymbols)
	if err != nil {
		return nil, err
	}
	return &structpb.Struct{Fields: map[string]*structpb.Value{
		"surface_symbols": surface,
		"ast":             root,
		"folang_symbols":  symbols,
	}}, nil
}

func decodeArtifact(artifact *structpb.Struct, components *SerializableLibComponents) error {
	if artifact == nil || components == nil {
		return errInvalidArtifact
	}
	fields := artifact.GetFields()
	if err := decodePortable(fields["surface_symbols"], &components.SurfaceSymbols); err != nil {
		return err
	}
	if err := decodePortable(fields["ast"], &components.Ast); err != nil {
		return err
	}
	if err := decodePortable(fields["folang_symbols"], &components.FolangSymbols); err != nil {
		return err
	}
	return nil
}
