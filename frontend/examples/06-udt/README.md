# 06 — User Defined Data Types

Spec: *UDT (User defined Data types)*, *CStructs*, *Structs*, *Unions*,
*Enums*, *classes*, *Interfaces*, *Signatures*, *Modules*, *Units in detail*,
*Operators*, *Indexer*, *Local and/or Nested types and functions*.

Every file here is a package source file in the package `06-udt`, so each holds
exactly one primary declaration.

| File | Construct |
|---|---|
| `Point.fol`, `Rect.fol` | `co.lang.cstruct`, cstruct composition |
| `Address.fol`, `Person.fol` | `co.lang.struct`, embedding vs composition |
| `Status.enum.fol` | `co.lang.enum` |
| `Payload.fol` | `co.lang.union` (untagged ADT) |
| `Option.fol` | `co.lang.data` (tagged ADT type constructor) |
| `BaseAccount.fol` | abstract base class, abstract / virtual / sealed methods |
| `Account.fol` | `co.lang.class`, `implements=` / `inherits=`, `@co.dap.oops`, four method types, `@@new` / `@@init`, `this` vs `self`, delegation |
| `SavingsAccount.fol` | a second subclass — many classes per interface, differing overrides |
| `AccountUsage.fol` | construction, static / class / instance calls, virtual dispatch, anonymous classes, class-vs-module cardinality |
| `IAccount.fol` | `co.lang.interface` |
| `PersonStore.signature.fol` | `co.lang.signature`, value and function specs |
| `Repository.signature.fol` | abstract, fixed, and generic type components |
| `PersonStoreImpl.fol` | `co.lang.module` matching a signature |
| `ListStackModule.fol` | binding an abstract generic type component |
| `Text.fol` | standalone `co.lang.unit` |
| `Vector.fol`, `Vector.unit.fol` | struct + companion unit + operators |
| `MyList.fol`, `MyList.unit.fol` | indexer functions |
| `PersonState.fol` | `@co.dap.local` target-local declaration |
| `Anonymous.fol` | anonymous class expressions |

## The rule that shapes all of this

FoLang has **no physical nesting** of named declarations. A struct, class,
module, signature, or interface body may not contain another named
declaration. Instead a declaration stays in its ordinary source location and is
restricted to one or more exact targets with `@co.dap.local`.

## Which container to reach for

```text
struct   → pure data; a same-name companion unit carries the behaviour
cstruct  → ABI-compatible value data crossing zone or native boundaries
class    → behaviour, lifecycle, many instances
module   → one named component with shared state, governed by a signature
unit     → named function container; a same-name struct makes it a companion
package  → folder-based grouping only, never a value
```
