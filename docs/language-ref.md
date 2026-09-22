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



```folang

name co.string = "SomeName";

```

> folang has specific style declaring variables or any other constructs

> identifier < typeorkind >[=[{}][value]];

> Identifiers are valid folang group of characters

> types or kinds are either Builtin or User defined

***

### Types 

Folang provide four kinds of types

    1. Built In Types
    2. Built In Collection Types
    3. User Defined Types
    4. Derived Types 

#### Built In types

| Type | Kind |
|---|---|
|`co.string`||
|`co.int`||
|`co.bit`||
|`co.double`||
|`co.float`||
|`co.long`||
|`co.byte`||
|`co.char`||
|`co.any`||
|`co.bool`||
|`co.void`||
|`co.value`| value types stores values when take snapshot|
|`co.untyped`||
|`co.word`||
|`co.MatchBindings`||
|`co.number`||
|`co.uninit`||
|`co.error`|standard recoverable-error interface and first-error result contract; every emitted effect object is a class instance satisfying this interface|
|`co.AbstractError`|standard mixin supplying common `co.error` state and behavior; custom recoverable-error classes normally compose it while declaring the `co.error` interface|
|`co.literal`|literal representation for simple and compound literal objects|
|`co.operator`|declaration kind valid only in the `components/operators/component.fol` component context; parsed by the common FoLang parser and invalid in all other source contexts|
|`co.delegate`||
|`co.condition`||

#### Buit In Collection Types

| Name | Purpose|
|---|---|
|`co.List`||
|`co.Set`||
|`co.Map`||
|`co.Tree`||
|`co.Trie`||
|`co.Array`||
|`co.Tuple`||
|`co.Comparable`||
|`co.Stack`||
|`co.Queue`||
|`co.StructObject`||
|`co.ClassObject`||
|`co.ModuleObject`||
|`co.InstanceObject`||
|`co.ObjectObject`||
|`co.Matrix`||

#### User Defined Types

In folang user defined types are created by either using

    1. Built In Kinds
    2. Type Specializations


#### Derived types

Folang provides use ful types derived from above types

   1. Pointer
        
        a. normal
        b. fat

   2. array
       a. single dimension
       b. multi dimension
       c. jagged
       d. zero dimension
       e. zero length
       f. variable length
       g. size derived from initialization expression
   
   3. Thunks

   4. Slice
   5. Range
   6. Reference

        a. LValue
        b. RValue
        c. Heap References
   
   7. address
   9. word

***

### Built In Kinds

|Kind | Purpose
|---|---|
|`co.struct`||
|`co.cstruct`||
|`co.class`|struct-like ordinary mutable per-instance storage plus inheritance, abstraction, polymorphism, encapsulation, trait/mixin composition, behaviour extension/modification, and dynamic dispatch; field-level constant/immutable/shared/COW/locking policies are forbidden|
|`co.interface`| all abstract methods|
|`co.union`||
|`co.object`|one named singleton used for annotation implementation or as an explicit non-owning support object associated through `for=` with one or more classes; may own constants, immutable bindings, globals, shared state, and locks|
|`co.instance`||
|`co.matcher`||
| `co.loader`||
|`co.trait`| interfaces with default implementations |
|`co.mixin`| abstract classes alias|
|`co.extension`|reusable implemented functions that can be composed with supported classes without inheritance|
|`co.typeclass`||
|`co.module`||
|`co.unit`|stateless file-level container; ordinary units merge into the package namespace and `*.comp.unit.fol` attaches to a struct|
|`co.block`||
|`co.kind`||
|`co.signature`||
|`co.function`||
|`co.enum`|Closed tagged ADT. Its members are enum states; parameterized states are compiler-provided state functions returning the enclosing enum type. State-function payload parameters and calls are always named (`name=value`) and never positional; zero-parameter states are referenced directly without `()`.|
|`co.symbol`|  Used by AST |
|`co.component`|structural surface/container valid only in `src/component.fol` and standardized `components/<kind>/component.fol`; source context determines projected, packaged, or operator semantics|


### Type Specializations

