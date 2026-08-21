# Parser conformance audit

This audit compares the normative `docs/language-ref.md`, the consolidated
`docs/grammar/folang.ebnf`, and the parser under `frontend/src/parser`.

## Round 10 — 2026-08-20, disjoint grammars are not a shared ambiguity

### Fixed — initialized `co.lang.data` fields were rejected as nested ADTs

Round 9 made `co.lang.data` head a declaration unconditionally in a container
member, which rejected every initialized carrier field in the language:

```text
_ co.lang.class = { payload co.lang.data = someValue; }
    -> a named co.lang.data declaration cannot be physically nested …
```

The grammar says otherwise, and it is not ambiguous about it:

```text
class-member      = field-declaration | function-declaration | lifecycle-…
field-declaration = annotations, identifier, type-expression,
                    [ "=", expression ], statement-end
```

`co.lang.data` is a usable built-in type, `field-declaration` admits an
initializer, and `class-member` offers no `data-declaration` alternative at all.
Exactly one production matches, and it is the field.

**The reasoning error is worth naming precisely**, because it is not the one
Round 9 thought it was avoiding. Round 9 argued that a unit body reads the same
tokens as a declaration, so a class body should agree. But the two member
grammars are DISJOINT rather than competing:

```text
unit-member   has data-declaration, has no field-declaration
class-member  has field-declaration, has no data-declaration
```

Neither body is resolving a conflict. Each has exactly one matching production,
and they differ because the grammars differ. Reasoning by analogy from one body to
the other imported an ambiguity that exists in neither.

The discriminator is the one `atLocalKindDeclaration` has always used: a
declaration-head generic clause, which no field declarator takes.

```text
payload co.lang.data = someValue;      an initialized field
Shape(T) co.lang.data = Some(T) | …;   unmistakably a declaration
```

`requiresGenericClauseToNest` demands that evidence for every spelling listed in
`scanlex.Builtin_types` — `co.lang.data`, and `co.lang.typeclass` and
`co.lang.dependentType`, which overlap the same way. Spellings that are NOT usable
types keep Round 7's rule and need no clause: nothing can be typed
`co.lang.type` or `co.lang.struct`, so those are unambiguous alone.

`rejected/nested-data-declaration-in-class` had codified the wrong reading and now
carries the parameterized form. `accepted/carrier-typed-fields` gained the
initialized members it was missing — `seeded co.lang.data = someValue` beside the
bare `payload co.lang.data`, and initialized `co.lang.value` and
`co.lang.typevalue` fields. Testing only the uninitialized spelling is what let
the regression through.

### Evidence

```text
go test -count=1 ./...
go vet ./src/... ./tests/...
git diff --check
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is 190 fixtures plus EXPECTATIONS.tsv and the
accepted corpus 85.

## Round 9 — 2026-08-20, a closed set instead of a negation

### Fixed — the guard treated every non-data kind as a misplaced primary

`isNestableDeclarationKind` was written as a negation: anything absent from
`scanlex.Builtin_types` heads a declaration. That is not the set of kinds with a
declaration form, and the gap in both directions was doing damage.

**Reserved names were reported as misplaced.** The built-in kind table lists about
twenty names the reference never gives a declaration form —`co.lang.loader`,
`co.lang.macro`, `co.lang.role` and the rest. The guard called each a physically
nested declaration and told the author to move it to a source file of its own with
`@co.dap.local`, which is placement advice for a declaration the language does not
have. DECISION-KIND-001's "no declaration form" diagnostic is the right answer
wherever the spelling appears, and it now stays authoritative here too.

**Non-primary forms got a primary's home.** A function object, a delegate and a
named block are not `<Name>.fol` declarations, and each has a home of its own:

```text
was:  a named co.lang.function declaration cannot be physically nested in a class
      body; declare it in its own package source file and restrict it to this
      declaration with @co.dap.local

now:  … it belongs in an ordinary <Fragment>.unit.fol unit file, written
      "<name> co.lang.function = …"
      … it belongs inside a function or method body, written
      "<name> co.lang.block = { … }"                          (co.lang.block)
      … it belongs in src/component.fol or components/<kind>/component.fol
```

`nonPrimaryKindHomes` already worded every one of these for the misplaced-primary
diagnostic, so the fix is to reuse it rather than to write a second set of
answers that could drift from the first.

**And one kind was silently accepted.** `co.lang.data` is in BOTH built-in tables
— a data type and the head of `data-declaration` — so the negation swept it up
with the types and a nested ADT was taken as a field named for it, carrying its
variant list as a default:

```text
_ co.lang.class = { Shape co.lang.data = Circle(co.lang.float) | Square(…); }
    -> parsed
```

The fix shipped in this round over-corrected and is superseded by Round 10: the
bare `Shape co.lang.data = …` spelling is an initialized FIELD in a class body,
and only the parameterized form is unmistakably a declaration. The reasoning
recorded here — that a unit body reads the same tokens as a declaration, so a
class body should too — was wrong, and Round 10 says why.

The probe asks `hasDeclarationForm` instead, over three sets the parser already
dispatches on — `fileBackedPrimaryKinds`, `typeDeclarationKinds` and
`nonPrimaryKindHomes` — so a kind cannot gain a production in one place without
gaining one in the other.

**A consequence worth stating** — and the one that turned out to be the defect.
`payload co.lang.data = someValue;` in a class body was read as a nested ADT
rather than as a field with a default. See Round 10.

### Corpus

```text
rejected/reserved-kind-in-class            co.lang.loader keeps the "no
                                           declaration form" diagnostic
rejected/nested-function-object-in-class   the unit-member home
rejected/nested-named-block-in-class       the block home
rejected/nested-data-declaration-in-class  the silently-accepted ADT
accepted/carrier-typed-fields              fields typed co.lang.data, any, value,
                                           typevalue, untyped and MatchBindings —
                                           the reading the declaration decision
                                           must not reach
```

### A note on this sequence

Rounds 5 through 9 are one guard, corrected five times, and every correction was
the same mistake in a different place: a boundary drawn by asking what a shape is
NOT rather than enumerating what it is. Blind to annotations, then skipping them
instead of validating them, then folding type kinds in with data types, then
naming one home of two, then treating every unlisted kind as a primary. Each
narrow fix made the symptom go away and left the shape of the error intact.

The corpus could not see any of it, because all five are a guard over- or
under-claiming and every fixture added was a rejected one. The accepted fixtures
added in Rounds 6 through 9 — `annotated-members`,
`extern-forward-type-declarations`, `builtin-typed-fields`,
`companion-unit-type-declarations`, `carrier-typed-fields` — are the ones that now
hold the boundary from the other side.

### Evidence

```text
go test ./...
go vet ./src/... ./tests/...
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is 190 fixtures plus EXPECTATIONS.tsv and the
accepted corpus 85.

## Round 8 — 2026-08-20, ordering and the second home

