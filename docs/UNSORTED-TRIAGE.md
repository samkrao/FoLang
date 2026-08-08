# Triage of the 46 unsorted reference blocks

Re-extracting the reference-block corpora from `docs/language-ref.md`
(`go run ./cmd/refblocks`) left 46 blocks the extractor would not classify on its
own: they neither parse nor match a block whose classification a person had
already made. This is the triage of those 46.

**Nothing in the parser or the reference was changed.** The corpora and
`excluded/MANIFEST.tsv` were reclassified; every correction the reference needs
is listed below for someone to apply.

## Result

| Category | Count | Where it went |
|---|---:|---|
| `invalid` — the block demonstrates a rejected construct | 6 | moved to `testdata/refblocks/invalid/` |
| `ref-bug` — the reference violates a current rule | 10 | `excluded/`, manifest category `ref-bug` |
| `by-design` — not a compilation unit as written | 30 | `excluded/`, manifest category `by-design` |
| `parser-gap` — valid FoLang the parser wrongly rejects | **0** | — |

**No parser gaps were found.** Every block that looked like one turned out to be
either prose, a construct shown outside its container, or the reference
contradicting its own grammar. That is worth stating plainly, because "the
corpus does not parse" is easy to read as "the parser is wrong" and here it was
not.

Each ref-bug was checked both ways before being called one: the block as written
is rejected, and the same block in the form the grammar specifies parses. That
test is what separates a reference error from a parser gap, and it is why the
parser-gap column is empty rather than merely unexamined.

---

## ref-bug — corrections needed in `docs/language-ref.md`

Line numbers are the first line of each fenced block in the current
`language-ref.md`.

### Hyphenated source filenames (3 blocks)

`DECISION-FILE-002` gives a filename component the ordinary ASCII identifier
shape: an ASCII letter first, then letters, digits and *isolated internal
underscores*. A hyphen is not admissible, so these files can derive no
declaration name.

| Line | Written | Should be |
|---:|---|---|
| 1060 | `// string-extension.unit.fol` | `// string_extension.unit.fol` |
| 5562 | `// math-functions.unit.fol` | `// math_functions.unit.fol` |
| 5750 | `// scope-example.unit.fol` | `// scope_example.unit.fol` |

Verified: the hyphenated name is rejected; the underscored name parses.

### Trailing `;` after a self-delimiting directive (2 blocks)

`DECISION-DIR-001` makes a built-in directive self-delimiting: it ends at its
closing argument parenthesis and takes no terminator.

| Line | Written | Should be |
|---:|---|---|
| 1272 | `@co.ddap.use(...);` — 3 occurrences | drop each trailing `;` |
| 1282 | `@co.ddap.use(from="tc.ListFunctor", methods=[map, reduce]);` | drop the trailing `;` |

Note that the reference is inconsistent with itself here: the accepted fixture
corpus and other `@co.ddap.use` examples omit the semicolon.

### Trailing `;` after a body brace (1 block)

`body-closure-guard` forbids a terminator after a body that ends at its brace.

| Line | Written | Should be |
|---:|---|---|
| 7167 | `_ co.lang.struct = {`<br>`    value T;`<br>`};` | drop the `;` after `}` |

### Missing statement terminator (1 block)

`DECISION-SYN-001` makes the statement terminator mandatory.

| Line | Written | Should be |
|---:|---|---|
| 3476 | `@@new()->(co.lang.uninit) = { self.return co.const.none }` | `{ self.return co.const.none; }` |

### ~~Explicit name where `filename-derived-name` is required (2 blocks)~~ — resolved

Blocks 5533 (`delegate-declaration`) and 5676 (`function-object-declaration`)
were triaged as reference errors for naming themselves rather than spelling `_`.
Grammar revision 27 decided the other way: `DECISION-DECL-002` makes a delegate
and a function object **unit members**, which name themselves in their head, so
the reference was right and the grammar was wrong. Both blocks stay in
`excluded/` as `by-design`, because each shows a member without its enclosing
`<Fragment>.unit.fol` file.

The ref-bug count above therefore stands at 8, not 10.

