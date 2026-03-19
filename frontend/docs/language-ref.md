# FoLang Language Reference

A practical guide for developers writing FoLang programs.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Variables and Types](#variables-and-types)
3. [Functions](#functions)
4. [Data Structures](#data-structures)
5. [Packages and Imports](#packages-and-imports)
6. [Control Flow](#control-flow)
7. [Pattern Matching](#pattern-matching)
8. [Generics](#generics)
9. [Zones](#zones)
10. [Collections](#collections)
11. [Error Reference](#error-reference)

---

## Quick Start

### Hello World

```folang
// hello.fol — entry file, no package needed
@co.ddap.import(path="co.out", package="out", as="out")

co.out.println("Hello FoLang!")
```

### Variables

```folang
// typed
name co.lang.string = "Rao";
age  co.lang.int    = 30;

// inferred
name := "Rao";
age  := 30;

// assign if not defined, otherwise reassign
name ?= "Kumar";
```

### Simple Function

```folang
myPackage co.lang.package = {

    greet(name co.lang.string)->() = {
        co.out.println("Hello " + name);
    }

}
```

### Running

```bash
folang hello.fol -> hello binary
./hello
```

---

## Variables and Types

### Declaration

```folang
// with type
someVar    co.lang.int;
someString co.lang.string;

// with type and value
someBool co.lang.bool = co.const.true;
someInt  co.lang.int  = 42;

// inferred from value — := errors if already declared
someVal := "Hello World";
someNum := 3.14;

// inferred — declare if not exists, reassign if exists
someR ?= "Rao";
```

### Built-in Types

| Type | Description | Example |
|---|---|---|
| `co.lang.int` | Integer | `42` |
| `co.lang.float` | Floating point | `3.14` |
| `co.lang.bool` | Boolean | `co.const.true` |
| `co.lang.string` | String | `"hello"` |
| `co.lang.char` | Character | `'a'` |
| `co.lang.byte` | Byte | `0xFF` |
| `co.lang.auto` | Inferred at compile time | `co.lang.auto = "hello"` |
| `co.lang.dynamic` | Dynamic typing | `co.lang.dynamic` |

### Arrays

```folang
// fixed size
arr co.lang.int->([5]) = [1, 2, 3, 4, 5];

// 2D array
grid co.lang.int->([2, 3]) = [[1,2,3],[4,5,6]];

// dynamic size
dynArr co.lang.int->([...]);

// initialize with inferred size
nums co.lang.int->([]) = [1, 2, 3];
```

### Ranges

```folang
rangeI := 1..10;      // [1, 10]  inclusive both ends
rangeS := 0<..5;      // (0, 5]   exclude start
rangeL := 0..<100;    // [0, 100) exclude end
rangeB := 0<..<100;   // (0, 100) exclude both
```

### Type Declarations

```folang
// type alias
MyInt co.lang.type = co.lang.int;

// new type — not interchangeable with base type
UserId co.lang.newtype = co.lang.int;

// ADT — sum type
Status co.lang.type = Active | Inactive | Pending;

// opaque type — internal representation hidden
Token co.lang.opaquetype = co.lang.string;
```

---

## Functions

### Basic Function

```folang
myPackage co.lang.package = {

    // simple function
    add(a co.lang.int, b co.lang.int)->(co.lang.int) = {
        this.return a + b;
    }

    // no return value
    greet(name co.lang.string)->() = {
        co.out.println("Hello " + name);
    }

}
```

### Multiple Return Values

```folang
divide(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.bool) = {
    (b == 0).return(0, co.const.false)
             .otherwise.return(a / b, co.const.true);
}

result, ok := divide(10, 2);
```

### Default Parameters

```folang
greet(name co.lang.string, greeting co.lang.string = "Hello")->() = {
    co.out.println(greeting + " " + name);
}

greet("Rao")            // Hello Rao
greet("Rao", "Hi")      // Hi Rao
```

### Optional Parameters

```folang
greet(name co.lang.string, title? co.lang.string)->() = {
    (title.omitted).do({
        co.out.println("Hello " + name);
    }).otherwise.do({
        co.out.println("Hello " + title + " " + name);
    });
}

greet("Rao")           // Hello Rao
greet("Rao", "Dr")     // Hello Dr Rao
```

### Named Parameters

```folang
connect(~host co.lang.string, ~port co.lang.int)->() = { }

connect(host="localhost", port=8080)
connect(port=8080, host="localhost")   // order does not matter
```

### Variadic Functions

```folang
sum(...nums co.lang.int)->(co.lang.int) = {
    total co.lang.int = 0;
    nums.each(_, n).do({ total += n; });
    this.return total;
}

sum(1, 2, 3, 4, 5)   // 15
```

### Curried Functions

```folang
add(a co.lang.int)(b co.lang.int)->(co.lang.int) = {
    a + b
}

add5 := add(5);   // partial application
add5(3)           // 8
add5(10)          // 15
```

### Closures

```folang
counter()->( ()->(co.lang.int) ) = {
    count co.lang.int = 0;
    this.return ()->(co.lang.int) = {
        count += 1;
        this.return count;
    }
}

next := counter();
next()   // 1
next()   // 2
next()   // 3
```

### Lambdas

Lambdas are only allowed as inline arguments to collection operations:

```folang
nums  := [1, 2, 3, 4, 5];
evens := nums.filter(|x| => x % 2 == 0);   // [2, 4]
squared := nums.map(|x| => x * x);          // [1, 4, 9, 16, 25]
total   := nums.reduce(|acc, x| => acc + x, 0); // 15
```

### Associated Functions

Associated functions are attached to a type:

```folang
Employee co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}

// associated function — (emp Employee) is the receiver
(emp Employee) greet()->() = {
    co.out.println("Hello " + emp.name);
}

// usage
e := Employee{ id: 1, name: "Rao" };
e.greet()   // Hello Rao
```

---

## Data Structures

### Struct — Pure Data

```folang
myPackage co.lang.package = {

    Employee co.lang.struct = {
        id      co.lang.int;
        name    co.lang.string;
        salary  co.lang.float;
    }

    // create
    emp := Employee{ id: 1, name: "Rao", salary: 50000.0 };

    // access
    co.out.println(emp.name);

}
```

Structs are passed by reference. They cannot have methods — use associated functions instead.

### CStruct — C-Like Value Type

Use `co.lang.cstruct` when you need value semantics or need to cross zone boundaries:

```folang
Point co.lang.cstruct = {
    x co.lang.int;
    y co.lang.int;
}

// always passed by value — copied on pass
p := Point{ x: 10, y: 20 };
```

### Enum

```folang
Status co.lang.enum = {
    Active,
    Inactive,
    Pending
}

s := Status.Active;

s.match(co.pattern.Instance)
    .case(Status.Active   => co.out.println("active"))
    .case(Status.Inactive => co.out.println("inactive"))
    .default(co.out.println("pending"));
```

### Class — Behavior + Data

```folang
myPackage co.lang.package = {

    Employee co.lang.class = {

        id   co.lang.int;
        name co.lang.string;

        @co.dap.constructor(access=public)
        @@init(id co.lang.int, name co.lang.string) = {
            this.id   = id;
            this.name = name;
        }

        @co.dap.method.instance
        greet()->() = {
            co.out.println("Hello " + this.name);
        }

        @co.dap.method.static
        create(id co.lang.int, name co.lang.string)->(Employee) = {
            this.return Employee.@@init(id, name);
        }
    }

    emp := Employee.create(1, "Rao");
    emp.greet();   // Hello Rao

}
```

### Module — Named Function Bundle

```folang
myPackage co.lang.package = {

    // signature — contract
    EmployeeSig co.lang.signature = {
        Employee co.lang.struct;
        getEmployee(id co.lang.int)->(Employee);
        storeEmployee(emp Employee)->(co.lang.bool);
    }

    // implementation
    @co.dap.module(signature=EmployeeSig)
    EmployeeModule co.lang.module->(matches=EmployeeSig) = {

        Employee co.lang.struct = {
            id   co.lang.int;
            name co.lang.string;
        }

        getEmployee(id co.lang.int)->(Employee) = {
            this.return Employee{ id: id, name: "Rao" };
        }

        storeEmployee(emp Employee)->(co.lang.bool) = {
            this.return co.const.true;
        }
    }

    // use module via signature — first class
    em EmployeeSig = EmployeeModule;
    emp := em.getEmployee(1);

}
```

### Struct Embedding

```folang
Base co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}

// embed Base — id and name promoted to Employee
Employee co.lang.struct = {
    Base;                    // embedded
    salary co.lang.float;
}

emp := Employee{ id: 1, name: "Rao", salary: 50000.0 };
emp.id     // direct access — no emp.Base.id needed
emp.name   // direct access
emp.salary // direct access
```

---

## Packages and Imports

### Creating a Package

```folang
// employee.fol
myPackage co.lang.package = {

    Employee co.lang.struct = {
        id   co.lang.int;
        name co.lang.string;
    }

    getEmployee(id co.lang.int)->(Employee) = {
        this.return Employee{ id: id, name: "Rao" };
    }

}
```

### Importing a Package

```folang
// app.fol — entry file
@co.ddap.import(path="myapp.employee", package="myPackage", realm="app", as="emp")

e := emp.getEmployee(1);
co.out.println(e.name);
```

### Import Rules

```
path   →  dot separated logical path — no OS separators
as     →  mandatory alias — valid identifier, no dots
realm  →  default is app
```

### Aggregation — Multiple Packages Under One Alias

```folang
// combine two packages under one alias
@co.ddap.import(path="myapp.empA", package="a", as="hr")
@co.ddap.import(path="myapp.empB", package="b", as="hr")

// access both via hr alias
hr.Employee     // from package a
hr.Attendance   // from package b
```

### Version Coexistence

```folang
// two versions of same package
@co.ddap.import(path="libs.empv1", package="emp", realm="app", as="hr")
@co.ddap.import(path="libs.empv2", package="emp", realm="x",   as="v2_hr")

hr.Employee     // v1
v2_hr.Employee  // v2
```

### Shadowing — New Version Overrides Old

```folang
@co.ddap.import(path="libs.empv1", package="emp", realm="app", as="hr")
@co.ddap.import(path="libs.empv2", package="emp", realm="x", parent-realm="app", as="hr")
// realm x shadows app — v2 overrides v1
// access via hr — gets v2
```

### Package Nesting

Declare parent relationship inside child package source:

```folang
// child.fol
childPackage co.lang.package = {

    @co.ddap.parent(path="myapp.parent", package="parentPackage")

    // parent's types available directly — no prefix needed
    x ParentType = ParentType{ id: 1 };
}
```

---

## Control Flow

### Conditions

```folang
x co.lang.int = 10;

// if
(x > 5).do({
    co.out.println("greater");
});

// if-else
(x > 5).do({
    co.out.println("greater");
}).otherwise.do({
    co.out.println("not greater");
});

// if-else if-else
(x > 10).do({
    co.out.println("GT");
}).otherwise(x == 10).do({
    co.out.println("EQ");
}).otherwise.do({
    co.out.println("LT");
});
```

### Ternary

```folang
result := (x > 5).return("big").otherwise.return("small");
```

### Loops

```folang
x co.lang.int = 0;

// while loop
(x < 10).loop({
    x += 1;
});

// do-while equivalent
co.out.println(x);
(x < 10).loop({
    x += 1;
    co.out.println(x);
});
```

### Iterating Arrays

```folang
arr co.lang.int->([5]) = [1, 2, 3, 4, 5];

// with index
arr.each(idx, val).do({
    co.out.print(idx);
    co.out.print(": ");
    co.out.println(val);
});

// without index
arr.each(_, val).do({
    co.out.println(val);
});
```

### Contains

```folang
arr co.lang.int->([5]) = [1, 2, 3, 4, 5];
k   co.lang.int        = 3;

arr.contains(k).do({
    co.out.println("found");
}).otherwise.do({
    co.out.println("not found");
});
```

---

## Pattern Matching

### Value Matching

```folang
x co.lang.int = 10;

x.match(co.pattern.Value)
    .case(10 => co.out.println("ten"))
    .case(20 => co.out.println("twenty"))
    .default(co.out.println("other"));
```

### Type Matching

```folang
x co.lang.dynamic = 10;

x.match(co.pattern.Type)
    .case(co.lang.int    => co.out.println("int"))
    .case(co.lang.string => co.out.println("string"))
    .default(co.out.println("other"));
```

### Shape Matching

```folang
Point co.lang.struct = { x co.lang.int; y co.lang.int; }

p := Point{ x: 10, y: 20 };

p.match(co.pattern.Shape)
    .case(Point{x, y} => co.out.println(x + " " + y))
    .default(co.out.println("unknown"));
```

### ADT Matching

```folang
Status co.lang.type = Active | Inactive | Pending;
s := Active;

s.match(co.pattern.Instance)
    .case(Active   => co.out.println("active"))
    .case(Inactive => co.out.println("inactive"))
    .default(co.out.println("pending"));
```

### Function Pattern

```folang
Option co.lang.type = Some(co.lang.int) | None();

f(Some(x)) => { x + 1 }
f(None())  => { 0 }
```

---

## Generics

### Generic Function

```folang
@co.dap.generic(type={T:{variance:invariant}})
identity(x T)->(T) = {
    this.return x;
}

identity(42)       // T = int
identity("hello")  // T = string
```

### Generic Struct

```folang
@co.dap.generic(typename=T)
Box co.lang.struct = {
    value T;
}

intBox    := Box{ value: 42 };
stringBox := Box{ value: "hello" };
```

### Generic Class

```folang
@co.dap.generic(type={T:{typename}})
Stack co.lang.class = {
    items co.lang.core.list->(T);

    @co.dap.method.instance
    push(item T)->() = {
        this.items.append(item);
    }

    @co.dap.method.instance
    pop()->(T) = {
        this.return this.items.removeLast();
    }
}

s := Stack.new(co.lang.int);
s.push(1);
s.push(2);
s.pop();   // 2
```

### Bounded Generics

```folang
@co.dap.generic(type={T:{variance:invariant, bound=Number}})
sum(a T, b T)->(T) = {
    this.return a + b;
}

sum(1, 2)       // ✅ int is Number
sum(1.0, 2.0)   // ✅ float is Number
sum("a", "b")   // ❌ compiler error — string is not Number
```

---

## Zones

Zones control what capabilities a package can use.

### Application Zone — Default

All packages are application zone by default. Full language features available:

```folang
hrPackage co.lang.package = {
    // all features available
    Employee co.lang.class = { ... }
    getEmployee(id co.lang.int)->(Employee) = { ... }
}
```

### Systems Zone — Low Level Code

Use systems zone when you need raw pointers, MMIO, or hardware access:

```folang
driversPackage co.lang.package = {
    @co.dap.zone(level=systems)

    // pointers allowed inside
    @co.dap.private
    gpio co.lang.word->(*);

    // public interface — simple types only
    @co.dap.public
    init()->(co.lang.bool) = { ... }

    @co.dap.public
    readSensor(id co.lang.int)->(co.lang.float) = { ... }
}
```

> Systems zone requires the `systems` feature to be enabled at install time.

### FFI Zone — C Bindings

Use ffi zone when binding to C libraries:

```folang
bindingsPackage co.lang.package = {
    @co.dap.zone(level=ffi)

    @co.dap.native
    cFunction(id co.lang.int)->(co.lang.int) = { }
}
```

> FFI zone requires the `ffi` feature to be enabled at install time.

### Crossing Zones

Application code can call into systems/ffi code through the public interface:

```folang
// application calling systems
@co.ddap.import(path="drivers", package="driversPackage", as="drv")

appPackage co.lang.package = {
    setup()->() = {
        drv.init();                // ✅ public function
        drv.readSensor(1);         // ✅ public function
    }
}
```

Systems packages cannot call back into application packages:

```folang
@co.dap.zone(level=systems)
driversPackage co.lang.package = {
    @co.ddap.import(path="hr", package="hrPackage", as="hr")  // ❌ compiler error
}
```

---

## Collections

FoLang provides built-in collection operations via `co.core`:

### List

```folang
@co.ddap.import(path="co.core.list", package="list", as="lst")

nums := lst.list.of(1, 2, 3, 4, 5);

// map
squared := nums.map(|x| => x * x);

// filter
evens := nums.filter(|x| => x % 2 == 0);

// reduce
total := nums.reduce(|acc, x| => acc + x, 0);

// sort
sorted    := nums.sortBy(|x| => x);
sortedDesc := nums.sortWith(|a, b| => a > b);
```

### Map

```folang
@co.ddap.import(path="co.core.map", package="map", as="m")

ages := m.map.of("Rao", 30, "Kumar", 25);

ages.each(name, age).do({
    co.out.println(name + ": " + age);
});
```

### Set

```folang
@co.ddap.import(path="co.core.set", package="set", as="s")

nums := s.set.of(1, 2, 3, 4, 5);
nums.contains(3).do({ co.out.println("found"); });
```

---

## Error Reference

### Common Errors

**Cyclic import:**
```
ERROR: cyclic import detected among packages: packageA, packageB
FIX:   remove circular dependency between packages
```

**Symbol conflict in aggregation:**
```
ERROR: Employee found in both package a and package b under alias hr
FIX:   use different as for each package
       @co.ddap.import(path="...", package="a", as="hrA")
       @co.ddap.import(path="...", package="b", as="hrB")
```

**Zone violation:**
```
ERROR: zone=systems package cannot import zone=application package
FIX:   systems packages can only import other systems or ffi packages
```

**Free code in package:**
```
ERROR: free code not allowed inside package declaration
FIX:   move statements to entry file
       move declarations to package
```

**Type declaration in entry file:**
```
ERROR: type declarations not allowed in entry file
FIX:   move struct/class/module/enum to a package file
```

**Multiple entry files:**
```
ERROR: multiple entry files found — only one allowed per application
FIX:   ensure only one .fol file has no package declaration
```

**Struct in public interface of systems/ffi package:**
```
ERROR: co.lang.struct not allowed in public interface of systems/ffi package
FIX:   use co.lang.cstruct for zone boundary crossing
```

**Pointer in public interface:**
```
ERROR: pointer type not allowed in public interface of systems/ffi package
FIX:   wrap in simple types or co.lang.cstruct
```

**Same path + same package + different alias:**
```
ERROR: same path and package cannot have multiple aliases
FIX:   use one alias per path+package combination
```

**Floating annotation:**
```
ERROR: annotation must be immediately above its target
FIX:   place annotation directly above the declaration it applies to
```

**Import after code:**
```
ERROR: import directives must appear before any declarations
FIX:   move all @co.ddap.import to top of package body
```