Two review findings against Round 7's guard, both correct, and both about the
same thing: a check placed where it could speak before the rules that outrank it,
and a message naming one of two valid homes.

### Fixed 1 — the nesting diagnostic masked metadata errors

Round 6 taught the guard to skip annotations in lookahead so it could see the
declaration name behind them. That fixed the blindness and introduced an
ordering bug: skipping metadata is not the same as validating it, so the guard
reached its verdict before `parseAnnotations` ever ran.

The same annotation reported itself on an ordinary field and was silenced on a
nested one:

```text
_ co.lang.class = { @co.dap.nosuchthing counter co.lang.int; }
    -> "@co.dap.nosuchthing" is not a built-in FoLang metadata name

_ co.lang.class = { @co.dap.nosuchthing Inner co.lang.struct = { … } }
    -> a named co.lang.struct declaration cannot be physically nested …
```

An unregistered `@co.*` name and a malformed argument list are errors in their
own right, they come first in source order, and the author has to fix them either
way. Announcing the nesting rule over them buries a problem rather than
reporting one.

The guard now runs AFTER each member's annotations are parsed, which is also what
puts the cursor on the declaration name — so the lookahead no longer needs to
skip anything and `atNestedKindDefinition` lost that step entirely. Thirteen call
sites moved: the ones whose member parser reads its own annotations had the guard
pushed down into that parser, immediately after the read.

The malformed-annotation case had appeared to work. It did not: `skipBalanced`
simply could not balance the unclosed list, so the lookahead failed and the
annotation error surfaced by accident. It is now correct for the same reason as
the others.

### Fixed 2 — the type-declaration home named one container of two

```text
was:  a non-UDT type declaration belongs in an ordinary <Fragment>.unit.fol unit
      file, which is the one container that admits it
```

It is not the one container. "Physical Nesting Rules" permits these declarations
"directly inside an ordinary unit, **and inside a companion unit** where their own
rules permit association with the owner", and "Struct Companion Units" lists
"non-UDT type declarations associated with the owner" among what a companion may
declare. An author writing a type for one struct was being sent to the wrong
file by a message that was confidently wrong rather than merely incomplete.

### Corpus

```text
rejected/unregistered-metadata-on-nested-declaration
        both rules broken at once; the metadata error must win
accepted/companion-unit-type-declarations
        the second home actually parsing — an alias, a newtype, a refinement type
        and a parameterized type in a Vector.comp.unit.fol, beside the companion
        functions that share it
```

The accepted fixture is the one that would have caught Fixed 2. A diagnostic that
names a home is a claim about what parses, and only a positive fixture tests that
claim; the wrong text had been sitting behind a passing suite for a full round.

### Evidence

```text
go test ./...
go vet ./src/... ./tests/...
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is 186 fixtures plus EXPECTATIONS.tsv and the
accepted corpus 84.

## Round 7 — 2026-08-20, the container probe gets its own ambiguity rule

### Fixed — non-UDT type definitions bypassed the nesting guard

`atNestedKindDefinition` delegated to `atLocalKindDeclaration`, whose last line is

```go
return hasGenerics || !isTypeFirstKind(p.lexeme())
```

and `isTypeFirstKind` folds two sets together: the built-in DATA types
(`co.lang.int`, `co.lang.string`, …) and the dedicated TYPE-DECLARATION kinds
(`co.lang.type`, `co.lang.newtype`, `co.lang.opaquetype`, `co.lang.subtype`,
`co.lang.supertype`, `co.lang.dependentType`, `co.lang.kind`). Both were read as
"a type, so this declarator is a variable", so the guard declined every one of
the second set.

In a class and an instance the result was not a wrong diagnostic but a missing
rejection:

```text
_ co.lang.class = { Alias co.lang.type = co.lang.int; }
    -> parsed, as a field named Alias carrying a default
```

In a struct and a trait it surfaced as the field-grammar error the guard was
added to replace.

The reference is direct about this shape. "Physical Nesting Rules" makes non-UDT
type declarations the deliberate UNIT exception — aliases, parameterized and
variant `co.lang.type`, newtypes, opaque types, refinement types, subtypes and
supertypes "may be declared directly inside an ordinary unit" — and then says
they "are not permitted loose at package-file scope or physically inside classes,
structs, modules, functions, or executable blocks unless another section
explicitly grants that context".

**Why one predicate could not serve both callers.** `name KIND = value` is
ambiguous in both places, but not with the same alternative:

```text
executable block      T co.lang.type = a;      a type-level binding — the
                                               lifecycle @@new example writes
                                               exactly this
container member      Alias co.lang.type = …;  a nested type DEFINITION; nothing
                                               can be typed co.lang.type
```

A field CAN be typed `co.lang.int`, so a built-in data type still means a field
in both. Nothing can be typed `co.lang.type`, so a type-declaration kind means a
declaration in a container and a binding in a block. `isNestableDeclarationKind`
therefore excludes only `scanlex.Builtin_types`, and `atLocalKindDeclaration` is
left exactly as it was for the block probe.

The three containers the reference DOES grant this context — unit, signature and
module, through `unit-member` and `signature-type-component` — never reach the
probe; they are exempt from the guard entirely.

The diagnostic now names the right home per family. The two are different and the
reference is specific about both, so one message cannot serve them: a file-backed
primary keeps its own `<Name>.fol` and reaches its target with `@co.dap.local`,
while a non-UDT type declaration belongs in a unit and `@co.dap.local` is not
what it needs.

```text
Address co.lang.struct = { … }   -> … declare it in its own package source file
                                    and restrict it to this declaration with
                                    @co.dap.local
Alias co.lang.type = co.lang.int -> … a non-UDT type declaration belongs in an
                                    ordinary <Fragment>.unit.fol unit file,
                                    which is the one container that admits it
```

### Corpus

```text
rejected/nested-type-alias-in-class   the silently-accepted case
rejected/nested-newtype-in-struct     the same rule in a body whose members
                                      really are fields, so the two readings
                                      compete directly
accepted/builtin-typed-fields         every built-in field spelling the probe
                                      must not claim — int, string, bool, float,
                                      dynamic, auto, a pointer derivation, an
                                      array with an initializer, and a
                                      user-typed field
```

`accepted/builtin-typed-fields` is the fixture that matters. This defect and the
two before it are all a guard mis-drawing one boundary, and the corpus can only
see the side it has a positive case for. The lifecycle block in
`refblocks/parsing` already pins the other side — `T co.lang.type = a;` inside
`@@new` must keep parsing, and it does.

### Evidence

```text
go test ./...
go vet ./src/... ./tests/...
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is 185 fixtures plus EXPECTATIONS.tsv and the
accepted corpus 83.

## Round 6 — 2026-08-20, review follow-up on the nesting guard

Two review findings against Round 5, both correct, and a third defect the first
one had been hiding.

### Fixed 1 — folder fixtures in the accepted corpus were never executed

