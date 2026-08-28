package symboltable

import (
	"encoding/hex"
	"fmt"
	"reflect"
)

// SymbolFormatVersion is the version of the backend-neutral symbol record.
const SymbolFormatVersion = 1

// SymbolFlag describes one permanent bit in the serialized symbol flag layout.
type SymbolFlag struct {
	Position int
	Name     string
}

// SymbolFlagRegistry is append-only. Positions are wire-format constants: never
// reorder or reuse them. FuntionTyoe is the legacy Go field spelling for the
// canonical FunctionType flag at position 174.
var SymbolFlagRegistry = []SymbolFlag{
	{0, "IsInternal"}, {1, "Inferred"}, {2, "Dynamic"}, {3, "Auto"}, {4, "Mutable"}, {5, "IsParam"}, {6, "IsArg"}, {7, "ReturnVar"},
	{8, "IsSealed"}, {9, "IsPublic"}, {10, "IsPrivate"}, {11, "IsPackageScope"}, {12, "IsFriend"}, {13, "IsInternalScope"}, {14, "DuckType"}, {15, "HasInitValue"},
	{16, "LocalBinding"}, {17, "ExplicitType"}, {18, "Optional"}, {19, "IsInner"}, {20, "ThunkVar"}, {21, "IsCompound"}, {22, "IsListType"}, {23, "NamedArg"},
	{24, "VariadicParam"}, {25, "DefaultArgs"}, {26, "Discard"}, {27, "BindVar"}, {28, "Extern"}, {29, "Forward"}, {30, "Mangled"}, {31, "Alias"},
	{32, "Weak"}, {33, "AutoCreate"}, {34, "ExistsAssign"}, {35, "IsRaw"}, {36, "PtrToConstType"}, {37, "ConstPtrToType"}, {38, "ConstPtrToConstType"}, {39, "ISFatPointer"},
	{40, "IsShared"}, {41, "IsWeak"}, {42, "IsUnique"}, {43, "IsJagged"}, {44, "IsMultiDimesion"}, {45, "IsSlice"}, {46, "IsZeroLen"}, {47, "IsZeroDim"},
	{48, "VLA"}, {49, "IsRawArray"}, {50, "ElementLenDecl"}, {51, "IsDynamic"}, {52, "SizeFromInit"}, {53, "Lref"}, {54, "Ref"}, {55, "Heap"},
	{56, "Addressop"}, {57, "Wordtype"}, {58, "Curried"}, {59, "IsGeneric"}, {60, "Closure"}, {61, "Anonymous"}, {62, "FunctionObject"}, {63, "NamedParams"},
	{64, "Variadic"}, {65, "VariantTypeArgs"}, {66, "OptionalArgs"}, {67, "DefaultParams"}, {68, "FunctionExpression"}, {69, "Associated"}, {70, "Delegate"}, {71, "InnerFunction"},
	{72, "FWRF"}, {73, "FWPF"}, {74, "IsMethod"}, {75, "ClassMethod"}, {76, "StaticMethod"}, {77, "InstanceMethod"}, {78, "ObjectMethod"}, {79, "FunctionChain"},
	{80, "Lazy"}, {81, "Eager"}, {82, "LexicalStaticScope"}, {83, "DynamicScope"}, {84, "MixedScope"}, {85, "CContinuation"}, {86, "SRContinuation"}, {87, "PCContinuation"},
	{88, "TrampolineCC"}, {89, "YeildCC"}, {90, "Fiber"}, {91, "Thread"}, {92, "Task"}, {93, "Parallel"}, {94, "Concurrent"}, {95, "HScale"},
	{96, "VScale"}, {97, "Fork"}, {98, "Spawn"}, {99, "Exec"}, {100, "Process"}, {101, "Async"}, {102, "Channel"}, {103, "Promise"},
	{104, "Future"}, {105, "Defer"}, {106, "CPS"}, {107, "Coroutine"}, {108, "Goroutine"}, {109, "Actor"}, {110, "Event"}, {111, "Generator"},
	{112, "Iterator"}, {113, "Inline"}, {114, "Lambda"}, {115, "Overrloaded"}, {116, "Overrridden"}, {117, "Abstract"}, {118, "Virtual"}, {119, "Native"},
	{120, "ISMachincode"}, {121, "ISAsm"}, {122, "ISNaked"}, {123, "Issealed"}, {124, "IsProtected"}, {125, "IsBody"}, {126, "IsTailCall"}, {127, "OnlyParamTypes"},
	{128, "AsFunctionParam"}, {129, "ParentIsFunc"}, {130, "FunTypeParRet"}, {131, "RestrictedToOverload"}, {132, "IsExportable"}, {133, "IsMeth"}, {134, "IsRestricted"}, {135, "Callback"},
	{136, "IsOperator"}, {137, "ISFunctor"}, {138, "ISApplicative"}, {139, "ISMonad"}, {140, "ISMonoid"}, {141, "ISTransormer"}, {142, "IsMonod"}, {143, "ISFoldeable"},
	{144, "IsNamed"}, {145, "IsLetForm"}, {146, "Library"}, {147, "IsParent"}, {148, "IsFFI"}, {149, "IsSystem"}, {150, "AsExpr"}, {151, "IsCodeBlock"},
	{152, "CaseClass"}, {153, "Property"}, {154, "InnerClass"}, {155, "IsImpl"}, {156, "InnerInterface"}, {157, "InnerEnum"}, {158, "CStruct"}, {159, "Embedded"},
	{160, "Composed"}, {161, "InnerStruct"}, {162, "HKType"}, {163, "HRType"}, {164, "HOType"}, {165, "HLType"}, {166, "NewType"}, {167, "SubType"},
	{168, "SuperType"}, {169, "DependentType"}, {170, "RefinementType"}, {171, "PredicateType"}, {172, "AssociatedType"}, {173, "OpaqueType"}, {174, "FunctionType"}, {175, "ForallType"},
	{176, "UDT"}, {177, "ADT"}, {178, "BDT"}, {179, "UnionType"}, {180, "FunType"}, {181, "AsStmt"}, {182, "No_user_name"}, {183, "Ephimeral"},
	{184, "Hidden"}, {185, "RtErased"}, {186, "IsGenericType"}, {187, "Runtime"}, {188, "Compiletime"}, {189, "Hygeine"}, {190, "GenSym"}, {191, "Escape"},
	{192, "Untyped"}, {193, "Typed"}, {194, "Define"}, {195, "Override"}, {196, "Overload"}, {197, "Default"}, {198, "Provided"}, {199, "Reified"},
	{200, "Nullable"}, {201, "Inclusive"}, {202, "Impredicative"}, {203, "IsPragma"},
}

