# Hello, FoLang! — Single-Source Applications

These examples use FoLang's fixed single-source application entry file:

```text
<project>/
└── src/
    └── appl.fol
```

Each example is a complete application unless it is explicitly labeled as a third-party library template. Use one example at a time as `src/appl.fol`. A single-source project has no user-created package directories beneath `src/`.

The examples follow one simple rhythm:

1. Read the developer's question.
2. Predict what the program will do.
3. Run the program.
4. Compare the result with the expected output.
5. Make the suggested change and run it again.

> **Reference note:** This is a teaching guide. The **FoLang Language Reference** is the normative definition of the language. If this guide and the applicable Language Reference differ, the Language Reference governs.

## 1. Can I print something?

> **Developer:** Do I need to create a `main` function before I can print a message?
>
> **Guide:** No. `src/appl.fol` is already the application entry file.

```folang
co.out.println("Hello, FoLang!");
```

Expected output:

```text
Hello, FoLang!
```

The entry file needs no entry-point annotation and no ceremonial `main` function.

**Try it:** Change the message and run the application again.

## 2. How do I create variables and make a choice?

> **Developer:** Do I have to write the type of every variable?
>
> **Guide:** No. You may declare a type explicitly, or use `:=` and let FoLang infer it from the initializer.

```folang
name co.lang.string = "Kameswara";
score := 82;

co.out.print("Hello, ");
co.out.println(name);

(score >= 50).then({
    co.out.println("Result: pass");
}).default({
    co.out.println("Result: try again");
});
```

Pause and predict the output before continuing.

Expected output:

```text
Hello, Kameswara
Result: pass
```

The first binding has an explicit `co.lang.string` type. The second obtains its type from `82`.

`then` handles the true branch. A final branch without another condition is written with `default`. When another condition is needed, FoLang uses `otherwise(condition).then(...)`.

**Try it:** Change `score` to `40` and run the application again.

## 3. How do I repeat work?

> **Developer:** How can I add the numbers from one through five?
>
> **Guide:** Use a condition with `loop`, updating the values on every iteration.

```folang
limit := 5;
current := 1;
total := 0;

(current <= limit).loop({
    total += current;
    current += 1;
});

co.out.print("Total: ");
co.out.println(total);
```

Expected output:

```text
Total: 15
```

`loop` has one condition and one body. When the condition becomes false, repetition ends.

**Try it:** Change `limit` to `10`. Predict the new total before running the program.

## 4. How do I visit every array element?

> **Developer:** Can I process every score without writing a separate index operation for each one?
>
> **Guide:** Yes. An array exposes `each` for element-by-element iteration.

```folang
scores co.lang.int->([5]) = [78, 92, 66, 85, 73];
total := 0;

scores.each(_, score, {
    co.out.println(score);
    total += score;
});

co.out.print("Total: ");
co.out.println(total);
```

Expected output:

```text
78
92
66
85
73
Total: 394
```

The first `each` binding is the index. This application does not need it, so `_` discards it. The second binding receives the element.

`each` performs the iteration itself. Do not append `.loop(...)` to it.

**Try it:** Replace `_` with `index`, then print `index` before each score.

## 5. Can I find a number in an array?

> **Developer:** I have an array. How do I ask whether it contains a particular value?
>
> **Guide:** Use `contains` and make a choice from its Boolean result.

```folang
number co.lang.int = 30;
numbers co.lang.int->([5]) = [10, 20, 30, 40, 50];

numbers.contains(number).then({
    co.out.print("Number ");
    co.out.print(number);
    co.out.println(" is present.");
}).default({
    co.out.print("Number ");
    co.out.print(number);
    co.out.println(" is not present.");
});
```

Expected output:

```text
Number 30 is present.
```

**Try it:** Change `number` to `100`. The array does not contain that value, so the `default` branch should run.

## 6. Can I solve the same problem with a List?

> **Developer:** Does changing from an array to a List require a completely different search?
>
> **Guide:** No. The standard List API also exposes `contains`.

```folang
number co.lang.int = 30;

numbers co.core.List->(co.lang.int) =
    co.core.List[10, 20, 30, 40, 50];

numbers.contains(number).then({
    co.out.print("Number ");
    co.out.print(number);
    co.out.println(" is present.");
}).default({
    co.out.print("Number ");
    co.out.print(number);
    co.out.println(" is not present.");
});
```

Expected output:

```text
Number 30 is present.
```

The array and the List are different data structures, but they provide a consistent containment operation. Consistent operations do not make the data structures identical; their storage, access, and mutation behavior may differ.

**Try it:** Change `number` to `100` and confirm that the other branch runs.

## 7. How do Map keys and values differ?

> **Developer:** A Map contains keys and values. What does `contains` search?
>
> **Guide:** `contains` searches keys. Use `containsVal` when you want to search values.

