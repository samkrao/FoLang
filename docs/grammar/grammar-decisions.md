# FoLang Grammar and Semantic Decision Register — Revision 24

- Grammar: `folang.ebnf`
- Grammar SHA-256: `12e1673fb7d2624ef4dad405a6859a1dd42952ec2d092db2ebcdb284469c30fc`
- Language reference basis: `../language-ref.md`
- Language reference SHA-256: `6aac6a6e782197c0da5da206209de695443434ca3a42dd6cf18ec014df638ea7`
- Status: decision-complete grammar and semantic register aligned with the current language reference
- Planned syntax policy: productions described as planned in `../language-ref.md` remain in the complete grammar unless explicitly removed. Release-specific availability is handled by the parser/compiler conformance profile.
- Revision 24 adds the whole-symbol-run and operator-boundary model. A contiguous symbolic run is preserved as one candidate and is never split into shorter operators as a fallback. Grammar context distinguishes structural spellings, metadata spellings, and registered expression operators. Every multi-symbol expression operator requires explicit operand-facing boundaries. `++` and `--` are removed from the built-in prefix/postfix grammar; when unregistered they fail through the general symbolic-run rule.

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

FoLang separates operator registration from operator implementation.

```text
language-owned built-in or pre-declared symbol
    -> symbol and parse properties already registered by the language

project-local custom symbol
    -> registered once inside the fixed operators.fol operator library
    -> carries parse and optional optimization metadata

all registered symbols
    -> implementations use mode=overload in a legal function owner
    -> multiple distinct normalized operand signatures are permitted
    -> mode=override and mode=extends are unsupported
```

`operator_library_folder` in `fol-conf.yaml` identifies a source-only bootstrap
area excluded from package discovery. Its fixed `operators.fol` is parsed by a
dedicated lexer/parser and must have this outer shape:

```folang
@co.dap.library(type=operator)
_ co.lang.library = {
    // co.lang.operator declarations only
}
```

The operator library is not imported and produces no artifact. Operators do not
cross ordinary library boundaries. Both lexers preserve one complete contiguous
symbol run and never fall back to shorter operators. Grammar context classifies
the whole spelling as structural syntax, contextual metadata, a registered
expression operator, or an unrecognized token. A multi-symbol expression
operator requires explicit boundaries on every operand-facing side, so a
registered `+-` is written `a +- b`; `a + -b` denotes separate operators.

Operator implementations are normal functions and cannot be loose at package
scope:

- built-in operand type: `@co.dap.extension` function inside a unit;
- struct: same-package companion unit;
- class: class operator method;
- module, enum, union, interface, signature, and cstruct: unsupported.

A duplicate custom symbol declaration is an error. A duplicate normalized
implementation signature is also an error, but one registered custom symbol may
have multiple distinct overload signatures. Using one symbol consistently for
one concept is recommended for readability and is not compiler-enforced.

Pre-declared mathematical/modifier glyphs are language-owned registrations with
no required built-in implementation. They cannot be declared locally, but they
can be implemented through `mode=overload` exactly like `+` or `*`.

## Decision index

