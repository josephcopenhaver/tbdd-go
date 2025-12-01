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

// BDDLifecycle describes an execution process with a specific order to it.
//
// New and NewI return test functions that implement this process.
//
// The order:
//
// - Arrange
//
// - AfterArrange (hook)
//
// - Given
//
// - AfterGiven (hook)
//
// - Describe
//
// - Act
//
// - AfterAct (hook)
//
// - Assert
//
// - AfterAssert (hook)
//
// - Variants
type BDDLifecycle[T any, R any] struct {
	Given, When, Then string
	hooks             Hooks[T, R]
	TC                T

	// CloneTC optionally specifies how to clone the Test Case type rather than using interface detection magic
	// which can be prone to receiver based semantic matching issues.
	CloneTC func(T) T

	// Variants allows for the construction of more test cases from a basis test case.
	// The T passed in is a copy of BDDLifecycle.TC taken before the basis test runs.
	// The resulting TestVariant.TC values will each be cloned with CloneTC (if non-nil)
	// before being executed, so they can mutate TC without affecting each other.
	Variants func(*testing.T, T) iter.Seq[TestVariant[T]]

	// Arrange sets hooks, test case defaults, and initial descriptions then returns a
	// given string and a function that sets up any context the test case requires. It will
	// be called shortly after being returned to setup the given context for the test case.
	Arrange func(*testing.T, Arrange[T, R]) (string, func(*testing.T))

	// Describe makes sure given (if applicable), when, and then descriptions are set
	Describe func(*testing.T, Describe[T]) DescribeResponse

	// Act exercises the component under test and stores results
	Act func(*testing.T, T) R

	// Assert: validate results + side-effects
	Assert func(*testing.T, Assert[T, R])
}

type Hooks[T any, R any] struct {
	AfterArrange func(*testing.T, AfterArrange[T])
	AfterGiven   func(*testing.T, AfterGiven[T])
	AfterAct     func(*testing.T, AfterAct[T, R])
	AfterAssert  func(*testing.T, AfterAssert[T, R])
}

// Arrange contains the mutable configuration of the rest of the test execution plan.
//
// Act, Assert, When, Then must be set to non-nil/non-empty values by the time Arrange returns.
type Arrange[T any, R any] struct {
	// TC can be altered by Arrange func if desired.
	TC *T
	// Hooks can be altered by Arrange func if desired.
	Hooks *Hooks[T, R]
	// Act can be altered by the Arrange func if desired.
	// This is a pointer to the lifecycle's Act function so Arrange can replace it.
	Act *(func(*testing.T, T) R)
	// Assert can be altered by the Arrange func if desired.
	// This is a pointer to the lifecycle's Assert function so Arrange can replace it.
	Assert *(func(*testing.T, Assert[T, R]))
	// Given is provided for seeding the first return argument context if desired.
	Given string
	// When can be altered by Arrange func if desired.
	// It must be non-empty by the end of the Describe phase which comes after Arrange.
	When *string
	// Then can be altered by Arrange func if desired.
	// It must be non-empty by the end of the Describe phase which comes after Arrange.
	Then *string
}

// AfterArrange describes the configuration of a test case arrangement for
// post-arrange hook use.
type AfterArrange[T any] struct {
	// TC can be altered by AfterArrange func if desired.
	TC         *T
	ArrangeRan bool
}

// AfterGiven describes the configuration of a test case arrangement for
// post-given hook use.
type AfterGiven[T any] struct {
	// TC can be altered by AfterGiven func if desired.
	TC *T
	// Given can be altered by AfterGiven func if desired.
	Given *string
	// When can be altered by AfterGiven func if desired.
	When *string
	// Then can be altered by AfterGiven func if desired.
	Then     *string
	GivenRan bool
}

// Describe contains the configuration of a test case and its Given, When, and then context
// strings. This configuration is used to finalize the values of When and Then in a Describe
// call.
type Describe[T any] struct {
	// TC and its internals are intended to be immutable during Describe phase.
	TC T
	// Given is intended to be immutable during Describe phase.
	Given string
	// When is the initial value of when which can be referenced and loaded into the returned DescribeResponse struct as desired.
	When string
	// Then is the initial value of then which can be referenced and loaded into the returned DescribeResponse struct as desired.
	Then string
}

