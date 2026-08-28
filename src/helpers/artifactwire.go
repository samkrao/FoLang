package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"google.golang.org/protobuf/encoding/protowire"
)

// The FoLang artifact wire format, defined by src/shared/folang-artifact.proto.
//
// It is google.protobuf.Value's shape with one field added: an int64 on tag 7.
// That field is why this schema exists rather than the well-known type. A
// google.protobuf.Value carries every number as a double, and FoLang's
// co.lang.int is 64-bit — ast.IntegerLiteral.Value is an int64 — so a literal
// past 2^53 cannot survive that encoding and would reach the backend as a
// different number with nothing reported.
//
// The messages are read and written directly with protowire rather than through
// generated code, so the frontend needs no protoc in its build. The .proto file
// is the contract a backend generates its own reader from; it and this file must
// change together.
const (
	artifactNullField   = 1
	artifactDoubleField = 2
	artifactStringField = 3
	artifactBoolField   = 4
	artifactStructField = 5
	artifactListField   = 6
	artifactIntField    = 7

	artifactStructEntriesField = 1
	artifactEntryKeyField      = 1
	artifactEntryValueField    = 2
	artifactListValuesField    = 1
)

// appendArtifactValue writes one Value message.
func appendArtifactValue(out []byte, value any) ([]byte, error) {
	switch typed := value.(type) {
	case nil:
		out = protowire.AppendTag(out, artifactNullField, protowire.VarintType)
		return protowire.AppendVarint(out, 0), nil
	case bool:
		out = protowire.AppendTag(out, artifactBoolField, protowire.VarintType)
		return protowire.AppendVarint(out, protowire.EncodeBool(typed)), nil
	case string:
		out = protowire.AppendTag(out, artifactStringField, protowire.BytesType)
		return protowire.AppendString(out, typed), nil
	case int64:
		out = protowire.AppendTag(out, artifactIntField, protowire.VarintType)
		return protowire.AppendVarint(out, protowire.EncodeZigZag(typed)), nil
	case float64:
		out = protowire.AppendTag(out, artifactDoubleField, protowire.Fixed64Type)
		return protowire.AppendFixed64(out, math.Float64bits(typed)), nil
	case map[string]any:
		nested, err := appendArtifactStruct(nil, typed)
		if err != nil {
			return nil, err
		}
		out = protowire.AppendTag(out, artifactStructField, protowire.BytesType)
		return protowire.AppendBytes(out, nested), nil
	case []any:
		nested, err := appendArtifactList(nil, typed)
		if err != nil {
			return nil, err
		}
		out = protowire.AppendTag(out, artifactListField, protowire.BytesType)
		return protowire.AppendBytes(out, nested), nil
	default:
		return nil, fmt.Errorf("encoding artifact: %T is not a value the artifact schema carries", value)
	}
}

// appendArtifactStruct writes a Struct message.
//
// Keys are sorted so one artifact encodes to one byte sequence. A map has no
// order of its own, and an artifact whose bytes changed between identical builds
// could not be compared, cached, or checksummed.
func appendArtifactStruct(out []byte, fields map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entry := protowire.AppendTag(nil, artifactEntryKeyField, protowire.BytesType)
		entry = protowire.AppendString(entry, key)

		value, err := appendArtifactValue(nil, fields[key])
		if err != nil {
			return nil, err
		}
		entry = protowire.AppendTag(entry, artifactEntryValueField, protowire.BytesType)
		entry = protowire.AppendBytes(entry, value)

		out = protowire.AppendTag(out, artifactStructEntriesField, protowire.BytesType)
		out = protowire.AppendBytes(out, entry)
	}
	return out, nil
}

// appendArtifactList writes a List message.
func appendArtifactList(out []byte, values []any) ([]byte, error) {
	for _, member := range values {
		encoded, err := appendArtifactValue(nil, member)
		if err != nil {
			return nil, err
		}
		out = protowire.AppendTag(out, artifactListValuesField, protowire.BytesType)
		out = protowire.AppendBytes(out, encoded)
	}
	return out, nil
}

// parseArtifactValue reads one Value message.
//
// A Value carries exactly one field. A message with none is malformed rather
// than null: null is written explicitly, so an empty Value means the bytes were
// truncated somewhere this reader cannot see.
func parseArtifactValue(data []byte) (any, error) {
	var (
		value  any
		filled bool
	)
	for len(data) > 0 {
		number, kind, taken := protowire.ConsumeTag(data)
		if taken < 0 {
			return nil, fmt.Errorf("decoding artifact value: %w", protowire.ParseError(taken))
		}
		data = data[taken:]

		var err error
		switch number {
		case artifactNullField:
			_, taken = protowire.ConsumeVarint(data)
			value, filled = nil, true
		case artifactBoolField:
			var raw uint64
			raw, taken = protowire.ConsumeVarint(data)
			value, filled = protowire.DecodeBool(raw), true
		case artifactStringField:
			var text string
			text, taken = protowire.ConsumeString(data)
			value, filled = text, true
		case artifactIntField:
			var raw uint64
			raw, taken = protowire.ConsumeVarint(data)
			value, filled = protowire.DecodeZigZag(raw), true
		case artifactDoubleField:
			var raw uint64
			raw, taken = protowire.ConsumeFixed64(data)
			value, filled = math.Float64frombits(raw), true
		case artifactStructField:
			var nested []byte
			nested, taken = protowire.ConsumeBytes(data)
			if taken >= 0 {
				value, err = parseArtifactStruct(nested)
				filled = true
			}
		case artifactListField:
			var nested []byte
			nested, taken = protowire.ConsumeBytes(data)
			if taken >= 0 {
				value, err = parseArtifactList(nested)
				filled = true
			}
		default:
			// An unrecognized field is skipped rather than refused, which is what
			// lets a newer producer add one without breaking this reader.
			taken = protowire.ConsumeFieldValue(number, kind, data)
		}
		if taken < 0 {
			return nil, fmt.Errorf("decoding artifact value: %w", protowire.ParseError(taken))
		}
		if err != nil {
			return nil, err
		}
		data = data[taken:]
	}
	if !filled {
		return nil, fmt.Errorf("decoding artifact value: the message carries no value")
	}
	return value, nil
}

