package preparser

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"google.golang.org/protobuf/types/known/structpb"
)

const portableTypeField = "$type"
const portableIntegerField = "$integer"

var errInvalidArtifact = errors.New("invalid protobuf artifact")

func encodePortable(value any) (*structpb.Value, error) {
	return encodeReflect(reflect.ValueOf(value))
}

func encodeReflect(value reflect.Value) (*structpb.Value, error) {
	if !value.IsValid() {
		return structpb.NewNullValue(), nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return structpb.NewNullValue(), nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Bool:
		return structpb.NewBoolValue(value.Bool()), nil
	case reflect.String:
		return structpb.NewStringValue(value.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return integerValue(value.Type().String(), strconv.FormatInt(value.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return integerValue(value.Type().String(), strconv.FormatUint(value.Uint(), 10)), nil
	case reflect.Float32, reflect.Float64:
		return structpb.NewNumberValue(value.Float()), nil
	case reflect.Slice, reflect.Array:
		values := make([]*structpb.Value, value.Len())
		for i := 0; i < value.Len(); i++ {
			item, err := encodeReflect(value.Index(i))
			if err != nil {
				return nil, err
			}
			values[i] = item
		}
		return structpb.NewListValue(&structpb.ListValue{Values: values}), nil
	case reflect.Map:
		if value.IsNil() {
			return structpb.NewNullValue(), nil
		}
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("protobuf artifact map key %s is not string-backed", value.Type().Key())
		}
		fields := make(map[string]*structpb.Value, value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			item, err := encodeReflect(iterator.Value())
			if err != nil {
				return nil, err
			}
			fields[iterator.Key().String()] = item
		}
		return structpb.NewStructValue(&structpb.Struct{Fields: fields}), nil
	case reflect.Struct:
		fields := map[string]*structpb.Value{
			portableTypeField: structpb.NewStringValue(portableTypeName(value.Type())),
		}
		typeInfo := value.Type()
		for i := 0; i < value.NumField(); i++ {
			fieldInfo := typeInfo.Field(i)
			if !fieldInfo.IsExported() {
				continue
			}
			fieldValue, err := encodeReflect(value.Field(i))
			if err != nil {
				return nil, fmt.Errorf("encode %s.%s: %w", typeInfo, fieldInfo.Name, err)
			}
			fields[portableFieldName(fieldInfo)] = fieldValue
		}
		return structpb.NewStructValue(&structpb.Struct{Fields: fields}), nil
	default:
		return nil, fmt.Errorf("protobuf artifact cannot encode %s", value.Type())
	}
}

func integerValue(typeName, value string) *structpb.Value {
	return structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
		portableTypeField:    structpb.NewStringValue(typeName),
		portableIntegerField: structpb.NewStringValue(value),
	}})
}

func decodePortable(value *structpb.Value, destination any) error {
	if destination == nil {
		return errInvalidArtifact
	}
	target := reflect.ValueOf(destination)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return fmt.Errorf("protobuf artifact destination must be a non-nil pointer")
	}
	if value == nil {
		return errInvalidArtifact
	}
	return decodeReflect(value, target.Elem())
}

