// Package tbdd provides a lightweight, Go-native BDD-style test harness.
//
// The name stands for **Test Behavior Dumbly Defined**. Here, “dumb” is used
// in the classical engineering sense: simple, direct, free of unnecessary
// machinery. Behavioral tests do not require elaborate DSLs (like Gherkin) or
// large frameworks (like Cucumber). They can be defined plainly and elegantly
// using everyday Go code.
//
// The tbdd philosophy:
//
//	dumb  → simple
//	simple → clear
//	clear → expressive and auditable
//
// By structuring behavior “dumbly,” tbdd avoids ceremony and focuses on what
// matters: test intent, not framework plumbing.
//
// This package is intended **exclusively for use in *_test.go files**. It
// should not be used in benchmarks or any other non-test context.
package tbdd

import (
	"iter"
	"strconv"
	"testing"
)

type DescIn[T any] struct {
	// TC and its internals are intended to be immutable during Describe phase.
	TC T
	// Given is intended to be immutable during Describe phase.
	Given string
}

type DescOut struct {
	// When should be set to non-empty by the end of the Describe phase.
	When *string
	// Then should be set to non-empty by the end of the Describe phase.
	Then *string
}

type AfterArrange[T any] struct {
	// TC can be altered by AfterArrange func if desired.
	TC         *T
	ArrangeRan bool
}

type AfterAct[T any, R any] struct {
	// TC can be altered by AfterAct func if desired.
	TC *T
	// Result can be altered by AfterAct func if desired.
	Result *R
}

type AfterAssert[T any, R any] struct {
	// TC can be altered by AfterAssert func if desired.
	TC *T
	// Result can be altered by AfterAssert func if desired.
	Result *R
}

type Hooks[T any, R any] struct {
	AfterArrange func(*testing.T, AfterArrange[T])
	AfterAct     func(*testing.T, AfterAct[T, R])
	AfterAssert  func(*testing.T, AfterAssert[T, R])
}

type Given[T any] struct {
	// TC can be altered by Given func if desired.
	TC *T
	// Context must be non-empty by the end of the Given phase if the Given func does run.
	Context *string
}

type Arrange[T any, R any] struct {
	// TC can be altered by Arrange func if desired.
	TC *T
	// Hooks can be altered by Arrange func if desired.
	Hooks *Hooks[T, R]
	// When can be altered by Arrange func if desired.
	// It must be non-empty by the end of the Describe phase which comes after Arrange.
	When *string
	// Then can be altered by Arrange func if desired.
	// It must be non-empty by the end of the Describe phase which comes after Arrange.
	Then *string
}

type ActIn[T any] struct {
	// TC and its internals are intended to be immutable during Act phase.
	TC T
}

type ActOut[R any] struct {
	// Res can be altered by Act func if desired.
	Res *R
}

type Assert[T any, R any] struct {
	// TC and its internals are intended to be immutable during Assert phase.
	TC T
	// Res and its internals are intended to be immutable during Assert phase.
	Res R
}

type TestVariation[T any] struct {
	TC T
	// Kind must be non-empty when returned by a test case implementing `Variants(*testing.T) iter.Seq[TestVariation[T]]`
	Kind string
	Skip bool
}

type BDDLifecycle[T any, R any] struct {
	Given, When, Then string
	hooks             Hooks[T, R]
	TC                T

	// CloneTC optionally specifies how to clone the Test Case type rather than using interface detection magic
	// which can be prone to receiver based semantic matching issues.
	CloneTC func(T) T

	// Variants allows for the construction of more test cases from a basis test case.
	// The T passed in is a copy of BDDLifecycle.TC taken before the basis test runs.
	// The resulting TestVariation.TC values will each be cloned with CloneTC (if non-nil)
	// before being executed, so they can mutate TC without affecting each other.
	Variants func(*testing.T, T) iter.Seq[TestVariation[T]]

	// GivenContext sets the given context string for a particular initial test configuration (if applicable)
	//
	// The Context string passed to this handler must be non-empty if Arrange func is non-nil
	GivenContext func(*testing.T, Given[T])

	// Arrange sets hooks, test case defaults, and initial descriptions
	//
	// Arrange can only be specified when a given context is set (either by GivenContext or by directly setting Given).
	Arrange func(*testing.T, Arrange[T, R])

	// Describe makes sure given (if applicable), when, and then descriptions are set
	Describe func(*testing.T, DescIn[T], DescOut)

	// Act exercises the component under test and stores results
	Act func(*testing.T, ActIn[T], ActOut[R])

	// Assert: validate results + side-effects
	Assert func(*testing.T, Assert[T, R])
}

