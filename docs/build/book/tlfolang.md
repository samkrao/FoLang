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

Consider a small example. Two employee records hold the same values:

```folang
a Employee = Employee{name: "Rao", id: 1};
b Employee = Employee{name: "Rao", id: 1};

a == b;            // true  — same values
a.sameRef(b);      // false — different objects
```

Now bind a third name:

```folang
c := a;

c == a;            // true
c.sameRef(a);      // true  — c and a are the same object
```

If a later change through `c` surprises you, the surprise came from reading
`c := a` as a copy. FoLang keeps the two questions apart on purpose: `==` asks
about value, `sameRef` asks about identity, and `:=` binds a name to an object.

Reading the language this way is most of what thinking like FoLang means. The
same habit applies elsewhere — `Employee{name: "Rao", id: 1}` is not only
construction syntax, it states which type of object is being created. When you
stop working through syntax and start aligning your domain model with the
language's abstractions, you have the mindset.

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