var flagPositionByField = func() map[string]int {
	m := make(map[string]int, len(SymbolFlagRegistry)+2)
	for _, flag := range SymbolFlagRegistry {
		m[flag.Name] = flag.Position
	}
	m["IsInternal_"] = 0
	m["FuntionTyoe"] = 174
	return m
}()

// EncodeSymbolFlags returns the canonical little-bit-numbered byte sequence,
// with trailing zero bytes omitted.
func EncodeSymbolFlags(symbol any) []byte {
	flags := make([]byte, (len(SymbolFlagRegistry)+7)/8)
	encodeBooleanFields(reflect.ValueOf(symbol), flags)
	for len(flags) > 0 && flags[len(flags)-1] == 0 {
		flags = flags[:len(flags)-1]
	}
	return flags
}

func encodeBooleanFields(value reflect.Value, flags []byte) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		encodeBooleanFields(value.Elem(), flags)
		return
	}
	if value.Kind() != reflect.Struct {
		return
	}
	type_ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field, fieldType := value.Field(i), type_.Field(i)
		if field.Kind() == reflect.Bool {
			if position, ok := flagPositionByField[fieldType.Name]; ok && field.Bool() {
				flags[position/8] |= byte(1 << (position % 8))
			}
			continue
		}
		if fieldType.Anonymous {
			encodeBooleanFields(field, flags)
		}
	}
}

// SymbolFlagsHex is the canonical compact JSON/debug representation.
func SymbolFlagsHex(symbol any) string { return hex.EncodeToString(EncodeSymbolFlags(symbol)) }

// DecodeSymbolFlags validates a version and returns set known flags. Unknown set
// bits are preserved by readers but ignored semantically until their registry is known.
func DecodeSymbolFlags(version int, flags []byte) (map[string]bool, error) {
	if version != SymbolFormatVersion {
		return nil, fmt.Errorf("unsupported symbol format version %d (supported: %d)", version, SymbolFormatVersion)
	}
	set := map[string]bool{}
	for _, flag := range SymbolFlagRegistry {
		if flag.Position/8 < len(flags) && flags[flag.Position/8]&(1<<uint(flag.Position%8)) != 0 {
			set[flag.Name] = true
		}
	}
	return set, nil
}