`conformanceFixtures` globbed `examples/<outcome>/*.fol`, flat files only, while
`rejectedFixtures` had always globbed both the flat and the `*/*.fol` forms. Five
accepted fixtures sit in folders, because a fixture whose rule needs a particular
source-file classification has to carry the name FoLang requires —
`EmployeeTrait.fol`, `tools.unit.fol` — and only a folder gives it that name while
keeping the case's own. All five were skipped in silence:

```text
trait-declaration              mixin-declaration
kind-members-in-exempt-bodies
signature-kind-members         module-kind-members
```

So Round 5's claimed regression coverage for the unit, signature and module
exemptions did not exist, and neither did the positive coverage for traits and
mixins — the whole feature's accepted side. `go test ./...` passed throughout,
because a fixture that is never discovered cannot fail.

Discovery now mirrors the rejected side, and the accepted case count went from 75
to 82. A silent skip is the failure mode worth guarding rather than the one
instance of it, so a subdirectory that yields no `.fol` file is now an error: both
an empty folder and one holding a misnamed file fail the suite instead of reading
as "no such case".

### Fixed 2 — the nesting guard did not look past annotations

Every declaration production begins `annotations, ...`, and each new call site
invoked the guard before `parseAnnotations()`. `atLocalKindDeclaration` requires
the cursor to be at the declaration name, so it saw the `"@"` and declined.

The case that matters is the one a reader actually writes:

```folang
_ co.lang.class = {
    @co.dap.local
    Address co.lang.struct = { … }
}
```

`@co.dap.local` is what the diagnostic recommends, so someone who half-remembers
the rule puts it here — in the nested position — rather than on a declaration in
its own file. That is precisely the reader the dedicated diagnostic exists for,
and precisely the reader who was sent back to `expected ";" after a field
declaration` instead.

The guard now skips metadata applications inside its own lookahead.
`skipAnnotationApplications` is extracted from `atPrimaryDeclaration`, which had
the same loop inline, so there is one implementation rather than two.

### Fixed 3 — the guard claimed forward declarations

Fixing the annotation blindness immediately failed `refblocks/parsing/L7080`, the
reference's "Types external declaration" example:

```folang
_ co.lang.class = {
    @co.dap.declare(extern)
    Dept co.lang.struct;
}
```

This is not a regression from Fixed 2. Round 5's guard had been rejecting the
UNANNOTATED spelling of the same form since the day it was added:

```text
_ co.lang.class = { Dept co.lang.struct; }   ->  rejected as physically nested
```

Nothing caught it because the reference's example carries `@co.dap.declare`, and
the annotation blindness let that one spelling through by accident. One bug was
masking the other, and the corpus agreed with both.

The reference settles which spelling is canonical by saying neither is: "For
functions and types `@co.dap.declare` is optional." So the annotation cannot be
the discriminator. The binding is:

```text
Dept co.lang.struct;          forward/extern declaration — a legal member
Dept co.lang.struct = { … }   a definition — physically nested, forbidden
```

`atNestedKindDefinition` requires the binding, which is also the more faithful
reading of the rule: a forward declaration introduces no nested body and no
nested scope, and physical nesting is about exactly that. Kind options are
skipped before the test, so `co.lang.module->( … ) = { … }` is still caught.

### Corpus

```text
rejected/annotated-nested-declaration        the @co.dap.local-in-the-wrong-place
                                             case
accepted/annotated-members                   an annotated field, an annotated
                                             field with a default, and annotated
                                             methods including one whose
                                             annotation carries a bracketed
                                             argument list — the shapes the new
                                             lookahead must not claim
accepted/extern-forward-type-declarations    BOTH spellings of the forward form,
                                             annotated and bare; the corpus had
                                             only ever carried the annotated one
```

The two accepted fixtures are the point of this round. Fixed 2 and Fixed 3 are
both over-claiming by a guard, and a rejected fixture cannot catch over-claiming
— only a positive one can.

### Evidence

```text
go test ./...
go vet ./src/... ./tests/...
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is 183 fixtures plus EXPECTATIONS.tsv and the
accepted corpus 82, all of which now actually run.

## Round 5 — 2026-08-20, closing the reported items

Rounds 3 and 4 reported six items that needed either a specification decision or
a removal wider than a parser fix. The grammar has since answered the open
question, so all of them are closed here.

### 1. Traits and mixins are implemented

`docs/grammar/folang.ebnf` gained `trait-declaration` and `mixin-declaration`
with their bodies, member rules and `trait-member-guard`, which settles Round 3's
Reported 3 in favour of supporting both. The reference already carried headed
`## Traits` and `## Mixins` sections and Builtin Kinds rows for them; what it
lacked was a place in its own inventory of permitted top-level declaration kinds,
so that list now names trait, mixin and extension — extension was missing from it
too, though `extension-declaration` has been a `primary-declaration` alternative
all along.

`decl_traitmixin.go` follows interface-declaration's representation rather than
class-declaration's: `ast.TypeDeclarationStmt` stores its symbol as an
`ITypeSymbol`, which only `TypeSymbol` satisfies, so the kind is recorded there
instead of through a new symbol type. Neither form pushes a lifecycle capability
or a `self` receiver context, because neither is instantiable and neither owns
lifecycle machinery — `self` belongs to a `co.lang.class` method or a
target-bound extension's `@co.dap.class` method, and a trait or mixin method is
neither until a class composes it.

The two differ by exactly one property, which is why they are two productions:

```text
trait   interface-like, MAY carry default implementations, carries NO instance
        state, admits no virtual method
mixin   the abstract-class-like form, which MAY carry state, abstract methods,
        implemented methods and virtual methods
```

`trait-member-guard` is enforced where the parser can decide it from the
declaration shape: a non-function member is state and is refused as such, and
`@co.dap.virtual` is refused by name and pointed at `co.lang.mixin`. An abstract
member needs no special case — `function-binding` already admits a bare
statement-end, so `someFunction()->();` is the same production as a defaulted one.

Two stale premises went with the change. `TestClosedPrimaryDeclarationRejectsRelocatedForms`
asserted that `co.lang.trait` has "no declaration form in the reference", and
`rejected/general-kind-primary` asserted the same in a fixture; the fixture now
names `co.lang.loader`, which is still listed in Builtin Kinds and still has no
production. `refblocks -write` promoted the trait block from `excluded/` to
`parsing/`.

One correction to Round 3. It recorded the mixin block's manifest reason —
`block elides code with "..."` — as wrong, on the grounds that the real cause was
the unsupported kind. With the kind supported the block still does not parse, and
the elision is exactly why, so the generator's reason was right and Round 3's
claim about it was not. What was true is the narrower point: a first-match-wins
heuristic can record a cosmetic cause while a substantive one is also present,
and only supporting the kind revealed which applied.

### 2. `src-library` is removed

