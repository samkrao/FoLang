# FoLang Language Specification

---

## Operators

###### Arithmetic operators
`+`, `-`, `*`, `/`, `%`, `**`, `++`, `--`

###### Logical operators
`&&`, `||`, `!`, `&`, `|`

###### Comparison operators
`==`, `!=`, `<`, `>`, `<=`, `>=`

###### Other operators
`@`, `#`, `!`, `~`, `$`, `^`, `(`, `)`, `_`, `` ` ``, `?`, `{`, `[`, `]`, `}`, `\`, `:`, `;`, `"`, `'`, `=`, `.`, `?=`, `:=`, `::=`, `,`, `..`, `...`, `<..`, `..<`, `<..<`, `=>>`, `=>`, `->`, `<-`, `->>`, `<->`

###### Reserved operators
`λ`, `⒪`, `â`, `Ť`, `∀`, `∃`, `○`, `ö`, `∪`, `Ṡ`, `Ŝ`, `ṁ`, `𝚷`, `⇛`, `𝑓`, `𝒯`, `𝘷`, `𝓕`, `↓`, `λ`, `∂`, `⊥`, `↧`, `⇓`

###### Reserved words
`co`, `let`, `this`, `self`, `for`, `forall`

###### Difference between `this` and `self`
- `this` is for instances and objects
- `self` is for classes
- `static` — no shortcut; can be on variable or classname
- Both `self` and `this` can access member variables

### Custom Operator Definition & Overloading

```folang
@co.dap.operator(symbol='+', mode=overload)
add(a Employee, b Employee)->(Employee)={}

// mode=override not supported in foreseeable future; compiler throws error

@co.dap.operator(
    symbol='∪',
    mode=define,
    fixity=infix,
    precedence=60,
    associativity=left,
    arity=binary,
    commutative=true,
    idempotent=true,
    identity="∅",
    foldable=true,
    vectorizable=false,
    distributes_over=['∩'],
    desugar="intrinsic:set_union"
)
```

**fixity values:** `infix`, `postfix`, `prefix`, `circumfix`, `postcircumfix`, `prescircumfix`, `mixfix`, `ternary`, `distfix`

---

## Import Statement

```folang
@co.ddap.import(path="", package="", realm="", parent-realm="", as="")
```

### Directive Fields

| Field | Description |
|---|---|
| `path` | Canonical path resolving to exactly one `.fol` file. No wildcards, no directory imports. |
| `package` | Package declared inside the referenced file. Must match exactly. |
| `realm` | Isolation domain. Default is `main`. For third-party libs, plugins, version coexistence. |
| `parent-realm` | Realm hierarchy parent. Defaults to `core`. |
| `as` | **Mandatory.** Domain/capability alias. All symbols accessed as `<as>.<package>.<symbol>`. |

### Realm Hierarchy

```
core  (folang core realm — restricted)
  |
  └── user defined
```

FoLang searches symbols from all leaf nodes traversing up to `core` before reporting not found.

### Alias–Realm Binding Rules

Within a single realm, a module identified by a given `path` **must not** be imported under more than one domain alias.

**Invalid — same realm, same path, different aliases:**
```folang
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main", as="hr")
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main", as="accounts") // ERROR
```

**Invalid — same realm, same alias, different paths:**
```folang
@co.ddap.import(path="/myapp/hr/User",    package="dto", realm="main", as="hr")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto", realm="main", as="hr")   // ERROR
```

**Valid — different realms with parent-realm:**
```folang
@co.ddap.import(path="/myapp/hr/User",    package="dto", realm="main",    as="hr")
@co.ddap.import(path="/myapp/hr/User",    package="dto", realm="plugin1", as="accounts")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto", realm="pluginA", parent-realm="main", as="hr")
```

```
core
  |
 / \
main  plugin1
 |
pluginA
```

**Valid — flat realms with distinct aliases:**
```folang
@co.ddap.import(path="/myapp/hr/User",    package="dto", realm="main",    as="hr")
@co.ddap.import(path="/myapp/hr/User",    package="dto", realm="plugin1", as="accounts")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto", realm="pluginA", as="v1.hr")
```

Usage:
```folang
hr.dto.Employee
v1.hr.dto.Employee
```

---

## Variable Declaration

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
someR ?= "Kamesh"; // if not defined, define and initialize; else assign value
```

### Pointer Declaration

```folang
somePtr    co.lang.int->(*);
someDblPtr co.lang.int->(**);
```

### Array Declaration

```folang
someArray       co.lang.int->([5]);
someDblArray    co.lang.int->([2,3]);
someJaggedArray co.lang.int->([2][3]);
someVLArray     co.lang.int->([...]);
someZeroLA      co.lang.int->([0]);
someZeroDimA    co.lang.int->([.]);
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
rangeI := 1..10;      // [1, 10]   ExcludeStart=false, ExcludeEnd=false
rangeS := 0<..5;      // (0, 5]    ExcludeStart=true,  ExcludeEnd=false
rangeL := 0..<100;    // [0, 100)  ExcludeStart=false, ExcludeEnd=true
rangeB := 0<..<100;   // (0, 100)  ExcludeStart=true,  ExcludeEnd=true
rangeE := ..100;      // open lower bound  (_, 100]
rangeF := 1..;        // open upper bound  [1, _)
```

### Auto and Dynamic Variable Declaration

```folang
someAutoVar    co.lang.auto    = "Hello"; // type inferred from value; initialization required
someDynamicVar co.lang.dynamic;           // dynamic typing
```

### Values

```folang
someVar co.lang.data = 10; // initialization required
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
```

### Named Returns

```folang
doManythings(a co.lang.int, b co.lang.int->(&, meta={type=out}))->(r co.lang.int, e co.lang.exception)={}
```

---

## Fat Pointers

```folang
x co.lang.int->(*, kind="", meta={});

