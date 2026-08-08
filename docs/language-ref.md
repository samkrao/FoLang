<p align="center">
  <img src="Banner_52_t.jpeg" width="600" alt="Foλang Logo"/>
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

---



## Design Overview

<p align="center">
  <img src="./design.png" alt="Design" width="600" style="max-width:100%;"/>
</p>


FoLang follows a deliberately different approach from conventional programming language designs.
The system is structured to ensure **clear separation of concerns**, **license isolation**, and **extensibility through well-defined integration boundaries**.

---

## Compiler and Backend 

### 1. Frontend

The Frontend is responsible for source-level analysis and semantic processing of FoLang programs.

#### Components

- Scanner / Lexer
- Parser
- AST / Parse Tree Generator
- Symbol Table Generator
- Semantic Analyzer

#### Implementation

- Implemented in **Go**
- Uses Go structures internally for AST, symbol-table, and semantic processing.
- The externally consumable frontend output is always serialized as a **Protocol Buffers binary** artifact.
- The frontend writes that artifact under the reserved project-root `build/` directory.
- The `build/` directory is generated output, never a source/package/library discovery root.
- If `build/` does not exist when compilation begins, the compiler creates it.

#### License

- **GNU General Public License v3 (GPLv3)**

---

### 2. Backend

The Backend is responsible for transforming validated frontend output into executable artifacts.


> **Default Backend shouuld be downloaded/build separately. They are not bundled with Frontend Binary**

#### Components

- Intermediate Representation (IR) Generator
- Native Binary Executable Generation

#### Implementation
    
A backend may be implemented in any language. It consumes the validated frontend artifact from the project-root `build/` directory. The frontend/backend interchange encoding is always Protocol Buffers binary; projects do not select another wire format or frontend output directory.

The exact protobuf schema/version belongs to the frontend/backend contract. The exact artifact basename may be specified separately by the compiler/toolchain, but its location is always beneath `<project-root>/build/`.

#### Default Backend

- Backend orchestration  implemented in **Go**
- Code generation target is **C++**
- Uses **Clang** or **GCC** to generate native binaries from generated C++ IR

#### License 

**3rd Party Backends can have their own licensing terms and implementation choices**. Default backend has the following license.
**Default backend is not part of the complete compiler binary and is separate**; it must be downloaded or built separately.

- **BSD 3-Clause License**
---

#### Frontend Output Contract


#### Configuration File Structure

Informs the frontend how to generate IR to be consumed by backend. This process is not different for default backend.

```json
{
  "protocol":   "folang-plugin/1.0",
  "hir_schema": "folang-hir/1",
  "wire":       "protobuf",
}
```



The frontend has one fixed external interchange contract:

```text
<project-root>/build/
    └── <frontend-artifact>    Protocol Buffers binary
```

Rules:

- frontend output is always Protocol Buffers binary;
- the output location is always the reserved root-level `build/` directory;
- the wire format is configurable and is needed for backend any backend should always look into `build` folderr inder `<project root>`
- `build/` is compiler-generated and excluded from all source, package, source-library, operator-source, and packaged-library discovery;
- `.fol` source files placed under `build/` are invalid project layout and are never compiled as source;
- backends consume the validated protobuf artifact from `build/`.

---

### Licensing Summary

| Layer    |  Responsibility                           | Implementation                    | License      |
|----------|------------------------------------------|-----------------------------------|--------------|
| Frontend | Parsing and semantic analysis            | Go                                | GPLv3        |
| Backend (default) | IR processing and native code generation | Go (orchestration) + C++ (target) | BSD 3-Clause |



