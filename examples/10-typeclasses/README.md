# 10 — Type Classes

Spec: *Type Classes*, *Monads, Applicatives, Functors, Monoids and
Transformers*, *Matchers*.

`@co.dap.typeclass(kind=..., shape=(...))` is the built-in annotation for every typeclass
definition. `kind` names the algebraic structure — `Functor`, `Applicative`,
`Monad`, `Monoid`, `Transformer`, or any user-defined kind.

Instances of **any** typeclass are declared with `co.lang.instance`, using
`->(for=..., type=...)` (or `types=[...]` when the structure takes more than
one).

| Definition | Instance |
|---|---|
| `Functor.fol` | `ListFunctor.fol` |
| `Applicative.fol` | `OptionApplicative.fol` |
| `Monad.fol` | `OptionMonad.fol` |
| `Monoid.fol` | `IntMonoid.fol` |
| `Transformer.fol` | `ListToSetTransformer.fol` |
| `Matcher.fol` | `PositiveEvenMatcher.fol` |

A typeclass definition is an `annotated-contract-declaration`: the built-in
annotation supplies its kind and shape, `_ co.lang.typeclass` supplies the
filename-derived name, and its body contains function and value specifications.
Each specification ends with a semicolon because it has no body.