co.lang.int->(*, meta={});

y co.lang.int->(*, meta={len:co.lang.usize, vtab:somepkg.VTable->(*)})

z co.lang.int->(*,kind=region, meta={})
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
z co.lang.int->(*,kind=relative, meta={})
```

---

## Types and Kinds

```folang
x co.lang.int = 10;
x.type() → co.lang.int
x.kind() → co.lang.nothing

x co.lang.data = 10;
x.type()        → co.lang.value
x.kind()        → co.lang.data
x.type().type() → co.lang.int   // to get the actual type

x co.lang.auto = 10;
x.type() → co.lang.int   // inferred at compile time, static
x.kind() → co.lang.data

x co.lang.dynamic = 10;
x.type() → co.lang.int   // can vary at runtime
x.kind() → co.lang.data
```

### Type Declarations

```folang
// Alias
x co.lang.type = co.lang.int;

// New
x co.lang.newtype = co.lang.int;

// Opaque
x co.lang.opaquetype = co.lang.int;

// ADT (tagged union)
y co.lang.type = co.lang.int | co.lang.char;

// Subtype / covariant
test co.lang.subtype = co.lang.int;

// Supertype / contravariant
test co.lang.supertype = co.lang.int;
```

### Type Constructor

```folang
@co.dap.hokrt
Option(T) co.lang.type = Some(T) | None();
```

### Dependent Types

```folang
identity(x co.lang.int)->(x.type) = x

x co.lang.dependentType->(kind=length) = co.lang.int->([5]);
```

#### Types Computed from Runtime Values

```folang
someType := somefun(value)

somefun(value co.lang.int)->(co.lang.type)={
    (value < 100).return(co.lang.string).otherwise.return(co.lang.bool);
}

// or with annotation
@co.dap.typefromvalue
somefun(value co.lang.int)->(co.lang.type)={
    (value < 100).return("hello").otherwise.return(co.const.true);
}

// compile-time eager
@co.dap.comptime
@co.dap.eager
chooseType(value co.lang.int)->(co.lang.type)={
    (value < 100).return(co.lang.string).otherwise.return(co.lang.bool);
}

// tagged value
somefun(value co.lang.int)->(co.lang.tag) = {
    (b < 100).return(co.lang.tag(co.lang.string, "Hello")).otherwise.return(co.lang.tag(co.lang.bool, co.const.true));
}
```

---

## Data Structures

### Package Declaration

```folang
mypackage co.lang.package={

}
```

### Struct Declaration

```folang
myStruct co.lang.struct={
    field1 co.lang.int;
    field2 co.lang.string;
    field3 co.lang.bool;
}
```

#### Struct Rules

```
structs cannot extend other structs
structs cannot have methods
structs can compose other structs
structs cannot have default values to fields/members
structs can embed other structs (Go lang like)
structs can have associated functions (Go lang like)
```

#### Struct Embedding

Embedding promotes fields of an embedded struct directly into the outer struct — they act as the outer struct's own fields at construction and access sites. This is distinct from composition where the embedded struct is a named field.

```folang
E co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}

