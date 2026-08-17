## Shell commands
- Commands run from the repo root. Do not prepend `cd`.
- Use relative paths: `./tests/parser/`, not absolute Windows paths.

# FoLang parser

## Spec — normative, do not modify to match code
- Reference: docs/language-ref.md
- Grammar: docs/grammar/folang.ebnf
- Validation report: docs/grammar/folang-conformance-validation.json

If the code disagrees with the grammar, the code is wrong.
The reference is normative and the grammar is a syntactic consolidation of it, so
where the two disagree the reference governs. Inside the reference, Appendix C
governs over an older example elsewhere in the document.

## Testing
- `go test ./...` after every change
## Corpora
Paths are relative to `frontend/`.
- tests/parser/examples/accepted/  curated, all must parse
- tests/parser/examples/rejected/  curated, all must be rejected
- tests/parser/examples/operator-source/  operator bootstrap surfaces
- testdata/refblocks/parsing/      doc blocks that parse
- testdata/refblocks/invalid/      doc blocks that must be rejected
- testdata/refblocks/excluded/     not parseable as written, or a tracked gap
                                   (MANIFEST.tsv gives a reason per block;
                                   rows marked `gap` are docs/REFBLOCK-GAPS.md,
                                   rows marked `unsorted` need classifying)

### Rejected fixtures state the diagnostic they expect
Each rejected corpus has an `EXPECTATIONS.tsv` pairing a fixture with text its
FIRST diagnostic must contain:
- tests/parser/examples/rejected/EXPECTATIONS.tsv        (all fixtures)
- tests/parser/examples/operator-source/rejected/EXPECTATIONS.tsv
- testdata/refblocks/invalid/EXPECTATIONS.tsv            (hand-written half only;
  the L<line>/ blocks renumber on doc edits, so they get a filename-artifact
  guard instead)

Adding a fixture means adding its row; the harness fails on a fixture with no
row and on a row with no fixture. Asserting only that a fixture fails lets it
pass while dying somewhere else entirely.

A rejected fixture whose rule needs a particular source-file classification lives
in a folder, `<case>/<Name>.fol`, holding one file under the name FoLang requires
— `Employee.fol` for a struct rule, `Employee.comp.unit.fol` for a companion
unit, a `.unit.fol` for anything about a block. A flat `<case>.fol` is parsed
under its own hyphenated name, which is not a filename-identifier and so
classifies as an application entry file. Getting this wrong stops the parse at
the filename rules before it reaches the rule under test.

### refblocks — generated, do not hand-edit
Extracted from every ```folang block in docs/language-ref.md.
Regenerate, from `frontend/`:
    go run ./cmd/refblocks          report what would change
    go run ./cmd/refblocks -write   rewrite the corpora

Each block is a folder `L<line>/` named after the line it opens on, holding one
file under the name the reference gives it (`// Employee.fol`). The folder
exists so a block can keep a reserved exact name such as `package.fol`,
`appl.fol` or `library.fol`; FoLang classifies a source file BY ITS NAME, so
parsing a block under a synthesized name misclassifies it.

Classification is carried across a re-extraction by block CONTENT, not by
filename, because editing the reference renumbers every block below the edit.
A block that neither parses nor matches a known one is reported for a person to
classify rather than guessed into a bucket.

Hand-written fixtures for rules no reference block covers sit flat beside the
`L<line>/` folders and are left untouched by regeneration.

## Generated files — never hand-edit
docs/trace.json, docs/callgraph.json, docs/grammar-map.json
Regenerate, from `frontend/`: `go run -tags partrace ./cmd/docgen`

trace.json needs the tag on docgen itself, because the spans come from the
parser instrumentation docgen links in. Without it docgen still writes
callgraph.json and grammar-map.json and leaves trace.json alone.

Snippets and call edges in docs/api/ come from those files only.