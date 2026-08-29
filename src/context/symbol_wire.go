package symboltable

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	if portable, ok := info.(*PortableSymbol); ok {
		return portable.Record
	}
	details := symbolDetailsOf(reflect.ValueOf(info))
	return SymbolRecord{
		SymbolFormatVersion: SymbolFormatVersion,
		SymbolID:            info.GetSymbolID(), SymbolType: info.GetSymbolType(), Name: info.GetName(),
		Type: info.GetType(), State: info.ResolutionState(), SymbolTableID: details.SymbolTableId,
		OwnedContextID: info.GetContextID(),
		SymbolFlags:    SymbolFlagsHex(info), Fields: nonBooleanSymbolFields(reflect.ValueOf(info)),
	}
}

// PortableSymbol is the concrete, backend-neutral SymbolInfo produced when a
// FolangSymbols artifact is decoded. Semantic phases that need a specialized Go
// symbol type may inflate it later; ordinary graph lookup needs only SymbolInfo.
type PortableSymbol struct {
	Record SymbolRecord
}

func registerSymbolGraph(fs *FolangSymbols, symbol SymbolInfo, visited map[uintptr]bool) {
	if symbol == nil || symbol.GetSymbolID() == "" {
		return
	}
	value := reflect.ValueOf(symbol)
	if value.Kind() == reflect.Pointer && !value.IsNil() {
		pointer := value.Pointer()
		if visited[pointer] {
			return
		}
		visited[pointer] = true
	}
	fs.SymbolsById[symbol.GetSymbolID()] = symbol
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	registerSymbolValues(fs, value, visited)
}

func registerSymbolValues(fs *FolangSymbols, value reflect.Value, visited map[uintptr]bool) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		if value.CanInterface() {
			if symbol, ok := value.Interface().(SymbolInfo); ok {
				registerSymbolGraph(fs, symbol, visited)
				return
			}
		}
		registerSymbolValues(fs, value.Elem(), visited)
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		type_ := value.Type()
		for i := 0; i < value.NumField(); i++ {
			if type_.Field(i).Anonymous || !type_.Field(i).IsExported() {
				continue
			}
			field := value.Field(i)
			if field.CanAddr() && field.Addr().CanInterface() {
				if symbol, ok := field.Addr().Interface().(SymbolInfo); ok {
					registerSymbolGraph(fs, symbol, visited)
					continue
				}
			}
			registerSymbolValues(fs, field, visited)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			registerSymbolValues(fs, value.Index(i), visited)
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			registerSymbolValues(fs, iter.Value(), visited)
		}
	}
}

func (s *PortableSymbol) GetSymbolID() string           { return s.Record.SymbolID }
func (s *PortableSymbol) GetSymbolType() string         { return s.Record.SymbolType }
func (s *PortableSymbol) GetType() string               { return s.Record.Type }
func (s *PortableSymbol) GetName() string               { return s.Record.Name }
func (s *PortableSymbol) ResolutionState() ResolveState { return s.Record.State }
func (s *PortableSymbol) GetContextID() string          { return s.Record.OwnedContextID }
func (s *PortableSymbol) SetOwnedContextID(id string)   { s.Record.OwnedContextID = id }
func (s *PortableSymbol) Clone() SymbolInfo             { clone := *s; return &clone }
func (s *PortableSymbol) IsInternal() bool {
	flags, err := hex.DecodeString(s.Record.SymbolFlags)
	if err != nil {
		return false
	}
	set, err := DecodeSymbolFlags(s.Record.SymbolFormatVersion, flags)
	return err == nil && set["IsInternal"]
}

type folangSymbolsWire struct {
	RootContextID  string                  `json:"RootContextId"`
	SymbolTables   map[string]*SymbolTable `json:"SymboltableMap"`
	Contexts       map[string]*Context     `json:"ContextMap"`
	SymbolsByID    map[string]SymbolRecord `json:"SymbolsById"`
	SurfaceSymbols *SurfaceSymbols         `json:"SurfaceSymbols,omitempty"`
}

