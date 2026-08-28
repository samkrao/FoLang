# Symbol flag layout

This is the normative backend-neutral layout for serialized FoLang symbols. Format version: `1`. Position `P` is byte `P / 8`, bit `P % 8`; bit zero is the least-significant bit of byte zero. JSON writes the canonical bytes as a lowercase hexadecimal `symbolFlags` string. Missing trailing bytes mean false. Positions are append-only; removed positions become reserved and are never reused. Readers reject unsupported format versions. Unknown set bits are preserved by transport and ignored semantically until a newer registry defines them. Backends may use any in-memory bitset, but must decode these bytes without native-word, alignment, ABI, or endianness assumptions.

Primary symbol category remains the stable `symbolType` string in version 1; relationship data such as subtype/supertype targets remains in symbol fields and is not reduced to a flag.

The final column names the Go record type that declares the property; embedded records inherit those flags. The names and positions are wire contracts, while these Go type names are implementation-location guidance.

| Position | Byte | Bit | Flag name | Meaning | Applicable symbol kinds |
|---:|---:|---:|---|---|---|
| 0 | 0 | 0 | `IsInternal` | Existing `IsInternal` symbol property | `ClassSymbol`, `ModuleSymbol`, `SymbolDetails`, `TypeSymbol`, `VariableDetails` |
| 1 | 0 | 1 | `Inferred` | Existing `Inferred` symbol property | `VariableDetails` |
| 2 | 0 | 2 | `Dynamic` | Existing `Dynamic` symbol property | `VariableDetails` |
| 3 | 0 | 3 | `Auto` | Existing `Auto` symbol property | `VariableDetails` |
| 4 | 0 | 4 | `Mutable` | Existing `Mutable` symbol property | `VariableDetails` |
| 5 | 0 | 5 | `IsParam` | Existing `IsParam` symbol property | `VariableDetails` |
| 6 | 0 | 6 | `IsArg` | Existing `IsArg` symbol property | `VariableDetails` |
| 7 | 0 | 7 | `ReturnVar` | Existing `ReturnVar` symbol property | `VariableDetails` |
| 8 | 1 | 0 | `IsSealed` | Existing `IsSealed` symbol property | `ClassSymbol`, `ModuleSymbol`, `PackageSymbol`, `StructSymbol`, `TypeSymbol`, `VariableDetails` |
| 9 | 1 | 1 | `IsPublic` | Existing `IsPublic` symbol property | `ClassSymbol`, `FunctionSymbol`, `ModuleSymbol`, `TypeSymbol`, `VariableDetails` |
| 10 | 1 | 2 | `IsPrivate` | Existing `IsPrivate` symbol property | `ClassSymbol`, `FunctionSymbol`, `ModuleSymbol`, `TypeSymbol`, `VariableDetails` |
| 11 | 1 | 3 | `IsPackageScope` | Existing `IsPackageScope` symbol property | `ClassSymbol`, `FunctionSymbol`, `ModuleSymbol`, `TypeSymbol`, `VariableDetails` |
| 12 | 1 | 4 | `IsFriend` | Existing `IsFriend` symbol property | `VariableDetails` |
| 13 | 1 | 5 | `IsInternalScope` | Existing `IsInternalScope` symbol property | `VariableDetails` |
| 14 | 1 | 6 | `DuckType` | Existing `DuckType` symbol property | `VariableDetails` |
| 15 | 1 | 7 | `HasInitValue` | Existing `HasInitValue` symbol property | `VariableDetails` |
| 16 | 2 | 0 | `LocalBinding` | Existing `LocalBinding` symbol property | `VariableDetails` |
| 17 | 2 | 1 | `ExplicitType` | Existing `ExplicitType` symbol property | `TypeSymbol`, `VariableDetails` |
| 18 | 2 | 2 | `Optional` | Existing `Optional` symbol property | `VariableDetails` |
| 19 | 2 | 3 | `IsInner` | Existing `IsInner` symbol property | `FunctionSymbol`, `VariableDetails` |
| 20 | 2 | 4 | `ThunkVar` | Existing `ThunkVar` symbol property | `VariableDetails` |
| 21 | 2 | 5 | `IsCompound` | Existing `IsCompound` symbol property | `VariableDetails` |
| 22 | 2 | 6 | `IsListType` | Existing `IsListType` symbol property | `VariableDetails` |
| 23 | 2 | 7 | `NamedArg` | Existing `NamedArg` symbol property | `VariableDetails` |
| 24 | 3 | 0 | `VariadicParam` | Existing `VariadicParam` symbol property | `VariableDetails` |
| 25 | 3 | 1 | `DefaultArgs` | Existing `DefaultArgs` symbol property | `VariableDetails` |
| 26 | 3 | 2 | `Discard` | Existing `Discard` symbol property | `VarSymbol` |
| 27 | 3 | 3 | `BindVar` | Existing `BindVar` symbol property | `VarSymbol` |
| 28 | 3 | 4 | `Extern` | Existing `Extern` symbol property | `VarSymbol` |
| 29 | 3 | 5 | `Forward` | Existing `Forward` symbol property | `VarSymbol` |
| 30 | 3 | 6 | `Mangled` | Existing `Mangled` symbol property | `VarSymbol` |
| 31 | 3 | 7 | `Alias` | Existing `Alias` symbol property | `TypeSymbol`, `VarSymbol` |
| 32 | 4 | 0 | `Weak` | Existing `Weak` symbol property | `VarSymbol` |
| 33 | 4 | 1 | `AutoCreate` | Existing `AutoCreate` symbol property | `VarSymbol` |
| 34 | 4 | 2 | `ExistsAssign` | Existing `ExistsAssign` symbol property | `VarSymbol` |
| 35 | 4 | 3 | `IsRaw` | Existing `IsRaw` symbol property | `PointerSymbol` |
| 36 | 4 | 4 | `PtrToConstType` | Existing `PtrToConstType` symbol property | `PointerSymbol` |
| 37 | 4 | 5 | `ConstPtrToType` | Existing `ConstPtrToType` symbol property | `PointerSymbol` |
| 38 | 4 | 6 | `ConstPtrToConstType` | Existing `ConstPtrToConstType` symbol property | `PointerSymbol` |
| 39 | 4 | 7 | `ISFatPointer` | Existing `ISFatPointer` symbol property | `PointerSymbol` |
| 40 | 5 | 0 | `IsShared` | Existing `IsShared` symbol property | `PointerSymbol` |
| 41 | 5 | 1 | `IsWeak` | Existing `IsWeak` symbol property | `PointerSymbol` |
| 42 | 5 | 2 | `IsUnique` | Existing `IsUnique` symbol property | `PointerSymbol` |
| 43 | 5 | 3 | `IsJagged` | Existing `IsJagged` symbol property | `ArraySymbol` |
| 44 | 5 | 4 | `IsMultiDimesion` | Existing `IsMultiDimesion` symbol property | `ArraySymbol` |
| 45 | 5 | 5 | `IsSlice` | Existing `IsSlice` symbol property | `ArraySymbol` |
| 46 | 5 | 6 | `IsZeroLen` | Existing `IsZeroLen` symbol property | `ArraySymbol` |
| 47 | 5 | 7 | `IsZeroDim` | Existing `IsZeroDim` symbol property | `ArraySymbol` |
| 48 | 6 | 0 | `VLA` | Existing `VLA` symbol property | `ArraySymbol` |
| 49 | 6 | 1 | `IsRawArray` | Existing `IsRawArray` symbol property | `ArraySymbol` |
| 50 | 6 | 2 | `ElementLenDecl` | Existing `ElementLenDecl` symbol property | `ArraySymbol` |
| 51 | 6 | 3 | `IsDynamic` | Existing `IsDynamic` symbol property | `ArraySymbol` |
| 52 | 6 | 4 | `SizeFromInit` | Existing `SizeFromInit` symbol property | `ArraySymbol` |
| 53 | 6 | 5 | `Lref` | Existing `Lref` symbol property | `ReferenceSymbol` |
| 54 | 6 | 6 | `Ref` | Existing `Ref` symbol property | `ReferenceSymbol` |
| 55 | 6 | 7 | `Heap` | Existing `Heap` symbol property | `ReferenceSymbol` |
| 56 | 7 | 0 | `Addressop` | Existing `Addressop` symbol property | `AddressSymbol` |
| 57 | 7 | 1 | `Wordtype` | Existing `Wordtype` symbol property | `AddressSymbol` |
| 58 | 7 | 2 | `Curried` | Existing `Curried` symbol property | `FunctionSymbol` |
| 59 | 7 | 3 | `IsGeneric` | Existing `IsGeneric` symbol property | `ClassSymbol`, `FunctionSymbol` |
| 60 | 7 | 4 | `Closure` | Existing `Closure` symbol property | `FunctionSymbol` |
| 61 | 7 | 5 | `Anonymous` | Existing `Anonymous` symbol property | `ClassSymbol`, `FunctionSymbol`, `StructSymbol` |
| 62 | 7 | 6 | `FunctionObject` | Existing `FunctionObject` symbol property | `FunctionSymbol` |
| 63 | 7 | 7 | `NamedParams` | Existing `NamedParams` symbol property | `FunctionSymbol` |
| 64 | 8 | 0 | `Variadic` | Existing `Variadic` symbol property | `FunctionSymbol` |
| 65 | 8 | 1 | `VariantTypeArgs` | Existing `VariantTypeArgs` symbol property | `FunctionSymbol` |
| 66 | 8 | 2 | `OptionalArgs` | Existing `OptionalArgs` symbol property | `FunctionSymbol` |
| 67 | 8 | 3 | `DefaultParams` | Existing `DefaultParams` symbol property | `FunctionSymbol` |
| 68 | 8 | 4 | `FunctionExpression` | Existing `FunctionExpression` symbol property | `FunctionSymbol` |
| 69 | 8 | 5 | `Associated` | Existing `Associated` symbol property | `FunctionSymbol`, `StructSymbol` |
| 70 | 8 | 6 | `Delegate` | Existing `Delegate` symbol property | `FunctionSymbol` |
| 71 | 8 | 7 | `InnerFunction` | Existing `InnerFunction` symbol property | `FunctionSymbol` |
| 72 | 9 | 0 | `FWRF` | Existing `FWRF` symbol property | `FunctionSymbol` |
| 73 | 9 | 1 | `FWPF` | Existing `FWPF` symbol property | `FunctionSymbol` |
| 74 | 9 | 2 | `IsMethod` | Existing `IsMethod` symbol property | `FunctionSymbol` |
| 75 | 9 | 3 | `ClassMethod` | Existing `ClassMethod` symbol property | `FunctionSymbol` |
| 76 | 9 | 4 | `StaticMethod` | Existing `StaticMethod` symbol property | `FunctionSymbol` |
| 77 | 9 | 5 | `InstanceMethod` | Existing `InstanceMethod` symbol property | `FunctionSymbol` |
| 78 | 9 | 6 | `ObjectMethod` | Existing `ObjectMethod` symbol property | `FunctionSymbol` |
| 79 | 9 | 7 | `FunctionChain` | Existing `FunctionChain` symbol property | `FunctionSymbol` |
| 80 | 10 | 0 | `Lazy` | Existing `Lazy` symbol property | `FunctionSymbol` |
| 81 | 10 | 1 | `Eager` | Existing `Eager` symbol property | `FunctionSymbol` |
| 82 | 10 | 2 | `LexicalStaticScope` | Existing `LexicalStaticScope` symbol property | `FunctionSymbol` |
| 83 | 10 | 3 | `DynamicScope` | Existing `DynamicScope` symbol property | `FunctionSymbol` |
| 84 | 10 | 4 | `MixedScope` | Existing `MixedScope` symbol property | `FunctionSymbol` |
| 85 | 10 | 5 | `CContinuation` | Existing `CContinuation` symbol property | `FunctionSymbol` |
| 86 | 10 | 6 | `SRContinuation` | Existing `SRContinuation` symbol property | `FunctionSymbol` |
| 87 | 10 | 7 | `PCContinuation` | Existing `PCContinuation` symbol property | `FunctionSymbol` |
| 88 | 11 | 0 | `TrampolineCC` | Existing `TrampolineCC` symbol property | `FunctionSymbol` |
| 89 | 11 | 1 | `YeildCC` | Existing `YeildCC` symbol property | `FunctionSymbol` |
| 90 | 11 | 2 | `Fiber` | Existing `Fiber` symbol property | `FunctionSymbol` |
| 91 | 11 | 3 | `Thread` | Existing `Thread` symbol property | `FunctionSymbol` |
| 92 | 11 | 4 | `Task` | Existing `Task` symbol property | `FunctionSymbol` |
| 93 | 11 | 5 | `Parallel` | Existing `Parallel` symbol property | `FunctionSymbol` |
| 94 | 11 | 6 | `Concurrent` | Existing `Concurrent` symbol property | `FunctionSymbol` |
| 95 | 11 | 7 | `HScale` | Existing `HScale` symbol property | `FunctionSymbol` |
| 96 | 12 | 0 | `VScale` | Existing `VScale` symbol property | `FunctionSymbol` |
| 97 | 12 | 1 | `Fork` | Existing `Fork` symbol property | `FunctionSymbol` |
| 98 | 12 | 2 | `Spawn` | Existing `Spawn` symbol property | `FunctionSymbol` |
| 99 | 12 | 3 | `Exec` | Existing `Exec` symbol property | `FunctionSymbol` |
| 100 | 12 | 4 | `Process` | Existing `Process` symbol property | `FunctionSymbol` |
| 101 | 12 | 5 | `Async` | Existing `Async` symbol property | `FunctionSymbol` |
| 102 | 12 | 6 | `Channel` | Existing `Channel` symbol property | `FunctionSymbol` |
| 103 | 12 | 7 | `Promise` | Existing `Promise` symbol property | `FunctionSymbol` |
| 104 | 13 | 0 | `Future` | Existing `Future` symbol property | `FunctionSymbol` |
| 105 | 13 | 1 | `Defer` | Existing `Defer` symbol property | `FunctionSymbol` |
| 106 | 13 | 2 | `CPS` | Existing `CPS` symbol property | `FunctionSymbol` |
| 107 | 13 | 3 | `Coroutine` | Existing `Coroutine` symbol property | `FunctionSymbol` |
| 108 | 13 | 4 | `Goroutine` | Existing `Goroutine` symbol property | `FunctionSymbol` |
| 109 | 13 | 5 | `Actor` | Existing `Actor` symbol property | `FunctionSymbol` |
| 110 | 13 | 6 | `Event` | Existing `Event` symbol property | `FunctionSymbol` |
| 111 | 13 | 7 | `Generator` | Existing `Generator` symbol property | `FunctionSymbol` |
| 112 | 14 | 0 | `Iterator` | Existing `Iterator` symbol property | `FunctionSymbol` |
| 113 | 14 | 1 | `Inline` | Existing `Inline` symbol property | `FunctionSymbol` |
| 114 | 14 | 2 | `Lambda` | Existing `Lambda` symbol property | `FunctionSymbol` |
| 115 | 14 | 3 | `Overrloaded` | Existing `Overrloaded` symbol property | `FunctionSymbol` |
| 116 | 14 | 4 | `Overrridden` | Existing `Overrridden` symbol property | `FunctionSymbol` |
| 117 | 14 | 5 | `Abstract` | Existing `Abstract` symbol property | `ClassSymbol`, `FunctionSymbol` |
| 118 | 14 | 6 | `Virtual` | Existing `Virtual` symbol property | `ClassSymbol`, `FunctionSymbol` |
| 119 | 14 | 7 | `Native` | Existing `Native` symbol property | `FunctionSymbol` |
| 120 | 15 | 0 | `ISMachincode` | Existing `ISMachincode` symbol property | `FunctionSymbol` |
| 121 | 15 | 1 | `ISAsm` | Existing `ISAsm` symbol property | `FunctionSymbol` |
| 122 | 15 | 2 | `ISNaked` | Existing `ISNaked` symbol property | `FunctionSymbol` |
| 123 | 15 | 3 | `Issealed` | Existing `Issealed` symbol property | `FunctionSymbol` |
| 124 | 15 | 4 | `IsProtected` | Existing `IsProtected` symbol property | `FunctionSymbol` |
| 125 | 15 | 5 | `IsBody` | Existing `IsBody` symbol property | `FunctionSymbol` |
| 126 | 15 | 6 | `IsTailCall` | Existing `IsTailCall` symbol property | `FunctionSymbol` |
| 127 | 15 | 7 | `OnlyParamTypes` | Existing `OnlyParamTypes` symbol property | `FunctionSymbol` |
| 128 | 16 | 0 | `AsFunctionParam` | Existing `AsFunctionParam` symbol property | `FunctionSymbol` |
| 129 | 16 | 1 | `ParentIsFunc` | Existing `ParentIsFunc` symbol property | `FunctionSymbol` |
| 130 | 16 | 2 | `FunTypeParRet` | Existing `FunTypeParRet` symbol property | `FunctionSymbol` |
| 131 | 16 | 3 | `RestrictedToOverload` | Existing `RestrictedToOverload` symbol property | `FunctionSymbol` |
| 132 | 16 | 4 | `IsExportable` | Existing `IsExportable` symbol property | `FunctionSymbol` |
| 133 | 16 | 5 | `IsMeth` | Existing `IsMeth` symbol property | `FunctionSymbol` |
| 134 | 16 | 6 | `IsRestricted` | Existing `IsRestricted` symbol property | `FunctionSymbol` |
| 135 | 16 | 7 | `Callback` | Existing `Callback` symbol property | `FunctionSymbol` |
| 136 | 17 | 0 | `IsOperator` | Existing `IsOperator` symbol property | `FunctionSymbol` |
| 137 | 17 | 1 | `ISFunctor` | Existing `ISFunctor` symbol property | `TypeclassSymbol` |
| 138 | 17 | 2 | `ISApplicative` | Existing `ISApplicative` symbol property | `TypeclassSymbol` |
| 139 | 17 | 3 | `ISMonad` | Existing `ISMonad` symbol property | `TypeclassSymbol` |
| 140 | 17 | 4 | `ISMonoid` | Existing `ISMonoid` symbol property | `TypeclassSymbol` |
| 141 | 17 | 5 | `ISTransormer` | Existing `ISTransormer` symbol property | `TypeclassSymbol` |
| 142 | 17 | 6 | `IsMonod` | Existing `IsMonod` symbol property | `TypeclassSymbol` |
| 143 | 17 | 7 | `ISFoldeable` | Existing `ISFoldeable` symbol property | `TypeclassSymbol` |
| 144 | 18 | 0 | `IsNamed` | Existing `IsNamed` symbol property | `BlockSymbol` |
| 145 | 18 | 1 | `IsLetForm` | Existing `IsLetForm` symbol property | `FunctionPattern` |
| 146 | 18 | 2 | `Library` | Existing `Library` symbol property | `PackageSymbol` |
| 147 | 18 | 3 | `IsParent` | Existing `IsParent` symbol property | `PackageSymbol` |
| 148 | 18 | 4 | `IsFFI` | Existing `IsFFI` symbol property | `PackageSymbol` |
| 149 | 18 | 5 | `IsSystem` | Existing `IsSystem` symbol property | `PackageSymbol` |
| 150 | 18 | 6 | `AsExpr` | Existing `AsExpr` symbol property | `PackageSymbol`, `TypeSymbol` |
| 151 | 18 | 7 | `IsCodeBlock` | Existing `IsCodeBlock` symbol property | `PackageSymbol` |
| 152 | 19 | 0 | `CaseClass` | Existing `CaseClass` symbol property | `ClassSymbol` |
| 153 | 19 | 1 | `Property` | Existing `Property` symbol property | `ClassSymbol` |
| 154 | 19 | 2 | `InnerClass` | Existing `InnerClass` symbol property | `ClassSymbol` |
| 155 | 19 | 3 | `IsImpl` | Existing `IsImpl` symbol property | `ModuleSymbol` |
| 156 | 19 | 4 | `InnerInterface` | Existing `InnerInterface` symbol property | `InterfaceSymbol` |
| 157 | 19 | 5 | `InnerEnum` | Existing `InnerEnum` symbol property | `EnumSymbol` |
| 158 | 19 | 6 | `CStruct` | Existing `CStruct` symbol property | `StructSymbol` |
| 159 | 19 | 7 | `Embedded` | Existing `Embedded` symbol property | `StructSymbol` |
| 160 | 20 | 0 | `Composed` | Existing `Composed` symbol property | `StructSymbol` |
| 161 | 20 | 1 | `InnerStruct` | Existing `InnerStruct` symbol property | `StructSymbol` |
| 162 | 20 | 2 | `HKType` | Existing `HKType` symbol property | `HokrtlSymbol` |
| 163 | 20 | 3 | `HRType` | Existing `HRType` symbol property | `HokrtlSymbol` |
| 164 | 20 | 4 | `HOType` | Existing `HOType` symbol property | `HokrtlSymbol` |
| 165 | 20 | 5 | `HLType` | Existing `HLType` symbol property | `HokrtlSymbol` |
| 166 | 20 | 6 | `NewType` | Existing `NewType` symbol property | `TypeSymbol` |
| 167 | 20 | 7 | `SubType` | Existing `SubType` symbol property | `TypeSymbol` |
| 168 | 21 | 0 | `SuperType` | Existing `SuperType` symbol property | `TypeSymbol` |
| 169 | 21 | 1 | `DependentType` | Existing `DependentType` symbol property | `TypeSymbol` |
| 170 | 21 | 2 | `RefinementType` | Existing `RefinementType` symbol property | `TypeSymbol` |
| 171 | 21 | 3 | `PredicateType` | Existing `PredicateType` symbol property | `TypeSymbol` |
| 172 | 21 | 4 | `AssociatedType` | Existing `AssociatedType` symbol property | `TypeSymbol` |
| 173 | 21 | 5 | `OpaqueType` | Existing `OpaqueType` symbol property | `TypeSymbol` |
| 174 | 21 | 6 | `FunctionType` | Existing `FunctionType` symbol property | `TypeSymbol` |
| 175 | 21 | 7 | `ForallType` | Existing `ForallType` symbol property | `TypeSymbol` |
| 176 | 22 | 0 | `UDT` | Existing `UDT` symbol property | `TypeSymbol` |
| 177 | 22 | 1 | `ADT` | Existing `ADT` symbol property | `TypeSymbol` |
| 178 | 22 | 2 | `BDT` | Existing `BDT` symbol property | `TypeSymbol` |
| 179 | 22 | 3 | `UnionType` | Existing `UnionType` symbol property | `TypeSymbol` |
| 180 | 22 | 4 | `FunType` | Existing `FunType` symbol property | `TypeSymbol` |
| 181 | 22 | 5 | `AsStmt` | Existing `AsStmt` symbol property | `TypeSymbol` |
| 182 | 22 | 6 | `No_user_name` | Existing `No_user_name` symbol property | `TypeSymbol` |
| 183 | 22 | 7 | `Ephimeral` | Existing `Ephimeral` symbol property | `TypeSymbol` |
| 184 | 23 | 0 | `Hidden` | Existing `Hidden` symbol property | `TypeSymbol` |
| 185 | 23 | 1 | `RtErased` | Existing `RtErased` symbol property | `TypeSymbol` |
| 186 | 23 | 2 | `IsGenericType` | Existing `IsGenericType` symbol property | `TypeSymbol` |
| 187 | 23 | 3 | `Runtime` | Existing `Runtime` symbol property | `AnnotationSymbol` |
| 188 | 23 | 4 | `Compiletime` | Existing `Compiletime` symbol property | `AnnotationSymbol` |
| 189 | 23 | 5 | `Hygeine` | Existing `Hygeine` symbol property | `MacroSymbol` |
| 190 | 23 | 6 | `GenSym` | Existing `GenSym` symbol property | `MacroSymbol` |
| 191 | 23 | 7 | `Escape` | Existing `Escape` symbol property | `MacroSymbol` |
| 192 | 24 | 0 | `Untyped` | Existing `Untyped` symbol property | `TemplateDetails` |
| 193 | 24 | 1 | `Typed` | Existing `Typed` symbol property | `TemplateDetails` |
| 194 | 24 | 2 | `Define` | Existing `Define` symbol property | `OperatorDetails` |
| 195 | 24 | 3 | `Override` | Existing `Override` symbol property | `OperatorDetails` |
| 196 | 24 | 4 | `Overload` | Existing `Overload` symbol property | `OperatorDetails` |
| 197 | 24 | 5 | `Default` | Existing `Default` symbol property | `OperatorDetails` |
| 198 | 24 | 6 | `Provided` | Existing `Provided` symbol property | `OperatorDetails` |
| 199 | 24 | 7 | `Reified` | Existing `Reified` symbol property | `GenericDetails` |
| 200 | 25 | 0 | `Nullable` | Existing `Nullable` symbol property | `GenericTypeParam` |
| 201 | 25 | 1 | `Inclusive` | Existing `Inclusive` symbol property | `GenericTypeParam` |
| 202 | 25 | 2 | `Impredicative` | Existing `Impredicative` symbol property | `GenericTypeParam` |
| 203 | 25 | 3 | `IsPragma` | Existing `IsPragma` symbol property | `DirectivePragmaDetails` |
