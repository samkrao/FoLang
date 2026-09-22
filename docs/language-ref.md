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

Packages have following access specifiers 

  1. @co.dap.local
  2. @co.dap.public (is default)

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

#### Specialized Types 

```folang

//Generic types must be aliased as derived types
IntList      co.type = co.List(co.int);
IntSet       co.type = co.Set(co.int);
StringIntMap co.type =
    co.Map(co.string, co.int);

```

When Geneircs are initialized with concrete type, Folang doesn't allow them to be inine instead they are needed to be declared as a type


***

### User Defined Types

Folang supports rich User Defined types, for various use cases and purposes.

Folang doesn't allow inner or nested or anonymoust types inside a type definiton.

#### CStructs

```folang

// Rect.fol

_ co.cstruct = {
    origin Point;
    width  co.int;
    height co.int;
}

```
`co.cstruct` is a C-like value type: it is passed by value, has a simple memory layout, and is safe to cross supported ABI boundaries.

**Worth Noting**  lke packages where names are fetched from folder names top level types names are fetched from file name and the name is canonical (case insensitive).

`_` is the place holder 

#### Structs

```folang
// Employee.fol
_ co.struct = {
    id      co.int;
    name    co.string;
    address Address;
}

```
Folang structs are pure types, no initialization no additional info like const, voldatile, final, lazy etc.,

No method of functions, to make structs act like objects we need to use companion units and asociated functions

structs don't support inheritance or any other oops concepts

structs allows composition or embedding.



#### interfaces

```folang
// IEmployee.fol
_ co.interface = {
    storeEmployee(emp Employee)->(Employee);
}
```
Folang interfaces are pure specification not subtype or partially implemented methods or state is allowed.

#### Classes

```folang
@co.dap.oops(
    interfaces=[someInterface1, someInterface2, someInterface3],
    classes=[someClass1, someClass2],
    mixins=[someMixin1, someMixin2, someMixin3],
    traits=[someTrait1, someTrait2, someTrait3],
)
// test.fol
_ co.class = {
    getTest(id co.int)->(test) ={}
}
```

> co.dap.oops annotation is optional

##### Classes with Operators

```folang
// Employee.fol
_ co.class = {
    @co.dap.operator(symbol="+")
    add(other Employee)->(Employee) = {
       // implicit instance method: operands are this and other
    }

    @co.dap.operator(symbol="==")
    @co.dap.static
    equals(
        left  Employee,
        right Employee
    )->(co.bool) = {
        // static operator implementation
    }

    @co.dap.operator(symbol=">")
    @co.dap.class
    greater(
        left  Employee,
        right Employee
    )->(co.bool) = {
        // class-associated operator implementation
    }

}
```

Folang classes can contain member fields and member methods

They have the following access specifiers

   1. @co.dap.public
   2. @co.dap.private ( default )
   3. @co.dap.internal
   4. @co.dap.protected
   5. @co.dap.package 


#### Mixins

```folang

_ co.mixin={

    someNum co.int;

    someFun1()->()={
        co.out.println(someNum);
    }

    @co.dap.abstract
    someotherFun()->();

    @co.dap.virtual
    someVirtFun()->()={
        ...
    }


}
```
> A mixin is the dedicated abstract-class-like composition form; it avoids declaring an ordinary class merely to mark it abstract.

> A mixin may contain state, abstract methods, fully implemented methods, and virtual methods where the mixin rules permit them.

> A class incorporates a mixin by listing it in the `mixins=[...]` field of `@co.dap.oops`, implements its abstract methods, and may override its virtual methods.



#### Traits


// EmployeeTrait.fol

```folang
_ co.trait={

    @co.dap.abstract
    someFunction()->();

    somerealFun()->()={
        co.out.println("Doing real work" );
    }

}
```
> A trait is interface-like but may provide default function implementations. A trait carries no instance state.

> A class participating in the `@co.dap.oops` model includes a trait by listing it in the annotation's `traits=[...]` field.

> A consuming class must implement every abstract function that remains unsatisfied.


#### Extensions