| Type | Kind |
|---|---|
| `co.variants` |Built-in variadic type used to define a closed variant-based type. Its arguments declare states owned by the enclosing `co.type`; parameterized entries are state functions and bare entries are zero-parameter state values.|
|`co.tag`||
|`co.hokrlt`||
|`co.newtype`||
|`co.opaquetype`||
|`co.subtype`||
|`co.supertype`||
|`co.dependentType`|built-in RHS type-expression constructor used by a `co.type` declaration to define a value-indexed dependent type family; not a declaration kind or callable result kind|
|`co.polymorphic`|built-in RHS type-expression constructor used by a `co.type` declaration to define a named polymorphic type; its first argument is the binder set `{...}` and its second argument is the polymorphic type body; when that body resolves to a callable type, an ordinary named function may implement the resulting named callable contract directly|
|`co.refinementType`|base type restricted by a Boolean predicate over the candidate value|
|`co.associatedType`|type parameter associated with another generic or parameterized signature component; a matching module supplies its concrete `co.associatedType` binding|
|`co.predicateType`| works on types unlike refinement type like type constraints|
|`co.data`||
|`co.type`||
|`co.generic`||
|`co.shape`| type expressions on right side can be shapes (A)->(B) or function type expression|


***

### Rules of folang programs

   1. folang doesn't have `main` method or passed file name as argument.
   2. folang program starts from `project-dir` the compiler picks an entry file called `appl.fol` under `project-dir/src` folder
   3. variables can be declared in `appl.fol`
   4. Apart from variable declaration `appl.fol` can contain

       a. pragmas
       b. import directives
       c. annotations and/or decorators if applicable
       d. variable declarations
       e. expressions
       f. calls
       g. type specializations
       h. conditions
       i. loops
       j. ternary operators
       k. pattern matching


> This file can be used to test `folang` as a single file application


### Folang Reserved words

`co`, `this`, and `fΦλ` are hard-reserved words.

`$` is a compiler-owned **context sigil**, not an identifier and not a reserved
word. Source cannot declare or redefine `$`.

`fΦλ` (`f` = U+0066, `Φ` = U+03A6, `λ` = U+03BB) is the permanently reserved
language mark and the compiler-owned **private standard-package root**.

***

### Folang Operators

#### Arithmetic operators
`+`, `-`, `*`, `/`, `%`, `**`

#### Logical operators
`&&`, `||`, `!`

#### Bitwise operators
`&`, `|`, `^`

#### Comparison operators
`==`, `!=`, `<`, `>`, `<=`, `>=`

#### Other operator and language-token spellings
`@`, `#`, `!`, `~`, `$`, `^`, `(`, `)`, `_`, `` ` ``, `?`, `{`, `[`, `]`, `}`, `\`, `:`, `;`, `"`, `'`, `=`, `.`, `::`, `?=`, `:=`, `::=`, `,`, `..`, `...`, `<..`, `..<`, `<..<`, `=>>`, `=>`, `->`, `<-`, `->>`, `<->`,`@@`, `+=`, `-=`, `*=`, `/=`, `%=`, `**=`, `&=`, `^=`, `|=`, `<:`,`:>`,`^=>`,`->|`

#### Pre-Declared Operator Glyphs
`∪`,`∩`


***



### Standard operator examples

```folang
left  co.int = 10;
right co.int = 3;

notEqual co.bool = left != right;  // true

bitsAnd co.int = 6 & 3;            // 2
bitsOr  co.int = 6 | 3;            // 7
bitsXor co.int = 6 ^ 3;            // 5

mulAssign co.int = 6;
mulAssign *= 3;                          // 18

divAssign co.int = 18;
divAssign /= 3;                          // 6

modAssign co.int = 17;
modAssign %= 5;                          // 2

powAssign co.int = 2;
powAssign **= 3;                         // 8

andAssign co.int = 6;
andAssign &= 3;                          // 2

xorAssign co.int = 6;
xorAssign ^= 3;                          // 5

orAssign co.int = 6;
orAssign |= 3;                           // 7

```

### Folang Large scale applications and libraries

Folang provide a way to create enterprise applications which need code to be designed and arranged in different layers.

To support this any programmaing language we need Libraries, packages etc.,

Folang supports the following

    1. Application Packages
    2. Libraries
    3. Components
    4. Standard Library named **co**
        
#### Application Packages or source packages

Folang Packages are folders under project-root/src

For example project-root is payroll

`payroll/src/hr/emp`