The parser, `ast.ImportStmt`, `importcheck.Import`, the cycle graph and the
driver no longer carry the field. "Import Directive Fields" defines exactly
`package`, `library`, `component` and `as`, so `src-library` now falls to the
same preservation path as any other unrecognized field of a recognized form
rather than to bespoke validation, and the diagnostic that pointed at
"the project-local srclib/ source library" is gone.

Three consequences were followed through rather than left dangling.
`packageCycleTarget` reduces to `imp.Package`, because only `package=` names a
package context; `hasSourceLibraryImport` and its `buildLibs` contribution are
gone; and `TestSourceLibrarySurfaceDistinguishesItsSlotFromASamelyNamedPackage`
was replaced by a test that asserts what now holds — that `library=` and
`component=` contribute no package-graph edge at all.

### 3. The withdrawn library and package surfaces are removed

Deleted: `decl_library.go` entirely, `parseLibrarySurfaceFile`,
`parseLibraryKindAnnotation`, `parsePackageMetadataSourceFile`,
`parsePackageAliasDeclaration`, `parsePackageAliasBody`, `sourceLibrarySlotOf`,
`logicalPathOf`, `librarySymbol`, `scanLibraryBodyImports`, the
`sourceClassLibrarySurface` and `sourceClassPackageMetadata` classifications, the
`unitLibrary` unit kind, the `co.lang.library` and `co.lang.package` dispatch
arms and their `nonPrimaryKindHomes` entries.

The `Implements:` claims that named productions the grammar no longer defines
went with them. `grammar-map.json` records the result: `missing` fell from 175 to
163, and the only remaining claim naming a production the grammar does not define
is `operator-source-file`, which `cmd/docgen/grammarmap.go` documents as
intentional.

The stale `package.fol` comment on `classifySourceFilename` was corrected rather
than deleted. Its behaviour was already right — the reference says such a file
"is classified by the ordinary `<Name>.fol` filename rule" — but the comment
still described it as reserved metadata, which is the opposite.

**One thing this removal exposes rather than fixes.** `importcheck.File`'s
`IsLibrarySurface`, `LibraryName`, `LibraryType` and `LibraryPath` were filled
only from the withdrawn `library.fol` root, so `direction.go` and the surface half
of `restricted.go` now receive no library at all. That is not a regression — the
fields have been unreachable for as long as `library.fol` has been, so those
checks were already inert — but it is a real gap: the reference's surviving
standalone-library surface is `src/component.fol` carrying `@co.dap.library`, and
nothing points the boundary rules at it. Both `parser.importFile` and
`ScanImportSurface` say so where the fields would have been set. Re-pointing them
RESTORES enforcement rather than removing dead code, so it is left as its own
change.

### 4. Physical nesting is reported by its own rule

The reference gives nested declarations a rule and names their replacement, but
the parser let them fall through to whatever member grammar failed first:

```text
_ co.lang.struct = { Address co.lang.struct = { … } }
    was:  a struct field cannot have a default value
_ co.lang.class  = { Address co.lang.struct = { … } }
    was:  expected ";" after a field declaration, found "}"
    now:  a named co.lang.struct declaration cannot be physically nested in a
          <container> body; declare it in its own package source file and
          restrict it to this declaration with @co.dap.local
```

`atLocalKindDeclaration` already answered this question for an executable block,
so `rejectNestedKindDeclaration` reuses it at the member position of every
container whose member grammar admits no kind-introduced declaration at all:

```text
struct   cstruct  union     enum       class
trait    mixin    interface typeclass  object
matcher  instance extension                     -> guarded

unit     signature module                       -> exempt
```

The exemptions are exactly the three member grammars that name a built-in kind:
`unit-member` admits data-declaration, type-declaration,
function-object-declaration and delegate-declaration, and `signature-member` and
`module-member` admit signature-type-component and the associated-type forms.
Guarding those would reject the reference's own examples.

The first pass covered only the first eight and exempted `instance` along with
the three, on the stated grounds that an instance "legitimately declares
type-related members". It does not: `instance-body` is
`{ function-declaration | variable-declaration }`, so that exemption matched
nothing in the grammar. A variable declarator's type is a type-expression, which
`atLocalKindDeclaration` already separates from a kind token through
`isTypeFirstKind`, so `cached co.lang.bool = …` stays an ordinary member while
`Inner co.lang.struct = { … }` is caught. Enum, extension, matcher and typeclass
were missed the same way and are guarded now.

Fixtures cover both directions. Eight nested cases are rejected by the nesting
rule, and three accepted fixtures pin the exempt bodies — a unit declaring a
parameterized type, an ADT, a newtype, a delegate and a function object; a
signature declaring associated types and type components; and a module binding
them — so the guard cannot later creep into the grammars it must not reach.

### 5. The canonical file key is defined by one rule

`canonicalFileKey` derived the key by case-folding the stem with underscores
removed, while `upperCamelFilenameName` derived the declaration name by a
separate walk — two implementations of one relationship, free to drift. The key
is now the case fold of the derived name, which is what "canonical file key =
caseFold(normalize(filename stem))" means once `normalize` is read as the
derivation the reference already defines.

Round 3 reported this as a defect in the PARTITION, on the grounds that
`employeeservice.fol` and `employee_service.fol` derive different names yet share
a key. The partition is unchanged and was right; the reasoning was wrong. Those
two must share a key, because `employeeservice.fol` is a case variant of
`EmployeeService.fol`, which the reference already requires to conflict, and
conflict has to be transitive for a key-based index to mean anything. The real
defect was narrower: a comment claiming the key set is "exactly the set of
spellings that all derive the name EmployeeService", which is false for the
fourth spelling.

The reference now writes that transitivity out, and a test asserts the invariant
directly — two stems share a key exactly when their derived names are case
variants — rather than restating the implementation.

### Evidence