```folang
// EmployeeExtension.fol

_ co.extension->(fortype=somePkg.Employee) = {

    someFun()->() = {
        co.out.println(this.someName);
    }

    @co.dap.class
    someOtherFun()->() = {
        co.out.println(this.clsVariable);
    }
}
```

A `co.extension` is a reusable collection of fully implemented methods that adds behavior to one explicitly selected class **without creating a subclass or changing the target class's nominal type identity or inheritance hierarchy**. The extension chooses its target through the mandatory `fortype` argument; the target class does not adopt the extension through `@co.dap.oops`.

Folang extensions are after thought to enhance existing class it can be done two ways 

1. Inheritance
2. Extensions

Inheritance is costly mechanism and also for simple addition of behaviour it is not advisable to inherit a class.

In such scenarios Folang allows extensions to classes.

#### Signatures
```folang
// StackSignature.fol
_ co.signature = {
    T     co.associatedType;
    Stack co.type;

    empty(value T)->(Stack);
    push(value T, stack Stack)->(Stack);
    pop(stack Stack)->(T, Stack);
}
```

Signatures contain Both methods and shared state. They fall between structs and classes  where we want multiple implementations but not as heavy as classes.



#### Modules

```folang
// ListStackModule.fol
_ co.module->(matches=StackSignature) = {
    T co.associatedType = co.int;
    Stack co.type = co.List(T);

    empty(value T)->(Stack) = { ... }
    push(value T, stack Stack)->(Stack) = { ... }
    pop(stack Stack)->(T, Stack) = { ... }
}
```


Module gives you  a kind of final non inheritable class model of main stream languages.

### Type classes

Folang, typeclass defines a type-directed behavioral contract used for ad-hoc polymorphism. An instance supplies the implementation of that contract for a specific type or valid type combination

#### Functors

```folang
//Functor.fol
@co.dap.typeclass(
    kind=Functor,
    shape=(F(_)),
    aliases=[
        {name=MapFunction,    type=(A)->(B)},
        {name=InputContainer, type=F(A)},
        {name=ResultContainer,type=F(B)}
    ]
)
_ co.typeclass = {
    map(value InputContainer, f MapFunction) -> (ResultContainer);
}
```

#### Applicatives
```folang
//Applicative.fol
@co.dap.typeclass(
    kind=Applicative,
    shape=(F(_)),
    aliases=[
        {name=MapFunction,      type=(A)->(B)},
        {name=FunctionContainer,type=F(MapFunction)},
        {name=InputContainer,   type=F(A)},
        {name=ResultContainer,  type=F(B)}
    ]
)
_ co.typeclass = {
    pure(x A) -> (InputContainer);
    apply(fab FunctionContainer, fa InputContainer) -> (ResultContainer);
}
```

#### Monads

```folang
//Monad.fol
@co.dap.typeclass(
    kind=Monad,
    shape=(F(_)),
    aliases=[
        {name=InputContainer, type=F(A)},
        {name=ResultContainer,type=F(B)},
        {name=FlatMapFunction,type=(A)->(ResultContainer)}
    ]
)
_ co.typeclass = {
    pure(x A) -> (InputContainer);
    flatMap(fa InputContainer, f FlatMapFunction) -> (ResultContainer);
}
```
#### Monoids


```folang
//Monoid.fol
@co.dap.typeclass(kind=Monoid, shape=(T))
_ co.typeclass = {
    empty() -> (T);
    combine(a T, b T) -> (T);
}
```

#### Transformers
```folang
//Transformer.fol
@co.dap.typeclass(
    kind=Transformer,
    shape=(F(_), G(_)),
    aliases=[
        {name=MapFunction,    type=(A)->(B)},
        {name=InputContainer, type=F(A)},
        {name=ResultContainer,type=G(B)}
    ]
)
_ co.typeclass = {
    map(value InputContainer, f MapFunction) -> (ResultContainer);
}
```

### Instances

Instances are implementations for one of the typclass type

```folang
// ListToSetTransformer.fol
_ co.instance->(for=Transformer, types=[co.List, co.Set]) = {
    map(value InputContainer, f MapFunction)->(ResultContainer) = {
        result := ResultContainer{};
        value.each(_, item, { result.insert(f(item)) });
        $=> result;
    }
}
```


### Custom Matcher

