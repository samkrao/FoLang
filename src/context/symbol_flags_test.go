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
