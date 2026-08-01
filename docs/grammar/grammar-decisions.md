# FoLang Grammar and Semantic Decision Register — Revision 20

- Grammar: `folang-r18.ebnf`
- Grammar SHA-256: `922f79feed2e48e538436c9fd4c613bdd8a67a2f4e5d5ecda98025fb5ea1732e`
- Language reference basis: `language-ref.md`
- Language reference SHA-256: `f9bafbcefdacabaafdc82b250dba5fe229835f6591020ea79640d0f73329f7fd`
- Status: decision-complete grammar and semantic register aligned with the current language reference
- Planned syntax policy: productions described as planned in `language-ref.md` remain in the complete grammar unless explicitly removed. Release-specific availability is handled by the parser/compiler conformance profile.
- Revision 20 makes type/container rules executable: type-first declaration routing includes `dependentType`, `typetype`, and `typekind`; a function-shaped type constructor returns exactly one type-producing kind (or one union of such kinds); supported named type/container declarations retain generic clauses; commas in grouped/type-head forms must introduce another item; and `_` is a contextual call argument only for the first index slot of `each`.

## Termination model

FoLang distinguishes **body braces** from **expression braces** by the enclosing production. The brace character alone does not determine termination.

```folang
emp := Employee{ id: 1, name: "Rao" };  // object construction expression: ; required

classify(n) => {                         // function-pattern body: no ; after }
    this.return "positive";
}

someFArg co.lang.function =              // inline function-kind body: no ; after }
    (a co.lang.int, b co.lang.int)->(co.lang.int) = {
        this.return a + b;
    }

Employee co.lang.struct = {              // UDT body: no ; after }
    id co.lang.int;
    name co.lang.string;
}
```

A comma is a soft end inside an enum or other grouped construct. It closes the current item but not the enclosing statement. Built-in directives are self-delimiting and take no semicolon.

## Physical nesting and scope model

FoLang distinguishes an independently named declaration from a construct that is merely nested syntactically.

```folang
outer()->() = {
    value co.lang.int = 10;

    inner()->() = {                         // named local function: permitted
        co.out.println(value);
    }

    operation := (x co.lang.int)->(co.lang.int) = {
        this.return x * 2;                  // anonymous function expression
    };

    worker := co.lang.class {               // anonymous class expression
        run(x co.lang.int)->(co.lang.int) = {
            this.return operation(x);
        }
    }.init();
}

transformer co.lang.type = forall(T).(T)->(T); // anonymous type expression
```

Independent package-owned classes, structs, cstructs, enums, unions, modules, units, interfaces, signatures, type declarations, instances, matchers, macros, templates, and similar primary declarations cannot be physically nested. Member methods and module/signature type components are members or contract slots rather than independent nested package declarations.

An ordinary local function is physically declared in its enclosing executable block and uses declaration-site lexical scope. `@co.dap.inner` is different: it annotates a separately declared association and executable declarations use call-site lexical-context resolution as defined by `DECISION-SCOPE-002`.

## Package-level function envelope

Ordinary loose functions are forbidden in package source files. The `annotated-function-primary` production exists for annotation-defined primary declaration kinds. Parsing that envelope does not establish legality; semantic analysis must confirm that a resolved annotation explicitly grants primary-declaration status.

## Operator bootstrap and artifact model

FoLang distinguishes existing language-owned operator symbols from genuinely
new custom symbols.

```text
existing FoLang symbol
    -> language-owned token/fixity/precedence/associativity/arity
    -> mode=overload supported only in a legal function owner
    -> mode=override unsupported
    -> ordinary class-method override remains a separate supported feature

new symbol
    -> mode=define only in <operator_library_folder>/operators.fol
    -> one complete callable signature
    -> cannot use a built-in or reserved/future glyph
    -> exported in the owning ordinary library artifact beside its symbol table
    -> duplicate symbol rejected without merge, overload, alias, or remap
```

`operator_library_folder` in `fol-conf.yaml` identifies a project-local
bootstrap source area. Its fixed `operators.fol` surface is parsed with core
FoLang syntax before ordinary application or library source and does not
produce an independent `.folib` or `.folenc`. The configured folder is excluded
from ordinary package discovery.