```folang
// PositiveEvenMatcher.fol
@co.dap.matcher
_ co.matcher->(type=co.int) = {
    matchCase(
        value   co.int,
        pattern co.untyped
    )->(co.int, co.MatchBindings) = {
        // user logic
        // 0 = no match, >0 = match
    }
}
```


### Object types

#### Annotations
```folang
// Annotation implementation — named singleton object, can carry data


// myAnnotation.fol

@co.dap.annotation
_ co.object = {
    value   co.string;
    enabled co.bool;
}
```

Folang provides a way to define annotations and use them using reflections in the code, to make decisions

#### Associated Objects

```folang
// AuditAnnotation.fol
_ co.object->(for=someAnnotation) = {
    ...
}
```
Folang doesn't provide global or shared variables, in classes the only way to have them is through associated objects they act like singleton and associate with single/multiple types at the same time.

```folang
// ProducerConsumerShared.fol
_ co.object->(
    for=[Producer, Consumer]
) = {
    @co.dap.const capacity co.int = 100;
    @co.dap.final configuration QueueConfiguration = loadConfiguration();

    queue Queue;
    queueLock co.lock;

    enqueue(value Item)->() = {
        lock(queueLock) {
            queue.add(value);
        }
    }

    dequeue()->(Item) = {
        lock(queueLock) {
            $=> queue.remove();
        }
    }
}
```

### Unions

`co.union` declares an untagged ADT. Its body lists the alternative members defined by the union.

```folang
 // myUnion.fol
 _ co.union={
    intValue co.int;
    strValue co.string;
}
```

### Blocks

#### Named Blocks
```folang
labelBlock co.block = {
}

labelBlock.expand();
```

Named blocks are reusable code

#### Anonymous Blocks

An ordinary anonymous block is a lexical execution scope:

```folang
{
    // statements
}
```
#### Label blocks

```folang
'outer: {
    // statements

    (someCondition).then({
        $->| 'outer;
    });
}
```


### Enums
`co.enum` is FoLang's closed tagged algebraic-data-type declaration. The body of an enum contains only **state declarations**. States are owned by the enclosing enum and form its complete set of alternatives.

```folang
// Shape.fol
_ co.enum = {
    Circle(radius co.float),
    Square(side co.float),
    Rectangle(width co.float, height co.float),
    Point
}
```

### Enum States and State Functions

Every enum member is a **state**. A state is not an independent type and does not introduce a struct, class, object, or constructor declaration.

A state with no parameters is a complete value of the enclosing enum type. It is referenced directly, without `()`:

```folang
origin Shape = Shape.Point;
```

`Shape.Point()` is invalid because `Point` declares no parameters and is not a zero-argument ordinary function call.

A state with one or more parameters defines a compiler-provided **state function**. Every state-function parameter is intrinsically named. The declaration therefore gives each payload component both a name and a type:

Calling a state function produces a value of the enclosing enum type. A call must bind every supplied payload value explicitly by parameter name using `name=value`; a positional state-function call is invalid:

```folang
circle Shape = Shape.Circle(radius=5.0);
square Shape = Shape.Square(side=4.0);
rect   Shape = Shape.Rectangle(width=4.0, height=6.0);
rect2  Shape = Shape.Rectangle(height=6.0, width=4.0); // order is irrelevant

// invalid: positional payloads are never accepted for enum state functions
// Shape.Circle(5.0);
// Shape.Rectangle(4.0, 6.0);
```

Conceptually:

```text
value                               static type    state       named payload
Shape.Circle(radius=5.0)            Shape          Circle      {radius=5.0}
Shape.Circle(radius=10.0)           Shape          Circle      {radius=10.0}
Shape.Square(side=5.0)              Shape          Square      {side=5.0}
Shape.Point                         Shape          Point       {}
```


### Symbols

Folang supports generic symbols through `co.symbol`

### Function Shaped Declarations

#### Functions

```folang

add ( a co.int, b co.int )->(co.int)={
    $ => a + b;
}

```
##### Multiple Return
```folang
returnPair( a co.int, b co.int)->(co.int, co.int)={
    $ => b, a;
}
```
##### Inner function