`hr` is a top level package and `emp` is subpackage,  there is no separate conventions for packages

`src` always contains entry file for application and it is always named as `appl.fol`.


#### Libraries
  
Folang Libraries are standalone distributable artifacts, similar to application with packages the only difference is instead of `appl.fol` src folder contains `component.fol` which is called surface file.

Contains all the types and functions exposed to client the inner packages are hidden behind this surface file and client application doesn't have access to those definitions.

Folang has different kinds of libraries:

    1. application
    2. dynamicvmrt
    3. native
    4. packaged exports

All the libraries are encoded to protobuf serialized data with extension `.folenc`.

##### Packaged Exports

Folang also has support for a library kind called packaged export. The only difference is packaged export directly exposes internal api through `export` directive rather than `api` kind model like other library kinds.

example standard library named `co` provided by folang itself.

> Developer needs to keep these `.folenc` under project-root/lib

#### Components

Folang provide developers with a feature where he need not build libraries upfront to seggregate large scale applications with reusable components still achieve library benefits using `components`

These are similar to libraries but they are in source form instead of serialized `.folenc` living in `lib` folder under project-root.

These componnents needs to be in `project-root/components` folder 

Restriction of components:

    1. There is one and only one folder for a given kind of component named after the kind like `native`, `application`, `dynamicvmrt`, `packaged`
    2. what ever exported or exposed apis should or must be used by application partial usage will result in compile time error.
    3. Cross component imports or references not allowed only way is through main application or library source under `project-root/src` folders


#### Operator Components

    These are special components provided by folang to add new operators for a given project and these are project specific only.
    
    These can neither be exported or used with other components and/or libraries


### Importing packages/components/libraries 

Folang provides import directive to import

    1. Packages
    2. Libraries
    3. Components

`@co.ddap.import` is the directive

#### Import package 

@co.ddap.import(package="hr.employee", as="emp")

> `as` is alias it is not mandatory if not provided `package` becomes alias

> imports all the `.fol` sources under `src/hr/employee`


#### Import Libraries

@co.ddap.import(library="hrlib", as="hr")

Where hrlib is library name `hrlib.folenc` in `project-root/lib`

#### Import components

@co.ddap.import(component="native", as="native")

components are imported by kind where `component` itself is kind 

`project-root/components/native`

### Symbol look up in Folang

a. No Qualifier

    1. current Symbol table

        not found

    2. Parent symbol table repeat till it reaches top 
    
        not found

    3. parent Context branched out symbol table (name it p)

        not found

    4. Parent Symbol table of p repeat till it reaches top

        not found

    5. repeat 3 and 4 steps till reaches top context

        not found 

    6. error 

b. Qualifier (package)

    try to find symbol in that specific package if not found error

c. Qualifier ( other Contexts except call context)

    try to find symbol in that specific context if not found error