func decodeReflect(value *structpb.Value, target reflect.Value) error {
	if !target.CanSet() {
		return fmt.Errorf("protobuf artifact cannot set %s", target.Type())
	}
	if _, isNull := value.Kind.(*structpb.Value_NullValue); isNull {
		target.SetZero()
		return nil
	}

	if target.Kind() == reflect.Pointer {
		target.Set(reflect.New(target.Type().Elem()))
		return decodeReflect(value, target.Elem())
	}
	if target.Kind() == reflect.Interface {
		decoded, err := decodeInterface(value, target.Type())
		if err != nil {
			return err
		}
		target.Set(decoded)
		return nil
	}

	switch target.Kind() {
	case reflect.Bool:
		if _, ok := value.Kind.(*structpb.Value_BoolValue); !ok {
			return typeMismatch(target.Type(), value)
		}
		target.SetBool(value.GetBoolValue())
	case reflect.String:
		if _, ok := value.Kind.(*structpb.Value_StringValue); !ok {
			return typeMismatch(target.Type(), value)
		}
		target.SetString(value.GetStringValue())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		text, err := encodedInteger(value)
		if err != nil {
			return err
		}
		integer, err := strconv.ParseInt(text, 10, target.Type().Bits())
		if err != nil {
			return err
		}
		target.SetInt(integer)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		text, err := encodedInteger(value)
		if err != nil {
			return err
		}
		integer, err := strconv.ParseUint(text, 10, target.Type().Bits())
		if err != nil {
			return err
		}
		target.SetUint(integer)
	case reflect.Float32, reflect.Float64:
		if _, ok := value.Kind.(*structpb.Value_NumberValue); !ok {
			return typeMismatch(target.Type(), value)
		}
		target.SetFloat(value.GetNumberValue())
	case reflect.Slice:
		list := value.GetListValue()
		if list == nil {
			return typeMismatch(target.Type(), value)
		}
		decoded := reflect.MakeSlice(target.Type(), len(list.Values), len(list.Values))
		for i, item := range list.Values {
			if err := decodeReflect(item, decoded.Index(i)); err != nil {
				return fmt.Errorf("decode %s[%d]: %w", target.Type(), i, err)
			}
		}
		target.Set(decoded)
	case reflect.Array:
		list := value.GetListValue()
		if list == nil || len(list.Values) != target.Len() {
			return typeMismatch(target.Type(), value)
		}
		for i, item := range list.Values {
			if err := decodeReflect(item, target.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		object := value.GetStructValue()
		if object == nil || target.Type().Key().Kind() != reflect.String {
			return typeMismatch(target.Type(), value)
		}
		decoded := reflect.MakeMapWithSize(target.Type(), len(object.Fields))
		for name, item := range object.Fields {
			key := reflect.New(target.Type().Key()).Elem()
			key.SetString(name)
			entry := reflect.New(target.Type().Elem()).Elem()
			if err := decodeReflect(item, entry); err != nil {
				return fmt.Errorf("decode %s[%q]: %w", target.Type(), name, err)
			}
			decoded.SetMapIndex(key, entry)
		}
		target.Set(decoded)
	case reflect.Struct:
		object := value.GetStructValue()
		if object == nil {
			return typeMismatch(target.Type(), value)
		}
		typeInfo := target.Type()
		for i := 0; i < target.NumField(); i++ {
			fieldInfo := typeInfo.Field(i)
			if !fieldInfo.IsExported() {
				continue
			}
			item, ok := object.Fields[portableFieldName(fieldInfo)]
			if !ok {
				continue
			}
			if err := decodeReflect(item, target.Field(i)); err != nil {
				return fmt.Errorf("decode %s.%s: %w", typeInfo, fieldInfo.Name, err)
			}
		}
	default:
		return fmt.Errorf("protobuf artifact cannot decode %s", target.Type())
	}
	return nil
}

func decodeInterface(value *structpb.Value, interfaceType reflect.Type) (reflect.Value, error) {
	if interfaceType.NumMethod() == 0 {
		decoded, err := decodeAny(value)
		if err != nil {
			return reflect.Value{}, err
		}
		if decoded == nil {
			return reflect.Zero(interfaceType), nil
		}
		return reflect.ValueOf(decoded), nil
	}

	object := value.GetStructValue()
	if object == nil {
		return reflect.Value{}, typeMismatch(interfaceType, value)
	}
	tag := object.Fields[portableTypeField].GetStringValue()
	concreteType, ok := portableConcreteTypes[tag]
	if !ok {
		return reflect.Value{}, fmt.Errorf("protobuf artifact has unknown concrete type %q", tag)
	}

	var concrete reflect.Value
	if concreteType.Kind() == reflect.Pointer {
		concrete = reflect.New(concreteType.Elem())
		if err := decodeReflect(value, concrete.Elem()); err != nil {
			return reflect.Value{}, err
		}
	} else {
		concrete = reflect.New(concreteType).Elem()
		if err := decodeReflect(value, concrete); err != nil {
			return reflect.Value{}, err
		}
	}
	if !concrete.Type().AssignableTo(interfaceType) {
		return reflect.Value{}, fmt.Errorf("protobuf artifact type %s does not implement %s", concrete.Type(), interfaceType)
	}
	return concrete, nil
}

func decodeAny(value *structpb.Value) (any, error) {
	switch kind := value.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil, nil
	case *structpb.Value_BoolValue:
		return kind.BoolValue, nil
	case *structpb.Value_StringValue:
		return kind.StringValue, nil
	case *structpb.Value_NumberValue:
		return kind.NumberValue, nil
	case *structpb.Value_ListValue:
		items := make([]any, len(kind.ListValue.Values))
		for i, item := range kind.ListValue.Values {
			decoded, err := decodeAny(item)
			if err != nil {
				return nil, err
			}
			items[i] = decoded
		}
		return items, nil
	case *structpb.Value_StructValue:
		if _, ok := kind.StructValue.Fields[portableIntegerField]; ok {
			return decodeDynamicInteger(kind.StructValue)
		}
		if tag := kind.StructValue.Fields[portableTypeField].GetStringValue(); tag != "" {
			concreteType, ok := portableConcreteTypes[tag]
			if !ok {
				return nil, fmt.Errorf("protobuf artifact has unknown concrete type %q", tag)
			}
			if concreteType.Kind() == reflect.Pointer {
				decoded := reflect.New(concreteType.Elem())
				if err := decodeReflect(value, decoded.Elem()); err != nil {
					return nil, err
				}
				return decoded.Interface(), nil
			}
			decoded := reflect.New(concreteType).Elem()
			if err := decodeReflect(value, decoded); err != nil {
				return nil, err
			}
			return decoded.Interface(), nil
		}
		fields := make(map[string]any, len(kind.StructValue.Fields))
		for name, item := range kind.StructValue.Fields {
			decoded, err := decodeAny(item)
			if err != nil {
				return nil, err
			}
			fields[name] = decoded
		}
		return fields, nil
	default:
		return nil, errInvalidArtifact
	}
}

func decodeDynamicInteger(object *structpb.Struct) (any, error) {
	typeName := object.Fields[portableTypeField].GetStringValue()
	text := object.Fields[portableIntegerField].GetStringValue()
	switch typeName {
	case "int":
		value, err := strconv.ParseInt(text, 10, strconv.IntSize)
		return int(value), err
	case "int8":
		value, err := strconv.ParseInt(text, 10, 8)
		return int8(value), err
	case "int16":
		value, err := strconv.ParseInt(text, 10, 16)
		return int16(value), err
	case "int32":
		value, err := strconv.ParseInt(text, 10, 32)
		return int32(value), err
	case "int64":
		value, err := strconv.ParseInt(text, 10, 64)
		return value, err
	case "uint":
		value, err := strconv.ParseUint(text, 10, strconv.IntSize)
		return uint(value), err
	case "uint8":
		value, err := strconv.ParseUint(text, 10, 8)
		return uint8(value), err
	case "uint16":
		value, err := strconv.ParseUint(text, 10, 16)
		return uint16(value), err
	case "uint32":
		value, err := strconv.ParseUint(text, 10, 32)
		return uint32(value), err
	case "uint64":
		value, err := strconv.ParseUint(text, 10, 64)
		return value, err
	default:
		return nil, fmt.Errorf("protobuf artifact has unsupported integer type %q", typeName)
	}
}

func encodedInteger(value *structpb.Value) (string, error) {
	object := value.GetStructValue()
	if object == nil {
		return "", typeMismatch(reflect.TypeFor[int](), value)
	}
	integer, ok := object.Fields[portableIntegerField]
	if !ok {
		return "", errInvalidArtifact
	}
	return integer.GetStringValue(), nil
}

func typeMismatch(want reflect.Type, value *structpb.Value) error {
	return fmt.Errorf("protobuf artifact value %T cannot decode into %s", value.Kind, want)
}

func portableFieldName(field reflect.StructField) string {
	if tag := strings.Split(field.Tag.Get("json"), ",")[0]; tag != "" && tag != "-" {
		return tag
	}
	var name strings.Builder
	for i, r := range field.Name {
		if unicode.IsUpper(r) && i > 0 {
			name.WriteByte('_')
		}
		name.WriteRune(unicode.ToLower(r))
	}
	return name.String()
}

func portableTypeName(valueType reflect.Type) string {
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	prefix := valueType.PkgPath()
	switch {
	case strings.HasSuffix(prefix, "/ast"):
		prefix = "ast"
	case strings.HasSuffix(prefix, "/context"):
		prefix = "symbol"
	case strings.HasSuffix(prefix, "/helpers"):
		prefix = "source"
	}
	if prefix == "" {
		return valueType.Name()
	}
	return prefix + "." + valueType.Name()
}

func registerPortableTypes(values ...any) map[string]reflect.Type {
	registered := make(map[string]reflect.Type, len(values))
	for _, value := range values {
		valueType := reflect.TypeOf(value)
		registered[portableTypeName(valueType)] = valueType
	}
	return registered
}

var portableConcreteTypes = registerPortableTypes(
	ast.SourceFile{}, ast.MetadataRef{}, ast.Block{},
	ast.Declaration{}, ast.ValueDefinition{}, ast.TypeDefinition{},
	ast.ContextDefinition{}, ast.ParameterDefault{}, ast.CallableDefinition{},
	ast.LiteralExpr{}, ast.TypeValueExpr{}, ast.BindingExpr{}, ast.VariableAccessExpr{},
	ast.MemberAccessExpr{}, ast.IndexExpr{}, ast.CallArgument{}, ast.CallExpr{},
	ast.UnaryExpr{}, ast.BinaryExpr{}, ast.AssignmentExpr{}, ast.CompoundAssignmentExpr{},
	ast.TupleExpr{}, ast.ConstructionElement{}, ast.ConstructionExpr{}, ast.MatchCase{}, ast.MatchExpr{},
	ast.AnonymousFunctionExpr{}, ast.LambdaExpr{}, ast.AnonymousClassExpr{},
	ast.ComprehensionExpr{}, ast.WildcardArgument{},
	ast.SelectionBranch{}, ast.SelectionExpr{},
	ast.EmptyStatement{}, ast.ExpressionStatement{}, ast.MultipleAssignmentStatement{},
	ast.ReturnStatement{}, ast.EnclosingCallableReturnStatement{}, ast.LoopStatement{},
	ast.LockStatement{}, ast.BlockStatement{}, ast.LabeledBlockStatement{},
	ast.BreakStatement{}, ast.ContinueStatement{},
	ast.WildcardPattern{}, ast.BindingPattern{}, ast.LiteralPattern{},
	ast.PatternArgument{}, ast.ConstructorPattern{}, ast.RecordPatternField{},
	ast.RecordPattern{}, ast.TuplePattern{},
	ast.NamedType{}, ast.TypeApplication{}, ast.FunctionType{}, ast.TupleType{}, ast.DerivedType{},
	ast.RefinementTypeExpr{}, ast.PredicateTypeExpr{},

	&symboltable.Context{}, &symboltable.FolContext{},
	&symboltable.SymbolDetails{}, &symboltable.ProgramSymbol{}, &symboltable.ApplicationSymbol{},
	&symboltable.AbstractType{}, &symboltable.BDTtype{}, &symboltable.UDTtype{},
	&symboltable.PredefinedCollections{}, &symboltable.AliasType{}, &symboltable.NewType{},
	&symboltable.SuperType{}, &symboltable.SubType{}, &symboltable.OpaqueType{}, &symboltable.ADTtype{},
	&symboltable.PredicateType{}, &symboltable.AssociatedType{}, &symboltable.DependentType{},
	&symboltable.RefinementType{}, &symboltable.GenericType{}, &symboltable.Hokrltype{},
	&symboltable.ShapeType{}, &symboltable.UninitType{}, &symboltable.KindType{},
	&symboltable.FunctionType{}, &symboltable.DelegateType{}, &symboltable.ParameterizedType{},
	&symboltable.DataType{}, &symboltable.TagType{}, &symboltable.PolymorphicType{},
	&symboltable.Impredicativetypes{}, &symboltable.ArrayType{}, &symboltable.PointerType{},
	&symboltable.ReferenceType{}, &symboltable.AddressType{}, &symboltable.WordType{},
	&symboltable.RangeType{}, &symboltable.ThunkType{}, &symboltable.SliceType{},
	&symboltable.GenericSpecializationType{},
	&symboltable.KindSymbol{}, &symboltable.StructSymbol{}, &symboltable.CStructSymbol{},
	&symboltable.EnumSymbol{}, &symboltable.ModuleSymbol{}, &symboltable.SignatureSymbol{},
	&symboltable.InterfaceSymbol{}, &symboltable.ClassSymbol{}, &symboltable.TypeClassSymbol{},
	&symboltable.InstanceSymbol{}, &symboltable.TraitSymbol{}, &symboltable.MixinSymbol{},
	&symboltable.ComponentSymbol{}, &symboltable.UnitSymbol{}, &symboltable.ExtensionSymbol{},
	&symboltable.ObjectSymbol{}, &symboltable.AnnotationSymbol{}, &symboltable.MatcherSymbol{},
	&symboltable.UnionSymbol{}, &symboltable.BlockSymbol{}, &symboltable.SymbolSymbol{},
	&symboltable.FunctionSymbol{}, &symboltable.ExpressionBodiedFunction{},
	&symboltable.DecoratorSymbol{}, &symboltable.ExtensionMethodSymbol{},
	&symboltable.NativeFunctionSymbol{}, &symboltable.MacroSymbol{}, &symboltable.TemplateSymbol{},
	&symboltable.ExecutionModelSymbol{}, &symboltable.CurryingFunctionSymbol{},
	&symboltable.DeferredFunctionSymbol{}, &symboltable.BindCallableSymbol{},
	&symboltable.VariadicFunctionSymbol{}, &symboltable.NamedParameterFunctionSymbol{},
	&symboltable.OptionalParameterFunctionSymbol{}, &symboltable.DefaultParameterFunctionSymbol{},
	&symboltable.IndexerSymbol{}, &symboltable.OperatorFunctionSymbol{},
	&symboltable.InlineFunctionSymbol{}, &symboltable.LocalFunctionSymbol{},
	&symboltable.NestedFunctionSymbol{}, &symboltable.InnerFunctionSymbol{},
	&symboltable.LifecycleSymbol{}, &symboltable.AssociatedFunction{},
	&symboltable.FunctionObject{}, &symboltable.Callable{}, &symboltable.Variable{},
	&symboltable.Parameter{}, &symboltable.Return{},
	&symboltable.MetaDataApplication{},
	&symboltable.OperatorSymbol{},
	&symboltable.BuiltInProtoTypalProp{}, &symboltable.Literal{},
	&symboltable.ThisProperties{}, &symboltable.CoProperties{},
	&symboltable.ReservedWord{}, &symboltable.LabelSymbol{},
	&symboltable.ChainedMethodSymbol{}, &symboltable.MatchBindings{},
	&symboltable.LambdaExpression{},
)
