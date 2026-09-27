package preparser

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
)

// normalizeSourceBytes converts supported source-file encodings to the UTF-8
// representation consumed by scanlex. UTF-16 is recognized by its byte-order
// mark; without one, its byte order cannot be determined reliably.
func normalizeSourceBytes(source []byte) []byte {
	switch {
	case bytes.HasPrefix(source, utf8BOM):
		return source[len(utf8BOM):]
	case bytes.HasPrefix(source, utf16LEBOM):
		return decodeUTF16(source[len(utf16LEBOM):], binary.LittleEndian)
	case bytes.HasPrefix(source, utf16BEBOM):
		return decodeUTF16(source[len(utf16BEBOM):], binary.BigEndian)
	default:
		return source
	}
}

func decodeUTF16(source []byte, order binary.ByteOrder) []byte {
	units := make([]uint16, (len(source)+1)/2)
	completeBytes := len(source) - len(source)%2
	for offset := 0; offset < completeBytes; offset += 2 {
		units[offset/2] = order.Uint16(source[offset : offset+2])
	}
	if len(source)%2 != 0 {
		// Preserve malformed trailing input as U+FFFD instead of silently
		// dropping it or rejecting the entire source file.
		units[len(units)-1] = uint16('\uFFFD')
	}
	return []byte(string(utf16.Decode(units)))
}