```text
go test ./...
go vet ./src/... ./cmd/...
go run ./cmd/refblocks -write
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is 182 fixtures plus EXPECTATIONS.tsv and the
accepted corpus 80.

## Round 4 — 2026-08-20, negative and corner cases

Round 3 probed whether each construct is admitted. This round probed the
boundaries of each construct instead: the malformed spellings next to a legal
one, the repetition of something admitted once, the empty and doubled forms of
every delimited list, and the shapes two rules compete for. Four defects came out
of it, and the rejected corpus gained 35 fixtures.

### Fixed 1 — an unterminated block comment crashed the parser

```text
count := 1; /* this comment is never closed
```

The scanner reports this correctly. Rendering the report is what failed:

```text
panic: strings: negative Repeat count
    strings.Repeat(...)
    helpers.stringWithArrows(...)          src/helpers/strwitharros.go:62
    helpers.(*InvalidSyntaxError).AsString  src/helpers/error.go:159
```

`stringWithArrows` draws the caret run with `strings.Repeat("^", colEnd-colStart)`
and the leading indent with `strings.Repeat(" ", colStart)`, neither of them
clamped. An unterminated comment runs to end of file, so its span ends at a
SMALLER column than it starts at and the caret count goes negative.

The severity is in WHERE it fired. `ParseFile` exists so an embedding consumer —
a language server above all — gets diagnostics from a malformed file instead of a
crash, and its doc comment says so. This crash happened after the diagnostic had
already been produced, when the consumer went to read it, so it defeated that
guarantee at the last possible step and on exactly the input the guarantee is for.
Nothing in the corpus caught it because a fixture's contract is "produces a
diagnostic", and it did.

Every index is now clamped. `strwitharros_test.go` covers the five degenerate
spans that reached a negative count — end-before-start on one line and across
lines, a start column past the end of its line, negative columns, and an empty
source — and pins the ordinary rendering so the clamps cannot quietly swallow a
correct caret.

### Fixed 2 — repeated integer suffixes were accepted

`1uu`, `1zz`, `1lll`, `1lul`, `1ulu` and `1uul` all parsed. The production admits
each marker once:

```text
integer-suffix = unsigned-suffix, [ long-suffix | long-long-suffix | size-suffix ]
               | long-suffix,      [ unsigned-suffix ]
               | long-long-suffix, [ unsigned-suffix ]
               | size-suffix,      [ unsigned-suffix ]
```

Two places let the repetition through. `fuseNumericLiteral` re-attaches a suffix
to a lexeme the scanner had already suffixed, because `numericContinuation` asks
only whether the TAIL is a suffix and never whether the head still needs one. The
validator then accepted the result: `parseIntegerLexeme` stripped the suffix with
`strings.TrimRight(lexeme, "uUlLzZ")`, which asks which characters are suffix
characters and never how many of each the production admits.

The check now lives in the validator, which already owns the "malformed integer
literal" diagnostic, and reads the suffix as the production writes it —
longest-first so `ll` is not two `l`s, and case-consistent so `lL` is neither.
The floating side already rejected `3.14ff`, `1ue5` and `1e5u` through
`parseFloatLexeme`; only the integer path was open.

`accepted/integer-suffix-combinations.fol` pins all 22 admissible spellings, so
the new check cannot start over-rejecting.

### Fixed 3 — `@co` was accepted as a custom annotation

`@co.dap` and `@co.ddap` were correctly refused as unregistered language-owned
metadata names, but the bare root was not:

```text
@co
count := 1;      -> parsed
```

`IsLanguageOwnedMetadataName` tested `strings.HasPrefix(name, "@co.")`, so `@co`
fell through to the custom-annotation path to be resolved later through the
symbol table. It can never resolve there: `co` is a hard-reserved word and the
built-in package root, so no user-defined annotation or decorator can carry that
name. The bare root now counts as language-owned and reports the same
unregistered-name error as every other `co.*` spelling.

### Fixed 4 — two rules were broken with diagnostics that named neither

Both are the failure mode this corpus's manifest exists to prevent, on the
production side rather than the fixture side: the parser refused the construct,
but for a reason that does not lead a reader to the rule.

```text
employee := Employee{id = 1};
    was:  expected ";" after an inferred variable declaration, found "{"
    now:  an object field initializer binds its value with ":", as in
          "Employee{id: 1}"; "=" is not an object-field initializer binder
```

The reference states that binder outright in "Canonical Object and Collection
Construction". `looksLikeObjectConstruction` simply declined the shape, so the
expression fell through to the type-as-value reading and died on the block that
followed. A guard now recognises the `Name "{" identifier "="` shape after the
construction guard declines, which is late enough that a well-formed
construction never reaches it and a bare block never does either.

```text
letter := 'ab';
    was:  expected an expression, found "'"
    now:  a character literal contains exactly one character; 'ab' encloses
          more than one

letter := '';
    was:  expected an expression, found "'"
    now:  a character literal contains exactly one character; '' encloses none
```

The scanner already declined this span deliberately — `labelIdentifierLength`
rejects a trailing apostrophe with the comment that reporting it as an
unterminated label "would name the wrong construct" — but nothing then named the
right one. It now does, after both the character-literal and label rules have
declined, and only for a span that closes on its own line. A backslash still
falls through to the escape rules, whose unsupported-feature error is more
specific.

The two ways of breaking the rule are reported apart. Both `''` and `'ab'` hold
the wrong NUMBER of characters, so one message covered both at first — and told a
reader looking at `''` that it "encloses more than one", which is plainly untrue
of the source in front of them. A correct rejection is not the whole job when the
reason given is false. `character_literal_test.go` pins the two wordings apart,
along with the spellings the rule must not capture: a non-ASCII one-CHARACTER
literal, a label, and an escape whose unsupported-feature diagnostic is more
specific than the count rule.

### Corpus — 35 rejected fixtures and 2 accepted ones

Every fixture states the diagnostic its FIRST finding must contain, and each was
checked to die on its own rule rather than somewhere earlier.

```text
literals      integer-suffix-repeated, integer-suffix-length-repeated,
              octal-literal-invalid-digit, character-literal-two-characters,
              character-literal-empty, unterminated-block-comment,
              unknown-symbolic-run
collections   collection-body-form-list/-map/-set,
              reserved-collection-constructor, object-field-equals-binder,
              argument-list-trailing-comma
match         match-two-defaults, match-default-uncalled,
              match-case-empty-pattern, match-two-matcher-arguments
binding       comprehension-two-bindings, comprehension-missing-yield,
              let-bindings-unbraced, let-missing-in
metadata      metadata-name-bare-co-root, annotation-list-unclosed,
              pragma-after-entry-statement
declarations  enum-double-separator, enum-body-semicolon,
              cstruct-embedded-field, interface-member-with-body,
              signature-member-with-body, extension-unknown-option,
              extension-missing-target, unit-member-field,
              unit-member-loose-statement, unit-file-non-unit-kind,
              companion-unit-explicit-name
```

The three collection-body fixtures are worth naming separately: `co.core.List`,
`co.core.Set` and `co.core.Map` each take one fixed body form, and the parser
already reported the right rule for all three — the corpus simply had no case for
it. `reserved-collection-constructor` covers the four registry names whose body
forms the alpha profile does not define.

`accepted/integer-suffix-combinations.fol` and
`accepted/character-and-comment-corners.fol` hold the positive side of this
round's two lexical changes: every admissible integer suffix, a non-ASCII
character literal, the non-nesting block comment, and the separated `+ +x` the
reference gives as valid where `++x` is not.

### Not gaps

Probes that look like over-acceptance but are the parser correctly declining to
enforce a semantic rule, or correctly following the grammar:

- `this.continue 'blockLabel;` parses even though a plain labeled block is not a
  valid continue target. `continue-target-guard` is a semantic condition and the
  parser says so in `parseContinueStatement`: whether an enclosing region carries
  the label, and whether that region is a loop, are questions about the enclosing
  declaration's control regions rather than about the token stream.
- `.each()`, `.each(i, v)`, `.each(i, v, w, {})`, `.then()`, `.loop()` and
  `.loop({}, {})` all parse. Section 12 is informative, exactly as Round 3
  recorded for `.loop(…).default(…)`; lowering declines each and leaves the
  ordinary member chain.
- `arr[]` and `co.lang.int->()` parse: `index-suffix` and `parenthesized-type-list`
  both make their contents optional.
- `/* a /* b */` closes at the first `*/`, which is the documented non-nesting
  rule, not an oversight.
- Enum trailing separators and variant payloads parse, which `enum-body` and
  `enum-variant` both admit.

### Reported — additions to Round 3's list

- `extension-target-options` is spelled with a literal `"="` while
  `matcher-options` uses `annotation-binder`, so the grammar admits
  `co.lang.matcher->(type: T)` but not `co.lang.extension->(fortype: T)`. The
  parser accepts both, through the shared kind-options reader. The reference
  writes `=` for both and DECISION-ANN-001 makes the two binders interchangeable
  generally, so the asymmetry looks like a consolidation slip rather than an
  intended distinction; nothing was changed pending that reading.
- Round 3's "nested declarations are rejected by the wrong rule" has more
  members than the two recorded there. A method in a union body reports "a union
  field cannot have a default value", and a field or a loose statement in a unit
  body reports "expected \"(\" to open a parameter list" — none of which names
  the member rule the reference states for those containers. The two fixtures
  `unit-member-field` and `unit-member-loose-statement` pin the current wording
  so a later correction is visible.

### Evidence

```text
go test ./...
go test ./tests/parser -count=1 -run "Test(EBNFConformance|GrammarProductionsHaveImplementationTrace|RefBlocks)"
go run -tags partrace ./cmd/docgen
```

All pass. The rejected corpus is now 173 fixtures plus EXPECTATIONS.tsv, and the
accepted corpus 75.

## Round 3 — 2026-08-20

The parser was probed construct by construct against the reference, one
compilation unit per rule, in the manner Round 2's "Evidence and limits"
prescribes. Lexical profile, statement termination, control-chain vocabulary,
match chains, wildcard placement, collection construction, type derivations,
generic arrow tails, function/lambda/closure forms, pattern clauses, labels,
lifecycle declaration and `::` invocation, entry-file restrictions, primary
declarations, companion units, component surfaces, operator source, and BOM
handling all answered as the reference requires.

Two behavioural gaps were found and fixed, one generated-artifact corruption was
repaired, and the rest of what follows is spec-side or deferred and is reported
rather than changed.

### Fixed 1 — `@co.ddap.use` and `@co.ddap.alias` closed their field sets

"Built-in Metadata Parsing" is explicit that the built-in registry closes the
metadata NAME and deliberately not its fields:

> when the frontend has no defined knowledge or semantic handling for that
> field, the field is still accepted, collected, and preserved as parsed; lack
> of frontend field knowledge alone is not an error and does not block frontend
> artifact generation

and it names the two cases as deliberately different — an unknown built-in FORM
name is a parse error, an unknown FIELD of a known form is not. The consolidated
grammar says the same in three places, and `import-directive-guard` spells out
that "additional unhandled fields remain preserved".

`@co.ddap.import` already complied. The other two did not:

```text
@co.ddap.use(from="tu", scope="file")     -> unknown use field "scope"
@co.ddap.alias(co.out, as="out", extra=1) -> an alias directive has exactly two fields
```

The alias diagnostic also blamed a trailing comma for what was a third field.

Both now parse the unfamiliar field through the common annotation-value grammar
and preserve it AS PARSED, which is the other half of the same rule: the parser
must collect "every supplied positional argument, named argument, field,
attribute, and argument expression", so a preserved value keeps its own shape.
`enabled=co.const.true` stays a bool, `retries=3` stays an integer,
`extensions=[a, b]` stays a list and `options={mode: eager}` stays a map.

That shape cannot survive the use directive's `map[string][]string` field map,
which exists to hold the two reduced known fields, so `UseStmtDirective` gained a
separate `Preserved map[string]any`. Rendering a value into the string map would
have been irreversible — `[a, b]` reaching a later stage as the text `"[a b]"` —
and would have satisfied a fixture that only checked the file parses while still
losing the application the reference requires to be kept.

A REPEATED field is reported rather than resolved silently. Keeping the last
discards the first and keeping the first discards the last, and neither preserves
the complete application; for the alias directive this covers a field repeating
what the positional target or `as` already bound, which would otherwise overwrite
a validated field or be silently dropped.

Validation of the fields the frontend *does* understand is unchanged: a non-`co.*`
alias target, an unspellable alias name, a bare `methods=upperCase`, a trailing
comma and a malformed value are all still rejected exactly as before. Only the
unknown NAME is now tolerated.

`examples/rejected/use-directive-unknown-field.fol` asserted the removed
behaviour and has moved to `examples/accepted/`, joined by
`alias-directive-unknown-field.fol`; `use-directive-repeated-field.fol` covers the
new rejection, and the manifest rows moved with them.
`src/parser/metadata_preservation_test.go` reads the preserved values back out of
the AST, because a corpus fixture proves only that a file parses and the
flattening above is invisible to one.

### Fixed 2 — a diagnostic directed developers to a withdrawn project layout

An implementation of an unregistered custom operator reported:

```text
… declare its parse properties once in srclib/operators/library.fol
```

`srclib/` appears nowhere in the reference or the grammar. The canonical
location is `components/operators/component.fol`, which is what
`loadProjectOperatorBootstrap` actually reads. The message and the three
neighbouring comments that named the same withdrawn path were corrected.

### Fixed 3 — three generated artifacts were committed with merge-conflict markers

`docs/grammar-map.json`, `docs/callgraph.json` and `docs/trace.json` each carried
unresolved `<<<<<<< Updated upstream` / `>>>>>>> Stashed changes` blocks, so none
of the three was valid JSON and nothing that consumes them could load them.
Regenerating with `go run -tags partrace ./cmd/docgen` resolved all three; the
regenerated content matches the "Updated upstream" side, which is the current
parser — `parseBreakStatement`/`parseContinueStatement` rather than the withdrawn
`parseLoopControlStatement`, and `map-literal`/`map-entry` correctly listed as
grammar productions the parser deliberately does not implement.

### Reported 1 — `src-library` is an import field the specification does not define

"Import Directive Fields" admits exactly `package`, `library`, `component` and
`as`. The parser additionally implements `src-library=true`, validates its value,
requires it to accompany `library=`, sets `p.buildLibs`, and carries it into
`ast.ImportStmt` and `importcheck.Import`. Its diagnostic points at
"the project-local srclib/ source library", a layout the reference no longer has.

This is live behaviour rather than dead code, so removing it touches the AST, the
import checker and the driver's build-libraries flag. It is left for the same
decision that governs the dead-code cluster below.

### Reported 2 — the withdrawn library/package surfaces are still carried

Round 2 recorded `sourceClassLibrarySurface`, `sourceClassPackageMetadata` and
their parse functions as unreachable and deferred removal pending a `src/project`
decision. That is still true and the cluster is larger than described: the
`co.lang.library` and `co.lang.package` arms of `dispatchKindDeclaration` are
unreachable too, because neither spelling is in the scanner's `Builtin_Kinds` any
more. A file written that way therefore dies on `"_" is a contextual wildcard`
instead of a diagnostic naming the withdrawn kind.

The cost is measurable rather than cosmetic. Fourteen `Implements:` claims in the
parser name productions the grammar no longer defines, and every one of them is
counted in `grammar-map.json`'s `missing` list:

```text
co-path                          library-surface-file
import-field                     operator-source-file
library-declaration              package-alias-body
library-kind-string              package-alias-declaration
library-member                   package-metadata-source-file
source-library-surface-file      standalone-library-kind-annotation
standalone-library-surface-file  use-field
```

`operator-source-file` is intentionally missing and documented as such in
`cmd/docgen/grammarmap.go`. The rest are the withdrawn surfaces plus the two
directive sub-productions the consolidation folded away.

### Reported 3 — the reference declares Traits and Mixins that no production admits

`## Traits` and `## Mixins` are headed reference sections with `folang` blocks
declaring `_ co.lang.trait = { … }` and `_ co.lang.mixin = { … }` in their own
`<Name>.fol` files, and the Builtin Kinds table lists both with stated purposes.
The parser rejects each with "is a built-in kind name with no declaration form
and cannot be declared", and `TestClosedPrimaryDeclarationRejectsRelocatedForms`
states as fact that neither "has a declaration form in the reference".