> For more information about contexts , symbol tables and others please refer section [Contexts, Symbol Tables and Symbols](#contexts-symboltables-and-symbols).



### Types in Detail

Most of the Built in data types or Built in collection types are similar to any other programming languages.

#### HOKRLT Types

```folang

   x co.hokrlt = co.int;
```
   These are the types who hold types as objects (values)

#### Value Type

`co.value` this is internal type to hold serialized data in folang

#### Literal Type

`co.literal` this is internal type to hold literal values as objects

#### Uninit Type

`co.uninit` this type is used with only oops programming where developer uses class and lifecycle methods some lifecycle methods return uninitialized object which of type `co.uninit`


#### Untyped tupe

`co.untyped` is to tell compiler the data/value/object is not typed one. especially used with macros and custom matchers

#### MatchBindings

`co.MatchBindings` folang internal for holding Bindigns object in custom matcher

#### Condition

`co.condition` folang internal type for holding condition object for loops/conditions/ternary operations. It is more than just boolean

#### Operators

`co.operator` folang provides this type for declaring new operators, there are restrictions in using this, folang restricts its usage to specifically in a component whose kind is operators, other places it will throw compiler error.

#### Error and AbstractError

`co.error` and `co.AbstractError` to hold errors/exception objects in folang.

### Type Specialization in Detail

#### Type Alias

```folang
   someInt co.type = co.int;

```

  1. Aliases are useful when we want to shorten the long fully qualified type.
  2. Aliases are useful when we want to represent a type with meaningful name

```folang

   EmpId co.type = co.string;
   DeptId co.type = co.string;

```

Aliases are representation of same type so they are exchangable and assignable from one another

```folang
   someInt co.int = 30; 
   empId EmpId =10;
   deptId DeptId = empId; // valid
   deptId =20; //valid

   someInt = deptId; //valid

```

#### Opaque Types

```folang
   
   EmpId co.opaqueType = co.int;
   DeptId co.paqueType = co.int;
   
   someInt co.int = 20;
   empId EmpId = 10; // valid 
   deptId DeptId = empId // In valid compiler error
   deptId = 20 ; // valid
   deptId = someInt; // valid
   someInt = deptId; // invalid
```
Opaque Types are representation of some base type whose values can be assigned from base type but not viceversa also similar opaque types are not interchangeble

Need for Opaque types, accidentally should not make mistake of passing one value to another

#### New Types

```folang
   SomeType co.newType = co.int;

   x someType = 10;
   y co.int = 20;

   x = y; // Invalid compiler error
   y = x; // Invalid compiler error
```

New Types in folang provides a way to create distinct type from existing types.
these are completely different types and not exchangable even though base type is same.

#### ADT types

```folang
  someADT co.type = co.int | co.string;
```

These are tagged unions are  types where someADT can be either integer type or string type


#### Super Types

  ```folang
     empType co.type = some.Employee;

     superType co.supertype = some.ContractEmployee; 

  ```
Here ContractEmployee is subtype of Employee, so superType holds any parent type chain of ContractEmployee excluding ContractEmployee


#### Sub Types
  ```folang
     empType co.type = some.Employee;

     subType co.subtype = some.Employee; 

  ```

Here subType holds any subtype of Employee type excluding Employee

If someone wants both base and super/sub types

```folang
someType co.type = subType | some.Employee;
someOtherType co.type = superType | some.ContractEmployee;
```

#### Refinement Types

```folang

percentage co.refinementType =
    (co.int).where(_ >= 0 && _ <= 100);

k percentage = 200 ; // compiler error as it should be between 0 and 100
```

Refinement types provides a way to restrict the values a type can accept. In the sense it modifies existing types for accpeted values


#### Dependent Types

```folang

    Vector(n) co.type =
        co.dependentType(
            co.int->([n])
        );

    v3 Vector(3) = Vector(3){1, 2, 3};
    v4 Vector(4) = Vector(4){1, 2, 3, 4};


```
Here v3 and v4 are not same it is length dependent type where array length and type are matched not just type of the array.

It differs with refinement type in accepting value it doesn't restrict v3 or v4 what kind of values it can accept for a given type like refinementtypes.

Path Dependent type

```folang
identity(x co.int)->(x.type) = { $=> x; }
```

#### Predicate Types

```folang
    someType co.predicateType =
        co.type.where(
            candidate =>
                candidate == co.int ||
                candidate == co.string
        );

```
Predicate types are not for general use they are used with Generics to contraint concrete types

Variants work at value level predicate types work at type level.


#### Polymorphic Types

```folang

SomeFArg co.type = co.polymorphic({T}, (T, T)->(T));

```
These like predicate types used with Generics. The type T is supplied by the Generic 


#### Variant Types/Parameterized

```folang
Option(T) co.type =
        co.variants(Some(T), None);

```

where T is the value 


#### Associated Types

```folang
// In signature
 T     co.associatedType;

 // in implementing Module

T co.associatedType = co.int;
 ```

Assocated types are used only in the context of signatures and modules of folang to inform the Generic type is an associated type which is provided by implementing module.

> Folang Modules don't support Generics and to provide the capability these associated types are used.


#### Data Types

```folang

SelectedValue co.type = 
    co.data( 
        StringValue(co.string),
        BoolValue(co.bool)
        );

```

These are kind of specialized variants, like specialization of generics in folang these are specialization with actual types/values of variants. They are fixed.

#### Tag Types

```folang

co.tag(co.string, "Hello")
```
These are runtime type descriptors mainly used in pattern matching.

#### Kind Type

```folang
 blockormacro co.kind = block | macro
```

Kind types in folang are ADTs for Kinds not for types

These are mainly useful in macros where we need AST to be transformed

#### Generic Type

```folang
   someType co.type = co.generic(T);

```

These are like associatedtypes right now reserved for future. folang don't use this courrently

#### Function Types

```folang

someFuntype co.type = (co.int, co.int)->(co.int);
```
Folang provides function types to pass functions as parameters and results from a function 

Folang doesn't provide inline function sytax for parameters and results


#### Delegate Types

```folang
someDelegate co.delegate =  (a co.int, b co.int)->(co.int, co.int);
```
Eventhough looks like funnctiuon type the intent is different these support of delegates.

```folang
someDelegate = myFunc;
someDelegate(10, 20); // invokes the currently registered function

someDelegate = myFunc;
someDelegate += mySecondFun;
someDelegate(10, 20); // invokes the registered delegate functions
```

These are more powerful then simple function chaning provided by folang for simple operations. Please refer [Function Chaining](#function-chaining) for more details.



#### Derived Types

```folang
IntRef       co.type = co.int->(&);   // reference
IntLValueRef co.type = co.int->(&&);  // LValue reference
IntHeapRef   co.type = co.int->(~);   // heap allocated reference
IntAddress   co.type = co.int->(@);   // address
IntThunk     co.type = co.int->(^);   // thunk
IntSlice     co.type = co.int->([:]); // slice


ThreeInts           co.type = co.int->([3]);
InferredInts        co.type = co.int->([]);
InferredGrid        co.type = co.int->([,]);
ZeroLengthArr       co.type = co.int->([0]);
ZeroDimArr          co.type = co.int->([.]);
JaggedArray         co.type = co.int->([][]);
VariableLengArray   co.type = co.int->([...]);


IntPtr     co.type = co.int->(*);
IntDblPtr  co.type = co.int->(**);
IntDeepPtr co.type = co.int->(*****);

IntRange co.type = co.int->(..);

```

***



### Contexts Symboltables and Symbols

##### Context

``` text
Context {
    ParentId:                  string,            // nearest enclosing named-definition/structural Context
    ParentCtxSymbolTableId:   string,            // exact enclosing visibility table at definition branch point
    Id:                        string,            // unique Context ID

    ImportedContextIds:        { <alias>: <context-id> },

    Prefix:                    string,
    ContextType_:              string,
    SymbolTables_:             [string], // symboltables atleast one 
    ChildCtxIds:               [string],          // direct child named-definition Context IDs
    ResolutionPolicy:          string,
    OwnerSymbolId:             string             // named definition owning this Context; empty for structural roots
}
```


##### Symbol Tables


```text
SymbolTable {
    Id:              string,   // unique symbol-table ID
    ParentId:        string,   // previous declaration-order segment in the same lexical scope
    LexicalParentId: string,   // enclosing lexical-scope table active at scope entry
    ContextId:       string,   // named-definition/structural Context that owns this scope
    Prefix:          string,   // frontend qualification/debug prefix

    SymbolIds:     [ <symbol-id> ],
    SymbolsByName: { <declaration-key>: [ <symbol-id> ] }
}
```



##### Symbols

```text
SymbolInfo {
    GetSymbolType() -> string
    GetSymbolID() -> string
    GetContextID() -> string // context owned by this symbol, or empty
    GetType() -> string
    GetName() -> string
    IsInternal() -> bool
  	SetOwnedContextID(string) // OwnedcontextId if the symbole is qualified to own context or create context then the contextid
	GetSymbolTableID()-> string  // symboltableid where current symbol is stored
	Anchor() -> string  // is same as GetSymbolTableID
}
```
> Common Data

```text
SymbolDetails  {
	SymbolId_        string
	OwnedContextId   context // context owned by this symbol, if any
	SymbolType_      string
	Name_            string
	IsInternal_      bool
	Type_            string
	SymbolTableId    string   //symboltableID where this symbol is defined
	ResolutionState_ string // "resolved" | "unresolved" | "partially_resolved"

}

```




## Appendix A - Complete FoLang EBNF Grammar

The standalone consolidated EBNF referenced below is the normative lexical and syntactic grammar for FoLang. The prose sections of this reference define semantics and parser-validity constraints without maintaining a second embedded copy of the grammar.

[{{FOLANG_EBNF}}](./grammar/folang.ebnf)
