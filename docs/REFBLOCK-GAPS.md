# Reference block gaps

Maintained alongside the corpora under `frontend/testdata/refblocks/`. Refresh the
counts after `go run ./cmd/refblocks -write` changes them.

Every ` ```folang ` block in `docs/language-ref.md` is sorted into one of three
corpora. All 332 blocks are accounted for; `invalid/` additionally holds
hand-written cases for rules no reference example exercises.

| Corpus | Files | of which doc blocks | Contract |
|---|---:|---:|---|
| `frontend/testdata/refblocks/parsing/` | 182 | 182 | all must parse |
| `frontend/testdata/refblocks/invalid/` | 31 | 18 | all must be rejected |
| `frontend/testdata/refblocks/excluded/` | 132 | 132 | see `MANIFEST.tsv` |

The 13 hand-written files in `invalid/` are named for the rule they exercise
rather than a line number: operand-boundary violations, symbolic runs the
lexer must not split, dependent-index arithmetic and negation, and a body brace
followed by `;`. `TestRefBlocksInvalidAreRejected` asserts the whole corpus.

This file lists the subset of `excluded/` that `MANIFEST.tsv` marks `gap`:
blocks that are **not** error-marked, **not** elided, and **not** fragments,
but that the parser still rejects. Each is a divergence between a reference
example and the grammar. The other 118 excluded blocks are prose — elided
bodies, metasyntax, diagrams, tables, and members shown outside their container
— and are not parseable as written by design.

## How a block gets its name

A block is stored in a folder `L<line>/` under the filename the reference gives
it, which the extractor reads from a comment either inside the block or on the
line above its opening fence. That name is not decoration: FoLang classifies a
source file BY ITS NAME, so a unit body stored as `L6711.fol` is parsed as a
file-backed primary and rejected for holding `co.lang.unit`. Recovering the
name from above the fence is what moved 46 blocks out of `excluded/` and into
`parsing/`; a block whose name cannot be recovered is still read as an ordinary
`<Name>.fol` primary.

## Summary

14 blocks.

| Cause | Blocks |
|---|---:|
| Example omits the mandatory `;` | 13 |
| Value-constructor call read as a dependent-type argument list | 1 |

| Production | Blocks |
|---|---:|
| `expression-statement` | 11 |
| `inferred-variable-declaration` | 1 |
| `dependent-type-argument` | 1 |
| `variable-declaration` | 1 |

### Example omits the mandatory `;` — 13 blocks

`statement-end` is a mandatory `;` after every simple statement, an
expression-bodied declaration, and every forward declaration. A newline never
terminates anything, and there is no semicolon insertion.

These blocks drop the terminator, usually on a snippet's last line or between
two declarations shown in sequence. The grammar is unambiguous here, so the
examples are what need correcting.

| Block | Reference line | Diagnostic |
|---|---:|---|
| `L5335/L5335.fol` | 5335 | missing `;` at end of an expression statement |
| `L5343/L5343.fol` | 5343 | missing `;` at end of an expression statement |
| `L5351/L5351.fol` | 5351 | missing `;` at end of an expression statement |
| `L5367/L5367.fol` | 5367 | missing `;` at end of an expression statement |
| `L5379/L5379.fol` | 5379 | missing `;` at end of an expression statement |
| `L9053/L9053.fol` | 9053 | missing `;` at end of an inferred variable declaration |
| `L9094/L9094.fol` | 9094 | missing `;` at end of an expression statement |
| `L9118/L9118.fol` | 9118 | expected `;` after an expression statement, found `emp.name` |
| `L9164/L9164.fol` | 9164 | missing `;` at end of an expression statement |
| `L9298/L9298.fol` | 9298 | missing `;` at end of an expression statement |
| `L9392/L9392.fol` | 9392 | missing `;` at end of an expression statement |
| `L9420/L9420.fol` | 9420 | missing `;` at end of an expression statement |
| `L9449/L9449.fol` | 9449 | missing `;` at end of an expression statement |

### Value-constructor call read as a dependent-type argument list — 1 block

`L7520/someruntype1.unit.fol`, reference line 7520. The block builds a tagged
value in expression position:

```folang
(value < 100)
    .then(co.lang.tag(co.lang.string, "Hello"))
```

The parser reads `co.lang.tag(…)` as a dependent-type application, whose
arguments `dependent-type-argument` restricts to an integer literal or a name,
and so rejects the string. In EXPRESSION position the same tokens are an
ordinary call. This is the one entry here that looks like a parser gap rather
than a reference correction: the built-in type name is being resolved to the
type reading in a context where only the expression reading applies.