```folang
number := 100;

numbers co.core.Map->(
    key=co.lang.int,
    val=co.lang.int
) = co.core.Map{
    10: 10,
    100: 20,
    50: 30
};

numbers.contains(number).then({
    co.out.print(number);
    co.out.println(" is a key.");
}).default({
    co.out.print(number);
    co.out.println(" is not a key.");
});

numbers.containsVal(number).then({
    co.out.print(number);
    co.out.println(" is a value.");
}).default({
    co.out.print(number);
    co.out.println(" is not a value.");
});
```

Pause and predict both results.

Expected output:

```text
100 is a key.
100 is not a value.
```

The Map has a key `100`, but its values are `10`, `20`, and `30`.

**Try it:** Change `number` to `20`. Which containment operation succeeds now?

## 8. Can a Set use the same containment question?

> **Developer:** What if I need a collection of unique values?
>
> **Guide:** Use a Set. Set Literal notation uses parentheses unlike list Literal which is square brackets and map Literal which is braces, and its `contains` operation asks whether an element is present.

```folang
number := 100;

numbers co.core.Set->(co.lang.int) =
    co.core.Set(10, 100, 50);

numbers.contains(number).then({
    co.out.print("Number ");
    co.out.print(number);
    co.out.println(" is present.");
}).default({
    co.out.print("Number ");
    co.out.print(number);
    co.out.println(" is not present.");
});
```

Expected output:

```text
Number 100 is present.
```

Arrays, lists, maps, and sets may share an operation name such as `contains`, but each collection retains its own meaning and behavior.

**Try it:** Add `100` to the Set Literal a second time and observe how the installed standard-library implementation presents the resulting Set.

## 9. Can I iterate over a Range?

> **Developer:** Do I have to construct a collection just to work with a sequence of numbers?
>
> **Guide:** No. A finite Range can describe the sequence directly.

```folang
numbers := 1 .. 5;

numbers.each(_, value, {
    co.out.println(value);
});

numbers.contains(3).then({
    co.out.println("The range contains 3.");
}).default({
    co.out.println("The range does not contain 3.");
});
```

Expected output:

```text
1
2
3
4
5
The range contains 3.
```

`1 .. 5` includes both endpoints. FoLang supports other boundary combinations, including half-open ranges such as `0 ..< 5`. Consult the Language Reference when you need the complete range syntax.

**Try it:** Change the declaration to `numbers := 0 ..< 5;` and predict the values that will be printed.

## 10. How do I use a third-party library?

> **Developer:** Can my single-source application use an external library?
>
> **Guide:** Yes. Single-source means that your own application source remains in `src/appl.fol`; it does not mean that the application must be dependency-free.

Place the prebuilt `.folenc` artifact supplied by the library publisher in the project-root `lib/` directory:

```text
<project>/
├── lib/
│   └── example.folenc
└── src/
    └── appl.fol
```

The publisher's documentation tells you which import form and public names the artifact exposes.

### Projected library surface

A projected library is imported with `library=`:

```folang
@co.ddap.import(library="example", as="ext")

ext.greet("FoLang");
```

### Packaged/open package surface

A packaged artifact exposes selected package contexts. Import one with `package=`:

```folang
@co.ddap.import(package="example.text", as="text")

text.greet("FoLang");
```

These are templates: replace `example`, the package path, alias, and `greet` call with the exact names documented by the library publisher.

Important rules:

- Put `@co.ddap.import` in the file preamble, before executable statements.
- Use `as=` to create the file-local import alias.
- Do not write a semicolon after the directive.
- The alias must be a valid, unambiguous identifier in that file.
- Call the imported API through the alias, such as `ext.greet(...)`.
- You are consuming a prebuilt dependency; you are not creating a package or component.

Build and run the project with the commands documented for your installed `folcc` release. The Language Reference defines the project and source semantics, while the applicable toolchain documentation supplies the exact command-line flags.

## 11. Why does `co.*` need no import?

> **Developer:** Third-party APIs need an import. Why can I call `co.out.println` directly?
>
> **Guide:** The `co` root belongs to FoLang's installed standard library and is made available automatically.

The compiler loads the installed standard-library artifact before it parses the application source. That is why this is valid without an import:

```folang
co.out.println("Hello from the standard library.");
```

You may use the standard APIs exported by the installed standard-library version. Their exact declarations and signatures remain governed by the applicable Language Reference and standard-library artifact.

You may shorten a built-in `co.*` path with a file-local alias:

```folang
@co.ddap.alias(co.out, as="out")

out.println("Hello through an alias.");
```

`@co.ddap.alias` does not remove the complete path; `co.out.println` remains valid in the same file.

## 12. Can I create my own data types?

> **Developer:** So far I have used built-in types and APIs. Can FoLang represent types from my own problem?
>
> **Guide:** Yes, but that is deliberately beyond the promise of this book.

*Hello, Foλang!* is complete within its single-source learning boundary. It teaches you to create, understand, build, and run useful programs from `src/appl.fol`, including programs that consume third-party libraries. It does not introduce the larger application structures used to model and organize your own domain.

That question begins the next stage of the journey.

> If you enjoyed this journey and would like to learn how FoLang approaches and structures a larger application step by step, please have a look at **Think Like FoLang**.