// ✅ No conflict — id and name promoted as B's own fields
B co.lang.struct = {
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
B co.lang.struct = {
    name co.lang.string;   // conflicts with E.name
    E;
    age  co.lang.float;
}
// Fix 1 — rename B's conflicting field
// Fix 2 — use explicit composition instead: e E;
```

```folang
// Explicit composition — no promotion, always qualified access
B co.lang.struct = {
    name co.lang.string;
    e    E;               // named field — no conflict, no promotion
    age  co.lang.float;
}

b.name    // B's own name
b.e.id    // E's id — always explicit
b.e.name  // E's name — always explicit
```

#### Embedding Rules

| Situation | Behavior |
|---|---|
| Embedded field, no conflict | Promoted — acts as the outer struct's own field |
| Embedded field, name conflict with outer | ❌ Compiler error — rename or use composition |
| Multiple embeds, no conflict between them | All fields promoted |
| Multiple embeds, conflict between embedded structs | ❌ Compiler error |
| Explicit composition (`e E`) | No promotion — always accessed via `b.e.field` |

> Unlike Go, FoLang does **not** silently shadow conflicting fields. Any name conflict is a compiler error — the programmer must make a conscious decision to rename or switch to explicit composition.

### Enum Declaration

```folang
myEnum co.lang.enum={
    Variant1,
    Variant2,
    Variant3
}
```

### Union Declaration

```folang
myUnion co.lang.union={
    intValue co.lang.int;
    strValue co.lang.string;
}
```

---

## Classes

```folang
Employee co.lang.class ={
    getEmployeeDetails()->(Employee) = empmodule.getEmployeeDetails;
    // assigning module function to class's method

    getEmployeeInfo()->(Employee) =>> empmodule.getEmployeeDetails();
    // delegating — internally redirecting the call to module function
}

// $1, $2, $3 ... are previous results captured as bind variables
Emp co.lang.class={
    dosomething(a co.lang.int, b co.lang.int)->(co.lang.int)=>>somePack.someMethod(a)=>>someOthPack.someOtherMeth($1, b);
}
```

### Inner Type Declarations

Classes can declare types inside their body. Inner types are scoped to the class and accessible outside via the qualified name `ClassName.TypeName`.

```folang
Employee co.lang.class = {

    // Inner ADT — scoped to Employee
    Status co.lang.type = Active | Inactive | Pending;

    // Inner struct — scoped to Employee
    EmployeeRecord co.lang.struct = {
        id     co.lang.int;
        status Status;
    }

    @co.dap.method.instance
    getStatus()->(Status) = { ... }

    @co.dap.method.instance
    getRecord()->(EmployeeRecord) = { ... }
}

// Accessing inner types from outside — qualified name
s Employee.Status = Employee.Status.Active;
r Employee.EmployeeRecord = Employee.EmployeeRecord{ id: 1, status: Employee.Status.Active };
```

Inner types follow the same access rules as methods — `@co.dap.private`, `@co.dap.public` etc. apply.

### Method Types

```folang
Employee co.lang.class ={

    @co.dap.method.static
    getEmployee()->(Employee) ={}

    @co.dap.method.instance
    getEmployee()->(Employee)={}

    @co.dap.method.class
    getEmployee()->(Employee) ={}

    @co.dap.method.object
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
    H: { with:true }
)
test co.lang.class->(uses=[], implements=[], extends=[], inherits=[], with=package.type, composes=[]) ={
    getTest(id int)->(test) ={}
}
```

### The @@new and @@init Methods

`@@new` and `@@init` are lifecycle methods — compiler-owned, not user-definable outside the class. `@@` signals they are restricted lifecycle symbols, not regular methods.

```folang
@co.dap.generic(type={T:{typename}, R:{typename}})
Employee co.lang.class = {

    id T
    name R

    // @@new is provided by default even if not overridden.
    // Override only when you genuinely need to change allocation behavior.

    @co.dap.method.class
    @co.dap.private
    @@new()->(co.lang.uninit) = { self.return co.const.none }

    @co.dap.method.class
    @co.dap.public
    @@new(a co.lang.typevalue, b co.lang.typevalue)->(co.lang.uninit) = {
        // Manual type aliasing — @co.dap.generic handles this automatically
        // Override @@new only when you need custom allocation logic
        T co.lang.type = a
        R co.lang.type = b

        // self keyword is allowed only in class methods
        self.parent.@@new();

        // uninit instance method internally calls @@new and @@init
        self.return co.lang.uninit.instance(Employee, self);
    }

    @co.dap.override
    @co.dap.constructor(access=private)
    @@init() = {}

    @co.dap.override
    @co.dap.constructor(access=public)
    @@init(id T, name R) = {
        this.parent.@@init();
        this.id   = id;
        this.name = name;
    }

    getEmployee(id T)->(Employee) = {}
}
```

---

## Module Declaration  🟩

```folang
EmployeeModule co.lang.signature={
    // module contents
    getEmployee(id co.lang.int)->(Employee);

}

@co.dap.module(signature=EmployeeModule)
EmployeeModImpl co.lang.module->(signature=EmployeeModule, matches=EmployeeModule) = {

    Employee co.lang.struct={
        Id co.lang.int;
        Name co.lang.string;
    }

    getEmployee(id co.lang.int)->(Employee)={
        this.return Employee{
            Id:10, Name:"Rao",
        };
    }

}

mm EmployeeModule = EmployeeModuleImpl;
v mm.Employee = mm.Employee{Id:10, Name:"Rao"};
mm.getEmployee(10);
```

---

## Functions

### Normal

```folang
fun1 (k co.lang.int, b co.lang.char)->(co.lang.int, co.lang.char)={
    // function body
}
```

### Local Type Declarations

Functions can declare types locally. Local types are scoped to the function body only — they cannot appear in the function's parameter or return types, and are not accessible outside.

```folang
processEmployee()->(co.lang.bool) = {

    // Local ADT — scoped to this function only
    Status co.lang.type = Active | Inactive | Pending;

    // Local struct — scoped to this function only
    LocalRecord co.lang.struct = {
        id     co.lang.int;
        status Status;
    }

    r := LocalRecord{ id: 1, status: Active };
    this.return r.status == Active;
}
```

```folang
// ❌ Compiler error — local type cannot appear in return type
getRecord()->(LocalRecord) = {
    LocalRecord co.lang.struct = { id co.lang.int; }
    this.return LocalRecord{ id: 1 };
}
```

Local types can be passed to inner functions defined within the same scope:

```folang
process()->(co.lang.int) = {
    Status co.lang.type = Active | Inactive;

    // Inner function — can use Status because it shares the same scope
    check(s Status)->(co.lang.bool) = {
        this.return s == Active;
    }

    this.return check(Active).return(1).otherwise.return(0);
}
```

### Curried

```folang
add(first co.lang.int)(second co.lang.int)->(co.lang.int)={
    first + second
}
```

### Closure

```folang
adder() -> ((co.lang.int) -> co.lang.int) ={
    sum co.lang.int = 0
    this.return  (x co.lang.int) -> (co.lang.int) = {
        sum += x
        this.return sum
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
someFArg co.lang.type = (co.lang.int, co.lang.int)->(co.lang.int)
someFRet co.lang.type = (co.lang.int)->(co.lang.int)

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

someFunction (r someFArg)->(someFRet)={}
```

### Anonymous Functions and Objects

#### Anonymous Classes/Types

```folang
emp := co.lang.class{};

empObj := emp.new();

empobj1 := co.lang.class{
    name string
}.new();
```

#### Anonymous Functions

```folang
add := (a int, b int) -> (int) {
    this.return a + b;
};

res := (a int, b int) -> (int) {
    this.return a * b;
})(10, 20);
```

#### Lambda

Only allowed as an inline callback argument to collection operations (e.g. `map`, `filter`, `reduce`, `forEach`, `sortBy`, `groupBy`). Using `|...|` anywhere else is a syntax/lint error.

```folang
// Syntax
|x, y| => x + y;

// Collection use — allowed
nums.map(|x| => x*x)
words.filter(|s| => s.len() > 3)
pairs.reduce(|acc, e| => acc + e, 0)
dict.map(|k, v| => v * 10)
list.sortBy(|a, b| => a.score - b.score)
```

#### Inner Function

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

### Function Objects

```folang
myobj co.lang.function = (a co.lang.int, b co.lang.int)->(co.lang.int)={
    this.return a + b;
}

add (a co.lang.int, b co.lang.int)->(co.lang.int){ this.return a + b; }
oObj co.lang.function = add;

funtype co.lang.type = (a co.lang.int, b co.lang.int)->(co.lang.int);

closure(factor int) => (x int) = x * factor;

curry(factor int)(val int) = factory * val;
```

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
    if k.omitted{

    }else{

    }
}
```

### Named Parameters

```folang
fun1(~k co.lang.int)->()={

}
```

### Function Delegates

```folang
@co.dap.delegate someDelegate co.lang.delegate = (a co.lang.int, b co.lang.int) -> (co.lang.int, co.lang.int);
```

### Function Chaining

```folang
fetchEmployee(empId co.lang.string)->(Employee)=>>empMod.getEmployee(this, empId);
```

### Associated Functions

```folang
(emp empMod.Employee) fetchEmployee(empId co.lang.string)->(empMod.Employee)=>>empMod.getEmployee(emp, empId);
```

### Indexer

```folang
MyList co.lang.struct ={
    eles co.lang.int->([*]);
}

@co.dap.indexer(symbol="[]")
(g MyList) get(index co.lang.int)->(co.lang.int) ={
    this.return g.eles[index]
}

@co.dap.indexer(symbol="[]=")
(g MyList) set(index co.lang.int, value co.lang.int)->() ={
    g.ele[index] = value
}
```

### Inline

```folang
@co.dap.inline
add(a co.lang.int, b co.lang.int)->(co.lang.int) ={
    this.return a + b;
}
```

### Lazy

```folang
@co.dap.lazy
x = add(1, 2);
```

### Native Functions

```folang
@co.dap.native
nativeMethod(a co.lang.int, b co.lang.int)->(co.lang.int) ={
    // native implementation
}
```

---

## Generics

```folang
@co.dap.generic(
    at=runtime,
    type={
        T: {variance:invariant, bound=Number, kind:param, impredicative:false},
        R: {variance:invariant, bound=Number, kind:return}
    }
)
add(a T, b T)->(R) = { this.return a + b; }
```

**Generic annotation fields:**

- `at` — `runtime` | `compiletime` (acts like C++ templates)
- `refied` — `true` | `false`
- `where` — `usesite` | `callsite`

**typename/type attributes:**

| Attribute | Values |
|---|---|
| variance | `covariant`, `invariant`, `contravariant` |
| bound | type to bind |
| kind | `param`, `result`, `var`, `arg` |
| default | default type |
| nullable | bool |
| inclusive | bool |
| impredicative | bool — when `true`, allows `T` to be instantiated with a `forall` type (v2) |
| typekind | `type`, `class`, `function`, `module`, `package` |
| types | list of allowed types for constraints |

### Generic Functions — Parameters and Return Values

#### Rank-1: Outer function is generic; parameter uses the same type variable

`T` is fixed at the call site before the function parameter is used. The passed function is already monomorphic inside the body.

**Syntax 1 — Inline signature**
```folang
@co.dap.generic(type={T:{variance:invariant}})
someFunction(f (T, T)->(T), a T)->(T) = {}
```

**Syntax 2 — Named type alias**
```folang
@co.dap.generic(type={T:{variance:invariant}})
someFArg co.lang.type = (T, T)->(T)

someFunction(f someFArg, a T, b T)->(T) = {}
```

**Syntax 3 — Function objects**
```folang
someFArg co.lang.function = (a co.lang.int, b co.lang.int)->(co.lang.int) = {
    this.return a + b;
}

@co.dap.generic(type={T:{variance:invariant}})
someFunction(f someFArg, a T)->(T) = {}
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
someFArg co.lang.type = forall(T).(T, T)->(T)

someFunction(f someFArg)->(co.lang.int) = {}
```

**Syntax 3 — Function objects**
❌ Compiler error. `co.lang.function` is a concrete value — it holds a monomorphic function instance and cannot carry `forall`. There is no need for a separate `co.lang.polyfunction` kind either; `forall(T).(T)->(T)` is already a type, and `co.lang.type` is the correct and sufficient container for it. Use Syntax 1 or Syntax 2 for Rank-2:

```folang
// Correct — Syntax 2 with co.lang.type
someFArg co.lang.type = forall(T).(T, T)->(T)

someFunction(f someFArg)->(co.lang.int) = {}
```

---

#### Returning Generic Functions

**Rank-1 return**
```folang
@co.dap.generic(type={T:{variance:invariant}})
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
rank2FnType  co.lang.type = forall(T).(T, T)->(T)
rank3ArgType co.lang.type = (rank2FnType) -> (co.lang.int)

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
@co.dap.generic(type={T:{variance:invariant}})
box(x T) -> (Box(T)) = {}

// Impredicative call — T being set to forall(U).(U)->(U)
result := box(forall(U).(U)->(U));   // ❌ not legal without explicit opt-in
```

Most type systems reject this by default. FoLang takes an opt-in approach.

**v1 Workaround — Option C: Wrapping with `co.lang.type`**

Not true impredicativity but solves 90% of practical cases:

```folang
polyId co.lang.type = forall(U).(U)->(U)

// box takes co.lang.type — no impredicative unification needed
box(x co.lang.type) -> (Box(co.lang.type)) = {}

result := box(polyId);   // ✅ works — T is co.lang.type, not a forall type
```

**v2 — Option A: `impredicative:true` in `@co.dap.generic`**

Explicit opt-in via the existing annotation. The compiler only permits `forall` instantiation where declared:

```folang
@co.dap.generic(
    type={T:{variance:invariant, impredicative:true}}
)
box(x T) -> (Box(T)) = {}

polyId co.lang.type = forall(U).(U)->(U)
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

### Generic Types

```folang
@co.dap.generic(typename=T)
LinkedList co.lang.struct={
    value T
    next  LinkedList
    prev  LinkedList
}

k := LinkedList.new(co.lang.int);

@co.dap.generic(type={T:{typename}, R:{typename}})
Employee co.lang.class ={
    id   T
    name R

    @co.dap.override
    @co.dap.constructor(access=private)
    @@init() = {}

    @co.dap.override
    @co.dap.constructor(access=public)
    @@init(id T, name R) = {
        this.parent.@@init();
        this.id   = id;
        this.name = name;
    }

    getEmployee(id T)->(Employee)={}
}

a := Employee.new(co.lang.int, co.lang.string);
b := a.@@init(1, "Rao");
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

`@co.dap.generic` is the **one and only** way to declare generic functions, structs, classes, and other named things. `forall` at declaration level is a **compiler error**.

---

#### Where `forall` Is Allowed — Type Expression Form Only

`forall(T).` followed by an anonymous type body. The `.` is the syntactic signal that what follows is a type body, not a declaration name.

Pattern:
```
forall(T).  <anonymous type body>
```

```folang
// co.lang.type alias — naming a polymorphic type for reuse
someFArg co.lang.type = forall(T).(T, T)->(T)

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
@co.dap.generic(type={T:{variance:invariant}})
identity(x T)->(T) = {}
```

```folang
// ❌ compiler error
forall(T) LinkedList co.lang.struct = { value T; next LinkedList; }

// ✅ correct
@co.dap.generic(typename=T)
LinkedList co.lang.struct = { value T; next LinkedList; }
```

```folang
// ❌ compiler error — Rank-1 generics belong to @co.dap.generic
forall(T) someFunction(f (T,T)->(T), a T)->(T) = {}

// ✅ correct
@co.dap.generic(type={T:{variance:invariant}})
someFunction(f (T,T)->(T), a T)->(T) = {}
```

---

#### Quick Reference

| Form | Status | Context |
|---|---|---|
| `forall(T) name ...` | ❌ Compiler error | Declaration level — use `@co.dap.generic` instead |
| `forall(T).(T)->(T)` | ✅ Allowed | Type level only — Rank-2/3 param, return, `co.lang.type` alias |

**The rule in one sentence:** `forall(T).` is a type constructor for anonymous polymorphic types; it is never a declaration keyword.

---

## Templates

### Typed

```folang
@co.dap.template
add(a co.lang.int, b co.lang.int)->(co.lang.int) ={
    this.return a + b;
}
```

### Untyped

```folang
@co.dap.template
add(a, b)->(co.lang.untyped) ={
    this.return a + b;
}
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
        co.macro.esc(y) = 42
        println("Inside macro: y = ", y)
    });
}

// c. Debug macro with gensym
@co.dap.macro
debug(expr)->(co.lang.untyped)={
    let tmp = co.macro.gensym(co.lang.var, "tmp")
    this.return co.macro.quote({
        tmp = co.macro.esc(expr)
        println("Result: ", tmp)
        tmp
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

blockormacro co.lang.Kind = block | macro

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
## Annotations, Directives, Pragmas and Decorators

```
// Annotation — static object, can carry data


myAnnotation co.lang.object->(for=annotation) = {
    value   co.lang.string;
    enabled co.lang.bool;
}

// Directive — static object, compiler instruction, can carry data
myDirective co.lang.object->(for=directive) = {
    target co.lang.string;
}

// Pragma — static object, compiler hint, can carry data
myPragma co.lang.object->(for=pragma) = {
    level co.lang.int;
}

// Decorator — function, transforms target, returns
@co.dap.decorator
myDecorator(target co.lang.function)->(co.lang.function) = { }
```
---

## Zone Declaration

`@co.dap.zone` applies to `co.lang.package` only. It declares the capability tier of the entire package — not individual functions or files. A package has exactly one zone. Zone is a **boundary wall** — the only way to communicate across zones is through the public interface exposed by the package.

### Zone Levels

| Level | Purpose | Default |
|---|---|---|
| `application` | Standard application code | ✅ Default — no annotation needed |
| `systems` | Raw pointers, MMIO, heap allocators, `co.sys.unsafe` | 🔒 Requires feature enabled at install |
| `ffi` | C bindings, `@co.dap.native`, extern types, `co.sys.ffi` | 🔒 Requires feature enabled at install |

### Zone Declaration Syntax

```folang
// application zone — default, annotation optional
hrPackage co.lang.package = {
    calculateSalary()->(co.lang.float) = { ... }
    getEmployee(id co.lang.int)->(Employee) = { ... }
}

// systems zone — whole package is systems, no mixing
@co.dap.zone(level=systems)
driversPackage co.lang.package = {
    @co.dap.private
    doGpio()->() = { ... }

    @co.dap.private
    setupMmio()->() = { ... }

    @co.dap.public
    init()->(co.lang.bool) = { ... }    // only door out
}

or

driversPackage co.lang.package->(kind=system) = {
    @co.dap.private
    doGpio()->() = { ... }

    @co.dap.private
    setupMmio()->() = { ... }

    @co.dap.public
    init()->(co.lang.bool) = { ... }    // only door out
}

// ffi zone — C bindings only
@co.dap.zone(level=ffi)
bindingsPackage co.lang.package = {
    @co.dap.native
    getEmployee(id co.lang.int)->(CEmployee->(*)) = { }
}


or


bindingsPackage co.lang.package ->(kind=ffi)= {
    @co.dap.native
    getEmployee(id co.lang.int)->(CEmployee->(*)) = { }
}
```

### Zone Rules

```
@co.dap.zone applies to co.lang.package only — not functions, classes, modules
One zone per package — cannot mix zone levels within a package
Default zone is application — no annotation required
zone=systems requires systems feature enabled at install time
zone=ffi requires ffi feature enabled at install time
```

### Dependency Direction — One Way Only

```
application
    ↓ through public interface only
systems
    ↓ through public interface only
ffi
```

```folang
// ✅ application → systems through public interface
@co.ddap.import(path="src/drivers", package="driversPackage", as="drivers")

hrPackage co.lang.package = {
    setupHardware()->() = {
        drivers.driversPackage.init();       // ✅ public — allowed
        drivers.driversPackage.doGpio();     // ❌ private — compiler error
    }
}

// ❌ systems → application — compiler error
@co.ddap.import(path="src/hr", package="hrPackage", as="hr")   // ❌
@co.dap.zone(level=systems)
driversPackage co.lang.package = { }
// ERROR: zone=systems package cannot import zone=application package

// ❌ mixing zones in one package — compiler error
@co.dap.zone(level=systems)
mixedPackage co.lang.package = {
    doGpio()->() = { ... }            // systems
    calculateSalary()->() = { ... }   // ❌ application construct in systems zone
}
```

### Public Interface Must Use Application-Safe Types

The public signature of a systems or ffi package cannot expose raw pointers or systems types — the boundary wall keeps internal details internal:

```folang
// ✅ Clean public interface — application-safe types only
DriversSignature co.lang.signature = {
    init()                     -> (co.lang.bool);
    readSensor(id co.lang.int) -> (co.lang.float);
}

// ❌ Compiler error — raw pointer leaking through public interface
DriversSignature co.lang.signature = {
    getBuffer() -> (co.lang.byte->(*));   // error — systems type cannot cross zone boundary
}
```

### Cross-Zone Communication Summary

| Direction | Allowed? | How |
|---|---|---|
| `application` → `systems` | ✅ | Through public signature only |
| `application` → `ffi` | ✅ | Through public signature only |
| `systems` → `ffi` | ✅ | Through public signature only |
| `systems` → `application` | ❌ | Compiler error |
| `ffi` → `application` | ❌ | Compiler error |
| `ffi` → `systems` | ❌ | Compiler error |

---

## Pattern Matching

```folang
x co.lang.int = 10;

x.match.case(n: n > 10 => { n = n+100; "GT" }).case(_: n < 10 => "LT").default("EQ");

x.match(co.pattern.Type).case(co.lang.int => ...).case(co.lang.float => ...);
x.match(co.pattern.Value).case(0 => ...).case(1 => ...);
x.match(co.pattern.Instance).case(xx.CAT => ...).case(xx.DOG => ...).default("Animal");
x.match(co.pattern.Object).case(xx.Ball => "Ball").case(xx.CAT => "CAT").default("Unknown");
x.match(co.pattern.Shape).case(Point{x, y} => ...).default(_ => ...);

x.match(co.pattern.Any).case(co.lang.int => ...).case(co.lang.float => ...).case(0 => ...).default(_ => ...);

x.match(PositiveEvenMatcher).case(0 => "Neither even nor odd").case(2 => "First Even Prime").default(...);
```

> **Object vs Instance in FoLang:** Instance is from types of class/structs. Objects are anything — functions, classes, structs, types, etc.

> `_` is a special discard/wildcard variable usable only inside pattern matching, contains, and iterator constructs. Elsewhere `_` must be accompanied by an ASCII letter or number.

### Custom Matcher

```folang
@co.dap.matcher
Matcher(T) = {
    matchCase(value T, pattern co.lang.untyped) -> (co.lang.int, co.lang.MatchBindings);
    // int return: 0 = no match, >0 = match
}

PositiveEvenMatcher co.lang.Matcher->(for=Matcher, type=co.lang.int) = {
    matchCase(value co.lang.int, pat co.lang.untyped)->(co.lang.int, co.lang.MatchBindings) = {
        // user logic
    }
}
```

### Function Pattern

```folang
f(Some(x)) => { x + 1 }
f(None())  => { 0 }

// desugars to:
f(v) =>{
    v.match().case(x: Some(x) => x + 1).case(_: None() => 0);
}
```

---

## Monads, Applicatives, Functors, Monoids and Transformers

> `@co.dap.typeclass(kind=...)` is the single annotation for all typeclass definitions. `kind` specifies the algebraic structure — `Functor`, `Applicative`, `Monad`, `Monoid`, `Transformer`, or any user-defined kind. Instances of any typeclass always use `co.lang.instance`.

### Functor

```folang
@co.dap.typeclass(kind=Functor)
Functor(F) = {
    map(value F(A), f (A)->B) -> (F(B));
}

ListFunctor co.lang.instance->(for=Functor, type=List) = {
    map(value List(A), f (A)->B)->(List(B)) = {
        result = List(B){}
        value.each(_, item).do({ result.append(f(item)) });
        this.return result
    }
}
```

### Applicative

```folang
@co.dap.typeclass(kind=Applicative)
Applicative(F) = {
    pure(x A) -> (F(A));
    apply(fab F(A->B), fa F(A)) -> (F(B));
}

OptionApplicative co.lang.instance->(for=Applicative, type=Option) = {
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
@co.dap.typeclass(kind=Monad)
Monad(F) = {
    pure(x A) -> (F(A));
    flatMap(fa F(A), f (A)->F(B)) -> (F(B));
}

OptionMonad co.lang.instance->(for=Monad, type=Option) = {
    pure(x A)->(Option(A)) = { this.return Some(x); }
    flatMap(fa Option(A), f (A)->Option(B))->(Option(B)) = {
        this.return fa.match().case(Some(x) => f(x)).default(None);
    }
}
```

### Monoid

```folang
@co.dap.typeclass(kind=Monoid)
Monoid(T) = {
    empty() -> (T);
    combine(a T, b T) -> (T);
}

IntMonoid co.lang.instance->(for=Monoid, type=co.lang.int) = {
    empty()->(co.lang.int) = { this.return 0; }
    combine(a co.lang.int, b co.lang.int)->(co.lang.int) = { this.return a + b; }
}
```

### Transformer

```folang
@co.dap.typeclass(kind=Transformer)
Transformer(F(_), G(_)) = {
    map(value F(A), f (A)->B) -> (G(B));
}

ListToSetTransformer co.lang.instance->(for=Transformer, types=[List, Set]) = {
    map(value List(A), f (A)->B)->(Set(B)) = {
        result = Set(B){}
        value.each(_, item).do({ result.insert(f(item)) });
        this.return result;
    }
}
```

---

## Let Bindings

```folang
y co.lang.int = let({x = 10}).in({x + 1});
y co.lang.int = let({$ = 10}).in({$ + 1});  // $ refers to the value being defined

x co.lang.int = (x + 1).where(x = 10);
x co.lang.int = ($ + 1).where($ = 10);

let fib(0) = 1
let fib(1) = 1
let fib(n) = fib(n-1) + fib(n-2)
```

> `$` is a special identifier usable in `let` bindings for recursive or self-referential expressions.

---

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
k := (1..10).filter(|x| => x % 2 == 0).map(|x| => x * x);

result := for (x <- List(1,2,3)).yield(x * 2)         // List(2, 4, 6)
result := for (x <- Set(1,2,3)).yield(x * 2)           // Set(2, 4, 6)
result := for (x <- Some(5)).yield(x * 2)              // Some(10)
result := for (x <- fetchData()).yield(x.process())    // Future

ages  := {"A":30,"B":40,"c":66,"e":88};
upper := for ((name, age) <- ages).yield(name.toUpperCase, age)
```

---

## Extensions

```folang
@co.dap.extension(fortype=co.lang.string, what=extends)
upperCase()->(string)={
    return this.upper()
}

@co.dap.extension(fortype=[co.lang.string], what=overrides)
equals(str string)->(bool)={
    this.return this == str
}
```

Extensions must be **explicitly activated** — they are block-scoped:

```folang
@co.dap.use(extensions=[equals, upperCase])
k.upperCase();  // ✅ explicitly activated
```

---

## Labels and Named Blocks

```folang
// Label
outer:{
    // statements
}

// Named Block
labelBlock co.lang.block={

}

labelBlock.inline();
```

---

## Reflection

```folang
@co.dap.reflection(enable=True, package="co.meta")

x co.lang.int = 10;
x.reflect().getType()  → co.lang.int
x.reflect().getValue() → 10
x.reflect().getKind()  → value
```

---

## Application Libraries

### 1. Structs and Free Functions

```folang
@co.dap.library
EmpPackage co.lang.package={

    @co.dap.export
    SEmployee co.lang.signature={
        Employee co.lang.struct;
        storeEmployee(emp Employee)->(Employee);
    }

    Employee co.lang.struct={
        empId   co.lang.int;
        empName co.lang.string;
    }

    storeEmployee(emp Employee)->(Employee)={
        e = Employee();
        this.return e;
    }
}

or 

EmpPackage co.lang.package->(type=library)={

    @co.dap.export
    SEmployee co.lang.signature={
        Employee co.lang.struct;
        storeEmployee(emp Employee)->(Employee);
    }

    Employee co.lang.struct={
        empId   co.lang.int;
        empName co.lang.string;
    }

    storeEmployee(emp Employee)->(Employee)={
        e = Employee();
        this.return e;
    }
}
```

### 2. Classes

```folang
@co.dap.library
EmpPackage co.lang.package={

    @co.dap.export
    IEmployee co.lang.interface={
        storeEmployee(emp Employee)->(Employee);
    }

    @co.dap.oops(Implements:[IEmployee])
    Employee co.lang.class->(implements=[IEmployee])={
        empId   co.lang.int;
        empName co.lang.string;

        storeEmployee(emp Employee)->(Employee)={
            e = Employee();
            this.return e;
        }
    }
}
```

### 3. Modules

```folang
@co.dap.library
EmpPackage co.lang.package={

    @co.dap.export
    MEmployee co.lang.signature={
        Employee struct;
        storeEmployee(emp Employee)->(Employee);
    }

    @co.dap.module(signature=MEmployee)
    EmployeeImpl co.lang.module->(signature=MEmployee, matches=MEmployee) = {
        Employee co.lang.struct = {
            empId   co.lang.int;
            empName co.lang.string;
        }

        storeEmployee(emp Employee)->(Employee)={
            e = Employee();
            this.return e;
        }
    }
}
```

---

## Forward / Extern Declarations

#### Variable

```folang
@co.dap.extern
someBool co.lang.bool;
```

#### Functions

```folang
@co.dap.extern
getEmployee(id co.lang.int)->(somepack.Employee);

// or — @co.dap.extern is optional for functions
getEmployee(id co.lang.int)->(somepack.Employee);
```

#### Types

```folang
@co.dap.extern
Employee co.lang.struct;

// or — @co.dap.extern is optional for types
Employee co.lang.struct;
```

> For functions and types `@co.dap.extern` is optional. For variables it is required.

---

## Interface vs Signature

```folang
MEmployee co.lang.signature = {
    Employee co.lang.struct;
    storeEmployee(emp Employee) -> (Employee);
}

IEmployee co.lang.interface = {
    storeEmployee(emp Employee) -> (Employee);
}
```

Structurally they look similar — both are lists of contracts. The difference is **who implements them and how**.

| | `co.lang.signature` | `co.lang.interface` |
|---|---|---|
| Implemented by | module via `matches=` | class via `implements=[]` |
| Can include types/structs | ✅ | ❌ |
| Instantiation involved | ❌ | ✅ |
| OOP dispatch | ❌ | ✅ virtual/dynamic |
| Behavior only | ❌ | ✅ (like Go) |
| Origin | ML/OCaml modules | Java/C#/Go interfaces |

- A `signature` is a **structural contract** — can include types, structs, nested definitions. Describes a whole capability unit.
- An `interface` is a **behavioral contract** — methods only, no fields, no type declarations. Tied to OOP dispatch and polymorphism.

---

## Structs vs Classes vs Modules

| | Struct | Class | Module |
|---|---|---|---|
| **Purpose** | Pure data shape | Behavior + data | Named function bundle |
| **Fields** | ✅ | ✅ per instance | ❌ |
| **Methods** | ❌ | ✅ | ✅ free functions |
| **Lifecycle** (`@@new`/`@@init`) | ❌ | ✅ | ❌ |
| **`this` / `self`** | ❌ | ✅ | ❌ |
| **Instantiable** | ❌ | ✅ multiple objects | ❌ single impl |
| **Implements** | ❌ | `interface` via `implements=[]` | `signature` via `matches=` |
| **OOP / inheritance** | ❌ | ✅ | ❌ |
| **Contains type declarations** | ❌ | ✅ inner, accessible as `Class.Type` | ✅ via signature |
| **Pattern matching** | ✅ | ✅ | ❌ |
| **Declared with** | `co.lang.struct` | `co.lang.class` | `co.lang.module` |
| **Contract type** | — | `co.lang.interface` | `co.lang.signature` |
| **Closest analogy** | C struct / Rust struct | Java / C# class | ML module / F# module |

**Mental model:** Reach for a struct when you only need data. Reach for a class when you need behavior, lifecycle, or multiple independent instances. Reach for a module when you need a named bundle of functions and types with no instantiation.

> **Type declaration scoping rule:** Modules own types as part of their public contract via signature. Classes own inner types accessible via `ClassName.TypeName`. Functions may declare types locally — local types are scoped to the function body only and cannot appear in parameter or return types. Structs cannot declare types — they are pure data with no scope.

---

## Built-in Packages

### `co` — root (reserved word)

The only package provided by default.

| Sub-package | Responsibility |
|---|---|
| `co.lang` | All data types and kinds |
| `co.sys` | file, concurrent, parallel, goto, invoke, bind, call, apply, settimeout, setinterval, scheduler, cron, event |
| `co.os` | signal, cmd, execute, run, env, getenv, setenv, sleep, exit, cwd, chdir, fork, wait, pipe, dup, close, readfd, writefd |
| `co.meta` | patch, instrument, ast, reflect, introspect, transform, inject, create, augment, runtime (eval) |
| `co.core` | list, set, map, tree, tries, sort, search, matrix |
| `co.native` | load, register, asm, inline, emit, ffi |
| `co.in` | read, readln |
| `co.out` | println, print |
| `co.regex` | stex, pattern, match, search |
| `co.crypto` | rsa, aes, hash, md5, rand, uuid, ssl, tls |
| `co.dap` | built-in directives, decorators, annotations, pragmas |
| `co.ddap` | built-in directives, decorators, annotations, pragmas |
| `co.net` | tcp, udp, http |
| `co.const` | `true`, `false`, `none` |
| `co.encoding` | base64Encode, base64Decode, json, yml, bson |
