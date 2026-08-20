# Parser conformance audit

This audit compares the normative `docs/language-ref.md`, the consolidated
`docs/grammar/folang.ebnf`, and the parser under `frontend/src/parser`.

## Round 4 — 2026-08-20, negative and corner cases

Round 3 probed whether each construct is admitted. This round probed the
boundaries of each construct instead: the malformed spellings next to a legal
one, the repetition of something admitted once, the empty and doubled forms of
every delimited list, and the shapes two rules compete for. Four defects came out
of it, and the rejected corpus gained 34 fixtures.

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
```

The scanner already declined this span deliberately — `labelIdentifierLength`
rejects a trailing apostrophe with the comment that reporting it as an
unterminated label "would name the wrong construct" — but nothing then named the
right one. It now does, after both the character-literal and label rules have
declined, and only for a span that closes on its own line. A backslash still
falls through to the escape rules, whose unsupported-feature error is more
specific.

### Corpus — 34 rejected fixtures and 2 accepted ones

Every fixture states the diagnostic its FIRST finding must contain, and each was
checked to die on its own rule rather than somewhere earlier.

```text
literals      integer-suffix-repeated, integer-suffix-length-repeated,
              octal-literal-invalid-digit, character-literal-two-characters,
              unterminated-block-comment, unknown-symbolic-run
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

All pass. The rejected corpus is now 172 fixtures plus EXPECTATIONS.tsv, and the
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
