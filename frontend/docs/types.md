Type : Type 1 : Type 2 : Type 3 : ...
```

**Why the infinite levels exist — Girard's Paradox:**
```
// If Type : Type (type is its own type)
// You can encode Russell's Paradox
// The language becomes logically inconsistent
// Proofs become meaningless

// Solution: infinite hierarchy
// No level contains itself
// Type₀ : Type₁ (not Type₀ : Type₀)
```

**Visual map:**
```
Level   Name              Example              Who cares
──────────────────────────────────────────────────────────
  0     Value             42                   Everyone
  1     Type              Int                  Everyone
  2     Kind              * → *                Haskell, Scala
  3     Sort              □                    GHC internals
  4     Superkind         △                    Type theorists
  5     Universe₅         Type₅                Coq, Agda, Lean
  6     Universe₆         Type₆                Coq, Agda, Lean
  ...   ...               ...                  ...
  ω     Universeω         Typeω                Agda (cumulative)
  ∞     Infinite tower    never ends           Math
```

**Practical reality:**
```
Level 0-1:  Every language           (values, types)
Level 2:    Haskell, Scala, your folang  (kinds)
Level 3:    GHC internals, some Scala    (sorts)
Level 4+:   Only proof assistants        (Coq, Agda, Lean, Idris)
```

**For your folang, you already cover 0-2:**
```
// Level 0: values
x co.lang.int = 42;
x.type()  → co.lang.int

// Level 1: types
Employee co.lang.struct = { ... }
x.type()  → Employee

// Level 2: kinds
x co.lang.data = 42;
x.kind()  → data

Option(T) co.lang.type = Some(T) | None();
Option.kind() → type → type


The Lambda Cube is a framework that organizes type systems by **three axes of expressiveness**. It was formalized by Henk Barendregt.

**Start with the simplest system — Simply Typed Lambda Calculus (λ→):**
```
Values depend on values:  42 + 1 = 43
That's it. Nothing fancy.
```

**The three axes — each adds one new power:**

**Axis 1: Polymorphism (terms depend on types)**
```
// "Give me a type, I'll give you a value"

// Without: write separate functions
identityInt(x int) → int
identityStr(x string) → string

// With: one function works for all types
identity(T)(x T) → T

// This is generics / parametric polymorphism
// System F, Java generics, your @co.dap.generic
```

**Axis 2: Type operators (types depend on types)**
```
// "Give me a type, I'll give you a type"

List(Int)     → List of Int       // type → type
Map(String, Int) → Map            // type → type → type
Option(T)     → Some(T) | None    // type constructor

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
divide(a int, b NonZero(int)) → int
// compiler PROVES b can't be zero
```

**The Cube — 8 corners from combining these three axes:**

```
                    Dependent Types
                    (types ← values)
                        │
                        │
            ┌───────────┼───────────┐
           λP          λPω         λP2
           │╱          ╱│          ╱│
          ╱ │        ╱  │        ╱  │
        ╱   │      ╱    │      ╱    │
      λω────┼────λPω    │    ╱     │
       │    │     │      │  ╱       │
       │   λ→────┼──────λ2         │
       │  ╱      │     ╱           │
       │╱        │   ╱             │
      λω─────────┼─╱              │
                 λ→───────────────λ2
                        │
                 Type Operators    Polymorphism
                 (types ← types)  (terms ← types)
```

**Each corner is a type system:**

```
Corner    Powers                          Real language
──────────────────────────────────────────────────────
λ→        none                            Simply typed (basic)
λ2        polymorphism                    System F (Haskell basics)
λω        type operators                  System Fω (kinds)
λP        dependent types                 LF
λ2ω       poly + type ops                 System Fω (Haskell full)
λP2       poly + dependent                (rare)
λPω       dependent + type ops            (rare)
λPω2      ALL THREE                       Calculus of Constructions
```

**The corners mapped to real languages:**

```
λ→     Simply Typed
       → C, Go, early Java
       "Types exist but are simple"

λ2     + Polymorphism
       → Java generics, C# generics, your folang generics
       "Functions can be generic over types"

λω     + Type Operators
       → Haskell, Scala, your folang kinds
       "Types can take type parameters"

λ2ω    + Both (poly + type ops)
       → Full Haskell, full Scala
       "Generic types that take generic types"
       // Functor, Monad — your folang has these!

λP     + Dependent Types
       → Idris, Agda, Coq, Lean
       "Types can depend on runtime values"

λPω2   ALL THREE — Calculus of Constructions
       → Coq's core, Lean's core
       "Everything depends on everything"
       "Types are values, values are types"
```

**Where your folang sits:**

```
λ2ω — Polymorphism + Type Operators

You have:
✅ Polymorphism:  @co.dap.generic(type={T: typename})
✅ Type Operators: Option(T) co.lang.type = Some(T) | None()
✅ Kinds:         type → type
✅ Functors:      @co.dap.Functor
✅ Monads:        @co.dap.monad

You don't have:
❌ Dependent types: Vector(3) — type depends on value 3
```

**The "Grand Unification" means:**

All possible type systems are just **combinations of these three axes**. Every language's type system is a point on this cube. You don't need to invent new theory — just pick which axes you want.

```
Want generics?        → turn on Axis 1
Want higher-kinded?   → turn on Axis 2
Want dependent types? → turn on Axis 3
Want everything?      → Calculus of Constructions
```

Your folang chose axes 1 + 2 (λ2ω). The sweet spot — expressive enough for monads and functors, practical enough for real software.


