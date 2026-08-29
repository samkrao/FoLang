# Preface

## Why This Book Exists

A programming language is more than syntax; it is the lens through which we view
a system's architecture. It shapes how we define boundaries, express state, and
whether complexity feels manageable or overwhelming.

FoLang was designed with that in mind.

The language reference gives you the normative definition — syntax, semantics,
type rules, source structure, and capability boundaries. This book asks a
different question:

> **How do we think when designing a solution in FoLang?**

*Think Like FoLang* is a guide to building the mental model that connects
high-level architectural expression to low-level certainty. It is about looking
at a problem and seeing how it wants to be represented.

## The FoLang Mental Model

FoLang starts from a uniform premise:

> **Everything in FoLang is an object.**

That consistency means whether you are composing generic abstractions,
leveraging pattern matching, or managing mutation, you are working within a
single, cohesive model. Not every feature is forced into one implementation
strategy — `co.lang.cstruct`, for instance, is an ABI-oriented value-semantic
representation — but the reasoning stays unbroken from top to bottom.

This book will help you work out:

- When a type should carry domain meaning rather than just structural shape.
- Where behavior belongs, and which abstraction expresses the domain directly.
- How to use reference semantics, immutability, sharing, and copy-on-write
  cleanly.
- When pattern matching expresses intent better than branching.
- The boundary between value equality and reference identity.
- When an ordinary mechanism is enough, and when an advanced one is warranted.

## Who This Book Is For

This book assumes you have written programs in at least one other language and
are comfortable with types, functions, and objects. It does not assume you have
used FoLang, and it does not assume a particular background language.

Where FoLang differs from what you may be used to, the book says so directly.

## Specification and Book

Where exact syntax or semantics matter, the language reference governs. The
examples here follow the current reference.

```text
language-ref.md
    -> normative definition
    -> compiler-facing precision
    -> complete structural rules

Think Like FoLang
    -> developer-facing explanation
    -> mental models
    -> examples and idioms
    -> problem-solving
```

When a feature appears here, the goal is to explain why it exists, how to reason
about it, and when it is appropriate. The reference remains the place to check
the complete rule.

> **A note on stability.** FoLang is still evolving toward version 1.0. Syntax
> and semantics may be refined during that period, and examples in this book may
> be updated to match.

## How This Book Is Organized

The book builds up in layers.

**Part I — Learning to Think Like FoLang** establishes the core model: objects,
bindings, references, equality, and how expressions and matching fit together.

**Part II — Building with FoLang** develops the main abstractions: functions,
units, structs, classes, types, generics, interfaces, signatures, type classes,
extensions, and collections.

**Part III — Designing FoLang Applications** moves from single declarations to
systems: packages, imports, project layout, applications, libraries, components,
metadata, and execution models.

**Part IV — Advanced FoLang** covers mechanisms for cases where the ordinary
abstractions are not enough: refinement and dependent types, macros, operators,
reflection, specialization, native facilities, and longer case studies.

You do not need all of this before writing useful FoLang. The early chapters
start with a small model and add to it when there is a reason to.

## How to Read the Examples

FoLang uses explicit statement termination. A simple statement ends with `;`,
while a block or declaration body ends with its closing `}`.

```folang
name co.lang.string = "Rao";
age  co.lang.int = 30;
```

```folang
(co.const.true).then({
    co.out.println("ready");
}).default({
    co.out.println("not ready");
});
```

Examples favor complete, readable forms first. Shorter forms and advanced
facilities appear once their meaning is clear.

Where a FoLang construct resembles one from another language, the book uses
FoLang's own vocabulary, since similar syntax does not always mean similar
semantics.

## What "Thinking Like FoLang" Means

Imagine FoLang itself looking at your system. What would it notice first? Which
distinctions would it keep visible?

Consider a small example. Two Employee objects hold the same values:

```folang
a Employee = Employee{name: "Rao", id: 1};
b Employee = Employee{name: "Rao", id: 1};

a == b;            // true  — equal values
a.sameRef(b);      // false — different objects
```

