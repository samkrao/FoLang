package parser

import (
	"reflect"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
)

// The canonical symbol registry.
//
// FolangSymbols.SymbolsById is where a symbol record LIVES; a symbol table and an
// AST node hold only its id. That makes the registry the artifact's inventory of
// declarations, and gives it two invariants a pointer graph enforced for free:
//
//   - Nothing survives that nothing refers to. A speculative branch the parser
//     threw away has no AST left and no binding left, so a record minted inside it
//     would reach the backend as a declaration the program never made.
//   - Every id that is referred to resolves. Context.OwnerSymbolId names a scope's
//     owning symbol by id, so the owner has to BE in the registry for the link to
//     mean anything.

// symbolGraphReferences collects every id the parse still refers to: from a symbol
// table, from a context's owner link, and from the AST.
func symbolGraphReferences(root ast.Stmt, p *parser) map[string]bool {
	referenced := map[string]bool{}
	for _, table := range p.fs.SymboltableMap {
		for _, id := range table.SymbolIds {
			referenced[id] = true
		}
		for _, ids := range table.SymbolsByName {
			for _, id := range ids {
				referenced[id] = true
			}
		}
	}
	for _, ctx := range p.fs.ContextMap {
		if ctx.OwnerSymbolId != "" {
			referenced[ctx.OwnerSymbolId] = true
		}
	}
	collectASTSymbolReferences(reflect.ValueOf(root), referenced, map[uintptr]bool{}, 0)
	return referenced
}

func collectASTSymbolReferences(value reflect.Value, out map[string]bool, seen map[uintptr]bool, depth int) {
	if !value.IsValid() || depth > 64 {
		return
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return
		}
		if value.Kind() == reflect.Pointer {
			if seen[value.Pointer()] {
				return
			}
			seen[value.Pointer()] = true
		}
		if value.CanInterface() {
			if info, ok := value.Interface().(symboltable.SymbolInfo); ok {
				out[info.GetSymbolID()] = true
			}
		}
		collectASTSymbolReferences(value.Elem(), out, seen, depth+1)
	case reflect.Struct:
		type_ := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := type_.Field(i)
			if !field.IsExported() {
				continue
			}
			if field.Name == "SymbolId" && value.Field(i).Kind() == reflect.String {
				out[value.Field(i).String()] = true
				continue
			}
			collectASTSymbolReferences(value.Field(i), out, seen, depth+1)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			collectASTSymbolReferences(value.Index(i), out, seen, depth+1)
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			collectASTSymbolReferences(iter.Value(), out, seen, depth+1)
		}
	}
}

// A block body is read speculatively — the parser cannot tell a bare block from a
// composite literal until it has read into it — so a labelled or bare block whose
// first reading is discarded mints declarations that the rollback must erase.
func TestDiscardedSpeculationLeavesNoRecordInTheRegistry(t *testing.T) {
	sources := map[string]string{
		"blocks": `_ co.lang.unit = {
    run(values co.lang.int)->() = {
        'outer: {
            inner co.lang.int = 1;
        }

        {
            bare co.lang.int = 2;
        }

        values.each(index, value, {
            'label: {
                shadowed co.lang.int = 3;
            }
        });
    }
}`,
		"labels": `_ co.lang.unit = {
    scan(limit co.lang.int)->() = {
        marker co.lang.char = 'c';

        'outer: {
            (limit > 0).then({
                this.break 'outer;
            });

            this.break;
        }

        'repeat: (limit > 0).loop({
            (marker == 'x').then({
                this.continue 'repeat;
            });

            this.continue;
        });
    }
}`,
	}

	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			root, p := parsePackageSource(t, source, "speculation.unit.fol")
			if len(p.diags) != 0 {
				t.Fatalf("source produced diagnostics: %v", p.diags)
			}

			referenced := symbolGraphReferences(root, p)
			for id, symbol := range p.fs.SymbolsById {
				if !referenced[id] {
					t.Errorf("registry keeps %s %q (%s), which the parse no longer refers to",
						symbol.GetSymbolType(), symbol.GetName(), id)
				}
			}
		})
	}
}

// A scope owner is minted before its body opens and bound after it closes, and a
// construct that lowering rewrites has an owner that is never bound at all. Waiting
// for the binding to register it therefore leaves OwnerSymbolId unresolvable for
// exactly the symbols the field exists to name.
func TestEveryContextOwnerResolvesInTheRegistry(t *testing.T) {
	sources := map[string]string{
		"named block": `_ co.lang.unit = {
    run()->() = {
        labelBlock co.lang.block = {
            x co.lang.int = 1;
        }

        labelBlock.expand();
    }
}`,
		"unit": `_ co.lang.unit = {
    run()->() = {
        x co.lang.int = 1;
    }
}`,
	}

	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			_, p := parsePackageSource(t, source, "owners.unit.fol")
			if len(p.diags) != 0 {
				t.Fatalf("source produced diagnostics: %v", p.diags)
			}

			owned := 0
			for _, ctx := range p.fs.ContextMap {
				if ctx.OwnerSymbolId == "" {
					continue
				}
				owned++
				if p.fs.GetSymbol(ctx.OwnerSymbolId) == nil {
					t.Errorf("context %s (%s) is owned by %s, which is absent from the registry",
						ctx.Id, ctx.ContextType_, ctx.OwnerSymbolId)
				}
			}
			if owned == 0 {
				t.Fatal("no context recorded an owning symbol; the test proves nothing")
			}
		})
	}
}
