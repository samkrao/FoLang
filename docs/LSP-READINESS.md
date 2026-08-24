# LSP Readiness — FoLang frontend

Status of `src/parser` and `src/scanlex` for use behind a
Language Server Protocol implementation.

This document was first written as a read-only assessment. The blockers it
identified have since been fixed; it now records both what was wrong and what
guarantees the frontend offers, with the test that proves each one.

---

## Status

| Capability | State | Proof |
|---|---|---|
| Error recovery inside a parse | **Working** | `TestParseFileRecoversTheWellFormedMembers` |
| Partial tree delivered to the caller | **Working** | `TestParseFileSurvivesSyntaxErrors` |
| Survives lexical errors | **Working** | `TestParseFileSurvivesLexicalErrors` |
| Survives a malformed declaration head | **Working** | `TestParseFileSurvivesAMalformedDeclarationHead` |
| Source ranges on AST nodes | **Working** | `TestEveryNodeCarriesASpan` |
| Usable over stdio | **Working** | `TestParseFileWritesNothingToStdout` |
| Deterministic across re-parses | **Working** | `TestParseIsDeterministic` |
| Bounded diagnostics | **Working** | `TestDiagnosticsAreCapped` |
| Concurrent parses | **Working, race-tested** | `concurrency_test.go` |
| Incremental reparse | **Not implemented, not recommended** | — |

The batch compiler's behaviour is deliberately unchanged. `parser.Parse` still
stops the process at the first diagnostic; the non-fatal path is a separate
entry point.

---

## Using the frontend from a language server

```go
result := parser.ParseFile(source, name, dir, basename, packagePath)

result.Root         // the tree, partial when the file is malformed
result.Tokens       // always complete, including for lexical errors
result.Diagnostics  // lexical then syntactic, each carrying a source range
result.Truncated    // true when MaxParseErrors was reached
```

`ParseFile` never calls `os.Exit` and never panics on malformed input. A panic
that is not the parser's own recovery sentinel is still a bug and is left to
propagate deliberately — a server should recover at its request boundary, not
have the parser swallow it.

For a project with custom operators, load the catalog once and pass it, or a
registered symbol scans as an unknown run:

```go
operators, findings := parser.LoadOperators(projectRoot)
result := parser.ParseFileWithOperators(source, name, dir, basename, pkg, operators)
```

Node ranges come from the embedded span:

```go
span := node.GetSpan()          // ast.Span{Start, End helpers.Position}
span.Contains(line, column)     // the primitive navigation is built from
```

Walking the tree and keeping the innermost node whose span contains the cursor
yields the node to hover, rename or navigate from.

---

## What was wrong, and what changed

### 1. The partial tree never reached the caller

`parseIntoConfigured` called `foerrors.HandleErrors` after parsing, which calls
`os.Exit(1)`. Recovery worked and a partial tree was built — and was then
discarded by the exit. There was no mode that returned a tree plus diagnostics.

Parsing is now split. `parseCollecting` does the work and never reports;
`parseIntoConfigured` wraps it and applies the fatal policy for the batch
compiler. `ParseFile` is the same core with no fatal step.

### 2. The lexer was fatal too

Five call sites reported through `HandleErrors`, so one bad byte ended the
process before the parser ran — no tokens, no tree, no highlighting. A comment
in `tokenizer.go` described a non-fatal path whose `continue` was unreachable as
written.

The boolean `quiet` flag is now a `diagnosticSink` with three policies: nil is
fatal (batch, unchanged), discard survives silently (the project surface scan),
collect survives and retains. `TokenizeCollecting` is the new entry point. An
unclassifiable byte is reported and then consumed, so scanning always terminates
with a complete stream.

A malformed identifier is now still **emitted** as a token rather than dropped.
For a batch compile the difference is invisible; for an editor an identifier
ending in an underscore is a name the user is halfway through typing, and
dropping it would cost the whole enclosing declaration.

### 3. Recovery was disproportionate

This was the most damaging defect and was not in the original assessment — it
surfaced only when the contract tests were written.

`synchronize` stopped only at `;` and `}`. A garbled line with no `;` of its own
ran on to the next terminator, and because a brace group is skipped whole on the
way, the next well-formed declaration was swallowed with it:

```folang
first()->(co.lang.int) = { this.return 1; }
&&& broken &&&
second()->(co.lang.int) = { this.return 2; }
```

The first `;` at that level is inside `second`'s body, so `second` disappeared
entirely and the file reported **one** diagnostic. That is the ordinary state of
a buffer being typed into, so an editor lost its outline on almost every
keystroke.

Recovery now also stops at a line boundary, at the first token that could begin a
new construct. The broken line costs itself and nothing more. Measured on the
example above: `second` survives; on a 200-line garbage file the diagnostic count
went from 1 to the 50-diagnostic cap.

The cost is that a construct deliberately spread over several lines may now
report more than one diagnostic. That is the right trade for an editor, and the
cap bounds it.

### 4. A malformed declaration head discarded the file

`parsePackageSourceFile` and `parseLibrarySurfaceFile` called their declaration
parser with no recovery point, so a bailout in the head unwound to
`parseTopLevel`, which returns `ast.DummyStmt` — the whole tree. The head is
exactly what is incomplete while the user types it, and the body below is often
already well formed. Both now recover at the declaration.

### 5. The frontend wrote to stdout

