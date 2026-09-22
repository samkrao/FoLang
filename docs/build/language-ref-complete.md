<p align="center">
  <img src="F.png" width="200" alt="Foλang Logo"/>
</p>


<a id="folang"></a>
[Foλang](https://github.com/samkrao/folang) is a general-purpose programming language designed to be **expressive, consistent, and extensible**, merging functional fluency with object-centric abstractions.




## Normative Status

This document is the authoritative and normative definition of the FoLang programming language.

All FoLang syntax, semantics, type-system rules, name-resolution rules, execution behavior, and other language requirements are defined by this document.

Every addition, correction, clarification, modification, or extension to the FoLang language must be recorded in this document before it becomes part of the language.

A compiler or other language implementation may use any internal architecture, parser, intermediate representation, runtime, or backend. However, to be identified as a conforming FoLang implementation, its externally observable behavior must adhere to the requirements defined in this document.

When an implementation conflicts with this specification, the specification governs unless the discrepancy is formally accepted and incorporated as a specification revision.

Implementation-specific extensions must be clearly identified as extensions and must not be represented as standard FoLang features unless they have been incorporated into this specification.

### Audience and Scope

This document is the normative FoLang language and implementation reference. Its primary audience is frontend/compiler implementers, backend/code-generator implementers, tooling implementers, runtime/library implementers where language contracts apply, and conformance-test authors.

The document defines accepted source forms, semantic relationships, validation requirements, diagnostics, frontend obligations, backend-observable contracts, conformance requirements, and areas intentionally left implementation-defined. It is not a getting-started guide or a tutorial for learning FoLang. Examples are retained when they establish, distinguish, or test normative behavior; pedagogical progression and programmer-oriented introductory material are outside the purpose of this reference.

A backend or tool may use any internal representation or optimization permitted by this specification, but implementation techniques do not redefine FoLang syntax, type relationships, member relationships, or observable semantics.

### Grammar, Semantics, and Examples

The lexical and syntactic grammar defined by this specification is the authoritative definition of the FoLang source forms accepted by the current language profile. Normative semantic rules define the meaning of those forms and any additional validity constraints that are checked after parsing.

Examples are **illustrative only**. The presence or absence of an example does not enable, disable, reserve, or otherwise modify a grammar production, token, declaration form, operator, or semantic rule. Likewise, an inventory or explanatory table does not create new source syntax unless the specification explicitly defines the corresponding grammar or identifies the spelling as an explicitly reserved future form.

For parser and grammar generation, the governing classification is therefore deterministic:

```text
active lexical/syntactic grammar
    -> accepted and parsed according to the applicable grammar rules

explicitly identified reserved/future form
    -> recognized as language-owned where its reserved spelling is defined
    -> rejected with an unsupported-feature diagnostic in the current profile

all other source text
    -> ordinary lexical or syntax error when it does not match the active grammar
```

### Language Evolution and Compatibility

The alpha period permits experimentation, consolidation, implementation, renaming, syntax changes, and removal. Before version 1.0, a proposed structural feature becomes part of the active language only when a specification revision explicitly incorporates it into the current lexical/syntactic grammar and defines its applicable normative semantics. A proposed or future feature that is mentioned only descriptively does not become active grammar.

Nothing is carried into version 1.0 as implicitly reserved or unsupported merely because it appeared in an example, table, inventory, or design discussion. A spelling is reserved while unsupported only when this specification explicitly identifies that spelling or form as reserved/future syntax.

At version 1.0, the **structural FoLang language surface** closes permanently. After version 1.0, no later major or minor release introduces new core grammar forms, hard/contextual keywords, declaration kinds, operator spellings or fixities, or built-in `@co.*` metadata-form names. Existing structural constructs retain the externally observable semantics defined by the 1.0 specification, except for corrections that restore already-stated intent.

The standard-package rule is narrower and separate. After version 1.0, the `co.*` **package/subpackage hierarchy** is frozen, while ordinary declarations inside those existing packages may evolve through later language-provided `.folenc` artifacts that expose the standard package contexts. Adding or updating an ordinary type, unit-level function, method, module, algorithm, data structure, or other declaration inside an existing package is package-API evolution and does not create new grammar or a new package path.

FoLang remains extensible after 1.0 through extension mechanisms already defined by the 1.0 language, including third-party libraries, user-defined annotations and decorators, macros, custom operators, native integration, including FFI, dynamic-runtime facilities, backend integration points, and other explicitly defined extension mechanisms. Frontend, backend, runtime, optimizer, intermediate representation, diagnostics, compilation strategy, memory management, code generation, performance, and hardware support may evolve provided externally observable FoLang semantics remain conforming.

A post-1.0 correction may fix an implementation/specification discrepancy or remove an internal contradiction when doing so restores the already-stated intent of an existing feature. A correction does not introduce a new grammar form, keyword, declaration kind, operator grammar, metadata-form name, or previously unavailable structural capability.


***


## Lexical Profile and Statement Termination

The consolidated FoLang EBNF referenced by [Appendix A](#appendix-a---complete-folang-ebnf-grammar) is the formal lexical and syntactic grammar. The rules below restate source-level constraints needed by parser/frontend implementations and conformance tests without duplicating the complete EBNF in this document.

### Source Encoding and Identifiers

FoLang source text is UTF-8. A U+FEFF byte-order mark is permitted only as the first code point of a source file; U+FEFF anywhere else is an error.

Ordinary FoLang identifiers are ASCII-only. An identifier begins with an ASCII letter, may continue with ASCII letters or decimal digits, and may contain `_` only between two non-empty alphanumeric segments. An identifier cannot begin or end with `_`, cannot contain consecutive underscores, and cannot be the single spelling `_`. The single `_` is a contextual language token whose meanings are defined by the applicable wildcard/discard, filename-derived declaration, parameterized-type-placeholder, and refinement-predicate rules.

After its character sequence is recognized, an ordinary identifier is checked against the reserved-word table. Hard-reserved words are emitted as reserved tokens rather than identifiers.

Examples:

```text
name        valid
myVar2      valid
v1_hr       valid
a_b_c       valid

123hr       invalid: identifier cannot begin with a digit
_x          invalid: identifier cannot begin with underscore
_1          invalid: identifier cannot begin with underscore
a_          invalid: trailing underscore
a__b        invalid: consecutive underscores
_           contextual token, not an ordinary identifier
```

Control labels use a separate apostrophe-prefixed lexical form such as
`'outer`. They are not ordinary identifiers. `'outer:` declares a structured
control label and `'outer` references it; see [Labels and Named Blocks](#labels-and-named-blocks).
A character literal remains distinct because it has a closing apostrophe, for
example `'c'`.

### Numeric Literals

FoLang supports the integer and floating literal families defined by the consolidated EBNF. Numeric digit separators are not part of the current profile, so forms such as `1'000`, `0x1'a`, and `0b1011'0010` are invalid.

A numeric sign is not part of an ordinary numeric literal token. In expressions, unary `+` or `-` is parsed as a prefix operator. Pattern syntax separately permits a leading `+` or `-` before an integer or floating literal.

Integer literals support binary, octal, decimal, and hexadecimal forms together with the suffixes admitted by the grammar. Floating literals support decimal and hexadecimal forms. A radix-point form requires at least one digit on each side of the point: `1.0` and `0.10` are valid, while `1.` and `.10` are invalid. Scientific notation without a radix point, such as `1e5`, remains valid. A backend-conditional floating suffix is accepted only when the selected backend/compiler contract supports the corresponding representation.

### Comments, Whitespace, and Line Breaks

FoLang supports `//` line comments and non-nesting `/* ... */` block comments. A line comment ends before the next CR or LF terminator. A block comment ends at the first following `*/`; an inner `/*` does not nest.

Comments are recognized before ordinary symbolic-run scanning. Spaces, horizontal tabs, form-feed characters, line breaks, and comments are token separators and are discarded between tokens. A line break never terminates a FoLang statement; FoLang has no automatic semicolon insertion.

### Statement and Expression Termination

FoLang uses explicit termination:

```text
simple statement or expression statement    -> terminated by ;
direct block/body                            -> terminated by its closing }
braced expression/literal                    -> } closes the expression, then ; closes its statement
```

A semicolon is required after simple declarations, assignments, compound assignments, calls used as statements, `$=>` callable-result statements, `$->>` iteration-advance statements, `$->|` structured-exit statements, object/collection construction expressions used in a statement, type-alias declarations containing generic instantiations, literal expression statements, forward declarations, and other simple declaration forms.

A direct declaration body or function/method body terminates at its closing `}` and must not be followed by `;`.

A braced **expression** is different from a direct block/body. Object construction and typed map/collection values still require the enclosing statement's semicolon:

```folang
emp := Employee{id: 1, name: "Rao"};
$=> Employee{id: 1};
StringIntMap co.type =
    co.Map(co.string, co.int);
cfg := StringIntMap{"a": 1, "b": 2};
```

Built-in directives, annotations, pragmas, and decorators are independently delimited metadata applications and do not acquire a trailing semicolon merely because they appear on their own source line.

***

## Design Overview

<p align="center">
  <img src="./design.png" alt="Design" width="600" style="max-width:100%;"/>
</p>


FoLang follows a deliberately different approach from conventional programming language designs.
The system is structured to ensure **clear separation of concerns**, **license isolation**, and **extensibility through well-defined integration boundaries**.

***

## Compiler and Backend 

### 1. Frontend

The Frontend is responsible for source-level analysis and semantic processing of FoLang programs.

#### Frontend Components

- Scanner / Lexer
- Parser
- AST / Parse Tree Generator
- Symbol Table Generator
- Installed Standard-Package Loader
- `.folenc` Artifact and Symbol Loader
- Semantic Analyzer

#### Implementation

- Implemented in **Go**
- Uses Go structures internally for AST, symbol-table, and semantic processing.
- The externally consumable frontend output is serialized according to the selected backend's interchange contract.
- The frontend writes that artifact beneath the reserved project-root `build/` domain.
- The backend-supplied contract identifies the supported FoLang/plugin protocol version, HIR schema version, and wire format.
- The backend installs/provides this interchange-contract file in the same installation directory as the FoLang compiler executable.
- The frontend reads that installed backend contract to determine the interchange representation required by the selected backend.
- Canonical `build/` presence, ownership, and source-discovery rules are defined in [Project Layout](#project-layout).

#### Named Diagnostic Registry

Named diagnostics use stable, case-sensitive identifiers shared by the
compiler, IDE tooling, conformance tests, and serialized diagnostic output.
Implementations may add explanatory text, source locations, and internal
numeric identifiers, but they must preserve the registered name.

The registry groups related normative rejection rules under reusable names.
The compiler emits the most specific applicable diagnostic and places the
particular declaration kind, symbol, metadata form, type, relationship, or
effect in its structured details. It must not create a different public name
merely to restate the same rule for another declaration kind. Secondary errors
caused solely by one primary failure should be suppressed or reported as
related notes rather than independent cascades. All entries currently have
severity `Error` and prevent successful compilation.
They are frontend diagnostics, not runtime `co.error` values or effects.

| Diagnostic name | Severity | Phase | Normative use |
|---|---|---|---|
| `Unclassified` | Error | Diagnostic migration | A legacy diagnostic has not yet been assigned a more specific stable name |
| `UnsupportedFeature` | Error | Profile validation | Source uses a recognized feature or reserved future form that the active language profile does not support |
| `UnsupportedBackendFeature` | Error | Backend compatibility | Source requires a representation or facility not supported by the selected backend contract |
| `InvalidIdentifier` | Error | Lexing | An identifier violates FoLang spelling, underscore, reserved-word, or encoding rules |
| `InvalidLiteral` | Error | Lexing/parsing | A scalar literal token violates its normative lexical form |
| `UnexpectedToken` | Error | Parsing | A token cannot occur at the current grammar position |
| `ExpectedToken` | Error | Parsing | A required delimiter, binder, separator, operand, or other token is absent |
| `InvalidSyntax` | Error | Parsing | Source violates a grammar rule for which no more specific registered parser diagnostic applies |
| `UnknownOperator` | Error | Operator-aware scanning | A complete symbolic run is neither built in nor registered in the active operator table |
| `InvalidProjectLayout` | Error | Project validation | A required root, component tree, fixed surface, filename role, or project-kind layout rule is violated |
| `InvalidSourcePlacement` | Error | Source-structure validation | A file, declaration, unit, block form, directive, pragma, or component surface appears in a forbidden source context |
| `InvalidDeclarationForm` | Error | Declaration validation | A declaration has an unsupported head, nesting, body, filename-derived name, classifier combination, or structural shape |
| `ReservedPackageShadowing` | Error | Project validation | Project or dependency content attempts to shadow the installed `co.*` package identity |
| `InvalidImport` | Error | Import resolution | An import has an invalid form, target, alias, visibility, package/component/library role, or source context |
| `InvalidDependency` | Error | Dependency analysis | A dependency crosses a forbidden project, component, capability, or packaged-library direction |
| `DependencyCycle` | Error | Dependency analysis | Effective package, component-surface, or standalone-library dependencies form a cycle |
| `CapabilityViolation` | Error | Capability analysis | Source accesses or enables a facility outside its permitted application, native, dynamic-runtime, or other capability domain |
| `ExportBoundaryViolation` | Error | Boundary analysis | A public/projected/packaged surface exposes a forbidden, inaccessible, private, internal, native, or representation-reachable type |
| `DuplicateDeclaration` | Error | Symbol analysis | Two variables, types, members, aliases, fields, variants, metadata keys, or other non-callable declarations normalize to one symbol in the same namespace |
| `DuplicateCallableSignature` | Error | Symbol/overload analysis | Two callables normalize to the same owner, receiver category, name, and parameter signature, including declarations differing only by result type |
| `UnresolvedSymbol` | Error | Name resolution | A required package, type, value, callable, label, declaration reference, or other symbol cannot be resolved |
| `AmbiguousSymbol` | Error | Name resolution | More than one symbol remains after the applicable qualification and resolution rules |
| `InaccessibleSymbol` | Error | Access analysis | A resolved declaration exists but is not accessible from the requesting context |
| `InvalidReceiver` | Error | Receiver analysis | A value/type/companion/extension receiver is absent, has the wrong category, or does not match its required owner |
| `UnusedImport` | Error | Usage analysis | An import contributes no live resolved use under the strict import-liveness rules |
| `UnusedSymbol` | Error | Usage analysis | A declaration subject to strict liveness is not reachable or used as required by its declaration category |
| `TypeMismatch` | Error | Type checking | Actual and required canonical types, result positions, receiver types, or callable shapes are incompatible |
| `ConstraintViolation` | Error | Constraint checking | A value or type violates a refinement predicate, subtype/supertype set, bound, variance, capability, or other declared constraint |
| `InvalidAssignment` | Error | Type/assignment analysis | An assignment target, source, binding operator, mutability rule, or value-transfer relationship is invalid |
| `Uninitialized` | Error | Flow analysis | A refinement/dependent value is read where a valid assignment is not proven on every reachable path |
| `InvalidNoneUse` | Error | Type/flow analysis | `co.const.none`, `isNone()`, or missing-result none substitution is used with a refinement/dependent position that does not admit none |
| `InvalidDependentIndex` | Error | Dependent-type analysis | A dependent index has an invalid form, resolution, evaluability, sign, or permitted source |
| `DependentTypeMismatch` | Error | Dependent-type analysis | Dependent constructors or corresponding normalized indices do not match where equality is required |
| `InvalidConstruction` | Error | Construction analysis | A struct/class/object construction or initialization path cannot produce a valid readable value |
| `MutationNotAllowed` | Error | Mutability analysis | Source attempts a statically detectable mutation through an immutable, constant, literal, or otherwise non-mutable root |
| `OverloadNotAllowed` | Error | Callable validation | A declaration category or callable identity is not permitted to introduce the requested overload |
| `NoApplicableOverload` | Error | Overload resolution | No visible overload accepts the resolved receiver and argument types |
| `AmbiguousOverload` | Error | Overload resolution | Multiple incomparable most-specific overload candidates remain applicable |
| `InvalidReturn` | Error | Callable validation | A return count, type, or required result production is invalid |
| `MissingImplementation` | Error | Implementation analysis | A required callable body, interface method, abstract/virtual slot, runtime binding, or other implementation is absent |
| `InvalidForwardDeclaration` | Error | Declaration validation | A bodyless declaration has invalid classification, mapping, signature, or forward-declaration metadata |
| `InvalidGenericDeclaration` | Error | Generic analysis | Generic markers, rank, placement, declaration-head form, classifier combination, or parameter structure is invalid |
| `InvalidGenericMapping` | Error | Generic analysis | Generic mapping rows are absent where required, conflicting, cyclic, structurally incompatible, or leave a required marker unresolved |
| `GenericResolutionFailure` | Error | Generic analysis | Generic arguments or result markers cannot be resolved to one consistent permitted assignment |
| `SignatureConformanceFailure` | Error | Contract conformance | A module, instance, handler, interface implementation, or other declaration fails its required structural signature |
| `InvalidAssociatedTypeBinding` | Error | Contract conformance | An associated-type binding is missing, unknown, fixed incompatibly, or violates generic arity, bounds, variance, or referenced signature requirements |
| `InvalidRelationship` | Error | OOP validation | An interface/class/mixin/trait relationship has an invalid kind, target, count, selector, accessibility, or canonical identity |
| `InheritedMemberConflict` | Error | OOP validation | Inherited state or behavior conflicts and the normative merge/selection rules cannot resolve it automatically |
| `InvalidOverride` | Error | OOP validation | An override target, source category, accessibility, sealing rule, or exact signature does not satisfy override rules |
| `InvalidImplementation` | Error | OOP/contract validation | An explicit implementation mapping selects the wrong source, target, method, accessibility, or signature |
| `InvalidLifecycleDeclaration` | Error | Lifecycle validation | `@@new`, `@@init`, `::` invocation, lifecycle metadata, placement, accessibility, or overload/override use violates lifecycle rules |
| `InvalidMemberPolicy` | Error | Member-policy validation | A field or member carries a forbidden constant, immutable, shared, locking, CopyOnWrite, static/class, or related policy |
| `UnresolvedMetadataForm` | Error | Metadata resolution | A user-defined annotation/decorator name cannot be resolved to a valid metadata declaration |
| `InvalidMetadataPlacement` | Error | Metadata validation | An annotation, decorator, directive, or pragma appears in a source location where that metadata category is forbidden |
| `InvalidMetadataTarget` | Error | Metadata validation | A resolved metadata form does not permit the declaration or expression kind to which it is attached |
| `MissingMetadataField` | Error | Metadata validation | A required field of a recognized metadata form is absent |
| `InvalidMetadataField` | Error | Metadata validation | A metadata field is unknown, duplicated, uses an invalid binder, or is forbidden for the selected form/context |
| `InvalidMetadataValue` | Error | Metadata validation | A metadata value has the wrong type, cardinality, range, enumeration value, declaration category, or compile-time status |
| `ConflictingMetadata` | Error | Metadata/classification validation | Two or more individually valid metadata forms cannot legally classify or govern the same declaration together |
| `InvalidOperatorDeclaration` | Error | Operator registration | A custom operator declaration has invalid spelling, fixity, precedence, associativity, arity, attributes, ownership, or component placement |
| `InvalidOperatorOverload` | Error | Operator overload validation | An operator implementation violates operand ownership, receiver, signature, arity/fixity, genericity, accessibility, or result-independent identity rules |
| `InvalidEffectDeclaration` | Error | Effect analysis | `@co.dap.effects`, emitted error types, or effect metadata placement violates the effect declaration rules |
| `InvalidEffectPolicy` | Error | Effect analysis | A call-site `@co.dap.onEffect` record has an invalid effect key, structure, placement, or ordinary/execution-model policy shape |
| `InvalidEffectHandler` | Error | Effect analysis | A handler list is empty, duplicated, inaccessible, non-module, signature-incompatible, or illegally contains local `@co.dap.onEffect` handling |
| `InvalidEffectResolution` | Error | Effect analysis | A resolution is missing, duplicated, unknown, contextually forbidden, or incompatible with the enclosing callable results |
| `InvalidRetryPolicy` | Error | Effect analysis | Retry configuration has an invalid attempt count, exhaustion action, placement, or structure |
| `InvalidExecutionModel` | Error | Execution-model validation | An execution-model declaration or invocation violates its result, effects, submission, boundary, or call-site handling rules |

The registry name is the stable machine-readable identifier. A diagnostic
instance additionally carries at least its severity, compiler phase, human
message, primary source span, zero or more related spans, and structured
details. Parser recovery may emit `UnexpectedToken`, `ExpectedToken`, or the
fallback `InvalidSyntax`; once a valid AST construct exists, semantic analysis
should prefer its more specific registered diagnostic.

Examples of the stable name plus specific message are:

```text
error[DuplicateDeclaration]: variable 'count' is already declared in this scope
error[OverloadNotAllowed]: this callable category does not permit overloading
error[InvalidMetadataPlacement]: @co.ddap.import is valid only in a file preamble
```

#### License

- **GNU General Public License v3 (GPLv3)**

***

### 2. Backend

The Backend is responsible for transforming validated frontend output into executable artifacts.


> **The default backend must be downloaded or built separately. It is not bundled with the frontend binary.**

#### Backend Components

- Intermediate Representation (IR) Generator
- Native Binary Executable Generation

#### Implementation
    
A backend may be implemented in any language. It consumes the validated frontend artifact from the project-root `build/` directory. The artifact encoding is selected by the backend-supplied interchange contract.

During backend installation, that interchange-contract file is placed in the same installation directory as the FoLang compiler executable. The frontend reads the installed contract to determine the compatible FoLang/plugin protocol version, HIR schema version, and wire format required by the selected backend. The exact artifact basename and format-specific schema/version may therefore vary by backend contract, but the artifact location is always beneath `<project-root>/build/`.

#### Runtime-operation handlers

Backend-neutral HIR may contain a resolved runtime-operation identifier originating from an exported standard declaration marked with `@co.dap.implementation`. The identifier states the required semantic operation, not its target-language spelling or implementation. Each backend owns an internal mapping from supported `co.runtime.operation.*` identifiers to backend handlers.

```text
co.runtime.operation.out.println
    -> reference C++ HIR backend handler
    -> generated call to the reference runtime implementation
```

The reference C++ HIR backend may generate a call to a function declared in a separately maintained C++ header and implemented in a separately compiled C++ source/runtime library. Other backends may lower the same operation to different runtime calls, VM instructions, imports, or target-specific code. Backend headers, implementation filenames, source fragments, and native symbol spellings do not appear in the backend-independent FoLang declaration.

The backend interchange contract must identify the runtime-operation contract version or otherwise provide an equivalent compatibility check. Before code generation, the selected backend must reject any required operation for which it has no compatible handler. A backend handler is selected by the resolved operation ID carried in HIR, not by appending arbitrary target-language text obtained from source annotations.

#### Default / Reference Backend

The default FoLang backend is also the **reference backend implementation**. Its purpose is to provide an executable, inspectable example of how the FoLang specification can be implemented and to give backend implementers a concrete behavioural baseline for conformance testing.

The written FoLang specification remains normative. The reference backend demonstrates the required externally observable semantics, but its internal algorithms, allocation strategy, memory-management choices, data structures, optimization level, and performance characteristics are not themselves language requirements unless this specification explicitly says otherwise. If an implementation defect in the reference backend conflicts with the written specification, the written specification takes precedence.

A third-party backend may use a completely different runtime architecture or memory model and may optimize semantics differently, provided that FoLang programs observe behaviour conforming to this specification. Backend implementations may validate their behaviour against the reference backend and the FoLang conformance tests where applicable.

- Backend orchestration is implemented in **Go**
- Code generation target is **C++**
- Uses **Clang** or **GCC** to generate native binaries from generated C++ IR

#### License 

**Third-party backends may use their own licensing terms and implementation choices.** The default backend uses the following license.
**Default backend is not part of the complete compiler binary and is separate**; it must be downloaded or built separately.

- **BSD 3-Clause License**
***

#### Frontend Output Contract


#### Installed Backend Interchange Contract

The selected backend may supply this contract to tell the frontend which FoLang/plugin protocol, HIR schema, and wire format it accepts. During backend installation, the contract file is placed in the same installation directory as the FoLang compiler executable, and the frontend reads it from that installation location. Every frontend build also embeds a complete default backend contract. If `backend-conf.json` is absent, the frontend uses protobuf and that frontend release's embedded current protocol, HIR-schema, and runtime-operations versions; these values are fixed for that binary, not resolved dynamically. A present contract replaces those defaults and must be complete, well-formed, and supported, or compilation fails.

```json
{
  "protocol":           "folang-plugin/1.0",
  "hir_schema":         "folang-hir/1",
  "wire":               "protobuf",
  "runtime_operations": "folang-runtime-operations/1"
}
```



The frontend has one fixed **location** contract and a backend-selected **encoding** contract:

```text
<project-root>/build/
    └── <frontend-artifact>    encoding/version selected by backend contract
```

Rules:

- the frontend/backend interchange artifact is written beneath the reserved root-level `build/` domain;
- each frontend build embeds default protocol, HIR-schema, wire-format, and runtime-operations versions; an absent installed contract selects those fixed defaults, while a present contract must supply every field and be well-formed and supported;
- that contract file resides in the same installation directory as the FoLang compiler executable;
- `wire="protobuf"` in the example above requests Protocol Buffers; the current protobuf representation is a provisional, language-neutral `Value`/`Struct`/`List` artifact tree carrying the logical `folang-hir/1` model, not yet a typed protobuf definition of every HIR node and interface, and a typed schema with node messages and interface `oneof` declarations is required before treating it as the stable third-party-backend schema; another supported contract may request a different compatible wire format/version;
- `runtime_operations` identifies the backend-neutral runtime-operation contract understood by the selected backend;
- backends consume the validated frontend artifact from `build/`;
- canonical filesystem rules for `build/` are defined only in [Project Layout](#project-layout).

***

### Licensing Summary

| Layer    |  Responsibility                           | Implementation                    | License      |
|----------|------------------------------------------|-----------------------------------|--------------|
| Frontend | Parsing and semantic analysis            | Go                                | GPLv3        |
| Backend (default) | IR processing and native code generation | Go (orchestration) + C++ (target) | BSD 3-Clause |



> The copyrightable material in the [FoLang Language Definition and Documentation](#folang-definition-and-documentation-license), including its syntax, grammar, and semantic-rule descriptions, is licensed separately under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
***

### 3. Capability Security Model

FoLang's compiler ships with all language features compiled in but **native capabilities are disabled by default**. The compiler has no hardcoded keys — capability configuration happens entirely at install time. This moves authorization from source code (developer-controlled) to the compiler installation (organization-controlled).

***

#### Capability Domains and Install-Time Gates

| Domain | Features | Default State |
|---|---|---|
| `application` | Ordinary FoLang language/application capabilities, including macros, templates, concurrency/control abstractions, `co.http`, core declarations such as `co.List`, `co.encoding`, `co.crypto`, etc. | ✅ enabled |
| `dynamicvmrt` | application capabilities plus the defined dynamic-runtime / `co.meta` facilities | explicit source capability domain; toolchain policy may restrict availability |
| `native` | Raw pointers/references/addresses, pointer arithmetic, `@co.dap.native`, `co.native`, `co.sys.unsafe`, MMIO, heap allocators, low-level platform/runtime implementation, `co.sys.ffi`, extern declarations/types, foreign symbols, calling conventions, linkage, C/native ABI work, and ABI-compatible `cstruct` values | 🔒 disabled by default — requires install-time configuration |

`packaged` is an exposure model using application capabilities, not a separate privileged capability domain.

***

### Variable Syntax

A FoLang variable declaration places the identifier first and the declared type second:

```folang
name co.string = "SomeName";
count co.int;
```

The general source shape is:

```text
identifier type [= initializer] ;
```

`identifier` must satisfy the FoLang identifier rules. `type` may be a built-in type, a user-defined type, or a named specialization/derived type. An initializer, when present, must be assignable to the declared type.

FoLang also provides declaration operators with separate semantics:

```folang
name := "SomeName";    // declare and infer one static type
value ::= 10;          // declare a dynamic binding
cached ?= load();      // define/infer when absent; otherwise assign compatibly
```

These operators are declaration/binding forms, not type expressions.

***

### Types

For reference organization, FoLang types are grouped into four broad families:

1. built-in data types;
2. built-in collection/protocol types;
3. user-defined types;
4. derived and specialized types.

These categories organize the standard surface; they do not introduce separate runtime type systems.

#### Built-in Types

| Type | Purpose |
|---|---|
| `co.string` | Built-in string type. |
| `co.int` | Built-in integer type. |
| `co.bit` | Built-in bit type. |
| `co.double` | Built-in double-precision floating type. |
| `co.float` | Built-in floating type. |
| `co.long` | Built-in long integer type. |
| `co.byte` | Built-in byte type. |
| `co.char` | Built-in character type. |
| `co.any` | General language-provided value supertype where permitted. |
| `co.bool` | Built-in Boolean type. |
| `co.void` | Built-in no-value/void type where the applicable construct permits it. |
| `co.value` | Language/runtime value representation used by value/snapshot-oriented operations. |
| `co.untyped` | Explicitly untyped value representation used by facilities such as macros and custom matchers. |
| `co.word` | Built-in machine-word-oriented type family root. |
| `co.MatchBindings` | Binding container returned by custom matcher protocol implementations. |
| `co.number` | Built-in numeric type/category used by numeric constraints and APIs. |
| `co.uninit` | Lifecycle-only uninitialized-object state used by class construction. |
| `co.error` | Standard recoverable-error interface and first-error result contract. Every emitted effect object is a class instance satisfying this interface. |
| `co.AbstractError` | Standard mixin supplying common `co.error` state and behavior. |
| `co.literal` | Language/runtime representation of literal objects. |
| `co.operator` | Operator-declaration type/kind valid only in the project operator-component context. |
| `co.delegate` | Delegate declaration/type facility for one or more registered callable targets. |
| `co.condition` | Language condition object used by condition, loop, and value-selection machinery; it is not merely an alias for `co.bool`. |

#### Built-in Collection and Protocol Types

| Name | Purpose |
|---|---|
| `co.List` | Ordered list collection. |
| `co.Set` | Set collection. |
| `co.Map` | Key/value collection. |
| `co.Tree` | Tree collection. |
| `co.Trie` | Trie collection. |
| `co.Array` | Array collection abstraction. |
| `co.Tuple` | Tuple value type. |
| `co.Comparable` | Standard comparison protocol/type. |
| `co.Stack` | Stack collection. |
| `co.Queue` | Queue collection. |
| `co.StructObject` | Runtime/type descriptor surface for struct objects where provided by the standard package. |
| `co.ClassObject` | Runtime/type descriptor surface for class objects where provided by the standard package. |
| `co.ModuleObject` | Runtime/type descriptor surface for module objects where provided by the standard package. |
| `co.InstanceObject` | Runtime/type descriptor surface for instance objects where provided by the standard package. |
| `co.ObjectObject` | Runtime/type descriptor surface for `co.object` objects where provided by the standard package. |
| `co.Matrix` | Matrix collection/type abstraction. |

#### User-Defined Types

User-defined types are introduced through:

1. built-in declaration kinds such as `co.struct`, `co.class`, `co.interface`, `co.module`, and related kinds; or
2. named `co.type` declarations and the type constructors/specializations described below.

A source declaration is not considered a distinct user-defined type merely because it introduces a variable or value.

#### Derived Types

FoLang provides named derived types built from existing types. The current families include:

1. pointers;
   - ordinary pointers;
   - fat pointers;
2. arrays;
   - single-dimensional;
   - multidimensional;
   - jagged;
   - zero-dimensional;
   - zero-length;
   - variable-length;
   - initializer-sized forms where defined;
3. thunks;
4. slices;
5. ranges;
6. references;
   - ordinary references;
   - lvalue references;
   - heap references;
7. addresses;
8. word-representation specializations.

Derived types are named through `co.type` declarations before ordinary variable use.

***

### Built-in Kinds

| Kind | Purpose |
|---|---|
| `co.struct` | Value-oriented aggregate declaration with fields and associated behavior under the struct rules. |
| `co.cstruct` | ABI-oriented value aggregate whose layout and boundary rules are defined by the native/ABI contract. |
| `co.class` | Concrete reference-semantic object type supporting instance state, inheritance/composition, encapsulation, overriding, lifecycle operations, and dynamic dispatch. |
| `co.interface` | Pure behavioral contract; interface methods are abstract requirements. |
| `co.union` | Union declaration kind. |
| `co.object` | One named singleton object that may own constants, immutable/global/shared state, locks, annotations, or support state associated with other declarations. |
| `co.instance` | Typeclass/contract instance declaration kind. |
| `co.matcher` | Custom matcher declaration kind used by the pattern-matching protocol. |
| `co.loader` | Loader declaration kind where the applicable runtime/package rules permit it. |
| `co.trait` | Reusable abstract composition contract that may provide concrete/default behavior and abstract/virtual requirements according to trait rules. |
| `co.mixin` | Reusable abstract composition unit that may provide state/behavior and abstract/virtual requirements according to mixin rules. |
| `co.extension` | Reusable implemented behavior composed with supported target types without class inheritance. |
| `co.typeclass` | Typeclass contract declaration kind. |
| `co.module` | Named singleton module satisfying its declared module/signature contracts. |
| `co.unit` | Stateless source grouping. Ordinary units merge into the package namespace; `*.comp.unit.fol` contributes associated behavior to its target struct. A unit does not own a source-visible context object. |
| `co.block` | Language block kind where block-kind values/AST classification are required. |
| `co.kind` | Kind-level declaration/value facility, primarily for metaprogramming and AST/kind classification. |
| `co.signature` | Structural module/signature contract declaration kind. |
| `co.function` | Function/callable declaration kind used by the callable model. |
| `co.enum` | Closed tagged ADT declaration. Parameterized states are compiler-provided state functions returning the enclosing enum type; zero-parameter states are state values. |
| `co.symbol` | Language/compiler symbol representation used by AST/compiler-facing facilities. |
| `co.component` | Structural component/surface declaration valid in component surface files and standardized component locations. |

***

### Type Specializations and Type Constructors

| Type form | Purpose |
|---|---|
| `co.variants(...)` | Defines a closed parameterized variant family. Entries may reference declaration type parameters. |
| `co.tag(...)` | Runtime type/value package used where a value must carry its concrete runtime type descriptor. |
| `co.hokrlt` | Type-valued holder used where a type itself is carried as a value/object. |
| `co.newtype` | Introduces a type identity distinct from its representation/base type. |
| `co.opaquetype` | Introduces an opaque wrapper identity with the one-way representation/assignment rules defined below. |
| `co.subtype` | Type-level set/range selecting strict subtypes of the referenced type. |
| `co.supertype` | Type-level set/range selecting strict supertypes of the referenced type. |
| `co.dependentType(...)` | RHS type-expression constructor used by a `co.type` declaration to define a value-indexed dependent type family. It is not a declaration kind or callable result kind. |
| `co.polymorphic(...)` | RHS type-expression constructor defining a named polymorphic type from a binder set and a type body. |
| `co.refinementType` | Base type restricted by a predicate over candidate values. |
| `co.associatedType` | Associated type placeholder/binding used by signature/module contracts. |
| `co.predicateType` | Type-level predicate/filter over candidate types. |
| `co.data(...)` | Concrete closed ADT constructor whose payload types are already concrete/resolved. It is not parameterized by declaration-head type parameters. |
| `co.type` | General named type declaration kind. |
| `co.generic` | Reserved/future generic-type constructor; it is not part of the active generic declaration model unless explicitly enabled by the grammar/profile. |
| `co.shape` | Shape/type-expression facility for callable/type shapes where defined by the grammar and semantic rules. |

`co.variants(...)`, `co.data(...)`, `co.dependentType(...)`, and `co.polymorphic(...)` are not interchangeable:

```text
co.variants(...)      parameterized closed variant family
co.data(...)          concrete closed ADT
co.dependentType(...) value-indexed dependent type family
co.polymorphic(...)   explicitly bound polymorphic type
```

***

### Rules of FoLang Programs

A FoLang application does not declare a `main` method. The compiler discovers the application from the project root and uses the fixed entry file:

```text
<project-root>/src/appl.fol
```

`appl.fol` may contain the entry-file constructs permitted by the application-entry grammar, including:

- pragmas;
- import directives and aliases;
- metadata forms valid in entry-file context;
- variable declarations and assignments;
- expressions and calls;
- named non-UDT type definitions/type specializations allowed by the entry grammar;
- conditions and loops;
- value-producing conditional selection;
- pattern matching and destructuring;
- comprehensions and other explicitly permitted entry expressions.

`appl.fol` can therefore serve as a complete single-source application when no additional application packages are required. In a full application it remains the entry source while additional package source lives under `src/`.

***

### FoLang Reserved Words and Context Sigils

The hard-reserved words are:

```text
co
this
fΦλ
```

`$` is the compiler-owned current-context sigil. It is not an identifier and cannot be declared or redefined by source.

`fΦλ` (`f` = U+0066, `Φ` = U+03A6, `λ` = U+03BB) is the permanently reserved private standard-package root used by privileged standard-library/bootstrap source. Ordinary project source consumes the projected public `co.*` surface instead.

`this` denotes the nearest applicable non-callable object context according to the context rules.

***

### FoLang Operators and Language Tokens

#### Arithmetic Operators

```text
+  -  *  /  %  **
```

#### Logical Operators

```text
&&  ||  !
```

#### Bitwise Operators

```text
&  |  ^
```

#### Comparison and Type-Relation Operators

```text
==  !=  <  >  <=  >=  <:  :>
```

`<:` denotes subtype relation and `:>` denotes supertype relation. These relation operators are non-associative.

#### Assignment and Declaration Operators

```text
=  ?=  :=  ::=  +=  -=  *=  /=  %=  **=  &=  ^=  |=
```

#### Range, Function, Dispatch, and Structural Tokens

```text
..  ...  <..  ..<  <..<
=>>  =>  ->
<-   ->>  ->|
::   @@
@    #    ~    $
( )  { }  [ ]
_    `    ?
:    ;    ,    .
"    '    \
```

The exact lexical/syntactic role of each spelling is defined by the consolidated EBNF and the corresponding semantic section.

`$=>`, `$->>`, and `$->|` are contextual control productions formed from the `$` context sigil and the applicable control token. The former `this =>`, `this ^=>`, `this ->>`, and `this ->|` control spellings are not part of the current model.

#### Pre-declared Operator Glyphs

```text
∪  ∩
```

Project-specific custom operators are declared only through the operator-component mechanism.

***

### Standard Operator Examples

```folang
left  co.int = 10;
right co.int = 3;

notEqual co.bool = left != right;

bitsAnd co.int = 6 & 3;
bitsOr  co.int = 6 | 3;
bitsXor co.int = 6 ^ 3;

mulAssign co.int = 6;
mulAssign *= 3;

divAssign co.int = 18;
divAssign /= 3;

modAssign co.int = 17;
modAssign %= 5;

powAssign co.int = 2;
powAssign **= 3;

andAssign co.int = 6;
andAssign &= 3;

xorAssign co.int = 6;
xorAssign ^= 3;

orAssign co.int = 6;
orAssign |= 3;
```

***

### Large-Scale Applications, Libraries, and Components

FoLang supports both small single-source applications and large projects split across packages, libraries, and source components.

The principal organization forms are:

1. application source packages;
2. standalone libraries (`.folenc`);
3. source components;
4. the installed standard library projected as `co.*`.

These forms affect source placement, dependency direction, capability boundaries, and API exposure. They do not define separate language semantics.

#### Application Packages

Application packages are directories beneath:

```text
<project-root>/src/
```

For example:

```text
payroll/
└── src/
    ├── appl.fol
    └── hr/
        └── emp/
```

`hr` is a top-level application package and `hr.emp` is a subpackage. Package identity follows the directory/package rules; no additional package declaration syntax is required merely to create the directory hierarchy.

`src/appl.fol` remains the application entry source.

#### Libraries

A standalone library has source/package organization analogous to an application but exposes its public surface through `src/component.fol` rather than `src/appl.fol`.

The surface file defines what clients may consume. Internal library packages remain hidden unless projected by the library surface rules.

FoLang capability domains relevant to library/project boundaries are:

- `application`;
- `dynamicvmrt`;
- `native`.

`packaged` is an exposure/projection model, not a separate privileged capability domain.

A built standalone library is represented by a `.folenc` artifact and is normally placed beneath:

```text
<project-root>/lib/
```

for consumption by a project.

##### Packaged Exports

Packaged export surfaces project selected internal APIs directly through the export/projection mechanism rather than through the ordinary API-surface model.

The FoLang standard library uses controlled projection from its private canonical namespace into the public `co.*` namespace.

#### Components

Components provide library-like isolation while remaining in source form inside the project rather than being prebuilt `.folenc` libraries.

Components live beneath:

```text
<project-root>/components/
```

Their exact standardized subdirectories and surface files are defined by the project-layout grammar.

Component rules include:

1. a standardized component kind has its designated project location;
2. component exports are checked for use/liveness according to the component rules;
3. peer components do not directly import/reference each other; interaction is mediated through the application/package/library surfaces permitted by the dependency model.

#### Operator Components

Operator components are project-specific components used to declare custom operators.

They are not general-purpose libraries and are not exported as ordinary reusable component/library APIs. Operator declarations are valid only in the dedicated operator-component source context.

***

### Importing Packages, Components, and Libraries

FoLang uses the built-in directive:

```folang
@co.ddap.import(...)
```

to import application packages, libraries, and components.

#### Importing a Package

```folang
@co.ddap.import(package="hr.employee", as="emp")
```

When `as=` is present, source accesses the imported package through that alias:

```folang
emp.EmployeeService.find(1001);
```

When `as=` is omitted, the complete imported package path is used. Omission of `as=` does not implicitly invent a short alias.

#### Importing a Library

```folang
@co.ddap.import(library="hrlib", as="hr")
```

`hrlib` resolves to the corresponding library artifact, such as:

```text
<project-root>/lib/hrlib.folenc
```

subject to the project/library resolution rules.

#### Importing a Component

```folang
@co.ddap.import(component="native", as="native")
```

A component import resolves the project component surface identified by the component kind/name defined by the project-layout rules.

***

### Symbol Lookup in FoLang

Unqualified lexical lookup follows the active symbol-table scope first and then crosses named-definition context boundaries only after the current lexical scopes are exhausted.

Conceptually:

```text
1. Search the SymbolTable active at the use site.
2. Follow same-scope declaration-order ParentId links as required.
3. Follow LexicalParentId to enclosing lexical scopes and repeat.
4. When the current Context's lexical scopes are exhausted:
     a. move to Context.ParentId;
     b. resume from Context.ParentCtxSymbolTableId,
        the exact enclosing visibility point at which the child Context branched.
5. Repeat until the permitted lexical/context chain is exhausted.
6. Apply imports/aliases and any other lookup domain allowed by the current ResolutionPolicy.
7. If no valid declaration resolves, report UnresolvedSymbol.
```

Qualified lookup does not perform arbitrary fallback:

- a package/library/component qualifier resolves within that imported/projected context;
- another explicit context qualifier resolves within that context according to its access rules;
- failure to find a valid member in the selected context is an error.

See [Contexts, Symbol Tables, and Symbols](#contexts-symbol-tables-and-symbols).

***

### Types in Detail

Most ordinary built-in scalar and collection types behave as their declared standard-package contracts specify. The following types need additional language-level explanation.

#### HOKRLT Type

```folang
x co.hokrlt = co.int;
```

`co.hokrlt` carries a type as a value/object in contexts that require type-valued data.

It is not the declaration kind used to define a new source type; named source types continue to use `co.type` and the applicable type constructors.

#### Value Type

`co.value` is the language/runtime value representation used by value/snapshot-oriented facilities where data is carried independently of its original object identity.

It should not be treated as a synonym for every ordinary source type.

#### Literal Type

`co.literal` represents literal values/objects in compiler/runtime facilities that need the literal as data rather than merely its evaluated source value.

#### Uninitialized Type

`co.uninit` is used by the class lifecycle model. Allocation/lifecycle operations may temporarily produce an uninitialized class object before a successful `@@init` establishes a readable instance.

It is not the universal representation of an ordinary uninitialized local variable; ordinary none/uninitialized rules are defined separately.

#### Untyped Type

`co.untyped` explicitly denotes data for which the ordinary static value type is not carried by the source-level contract.

Its principal uses include metaprogramming and the custom matcher protocol, where a pattern or AST-like value may be inspected dynamically.

#### `co.MatchBindings`

`co.MatchBindings` is the binding container returned by the custom matcher protocol.

Conceptually it contains zero or more named match bindings produced by a successful custom match. A binding name is unique within one result. Bound values may have unrelated concrete types, so implementations may internally represent each entry with its value together with the value's concrete type descriptor.

Bindings become visible only for the selected successful match case. The exact construction/manipulation API is provided by the standard package and matcher contract.

#### Condition Type

`co.condition` is the language condition object used by condition, loop, and value-selection machinery.

It is richer than treating every control construct as syntactic sugar around a raw `co.bool`; the standard condition operations define the applicable `then`, `when`, `default`, and `loop` behavior.

#### Operator Type

`co.operator` is the declaration facility for project-defined operators.

Its use is restricted to the dedicated operator component. A `co.operator` declaration in any other source context is rejected.

#### Error and AbstractError

`co.error` is the standard recoverable-error interface used by FoLang effects.

`co.AbstractError` is the standard reusable mixin that provides common `co.error` state/behavior. User-defined recoverable error classes normally satisfy `co.error` and may compose `co.AbstractError`.

***

### Type Specializations in Detail

#### Type Alias

```folang
SomeInt co.type = co.int;

EmpId  co.type = co.string;
DeptId co.type = co.string;
```

A `co.type` alias denotes the same underlying type identity when the declaration is an alias form.

Aliases are useful for shortening long qualified type names and for giving domain-significant names without introducing a new type identity.

```folang
empId EmpId = "E-100";
deptId DeptId = empId;     // valid: same aliased underlying type

someString co.string = deptId; // valid
```

An alias is therefore not a protection against accidental interchange. Use a distinct/opaque type form when that distinction is required.

#### Opaque Types

```folang
EmpId  co.opaquetype = co.int;
DeptId co.opaquetype = co.int;

someInt co.int = 20;
empId EmpId = 10;
deptId DeptId;

deptId = empId;   // invalid: different opaque identities
deptId = 20;      // valid from permitted representation/base input
deptId = someInt; // valid where the opaque-type input rule permits the base type
someInt = deptId; // invalid: opaque value does not implicitly expose its base type
```

An opaque type has its own identity while permitting only the explicitly defined representation/base-direction conversions. Two opaque types with the same representation are not interchangeable.

Opaque types are useful when source should accept validated/approved base values but must prevent accidental interchange of domain identities.

#### New Types

```folang
SomeType co.newtype = co.int;

x SomeType;
y co.int = 20;

x = y; // invalid without an explicitly permitted conversion
y = x; // invalid without an explicitly permitted conversion
```

`co.newtype` creates a type identity distinct from its base/representation type. Ordinary assignment does not treat the new type and base type as interchangeable.

#### Type Unions

```folang
NumberOrText co.type = co.int | co.string;
```

This is a type-level union/predicate-style type expression: the resulting type position admits one of the selected types.

It is **not** by itself a value-level tagged ADT and does not manufacture variant tags. Closed value alternatives with explicit states use `co.variants(...)`, `co.data(...)`, or an enum as applicable.

#### Supertypes

```folang
EmployeeSupers co.supertype = some.ContractEmployee;
```

`co.supertype` denotes the strict supertypes of the referenced type according to FoLang's type-relation rules. The referenced base type itself is excluded unless it is explicitly reintroduced through another type expression.

#### Subtypes

```folang
EmployeeSubs co.subtype = some.Employee;
```

`co.subtype` denotes the strict subtypes of the referenced type. The referenced base type itself is excluded unless explicitly reintroduced.

To admit both the base and the strict relation set, combine them explicitly:

```folang
EmployeeOrSubtype co.type = EmployeeSubs | some.Employee;
```

#### Refinement Types

```folang
Percentage co.refinementType =
    (co.int).where(_ >= 0 && _ <= 100);

p Percentage = 200; // compile-time rejection when the value is statically known
```

A refinement type restricts the values admitted by a base type using a Boolean predicate over the candidate value `_`.

Refinement-type storage is non-none and follows the definite-initialization rules defined for refinement/dependent values.

#### Dependent Types

```folang
Vector(n) co.type =
    co.dependentType(
        co.int->([n])
    );

v3 Vector(3) = Vector(3){1, 2, 3};
v4 Vector(4) = Vector(4){1, 2, 3, 4};
```

A dependent type includes value indices in type identity. Therefore:

```text
Vector(3) != Vector(4)
```

even though both families ultimately contain `co.int` elements.

A refinement type restricts which **values** of a base type are valid. A dependent type makes one or more compile-time/index values part of the **type identity** itself.

Path-dependent result types may refer to a value/type path when the applicable grammar and type-resolution rules permit it:

```folang
identity(x co.int)->(x.type) = x;
```

#### Predicate Types

```folang
SomeType co.predicateType =
    co.type.where(
        candidate =>
            candidate == co.int ||
            candidate == co.string
    );
```

A predicate type filters candidate **types**, not candidate runtime values.

Predicate types are especially useful in generic bounds and other type positions that need semantic constraints richer than one nominal base type. They are not limited exclusively to generics.

#### Polymorphic Types

```folang
SomeFArg co.type =
    co.polymorphic({T}, (T, T)->(T));
```

`T` is a type binder owned by the polymorphic type declaration. Each valid instantiation binds `T` to one type for that instantiation; the binder is not supplied by an unrelated `co.generic` declaration.

When the polymorphic body is callable, an ordinary named function may implement that named callable contract directly:

```folang
identity SomeFArg(value, other) = value;
```

subject to the callable shape and generic-resolution rules.

#### Parameterized Variant Types

```folang
Option(T) co.type =
    co.variants(
        Some(T),
        None
    );
```

`T` is a **type parameter** of the type family.

`Some(T)` is a parameterized state whose payload type depends on `T`; `None` is a zero-payload state. Concrete applications such as `Option(co.int)` are ordinary type applications.

#### Associated Types

In a signature:

```folang
T co.associatedType;
```

and in a module satisfying that signature:

```folang
T co.associatedType = co.int;
```

An associated type is a type member required by a signature and supplied by the implementing/satisfying module.

Associated types provide type-level variation for module/signature contracts without making the module itself a generic declaration.

#### Data Types

```folang
SelectedValue co.type =
    co.data(
        StringValue(co.string),
        BoolValue(co.bool)
    );
```

`co.data(...)` defines a **concrete closed ADT**. Its state payload types are already concrete/resolved at the declaration site.

Parameterized closed families belong to `co.variants(...)`; a declaration-head parameterized `co.data(...)` form is not the intended model.

#### Tag Values

```folang
co.tag(co.string, "Hello")
```

`co.tag` packages a runtime value together with the concrete type descriptor associated with that value.

It is an open runtime tagging mechanism, not a closed ADT declaration facility. It is useful where heterogeneous runtime values must retain their concrete type information, including matcher/runtime-introspection scenarios.

#### Kind Types

```folang
blockOrMacro co.kind = block | macro;
```

`co.kind` represents alternatives among language/compiler declaration/AST kinds rather than ordinary user value types.

It is primarily useful in macros and other metaprogramming facilities that inspect or transform program structure.

#### Generic Type Constructor

```folang
SomeType co.type = co.generic(T);
```

`co.generic(...)` is reserved/future syntax unless the active language profile explicitly enables it.

Current FoLang generic declarations use the generic declaration/binder mechanisms defined by the generic rules rather than relying on this constructor.

#### Function Types

```folang
IntBinary co.type =
    (co.int, co.int)->(co.int);
```

A callable type may be used for variables, parameters, and results that hold/reference named function objects:

```folang
add IntBinary(a, b) = a + b;

operation IntBinary = add;
result := operation(10, 20);
```

FoLang does not require or introduce a general inline function-expression syntax for parameter/result positions. Named functions are first-class callable objects; restricted lambdas remain the separate callback-expression mechanism where explicitly allowed.

#### Delegate Types

```folang
SomeDelegate co.delegate =
    (a co.int, b co.int)->(co.int, co.int);
```

A delegate is a callable registration/invocation facility rather than merely an alias for a function type.

```folang
someDelegate = myFunc;
someDelegate(10, 20);

someDelegate += mySecondFun;
someDelegate(10, 20);
```

The exact invocation/result combination rules are defined by the delegate contract. Delegates are distinct from ordinary function chaining (`=>>`).

#### Derived Types

```folang
IntRef       co.type = co.int->(&);
IntLValueRef co.type = co.int->(&&);
IntHeapRef   co.type = co.int->(~);
IntAddress   co.type = co.int->(@);
IntThunk     co.type = co.int->(^);
IntSlice     co.type = co.int->([:]);

ThreeInts         co.type = co.int->([3]);
InferredInts      co.type = co.int->([]);
InferredGrid      co.type = co.int->([,]);
ZeroLengthArray   co.type = co.int->([0]);
ZeroDimArray      co.type = co.int->([.]);
JaggedArray       co.type = co.int->([][]);
VariableLenArray  co.type = co.int->([...]);

IntPtr     co.type = co.int->(*);
IntDblPtr  co.type = co.int->(**);
IntDeepPtr co.type = co.int->(*****);

IntRange co.type = co.int->(..);
```

The arrow-tail syntax is part of the type expression. A variable subsequently uses the named derived type rather than repeating the derivation inline.

***

### Contexts, Symbol Tables, and Symbols

FoLang distinguishes **named-definition contexts** from **lexical symbol-table scopes**.

- A named definition owns a `Context` and therefore participates in `$` / `$->parent` navigation.
- An anonymous or labelled block creates a lexical child `SymbolTable` but no new source-visible Context.
- An ordinary declaration introduces a `SymbolInfo` entry into the active SymbolTable.
- `co.unit` contributes declarations to its package/companion owner and does not create a source-visible context object merely because it is a source grouping.

##### Context

```text
Context {
    ParentId:                string,   // nearest enclosing named-definition/structural Context
    ParentCtxSymbolTableId: string,   // exact enclosing visibility table at the definition branch point
    Id:                      string,   // unique Context ID

    ImportedContextIds:      { <alias>: <context-id> },

    Prefix:                  string,
    ContextType_:            string,
    SymbolTables_:           [string], // lexical tables owned by this Context; at least the root table
    ChildCtxIds:             [string], // direct child named-definition Context IDs
    ResolutionPolicy:        string,
    OwnerSymbolId:           string    // named definition owning this Context; empty for structural roots
}
```

`ParentId` is a named-definition/structural ownership relationship. Anonymous and labelled block scopes are not inserted into this Context chain.

`ParentCtxSymbolTableId` records the exact lexical visibility point in the parent Context at which this Context was created. This allows a nested named function declared inside a block to capture declarations visible at that definition point even though the block itself owns no Context.

##### Symbol Tables

```text
SymbolTable {
    Id:              string,   // unique SymbolTable ID
    ParentId:        string,   // previous declaration-order segment in the same lexical scope
    LexicalParentId: string,   // enclosing lexical-scope table active at scope entry
    ContextId:       string,   // owning named-definition/structural Context
    Prefix:          string,

    SymbolIds:       [ <symbol-id> ],
    SymbolsByName:   { <declaration-key>: [ <symbol-id> ] }
}
```

`ParentId` and `LexicalParentId` are intentionally different:

```text
ParentId
    earlier declaration-order segment in the same lexical scope

LexicalParentId
    immediately enclosing lexical scope
```

Multiple anonymous/labelled block SymbolTables may therefore share the same `ContextId`.

##### Symbols

```text
SymbolInfo {
    GetSymbolType() -> string
    GetSymbolID() -> string
    GetContextID() -> string
    GetType() -> string
    GetName() -> string
    IsInternal() -> bool
    SetOwnedContextID(string)
    GetSymbolTableID() -> string
    Anchor() -> string
}
```

Common symbol data:

```text
SymbolDetails {
    SymbolId_        string
    OwnedContextId   context
    SymbolType_      string
    Name_            string
    IsInternal_      bool
    Type_            string
    SymbolTableId    string
    ResolutionState_ string // "resolved" | "unresolved" | "partially_resolved"
}
```

`OwnedContextId` is nonempty only when the declaration represented by the symbol owns a Context.

`SymbolTableId` identifies the table in which the symbol is defined. `Anchor()` returns the source-visibility anchor used by later resolution/semantic passes.

The source-level contextual model is therefore:

```text
named definition
    -> Context + root lexical SymbolTable
    -> establishes $

anonymous/labelled block
    -> child lexical SymbolTable only
    -> does not establish a new $

ordinary declaration
    -> SymbolInfo in the active SymbolTable
```



## Appendix A - Complete FoLang EBNF Grammar

The standalone consolidated EBNF referenced below is the normative lexical and syntactic grammar for FoLang. The prose sections of this reference define semantics and parser-validity constraints without maintaining a second embedded copy of the grammar.

```ebnf
(*
   FoLang EBNF Grammar — current alpha profile

   Consolidated from:
     docs/language-ref.md
     SHA-256: c9a9b6dadfb61b04325b1c2f4169a99ef1444b9856b23ca70ee592b8ffda85ef
     Bytes: 711896
     Lines: 16269

   Authority:
     The language reference is normative. This grammar is a syntactic
     consolidation of the source forms defined there.

   Notation:
     =       defines a production
     ;       ends a production
     |       alternative
     [ ... ] optional
     { ... } zero or more
     ( ... ) grouping
     "..."   terminal text
     ? ... ? lexical or context-sensitive condition described in prose

   Parser/lexer layering:
     - Source text is UTF-8.
     - U+FEFF is permitted only as the first source code point.
     - white-space/comments are discarded between tokens.
     - newlines never terminate statements.
     - the parser consumes the token stream produced by the lexer.

   Filesystem-selected parser roles:
     src/appl.fol
         -> application-entry-file

     src/component.fol
     components/application/component.fol
     components/native/component.fol
     components/dynamicvmrt/component.fol
     components/packaged/component.fol
     components/operators/component.fol
         -> component-surface-file

     ordinary package files
         <Name>.fol
         <Fragment>.unit.fol
         <StructName>.comp.unit.fol
         -> the matching package-source-file alternative selected from the
            externally validated filename class

     All FoLang source under src/ and components/ uses this common grammar.
     Filesystem placement supplies the structural role/component kind and the
     semantic validation context. lib/*.folenc is deserialized rather than
     source-parsed.

   Current-alpha rules that are deliberately explicit here:
     - A named function block body requires "=" before the body.
     - An anonymous function literal does not use "=" between signature/body.
     - Direct declaration/function/block bodies end at "}" and cannot take ";".
     - Braced expressions/literals still require the enclosing statement ";".
     - Object construction is Type{field: value, ...}.
     - A type's arrow tail "T->( ... )" carries two structural meanings that
       the base and tail shape separate: a derivation applied to T or the result
       list of a function type. Generic and parameterized type applications use
       the uniform Type(...) postfix form. Type arguments are positional and
       bind in the generic declaration's parameter order:
           co.int->([5])                              derivation
           co.int->(&, meta={type=out})               derivation, attributes
           (co.int)->(co.int)                    function type
           co.List(co.string)                  positional type args
           co.Map(co.string, co.int)          key, value
           co.Array(2, co.int, [2,4])              bound values
       An argument's value may be a type, an integer, or a bracketed list,
       which is what the Array and Matrix declarations bind.
     - Generic arguments are positional; `name=value` remains exclusive to
       metadata and derivation attributes.
     - A generic collection application is first named by a concrete type alias.
       That alias is followed directly by its literal body:
           StringList{"A","B","C"}
           StringIntMap{"A": 1, "B": 2}
           IntSet{1,2,3}
       The resolved type determines the entry interpretation.
       builtin-collection-type-name records the reference's Builtin Collections
       registry, which is the closed set of built-in names admissible in that
       prefix position.
     - There is no untyped composite literal. Every runtime list, array, set,
       map, struct, or class value names its concrete type and uses `{ ... }`.
       Bare `{ ... }` and `[ ... ]` are not primary expressions. Bracketed lists
       remain available only where a type production explicitly admits a
       dependent value such as array dimensions. See "Canonical Object and
       Collection Construction" in the language reference.
     - .match() means automatic matcher selection;
       .match(expr) selects an explicit matcher. Bare .match is invalid.
     - Control-chain vocabulary uses .then(X) for one-shot conditional
       selection, .loop(block) for single-condition repeated execution,
       .otherwise(condition) only for another selection condition, and
       .default(X) only for a terminal selection fallback. A .loop(block) form
       has exactly one condition and cannot be followed by .otherwise(condition)
       or .default(X). Element iteration is owned by .each(...) itself: its
       explicit-binding form takes index/key, value, and the per-element action
       directly, while its single-argument form takes a callable callback. An
       .each(...) call cannot be followed by .loop(...). There is no conditionless
       .otherwise form and no .do control verb in the current alpha profile.
     - Built-in precedence follows "Built-In Operator Parse Table" in the
       language reference.
     - The built-in type-relation operators <: and :> are binary infix
       relational operators at precedence 350. Relational and equality
       expressions are non-associative and therefore admit at most one
       operator at their respective precedence levels.
     - A co.predicateType declaration has a dedicated named predicate
       binder inside co.type.where(name => expression). This binder form
       is not a general lambda-expression and does not relax the collection-only
       placement rule for |...| => ... lambdas.
     - The only current-alpha pre-declared operator glyphs are ∪ and ∩. They
       are active binary infix operators at precedence 500, left associative;
       missing overload implementations fail during resolution, not parsing.
     - reserved-future-operator-fixity is a diagnostic-recognition production:
       encountering an explicitly specification-reserved future fixity in the
       current alpha profile produces an unsupported-feature parser error.
     - Examples never enable, disable, or reserve syntax. Active syntax comes
       from this grammar; reserved-future treatment applies only to spellings or
       forms the language reference explicitly identifies as future/reserved.
     - All built-in @co.* directives, annotations, pragmas and decorators use
       the single generic annotation application shape, but the complete @co.*
       name must first match the compiler's predefined built-in metadata
       registry. An unregistered @co.* name is a parser error. Once a known
       form is recognized, every supplied field/argument is collected and
       preserved; fields the frontend already understands may be validated,
       while unknown/unhandled fields of that known form do not block parsing
       or frontend artifact generation.
     - Every named metadata field/attribute and every co.operator
       declaration attribute uses "=". The same binder applies at every
       metadata nesting level. The ":" token remains the entry separator for
       ordinary object values and runtime map literals; it is never a
       declarative attribute binder.
     - For a class with ordered direct class parents, `this->parent` and its
       `this->super` alias select the primary parent. The type-named plural form
       `this->parents[SomeParent]` aliases `this->classes[SomeParent]`.
     - `classes[Type]`, `mixins[Type]`, `traits[Type]`, and `interfaces[Type]`
       are contextual compile-time relationship selectors available through
       `this`. In a class method `this` denotes the current class/type; in an
       instance method it denotes the current instance. The key is a complete
       type name or import-alias-based type name that resolves to a direct entry
       of the matching ordered @co.dap.oops field. Numeric, string, computed,
       and runtime type-object keys are invalid. The selector categories are
       not runtime collections.
     - Accessible inherited class/mixin fields are grouped by identifier.
       Matching canonical types merge into one child instance-state slot;
       mismatching canonical types are a semantic error. Field-level constant,
       immutable, Shared, locking, and CopyOnWrite policies are rejected on
       class and mixin storage before merging. The broader accessible level is
       selected as public > protected > internal, while private fields do not
       enter the child-visible merge.
     - A co.class contains ordinary mutable per-instance fields. An
       Immutable, Shared, or CopyOnWrite policy may target the complete class
       instance graph, but a selected class-owned field cannot be an
       independent policy root. Class-associated constant, immutable, global,
       shared, or locked state belongs to a named co.object.
     - A co.object is one named singleton declaration and requires
       ->(for=Target) or ->(for=[Target, ...]). An annotation target denotes an
       annotation implementation; a homogeneous non-empty class-target list
       denotes an explicitly associated support object. The association is
       non-owning, non-structural, non-transitive, and never becomes a managed
       reference edge merely through for=.
     - For methods, repeated paths to one canonical declaration deduplicate.
       Distinct matching concrete methods within one relationship category use
       the first source in that ordered list. Matching concrete methods across
       categories require an explicit child resolution. Matching abstract
       obligations may share one implementation, and matching mandatory
       virtual slots may share one override.
     - `forall` is a contextual spelling recognized only in the value of a
       co.type declaration. Higher-rank parameters and results use the
       resulting named polymorphic type; functions and anonymous functions do
       not introduce forall binders directly.
     - A co.type declaration may have a parameterized declaration head,
       as in Option(T) co.type or Vector(n) co.type. Declaration parameters
       are classified from their semantic use in the RHS. A type-position use
       makes a type parameter; a dependent index/value use under an outermost
       co.dependentType(...) RHS makes a value parameter. co.dependentType is
       an RHS type-expression constructor, not a declaration or result kind.
     - In a signature, `Name co.type;` is an ordinary abstract type slot.
       `T co.associatedType;` is narrower: T must parameterize at least one
       required co.type alias whose matching-module definition instantiates
       a generic container. The module repeats co.associatedType when
       binding T. An unused associated type is invalid.
     - Typeclass shape=(F(_)) metadata and F(A)/F(B) contract applications
       remain a context-bounded higher-kinded exception and are not general
       type-alias syntax.

   Context-sensitive/semantic requirements intentionally remain outside EBNF:
     filename canonicalization and package ownership; project layout and
     reachability; component/project-kind placement; import target
     uniqueness/mutual exclusion; projected-library capability restrictions;
     interface/signature/typeclass conformance;
     instance placement/liveness; overload resolution; dependent-index name
     resolution; generic-argument binding by declared positional order;
     whether a type prefix on a
     collection body names a collection type; built-in metadata field schemas
     beyond frontend-known checks; custom annotation/decorator symbol
     resolution; annotation target/type checks; operator metadata uniqueness,
     ranges and arity/fixity compatibility; operator ownership normalization,
     where an operator belonging to or extending a generic type is associated
     with that type's canonical declaration identity rather than with operator-level
     generic parameters; rejection of co.dap.operator + co.dap.generic on the same
     function-shaped declaration; and all type/runtime semantics.
     Predicate-type binder scope and immutability, the requirement that its
     body resolve to co.bool, type-object membership, canonical type
     identity, and the subtype/supertype meaning of <: and :> are likewise
     semantic checks outside the context-free grammar.
     An omitted initializer ordinarily places that storage location in the
     universal co.const.none state. Refinement-type and concrete dependent-type
     values are non-none exceptions: they cannot store, receive, pass, return,
     or otherwise admit co.const.none. Their declarations may omit an
     initializer, but then denote definitely-uninitialized storage with no
     readable language-level value. Flow-sensitive definite-initialization
     analysis must prove a valid assignment on every reachable path before
     every value-producing access; otherwise the compiler reports
     Uninitialized. Uninitialized is the canonical case-sensitive name in the
     language's named diagnostic registry; messages, locations, and internal
     numeric identifiers may vary without changing that name. The complete
     registry is normative in the language reference and groups lexical,
     parser, project, symbol, type, generic, OOP, metadata, operator, effect,
     execution-model, usage, and backend rejection families. Diagnostic-name
     selection is semantic metadata rather than a context-free production.
     Parser recovery uses UnexpectedToken, ExpectedToken, or fallback
     InvalidSyntax; once a valid AST construct exists, later phases select the
     most specific applicable registered diagnostic and suppress derivative
     cascades. A simple assignment target is not a read. The intrinsic
     isNone() operation is semantically invalid for statically resolved
     refinement-type and dependent-type values. A construct whose missing-
     result behavior would substitute co.const.none into such a position is
     invalid, including applicable continue, return, return_with_error, and
     execution-model failure paths.
     Aggregate construction is checked per path. Every refinement/dependent
     struct field must be initialized before the struct value is returned or
     read. Class @@new may yield co.uninit, but every successful @@init
     path must initialize each refinement/dependent field before the instance
     becomes readable. Each @@init overload is analysed independently;
     multiple @@new overloads do not merge initialization proofs.
     For ordinary types that admit none, none is not a singleton and has no
     stable observable identity: a runtime may reuse or rematerialize its
     representation between accesses. Only intrinsic value.isNone() reliably
     observes the state and returns co.const.true or co.const.false. Equality,
     inequality, identity, ordering, hashing, and other observations involving
     none are semantically unspecified unless a construct defines explicit
     none behavior. The backend must not expose raw host-memory contents or
     backend undefined behavior while implementing this indeterminate
     language-level state. Optional-argument omission remains a separate state
     and is not co.const.none.
     Every co.class declaration has class construction semantics. FoLang
     has no abstract or unconstructable class category. co.dap.sealed on a
     class prevents its selection as a parent but does not prevent constructing
     that class; on a method it prevents overriding but not invocation. A
     co.mixin is a distinct composition declaration kind with no
     independent class-instance construction operation, not an unconstructable
     class.
     A closed co.variants(...) result is the recommended, non-mandatory
     representation for expected domain alternatives such as Found/NotFound.
     Such a declared None variant is an ordinary initialized variant value
     and is distinct from co.const.none. No general rule requires a variant
     destination merely because resolution=continue may substitute
     co.const.none; tooling may issue only an advisory diagnostic for ordinary
     none-admitting types. Refinement/dependent result positions are the strict
     exception and reject the none substitution. Continue never manufactures
     a user-declared variant constructor.
     Object-association target resolution, homogeneous target-kind checking,
     duplicate-target rejection, singleton lifetime, access rules, and the
     prohibition against propagating class inheritance or class-instance
     object policies through a for= association are semantic checks outside
     the context-free grammar. Rejection of field-level policy metadata on
     class/mixin storage and rejection of a class-owned selected field as an
     independent makeImmutable/makeShared/copyOnWrite policy root are likewise
     semantic checks.
     Every recoverable effect is an instance of a class satisfying the standard
     co.error interface. co.AbstractError is the standard
     co.mixin supplying common error state and concrete behavior; custom
     error classes normally compose that mixin while declaring co.error as
     an interface. The mixin is not a base class and has no independent
     class-instance construction semantics. Exact matching composed methods
     may satisfy interface slots; explicit source mappings use co.dap.implement on the consuming
     class. co.dap.implementation remains the distinct runtime-operation
     marker. co.runtime.operation.effect.emit is the backend-neutral
     emission operation and accepts only co.error. Recoverable backend
     failures must be translated to such class instances. Fatal failures do not
     satisfy co.error and cannot participate in effect metadata, handling,
     resolution, or propagation.
     Effect declaration and handling use separate placements. co.dap.effects
     is legal only on ordinary callable declarations/definitions and requires
     one emits list advertising known effects that may leave that callable. It
     is invalid on co.dap.executionmodel declarations. Every emits item must be
     a class satisfying co.error. The list is open and non-exhaustive:
     absence of co.dap.effects advertises no explicit effects but does not
     prohibit undeclared effects from propagating.
     co.dap.onEffect is illegal as declaration metadata and is parsed only by
     effect-handled-call-expression immediately before one call expression.
     Each direct qualified-name field of co.dap.onEffect must resolve to a valid
     effect type; it need not appear in the callee's open advertised set. Each
     ordinary-call record must contain exactly one singular resolution equal to
     continue, retry, return, return_with_error, or propagate. The optional
     handlers field must be a non-empty ordered list of accessible caller-side
     module values matching co.EffectHandler. For a call whose resolved
     callee is a co.dap.executionmodel declaration, each record instead requires
     a non-empty handlers list and forbids resolution; return_with_error is
     implicit and fixed by the language. Any explicit resolution is invalid.
     That signature constrains only handle(error co.error)->(); each
     concrete handle implementation may independently carry co.dap.effects,
     and both its advertised and unexpected effects may escape the handler.
     co.dap.onEffect is forbidden throughout an effect-handler module,
     including call sites in private handler functions, so a handler cannot
     start a nested local effect-handling pipeline.
     Their handle(error co.error)->() functions execute synchronously in
     list order before the resolution. Ordinary calls use the receiving caller
     context; execution-model calls use the context observing the effect before
     completion is published.
     If handler H emits recoverable effect E1 while handling original effect
     E0, remaining handlers are skipped and E0's selected resolution is
     cancelled. The standard library creates a co.GenericError class
     instance satisfying co.error and wrapping E0, E1, H, and available
     causal/source/runtime diagnostic context. At an ordinary call
     this wrapper propagates from the enclosing caller; at an execution-model
     boundary it is returned through the ordinary error-result position and
     otherwise unproduced none-admitting results receive co.const.none. Such a
     missing-result path is invalid for refinement/dependent positions. The wrapper is not
     reprocessed by the same handler list. GenericError is not created for
     normal propagate or return_with_error paths, which preserve E0, and a
     fatal handler failure terminates immediately without wrapping.
     retry requires retry={max_attempts=PositiveCompileTimeInteger,
     on_exhausted=TerminalResolution}; TerminalResolution is continue, return,
     return_with_error, or propagate. Receiver and argument expressions are
     evaluated once and their values are reused by retry. continue substitutes
     co.const.none for each missing call result. return and return_with_error
     likewise use co.const.none for otherwise unproduced caller result slots;
     return_with_error is checked against the enclosing caller's error result.
     Propagate and unmatched effects leave the caller whether or not they were
     explicitly advertised; statically known escaping effects are included in
     computed outgoing symbol metadata. Every co.dap.executionmodel declaration
     must expose exactly one co.error-compatible result and forms an effect
     boundary: its first observed unhandled recoverable effect is converted to
     that ordinary result, unmatched effects receive the same conversion without
     handlers, and no recoverable effect propagates to the submitting caller.
     Known internal effects may remain implementation metadata but are not
     advertised outward. For concurrent/parallel execution, first observed may
     depend on scheduling unless a model defines deterministic priority.
     There is no effects wrapper, resolutions list, or invoke resolution in
     co.dap.onEffect.
     co.dap.defer provides no injected error or completion-status binding; it
     registers explicitly parameterized completion work for LIFO execution on
     every enclosing-function exit after registration is reached.
*)


(* ====================================================================== *)

(* 1. Source roots and externally selected file forms *)

(* ====================================================================== *)

compilation-unit = package-source-file
                 | application-entry-file
                 | component-surface-file ;

source-filename = companion-unit-filename
                | ordinary-unit-filename
                | application-entry-filename
                | component-surface-filename
                | ordinary-source-filename ;

companion-unit-filename = filename-identifier, ".comp.unit.fol" ;

ordinary-unit-filename = filename-identifier, ".unit.fol" ;

application-entry-filename = "appl.fol" ;

component-surface-filename = "component.fol" ;

ordinary-source-filename = filename-identifier, ".fol" ;

filename-identifier = identifier-head, { "_", identifier-segment } ;

package-source-file = package-primary-source-file
                    | ordinary-unit-source-file
                    | companion-unit-source-file ;

package-primary-source-file = file-preamble, primary-declaration ;

ordinary-unit-source-file = file-preamble, unit-declaration ;

companion-unit-source-file = file-preamble, unit-declaration ;

application-entry-file = file-preamble, { entry-item } ;

component-surface-file = file-preamble, component-declaration ;

file-preamble = { file-directive } ;

(* The file preamble is the only syntactic position for language-owned
   DIRECTIVE/PRAGMA metadata. It is parsed before the primary top-level
   declaration (or before the first non-metadata entry item in src/appl.fol).
   Directives are recorded on the source-file/top-level semantic context; they
   do not introduce lexical names, scopes, or symbol-table entries. A directive
   may affect the file's primary declaration when its semantic contract says so,
   without becoming a declaration annotation or lexical member.

   File directives and pragmas use the generic metadata application shape.
   The zero-width category guard consults the predefined built-in metadata
   registry. Unknown @co.* names fail during parsing; unknown fields of a known
   form remain collectable/preservable. *)
file-directive = import-directive
               | alias-directive
               | use-directive
               | dynamic-runtime-directive
               | dynamic-dispatch-directive
               | other-file-metadata-directive ;

other-file-metadata-directive =
    annotation, file-directive-category-guard ;

file-directive-category-guard =
    ? zero-width parser condition: the just-parsed metadata name is registered
      as a built-in DIRECTIVE or PRAGMA, is not one of the specifically guarded
      directive forms above, and is permitted in this file context; custom
      annotation/decorator applications are not standalone file directives ? ;

entry-item = type-declaration
           | let-function-pattern-clause
           | entry-statement ;

entry-statement = variable-declaration
                | inferred-variable-declaration
                | grouped-variable-declaration
                | multiple-assignment-statement
                | expression-statement
                | empty-statement ;

primary-declaration = struct-declaration
                    | cstruct-declaration
                    | enum-declaration
                    | union-declaration
                    | class-declaration
                    | trait-declaration
                    | mixin-declaration
                    | interface-declaration
                    | signature-declaration
                    | module-declaration
                    | typeclass-declaration
                    | object-declaration
                    | instance-declaration
                    | matcher-instance-declaration
                    | extension-declaration ;


(* ====================================================================== *)

(* 2. Directives, annotations, and metadata *)

(* ====================================================================== *)

annotations = { declaration-metadata } ;

declaration-metadata = annotation, declaration-metadata-category-guard ;

declaration-metadata-category-guard =
    ? zero-width parser condition: for a built-in co.* metadata name, the
      registry category is ANNOTATION or DECORATOR; DIRECTIVE and PRAGMA are
      rejected in declaration/member/block annotation positions.
      co.dap.onEffect is also rejected here because its only legal parser path
      is on-effect-call-metadata before a call expression. co.dap.effects is
      accepted here only when the annotated node is callable; that target check
      is semantic. A non-co.* metadata name is accepted here as custom
      annotation/decorator syntax and is resolved later through the ordinary
      symbol table ? ;

(* Generic application shape. Sharing this surface syntax does not grant
   placement permission: the surrounding category guard decides whether the
   parsed metadata application is legal in the current syntactic position.
   The complete qualified name is classified
   immediately after it is read:
     - co.* -> must match builtin-metadata-name; otherwise parser error.
     - non-co.* -> syntactically accepted only as a custom annotation/decorator
       application and later resolved through the ordinary symbol table.
   Field names are intentionally NOT closed by this registry. Every field of a
   recognized form is parsed and preserved; frontend-known fields may receive
   additional validation, while unknown/unhandled fields remain available to
   later stages. *)
annotation = "@", qualified-name,
             [ "(", [ annotation-argument-list ], ")" ] ;

builtin-metadata-name-check =
    ? parser validation applied immediately after annotation qualified-name:
      when the name begins with co.*, the complete name must match
      builtin-metadata-name (or a registered reserved/future entry) and an
      unregistered co.* name is a parse error; a non-co.* name is collected as
      custom metadata and must later resolve through the symbol table to a
      user-defined annotation or decorator ? ;

builtin-metadata-name = builtin-pragma-name
                      | builtin-directive-name
                      | builtin-annotation-name
                      | builtin-decorator-name ;

builtin-pragma-name = "co.pdap.threadpool"
                    | "co.pdap.schedularpool" ;

builtin-directive-name = "co.ddap.import"
                       | "co.ddap.dynamicruntime"
                       | "co.ddap.use"
                       | "co.ddap.alias"
                       | "co.ddap.dynamicdispatch"
                       | "co.ddap.overload" ;

builtin-annotation-name = "co.dap.template"
                        | "co.dap.macro"
                        | "co.dap.operator"
                        | "co.dap.annotation"
                        | "co.dap.library"
                        | "co.dap.module"
                        | "co.dap.native"
                        | "co.dap.class"
                        | "co.dap.static"
                        | "co.dap.instance"
                        | "co.dap.object"
                        | "co.dap.inline"
                        | "co.dap.ctfe"
                        | "co.dap.friend"
                        | "co.dap.sealed"
                        | "co.dap.extension"
                        | "co.dap.override"
                        | "co.dap.implement"
                        | "co.dap.virtual"
                        | "co.dap.abstract"
                        | "co.dap.delegate"
                        | "co.dap.dynamicscope"
                        | "co.dap.lexicalscope"
                        | "co.dap.staticscope"
                        | "co.dap.mixedscope"
                        | "co.dap.typeclass"
                        | "co.dap.matcher"
                        | "co.dap.constructor"
                        | "co.dap.oops"
                        | "co.dap.extends"
                        | "co.dap.hokrlt"
                        | "co.dap.indexer"
                        | "co.dap.generic"
                        | "co.dap.comptime"
                        | "co.dap.typefromvalue"
                        | "co.dap.local"
                        | "co.dap.private"
                        | "co.dap.public"
                        | "co.dap.compose"
                        | "co.dap.guard"
                        | "co.dap.package"
                        | "co.dap.protected"
                        | "co.dap.internal"
                        | "co.dap.export"
                        | "co.dap.eager"
                        | "co.dap.lazy"
                        | "co.dap.packed"
                        | "co.dap.declare"
                        | "co.dap.implementation"
                        | "co.dap.simd"
                        | "co.dap.reflection"
                        | "co.dap.mop"
                        | "co.dap.nested"
                        | "co.dap.inner"
                        | "co.dap.final"
                        | "co.dap.const"
                        | "co.dap.decorator"
                        | "co.dap.specialize"
                        | "co.dap.symbol" ;

builtin-decorator-name = "co.dap.before"
                       | "co.dap.after"
                       | "co.dap.around"
                       | "co.dap.effects"
                       | "co.dap.onEffect"
                       | "co.dap.defer"
                       | "co.dap.callable"
                       | "co.dap.executionmodel" ;

annotation-argument-list = annotation-argument,
                           { ",", annotation-argument }, [ "," ] ;

annotation-argument = [ annotation-key, annotation-binder ], annotation-value ;

annotation-binder = declarative-attribute-binder ;

declarative-attribute-binder = "=" ;

annotation-key = annotation-key-segment,
                 { "-", annotation-key-segment }
               | qualified-name ;

annotation-key-segment = identifier | "for" ;

annotation-value = literal
                 | qualified-name
                 | annotation-declaration-reference
                 | annotation-list
                 | annotation-map
                 | annotation-arrow-pair ;

annotation-list = "[", [ annotation-value,
                         { ",", annotation-value }, [ "," ] ], "]" ;

annotation-map = "{", [ annotation-map-entry,
                        { ",", annotation-map-entry }, [ "," ] ], "}" ;

annotation-map-entry = annotation-map-key, annotation-binder,
                       annotation-value ;

annotation-map-key = annotation-key | qualified-name ;

annotation-arrow-pair = string-literal, "=>", string-literal ;

annotation-declaration-reference = qualified-name, "(",
                                   [ annotation-reference-type,
                                     { ",", annotation-reference-type } ], ")",
                                   "->", "(",
                                   [ annotation-reference-type,
                                     { ",", annotation-reference-type } ], ")" ;

annotation-reference-type = qualified-name ;

(* These productions are contextual views over annotation. They deliberately do
   not close the field list: recognized fields may be validated by the frontend
   while additional fields of the same known form remain parsed/preserved. *)
import-directive = annotation, import-directive-guard ;

import-directive-guard =
    ? zero-width parser condition: metadata name is exactly co.ddap.import;
      known target fields are package, library, component and as; semantic
      validation requires exactly one of package/library/component, but
      additional unhandled fields remain preserved ? ;

alias-directive = annotation, alias-directive-guard ;

alias-directive-guard =
    ? zero-width parser condition: metadata name is exactly co.ddap.alias ? ;

use-directive = annotation, use-directive-guard ;

use-directive-guard =
    ? zero-width parser condition: metadata name is exactly co.ddap.use ? ;

dynamic-runtime-directive = annotation, dynamic-runtime-directive-guard ;

dynamic-runtime-directive-guard =
    ? zero-width parser condition: metadata name is exactly
      co.ddap.dynamicruntime ? ;

dynamic-dispatch-directive = annotation, dynamic-dispatch-directive-guard ;

dynamic-dispatch-directive-guard =
    ? zero-width parser condition: metadata name is exactly
      co.ddap.dynamicdispatch ? ;


(* ====================================================================== *)

(* 3. Names and references *)

(* ====================================================================== *)

filename-derived-name = "_" ;

qualified-name = ( identifier | "co" ), { ".", identifier } ;


declaration-reference = qualified-function-reference | qualified-name ;

qualified-function-reference = qualified-name, "(", [ type-list ], ")",
                               declaration-return-type-clause ;

lifecycle-declaration-name = "@@new" | "@@init" ;

lifecycle-invocation-name = "new" | "init" ;

special-binding = result-binding | recursive-binding ;

this-receiver-expression = "this", this-receiver-context-guard,
                           ordinary-relationship-selector-exclusion-guard ;

relationship-selector-expression =
    "this", this-receiver-context-guard,
    "->", relationship-arrow-guard, relationship-category,
    "[", relationship-type-name, "]",
    relationship-selector-guard,
    compiler-owned-property-call-exclusion-guard ;

relationship-category = "classes"
                      | "mixins"
                      | "traits"
                      | "interfaces" ;

relationship-arrow-guard =
    ? zero-width contextual condition: "->" has relationship-selection meaning
      only when its left operand is the hard-reserved this token and its right
      token is a language-defined relationship name; value->member is invalid,
      while this.member remains ordinary member access ? ;

compiler-owned-this-selector-expression =
    "this", "->", compiler-owned-this-selector-name,
    compiler-owned-this-selector-guard,
    compiler-owned-property-call-exclusion-guard ;

compiler-owned-this-selector-name =
      "object" | "class" | "module" | "kind" | "type" | "struct"
    | "instance" | "callee" | "args" | "params" | "results" | "associatedtype"
    | "owner" | "caller" | "fallthrough" | "yield" | "builtins" ;

compiler-owned-this-selector-guard =
    ? zero-width contextual and semantic condition: the selected compiler-owned
      property or operation is available in the enclosing executable context;
      these names are contextual after this-> rather than globally reserved;
      this->args cannot implicitly capture an enclosing invocation's arguments
      when the nearest enclosing callable is annotated @co.dap.defer; values
      needed by that deferred callable must be passed explicitly at registration
      and received through its declared parameters ? ;

compiler-owned-property-call-exclusion-guard =
    ? zero-width syntactic condition: a compiler-owned this-> attribute is a
      property and cannot be followed immediately by call-suffix. Indexing that
      belongs to a keyed relationship property is part of its primary form;
      ordinary member-suffix, lifecycle-call-suffix where otherwise permitted,
      and a later call of an ordinary member remain available ? ;

relationship-type-name = qualified-name ;

relationship-selector-guard =
    ? zero-width contextual and semantic condition:
      - the occurrence is inside the applicable class context;
      - the selected category maps exactly to the ordered direct relationship
        list with the same name in @co.dap.oops;
      - only directly declared relationships are present; transitive
        relationships are not appended;
      - relationship-type-name is a compile-time declaration reference written
        as a complete type name or through an applicable import alias;
      - the written name resolves to one canonical declaration of the kind
        required by the selected category and that declaration occurs in the
        corresponding direct relationship list;
      - a numeric, string, runtime expression, computed expression,
        co.type value, unresolved name, ambiguous name, wrong-kind name,
        absent direct relationship, or missing key is invalid;
      - classes, mixins, traits, and interfaces are not standalone runtime
        values or collections;
      - only classes selects a class-parent lifecycle branch;
      - mixins and traits may qualify a composed implementation branch;
      - interfaces produces an interface type/view and ordinary interface
        dispatch, not an implementation-branch selection;
      - whenever a complete this->classes, this->mixins, this->traits, or
        this->interfaces prefix occurs in an applicable class context, this
        specialized production is selected before ordinary
        qualified-name/member/index postfix parsing ? ;

parent-selector-expression =
    "this", this-receiver-context-guard,
    ( "->", relationship-arrow-guard, ( "parent" | "super" )
    | "->", relationship-arrow-guard, "parents", "[", relationship-type-name, "]" ),
    direct-parent-selector-guard,
    compiler-owned-property-call-exclusion-guard ;

direct-parent-selector-guard =
    ? zero-width contextual and semantic condition:
      - the occurrence is inside the applicable class context;
      - direct class parents are selected in their declared classes=[...]
        order;
      - singular this->parent and this->super select the first direct class parent;
      - plural this->parents[Type] is identical to this->classes[Type];
      - only plural this->parents is keyed, and it always requires a compile-time
        relationship-type-name;
      - the type name must resolve canonically to a direct class parent;
      - this->parent[Type], this->super[Type], numeric keys, string keys, computed keys, runtime
        co.type values, and unkeyed this->parents are invalid;
      - this is a compile-time branch selector, not runtime indexing;
      - repeated canonical ancestors use one virtual-base identity;
      - whenever a complete this->parent, this->super, or this->parents prefix occurs in an
        applicable class context, this
        specialized production is selected before ordinary
        qualified-name/member/index postfix parsing ? ;

ordinary-relationship-selector-exclusion-guard =
    ? zero-width parser lookahead condition: the immediately following tokens
      are none of the complete contextual prefixes "->", "classes", "->", "mixins",
      "->", "traits", "->", "interfaces", "->", "parent", "->", "super", or "->", "parents"
      in a context where relationship-selector-expression or
      parent-selector-expression applies ? ;

this-receiver-context-guard =
    ? zero-width contextual condition: this occurrence denotes a receiver in
      one of the following contexts:
      - an instance method of co.class, where it denotes the current
        instance;
      - a co.dap.class method of co.class, where it denotes the current
        class/type;
      - a developer lifecycle declaration admitted by the lifecycle
        customization guard, where @@new supplies the class/type receiver and
        @@init supplies the instance receiver;
      - an instance or co.dap.class method of a target-bound co.extension,
        where it denotes respectively an instance of fortype or the fortype
        class/type; or
      - a concrete/default method of co.trait or co.mixin, where it
        denotes the current composed instance; or
      - a function of co.object, where it denotes the current singleton
        object.
      A co.dap.static method and an ordinary free, module, signature, interface,
      typeclass, instance, trait, mixin, or unit function do not acquire a
      receiver through this production. The separately parsed this =>,
      this ^=>, this ->>, and this ->| control forms do not require a receiver ? ;

refinement-candidate = "_", refinement-candidate-guard ;

refinement-candidate-guard =
    ? zero-width contextual condition: this occurrence is inside the predicate
      of the co.refinementType declaration currently being parsed ? ;

recursive-binding = "$" ;

result-binding = "$", digit, { digit } ;

wildcard = "_" ;


(* ====================================================================== *)

(* 4. Type syntax *)

(* ====================================================================== *)

type-expression = union-type-expression ;

polymorphic-type-expression = "forall", "(", type-parameter-list, ")", ".",
                              type-expression,
                              forall-context-guard ;

forall-context-guard =
    ? zero-width contextual condition: this occurrence is the value of a
      co.type declaration; every other direct use of forall as a type
      binder is invalid ? ;

type-declaration-value = polymorphic-type-expression | type-expression ;

(* co.dependentType(type-expression) is parsed by the ordinary type-application
   grammar below but has a reserved semantic role when it is the outermost RHS
   of a co.type declaration: it marks that family as value-indexed. Its head
   parameters are still syntactically bare identifiers; semantic resolution
   classifies each one as a type or value parameter from its RHS positions. *)

type-parameter-list = identifier, { ",", identifier } ;

union-type-expression = arrow-type-expression,
                        { "|", arrow-type-expression } ;

arrow-type-expression = type-postfix-expression,
                        [ "->", arrow-type-tail ]
                      | "(", [ function-type-parameter,
                               { ",", function-type-parameter } ],
                        ")", "->", arrow-type-tail ;

arrow-type-tail = type-derivation
                | parenthesized-type-list
                | type-expression ;

type-postfix-expression = type-atom, { type-argument-list } ;

type-atom = qualified-name
          | "(", type-expression, ")" ;

type-argument-list = "(", [ type-or-value-argument,
                            { ",", type-or-value-argument } ], ")" ;

type-or-value-argument = type-expression | dependent-index | annotation-list ;

dependent-index = integer-literal | qualified-name ;

type-list = type-expression, { ",", type-expression } ;

(* A parenthesized postfix on a type atom is the uniform type-application form.
   Symbol resolution classifies its target as a built-in or annotation-declared
   generic, a parameterized co.type family, a typeclass, or an abstract type
   parameter. Arguments bind positionally in declaration order, and non-type
   values use the same argument slots:

       co.Map(co.string, co.int)
       co.Array(2, co.int, [2,4])
       LinkedList(co.int)

   Named bindings are deliberately absent. Parentheses after a generic
   declaration bind that declaration's parameters by position, while an arrow
   tail may carry the derivation-attribute-list described by "Type Application
   and Arrow Tails". *)
parenthesized-type-list = "(", [ return-item-list ], ")" ;

type-derivation = "(", derivation-specification, ")" ;

derivation-specification = pointer-specification
                         | array-specification
                         | reference-specification
                         | range-type-specification
                         | slice-type-specification
                         | thunk-type-specification
                         | address-type-specification
                         | derivation-attribute-list ;

pointer-specification = pointer-stars,
                        [ ",", derivation-attribute-list ] ;

pointer-stars = ? one contiguous symbolic run consisting only of one or
                  more "*" characters; its length is the pointer degree ? ;

reference-specification = ( "&" | "&&" | "~" ),
                          [ ",", derivation-attribute-list ] ;

address-type-specification = "@", [ ",", derivation-attribute-list ] ;

thunk-type-specification = "^", [ ",", derivation-attribute-list ] ;

slice-type-specification = "[:]", [ ",", derivation-attribute-list ] ;

range-type-specification = "..", [ ",", derivation-attribute-list ] ;

array-specification = array-dimension-group, { array-dimension-group },
                      [ ",", derivation-attribute-list ] ;

array-dimension-group = "[", array-dimension-content, "]" ;

array-dimension-content = [ array-dimension ], { ",", [ array-dimension ] } ;

array-dimension = "..." | "." | dependent-index ;

(* Attributes carried by a type derivation. Generic type applications instead
   use the positional type-argument-list production. *)
derivation-attribute-list = derivation-attribute,
                            { ",", derivation-attribute } ;

derivation-attribute = annotation-key, "=", annotation-value ;

return-type-clause = "->", "(", [ return-item-list ], ")" ;

return-item-list = return-item, { ",", return-item } ;

return-item = type-expression ;

type-use = qualified-name, { type-argument-list } ;

type-use-argument = type-use
                  | dependent-index
                  | identifier, "=", ( type-use | dependent-index ) ;

declaration-return-type-clause =
    "->", "(", [ declaration-return-item,
                  { ",", declaration-return-item } ], ")" ;

declaration-return-item = type-use ;


(* ====================================================================== *)

(* 5. Common declaration components *)

(* ====================================================================== *)

generic-parameter-clause = "(", generic-parameter,
                           { ",", generic-parameter }, ")" ;

generic-parameter = identifier, [ generic-arity-clause ] ;

generic-arity-clause = "(", generic-arity-slot,
                       { ",", generic-arity-slot }, ")" ;

generic-arity-slot = "_" | identifier ;

kind-options = "->", "(", [ annotation-argument-list ], ")" ;

field-declaration = annotations, identifier, type-use,
                    [ "=", expression ], statement-end ;

embedded-field-declaration = annotations, type-use, statement-end ;

value-specification = annotations, identifier, type-use, statement-end ;

variable-declaration = annotations, typed-variable-declarator,
                       { ",", typed-variable-declarator }, statement-end ;

typed-variable-declarator = identifier, type-use,
                            [ "=", expression ] ;

inferred-variable-declaration = annotations, inferred-variable-declarator,
                                { ",", inferred-variable-declarator },
                                statement-end ;

inferred-variable-declarator = identifier, definition-operator, expression ;

definition-operator = ( ":=" | "::=" | "?=" ),
                      multi-symbol-infix-operator-boundary-guard ;


(* ====================================================================== *)

(* 6. Data and type declarations *)

(* ====================================================================== *)

struct-declaration = annotations, filename-derived-name,
                     "co.struct", "=", struct-body ;

surface-struct-declaration = annotations, identifier,
                             "co.struct", "=", struct-body ;

struct-body = "{", { struct-member }, body-close ;

pure-field-declaration = annotations, identifier, type-use,
                         statement-end ;

struct-member = pure-field-declaration | embedded-field-declaration ;

cstruct-declaration = annotations, filename-derived-name,
                      "co.cstruct", "=", cstruct-body ;

surface-cstruct-declaration = annotations, identifier,
                              "co.cstruct", "=", cstruct-body ;

cstruct-body = "{", { pure-field-declaration }, body-close ;

enum-declaration = annotations, filename-derived-name,
                   "co.enum", "=", enum-body ;

enum-body = "{", [ enum-variant,
                    { enum-separator, enum-variant }, [ enum-separator ] ],
            body-close ;

enum-separator = "," ;

enum-variant = annotations, identifier,
               [ "(", enum-state-parameter-list, ")" ],
               [ "=", constant-expression ] ;

enum-state-parameter-list = enum-state-parameter,
                            { ",", enum-state-parameter } ;

enum-state-parameter = identifier, type-expression ;

union-declaration = annotations, filename-derived-name,
                    "co.union", "=", union-body ;

union-body = "{", { pure-field-declaration }, body-close ;

data-declaration = annotations, identifier,
                   [ generic-parameter-clause ], "co.data", "=",
                   data-variant, { "|", data-variant }, statement-end ;

data-variant = qualified-name,
               [ "(", type-list, ")" ] ;

type-declaration = ( polymorphic-type-declaration
                   | simple-type-declaration
                   | refinement-type-declaration
                   | predicate-type-declaration ),
                   type-declaration-context-guard ;

type-declaration-context-guard =
    ? zero-width condition: a named non-UDT type declaration occurs in
      src/appl.fol, a non-operator component surface, a unit, class, interface,
      module, mixin, signature, function, or nested executable block;
      co.associatedType is governed separately and remains limited to
      signatures and matching modules ? ;

polymorphic-type-declaration = annotations, identifier,
                               [ generic-parameter-clause ], "co.type",
                               [ kind-options ],
                               [ "=", type-declaration-value ],
                               type-family-parameter-role-guard,
                               statement-end ;

type-family-parameter-role-guard =
    ? zero-width semantic condition: for a co.type declaration with a
      parameterized head, each head parameter is classified from its resolved
      RHS uses. Use in a type position classifies a type parameter. Use in a
      dependent index/value position under an outermost co.dependentType(...)
      RHS classifies a value parameter whose domain is imposed by that
      consuming construct. A parameter with incompatible roles, no resolvable
      role, or an outermost co.dependentType(...) RHS with no value parameter
      is InvalidGenericDeclaration. co.dependentType(...) is not a declaration
      kind, variable type, or callable result kind ? ;

simple-type-declaration = annotations, identifier, nonpolymorphic-type-declaration-kind,
                          [ kind-options ], [ "=", type-expression ],
                          statement-end ;

nonpolymorphic-type-declaration-kind = "co.newtype"
                      | "co.opaquetype"
                      | "co.subtype"
                      | "co.supertype"
                      | "co.kind" ;

refinement-type-declaration =
    annotations, identifier, "co.refinementType", "=",
    refinement-type-expression, statement-end ;

refinement-type-expression =
    "(", type-expression, ")", ".where", "(", expression, ")" ;

predicate-type-declaration =
    annotations, identifier, "co.predicateType", "=",
    predicate-type-expression, statement-end ;

predicate-type-expression =
    "co.type", ".where", "(", predicate-type-binder, "=>",
    expression, ")" ;

predicate-type-binder = identifier ;

(* ====================================================================== *)

(* 7. Containers, contracts, instances, and component surfaces *)

(* ====================================================================== *)

unit-declaration = annotations, filename-derived-name,
                   "co.unit", "=", unit-body ;

unit-body = "{", { unit-member }, body-close ;

unit-member = function-declaration
            | data-declaration
            | type-declaration
            | function-object-declaration
            | delegate-declaration
            | extern-variable-declaration ;

extern-variable-declaration =
    "@co.dap.declare", "(", "type", "=", "extern", ")",
    identifier, type-expression,
    statement-end ;

class-declaration = annotations, filename-derived-name,
                    "co.class", [ kind-options ], "=", class-body,
                    class-lifecycle-capability-guard ;

class-lifecycle-capability-guard =
    ? zero-width semantic condition:
      - every co.class has compiler-owned inherited lifecycle implementations
        for @@new and @@init as non-public/protected lifecycle machinery;
      - lifecycle is an optional field of the enclosing class's co.dap.generic
        metadata application, not a separate annotation;
      - for a generic co.class, lifecycle=true grants source permission to
        override or overload the existing compiler-owned @@new / @@init family;
        lifecycle absent or lifecycle=false forbids developer lifecycle
        customization but does not remove the inherited compiler lifecycle;
      - if lifecycle is present on a generic struct, generic function, or generic
        method, that field is accepted but is not considered for lifecycle
        semantics;
      - if any lifecycle-method-declaration occurs in the class body, the class
        must carry valid co.dap.generic(types=[...], lifecycle=true) metadata;
      - lifecycle=true does not itself expose ::new(...) or ::init(...);
      - no filesystem/package/component placement condition is imposed by the
        lifecycle facility itself ? ;

class-body = "{", { class-member }, body-close ;

class-member = class-instance-field-declaration
             | function-declaration
             | type-declaration
             | lifecycle-method-declaration ;

class-instance-field-declaration =
    pure-field-declaration, class-instance-field-policy-guard ;

class-instance-field-policy-guard =
    ? zero-width semantic condition:
      - the declaration is ordinary mutable per-instance storage;
      - co.dap.const and co.dap.final are forbidden on the field;
      - field-level Shared, locking, CopyOnWrite, static/class/global storage,
        ownership, atomicity, and inline initialization are forbidden;
      - an object-graph policy may later target the complete class instance,
        never this selected field as an independent class-storage root ? ;

trait-declaration = annotations, filename-derived-name,
                    "co.trait", "=", trait-body ;

trait-body = "{", { trait-member }, body-close ;

trait-member = function-declaration, trait-member-guard ;

trait-member-guard =
    ? zero-width semantic condition:
      - a trait carries no instance state, so fields and other value-bearing
        members are not permitted;
      - a function may be abstract/bodyless, may provide a default
        implementation, or may be declared virtual when it has the
        implementation required by the virtual-method rules ? ;

mixin-declaration = annotations, filename-derived-name,
                    "co.mixin", "=", mixin-body ;

mixin-body = "{", { mixin-member }, body-close ;

mixin-member = class-instance-field-declaration
             | function-declaration
             | type-declaration ;

lifecycle-method-declaration = annotations, lifecycle-declaration-name,
                               parameter-list, [ declaration-return-type-clause ],
                               function-definition,
                               lifecycle-declaration-context-guard ;

lifecycle-declaration-context-guard =
    ? zero-width semantic condition:
      - the enclosing declaration is a co.class carrying valid
        co.dap.generic metadata with an explicit types=[...] list and
        lifecycle=true;
      - the source declaration is an override of an existing compiler lifecycle
        signature or an overload that adds another signature to the same
        language-owned lifecycle name; it never creates a new lifecycle name;
      - a non-generic class, a generic class with lifecycle absent/false, or any
        non-class declaration cannot source-declare @@new or @@init;
      - each admitted lifecycle implementation retains its ordinary declared
        accessibility and other applicable method metadata;
      - lifecycle=true on a generic struct, function, or method does not change
        this class-only customization rule ? ;

interface-declaration = annotations, filename-derived-name,
                        "co.interface", "=", interface-body ;

interface-body = "{", { function-specification | type-declaration }, body-close ;

signature-declaration = annotations, filename-derived-name,
                        "co.signature", "=", signature-body ;

signature-body = "{", { signature-member }, body-close ;

signature-member = value-specification
                 | function-specification
                 | signature-type-component
                 | associated-type-requirement
                 | type-declaration ;

signature-type-component = annotations, identifier,
                           "co.type",
                           [ "=", type-expression ], statement-end ;

(* `Name co.type;` is an ordinary abstract type component and does not
   imply genericity. Every co.associatedType component must name a type
   argument used by at least one bodyless co.type component of the same
   signature. A matching module must fulfill that alias by instantiating a
   generic container with the associated type; otherwise the associated-type
   declaration is invalid. *)
associated-type-requirement = annotations, identifier,
                              "co.associatedType", statement-end ;

module-declaration = annotations, filename-derived-name,
                     "co.module", [ kind-options ], "=", module-body ;

module-body = "{", { module-member }, body-close ;

module-member = variable-declaration
              | inferred-variable-declaration
              | function-declaration
              | type-declaration
              | associated-type-binding ;

associated-type-binding = annotations, identifier,
                          "co.associatedType", "=", type-expression,
                          statement-end ;

(* Each module co.associatedType binding corresponds by name to a
   requirement in the matched signature. The module must also contain the
   corresponding required co.type alias, and that alias must instantiate
   a generic container using the associated type. Missing, unrelated, or unused
   bindings are compiler errors. Typeclass and instance higher-kinded forms are
   unaffected. *)

component-declaration = annotations, filename-derived-name,
                        "co.component", "=", component-body,
                        component-surface-context-guard ;

component-body = "{", { component-member }, body-close ;

component-surface-context-guard =
    ? zero-width structural validation supplied by filesystem role:
      src/component.fol is either a projected standalone library annotated with
      co.dap.library (application when type is omitted; otherwise application,
      native, or dynamicvmrt) or a packaged standalone component containing the
      co.dap.export package selector, never both; components/application,
      components/native, and components/dynamicvmrt expose only their permitted
      projected boundary declarations and carry no co.dap.library annotation;
      components/packaged uses the export selector; components/operators
      contains operator declarations only; the operators component is permitted
      only for an executable application or a projected application library ? ;

component-member = import-directive
                 | surface-struct-declaration
                 | surface-cstruct-declaration
                 | type-declaration
                 | function-declaration
                 | component-export-selector
                 | operator-declaration ;

component-export-selector = annotation, component-export-selector-guard ;

component-export-selector-guard =
    ? zero-width structural/context condition: metadata name is exactly
      co.dap.export and this standalone component-body entry is used only for
      packaged src/component.fol or components/packaged/component.fol; the
      parser collects the complete field payload, while semantic validation
      checks the packages selector and project/component context ? ;

extension-declaration = annotations, filename-derived-name,
                        "co.extension", extension-target-options, "=",
                        extension-body ;

extension-target-options =
    "->", "(", "fortype", "=", type-expression, ")" ;

extension-body = "{", { extension-member }, body-close ;

extension-member = field-declaration | function-declaration ;


object-declaration = annotations, filename-derived-name,
                     "co.object", object-association-options,
                     "=", object-body, object-association-guard ;

object-association-options =
    "->", "(", "for", "=", object-association-targets, ")" ;

object-association-targets =
      qualified-name
    | "[", qualified-name, { ",", qualified-name }, [ "," ], "]" ;

object-association-guard =
    ? zero-width semantic condition:
      - the scalar target resolves to exactly one annotation declaration or
        exactly one co.class declaration;
      - the list form is non-empty, contains only co.class declaration
        references, contains no duplicate canonical target, and cannot mix
        annotation and class declaration kinds;
      - the object name follows the ordinary filename-derived primary-name
        rule and is independent of every target name; no companion filename or
        companion-owner relationship is created;
      - the declaration denotes one singleton object whose association is
        explicit, non-owning, non-structural, and non-transitive;
      - association does not promote fields or methods, participate in class
        inheritance/dynamic dispatch, add a managed-reference graph edge, or
        propagate construction, copying, freezing, sharing, locking,
        CopyOnWrite behavior, or destruction between either side ? ;

object-body = "{", { field-declaration | function-declaration }, body-close ;

instance-declaration = annotations, filename-derived-name,
                       "co.instance", [ kind-options ], "=",
                       instance-body ;

instance-body = "{", { function-declaration | variable-declaration }, body-close ;

typeclass-declaration = typeclass-annotations, filename-derived-name,
                        "co.typeclass", "=", contract-body ;

typeclass-annotations = { annotation }, typeclass-annotation, { annotation } ;

typeclass-annotation = "@co.dap.typeclass", "(",
                       typeclass-option, { ",", typeclass-option }, ")" ;

typeclass-option = typeclass-kind-option
                 | typeclass-shape-option
                 | typeclass-aliases-option ;

typeclass-kind-option = "kind", annotation-binder, identifier ;
typeclass-shape-option = "shape", annotation-binder, typeclass-shape ;
typeclass-aliases-option = "aliases", annotation-binder,
                           "[", typeclass-alias,
                           { ",", typeclass-alias }, "]" ;
typeclass-alias = "{", "name", annotation-binder, identifier, ",",
                  "type", annotation-binder, type-expression, "}" ;

(* Exactly one typeclass-annotation is required. Kind and shape are required;
   aliases is optional, option order is immaterial, and duplicate options are
   invalid. Its shape is metadata-owned; no parameter clause may occur after
   filename-derived-name. *)

typeclass-shape = generic-parameter-clause ;

matcher-instance-declaration = annotations, filename-derived-name,
                               "co.matcher", matcher-options, "=",
                               matcher-body ;

matcher-options = "->", "(", "type", annotation-binder, type-expression, ")" ;

matcher-body = "{", { field-declaration | function-declaration }, body-close ;

contract-body = "{", { function-specification | value-specification }, body-close ;

named-block-declaration = annotations, identifier,
                          "co.block", "=", block, body-closure-guard ;


(* ====================================================================== *)

(* 8. Functions and callable declarations *)

(* ====================================================================== *)

delegate-declaration = annotations, identifier,
                       "co.delegate", "=", function-type, statement-end ;

function-object-declaration = annotations, identifier,
                              "co.function", "=",
                              function-object-binding ;

function-object-binding = expression, statement-end ;

function-declaration = annotations,
                       [ receiver-clause, receiver-clause-context-guard ],
                       function-name,
                       function-parameter-lists,
                       [ declaration-return-type-clause ], function-binding,
                       function-shaped-declaration-classification-guard ;

function-parameter-lists = parameter-list
                         | nonempty-parameter-list, nonempty-parameter-list,
                           { nonempty-parameter-list } ;

function-shaped-declaration-classification-guard =
    ? zero-width semantic classification:
      when attached directly to this function-shaped declaration, the following
      classifying metadata forms select specialized AST kinds:
          co.dap.generic        -> GenericFunctionDecl
          co.dap.decorator      -> DecoratorDecl
          co.dap.extension      -> ExtensionMethodDecl
          co.dap.macro          -> MacroDecl
          co.dap.template       -> TemplateDecl
          co.dap.native         -> NativeFunctionDecl
          co.dap.executionmodel -> ExecutionModelFunctionDecl
          co.dap.operator       -> OperatorOverloadDecl
          co.dap.indexer        -> IndexerDecl
      these classifying forms are mutually exclusive on one function-shaped
      declaration except for the deliberate co.dap.operator + co.dap.extension
      composition. In that combination the declaration remains an
      OperatorOverloadDecl and co.dap.extension supplies the existing target/owner;
      it does not create a second AST declaration kind. co.dap.operator and
      co.dap.generic are incompatible on the same declaration, with or without
      co.dap.extension: an operator declaration never introduces operator-level
      generic parameters. A generic enclosing class, struct, or extension target
      does not make the operator declaration generic; operator ownership is
      associated with the canonical owner declaration identity. Every other
      combination of two classifying forms is a compiler error. When none is
      present the declaration is an ordinary FunctionDecl; non-classifying
      metadata does not change its AST declaration kind. An
      ExecutionModelFunctionDecl must have exactly one co.error-compatible
      result and must not carry co.dap.effects ? ;

function-name = identifier ;

receiver-clause = "(", ( type-use
                        | identifier, type-use ), ")" ;

receiver-clause-context-guard =
    ? zero-width condition: an explicit receiver occurs only on a function
      declared directly in a validated <StructName>.comp.unit.fol companion;
      ordinary units and nested class, module, signature, interface, function,
      or executable-block contexts do not inherit receiver permission ? ;

parameter-list = "(", [ parameter,
                        { ",", parameter } ], ")" ;

nonempty-parameter-list = "(", typed-parameter,
                          { ",", typed-parameter }, ")" ;

parameter = typed-parameter | untyped-template-parameter ;

typed-parameter = [ "..." ], [ "~" ], identifier, [ "?" ],
                  type-use, [ "=", expression ] ;

untyped-template-parameter = [ "..." ], [ "~" ], identifier, [ "?" ],
                             [ "=", expression ],
                             template-parameter-context-guard ;

template-parameter-context-guard =
    ? zero-width condition: the enclosing function-shaped declaration carries
      built-in @co.dap.template metadata; ordinary functions and every other
      callable category require an explicit parameter type ? ;

function-binding = function-definition
                 | function-delegation
                 | function-type-value-binding
                 | function-alias-binding
                 | statement-end ;

function-definition = "=", block, body-closure-guard ;

function-delegation = "=>>", expression,
                      { "=>>", expression }, statement-end ;

function-type-value-binding = "=", type-expression, statement-end ;

function-alias-binding = "=", non-block-expression, statement-end ;

function-specification = annotations, function-name,
                         parameter-list, { parameter-list },
                         [ declaration-return-type-clause ], statement-end ;

function-type = "(", [ function-type-parameter,
                       { ",", function-type-parameter } ], ")",
                return-type-clause ;

function-type-parameter = type-expression
                        | identifier, type-expression ;

anonymous-function-expression = parameter-list, declaration-return-type-clause, block,
                                anonymous-function-binding-context-guard,
                                anonymous-function-generic-context-guard ;

anonymous-function-binding-context-guard =
    ? zero-width condition: the anonymous-function expression is the root value
      of a typed, inferred, let, field or co.function binding initializer;
      an optional postfix invocation may immediately follow it, but the literal
      cannot be a standalone statement, argument, return value or nested operand ? ;

anonymous-function-generic-context-guard =
    ? zero-width condition: the anonymous function declares no generic names;
      any generic names in its signature belong to an enclosing
      @co.dap.generic struct, class, function or method ? ;

lambda-expression = "|", [ lambda-parameter,
                            { ",", lambda-parameter } ], "|", "=>",
                    ( expression | block ) ;

lambda-parameter = identifier, [ type-use ] ;


local-function-declaration = annotations, function-name, parameter-list,
                             { parameter-list }, declaration-return-type-clause,
                             function-definition,
                             local-function-nongeneric-guard ;

local-function-nongeneric-guard =
    ? zero-width condition: an inner/local function must not carry
      co.dap.generic and cannot introduce generic parameters; it may use types
      already visible from an enclosing generic class or function ? ;


(* ====================================================================== *)

(* 9. Function-pattern clauses and patterns *)

(* ====================================================================== *)

let-function-pattern-clause = annotations, "let", identifier,
                              pattern-parameter-list,
                              [ where-clause ], "=", pattern-result ;

pattern-parameter-list = "(", [ pattern,
                                { ",", pattern } ], ")" ;

where-clause = ".where", "(", expression, ")" ;

pattern-result = block, body-closure-guard
               | non-block-expression, statement-end ;

pattern = wildcard
        | literal-pattern
        | binding-pattern
        | constructor-pattern
        | record-pattern
        | tuple-pattern
        | qualified-name ;

literal-pattern = literal
                | ( "+" | "-" ),
                  ( integer-literal | floating-literal ) ;

binding-pattern = identifier ;

constructor-pattern = qualified-name, "(", constructor-pattern-item,
                        { ",", constructor-pattern-item }, ")",
                        enum-state-pattern-guard ;

constructor-pattern-item = [ identifier, "=" ], pattern ;

enum-state-pattern-guard =
    ? zero-width semantic condition: when qualified-name resolves to a
      parameterized co.enum state, every item has identifier "=", every
      declared state parameter occurs exactly once, and no unknown or duplicate
      label occurs; when it resolves to a zero-parameter enum state the bare
      qualified-name form is required instead; co.variants and co.data
      constructors retain their separately specified positional patterns ? ;

record-pattern = qualified-name, "{", [ record-pattern-field,
                                        { ",", record-pattern-field } ], "}" ;

record-pattern-field = identifier, [ ":", pattern ] ;

tuple-pattern = "(", pattern, ",", pattern,
                { ",", pattern }, ")" ;

match-case = ".case", "(", match-case-body, ")" ;

match-case-body = pattern, [ ":", expression ], "=>",
                  ( expression | block ) ;

match-default = ".default", "(", ( expression | block ), ")" ;


(* ====================================================================== *)

(* 10. Statements, blocks, and termination *)

(* ====================================================================== *)

block = "{", { block-item }, [ block-tail-expression ], "}" ;

block-item = statement ;

block-tail-expression = expression ;

statement = named-block-declaration
          | type-declaration
          | variable-declaration
          | inferred-variable-declaration
          | grouped-variable-declaration
          | let-value-declaration
          | local-function-declaration
          | multiple-assignment-statement
          | enclosing-callable-return-statement
          | return-statement
          | break-statement
          | continue-statement
          | lock-statement
          | labeled-block
          | labeled-loop-statement
          | expression-statement
          | block-statement
          | empty-statement ;

grouped-variable-declaration = "(", typed-variable-declarator,
                               { ",", typed-variable-declarator }, ")",
                               statement-end ;

let-value-declaration = "let", identifier, [ type-expression ], "=",
                        expression, statement-end ;

multiple-assignment-statement = assignment-target, ",", assignment-target,
                                { ",", assignment-target }, "=",
                                expression-list, statement-end ;

assignment-target = postfix-expression
                  | tuple-assignment-target ;

tuple-assignment-target = "(", assignment-target, ",", assignment-target,
                          { ",", assignment-target }, ")" ;

return-statement = "this", "=>", [ expression-list ], statement-end ;

enclosing-callable-return-statement =
    "this", "^=>", [ expression-list ], statement-end,
    enclosing-callable-return-guard ;

enclosing-callable-return-guard =
    ? zero-width contextual and semantic condition:
      - the occurrence is inside an anonymous block supplied as a call argument;
      - no named function, method, anonymous function, function expression, or
        lambda boundary occurs between the statement and that argument block;
      - the statement terminates the anonymous block and its nearest lexically
        enclosing callable, producing values for that callable;
      - the values satisfy that callable's declared result positions;
      - a direct callable-body occurrence, bare or named block occurrence,
        function-expression occurrence, and every ^^=> or multi-level spelling
        is invalid ? ;

expression-statement = non-block-expression, statement-end ;

break-statement =
    "this", "->|", [ label-reference ], statement-end,
    break-target-guard ;

break-target-guard =
    ? zero-width semantic condition:
      - without label-reference, this ->| targets the ordinary nearest
        breakable control region;
      - with label-reference, the reference resolves only to an active enclosing
        labeled-block or labeled-loop-statement;
      - resolution is lexical and selects the innermost enclosing matching label;
      - the target is exited structurally; no arbitrary goto/address jump exists ? ;

continue-statement =
    "this", "->>", [ label-reference ], statement-end,
    continue-target-guard ;

continue-target-guard =
    ? zero-width semantic condition:
      - without label-reference, this ->> targets the ordinary nearest loop;
      - with label-reference, the reference must resolve to an active enclosing
        labeled-loop-statement;
      - a plain labeled-block is not a valid continue target;
      - resolution is lexical and selects the innermost enclosing matching label ? ;

labeled-block =
    label-declaration, ":", block, body-closure-guard ;

labeled-loop-statement =
    label-declaration, ":", expression-statement,
    labeled-loop-statement-guard ;

labeled-loop-statement-guard =
    ? zero-width semantic condition: the labeled expression-statement is a
      current-profile loop form whose outer control operation is .loop(...);
      labeling an arbitrary expression statement does not turn it into a loop ? ;

label-declaration = label-identifier ;

label-reference = label-identifier ;

empty-statement = ";" ;

expression-list = expression, { ",", expression } ;

statement-end = ";" ;

body-close = "}", body-closure-guard ;

body-closure-guard =
    ? zero-width condition: the next significant token is not ";", or there is no next token ? ;

block-statement = block, body-closure-guard ;

lock-statement = "lock", "(", expression, ")", block,
                 body-closure-guard ;

non-block-expression = expression, non-block-expression-guard ;

non-block-expression-guard =
    ? zero-width condition: the complete source span is not admissible as an
      unparenthesized block production; when both block and another braced
      expression are possible, the block reading has priority ? ;


(* ====================================================================== *)

(* 11. Expressions and built-in operator precedence *)

(* ====================================================================== *)

expression = assignment-expression
           | extended-operator-expression ;

assignment-expression = logical-or-expression,
                        [ runtime-assignment-operator,
                          assignment-expression ] ;

runtime-assignment-operator = "="
                            | compound-assignment-operator ;

compound-assignment-operator = ( "+=" | "-=" | "*=" | "/=" | "%="
                               | "**=" | "&=" | "^=" | "|=" ),
                               multi-symbol-infix-operator-boundary-guard ;

constant-expression = ( logical-or-expression
                      | extended-operator-expression ),
                      ? zero-width condition: no runtime-assignment-operator
                        occurs anywhere in this constant-expression subtree ? ;

logical-or-expression = logical-and-expression,
                        { logical-or-operator, logical-and-expression } ;

logical-or-operator = "||", multi-symbol-infix-operator-boundary-guard ;

logical-and-expression = bitwise-or-expression,
                         { logical-and-operator, bitwise-or-expression } ;

logical-and-operator = "&&", multi-symbol-infix-operator-boundary-guard ;

bitwise-or-expression = bitwise-xor-expression,
                        { "|", bitwise-xor-expression } ;

bitwise-xor-expression = bitwise-and-expression,
                         { "^", bitwise-and-expression } ;

bitwise-and-expression = equality-expression,
                         { "&", equality-expression } ;

equality-expression = relational-expression,
                      [ equality-operator, relational-expression ] ;

equality-operator = ( "==" | "!=" ),
                    multi-symbol-infix-operator-boundary-guard ;

relational-expression = range-expression,
                        [ relational-operator, range-expression ] ;

relational-operator = "<" | ">" | multi-symbol-relational-operator ;

multi-symbol-relational-operator = ( "<=" | ">=" | "<:" | ":>" ),
                                   multi-symbol-infix-operator-boundary-guard ;

range-expression = additive-expression,
                   [ range-operator, [ additive-expression ] ]
                 | range-operator, additive-expression ;

range-operator = ( ".." | "<.." | "..<" | "<..<" ),
                 multi-symbol-range-operator-boundary-guard ;

additive-expression = predeclared-glyph-expression,
                      { additive-operator, predeclared-glyph-expression } ;

additive-operator = "+" | "-" ;

predeclared-glyph-expression = multiplicative-expression,
                               { predeclared-operator-glyph,
                                 multiplicative-expression } ;

multiplicative-expression = unary-expression,
                            { multiplicative-operator, unary-expression } ;

multiplicative-operator = "*" | "/" | "%" ;

unary-expression = { prefix-operator }, power-expression ;

prefix-operator = "+" | "-" | "!" ;

power-expression = ( effect-handled-call-expression | postfix-expression ),
                   [ power-operator, unary-expression ] ;

effect-handled-call-expression = on-effect-call-metadata,
                                 postfix-expression,
                                 effect-handled-call-context-guard ;

on-effect-call-metadata = annotation, on-effect-call-metadata-guard ;

on-effect-call-metadata-guard =
    ? zero-width parser condition: the complete metadata name just parsed is
      exactly co.dap.onEffect; co.dap.effects and every other metadata name are
      invalid in this expression-prefix position ? ;

effect-handled-call-context-guard =
    ? zero-width semantic condition: the decorated postfix-expression's
      outermost operation is an ordinary or lifecycle invocation. The metadata
      belongs to that one call AST node, not to calls nested in its receiver or
      argument expressions. Each effect key must resolve to a class satisfying
      co.error and need not be present in the callee's open advertised
      emits set. For an ordinary callee every record requires exactly one
      resolution and may contain a non-empty handlers list. For a callee
      classified as ExecutionModelFunctionDecl, or a canonical submission call
      statically bound to such a target, every record requires a non-empty
      handlers list, forbids an explicit resolution, and receives implicit
      return_with_error. Retry captures the evaluated receiver and argument
      values once and reinvokes only an ordinary decorated call node ? ;

power-operator = "**", multi-symbol-infix-operator-boundary-guard ;

postfix-expression = primary-expression,
                     { postfix-suffix | postfix-operator } ;

postfix-operator = "!" ;

postfix-suffix = call-suffix
               | index-suffix
               | member-suffix
               | lifecycle-call-suffix
               | match-suffix ;

call-suffix = "(", [ argument-list ], ")", enum-state-call-guard ;

enum-state-call-guard =
    ? zero-width semantic condition: when the call target resolves to a
      parameterized co.enum state, every argument has identifier "=",
      every declared state parameter occurs exactly once, and no unknown or
      duplicate label occurs; a zero-parameter enum state is a bare value and
      cannot take a call suffix; ordinary functions and co.variants or
      co.data state functions retain their own argument rules ? ;

argument-list = argument, { ",", argument } ;

argument = ( [ identifier, "=" ], expression )
         | block
         | lambda-expression
         | wildcard ;

index-suffix = "[", [ expression-list ], "]" ;

member-suffix = ".", ( member-identifier | "for" ) ;

(* Lifecycle members are invoked through a dedicated call form rather than
   ordinary member lookup. Every class has compiler-owned inherited @@new and
   @@init lifecycle machinery, but those inherited implementations are not
   automatically source-callable through `::`. A generic class with
   co.dap.generic(..., lifecycle=true) may source-define lifecycle overrides or
   overloads. Only matching developer-defined lifecycle candidates participate
   in ordinary source `::` lookup, and each candidate retains its declared
   accessibility. The lifecycle field is ignored for lifecycle semantics on
   generic structs/functions/methods. The call parentheses are part of the
   production so lifecycle members cannot be selected as first-class ordinary
   member values.
   Current invocation names map to declaration names as follows:
       new  -> @@new
       init -> @@init
   Future language-defined lifecycle names extend both the declaration-name
   and invocation-name sets; every lifecycle invocation still uses `::`.
   Semantic validation checks receiver kind, developer customization,
   accessibility, overload resolution, and construction state. *)
lifecycle-call-suffix = lifecycle-invocation-marker, lifecycle-invocation-name,
                        "(", [ argument-list ], ")",
                        lifecycle-call-context-guard ;

lifecycle-call-context-guard =
    ? zero-width semantic condition checked after receiver/type resolution:
      - lifecycle invocation performs lookup only in source-defined lifecycle
        override/overload candidates admitted for the resolved class;
      - such candidates can exist only on a generic co.class carrying valid
        co.dap.generic(types=[...], lifecycle=true) metadata;
      - the inherited compiler-provided lifecycle implementation is not an
        automatically exposed ordinary-source `::` candidate;
      - overload resolution selects a matching developer-defined lifecycle
        signature and ordinary accessibility rules must permit the caller to
        reach that candidate;
      - no matching accessible candidate is a compile-time error;
      - ordinary methods named new/init remain unrelated and continue to use
        ordinary '.' member lookup ? ;

(* value.to(TargetType) is therefore parsed by the existing member-suffix plus
   call-suffix path; explicit conversion/cast syntax introduces no dedicated
   cast production. Target-type validity is resolved semantically. *)

member-identifier =
    ? an identifier token other than "match"; "match" is routed through
      match-suffix so matcher-chain syntax has one parser path ? ;

(* No unprefixed composite literal exists. Lists, arrays, sets, maps, records,
   and objects all use the explicit Type{...} construction production. Blocks occur
   only in grammar positions that explicitly request a block; a bare block is not
   a general primary expression. *)
primary-expression = literal
                   | special-binding
                   | relationship-selector-expression
                   | parent-selector-expression
                   | compiler-owned-this-selector-expression
                   | this-receiver-expression
                   | refinement-candidate
                   | qualified-name
                   | grouped-expression
                   | tuple-expression
                   | composite-construction
                   | anonymous-class-expression
                   | anonymous-function-expression
                   | let-expression
                   | comprehension-expression ;

grouped-expression = "(", expression, ")" ;

tuple-expression = "(", expression, ",", expression,
                   { ",", expression }, ")" ;

composite-construction = type-postfix-expression, "{",
                         [ construction-element,
                           { ",", construction-element }, [ "," ] ], "}",
                         composite-construction-guard ;

construction-element = expression, [ ":", expression ] ;

composite-construction-guard =
    ? zero-width semantic condition: the prefix resolves to a constructible
      concrete type; list, array and set values admit element expressions, maps
      admit key:value entries, and object/record values admit identifier:value
      field initializers; "=" is forbidden as an entry binder because it belongs
      to metadata records, not typed JSON-like runtime representations; a generic
      collection declaration itself must first be specialized and named by a
      co.type alias ? ;

(* A built-in generic collection application must first be named by a concrete
   co.type alias. Every constructed value then uses Type{...}:

       StringList co.type = co.List(co.string)
       StringIntMap co.type = co.Map(co.string, co.int)
       IntSet co.type = co.Set(co.int)
       StringList{"A","B","C"}
       StringIntMap{"A": 1}
       IntSet{1,2,3}

   Type[...] remains indexing and Type(...) remains invocation. An unspecialized
   generic collection symbol is not a value constructor. *)

(* The reference's Builtin Collections registry. These names identify generic
   declarations and cannot directly stand in a value-construction prefix. Their
   specialized co.type aliases carry the applicable body form. *)
builtin-collection-type-name = "co.List"
                             | "co.Set"
                             | "co.Map"
                             | "co.Tree"
                             | "co.Trie"
                             | "co.Array"
                             | "co.Tuple" ;

anonymous-class-expression = "co.class", "{",
                             { class-member }, "}" ;

let-expression = "let", "(", "{", let-binding,
                 { ",", let-binding }, "}", ")",
                 ".in", "(", "{", expression, "}", ")" ;

let-binding = ( identifier | special-binding ), "=", expression ;

comprehension-expression = "for", "(", comprehension-binding, ")",
                           ".yield", "(", expression-list, ")" ;

comprehension-binding = pattern, "<-", expression ;

match-suffix = ".match", "(", [ expression ], ")",
               match-case, { match-case }, [ match-default ] ;

multi-symbol-infix-operator-boundary-guard =
    ? zero-width condition: a multi-symbol infix operator has an explicit
      boundary on both operand-facing sides; a boundary is whitespace, a
      comment, or an applicable delimiter, checked before separators are
      discarded ? ;

multi-symbol-range-operator-boundary-guard =
    ? zero-width condition: a multi-symbol range operator has an explicit
      boundary on every side for which an operand is present ? ;

extended-operator-expression =
    ? expression containing a registered project-local custom operator,
      parsed by precedence climbing from its declared alpha-supported fixity,
      precedence, associativity, and arity; ∪ and ∩ are not handled by this
      extension path because they are language-owned operators in the ordinary
      precedence grammar; explicitly reserved future fixities remain
      unsupported; every registered multi-symbol operator satisfies the
      operand-facing boundary rule for its fixity ? ;


(* ====================================================================== *)

(* 12. Informative control-chain shapes (not parser entry paths) *)

(* ====================================================================== *)

(* Control vocabulary in the current alpha profile:
     - .then(X) conditionally evaluates X once; X may be a block or value.
     - .then(...) may participate in .otherwise(condition) selection chains.
     - .loop(X) is a single-condition repetition form and cannot participate in
       .otherwise(condition) chains or take a .default(X) fallback.
     - .otherwise(condition) always introduces another selection condition; there
       is no conditionless .otherwise form.
     - .default(X) is the optional terminal fallback of a selection chain; it is
       not part of loop syntax.
     - .each(...) is itself element iteration and cannot be followed by .loop.
       The explicit-binding form is .each(index-or-key, value, action), where
       action may be any expression (including a block or anonymous function) or
       a lambda-expression. The single-argument form accepts a callable callback;
       semantic analysis requires it to match the receiver's iteration tuple.
   These informative productions document recognizable shapes; ordinary parsing
   still proceeds through the general postfix/member/call grammar above. *)

informative-condition-chain =
    "(", expression, ")", ".then", "(", block, ")",
    { ".otherwise", "(", expression, ")", ".then", "(", block, ")" },
    [ ".default", "(", block, ")" ] ;

informative-loop-chain =
    "(", expression, ")", ".loop", "(", block, ")" ;

informative-labeled-loop =
    label-declaration, ":", "(", expression, ")", ".loop", "(", block, ")" ;

informative-ternary-chain =
    "(", expression, ")", ".then", "(", expression, ")",
    { ".otherwise", "(", expression, ")", ".then", "(", expression, ")" },
    ".default", "(", expression, ")" ;

informative-each-chain =
    postfix-expression, ".each", "(",
    ( ( identifier | "_" ), ",", identifier, ",",
      ( expression | lambda-expression )
    | ( expression | lambda-expression ) ), ")" ;

informative-contains-chain =
    postfix-expression, ".contains", "(", expression, ")",
    ".then", "(", block, ")",
    [ ".default", "(", block, ")" ] ;

informative-pipeline-chain =
    postfix-expression,
    { ( ".filter" | ".map" | ".reduce" | ".forEach"
      | ".sortBy" | ".groupBy" | ".fold" ),
      "(", argument-list, ")" } ;


(* ====================================================================== *)

(* 13. Literals and lexical grammar *)

(* ====================================================================== *)

literal = builtin-literal ;

builtin-literal = integer-literal
                | floating-literal
                | string-literal
                | character-literal
                | boolean-literal
                | none-literal ;

integer-literal = ( binary-integer-literal
                  | octal-integer-literal
                  | decimal-integer-literal
                  | hexadecimal-integer-literal ),
                  [ integer-suffix ] ;

binary-integer-literal = ( "0b" | "0B" ), binary-digit-sequence ;

octal-integer-literal = "0", [ octal-digit-sequence ] ;

decimal-integer-literal = nonzero-digit, { decimal-digit } ;

hexadecimal-integer-literal = hexadecimal-prefix,
                              hexadecimal-digit-sequence ;

hexadecimal-prefix = "0x" | "0X" ;

binary-digit-sequence = binary-digit, { binary-digit } ;

octal-digit-sequence = octal-digit, { octal-digit } ;

decimal-digit-sequence = decimal-digit, { decimal-digit } ;

hexadecimal-digit-sequence = hexadecimal-digit, { hexadecimal-digit } ;

integer-suffix = unsigned-suffix,
                 [ long-suffix | long-long-suffix | size-suffix ]
               | long-suffix, [ unsigned-suffix ]
               | long-long-suffix, [ unsigned-suffix ]
               | size-suffix, [ unsigned-suffix ] ;

unsigned-suffix = "u" | "U" ;

long-suffix = "l" | "L" ;

long-long-suffix = "ll" | "LL" ;

size-suffix = "z" | "Z" ;

floating-literal = decimal-floating-literal
                 | hexadecimal-floating-literal ;

decimal-floating-literal = fractional-constant,
                           [ exponent-part ],
                           [ floating-point-suffix ]
                         | decimal-digit-sequence,
                           exponent-part,
                           [ floating-point-suffix ] ;

fractional-constant = decimal-digit-sequence, ".",
                      decimal-digit-sequence ;

hexadecimal-floating-literal = hexadecimal-prefix,
                               ( hexadecimal-fractional-constant
                               | hexadecimal-digit-sequence ),
                               binary-exponent-part,
                               [ floating-point-suffix ] ;

hexadecimal-fractional-constant = hexadecimal-digit-sequence, ".",
                                  hexadecimal-digit-sequence ;

exponent-part = ( "e" | "E" ), [ sign ], decimal-digit-sequence ;

binary-exponent-part = ( "p" | "P" ), [ sign ], decimal-digit-sequence ;

sign = "+" | "-" ;

floating-point-suffix = "f" | "F" | "l" | "L"
                      | "f16" | "F16"
                      | "f32" | "F32"
                      | "f64" | "F64"
                      | "f128" | "F128"
                      | "bf16" | "BF16" ;

character-literal = single-quote, alpha-basic-c-character, single-quote ;

alpha-basic-c-character =
    ? any translation character except apostrophe, backslash,
      carriage return, or line feed ? ;

string-literal = double-quote, { alpha-basic-s-character }, double-quote ;

alpha-basic-s-character =
    ? any translation character except double quote, backslash,
      carriage return, or line feed ? ;

double-quote = ? Unicode scalar value U+0022 ? ;

single-quote = ? Unicode scalar value U+0027 ? ;

backslash = ? Unicode scalar value U+005C ? ;

boolean-literal = "co.const.true" | "co.const.false" ;

none-literal = "co.const.none" ;

byte-order-mark =
    ? U+FEFF, permitted only as the first code point of a source file ? ;

(* Structured control labels use a dedicated apostrophe-prefixed lexical form.
   The same token shape is used in declarations and references:
       'outer:     declaration prefix
       'outer      reference
   A character literal remains distinct because it has a closing apostrophe,
   e.g. 'c'. Labels are not ordinary identifiers and are not operators. *)
label-identifier =
    single-quote, identifier, label-identifier-guard ;

label-identifier-guard =
    ? a zero-width assertion that the next character is not single-quote;
      this prevents a complete character literal such as 'c' from being
      tokenized as a label identifier plus a trailing apostrophe ? ;

identifier = identifier-head, { "_", identifier-segment },
             identifier-trailing-guard ;

identifier-trailing-guard =
    ? a zero-width assertion that the next character is not "_" ? ;

identifier-head = ascii-letter, { ascii-alphanumeric } ;

identifier-segment = ascii-alphanumeric, { ascii-alphanumeric } ;

ascii-alphanumeric = ascii-letter | decimal-digit ;

ascii-letter = "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H"
             | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P"
             | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X"
             | "Y" | "Z"
             | "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h"
             | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p"
             | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x"
             | "y" | "z" ;

binary-digit = "0" | "1" ;

octal-digit = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" ;

hexadecimal-digit = decimal-digit
                  | "a" | "b" | "c" | "d" | "e" | "f"
                  | "A" | "B" | "C" | "D" | "E" | "F" ;

digit = decimal-digit ;

decimal-digit = "0" | nonzero-digit ;

nonzero-digit = "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;

hard-reserved-word = "co" | "let" | "this" | "for" | "fΦλ" ;

contextual-keyword = "forall" ;

(* `::`, `->|`, and `^=>` are active structural markers consumed respectively
   by lifecycle-call-suffix, break-statement, and
   enclosing-callable-return-statement. They are not ordinary infix operators
   and are therefore intentionally absent from reserved-operator. *)
lifecycle-invocation-marker = "::" ;

reserved-operator = "->>" | "<->" | "`" | backslash | "#" ;

predeclared-operator-glyph = "∪" | "∩" ;

reserved-future-operator =
    ? a complete documented reserved/future symbolic spelling that is not a
      current built-in operator and not a predeclared-operator-glyph; the lexer
      recognizes the complete spelling and the parser reports unsupported ? ;

token = label-identifier
      | identifier
      | keyword-token
      | literal
      | delimiter-token
      | predeclared-operator-glyph
      | reserved-operator
      | symbolic-token
      | reserved-future-operator ;

keyword-token = hard-reserved-word ;

delimiter-token = "(" | ")" | "{" | "}" | "[" | "]"
                | "," | ";" | double-quote | single-quote ;

symbolic-token = ? the complete maximal contiguous run of one or more symbol
                   characters after comments, literals, and closed composite
                   spellings are recognized; the run is preserved whole for
                   contextual classification and is never split as a
                   fallback ? ;

token-separator = white-space ;

line-comment = "//", { ? any Unicode scalar value except CR or LF ? } ;

block-comment = "/*", { block-comment-character }, "*/" ;

block-comment-character = ? any Unicode scalar value that does not begin the
                            two-character sequence */ ? ;

line-break = "\r\n" | "\n" | "\r" ;

horizontal-white-space = " " | "\t" | "\f" ;

white-space = horizontal-white-space | line-break | line-comment | block-comment ;


(* ====================================================================== *)

(* 14. Operator declarations inside components/operators/component.fol *)

(* ====================================================================== *)

(* The operator component uses the same component-surface-file parser root as
   every other component.fol. Filesystem context restricts co.operator to
   components/operators/component.fol and forbids all non-operator members
   there. The scanner still recognizes each declaration head as one maximal
   symbolic run while this component builds the owning ProjectOperatorTable. *)

operator-declaration = operator-symbol, "co.operator", "=",
                       operator-body, statement-end,
                       operator-declaration-context-guard ;

operator-declaration-context-guard =
    ? zero-width structural condition: declaration occurs only inside the
      _ co.component body of components/operators/component.fol; that
      component body contains only operator declarations and its project kind
      is either executable application or projected application library ? ;

operator-body = "{", operator-property,
                { ",", operator-property }, "}" ;

operator-property = "fixity", declarative-attribute-binder, operator-fixity
                  | "precedence", declarative-attribute-binder,
                    ( "0" | decimal-integer-literal )
                  | "associativity", declarative-attribute-binder,
                    operator-associativity
                  | "arity", declarative-attribute-binder, operator-arity
                  | "commutative", declarative-attribute-binder,
                    boolean-literal
                  | "idempotent", declarative-attribute-binder,
                    boolean-literal
                  | "identity", declarative-attribute-binder,
                    operator-identity-value
                  | "foldable", declarative-attribute-binder, boolean-literal
                  | "vectorizable", declarative-attribute-binder,
                    boolean-literal
                  | "distributes_over", declarative-attribute-binder,
                    operator-symbol-list
                  | "desugar", declarative-attribute-binder, string-literal ;

operator-fixity = "co.operator.fixity.infix"
                | "co.operator.fixity.prefix"
                | "co.operator.fixity.postfix"
                | reserved-future-operator-fixity ;

reserved-future-operator-fixity =
      "co.operator.fixity.circumfix"
    | "co.operator.fixity.postcircumfix"
    | "co.operator.fixity.precircumfix"
    | "co.operator.fixity.mixfix"
    | "co.operator.fixity.ternary"
    | "co.operator.fixity.distfix" ;

operator-associativity = "co.operator.associativity.left"
                       | "co.operator.associativity.right"
                       | "co.operator.associativity.none" ;

operator-arity = "co.operator.arity.unary"
               | "co.operator.arity.binary"
               | "co.operator.arity.ternary"
               | decimal-integer-literal ;

operator-identity-value = literal ;

operator-symbol-list = "[", [ operator-symbol-reference,
                       { ",", operator-symbol-reference } ], "]" ;

operator-symbol-reference = character-literal | string-literal ;

operator-symbol = ? a maximal run of one or more symbol characters, where a
                    symbol character is any character that is not an ASCII
                    letter, digit, underscore, whitespace, or one of the
                    delimiters ( ) { } [ ] , ; " ' ; the run must not be a
                    language-owned or hard-reserved symbol and must not contain
                    // or /* ? ;
```