Why does FoLang distinguish these two questions?

Imagine walking into a shop to buy a 100 g jar of coffee beans. On the shelf are
ten sealed jars of the same product. They contain the same quantity and carry
the same relevant product information.

From the point of view of those values, the jars are equal.

Now you pick up one jar and another customer picks up another.

Are you holding the **same jar**?

No.

The two jars may be equal in the relevant values we are comparing, but they are still two distinct physical
objects.

FoLang thinks about managed objects in the same way:

```text
equality
    -> do these objects have equal values?

identity
    -> do these references point to the same object?
```

That is why these are deliberately different questions:

```folang
a == b;            // true  — equal values
a.sameRef(b);      // false — different objects
```

If you open your jar and pour out half the beans, the other customer's jar remains full. The two jars were equal in value, but mutating one does not mutate the other because they are distinct objects.


Now bind a third name:

```folang
c := a;

c == a;            // true
c.sameRef(a);      // true  — c and a refer to the same object
```

Now imagine two people visiting a clock shop. They both notice a clock hanging on the wall. Each points to it and asks the salesperson about it. If we ask the salesperson whether both people are pointing/referring to the same clock, the answer is yes. There are two people making the reference, but both references point to one physical clock. FoLang thinks about `c := a` in the same way.

Here `c := a` does not create another Employee object. For managed objects, it
binds `c` to the object already referenced by `a`.

If the shopkeeper adjusts the time on the clock, both people see the new time. They are not observing separate clocks; they are observing the state of the same physical clock.

Conceptually:

```text
a ─────┐
       ├────> Employee{name: "Rao", id: 1}
c ─────┘

b ──────────> Employee{name: "Rao", id: 1}
```

So `a`, `b`, and `c` can all be equal in value, while only `a` and `c` refer to
the same object.

This distinction is central to thinking like FoLang:

- **equal** describes value;
- **the same object** describes identity;
- a binding is not the object itself.

The same habit applies elsewhere. `Employee{name: "Rao", id: 1}` is not only
construction syntax; it explicitly states which type of object is being
created. When you stop reading only the syntax and start aligning the domain
model with the language's abstractions, you are beginning to think like FoLang.

```text
Coffee jars
    -> two distinct objects
    -> initially equal values
    -> mutate one
    -> the other is unaffected
    -> sameRef(...) = false

People pointing to one clock
    -> two references
    -> one object
    -> mutate the clock
    -> both references observe the changed state
    -> sameRef(...) = true
```

## Where FoLang's Ideas Come From

FoLang did not invent most of its ideas. It borrowed them, from languages whose
designers solved a problem well, and then adjusted each borrowing so that the
whole stays uniform.

Naming the sources is useful for two reasons. It is honest about what is
original and what is not. And if you already know one of these languages, the
attribution gives you a foothold: you can start from what you know and learn
only what changed.

The change is the interesting part, so each entry below records both.

### Control flow

**Conditions and loops — Smalltalk.** In most languages `if` is a statement: you
command the machine to test something. In Smalltalk a condition is an object
that receives a message, and FoLang follows that reading.

```folang
(score >= 50).then({ ... }).default({ ... });
```

The condition answers; you attach the consequence. This matches how the English
sentence works — *"score is at least 50, so do this, otherwise do that"* — and
it means conditions, loops, and iteration are ordinary method chains rather than
three separate statement forms. FoLang extends the idea further than Smalltalk
did: `.loop` and `.each` follow the same shape.

**Continuations — Lisp and Scheme.** Explicit control operations rather than a
general `call/cc`. FoLang keeps CPS-style control available under `co.control`
without making the whole language continuation-passing.

**Deferred functions — Go.** Completion work registered as execution reaches it,
run in reverse order. FoLang specifies the interaction with its effect system
precisely, which Go's `defer`/`recover` pairing leaves informal.

### Types and abstraction

**Type classes — Haskell.** The contract-and-instance model, with implicit
resolution removed. A FoLang instance is named at the point of use rather than
searched for by the compiler. That single change eliminates coherence rules, the
orphan rule as a correctness requirement, and the newtype-wrapper workaround. The
cost is that generic code passes its instance explicitly.

