package symboltable

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
)

func TestEveryRegisteredSymbolFlagEncodesAtItsPermanentPosition(t *testing.T) {
	fields := make([]reflect.StructField, len(SymbolFlagRegistry))
	for i, flag := range SymbolFlagRegistry {
		fields[i] = reflect.StructField{Name: flag.Name, Type: reflect.TypeOf(false)}
	}
	type_ := reflect.StructOf(fields)
	for i, flag := range SymbolFlagRegistry {
		value := reflect.New(type_)
		value.Elem().Field(i).SetBool(true)
		encoded := EncodeSymbolFlags(value.Interface())
		if len(encoded) != flag.Position/8+1 || encoded[flag.Position/8] != 1<<uint(flag.Position%8) {
			t.Errorf("flag %s[%d] encoded as %x", flag.Name, flag.Position, encoded)
		}
	}
}

func TestEveryConcreteSymbolBooleanHasARegisteredFlag(t *testing.T) {
	symbols := []any{
		SymbolDetails{}, VariableDetails{}, VarSymbol{}, PointerSymbol{}, ArraySymbol{}, RangeSymbol{}, ThunkSymbol{}, ReferenceSymbol{}, AddressSymbol{},
		FunctionSymbol{}, DelegateSymbol{}, LambdaSymbol{}, Indexer{}, TypeclassSymbol{}, TypeConstructor{}, VariantConstructor{}, LetBindings{}, LabelSymbol{},
		BlockSymbol{}, FunctionPattern{}, MatcherSymbol{}, MatcherImplSymbol{}, ExtensionSymbol{}, ComponentSymbol{}, ApplicationSymbol{}, LibrarySymbol{}, PackageSymbol{},
		ClassSymbol{}, ModuleSymbol{}, InterfaceSymbol{}, SignatureSymbol{}, EnumSymbol{}, StructSymbol{}, UnionSymbol{}, KindSymbol{}, HokrtlSymbol{}, TypeSymbol{},
		ForComprehension{}, InstanceSymbol{}, ObjectSymbol{}, StaticSymbol{}, AnnotationSymbol{}, DecoratorSymbol{}, MacroSymbol{}, TemplateDetails{}, OperatorDetails{},
		GenericDetails{}, DirectivePragmaDetails{}, DymanicRuntime{}, UseSymbol{}, ExpressionSymbol{}, StatmentSymbol{}, Symbol{},
	}
	var check func(reflect.Type)
	check = func(type_ reflect.Type) {
		for i := 0; i < type_.NumField(); i++ {
			field := type_.Field(i)
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				check(field.Type)
				continue
			}
			if field.Type.Kind() != reflect.Bool {
				continue
			}
			if _, ok := flagPositionByField[field.Name]; !ok {
				t.Errorf("%s.%s has no permanent symbol flag position", type_.Name(), field.Name)
			}
		}
	}
	for _, symbol := range symbols {
		check(reflect.TypeOf(symbol))
	}
}

func TestSymbolFlagGoldenVectorAndTrailingZeros(t *testing.T) {
	type golden struct {
		IsInternal_  bool
		ReturnVar    bool
		IsSealed     bool
		HasInitValue bool
		NamedParams  bool
		Variadic     bool
	}
	got := EncodeSymbolFlags(golden{true, true, true, true, true, true})
	// Positions 0, 7, 8, 15, 63, and 64 => bytes 81 81 00 00 00 00 00 80 01.
	if hex.EncodeToString(got) != "818100000000008001" {
		t.Fatalf("golden vector = %x", got)
	}
	if empty := EncodeSymbolFlags(golden{}); len(empty) != 0 {
		t.Fatalf("empty flags = %x", empty)
	}
}

func TestSymbolFlagsBeyond64AndUnknownBits(t *testing.T) {
	encoded := EncodeSymbolFlags(TypeSymbol{Alias: true, IsGenericType: true})
	if len(encoded) != 24 {
		t.Fatalf("encoded length = %d, want 24 for position 186", len(encoded))
	}
	set, err := DecodeSymbolFlags(SymbolFormatVersion, append(encoded, 0, 0x80))
	if err != nil || !set["Alias"] || !set["IsGenericType"] {
		t.Fatalf("decoded = %v, %v", set, err)
	}
	if _, err := DecodeSymbolFlags(SymbolFormatVersion+1, nil); err == nil {
		t.Fatal("unsupported version accepted")
	}
}

func TestLegacyFunctionTypeSpellingUsesCanonicalFlag(t *testing.T) {
	encoded := EncodeSymbolFlags(TypeSymbol{FuntionTyoe: true})
	set, err := DecodeSymbolFlags(SymbolFormatVersion, encoded)
	if err != nil || !set["FunctionType"] || set["FuntionTyoe"] {
		t.Fatalf("decoded = %v, %v", set, err)
	}
}

func TestPortableSymbolRecordRoundTripIsDeterministic(t *testing.T) {
	symbol := &TypeSymbol{SymbolDetails: SymbolDetails{SymbolId_: "symbol_7", SymbolType_: "Type", Name_: "Count", State: Unresolved, SymbolTableId: "sym_2"}, Alias: true, Hidden: true}
	record := ProjectSymbol(symbol)
	first, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("serialization is nondeterministic:\n%s\n%s", first, second)
	}
	var decoded SymbolRecord
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SymbolID != "symbol_7" || decoded.SymbolFlags != SymbolFlagsHex(symbol) {
		t.Fatalf("round trip = %#v", decoded)
	}
	if _, leaked := decoded.Fields["Alias"]; leaked {
		t.Fatal("boolean Alias leaked outside symbolFlags")
	}
}

func TestPortableSymbolRecordPreservesNestedAttributes(t *testing.T) {
	statement := &StatmentSymbol{
		SymbolDetails: SymbolDetails{SymbolId_: "statement", SymbolType_: string(S_StatmentSymbol)},
		Type_:         "lowered-control-flow",
	}
	if got := ProjectSymbol(statement).Fields["Type_"]; got != "lowered-control-flow" {
		t.Fatalf("concrete field shadowing a core field was lost: %#v", got)
	}

	function := &FunctionSymbol{
		SymbolDetails: SymbolDetails{SymbolId_: "function", SymbolType_: string(S_FunctionSymbol)},
		AssociatedNode: StructSymbol{
			SymbolDetails: SymbolDetails{SymbolId_: "associated", SymbolType_: string(S_StructSymbol), Name_: "Entry"},
			Embedded:      true,
		},
	}
	record := ProjectSymbol(function)
	nested, ok := record.Fields["AssociatedNode"].(SymbolRecord)
	if !ok || nested.SymbolID != "associated" {
		t.Fatalf("nested symbol record = %#v", record.Fields["AssociatedNode"])
	}
	set, err := hex.DecodeString(nested.SymbolFlags)
	if err != nil {
		t.Fatal(err)
	}
	flags, err := DecodeSymbolFlags(nested.SymbolFormatVersion, set)
	if err != nil || !flags["Embedded"] {
		t.Fatalf("nested symbol flags = %v, %v", flags, err)
	}

	generic := &GenericDetails{
		SymbolDetails:    SymbolDetails{SymbolId_: "generic", SymbolType_: string(S_GenericDetails)},
		GenericTypeParam: []GenericTypeParam{{Name: "T", Nullable: true, Impredicative: true}},
	}
	params := ProjectSymbol(generic).Fields["GenericTypeParam"].([]any)
	param := params[0].(map[string]any)
	if param["Nullable"] != true || param["Impredicative"] != true {
		t.Fatalf("nested non-symbol Boolean attributes were lost: %#v", param)
	}
}
