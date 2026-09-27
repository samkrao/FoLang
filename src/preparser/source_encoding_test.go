package preparser

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestNormalizeSourceBytes(t *testing.T) {
	const source = "name := 变量 ⊗ 𐐀\n"
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "UTF-8", input: []byte(source)},
		{name: "UTF-8 BOM", input: append([]byte{0xEF, 0xBB, 0xBF}, []byte(source)...)},
		{name: "UTF-16LE", input: encodeUTF16Source(source, binary.LittleEndian, []byte{0xFF, 0xFE})},
		{name: "UTF-16BE", input: encodeUTF16Source(source, binary.BigEndian, []byte{0xFE, 0xFF})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := string(normalizeSourceBytes(test.input)); got != source {
				t.Fatalf("normalizeSourceBytes() = %q, want %q", got, source)
			}
		})
	}
}

func TestNormalizeSourceBytesPreservesMalformedUTF16AsReplacementRune(t *testing.T) {
	input := []byte{0xFF, 0xFE, 'a', 0x00, 0xFF}
	if got, want := string(normalizeSourceBytes(input)), "a\uFFFD"; got != want {
		t.Fatalf("normalizeSourceBytes() = %q, want %q", got, want)
	}
}

func encodeUTF16Source(source string, order binary.ByteOrder, bom []byte) []byte {
	units := utf16.Encode([]rune(source))
	encoded := make([]byte, len(bom)+len(units)*2)
	copy(encoded, bom)
	for i, unit := range units {
		order.PutUint16(encoded[len(bom)+i*2:], unit)
	}
	return encoded
}
