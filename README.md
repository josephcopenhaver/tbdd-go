# tbdd-go

[![Go Report Card](https://goreportcard.com/badge/github.com/josephcopenhaver/tbdd-go)](https://goreportcard.com/report/github.com/josephcopenhaver/tbdd-go)
![tests](https://github.com/josephcopenhaver/tbdd-go/actions/workflows/tests.yaml/badge.svg)
![code-coverage](https://img.shields.io/badge/code_coverage-100%25-rgb%2852%2C208%2C88%29)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

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

---

## Quickstart: GWT & WT (what most users need)

Most users only need two helpers:

- `GWT` — full **Given / When / Then**.
- `WT` — simplified **When / Then** when there’s no meaningful `Given`.

You do **not** need to understand the internals of `Lifecycle` to use these.

### GWT: Given / When / Then

Use `GWT` when you want to explicitly model the setup, action, and expected outcome.

```go
package yourpkg_test

import (
    "net/http"
    "testing"

    "github.com/josephcopenhaver/tbdd-go"
)

func TestLogin(t *testing.T) {
    type TestCase struct {
        UserID string
    }

    type Result struct {
        RedirectURL string
        Err         error
    }

    tc := TestCase{UserID: "u1"}

    b := tbdd.GWT(
        tc,
        // Given
        "a registered user",
        func(t *testing.T, tc *TestCase) {
            seedUserInDB(t, tc.UserID)
        },
        // When
        "they log in",
        func(t *testing.T, tc TestCase) Result {
            return doLogin(t, tc.UserID)
        },
        // Then
        "they see their dashboard",
        func(t *testing.T, tc TestCase, r Result) {
            if r.Err != nil {
                t.Fatalf("unexpected error: %v", r.Err)
            }
            if got, want := r.RedirectURL, "/home"; got != want {
                t.Fatalf("redirect: got %q want %q", got, want)
            }
        },
    )

    t.Run("login succeeds", b.New(t))
}
```

**Mental model:**

- `Given` (`givenF`) mutates the test case (`*T`) to install scenario-specific context
  (DB records, fake services, seeded state, etc.).
- `When` (`whenF`) acts on the (possibly mutated) test case and returns a result (`R`).
- `Then` (`thenF`) asserts using both the test case (`T`) and the result (`R`).

tbdd wires this into nested subtests under the hood, but you only see normal `t.Run` calls.

### WT: When / Then only

Use `WT` when the initial state is already encoded in the test case and there’s no extra `Given` step.

```go
func TestHealthCheck(t *testing.T) {
    type TestCase struct {
        URL string
    }

    type Result struct {
        StatusCode int
        Err        error
    }

    tc := TestCase{URL: "http://127.0.0.1:8080/healthz"}

    b := tbdd.WT(
        tc,
        // When
        "a client calls /healthz",
        func(t *testing.T, tc TestCase) Result {
            resp, err := http.Get(tc.URL)
            if resp != nil {
                defer resp.Body.Close()
            }
            code := 0
            if resp != nil {
                code = resp.StatusCode
            }
            return Result{StatusCode: code, Err: err}
        },
        // Then
        "it returns 200 OK",
        func(t *testing.T, tc TestCase, r Result) {
            if r.Err != nil {
                t.Fatalf("unexpected error: %v", r.Err)
            }
            if r.StatusCode != http.StatusOK {
                t.Fatalf("status: got=%d want=%d", r.StatusCode, http.StatusOK)
            }
        },
    )

    t.Run("health check", b.New(t))
}
```

---

## How it integrates with `go test`

tbdd never runs tests directly. It only builds `func(testingT)` values that you pass to `t.Run` or directly call.

For any `Lifecycle[T,R]` returned by `GWT` / `WT`:

```go
b := tbdd.GWT(/* ... */)

// b.New(t) returns func(testingT) that executes the scenario.
t.Run("scenario name", b.New(t))
```

You can:

- Nest these under other subtests.
- Mix them with plain table-driven tests.
- Wrap them in your own helpers.

There are no registries, discovery phases, or magic entrypoints. `go test` is still in charge.

---

## Why not just plain table tests?

You can absolutely write all of this with plain table-driven tests and nested `t.Run` calls. tbdd is intentionally small; it is not trying to be a framework.

`GWT` / `WT` are about:

- **Centralizing invariants**
  - Non-empty `Given` / `When` / `Then` descriptions.
  - Non-nil `When` / `Then` functions.
- **Making context explicit**
  - `Given` mutates `*T` — the test case **is** your scenario + environment.
  - `When` and `Then` explicitly consume `T` (and `R`), not hidden globals.
- **Providing a stable pattern**
  - Every scenario has the same shape.
  - It’s easy to scan a file and read behaviors as sentences:
    > Given X, When Y, Then Z.

If you’re happy hand-rolling `t.Run` everywhere, tbdd is optional.
If you want a small, consistent BDD-ish structure with minimal ceremony, `GWT` and `WT` give you that in pure Go.

---

## Power users: Lifecycle, hooks, variants

If all you need is “nicely structured tests,” you can stop here.
The rest of this document is for users who want to build richer harnesses on top of tbdd.

### Lifecycle

Under the hood, `GWT` and `WT` return a `Lifecycle[T,R]`:

```go
type Lifecycle[T any, R any] struct {
    Given string
    When  string
    Then  string

    TC      T
    CloneTC func(T) T

    Arrange  func(*testing.T, Arrange[T, R]) (string, func(*testing.T))
    Describe func(*testing.T, Describe[T]) DescribeResponse
    Act      func(*testing.T, T) R
    Assert   func(*testing.T, Assert[T, R])

    Variants func(*testing.T, T) iter.Seq[TestVariant[T]]

    // plus internal wiring / hooks
}
```

The default `GWT` wiring uses:

- `Given` + `Arrange` to mutate `TC` for the scenario.
- `When` + `Act` as the action under test.
- `Then` + `Assert` as the verification step.

You can:

- Construct `Lifecycle` values directly for more advanced patterns.
- Override `Arrange` / `Describe` / `Variants` to customize naming, variant generation, and subtest layout.
- Use hooks (not shown here) to attach cross-cutting behavior like logging, metrics, or debugging.

### Variants

`Lifecycle.Variants` lets you fan out from a basis test case into multiple table-driven variants while preserving the same GWT/WT structure:

```go
b.Variants = func(t *testing.T, basis TestCase) iter.Seq[tbdd.TestVariant[TestCase]] {
    return func(yield func(tbdd.TestVariant[TestCase]) bool) {
        if !yield(tbdd.TestVariant[TestCase]{
            Name: "admin user",
            TC:   basisWithRole(basis, "admin"),
        }) {
            return
        }
        _ = yield(tbdd.TestVariant[TestCase]{
            Name: "suspended user",
            TC:   basisWithSuspension(basis),
        })
    }
}
```

tbdd will create additional subtests for each variant using your existing `Given / When / Then` functions.

---

[![Go Reference](https://pkg.go.dev/badge/github.com/josephcopenhaver/tbdd-go.svg)](https://pkg.go.dev/github.com/josephcopenhaver/tbdd-go)