395 print calls in parser and scanlex, of which **129** were unconditional
per-function entry traces — not behind the existing `traceEnabled` build tag.
Anything on stdout corrupts a JSON-RPC stream and disconnects the client.

All 129 were removed with an AST-aware rewriter. `HandleErrors` now writes
diagnostics to stderr. The dead `WhoCalledMe` debug helper was deleted rather
than gated: a library a server embeds should not hold a stdout writer at all.

What remains on stdout is CLI output only — the driver's `--stopAt Tokens` dump,
usage text, and opt-in `Print()` debug methods with no callers in the parse path.
`TestParseFileWritesNothingToStdout` captures the real file descriptor and
asserts the parse path emits nothing.

### 6. No AST node carried a position

No node in `src/ast` referenced `helpers.Position`; positions existed only on
tokens and diagnostics. Once parsing finished the tree could not say where
anything came from, which blocks go-to-definition, hover, document symbols,
rename, folding and selection ranges.

`ast.Span` is now embedded in all 92 node types and populated at every
construction site.

The span is a **field**, not an interface method. Node types are used as values
and satisfy `Stmt`/`Expr`/`Type` with value receivers, so a pointer-receiver
setter would stop those values implementing the interfaces at all.

Three kinds of site needed different treatment:

- **Parse sites** measure from the enclosing function's entry cursor.
- **Derived nodes** — a derived type over its element, a synthesized parameter
  standing for its type — inherit the span of what they wrap.
- **Lowering** runs *after* parsing, when the cursor is at EOF. Every node it
  builds takes its span from the expression it replaces, carried on `chain.span`.
  Measuring from the cursor would have given every rewritten node the end of the
  file.

### 7. Symbol identity was random

`NewContextId` and `NewSymbolTableId` appended a random suffix, so two parses of
unchanged source produced different identifiers — defeating any diff between
parses or cache keyed on symbol identity. The suffix is gone; the counter alone
is unique within a process, which is all these per-parse scope handles ever
required. `helpers.ResetIdCounters` makes exact reproducibility available.

### 8. The diagnostic cap was never enforced

`MaxParseErrors` was referenced only by the message that printed it; `p.diags`
was an unbounded append. And the message claimed *"too many errors (50)"* on a
single typo. Both fixed: `record()` enforces the cap and sets `Truncated`, and
the message reports the real count.

---

## Incremental reparse

Still not implemented, and still not recommended. Full-file reparse is the
standard first implementation for a language server and is adequate here.

The preconditions are now in better shape than they were — nodes carry spans,
parses are deterministic, cursor rewind is side-effect-free, and parser state is
confined to one struct — so it remains available later. It should be driven by a
measured latency problem, not added speculatively.

---

## Concurrency

Concurrent parses are supported and race-tested. A server may parse several open
files at once, re-parse a buffer while an earlier parse of it is still running,
and read a cached tree from request handlers in parallel.

**What was verified.** `tests/parser/concurrency_test.go` runs 32 goroutines
released from a barrier so the calls genuinely overlap, across five patterns:
distinct files, the same file, malformed files (contending on the diagnostic and
recovery paths), tokenization mixed with parsing, and concurrent reads of one
finished tree. The full suite was then run as

```
CGO_ENABLED=1 go test -race -count=3 -cpu=1,4,8 ./...
```

— nine schedulings, no data race and no failure.

**The harness was validated, not assumed.** A race detector only reports what it
observes, so a deliberate unsynchronised global write was temporarily added to
the parse path; `-race` flagged it immediately, and it was removed. Without that
check the passing result would only have meant "nothing was looked at".

**What makes it safe.** Every parse builds its own `parser`, its own symbol
records and its own token stream. The id counters are `atomic.Int64`. The
scanner's tables are package-level but written only at initialisation. The one
object deliberately shared between parses — the operator catalog behind
`parser.Operators` — is built once and only read, which
`TestConcurrentParsesShareOneOperatorCatalog` exercises directly.

**The remaining hazard is `foerrors.GenPanic`**, a process-global that selects
between exiting and panicking. It is meaningful only to the batch compiler.
`ParseFile` never reaches the code that reads it, which
`TestParseFileNeverReachesTheFatalPath` proves by restoring the production
default around a malformed parse: had the fatal path been reachable, the test
binary would have exited. An embedding consumer should never write it.

`helpers.ResetIdCounters` is likewise not safe to call while a parse is in
flight, and is deliberately explicit rather than automatic for that reason.

Running `-race` needs `CGO_ENABLED=1` and a C toolchain; it is off by default in
this environment.

---

## Remaining caveats

- **Semantic phases do not exist yet.** Name resolution and type checking are
  future work, so features needing them — accurate go-to-definition across
  files, type-aware completion — are not reachable from the frontend alone.
  What the frontend now supports is everything derivable from one file's syntax.

---

## Test inventory

| File | Covers |
|---|---|
| `tests/parser/lsp_contract_test.go` | non-fatal parsing, recovery quality, stdout silence, determinism, diagnostic cap and ranges, fatal-path unreachability |
| `tests/parser/span_coverage_test.go` | every node in every accepted fixture carries a well-formed span; `Contains` locates the innermost node |
| `tests/parser/concurrency_test.go` | concurrent parses of distinct and identical files, malformed files, mixed tokenize/parse, a shared operator catalog, and concurrent tree reads |

`span_coverage_test.go` walks with reflection rather than the AST visitor on
purpose: the visitor only descends the shapes it knows, so a node type it does
not handle would be skipped and a missing span never noticed.