The reference is internally divided on this. Its own enumeration of permitted
top-level declaration kinds in "Package Source Files" lists fourteen and includes
neither, and `primary-declaration` in the consolidated grammar has no alternative
for either. So the reference both defines these declarations and omits them from
the list of what a package source file may hold.

This needs a specification decision, not a parser guess: either the two sections
and the two Builtin Kinds rows are withdrawn, or `primary-declaration` gains two
alternatives and the member rules for a stateless trait and a stateful mixin are
written down. Nothing was changed here.

`testdata/refblocks/excluded/MANIFEST.tsv` currently obscures half of this. The
trait block is classified correctly:

```text
L1814/EmployeeTrait.fol   by-design  uses the deliberately unsupported co.lang.trait declaration kind
```

but the mixin block is not:

```text
L1840/EmployeeMixin.fol   by-design  block elides code with "..." and is illustrative
```

The elision is incidental — the block still fails on `co.lang.mixin` with the
`...` removed. The generator's `hasElision` check simply runs before a person
sees the block, and first match wins, so a substantive exclusion was recorded as
a cosmetic one. The manifest is generated and was not hand-edited.

### Reported 4 — `block-item` admits a directive the reference forbids

```text
block-item = use-directive | statement ;
```

"Directive Placement" is category-wide and states the opposite in a table that
names the case directly — `inside ordinary/nested block -> compiler error` — and
adds that "a directive's semantic scope is defined by the individual directive,
but its syntactic placement is always file-level", naming `@co.ddap.use` as an
example. The parser follows the reference and rejects a `use` directive inside a
block. The grammar's alternative is a consolidation error; the reference governs,
so the code is right and the grammar is wrong.