func (b BDDLifecycle[T, R]) NewI(t *testing.T, tci int) func(*testing.T) {
	t.Helper()

	f := func(t *testing.T, tc T, prefix string) func(*testing.T) {
		b := b

		var result R
		test := func(t *testing.T) {
			t.Helper()

			if f := b.Describe; f != nil {
				f(t, DescIn[T]{tc, b.Given}, DescOut{&b.When, &b.Then})
			}

			if b.When == "" {
				t.Error("When string of BDD test must not be empty")
			}
			if b.Then == "" {
				t.Error("Then string of BDD test must not be empty")
			}
			if b.Act == nil {
				t.Error("Act function of BDD test is not defined")
			}
			if b.Assert == nil {
				t.Error("Assert function of BDD test is not defined")
			}
			if b.When == "" || b.Then == "" || b.Act == nil || b.Assert == nil {
				t.Fatal("when+then not run: BDD test not configured properly")
				return
			}

			t.Run("when "+b.When, func(t *testing.T) {
				t.Helper()

				b.Act(t, ActIn[T]{tc}, ActOut[R]{&result})
				if f := b.hooks.AfterAct; f != nil {
					f(t, AfterAct[T, R]{&tc, &result})
				}

				t.Run("then "+b.Then, func(t *testing.T) {
					t.Helper()

					b.Assert(t, Assert[T, R]{tc, result})
					if f := b.hooks.AfterAssert; f != nil {
						f(t, AfterAssert[T, R]{&tc, &result})
					}
				})
			})
		}

		if f := b.GivenContext; f != nil {
			f(t, Given[T]{&tc, &b.Given})
		}

		if b.Arrange != nil || b.GivenContext != nil || b.Given != "" {
			next := test

			var arrangeRan bool
			test = func(t *testing.T) {
				t.Helper()

				if b.Given == "" {
					t.Fatal("test setup not run: Given function of BDD test is not defined (while Arrange is) or failed to set context")
					return
				}

				t.Run("given "+b.Given, func(t *testing.T) {
					t.Helper()

					if f := b.Arrange; f != nil {
						arrangeRan = true
						f(t, Arrange[T, R]{&tc, &b.hooks, &b.When, &b.Then})
					}

					if f := b.hooks.AfterArrange; f != nil {
						f(t, AfterArrange[T]{&tc, arrangeRan})
					}

					next(t)
				})
			}
		}

		if tci >= 0 || prefix != "" {
			next := test

			if tci >= 0 {
				s := strconv.Itoa(tci)
				if prefix == "" {
					prefix = s
				} else {
					prefix = s + "/" + prefix
				}
			}

			test = func(t *testing.T) {
				t.Helper()

				t.Run(prefix, next)
			}
		}

		return test
	}

	return func(t *testing.T) {
		// `tc := b.TC` is required so the basis test works on a copy of the lifecycle's TC value.
		// The inner `tc := tc` plus optional CloneTC call let the basis test freely mutate its TC
		// without affecting:
		//   - the BDDLifecycle's stored TC, and
		//   - the value passed to Variants,
		// except for any shared mutable pointer types when CloneTC is nil or shallow.
		tc := b.TC

		// run non-variant basis test case
		{
			tc := tc // don't delete this line, see above comment block
			if f := b.CloneTC; f != nil {
				tc = f(tc)
			}

			f(t, tc, "")(t)
		}

		variants := b.Variants
		if variants == nil {
			return
		}

		// run test case variations

		i := -1
		for v := range variants(t, tc) {
			i++

			if v.Skip {
				continue
			}

			if v.Kind == "" {
				t.Fatalf("BDD configuration error: test case variant at index %d has no Kind details", i)
			}

			tc := v.TC
			if f := b.CloneTC; f != nil {
				tc = f(tc)
			}

			f(t, tc, v.Kind)(t)
		}
	}
}

func (b BDDLifecycle[T, R]) New(t *testing.T) func(*testing.T) {
	t.Helper()

	return b.NewI(t, -1)
}
