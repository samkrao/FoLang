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

#### Built-In Operator Parse Table

Larger precedence numbers bind more tightly. Precedence and associativity determine the parse tree; FoLang's left-to-right operand evaluation rule independently determines evaluation order within that tree.

| Precedence | Operator/form | Fixity | Associativity | Arity / parse role |
|---:|---|---|---|---|
| 700 | call `(...)`, index `[...]`, member `.`, lifecycle call `::name(...)`, postfix `!` | postfix | left | call/index/member/lifecycle-call syntax; postfix `!` unary |
| 650 | `**` | infix | right | binary |
| 600 | `+`, `-`, `!` | prefix | right | unary |
| 550 | `*`, `/`, `%` | infix | left | binary |
| 500 | `∪`, `∩` | infix | left | binary |
| 450 | `+`, `-` | infix | left | binary |
| 400 | `..`, `<..`, `..<`, `<..<` | infix/range | none | range form; a bound may be omitted where the range grammar permits |
| 350 | `<`, `<=`, `>`, `>=`, `:>`,`<:` | infix | none | binary |
| 300 | `==`, `!=` | infix | none | binary |
| 250 | `&` | infix | left | binary |
| 200 | `^` | infix | left | binary |
| 150 | `|` | infix | left | binary |
| 100 | `&&` | infix | left | binary; short-circuit |
| 50 | `||` | infix | left | binary; short-circuit |
| 10 | `=`, `+=`, `-=`, `*=`, `/=`, `%=`, `**=`, `&=`, `^=`, `|=` | infix assignment | right | binary assignment |

The definition spellings `:=`, `::=`, and `?=` are statement-level definition operators, not general expression operators, so they do not receive an expression-precedence level. Structural spellings such as `::`, `=>`, `=>>`, `->`, `<-`, `->>`, and `<->` are likewise not ordinary expression operators merely because they contain symbol characters.


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

```folang
  ⊗ co.operator = {
        fixity = co.operator.fixity.infix,
        precedence = 60,
        associativity = co.operator.associativity.left,
        arity = co.operator.arity.binary,
        commutative = co.const.false,
        idempotent = co.const.false,
        identity = co.const.none,
        foldable = co.const.false,
        vectorizable = co.const.false,
        distributes_over = [],
        desugar = "intrinsic:tensor_product"
    };

    +- co.operator = {
        fixity = co.operator.fixity.infix,
        precedence = 60,
        associativity = co.operator.associativity.left,
        arity = co.operator.arity.binary
    };
```

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

***

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
***

### Instances

Instances are implementations for one of the typclass type

```folang
// ListFunctor.fol
_ co.instance->(for=Functor, type=co.List) = {
    map(value InputContainer, f MapFunction)->(ResultContainer) = {
        result := ResultContainer{};
        value.each(_, item, { result.append(f(item)) });
        $=> result;
    }
}
```


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
#### Using an Instance

An instance is selected **by name**. There is no implicit search.

```folang
@co.ddap.import(package="abc.tc", as="tc")

IntList co.type = co.List(co.int);
xs IntList = IntList{1, 2, 3};
double(x co.int)->(co.int) = { $=> x * 2; }

ys := tc.ListFunctor.map(xs, double);
```
#### Activating Instance Methods

An instance may also be **activated**, which makes its functions callable as
methods on the receiver. Activation uses the directive `@co.dap.use`

```folang
@co.ddap.import(package="abc.tc", as="tc")
@co.ddap.use(from="tc.ListFunctor", methods=[map, reduce])

ys := xs.map(double);        // resolves to tc.ListFunctor.map(xs, double)
```

***

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

***

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
***

### Unions

`co.union` declares an untagged ADT. Its body lists the alternative members defined by the union.

```folang
 // myUnion.fol
 _ co.union={
    intValue co.int;
    strValue co.string;
}
```

***

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

***

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

#### Enum States and State Functions

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

***

### Symbols

Folang supports generic symbols through `co.symbol`

***

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

#### Named Callable-Type Implementation 

An ordinary named function may implement a named `co.type` whose resolved underlying type is callable. This form reuses the callable contract instead of restating parameter and result types.

Canonical forms:

```folang
name CallableType(parameterNames) = expression; //expression bodied functions

name CallableType(parameterNames) = {
    ...
}
```

Example with an ordinary function type:

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

#### Inline Functions
```folang
    @co.dap.inline
    add(a co.int, b co.int)->(co.int) ={
        $=> a + b;
    }
```
Function iniling is making the code copied to call site.

#### Deferred Function
```folang
someErrorFun(a co.int)->() = {
    resource Resource = acquireResource();

    @co.dap.defer
    cleanup(resource Resource, originalArgument co.int)->() = {
        releaseResource(resource);
        logArgument(originalArgument);
    }(resource, a);

    performOperation(resource);
}
```
Folang provides deferred function feature which executes at the end of function irrespective of successful or effect occurred or not.

#### Function scopes

Folang provides 3 different scopes to functions 

    1. @co.dap.dynamicscope
    2. @co.dap.mixedscope

default `lexicalscope`


#### Chained Functions

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

##### Activating Extension methods

Activation uses the directive `@co.dap.use` For the current package, omit `from`:

```folang
@co.ddap.use(methods=[equals, upperCase])
k.upperCase();
```

For another package, use its alias or complete package path:

```folang
@co.ddap.import(package="text.util", as="tu")

@co.ddap.use(from="tu", methods=[upperCase])
@co.ddap.use(from="text.util", methods=[upperCase])
```


#### Templates
```folang
    @co.dap.template
    add(a co.int, b co.int)->(co.int) ={
        $=> a + b;
    }
```
```folang
  @co.dap.template
    add(a, b)->(co.untyped) ={
        $=> a + b;
    }
```

Templates are text replacing mechanism  where typed ones are inline function calls



#### Indexers

```folang
    @co.dap.indexer(symbol="[]")
    (g MyList) get(index co.int)->(co.int) ={
        $=> g.eles[index];
    }

    @co.dap.indexer(symbol="[]=")
    (g MyList) set(index co.int, value co.int)->() ={
        g.eles[index] = value;
    }
lst MyList;
co.out.println(lst[0]);
lst[1] = 22;
```

Indexers provides array like access to User defined data type.

#### Execution Model

Foλang executes ordinary code **sequentially by default**. A normal function or
method declaration therefore requires no execution-model decorator merely to be
called sequentially.

The built-in decorator `@co.dap.executionmodel(...)` is used only when a
declaration requires non-default execution semantics that must remain observable
across conforming FoLang implementations. The language exposes an execution-model choice only when that choice changes required FoLang behaviour. A distinction that changes only a backend's internal implementation strategy is not a separate FoLang execution model.


```folang
   @co.dap.executionmodel(type=concurrent, kind=task)
    someConcurrent(a co.int)->(co.int,co.error) = {
        ...
    }

co.cpca.submit(someConcurrent, params=SubmitParams{10}, results=SubmitResults{val, errors});
(errors.isEmpty).then(
    co.out.println(val)
).default(
    co.out.println(errors)
);
```
for scheduling

```folang
@co.dap.executionmodel(
    type=concurrent,
    kind=task,
    scheduling=cooperative
)
```

### Execution-Model Summary

| Type | Source-level kind/model | Optional semantic dimension | Package responsible for operations |
|---|---|---|---|
| default sequential | — | — | ordinary function/method call |
| `concurrent` | `task`, `thread` | `scheduling=cooperative|preemptive`; structural policies such as `fork-join` when separately defined | `co.cpca` |
| `parallel` | — | parallel-specific policy only when separately specified | `co.cpca` |
| `async` | — | `completion=future|callback` | `co.cpca` |
| `continuation` | `full`, `delimited` | delimited `control="shift-reset"|"prompt-control"|"spawn-yield"` | `co.control` |



#### Native methods