> The copyrightable material in the [FoLang Language Definition and Documentation](#folang-definition-and-documentation-license), including its syntax, grammar, and semantic-rule descriptions, is licensed separately under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
---

### 3. Capability Security Model

FoLang's compiler ships with all language features compiled in but **systems and FFI features are disabled by default**. The compiler has no hardcoded keys — capability configuration happens entirely at install time. This moves authorization from source code (developer-controlled) to the compiler installation (organization-controlled).

---

#### Feature Tiers

| Tier | Features | Default State |
|---|---|---|
| `application` | All standard language features, `co.net`, `co.core`, `co.encoding`, `co.crypto`, etc. | ✅ Always enabled |
| `system` | Raw pointers, pointer arithmetic, `co.sys.unsafe`, MMIO, heap allocators | 🔒 Disabled — requires install-time configuration |
| `ffi` | `@co.dap.native`, `co.sys.ffi`, extern types, `co.lang.void` pointers, C ABI | 🔒 Disabled — requires install-time configuration |

---

## Quick Start

### Hello World

```folang
// hello.fol — entry file, no annotation needed
co.out.println("Hello FoLang!");
```

Or with an alias for shorter form:

```folang
@co.ddap.alias(co.out, as="out")

out.println("Hello FoLang!");
```

## Variables

```folang
// typed
name co.lang.string = "Rao";
age  co.lang.int    = 30;

// inferred from value — := errors if already declared
name := "Rao";
age  := 30;

// define infer and assign if not defined, otherwise assing new value
name ?= "Kumar";
```

`co.lang.string` and `co.lang.int` are Builtin Data types to know more about Builtin Data types, plese refer secion [Builtin Data Types](#builtin-data-types)

`=`, `:=` and `?=` are built in operators to know more about Built in operators please refer section [Builtin Operators](#builtin-operators)

## Constants and Immutability

FoLang separates two properties that are often confused. One is about *when* a
value is known. The other is about *whether* it can change.

```folang
// compile-time constant — value known while compiling, substitutable
@co.dap.const SIZE co.lang.int = 1024;

// immutable binding — cannot be reassigned, value need not be known early
@co.dap.final startedAt co.lang.int = co.sys.now();
```

| Annotation | Guarantees | Value known at compile time | Usable as an index |
|---|---|---|---|
| `@co.dap.const` | value is fixed and known while compiling | ✅ | ✅ |
| `@co.dap.final` | binding cannot be reassigned | ❌ not required | ❌ |

Every `@co.dap.const` is also immutable. The reverse does not hold: a
`@co.dap.final` binding may be initialised from a function call, a file, or
user input, so the compiler cannot substitute a literal for it.

`@co.dap.final` is the declaration-site form of the immutable object kind
described in the object mutation policy. `makeImmutable` applies the same
property to a value at run time.

Only `@co.dap.const` may name an array size or a dependent type index, because
those positions require a value the compiler can substitute. See section
[Dependent Type Index Rules](#dependent-type-index-rules).

## Single Source Application File 

FoLang developers can create a complete executable program in one source file. A **single-source application** is an application whose entry file contains the complete program and does not depend on user package source files.

A single-source application file and an application entry file use the same entry-file grammar, context, and restrictions. This section presents the allowed constructs, so a developer can start programming without first reading the complete specification.

#### Allowed Constructs

The application file may contain:

- built-in directives that are valid for an entry file
- package and library imports
- import aliases declared with `as=`
- file-local aliases for `co.*` paths declared with `@co.ddap.alias`
- type aliases and ADTs declared with `co.lang.type`
- parameterized `co.lang.type` constructors such as `Option(T) co.lang.type = Some(T) | None()`
- new types declared with `co.lang.newtype`
- opaque types declared with `co.lang.opaquetype`
- dependent-type aliases and dependent-type usages that do not declare an ordinary or type-level function
- subtype declarations
- supertype declarations
- non-capturing entry-local function-pattern groups
- capturing entry-local `let` function-pattern groups
- variable declarations, initialization, assignment, and mutation
- calls to built-in methods and functions
- calls to imported package and library APIs
- ordinary expressions
- comprehensions
- ternary expressions
- conditions and loops
- containment operations such as `contains`
- iterators such as `each`
- pattern matching and destructuring

```appl.fol

  a co.lang.int = 10;
  b co.lang.int = 20;

  co.out.println( a + b);

```
--- 
> `co` is a keyword/reservedword in `folang` for more details please refer section [Reserved Words](#reserved-words)

> `co` is a built in package in `folang` for more details please check [Built in Packages](#builtin-packages)

> a + b is an expression in `folang`

> More about Expression rules please refer section [Expressions](#Expressions).

#### Built-in and Imported Names

All `co.*` paths are always available.

A developer may use the complete built-in path:

```folang
co.out.println("Hello");
co.core.list.of(1, 2, 3);
```

A developer may optionally create a file-local alias:

```folang
@co.ddap.alias(co.out, as="out")
@co.ddap.alias(co.core.list, as="list")

out.println("Hello");
values := list.of(1, 2, 3);
```

Creating an alias does not hide the complete `co.*` name; both forms remain valid in that file.

Third party packages User packages and libraries are not automatically available. They must be imported using `@co.ddap.import`. When `as=` is present, the imported API is accessed through that alias. When `as=` is omitted, the complete imported package or library path must be used.

```folang
@co.ddap.import(package="hr.employee", as="emp")
first := emp.EmployeeService.find(1001);

@co.ddap.import(package="finance.payroll")
second := finance.payroll.calculate(request);
```

A developer needs to import the package to use

> `@co.ddap.import` and `@co.ddap.alias` are built in directives to know more about built in directives please refer section [Built-in Directives](#builtin-directives)

> for more details on `folang` import directive please check [ Import details ](#imports) 

> `println` is a built in method in `folang` for more details please check [Built in  Methods](#builtin-methods)

### Variable Kinds

`folang` supports different kind of variables for different purposes, even though language provides, developers cannot use at random places. 
For more information where they are supported refer section [Variable Kind support](#variable-kinds-support) 

### Simple Variable Declaration

```folang
someVar co.lang.int;
someString co.lang.string;
```

### With Initialization

```folang
someBool co.lang.bool = co.const.true;
someInt co.lang.int = 42;
```

### With Type Inference

```folang
someVal := "Hello, World!";
someNum := 3.14;   // if not defined, define and initialize; else throws error
someR ?= "Kamesh"; // if not defined, define and initialize; else reassign
```

### Alpha Character and String Literals

The alpha release accepts only the basic literal subset:

```folang
message := "hello";
letter := 'A';
```

A string is unprefixed, remains on one source line, and cannot contain a
backslash or an embedded double quote. A character literal contains exactly
one non-backslash character.

Encoding prefixes, escaped characters, raw strings, and universal character
names are reserved for a later release. The scanner recognizes the complete
spelling and reports an unsupported-feature error instead of tokenizing it as
several unrelated expressions. For example, all of these are rejected:

```folang
x := "quoted: \"text\"";
x := '\n';
x := u8"hello";
x := R"(raw text)";
x := '\N{LATIN CAPITAL LETTER A}';
```

### Pointer Declaration

```folang
somePtr    co.lang.int->(*);
someDblPtr co.lang.int->(**);
someDeepPtr co.lang.int->(*****);
```

The number of consecutive `*` characters is the pointer degree. Any positive
degree is permitted; `*` and `**` above are common examples, not a maximum.

### Array Declaration

```folang
someArray       co.lang.int->([5]); // single dimension
someDblArray    co.lang.int->([2,3]); // multi dimension
someJaggedArray co.lang.int->([2][3]); // jagged
someVLArray     co.lang.int->([...]); //varialbe length
someZeroLA      co.lang.int->([0]); //zero length array
someZeroDimA    co.lang.int->([.]); // zerodimension array
```

### Array Declaration with Initialization

```folang
someInitializedArray    co.lang.int->([3])  = [1, 2, 3];
someInitializedArray1   co.lang.int->([])   = [1, 2, 3];
someInitializedDblArray co.lang.int->([,])  = [[1, 2], [3, 4]];
```

### Reference Declaration

```folang
someRef       co.lang.int->(&);    // reference
someLValueRef co.lang.int->(&&);   // LValue reference
someHpRef     co.lang.int->(~);    // heap allocated reference
someAddr      co.lang.int->(@);    // address
someThunk     co.lang.int->(^);    // thunk
someSlice     co.lang.int->([:]);  // slice
```

### Range Declaration

```folang
// Typed range variable declaration
someRange co.lang.int->(..);

// Inferred range declarations
rangeI := 1 .. 10;      // [1, 10]   ExcludeStart=false, ExcludeEnd=false
rangeS := 0 <.. 5;      // (0, 5]    ExcludeStart=true,  ExcludeEnd=false
rangeL := 0 ..< 100;    // [0, 100)  ExcludeStart=false, ExcludeEnd=true
rangeB := 0 <..< 100;   // (0, 100)  ExcludeStart=true,  ExcludeEnd=true
rangeE := .. 100;      // open lower bound  (_, 100]
rangeF := 1 ..;        // open upper bound  [1, _)
```

### Auto and Dynamic Variable Declaration

```folang
someAutoVar    co.lang.auto    = "Hello"; // type inferred from value; initialization required
someDynamicVar co.lang.dynamic;           // dynamic typing
```
### Lazy

```folang
@co.dap.lazy
x = add(1, 2);  //on doing some thing on x calls add(1,2) till that time add function on right hand side is not invoked
```

### Bind Variables

`$[0-9]*`

### Discard / Wildcard Variable

`_`

### Comma and Grouping

```folang
// Comma
x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true;

// Grouping
(x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true);

// Tuple/grouping expression
pair := (x, y);
```

A comma inside a parenthesized expression or grouped declaration must introduce
another expression or declarator. Therefore `(x,)`, `(x, y,)`, and
`(x co.lang.int = 10,)` are invalid. Collection literals have their own rules
and may permit a trailing comma where their production says so. Record patterns
also require another field after every comma; `Employee{id: value,}` is invalid.

---

## Fat Pointers

```folang
x co.lang.int->(*, kind="", meta={});

co.lang.int->(*, meta={});

y co.lang.int->(*, meta={len:co.lang.usize, vtab:somepkg.VTable->(*)});

z co.lang.int->(*,kind=region, meta={});
```

```
Pointer
├── base_type: T
├── kind: <FatKind>
│    ├── thin
│    ├── slice
│    ├── relative
│    ├── trait
│    ├── buffer
│    ├── view
│    ├── opaque
│    ├── custom
|    ├── mem
|    ├── nullptr
|    ├── sptr
|    ├── uptr
|    ├── ptrdiff
|    ├── usize
|    ├── ssize
│    └── (region)  ← optional syntactic sugar
└── meta:
     ├── region: heap | stack | global | numa(N) | mmio | constant | …
     ├── len, cap, vtab, bits, endian, …
```

### Pointers for address manipulation

```folang
y co.lang.word->(repr=intptr);
z co.lang.word->(sign=unsigned, repr=uintptr);
p co.lang.word->(repr=ptrdiff);
n co.lang.word->(sign=unsigned, repr=usize);
m co.lang.word->(repr=isize);
o co.lang.void->(repr=nullptr);
```

### Relative Pointers

```folang
z co.lang.int->(*,kind=relative, meta={});
```
---

**Note** Other than normal variable declaration rest have restrictions

## Conditions, Loops and Iterators

### Conditions

```folang
(boolean truth).do({
}).otherwise(boolean truth).do({
}).otherwise.do({
});
```

### Loops

```folang
(boolean truth).loop({
}).otherwise(boolean truth).loop({
}).otherwise.loop({
});
```

### Condition and Loop Mix

```folang
(boolean truth).do({
}).otherwise(boolean truth).loop({
}).otherwise(boolean truth).do({
}).otherwise.loop({
});
```

### Ternary Operator

```folang
s = (boolean truth).return(some var/value).otherwise.return(some val/var);
s = (boolean truth).return(some var/val).otherwise(boolean truth).return(some var/val).otherwise.return(some var/val);
```

### Looping Arrays / Lists / Maps / Ranges

```folang
arr co.lang.int->([5]) = [6,7,8,9,10];
arr.each(idx, val).do({
    co.out.print(idx);
    co.out.print(" :: ");
    co.out.println(val);
});

arr.each(_, val).do({
    co.out.println(val);
});
```

### Array / List / Map / Range — Contains Element

```folang
arr co.lang.int->([5]) = [35,57,96,81,31];
k co.lang.int = 31;
arr.contains(k).do({
    co.out.println(k);
}).otherwise.do({
    co.out.println("Not Found");
});
```

### Comprehensions *(planned)*

```folang
k := (1 .. 10).filter(|x| => x % 2 == 0).map(|x| => x * x);

result := for (x <- List(1,2,3)).yield(x * 2);         // List(2, 4, 6)
result := for (x <- Set(1,2,3)).yield(x * 2);          // Set(2, 4, 6)
result := for (x <- Some(5)).yield(x * 2);             // Some(10)
result := for (x <- fetchData()).yield(x.process());   // Future

ages  := {"A":30,"B":40,"c":66,"e":88};
upper := for ((name, age) <- ages).yield(name.toUpperCase, age);
```

---
### Pattern Matching

```folang
x co.lang.int = 10;

x.match.case(n: n > 10 => { n = n+100; "GT" }).case( n: n < 10 => "LT").default("EQ");
x.match.case(n: n > 10 => { n = n+100; "GT" }).case( n: n < 10 => "LT").case(_=>"EQ");
x.match(co.pattern.Type).case(co.lang.int => ...).case(co.lang.float => ...);
x.match(co.pattern.Value).case(0 => ...).case(1 => ...);
x.match(co.pattern.Instance).case(xx.CAT => ...).case(xx.DOG => ...).default("Animal");
x.match(co.pattern.Object).case(xx.Ball => "Ball").case(xx.CAT => "CAT").default("Unknown");
x.match(co.pattern.Shape).case(Point{x, y} => ...).default(...);

x.match(co.pattern.Any).case(co.lang.int => ...).case(co.lang.float => ...).case(0 => ...).default( ...);

x.match(PositiveEvenMatcher).case(0 => "Neither even nor odd").case(2 => "First Even Prime").default(...);
```

A match chain contains one or more `.case(...)` arms followed by at most one
terminal `.default(...)` arm. `.otherwise` is not a match arm; it belongs to
conditional and ternary return chains. A case or default result may itself be a
ternary return expression, and that nested expression may consequently contain
`.otherwise.return(...)`.

> **Object vs Instance in FoLang:** Instance is from types of class/structs. Objects are anything — functions, classes, structs, types, etc.

> `_` is a special discard/wildcard variable. In a call it is permitted only as
> the first, index/key argument of a receiver-qualified `.each`, as in
> `items.each(_, value)`. Transparent grouping around the member callee, for
> example `(items.each)(_, value)`, does not change this rule. The value
> argument of `each` cannot be discarded, and `contains(_)` and `containsVal(_)`
> are invalid because containment must compare a real value. Patterns and the
> filename-derived declaration-name form give `_` their own explicitly described
> meanings; it is not a general expression identifier.

> `PositiveEvenMatcher` is custom matcher for more details about creating custom matcher please refer to section [Custom Matcher](#matchers)

### Type Declarations

```folang
// Alias
x co.lang.type = co.lang.int;

// New
x co.lang.newtype = co.lang.int;

// Opaque
x co.lang.opaquetype = co.lang.int;  //opaque types act like alias with base type from which they declared and act like distinct type (new types) for others even when two opaque types derived from same base types

// ADT (tagged union)
y co.lang.type = co.lang.int | co.lang.char;

// Subtype / covariant
test co.lang.subtype = co.lang.int;

// Supertype / contravariant
test co.lang.supertype = co.lang.int;
```
---

### Let Bindings

```folang
y co.lang.int = let({x = 10}).in({x + 1});
y co.lang.int = let({$ = 10}).in({$ + 1});  // $ refers to the value being defined

x co.lang.int = (x + 1).where(x = 10);
x co.lang.int = ($ + 1).where($ = 10);

offset := 100;

let adjust(0) = offset;
let adjust(n) = n + offset;
```

> `$` is a special identifier usable in ordinary `let` binding expressions for recursive or self-referential expressions.
>
> Ordinary `let` value-binding expressions remain available in language contexts that permit them, but they are forbidden directly in the application entry file. In the entry file, `let` is reserved exclusively for a named function-pattern group that captures at least one surrounding runtime binding. It cannot introduce an anonymous function, a general closure value, or a curried function.

---

### Function Pattern

```folang

Option(T) co.lang.type= Some(T) | None();

f(Some(x)) => { this.return x + 1; }
f(None())  => { this.return 0; }

// desugars to:
f(v Option(co.lang.int))->(co.lang.int) = {
    this.return v.match()
        .case(x: Some(x) => x + 1)
        .case(_: None() => 0);
}
```

`=>` introduces a bare function-pattern clause. `=>>` is the distinct function-delegation operator and `==>>` is the clousre/curry expression and does not introduce a function pattern.

Function-pattern groups are permitted in the application entry file as restricted entry-local dispatch helpers. A bare group cannot capture surrounding runtime variables. A `let` function-pattern group must capture at least one already initialized entry-file runtime binding and is the only entry-file construct that permits such capture. Neither form permits ordinary function declarations, anonymous functions, general closure values, currying, partial application, or escape as a function value.

> More about Let and function patterns please refer section [Let and Function Patterns](#let-and-function-patterns)

> Single source application file is for testing and getting feel of `folang`
> Real world applications contain more than single source application file they use concepts of abstraction, encapsulation, inheritance and polymorphism at the core with many other features. Also these applications depend on external libraries, packages to achive clear boundaries with above features. To develop these kind of applications we need following features at minimum.

   1. [package source files](#package-source-files) under [packages](#package-in-detail)
   2. [Entry File](#application-entry-file)
   3. [Libraries](#libraries) and [Library surface files](#library-surface-file)
   4. [imports](#imports)

Foλang Supports many features to develop enterprise application with [intent](#folang) and [FoLang Philosophy — Uniform Object Model](#folang-philosophy--uniform-object-model) are listed below

 Complete Feature list:

   1. [Packages](#packages)
   2. [UDT](#udt-user-defined-data-types)
   3. [Functions](#functions)
   4. [Units](#units)
   5. [Imports](#imports)
   6. [Macros](#macros)
   7. [Templates](#templates)
   8. [Annotations and Decorators](#annotations-and-decorators)
   9. [Type Classes](#type-classes)
  10. [Types](#types)
  11. [Generics](#generics)
  12. [Matchers](#matchers)
  13. [Lambdas](#lambda)
  14. [Execution Models and Control Abstractions](#execution-models-and-control-abstractions-library-typeadvanced)
  15. [Extensions](#extensions)
  16. [Native code](#native-code-library-type-systemffi)
  17. [Indexers](#indexer)
  18. [Dependent Types and Type-Level Functions](#dependent-types)
  19. [Dynamic Runtime](#dynamic-runtime-library-typedynamicvmrt)
  20. [Local/Nested Types and Functions](#local-andor-nested-types-and-functions)
  21. [Libraries](#libraries)
  22. [Operators](#operators)
  23. [Forward / Extern Declarations](#forward--extern-declarations)
  24. [Labels and Named Blocks](#labels-and-named-blocks)
  25. [Reflections](#reflections)
  26. [Comprehensions](#comprehensions)

---

In FoLang, file-backed primary declarations use their own `<Name>.fol` files. Package functions and non-UDT type declarations are grouped in any number of `*.unit.fol` files, while struct-associated behavior is placed in `<StructName>.comp.unit.fol`. These are all [package source files](#package-source-files).
Lets discuss packages before going to UDTs and Functions


## Packages

### Package Identity

FoLang applications use four fixed project-root domains: `src/`, `srclib/`, `lib/`, and `build/`. These directory names are compiler-defined filesystem domains, not packages, and never contribute namespace components.

Ordinary application package discovery occurs only below `src/`:

- `src/` itself is not a package.
- `src/` must contain exactly one direct file, `<entryfilenam>.fol`.
- no other file may occur directly in `src/`.
- every other entry directly under `src/` must be a package directory.
- package dot paths are relative to `src/`.
- nested package directories may form an arbitrarily deep package hierarchy.
- `srclib/`, `lib/`, and `build/` never participate in application package discovery.

Examples:

```text
/appl/src/hr/           -> package "hr"
/appl/src/hr/employee/  -> package "hr.employee"
/appl/src/auth/         -> package "auth"
```

Neither the application root nor `src/` is a package.

### Multi-File Packages

Multiple `.fol` files in the same package directory automatically belong to that package:

```text
src/hr/employee/
├── Employee.fol      -> hr.employee
├── EmpService.fol    -> hr.employee
└── EmpValidator.fol  -> hr.employee
```

---

### Application Project Layout

The canonical application layout is:

```text
/appl/
├── src/                              <- application source domain; NOT a package
│   ├── app.fol                       <- exactly one direct file; application entry surface
│   ├── hr/                           <- package hr
│   │   ├── employee/                 <- package hr.employee
│   │   │   ├── Employee.fol
│   │   │   └── EmpService.fol
│   │   └── payroll/                  <- package hr.payroll
│   │       └── Payroll.fol
│   └── auth/                         <- package auth
│       └── Auth.fol
│
├── srclib/                           <- application-local special-source domain; NOT a package
│   ├── ffi/                          <- the one optional local FFI source library; NOT a package
│   │   ├── library.fol               <- exactly one direct file; fixed surface filename
│   │   ├── native/                   <- internal package native
│   │   │   └── Binding.fol
│   │   └── conversion/               <- internal package conversion
│   │       └── Convert.fol
│   ├── system/                       <- the one optional local system source library; NOT a package
│   │   ├── library.fol
│   │   └── platform/                 <- internal package platform
│   │       └── Platform.fol
│   ├── advanced/                     <- the one optional local advanced source library; NOT a package
│   │   ├── library.fol
│   │   └── runtime/                  <- internal package runtime
│   │       └── Runtime.fol
│   ├── dynamicvmrt/                  <- the one optional local dynamic-runtime source library; NOT a package
│   │   ├── library.fol
│   │   └── meta/                     <- internal package meta
│   │       └── RuntimeMeta.fol
│   └── operators/                    <- the one optional local operator source; NOT a package
│       └── library.fol               <- exactly one file; fixed operator-library surface filename; no subdirectories
│
├── lib/                              <- compiled-library artifact domain; NOT a package
│   ├── postgres.folenc               <- Protocol Buffers binary library artifact
│   ├── crypto.folenc
│   └── device.folenc
│
└── build/                            <- generated-output domain; NOT a package
    └── <frontend-artifact>           <- frontend Protocol Buffers binary
```

#### Reserved-Domain Rules

`src/`, `srclib/`, `lib/`, and `build/` are fixed application-root directory names. They are filesystem/compiler domains only; none is a package.

`src/` rules:

- `src/app.fol` is the application entry file and the only file permitted directly in `src/`.
- all reusable application source must reside in package subdirectories below `src/`.
- package names begin below `src/`; therefore `src/hr/employee/` is `hr.employee`, never `src.hr.employee`.

`srclib/` rules:

- `srclib/` itself is not a package or a library.
- only the standardized immediate child directories `ffi/`, `system/`, `advanced/`, `dynamicvmrt/`, and `operators/` are permitted.
- because every special kind has one fixed directory name, an application can contain at most one application-local source library of each kind.
- `ffi/`, `system/`, `advanced/`, `dynamicvmrt/`, and `operators/` are source-domain/library roots, not packages, and their names do not become package namespace components.
- `srclib/ffi/`, `srclib/system/`, `srclib/advanced/`, and `srclib/dynamicvmrt/`, when present, must each contain exactly one direct file named `library.fol`; every other entry must be an internal package directory.
- `library.fol` is a reserved structural filename. It does not create a library named `library`; the fixed enclosing directory determines both the application-local source-library identity and its kind.
- no nested `library.fol` is permitted. Nested source-library boundaries are forbidden.
- implementation package paths begin below the source-library root and may be arbitrarily deep. For example, `srclib/ffi/native/marshal/` is internal package `native.marshal` of the FFI source library.
- source-library implementation packages never enter the application's package index and cannot be bypass-imported as application packages.
- `srclib/operators/`, when present, must contain exactly one file named `library.fol`, with no additional files or subdirectories.
- for every standardized child of `srclib/`, `library.fol` is the fixed structural surface filename. In the `operators/` slot it is parsed as the operator bootstrap surface rather than as an importable API surface.
- `srclib/`, `operators/`, and `library.fol` create no package namespace.

`lib/` rules:

- `lib/` contains zero or more compiled FoLang library artifacts named `*.folenc`.
- `.folenc` is the standard Protocol Buffers binary library artifact format.
- no alternate non-binary compiled-library artifact format is defined.
- `.fol` source files are invalid in `lib/`, and `lib/` never participates in source discovery.
- multiple separately built FFI, system, advanced, dynamic-runtime, or application libraries may coexist in `lib/`; the one-per-kind rule applies only to application-local source libraries under `srclib/`.

`build/` rules:

- `build/` contains compiler-generated output and never participates in source discovery.
- the frontend always emits its validated external representation as Protocol Buffers binary under the project-root `build/` directory.
- `.fol` source files placed under `build/` are invalid project layout.

#### Structural Surface and Metadata Filenames

FoLang standardizes structural filenames so source context is determined without arbitrary surface-file naming:

| Structural file | Valid location | Meaning | Does the containing directory define a package? |
|---|---|---|---|
| `<entryfilename>.fol` | `src/<entryfilename>.fol` | application entry surface | no; `src/` is not a package |
| `library.fol` | `srclib/ffi/`, `srclib/system/`, `srclib/advanced/`, `srclib/dynamicvmrt/` | application-local source-library API surface | no; each fixed `<kind>/` directory is a library root, not a package |
| `library.fol` | `srclib/operators/` | operator bootstrap surface | no; `operators/` is not a package and permits no subdirectories |
| `<surfacefilename>.fol` | standalone packaged-library project root | packaged-library API surface; kind declared explicitly by `@co.dap.library(type=...)` | no; project root is not a package |
| `package.fol` | inside an ordinary package directory | package metadata/aliasing | yes; the directory is already the package |

`app.fol`, `library.fol`, and `package.fol` are structural filenames. They do not derive declarations named `App`, `Library`, or `Package`. `package.fol` does not create its enclosing package; it only supplies package metadata. `library.fol` does not choose an arbitrary library name. In an application-local source library, the standardized enclosing `srclib/<kind>/` slot supplies identity and kind; in a standalone packaged-library project, the project supplies library identity and `@co.dap.library(type=...)` supplies the kind.

---

## Package Aliasing

If there is a folder /appl/src/hr/empl and under that there is a fol file called Employee.fol then the import statement as we know will be

`@co.ddap.import(package="hr.empl", as="emp")` where `as` is not a mandatory attribute

> An import names a **package**, never a declaration inside it. The package is the folder, so `Employee.fol` under `/appl/src/hr/empl` belongs to package `hr.empl`; the file name is not part of the path. Once the package is imported, the declaration is reached as `emp.Employee`.

Now we want to change empl to emp, simple way is `change the folder name`, but we want to keep the `physical folder name` as is.

For example, `/appl/src/hr/empl` may be given the logical package name `hr.emp` while the physical folder remains `/appl/src/hr/empl`.

```folang
// /appl/src/hr/empl/package.fol
_ co.lang.package = {
    name: "emp"
};
```

`package.fol` is a reserved source filename. At most one may exist in a package folder, and its only top-level declaration must be `_ co.lang.package`. In this special source form, `_` occupies the declaration-name position but does not derive a package named `package` from the filename. The required `name` member supplies the logical name of the current package segment. Parent package segments are unchanged.

The import will be as below:

`@co.ddap.import(package="hr.emp", as="emp")` 

> Note:  This is a **Planned** Feature not finalized to be part of initial release.
   
---

## UDT (User defined Data types)

FoLang provides the following user-defined data declaration kinds:

1. cstructs
2. structs
3. unions
4. enums
5. classes
6. modules
7. interfaces
8. signatures

Each ordinary file-backed primary declaration is placed in its own `<Name>.fol` file. The declaration name is never repeated in source; `_` occupies the declaration-name position and the compiler derives the name from the filename.

> For more information about UDTs, see [Built In Kinds](#builtin-kinds).

---

### Struct Declaration

```folang
// Employee.fol
_ co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}
```

> More about structs: [`Structs in detail`](#structs).

---

### C-Struct Declaration

`co.lang.cstruct` is a C-like value type: it is passed by value, has a simple memory layout, and is safe to cross supported ABI boundaries.

```folang
// Point.fol
_ co.lang.cstruct = {
    x co.lang.int;
    y co.lang.int;
}
```

```folang
// Rect.fol
_ co.lang.cstruct = {
    origin Point;
    width  co.lang.int;
    height co.lang.int;
}
```

---

### Enum Declaration

```folang
// Status.fol
_ co.lang.enum = {
    Active,
    Inactive
}
```

---

### Union Declaration

```folang
// NumberOrText.fol
_ co.lang.union = {
    intValue co.lang.int;
    strValue co.lang.string;
}
```

---

### Class Declaration

```folang
// Employee.fol
_ co.lang.class = {
    getEmployeeDetails()->(Employee) = empmodule.getEmployeeDetails;

    getEmployeeInfo()->(Employee) =>> empmodule.getEmployeeDetails();
    // delegating — internally redirecting the call to module function
}

// $1, $2, $3 ... are previous results captured as bind variables
//Emp.fol
_ co.lang.class={
    dosomething(a co.lang.int, b co.lang.int)->(co.lang.int)=>>somePack.someMethod(a)=>>someOthPack.someOtherMeth($1, b);
}
```

> More about classes: [`Classes in detail`](#classes).

---

### Interface vs Signature

```folang
// EmployeeContract.fol
_ co.lang.signature = {
    ...
}
```

> More about signatures: [`signatures in detail`](#signatures).

```folang
// EmployeeApi.fol
_ co.lang.interface = {
    ...
}
```

> More about interfaces: [`interfaces in detail`](#interfaces).

---

### Module Declaration

```folang
// Employee.fol
_ co.lang.struct = {
    ...
}
```

```folang
// EmployeeModule.fol
_ co.lang.signature = {
    ...
}
```

```folang
// EmployeeModImpl.fol
@co.dap.module(signature=EmployeeModule)
_ co.lang.module->(
    signature=EmployeeModule,
    matches=EmployeeModule
) = {
    ...
}
```

> More about modules: [`Modules in detail`](#modules).

---

## Units

A unit is a stateless source container. A package may contain any number of ordinary unit files, and all their members are consolidated directly into the package namespace.

```text
arithmetic.unit.fol
conversion.unit.fol
optional.unit.fol
```

Each ordinary unit file contains:

```folang
_ co.lang.unit = {
    ...
}
```

The filename is organizational only. It does not create a unit symbol or another qualification level. For package `math`, a function `abs` declared in any ordinary unit is accessed as `math.abs(...)`, not `math.Arithmetic.abs(...)`.

A companion unit uses the reserved filename form `<StructName>.comp.unit.fol`. Its members attach to the matching same-package struct:

```text
Employee.fol
Employee.comp.unit.fol
```

Only `co.lang.struct` supports a companion unit. See [`Units in detail`](#units-in-detail) and [Struct Companion Units](#struct-companion-units).

## Matchers

### Custom Matcher

```folang
// PositiveEvenMatcher.fol
@co.dap.matcher
_ co.lang.matcher->(type=co.lang.int) = {
    matchCase(
        value   co.lang.int,
        pattern co.lang.untyped
    )->(co.lang.int, co.lang.MatchBindings) = {
        // user logic
        // 0 = no match, >0 = match
    }
}
```

A matcher declaration supports exactly one matched-subject type, specified by `type=`. It must declare exactly one protocol function named `matchCase`; overloading `matchCase` inside a matcher is not permitted. A matcher for another subject type must be declared as a separate `<Name>.fol` primary declaration.

At compile time, the compiler resolves the type named by `type=` and the type of the first `matchCase` parameter. The two resolved types must be equivalent. Type aliases are compared after alias resolution. The second parameter must have type `co.lang.untyped`, and the result must be exactly `(co.lang.int, co.lang.MatchBindings)`.

```folang
// InvalidMatcher.fol
_ co.lang.matcher->(type=co.lang.int) = {
    matchCase(
        value   co.lang.string,
        pattern co.lang.untyped
    )->(co.lang.int, co.lang.MatchBindings) = {
        ...
    }
    // compiler error: declared matcher type and first parameter type differ
}
```

The parameter names themselves are not significant; their order and resolved types define the protocol shape.

---
<a id="comprehensions"></a>

## Comprehensions *(planned)*

```folang
k := (1 .. 10).filter(|x| => x % 2 == 0).map(|x| => x * x);

result := for (x <- List(1,2,3)).yield(x * 2);          // List(2, 4, 6)
result := for (x <- Set(1,2,3)).yield(x * 2);           // Set(2, 4, 6)
result := for (x <- Some(5)).yield(x * 2);              // Some(10)
result := for (x <- fetchData()).yield(x.process());    // Future

ages  := {"A":30,"B":40,"c":66,"e":88};
upper := for ((name, age) <- ages).yield(name.toUpperCase, age);
```

---

## Extensions

Extension functions may be declared in an ordinary package unit:

```folang
// string-extension.unit.fol
_ co.lang.unit = {

    @co.dap.extension(fortype=co.lang.string, what=extends)
    upperCase()->(string) = {
        this.return this.upper();
    }

    @co.dap.extension(fortype=[co.lang.string], what=overrides)
    equals(str co.lang.string)->(co.lang.bool) = {
        this.return this == str;
    }
}
```

Because ordinary unit filenames create no symbol, extension activation identifies the contributing package, not the unit file.

For the current package, omit `from`:

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

See [Activating Instance Methods](#activating-instance-methods) for activation of typeclass instances.

---
## Reflections
```folang
@co.dap.reflection(enable=True, package="co.meta")

x co.lang.int = 10;
x.reflect().getType();   //co.lang.int
x.reflect().getValue();  //10;
x.reflect().getKind();   // value
```

---

## Type Classes
## Monads, Applicatives, Functors, Monoids and Transformers

> `@co.dap.typeclass(kind=...)` is the single annotation for all typeclass definitions. `kind` specifies the algebraic structure — `Functor`, `Applicative`, `Monad`, `Monoid`, `Transformer`, or any user-defined kind. Instances of any typeclass always use `co.lang.instance`.

In a file-backed typeclass declaration, `_` is the filename-derived declaration-name placeholder and the following parenthesized clause declares the typeclass parameters. They are separate grammar components, so the canonical spelling includes a space: `_ (F(_))`, not `_(F(_))`. A parameter such as `T` denotes an ordinary type, while `F(_)` denotes a unary type constructor and `G(_, _)` denotes a binary type constructor. Otherwise-unbound type variables introduced in an operation signature, such as `A` and `B`, are implicitly universally quantified within that operation.

### Functor

```folang
//Functor.fol
@co.dap.typeclass(kind=Functor)
_ (F(_)) co.lang.typeclass = {
    map(value F(A), f (A)->B) -> (F(B));
}

// ListFunctor.fol
_ co.lang.instance->(for=Functor, type=List) = {
    map(value List(A), f (A)->B)->(List(B)) = {
        result = List(B){};
        value.each(_, item).do({ result.append(f(item)) });
        this.return result;
    }
}
```

### Applicative

```folang
//Applicative.fol
@co.dap.typeclass(kind=Applicative)
_ (F(_)) co.lang.typeclass = {
    pure(x A) -> (F(A));
    apply(fab F(A->B), fa F(A)) -> (F(B));
}

// OptionApplicative.fol
_ co.lang.instance->(for=Applicative, type=Option) = {
    pure(x A)->(Option(A)) = { this.return Some(x); }
    apply(fab Option(A->B), fa Option(A))->(Option(B)) = {
        this.return (fab, fa)
            .match
            .case((Some(f), Some(x)) => Some(f(x)))
            .default(None());
    }
}
```

### Monad

```folang
//Monad.fol
@co.dap.typeclass(kind=Monad)
_ (F(_)) co.lang.typeclass = {
    pure(x A) -> (F(A));
    flatMap(fa F(A), f (A)->F(B)) -> (F(B));
}

// OptionMonad.fol
_ co.lang.instance->(for=Monad, type=Option) = {
    pure(x A)->(Option(A)) = { this.return Some(x); }
    flatMap(fa Option(A), f (A)->Option(B))->(Option(B)) = {
        this.return fa.match().case(Some(x) => f(x)).default(None);
    }
}
```

### Monoid

```folang
//Monoid.fol
@co.dap.typeclass(kind=Monoid)
_ (T) co.lang.typeclass = {
    empty() -> (T);
    combine(a T, b T) -> (T);
}

// IntMonoid.fol
_ co.lang.instance->(for=Monoid, type=co.lang.int) = {
    empty()->(co.lang.int) = { this.return 0; }
    combine(a co.lang.int, b co.lang.int)->(co.lang.int) = { this.return a + b; }
}
```

### Transformer

```folang
//Transformer.fol
@co.dap.typeclass(kind=Transformer)
_ (F(_), G(_)) co.lang.typeclass = {
    map(value F(A), f (A)->B) -> (G(B));
}

// ListToSetTransformer.fol
_ co.lang.instance->(for=Transformer, types=[List, Set]) = {
    map(value List(A), f (A)->B)->(Set(B)) = {
        result = Set(B){};
        value.each(_, item).do({ result.insert(f(item)) });
        this.return result;
    }
}
```

---



### Using an Instance

An instance is selected **by name**. There is no implicit search.

```folang
@co.ddap.import(package="abc.tc", as="tc")

xs List(co.lang.int) = [1, 2, 3];
double(x co.lang.int)->(co.lang.int) = { this.return x * 2; }

ys := tc.ListFunctor.map(xs, double);
```

`map` takes the container as its first argument, exactly as the typeclass
declares it. The call names `ListFunctor`, so the compiler resolves it the same
way it resolves any other imported declaration.

FoLang does **not** search visible packages for an instance that happens to
match a type. A typeclass is a contract, an instance is a named implementation
of that contract, and the caller names the one it means. Nothing is inferred,
so an unresolved call reports a name the developer actually wrote rather than a
failed search they never saw.

> Method syntax such as `xs.map(f)` is unrelated to typeclasses. It calls an
> associated function declared in a companion unit for the type. Companion
> units live in the type's own package, so method calls are unambiguous by
> construction. A companion function and a typeclass instance may both be
> named `map`; they are different declarations and are reached differently.

### Activating Instance Methods

An instance may also be **activated**, which makes its functions callable as
methods on the receiver. Activation uses the same directive that activates
extensions.

```folang
@co.ddap.import(package="abc.tc", as="tc")
@co.ddap.use(from="tc.ListFunctor", methods=[map, reduce])

ys := xs.map(double);        // resolves to tc.ListFunctor.map(xs, double)
```

Activation is explicit and block-scoped. Importing a package does **not**
activate anything in it, so adding an import can never change how an existing
call resolves.

#### `methods` and `from`

`methods` selects the functions to activate.

For extension functions contributed by ordinary package units:

- omit `from` to select the current package
- use an imported package alias to select another package
- use the complete package path when no alias exists

```folang
@co.ddap.use(methods=[upperCase]);               // current package
@co.ddap.use(from="tu", methods=[upperCase]);   // imported package alias
@co.ddap.use(from="text.util", methods=[upperCase]); // complete package path
```

Ordinary unit filenames are not accepted by `from`, because they do not create symbols.

For a typeclass instance, `from` continues to name the instance declaration:

```folang
@co.ddap.use(from="tc.ListFunctor", methods=[map, reduce]);
```

Listing names is optional. Omit `methods` to activate every eligible method from the selected package or instance; provide it to activate a subset. Conflict detection remains receiver-aware and block-scoped.

#### How a method call resolves

For `xs.map(f)`, where `xs` has type `List(A)`:

1. a class method or companion-unit function on `List`
2. an activated extension for `List`
3. an activated instance function whose typeclass declares `map` with the
   receiver as its first parameter
4. otherwise, an error

The first match wins. A type's own declarations therefore always take
precedence over anything activated into scope, and no activation can silently
replace behaviour the type already defines.

Within one scope a given method name may be activated at most once for a given
receiver type. Activating `map` for `List` from two sources is an error at the
second `@co.ddap.use`, which names both. The conflict is reported where the
activation is written, never at a distant call site.

The parser's built-in-method classification is only a lookup candidate. If the
frontend recognizes a control-chain shape before receiver-aware lookup, it
must retain the complete original member/call chain. A receiver-owned,
companion, extension, or activated-instance declaration that wins the order
above restores/remains the ordinary method call represented by that chain. An
early dedicated control-flow node is therefore only a lowering candidate and
becomes final only when the built-in meaning wins.

Activation never affects generic code. A function polymorphic over a typeclass
takes the instance as an ordinary parameter and calls it by name.

### A Typeclass Is a Type

A typeclass may be used wherever a type is expected. Its values are its
instances. This is the same relationship a signature has to a module:

```folang
mm EmployeeModule = EmployeeModImpl;     // signature as type, module as value
f  Functor(List)  = tc.ListFunctor;      // typeclass as type, instance as value
```

An instance is therefore an ordinary first-class value. It can be held in a
variable, stored in a collection, returned from a function, and chosen at run
time.

```folang
inst := (useCache).return(cachedFunctor).otherwise.return(plainFunctor);
result := inst.map(xs, transform);
```

Nothing about an instance is special to the compiler, which is why FoLang needs
no instance resolution algorithm and no coherence rules.

### Writing Code Generic Over a Typeclass

Activation makes `xs.map(f)` work for a **known** container. Code that must work
for **any** container cannot activate anything, because the container type is
not known until the caller supplies it. Such code takes the instance as an
ordinary parameter.

```folang
@co.dap.generic(type={F:{typename}, A:{typename}, B:{typename}})
mapAll(inst Functor(F), value F(A), fn (A)->B)->(F(B)) = {
    this.return inst.map(value, fn);
}
```

| Parameter | What it is |
|---|---|
| `F` | the container kind — `List`, `Option`, `Tree` |
| `inst` | the instance, an implementation of `Functor` for `F` |
| `value` | the container itself, an ordinary value |

The caller supplies the instance:

```folang
ys   := mapAll(tc.ListFunctor,   xs,  double);
opt2 := mapAll(tc.OptionFunctor, opt, double);
```

The function never learns what `F` is. It knows only that `inst` provides `map`,
which is the whole contract. One definition therefore serves every container
that has a `Functor` instance.

> A type is never "a Functor" in FoLang. `List` does not become a Functor; an
> instance implements Functor operations *for* `List`, and the list stays a
> plain list. The Functor-ness lives entirely in `inst`.

#### When a wrapper is worth writing

`mapAll` above forwards directly to `inst.map`, so it adds nothing a caller
could not write themselves. A wrapper earns its place when it does more than
forward — when it combines several typeclass operations, adds logic, or fixes
some parameters and leaves others open.

```folang
@co.dap.generic(type={F:{typename}})
doubleAll(inst Functor(F), value F(co.lang.int))->(F(co.lang.int)) = {
    this.return inst.map(value, (x co.lang.int)->(co.lang.int) = {
        this.return x * 2;
    });
}
```

This one fixes the element type and the operation, so it says something a bare
`inst.map` call does not.

### Where an Instance Is Declared

An instance is declared in **the package that defines the typeclass**, or in
**the package that defines the type**. That exact package, not a sub-package.

```folang
abc.tc.ListFunctor      for=Functor, type=List           // OK  typeclass's package
myapp.ab.TreeFunctor    for=Functor, type=myapp.ab.Tree  // OK  type's package
other.util.ListFunctor  for=Functor, type=List           // ERR neither is theirs
```

A typeclass is an ordinary declaration and may live in any package; `abc.tc`
above is a user package, not a built-in one. Sub-packages are distinct
packages, so an instance for `myapp.ab.Tree` belongs in `myapp.ab`, not in
`myapp` or `myapp.ab.instances`. This matches the rule for companion units,
which also sit in their type's own package. Because a package spans every `.fol` file in its folder, each instance still gets its own `<InstanceName>.fol` file and uses `_ co.lang.instance` in source.

The rule is permissive on both sides on purpose. Requiring the typeclass's
package alone would make a typeclass usable only by its own author, since
nobody could implement it for their own types. Requiring the type's package
alone would stop a library from shipping instances for common built-in types.

**For library authors.** If you define a type and want it usable with someone
else's typeclass, declare the instance in your package. If you define a
typeclass, you may ship instances for types you do not own, including built-in
ones — `IntMonoid` above sits beside `Monoid`, which is exactly this case. If
you need an instance for a typeclass and a type you both do not own, wrap the
type in a `co.lang.newtype` you do own and declare the instance for the wrapper.

This placement rule is semantic, not syntactic. A misplaced instance parses
correctly and is reported during name resolution, so the diagnostic can name
the typeclass, the type, and the two packages in which the instance would have
been legal.

---


## Labels and Named Blocks

```folang


// Label
outer:{
    // statements
}

//Blocks Anonymous block
 {
    // statements
 }

// Named Block
labelBlock co.lang.block={

}

labelBlock.expand();
```
> Blocks have their own scope and context for variables, a variable pre declarred outside the block will be accessible in side the block, a block can have its own variable with same name and different type or same type which overrides/shadows parent or outer blocks variables, and the scope of such variables are limited to that block it is very similar to C/C++

> Blocks cannot live outside functions they must be inside functions or methods only

> Inner blocks for class, struct, typeclass, module or anyother consttruct other than functions/methods are prohibited. // throws compiler error

```folang
somefun (a co.lang.int, b co.lang.int)->(co.lang.int)={

     some_other co.lang.float =  20.1f;
     {
        some_other co.lang.char = 'c';
        co.out.println( some_other);   //prints c
     }

     co.out.println(some_other); //prints 20.1;

     {
        some_other co.lang.float = 11.1f;
        co.out.println(some_other); // prints 11.1f
     }
      
     co.out.println(some_other); // still prints 20.1f;

     {
        some_other = some_other + 1.1f;
        co.out.println(some_other); // prints 21.2
     }

     co.out.println(some_other); // prints 21.2; as this was changed in third block
}
```
---



## imports
FoLang supports three import forms. User packages and libraries must be imported before use. When an import declares `as=`, symbols are accessed through that alias. When `as=` is omitted, the complete imported package or library path is used.

### 1. Normal Package Import

Use this for ordinary project packages.

```folang
@co.ddap.import(package="hr.employee", as="emp")

e := emp.getEmployee(1);
co.out.println(e.name);
```

Resolution:

```text
package="hr.employee" -> /appl/src/hr/employee/
```

### 2. Source Library Import

Use this for same-owner application-workspace libraries whose source is available under the reserved `srclib/` root.

```folang
@co.ddap.import(package="ffi", src-library=true, as="ffilib")
```

Resolution:

```text
package="ffi", src-library=true
    -> /appl/srclib/ffi/library.fol
```

For `src-library=true`, the `package` value selects one standardized application-local source-library slot under `srclib/`. The permitted values are `ffi`, `system`, `advanced`, and `dynamicvmrt`. These names are fixed by FoLang and identify both the source library and its library kind.

The resolved file must be the fixed source-library surface file:

```folang
// /appl/srclib/ffi/library.fol
_ co.lang.library={

}
```

Meaning:

- `package` selects one of the standardized application-local source-library slots under `srclib/`
- `src-library=true` switches lookup from the application package index to the source-library index
- `library.fol` is the mandatory and only direct source file at that source-library root
- the enclosing fixed directory determines the library kind; a `@co.dap.library(type=...)` annotation is neither required nor permitted on an application-local source-library surface
- only the projected surface API is visible to the application
- implementation packages beneath the source-library root are private to that source-library compilation domain and cannot be imported through ordinary package imports

### 3. Packaged Library Import

Use this for third-party or prebuilt libraries.

```folang
@co.ddap.import(library="hrlib", as="hr")
@co.ddap.import(library="paylib", as="pay")
```

Resolution:

```text
library="hrlib" -> /appl/lib/hrlib.folenc
```

Only the packaged library's projected surface API is visible to the consumer.

---

## Import Directive Fields

| Field | Required | Default | Meaning |
|---|---|---|---|
| `package` or `library`| one required | — | logical package path, standardized source-library slot when `src-library=true`, or packaged `.folenc` library name |
| `src-library` | ❌ | `false` | when `true`, `package=` selects `ffi`, `system`, `advanced`, or `dynamicvmrt` under reserved `srclib/` and resolves to its `library.fol` surface |
| `as` | ❌ | none — full dot path required when omitted | local alias; valid FoLang identifier |

Notes:

- `as` is optional — when omitted, no short alias is created and the full imported package path must be used to access symbols
- dots are not allowed in `as`
- for an application-local source library, the fixed `srclib/<kind>/` path is the source of truth for library kind
- `@co.dap.library(type=...)` is not used on application-local `library.fol` surfaces
- packaged libraries may retain kind metadata in their compiled `.folenc` projection

Examples:

```folang
@co.ddap.import(package="hr.employee", as="emp")
em emp.Employee;

@co.ddap.import(package="hr.employee")
em hr.employee.Employee;
```

### Valid `as` Values

```text
as="hr"       ✅
as="v1_hr"    ✅
as="v1.hr"    ❌
as="123hr"    ❌
```

---

#### Cycles

Compiler error if any cycle exists through:

- package imports

Examples:

- `packageA` imports `packageB`, and `packageB` imports `packageA`

---

### Symbol Resolution

#### Resolution Order

For symbols without a prefix:

1. current package symbol table
2. parent package chain upward
3. error if not found

For symbols with an alias prefix:

1. current package imported-context map
2. parent package chain imported-context maps
3. error if not found

For imports declared **without `as`**:

1. use the full imported package path directly
2. resolve it through imported-context lookup using that full path
3. error if not found

#### Example Context Graph

```text
pp1 (grandparent package)
    ├── SymbolTable
    ├── ImportedContexts
    └── ChildCtxs

p1 (parent package)
    ├── Parent -> &pp1
    ├── SymbolTable
    ├── ImportedContexts
    └── ChildCtxs

c1 (current package)
    ├── Parent -> &p1
    ├── SymbolTable
    ├── ImportedContexts
    └── ChildCtxs
```

#### Resolution Examples

```text
lookup "OwnType"
    -> current symbol table

lookup "ParentType"
    -> parent chain

lookup "hr.Employee"
    -> imported-context lookup for alias "hr"

lookup "hr.employee.Employee"
    -> imported-context lookup for full imported package path when no alias exists

lookup "unknown.Type"
    -> imported-context lookup fails -> compiler error
```

---

### Short Summary

- the application root contains the four fixed domains `src/`, `srclib/`, `lib/`, and `build/`; none is a package
- `src/` contains exactly one direct file, `app.fol`; all other application source lives in package subdirectories below `src/`
- package dot paths begin below `src/`
- `srclib/` permits only `ffi/`, `system/`, `advanced/`, `dynamicvmrt/`, and `operators/` as immediate children
- the `ffi`, `system`, `advanced`, and `dynamicvmrt` source-library roots are not packages and each contains exactly one direct file named `library.fol`; their remaining source lives in arbitrarily deep internal package subdirectories
- the fixed `srclib/<kind>/` directory determines the application-local source-library identity and kind; no `@co.dap.library(type=...)` annotation is used there
- `srclib/operators/` contains exactly one `library.fol`, has no subdirectories, and creates no package namespace
- application-local source-library internals never enter the application's package index
- at most one application-local source library exists for each standardized kind; additional independently built libraries of any kind are consumed as `.folenc` artifacts through `lib/`
- `lib/` contains zero or more Protocol Buffers binary `.folenc` libraries; no alternate non-binary compiled-library format is defined
- `build/` contains generated output; the frontend external output is always Protocol Buffers binary
- each ordinary package source file has exactly one primary top-level declaration
- package functions, templates, macros, and non-UDT type declarations must be enclosed in ordinary `*.unit.fol` files
- all ordinary unit members are consolidated directly into the package namespace
- struct companion behavior must be declared in `<StructName>.comp.unit.fol`
- the application entry file is `src/<entryfilename>.fol`, an executable non-package context with its own restricted declaration rules
- `co.*` is always available and never imported
- `@co.ddap.import(package="...")` imports normal packages
- `@co.ddap.import(package="ffi|system|advanced|dynamicvmrt", src-library=true, ...)` imports the corresponding application-local source library through its fixed `library.fol` surface
- `@co.ddap.import(library="...")` imports a packaged `.folenc` library from `lib/`
- every ordinary library surface exports only permitted boundary contracts and public function signatures; implementation packages remain hidden

---
## Let and Function Patterns

##### Bare Function-Pattern Group

A bare function-pattern group does not capture surrounding runtime bindings:

```folang
classify(0) => { this.return "zero"; }
classify(n).where(n > 0) => { this.return "positive"; }
classify(_) => { this.return "negative"; }
```

The formal clause shape is:

```text
name(patterns) [.where(guard)] => result
```

`=>` is mandatory. `=>>` belongs to ordinary function delegation, `==>>` belongs to closure/curry expression and are not accepted here.

A bare group may use:

- its parameters and names introduced by its patterns
- its own name for recursion
- entry-local type names
- `co.*` APIs
- imported package and library APIs
- compile-time symbols that do not require runtime capture

It may not reference an entry-file runtime variable from the surrounding context:

```folang
offset := 100;

adjust(n) => { this.return  n + offset; }
// compiler error: bare function-pattern groups cannot capture `offset`
```

##### Capturing `let` Function-Pattern Group

The `let` form exists only to declare an entry-local function-pattern group that captures one or more surrounding entry-file runtime bindings:

```folang
offset := 100;

let adjust(0) = offset;
let adjust(n) = n + offset;

result := adjust(10);
```

The captured names must resolve to surrounding runtime bindings that are already declared and definitely initialized before the first clause of the group. Built-in names, imported declarations, type names, parameters, and the function's own recursive name are not captures.

A `let` function-pattern group must capture at least one surrounding runtime binding. When no capture is required, the bare form must be used:

```folang
let fib(0) = 1;
let fib(1) = 1;
let fib(n) = fib(n - 1) + fib(n - 2);
// compiler error: this group captures nothing; remove `let`

fib(0) => { this.return 1; }
fib(1) => { this.return 1; }
fib(n) => { this.return fib(n - 1) + fib(n - 2); }
```

The `let` marker is therefore not an optional spelling of a bare function-pattern group. It explicitly requests restricted lexical capture.

##### Similarities

Bare and capturing `let` function-pattern groups both:

- group compatible clauses by function name and arity
- dispatch by parameter patterns and optional `.where(...)` guards
- evaluate clauses in normal pattern-selection order
- may call themselves recursively
- must maintain compatible parameter and result types across all clauses
- follow the normal pattern ordering, reachability, overlap, and exhaustiveness rules
- are private to the entry file
- cannot be imported, exported, or annotated with package visibility
- cannot be converted to, assigned as, passed as, or returned as function values
- cannot be partially applied or curried

##### Clause Syntax and Termination

Both forms accept the same pattern list and optional guard:

```folang
classify(0) => "zero";
classify(n).where(n > 0) => "positive";
classify(_) => { this.return "negative"; }

offset := 10;
let adjust(0) = offset;
let adjust(n).where(n > 0) = n + offset;
let adjust(_) = { this.return offset; }
```

Annotations may precede either clause form and are retained on the clause for
later checking. Because every function-pattern group is private to the entry
file, package visibility/export annotations are invalid even though other
entry-local metadata annotations may be used.

An expression-bodied clause is a simple statement and must end with `;`.
A block-bodied clause ends at its closing `}` and must not be followed by `;`.
Newlines never terminate clauses.

Supported parameter patterns are:

- `_` wildcard patterns;
- literals, including signed numeric literals;
- binding names;
- qualified identity names;
- constructor patterns such as `Some(value)`;
- record patterns such as `Employee{id: value, name: _}`;
- tuple patterns with at least two elements, such as `(left, right)`.

`.where(expression)` is evaluated only after the clause's parameter patterns
match. Names bound by those patterns are available to the guard and result.
The guard expression must have type `co.lang.bool`.

Clauses are considered in source order. A clause is eligible when its arity
matches, all parameter patterns match, and its optional guard evaluates to
`co.const.true`. The compiler rejects incompatible arities or result types,
unreachable clauses, invalid overlaps, and non-exhaustive groups where
exhaustiveness is required by the declared result contract.

##### Differences

| Form | Surrounding runtime capture | Intended use |
|---|---:|---|
| `name(pattern) => result` | No | Entry-local pattern dispatch that depends only on its arguments, recursion, built-ins, imports, and compile-time names |
| `let name(pattern) = result` | Yes, at least one capture required | Entry-local pattern dispatch that also depends on existing entry-file runtime bindings |

Clauses of the same name and arity must all use the same form. Mixing bare and `let` clauses in one function-pattern group is a compiler error.

A capturing `let` function-pattern group is still not a general closure facility. It is named, entry-local, non-first-class, and non-escaping. The compiler may internally retain a capture environment, but FoLang does not expose that environment as a closure object.

`let` cannot be used directly in the entry file for value bindings, anonymous functions, general closure expressions, or curried functions:

```folang
let base = 10;                   // compiler error: entry-file let value binding
let result = base + 1;           // compiler error: entry-file let value binding
let add = (a, b) => a + b;       // compiler error: anonymous function
let counter = closure { ... };   // compiler error: general closure expression
let add = a => b => a + b;       // compiler error: curried function
```

    The compiler may lower either function-pattern form to a private entry helper, but neither source construct is a general-purpose function declaration.
---

## Package in detail 

### Package Identity

Ordinary application packages exist only in subdirectories below the reserved `src/` domain. `src/` itself is not a package and contains exactly one direct file, `app.fol`. No other direct file is permitted there.

- package dot paths start below `src/`
- the application root and `src/` are not packages
- `srclib/`, `lib/`, and `build/` are separate reserved domains and are excluded from application package discovery
- source-library roots under `srclib/` are not packages; only their descendant implementation directories define packages within that source-library compilation domain

Examples:

```text
/appl/src/hr/           -> package "hr"
/appl/src/hr/employee/  -> package "hr.employee"
/appl/src/auth/         -> package "auth"
```

### Multi-File Packages

Multiple `.fol` files in the same package folder below `src/` automatically belong to the same package:

```text
src/hr/employee/
├── Employee.fol      -> hr.employee
├── EmpService.fol    -> hr.employee
└── EmpValidator.fol  -> hr.employee
```

---

### Application Project Layout

See the canonical [Application Project Layout](#application-project-layout). The key ownership boundaries are:

```text
/appl/
├── src/                  -> app.fol + application package directories
├── srclib/               -> standardized application-local special source libraries
│   ├── ffi/              -> library.fol + internal package directories
│   ├── system/           -> library.fol + internal package directories
│   ├── advanced/         -> library.fol + internal package directories
│   ├── dynamicvmrt/      -> library.fol + internal package directories
│   └── operators/        -> library.fol only
├── lib/                  -> zero or more *.folenc Protocol Buffers binary libraries
└── build/                -> generated compiler output
```

Only paths below `src/` enter the application package index. Package discovery inside a source library begins only below its fixed `srclib/<kind>/` root and remains private to that library.

---

### Package Access Rules

Four access levels control visibility across package boundaries:

| Annotation | Same Package | Parent | Sub-Package | Unrelated | Entry / Surface |
|---|---|---|---|---|---|
| `@co.dap.public` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `@co.dap.package` | ✅ | ✅ | ✅ | ❌ | ❌ |
| `@co.dap.protected` | ✅ | ❌ | ✅ | ❌ | ❌ |
| `@co.dap.private` | ✅ | ❌ | ❌ | ❌ | ❌ |

Meaning:

- `@co.dap.public` -> visible everywhere
- `@co.dap.package` -> visible across the package family
- `@co.dap.protected` -> visible downward into subpackages only
- `@co.dap.private` -> visible only inside the declaring package

Example:

```folang
// hr/employee-access.unit.fol — package "hr"

@co.dap.public
_ co.lang.unit = {

    @co.dap.public
    getEmployee(id co.lang.int)->(Employee) = { ... }     // anyone can call

    @co.dap.package
    validateId(id co.lang.int)->(co.lang.bool) = { ... }  // hr package family

    @co.dap.protected
    baseQuery()->(co.lang.string) = { ... }               // visible to subpackages

    @co.dap.private
    normalizeId(id co.lang.int)->(co.lang.int) = { ... }  // private declaration
}
```

---
## `co.*` Paths and Aliases

### `co.*` Is Always Available

All `co.*` paths are part of the language and are always in scope.
They are never imported through `@co.ddap.import`.

```folang
co.out.println("hello");
co.in.readln();
x co.lang.int = 42;
```

Being **in scope** does not automatically mean **permitted in every context**.

A package rule, library kind, or other compiler restriction may still forbid use of a particular `co.*` facility. In such cases the name is still known to the compiler, but its use is rejected at compile time.

### `@co.ddap.alias`

Use aliases only to shorten `co.*` paths.

```folang
@co.ddap.alias(co.out, as="out")
@co.ddap.alias(co.core.list, as="list")
@co.ddap.alias(co.encoding, as="enc")

out.println("hello");
list.of(1, 2, 3);
enc.json.serialize(data);

// full form still works alongside
co.out.println("world");
```

Rules:

- target must be a `co.*` path
- alias must be a valid identifier
- alias scope is file-local
- aliasing user packages is not allowed; use `as=` in `@co.ddap.import`
- duplicate aliases in the same file are compiler errors
- when no alias is declared, the complete `co.*` path is used
- declaring an alias does not disable or hide the complete `co.*` path
---

## Package Source Files

A package folder may contain three ordinary source-file categories plus the reserved `package.fol` metadata form. Structural surface names such as `library.fol` is not ordinary package-source filenames and are invalid inside a package. The compiler classifies ordinary package files from filenames before parsing, using the longest recognized suffix first:

```text
<Name>.comp.unit.fol  -> companion unit
<Fragment>.unit.fol   -> ordinary package unit
package.fol           -> reserved package metadata/alias form
<Name>.fol            -> file-backed primary declaration
```

Reserved structural filenames are recognized before the generic `<Name>.fol` rule, and a filename ending in `.comp.unit.fol` is never classified as an ordinary `.unit.fol` file.

### File-Backed Primary Declarations

A primary declaration file has the form `<Name>.fol` and contains exactly one primary top-level declaration. The source must use `_` in the declaration-name position:

```folang
// Employee.fol
_ co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}
```

The compiler derives the declaration name `Employee` from the filename. An explicit primary name is invalid:

```folang
// Employee.fol
Employee co.lang.struct = { ... }
// compiler error: file-backed primary declarations must use `_`
```

This rule applies to ordinary file-backed declarations that use a `<name> co.lang.<kind>` primary form, including classes, structs, cstructs, enums, unions, interfaces, signatures, modules, instances, matchers, objects, and other ordinary `co.lang.*` primary kinds. A `co.lang.library` declaration is not an ordinary `<Name>.fol` primary: it is valid only in the reserved `library.fol` surface form described in [Libraries](#libraries). Declaration families with a different surface grammar define their filename binding in their own sections.

The following declaration forms are stated exceptions and keep an explicit name in the head, because filename derivation cannot express what they need:

| Form | Why |
|---|---|
| struct and cstruct declared inside `library.fol` | one library surface may carry several boundary declarations |
| `co.lang.data` algebraic data type | the head names the variants |
| parameterized `co.lang.type` | a filename cannot carry `(T)` |
| type declarations in `src/<entryfilename>.fol` | the entry file is not file-backed |

File-level directives, imports, aliases, annotations, and decorators may appear before the primary declaration. They do not count as additional primary declarations.

FoLang permits the following top-level declaration kinds across ordinary package source files and their reserved special source forms:

1. struct
2. cstruct
3. class
4. module
5. signature
6. interface
7. enum
8. union
9. typeclass
10. instance
11. matcher
12. annotation or object declaration
13. package declaration
14. library declaration
15. unit declaration

Their source-file placement is part of the grammar:

- ordinary filename-backed primary declarations use `<Name>.fol`
- a package declaration uses the reserved `package.fol` source form
- an ordinary unit uses `<Fragment>.unit.fol`
- a companion unit uses `<StructName>.comp.unit.fol`
- a library declaration uses the reserved `library.fol` surface form; arbitrary `<Name>.fol` library-surface filenames are invalid

A source file is invalid when it contains multiple unrelated primary declarations, places a package declaration outside `package.fol`, places a unit declaration outside a recognized unit filename, places a library declaration outside the reserved `library.fol` surface form, or places project/library metadata outside its dedicated source form.

Package-level functions and non-UDT type declarations belong inside ordinary unit files.

### Filename Canonicalization

Filename-derived declaration identity is independent of filesystem case sensitivity. The compiler normalizes and case-folds the filename stem to construct the duplicate-detection and lookup key:

```text
canonical file key = caseFold(normalize(filename stem))
```

The language-level declaration is then represented using FoLang's canonical declaration spelling. Therefore:

```text
employee.fol
Employee.fol
EMPLOYEE.fol
```

all denote the canonical declaration name `Employee`. They conflict rather than declaring separate types, even on a case-sensitive filesystem.

The same rule applies to companion owners:

```text
employee.comp.unit.fol -> companion of Employee
```

This rule is specific to filename-derived primary declarations and companion-owner identity; it does not make every FoLang identifier case-insensitive. The compiler must not rely on operating-system filename comparison.

The filename stem must form a valid FoLang declaration identifier after normalization. Case variants and canonically equivalent spellings produce the same package-index key.

### Ordinary Package Units

An ordinary package-unit file has the form `<Fragment>.unit.fol`:

```folang
// arithmetic.unit.fol
_ co.lang.unit = {
    abs(value co.lang.int)->(co.lang.int) = {
        ...
    }
}
```

The fragment name is organizational only. It does not create a public `Arithmetic` symbol. All ordinary unit members in the package are merged into one package namespace:

```text
math/
├── arithmetic.unit.fol
├── comparison.unit.fol
└── optional.unit.fol
```

```folang
math.abs(-10);
math.max(10, 20);
value math.Option(co.lang.int);
```

The following qualification is invalid:

```folang
math.Arithmetic.abs(-10); // compiler error
```

Any number of ordinary unit files may exist in one package. During package indexing, the compiler shallow-parses their declarations, merges their members with filename-derived primary declarations, and reports duplicate names or duplicate function signatures according to FoLang overload rules.

### Companion Unit Files

A companion unit has the form `<StructName>.comp.unit.fol` and must contain `_ co.lang.unit`. The canonical owner name comes from the filename, not from the source body.

```text
Employee.fol
Employee.comp.unit.fol
```

The compiler can detect the following before parsing either body:

- duplicate primary declarations after filename canonicalization
- duplicate companion files for the same canonical owner
- an orphan companion file for which no same-package primary declaration exists

After the owner header is known, the compiler also verifies that the owner is a `co.lang.struct`. Other primary declaration kinds cannot own companion units.

### Package Indexing Levels

```text
Level 1: filename and package indexing
    classify primary, ordinary-unit, and companion-unit files
    canonicalize primary names and companion owners
    detect duplicate primary files
    detect duplicate or orphan companion files
    shallow-parse ordinary units
    merge ordinary-unit members into the package namespace
    report package-namespace conflicts

Level 2: companion validation
    parse companion declarations
    validate explicit receivers against the filename-derived owner
    merge companion members into the owner's companion namespace
    report duplicate companion members and receiver mismatches

Level 3: full semantic resolution
    resolve declaration bodies, expressions, calls, patterns, and types
```

This organization lets an imported source package build most of its symbol index cheaply before full parsing. Compiled packages may store the same index as metadata so imports need no source parsing.

## Application Entry File

The application entry file is the fixed a **special executable source form** `src/<entryfilename>.fol`. It is not a package, does not create an importable namespace, and is not subject to the ordinary package-file rule requiring exactly one primary declaration.

The compiler creates a dedicated **entry-file context** for it:

```text
ApplicationEntryContext
├── file directives, imports, and aliases
├── entry-local non-structural type declarations
├── non-capturing function-pattern groups
├── capturing `let` function-pattern groups
└── executable statements and expressions
```

Everything declared directly in this context is private to the entry file. Entry-local symbols cannot be imported, exported, or referenced by package or library source files.

#### Allowed Entry-File Constructs

The application entry file uses exactly the same grammar and allowed-construct rules described in [Single Source Application File](#single-source-application-file). The earlier section provides the developer-facing overview; this section defines the entry file's formal context, privacy, and dependency direction.

#### Entry-Local Function Patterns

Function-pattern groups are allowed as a special entry-file construct even though ordinary function declarations are forbidden. FoLang provides two entry-file forms with the same pattern-dispatch model but different capture semantics.

#### Entry-File Dependency Direction

The entry file may depend on packages and libraries, but packages and libraries may never depend on the entry context:

```text
application entry file
    ↓ uses
packages and libraries
```

This allows the entry file to coordinate application startup while preserving package and library independence.

---

## Libraries

### Library Project Layout

```
/hrlib/
├── library.fol                  ←  fixed packaged-library surface — @co.dap.library(type=...)
├── emp/                         package "emp" — internal, invisible to consumer
│   ├── Employee.fol
│   └── EmpService.fol
├── auth/                        package "auth" — internal, invisible to consumer
│   ├── Auth.fol
│   └── AuthService.fol
└── build/                       compiler-generated frontend output
    └── <frontend-artifact>      Protocol Buffers binary
```

Consumer only sees what `library.fol` declares. All source subfolders are internal. The root-level `build/` directory is generated output and is not an internal package.

---

### Library Surface file

FoLang uses surface files in two situations:

1. **Packaged library project surfaces**
2. **Application-workspace source library surfaces**

A library surface is the reserved `library.fol` source form containing `_ co.lang.library`. It defines the public boundary data contracts and boundary-adapter functions through which consumers call the library. The filename is structural and does not create a declaration named `library`. For an application-local source library, identity and kind come from its fixed `srclib/<kind>/` directory. For a standalone packaged-library project, `library.fol` is at the project root and must declare its kind explicitly with `@co.dap.library(type=...)` so the compiler can create the correct library surface context/symbol table and validate all kind-specific restrictions.

```text
src/<entryfilename>.fol -> application entry
library.fol -> packaged library surface
srclib/ffi/library.fol -> source library surface when imported with src-library=true
```

A library surface is not an ordinary package file. It may contain multiple boundary data declarations and public functions inside one `co.lang.library` declaration.

### Packaged Library Project Surface

A packaged library project has no application entry file. It has exactly one direct FoLang source file at the project root, and that fixed surface filename is `library.fol`.

```text
/hrlib/
├── library.fol                  <- fixed library surface; explicit project-level kind declaration
├── emp/                         <- internal package
│   ├── Employee.fol
│   └── EmployeeService.fol
├── auth/                        <- internal package
│   └── AuthService.fol
└── build/                       <- compiler-generated frontend output
    └── <frontend-artifact>      Protocol Buffers binary
```

Rules:

- exactly one direct FoLang source file named `library.fol` exists at the packaged-library project root
- `library.fol` must contain the project-level `@co.dap.library(type=...)` declaration followed by `_ co.lang.library = { ... }`
- the explicit library kind is required because a standalone packaged-library project has no enclosing application `srclib/<kind>/` path from which the compiler can infer the kind
- the compiler records the declared kind on the library surface context/symbol table before validating the surface, then applies the declaration, boundary-type, public-signature, adapter-body, capability, and other restricted-construct rules for that library kind
- a missing, unsupported, or conflicting standalone library-kind declaration is a compile-time error
- the supported packaged-library kinds are `application`, `dynamicvmrt`, `advanced`, `system`, and `ffi`; `operator` is not a packaged-library kind and operator metadata is never exported through `.folenc`
- the project is compiled into the standard packaged library artifact `.folenc`, which is Protocol Buffers binary
- consumers import it with `@co.ddap.import(library="...")`
- internal package folders are compiled into the library but are not directly visible to consumers
- the frontend output is always a Protocol Buffers binary artifact under the library project's root-level `build/` directory
- `build/` is generated output and never an internal package or source-discovery root

### Application-Workspace Source Library Surface

Application-local special source libraries live only under the fixed `srclib/` domain. `srclib/` itself is neither a package nor a library. FoLang standardizes the only permitted immediate children:

```text
/appl/srclib/
├── ffi/
│   ├── library.fol
│   └── <internal packages>/
├── system/
│   ├── library.fol
│   └── <internal packages>/
├── advanced/
│   ├── library.fol
│   └── <internal packages>/
├── dynamicvmrt/
│   ├── library.fol
│   └── <internal packages>/
└── operators/
    └── library.fol
```

For `ffi`, `system`, `advanced`, and `dynamicvmrt`:

- the immediate directory name is fixed by FoLang and determines both library identity and library kind;
- the directory itself is not a package;
- exactly one direct file is permitted: `library.fol`;
- `library.fol` contains `_ co.lang.library = { ... }` and does not use `@co.dap.library(type=...)`;
- all implementation source must reside in descendant package directories;
- descendant package names are relative to the source-library root and may be arbitrarily deep;
- no nested `library.fol` or nested source-library boundary is permitted;
- internal packages remain private to that source-library compilation domain and never enter the application package index;
- application code accesses the library only through the projected `library.fol` API surface.
- These libraries don't generate `.folenc` they are embedded into the application and built as single application protobuf file for backend.

Example:

```text
/appl/srclib/ffi/
├── library.fol                    <- FFI public surface; ffi/ is not a package
├── native/                        <- internal package native
│   └── marshal/                   <- internal package native.marshal
│       └── Marshal.fol
└── database/                      <- internal package database
    └── Postgres.fol
```

```folang
// /appl/srclib/ffi/library.fol
_ co.lang.library = {
    ...
}
```

Import:

```folang
@co.ddap.import(package="ffi", src-library=true, as="ffilib")
```

Resolution:

```text
package="ffi", src-library=true
    -> /appl/srclib/ffi/library.fol
```

The same structural rule applies to `system`, `advanced`, and `dynamicvmrt`.

The one-per-kind restriction applies only to **application-local source libraries**. Applications may consume any number of separately built libraries of any library kind as `.folenc` artifacts from `lib/`.

`srclib/operators/` is the special bootstrap slot described in [Operators](#operators). It uses the same fixed structural surface filename as every other `srclib/` slot, so it contains exactly `library.fol` and no package subdirectories. Its filesystem position selects the dedicated operator-source grammar; it is not an ordinary importable source-library API surface.

### Unified Surface Model

Every ordinary library kind uses the same conceptual surface shape. The `srclib/operators/` bootstrap slot is deliberately excluded because it is parser metadata rather than an importable library API:

```text
library surface
├── boundary data contracts
└── public boundary-adapter functions
```

Library kind changes the permitted boundary representation and transfer semantics, not the conceptual form of the API.

| Library kind | Boundary data | Transfer semantics |
|---|---|---|
| `application` | `co.lang.struct` | automatic deep snapshot |
| `dynamicvmrt` | `co.lang.struct` | automatic deep snapshot |
| `advanced` | `co.lang.struct` | automatic deep snapshot |
| `system` | `co.lang.cstruct` | system ABI value |
| `ffi` | `co.lang.cstruct` | C ABI value |
| `operator` | no boundary data| declaration of operators in unit|

`co.lang.struct` and `co.lang.cstruct` solve different problems:

```text
struct  -> FoLang semantic data contract
cstruct -> physical ABI-compatible value contract
```

Value transfer must not be confused with C-compatible layout. Application-family libraries therefore use expressive `struct` contracts transferred as deep snapshots, while system and FFI libraries use restricted `cstruct` contracts.

### Allowed Surface Declarations

A library surface may contain only:

- file- or library-level imports needed by its adapter implementations
- `co.lang.struct` boundary declarations for `application`, `dynamicvmrt`, and `advanced`
- `co.lang.cstruct` boundary declarations for `system` and `ffi`
- public free-function API declarations with boundary-adapter definitions

The following declaration kinds are forbidden directly in every library surface:

- classes
- modules
- interfaces
- signatures
- units and companion units
- associated functions
- operator functions
- enums, unions, and other ADTs
- type aliases, newtypes, and opaque types
- objects, instances, type classes, and dependent types
- macros, templates, annotations, and decorators
- global variables, pointers, references, addresses, or mutable surface state

Surface `struct` and `cstruct` declarations are data contracts only. They cannot have companion units, associated functions, operators, methods, or other behavior on the surface.

Declarations directly inside the `co.lang.library` body are exported by default. Imports and implementation details are never exported.

### Public Signature Type Closure

Every public surface function signature must be closed over the library's exported boundary type set.

For `application`, `dynamicvmrt`, and `advanced` surfaces, parameters and results may use only:

- approved built-in types
- `co.lang.struct` types declared in the same library surface

For `system` and `ffi` surfaces, parameters and results may use only:

- ABI-safe built-in types
- `co.lang.cstruct` types declared in the same library surface

The same closure rule applies recursively to fields of surface boundary types:

- a surface `struct` field may use an approved built-in type or another surface-declared `struct`
- a surface `cstruct` field may use an ABI-safe built-in type or another surface-declared `cstruct`
- an internal package type may never appear in a public function signature or surface boundary-type field
- pointers, references, and addresses may never cross any public library surface

A built-in type is not automatically surface-safe merely because it belongs to `co.lang`. An approved surface built-in must be concrete, fully resolved, and transferable under the library kind's boundary semantics.

The following categories are forbidden in public surface fields and signatures:

- inference-only types such as `co.lang.auto` and `co.lang.infer`
- dynamically typed or unconstrained carriers such as `co.lang.dynamic`, `co.lang.any`, `co.lang.typed`, and `co.lang.untyped`
- function, closure, delegate, loader,  AST, reflection, or runtime implementation values
- pointer, reference, address, thunk, and implementation-handle types
- any type whose reachable representation contains a forbidden type

For application-family surfaces, managed built-ins such as `co.lang.string` are permitted when the compiler defines deep-snapshot reconstruction for them. For system and FFI surfaces, only built-ins with a defined ABI representation are permitted; for example, `co.lang.string` is not directly cstruct-compatible.

Valid:

```folang
// Employee.fol
_ co.lang.struct = {
    id      co.lang.int;
    name    co.lang.string;
    address Address;
}

// Address.fol
_ co.lang.struct = {
    city co.lang.string;
}
```

Invalid:

```folang
getEmployee(id co.lang.int)->(emp.internal.Employee);
// compiler error: an internal type escapes through the public signature
```

### Boundary-Adapter Functions

A public surface function may contain a definition, but its definition is restricted to boundary adaptation.

A boundary-adapter function may:

- read its input parameters and their fields
- declare temporary local values used only for conversion
- construct internal input values
- perform deterministic field-to-field or representation conversion
- invoke exactly one internal implementation operation
- receive that operation's result
- construct and return an allowed surface `struct`, surface `cstruct`, or built-in value
- use a direct delegation expression when no conversion is required

A boundary-adapter function may not:

- implement business rules or domain calculations
- perform business validation or authorization decisions
- orchestrate multiple implementation operations
- call another surface function
- perform persistence, networking, file I/O, retries, caching, or transactions
- start concurrency, scheduling, asynchronous work, processes, or continuations
- contain loops, recursion, workflow branching, or externally observable state mutation
- expose an internal value directly without converting it to an allowed boundary type

The compiler enforces the restricted adapter statement set. The restriction is structural: arbitrary executable logic is not accepted merely because it is placed in a surface file.

A direct delegate is the simplest adapter form:

```folang
health()->(co.lang.bool)
    =>> health.internal.Service.health();
```

A converting adapter may map between the public contract and an internal model.

### Standalone Packaged Application-Library Surface Example

```folang
// library.fol — standalone packaged application library
@co.dap.library(type="application")
_ co.lang.library = {

    @co.ddap.import(package="emp", as="emp")

    Employee co.lang.struct = {
        name co.lang.string;
        id   co.lang.int;
    }

    getEmployee(empId co.lang.int)->(Employee) = {
        internalEmployee := emp.EmployeeService.getEmployee(empId);

        this.return Employee{
            name: internalEmployee.name,
            id: internalEmployee.id
        };
    }
}
```

The surface owns conversion from the internal `emp.Employee` representation to the public `hrlib.Employee` contract. The `emp` package owns validation, business rules, persistence, and workflow.

The consumer sees the equivalent API contract:

```folang
// Employee.fol
_ co.lang.struct = {
    name co.lang.string;
    id   co.lang.int;
}

getEmployee(empId co.lang.int)->(Employee);
```

The consumer does not see the body of `getEmployee` or the `emp` package.

### Standalone Packaged System-Library Surface Example

The following example is a separately authored packaged-library project. Application-local `srclib/system/library.fol` does not repeat the kind annotation because its fixed path already establishes `system`.

```folang
// library.fol — standalone packaged system library
@co.dap.library(type="system")
_ co.lang.library = {

    @co.ddap.import(package="driver.internal", as="impl")

    Point co.lang.cstruct = {
        x co.lang.int32;
        y co.lang.int32;
    }

    getOrigin()->(Point) = {
        internalPoint := impl.DriverService.getOrigin();

        this.return Point{
            x: internalPoint.x,
            y: internalPoint.y
        };
    }
}
```

Pointers, addresses, native bindings, and hardware state belong in `driver.internal`. They are forbidden in the public surface and cannot appear in the exported signature.

### Boundary Transfer Semantics

For `application`, `dynamicvmrt`, and `advanced` libraries:

```text
consumer boundary value
    ↓ automatic deep snapshot of the complete reachable value graph
library-local built-in or surface struct
    ↓ surface conversion
internal implementation value
    ↓ surface conversion
library-local built-in or surface struct
    ↓ automatic deep snapshot
consumer-local boundary value
```

The snapshot rule applies to both surface structs and approved built-in arguments/results. The library never receives the consumer's live object identity, and the consumer never receives a live internal-library object.

For `system` libraries, `cstruct` values cross according to the selected backend/platform system ABI.

For `ffi` libraries, `cstruct` values cross according to the declared C ABI, including its layout, alignment, and calling-convention requirements.

### Surface-to-Internal Dependency Direction

The source-level dependency is one-way:

```text
library surface
    ↓ imports and invokes
internal packages
```

Internal packages:

- do not import the library surface
- do not use surface-declared `struct` or `cstruct` types
- define their own implementation and domain types
- return internal values to the surface adapter
- contain all business logic, validation, workflow, I/O, and state management

This prevents a surface/internal compilation cycle and keeps the public contract independent from the internal domain model.

An internal implementation symbol may be visible to its own library surface without becoming part of the consumer API. Library encapsulation takes precedence over ordinary package visibility: consumers cannot resolve internal package symbols even when those symbols are callable from the surface during library compilation.

### Consumer API Projection

The compiler does not expose the complete surface compilation context to a consumer. It creates a projected imported symbol table.

The projected API contains:

- the library identity and kind
- complete public surface `struct` or `cstruct` definitions, including field names and permitted field types
- public function names
- public function parameter names and types
- public function result types
- calling-convention, effect, error, and linkage metadata where applicable

The projected API does not contain:

- function bodies or implementation AST/HIR
- imports used by the surface
- local conversion variables
- delegate or implementation targets
- internal package paths or symbols
- private compiler-generated helpers
- business classes, modules, units, or internal data types

Therefore, a function definition may exist in the surface source and compiled artifact while the consumer's symbol table contains only its signature.

The backend may retain implementation bodies for code generation, linking, or whole-program optimization. Backend possession of implementation code does not make that code semantically visible to the consumer.

Source libraries follow the same rule. Availability of source code does not weaken the projected API boundary.

### Library Compilation Order

A library is processed in four logical stages:

1. parse the surface header, boundary data declarations, and public function signatures
2. compile internal packages without depending on surface types
3. type-check and link surface boundary-adapter bodies against internal package symbols
4. emit the compiled implementation and the projected consumer API

This order permits the surface to call internal packages while preventing internal packages from depending on the surface.

---

## Library Kinds

For application-local source libraries, the library kind is structural and comes from the fixed path:

```text
srclib/ffi/          -> ffi
srclib/system/       -> system
srclib/advanced/     -> advanced
srclib/dynamicvmrt/  -> dynamicvmrt
srclib/operators/    -> operator bootstrap context (not importable; not packaged)
```

All application-local `srclib/<kind>/library.fol` surfaces therefore do **not** repeat the kind with `@co.dap.library(type=...)`. This includes `srclib/operators/library.fol`; the enclosing `operators/` slot establishes the dedicated operator bootstrap context.

A separately authored packaged-library project is not inside an application's standardized `srclib/` slots. Its root `library.fol` must therefore retain the project-level `@co.dap.library(type=...)` declaration. The compiler uses that explicit kind to create/tag the library surface context and symbol table, then verifies that the surface contains only constructs allowed for that kind before producing the `.folenc` projection. The `application` kind is used for an ordinary safe packaged library.

### Shared Meaning

- `application` -> ordinary safe library implementation
- `dynamicvmrt` -> application features plus full `co.meta` dynamic-runtime support
- `advanced` -> macro, template, concurrency, transformation, and runtime-machinery implementation
- `system` -> unsafe low-level runtime, pointer, process, native, and platform-control implementation
- `ffi` -> foreign-interface implementation

Library kind controls internal capabilities. All kinds still expose only boundary data contracts and public boundary-adapter functions.

### `application` Libraries

Internally allowed:

- ordinary safe language features
- classes, modules, interfaces, signatures, units, and ordinary structs
- normal application-level packages and libraries

Internally forbidden:

- pointers, references, and addresses
- native functions annotated with `@co.dap.native`
- macros and templates
- low-level concurrency machinery
- advanced transformation or dynamic-runtime machinery

Public boundary:

- surface `struct` contracts
- approved built-in types
- deep-snapshot transfer

### `dynamicvmrt` Libraries

Internally allowed:

- all `application` capabilities
- full `co.meta` support
- runtime reflection, instrumentation, dynamic loading, patching, and eval-based facilities

Internally forbidden:

- system and FFI capabilities unless reached through an imported public library surface
- raw pointers, addresses, and native functions in its own implementation

Public boundary:

- surface `struct` contracts
- approved built-in types
- deep-snapshot transfer

### `advanced` Libraries

Internally allowed:

- all ordinary safe language features
- macros and templates
- async, parallel, continuation, scheduling, and transformation machinery
- compiler/runtime behavior definitions permitted to advanced libraries

Internally forbidden:

- raw pointers
- references and addresses
- native functions

Public boundary:

- surface `struct` contracts
- approved built-in types
- deep-snapshot transfer

### `system` Libraries

Internally allowed:

- pointers, references, addresses, and word-level types
- native functions
- structs and cstructs
- units containing free functions
- basic value-typed functions
- type aliases
- basic threading and process primitives

Internally forbidden:

- classes
- modules
- interfaces and signatures
- overloading and overriding
- new operator definitions
- generics and templates
- macros
- reflection and dynamic runtime
- dependent types and type classes
- function patterns
- closures and currying
- advanced concurrency machinery such as continuations, defer, lazy evaluation, and thunks
- variadic, named, optional, and default parameters

Public boundary:

- surface `cstruct` contracts
- ABI-safe built-in types
- system ABI value transfer

### `ffi` Libraries

Internally allowed:

- native and extern declarations
- pointers, references, addresses, and C ABI types
- cstructs and restricted implementation structs
- units containing free functions
- foreign calling-convention and symbol-linkage metadata

Internally forbidden:

- classes
- modules
- interfaces and signatures
- overloading and overriding
- new operator definitions
- generics and templates
- macros
- reflection and dynamic runtime
- dependent types and type classes
- closures, currying, and advanced concurrency machinery

Public boundary:

- surface `cstruct` contracts
- C-ABI-safe built-in types
- C ABI value transfer

---

## Dependency Direction

Allowed dependency flow is one-way:

```text
application
    ↓ through public surface APIs only
dynamicvmrt
    ↓ through public surface APIs only
advanced
    ↓ through public surface APIs only
system
    ↓ through public surface APIs only
ffi
```

A library may depend on a library at the same or a lower level only when the dependency does not create a cycle. Reverse dependencies are compiler errors.

### Cross-Library Communication

| From | To | Allowed? | Notes |
|---|---|---|---|
| `application` | `dynamicvmrt` | ✅ | projected public surface only |
| `application` | `advanced` | ✅ | projected public surface only |
| `application` | `system` | ✅ | projected public surface only |
| `application` | `ffi` | ✅ | projected public surface only |
| `dynamicvmrt` | `advanced` | ✅ | projected public surface only |
| `dynamicvmrt` | `system` | ✅ | projected public surface only |
| `dynamicvmrt` | `ffi` | ✅ | projected public surface only |
| `dynamicvmrt` | `application` | ❌ | reverse dependency |
| `advanced` | `system` | ✅ | projected public surface only |
| `advanced` | `ffi` | ✅ | projected public surface only |
| `advanced` | `application` | ❌ | reverse dependency |
| `advanced` | `dynamicvmrt` | ❌ | reverse dependency |
| `system` | `ffi` | ✅ | projected public surface only |
| `system` | `application` | ❌ | reverse dependency |
| `system` | `dynamicvmrt` | ❌ | reverse dependency |
| `system` | `advanced` | ❌ | reverse dependency |
| `ffi` | `application` | ❌ | reverse dependency |
| `ffi` | `dynamicvmrt` | ❌ | reverse dependency |
| `ffi` | `advanced` | ❌ | reverse dependency |
| `ffi` | `system` | ❌ | reverse dependency |


---

## Units in detail

A `co.lang.unit` is a stateless file-level declaration container. It is not instantiable and does not create an object, runtime scope, or public unit namespace.

FoLang has two unit-file forms:

1. **ordinary package unit** — `<Fragment>.unit.fol`
2. **struct companion unit** — `<StructName>.comp.unit.fol`

Both forms use `_ co.lang.unit`; explicit unit names are invalid.

### Ordinary Package Units

Example package layout:

```text
math/
├── arithmetic.unit.fol
├── comparison.unit.fol
└── optional.unit.fol
```

```folang
// arithmetic.unit.fol
_ co.lang.unit = {
    abs(value co.lang.int)->(co.lang.int) = { ... }

    max(
        a co.lang.int,
        b co.lang.int
    )->(co.lang.int) = { ... }
}
```

```folang
// optional.unit.fol
_ co.lang.unit = {
    Option(T) co.lang.type =
        Some(T) | None();

    isSome(value Option(co.lang.int))->(co.lang.bool) = {
        ...
    }
}
```

All declarations are contributed directly to the `math` package namespace:

```folang
math.abs(-10);
math.max(10, 20);
value math.Option(co.lang.int);
```

The unit filenames never appear in qualified names:

```folang
math.Arithmetic.abs(-10); // compiler error
math.Optional.Option(co.lang.int); // compiler error
```

Within the same package, the functions and types may be referenced without the package prefix according to ordinary package name-resolution rules.

An ordinary unit may contain:

- receiverless functions
- `co.lang.type` aliases and ADT/type-constructor declarations
- newtype and opaque-type declarations
- subtype and supertype declarations
- macros and template declarations
- other non-instantiable type declarations explicitly permitted by their sections
- public, package, protected, or private members

An ordinary unit may not contain:

- fields or mutable unit state
- executable loose statements
- classes, structs, cstructs, enums, unions, modules, interfaces, or signatures
- lifecycle declarations for the unit
- a declaration that creates an independent nested primary namespace

A unit file may group closely related declarations or merely organize a convenient set of package-level functions and types. Semantic cohesion is recommended but not enforced.

### Package-Namespace Consolidation

All ordinary unit files in one package are shallow-parsed and consolidated into one package symbol table. Their order and filenames have no semantic effect.

```text
package members =
    filename-derived primary declarations
    + declarations from every ordinary unit file
```

A duplicate is reported as soon as the package index is built. Conflicts include:

- the same type or value name contributed by two ordinary units
- an ordinary-unit member colliding with a filename-derived primary declaration
- duplicate function signatures not permitted by overload rules
- declarations that normalize to the same canonical package symbol

Diagnostics should identify both source files.

### Companion Units

A companion unit is governed by the filename and owner-validation rules in [Struct Companion Units](#struct-companion-units). Its members do not merge directly into the package namespace; they attach to the owner struct's companion namespace.

Units have no fields, identity, instances, inheritance, polymorphic dispatch, or independent construction.

## CStructs


`co.lang.cstruct` is a C-like value type — passed by value, simple memory layout, safe to cross zone boundaries. Unlike `co.lang.struct` which is passed by reference, `co.lang.cstruct` is always copied on pass.
```folang
// Point.fol
_ co.lang.cstruct = {
    x co.lang.int;
    y co.lang.int;
}

// Rect.fol
_ co.lang.cstruct = {
    origin Point;
    width  co.lang.int;
    height co.lang.int;
}
```

#### cstruct Rules
```
always passed by value — never by reference
simple memory layout — no metadata
can contain only simple types and other cstructs
cannot contain co.lang.struct                ❌  has metadata
cannot contain co.lang.string                ❌  heap allocated
cannot contain co.lang.dynamic               ❌  runtime type info
cannot contain classes                       ❌  vtable, metadata
cannot contain modules                       ❌
cannot contain any heap allocated type       ❌
cannot have methods
cannot have associated functions
cannot embed co.lang.struct
safe to cross direct ABI and zone boundaries

allowed field types:
    co.lang.int, co.lang.uint, co.lang.float  ✅  primitives
    co.lang.bool, co.lang.char, co.lang.byte  ✅  primitives
    co.lang.int->([N])                         ✅  fixed size arrays
    co.lang.cstruct                            ✅  other cstructs
```

#### Packed cstruct — no padding, exact memory layout
Used for hardware registers, binary protocols, exact memory mapped formats:
```folang
// Register.fol
@co.dap.packed
_ co.lang.cstruct = {
    flags  co.lang.uint8;
    status co.lang.uint8;
    data   co.lang.uint16;
}
```

#### SIMD cstruct — aligned for vector operations
Used for math, graphics, signal processing:
```folang
// Vec4.fol
@co.dap.simd(align=16)
_ co.lang.cstruct = {
    x co.lang.float;
    y co.lang.float;
    z co.lang.float;
    w co.lang.float;
}
```

#### Both together
```folang
// AVXVec.fol
@co.dap.packed
@co.dap.simd(align=32)
_ co.lang.cstruct = {
    data co.lang.float;
}
```

> `@co.dap.packed` and `@co.dap.simd` are specialisations of `co.lang.cstruct` — same rules, same zone boundary safety. They are not separate types.

---

## Structs

```folang
// myStruct.fol
_ co.lang.struct={
    field1 co.lang.int;
    field2 co.lang.string;
    field3 co.lang.bool;
}
```

#### Struct Rules

```
structs cannot extend other structs
structs cannot contain methods directly
structs can compose other structs
structs cannot have default values to fields/members
structs can embed other structs (Go lang like)
structs can have a same-package companion file named `<StructName>.comp.unit.fol`
```

The struct declaration remains pure data. Associated behaviour is declared separately in `<StructName>.comp.unit.fol`.

#### Struct Embedding

Embedding promotes fields of an embedded struct directly into the outer struct — they act as the outer struct's own fields at construction and access sites. This is distinct from composition where the embedded struct is a named field.

```folang
// E.fol
_ co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}

// ✅ No conflict — id and name promoted as B's own fields
// B.fol
_ co.lang.struct = {
    age co.lang.float;
    E;                    // embedded — id and name promoted
}

b := B{ age: 25.0, id: 1, name: "Rao" };   // all fields at same level
b.id    // direct — no b.E.id needed
b.name  // direct — no b.E.name needed
b.age   // direct
```

```folang
// ❌ Compiler error — name conflict between B.name and E.name
// B.fol
_ co.lang.struct = {
    name co.lang.string;   // conflicts with E.name
    E;
    age  co.lang.float;
}
// Fix 1 — rename B's conflicting field
// Fix 2 — use explicit composition instead: e E;
```

```folang
// Explicit composition — no promotion, always qualified access
// B.fol
_ co.lang.struct = {
    name co.lang.string;
    e    E;               // named field — no conflict, no promotion
    age  co.lang.float;
}

b.name ;   // B's own name
b.e.id ;   // E's id — always explicit
b.e.name;  // E's name — always explicit
```

#### Embedding Rules

| Situation | Behavior |
|---|---|
| Embedded field, no conflict | Promoted — acts as the outer struct's own field |
| Embedded field, name conflict with outer | ❌ Compiler error — rename or use composition |
| Multiple embeds, no conflict between them | All fields promoted |
| Multiple embeds, conflict between embedded structs | ❌ Compiler error |
| Explicit composition (`e E`) | No promotion — always accessed via `b.e.field` |

> FoLang does **not** silently shadow conflicting fields. Any name conflict is a compiler error — the programmer must make a conscious decision to rename or switch to explicit composition.

#### Struct Declaration Relationships

A struct cannot physically declare another struct, enum, class, module, function, signature, interface, or other named declaration inside its body. A struct body remains a pure data declaration containing fields and permitted embeddings only.

Use one of the following instead:

- **composition** through a named field
- **embedding** where the declaration kind's embedding rules permit it
- a separately declared target-local declaration restricted with `@co.dap.local`

```folang
// EmployeeAddress.fol
@co.dap.local(for=hr.employee.Employee)
_ co.lang.struct = {
    street co.lang.string;
    city   co.lang.string;
}
```


```folang
// Employee.fol
_ co.lang.struct = {
    id      co.lang.int;
    name    co.lang.string;
    address EmployeeAddress; // composition
}
```

The following physical nesting is invalid:

```folang
// Employee.fol
_ co.lang.struct = {
    Address co.lang.struct = { // ❌ nested declaration
        city co.lang.string;
    }

    address Address;
}
```
```
structs cannot declare inner structs     ❌  compiler error — only through @co.dap.local
structs can declare inner enums/ADTs     ❌  compiler error — only through @co.dap.local or @co.dap.nested
structs cannot declare inner classes     ❌  compiler error — struct is pure data
structs cannot declare inner modules     ❌  compiler error — struct is pure data
```

`@co.dap.local` controls declaration visibility only. It does not automatically compose or embed the annotated declaration into the target struct.

### Struct Companion Units

A struct companion unit is declared in a file named `<StructName>.comp.unit.fol`. The matching struct remains in `<StructName>.fol`, and both files must belong to the same package.

```folang
// Vector.fol
_ co.lang.struct = {
    x co.lang.float;
    y co.lang.float;
}
```

```folang
// Vector.comp.unit.fol
_ co.lang.unit = {
    distance(
        left  Vector,
        right Vector
    )->(co.lang.float) = {
        ...
    }

    zero()->(Vector) = {
        this.return Vector{x: 0.0, y: 0.0};
    }

    (value Vector) magnitude()->(co.lang.float) = {
        this.return co.math.sqrt(
            value.x * value.x + value.y * value.y
        );
    }

    (Vector) create(
        x co.lang.float,
        y co.lang.float
    )->(Vector) = {
        this.return Vector{x: x, y: y};
    }
}
```

The filename establishes ownership. No companion member needs an ordinary parameter merely to prove association.

Call forms:

```folang
d      := Vector.distance(first, second);
origin := Vector.zero();
length := value.magnitude();
point  := Vector.create(10.0, 20.0);
```

A companion unit may declare:

- receiverless companion functions, including zero-parameter functions and factories
- instance-associated functions with an explicit value receiver
- type-associated functions with an explicit type receiver
- operator functions permitted for the owner struct
- non-UDT type declarations associated with the owner when their own sections permit them
```text
receiverless non-operator companion function
    -> has no explicit receiver
    -> ordinary parameters may have any legal types
    -> no owner-typed parameter is required merely for association

instance-associated function
    -> explicit receiver is a value of the matching struct

type-associated function
    -> explicit receiver is the matching struct type itself
```

These are the general companion rules. Operator functions have the additional operand requirement defined in the Operator Functions subsection.

Companion declarations do not make the struct a class and do not add object identity, inheritance, virtual dispatch, lifecycle methods, or unit-level state.

#### Filename and Owner Validation

The compiler classifies companion files before parsing:

```text
Vector.comp.unit.fol -> canonical owner Vector
```

At package-indexing level, the compiler verifies:

- a canonical primary declaration named `Vector` exists in the same package
- no second companion file resolves to the same canonical owner
- the companion is not orphaned

After the owner declaration header is available, the compiler verifies that the owner is a `co.lang.struct`. A class, cstruct, enum, union, module, interface, signature, or imported declaration with the same short name cannot become the owner.

#### Receiver Validation

The companion filename determines the required receiver root. Any explicit receiver must match that owner.

```folang
// Employee.comp.unit.fol
_ co.lang.unit = {
    create(id co.lang.int)->(Employee) = { ... } // valid: receiverless

    (emp Employee) isValid()->(co.lang.bool) = { ... } // valid

    (Employee) empty()->(Employee) = { ... } // valid

    (dept Department) isValid()->(co.lang.bool) = { ... }
    // compiler error: receiver Department does not match companion owner Employee

    (Department) create()->(Department) = { ... }
    // compiler error: type receiver Department does not match companion owner Employee
}
```

Ordinary parameters may have any legal types. They are not used to establish companion ownership.

For a generic struct owner, receiver validation compares the canonical root declaration and generic arity. For example, a receiver based on `Box(T)` may belong to `Box.comp.unit.fol`; `Box(T, E)` or an unrelated root does not.

#### Companion Namespace and Conflicts

Companion members are resolved through the owner:

```folang
employee := Employee.create(1);
valid    := employee.isValid();
```

The complete companion namespace is formed from all legal owner-associated declarations and the single companion file. Duplicate names and normalized duplicate signatures are compile-time errors.

The following are invalid:

```text
Employee.comp.unit.fol without Employee.fol
Employee.comp.unit.fol and employee.comp.unit.fol in the same package
Employee.comp.unit.fol when Employee.fol declares a class or cstruct
an explicit receiver whose canonical root is not Employee
```

#### Operator Functions

Operator functions associated with a struct belong in its companion unit. Ownership comes from the companion filename; receiver validation follows the same rule as other companion functions.

```folang
// Employee.comp.unit.fol
_ co.lang.unit = {
    @co.dap.operator(symbol="+")
    (emp Employee) add(other Employee)->(Employee) = {
        ...
    }

    @co.dap.operator(symbol="==")
    equals(
        left  Employee,
        right Employee
    )->(co.lang.bool) = {
        ...
    }

    @co.dap.operator(symbol=">")
    (Employee) greater(
        left  Employee,
        right Employee
    )->(co.lang.bool) = {
        ...
    }
}
```

> Companion ownership always comes from `<StructName>.comp.unit.fol`. An operator function has an additional operand rule: a value receiver already supplies the owner instance, but a receiverless operator function or a type-receiver operator function must declare the companion owner type as its first ordinary parameter. This allows operator lowering to pass the actual owner instance to the function. The compiler compares the resolved first-parameter type with the companion owner type at compile time. This requirement establishes the operator operand, not companion ownership. An operator declaration that does not operate on the owner struct is invalid.

---

## Unions 
 Unions are untagged ADTs
```folang
 // myUnion.fol
 _ co.lang.union={
    intValue co.lang.int;
    strValue co.lang.string;
}
```
## Enums

```folang
// myEnum.fol
_ co.lang.enum={
    Variant1,
    Variant2,
    Variant3
}
```

An enum value's constant expression may use a registered custom operator at
any declared precedence. Runtime assignment is forbidden everywhere in that
constant-expression subtree, including inside grouping, arguments, and
collection or object elements. Whether the resulting expression can actually
be evaluated at compile time—including whether a custom operator is foldable—
is checked after parsing.

## classes 
```folang
// Employee.fol
_ co.lang.class ={
    getEmployeeDetails()->(Employee) = empmodule.getEmployeeDetails;
    // assigning module function to class's method

    getEmployeeInfo()->(Employee) =>> empmodule.getEmployeeDetails();
    // delegating — internally redirecting the call to module function
}

// $1, $2, $3 ... are previous results captured as bind variables
// Emp.fol
_ co.lang.class={
    dosomething(a co.lang.int, b co.lang.int)->(co.lang.int)=>>somePack.someMethod(a)=>>someOthPack.someOtherMeth($1, b);
}
```
### Classes with Operator methods

```folang
// Employee.fol
_ co.lang.class ={
    @co.dap.operator(symbol="+")
    add(other Employee)->(Employee) = {
       // implicit instance method: operands are this and other
    }

    @co.dap.operator(symbol="==")
    @co.dap.static
    equals(
        left  Employee,
        right Employee
    )->(co.lang.bool) = {
        // static operator implementation
    }

    @co.dap.operator(symbol=">")
    @co.dap.class
    greater(
        left  Employee,
        right Employee
    )->(co.lang.bool) = {
        // class-associated operator implementation
    }

}


```

An unannotated operator function in a class is an implicit instance method;
the hidden `this` value contributes the first operand. `@co.dap.instance` is
the explicit spelling of the same category. `@co.dap.static` and
`@co.dap.class` do not contribute an implicit operand, so their declared
parameters are the complete operator operand list. Operator signature
normalization includes these method categories so equivalent declarations are
diagnosed as duplicates.

### Class Declaration Relationships

A class cannot physically contain named class, struct, enum, module, function, signature, interface, or other declaration definitions as nested declarations. Class bodies contain fields, lifecycle declarations, and methods permitted by the class model.

Types and helper declarations used only by one class are declared in their ordinary legal source locations and restricted to that class with `@co.dap.local`:

```folang
// EmployeeAddress.fol
@co.dap.local(for=hr.employee.Employee)
_ co.lang.struct = {
    street co.lang.string;
    city   co.lang.string;
}
```

```folang
// EmployeeStatus.fol
@co.dap.local(for=hr.employee.Employee)
_ co.lang.enum = {
    Active,
    Inactive,
    Pending
}
```

```folang
// Employee.fol
_ co.lang.class = {
    address EmployeeAddress;
    status  EmployeeStatus;

    getAddress()->(EmployeeAddress) = {
        this.return this.address;
    }
}
```

The local declarations are visible while compiling `hr.employee.Employee`, but they do not become nested names such as `Employee.EmployeeAddress` or `Employee.EmployeeStatus`.

The following is invalid:

```folang
// Employee.fol
_ co.lang.class = {
    Address co.lang.struct = { // ❌ physical nested declaration
        city co.lang.string;
    }
}
```

Ordinary visibility annotations do not widen a target-local declaration beyond the declaration or closed target set named by `@co.dap.local` or `@co.dap.nested`.

### Method Types

```folang
// Employee.fol
_ co.lang.class ={

    @co.dap.static
    getEmployee()->(Employee) ={}

    @co.dap.instance
    getEmployee()->(Employee)={}

    @co.dap.class
    getEmployee()->(Employee) ={}

    @co.dap.object
    getEmployee()->(Employee)={}
}

@co.dap.oops(
    A: { inherit:true, virtual:true },
    B: { implements:true },
    C: { inherits:true, abstract=true },
    D: { inherits:true },
    E: { uses:true },
    F: { composes:true },
    G: { extends:true },
    H: { with:true },
    I: {assiociate:true},
)
// test.fol
_ co.lang.class ={
    getTest(id int)->(test) ={}
}
```

### The @@new and @@init Methods

`@@new` and `@@init` are lifecycle methods — compiler-owned, not user-definable outside the class. `@@` signals they are restricted lifecycle symbols, not regular methods.

Lifecycle names are valid only as class members. A unit (including a struct
companion unit), module, interface, signature, local block, or package
declaration cannot declare a lifecycle-named function.

```folang
// Employee.fol
@co.dap.generic(type={T:{typename}, R:{typename}})
_ co.lang.class = {

    id T;
    name R;

    // @@new is provided by default even if not overridden.
    // Override only when you genuinely need to change allocation behavior.

    @co.dap.class
    @co.dap.private
    @@new()->(co.lang.uninit) = { self.return co.const.none }

    @co.dap.class
    @co.dap.public
    @@new(a co.lang.typevalue, b co.lang.typevalue)->(co.lang.uninit) = {
        // Manual type aliasing — @co.dap.generic handles this automatically
        // Override @@new only when you need custom allocation logic
        T co.lang.type = a;
        R co.lang.type = b;

        // self keyword is allowed only in class methods
        self.parent.new();

        // uninit instance method internally calls new and init
        self.return co.lang.uninit.instance(Employee, self);
    }

    @co.dap.override
    @co.dap.constructor(access=private)
    @@init() = {}

    @co.dap.override
    @co.dap.constructor(access=public)
    @@init(id T, name R) = {
        this.parent.init();
        this.id   = id;
        this.name = name;
    }

    getEmployee(id T)->(Employee) = {}
}
```
---
#### Anonymous Classes/Types

```folang
emp := co.lang.class{};

empObj := emp.init();

empobj1 := co.lang.class{
    name string;
}.init();
```
---
---
## Interfaces
```folang
// IEmployee.fol
_ co.lang.interface = {
    storeEmployee(emp Employee)->(Employee);
}
```

## Signatures

```folang
// Employee is an ordinary package-level declaration.
// MEmployee.fol
_ co.lang.signature = {
    storeEmployee(emp Employee)->(Employee);
}
```

---

Structurally they look similar — both are lists of contracts. The difference is **who implements them and how**.

| | `co.lang.signature` | `co.lang.interface` |
|---|---|---|
| Implemented by | module via `matches=` | class via `implements=[]` |
| Number of implementations | Any number of distinct modules may match one signature | Any number of classes may implement one interface |
| Runtime cardinality of one implementation declaration | One module object per loaded module identity | Any number of independently constructed class objects |
| State model | One shared state for all references to the same module | Separate per-instance state for each class object |
| May specify required values | ✅ | ❌ — methods only |
| May reference package-level types | ✅ | ✅ |
| May declare abstract/fixed type components | ✅ | ❌ |
| May require generic type constructors | ✅ | ❌ |
| May declare physical nested/local types | ❌ | ❌ |
| May use `@co.dap.local` | ❌ | ❌ |
| Instantiation involved | Module is declared once, not constructed | Class objects are constructed |
| Reference use | Compatible module references may be used through the signature type without creating another module | Interface references may refer to any implementing object instance |
| OOP dispatch | ❌ | ✅ virtual/dynamic |
| Contract style | module values, functions, and type components | behavioral methods on object instances |
| Practical analogy | singleton component contract | object-instance behavioral contract |
| Origin | ML/OCaml-inspired modules | Java/C#/Go interfaces |

- A `signature` is a **module contract** over values, functions, existing package-level types, and abstract or fixed type components. A type-component specification is a contract slot, not a physical nested type definition. Multiple modules may match one signature, but each module declaration denotes one module object with shared module state.
- An `interface` is a **behavioral contract** tied to class dispatch and polymorphism. It cannot declare module type components or own nested type definitions. A class implementing an interface may create any number of independent runtime objects.
- The approximation `module + signature ≈ singleton object + interface` is useful for understanding cardinality and shared state, but a module is a language-level component rather than a class-based singleton pattern.

---

## Modules
A module is an ML/OCaml-style abstraction governed by an optional signature. A module may use package-level types and may satisfy type components declared by its signature, but it does not physically own or nest arbitrary type declarations. A module should not be introduced merely to prevent functions from appearing loose in a file; use `co.lang.unit` for that simpler structural purpose.

```folang
// Employee.fol — ordinary package-level type
_ co.lang.struct = {
    Id   co.lang.int;
    Name co.lang.string;
}

// EmployeeModule.fol
_ co.lang.signature = {
    getEmployee(id co.lang.int)->(Employee);
}

// EmployeeModImpl.fol
@co.dap.module(signature=EmployeeModule)
_ co.lang.module->(signature=EmployeeModule, matches=EmployeeModule) = {

    getEmployee(id co.lang.int)->(Employee) = {
        this.return Employee{
            Id: 10,
            Name: "Rao"
        };
    }
}

mm EmployeeModule = EmployeeModImpl;
v Employee = mm.getEmployee(10);
```

### Module Cardinality and Singleton Analogy

A module declaration defines exactly one named module object for that loaded module identity. It is not an instantiable blueprint and does not create a new module object each time its name is referenced.

```folang
first  EmployeeModule = EmployeeModImpl;
second EmployeeModule = EmployeeModImpl;
```

`first`, `second`, and `EmployeeModImpl` refer to the same module object. These bindings copy or retain the module reference; they do not clone or instantiate the module. Consequently, values declared directly by the module represent one shared module state for all references to that module.

A signature does not itself create a module object. It is a contract, and any number of separately declared modules may conform to the same signature:

```text
DatabaseBackend signature
├── PostgreSQLBackend module   -> one named module object
├── MySQLBackend module        -> one named module object
└── TestDatabaseBackend module -> one named module object
```

Each conforming module declaration contributes its own single module object and its own module state. The signature does not restrict the number of distinct conforming module declarations.

A useful analogy is:

```text
signature          ≈ interface contract
conforming module  ≈ singleton object implementing that contract
class              ≈ instantiable object type
```

The analogy is intentionally limited. A FoLang module is a compiler-recognized named component, not a class made singleton through a private constructor, static field, or runtime pattern. It cannot be repeatedly constructed. Because module references are first-class in FoLang, they may be bound and used through a compatible signature type, but every reference to the same module declaration still denotes the same module object.

Modules are also broader than ordinary interface implementations. A matching module may provide module values, functions, and abstract, fixed, or generic type-component bindings required by its signature. An interface constrains object behaviour; it does not provide the same module type-component abstraction.

By contrast, a class declaration defines an instantiable type. Every class construction creates a distinct object with independent identity, state, and lifetime:

```text
PostgreSQLBackend module
└── one shared module object and module state

PostgreSQLConnection class
├── connection1 -> independent object and state
├── connection2 -> independent object and state
└── connection3 -> independent object and state
```

> **Formal mental model:** A FoLang module is a single named implementation component that may conform to a signature. It is comparable to a singleton object implementing an interface, but it is not instantiated from a class. Multiple distinct modules may conform to the same signature, while each module declaration denotes one module object. Unlike an ordinary singleton-interface implementation, a module may also satisfy abstract, fixed, and generic type components required by its signature.

> **Module instantiation** A FoLang class or struct declaration introduces an instantiable type but does not create an instance. A FoLang module declaration introduces one named module component directly into its package. The module name acts as the binding for that component, so no separate construction expression is required. The module’s runtime state is initialized once according to the language’s module-initialization rules.

> A module declaration is a singleton component declaration and binding, rather than merely an object definition.

### Module Signature Contents

A `co.lang.signature` is a declarative contract for a module. It may specify required module values, functions, and type components. A signature does not allocate storage, initialize variables, execute statements, or provide function bodies.

A signature may contain:

- value specifications
- function signatures
- references to already existing accessible types
- abstract type-component specifications
- fixed or manifest type-component specifications
- abstract generic type-constructor specifications

A signature may not contain:

- value initializers
- executable statements
- function bodies
- concrete class, struct, enum, module, interface, or signature definitions
- arbitrary nested or target-local declarations

Type-component specifications are part of module conformance. They are not Java-, C++-, or C#-style inner types and do not participate in `@co.dap.local`.

#### Value Specifications

A declaration such as:

```folang
// Counter.fol
_ co.lang.signature = {
    count co.lang.int;
    increment(amount co.lang.int)->();
}
```

requires a matching module to provide a value named `count` of type `co.lang.int` and a compatible `increment` function. The signature does not initialize `count` and does not define `co.lang.int`; the built-in type already exists.

```folang
// CounterImpl.fol
_ co.lang.module->(
    signature=Counter,
    matches=Counter
) = {
    count co.lang.int = 0;

    increment(amount co.lang.int)->() = {
        count.value = count + amount;
    }
}
```

The same rule applies when a value or function specification uses an existing accessible package type:

```folang
// EmployeeRepository.fol
_ co.lang.signature = {
    current hr.employee.Employee;
    find(id co.lang.int)->(hr.employee.Employee);
}
```

The matching module must provide `current` and `find`. It does not redefine `hr.employee.Employee`.

#### Abstract Type Components

An abstract type component declares that every matching module must supply a type binding for that component:

```folang
// Repository.fol
_ co.lang.signature = {
    Entity co.lang.type;   

    current Entity;
    find(id co.lang.int)->(Entity);
}
```

`Entity co.lang.type;` does not define the representation of `Entity`. It defines a required module type component. A matching module binds it to a compatible existing type:

```folang
// EmployeeRepositoryImpl.fol
_ co.lang.module->(
    signature=Repository,
    matches=Repository
) = {
    Entity co.lang.type = hr.employee.Employee;

    current Entity = ...;
    find(id co.lang.int)->(Entity) = { ... }
}
```

Within a matching module, `Entity co.lang.type = ...` is a **signature-component binding**, not an arbitrary nested type declaration. Its name must correspond to an abstract type component declared by the matched signature. A module cannot use this form to introduce unrelated module-local types.

An abstract type component differs from `forward` and `extern` declarations:

```text
abstract signature type component
    -> each matching module supplies its own compatible type binding

forward declaration
    -> one specific declaration is completed later

extern declaration
    -> one specific declaration is defined in another linkage or compilation unit
```

#### Fixed or Manifest Type Components

A signature may fix a type component to an already known type:

```folang
// IntegerRepository.fol
_ co.lang.signature = {
    Id co.lang.type = co.lang.int;

    find(id Id)->(co.lang.bool);
}
```

Here `Id` is predetermined as `co.lang.int`. A matching module uses that type component but does not choose or redefine it.

```text
Entity co.lang.type;               -> abstract; implementor supplies the binding
Id co.lang.type = co.lang.int;     -> fixed; signature supplies the type equality
```

#### Abstract Generic Type Constructors

A signature may require a generic type constructor without defining its representation:

```folang
// StackSignature.fol
_ co.lang.signature = {
    Stack(T) co.lang.type; 

    empty(T)->(Stack(T));
    push(value T, stack Stack(T))->(Stack(T));
    pop(stack Stack(T))->(T, Stack(T));
}
```

`Stack(T) co.lang.type;` declares an **abstract generic type component**, also described as an abstract type constructor of arity one. The signature specifies that `Stack` accepts one type argument, but it does not define what `Stack(T)` is.

A matching module must provide a compatible type-constructor binding with the same name, arity, and declared constraints:

```folang
// ListStackModule.fol
_ co.lang.module->(
    signature=StackSignature,
    matches=StackSignature
) = {
    Stack(T) co.lang.type = co.core.list(T);

    empty(T)->(Stack(T)) = { ... }
    push(value T, stack Stack(T))->(Stack(T)) = { ... }
    pop(stack Stack(T))->(T, Stack(T)) = { ... }
}
```

Another matching module may choose another representation:

```folang
// ArrayStackModule.fol
_ co.lang.module->(
    signature=StackSignature,
    matches=StackSignature
) = {
    Stack(T) co.lang.type = collections.ArrayStack(T);
    ...
}
```

Therefore:

```text
StackSignature
    -> requires a generic type constructor Stack(T)

ListStackModule
    -> binds Stack(T) to co.core.list(T)

ArrayStackModule
    -> binds Stack(T) to collections.ArrayStack(T)
```

A signature-component binding does not permit physical type nesting. When an implementation needs a new concrete struct, class, or enum representation, that declaration remains an ordinary package declaration and may be restricted to the implementing module with `@co.dap.local`; the module then binds the signature component to that declaration.

#### Signature Conformance Rules for Type Components

For every type component in a matched signature:

- an abstract component must receive exactly one compatible module binding
- a fixed component must retain the type equality declared by the signature
- a generic component binding must preserve generic arity, parameter kinds, bounds, variance, and other declared constraints
- component names must be unique within the signature
- component bindings cannot contain executable code
- extra module-local type bindings that do not correspond to signature components are compiler errors
- types referenced by value and function specifications must resolve after applying the module's type-component bindings

#### Module Declaration Relationships

A module cannot physically declare nested structs, enums, classes, modules, signatures, interfaces, or other arbitrary named declarations. It references ordinary package-level declarations through its functions and signature. The only type-like declarations permitted directly in a matching module are signature-component bindings that satisfy abstract type components declared by its matched signature; such bindings do not create independent nested declarations.

A declaration intended only for one module may be restricted with `@co.dap.local`:

```folang
// EmployeeModuleConfig.fol
@co.dap.local(for=hr.employee.EmployeeModImpl)
_ co.lang.struct = {
    timeout co.lang.int;
    retries co.lang.int;
}
```

```folang
// EmployeeModImpl.fol
_ co.lang.module = {
    connect(cfg EmployeeModuleConfig)->(co.lang.bool) = {
        ...
    }
}
```

The following remains invalid:

```folang
// EmployeeModImpl.fol
_ co.lang.module = {
    Config co.lang.struct = { // ❌ physical nested declaration
        timeout co.lang.int;
    }
}
```

    A target-local declaration does not automatically become a module member name and is not projected through the module's signature. It becomes part of the signature view only when an explicit signature type component is bound to it or a signature value/function specification references it through an allowed type component.

---
## Structs vs Classes vs Modules vs Units vs Packages

| | Struct | CStruct | Class | Module | Unit | Package |
|---|---|---|---|---|---|---|
| **Purpose** | Pure data shape | C-like value type | Behaviour + data | Signature-backed ML-style abstraction | Stateless package-fragment or struct-companion container | Folder-based grouping |
| **Fields** | ✅ | ✅ simple only | ✅ per instance | ❌ | ❌ | ❌ |
| **Module-level values** | ❌ | ❌ | ❌ | ✅ when declared directly or required by a signature | ❌ | ❌ |
| **Functions / methods** | Companion functions through `<StructName>.comp.unit.fol`; explicit receivers must match the struct | ❌ | ✅ methods | ✅ module functions | ✅ package functions in ordinary units; companion functions in companion units | ❌ |
| **Lifecycle** (`@@new`/`@@init`) | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| **`this` / `self`** | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| **Value/literal construction** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Lifecycle instantiation (`new`/`init`)** | ❌ | ❌ | ✅ multiple objects | ❌ — one module object per declaration | ❌ | ❌ |
| **Runtime state cardinality** | Per bound struct object | Per value | Per class object | One shared state for the module declaration | — | — |
| **First class** | ✅ | ✅ | ✅ | ✅ module reference; referencing does not instantiate | ❌ | ❌ |
| **Pass by** | Reference | Value | Reference | Reference to the same module object | — | — |
| **Contract** | — | — | `interface` via `implements=[]` | `signature` via `matches=` | none | — |
| **OOP / inheritance** | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| **Physically nested independent named type/container declarations** | ❌ | ❌ | ❌ | ❌ | ❌ | N/A — packages contain separate source declarations |
| **May be an `@co.dap.local` target** | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ |
| **Pattern matching** | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Direct ABI / zone boundary safe** | ❌ — library boundaries require snapshots | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Associated functions** | ✅ through `<StructName>.comp.unit.fol` | ❌ | — | — | ✅ only in a struct companion unit | ❌ |
| **Embedding** | ✅ | ❌ | — | — | ❌ | ❌ |
| **Declared with** | `co.lang.struct` | `co.lang.cstruct` | `co.lang.class` | `co.lang.module` | `co.lang.unit` | folder path |
| **C++ backend analogy** | struct without methods | plain C struct | class | struct/class abstraction | package namespace fragment or static companion scope | namespace |
| **Closest mental model** | Rust struct | C struct | Java/C# class | singleton implementation component with ML-style type members | source fragment merged into a package, or a filename-bound struct companion | filesystem namespace |

**Mental model:**

```text
reach for struct   → pure data; use `<StructName>.comp.unit.fol` for associated behaviour
reach for cstruct  → physical ABI-compatible value data crossing direct zone or native boundaries
reach for class    → behaviour, lifecycle, multiple instances
reach for module   → one named implementation component with shared state, governed by an optional signature and capable of satisfying type components
reach for unit     → package fragment (`*.unit.fol`) or struct companion (`*.comp.unit.fol`)
reach for package  → folder-based grouping only, not a value
```

> **Declaration scoping rule:** FoLang does not permit physical nesting of independent file-backed primary declarations. Classes, structs, cstructs, enums, unions, modules, interfaces, signatures, instances, matchers, and other package-owned primary declarations remain in their own `<Name>.fol` files. Ordinary and companion unit files are explicit package containers: they may contain functions and the non-UDT type declarations permitted by the unit rules, but they may not contain independent primary declarations such as classes, structs, enums, modules, interfaces, or signatures. Ordinary local functions and anonymous expressions remain the other explicit nesting exceptions. Supported package declarations may restrict visibility to exact same-package targets with `@co.dap.local`; the annotation changes visibility, not physical ownership. Signature type components and matching module bindings are contract slots rather than arbitrary nested package declarations.
---

## Local and/or Nested types and functions

FoLang does not provide Java-, C++-, or C#-style physical nesting of independent named type and container declarations. Such declarations remain in their ordinary legal source locations. Ordinary local functions and anonymous expressions are explicit exceptions governed by the rules below.

### Physical Nesting Rules

#### Prohibited Independent Named Declarations

Independent file-backed primary declarations cannot be physically declared inside another class, struct, cstruct, enum, union, module, unit, interface, signature, function, or executable block. This includes:

- classes, structs, cstructs, enums, and unions;
- modules, interfaces, signatures, and additional units;
- instances, matchers and other file-backed primary declarations.

Non-UDT type declarations are the deliberate unit exception. macros, templates, decorators , Type aliases, `co.lang.type` ADTs and type constructors, newtypes, opaque types, subtypes, and supertypes may be declared directly inside an ordinary unit, and inside a companion unit where their own rules permit association with the owner. They are not permitted loose at package-file scope or physically inside classes, structs, modules, functions, or executable blocks unless another section explicitly grants that context.

File-backed primary declarations retain package-owned identity and follow their normal `<Name>.fol` placement rules. An association or visibility annotation such as `@co.dap.local`, `@co.dap.nested`, or `@co.dap.inner` does not physically move a separately declared declaration inside its target.

#### Local-Function Exception

A function body may contain an ordinary named local or inner function where the grammar permits a local-function declaration:

```folang
outer()->() = {
    value co.lang.int = 10;

    inner()->() = {
        co.out.println(value);
    }

    inner();
}
```

The local function has block-local declaration identity. Its free runtime names are resolved from its lexical declaration context, and its lifetime and escape behavior follow the ordinary inner-function rules. It is not a package member and cannot be independently imported or exported.

This exception permits local functions only. It does not permit a named class, struct, enum, module, unit, interface, signature, or another named type/container declaration inside a function body.

#### Anonymous-Expression Exception

The physical-nesting restriction does not apply to anonymous constructs that are expressions or type expressions rather than independent named declarations. They may be nested wherever their specific grammar category is permitted.

These include:

- anonymous function expressions;
- lambdas and callback blocks;
- anonymous class or anonymous type expressions;
- permitted anonymous polymorphic type expressions introduced by `forall`;
- ordinary nested block, object-construction, map, collection, and other value-producing expressions.

```folang
process()->() = {
    operation := (value co.lang.int)->(co.lang.int) = {
        this.return value * 2;
    };

    worker := co.lang.class {
        run(value co.lang.int)->(co.lang.int) = {
            this.return operation(value);
        }
    }.init();
}

transformer co.lang.type = forall(T).(T)->(T);
```

An anonymous construct has no independently addressable package declaration identity. Its scope, capture, lifetime, type, and escape behavior are determined by the rules for that specific construct. Syntactic containment of an anonymous expression does not create a Java-, C++-, or C#-style named nested declaration and does not violate the one-primary-declaration-per-package-file rule.

#### Relationship to Association Annotations

The exceptions above are distinct from FoLang's association annotations:

- an ordinary local function is physically declared inside an enclosing function and is lexically scoped;
- an anonymous construct is an expression without independent declaration identity;
- `@co.dap.local`, `@co.dap.nested`, and `@co.dap.inner` apply to separately declared declarations that retain their normal source identity.

The remainder of this section defines those separately declared association forms.

```folang
@co.dap.local(for=<declaration-reference>)
```

or:

```folang
@co.dap.local(
    for=[
        <declaration-reference>,
        <declaration-reference>
    ]
)
```

The annotated declaration is called a **target-local declaration**. The declarations named by `for` form its **local target set**.

### Supported Declaration and Target Kinds

`@co.dap.local` may be applied only to these declaration kinds:

- `co.lang.class`
- `co.lang.struct`
- `co.lang.enum`
- `co.lang.module`
- a named function declared in a context that normally permits functions

Every entry in the local target set must resolve to exactly one:

- class
- struct
- enum
- module
- function overload

A target list may contain any combination of supported target kinds.

Signatures and interfaces do not support target-local or physically nested declarations. A `co.lang.signature` or `co.lang.interface` cannot be annotated with `@co.dap.local` and cannot appear in a local target set. A signature may declare abstract, fixed, or generic **type-component specifications** as part of its module contract; these are contract requirements rather than local or nested type definitions. Interfaces do not declare signature type components.

Other declaration kinds are not target-local unless a later section explicitly permits them.

### Target-Set Syntax

The `for` argument accepts either:

1. one declaration reference; or
2. a non-empty list of declaration references.

The scalar form is equivalent to a singleton target list:

```folang
@co.dap.local(for=hr.employee.Employee)
```

```folang
@co.dap.local(for=[hr.employee.Employee])
```

The scalar form is canonical when only one target is required. Use the list form when two or more declarations require access:

```folang
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.employee.EmployeeService,
        hr.employee.EmployeeValidator
    ]
)
// EmployeeState.fol
_ co.lang.enum = {
    Active,
    Inactive
}
```

Rules:

- the target list must contain at least one declaration reference
- each reference must resolve independently and unambiguously
- duplicate references to the same resolved declaration are a compiler error
- list order does not affect visibility or declaration identity
- the target list is a closed set; access is not granted to declarations omitted from it
- repeated `@co.dap.local` annotations are not used to accumulate targets; use one annotation with one complete target list

Invalid:

```folang
// State.fol
@co.dap.local(for=[])
_ co.lang.struct = { ... }
// ❌ empty target list
```

```folang
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.employee.Employee
    ]
)
// State.fol
_ co.lang.struct = { ... }
// ❌ duplicate resolved target
```

### Declaration-Reference Syntax

Each `for` entry is a compiler-resolved declaration reference, not a string, runtime expression, function call, or `co.meta.symbol` value.

For a non-function target, use only its complete qualified declaration name:

```folang
@co.dap.local(for=hr.employee.Employee)
@co.dap.local(for=hr.employee.EmployeeStatus)
@co.dap.local(for=hr.employee.EmployeeRules)
```

For a function target, use its complete qualified function signature because FoLang permits overloads:

```folang
@co.dap.local(
    for=hr.employee.Employee.calculate(co.lang.decimal)->()
)
```

Function references in a target list follow the same rule:

```folang
@co.dap.local(
    for=[
        hr.employee.Employee.calculate(co.lang.decimal)->(),
        hr.employee.Employee.calculate(co.lang.int)->(),
        hr.employee.Employee.validate()->(co.lang.bool)
    ]
)
// CalculationState.fol
_ co.lang.struct = { ... }
```

Parameter names are not part of the reference:

```folang
// ✅ canonical
hr.employee.Employee.calculate(co.lang.decimal)->()

// ❌ parameter names are not declaration identity
hr.employee.Employee.calculate(amount co.lang.decimal)->()
```

The parameter and return types must match one exact overload. An abbreviated function name, unresolved target, or ambiguous overload is a compiler error.

```folang
@co.dap.local(for=hr.employee.Employee.calculate)
// ❌ ambiguous when calculate is overloaded
```

Each listed overload is an independent target. Listing one overload does not grant access to sibling overloads with the same function name.

### Same-Package Requirement

The target-local declaration and **every declaration in its local target set** must belong to the same exact package. The compiler compares their complete folder-derived package identities; matching only a parent package, package family, import alias, library, or source root is not sufficient.

For a function target, the target package is the package that owns the function's enclosing class, struct companion unit, module, unit, or other legal function container. A target-local function is checked in the same way: the package containing its legal function-owning declaration must match the package of every listed target.

The invariant is:

```text
package(target-local declaration)
    == package(target 1)
    == package(target 2)
    == ...
```

```folang
// hr/employee/Employee.fol
_ co.lang.class = { ... }

// hr/employee/EmployeeService.fol
_ co.lang.class = { ... }

// hr/employee/EmployeeState.fol
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.employee.EmployeeService
    ]
)
// EmployeeState.fol
_ co.lang.enum = { Active, Inactive }
// ✅ all declarations belong to package hr.employee
```

A declaration in another package cannot participate in the local target set, even when ordinary package visibility, imports, aliases, parent/subpackage relationships, or library membership would otherwise make the declaration resolvable.

```folang
// EmployeeState is declared in package hr.internal
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.internal.EmployeeService
    ]
)
// EmployeeState.fol
_ co.lang.enum = { Active, Inactive }
// ❌ local declaration and every target must have the same package
```

```folang
// CalculationState is declared in package hr.employee.internal
@co.dap.local(
    for=hr.employee.Employee.calculate(co.lang.decimal)->()
)
// CalculationState.fol
_ co.lang.struct = { ... }
// ❌ subpackages are distinct packages; both sides must be exactly hr.employee
```

The same-package rule prevents `@co.dap.local` from becoming a cross-package friendship or visibility-bypass mechanism. A local declaration is a selectively shared implementation detail inside one package, not an imported helper attached from elsewhere.

### Visibility Domain

A target-local declaration may be resolved only from:

1. each exact declaration in its local target set; and
2. lexical scopes contained within each listed target's implementation.

For targets `A`, `B`, and `C`, the visibility domain is the union of their implementation scopes:

```text
visibility(local declaration)
    = scope(A) ∪ scope(B) ∪ scope(C)
```

It is not an intersection, and code does not need to be simultaneously inside every target.

Consequences:

- a declaration local to a class is available to that class body and its methods
- a declaration local to a module is available to that module body and its functions
- a declaration local to a struct is available while resolving that struct declaration
- a declaration local to an enum is available while resolving that enum declaration
- a declaration local to a function is available only in the body of the exact targeted overload and its nested statement scopes
- when several targets are listed, each listed target receives access independently
- sibling declarations and sibling overloads not listed cannot resolve it
- subclasses, companion units, extensions, callees, callers, and related declarations do not gain access automatically
- visibility is not inherited, transitive, or propagated through calls, composition, embedding, imports, or other local declarations
- it cannot be imported, exported, or projected through a library surface

```folang
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.employee.EmployeeService
    ]
)
// EmployeeState.fol
_ co.lang.enum = {
    Active,
    Inactive
}

// Employee.fol
_ co.lang.class = {
    state EmployeeState; // ✅ listed target
}

// EmployeeService.fol
_ co.lang.class = {
    state EmployeeState; // ✅ listed target
}

// Payroll.fol
_ co.lang.class = {
    state EmployeeState; // ❌ not listed
}
```

### Interaction with Ordinary Visibility

`@co.dap.local` establishes the final visibility boundary. Ordinary visibility annotations such as `@co.dap.public`, `@co.dap.package`, `@co.dap.protected`, or `@co.dap.private` cannot widen the declaration beyond its closed local target set.

```folang
@co.dap.public
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.employee.EmployeeService
    ]
)
// EmployeeState.fol
_ co.lang.enum = { Active, Inactive }

```

`EmployeeState` is still visible only to the listed targets. The ordinary visibility annotation is redundant for external resolution and may produce a compiler warning, but it never overrides `@co.dap.local`.

When most or all declarations in a package require access, ordinary package visibility should be used instead of maintaining a large target list.

### Source Placement and Identity

A target-local declaration remains in the source position normally required for its declaration kind:

- a class, struct, enum, or module remains a normal primary declaration and follows the one-primary-declaration-per-package-file rule
- a function remains inside a unit, class, module, or another declaration context that normally permits functions
- `@co.dap.local` does not permit a free function to appear loose at package-file scope

The declaration retains a stable package-owned compiler identity. Its package identity must be exactly the same as the package identity of every target. It does not become physically nested and is not addressed as `Target.LocalName`.

```text
hr.employee.EmployeeState
    local-for [
        hr.employee.Employee,
        hr.employee.EmployeeService
    ]
```

The normalized local target set is declaration metadata. Target-list order does not create a different declaration identity.

### No Implicit Relationship or Privilege

`@co.dap.local` changes visibility only. It does not imply:

- composition
- embedding
- inheritance
- lifecycle ownership
- memory ownership
- friendship or privileged private-member access
- automatic membership in any listed target
- shared runtime state among the targets

Composition must still be written explicitly through a field or another supported relation. Embedding remains a separate facility: only struct and enum declarations are eligible for embedding, and each use must follow the embedding rules defined for that declaration kind.

### Escape Restrictions

A target-local type must not leak outside its complete visibility domain.

It cannot appear in:

- a public, package, or protected API visible outside the local target set
- a library surface
- the parameter or return types of a function to which it is local
- a field or method signature that exposes it beyond the local target set
- an exported generic specialization or type alias

A value whose static type is target-local must be converted to an externally visible type before leaving the union of the listed targets' visibility domains.


### Usage

```folang
@co.dap.public
@co.dap.local(
    for=[
        hr.employee.Employee,
        hr.employee.EmployeeService
    ]
)
// EmployeeState.fol
_ co.lang.enum = { Active, Inactive }

// Employee.fol
_ co.lang.struct={

    state EmployeeState;
}

```


### Invalid Physical Nesting

```folang
// Employee.fol
_ co.lang.class = {
    Address co.lang.struct = { ... } // ❌
}

// EmployeeModule.fol
_ co.lang.module = {
    Config co.lang.struct = { ... } // ❌
}

process()->() = {
    State co.lang.enum = { Ready, Done } // ❌ named type declaration
}

// EmployeeContract.fol
_ co.lang.signature = {
    Employee co.lang.struct; // ❌
}

// EmployeeApi.fol
_ co.lang.interface = {
    Result co.lang.struct; // ❌
}
```

The `process` example rejects the named enum declaration; it does not reject ordinary local functions or anonymous expressions. Those are permitted only under the explicit exceptions in [Physical Nesting Rules](#physical-nesting-rules).

Use separately declared package declarations, composition, embedding where allowed, and `@co.dap.local` when a closed set of declarations requires selective access.

There is another annotation `@co.dap.nested` which is similar to local but captures target state.

@co.dap.nested(target=hr.emp.Employee)

Instead of `for` we use `target` and `target` is always a single fully qualified type/function.

`@co.dap.nested` bechaves exactly like nested or inner classes or functions

Comparision Table

|Annotation|attribute|multiple targets|capture the target state|
|---|---|---|---|
|@co.dap.local  | for | ✅ as list for single target can mention without list syntax| ❌|
|@co.dap.nested | target | ❌ | ✅ |

---

## `@co.dap.inner` Declarations

`@co.dap.inner` is an association annotation. It does **not** create a
physically nested type, method, or function. The annotated declaration remains
in its ordinary legal source location, retains its package-owned identity, and
must satisfy the same exact-package rule that applies to `@co.dap.local` and
`@co.dap.nested`.

Unlike `@co.dap.local` and `@co.dap.nested`, `@co.dap.inner` does not contain a
`for` or `target` attribute. Its target relationship is established at the use
site by explicitly embedding or attaching the declaration to a legal target.
An `@co.dap.inner` declaration cannot be used as a standalone declaration.

### Usage

```folang
@co.dap.public
// EmployeeState.fol
@co.dap.inner
_ co.lang.enum = {
    Active,
    Inactive
}

// Employee.fol
_ co.lang.struct = {
    EmployeeState;
    state EmployeeState;
}
```

The use of `EmployeeState;` establishes the inner association for this target.
The annotation itself does not name the target.

### Scope of `@co.dap.inner` Executable Declarations

For an `@co.dap.inner` function or method, parameters and local declarations
resolve normally. A free runtime name is resolved from the **lexical context of
the active call or attachment site**, not from the declaration site of the
separately declared `@co.dap.inner` function.

The lookup order is:

```text
1. parameters and local declarations of the @co.dap.inner function or method
2. the innermost lexical scope at the active call or attachment site
3. enclosing lexical scopes of that call or attachment site
4. statically resolved types, annotations, imports, and built-in co.* names
5. compiler error
```

This model is called **call-site lexical-context resolution**.

It is distinct from an ordinary inner function. An ordinary inner function is
physically declared inside another function and resolves free runtime names
from its enclosing lexical **declaration** context. An `@co.dap.inner` function
is declared separately and therefore obtains its runtime context from the
lexical scope in which it is actively attached or called.

It is also distinct from `@co.dap.dynamicscope`. Call-site lexical-context
resolution does not search arbitrary runtime caller frames beyond the lexical
scope chain of the active call site, and it does not use mixed-scope fallback.
At every statically known use or call site, the compiler validates that each
required runtime binding exists, is definitely initialized, has a compatible
type, and permits the requested access or mutation.

For `@co.dap.inner` types and other non-executable declarations, ordinary static
name and type resolution continues to apply. Call-site lexical-context
resolution concerns only free runtime names used by executable function or
method bodies.

An implementation may lower an `@co.dap.inner` declaration by copying,
inlining, specialization, or another equivalent mechanism. Such lowering is
not observable language semantics; the association and scope rules above are
normative.

### Exception

> `@co.dap.inner`, `@co.dap.local`, and `@co.dap.nested` cannot be used with
> `co.lang.cstruct`.

---

## Statements
   
A statement is a complete executable or declarative instruction. It may contain one or more expressions and may change program state, control execution, introduce declarations, or produce observable effects.

Common statement categories in `Folang`

   1. Declaration Statement
   2. Initialization Statement
   3. Expression Statement
   4. Conditional Statement
   5. Loop Statement etc,.

---

## Expressions

An expression is a construct that evaluates to a value, produces an effect, or both, and may be contained within a larger expression or statement.

## Expression Evaluation Order

FoLang defines a deterministic evaluation order for expressions.

Expression structure is determined first by:

1. explicit grouping with parentheses;
2. operator precedence;
3. operator associativity.

After the expression structure has been determined, operands and subexpressions are evaluated from left to right in their source-code order, except where a construct explicitly defines conditional, lazy, asynchronous, concurrent, or otherwise specialized evaluation behavior.

### Ordinary Expressions

For an ordinary expression:

```folang
result = first() + second();
```

the evaluation order is:

1. evaluate `first()`;
2. evaluate `second()`;
3. apply the `+` operator;
4. assign the resulting value to `result`.

### Precedence and Evaluation Order

Left-to-right evaluation does not override operator precedence.

For example:

```folang
result = first() + second() * third();
```

operator precedence determines the expression structure as:

```folang
result = first() + (second() * third());
```

The function calls are evaluated in source order:

1. evaluate `first()`;
2. evaluate `second()`;
3. evaluate `third()`;
4. multiply the results of `second()` and `third()`;
5. add the multiplication result to the result of `first()`;
6. assign the final result to `result`.

Therefore, precedence determines how values are combined, while evaluation order determines when each operand or subexpression is evaluated.

### Function Calls

For a function call, the callable expression is evaluated first, followed by the arguments from left to right.

```folang
result = calculate(first(), second(), third());
```

The evaluation order is:

1. evaluate the callable expression `calculate`;
2. evaluate `first()`;
3. evaluate `second()`;
4. evaluate `third()`;
5. invoke `calculate` with the resulting argument values;
6. assign the returned value to `result`.

Name resolution performed at compile time does not constitute runtime evaluation.

### Method Calls

For a method call, the receiver expression is evaluated first, followed by the arguments from left to right.

```folang
result = getEmployee().calculate(first(), second());
```

The evaluation order is:

1. evaluate `getEmployee()`;
2. resolve the resulting object as the method receiver;
3. evaluate `first()`;
4. evaluate `second()`;
5. invoke `calculate`;
6. assign the returned value to `result`.

The receiver expression must be evaluated exactly once.

### Member Access

For member access, the expression producing the containing object is evaluated before the member is accessed.

```folang
value = getEmployee().address.city;
```

The evaluation order is:

1. evaluate `getEmployee()`;
2. access `address` on the resulting object;
3. access `city` on the resulting address;
4. assign the resulting value to `value`.

Each intermediate receiver is evaluated only once.

### Indexing

For an indexing expression, the indexed object is evaluated first, followed by index expressions from left to right.

```folang
value = getMatrix()[row()][column()];
```

The evaluation order is:

1. evaluate `getMatrix()`;
2. evaluate `row()`;
3. perform the first indexing operation;
4. evaluate `column()`;
5. perform the second indexing operation;
6. assign the resulting value to `value`.

### Assignment

For assignment, FoLang evaluates the assignment target first, then evaluates the right-hand-side expression, and finally performs the assignment.

```folang
array[index()] = calculate();
```

The evaluation order is:

1. evaluate `array`;
2. evaluate `index()`;
3. determine the destination location;
4. evaluate `calculate()`;
5. assign the resulting value to the destination.

Determining the assignment target does not read the previous value stored at that location unless the assignment operation explicitly requires it.

### Simple Variable Assignment

For assignment to a simple variable:

```folang
result = first() + second();
```

the evaluation order is:

1. determine the binding represented by `result`;
2. evaluate `first()`;
3. evaluate `second()`;
4. apply the `+` operator;
5. assign the resulting value to `result`.

### Compound Assignment

A compound assignment evaluates its target exactly once.

```folang
array[index()] += calculate();
```

The evaluation order is:

1. evaluate `array`;
2. evaluate `index()`;
3. determine the destination location;
4. read the current value from that location;
5. evaluate `calculate()`;
6. apply the `+` operator;
7. assign the resulting value to the same destination.

The expression above must not behave as though it were expanded into a form that evaluates `array` or `index()` more than once.

### Multiple Assignment

When an assignment contains multiple right-hand-side expressions, those expressions are evaluated completely from left to right before values are assigned to their targets.

```folang
a, b = first(), second();
```

The evaluation order is:

1. determine the target for `a`;
2. determine the target for `b`;
3. evaluate `first()`;
4. evaluate `second()`;
5. assign the first resulting value to `a`;
6. assign the second resulting value to `b`.

Right-hand-side evaluation must complete before any target receives its new value.

This permits value exchange without an explicit temporary variable:

```folang
a, b = b, a;
```

### Operator Expressions

Built-in and user-defined operators follow the same operand-evaluation rules.

For a binary operator:

```folang
left() + right()
```

FoLang evaluates `left()` before `right()`.

For a prefix unary operator:

```folang
-operand()
```

FoLang evaluates `operand()` before applying the operator.

For a postfix unary operator:

```folang
operand()!
```

FoLang evaluates `operand()` before applying the operator.

Operator overloading must not change the specified evaluation order of operands.

### Short-Circuit Boolean Operations

Short-circuit Boolean operators evaluate the left operand first.

The right operand is evaluated only when it is required to determine the result.

For logical AND:

```folang
left() && right()
```

the evaluation order is:

1. evaluate `left()`;
2. when the result is false, return false without evaluating `right()`;
3. otherwise, evaluate `right()` and use its Boolean result.

For logical OR:

```folang
left() || right()
```

the evaluation order is:

1. evaluate `left()`;
2. when the result is true, return true without evaluating `right()`;
3. otherwise, evaluate `right()` and use its Boolean result.

A skipped operand produces no value, mutation, side effect, or error.

### Conditional Expressions

A conditional expression evaluates its condition first and then evaluates exactly one selected result expression.

```folang
result = condition()
    .return(whenTrue())
    .otherwise.return(whenFalse());
```

The evaluation order is:

1. evaluate `condition()`;
2. when the condition is true, evaluate `whenTrue()` only;
3. otherwise, evaluate `whenFalse()` only;
4. assign the selected result to `result`.

The unselected expression must not be evaluated.

### Conditions and Branches

For a condition-and-branch construct, conditions are evaluated sequentially from left to right.

```folang
firstCondition().do({
    firstBranch();
}).otherwise(secondCondition()).do({
    secondBranch();
}).otherwise.do({
    finalBranch();
});
```

The evaluation order is:

1. evaluate `firstCondition()`;
2. when true, execute `firstBranch()` and skip the remaining conditions and branches;
3. otherwise, evaluate `secondCondition()`;
4. when true, execute `secondBranch()` and skip the final branch;
5. otherwise, execute `finalBranch()`.

Only the selected branch is evaluated.

### Pattern Matching

A pattern-matching subject is evaluated exactly once.

Cases are examined in source order unless a particular matcher explicitly defines another ordering rule.

```folang
getValue().match
    .case(firstPattern => firstResult())
    .case(secondPattern => secondResult())
    .default(defaultResult());
```

The evaluation order is:

1. evaluate `getValue()` exactly once;
2. test `firstPattern`;
3. when it matches, evaluate `firstResult()` and stop;
4. otherwise, test `secondPattern`;
5. when it matches, evaluate `secondResult()` and stop;
6. otherwise, evaluate `defaultResult()`.

Results belonging to unmatched cases are not evaluated.

Pattern guards are evaluated only after their corresponding structural pattern has matched.

### Collection Literals

Elements of an array, list, tuple, set, map, or other collection literal are evaluated from left to right as they appear in the source.

```folang
values = [first(), second(), third()];
```

The evaluation order is:

1. evaluate `first()`;
2. evaluate `second()`;
3. evaluate `third()`;
4. construct the collection from the resulting values.

For a map entry, the key expression is evaluated before its corresponding value expression.

```folang
values = {
    firstKey(): firstValue(),
    secondKey(): secondValue()
};
```

The evaluation order is:

1. evaluate `firstKey()`;
2. evaluate `firstValue()`;
3. evaluate `secondKey()`;
4. evaluate `secondValue()`;
5. construct the map.

### Range Expressions

For a bounded range, the lower-bound expression is evaluated before the upper-bound expression.

```folang
range = lower() .. upper();
```

The evaluation order is:

1. evaluate `lower()`;
2. evaluate `upper()`;
3. construct the range.

An omitted bound is not evaluated and does not produce an implicit function call or side effect.

### Comprehensions and Iteration

A comprehension or iteration construct follows its own defined iteration semantics.

Within each iteration:

* source expressions are evaluated in the order declared;
* filter conditions are evaluated before result expressions;
* a result expression is evaluated only when its filters succeed;
* individual operand evaluation remains left to right.

The language does not implicitly evaluate comprehension iterations concurrently unless concurrency is explicitly requested.

### Lazy Expressions

An expression declared lazy is not evaluated when the lazy binding is created.

Its evaluation occurs only when demanded according to the rules of the corresponding lazy construct.

Once evaluation begins, the expression itself follows the normal FoLang evaluation-order rules unless the lazy construct specifies otherwise.

The specification for each lazy construct must also state whether its result is:

* evaluated once and cached;
* evaluated again for every demand;
* safe for concurrent demand;
* permitted to propagate errors more than once.

### Asynchronous and Concurrent Expressions

Left-to-right evaluation determines the order in which asynchronous or concurrent operations are created, invoked, or submitted.

It does not necessarily determine the order in which independently executing operations complete.

```folang
submit(first());
submit(second());
```

FoLang evaluates and submits `first()` before `second()`. However, completion order depends on the execution-model rules unless explicit ordering is requested.

Concurrent execution never arises implicitly from an ordinary expression. It must be introduced by an explicit FoLang concurrency, parallelism, asynchronous-execution, scheduling, or execution-model construct.

### Errors During Evaluation

When evaluation of a subexpression produces an error that prevents further evaluation, later subexpressions are not evaluated.

```folang
result = first() + failing() + third();
```

When `failing()` terminates evaluation with an error:

1. `first()` has already been evaluated;
2. `failing()` produces the error;
3. `third()` is not evaluated;
4. no final value is assigned to `result`.

The applicable error-handling rules determine whether the error is propagated, matched, converted, recovered from, or terminates execution.

### Side Effects

All observable side effects occur according to the defined evaluation order.

Observable effects include:

* object mutation;
* variable rebinding;
* input and output;
* file-system operations;
* network operations;
* synchronization;
* exception or error generation;
* interaction with foreign code;
* interaction with the runtime environment.

An implementation must not reorder operations when doing so could change any observable effect.

### Compiler Optimizations

A compiler may optimize, combine, eliminate, or internally reorder operations only when the transformation preserves all behavior required by this specification.

In particular, an optimization must not change:

* the resulting value;
* the order or presence of observable side effects;
* the identity or aliasing behavior of objects;
* the timing or presence of specified errors;
* the selected conditional or pattern-matching branch;
* the number of times an effectful expression is evaluated;
* synchronization or concurrency guarantees.

Pure internal operations may be reordered when no conforming FoLang program can observe the difference.

### Implementation Requirement

Every conforming FoLang implementation must preserve the evaluation order defined in this section.

An implementation may use any parser, intermediate representation, optimizer, runtime, or backend, but those implementation choices must not alter the externally observable behavior required by these rules.

---


## Operators

FoLang distinguishes a **symbolic spelling** from an **operator**. A sequence of
symbol characters is not automatically an operator merely because it resembles
one. Its syntactic role is determined by the grammar context in which the whole
spelling occurs.

For example:

```folang
a co.lang.int->(**);
```

In this declaration, `->` is the structural type-derivation marker and `**` is
pointer-degree metadata inside the derivation. Neither occurrence is parsed as
an expression operator. The same `**` spelling in `left ** right` is parsed as
the registered power operator because it occurs in an operator-expression
position.

FoLang uses one uniform implementation model for every expression operator. The
difference between built-in, pre-declared, and project-local custom operators is
only who registers the symbol and its parse properties.

```text
built-in operator
    symbol and parse properties registered by the language
    one or more implementations may already exist

pre-declared operator glyph
    symbol and parse properties registered by the language
    no implementation is required to exist

project-local custom operator
    symbol and parse properties registered once in srclib/operators/library.fol
    no implementation is contained in the operator declaration

all three categories
    implementations are ordinary mode=overload operator functions
    duplicate normalized operand signatures are errors
```

### Symbolic Runs, Classification, and Boundaries

After comments, literals, and closed scanner-known composite spellings such as
`@@new` and `@@init` are recognized, the lexer consumes each remaining complete
contiguous run of symbol characters as one symbolic token candidate. It does not
backtrack or split an unrecognized run into shorter valid tokens. Whitespace,
comments, and delimiters such as `(`, `)`, `[`, `]`, `{`, and `}` end a symbolic
run. Comment openers are recognized before ordinary symbolic-run scanning.

The complete run is then classified by its grammar context as one of the
following:

1. a fixed structural or declaration spelling such as `->`, `=>`, `:=`, or
   `?=`;
2. a contextual metadata spelling, such as a contiguous run of one or more `*`
   characters inside `->(...)` to express pointer degree;
3. a registered expression operator; or
4. an unrecognized symbolic token, which is a compilation error.

There is no fallback splitting. Therefore, when `++` is not registered:

```folang
++ a;       // error: unrecognized symbolic token `++`
a ++ b;     // error: unrecognized symbolic token `++`
a++b;       // error: the contiguous run is still the single candidate `++`

+ +a;       // valid: two `+` operators separated by whitespace
+(+a);      // valid: `(` creates the token and operand boundary
a + +b;    // valid: binary `+` followed by unary `+`
```

A registered expression operator whose spelling contains more than one symbol
character must have an explicit boundary on every operand-facing side. A
boundary may be whitespace, a comment, or an applicable delimiter. Boundary
presence is checked from the original source before whitespace and comments are
discarded. The rule is based on the operator's fixity:

- an infix multi-symbol operator requires a boundary before and after it;
- a prefix multi-symbol operator requires a boundary after it;
- a postfix multi-symbol operator requires a boundary before it.

Suppose `+-` is registered as an infix operator:

```folang
a +- b;       // valid
(a)+-(b);     // valid: the parentheses provide both operand boundaries
a+-b;         // error: missing boundaries
a +-b;        // error: missing the right boundary
a+- b;        // error: missing the left boundary
a + -b;       // valid: separate `+` and unary `-` operators
a + (-b);     // valid: the parenthesis separates the operators
```

This boundary requirement applies uniformly to built-in, pre-declared, and
custom multi-symbol expression operators. It does not apply merely because a
structural token or metadata spelling contains multiple symbols. Thus
`co.lang.int->(**)` remains valid without spaces around `->` or inside the
pointer metadata.

The statement-level definition spellings `:=` and `?=` are not expression
operators, but they deliberately use the analogous two-sided boundary rule.
Write `name := value` and `name ?= value`; compact forms such as `name:=value`
and `name?=value` are invalid.

Operator `mode=override` and `mode=extends` are unsupported. Ordinary class
method overriding through `@co.dap.override` remains a separate class feature.

### Operator Implementations

An operator implementation is a normal function with operator metadata. It
cannot be declared loose at package scope. Every implementation must be in one
of these legal function-owning locations:

| Operand owner | Required implementation location |
|---|---|
| built-in type | an `@co.dap.extension` function inside a unit |
| `co.lang.struct` | the struct's same-package companion unit |
| `co.lang.class` | an operator method declared by the class |
| module, enum, union, interface, signature, `co.lang.cstruct` | unsupported |

An implementation uses `mode=overload`, or omits `mode` because omission is
shorthand for `mode=overload`. A one-character symbol may use a character
literal; a multi-character symbol must use a string literal:

```folang
@co.dap.operator(symbol='+', mode=overload)
add(left Employee, right Employee)->(Employee) = {
    ...
}

@co.dap.operator(symbol="+-", mode=overload)
combine(left Employee, right Employee)->(Employee) = {
    ...
}
```

FoLang does not attempt to prove that an overload follows the conventional
meaning of its symbol. A developer may overload `+` with any implementation.
Using one symbol consistently for one recognizable concept is recommended for
readability, but it is not a compiler-enforced semantic restriction.

For a receiverless struct-companion operator function, the first declared
operand must have the matching struct type. A matching struct type in a later
operand is insufficient. For a unary operator, the sole operand must have the
matching struct type.

A matching instance receiver contributes the first operand. A matching type
receiver establishes ownership but contributes no operand; its ordinary
parameter list is the complete operator signature. In a class, an ordinary or
`@co.dap.instance` operator method has an implicit `this` first operand, while
`@co.dap.static` and `@co.dap.class` operator methods use only their declared
parameters and require the first declared operand to have the enclosing class
type.

After receiver normalization, the operand count must match the registered arity
of the operator. Each distinct normalized operand signature is an overload. A
second implementation with the same normalized signature is a duplicate and is
rejected. This rule applies equally to built-in operators, pre-declared glyphs,
and project-local custom operators.

### Language-Owned Operators

Language-owned operators already have a registered symbol, fixity, precedence,
associativity, and arity. They must not be redeclared in the project operator
source. They receive additional implementations through `mode=overload`.

This category includes:

1. ordinary built-in operators such as `+`, `-`, `*`, and `==`;
2. pre-declared operator glyphs whose parse properties are fixed by the
   language but which may initially have no implementation.

#### Pre-Declared Operator Glyphs

The language pre-declares the documented mathematical and modifier glyph set,
including `λ`, `∀`, `∃`, `∪`, `⇛`, `∂`, `⊥`, `↓`, `⇓`, `○`, `𝚷`, and `𝒯`, with
fixed parse properties. These symbols are reserved against project-local
re-declaration, but they are available for implementation through ordinary
operator overloads:

```folang
// ∪ is already registered by the language; only its implementation is supplied.
@co.dap.operator(symbol='∪', mode=overload)
union(left Set, right Set)->(Set) = {
    ...
}
```

Until a visible implementation matches the operand types, an expression using a
pre-declared glyph parses successfully and then fails during operator
resolution.

Hard-reserved spellings such as `::=`, `->>`, `<->`, backtick, backslash, `#`,
and comment openers are different: they are not overloadable or declarable
unless a later language revision explicitly assigns them operator semantics.

### Project-Local Custom Operator Source

A custom operator is a symbol that is neither language-owned nor hard-reserved. Its symbol and parse properties are registered only in the fixed application-local operator bootstrap surface:

```text
<application-root>/srclib/operators/library.fol
```

`srclib/` and `srclib/operators/` are not packages. `operators/` is the one standardized operator slot under `srclib/`. If it is absent, the application introduces no project-local custom operator symbols. If it is present, it must contain exactly one file named `library.fol` and no additional files or subdirectories. The fixed `library.fol` name is shared by all `srclib/` library slots; the enclosing `operators/` directory selects the dedicated operator bootstrap semantics.

The fixed file is parsed by the dedicated operator-source lexer and parser before the ordinary FoLang lexer and parser run. Its filesystem position already establishes the operator bootstrap context, so no library-kind annotation is used:

```folang
// srclib/operators/library.fol
_ co.lang.library = {

    ⊗ co.lang.operator = {
        fixity: co.operator.fixity.infix,
        precedence: 60,
        associativity: co.operator.associativity.left,
        arity: co.operator.arity.binary,
        commutative: co.const.false,
        idempotent: co.const.false,
        identity: co.const.none,
        foldable: co.const.false,
        vectorizable: co.const.false,
        distributes_over: [],
        desugar: "intrinsic:tensor_product"
    };

    +- co.lang.operator = {
        fixity: co.operator.fixity.infix,
        precedence: 60,
        associativity: co.operator.associativity.left,
        arity=co.operator.arity.binary
    };
}
```

`_ co.lang.library` identifies the structural operator bootstrap surface. It is not an ordinary importable library surface and does not produce a `.folenc` artifact. Its body may contain only `co.lang.operator` declarations. Imports, functions, types, variables, expressions, implementation packages, and nested libraries are forbidden.

`co.lang.operator` is valid only in this dedicated source grammar. It is not an ordinary FoLang declaration kind and cannot appear in package source, `src/<entryfilename>.fol`, or an ordinary library surface.

#### Operator declaration attributes

Every custom declaration must contain each required parse attribute exactly
once. Optional metadata may appear at most once. Duplicate and unknown
attributes are errors.

| Attribute | Required | Accepted value | Meaning |
|---|---:|---|---|
| `fixity` | yes | `infix`, `prefix`, `postfix`, or a reserved future fixity name (`circumfix`,`postcircumfix`,`precircumfix`,etc) | parser placement |
| `precedence` | yes | decimal integer from `0` through `100` | binding strength |
| `associativity` | yes | `left`, `right`, or `none` | grouping for equal precedence |
| `arity` | yes | `unary`, `binary`, `ternary`, or a positive decimal integer | normalized operand count |
| `commutative` | no | `co.const.true` or `co.const.false` | semantic/optimization metadata |
| `idempotent` | no | `co.const.true` or `co.const.false` | semantic/optimization metadata |
| `identity` | no | a FoLang literal, including `co.const.none` | identity-value metadata |
| `foldable` | no | `co.const.true` or `co.const.false` | compile-time folding metadata |
| `vectorizable` | no | `co.const.true` or `co.const.false` | vectorization metadata |
| `distributes_over` | no | a list of quoted operator symbols | distributivity metadata |
| `desugar` | no | a string literal | intrinsic or lowering hook metadata |

In the alpha profile, only `infix`, `prefix`, and `postfix` are implemented.
Consequently, an alpha `prefix` or `postfix` declaration must be unary and an
alpha `infix` declaration must be binary. Other fixity names remain reserved for
future delimiter and slot grammars and are rejected by the alpha conformance
profile.

The four required parse attributes construct the tokenizer and precedence
table. Optional semantic/optimization attributes do not change tokenization or
parsing.

#### Declaration and implementation are separate

The operator library registers only the symbol and metadata. It contains no
implementation. Implementations use the same `mode=overload` form as built-in
and pre-declared operators:

```folang
// vector/Vector.comp.unit.fol
_ co.lang.unit = {

    @co.dap.operator(symbol='⊗', mode=overload)
    tensorProduct(left Vector, right Vector)->(Vector) = {
        ...
    }
}
```

A custom symbol has exactly one declaration in the operator library, but it may
have any number of distinct implementation overloads in legal owners:

```text
one ⊗ declaration
    Vector ⊗ Vector -> Vector
    Matrix ⊗ Matrix -> Matrix
    Tensor ⊗ Tensor -> Tensor
```

The declaration cannot be duplicated, aliased, merged, selected, or remapped.
The implementations participate in ordinary operator overload resolution.

#### Symbol registration and exact recognition

The dedicated operator-source lexer reads the declaration name as one maximal
contiguous symbol run. After the operator table is built, the ordinary lexer
uses the same whole-run rule. A registered custom symbol is recognized only when
the complete run matches that symbol; an unknown run is rejected without being
split into shorter operators.

Multi-symbol operator uses must also satisfy the operand-boundary rule defined in
[Symbolic Runs, Classification, and Boundaries](#symbolic-runs-classification-and-boundaries).
Consequently, registering `+-` permits `a +- b`, not `a+-b`. Unicode is not
required for custom operators.

A custom symbol:

- must not be language-owned;
- must not be a hard-reserved spelling;
- must not contain `//` or `/*`, because a comment opener terminates the
  preceding symbolic run and is recognized before further operator matching;
- must have exactly one declaration in the operator library.

#### Symbols are global; implementations are resolved by scope and type

Once a custom symbol is registered—or a language-owned symbol is loaded—the
ordinary tokenizer recognizes it throughout that compilation. The parser can
therefore build the same operator expression in every package.

Whether an implementation is available is determined later through ordinary
operator resolution. Receiver-owned class and companion implementations follow
their normal lookup rules. Extension implementations require normal
`@co.ddap.use` activation. A symbol with no matching visible implementation
produces a resolution error, not a syntax error.

Using one symbol for one recognizable concept across its overloads is strongly
recommended for readability, but the compiler does not and cannot enforce that
recommendation. The type/signature rules, not the conventional meaning of the
symbol, determine whether an overload is legal.

### Operators Do Not Cross a Library Boundary

Operator notation is local compilation syntax, not a library API element. An
ordinary library surface exports named function signatures and boundary data
contracts. It exports no operator table.

```text
geometry.folenc
├── projected symbol table
└── compiled implementation/linkage        (no operator table)
```

A library may use custom or pre-declared operators internally. Those expressions
are resolved and lowered while compiling that library. Importing the resulting
artifact does not add any symbols or parse properties to the importing
compilation.

An application that wants operator notation for an imported boundary type must
register a local custom symbol when necessary and provide a legal local extension
implementation that delegates to the library's named public function.

### Bootstrap Order

```text
1. Classify the fixed application-root domains: src/, srclib/, lib/, and build/.
2. Validate src/: its only direct file is app.fol; all other direct entries are package directories.
3. Validate srclib/: only ffi/, system/, advanced/, dynamicvmrt/, and operators/ may occur directly beneath it.
4. For ffi/, system/, advanced/, and dynamicvmrt/, require exactly one direct library.fol and treat all other entries as internal package directories.
5. If srclib/operators/ exists, require exactly one direct library.fol and no other entries.
6. Parse srclib/operators/library.fol with the dedicated operator-source lexer/parser; if the operators slot is absent, continue with no project-local custom symbols.
7. Reject duplicate custom operator declarations, language-owned redeclarations, hard-reserved spellings, invalid attributes, and invalid alpha fixity/arity combinations.
8. Combine language-owned registrations with project-local custom declarations and build the immutable maximal-munch and precedence tables.
9. Discover application packages only below src/ and each source-library's private packages only below its fixed srclib/<kind>/ root.
10. Resolve packaged-library dependencies only from .folenc artifacts under lib/.
11. Run the ordinary FoLang lexer/parser and semantic pipeline over the selected entry, package, and library sources.
12. Resolve operator implementations by owner, scope, operand types, and normalized callable signature.
13. Serialize the validated frontend result as Protocol Buffers binary under <project-root>/build/ for backend consumption.
```

Imports contribute no operator metadata, so this bootstrap has no import-order
or transitive-dependency dependency.

---

## Forward / Extern Declarations

### Variables extern declaration

```folang
@co.dap.declare(extern)
someBool co.lang.bool;
```

### Functions forward declaration

```folang
@co.dap.declare(forward)
getEmployee(id co.lang.int)->(somepack.Employee);

// or — @co.dap.declare is optional for functions
getEmployee(id co.lang.int)->(somepack.Employee);
```

### Types external declaration

```folang
// Employee.fol
@co.dap.declare(extern)
_ co.lang.struct;

// or — @co.dap.declare is optional for types
_ co.lang.struct;
```

> For functions and types `@co.dap.declare` is optional. For variables it is required.



---
## Functions

FoLang does not allow free-flowing package functions. Package functions must be declared inside an ordinary `<Fragment>.unit.fol` file. Their public identity is the package member name, not the unit filename.



### Normal

```folang
// general.unit.fol
_ co.lang.unit = {

    fun1(k co.lang.int, b co.lang.char)->(co.lang.int, co.lang.char) = {
        // function body
    }
}
```
`folang` function can return multiple values

### Default Parameters

```folang
fun1 (k co.lang.int, b co.lang.char = 10)->(co.lang.int, co.lang.char)={
}
```

### Variadic Functions

Curried functions are not allowed to be variadic, and vice versa.

```folang
fun1 (k co.lang.int, ...b co.lang.char)->(co.lang.int, co.lang.char)={
}
```

### Optional Parameters

```folang
fun1(k? co.lang.int)->()={
    k.omitted.do({

    }).otherwise({

    });

}
```

### Named Parameters

```folang
fun1(~k co.lang.int)->()={

}
```

### Named Returns

```folang
doManythings(a co.lang.int, b co.lang.int->(&, meta={type=out}))->(r co.lang.int, e co.lang.exception)={}
```

### Function Delegates

```folang
@co.dap.delegate someDelegate co.lang.delegate = (a co.lang.int, b co.lang.int) -> (co.lang.int, co.lang.int);
```

### Function Chaining

```folang
fetchEmployee(empId co.lang.string)->(Employee)=>>empMod.getEmployee(this, empId);
```
### Anonymous Functions

```folang
add := (a int, b int) -> (int) {
    this.return a + b;
};

res := (a int, b int) -> (int) {
    this.return a * b;
}(10, 20);
```
> for more details on functions please refer section [Functions in Details](#functions-in-detail)

---


## Functions in detail

### Inline

```folang
// math-functions.unit.fol
_ co.lang.unit = {
    @co.dap.inline
    add(a co.lang.int, b co.lang.int)->(co.lang.int) ={
        this.return a + b;
    }
}
```


### Anonymous Functions

```folang
add := (a int, b int) -> (int) {
    this.return a + b;
};

res := (a int, b int) -> (int) {
    this.return a * b;
})(10, 20);
```

### Lambda

Only allowed as an inline callback argument to receiver-qualified collection
operations (e.g. `map`, `filter`, `reduce`, `forEach`, `sortBy`, `groupBy`).
Transparent grouping around the member callee is permitted, so
`(nums.map)(|x| => x*x)` is equivalent to the ordinary call spelling. A bare
function call such as `map(|x| => x)` is not a collection-method context. Using
`|...|` anywhere else is a syntax/lint error.

```folang
// Callback literal syntax, shown only in its valid collection-call context
nums.map(|x| => x*x);
words.filter(|s| => s.len() > 3);
pairs.reduce(|acc, e| => acc + e, 0);
dict.map(|k, v| => v * 10);
list.sortBy(|a, b| => a.score - b.score);
```

The lambda must be a direct argument of the allowed collection call. That call
may itself be nested, for example `consume(nums.map(|x| => x*x))`; the enclosing
call does not make the lambda an argument of `consume`.

### Inner Function

```folang
myfun(a co.lang.int, b co.lang.int)->(co.lang.int)={
    p co.lang.int = 10;
    someother()->()={
        co.out.println(p);
    }
    someother();
    p = 20;
    someother();
}
```
The function `someother` is an ordinary inner function because it is physically
declared inside `myfun`. Its free runtime names are resolved from the lexical
declaration context of `myfun`; therefore `p` denotes the binding declared in
`myfun` regardless of where `someother` is called within its permitted lifetime.

`@co.dap.local`, `@co.dap.nested`, and `@co.dap.inner` apply to separately
declared functions and do not change the lexical-scope rule for an ordinary
inner function. They provide visibility, target-state capture, or call-site
association without requiring the function body to be physically placed inside
the target declaration.

The permitted local-function form above is useful when the inner function is small and
belongs naturally to one enclosing function. This is the named local-function
exception described in [Physical Nesting Rules](#physical-nesting-rules).


### Curried

```folang
add(first co.lang.int)(second co.lang.int)->(co.lang.int)={
    this.return first + second;
    
}
```

### Closure

```folang
adder() -> ((co.lang.int) -> co.lang.int) ={
    sum co.lang.int = 0;
    this.return  (x co.lang.int) -> (co.lang.int) = {
        sum += x;
        this.return sum;
    }
}
```

### Functions Taking and Returning Functions

#### Syntax 1 — Inline signature

```folang
someFunction (r (co.lang.int, co.lang.int)->(co.lang.int))->((co.lang.int)->(co.lang.int))={}
```

#### Syntax 2 — Named type alias

```folang
someFArg co.lang.type = (co.lang.int, co.lang.int)->(co.lang.int);
someFRet co.lang.type = (co.lang.int)->(co.lang.int);

someFunction (r someFArg)->(someFRet)={}
```

#### Syntax 3 — Function objects

```folang
someFArg co.lang.function = (a co.lang.int, b co.lang.int) -> (co.lang.int)={
    this.return a + b;
}

someFRet co.lang.function = (a co.lang.int) -> (co.lang.int)={
    this.return a * 2;
}

```

### Other ways to declare clsures/function objects and types/ curried functions

```folang
myobj co.lang.function = (a co.lang.int, b co.lang.int)->(co.lang.int)={
    this.return a + b;
}

add (a co.lang.int, b co.lang.int)->(co.lang.int){ this.return a + b; }
oObj co.lang.function = add;

funtype co.lang.type = (a co.lang.int, b co.lang.int)->(co.lang.int);

closure=(factor co.lang.int, val co.lang.int) ==>> factory * val;

curry = (factor co.lang.int) (x co.lang.int) ==>> x * factor;

```
---

### Associated Functions

For a user-defined struct, associated functions must be declared inside the same-package companion unit whose name matches the struct. For more details on associated function please refer section [Associated Functions in a Companion Unit](#associated-functions-in-a-companion-unit)

### Some Restrictions on Special Functions

1. **Special functions**
   - Curried functions
   - Functions with named arguments
   - Functions with optional arguments
   - Functions with default arguments
   - Variadic functions
   - Functions that take or return functions or function types
   - Dynamically scoped and mixed-scoped functions

2. **Restrictions**
   - They cannot be overloaded.
   - They cannot be used as callbacks.
   - They cannot participate in [Execution Models and Control Abstractions](#execution-models-and-control-abstractions-library-typeadvanced).
   
#### Scoping Rules for Functions

All functions in FoLang have a defined scope — the set of variables a function can access.

---

##### Default — Lexical / Static Scope

All of the following are **always** lexically scoped — this cannot be changed:
```
methods
inner methods
free functions
inner functions
target-local named functions
closures
lambdas
Generic Functions
Anonymous functions
Curried functions
```

Lexical scope means a function resolves names from its **declaration site**, not from the scope of its caller. Unit-level variables are forbidden, so a unit-scoped free function receives runtime values through parameters or introduces them locally. An ordinary inner function captures from the enclosing lexical declaration context. Calling that inner function does not replace its captured context with the caller's runtime scope.

```folang
// scope-example.unit.fol
_ co.lang.unit = {

    foo()->() = {
        x co.lang.int = 10;

        printX()->() = {
            co.out.println(x);  // ✅ x from the enclosing lexical scope
        }

        printX();
    }
}
```

---

##### `@co.dap.inner` — Call-Site Lexical Context

`@co.dap.inner` is not the syntax for an ordinary inner function. It annotates a
separately declared function or method that becomes associated with a target at
an explicit use or attachment site.

Its free runtime names use **call-site lexical-context resolution**:

```text
1. @co.dap.inner parameters and local declarations
2. the innermost lexical scope at the active call or attachment site
3. enclosing lexical scopes of that call or attachment site
4. statically resolved types, annotations, imports, and co.* names
5. compiler error
```

The compiler validates these requirements at every statically known use or call
site. This is a fixed property of `@co.dap.inner`; it is not selected through
`@co.dap.lexicalscope`, `@co.dap.dynamicscope`, or `@co.dap.mixedscope`.

This differs from ordinary lexical inner functions, whose free runtime names are
fixed by their declaration site, and from dynamically scoped associated
functions, which may search outward through the runtime caller activation chain.

---

##### Associated Functions — Additional Scope Options

Only associated functions support non-lexical scoping via annotations:

The examples below are members of the same-package `Employee` companion unit.

**`@co.dap.lexicalscope`** — default, explicit declaration
```folang
// Employee.comp.unit.fol
_ co.lang.unit = {
    @co.dap.lexicalscope
    (emp Employee) process()->() = {
        co.out.println(emp.name);   // ✅ declaration scope
    }
}
```

**`@co.dap.dynamicscope`** — accesses caller's scope
```folang
// Employee.comp.unit.fol
_ co.lang.unit = {
    @co.dap.dynamicscope
    (emp Employee) process()->() = {
        co.out.println(name);   // name comes from caller's scope
    }
}
```

**`@co.dap.mixedscope`** — accesses both scopes
```folang
// Employee.comp.unit.fol
_ co.lang.unit = {
    // caller scope takes priority — shadows declaration scope on conflict
    @co.dap.mixedscope
    (emp Employee) process()->() = {
        co.out.println(name);   // caller scope takes priority
        co.out.println(emp.id); // falls back to declaration scope
    }
}
```

---

##### Callback Scope Inside Dynamically Scoped Associated Functions

A callback block or lambda does not independently select lexical, dynamic, or mixed scope. The associated function that executes the callback determines how the callback's free runtime names are resolved.

Callback parameters and declarations made inside the callback always belong to the callback's local scope. Names that are not callback parameters or callback-local declarations follow the executing associated function's scope policy.

```folang
nums.reduce(|acc, e| => {
    total.value += e;
    acc + e;
}, 0);
```

For this example:

```text
acc, e              -> callback parameters
temporary locals    -> callback-local context
total               -> reduce's dynamic caller context
co.* and imports    -> normal built-in/import resolution
```

The callback syntax remains uniform. `reduce` is dynamically scoped, so unresolved runtime names in the callback are resolved through the active caller-context chain. A lexically scoped associated function would instead resolve free runtime names through its declaration context.

---

##### Why Dynamic Scope Exists — `.do`, `.loop`, `.each`, and Collections

FoLang's control-flow model is built on dynamically scoped associated functions. The executing function supplies the scope policy, so blocks do not require separate capturing and non-capturing forms.

```folang
x     co.lang.int = 10;
total co.lang.int = 0;
arr   co.lang.int->([5]) = [1, 2, 3, 4, 5];

// .do reads and modifies the caller's x
(x > 5).do({
    x.value = 20;
    co.out.println(x);
});

// .loop modifies the caller's x
(x > 0).loop({
    x.value -= 1;
});

// .each modifies the caller's total
arr.each(_, val).do({
    total.value += val;
});

// .filter, .map, and .reduce use the same dynamic caller context
nums.filter(|x| => x > 10);
nums.reduce(|acc, e| => {
    total.value += e;
    acc + e
}, 0);
```

The compiler does not create a separate capture description for each control block. It resolves names according to the scope mode of the associated function executing that block.

---

##### FoLang Control Flow Uses Dynamic Scope

```text
no if/else keywords    -> .do / .otherwise  — dynamic scope
no for/while keywords  -> .loop             — dynamic scope
no foreach keywords    -> .each             — dynamic scope
no in keyword          -> .contains         — dynamic scope
no map/filter keywords -> .map / .filter     — dynamic scope

all control flow is implemented as associated functions
control associated functions are dynamically scoped
caller variables are accessible and mutable under the dynamic lookup rules
block syntax requires no separate capture mode
```

---

##### Dynamic and Mixed Lookup Order

For `@co.dap.dynamicscope`, runtime variable lookup follows this order:

```text
1. function/callback parameters and local declarations
2. immediate caller activation context
3. caller activation contexts outward through the dynamic call chain
4. built-in `co.*` names and imported declarations visible to the source file
5. compiler error if no compatible declaration is available
```

For `@co.dap.mixedscope`, lookup follows this order:

```text
1. function/callback parameters and local declarations
2. immediate caller activation context
3. caller activation contexts outward through the dynamic call chain
4. lexical declaration context
5. built-in `co.*` names and imported declarations visible to the source file
6. compiler error if no compatible declaration is available
```

Local declarations shadow caller declarations. Caller declarations shadow declarations from the lexical declaration context in mixed scope.

Types, annotations, imports, and other compile-time declarations continue to use ordinary static resolution. Dynamic lookup applies to runtime bindings, not to the identity of types or imported APIs.

##### Call-Site Validation of Dynamic Requirements

A dynamically or mixed-scoped associated function may reference a caller-provided runtime name that is not declared in its lexical context:

```folang
@co.dap.dynamicscope
(Employee) addToTotal(value co.lang.int)->() = {
    total.value += value;
}
```

At every statically known call site, the compiler validates that the active caller-context chain provides a definitely initialized binding named `total` with a compatible type and permitted mutability. A call is rejected when the required binding cannot be proven to exist.

Because non-lexically scoped functions are non-first-class and non-escaping, their call sites remain statically identifiable. This avoids runtime string-based name lookup and allows the compiler to represent dynamic requirements using context and symbol-table references.

---

##### Scope Rules Summary

| Function Type | Lexical | Dynamic | Mixed |
|---|---|---|---|
| methods | ✅ declaration-site only | ❌ | ❌ |
| ordinary inner methods | ✅ declaration-site only | ❌ | ❌ |
| free functions | ✅ declaration-site only | ❌ | ❌ |
| ordinary inner functions | ✅ enclosing declaration context only | ❌ | ❌ |
| target-local named functions | ✅ declaration-site only | ❌ | ❌ |
| `@co.dap.inner` functions/methods | call-site lexical context | ❌ no unrestricted caller-chain lookup | ❌ |
| anonymous functions | ✅ declaration-site only | ❌ | ❌ |
| closures | ✅ declaration-site capture | ❌ | ❌ |
| lambdas/callback blocks | determined by executing associated function | determined by executing associated function | determined by executing associated function |
| associated functions | ✅ default | ✅ opt-in | ✅ opt-in |
| `.do` / `.loop` / `.each` | ❌ | ✅ built-in | ❌ |
| `.map` / `.filter` / `.reduce` | ❌ | ✅ built-in | ❌ |

##### Additional Restrictions

- dynamically or mixed-scoped functions cannot be returned
- dynamically or mixed-scoped functions cannot be stored as values
- dynamically or mixed-scoped functions cannot be passed as ordinary callbacks
- dynamic or mixed caller contexts cannot cross thread or process boundaries
- dynamic or mixed execution cannot continue after the caller frame ends
- dynamic or mixed-scoped functions cannot participate in [Execution Models and Control Abstractions (library type=advanced)](#execution-models-and-control-abstractions-library-typeadvanced)
- dynamically or mixed-scoped functions cannot be curried
- named, optional, variadic, and default-parameter forms follow the same non-escaping and call-site-validation rules


## Types

In ordinary package source, `co.lang.type`, type aliases, newtypes, opaque types, subtypes, supertypes, and parameterized `co.lang.type` constructors must be declared inside an ordinary `*.unit.fol` file. They are contributed directly to the package namespace. Entry files, signatures, modules satisfying signature type components, and dedicated library surfaces follow their own explicitly stated rules.

Examples in this section that show only a type declaration are fragments from inside a legal unit or other legal enclosing declaration.

**The three axes — each adds one new power:**

**Axis 1: Polymorphism (terms depend on types)**
```
// "Give me a type, I'll give you a value"

// Without: write separate functions
identityInt(x int) → int={}
identityStr(x string) → string={}

// With: one function works for all types
identity(T)(x T) → T={}

// This is generics / parametric polymorphism
// System F, Java generics, your @co.dap.generic
```

**Axis 2: Type operators (types depend on types)**
```
// "Give me a type, I'll give you a type"

List(Int)     → List ={}            // List of ints type → type
Map(String, Int) → Map  ={}         // type → type → type
Option(T)     → Some(T) | None;     // type constructor

// This is kinds / higher-kinded types
// Your folang: Option(T) co.lang.type = Some(T) | None()
```

**Axis 3: Dependent types (types depend on values)**
```
// "Give me a value, I'll give you a type"

Vector(3)      → array of exactly 3 elements
Matrix(2, 3)   → 2x3 matrix
NonZero(n)     → proof that n ≠ 0

// The type changes based on a runtime value
divide(a int, b NonZero(int)) → int={}
// compiler PROVES b can't be zero
```

![Lambda Cube ](lambda-cube.svg)
 
---

## Dependent Types

### Type-Level Functions — Functions That Return Types

A function that accepts a type or value and returns a type is a **type-level function**. When its result depends on a value argument, it defines or selects a dependent type. This is distinct from a parameterized `co.lang.type` constructor such as `Option(T)`, whose declaration directly defines a family of types.
```folang
// Vector — value-indexed type-level function
// takes  → co.lang.int (size)
// returns → co.lang.dependentType (a type)
Vector(n co.lang.int)->(co.lang.dependentType) =
    co.lang.int->([n]);

// calling Vector(3) returns a TYPE at compile time
// that type is co.lang.int->([3])
v3 Vector(3) = [1, 2, 3];    // type is Vector(3)
v4 Vector(4) = [1, 2, 3, 4]; // type is Vector(4) — different type!

// Vector(3) ≠ Vector(4) — completely different types
// size is part of the type — compiler knows at compile time
```

---
### More About Type 

    Name(T) co.lang.data = variants;
        → concrete ADT type-constructor definition
        → right-hand-side definition is mandatory

    Name(T) co.lang.type;
        → abstract type-constructor requirement
        → permitted only inside a signature

    Name(T) co.lang.type = ExistingType(T);
        → concrete type alias or signature-component binding
        → permitted in modules and ordinary type declarations

---

### A Type-Level Function Returns a Type
```
Vector        →  type-level function
Vector(3)     →  function call → returns type co.lang.int->([3])
Vector(4)     →  function call → returns type co.lang.int->([4])

just like:
    add(1, 2)  →  returns a value  (3)
    Vector(3)  →  returns a type   (int[3])
```

---

### Compiler Enforced Size Safety
```folang
// dot product — only valid for same size vectors
// compiler enforces this via dependent types
dotProduct(a Vector(n), b Vector(n))->(co.lang.int) = {
    // n is same for both — compiler verified
}

v3 Vector(3) = [1, 2, 3];
v4 Vector(4) = [1, 2, 3, 4];

dotProduct(v3, v3);   // ✅ same type Vector(3)
dotProduct(v3, v4);   // ❌ compiler error — Vector(3) ≠ Vector(4)
```

---

### Matrix — Two-Parameter Type-Level Function
```folang
// Matrix — takes rows and cols, returns dependent type
Matrix(r co.lang.int, c co.lang.int)->(co.lang.dependentType) =
    co.lang.int->([r, c]);

m34 Matrix(3, 4) = [[1,2,3,4],[5,6,7,8],[9,10,11,12]];
m45 Matrix(4, 5) = ...;

// matrix multiply — cols of A must equal rows of B
// n must match — compiler verified
multiply(a Matrix(r, n), b Matrix(n, c))->(Matrix(r, c)) = {
    // compiler ensures dimensions are compatible
}

multiply(m34, m45);   // ✅ Matrix(3,4) × Matrix(4,5) = Matrix(3,5)
multiply(m34, m34);   // ❌ compiler error — 4 ≠ 3
```

---

### Stack — Value and Type Parameter
```folang
// Stack — takes size and element type
Stack(n co.lang.int, T co.lang.type)->(co.lang.dependentType) =
    T->([n]);

s Stack(10, co.lang.int)    = ...;  // stack of max 10 ints
t Stack(5,  co.lang.string) = ...;  // stack of max 5 strings
```

---

### Type Is Value + Kind Combined
```
Vector(3):
    kind  = Vector    (what it is)
    value = 3         (how many)
    type  = Vector(3) (both together — the dependent type)

Vector(3) ≠ Vector(4)   ←  different types entirely
Vector(3) = Vector(3)   ←  same type
```

---

### Type Constructors and Type-Level Functions

```folang
// option.unit.fol
_ co.lang.unit = {
    // Parameterized type declaration: Option is a type constructor.
    Option(T) co.lang.type =
        Some(T) | None();

    // Value-indexed type-level function: Vector computes a dependent type.
    Vector(n co.lang.int)->(co.lang.dependentType) =
        co.lang.int->([n]);
}
```

`Option` and `Vector` both operate at the type level, but they are different declaration categories:

```text
Option(T) co.lang.type
    -> type-constructor declaration
    -> substitution produces Option(T)

Vector(n)->(co.lang.dependentType)
    -> type-level function
    -> computation produces a type
```

---

### Simple Dependent Type
```folang
identity(x co.lang.int)->(x.type) ={ this.return x; }

```
---

### Compile-Time and Runtime Values in Type-Related Computation

FoLang distinguishes value-indexed dependent types, compile-time type computation, runtime type descriptors, and runtime values whose concrete types may differ. These mechanisms are related, but they are not interchangeable.

#### 1. Value-Indexed Dependent Types

A dependent type may contain a value as part of its type identity. The value index may be a compile-time constant, a symbolic type-level value, or a runtime function parameter whose relationship to the result is tracked by the compiler.

```folang
readVector(n co.lang.int)->(Vector(n)) = {
    ...
}
```

The compiler does not need to know the concrete value of `n` while compiling `readVector`. It records the relationship:

```text
input index = n
result type = Vector(n)
```

Likewise:

```folang
dotProduct(
    left  Vector(n),
    right Vector(n)
)->(co.lang.int) = {
    ...
}
```

requires both vectors to have the same value index. Dependent typing means that a type contains or is constrained by a value; it does not mean that an arbitrary runtime branch can silently change the static type of an already compiled variable.

---

#### 2. Compile-Time Type Computation

A function may compute and return a type when it is guaranteed to execute during compilation.

```folang
@co.dap.comptime
@co.dap.eager
chooseType(value co.lang.int)->(co.lang.type) = {
    (value < 100)
        .return(co.lang.string)
        .otherwise.return(co.lang.bool);
}
```

The arguments must be compile-time evaluable when the result is used in a static type position:

```folang
Selected co.lang.type = chooseType(10);
value Selected = "Hello";
```

Invalid:

```folang
input := co.in.readInt();
Selected co.lang.type = chooseType(input);
// compiler error: `input` is not compile-time evaluable
```

Conceptually:

```text
compile-time value
    -> compiler executes the type function
    -> one concrete static type is available before ordinary type checking completes
```
---
#### 3. Built in compile type computation

A function may compute and return a type when it is guaranteed to execute during compilation.

> `decltype` built in method

The arguments must be compile-time evaluable when the result is used in a static type position:

```folang
    someIntVar co.lang.int ;
    someVar co.hokrlt.type.decltype(someIntVar) = 200;
```
---

#### 4. Runtime Type Descriptors

An ordinary function returning `co.lang.type` produces a runtime type descriptor when it is not executed at compile time.

```folang
selectType(value co.lang.int)->(co.lang.type) = {
    (value < 100)
        .return(co.lang.string)
        .otherwise.return(co.lang.bool);
}

selectedType co.lang.type = selectType(input);
```

Here, `selectedType` is a runtime object that describes a type. It may be used for reflection, runtime validation, dynamic loading, metadata inspection, or dynamic object creation where those capabilities are permitted.

A runtime type descriptor cannot ordinarily be used as the static type of a declaration:

```folang
value selectType(input);
// compiler error: runtime type descriptor used in a static type position
```

Conceptually:

```text
runtime value
    -> runtime type-selection function
    -> co.lang.type descriptor object
```

A value that represents a type is not automatically a compile-time-resolved static type.
---

#### 5. `@co.dap.typefromvalue`

`@co.dap.typefromvalue` derives a type from the type of a returned compile-time value. In the initial FoLang specification, it is permitted only for compile-time evaluation.

```folang
@co.dap.comptime
@co.dap.typefromvalue
inferType(value co.lang.int)->(co.lang.type) = {
    (value < 100)
        .return("Hello")
        .otherwise.return(co.const.true);
}
```

The compiler derives:

```text
"Hello"       -> co.lang.string
co.const.true -> co.lang.bool
```

Every argument must be compile-time evaluable when the result is used in a type position. Runtime value-based type selection must instead use an explicit runtime representation.

---

#### 6. Runtime Values with Different Concrete Types

When a runtime branch may produce unrelated concrete value types, the function must return one stable outer type. FoLang may use a tagged value, an ADT, `co.lang.dynamic` where permitted, or another explicitly packaged representation.

Using `co.lang.tag`:

```folang
selectValue(value co.lang.int)->(co.lang.tag) = {
    (value < 100)
        .return(co.lang.tag(co.lang.string, "Hello"))
        .otherwise.return(
            co.lang.tag(co.lang.bool, co.const.true)
        );
}
```

The outer static type is always `co.lang.tag`:

```text
co.lang.tag
├── runtime type descriptor
└── value compatible with that descriptor
```

An ADT provides a more strongly typed alternative:

```folang
SelectedValue co.lang.data =
      StringValue(co.lang.string)
    | BoolValue(co.lang.bool);

selectValue(value co.lang.int)->(SelectedValue) = {
    (value < 100)
        .return(StringValue("Hello"))
        .otherwise.return(BoolValue(co.const.true));
}
```
---

#### Summary

```text
Vector(n)
    -> value-indexed dependent type

@co.dap.comptime function returning co.lang.type
    -> compile-time type computation

ordinary runtime function returning co.lang.type
    -> runtime type descriptor

runtime branch returning unrelated value types
    -> tag, ADT, dynamic value, or another explicit package
```

A runtime type descriptor is a value that represents a type. It must not be confused with a statically resolved dependent type.

---

### Parameterized Type Declarations and Type-Level Functions

Two declaration families produce types from parameters. The spelling depends on whether the declaration directly defines a type family or computes a type through a function body.

```folang
// all parameters are types -> parameterized co.lang.type declaration
Option(T) co.lang.type = Some(T) | None();
someAlias(F) co.lang.type = Functor(F);

// a value parameter is present -> type-level function syntax
Vector(n co.lang.int)->(co.lang.dependentType) = co.lang.int->([n]);
Stack(n co.lang.int, T co.lang.type)->(co.lang.dependentType) = T->([n]);
```

A parameterized `co.lang.type` declaration is a type constructor. Its type parameters appear directly in the declaration head and it does not use `@co.dap.generic`.

A function that accepts values or type values and returns `co.lang.dependentType` is a type-level function. `Stack` demonstrates why the function form exists: it can mix value parameters and type-valued parameters and compute the resulting type.

`co.lang.dependentType` is both a type-producing return kind and a direct type-declaration kind. A type-level function uses it when a value parameter determines the produced type. A direct declaration may use it when no parameter list is required:

```folang
LengthBound co.lang.dependentType = co.lang.int;
```

The kind is also usable in a declarator. If a function returns `co.lang.dependentType`, a binding receiving that result may therefore be declared `co.lang.dependentType`.

A type-level function has exactly one unnamed type-producing result. That result may be a union using `|`, but comma-separated multiple results are invalid:

```folang
Choice(n co.lang.int)->(co.lang.dependentType | co.lang.type) = co.lang.int;
Bad(n co.lang.int)->(co.lang.dependentType, co.lang.type) = co.lang.int; // invalid
```

#### Parameterized aliases are transparent

An alias declared with `co.lang.type` names the same type, not a new one.

```folang
someAlias(F) co.lang.type = Functor(F);

someAlias(List);    // the same type as Functor(List)
someAlias(Option);  // the same type as Functor(Option)
```

Because the alias adds no identity, it creates no separate instance slot: an
instance cannot be declared for `someAlias` as distinct from one for `Functor`.
Declaring one would be a duplicate.

Named parameters also allow reordering and partial application, which a
positional placeholder could not express.

```folang
Pair(F, G) co.lang.type = Transformer(F, G);
Flip(F, G) co.lang.type = Transformer(G, F);
Fixed(F)   co.lang.type = Transformer(F, Set);
```

Use `co.lang.newtype` instead when a **distinct** identity is wanted, such as
the wrapper described in [Where an Instance Is
Declared](#where-an-instance-is-declared).

### Dependent Type Index Rules

An **index** is an argument to a dependent type, such as the `n` in
`Vector(n)`, or a dimension in an array derivation, such as the `n` in
`co.lang.int->([n])`. Both positions obey the same rules.

#### An index is a literal or a name

An index is an integer literal or a name. Arithmetic, function calls, indexing
and every other operator are rejected.

```folang
@co.dap.const SIZE co.lang.int = 1024;

v Vector(3);                    // ✅ literal
v Vector(SIZE);                 // ✅ @co.dap.const name
buf co.lang.int->([SIZE]);      // ✅ same rule for array sizes

v Vector(n + 1);                // ❌ arithmetic is not permitted in an index
v Vector(computeSize());        // ❌ a call is not permitted in an index
buf co.lang.int->([n * 2]);     // ❌ same rule for array sizes
```

This restriction applies only to the **size** of an array, never to element
access. Indexing an array is an ordinary expression and arithmetic is fine.

```folang
buf co.lang.int->([SIZE]);      // size — restricted
buf[i + 1] = 42;                // access — unrestricted
buf[compute(x)] = 7;            // access — unrestricted
```

#### What a named index may resolve to

A name used as an index resolves in exactly one of two ways.

**A parameter bound by the enclosing signature.** A type constructor or a
function signature introduces the name, and every use of it inside that
signature and its body refers to the bound parameter. The name is not a
constant; it stands for whatever value the caller supplies.

```folang
// n is introduced here, and bound for the whole declaration
Vector(n co.lang.int)->(co.lang.dependentType) = co.lang.int->([n]);

// n is introduced by this signature, and both parameters must share it
dotProduct(a Vector(n), b Vector(n))->(co.lang.int) = {
    // ...
}

// n is introduced as a value parameter and reused in the return type
readVector(n co.lang.int)->(Vector(n)) = {
    // ...
}
```

**A `@co.dap.const` compile-time constant.** Outside a signature that binds it,
a name has nothing to bind to, so it must be a constant the compiler can
substitute.

```folang
@co.dap.const SIZE co.lang.int = 1024;
buf co.lang.int->([SIZE]);      // ✅ SIZE substitutes to 1024
v Vector(SIZE);                 // ✅ same rule for dependent types
```

Nothing else qualifies. `@co.dap.final` marks an immutable binding, and an
immutable value need not be known while compiling, so it cannot be substituted.

```folang
@co.dap.final n co.lang.int = readInput();
bad Vector(n);                  // ❌ immutable, but not known at compile time

m co.lang.int = 10;
alsoBad Vector(m);              // ❌ an ordinary variable is not an index
```

So in a plain variable declaration, where no signature is binding anything, the
only legal names are `@co.dap.const` constants.

#### An index is non-negative

Zero is permitted; a negative index is not.

```folang
empty co.lang.int->([0]);       // ✅ zero-length array

buf co.lang.int->([-1]);        // ❌ rejected while parsing
v Vector(-1);                   // ❌ rejected while parsing

@co.dap.const OFFSET co.lang.int = -1;
buf co.lang.int->([OFFSET]);    // ❌ rejected after substitution
```

A negative literal cannot be written at all, because no prefix operator is
reachable in an index position. A negative constant is rejected when the
compiler substitutes it. Both are compile-time errors.

#### When two dependent types are equal

Two dependent types are equal when their constructors are the same and their
indices are pairwise equal. An index comparison has exactly three cases.

| Index form | Compared by |
|---|---|
| integer literal | value |
| `@co.dap.const` name | substituted literal value |
| parameter | name identity |

```folang
Vector(3)    vs Vector(3)       // equal
Vector(n)    vs Vector(n)       // equal
Vector(n)    vs Vector(m)       // NOT equal — rejected
Vector(SIZE) vs Vector(1024)    // equal when @co.dap.const SIZE = 1024
```

Rejecting `Vector(n)` against `Vector(m)` is the point of the feature. It is
what lets the compiler catch a size mismatch without the developer writing a
single check.

#### What FoLang deliberately does not do

FoLang does not decide index equality up to arithmetic. `Vector(n+1)` and
`Vector(1+n)` are not merely unequal — they cannot be written.

Accepting them would require symbolic reasoning, and there is no partial
version of it. Once `n+1 == 1+n` is accepted, the next reasonable request is
`2*n == n+n`, and the type checker becomes a theorem prover by accretion. That
is the complexity FoLang is built to avoid.

The cost is narrow. Length-arithmetic signatures such as
`concat(Vector(n), Vector(m)) -> Vector(n+m)` are out of scope; return a
dynamically sized type and check at run time instead. Everything that needs
only same-parameter identity still works, and that covers the common cases.

```folang
multiply(a Matrix(r, n), b Matrix(n, c)) -> (Matrix(r, c))
dotProduct(a Vector(n), b Vector(n))     -> (co.lang.int)
zip(a Vector(n), b Vector(n))            -> (Vector(n))
```

Matrix multiplication, the usual demonstration of dependent types, needs only
that the shared `n` matches.

#### Dependent types are checked, never inferred

Every dependent type appears in a written signature. FoLang never infers one.

This is what keeps checking decidable without a constraint solver, and it is
why FoLang does not adopt Hindley-Milner style whole-program inference.
Inferring a dependent type would mean inferring the index **value**, not merely
the type, which is the step that makes checking undecidable in general.

---

## Indexer

Indexer functions for a struct are associated functions and must be declared inside `<StructName>.comp.unit.fol`.

```folang
// MyList.fol
_ co.lang.struct ={
    eles co.lang.int->([...]);
}

// MyList.comp.unit.fol
_ co.lang.unit = {

    @co.dap.indexer(symbol="[]")
    (g MyList) get(index co.lang.int)->(co.lang.int) ={
        this.return g.eles[index];
    }

    @co.dap.indexer(symbol="[]=")
    (g MyList) set(index co.lang.int, value co.lang.int)->() ={
        g.eles[index] = value;
    }
}

lst MyList;
co.out.println(lst[0]);
lst[1] = 22;
```

---
## Generics

```folang
@co.dap.generic(
    at=runtime,
    types=[
        {name= T, variance=invariant, bound=Number, kind=param},
        {name= R ,variance=invariant, bound=Number, kind=result}
    ],
    impredicative=false,
    resolution=compiletime
)
add(a T, b T)->(R) = { this.return a + b; }
```

**Generic annotation fields:**

| Attribute | Values |
|---|---|
| types | map with type and other details |
|requires||
|resolution| `runtime`, `compiletime`|
|refied| `true` or `false`|
|where| `usesite` or `callsite`|
|specializable| `true` or `false` |
|impredicative| `true` or `false`|


**types attributes**
|Attribute | Values|
|---|---|
|name||
|constraints||
|upper-bound||
|lower-bound||
|default||
|variance| `covariant`, `invariant`, `contravariant`|
|nullable||
|inference| `param` , `result`, `arg`,`var` |
|capabilities||
|where-rules||
|typekind| `type`,`class`,`function`,`struct`,`typeconstructor`| 
|inclusive||

> The above ones will be `types` attribute's sub attributes in a map format

> E.g.,  @co.dap.generic(types={ T: {variance:..., bound:..., ....} })

> These are not independent attributes these are depend on type 

### Generic Functions — Parameters and Return Values

#### Rank-1: Outer function is generic; parameter uses the same type variable

`T` is fixed at the call site before the function parameter is used. The passed function is already monomorphic inside the body.

**Syntax 1 — Inline signature**
```folang
@co.dap.generic(types=[{name=T}])
someFunction(f (T, T)->(T), a T)->(T) = {}
```

**Syntax 2 — Named type alias**
```folang
@co.dap.generic(types=[{ name=T}])
someFArg co.lang.type = (T, T)->(T);

@co.dap.generic(types=[{ name=T}])
someFunction(f someFArg, a T, b T)->(T) = {}
```

---

#### Rank-2: The function parameter is itself polymorphic (higher-rank)

The passed function stays generic **inside the callee**. The callee decides what `T` is. Uses existing `forall`.

**Syntax 1 — Inline signature**
```folang
someFunction(f forall(T).(T, T)->(T))->(co.lang.int) = {
    this.return f(1, 2);
}
```

**Syntax 2 — Named type alias**
```folang
someFArg co.lang.type = forall(T).(T, T)->(T);

someFunction(f someFArg)->(co.lang.int) = {}
```

```folang
// Correct — Syntax 2 with co.lang.type
someFArg co.lang.type = forall(T).(T, T)->(T);

someFunction(f someFArg)->(co.lang.int) = {}
```

---

#### Returning Generic Functions

**Rank-1 return**
```folang
@co.dap.generic(types=[{name=T}])
makeAdder(a T)->((T)->(T)) = {
    this.return (b T)->(T) = { this.return a + b; };
}
```

**Rank-2 return — returning a polymorphic function**
```folang
makeIdentity()->( forall(T).(T)->(T) ) = {
    this.return forall(T).(x T)->(T) = { this.return x; };
}
```

---

#### Rank-3: A Parameter is Itself a Rank-2 Function

Rank-3 works naturally in FoLang via `forall` nesting. No new constructs needed.

**Syntax 1 — Inline**
```folang
// f takes a Rank-2 function as its argument — that is Rank-3
applyRank2(
    f (forall(T).(T, T)->(T)) -> (co.lang.int)
) -> (co.lang.int) = {
    this.return f(1, 1);
}
```

**Syntax 2 — Named type aliases (cleaner)**
```folang
rank2FnType  co.lang.type = forall(T).(T, T)->(T);
rank3ArgType co.lang.type = (rank2FnType) -> (co.lang.int);

applyRank2(f rank3ArgType) -> (co.lang.int) = {
    this.return f(1, 1);
}
```

**Rank-3 return**
```folang
makeRank2Consumer() -> ((forall(T).(T)->(T)) -> (co.lang.int)) = {
    this.return (f forall(T).(T)->(T)) -> (co.lang.int) = {
        this.return f(42);
    };
}
```

---

#### Impredicativity — Instantiating `T` with a `forall` Type

Impredicativity is when a type variable `T` in a generic is itself instantiated with a `forall` type. Example of what this means:

```folang
@co.dap.generic(types=[{name=T}])
box(x T) -> (Box(T)) = {}

// Impredicative call — T being set to forall(U).(U)->(U)
result := box(forall(U).(U)->(U));   // ❌ not legal without explicit opt-in
```

Most type systems reject this by default. FoLang takes an opt-in approach.

**v1 Workaround — Option C: Wrapping with `co.lang.type`**

Not true impredicativity but solves 90% of practical cases:

```folang
polyId co.lang.type = forall(U).(U)->(U);

// box takes co.lang.type — no impredicative unification needed
box(x co.lang.type) -> (Box(co.lang.type)) = {}

result := box(polyId);   // ✅ works — x is co.lang.type, not a forall type
```

**v2 — Option A: `impredicative:true` in `@co.dap.generic`**

Explicit opt-in via the existing annotation. The compiler only permits `forall` instantiation where declared:

```folang
@co.dap.generic(
    types=[{name=T,variance=invariant}],
    impredicative=true
)
box(x T) -> (Box(T)) = {}

polyId co.lang.type = forall(U).(U)->(U);
result := box(polyId);   // ✅ legal — impredicative:true explicitly opts in
```

---

#### Generic Function Rank Support Matrix

| Scenario | Allow? | Notes |
|---|---|---|
| Rank-1 generic param (Syntax 1, 2, 3) | ✅ Yes | Natural extension, no new concepts |
| Rank-1 generic return (Syntax 1, 2, 3) | ✅ Yes | Same as above |
| Rank-2 param via `forall` (Syntax 1, 2) | ✅ Yes | `co.lang.type` naturally holds polymorphic types; no new kind needed |
| Rank-2 param via Syntax 3 `co.lang.function` | ❌ Compiler error | Function objects are concrete values; use `co.lang.type = forall(T).(T)->(T)` instead |
| Rank-2 return via `forall` (Syntax 1, 2) | ✅ Yes | Same reasoning as param |
| Rank-3 via `forall` nesting (Syntax 1, 2) | ✅ Yes | No new constructs — `forall` nesting works naturally |
| Rank-3 return | ✅ Yes | Same reasoning as Rank-3 param |
| Rank-3 via Syntax 3 `co.lang.function` | ❌ Compiler error | Same rule as Rank-2; function objects are concrete |
| Impredicative — v1 workaround (Option C) | ✅ Yes | Wrap `forall` type in `co.lang.type`; solves 90% of real cases |
| Impredicative — true opt-in (Option A) | 🔜 v2 | `impredicative:true` in `@co.dap.generic`; explicit opt-in |

`@co.dap.generic` remains the declaration annotation for constraints, variance,
reification, and the other generic metadata described below. A direct generic
clause supplies the named parameters and their arity; the two forms may be used
together when that metadata is required.

```folang
// LinkedList.fol
@co.dap.generic(types=[{name=T}])
_ co.lang.struct={
    value T;
    next  LinkedList;
    prev  LinkedList;
}

k := LinkedList.new(co.lang.int); // when we call new it returns an object of type co.lang.uninit
actualList := k.init(); // this is what create a fully formed object of type class



// Employee.fol
@co.dap.generic(types=[{name=T},{name= R}])
_ co.lang.class ={
    id   T;
    name R;

    @co.dap.override
    @co.dap.constructor(access=private)
    @@init() = {}

    @co.dap.override
    @co.dap.constructor(access=public)
    @@init(id T, name R) = {
        this.parent.init();
        this.id   = id;
        this.name = name;
    }

    getEmployee(id T)->(Employee)={}
}

a := Employee.new(co.lang.int, co.lang.string);
b := a.init(1, "Rao");

Normally we need not use @@new and @@init it is special case only applicable for Generics when doing something really different,

Normal conditions to create/instantiate object of class we just call init which internally call new 
In specific cases as above we need to do two calls or use call chain like below


c := Employee.new(co.lang.int,co.lang.string).init(1,"Rao");

This new and init methods will be available without you overriding/overloading new and init 


// Employee.fol
@co.dap.generic(types=[{name=T},{name= R}])
_ co.lang.class ={
    id T;
    name R;

}

c := Employee.new(co.lang.int,co.lang.string).init(1,"Rao");

This works folang automaticall provides all type parameters new and all field init implementations.
    
```

### Generics Inheritances and Types

```
This is in conceptual stage not supported.

A) Abstract vs concrete type members
B) Path-dependent types
    1. Type-level projection
    2. Path-dependent In folang how it would be
```

### forall

#### What `forall` Is — and Is Not

`forall` is **not** a general-purpose generic declaration keyword. It is a **type-level expression only**, restricted to contexts where a polymorphic type must appear as an anonymous value inline — specifically Rank-2 and Rank-3 parameter and return positions, and `co.lang.type` aliases.

Named generic structs, classes, functions, and methods use `@co.dap.generic` as their sole generic-parameter declaration mechanism. `forall` is not a declaration mechanism;
`forall` at declaration level is a **compiler error**.

---

#### Where `forall` Is Allowed — Type Expression Form Only

`forall(T).` followed by an anonymous type body. The `.` is the syntactic signal that what follows is a type body, not a declaration name.

Pattern:
```
forall(T).  <anonymous type body>
```

```folang
// co.lang.type alias — naming a polymorphic type for reuse
someFArg co.lang.type = forall(T).(T, T)->(T);

// Rank-2 inline parameter — callee decides what T is
someFunction(f forall(T).(T)->(T)) -> (co.lang.int) = {}

// Rank-2 return type — returning a polymorphic function
makeIdentity() -> (forall(T).(T)->(T)) = {}

// Rank-3 inline parameter — f takes a Rank-2 function
applyRank2(f (forall(T).(T, T)->(T)) -> (co.lang.int)) -> (co.lang.int) = {}
```

---

#### Where `forall` Is Banned — Use `@co.dap.generic` Instead

```folang
// ❌ compiler error — forall at declaration level
forall(T) identity(x T)->(T) = {}

// ✅ correct
@co.dap.generic(types=[{name=T,variance=invariant}])
identity(x T)->(T) = {}
```

```folang
// ❌ compiler error
// LinkedList.fol
forall(T)
_(T) co.lang.struct = { value T; next LinkedList; }

// ✅ correct
// LinkedList.fol
@co.dap.generic(types=[{name=T}])
_ co.lang.struct = { value T; next LinkedList; }
```

```folang
// ❌ compiler error — Rank-1 generics belong to @co.dap.generic
forall(T) someFunction(f (T,T)->(T), a T)->(T) = {}

// ✅ correct
@co.dap.generic(types=[{name=T,variance=invariant}])
someFunction(f (T,T)->(T), a T)->(T) = {}
```

---

#### Quick Reference

| Form | Status | Context |
|---|---|---|
| `forall(T) name ...` | ❌ Compiler error | Declaration level — use `@co.dap.generic` instead |
| `forall(T).(T)->(T)` | ✅ Allowed | Type level only — Rank-2/3 param, return, `co.lang.type` alias |

**The rule in one sentence:** `forall(T).` forms an anonymous polymorphic type expression; it is never a declaration keyword or a file-backed declaration-name mechanism.


> Generic declarations are supported only for structs, classes, functions, and methods. Their type parameters are introduced exclusively by `@co.dap.generic`.

The following declaration-head generic forms are invalid:

```folang
// Cache.fol
_(T) co.lang.module = {}             // compiler error
// operations.unit.fol
_(F(_)) co.lang.unit = {}            // compiler error
Callback(T) co.lang.delegate = (T)->(T); // compiler error
```

A parameterized `co.lang.type` declaration is the separate type-constructor form and does not use `@co.dap.generic`:

```folang
// option.unit.fol
_ co.lang.unit = {
    Option(T) co.lang.type =
        Some(T) | None();
}
```

Generic structs and classes remain file-backed primary declarations. Their names come from filenames:

```folang
// LinkedList.fol
@co.dap.generic(types=[{name=T}])
_ co.lang.struct = {
    value T;
    next  LinkedList;
    prev  LinkedList;
}

myIntList LinkedList = LinkedList.withTypes(co.lang.int);
```

```folang
// Employee.fol
@co.dap.generic(types=[{name=T}, {name=R}])
_ co.lang.class = {
    id   T;
    name R;
}

emp Employee = Employee.new(co.lang.int, co.lang.string).init;
```

Generic functions use the same annotation but are declared inside a legal function-owning context such as an ordinary unit, class, or companion unit:

```folang
@co.dap.generic(types=[{name=T}, {name=R}])
add(a T, b T)->(R) = {
    ...
}

 add_int_int := add.withTypes(co.lang.int,co.lang.int);

    or

 add_int_int co.lang.function =  add.withTypes(co.lang.int,co.lang.int);

 k := add_int_int(12,10);
```

---

## Specialization

`@co.dap.specialize` to specialize generics for specific types upfront

```folang

@co.dap.generic(
    types=[
        {name=T}
    ],
    requires=[
        co.lang.Add(left=T, right=T, result=T)
    ]
)
add(a T, b T)->(T) = {
    this.return a + b;
}

```

for the above generic want to specialize for `co.lang.int`

```folang

@co.dap.specialize(
    target=add,
    types=[
        {name=T, type=co.lang.int}
    ]
)
addInt(a co.lang.int, b co.lang.int)->(co.lang.int) = {
    this.return co.intrinsic.intAdd(a, b);
}

```


`folang` provides partial specialization below is the example for partial specializationn

```folang

@co.dap.generic(
    types=[
        {name=T},
        {name=R}
    ]
)
transform(value T)->(R) = {
    ...
}
```

```folang

@co.dap.specialize(
    target=transform,
    types=[
        {name=T, type=co.lang.string},
        {name=R}
    ]
)
transformString(value co.lang.string)->(R) = {
    ...
}
```


***fields of specialize**

|Attribute|Values|
|---|---|
| target| the generic fully qualified name includes package name if omitted it is current package |
| types| resoultion types|
| priority||
| strategy|intrinsic|

---

## Generic Declarations and Type Constructors

FoLang distinguishes annotation-based generic declarations from parameterized `co.lang.type` constructors.

### Generic Structs, Classes, Functions, and Methods

`@co.dap.generic` is the sole mechanism for declaring generic structs, classes, functions, and methods. Generic parameters for these declaration kinds must not appear in the declaration head.

```folang
// Box.fol
@co.dap.generic(types=[{name=T}])
_ co.lang.struct = {
    value T;
};
```

```folang
// conversion.unit.fol
_ co.lang.unit = {
    @co.dap.generic(types=[{name=T}, {name=R}])
    convert(value T)->(R) = {
        ...
    }
}
```

These forms are invalid:

```folang
// Box.fol
_(T) co.lang.struct = { ... }        // compiler error
// Container.fol
_(T) co.lang.class = { ... }         // compiler error
```

### `co.lang.type` Constructors

A `co.lang.type` constructor does not use `@co.dap.generic`. Its type parameters appear directly in the type declaration head, and the declaration must be inside an ordinary unit or another explicitly legal type-declaration context.

```folang
// option.unit.fol
_ co.lang.unit = {
    Option(T) co.lang.type =
        Some(T) | None();
}
```

`Option` denotes a unary type constructor:

```text
Option : Type -> Type
```

Applying it produces a type:

```folang
value Option(co.lang.int);
employeeOption Option(Employee);
```

`Some(T)` and `None()` are value constructors declared by the `Option` definition itself. They do not require separate function implementations.

`@co.dap.generic` is invalid on `co.lang.type`, and declaration-head type parameters are invalid on structs, classes, functions, methods, signatures, interfaces, modules, enums, unions, cstructs, units, and other declaration kinds unless a later specification version explicitly adds support.

### No Type-Constructor or Type-Function Annotation

FoLang requires neither `@co.dap.typeconstructor` nor `@co.dap.typefunction`.

```text
Option(T) co.lang.type = ...
    -> recognized syntactically as a type-constructor declaration

ElementType(container co.lang.type)->(co.lang.type) = ...
    -> recognized syntactically as a type-level function
```

The declaration form already determines the category unambiguously.

---

## Templates

### Typed

```folang
// myttypedtemplate.unit.fol

_ co.lang.unit={
    @co.dap.template
    add(a co.lang.int, b co.lang.int)->(co.lang.int) ={
        this.return a + b;
    }
}
```

### Untyped

```folang
// MyTemplate.unit.fol

_ co.lang.unit={

    @co.dap.template
    add(a, b)->(co.lang.untyped) ={
        this.return a + b;
    }
}
```
---

## Annotations and Decorators

```folang
// Annotation — static object, can carry data


// myAnnotation.fol

_ co.lang.object->(for=annotation) = {
    value   co.lang.string;
    enabled co.lang.bool;
}

// Decorator — function, transforms target, returns
// Decorator.unit.fol

_ co.lang.unit={

    @co.dap.decorator
    myDecorator(target co.lang.function)->(co.lang.function) = { }
}

Note Directives and Pragmas are not allowed to create as they are language internals

```
---

## Macros

```folang
// a. Basic macro
@co.dap.macro
say()->()={ this.return co.macro.quote({ println("Line 1") println("Line 2") }); }

// b. Escape assign
@co.dap.macro
yes_esc_assign()->(co.lang.untyped)={
    this.return co.macro.quote({
        co.macro.esc(y) = 42;
        co.out.println("Inside macro: y = ", y);
    });
}

// c. Debug macro with gensym
@co.dap.macro
debug(expr)->(co.lang.untyped)={
    tmp := co.macro.gensym(co.lang.var, "tmp");
    this.return co.macro.quote({
        tmp = co.macro.esc(expr);
        co.out.println("Result: ", tmp);
        tmp;
    });
}

// d. if/else condition macro
@co.dap.macro(
    group = {items:["if","else"], chain:true},
    sugarform={forms:["if expr block"]},
    bind={vars:["x"]},
    isolate={vars:["temp", "index"]},
    gensym={prefix:"tmp_"},
    hygienic=true,
    argtransform={param:"body", wrap:"lambda", whentype:"block"},
    desugar={exprs:["if($cond) { $block }" => "if($cond,$block)"]},
    mode="inject"
)
if(condition expr, body block)->()={}

blockormacro co.lang.kind = block | macro

@co.dap.macro(
    group={items:["if","else"], chain:true},
    sugarform={forms:["else block","else if"]},
    chainswith={macro:"if", position:"immediate", required:true},
    argtransform={param:"body", wrap:"lambda", whentype:"block"},
    standalone=false,
    desugar={exprs:[
        "else if($cond) { $block }" => "else(if($cond, $block))",
        "else { $elseblock }" => "else($elseblock)"
    ]},
)
else(body blockormacro)->()={}
```

Other macro utilities:
1. `@co.dap.compose(using=["base_if", "blockify"])`
2. `@co.dap.guard(expr="is_bool_expr(expr)")`
3. Quasiquote macros use `co.macro.quote` and `co.macro.unquote`

---

## Execution Models and Control Abstractions (library type=advanced)

Foλang executes code sequentially by default. It also provides a uniform execution model for concurrency, parallelism, asynchronous execution, coroutines, continuations, scheduling, and structured task execution.

Developers express the intended execution semantics by applying annotations such as `@co.dap.thread`, `@co.dap.task`, or `@co.dap.process` to a method. When the method is submitted through facilities such as `co.cpca.submitToPool`, `co.cpca.submitThread`, or `co.cpca.submitToEventLoop`, the Foλang runtime selects and manages the appropriate execution mechanism.

Depending on the annotation, submission operation, runtime environment, and execution policy, Foλang may use a thread pool, virtual or green threads, an event loop, a dedicated operating-system thread, or a separate process. Communication operations such as sending and receiving values are also handled through the `co.cpca` package. Developers therefore describe the required execution behavior without directly managing the underlying threads, processes, pools, or event loops.

The `@co.dap.continuation` annotation enables continuation support for a function. An annotated function can use constructs provided by the `co.cpca` package to suspend execution, yield control or a value, preserve its execution state, and later resume from the suspension point. 

---

## Native Code (Library type system/ffi)

The `@co.dap.native` annotation enables access to the `co.native` package. Through this package, developers can write assembly and machine-level code using facilities such as `co.native.asm` and `co.native.inline`, providing low-level capabilities similar to those available in C++.

### Native Functions

```folang
@co.dap.native
nativeMethod(a co.lang.int, b co.lang.int)->(co.lang.int) ={
    // native implementation
}
```

---
## Dynamic Runtime (library type=dynamicvmrt)

The `@co.ddap.dynamicruntime` annotation enables full access to the `co.meta` package. Through this package, developers can use dynamic class and type loading, monkey patching, runtime reflection, instrumentation, eval-based code execution, and other advanced metaprogramming capabilities.

---






## Variable Kinds Support

| Kind | Where |
|---|---|
|  Normal | All |
|  Pointers | Library of type `system` and `ffi`|
|  Arrays   | All |
|  References Heap, Lvalue, Rvalue| Library of type `system` and `ffi` |
|  Addresses | Library of type `system` and `ffi` |
|  Thunks |  Library `application` or `system` or `ffi` or `advanced` or `dynamicvmrt` |
|  Ranges | All |
|  Slices | All |

---

## Builtin Data Types

| Type | Kind |
|---|---|
|`co.lang.string`||
|`co.lang.int`||
|`co.lang.bit`||
|`co.lang.double`||
|`co.lang.float`||
|`co.lang.long`||
|`co.lang.byte`||
|`co.lang.char`||
|`co.lang.any`||
|`co.lang.dynamic`||
|`co.lang.auto`||
|`co.lang.infer`||
|`co.lang.bool`||
|`co.lang.void`||
|`co.lang.data`||
|`co.lang.value`||
|`co.lang.typed`||
|`co.lang.untyped`||
|`co.mem.region`||
|`co.lang.nothing`||
|`co.lang.word`||
|`co.lang.MatchBindings`||
|`co.lang.tag`||
|`co.lang.typevalue`||
|`co.lang.uninit`||
|`co.lang.literal`||
|`co.lang.pointer`||
|`co.lang.address`||
|`co.lang.reference`||
|`co.lang.thunk`||
|`co.lang.array`||
|`co.lang.slice`||
|`co.lang.range`||
|`co.lang.just`||
|`co.lang.operator`|operator-source-only declaration kind; invalid in ordinary FoLang source|
|`co.lang.typeclass`||
|`co.lang.typeconstructor`|reserved for a future language version; currently unsupported in source declarations|
|`co.lang.typefunction`|reserved for a future language version; currently unsupported in source declarations|

A name appearing in this registry is not necessarily an enabled source-language feature. A built-in kind is usable only when this specification defines its declaration syntax and semantics. An undocumented or explicitly reserved kind remains unavailable and must produce an unsupported-feature diagnostic when used.

---

## Built-in Directives
|Kind | ||
|---|---|---|
|`PRAGMA`|"@co.pdap.compiler", "@co.pdap.scale"||
|`DIRECTIVE`|"@co.ddap.movetotop", "@co.ddap.import", "@co.ddap.dynamicruntime", "@co.ddap.use",  "@co.ddap.alias"||
|`ANNOTATION`| "@co.dap.template", "@co.dap.macro","@co.dap.operator", "@co.dap.annotation", "@co.dap.library", "@co.dap.module", "@co.dap.pragma", "@co.dap.directive","@co.dap.native", "@co.dap.class", "@co.dap.static","@co.dap.instance", "@co.dap.object", "@co.dap.inline","@co.dap.ctfe", "@co.dap.friend", "@co.dap.sealed", "@co.dap.extension","@co.dap.override", "@co.dap.virtual", "@co.dap.abstract", "@co.dap.delegate", "@co.dap.dynamicscope","@co.dap.lexicalscope","@co.dap.staticscope","@co.dap.mixedscope", "@co.dap.typeclass","@co.dap.matcher", "@co.dap.constructor", "@co.dap.oops", "@co.dap.hokrt","@co.dap.hokrlt", "@co.dap.indexer", "@co.dap.generic", "@co.dap.comptime", "@co.dap.typefromvalue", "@co.dap.local", "@co.dap.private","@co.dap.public","@co.dap.package","@co.dap.protected","@co.dap.internal","@co.dap.export","@co.dap.eager", "@co.dap.lazy", "@co.dap.packed", "@co.dap.declare","@co.dap.simd", "@co.dap.reflection", "@co.dap.mop","@co.dap.nested","@co.dap.inner","@co.dap.final","@co.dap.const","@co.dap.decorator","@co.dap.specialize"|//mop => meta object programming|
|`DECORATOR`|"@co.dap.before", "@co.dap.after","@co.dap.around", "@co.fx.onErrExcept", "@co.fx.InvokeAlways","@co.fx.HandleEffect", "@co.dap.callback", "@co.dap.defer","@co.dap.continuation", "@co.dap.event", "@co.dap.scale", "@co.dap.distributed","@co.dap.concurrent", "@co.dap.parallel", "@co.dap.subroutine",	"@co.dap.generator", "@co.dap.goroutine", "@co.dap.coroutine","@co.dap.async", "@co.dap.promise", "@co.dap.future",	"@co.dap.thread", "@co.dap.task", "@co.dap.fiber", "@co.dap.process","@co.dap.spawn", "@co.dap.exec", "@co.dap.fork", "@co.dap.csp","@co.dap.actor", "@co.dap.synthetic", "@co.dap.bridge","@co.dap.greenlet", "@co.dap.channel", "@co.dap.callable", "@co.dap.iterator"||

---

## Builtin Kinds
|Kind | Purpose
|---|---|
|`co.lang.type`||
|`co.lang.struct`||
|`co.lang.cstruct`||
|`co.lang.realm`|  similar to loader where symbols reside |
|`co.lang.loader`| class, type, functions loader where objects reside loader takes realm as parameter|
|`co.lang.class`||
|`co.lang.interface`||
|`co.lang.union`||
|`co.lang.role`||
|`co.lang.record`||
|`co.lang.property`||
|`co.lang.indexer`||
|`co.lang.object`||
|`co.lang.instance`||
|`co.lang.matcher`||
|`co.lang.trait`||
|`co.lang.mixin`||
|`co.lang.extension`||
|`co.lang.delegate`||
|`co.lang.typeclass`||
|`co.lang.concept`||
|`co.lang.typealias`||
|`co.lang.module`||
|`co.lang.unit`|stateless file-level container; ordinary units merge into the package namespace and `*.comp.unit.fol` attaches to a struct|
|`co.lang.macro`||
|`co.lang.template`||
|`co.lang.lambda`||
|`co.lang.block`||
|`co.lang.behavior`||
|`co.lang.package`||
|`co.lang.signature`||
|`co.lang.function`||
|`co.lang.method`||
|`co.lang.namespace`||
|`co.lang.stex`||
|`co.lang.kind`||
|`co.lang.level`||
|`co.lang.order`||
|`co.lang.rank`||
|`co.lang.newtype`||
|`co.lang.opaquetype`||
|`co.lang.subtype`||
|`co.lang.supertype`||
|`co.lang.dependentType`||
|`co.lang.refinementType`||
|`co.lang.associatedtype`||
|`co.lang.hokrlt`| higer rank order kind level types |
|`co.lang.data`||
|`co.lang.enum`||
|`co.lang.typetype`||
|`co.lang.typekind`||
|`co.lang.alias`||
|`co.lang.value`||
|`co.lang.just`||
|`co.lang.nothing`||
|`co.lang.library`||
|`co.lang.symbol` ||
|`co.lang.reservedkeyword`||


## Builtin Operators

###  Arithmetic operators
`+`, `-`, `*`, `/`, `%`, `**`

### Logical operators
`&&`, `||`, `!`, `&`, `|`

### Comparison operators
`==`, `!=`, `<`, `>`, `<=`, `>=`

### Other operator and language-token spellings
`@`, `#`, `!`, `~`, `$`, `^`, `(`, `)`, `_`, `` ` ``, `?`, `{`, `[`, `]`, `}`, `\`, `:`, `;`, `"`, `'`, `=`, `.`, `?=`, `:=`, `::=`, `,`, `..`, `...`, `<..`, `..<`, `<..<`, `==>>`, `=>>`, `=>`, `->`, `<-`, `->>`, `<->`,`@@`

Contiguous symbolic spellings that are absent from this inventory and from the
active custom-operator table are unrecognized symbolic tokens; the lexer does
not split them into shorter operators.

This inventory includes punctuation and reserved token spellings. Presence in
this list does not make a spelling declarable as a `co.lang.operator`, nor
usable with `mode=overload`, `mode=implements`,`mode=extends`, `mode=inherits` or `mode=override`.

### Pre-Declared Operator Glyphs
`λ`,`⒪`,`â`,`Ť`,`∀`,`∃`,`○`,`ö`,`∪`,`Ṡ`,`Ŝ`,`ṁ`,`𝚷`,`⇛`,`𝑓`,`𝒯`,`𝘷`,`𝓕`,`↓`,`∂`,`⊥`,`↧`,`⇓`

Every glyph in this list is language-owned and already has fixed parse
properties. It cannot be redeclared with `co.lang.operator`, but it can receive
implementations through `mode=overload` in a class, struct companion unit, or
built-in extension package contribution. Until a matching implementation is visible, use of the
glyph fails during resolution.

See [Pre-Declared Operator Glyphs](#pre-declared-operator-glyphs).

### Reserved words
`co`, `let`, `this`, `self (contextual keyword)`, `for`, `forall`, `fo (reserved word)`

### Difference between `this` and `self` 
- `this` is for instances and objects
- `self` is for classes
- `static` — no shortcut; can be on variable or classname
- Both `self` and `this` can access member variables

----

## Builtin Methods  
| method | Responsibilit|
|---|---|
| to_str||
| to_int||
| to_float||
| to_double ||
| classof ||
| typeof ||
| new ||
| prototype ||
| proto ||
| make ||
| objectof ||
| instanceof ||
| is ||
| as ||
| iskindof ||
| has ||
| hasown ||
| uses ||
| match ||
| matchall ||
| matchany ||
| matchnone ||
| matchtype ||
| case ||
| with ||
| print ||
| println ||
| printsp ||
| echo ||
| contains ||
| cast ||
| to ||
| dummy ||
| clone ||
| of ||
| for ||
| when ||
| where ||
| then ||
| callback ||
| getAttr ||
| inject ||
| isinstance ||
| cast_to ||
| cast_from ||
| do ||
| map | collection operation — admits a lambda callback |
| flatMap ||
| orElse ||
| filter | collection operation — admits a lambda callback |
| fold ||
| recover ||
| peek ||
| loop ||
| reduce | collection operation — admits a lambda callback |
| forEach | collection operation — admits a lambda callback |
| sortBy | collection operation — admits a lambda callback |
| groupBy | collection operation — admits a lambda callback |
| istrue ||
| isfalse ||
| if ||
| elif ||
| else ||
| return ||
| otherwise ||
| each ||
| containsVal ||
| in ||
| decltype | deduce the type at compile time |
| replace ||
| send ||
| receive ||
| submitToPool||
| submitToEventLoop||
| withTypes| will create appropriate object with types for generics (classes, structs, and functions/methods)|

## Special methods
|Method| Responsibility|
|---|---|
|@@new| Lifecycle method for class|
|@@init| Lifecycle method for class|

## Builtin Packages

### `co` — root (reserved word)

The only package provided by default.

| Sub-package | Responsibility |
|---|---|
| `co.lang` | All data types and kinds |
| `co.sys` | file, concurrent, parallel, goto, invoke, bind, call, apply, settimeout, setinterval, scheduler, cron, event |
| `co.os` | signal, cmd, execute, run, env, getenv, setenv, sleep, exit, cwd, chdir, fork, wait, pipe, dup, dup2, close, readfd, writefd ,random|
| `co.meta` | ast, instrument, transform, augment, reflect, introspect, patch, inject, create, runtime(eval,proto,prototype,etc), realm |
| `co.core` | list, set, map, tree, trie, sort, search, array, pointer, ref, address, ptr, matrix, word |
| `co.native` | load, register, asm, inline, emit, ffi, spawnon[gpu,cpu,npu,apu,fpga,asic,tpu,mki,mcu],arch[x86,x86-64,risc,arm,vliw] |
| `co.in` | read, readln |
| `co.out` | println, print |
| `co.regex` | stex, pattern, match, search |
| `co.crypto` | rsa, aes, hash, md5, rand, uuid, ssl, tls |
| `co.dap` | built-in decorators, annotations|
| `co.ddap` | built-in directives|
| `co.pdap` | built-in  pragmas |
| `co.net` | tcp, udp, http |
| `co.const` | `true`, `false`, `none` |
| `co.encoding` | base64Encode, base64Decode, json, yml, bson |
| `co.utils` | makeImmutable, makeShared, copyOnWrite, toSnapshot — object behaviour policies |
| `co.dynamic` | dynamic capabilities |
| `co.runtime`||
| `co.compiletime`||
| `co.macro`||
| `co.pattern`||
| `co.control` ||
| `co.cpca`| concurrent, async, await, defer, lazy, parallel, process, thread,fiber, task, coroutine,continuation,cps, pool, channel ...|
| `co.hokrtl`||
| `co.hokrt` ||




## FoLang Philosophy — Uniform Object Model

### Core Principle

Everything in **FoLang** is an object. That is the reason for the name **FO** — **Functional Objects**.

FoLang gives the programmer one uniform object model across all types instead of forcing them to switch between separate type families for:

- ordinary values
- immutable values
- shared/concurrent values
- copy-on-write values
- literal values

The programmer writes against a single conceptual model and opts into the required object behaviour only when needed.

---


### Uniform Object Principle

In FoLang, **everything is an object**.

Unless explicitly stated otherwise, the reference and mutation rules in this section apply to **managed FoLang objects**. `co.lang.cstruct` remains an object in the language model, but it is an explicitly value-semantic ABI representation: assignment and parameter passing copy its value rather than a managed object reference.

This includes:

- scalar objects
- collection objects
- user-defined objects
- ADT objects
- function objects

Functions are objects in the same sense as all other values.  
This is true for both **named functions** and **anonymous functions**.

FoLang does not introduce a separate semantic model for functions, scalars, UDTs, or ADTs.  
They all follow the same core object principles:

- assignment of managed objects copies references
- assignment of `co.lang.cstruct` copies its value
- `==` compares values
- mutation is applied through the object
- behaviour policies such as immutability, shared, copy-on-write, and literal conversion apply uniformly where meaningful

So in FoLang, the programmer does not need one mental model for data objects and another for function values.  
The language treats them under one consistent object model.

---

### 1. Default Object Model

Every value in FoLang is an object and is **mutable by default**.

This includes:

```text
co.lang.int, co.lang.float, co.lang.string   → built-in scalar types
user-defined types (structs, classes, ADTs)  → user-defined objects
co.core.list, co.core.map, co.core.array     → built-in collections
```

Example:

```folang
positive_int co.lang.int = 10;
```

`positive_int` denotes an object.  
Its current value is `10`.  
By default, it is mutable.

---

### 2. Assignment, Aliasing, and Mutation

FoLang distinguishes three related ideas:

- **assignment**
- **aliasing**
- **mutation**

#### 2.1 Assignment copies references

When one object variable is assigned to another, FoLang copies the **reference**, not the internal contents.

```folang
b := a
```

After this, `a` and `b` refer to the same object.

#### 2.2 Mutation changes object contents

If two names refer to the same object, mutating through one name is visible through the other.

```folang
// Employee.fol
_ co.lang.class = {
    Name co.lang.string
}

a Employee = { Name = "Kamesh" };
b := a;
c Employee = { Name = "Kamesh" };

a == b;   // true
a == c;   // true
b == c;   // true

b.Name = "Ramesh";   // changes a.Name too
c.Name = "Ramesh";   // does not change a.Name
```

Why:

- `a` and `b` are the **same object**
- `c` is a **different object** with the same earlier value

#### 2.3 `==` means value equality

In FoLang, `==` compares **values**, not object identity.

That rule is uniform across all object kinds, including built-in types and user-defined types.

So:

```folang
a == b
```

means:

- compare contents deeply
- return `true` when the values match
- even if `a` and `b` are not the same object

#### 2.4 Mutation accessors

The accessor used for mutation depends on the object type:

```text
scalar (int, float, bool, string, char, byte)   → .value
struct field                                    → .fieldname
class member                                    → .membername
map entry                                       → .key
array / list element                            → .index
```

Examples:

```folang
count.value = 30
emp.name = "Rao"
marks.math = 95
nums.0 = 42  or nums[0] = 42 //both forms supported
```

#### 2.5 Function-call behaviour

Function parameters introduce a **new local binding**.

So:

- rebinding a parameter does **not** affect the caller
- mutating the object through the parameter **does** affect the caller if both still refer to the same object

```folang
checkPositive(a co.lang.int)->(co.lang.bool) = {
    a = 20;         // local rebinding only — caller unchanged
    a.value = 30;   // object mutation — caller sees 30
}
```

If `positive_int` is passed to `checkPositive(a)`:

```text
a = 20         → local rebinding only
a.value = 30   → mutates the passed object
```

---

### 3. Literal Objects

All literals in FoLang create **Literal Objects**.

```folang
"Kumar"   → Literal Object — string
42        → Literal Object — int
true      → Literal Object — bool
```

A Literal Object is an **anonymous object created from a literal expression**.

Literal Objects participate in value equality just like every other object in FoLang:

```folang
10 == 10   // true
```

In FoLang, `==` compares **values**, not object identity.

So two literal expressions with the same value compare equal, but that does not mean they are the same object.

#### Binding a Literal Object

When a literal is assigned to a named object, the name refers to the object created from that literal expression.

```folang
a co.lang.int = 10;
```

Here, `a` refers to the anonymous integer object created from literal `10`.

So:

```folang
a.value = 30;
```

mutates that same object, and `a` now has value `30`.

The important points are:

- literal expressions create anonymous objects
- literal-created objects are still ordinary objects unless another policy is applied
- `==` compares values, not identity
- once bound to a name, a literal-created object behaves like any other normal mutable object
- immutability applies only when explicitly requested, for example through `makeImmutable(...)`

#### Why Literal Objects Cannot Be Mutated Directly

Mutation can occur in two ways:

1. Through rebinding

    a co.lang.int = 10;
    a = 20;

    The above statement is valid because a is valid identifier and named handle to `co.lang.int` type.

    10 = 20; // ❌ invalid

    The above statement is invalid because  10 is not a valid identifier and if 10 in literal object type there is no named handle.
    
    The rebinding happens through named handles which are valid identifiers of `folang`

2. Through a property or method

    An object may also be mutated through one of its mutable properties or methods.
    a co.lang.int = 10;
    a.value = 20;

    A literal object cannot be mutated directly because it does not provide properties/methods that can be accessed.

    10.value = 20; // ❌ invalid

---

```folang
a co.lang.int = 10;
a.value = 30;
```
is valid after binding, a bare literal such as `10` cannot be mutated directly because there is no name or handle through which to perform mutation.


#### Not the Same as `toSnapshot(...)`

A literal object created by a literal expression is **not** the same thing as the internal snapshot representation produced by `co.utils.toSnapshot(...)`.

- literal expression → anonymous object
- `toSnapshot(x)` → internal snapshot representation used to reconstruct a fresh local object later

These are related concepts, but they are not the same mechanism.

#### Types of Literal objects:
   
     1. Simple types 
     2. Complex or compound types (UDTS)

'''folang
Simple types are in lteral form like 10, 'A', "A" etc.,

k co.lang.int=10;

Compound types are in Json form 


// Employee.fol
_  co.lang.class{

     id co.lang.string;
     name co.lang.string;
}

k Employee = Employee { id: '10', name: "ABC" };

> Even though compound literals are ended with brace block we need to end with semicolon as it is value not block.

'''

#### Summary

- literal objects are real objects
- literal expressions create anonymous objects
- identical literals compare equal by value
- identical literals are not automatically the same object
- literal-created objects are mutable by default once bound to a handle
- only `makeImmutable(...)` makes an object immutable

---

### 4. Object Behaviour Policies

Any object can be given a behaviour policy using `co.utils.*`.

All four policy calls are **in-place transformations**:

- the object itself changes behaviour kind
- there is no wrapper object
- there is no alternate binding to capture
- the original name now refers to the transformed object

All policies are **deep by default**.  
They flow through nested structs, members, collection elements, and all reachable objects in the graph unless the specification later states otherwise.

---

#### 4.1 Immutable

```folang
co.utils.makeImmutable(positive_int)
```

After this call the object itself is immutable.

This is an in-place transformation, not a wrapper.

Any attempt to mutate the object or any reachable object beneath it is a **compiler error** where detectable statically, and a **runtime error** otherwise.

```folang
positvie_int co.lang.int = 10;
positive_int.value = 30   // ❌ compiler error
positive_int = 30         // ❌ compiler error
```

For nested objects:

```folang
// Employee.fol
_ co.lang.struct = { address Address; }
Address co.lang.Address = {city co.lang.string; state co.lang.string; lane co.lang.string;pin co.lang.string;}


emp Employee = Employee{
    address = Address{
            city: "Pune";

    }
}
co.utils.makeImmutable(emp);

emp = Employee{
    address=Address{
            city: "ABC";

    }
} // ❌ compiler error


emp.address = newAddress              // ❌ compiler error
emp.address.city.value = "Mumbai"     // ❌ compiler error
```

Immutability is deep and total.

---

#### 4.2 Value Immutable

```folang
positive_int co.lang.int = 20;

co.utils.makeValueImmutable(positive_int);

positive_int.value = 30; // ❌ current object is immutable
positive_int = 30;       // ✅ binding may reference a new object

co.utils.makeValueImmutable(emp);

emp.address.city = "ABC";       // ❌
emp.address.city.value = "ABC"; // ❌

emp = Employee{
    address= Address{
        city: "ABC";
    }
}; // ✅

```
---
#### Difference between Immutable and Immutable Value
```folang
    makeValueImmutable(x)
        └── current object graph cannot change
    
    makeImmutable(x)
        ├── current object graph cannot change
        └── binding cannot be reassigned

```
---

##### Table

| Operation               | Binding   | Current value/object graph |
| ----------------------- | --------- | -------------------------- |
| `makeValueImmutable(x)` | Mutable   | Immutable                  |
| `makeImmutable(x)`      | Immutable | Immutable                  |

---

#### 4.3 Shared

```folang
co.utils.makeShared(positive_int)
```

A Shared Object is safe for concurrent and multi-threaded use.

The programmer continues to use the same object model — same accessors, same syntax — while FoLang guarantees the required safety properties internally.

Shared behaviour is deep:

- all reachable objects within the shared object are also shared

##### Note on Analogies

Comparisons to things like Java's `AtomicInteger` or `ConcurrentHashMap` are **analogies only**.

They may help explain the concept, but they are **not part of the formal language contract**.

FoLang specifies the behavioural guarantee, not the exact runtime implementation.

A good explanatory statement is:

> A shared `co.lang.int` may be thought of similarly to an atomic integer, and a shared map may be thought of similarly to a concurrent map. These comparisons are explanatory only. FoLang does not require any particular internal runtime representation.

---

#### 4.4 CopyOnWrite

```folang
co.utils.copyOnWrite(positive_int)
```

A CopyOnWrite Object passes by reference like a normal object.

The copy is deferred.  
It is created only when mutation is attempted in a context that must not affect the original source object.

```folang
process(a co.lang.int)->() = {
    a.value = 99;   // deep copy is made here — caller's object unchanged
}

co.utils.copyOnWrite(positive_int)
process(positive_int)

// positive_int still holds its old value
```

Copy-on-write is deep.  
When a copy is made, the entire reachable object graph is copied.

Cyclic references must be handled by the runtime/compiler's structural clone logic with proper identity tracking.

---

#### 4.5 toSnapshot

```folang
co.utils.toSnapshot(positive_int)
```

`toSnapshot` converts an object into a **snapshot representation** — a value descriptor, not a live object. The snapshot representation itself cannot be mutated.

When passed to a function, the compiler/runtime uses the snapshot representation to construct a fresh independent local variable bound to the parameter name. That local is a normal mutable object with no shared identity with the original. All of this happens automatically — the developer writes only `co.utils.toSnapshot(positive_int)`.

```folang
process(a co.lang.int)->() = {
    a.value = 99;   // mutates the fresh local — positive_int completely unaffected
}

process(co.utils.toSnapshot(positive_int))

// positive_int unchanged
```

The flow:

```
positive_int
    ↓  co.utils.toSnapshot(positive_int)
    snapshot representation         ←  value descriptor — immutable by nature, not by policy
    ↓  passed to function
    compiler constructs fresh local 'a' from the snapshot representation
    'a' is a new independent normal object — mutations never reach positive_int
```

`toSnapshot` is deep — the snapshot representation covers the entire reachable object graph.

---

### 5. Policy Summary

| Object Kind | Mutation allowed | Caller sees mutation | Thread safe | Copy on write | Deep |
|---|---|---|---|---|---|
| Literal expression object | ❌ directly in source — no handle | — | value-like use | — | ✅ |
| Normal (default) | ✅ via accessor | ✅ | ❌ | ❌ | — |
| Immutable | ❌ | — | ✅ | — | ✅ |
| Shared | ✅ via accessor | ✅ | ✅ | ❌ | ✅ |
| CopyOnWrite | ✅ on own copy | ❌ | ✅ | ✅ | ✅ |
| toSnapshot result | ✅ on reconstructed local object | ❌ | independent snapshot | — | ✅ |

---

### 6. No Type Fragmentation

FoLang deliberately avoids special types for mutability or concurrency concerns.

There is no need for separate public type families such as:

```text
ImmutableInt
SharedMap
CopyOnWriteList
AtomicInt
ConcurrentMap
```

Instead, FoLang keeps the type model uniform and applies behaviour policies through `co.utils.*`.

In many ecosystems, a programmer must constantly choose between:

```text
normal integer      vs atomic integer
normal map          vs concurrent map
mutable value       vs immutable wrapper
raw structure       vs copy-on-write structure
```

FoLang instead aims to provide:

- one object model
- one programming style
- one mental model

while still allowing the programmer to opt into immutability, sharing, copy-on-write, or literal conversion when needed.

---

### 7. What Still Needs Precision

The philosophy is sound, but the formal specification still needs to define these precisely:

```text
aliasing behaviour                     → when two names refer to the same object
rebinding vs mutation interaction      → edge-case coverage
deep policy propagation rules          → exactly which objects are reachable
cyclic reference handling in COW       → identity-tracking specification
visibility of mutation across calls    → formal function-call model
identity vs value equality             → what operators or built-ins expose identity
policy stacking                        → can an object be both shared and COW?
                                          can an object be both shared and immutable?
```

---

### 8. Formal Philosophy Statement

> All managed FoLang objects use reference semantics by default. `co.lang.cstruct` is an explicitly value-semantic ABI representation and is an exception to managed-object reference semantics.  
> In FoLang, everything is an object and managed objects are mutable by default.  
> Assignment of managed objects copies references, `co.lang.cstruct` assignment copies values, and `==` compares values deeply.  
> Developers may opt into immutability, shared behaviour, copy-on-write behaviour, or literal conversion depending on their needs.  
> All behaviour policies are deep — they apply to the entire reachable object graph unless stated otherwise by the formal specification.  
> This policy model is uniform across managed types, so programmers do not need separate type families for ordinary, immutable, concurrent, or snapshot-oriented use.  
> Familiar analogies such as atomic integers or concurrent maps may help explain the design, but they are not part of the formal implementation contract.


----

<a id="folang-definition-and-documentation-license"></a>

## FoLang Language Definition and Documentation License

Unless otherwise stated, the copyrightable material contained in the FoLang language definition and documentation is licensed under the [Creative Commons Attribution 4.0 International License](https://creativecommons.org/licenses/by/4.0/).

This licensed material includes the original expression, organization, and presentation of:
The CC BY 4.0 licence applies to the copyrightable expression, organization, and presentation of the FoLang language definition and documentation, including:

* the FoLang language specification;
* original FoLang-specific syntax forms;
* grammar productions and grammar notation;
* rules governing how FoLang syntax forms may be combined;
* semantic rules and behavioural descriptions;
* name-resolution and scope-resolution rules;
* type-system definitions and constraints;
* execution-model and control-abstraction descriptions;
* compiler-behaviour descriptions;
* source-code examples demonstrating FoLang syntax and semantics;
* diagrams, tables, illustrations, and formal notation;
* reference documentation;
* explanatory and instructional material.

This includes the documented expression and presentation of FoLang-specific constructs such as:

```folang
(booleanExpression).do({
    ...
}).otherwise.do({
    ...
});
```
and other original FoLang syntax, grammar, and semantic-rule descriptions contained in this document.


Under CC BY 4.0, this material may be:

- copied and redistributed in any medium or format;
- adapted, modified, translated, and extended;
- used for commercial or non-commercial purposes.

A person exercising these permissions must:

- give appropriate attribution;
- provide a link to the CC BY 4.0 licence;
- indicate whether modifications were made;
- retain notices supplied with the licensed material where required;
- not imply endorsement by the FoLang project or its contributors.

### Independent Use and Implementation

This licence applies to the copyrightable expression contained in the FoLang language definition and documentation.

It does not restrict the independent use or implementation of programming-language ideas, concepts, features, procedures, systems, or methods of operation.

For example, this licence does not claim exclusive rights over general programming-language concepts such as:

- continuations;
- modules;
- structs and classes;
- dynamic, lexical, or mixed scope;
- functions and closures;
- pattern matching;
- algebraic data types;
- dependent types;
- concurrency and parallelism;
- coroutines and asynchronous execution. etc.,

Another project may independently implement such concepts without copying the copyrightable expression of the FoLang documentation.

The FoLang-specific documentation describing how these concepts are expressed through FoLang syntax, grammar, semantic rules, examples, diagrams, and explanatory text remains subject to this licence when that material is copied or adapted.

### Software Licences

This licence does not change or replace the licences applied to:

- the FoLang frontend source code;
- the default backend source code;
- other FoLang software components;
- generated compiler binaries;
- third-party software;
- third-party documentation or assets.

The FoLang frontend and backend continue to be governed by their separately stated software licences.

### Other Rights

CC BY 4.0 does not grant rights relating to:

- trademarks;
- project names and branding;
- logos;
- patents;
- third-party material;
- privacy, publicity, or personality rights.

Such rights, where applicable, remain governed separately.

### Suggested Attribution

> FoLang Language Definition and Documentation, including its syntax, grammar, and semantic-rule descriptions, by [Kemeswara Rao Mithipati](mailto:samkrao@gmail.com) and FoLang contributors, licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

When modifications are made, the attribution should also indicate that the material was modified.

Example:

> Based on the FoLang Language Definition and Documentation by [Kemeswara Rao Mithipati](mailto:samkrao@gmail.com) and FoLang contributors, licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). Modified from the original.
---


# Appendix A - Complete FoLang EBNF Grammar

The following grammar is the normative lexical and syntactic grammar for
FoLang.

[{{FOLANG_EBNF}}](./grammar/folang.ebnf)

# Appendix B - Grammar Decisions and Rationale

[{{GRAMMAR_DECISIONS}}](./grammar/grammar-decisions.md)
