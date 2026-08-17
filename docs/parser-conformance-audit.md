# Parser conformance audit

This audit compares the normative `docs/language-ref.md`, the consolidated
`docs/grammar/folang.ebnf`, and the parser under `frontend/src/parser`.

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