// ArtifactCarriesSymbol reports whether a live parser symbol has semantic data
// that an artifact consumer can use.
//
// Statement symbols are parser identities for AST nodes. Their records contain
// only the statement spelling and an empty type, while NodeName already carries
// that information. ApplicationSymbol is likewise the fixed appl.fol scope
// anchor used while parsing; ProjectStmt and its Application EntryStmt describe
// the application on the wire. Both remain in the live graph but are omitted
// from the portable graph so they do not masquerade as declarations.
func ArtifactCarriesSymbol(symbol SymbolInfo) bool {
	switch typed := symbol.(type) {
	case *StatmentSymbol, *ApplicationSymbol, *ExpressionSymbol:
		return false
	case *ComponentSymbol:
		// The project wrapper is structural, not a declared component. The
		// ProjectStmt already carries its name and project kind. Real
		// components have their filesystem-selected component kind and remain.
		return typed.Kind != "project"
	}
	if portable, ok := symbol.(*PortableSymbol); ok {
		if portable.Record.SymbolType == string(S_StatmentSymbol) || portable.Record.SymbolType == string(S_ExpressionSymbol) ||
			(portable.Record.SymbolType == string(S_PackageSymbol) && portable.Record.Name == "appl.fol") {
			return false
		}
		if portable.Record.SymbolType == string(S_ComponentSymbol) && portable.Record.Fields["Kind"] == "project" {
			return false
		}
	}
	return true
}

// artifactSymbolTables copies the table index without parser-only symbol IDs.
// The live tables are not mutated: the parser still uses its application anchor
// and statement identities while constructing and checking the tree.
func artifactSymbolTables(tables map[string]*SymbolTable, carried map[string]bool) map[string]*SymbolTable {
	projected := make(map[string]*SymbolTable, len(tables))
	for id, table := range tables {
		if table == nil {
			projected[id] = nil
			continue
		}
		copy := *table
		copy.SymbolIds = copy.SymbolIds[:0:0]
		for _, symbolID := range table.SymbolIds {
			if carried[symbolID] {
				copy.SymbolIds = append(copy.SymbolIds, symbolID)
			}
		}
		copy.SymbolsByName = make(map[string][]string, len(table.SymbolsByName))
		for name, ids := range table.SymbolsByName {
			for _, symbolID := range ids {
				if carried[symbolID] {
					copy.SymbolsByName[name] = append(copy.SymbolsByName[name], symbolID)
				}
			}
			if len(copy.SymbolsByName[name]) == 0 {
				delete(copy.SymbolsByName, name)
			}
		}
		projected[id] = &copy
	}
	return projected
}

// MarshalJSON projects the live interface registry into concrete portable
// records so JSON and a future protobuf schema share one symbol representation.
func (fs FolangSymbols) MarshalJSON() ([]byte, error) {
	records := make(map[string]SymbolRecord, len(fs.SymbolsById))
	carried := make(map[string]bool, len(fs.SymbolsById))
	for id, symbol := range fs.SymbolsById {
		if !ArtifactCarriesSymbol(symbol) {
			continue
		}
		records[id] = ProjectSymbol(symbol)
		carried[id] = true
	}
	return json.Marshal(folangSymbolsWire{
		RootContextID: fs.RootContextId, SymbolTables: artifactSymbolTables(fs.SymboltableMap, carried),
		Contexts: fs.ContextMap, SymbolsByID: records, SurfaceSymbols: fs.SurfaceSymbols,
	})
}