### Reported 5 — the grammar's provenance header is stale

The header records the reference it was consolidated from by name and SHA-256:

```text
language-ref(20260815-extension-metadata-rules).md
SHA-256: 2bcabef8408abc49f0599dfbe4519e862b8000c73dbd8675c2bc17b744e821f7
```

`docs/language-ref.md` no longer hashes to that value. The header, several inline
notes (C.4, C.4.1, C.9, C.11) and `CLAUDE.md` all cite an Appendix C that the
current reference does not contain — it ends at Appendix B. Every cross-reference
into Appendix C is therefore unresolvable, including the one CLAUDE.md relies on
to break ties inside the reference.

### Reported 6 — nested declarations are rejected by the wrong rule

The reference gives inner declarations their own rule and their own example:

```text
structs cannot declare inner structs     ❌  compiler error — only through @co.dap.local
```

The parser refuses them, but never for that reason:

```text
_ co.lang.struct = { Address co.lang.struct = { … } }
    -> a struct field cannot have a default value

_ co.lang.class  = { Address co.lang.struct = { … } }
    -> expected ";" after a field declaration, found "}"
```

Both die on the field grammar before the nesting rule is reached. By this
corpus's own standard — a rejected fixture must fail on the rule it is named for,
not somewhere else entirely — these should name the physical-nesting rule and
point at `@co.dap.local`.

### Reported 7 — the filename canonical key and the derived name disagree

`canonicalFileKey` drops underscores before case folding, and its comment claims
the resulting key set is "exactly the set of spellings that all derive the name
EmployeeService". It is not: `employeeservice.fol` shares that key while
`upperCamelFilenameName` derives `Employeeservice` from it, because a single-case
segment carries no word boundary to recover. Two files deriving *different*
declaration names are therefore reported as declaring the same one.

The reference specifies the key as `caseFold(normalize(filename stem))` and
illustrates only case variance, so whether `normalize` is meant to remove
underscores is a reading the reference does not settle.

### Not gaps

Several shapes that look like gaps are the parser correctly declining to enforce
a semantic rule:

- `.loop({…}).default({…})`, `.loop({…}).otherwise(c).then({…})` and
  `.each(…).loop({…})` all parse. Section 12 is explicitly informative and "not a
  second parse path", and the reference adds that the parser's built-in-method
  classification "is only a lookup candidate" whose control-flow node "becomes
  final only when the built-in meaning wins". Lowering correctly refuses all
  three, leaving each as the ordinary member chain it must remain in case a
  receiver-owned or activated `default`/`otherwise`/`loop` wins resolution.
- `@co.ddap.import(component="packaged")` parses. The closed
  `application`/`native`/`dynamicvmrt` value set is import-target resolution,
  which `import-directive-guard` assigns to semantic validation.
- An operator declaration with a reserved future fixity, a precedence outside
  0–100, or a language-owned symbol is accepted through the ordinary component
  root but rejected by `parseOperatorSource`, which is the reader every project
  compilation actually uses for `components/operators/component.fol`. The
  operator-source corpus already covers all three.

### Evidence

```text
go test ./...
go test ./tests/parser -count=1 -run "Test(EBNFConformance|GrammarProductionsHaveImplementationTrace|RefBlocks)"
go run -tags partrace ./cmd/docgen
```

All pass. The probes behind this round were complete compilation units placed
under the filename their rule requires, checked for accept/reject against the
reference rather than against the implementation traces.

## Round 2 — 2026-08-17

Three conformance gaps were found by probing the parser construct by construct
against the grammar, rather than by inventorying implementation traces. All
three are fixed; the reference was not modified.