```folang
@co.dap.native
nativeMethod(a co.int, b co.int)->(co.int) ={
    // native implementation
}

```
Folang provides various constructs which must be used with `native` annotations
like functions containing `co.native` refer [Built in Packages](#built-in-packages)

These native methods should be in component kind `native` or library of type `native`.

#### Associated Functions

```folang
   @co.dap.operator(symbol="+")
    (emp Employee) add(other Employee)->(Employee) = {
        ...
    }

    @co.dap.operator(symbol="==")
    equals(
        left  Employee,
        right Employee
    )->(co.bool) = {
        ...
    }

    @co.dap.operator(symbol=">")
    (Employee) greater(
        left  Employee,
        right Employee
    )->(co.bool) = {
        ...
    }

```
Associated function must be declared in companion units named after struct type .comp.unit

***

### Comprehensions
A FoLang comprehension is a source-driven transformation. Its canonical form is:

```folang
result := (pattern <- source).yield(resultExpression);
```
#### Permitted Comprehension Sources

FoLang comprehensions intentionally accept only the following source categories:

1. **Any FoLang iterable**, including standard iterable forms such as arrays, lists, sets, maps/dictionaries, and ranges.
2. **`Some(T)`**.
3. **`Future(T)`**.

For example:

```folang
(x <- IntList{1,2,3}).yield(x * 2); // valid: iterable
(x <- 1 .. 10).yield(x * 2);       // valid: iterable range
((k, v) <- valuesMap).yield(k, v);  // valid: iterable map/dictionary
(x <- Some(5)).yield(x * 2);        // valid: permitted non-iterable source
(x <- someFuture).yield(f(x));      // valid: permitted non-iterable source
```

***

### Lambda Expressions

```folang
k := (1 .. 10)
    .filter(|x| => x % 2 == 0)
    .map(|x| => x * x);
```

Only allowed as an inline callback argument to receiver-qualified collection
operations (e.g. `each`, `map`, `filter`, `reduce`, `forEach`, `sortBy`, `groupBy`).

The lambda must be a direct argument of the allowed collection call. That call
may itself be nested, for example `consume(nums.map(|x| => x*x))`; the enclosing
call does not make the lambda an argument of `consume`.

***

### Condition Loops and Ternary operations

#### Conditions

`then` is the one-shot conditional branch verb. Its argument may be a block or an ordinary value/expression. `when(condition)` introduces each additional Boolean condition and must be followed by `.then(...)`. `default(result)` is the optional terminal fallback and may likewise receive a block or value. The same `then` / `when(condition)` / `default` chain is used for statement-style conditional execution and value-producing ternary selection.

```text
(boolean truth).then({
}).when(boolean truth).then({
}).default({
});
```
```folang
        x co.bool =co.const.true;
        (x).then({   // parentheses optional

        }).default({

        });
```


#### Loop

```folang
(boolean truth).loop({
});


x := co.const.true;

x.loop({

});

```

#### Ternary
A value-producing multi-branch conditional therefore has the canonical shape:

```text
s = (boolean truth).then(some var/val).when(boolean truth).then(some var/val).default(some var/val);
```
```folang
(n > 0)
    .then("positive")
    .when(n < 0)
    .then("negative")
    .default("zero");
```

| Chain head | Parentheses | Why |
|---|---|---|
| `x` | optional | an identifier is already a complete postfix expression |
| `arr[0]`, `f()` | optional | index and call suffixes are postfix too |
| `co.const.true`, `myPkg.flag` | **required** | a qualified name would absorb `.then` as a further segment, giving `co.const.true.then` |
| `x > y` | **required** | `.then` binds tighter than `>`, so `x > y.then({...})` groups as `x > (y.then({...}))` |

A qualified name needs parentheses whether it names a literal or an identifier,
so `myPkg.flag` is the same case as `co.const.true`.

***

### Pattern Matching

```folang
x co.int = 10;

x.match().case(n: n > 10 => { n = n+100; "GT" }).case( n: n < 10 => "LT").default("EQ");
x.match().case(n: n > 10 => { n = n+100; "GT" }).case( n: n < 10 => "LT").case(_=>"EQ");
x.match(co.pattern.Type).case(co.int => ...).case(co.float => ...);
x.match(co.pattern.Value).case(0 => ...).case(1 => ...);
x.match(co.pattern.Instance).case(xx.CAT => ...).case(xx.DOG => ...).default("Animal");
x.match(co.pattern.Object).case(xx.Ball => "Ball").case(xx.CAT => "CAT").default("Unknown");
x.match(co.pattern.Shape).case(Point{x, y} => ...).default(...);

x.match(co.pattern.Any).case(co.int => ...).case(co.float => ...).case(0 => ...).default( ...);

x.match(PositiveEvenMatcher).case(0 => "Neither even nor odd").case(2 => "First Even Prime").default(...);
```

Folang provides powerful pattern matching system with ability to write custom matchers

Folang has default matchers built in 
match() means value based matching there are other types of built in matchers
like 

   1. co.pattern.Type
   2. co.patter.Value -> (default)
   3. co.pattern.Instance
   4. co.pattern.Object
   5. co.pattern.Shape
   6. co.pattern.Kind
   7. co.pattern.Any

***

### Units

A unit is a stateless source container. A package may contain any number of ordinary unit files, and all their members are consolidated directly into the package namespace.

#### Companion Units

#### Normal Units

### Generics and Specializations

#### Generics

```folang
@co.dap.generic(
    at=callsite,
    types=[
        {name=U, variance=invariant, bound=Number, inference=param},
        {name=T, variance=invariant, bound=Number}
    ],
    mapping=[
        {U=co.int,   T=co.int},
        {U=co.float, T=co.float}
    ],
    resolution=compiletime
)
add(a U, b U)->(T) = { $=> a + b; }
```

**Generic annotation fields:**

| Attribute | Values |
|---|---|
| types | list of generic-parameter records such as `[{name=T}, {name=R}]` |
|aliases| declaration-local derived type aliases such as `[{name=Result, type=F(B)}]`; alias expressions may reference markers declared by `types=` |
|requires| |
|mapping| compile-time relationships among declared generic parameters; valid only for compile-time generic resolution |
|resolution| `runtime`, `compiletime`|
|reified| `true` or `false`|
|at| `usesite` or `callsite`|
|specializable| `true` or `false` |
|lifecycle| `true` or `false`; interpreted only when the generic declaration target is `co.class`; ignored for lifecycle semantics on generic structs/functions/methods |


**types attributes**

|Attribute | Values|
|---|---|
|name||
|constraints||
|upper-bound||
|lower-bound||
| bound | not needed when `upper-bound` and/or `lower-bound` is specified |
|default||
|variance| `covariant`, `invariant`, `contravariant`|
|nullable||
|inference| `param`, `arg`, `var` |
|capabilities||
| isAKind | an interface, class, or trait; e.g. `SomeInterface` | 
|typekind| `type`,`class`,`function`,`struct`,`typeconstructor`| 
|inclusive||

> The entries above are fields of each generic-parameter record inside the `types=[...]` list.

> Example: `@co.dap.generic(types=[{name=T, variance=..., bound=...}])`

> These are not independent attributes; they describe each declared generic type entry.

> `lifecycle` is different from the per-type records above: it is a field of the outer `@co.dap.generic(...)` application. Every class already has compiler-owned inherited lifecycle implementations. For a generic class, `lifecycle=true` grants the developer permission to override or overload `@@new` and `@@init`; absent or `false` forbids developer lifecycle customization. The field does not itself expose a lifecycle call. A developer-defined lifecycle implementation participates in `::new(...)` or `::init(...)` lookup according to its own accessibility. On a generic struct, function, or method, the field is accepted but not considered for lifecycle semantics.


##### Generic Marker Classification

Names listed by the immediately associated `@co.dap.generic(types=[...])` are **generic markers** for that declaration. In applicable type positions the parser/frontend records those names as generic-marker references; they are not looked up as ordinary type symbols.

A type-position identifier that is **not** listed in that declaration's `types=[...]` list is an ordinary type reference. It is resolved through the normal symbol/type environment and is a compiler error when no such type exists.

```folang
@co.dap.generic(types=[{name=U}])
f(x U, y T)->(U) = { ... }
```

Here `U` is a generic marker. `T` is not declared as a marker by the annotation, so `T` must resolve as an actual type symbol. If no type named `T` is visible, the declaration is invalid.

By contrast:

```folang
@co.dap.generic(types=[{name=U}, {name=T}])
f(x U, y T)->(U) = { ... }
```

both `U` and `T` are generic markers.

Generic-marker spelling does not manufacture a distinct generic signature. These two declarations have the same generic parameter structure and the same parameter signature, so they conflict as duplicates in one callable identity:

```folang
@co.dap.generic(types=[{name=T}])
f(x T)->(T) = { ... }

@co.dap.generic(types=[{name=U}])
f(x U)->(U) = { ... } // compiler error: duplicate generic signature
```

The compiler compares declared generic-marker roles/positions rather than treating `T` and `U` as ordinary type names. This normalization applies **only to names already classified as generic markers**. An undeclared type-position name is never converted into a generic marker merely because another declaration uses a marker with a similar role.

For example, if an actual type named `T` exists, these parameter structures are distinct:

```folang
@co.dap.generic(types=[{name=U}])
f(x U, y T)->(U) = { ... } // second parameter is actual type T

@co.dap.generic(types=[{name=U}])
f(x U, y U)->(U) = { ... } // both parameters use generic marker U
```

Return types still do not participate in ordinary overload selection; the example uses the parameter signature deliberately because that is where the structural distinction matters.

##### Generic-Context Derived Aliases

The built-in `aliases=` field of `@co.dap.generic` declares derived type names
owned by that one generic declaration. Each record has exactly two fields:
`name`, the local alias name, and `type`, the type expression derived from the
markers declared by the same annotation's `types=` list.

```folang
@co.dap.generic(
    types=[{name=F}, {name=A}, {name=B}],
    aliases=[
        {name=FunctorOf,       type=Functor(F)},
        {name=MapFunction,     type=(A)->(B)},
        {name=InputContainer,  type=F(A)},
        {name=ResultContainer, type=F(B)}
    ]
)
mapAll(
    inst FunctorOf,
    value InputContainer,
    fn MapFunction
)->(ResultContainer) = {
    $=> inst.map(value, fn);
}
```

These aliases are not independently quantified polymorphic types and do not introduce another generic
parameter list, a new type, nominal identity, or a separate specialization
mechanism. The `aliases=` field only gives another name to a type representation
within the annotated generic declaration. Each alias resolves exactly like a
`co.type` alias, using the generic parameters introduced by the same
annotation. Ordinary generic substitution applies to the alias's underlying
representation. For a substitution with `F=co.List`, `A=co.int`, and
`B=co.string`, the effective representations are:

```text
FunctorOf       = Functor(co.List)
MapFunction     = (co.int)->(co.string)
InputContainer  = co.List(co.int)
ResultContainer = co.List(co.string)
```

All names referenced by an alias expression must be either declared generic
markers or ordinary visible type names. Alias names are available only in the
associated declaration's signature and body; they are not package members and
cannot be imported or used by another declaration. An alias name must not
duplicate a generic marker, parameter, sibling alias, or another
name in the declaration's signature scope. References between aliases use the
ordinary `co.type` alias-resolution and cycle-detection rules; `aliases=`
does not introduce separate ordering, expansion, or cycle semantics.

The `type=` member in this field is a deliberate built-in-metadata exception to
the ordinary annotation-value restriction. The parser reads it with the type
expression grammar because `@co.dap.generic` is compiler-defined and already
classifies the following declaration. Custom annotations cannot define an
equivalent field, introduce aliases, or affect parsing decisions.

##### Frontend Handling of Generic Metadata

The frontend must parse and preserve the complete `@co.dap.generic` metadata application. Fields needed to establish frontend syntax, symbol identity, generic-marker classification, type resolution, callable selection, or a concrete result contract are interpreted where required. In particular, `types=` establishes generic markers, `aliases=` establishes declaration-local derived type names after those markers are known, and `mapping=` participates in the compile-time result-resolution rules defined below.

Other generic fields and attributes may be backend- or later-stage-oriented. The frontend records and serializes them in the Final AST/backend interchange and does **not** fail frontend generation merely because it has no semantic handler for such a field. A later compiler/backend stage may interpret, validate, specialize, reify, or reject those preserved values according to the applicable feature contract. Malformed metadata syntax remains a parser error.

When `mapping=` is present, the mapping relation itself must be resolved by the frontend wherever it is necessary to produce a concrete callable/result contract. A backend-oriented field such as `resolution`, `reified`, `at`, or `specializable` does not by itself block frontend artifact generation merely because the frontend does not otherwise act on it.

Type classification is never supplied by a redundant generic-metadata flag. After name and type resolution, the resolved FoLang type is the authoritative source for whether a value or callable is concrete, polymorphic, dependent, variant-based, refinement-based, or otherwise type-classified. Metadata cannot override or restate a classification already determined by the resolved type.

#### Generic Mapping, Result Resolution, and Class-Inheritance Augmentation

`mapping=` defines a finite compile-time relationship among the generic parameters declared by the same `@co.dap.generic` annotation. It is **not required merely because a generic parameter appears in the return signature**. If every generic needed by the return signature is already resolvable from callable parameter-position generic information (or from explicit generic arguments), the return signature uses those already-resolved generic values directly and `mapping=` is unnecessary. `mapping=` is required only when a generic needed by the result contract would otherwise remain unresolved and is intended to be derived from already-resolved generic inputs.

```folang
@co.dap.generic(
    types=[
        {name=U, inference=param},
        {name=T}
    ],
    mapping=[
        {U=co.int,   T=co.int},
        {U=co.float, T=co.float}
    ]
)
f(x U, y U)->(T) = {
    ...
}
```

For `f(10, 20)`, ordinary parameter analysis first establishes `U=co.int`; the mapping then establishes `T=co.int`. For arguments statically typed `co.float`, the second row establishes `T=co.float`. The return type did not select the callable and the expected destination type was not consulted.

##### When `mapping=` Is Not Required

A generic result needs no mapping when every generic variable occurring in the
return signature is already determined directly from the callable's parameter
types or explicit generic arguments.

```folang
@co.dap.generic(types=[{name=T}])
identity(x T)->(T) = {
    $=> x;
}

@co.dap.generic(types=[{name=T}])
choose(x T, fallback T)->(T) = {
    ...
}
```

For these declarations, parameter-position inference establishes all generic
values needed by the result contract before result typing begins. For example,
passing `co.int` values to `identity` or `choose` establishes
`T=co.int`, so the result type is `co.int` without a mapping row.

By contrast, this declaration is incomplete without another explicit resolution source for `T`:

```folang
@co.dap.generic(types=[{name=U}, {name=T}])
f(x U, y U)->(T) = {
    ...
}
```

A call such as `f(10, 20)` can establish `U=co.int`, but nothing in the parameter signature establishes `T`. FoLang does not infer `T` from the assignment target, expected destination type, or another return context. Therefore `T` must be supplied explicitly, resolved by an applicable `mapping=` row, or resolved by another mechanism explicitly defined by this specification.

The generic-result rule is therefore:

```text
infer generic values from parameter positions and explicit generic arguments
    -> if every generic needed by the return signature is resolved
           mapping is unnecessary
    -> otherwise
           resolve the remaining generic through mapping= or another explicit mechanism
    -> expected/destination return type never participates
```

Generic candidate preparation and resolution proceed without consulting return context:

```text
resolve canonical callable identity and candidate declarations
    -> for each generic candidate, infer generic values available from
       ordinary call-parameter positions and explicit generic arguments
    -> instantiate enough of each candidate's parameter signature to test
       ordinary static overload applicability/specificity
    -> select the unique parameter overload
    -> apply mapping rows to the selected generic's already-resolved inputs
    -> resolve any remaining mapped generic parameters
    -> complete generic instantiation
    -> type-check the concrete return/result contract
```

A `mapping=` row is therefore never used to make a generic function win ordinary overload selection. It resolves mapped generic parameters only after the callable has been selected from parameter information.

A generic parameter that occurs only in the return signature is **never inferred from the expected destination or return context**. It must be resolved by an explicit generic argument, by an applicable `mapping=` row, or by another generic-resolution mechanism explicitly defined by this specification. Result-context generic inference is not part of FoLang.

When `mapping=` is present, its rows define the permitted mapped relationships for the generic parameters they mention. Mapping matches use canonical type identity; overload-style subtype widening is not performed while choosing a mapping row. Once the determining generic values are known, the mapping must yield one consistent assignment for every unresolved mapped generic. No applicable row, multiple conflicting applicable rows, a cyclic unresolved dependency, or a mapping that leaves a required generic unresolved is a compile-time error.

Mapping does not create sibling function overloads. The declaration above remains one generic callable with one body and one declared return-signature structure `->(T)` even though different valid instantiations may make `T` concrete as different types. This is generic instantiation, not return-type overloading.

##### Generic Mapping Augmentation Through Class Inheritance

Generic mapping augmentation is intentionally restricted to **class inheritance**. Package-level/free functions do not need cross-package augmentation because package ownership is part of callable identity: two same-named functions in different packages are different callables rather than one overload family. An application therefore cannot reopen another package merely to add `mapping=` rows to a package function.

When a class inherits a visible generic method, additional mapping entries may be associated with that **already inherited generic method** in the derived-class context. The thing being augmented is the inherited method's effective **mapping set**. The augmentation does not declare another callable, does not provide another body, and is not a bodyless/forward generic method declaration.

A bodyless generic method remains an ordinary forward declaration and **cannot contain `mapping=`**. Mapping metadata intended as an inheritance augmentation must resolve to exactly one inherited generic method and must match that method's callable/receiver category, declared generic-marker structure, ordinary parameter-signature structure, and declared return-signature structure. If no such inherited generic exists, the mapping contribution is a compiler error.

This can apply to classes defined by the application itself or classes from package contexts explicitly exported by a packaged component or standalone packaged library and therefore present in the executable application's open graph. It does not penetrate projected `application`, `native`, or `dynamicvmrt` boundaries because their internal classes and methods remain hidden behind surface APIs.

Conceptually:

```text
BaseProcessor.convert
    generic markers: U, T
    implementation body: owned by BaseProcessor
    mapping set:
        {U=co.int, T=co.int}

DerivedProcessor inherits BaseProcessor.convert
    augmentation mapping contribution:
        {U=abc.Employee, T=abc.SuperEmployee}

DerivedProcessor effective inherited convert mapping set:
        {U=co.int,   T=co.int}
        {U=abc.Employee, T=abc.SuperEmployee}
```

The compiler merges inherited and derived-context mapping entries as a set:

```text
identical row + identical row
    -> one logical row

same already-resolved input assignment -> same derived assignment
    -> duplicate; one logical row

same already-resolved input assignment -> different derived assignment
    -> conflict; compiler error

different input assignment
    -> additional valid mapping row
```

The augmentation affects the effective inherited generic in that derived-class context; it does not rewrite the base declaration globally and does not alter sibling derived classes. Generic mapping remains a compile-time frontend mechanism where it is required to establish the concrete callable/result contract, and it is not re-evaluated from runtime argument types by `@co.ddap.dynamicdispatch(true)`.

#### Generic Functions — Parameters and Return Values

##### Rank-1: Outer function is generic; parameter uses the same type variable

`T` is fixed at the call site before the function parameter is used. The passed function is already monomorphic inside the body.

**Invalid inline signature**

The inline form is a compiler error because `(T, T)->(T)` is a derived type
used directly in a parameter. It must be named by the generic declaration.

**Required named type alias**

//somGen2.unit.fol
```folang
_ co.unit = {
    @co.dap.generic(
        types=[{name=T}],
        aliases=[{name=SomeFArg, type=(T, T)->(T)}]
    )
    someFunction(f SomeFArg, a T, b T)->(T) = {}
}
```

`SomeFArg` belongs to the generic declaration context established by
`@co.dap.generic`; it is not a free-standing parameterized alias.

***

##### Rank-2: The function parameter is itself polymorphic (higher-rank)

The passed function stays generic **inside the callee**. The binder list of `co.polymorphic(...)` belongs to a named `co.type`; the consuming function does not own those binders and is not made generic merely by accepting the named polymorphic type.

**Named polymorphic type**
//someGen4.unit.fol
```folang
_ co.unit = {
    SomeFArg co.type = co.polymorphic({T}, (T, T)->(T));

    // T belongs to SomeFArg; this consumer is not itself generic.
    someFunction(f SomeFArg)->(co.int) = {}
}
```
***

##### Returning Generic Functions

**Rank-1 return**
//someGen6.unit.fol
```folang
_ co.unit = {
    @co.dap.generic(
        types=[{name=T}],
        aliases=[{name=Adder, type=(T)->(T)}]
    )
    makeAdder(a T)->(Adder) = {
        adder Adder = (b T)->(T) {
            $=> a + b;
        };
        $=> adder;
    }
}
```

**Rank-2 return — returning a polymorphic function**
//somGen7.unit.fol
```folang
_ co.unit = {
    PolyIdentity co.type = co.polymorphic({T}, (T)->(T));

    identity PolyIdentity(x) = x;

    makeIdentity()->(PolyIdentity) = {
        $=> identity;
    }
}
```

`PolyIdentity` owns the polymorphic binder and callable shape. `identity` implements that named callable type without a separate `@co.dap.generic` declaration. `makeIdentity` merely returns the already-polymorphic named callable. FoLang has no general anonymous-function literal that can introduce an independent polymorphic binder set.

***

##### Rank-3: A Parameter is Itself a Rank-2 Function

Rank-3 uses named `co.type` layers. Each `co.polymorphic(...)` binder list remains inside the type declaration that owns it; consuming and returning functions use the resulting named types.

**Named type layers**
//someGen9.unit.fol
```folang
_ co.unit = {
    Rank2FnType  co.type = co.polymorphic({T}, (T, T)->(T));
    Rank3ArgType co.type = (Rank2FnType)->(co.int);

    applyRank2(f Rank3ArgType, value Rank2FnType)->(co.int) = {
        $=> f(value);
    }
}
```

**Rank-3 return**
//somGen10.unit.fol
```folang
_ co.unit = {
    Rank2FnType co.type = co.polymorphic({T}, (T)->(T));
    Rank3ConsumerType co.type = (Rank2FnType)->(co.int);

    consumeRank2(f Rank2FnType)->(co.int) = {
        $=> f(42);
    }

    makeRank2Consumer()->(Rank3ConsumerType) = {
        $=> consumeRank2;
    }
}
```

***

##### Passing a Polymorphic Type Object to a Generic Function

FoLang makes no value/object distinction that excludes types: every type is an
object. A marker introduced by `@co.dap.generic(types=[{name=T}])` may therefore
resolve to `co.type`, and the corresponding parameter value may be a type
object such as `co.int`, `Employee`, or `PolyId`.

In this subsection, `Box(T)` denotes a parameterized `co.type` constructor;
`Box` is not an annotation-declared generic struct or class. The polymorphic
shape is first named by the required `co.type` declaration, and that named
type object is passed through the ordinary call syntax:

//somGen11.unit.fol
```folang
_ co.unit = {
    Box(T) co.type = co.variants(Boxed(T));
    PolyId co.type = co.polymorphic({U}, (U)->(U));

    @co.dap.generic(
        types=[{name=T}],
        aliases=[{name=BoxOfT, type=Box(T)}]
    )
    box(x T)->(BoxOfT) = {}

    someFun()->() = {
        result := box(PolyId);
        // T resolves to co.type; x contains the PolyId type object.
    }
}
```

The call above is ordinary generic inference over a type object. It does not bind `T` to the polymorphic callable type represented by `PolyId`; the argument value is the type object `PolyId`, so `T` resolves to `co.type`. No special generic metadata is involved.

##### Generic Binding to a Polymorphic Callable Type

When the argument is instead a callable value whose resolved type is a named polymorphic callable type, generic inference binds the generic marker directly to that resolved type. No opt-in flag is required because the callable's type already carries the complete classification.

```folang
_ co.unit = {
    Box(T) co.type = co.variants(Boxed(T));
    PolyId co.type = co.polymorphic({U}, (U)->(U));

    identity PolyId(value) = value;

    @co.dap.generic(
        types=[{name=T}],
        aliases=[{name=BoxOfT, type=Box(T)}]
    )
    box(x T)->(BoxOfT) = {}

    someFun()->() = {
        result := box(identity); // T = PolyId
    }
}
```

Here the frontend resolves `identity` as a named function whose callable type is `PolyId`. Because `PolyId` resolves to `co.polymorphic({U}, (U)->(U))`, `T` is bound to `PolyId` directly. FoLang does not require or permit a second metadata switch to repeat that fact. In type-theory terminology this permits a generic variable to range over a polymorphic type, but in FoLang the decision is entirely type-driven.

The named-type rule still applies. A complex polymorphic type expression must be named before it participates in ordinary type use or generic substitution. The inline spelling below therefore remains invalid:

```folang
result := box(co.polymorphic({U}, (U)->(U))); // compiler error: use PolyId
```

***

##### Generic Function Rank Support Matrix

| Scenario | Allow? | Notes |
|---|---|---|
| Rank-1 generic parameter | ✅ Yes | Declare markers and derived aliases in `@co.dap.generic` |
| Rank-1 generic return | ✅ Yes | Parameters and results use named aliases when their type is derived |
| Rank-2 param via named `co.type` | ✅ Yes | `co.polymorphic(...)` owns the binders inside the named type declaration; the consumer uses that type |
| Rank-2 param via a `co.function` value declaration | ❌ Compiler error | Function objects are concrete values; define a named `co.type` with `co.polymorphic(...)` instead |
| Rank-2 return via named `co.type` | ✅ Yes | Return a named callable that implements the polymorphic callable type |
| Implement a named polymorphic callable type | ✅ Yes | `identity PolyId(value) = value;` inherits parameter/result types and polymorphic binders from `PolyId` |
| Rank-3 via named `co.type` layers | ✅ Yes | Higher-rank structure is expressed by composing named types |
| Rank-3 return | ✅ Yes | Return a named callable matching the named Rank-3 type |
| Rank-3 via a `co.function` value declaration | ❌ Compiler error | Same rule as Rank-2; function objects are concrete |
| Pass a named polymorphic type object to a generic parameter | ✅ Yes | `T` resolves to `co.type`; the parameter value is the named type object |
| Pass a value whose type is a named polymorphic type | ✅ Yes | `T` resolves directly to that named polymorphic type from the value's resolved callable type; no classification flag is required |
| Inline `co.polymorphic(...)` call argument | ❌ Compiler error | Name the polymorphic type with `co.type` and pass that name |

`@co.dap.generic(types=[...])` remains the generic-marker mechanism for declarations that define their own generic signature. `co.polymorphic(...)` separately introduces binders owned by the named `co.type` value it constructs. An ordinary named function that implements such a callable type inherits that type's binder and callable contract and therefore does not redeclare the same binders with `@co.dap.generic`. Functions express higher-rank parameters and returns by using these named polymorphic callable types. See [Polymorphic Types](#polymorphic-types), [Named Callable-Type Implementation](#named-callable-type-implementation), and [Generic Declarations and Parameterized Types](#generic-declarations-and-parameterized-types).

#### Generics Inheritances and Types

```
This is in conceptual stage not supported.

A) Abstract vs concrete type members
B) Path-dependent types
    1. Type-level projection
    2. Path-dependent In folang how it would be
```

#### Polymorphic Types

`co.polymorphic(...)` is the built-in RHS type-expression constructor for a named polymorphic type. It is not a keyword and does not introduce a separate declaration category. The enclosing declaration remains an ordinary `co.type` declaration.

Canonical form:

```folang
PolyId co.type = co.polymorphic({T}, (T)->(T));
```

The first argument, `{...}`, introduces the polymorphic type binders owned by that one type expression. The second argument is the polymorphic type body and may reference those binders. Binder scope begins with the binder set of that constructor and ends with the enclosing `co.polymorphic(...)` expression. The binders do not become declarations in the surrounding unit, package, function, class, or other lexical scope.

The resolved type declaration is authoritative for polymorphic classification. A symbol whose resolved callable type is a named `co.polymorphic(...)` type is polymorphic by virtue of that type; a symbol whose resolved callable type is concrete is concrete. FoLang does not use a separate source flag to assert, enable, disable, or override this classification. For example, after `identity PolyId(value) = value;` resolves, the compiler knows that `identity` is polymorphic because its declared callable type is `PolyId`.

`co.polymorphic(...)` is valid only as the complete RHS type expression of a `co.type` declaration. A polymorphic type must therefore be named before it is used in a field, variable, parameter, receiver, function result, annotation value, or ordinary call argument.

```folang
SomeFArg co.type = co.polymorphic({T}, (T, T)->(T));
PolyId   co.type = co.polymorphic({U}, (U)->(U));

// Rank-2 parameter: the consumer is not itself generic.
someFunction(f SomeFArg)->(co.int) = {}

// The named callable type supplies the polymorphic contract.
identity PolyId(value) = value;

// Rank-2 return: returns an already-polymorphic named callable.
makeIdentity()->(PolyId) = { $=> identity; }
```

A declaration that defines its own generic signature continues to use `@co.dap.generic`. By contrast, an ordinary named function implementing a named `co.polymorphic(...)` callable type inherits that callable contract and does not redeclare its binders with `@co.dap.generic`. Restricted lambdas do not introduce a `co.polymorphic(...)` binder set of their own.

Inline polymorphic type construction is invalid in ordinary type-use or value-expression positions:

```folang
consume(
    value co.polymorphic({T}, (T)->(T))
)->(); // compiler error: name the polymorphic type first

result := box(
    co.polymorphic({U}, (U)->(U))
); // compiler error: pass the named type object instead
```

The valid type-object form is:

```folang
PolyId co.type = co.polymorphic({U}, (U)->(U));
result := box(PolyId);
```

A named function may implement the same callable type directly:

```folang
identity PolyId(value) = value;
```

##### Quick Reference

| Form | Status | Context |
|---|---|---|
| `name co.type = co.polymorphic(...);` | ✅ Allowed | Named polymorphic type definition |
| `identity PolyId(value) = value;` | ✅ Allowed | Named function implements the callable contract and binders owned by `PolyId` |
| polymorphic binder syntax in an ordinary declaration head | ❌ Compiler error | Define the binder set in a named `co.polymorphic(...)` type or use `@co.dap.generic` for a declaration that owns its own generic signature |
| inline `co.polymorphic(...)` in an ordinary parameter/result type | ❌ Compiler error | Declare a named polymorphic `co.type` and use that name |
| inline `co.polymorphic(...)` as an ordinary call argument | ❌ Compiler error | Pass the named polymorphic type object instead |

**The rule in one sentence:** `co.polymorphic(...)` owns the polymorphic callable contract; an ordinary named function may implement that named contract directly without redeclaring its binders.


> Generic declarations that own their own generic parameter set are supported only for structs, classes, ordinary functions, and ordinary methods, and introduce those parameters through `@co.dap.generic`. An ordinary function implementing a named polymorphic callable type is different: the binder set is owned by the named `co.type`, so the implementation does not redeclare it.
>
> `OperatorOverloadDecl` is deliberately excluded even though an operator implementation has a callable shape. A declaration carrying `@co.dap.operator` must not also carry `@co.dap.generic`. A generic class or struct may own an operator, but the operator itself remains non-generic and is associated with the canonical owner declaration rather than with operator-level type parameters.

The following declaration-head generic forms are invalid:

```folang
// Cache.fol
_(T) co.module = {}             // compiler error
// operations.unit.fol
_(F(_)) co.unit = {}            // compiler error
Callback(T) co.delegate = (T)->(T); // compiler error
```

A parameterized `co.type` declaration is a separate parameterized-type form and does not use `@co.dap.generic`:

```folang
// option.unit.fol
_ co.unit = {
    Option(T) co.type =
        co.variants(Some(T), None);
}
```

Generic structs and classes remain file-backed primary declarations. Their names come from filenames:

```folang
// LinkedList.fol
@co.dap.generic(types=[{name=T}])
_ co.struct = {
    value T;
    next  LinkedList;
    prev  LinkedList;
}
```

```folang
// linked_list_values.unit.fol
_ co.unit = {
    IntLinkedList co.type = LinkedList(co.int);
    someFun()->()={
    	myIntList IntLinkedList;
    }
}
```

```folang
// Employee.fol
@co.dap.generic(types=[{name=T}, {name=R}])
_ co.class = {
    id   T;
    name R;
}
```

```folang
// employee_types.unit.fol
_ co.unit = {
    EmployeeIntString co.type =
        Employee(co.int, co.string);
    emp EmployeeIntString;
}
```

Generic functions use the same annotation but are declared inside a legal function-owning context such as an ordinary unit, class, or companion unit:
//sommGen1.unit.fol
```folang
_ co.unit = {

    @co.dap.generic(types=[{name=T}, {name=R}])
    add(a T, b T)->(R) = {
        ...
    }
    someFun()->()={
        add_int_int := add.withTypes(co.int,co.int);

        //    or

        add_int_int co.function =  add.withTypes(co.int,co.int);

        k := add_int_int(12,10);
    }
}
```

***

#### Specialization

`@co.dap.specialize` to specialize generics for specific types upfront
//sommGen2.unit.fol
```folang
_ co.unit = {

    @co.dap.generic(
        types=[
            {name=T}
        ],
        requires=[
            co.Add(left=T, right=T, result=T)
        ]
    )
    add(a T, b T)->(T) = {
        $=> a + b;
    }
}
```

for the above generic want to specialize for `co.int`
//sommGen5.unit.fol
```folang
_ co.unit = {
    @co.dap.specialize(
        target=add,
        types=[
            {name=T, type=co.int}
        ]
    )
    addInt(a co.int, b co.int)->(co.int) = {
        $=> co.intrinsic.intAdd(a, b);
    }
}

```
`folang` provides partial specialization below is the example for partial specializationn
//sommGen7.unit.fol
```folang
_ co.unit = {
    @co.dap.generic(
        types=[
            {name=T},
            {name=R}
        ]
    )
    transform(value T)->(R) = {
        ...
    }
}
```
//sommGen8.unit.fol
```folang
_ co.unit = {
    @co.dap.specialize(
        target=transform,
        types=[
            {name=T, type=co.string},
            {name=R}
        ]
    )
    transformString(value co.string)->(R) = {
        ...
    }
}
```


**fields of specialize**

|Attribute|Values|
|---|---|
| target| the generic fully qualified name includes package name if omitted it is current package |
| types| resoultion types|
| priority||
| strategy|intrinsic|

***

### Effects and Handling Effects

FoLang handles recoverable execution failures as typed **effects** without
`try`, `catch`, `throw`, or `finally` statements. Effect declaration and effect
handling have deliberately separate owners:

| Construct | Owner and placement | Purpose |
|---|---|---|
| `@co.dap.effects` | Callable declaration or definition | Declare effects the callable may emit |
| `@co.dap.onEffect` | Immediately before a call expression | Handle effects from that invocation |
| `@co.dap.defer` | Inside a callable | Register unconditional completion work |

The defining library knows which effects an exported callable may emit, but it
does not know the caller's logging, retry, cleanup, resource, or return policy.
The definer therefore publishes only `@co.dap.effects`; the caller chooses
`@co.dap.onEffect` handlers and resolutions independently at each ordinary
call site. An execution-model invocation is the deliberate exception: the
caller may register handlers, but its resolution is fixed implicitly to
`return_with_error` because effects cannot propagate across that execution
boundary.

A non-handler library implementation may still use `@co.dap.onEffect` for
calls made inside its own implementation. At those sites the library is itself
the caller. It may not attach `@co.dap.onEffect` to an exported callable
definition to impose handling on external callers. Effect-handler modules are
the explicit exception: `@co.dap.onEffect` is forbidden throughout them.

#### Effect Channel and Error Result Channel

An emitted effect and a returned `co.error` value use different channels:

| Channel | Meaning |
|---|---|
| Effect channel | Non-local typed failure that must be handled, converted, or propagated |
| Ordinary result channel | A normal returned value, including a `co.error` value |

Returning a non-empty `co.error` does not emit it as an effect.
`return_with_error` explicitly consumes an effect at a call site, converts it
to a `co.error` object, and returns it from the enclosing caller through
the ordinary result channel.

An error object may carry its category, origin, message, source location,
cause, and backend/runtime diagnostics. Native or backend exceptions must be
translated into typed FoLang recoverable effects before FoLang resolution
begins; the translated type need not have been advertised by
`@co.dap.effects`.

#### Recoverable Error Contract, Mixin, and Effect Emission

Every recoverable effect value is an instance of a class satisfying the
standard `co.error` interface. Error classes may add category-specific
fields and helper methods. The standard `co.AbstractError` mixin supplies
common state and concrete behavior for that interface. Custom error classes
normally compose the mixin, declare `co.error` as an interface, and add
their own information:

```folang
@co.dap.oops(
    interfaces=[co.error],
    mixins=[co.AbstractError]
)
_ co.class = {
    // category-specific error fields and methods
}
```

`co.AbstractError` is declared as `_ co.mixin`, not as a class and
not as a distinct abstract-class kind. As a mixin it participates only in
composition and has no independent class-instance construction semantics. Its
exact matching concrete methods may satisfy `co.error` interface slots
when the consuming error class composes the mixin. If an explicit source
mapping or remapping is required, the consuming class uses
`@co.dap.implement(type=co.error)` according to the ordinary method-
resolution rules. Interface method declarations themselves do not carry that
annotation.

`@co.dap.implement` and `@co.dap.implementation` remain distinct:

| Form | Purpose |
|---|---|
| `@co.dap.implement` | Satisfy an interface, trait, or mixin method contract |
| `@co.dap.implementation` | Bind an authorized bodyless standard declaration to a backend-neutral runtime operation |

FoLang source and HIR use one backend-neutral effect-emission operation. Its
standard declaration is conceptually:

```folang
@co.dap.implementation(
    kind=co.dap.implementationKind.runtime,
    operation=co.runtime.operation.effect.emit
)
emit(error co.error)->();
```

The operation accepts only a `co.error` value. Each backend may implement
it using native exceptions, tagged control results, runtime unwinding, task
completion, or another mechanism, but it must preserve FoLang matching,
handler, resolution, and defer semantics. A backend or native-library failure
that is recoverable in FoLang must first be translated into a class instance
satisfying `co.error`; a raw backend exception must not enter FoLang
handling directly.

Fatal runtime failures are outside this hierarchy. They do not satisfy
`co.error`, cannot appear in `@co.dap.effects` or `@co.dap.onEffect`, and
cannot be continued, retried, returned as an error, propagated as a FoLang
effect, or delivered to an effect handler. They terminate the affected
application or runtime execution according to runtime policy.

### `@co.dap.effects`: Definer Contract

`@co.dap.effects` is informative public symbol metadata, not a closed or
enforced catch-or-declare contract. Its `emits=[...]` list advertises effect
types currently known to possibly leave the callable:

```folang
@co.dap.effects(
    emits=[
        co.DatabaseError,
        co.NetworkError
    ]
)
loadCustomer(id co.int)->(Customer) = {
    ...
}
```

It is valid on ordinary callable definitions, bodyless declarations, interface
methods, native declarations, and other ordinary callable contracts. It is
invalid on an execution-model declaration because that declaration converts
all recoverable effects at its public boundary. It is also invalid on a call,
expression, statement, field, or non-callable declaration.

Every `emits` entry must resolve to an accessible class type satisfying
`co.error`. Duplicate canonical types are invalid, and a broader declared
error class covers its subtypes. The list is an open set of known possible
effects, not an exhaustive closed effect row and not a claim that every listed
effect occurs on every invocation.

Absence of `@co.dap.effects`, or an empty `emits` list, means that the source
explicitly advertises no effects. It does not mean that the callable is
effect-free. An undeclared effect may still arise and propagate like an
unchecked exception.

The compiler computes the callable's known outgoing-effect metadata from its
explicitly declared effects plus statically known unhandled callee effects,
minus effects consumed at call sites. When a statically selected concrete
handler has a known outgoing effect, the compiler adds the possibility of
`co.GenericError`, because that is the value exposed beyond the failed
handler pipeline. It serializes this known set in exported `.folenc` symbols.
Unknown runtime effects remain possible even when absent from that metadata.

Because the contract is open, an implementation or override may advertise
additional effects not named by its interface or overridden declaration.
Effect metadata informs callers and tooling but does not distinguish overloads
or make undeclared propagation invalid.

#### `co.EffectHandler`

The standard library defines the handler-module contract:

```folang
// co.EffectHandler
_ co.signature = {
    handle(error co.error)->();
}
```

Every `handlers=[...]` entry must be an accessible singleton module matching
this signature. It is not a constructed class instance or arbitrary function
reference. The signature limits only the module's public contract; a handler
may use private functions, associated singleton objects, configuration,
registries, locks, thread-local state, and other accessible facilities.

`co.EffectHandler` constrains only the structural member signature. The
signature itself does not declare handler effects. Each concrete handler
implementation may independently advertise effects on its actual `handle`
implementation:

```folang
@co.dap.effects(
    emits=[co.LogWriteError]
)
handle(error co.error)->() = {
    writeLog(error);
}
```

Omitting `@co.dap.effects` from that implementation does not prohibit an
unexpected effect. It means only that the implementation advertises no effect
explicitly. Since `handlers=[...]` contains statically resolved concrete module
references, the compiler can use their declared and inferred known effects to
include the possibility of `co.GenericError` when analysing the caller.

An effect-handler module must not use `@co.dap.onEffect`, either on its public
`handle` implementation or at a call site inside one of its private functions.
Handlers are the terminal declarative stage of the current handling pipeline;
allowing them to start another local handler pipeline would permit recursive
handling chains with no clear semantic boundary. A handler should therefore
use ordinary values, variants, explicit checks, or non-effectful operations for
problems it can recover from locally. If it nevertheless emits a recoverable
effect, the handler-failure rules below apply. `@co.dap.effects` remains valid
on the concrete `handle` implementation because it advertises such possible
outgoing handler effects without handling them locally.

#### `@co.dap.onEffect`: Caller Policy

`@co.dap.onEffect` prefixes exactly one call expression. It defines how that
specific invocation handles matching effects from its callee:

```folang
showCustomer(id co.int)->(co.error) = {
    customer Customer =
        @co.dap.onEffect(
            co.DatabaseError={
                handlers=[
                    DBConnectionCloseHandler,
                    LogErrorHandler
                ],
                resolution=return_with_error
            },
            co.NetworkError={
                resolution=propagate
            }
        )
        loadCustomer(id);
}
```

It is invalid on a callable definition or declaration, and it does not create
a statement-oriented `try`/`catch` region. The prefix belongs to the immediately
following call AST node. For nested calls, it handles only that decorated call:

```folang
outer(
    @co.dap.onEffect(
        co.NetworkError={resolution=propagate}
    )
    inner()
);
```

The policy above applies to `inner()`, not to `outer(...)`. A separately placed
`@co.dap.onEffect` is required to handle effects emitted by `outer`.

Each top-level field name must resolve to a valid effect type. It need not
appear in the callee's advertised `emits` list because that list is open and an
unexpected effect may still occur. A call-site entry for a non-advertised type
is valid proactive handling, although tooling may identify it as not currently
advertised by the callee. Keys normalize to canonical type identity; duplicates
are invalid. When multiple entries match, the most specific type wins.

For an ordinary invocation, each effect record contains exactly one singular
`resolution=` field. A record may additionally contain an optional non-empty
ordered `handlers=[...]` list. When handlers are present, all of them execute
before the resolution. The absence of `handlers` does not imply propagation: a
matching ordinary-call record without handlers still applies its declared
resolution.

For an invocation of a callable declared with `@co.dap.executionmodel`, an
effect record instead contains a required non-empty `handlers=[...]` list and
must omit `resolution`. Its resolution is fixed by the language to
`return_with_error`. Supplying any explicit resolution, including
`return_with_error`, is a compile-time error. The call-site annotation remains
useful because it registers application handlers even though the developer
cannot change the execution-boundary resolution.

For an ordinary invocation, an effect with no matching `@co.dap.onEffect`
entry propagates from the enclosing caller by default. A statically known
unmatched effect is included in the caller's computed outgoing-effect metadata,
and an unknown runtime effect propagates even when it appears in no
`@co.dap.effects` list. For an execution-model invocation, an unmatched effect
cannot propagate across the execution boundary; it is converted to the
callable's ordinary `co.error`-compatible result without running
call-site handlers.

There is no `effects={...}` wrapper, no `resolutions=[...]` list, and no
`invoke` resolution.

#### Execution-Model Call-Site Policy

An execution-model declaration must expose exactly one result position
compatible with `co.error`. The first unhandled recoverable effect that
reaches its execution boundary is retained as the primary error object and is
returned through that ordinary result position. Other ordinary result
positions that the failed execution did not produce receive `co.const.none`
only when those positions admit none. A refinement/dependent result position
makes this failure-conversion shape invalid and must be redesigned with an
explicit initialized alternative, commonly a variant.

The caller may attach ordered handlers to a particular submitted invocation.
For execution governed by `co.cpca`, the prefix is placed on the canonical
submission call and its policy is bound to the statically selected
execution-model target:

```folang
// appl.fol
SubmitParams  co.type = co.List(co.any);
SubmitResults co.type = co.List(co.any);

@co.dap.onEffect(
    co.DatabaseError={
        handlers=[
            DBConnectionCloseHandler,
            LogErrorHandler
        ]
    }
)
co.cpca.submit(
    loadCustomerAsync,
    params=SubmitParams{id},
    results=SubmitResults{customer, error}
);
```

The omitted resolution is not default propagation. In this context it denotes
the mandatory implicit `return_with_error` policy. These records are invalid:

```folang
co.DatabaseError={resolution=continue}
co.DatabaseError={resolution=propagate}
co.DatabaseError={resolution=return_with_error}
co.DatabaseError={} // no handler and no caller policy to add
```

The handlers execute synchronously and sequentially in the execution context
that receives the effect, before completion is published to the submitting
caller. The annotation registers statically resolved singleton modules with
the invocation; it does not require the submitting caller's stack to remain
active. Unmatched effects receive the same implicit conversion without
handlers.

For concurrent or parallel execution, “first error” means the first effect
observed by the applicable execution-model runtime. Unless a particular model
defines deterministic observation priority, scheduling may affect which of
simultaneously occurring effects becomes primary.

#### Resolution Records

The available resolutions are:

- `continue`
- `retry`
- `return`
- `return_with_error`
- `propagate`

Examples:

```folang
co.ValidationError={
    resolution=continue
}

co.NetworkError={
    handlers=[LogRetryHandler],
    resolution=retry,
    retry={
        max_attempts=3,
        on_exhausted=propagate
    }
}

co.DatabaseError={
    handlers=[DBConnectionCloseHandler, LogErrorHandler],
    resolution=return_with_error
}
```

Invalid records include:

```folang
co.DatabaseError={} // missing resolution

co.DatabaseError={
    resolution=[continue, return] // resolution is singular
}

co.DatabaseError={
    resolution=continue,
    resolution=return // duplicate resolution
}

co.DatabaseError={
    handlers=[],
    resolution=propagate // an explicitly supplied handlers list is empty
}

co.DatabaseError={
    resolution=retry // missing retry configuration
}
```

#### Ordered Handler Execution

Handlers execute synchronously and sequentially in declared order. Every
handler receives the same original `co.error`, and each `handle` call must
complete before the next begins. The implementation must not implicitly
reorder, parallelize, schedule, or detach the list.

For an ordinary invocation, handlers run on the caller thread and execution
context receiving the effect. For an execution-model invocation, they run in
the execution context that observes the effect before completion is published
to the submitting caller. This permits handler modules to use state accessible
in the context where handling actually runs. An ordinary function, class
object, incompatible module, inaccessible module, duplicate module, or empty
list is invalid.

If a handler emits a recoverable effect, the remaining
handlers for the original effect are skipped and the original effect's
resolution is not applied. Because `@co.dap.onEffect` is forbidden throughout
an effect-handler module, the handler cannot start a nested local handler
pipeline for the new effect. A known handler effect makes
`co.GenericError` part of the caller's computed outgoing-effect metadata;
an undeclared handler effect remains valid and follows the same runtime rule
without necessarily appearing in advance metadata.

##### Handler Failure and `co.GenericError`

Let `E0` be the original effect being handled, `H` the handler currently
executing, and `E1` a recoverable effect emitted while `H` handles `E0`. The
standard library creates one `co.GenericError` class instance that
satisfies `co.error` and records:

- the original handled error `E0`;
- the handler failure `E1`;
- the canonical identity of `H`; and
- available causal, source, and runtime diagnostic context.

Creation of this wrapper terminates the current handler pipeline immediately.
The remaining handlers are skipped, the resolution selected for `E0` is
cancelled, and neither `E0` nor the new `GenericError` is submitted to the same
handler list again.

The boundary then determines what happens to the wrapper:

| Context | Result of handler failure |
|---|---|
| Ordinary call | `co.GenericError(E0, E1, H)` propagates from the enclosing caller to its caller |
| Execution-model invocation | `co.GenericError(E0, E1, H)` is returned through the execution model's ordinary error-result position; otherwise unproduced none-admitting results receive `co.const.none`, while refinement/dependent result positions make that substitution invalid |

`co.GenericError` is used only for a recoverable effect emitted by a
handler while processing another recoverable effect. It is not created for an
ordinary `propagate`, an ordinary `return_with_error`, or the normal implicit
`return_with_error` conversion at an execution-model boundary. When handlers
complete successfully, those paths preserve the original error object `E0`.
A higher ordinary caller may handle the propagated `GenericError` at its own
call site according to ordinary rules.

A fatal failure inside a handler is not `E1`, is never wrapped in
`co.GenericError`, and terminates the application immediately like every
other fatal failure.

#### Resolution Semantics

| Resolution | Consumes effect | Behavior at the decorated call |
|---|---:|---|
| `continue` | Yes | Skip the failed call result and continue after the call |
| `retry` | Conditionally | Invoke the same callee again until success or exhaustion |
| `return` | Yes | Exit the enclosing caller normally |
| `return_with_error` | Yes | Convert to `co.error` and return it from the enclosing caller |
| `propagate` | No | Emit the effect from the enclosing caller |

`continue` does not mean retry. It resumes after the decorated invocation. If
the failed call owes one or more values to its surrounding expression, every
missing result position receives the universal `co.const.none` value. An
assignment therefore completes with none in the affected destination rather
than becoming a compile-time error.

`return` exits the enclosing caller, not the failed callee. It is directly
valid for a unit-returning caller. In a non-unit caller, every ordinary result
position not otherwise produced by the resolution receives `co.const.none`
only if that position admits none. A missing refinement/dependent result makes
the resolution invalid.

`return_with_error` exits the enclosing caller. That caller's return signature
must contain exactly one position compatible with `co.error`. Additional
result positions not otherwise produced receive `co.const.none` only if they
admit none. An unproduced refinement/dependent result position, or a missing or
ambiguous error-compatible result, is a compile-time error.

`propagate` emits the effect from the enclosing caller. Explicit `propagate` is
useful when handlers must run before propagation; without a matching record,
propagation occurs automatically without handlers. A known propagated effect
is advertised through computed outgoing-effect metadata even if the caller did
not spell it in a source annotation.

#### Retry

`retry` repeats only the decorated invocation, never the complete enclosing
caller. It requires exactly one `retry={...}` record:

```folang
@co.dap.onEffect(
    co.NetworkError={
        handlers=[LogRetryHandler],
        resolution=retry,
        retry={
            max_attempts=3,
            on_exhausted=return_with_error
        }
    }
)
sendRequest();
```

`max_attempts` is a positive compile-time integer and includes the initial
invocation. `max_attempts=3` therefore permits at most three total invocations,
not one initial invocation plus three retries.

`on_exhausted` must be exactly one of `continue`, `return`,
`return_with_error`, or `propagate`; it cannot be `retry`. It is applied to the
final error after the final failed attempt. All ordinary validity rules for the
selected exhaustion action still apply.

The caller evaluates the decorated call's receiver and argument expressions
once before the first attempt and captures their resulting values. Retries use
those same values according to ordinary value/reference parameter semantics;
FoLang does not deep-copy mutable arguments. A caller that needs fresh argument
evaluation must move that computation into the retried callee.

For each failed attempt, handlers run in declared order before the next attempt
or the exhaustion action. A handler failure aborts the retry sequence. Each
callee invocation is a distinct invocation: defers registered inside the
callee execute when that attempt exits before another attempt begins. Defers
registered by the enclosing caller do not execute merely because one call
attempt failed or was retried.

Retry repeats observable callee behavior. The developer must therefore ensure
that the callable is idempotent, transactional, compensated, or otherwise safe
to invoke repeatedly.

#### Caller-Owned Database Handling

An exported database library declares effects but does not register application
handlers:

```folang
@co.dap.effects(
    emits=[co.DatabaseError]
)
executeDatabaseOperation(command DBCommand)->() = {
    ...
}
```

The application defines its own `DBConnectionCloseHandler` and
`LogErrorHandler` modules conforming to `co.EffectHandler`. It may keep a
connection pool and current connection in an associated singleton object or
another caller-owned facility. At the call site it chooses both handlers and
their order:

```folang
runDatabaseWork(command DBCommand)->(co.error) = {
    @co.dap.onEffect(
        co.DatabaseError={
            handlers=[
                DBConnectionCloseHandler,
                LogErrorHandler
            ],
            resolution=return_with_error
        }
    )
    executeDatabaseOperation(command);
}
```

When the call emits `co.DatabaseError`, FoLang closes the caller-owned
connection, logs the same original error, converts it to `co.error`, and
returns it from `runDatabaseWork`. Another caller may choose retry or direct
propagation for the same library operation without changing the library.

#### Static Effect Information and Default Propagation

At each call the compiler combines:

```text
callee declared and inferred known effects
    + matching call-site @co.dap.onEffect entries
    + co.GenericError when concrete handlers have known outgoing effects
    + enclosing caller return signature
```

Effects consumed by `continue` or `return`, converted by `return_with_error`,
or successfully eliminated by `retry` do not escape. An exhausted retry ending
in `propagate`, an explicit `propagate`, and every unmatched callee effect do
escape. Statically known escaping effects are included in the caller's computed
outgoing metadata. Undeclared effects also escape but cannot be promised to
callers in advance. When the enclosing declaration is an execution-model
declaration, this analysis describes its internal implementation only: the
public execution boundary converts the first escaping recoverable effect to
its ordinary error result, so no effect is advertised outward.

At the outermost application/runtime boundary there is no caller to receive an
unhandled effect. The runtime reports the effect and terminates the affected
execution flow according to runtime policy.

#### `@co.dap.defer`

`@co.dap.defer` registers a callable for execution when the enclosing function
or method actually exits. It performs unconditional completion work whether
the exit is successful, ordinary, or effect-driven. It is not an effect handler,
does not inspect the completion error, and does not choose among effect
resolutions.

@co.dap.defer does not implicitly capture arguments through $->args. 
Values required by the deferred callable must be supplied/passed explicitly at registration.

```folang
someErrorFun(a co.int)->() = {
    resource Resource = acquireResource();

    @co.dap.defer
    cleanup(resource Resource, originalArgument co.int)->() = {
        releaseResource(resource);
        logArgument(originalArgument);
    }(resource, a);

    performOperation(resource);
}
```

The explicit arguments are evaluated and captured when execution reaches the
defer registration. The deferred callable itself executes later. FoLang never
injects a `co.error` object or any other completion-status binding into
that callable. Values needed by deferred work must be declared as ordinary
parameters and supplied explicitly at registration.

#### Defer Registration and Ordering

A defer is registered only when execution reaches it. A defer located after a
failing operation is not registered when that failure prevents execution from
reaching the defer. Completion work intended to cover the complete function
must therefore be registered before operations that may terminate the
function.

Every registered defer:

- executes exactly once when the enclosing function actually exits;
- executes after normal completion, ordinary return, `return`,
  `return_with_error`, or `propagate`;
- does not execute merely because an effect was handled with `continue` while
  the function remains active;
- participates in last-in, first-out ordering with other registered defers;
  and
- is not guaranteed after forced process termination, unrecoverable runtime
  shutdown, or a hardware failure that prevents further execution.

The completion order is:

```text
1. execute the function body
2. resolve locally handled effects
3. determine that the function will exit
4. execute registered deferred callables in LIFO order
5. incorporate an error produced by a deferred callable
6. complete the ordinary return, return_with_error, or propagation
```

If an error already caused the function to exit, that first error remains the
primary error. A later deferred-call error does not replace it and may be
attached only as causal/diagnostic information supported by the primary error
representation. If no primary error exists and a deferred callable emits an
effect, that effect becomes the primary completion effect and propagates from
the enclosing function unless it was handled at a call site inside the deferred
callable. A statically known deferred-call effect is included in the enclosing
function's computed outgoing-effect metadata; an unknown effect still
propagates. Remaining deferred callables continue to execute.

#### Effect-Handling Summary

| Construct | Purpose |
|---|---|
| `@co.dap.effects` | Definer-owned open metadata advertising known effects an ordinary callable may emit |
| `@co.dap.onEffect` | Caller-owned policy attached to one invocation; execution-model calls permit handlers but use implicit `return_with_error` |
| `co.EffectHandler` | Standard signature requiring `handle(error co.error)->()` |
| `handlers=[...]` | Ordered list of developer-defined handler-module references |
| `co.GenericError` | Standard wrapper created only when a handler emits a recoverable effect while processing another recoverable effect |
| `continue` | Consume the effect, substitute `co.const.none` for missing results, and continue |
| `retry` | Repeat only the decorated invocation under a bounded retry policy |
| `return` | Consume the effect and terminate through the ordinary return channel |
| `return_with_error` | Consume and return the current error as an ordinary result |
| `propagate` | Forward the effect from the enclosing caller; known effects remain advertised upstream |
| `@co.dap.defer` | Execute error-independent completion work in LIFO order on every function exit |

FoLang effect handling is therefore explicit, typed, lexically scoped, and
declarative. It provides local recovery, early return, propagation, callbacks,
and guaranteed completion work without statement-oriented exception syntax.

***



### Dynamic Vm Runtime

The `@co.ddap.dynamicruntime` directive enables full access to the `co.meta` package. It is valid **only for source files in a `dynamicvmrt` capability domain**: a standalone `@co.dap.library(type=dynamicvmrt)` project or the project-local `components/dynamicvmrt/` component. In every permitted source file it must obey the category-wide [Directive Placement](#directive-placement) rule and appear at file top level, never inside the `_ co.component` declaration or any nested declaration/body. Using `@co.ddap.dynamicruntime` in an executable application, packaged code, an application projected library/component, a `native` domain, or any other source context is a compiler error.

Within a valid `dynamicvmrt` domain, the directive enables dynamic class and type loading, monkey patching, runtime reflection, instrumentation, eval-based code execution, and other defined dynamic-runtime/metaprogramming capabilities through `co.meta`. These capabilities remain inside that projected dynamic-runtime boundary and do not automatically escape into ordinary application, packaged, or other library/component code.

   1. runtime type creation from strings, streams, files, and other supported dynamic-runtime inputs
   2. complete reflection and introspection
   3. Runtime code modification add/remove/update methods etc.

> When a library is marked `dynamicvmrt` and enables `@co.ddap.dynamicruntime`, the final binary includes the runtime support required to create and manage dynamic types, methods, and objects.

> Dynamically created objects may interact with compiled types according to the dynamic-runtime boundary rules; ordinary compiled code does not gain unrestricted reverse access to dynamic-runtime facilities.

> Through surface API's they connect to runtime and through runtime apis invoke the dynamic type and the results will be returned are compiled types only.

> Runtime can directly provide handle to compiled types to dynamic code running inside it.

> Loaders are the dynamic-runtime containers used to manage these runtime-created objects and types.

FoLang provides `BasicLoader`. A user-defined loader extends `co.meta.BaseLoader` using the ordinary loader declaration rules, for example:

//MySpecLoader.fol

```folang

@co.dap.extends(co.meta.BaseLoader)
_ co.loader={


}
```

> The basic loader provides the operations required to create, update, delete, and manage runtime-created objects.

> Loaders form a hierarchy. When a referenced runtime type is not found in the current loader realm, lookup proceeds through the base-loader chain and finally to the compiled-type environment.

***


### Native code

`@co.dap.native` marks a function or method declaration as a **native implementation declaration**. It does not grant native capability merely because the annotation is present. The annotation is valid only inside a `native` library/component domain when the installation permits the native capability.

A native library/component may use the `co.native` package to express low-level implementation such as assembly or machine-level operations through facilities including `co.native.asm` and `co.native.inline`. The same native domain also owns foreign-function interoperability: extern declarations, foreign symbols, C/native ABI-compatible types, calling conventions, symbol linkage, pointer/address forms, and permitted FoLang-side marshalling or invocation code.

These facilities intentionally share one capability boundary. A foreign call may produce an ABI value, address, or pointer that native memory, platform, assembly, or runtime code consumes directly, and native code may prepare values for a foreign call without routing them through a second projected API. `ffi` therefore names an interoperability feature area/API family where useful; it is **not** a separate FoLang capability, library, or component kind.

The frontend preserves native and foreign-interoperability declarations and their metadata through the backend interchange contract. The reference backend demonstrates one implementation of that contract. A conforming third-party backend is not required to reproduce the reference backend's internal native-code lowering, ABI lowering, instruction representation, allocation strategy, marshalling implementation, or code-generation mechanism, but the externally observable behavior required by the specification must be preserved.

#### Native Functions
// native.unit.fol
```folang
@co.dap.native
nativeMethod(a co.int, b co.int)->(co.int) ={
    // native implementation
}
```

***


#### FFI and ABI

    1. @co.dap.declare
    2. @co.dap.implementation

##### Runtime-Operation Declarations

###### Backend-neutral runtime-operation marker

The exported standard `co.*` package may declare a callable, property, constructor, type/layout declaration, or another explicitly supported standard declaration without a FoLang body when `@co.dap.implementation` supplies the implementation classification required for that declaration kind. A runtime classification identifies a compiler-owned runtime operation. The marker defines the operation's backend-independent meaning; it does not name a C++ header, C++ function, JVM member, WASM import, linker symbol, or any other backend-specific implementation.

```folang
// standard-package private source: fΦλ.out.Console
_ co.unit = {

    @co.dap.implementation(
        kind = co.dap.implementationKind.runtime,
        operation = co.runtime.operation.out.println
    )
    println(value co.string) -> ();
}
```

The declaration above contributes two linked identities:

```text
public FoLang symbol  = co.out.println
runtime operation     = co.runtime.operation.out.println
signature             = (co.string) -> co.unit
```

The currently defined standard implementation classification is:

| `kind` | Meaning |
|---|---|
| `co.dap.implementationKind.runtime` | Lower through a registered backend/runtime handler identified by `operation`. |



***

### Annotation Decorators, Pragmas and Directives

The entries in this language-defined inventory form the current built-in metadata registry used for `@co.*` name recognition. The parser must recognize a language-owned metadata name through this predefined registry before accepting the metadata application. Field/argument preservation and partial frontend field validation follow [Built-in Metadata Parsing](#built-in-metadata-parsing). Every entry classified as `DIRECTIVE` follows the category-wide [Directive Placement](#directive-placement) rule and is file-level only. Every entry classified as `PRAGMA` additionally follows the category-wide [Pragma Placement](#pragma-placement) rule and is valid only in an executable application's `src/appl.fol`.

|Kind | ||
|---|---|---|
|`PRAGMA`|"@co.pdap.threadpool","@co.pdap.schedularpool"||
|`DIRECTIVE`|"@co.ddap.import", "@co.ddap.dynamicruntime", "@co.ddap.use",  "@co.ddap.alias","@co.ddap.dynamicdispatch","@co.ddap.overload"|`@co.ddap.overload` is different from `@co.dap.overload` it has takes whether `paramtypes` or `paramandreturntypes` as attributevalue of `strategy`|
|`ANNOTATION`| "@co.dap.extend","@co.dap.template", "@co.dap.macro","@co.dap.operator", "@co.dap.annotation", "@co.dap.library", "@co.dap.native", "@co.dap.class", "@co.dap.static","@co.dap.object", "@co.dap.inline","@co.dap.ctfe", "@co.dap.friend", "@co.dap.sealed", "@co.dap.extension","@co.dap.override","@co.dap.implement", "@co.dap.virtual", "@co.dap.abstract", "@co.dap.delegate", "@co.dap.dynamicscope","@co.dap.mixedscope", "@co.dap.typeclass","@co.dap.matcher", "@co.dap.constructor", "@co.dap.oops","@co.dap.extends","@co.dap.hokrlt", "@co.dap.indexer", "@co.dap.generic", "@co.dap.comptime", "@co.dap.typefromvalue", "@co.dap.local", "@co.dap.private","@co.dap.public","@co.dap.compose", "@co.dap.guard","@co.dap.package","@co.dap.protected","@co.dap.internal","@co.dap.export","@co.dap.eager", "@co.dap.lazy", "@co.dap.packed", "@co.dap.declare","@co.dap.implementation","@co.dap.simd", "@co.dap.reflection", "@co.dap.mop","@co.dap.nested","@co.dap.inner","@co.dap.final","@co.dap.const","@co.dap.decorator","@co.dap.specialize","@co.dap.symbol"|//mop => meta object programming|
|`DECORATOR`|"@co.dap.before", "@co.dap.after","@co.dap.around", "@co.dap.effects", "@co.dap.onEffect", "@co.dap.defer","@co.dap.callable", "@co.dap.executionmodel"||

***

### Built-in Metadata Parsing

Built-in directives, annotations, pragmas, and decorators under `@co.*` use one generic metadata application grammar:

```ebnf
annotation = "@", qualified-name,
             [ "(", [ annotation-argument-list ], ")" ] ;
```

Every named field or attribute in a directive, annotation, pragma, or
decorator uses `=`. The same rule applies recursively to records/maps nested
inside a metadata application. `:` is not a metadata binder and its use in any
metadata field or nested metadata record is a syntax error.

```folang
@co.dap.generic(types=[{name=T, variance=covariant}]) // valid
@co.dap.generic(types=[{name=U, bound=co.number}]) // valid nested fields
@co.dap.implementation(
    kind=co.dap.implementationKind.runtime,
    operation=co.runtime.operation.out.println
) // valid backend-neutral runtime-operation marker
```

For `@co.dap.implementation`, `kind` classifies how a bodyless standard declaration is implemented and `operation` identifies the compiler-owned backend-neutral runtime operation. The `operation` value is resolved as a qualified operation symbol and preserved in `.folenc`/HIR; it is not target-language source text. The annotation is valid only on a declaration kind for which this specification permits a runtime-operation marker.

The reference intentionally contains no colon-bound `@co.*` metadata example;
all such spellings are rejected by the grammar.

This metadata rule does not change ordinary value syntax. Object field
initializers and runtime map entries continue to use `:` according to their
own grammar:

```folang
employee := Employee{name: "Rao"};
StringMap co.type = co.Map(co.string, co.string);
map := StringMap{"name": "Rao"};
```

The compiler maintains a predefined built-in metadata registry for language-owned `@co.*` forms. After reading the qualified metadata name, the parser must match the **complete name** against that registry. A registered enabled form is parsed according to the common metadata grammar and its applicable known frontend rules. A registered reserved/future form may be recognized and diagnosed as unsupported according to its registry entry. An `@co.*` metadata name that is not present in the predefined registry is a **parse error**; an unknown language-owned metadata name is never silently accepted.

```text
@co.* metadata name
    -> lookup complete name in predefined built-in metadata registry
        -> registered and enabled      -> parse/collect metadata application
        -> registered but unsupported  -> unsupported-feature diagnostic
        -> not registered              -> parse error
```

Recognition of the metadata **name** is strict; knowledge of every field is not. Once a built-in form has been recognized, the parser must collect and preserve the complete metadata application, including every supplied positional argument, named argument, field, attribute, and argument expression.

For each collected field or argument:

- when the frontend already has defined knowledge of that field, it may validate the applicable value shape, structural requirements, defaults, or other frontend rules;
- when the frontend has no defined knowledge or semantic handling for that field, the field is still accepted, collected, and preserved as parsed; lack of frontend field knowledge alone is not an error and does not block frontend artifact generation;
- a malformed argument expression or malformed metadata argument-list structure remains a syntax error under the common metadata grammar; and
- later semantic/backend stages may interpret preserved fields according to the applicable feature contract.

Accordingly, an unknown built-in **form name** and an unknown/unhandled **field of a known form** are deliberately different cases:

```text
@co.dap.unknownForm(...)
    -> name absent from built-in registry
    -> parse error

@co.dap.generic(knownField=..., backendField=...)
    -> co.dap.generic is registered
    -> collect both fields
    -> validate fields the frontend understands
    -> preserve fields the frontend does not understand
```

User-defined metadata outside `co.*` is limited to annotations and decorators. Their qualified names are not looked up in the built-in registry; they are resolved through the ordinary imported/package symbol table and must resolve to a valid user-defined annotation or decorator declaration. An unresolved custom metadata name is a name-resolution/compiler error. FoLang provides no user declaration construct for directives or pragmas, so custom directives and custom pragmas are not available.

### Directive Placement

Directives are **source-file-level compiler metadata**. Every language-owned metadata form registered in the `DIRECTIVE` category, including every `@co.ddap.*` directive, must occur in the top-level metadata region of a source file. A directive cannot occur inside the body of a component, unit, class, struct, module, function, method, typeclass, instance, extension, matcher, annotation declaration, block, or any other declaration or nested lexical context.

The placement rule is structural and category-wide:

```text
source-file top-level metadata region                         -> directive permitted
inside `_ co.component = { ... }`                       -> compiler error
inside `_ co.unit = { ... }`                            -> compiler error
inside class/struct/module/typeclass/instance/etc. body       -> compiler error
inside function/method/extension/matcher body                 -> compiler error
inside ordinary/nested block                                 -> compiler error
```

For file-backed declaration sources, directives appear before the file's primary declaration. For `src/component.fol` and `components/<kind>/component.fol`, directives therefore appear before the `_ co.component = { ... }` declaration, never inside its body. For the executable entry file `src/appl.fol`, directives belong to the entry-file metadata preamble before the first non-metadata declaration or executable statement.

A directive's **semantic scope** is defined by the individual directive, but its **syntactic placement** is always file-level. For example, `@co.ddap.import` and `@co.ddap.alias` establish file-local bindings; `@co.ddap.use` establishes file-scoped activation; `@co.ddap.dynamicdispatch` is application-wide but is written only in the application entry-file preamble; and `@co.ddap.dynamicruntime` is valid only in a permitted `dynamicvmrt` capability source while still being written at that source file's top level.

A directive immediately preceding a primary declaration is not an annotation on that declaration. The compiler classifies the metadata name through the built-in registry first; entries classified as `DIRECTIVE` are attached to the current source-file/top-level semantic context rather than to an inner declaration AST node. Encountering a directive after entering a declaration/body context is a compile-time **metadata-placement error**.

For an ordinary package source file, the file has one primary top-level declaration. A directive in that file's metadata preamble may therefore configure or otherwise affect that primary declaration when the directive's own semantic contract says so. This does **not** make the directive a lexical declaration or a member of the primary declaration. The directive remains file-level compiler metadata.

Directives do not introduce names, lexical scopes, or symbol-table entries. The frontend records them separately with the source-file/top-level semantic context and consults them while validating or compiling the primary declaration. By contrast, symbols and symbol tables model name-bearing declarations and lexical visibility and may therefore exist recursively for classes, functions, methods, blocks, nested declarations, and other scoped constructs.

The common `@qualified.name(...)` surface syntax does not weaken this rule. After parsing the common metadata shape, the frontend classifies the built-in name by registry category. A form classified as `DIRECTIVE` or `PRAGMA` is accepted only through the file-preamble path; declaration/member/block metadata positions accept annotations or decorators instead. Thus a directive cannot be smuggled into a nested scope merely because directives and annotations share the same lexical shape.

```text
SourceFileContext
├── directives / pragmas / import metadata   // compiler metadata, not symbols
└── primary top-level declaration
    └── lexical/semantic contexts
        └── symbol tables
            └── nested symbol tables as required
```

This restriction applies automatically to future entries added to the language-owned `DIRECTIVE` registry unless the specification explicitly changes the category-wide rule.

### Pragma Placement

Pragmas are **executable-application-owned configuration metadata**. Every language-owned pragma whose complete name is registered in the `PRAGMA` metadata category, including every `@co.pdap.*` form, is valid only in the executable application's fixed entry source, `src/appl.fol`. This is a source-role/metadata-placement rule, not a distinct grammar production.

The following placement invariant applies to the entire pragma category:

```text
executable application: src/appl.fol                 -> pragma permitted
application package source under src/<package>/      -> compiler error
project-local components/<kind>/...                  -> compiler error
standalone projected application library             -> compiler error
standalone native library                            -> compiler error
standalone dynamicvmrt library                       -> compiler error
standalone packaged library                          -> compiler error
exported/packaged package contexts                   -> compiler error
```

A component, package, or library may document operational assumptions or recommended settings, but it cannot publish, export, inherit, or impose a pragma on its consumer. The executable application owns final application-wide policy. A pragma found outside `src/appl.fol` is therefore a compile-time **metadata-placement error** after the built-in metadata name has been recognized.

This restriction applies automatically to future entries added to the language-owned `PRAGMA` registry unless the language specification explicitly changes the category-wide rule.

***

***

### Built in Packages

| Public path | Responsibility |
|---|---|
| `co` | language data types and kinds projected from `fΦλ.lang`; network declarations such as `http`, `tcp`, and `udp` projected from `fΦλ.net`; and core collections/types such as `List`, `Map`, `Set`, and `Comparable` projected from `fΦλ.core` |
| `co.sys` | file, concurrent, parallel, goto, invoke, bind, call, apply, settimeout, setinterval, scheduler, cron, event |
| `co.os` | signal, cmd, execute, run, env, getenv, setenv, sleep, exit, cwd, chdir, fork, wait, pipe, dup, dup2, close, readfd, writefd ,random|
| `co.meta` | ast, instrument, transform, augment, reflect, introspect, patch, inject, create, runtime(eval,etc), realm |
| `co.native` | load, register, asm, inline, emit, ffi, spawnon[gpu,cpu,npu,apu,fpga,asic,tpu,mki,mcu],arch[x86,x86-64,risc,arm,vliw] |
| `co.in` | read, readln |
| `co.out` | println, print |
| `co.regex` | pattern, match, search |
| `co.crypto` | rsa, aes, hash, md5, rand, uuid, ssl, tls |
| `co.dap` | built-in decorators and annotations, including backend-neutral standard runtime-operation implementation markers |
| `co.ddap` | built-in directives|
| `co.pdap` | built-in  pragmas |
| `co.const` | `true`, `false`, `none` |
| `co.encoding` | base64Encode, base64Decode, json, yml, bson |
| `co.utils` | makeImmutable, makeShared, copyOnWrite, toSnapshot — object behaviour policies |
| `co.dynamic` | dynamic capabilities |
| `co.runtime` | compiler-owned backend-neutral runtime-operation identifiers and semantic contracts; `co.runtime.operation.*` markers are implemented by the selected backend/runtime |
| `co.compiletime`||
| `co.macro`||
| `co.pattern`||
| `co.control` | continuation, CPS, full/delimited continuation control, shift/reset, prompt/control, and continuation-oriented control abstractions |
| `co.cpca` | concurrent/parallel/async submission, task/thread execution facilities, future/callback completion, await, pools, channels, events, actors, process/distributed facilities, scheduling, fiber/coroutine facilities, defer, lazy, and related execution APIs |
| `co.hokrlt`||
| `co.operator`||
|`co.hw`| cpu, memory|
|`co.stex`||



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