// parseArtifactStruct reads a Struct message into an ordinary map.
func parseArtifactStruct(data []byte) (map[string]any, error) {
	fields := map[string]any{}
	for len(data) > 0 {
		number, kind, taken := protowire.ConsumeTag(data)
		if taken < 0 {
			return nil, fmt.Errorf("decoding artifact struct: %w", protowire.ParseError(taken))
		}
		data = data[taken:]
		if number != artifactStructEntriesField {
			taken = protowire.ConsumeFieldValue(number, kind, data)
			if taken < 0 {
				return nil, fmt.Errorf("decoding artifact struct: %w", protowire.ParseError(taken))
			}
			data = data[taken:]
			continue
		}
		entry, taken := protowire.ConsumeBytes(data)
		if taken < 0 {
			return nil, fmt.Errorf("decoding artifact struct: %w", protowire.ParseError(taken))
		}
		data = data[taken:]

		key, value, err := parseArtifactEntry(entry)
		if err != nil {
			return nil, err
		}
		fields[key] = value
	}
	return fields, nil
}

// parseArtifactEntry reads one map entry of a Struct.
func parseArtifactEntry(data []byte) (string, any, error) {
	var (
		key   string
		value any
	)
	for len(data) > 0 {
		number, kind, taken := protowire.ConsumeTag(data)
		if taken < 0 {
			return "", nil, fmt.Errorf("decoding artifact field: %w", protowire.ParseError(taken))
		}
		data = data[taken:]

		switch number {
		case artifactEntryKeyField:
			key, taken = protowire.ConsumeString(data)
		case artifactEntryValueField:
			var nested []byte
			nested, taken = protowire.ConsumeBytes(data)
			if taken >= 0 {
				parsed, err := parseArtifactValue(nested)
				if err != nil {
					return "", nil, err
				}
				value = parsed
			}
		default:
			taken = protowire.ConsumeFieldValue(number, kind, data)
		}
		if taken < 0 {
			return "", nil, fmt.Errorf("decoding artifact field: %w", protowire.ParseError(taken))
		}
		data = data[taken:]
	}
	return key, value, nil
}

// parseArtifactList reads a List message.
func parseArtifactList(data []byte) ([]any, error) {
	values := []any{}
	for len(data) > 0 {
		number, kind, taken := protowire.ConsumeTag(data)
		if taken < 0 {
			return nil, fmt.Errorf("decoding artifact list: %w", protowire.ParseError(taken))
		}
		data = data[taken:]
		if number != artifactListValuesField {
			taken = protowire.ConsumeFieldValue(number, kind, data)
			if taken < 0 {
				return nil, fmt.Errorf("decoding artifact list: %w", protowire.ParseError(taken))
			}
			data = data[taken:]
			continue
		}
		nested, taken := protowire.ConsumeBytes(data)
		if taken < 0 {
			return nil, fmt.Errorf("decoding artifact list: %w", protowire.ParseError(taken))
		}
		data = data[taken:]

		parsed, err := parseArtifactValue(nested)
		if err != nil {
			return nil, err
		}
		values = append(values, parsed)
	}
	return values, nil
}

// artifactTree projects a value to the tree the schema carries.
//
// It decodes with UseNumber so a number arrives as its exact written text, and
// keeps an integer AS an integer. Decoding into `any` would turn every number
// into a float64 first, which is the step that loses the digit.
func artifactTree(value any) (any, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("projecting artifact: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	var exact any
	if err := decoder.Decode(&exact); err != nil {
		return nil, fmt.Errorf("projecting artifact: %w", err)
	}
	return artifactNumbers(exact, "")
}

// artifactNumbers rewrites each written number as the type the schema carries:
// an integer as an int64, anything with a fraction or an exponent as a double.
func artifactNumbers(value any, path string) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, member := range typed {
			converted, err := artifactNumbers(member, path+"/"+key)
			if err != nil {
				return nil, err
			}
			out[key] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, member := range typed {
			converted, err := artifactNumbers(member, fmt.Sprintf("%s[%d]", path, index))
			if err != nil {
				return nil, err
			}
			out[index] = converted
		}
		return out, nil
	case json.Number:
		if asInt, err := typed.Int64(); err == nil {
			return asInt, nil
		}
		asFloat, err := typed.Float64()
		if err != nil {
			return nil, fmt.Errorf("encoding artifact: %s is %s, which is outside the range the artifact schema carries",
				artifactPath(path), typed.String())
		}
		return asFloat, nil
	default:
		return value, nil
	}
}
