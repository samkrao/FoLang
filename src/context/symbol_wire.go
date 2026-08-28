package symboltable

import (
	"encoding/json"
	"reflect"
)

// SymbolRecord is the language-neutral serialized form of a symbol. Boolean
// properties live only in SymbolFlags; Fields contains the remaining data.
type SymbolRecord struct {
	SymbolFormatVersion int            `json:"symbolFormatVersion"`
	SymbolID            string         `json:"symbolId"`
	SymbolType          string         `json:"symbolType"`
	Name                string         `json:"name"`
	Type                string         `json:"type,omitempty"`
	State               ResolveState   `json:"state"`
	SymbolTableID       string         `json:"symbolTableId,omitempty"`
	OwnedContextID      string         `json:"ownedContextId,omitempty"`
	SymbolFlags         string         `json:"symbolFlags"`
	Fields              map[string]any `json:"fields,omitempty"`
}

// ProjectSymbol converts an in-memory Go symbol to its portable wire record.
func ProjectSymbol(info SymbolInfo) SymbolRecord {
	details := symbolDetailsOf(reflect.ValueOf(info))
	return SymbolRecord{
		SymbolFormatVersion: SymbolFormatVersion,
		SymbolID:            info.GetSymbolID(), SymbolType: info.GetSymbolType(), Name: info.GetName(),
		Type: info.GetType(), State: info.ResolutionState(), SymbolTableID: details.SymbolTableId,
		OwnedContextID: info.GetContextID(),
		SymbolFlags:    SymbolFlagsHex(info), Fields: nonBooleanSymbolFields(reflect.ValueOf(info)),
	}
}

func symbolDetailsOf(value reflect.Value) SymbolDetails {
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Struct {
		field := value.FieldByName("SymbolDetails")
		if field.IsValid() && field.CanInterface() {
			return field.Interface().(SymbolDetails)
		}
		if value.Type() == reflect.TypeOf(SymbolDetails{}) {
			return value.Interface().(SymbolDetails)
		}
	}
	return SymbolDetails{}
}

func nonBooleanSymbolFields(value reflect.Value) map[string]any {
	fields := map[string]any{}
	collectNonBooleanFields(value, fields)
	for _, core := range []string{"SymbolId_", "OwnedContextId", "SymbolType_", "Name_", "State", "Type_", "SymbolTableId"} {
		delete(fields, core)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func collectNonBooleanFields(value reflect.Value, out map[string]any) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}
	type_ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field, fieldType := value.Field(i), type_.Field(i)
		if !fieldType.IsExported() || field.Kind() == reflect.Bool {
			continue
		}
		if fieldType.Anonymous {
			collectNonBooleanFields(field, out)
			continue
		}
		if field.CanInterface() {
			out[fieldType.Name] = portableNonBooleanValue(field)
		}
	}
}

func portableNonBooleanValue(value reflect.Value) any {
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		if info, ok := value.Interface().(SymbolInfo); ok {
			return info.GetSymbolID()
		}
		return portableNonBooleanValue(value.Elem())
	}
	switch value.Kind() {
	case reflect.Struct:
		out := map[string]any{}
		collectNonBooleanFields(value, out)
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, value.Len())
		for i := range out {
			out[i] = portableNonBooleanValue(value.Index(i))
		}
		return out
	case reflect.Map:
		out := map[string]any{}
		iter := value.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = portableNonBooleanValue(iter.Value())
		}
		return out
	default:
		return value.Interface()
	}
}

// MarshalJSON keeps table topology unchanged while replacing repeated concrete
// symbol structs with portable records.
func (s SymbolTable) MarshalJSON() ([]byte, error) {
	records := make(map[string]SymbolRecord, len(s.Symboldetails))
	for key, info := range s.Symboldetails {
		records[key] = ProjectSymbol(info)
	}
	type wireTable struct {
		ID        string                  `json:"Id"`
		ParentID  string                  `json:"ParentId"`
		ContextID string                  `json:"ContextId"`
		Prefix    string                  `json:"Prefix"`
		Symbols   map[string]SymbolRecord `json:"Symboldetails"`
	}
	return json.Marshal(wireTable{s.Id, s.ParentId, s.ContextId, s.Prefix, records})
}
