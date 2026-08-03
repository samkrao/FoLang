# Reference block gaps

Generated. Do not hand-edit; regenerate from `docs/language-ref.md` and the
corpora under `frontend/testdata/refblocks/`.

Every ` ```folang ` block in `docs/language-ref.md` is sorted into one of three
corpora:

| Corpus | Blocks | Contract |
|---|---:|---|
| `frontend/testdata/refblocks/parsing/` | 144 | all must parse |
| `frontend/testdata/refblocks/invalid/` | 30 | all must be rejected |
| `frontend/testdata/refblocks/excluded/` | 145 | see `MANIFEST.tsv` |

This file lists the subset of `excluded/` that `MANIFEST.tsv` marks `gap`:
blocks that are **not** error-marked, **not** elided, and **not** fragments,
but that the parser still rejects. Each is a divergence between a reference
example and the grammar. The other 100 excluded blocks are prose — elided
bodies, metasyntax, diagrams, tables, and members shown outside their container
— and are not parseable as written by design.

The production column names the construct the parser was completing when it
stopped, taken from its own diagnostic.

## Summary

45 blocks. Every one is a statement-terminator divergence; no other kind of
construct in the reference fails to parse.

| Cause | Blocks |
|---|---:|
| Example omits the mandatory `;` | 44 |
| Example adds a `;` that a self-delimiting directive forbids | 1 |

| Production | Blocks |
|---|---:|
| `expression-statement` | 23 |
| `variable-declaration` | 7 |
| `type-declaration` | 7 |
| `field-declaration` | 4 |
| `inferred-variable-declaration` | 3 |
| `use-directive` | 1 |

### Example omits the mandatory `;` — 44 blocks

DECISION-SYN-001 makes a semicolon mandatory after every simple statement, and
DECISION-SYN-006 restates it: `;` ends a simple statement, an expression-bodied
declaration, and every forward declaration. A newline never terminates
anything, and there is no semicolon insertion.

These blocks drop the terminator, usually on a snippet's last line or between
two declarations shown in sequence. The grammar is unambiguous here, so the
examples are what need correcting.

### Example adds a `;` a self-delimiting directive forbids — 1 block

The inverse case, at line 1206. DECISION-DIR-001 makes built-in directives
self-delimiting: a directive ends at its complete form and takes no semicolon.
Both `@co.ddap.use(...)` lines in that block carry a trailing `;`. Every other
directive example in the reference is already correct.

## Blocks

Line numbers are the first line of the block's content in
`docs/language-ref.md`.

| Ref line | Production | Why it is rejected |
|---|---|---|
| 144 | `expression-statement` | missing `;` at end of an expression statement |
| 151 | `expression-statement` | missing `;` at end of an expression statement |
| 450 | `variable-declaration` | expected `;` after a variable declaration, found the name `z` |
| 498 | `variable-declaration` | missing `;` at end of a variable declaration |
| 571 | `inferred-variable-declaration` | expected `;` after an inferred variable declaration, found the name `result` |
| 986 | `inferred-variable-declaration` | expected `;` after an inferred variable declaration, found the name `result` |
| 1002 | `expression-statement` | expected `;` after an expression statement, found `this` |
| 1206 | `use-directive` | unexpected `;` after @co.ddap.use; a built-in directive is self-delimiting and takes no terminator |
| 1504 | `expression-statement` | expected `;` after an expression statement, found `@co.ddap.import` |
| 1963 | `expression-statement` | expected `;` after an expression statement, found `co.in` |
| 1977 | `expression-statement` | expected `;` after an expression statement, found the name `list` |
| 2403 | `variable-declaration` | expected `;` after a variable declaration, found `=>` |
| 3147 | `variable-declaration` | expected `;` after a variable declaration, found the name `length` |
| 3155 | `expression-statement` | missing `;` at end of an expression statement |
| 3529 | `field-declaration` | expected `;` after a field declaration, found the name `name` |
| 3576 | `field-declaration` | expected `;` after a field declaration, found `}` |
| 4772 | `expression-statement` | missing `;` at end of an expression statement |
| 4780 | `expression-statement` | missing `;` at end of an expression statement |
| 4788 | `expression-statement` | missing `;` at end of an expression statement |
| 4804 | `expression-statement` | missing `;` at end of an expression statement |
| 4816 | `expression-statement` | missing `;` at end of an expression statement |
| 5555 | `variable-declaration` | expected `;` after a variable declaration, found `=>>` |
| 5609 | `expression-statement` | expected `;` after an expression statement, found the name `words` |
| 5682 | `type-declaration` | expected `;` after a type declaration, found the name `someFRet` |
| 5854 | `expression-statement` | missing `;` at end of an expression statement |
| 5878 | `expression-statement` | expected `;` after an expression statement, found the name `arr` |
| 6182 | `variable-declaration` | missing `;` at end of a variable declaration |
| 6599 | `variable-declaration` | expected `;` after a variable declaration, found the name `dotProduct` |
| 6693 | `type-declaration` | expected `;` after a type declaration, found the name `someFunction` |
| 6714 | `type-declaration` | expected `;` after a type declaration, found the name `someFunction` |
| 6720 | `type-declaration` | expected `;` after a type declaration, found the name `someFunction` |
| 6763 | `type-declaration` | expected `;` after a type declaration, found the name `rank3ArgType` |
| 6801 | `type-declaration` | expected `;` after a type declaration, found the name `box` |
| 6862 | `field-declaration` | expected `;` after a field declaration, found the name `next` |
| 6939 | `type-declaration` | expected `;` after a type declaration, found the name `someFunction` |
| 7040 | `expression-statement` | expected `;` after an expression statement, found the name `println` |
| 7523 | `inferred-variable-declaration` | missing `;` at end of an inferred variable declaration |
| 7533 | `field-declaration` | expected `;` after a field declaration, found `}` |
| 7563 | `expression-statement` | missing `;` at end of an expression statement |
| 7587 | `expression-statement` | expected `;` after an expression statement, found the name `emp.name` |
| 7633 | `expression-statement` | missing `;` at end of an expression statement |
| 7740 | `expression-statement` | missing `;` at end of an expression statement |
| 7833 | `expression-statement` | missing `;` at end of an expression statement |
| 7861 | `expression-statement` | missing `;` at end of an expression statement |
| 7890 | `expression-statement` | missing `;` at end of an expression statement |
