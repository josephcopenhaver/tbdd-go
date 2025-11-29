# tbdd-go

tbdd stands for **Test Behavior Dumbly Defined**, using "dumb" in the classical engineering sense:
simple, direct, and free of unnecessary machinery.

Modern BDD ecosystems often attach themselves to heavy DSLs (Gherkin), sprawling frameworks (Cucumber), or layers of ceremony that obscure the actual intent of a test. Somewhere along the way, someone decided that "doing BDD" meant producing feature files in a custom DSL with heavy CI/CD toolchain bloat rather than describing context, behavior, and expected outcomes in simple, composable units.

BDD was intended to improve clarity and collaboration — not to mandate a grammar or toolset.

---

## What tbdd is about

tbdd returns to that spirit:

- **Dumb → Simple**
  No special syntax, no DSL, no parser — just Go.

- **Simple → Clear**
  Tests communicate intent first, not framework mechanics.

- **Clear → Expressive**
  Plain Go code is easier to audit, maintain, and reason about.

- **Expressive → Useful**
  Good BDD is about behavior transparency, not feature-file choreography.

tbdd favors plain imperative Go subtests with a declarative `Given` / `When` / `Then` structure — not a separate declarative DSL.

---

## What tbdd crucially does *not* do

- tbdd does **not** replace, wrap, or subvert Go’s test runner.
- There are **no** alternate execution engines.
- **No** custom runners.
- **No** registries.
- **No** global discovery phase.
- **No** framework that secretly drives `go test`.

What you write is what `go test` runs.

---

## Scope

tbdd is intended **exclusively** for `*_test.go` files — never for benchmarks or production code.

Elegance comes from restraint, not ceremony.