### Object construction syntax (1 block)

`object-construction` is `type "{" [ field { "," field } [ "," ] ] "}"` with
`object-field-initializer = identifier ":" expression`.

| Line | Written | Should be |
|---:|---|---|
| 8080 | `Employee{ address= Address{ city: "ABC"; } }` | `Employee{ address: Address{ city: "ABC" } }` |

Two errors in one expression: `=` where the separator is `:`, and a `;` inside
the braces where initializers are separated by `,`.

---

## ref-bugs inside blocks classified `by-design`

These blocks are excluded for a structural reason — several declarations, or a
member outside its container — so fixing the error below will not make them
parse. They are listed because the reference should still be correct.

| Line | Issue |
|---:|---|
| 5689 | `myobj co.lang.function = …` and `oObj co.lang.function = add;` use explicit names where `_` is required |
| 7298 | `blockormacro co.lang.kind = block | macro` is missing its `;` |
| 7795 | `Name co.lang.string` is missing its `;`; `Employee{ Name = "Kamesh" }` uses `=` where the initializer separator is `:` |
| 8048 | `Address{ city: "Pune"; }` has a `;` inside an object construction; `co.lang.Address` is not a built-in kind |
| 5647 | the nested anonymous function body is missing its `;` |

---

## invalid — moved to the invalid corpus

Each of these exists to demonstrate a construct the compiler rejects, and is
marked ❌ in the reference. They now live in `testdata/refblocks/invalid/`, where
`TestRefBlocksInvalidAreRejected` asserts the rejection.

| Line | Demonstrates |
|---:|---|
| 3105 | a struct declared inside a struct — physical nesting |
| 3421 | a struct declared inside a class — physical nesting |
| 3891 | a struct declared inside a module — physical nesting |
| 6973 | `forall(T)` at declaration level, where `@co.dap.generic` is required |
| 6982 | `forall(T)` on a struct declaration |
| 6994 | `forall(T)` on a Rank-1 generic function |

Blocks 6973, 6982 and 6994 each show the ❌ form beside the ✅ correction. The
file as a whole is still rejected, which is what the corpus asserts, so pairing
them this way loses nothing.

---

## by-design — not a compilation unit as written

Thirty blocks, in four groups.

**Member shown outside its required container (8):** lines 1459, 2443, 5506,
5638, 5647, 6186, and the type-level functions at 6401.

An ordinary function is not a legal entry-file item, so a bare function shown to
illustrate closures, currying or optional parameters cannot parse standalone.
Line 2443 was checked explicitly: the boundary adapter
`health()->(co.lang.bool) =>> health.internal.Service.health();` **parses
correctly** inside `_ co.lang.library = { … }`. It fails only as a bare
fragment.

**Several top-level declarations together (12):** lines 1433, 2479, 5455, 5667,
5689, 6706, 6728, 6734, 6777, 6955, 7298, and the two forward declarations at
5455.

A package source file holds exactly one declaration. A block pairing a type
alias with the function that consumes it is showing a relationship, not a file.

**A declaration beside bare statements (9):** lines 3048, 6094, 6801, 6815,
6828, 7039, 7051, 7795, 8048.

These show a declaration and then use it. The usage belongs in an entry file or
a function body, not beside the declaration.

**Prose and elision (3):** lines 667 (an entry-file pattern group beside its
desugared package-unit form — two different contexts), 6111 (`m45 Matrix(4, 5) =
...;`), and 8102 (a box-drawing tree diagram, not source).

---

## Reproducing this

```
cd frontend
go run ./cmd/refblocks          # report; lists anything still unclassified
go run ./cmd/refblocks -write   # rewrite the corpora
```

Classification is carried across a re-extraction by block **content**, so the
judgements recorded here survive the renumbering that happens whenever
`language-ref.md` is edited. A block whose text changes becomes unclassified
again and is reported by name — which is the intended behaviour: if the
reference changed, the judgement should be made again.

When a ref-bug above is corrected in `language-ref.md`, that block will parse and
the next `-write` moves it into `parsing/` automatically.