// DescribeResponse contains the definition of when + then for a BDD test case
// and is the result of a Describe call.
type DescribeResponse struct {
	// When should be set to non-empty by the end of the Describe phase.
	When string
	// Then should be set to non-empty by the end of the Describe phase.
	Then string
}

// AfterAct describes the configuration of a test case and its result for
// post-action hook use.
type AfterAct[T any, R any] struct {
	// TC can be altered by AfterAct func if desired.
	TC *T
	// Result can be altered by AfterAct func if desired.
	Result *R
}

// Assert describes the configuration of a test case and its result for analysis.
type Assert[T any, R any] struct {
	// TC and its internals are intended to be immutable during Assert phase.
	TC T
	// R and its internals are intended to be immutable during Assert phase.
	Result R
}

// AfterAssert describes the configuration of a test case and its result for
// post-assert hook use.
type AfterAssert[T any, R any] struct {
	// TC can be altered by AfterAssert func if desired.
	TC *T
	// Result can be altered by AfterAssert func if desired.
	Result *R
}

// TestVariant describes a new test case created from some basis case.
type TestVariant[T any] struct {
	TC T
	// Kind must be non-empty when returned by a Variants function
	Kind        string
	SkipTest    bool
	SkipCloneTC bool
}

func (b BDDLifecycle[T, R]) NewI(t *testing.T, tci int) func(*testing.T) {
	t.Helper()

	f := func(t *testing.T, tc T, prefix string) func(*testing.T) {
		t.Helper()

		b := b

		if tci >= 0 {
			s := strconv.Itoa(tci)
			if prefix == "" {
				prefix = s
			} else {
				prefix = s + "/" + prefix
			}
		}
		if prefix != "" {
			prefix += "/"
		}

		hasGivenPhase := (b.Arrange != nil || b.Given != "")

		test := func(t *testing.T) {
			t.Helper()

			if f := b.Describe; f != nil {
				r := f(t, Describe[T]{tc, b.Given, b.When, b.Then})

				b.When = r.When
				b.Then = r.Then
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

			whenStr := "when " + b.When
			if prefix != "" && !hasGivenPhase {
				whenStr = prefix + whenStr
			}

			t.Run(whenStr, func(t *testing.T) {
				t.Helper()

				result := b.Act(t, tc)
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

		if hasGivenPhase {
			next := test

			test = func(t *testing.T) {
				t.Helper()

				var arrangeRan bool
				var given func(*testing.T)
				if f := b.Arrange; f != nil {
					arrangeRan = true
					b.Given, given = f(t, Arrange[T, R]{&tc, &b.hooks, &b.Act, &b.Assert, b.Given, &b.When, &b.Then})
					if given == nil {
						t.Fatal("test setup not run: Arrange returned a nil given function")
						return
					}
				}

				if f := b.hooks.AfterArrange; f != nil {
					f(t, AfterArrange[T]{&tc, arrangeRan})
				}

				if b.Given == "" {
					t.Fatal("test setup not run: Arrange function returned an empty Given string")
					return
				}

				t.Run(prefix+"given "+b.Given, func(t *testing.T) {
					t.Helper()

					var givenRan bool
					if given != nil {
						givenRan = true
						given(t)
					}

					if f := b.hooks.AfterGiven; f != nil {
						f(t, AfterGiven[T]{&tc, &b.Given, &b.When, &b.Then, givenRan})
					}

					next(t)
				})
			}
		}

		return test
	}

	return func(t *testing.T) {
		t.Helper()

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

			if v.SkipTest {
				continue
			}

			if v.Kind == "" {
				t.Fatalf("BDD configuration error: test case variant at index %d has no Kind detail", i)
				continue
			}

			tc := v.TC
			if !v.SkipCloneTC {
				if f := b.CloneTC; f != nil {
					tc = f(tc)
				}
			}

			f(t, tc, v.Kind)(t)
		}
	}
}

func (b BDDLifecycle[T, R]) New(t *testing.T) func(*testing.T) {
	t.Helper()

	return b.NewI(t, -1)
}