// UnmarshalJSON restores a navigable symbol graph using PortableSymbol records.
func (fs *FolangSymbols) UnmarshalJSON(data []byte) error {
	var wire folangSymbolsWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	fs.RootContextId = wire.RootContextID
	fs.SymboltableMap = wire.SymbolTables
	fs.ContextMap = wire.Contexts
	fs.SurfaceSymbols = wire.SurfaceSymbols
	fs.SymbolsById = make(map[string]SymbolInfo, len(wire.SymbolsByID))
	for id, record := range wire.SymbolsByID {
		if record.SymbolFormatVersion != SymbolFormatVersion {
			return fmt.Errorf("symbol %q uses unsupported format version %d; want %d", id, record.SymbolFormatVersion, SymbolFormatVersion)
		}
		flags, err := hex.DecodeString(record.SymbolFlags)
		if err != nil {
			return fmt.Errorf("symbol %q has invalid symbolFlags: %w", id, err)
		}
		if _, err := DecodeSymbolFlags(record.SymbolFormatVersion, flags); err != nil {
			return fmt.Errorf("symbol %q has invalid symbolFlags: %w", id, err)
		}
		if record.SymbolID == "" {
			record.SymbolID = id
		}
		if record.SymbolID != id {
			return fmt.Errorf("symbol registry key %q disagrees with record id %q", id, record.SymbolID)
		}
		fs.SymbolsById[id] = &PortableSymbol{Record: record}
	}
	for tableID, table := range fs.SymboltableMap {
		if table == nil || table.Id != tableID {
			return fmt.Errorf("invalid symbol table entry %q", tableID)
		}
		for _, id := range table.SymbolIds {
			if fs.GetSymbol(id) == nil {
				return fmt.Errorf("symbol table %q references absent symbol %q", tableID, id)
			}
		}
		for key, ids := range table.SymbolsByName {
			for _, id := range ids {
				if fs.GetSymbol(id) == nil {
					return fmt.Errorf("symbol table %q key %q references absent symbol %q", tableID, key, id)
				}
			}
		}
	}
	if fs.RootContextId != "" && fs.GetContext(fs.RootContextId) == nil {
		return fmt.Errorf("root context %q is absent", fs.RootContextId)
	}
	for contextID, context := range fs.ContextMap {
		if context == nil || context.Id != contextID {
			return fmt.Errorf("invalid context entry %q", contextID)
		}
		if context.SymbolTable_ != "" && fs.GetSymbolTable(context.SymbolTable_) == nil {
			return fmt.Errorf("context %q references absent symbol table %q", contextID, context.SymbolTable_)
		}
	}
	return nil
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
		if type_ == reflect.TypeOf(SymbolDetails{}) && isCoreSymbolField(fieldType.Name) {
			continue
		}
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

func isCoreSymbolField(name string) bool {
	switch name {
	case "SymbolId_", "OwnedContextId", "SymbolType_", "Name_", "State", "Type_", "SymbolTableId":
		return true
	default:
		return false
	}
}

func portableNonBooleanValue(value reflect.Value) any {
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		if value.CanInterface() {
			if info, ok := value.Interface().(SymbolInfo); ok {
				return info.GetSymbolID()
			}
		}
		return portableNonBooleanValue(value.Elem())
	}
	if value.IsValid() && value.Kind() == reflect.Struct && value.CanAddr() && value.Addr().CanInterface() {
		if info, ok := value.Addr().Interface().(SymbolInfo); ok {
			return ProjectSymbol(info)
		}
	}
	switch value.Kind() {
	case reflect.Struct:
		out := map[string]any{}
		type_ := value.Type()
		for i := 0; i < value.NumField(); i++ {
			fieldType := type_.Field(i)
			if fieldType.IsExported() && value.Field(i).CanInterface() {
				out[fieldType.Name] = portableNonBooleanValue(value.Field(i))
			}
		}
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

// MarshalJSON emits only durable symbol references; complete portable records
// live once in the artifact's SymbolsById registry.
func (s SymbolTable) MarshalJSON() ([]byte, error) {
	type wireTable struct {
		ID            string              `json:"Id"`
		ParentID      string              `json:"ParentId"`
		ContextID     string              `json:"ContextId"`
		Prefix        string              `json:"Prefix"`
		SymbolIDs     []string            `json:"SymbolIds"`
		SymbolsByName map[string][]string `json:"SymbolsByName"`
	}
	return json.Marshal(wireTable{s.Id, s.ParentId, s.ContextId, s.Prefix, s.SymbolIds, s.SymbolsByName})
}
