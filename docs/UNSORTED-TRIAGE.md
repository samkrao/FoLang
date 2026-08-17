# Triage of the unsorted reference blocks

## Round 3 — the untyped map literal

One block moved, and one block was newly extracted.

| Block | Line | Category | Resolution |
|---|---:|---|---|
| `L5836/L5836.fol` | 5836 | `ref-bug` | moved `parsing/` → `excluded/` |
| `L11442/some.unit.fol` | 11442 | — | newly extracted into `parsing/`; the Appendix B.1 example parses |

`L5836` is the Expression Evaluation Order example for map entries:

```text
values = {
    firstKey(): firstValue(),
    secondKey(): secondValue()
};
```

It writes an UNTYPED braced map literal, which two normative statements refuse.
"Canonical Object and Collection Construction" says outright that "an untyped
`{ ... }` map literal is not a FoLang value", and Appendix A's consolidated
grammar — normative for syntax — leaves `map-literal` out of
`primary-expression` on purpose, making a braced map body reachable only through
`typed-collection-literal`, behind a type prefix. A bare braced group in
expression position is the block alternative instead.

**Correction the reference needs:** give the example its collection type, which
changes nothing about the evaluation order it is there to demonstrate:

```text
values = co.core.Map{
    firstKey(): firstValue(),
    secondKey(): secondValue()
};
```

The parser was corrected to match the normative rule rather than this example;
see [parser-conformance-audit.md](parser-conformance-audit.md).

Re-extracting the reference-block corpora from `docs/language-ref.md`
(`go run ./cmd/refblocks`) leaves any block the extractor will not classify on its
own: one that neither parses nor matches a block whose classification a person
had already made. This file records how those were resolved.

**Nothing in `docs/language-ref.md` was changed.** The corpora and
`excluded/MANIFEST.tsv` carry the classifications; every correction the
reference needs is listed below for someone to apply.

## Round 2 — after the grammar realignment

The reference rewrite that closed `primary-declaration`, required `=` before a
named function body, and fixed the project layout left **76 blocks unsorted**.
Most of them were not a grammar problem at all.

### The extractor was losing block filenames

46 of the 76 were valid FoLang rejected only because the corpus stored them
under the wrong name. The reference writes a block's filename in a comment
**above** the opening fence:

```text
### Inner Function
//someInnerFun.unit.fol
```folang
_ co.lang.unit = { … }
```
```

The extractor read filename comments only from inside the block, so these were
stored as `L<line>.fol`. FoLang classifies a source file BY ITS NAME, so a unit
body under an ordinary `<Name>.fol` name is parsed as a file-backed primary and
rejected for holding `co.lang.unit` — a naming artifact reported as a grammar
failure.

`cmd/refblocks` now reads the comment run directly above the fence when the
block names no file itself, and promotes an `excluded/` block that parses back
into `parsing/`. That alone moved 46 blocks into `parsing/`.

### Result

| Category | Count | Where it went |
|---|---:|---|
| recovered by reading the filename above the fence | 46 | `parsing/` |
| `invalid` — the block demonstrates a rejected construct | 4 | `invalid/` |
| `ref-bug` — the reference violates a current rule | 3 | `excluded/`, manifest category `ref-bug` |
| `by-design` — not a compilation unit as written | 22 | `excluded/`, manifest category `by-design` |
| `gap` — valid FoLang the parser wrongly rejects | 1 | `excluded/`, manifest category `gap` |

### The 4 moved to `invalid/`

Each carries its own error marker in the reference, so rejection is the point of
the example.

| Block | Line | What it demonstrates |
|---|---:|---|
| `L7605/somebadeg1.unit.fol` | 7605 | a type-level function returning two results, marked `// invalid` |
| `L7657/someIdxEG1.unit.fol` | 7657 | arithmetic and calls in a dependent-type index, marked ❌ |
| `L8024/somGen11.unit.fol` | 8024 | an impredicative call, marked ❌ |
| `L9900/L9900.fol` | 9900 | three object constructions marked `// invalid` |

### The 3 `ref-bug` corrections

| Block | Line | Correction the reference needs |
|---|---:|---|
| `L6634/someAnonymousFun.unit.fol` | 6634 | `add := (…)->(…){…};` binds a unit member with `:=`, which `unit-member` does not admit. Write `add co.lang.function = (…)->(…){…};` — the Function Objects spelling the reference itself documents. |
| `L6670/someOtherAnonymousfun.unit.fol` | 6670 | same correction |
| `L7616/someParameg1.unit.fol` | 7616 | `someAlias(F) co.lang.type = Functor(F);` is declared inside a function body. A named kind declaration cannot be physically nested; move it to the enclosing unit body. |

### The 1 `gap`

`L7520/someruntype1.unit.fol` — see [REFBLOCK-GAPS.md](REFBLOCK-GAPS.md).

### The 22 `by-design`

Prose rather than source, in the same categories as round 1: metasyntax
schematics sharing a fence with a real source file (the condition, loop and
ternary blocks), members shown outside their required container (`@co.dap.generic`
functions written without their `_ co.lang.unit` wrapper), variable declarations
and calls shown directly in a unit body, bare annotation and signature
fragments, and blocks mixing prose paragraphs with code.

One entry is a corpus limitation rather than a documentation one:
`L1997/L1997.fol` is a `srclib/ffi/library.fol` source-library surface, and the
one-folder-per-block corpus layout cannot reproduce the `srclib/<kind>/` path
that tells the parser it is a source library rather than the standalone
`src/library.fol` surface.

## Round 1 — 46 blocks

The first triage, against the previous reference revision, resolved 46 blocks:
6 to `invalid/`, 10 `ref-bug`, 30 `by-design`, and **no parser gaps**. Those
classifications are carried forward by block content and are still in
`excluded/MANIFEST.tsv`.

That round's conclusion is worth repeating, because it held again here: "the
corpus does not parse" is easy to read as "the parser is wrong", and across both
rounds exactly one block of 122 turned out to be a genuine parser gap.