```folang
someFun( a co.int, b co.int)->(co.int) = {

    someInner(b co.int)->(co.int)= {
        $ => b * 2;
    }

    $ => someInner( a) + b;

}
```
##### Anonymous Inner Functions

Anonymous function cannot be standalone functions

```folang

someFun( a co.int, b co.int )->(co.int)={

    $ => (a co.int)->(co.int){
        $ => a * 4;
    }(b) + a;

}
```

Anonymous functions either should be assigned to variable or immediately invoked like above

```folang
someFun( a co.int, b co.int )->(co.int)={

    temp co.function = (a co.int)->(co.int){
        $ => a * 4;
    }

    temp( b) + a;

}
```

#### Curried Functions

```folang

add ( a co.int)(b co.int)(c co.int)->(co.int)={
    $ => a + b + c;
}

someVar co.function = add(1);
someOtherVar co.function = someVar(2);

result co.int =  someOtherVar(3);

```

#### Named Parameter Functions

```folang
// someNamedParam.unit.fol
fun1(~k co.int, ~v co.int)->()={

}

Usage:
  fun1(v=10,k=20); // valid
  fun1(10,20); //valid here k =10 and v=20

```

#### Optional Parameter Functions
//someOptional.unit.fol
```folang

fun1(k? co.int)->()={
    k.omitted.then({

    }).default({

    });

}

```

#### Default Parameter Functions

// somefununit.unit.fol
```folang


fun1(k co.int, b co.char = 'A')->(co.int, co.char)={
}


usage:
  fun1(10, 'B'); // k = 10 and b = 'B'
  fun1(10);      // k = 10 and b = 'A', the declared default value


``

#### Variadic Functions

//someCurried.unit.fol
```folang
fun1 (k co.int, ...b co.char)->(co.int, co.char)={
}

```

#### Closures

//someClosure.unit.fol
```folang
IntAdder co.type = (co.int)->(co.int);

adder()->(IntAdder) ={
    sum co.int = 0;
    $=> (x co.int)->(co.int){
        sum += x;
        $=> sum;
    };
}

```
Any Function shape returning function then the inner function is called closure
which captures state of its outer function.

#### Expression Bodied Functions

```folang
    IntBinary co.type = (co.int, co.int)->(co.int);
    add IntBinary(a, b) = a + b;
```

> name CallableType(parameterNames) = expression;

#### Function patterns 

Folang supports function patterns through  Expression bodied functions

```folang
classify(n co.int)->(co.string) =
    n.match()
        .case(v: v > 0 => "positive")
        .case(v: v < 0 => "negative")
        .default("zero");
```

#### Expression Bodied function in polymorphic types
```folang
PolyId co.type = co.polymorphic({U}, (U)->(U));

identity PolyId(value) = value;
```

#### Local, Nested types

##### Local types

Folang provides `@co.dap.Local` for all the top level definitions including functions to emulate the kind of inner functions/types behavior

```folang
// EmployeeAddress.fol
@co.dap.local(for=[hr.employee.Employee,hr.manager.Manager])
_ co.struct = {
    street co.string;
    city   co.string;
}
```


```folang
// Employee.fol
_ co.struct = {
    id      co.int;
    name    co.string;
    address EmployeeAddress; // composition
}
```

if local is to refer single type then `for=hr.employee.Employee` is valid in case of attaching to multiple types we need list representation

##### Nested types


```folang
// EmployeeAddress.fol
@co.dap.nested(target=hr.employee.Employee)
_ co.struct = {
    street co.string;
    city   co.string;
}
```


```folang
// Employee.fol
_ co.struct = {
    id      co.int;
    name    co.string;
    address EmployeeAddress; // composition
}
```

> `target` is always single

```folang
@co.dap.local(for=addFunc(co.int, co.int)->(co.int))
someFunc(a co.int)->(co.int)={
    $=> a *100;
}

addFunc(a co.int , b co.int)->(co.int)={
    $=> someFunc(b) + a;
}

subFunc( a co.int, b co.int)->( co.int)={
    $ => someFunc(a) -b; // compiler error as someFunc is for only addFunc
}
```
Nested vs Local

Nested functions/types behave exactly like inner types/functions they can access outer types state/variables/memebers

```folang
@co.dap.local(for=addFunc(co.int, co.int)->(co.int))
someFunc()->(co.int)={
    $=> a *100;  // compiler error as a is undefined
}