### 1. An untyped `{ … }` map literal was accepted as a value

`primary-expression` deliberately omits `map-literal`: a braced map body is an
object-literal representation, so it is a collection BODY reachable only through
`typed-collection-literal` behind a type prefix, and "an untyped `{ ... }` map
literal is not a FoLang value" (docs/language-ref.md, "Canonical Object and
Collection Construction"). The parser had a `map-literal` alternative in
`parsePrimary`, guarded by a `looksLikeMapLiteral` lookahead.

That guard also decided, at five other sites, whether a braced group was a body
or a map, so it MISREAD any block whose first statement began `identifier ":"`.
A labeled block was the common casualty:

```text
run()->() = {
    outer: {          // read as a map key, so the whole body became an
        x co.lang.int = 1;   // alias-binding expression rather than a block
    }
}
```

`parseMapLiteral` and `looksLikeMapLiteral` are gone, and every braced group in
operand position now takes the block reading that `non-block-expression-guard`
requires. `parseCollectionBody` continues to read a typed map body, which is the
only place the grammar reaches one.

### 2. A lifecycle name could be invoked as a member

The consolidated grammar previously admitted `lifecycle-name` through
`member-suffix`, contrary to the lifecycle rule. `@@new` and `@@init` are
DECLARATION spellings, and construction is invoked through the ordinary member
names:

```text
c := Employee.new(co.lang.int, co.lang.string).init(1,"Rao");
```

Every invocation the reference writes uses the plain name — `self.parent.new()`
and `this.parent.init()` included — and none writes `value.@@new(…)`.

`parseMemberOrMatchSuffix` nevertheless built a member access from a lifecycle
name. `value.@@new(1)` did not reach it, because the scanner's maximal symbolic
run takes `.@@` as one token, but the whitespace-separated `value. @@new(1)`
scans as DOT then SPECIAL_METHODS and WAS accepted. The branch now reports the
construction chain instead, so both spellings are refused.

### 3. `self` and `forall` could not be used as ordinary identifiers

Both are `contextual-keyword`, not `hard-reserved-word`, and the `token`
production has no contextual-keyword alternative — they ARE identifier tokens
that the parser reclassifies only where their contextual form holds. The
reference devotes a section to it: `forall` "is **not globally hard-reserved**",
is recognized only when it "begins this polymorphic type-expression form", and
"outside that contextual polymorphic-type form, the spelling `forall` is an
ordinary identifier and follows the normal declaration and name-resolution rules
for the position in which it occurs" (docs/language-ref.md, "forall"). `self` is
reserved correspondingly narrowly: in the methods of a `co.lang.class` and in an
`@co.dap.class` method of a target-bound `co.lang.extension`.

Neither spelling was usable as a parameter name, a declaration name or a bare
operand, and `forall` in expression position was force-parsed as a polymorphic
anonymous function whatever followed it.

`atIdentifier` now admits the two contextual spellings, and the anonymous-function
dispatch is gated on the complete `forall(…).(…)->(…){` form. Both
reclassification sites — `parseTypeExpression` for the forall type and
`parsePrimary` for the receiver and the anonymous function — test for their
complete form before any identifier reading is reached, so the contextual
meanings are unchanged.

### Not gaps

Several looser spellings in the consolidated grammar are narrowed by the
reference, and the parser correctly follows the reference: a match chain needs
at least one `.case(...)` arm (reference: "one or more"); a lambda is admissible
only as a direct callback argument of a collection operation; `_` is admissible
only as `each`'s first key/index argument. The built-in metadata registry, the
operator fixity/associativity/arity vocabularies, the reserved-operator set and
the reserved built-in collection names were diffed against the grammar and the
reference and match exactly.

### Known dead code, not fixed here

`sourceClassLibrarySurface` and `sourceClassPackageMetadata` are never produced
by `classifySourceFilename`, so `parseLibrarySurfaceFile`,
`parseLibraryKindAnnotation` and `parsePackageMetadataSourceFile` are
unreachable, as is the `co.lang.library` arm of `classifyCompilationUnitBySyntax`
(the scanner has no such kind token). That agrees with the current reference,
which states that FoLang "defines no `co.lang.package` declaration kind and no
reserved `package.fol` metadata form" and no longer documents `library.fol` as a
source form at all. Removing the code would also touch `src/project`, which
still treats `srclib/<slot>/library.fol` as a real layout element, so the
question of whether that layout is still supported should be settled before the
parser code is deleted.

## Round 1 — 2026-08-10

As of 2026-08-10, the parser has no known syntax-conformance gaps.

- The grammar contains **352 productions**.
- Every production has an implementation trace or an explicit scanner, Pratt,
  filename-classification, contextual-guard, informative-shape, or operator-source
  classification.
- All curated accepted compilation units parse to a non-dummy AST.
- All curated rejected compilation units produce a syntax diagnostic.
- All **205** complete normative reference blocks parse.
- All **18** normative invalid examples are rejected.
- The remaining **109** reference blocks are incomplete or schematic by design;
  the excluded manifest contains no `gap`, `ref-bug`, or `unsorted` entries.

## Fixed during this audit

The expression/type ambiguity in `co.lang.tag(co.lang.string, "Hello")` was
resolved. In expression position, a built-in type followed by `(` is now parsed
as a call target; in type position, the same shape remains a type application.
A dedicated accepted fixture protects this distinction.

Reference examples were also corrected to follow the grammar's mandatory
statement terminators, self-delimiting directive syntax, filename rules,
unit-member declaration forms, and prohibition on nested named kind declarations.
The reference-block corpora were regenerated after those corrections.

## Evidence and limits

The following tests form the executable syntax audit:

```text
go test ./...
go test ./tests/parser -count=1 -run "Test(EBNFConformance|GrammarProductionsHaveImplementationTrace|RefBlocks)"
```

`TestGrammarProductionsHaveImplementationTrace` is an inventory guard, not by
itself behavioral proof. Behavioral evidence comes from complete accepted and
rejected compilation units and the extracted normative reference blocks.

Round 2's three gaps are the standing illustration of that limit: every one of
them sat behind a production the inventory guard already counted as implemented.
`map-literal` had a parse function, `member-suffix` had a lifecycle-name branch,
and `forall-context-guard` had a guard function — the first implemented a rule
the reference had withdrawn, the second admitted a form the reference does not
have, and the third was applied in type position only. An inventory cannot see
any of those. Finding them took writing a compilation unit per construct and
checking the parser's actual accept/reject answer against the reference.

One of the three also shows why a named production is not necessarily a
primary-expression alternative: `map-literal` exists as the body production of
a typed collection but is deliberately absent from `primary-expression`. The
lifecycle audit also found and corrected a consolidation error: lifecycle names
are declaration-only and therefore do not belong in `member-suffix`.

This audit covers parsing and parser-owned structural validation. Type
equivalence, definite initialization, capture resolution, import graphs, and
other semantic requirements require their respective later-phase audits.