| Decision | Status | Normative decision |
|---|---|---|
| `DECISION-ANN-001` | Active | Annotation arguments and annotation-map entries accept `=` or `:` as binders. A bare key is a flag. Mixed binders are permitted within one annotation. |
| `DECISION-BACKEND-001` | Active | Each resolved user-defined FoLang identifier is lowered to C++ by appending `_fo`. Built-ins, keywords, and compiler-generated names use separate compiler-defined lowering. |
| `DECISION-BLK-001` | Active | A block may end with one unterminated tail expression. That final expression is the block value and is not an expression statement. |
| `DECISION-COL-001` | Active | Commas separate enum variants, map entries, annotation-map entries, object initializers, parameters, arguments, and other grouped items. A comma is a soft end inside the enclosing construct; a permitted trailing comma does not terminate the enclosing statement. Object and annotation-map fields use `:`. |
| `DECISION-DIR-001` | Active | Built-in directives are self-delimiting. Their complete directive form ends the directive; no trailing semicolon is accepted or required. |
| `DECISION-EXT-001` | Active | The contextual registered precedence table is built from the language-owned operator registrations plus the current compilation's custom declarations from the configured operator library. Built-in, pre-declared, and custom symbols all receive implementations through `mode=overload`; omitted mode defaults to overload. `mode=override`, `mode=extends`, `mode=define`, and other explicit modes are rejected. A custom declaration supplies complete parse metadata. The alpha profile implements infix, prefix, and postfix; other fixities remain reserved until their delimiter/slot grammars are defined. |
| `DECISION-FUN-001` | Active | The `=` before a function block body is optional. Both `f()->T = { ... }` and `f()->T { ... }` are valid. |
| `DECISION-FUN-002` | Active | A named closure uses `name = (parameters) ==>> expression;`. Additional adjacent parameter lists make it curried: `name = (first)(second) ==>> expression;`. |
| `DECISION-GEN-001` | Active | Generic parameters may declare arity, including higher-kinded forms such as `Transformer(F(_), G(_))`. An arity slot is `_` or a named placeholder. Supported named type/container declarations retain their complete generic parameter clause; application-entry declarations, library declarations, and package aliases do not accept one. |
| `DECISION-KIND-001` | Active | A built-in kind without a dedicated production is parsed by `general-kind-declaration`, which supports block, type-expression, expression, and forward forms. Dedicated declarations and ordinary variable declarations retain priority in their contexts. |
| `DECISION-LEX-001` | Active | Source is UTF-8, but ordinary identifiers use ASCII letters, digits, and isolated internal underscores. An identifier begins with an ASCII letter, cannot contain consecutive underscores, cannot end in an underscore, and has no minimum-length requirement. Lone `_` is contextual, not an identifier. |
| `DECISION-LEX-002` | Active | FoLang supports `//` line comments and non-nesting `/* ... */` block comments. Line breaks are whitespace outside literals. |
| `DECISION-LEX-003` | Active | After comments, literals, and closed scanner-known composite spellings such as `@@new` are recognized, the lexer consumes each remaining complete maximal contiguous run of symbol characters as one candidate. The whole run is classified by grammar context as a fixed structural spelling, contextual metadata spelling, registered expression operator, or unrecognized symbolic token. It is never split into shorter operators as a fallback. Comment openers are recognized before symbolic-run scanning. |
| `DECISION-LEX-010` | Active | A registered expression operator containing more than one symbol character requires an explicit boundary on every operand-facing side. Whitespace, comments, and applicable delimiters supply boundaries, and boundary presence is checked before separators are discarded. Infix operators require both sides, prefix operators the operand side after the symbol, and postfix operators the operand side before it. Structural and metadata spellings are exempt unless parsed as expression operators. |
| `DECISION-LEX-005` | Withdrawn in revision 9 | The special numeric dot-scanning rule was removed after abbreviated floating forms such as `1.` and `.10` were rejected. Numeric-literal recognition and the current whole-symbol-run classifier now distinguish decimal points, range spellings, and member access without that rule. |
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
| `DECISION-OP-003` | Active | `:=` and `?=` are statement-level definition operators, not general expression operators, and cannot be chained. Each requires an explicit boundary on both the name-facing and value-facing side. `::=` remains reserved. |
| `DECISION-OP-004` | Withdrawn in revision 24 | Earlier drafts recognized `++` and `--` as built-in prefix and postfix operators. They are removed from the operator inventory, precedence table, and grammar. There is no special rejection rule: an unregistered contiguous `++` or `--` spelling is rejected by the general whole-symbol-run rule in `DECISION-LEX-003`, while separate operators must be written with a boundary, such as `+ +a` or `+(+a)`. |
| `DECISION-OP-005` | Active | `::=`, `->>`, `<->`, backtick, backslash, `#`, and comment openers remain hard-reserved and cannot be declared or overloaded. The documented mathematical/modifier glyph set is not in this rejected category: those glyphs are pre-declared language-owned operators and may receive `mode=overload` implementations. |
| `DECISION-OP-006` | Active | `~`, `#`, and `^` are reserved prefix spellings and are rejected in operand-prefix position. `#` has no semantics and cannot be used, defined, overloaded, or overridden. `^` remains the built-in infix XOR symbol and `~` retains its named-parameter and `->(~)` roles. `@` is not a prefix expression operator. |
| `DECISION-OP-007` | Active | A constant expression may use a registered custom operator at any declared precedence, but runtime assignment is recursively forbidden throughout the complete constant-expression subtree. Grouping, call arguments, collection/object elements, and other nested expression forms cannot hide an assignment from this guard. Constant evaluation and custom-operator foldability remain semantic checks after parsing. |
| `DECISION-OPLIB-001` | Active | Operator `mode=overload` is supported for every registered symbol—built-in, pre-declared, or project-local custom. Because operator implementations are normal functions, they cannot be declared loose at package scope: built-in operands use an `@co.dap.extension` function inside a unit, structs use their same-package companion unit, and classes use class operator methods. Receiver normalization determines the complete operand signature and its count must match the registered arity. Equivalent normalized signatures are duplicates; distinct signatures are overloads. Modules, enums, unions, interfaces, signatures, and cstructs cannot own operator implementations. Operator `mode=override` and `mode=extends` are unsupported; ordinary class-method overriding through `@co.dap.override` remains separate. |
| `DECISION-OPLIB-002` | Withdrawn in revision 17 | The separately distributable `operator` library-kind model was removed. `type=operator` remains only as the source marker for the configured project-local bootstrap surface and never produces an independently imported artifact. |
| `DECISION-OPLIB-003` | Active | A project-local custom operator symbol has exactly one registration in the compilation's fixed operator library. Duplicate declarations, aliases, merges, selection, and remapping are rejected. The registered symbol may have zero or more implementation overloads in legal owners; each distinct normalized operand signature is permitted and an equivalent signature is a duplicate error. Language-owned built-in and pre-declared symbols cannot be registered locally and are implemented through the same overload mechanism. |
| `DECISION-OPLIB-004` | Withdrawn in revision 17 | Operator-specific imports and activation were removed. Historical operator-specific/imported metadata activation was removed; ordinary imports do not supply operator metadata. |
| `DECISION-OPLIB-005` | Withdrawn in revision 17 | Historical operator manifest/export models were removed; no operator table is embedded in an ordinary library artifact. |
| `DECISION-OPLIB-006` | Withdrawn in revision 17 | Historical imported operator-table bootstrap was removed. Only language-owned registrations and the current compilation's local operator library build the contextual operator table. |
| `DECISION-OPBOOT-001` | Active | `operator_library_folder` in `fol-conf.yaml` identifies the project-local operator source area. A relative path is resolved from the project root. The compiler checks the fixed file `<operator_library_folder>/operators.fol`; absent configuration, folder, or file means no local new operators. The area is excluded from ordinary package discovery. |
| `DECISION-OPBOOT-002` | Active | The configured `operators.fol` is parsed first by a dedicated operator-source lexer and parser. It must contain exactly `@co.dap.library(type=operator)` followed by `_ co.lang.library = { ... }`, whose body contains only `co.lang.operator` declarations. It is a source-only bootstrap surface compiled into its owning project, produces no independent artifact, and is not ordinary FoLang/package source. |
| `DECISION-OPART-001` | Withdrawn in revision 22 | Library operator export was removed. An operator implementation is bound to a class, a companion unit, or an extension unit, and `library-member` admits only import directives, struct declarations, cstruct declarations, and function declarations. No production can therefore place an operator in a library surface file, and no operator table is emitted into a `.folib` or `.folenc` artifact. |
| `DECISION-OPART-002` | Withdrawn in revision 22 | Operator-table loading on import was removed. A direct library import loads the projected symbol table only. An importer that wants operator notation for an imported boundary struct declares its own symbol in its own operator source area and implements it in an extension unit. |
| `DECISION-OPART-003` | Withdrawn in revision 22 | The exported operator entry schema was removed with `DECISION-OPART-001`. Nothing operator-related is serialized into a library artifact. |
| `DECISION-OPBOOT-003` | Active | The frontend reads configuration, parses and validates the local operator library, combines its custom registrations with the language-owned built-in and pre-declared registrations, and builds immutable symbol/fixity/precedence tables before ordinary source tokenization. Imports contribute no operator metadata. Both lexers preserve a complete contiguous symbol run and perform exact whole-run classification without fallback splitting. The parse is single-pass and import-order independent. |
| `DECISION-OPDECL-001` | Active | A new custom symbol is registered only by `SYMBOL co.lang.operator = { ... }` inside the fixed `_ co.lang.library` operator-source body. The declaration carries required `fixity`, `precedence`, `associativity`, and `arity`, plus the optional `commutative`, `idempotent`, `identity`, `foldable`, `vectorizable`, `distributes_over`, and `desugar` metadata. Required properties occur exactly once; optional properties occur at most once; duplicates, omissions, unknown keys, invalid types, and invalid ranges are errors. `@co.dap.operator` at implementation sites carries only `symbol` and optional `mode`. |
| `DECISION-OPDECL-002` | Active | The documented mathematical/modifier glyph set is pre-declared by the language with symbol, fixity, precedence, associativity, and arity but no required implementation. These glyphs are language-owned registrations: they parse everywhere, cannot be redeclared in the local operator library, and may receive any number of distinct `mode=overload` implementation signatures under `DECISION-OPLIB-001`. A missing matching implementation is a resolution error. |
| `DECISION-OPDECL-003` | Active | Operator symbols are global to a compilation while implementation availability is resolved by owner, scope, activation, operand types, and normalized signature. The tokenizer recognizes each registered symbol throughout the compilation; an unavailable implementation therefore produces a resolution diagnostic rather than a syntax error. Using one symbol for one recognizable concept across overloads is recommended for readability but is not a compiler-enforced rule. |
| `DECISION-OPDECL-004` | Active | The operator source area is parsed by a separate grammar rooted at `operator-source-file`. The exact outer declaration is `@co.dap.library(type=operator) _ co.lang.library = { ... }`, and the body admits only operator registrations. `co.lang.operator` is absent from ordinary declaration kinds. The dedicated and ordinary lexers preserve complete contiguous symbol runs; a custom symbol is recognized only by an exact whole-run match and an unknown run is never split into shorter operators. Symbols containing `//` or `/*` are rejected because a comment opener terminates the preceding symbolic run and takes lexical priority. |
| `DECISION-SCOPE-001` | Active | An ordinary local/inner function has block-local identity and resolves free runtime names from its lexical declaration context. Calling it does not replace that environment with the caller's runtime scope. |
| `DECISION-SCOPE-002` | Active | `@co.dap.inner` is an association annotation on a separately declared declaration, not physical nesting. Executable inner-associated declarations resolve free runtime names through the lexical scope chain of the active attachment/call site, without unrestricted runtime caller-chain lookup. Compile-time names remain statically resolved. |
| `DECISION-SEM-001` | Active | Typeclass instances are selected explicitly by name; FoLang performs no implicit instance search. An instance must be declared in the exact package defining either the typeclass or the represented type. Selection and placement are semantic name-resolution checks. |
| `DECISION-SEM-002` | Active | `@co.ddap.use` explicitly and block-scopingly activates extension-unit or typeclass-instance functions as methods. Receiver-owned declarations take priority, followed by activated extensions and then activated instances. A method name may be activated at most once per receiver type in one scope. Parser `CallKind` values remain provisional; any early control-chain lowering must preserve the original member-call chain so resolution can select an overriding user declaration without reconstructing lost syntax. Contextual lambda/wildcard admission requires a receiver-qualified method and is unchanged by transparent grouping of that member callee. |
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