addFunc(a co.int , b co.int)->(co.int)={
    $=> someFunc(b) + a;
}

@co.dap.nested(target=addFunc(co.int, co.int)->(co.int))
someFunc()->(co.int)={
    $=> a *100; //works fine as addFunc defined the variable as parameter
}

addFunc(a co.int , b co.int)->(co.int)={
    $=> someFunc(b) + a;
}
```

#### Deferred Function


#### Function scopes

Folang provides 3 different scopes to functions 

    1. @co.dap.dynamicscope
    2. @co.dap.mixedscope

default `lexicalscope`


### Chained Functions

Folang provide a feature where you can chain the function call by passing the results to the next function.

```folang
addFunc(a co.int, b co.int)->(co.int)={
    $=> a + b;
}

subFunc( a co.int, b co.int)->(co.int)={
    $=> a - b;
}

myFunc(a co.int, b co.int)->(co.int,co.int)=>>addFunc($1,$2);
someOtherFunc()->(co.int, co.int)=>>subFunc($1,$2);
```
> $ is a bind variable which in this case bound to results

> $1 .. $N declared return order.

#### Decorators
Folang provides decorators for adding addition behaviour to functions and/or methods

```folang
  @co.dap.decorator
  myDecorator(target co.function)->(co.function) = { }
```

Attribbutes of @co.dap.decorator

    scope= runtime, compiletime
    when= before, after, around , afterEffect


#### Macros
 
 ```folang
 
    // b. Escape assign
    @co.dap.macro
    yes_esc_assign()->(co.untyped)={
        $=> co.macro.quote({
            co.macro.esc(y) = 42;
            co.out.println("Inside macro: y = ", y);
        });
    }

    @co.dap.macro
    debug(expr)->(co.untyped)={
        tmp := co.macro.gensym(co.var, "tmp");
        $=> co.macro.quote({
            tmp = co.macro.esc(expr);
            co.out.println("Result: ", tmp);
            tmp;
        });
    }

if else macro

    @co.dap.macro(
        group={items=["if","else"], chain=true},
        sugarform={forms=["if expr block"]},
        bind={vars=["x"]},
        isolate={vars=["temp", "index"]},
        gensym={prefix="tmp_"},
        hygienic=true,
        argtransform={param="body", wrap="lambda", whentype="block"},
        desugar={exprs=["if($cond) { $block }" => "if($cond,$block)"]},
        mode="inject"
    )
    if(condition expr, body block)->()={}

    blockormacro co.kind = block | macro

    @co.dap.macro(
        group={items=["if","else"], chain=true},
        sugarform={forms=["else block","else if"]},
        chainswith={macro="if", position="immediate", required=true},
        argtransform={param="body", wrap="lambda", whentype="block"},
        standalone=false,
        desugar={exprs=[
            "else if($cond) { $block }" => "else(if($cond, $block))",
            "else { $elseblock }" => "else($elseblock)"
        ]},
    )
    else(body blockormacro)->()={}

```

Folang Provides macros to create constructs that wraps folang logic. Macros in Folang are compile time rewrite to AST.

#### Extension methods

```folang
    @co.dap.extension(fortype=co.string, what=extends)
    upperCase()->(co.string) = {
        $=> this.upper();
    }

    @co.dap.extension(fortype=[co.string], what=overrides)
    equals(str co.string)->(co.bool) = {
        $=> this == str;
    }
```
Folang provides extension methods to extend structs and/or modules behaviour. For classes we folang recommends to use extensions

#### Templates


#### Indexers

#### Execution Model
#### Native methods
#### Associated Functions


### Comprehensions


### Lambda Expressions


### Expression Bodied Named Function

### Pattern Matching

### Condition Loops and Ternary operations


### Units

#### Companion Units
#### Normal Units

### Generics

### Exception/Error Handling

### Annotation Decorators, Pragmas and Directives

### Dynamic Vm Runtime

### Native code.

### FFI and ABI


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

[{{FOLANG_EBNF}}](./grammar/folang.ebnf)