A library that exports new operators is imported exactly like any other
library. No operator-specific import syntax or special import placement is
introduced. During ordinary direct-library dependency discovery, the frontend
loads the imported artifact's projected symbol table and exported operator
table together. Operators introduced only by transitive dependencies are not
activated automatically.

Existing-operator `mode=overload` is supported, but operator implementations
are normal functions and cannot be declared loose at package scope. Their legal
ownership locations are:

- built-in type: `@co.dap.extension` function inside a unit;
- struct: same-package companion unit;
- class: class operator method;
- module, enum, union, interface, signature, and cstruct: unsupported.

An overload requires an exact operand signature that is not already active.
Operator `mode=override` remains unsupported. Ordinary class-method overriding
through `@co.dap.override` is unaffected.

`~`, `#`, and `^` remain reserved prefix spellings; `#` has no operator
semantics, `^` retains infix XOR, and `@` is not a prefix expression operator.
Every glyph in the reserved-future mathematical/modifier set is unavailable to
`mode=define`.

## Decision index

| Decision | Status | Normative decision |
|---|---|---|
| `DECISION-ANN-001` | Active | Annotation arguments and annotation-map entries accept `=` or `:` as binders. A bare key is a flag. Mixed binders are permitted within one annotation. |
| `DECISION-BACKEND-001` | Active | Each resolved user-defined FoLang identifier is lowered to C++ by appending `_fo`. Built-ins, keywords, and compiler-generated names use separate compiler-defined lowering. |
| `DECISION-BLK-001` | Active | A block may end with one unterminated tail expression. That final expression is the block value and is not an expression statement. |
| `DECISION-COL-001` | Active | Commas separate enum variants, map entries, annotation-map entries, object initializers, parameters, arguments, and other grouped items. A comma is a soft end inside the enclosing construct; a permitted trailing comma does not terminate the enclosing statement. Object and annotation-map fields use `:`. |
| `DECISION-DIR-001` | Active | Built-in directives are self-delimiting. Their complete directive form ends the directive; no trailing semicolon is accepted or required. |
| `DECISION-EXT-001` | Active | The contextual registered precedence table parses new custom operators collected from the configured project-local operator source and exported operator tables of directly imported ordinary libraries. Existing-operator `mode=overload` reuses language-owned syntax and is resolved semantically, so it adds no parser entries; operator `mode=override` is rejected. New symbols declare complete syntax metadata. The alpha profile implements infix, prefix, and postfix; the other reserved fixities are rejected until delimiter/slot grammar is defined. |
| `DECISION-FUN-001` | Active | The `=` before a function block body is optional. Both `f()->T = { ... }` and `f()->T { ... }` are valid. |
| `DECISION-FUN-002` | Active | A named closure uses `name = (parameters) ==>> expression;`. Additional adjacent parameter lists make it curried: `name = (first)(second) ==>> expression;`. |
| `DECISION-GEN-001` | Active | Generic parameters may declare arity, including higher-kinded forms such as `Transformer(F(_), G(_))`. An arity slot is `_` or a named placeholder. Supported named type/container declarations retain their generic parameter clause; library and package-alias declarations do not accept one. |
| `DECISION-KIND-001` | Active | A built-in kind without a dedicated production is parsed by `general-kind-declaration`, which supports block, type-expression, expression, and forward forms. Dedicated declarations and ordinary variable declarations retain priority in their contexts. |
| `DECISION-LEX-001` | Active | Source is UTF-8, but ordinary identifiers use ASCII letters, digits, and isolated internal underscores. An identifier begins with an ASCII letter, cannot contain consecutive underscores, cannot end in an underscore, and has no minimum-length requirement. Lone `_` is contextual, not an identifier. |
| `DECISION-LEX-002` | Active | FoLang supports `//` line comments and non-nesting `/* ... */` block comments. Line breaks are whitespace outside literals. |
| `DECISION-LEX-003` | Active | The lexer uses maximal munch. Reserved multi-character operators and comment introducers are recognized before shorter prefixes. |
| `DECISION-LEX-005` | Withdrawn in revision 9 | The special dot-scanning rule was removed after abbreviated floating forms such as `1.` and `.10` were rejected. Ordinary maximal munch now handles ranges and member access. |
| `DECISION-LEX-006` | Active | After recognizing an identifier, the scanner verifies that the next character is not `_`. This converts trailing or doubled underscores into lexical errors instead of silently splitting them into multiple tokens. |
| `DECISION-LEX-007` | Withdrawn in revision 11 | The apostrophe digit-separator adjacency rule was removed because FoLang no longer supports numeric digit separators. |
| `DECISION-LEX-008` | Active | Adjacent encoding prefixes, raw-string introducers, and backslashes in quoted literals begin reserved post-alpha spellings. The scanner consumes the complete spelling and reports an unsupported feature; separated names remain identifiers. |
| `DECISION-LEX-009` | Active | A special method is one complete spelling from a closed scanner-known set. `@@` is not a prefix operator over an arbitrary identifier: spellings such as `@@new` and `@@init` are classified as whole tokens, and any unrecognized `@@` spelling is a lexical error that reports the admissible set. |
| `DECISION-LIT-000` | Active | FoLang accepts a selected C++-compatible subset of numeric, character, and string literal spellings. The frontend preserves their complete raw lexemes, and a C++ backend may emit those lexemes unchanged. `co.const.true`, `co.const.false`, and `co.const.none` use backend-defined lowering. `nullptr` is not introduced. |
| `DECISION-LIT-001` | Active | Integer literals support C++-compatible binary, leading-zero octal, decimal, hexadecimal, and standard integer suffix forms. Numeric digit separators are not supported. |
| `DECISION-LIT-002` | Active | Floating literals support selected C++-compatible decimal and hexadecimal forms, exponents, and configured suffixes. Digit separators are not supported, and a decimal point requires a digit on both sides. |
| `DECISION-LIT-003` | Active | The alpha release accepts only unprefixed character and string literals without escapes. Prefixes, escapes, universal character names, and raw strings are reserved and rejected with unsupported-feature diagnostics. |
| `DECISION-LIT-004` | Withdrawn in revision 7 | FoLang has no separate user-defined-literal token or C++ `operator""` mechanism. A value of a user-defined type is created through ordinary object construction. |
| `DECISION-LIT-005` | Active | Boolean literals are `co.const.true` and `co.const.false`; none/null is `co.const.none`. Bare `true`, `false`, and `none` are not FoLang literals. |
| `DECISION-LIT-006` | Active | A floating literal has at least one digit on each side of its decimal point. Write `1.0`, `0.10`, and `0x1.8p3`; forms such as `1.`, `.10`, `0x1.p3`, and `0x.8p3` are rejected. |
| `DECISION-LIT-007` | Active | FoLang does not adopt C++14 apostrophe digit separators or underscore digit separators. Numeric digit sequences contain digits only. |
| `DECISION-OP-001` | Active | Built-in operators use the precedence table encoded in the grammar. Runtime assignment has the lowest built-in precedence. |
| `DECISION-OP-002` | Active | Runtime assignments are right-associative, so `a = b = c` groups as `a = (b = c)`. Assignment expressions yield the assigned value; FoLang evaluation-order rules remain separate. |
| `DECISION-OP-003` | Active | `:=` and `?=` are statement-level definition operators, not general expression operators, and cannot be chained. `::=` remains reserved. |
| `DECISION-OP-004` | Active | `++` and `--` are recognized in both prefix and postfix positions. |
| `DECISION-OP-005` | Active | `::=`, `->>`, `<->`, backtick, backslash, and the reserved future mathematical/modifier glyph set are tokenized as reserved operators and rejected until assigned language meaning. They cannot be claimed by `mode=define`. |
| `DECISION-OP-006` | Active | `~`, `#`, and `^` are reserved prefix spellings and are rejected in operand-prefix position. `#` has no semantics and cannot be used, defined, overloaded, or overridden. `^` remains the built-in infix XOR symbol and `~` retains its named-parameter and `->(~)` roles. `@` is not a prefix expression operator. |
| `DECISION-OPLIB-001` | Active | Existing-operator `mode=overload` is supported. Because operator implementations are normal functions, they cannot be declared loose at package scope: built-in operands use an `@co.dap.extension` function inside a unit, structs use their same-package companion unit, and classes use class operator methods. Modules, enums, unions, interfaces, signatures, and cstructs cannot own operator implementations. An overload requires an exact operand signature that is not already active. Operator `mode=override` is unsupported; ordinary class-method overriding through `@co.dap.override` remains a separate supported feature. |
| `DECISION-OPLIB-002` | Withdrawn in revision 17 | The separately distributable `operator` library-kind model was removed. `type="operator"` remains only as the source marker for the configured project-local bootstrap surface and never produces an independently imported artifact. |
| `DECISION-OPLIB-003` | Active | A custom operator symbol has exactly one definition and one complete callable signature in an active project compilation. Any second use of the same symbol—within local operator source, between local and imported metadata, or across directly imported libraries, and regardless of operand types, result type, fixity, precedence, associativity, arity, or implementation—is an immediate compiler error. Custom symbols are not overloadable, aliasable, selectable, mergeable, or remappable. |
| `DECISION-OPLIB-004` | Withdrawn in revision 17 | Operator-specific imports and activation were removed. Ordinary direct library imports now supply operator metadata from the same `.folib`/`.folenc` that supplies the projected symbol table. |
| `DECISION-OPLIB-005` | Withdrawn in revision 17 | The separate operator-library manifest projection was replaced by an exported operator table embedded beside the projected symbol table in an ordinary library artifact. |
| `DECISION-OPLIB-006` | Withdrawn in revision 17 | The direct operator-library import bootstrap was replaced by local operator-source parsing plus ordinary imported-library symbol/operator-table loading. The contextual `extended-operator-expression` grammar hook remains. |
| `DECISION-OPBOOT-001` | Active | `operator_library_folder` in `fol-conf.yaml` identifies the project-local operator source area. A relative path is resolved from the project root. The compiler checks the fixed file `<operator_library_folder>/operators.fol`; absent configuration, folder, or file means no local new operators. The area is excluded from ordinary package discovery. |
| `DECISION-OPBOOT-002` | Active | The configured operator source area is parsed first using only core FoLang syntax. `@co.dap.library(type="operator")` is a source-only bootstrap marker, the source must not use custom operator expressions, and it is compiled into the owning project rather than an independent artifact. |
| `DECISION-OPART-001` | Active | Any ordinary compiled library that introduces new operators stores a complete exported operator table in the same `.folib` or `.folenc` artifact, parallel to its projected symbol table. No separately distributed operator-library artifact exists. |
| `DECISION-OPART-002` | Active | An ordinary direct library import causes the frontend to load the imported artifact's projected symbol table and exported operator table together. Only operators owned by directly imported libraries become active; transitive dependency operators are not activated automatically. |
| `DECISION-OPART-003` | Active | Each exported new-operator entry carries spelling, fixity, precedence, associativity, arity, resolved operand/result identities, generic constraints when present, implementation identity, owner/version, and schema version. Referenced non-built-in types must be present in the owning library's projected API; bodies and private symbols remain hidden. |
| `DECISION-OPBOOT-003` | Active | The frontend reads configuration, parses local operator source, discovers and resolves normal direct library imports through the ordinary import-directive processing rules, loads each imported symbol/operator table, rejects all new-symbol collisions, builds maximal-munch and precedence tables, and only then parses operator-dependent application or library source. No operator-specific import syntax or special import placement is introduced; `extended-operator-expression` is the contextual grammar hook. |
| `DECISION-SCOPE-001` | Active | An ordinary local/inner function has block-local identity and resolves free runtime names from its lexical declaration context. Calling it does not replace that environment with the caller's runtime scope. |
| `DECISION-SCOPE-002` | Active | `@co.dap.inner` is an association annotation on a separately declared declaration, not physical nesting. Executable inner-associated declarations resolve free runtime names through the lexical scope chain of the active attachment/call site, without unrestricted runtime caller-chain lookup. Compile-time names remain statically resolved. |
| `DECISION-SEM-001` | Active | Typeclass instances are selected explicitly by name; FoLang performs no implicit instance search. An instance must be declared in the exact package defining either the typeclass or the represented type. Selection and placement are semantic name-resolution checks. |
| `DECISION-SEM-002` | Active | `@co.ddap.use` explicitly and block-scopingly activates extension-unit or typeclass-instance functions as methods. Receiver-owned declarations take priority, followed by activated extensions and then activated instances. A method name may be activated at most once per receiver type in one scope. |
| `DECISION-SYN-001` | Active | Every simple statement whose production uses `statement-end` requires `;`. Newlines never terminate statements and FoLang performs no semicolon insertion. Built-in directives are exceptions because they are self-delimiting. |
| `DECISION-SYN-002` | Active | Comma-separated variable declarators form one declaration statement and share one final semicolon. |
| `DECISION-SYN-003` | Active | The sole named local-function syntax requires a return-type clause and a block body. This preserves `foo();` as an expression statement. The local function has block-local identity and declaration-site lexical scope. |
| `DECISION-SYN-004` | Active | Annotations may prefix an expression statement. |
| `DECISION-SYN-005` | Active | A standalone block is a statement and takes no trailing semicolon. |
| `DECISION-SYN-006` | Active | Termination depends on syntactic role, not merely on the final character. `;` ends simple, expression-bodied, type-bodied, and forward forms. A body-selected `}` ends UDT/container bodies, function bodies, function-pattern bodies, and standalone block statements. A brace that closes object construction, a map, an anonymous class, or another braced expression does not end the enclosing statement. |
| `DECISION-SYN-007` | Active | Body-versus-expression selection is explicit. Direct body branches use `body-closure-guard`, which rejects an immediately following semicolon. Competing expression branches use `non-block-expression` or `non-anonymous-function-expression`, preventing a body from being reparsed as an expression plus `;`. Grouped, postfixed, or otherwise composed braced forms remain expressions. |
| `DECISION-SYN-008` | Active | Independent named type and container declarations cannot be physically nested. Ordinary named local functions are the explicit named exception. Anonymous functions, lambdas/callback blocks, anonymous classes, ordinary value expressions, and `forall` type expressions may be nested wherever their expression/type-expression grammar permits and create no package-level declaration identity. |
| `DECISION-SYN-009` | Active | `annotated-function-primary` is only a syntactic envelope for annotation-defined primary declaration kinds. An arbitrary annotation does not legalize a loose ordinary function at package-file scope; that legality is checked semantically after annotation resolution. |
| `DECISION-TYP-001` | Active | Every type derivation may carry a trailing attribute list, not only pointer derivations. |
| `DECISION-TYP-002` | Active | A function-shaped type constructor has exactly one type-producing result. Its result may be one of `co.lang.dependentType`, `co.lang.type`, `co.lang.typetype`, `co.lang.typekind`, or `co.lang.kind`, or one union joined by `|`; commas and named/multiple result items are rejected. Its body may bind a type expression, with the type-expression reading taking priority. `co.lang.dependentType`, `co.lang.typetype`, and `co.lang.typekind` are also direct `type-declaration-kind` alternatives and are excluded from general-kind routing. |
| `DECISION-TYP-003` | Active | An array dimension may be elided in any position, including `->([])` and `->([,])`. |
| `DECISION-TYP-004` | Active | A dependent-type argument and an array dimension are index positions, not general expressions. An index is a non-negative integer literal or a name resolving to an in-scope parameter or `@co.dap.const` compile-time constant. Arithmetic, calls, indexing, and other operators are rejected. |
| `DECISION-TYP-005` | Active | Dependent types are equal when their constructors match and indices are pairwise equal. Literal and substituted `@co.dap.const` indices compare by value; parameter indices compare by declaration identity. Equality is not decided modulo arithmetic. |
| `DECISION-TYP-006` | Active | Dependent types are checked against written signatures and are never inferred. FoLang does not infer index values or perform whole-program dependent-type inference. |

## Current lexical examples

```text
Valid identifiers:   a, x, id, name, myVar2, v1_hr, a_b_c
Invalid identifiers: _x, _1, a_, a__b
Contextual token:    _

Valid numbers:       1000, 0b11110000, 0xFFFF0000, 3.141592
Invalid numbers:     1_000, 1'000, 1., .10
```

## Planned syntax retention

Package aliasing, comprehensions, and other planned constructs remain in the complete grammar. Their availability in alpha, 0.x, 1.0, or later compiler profiles is a version-conformance decision, not a reason to delete their productions from the complete grammar.