**Generics — C++ and Java.** Parameterised types and functions, without C++
template metaprogramming and without Java's erasure.

**Modules and signatures — OCaml, with Scala's objects.** A signature is a type;
a module is a value of it. This is what makes module binding ordinary assignment
rather than a separate linking mechanism.

**Structs, pure types, companion units — Go.** Data separated from behaviour, with
functions living beside the type rather than inside it. FoLang adds the filename
rule, so a struct and its companion unit are related by position rather than by
convention.

**Mixins and traits — Rust, Scala, and others.** FoLang separates the two:
a trait carries behaviour and no state, a mixin may carry both. Multiple
inheritance from Smalltalk-derived and C++ traditions is present but bounded —
at most two direct class parents, with virtual-base identity semantics so a
shared ancestor appears once.

**Extensions and extension methods — C#.** Adding behaviour to a type you do not
own. FoLang scopes activation explicitly with `@co.ddap.use`, so an extension is
never silently in scope.

**Associated and opaque types — Swift, Scala, and Haskell.** Type members
declared by a contract and supplied by its implementation.

**Refinement types — Liquid Haskell and F\*.** A base type narrowed by a
predicate. FoLang checks the predicate at construction rather than proving it
with a solver, which gives most of the guarantee without a theorem prover in the
compiler.

**Dependent types — ATS and Idris.** Types indexed by values, such as
`Vector(3)` and `Vector(n)`. FoLang restricts index equality to syntactic
identity and forbids arithmetic in an index, which is what keeps checking
decidable.

### Expressions and functions

**Pattern matching, let bindings, function bindings — Haskell and Scala.**

**Lambda expressions — Haskell.**

**For comprehensions — Haskell and Python.**

**The ternary form — Python and Scala**, expressed in FoLang as an ordinary
`then` / `otherwise` / `default` chain rather than as separate syntax.

**Functions as values, multiple returns — Go.**

**Callback attachment — JavaScript**, in FoLang's collection and effect
operations.

### Metaprogramming

**Macros — Julia and Nim.** Quotation and escape, operating on syntax rather than
on text.

**Templates — Nim.**

**Decorators — Python.**

**Operator overloading — C++.** Existing symbols given meaning for new types.

**New operator symbols — Raku.** Raku lets a program introduce operators the
language does not define. FoLang allows this too, but confines operator source
to one place and forbids operators from crossing a library boundary. Raku's
grammar changes as a file is parsed; FoLang's operator table is complete before
ordinary parsing begins, so the parse stays single-pass.

### Systems and execution

**Threads and coroutines — Go and Java.**

**Pointers, words, and low-level representation — C**, available only inside the
`system` capability domain.

**A dynamic runtime domain — Java.** Isolated behind a one-way boundary: types
cross inward, never outward.

**Path-based packages — Python.**

### What changed, and why

Most of the adjustments above have the same motivation. A borrowed feature is
kept when it earns its place, and reshaped when its original form would require
machinery FoLang avoids elsewhere — implicit resolution, mid-parse grammar
changes, undecidable checking, or a second way to express something the language
can already say.

The result is not a collection of features from other languages. It is one
language whose ideas came from many.

## A Developer's Guide to Elegance

Elegance here means clarity. It is not only about brevity; it is about leaving
nothing to infer.

Elegant FoLang code makes the shape of the problem easy to see. A reader should
be able to follow what the program is modelling, where its state lives, and why
each abstraction is there. The way a component is written should reveal its
intended semantics.

Clarity takes different forms. Sometimes it is a shorter expression. Sometimes
it is a more explicit one. Often it is a well-chosen type that makes an entire
category of question — and of bug — simply disappear.

The standard used throughout is a simple one:

> Does the code make the problem, the semantics, and the intent clearer?

When the answer is yes, the design is working.

## Before We Begin

The first chapter begins with the question the rest of the book depends on:

> **How does FoLang think about a program?**

Once that model is in place, the syntax follows.
