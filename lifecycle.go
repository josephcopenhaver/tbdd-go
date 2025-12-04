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

// lifecycle has a docstring on the exported alias Lifecycle
//
// see Lifecycle
type lifecycle[T, R any] struct {
	Given, When, Then string
	hooks             Hooks[T, R]
	TC                T

	// CloneTC optionally specifies how to clone the Test Case type rather than using interface detection magic
	// which can be prone to receiver based semantic matching issues.
	CloneTC func(T) T

	// Variants allows for the construction of more test cases from a basis test case.
	// The T passed in is a copy of Lifecycle.TC taken before the basis test runs.
	// The resulting TestVariant.TC values will each be cloned with CloneTC (if non-nil)
	// before being executed, so they can mutate TC without affecting each other.
	Variants func(*testing.T, T) iter.Seq[TestVariant[T]]

	// Arrange, when non-nil, sets hooks, test case defaults, and initial descriptions then returns a
	// "given" description string and a function that sets up any context the test case requires. It will
	// be called shortly after being returned to set up the "given" context for the test case. The returned
	// values must be non-empty and non-nil respectively.
	//
	// Arrange is also the last opportunity to ensure the Act and Assert are set to non-nil - which is a
	// requirement of all tests; otherwise a t.Fatal is called.
	Arrange func(*testing.T, Arrange[T, R]) (string, func(*testing.T))

	// Describe makes sure given (if applicable), when, and then descriptions are set
	Describe func(*testing.T, Describe[T]) DescribeResponse

	// Act exercises the component under test and stores results
	Act func(*testing.T, T) R

	// Assert: validate results + side-effects
	Assert func(*testing.T, Assert[T, R])

	getT    func(testingT) *testing.T
	runHook func(string)
}

type Hooks[T, R any] struct {
	AfterArrange func(*testing.T, AfterArrange[T])
	AfterGiven   func(*testing.T, AfterGiven[T])
	AfterAct     func(*testing.T, AfterAct[T, R])
	AfterAssert  func(*testing.T, AfterAssert[T, R])
}

// Arrange contains the mutable configuration of the rest of the test execution plan.
//
// Arrange is the last opportunity to ensure the Act and Assert are set to non-nil, which is a
// requirement of all tests; otherwise a t.Fatal is called. They can be set via this
// configuration along with other details, except the "Given" string, which is handled via
// the return value of the Arrange call that receives this configuration.
type Arrange[T, R any] struct {
	// TC can be altered by Arrange func if desired.
	TC *T
	// Hooks can be altered by Arrange func if desired.
	Hooks    *Hooks[T, R]
	Describe *func(*testing.T, Describe[T]) DescribeResponse
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
	TC *T
	// ArrangeRan is true if an Arrange function was configured and executed.
	ArrangeRan bool
	// NilGivenFunc is true if Arrange did not run or it did and returned a nil given function.
	NilGivenFunc bool
	// EmptyGivenString is true if the effective Given description of a BDD lifecycle is empty.
	EmptyGivenString bool
}

// AfterGiven describes the configuration of a test case for
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
type AfterAct[T, R any] struct {
	// TC can be altered by AfterAct func if desired.
	TC *T
	// Result can be altered by AfterAct func if desired.
	Result *R
}

// Assert describes the configuration of a test case and its result for analysis.
type Assert[T, R any] struct {
	// TC and its internals are intended to be immutable during Assert phase.
	TC T
	// R and its internals are intended to be immutable during Assert phase.
	Result R
}

// AfterAssert describes the configuration of a test case and its result for
// post-assert hook use.
type AfterAssert[T, R any] struct {
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
	SkipTC      bool
	SkipCloneTC bool
}

// testingT is a simplified version of the functions the *testing.T type implements.
//
// In normal use the caller should always be comfortable using a standard non-nil
// *testing.T value which will always satisfy the interface testingT.
type testingT interface {
	Helper()
	Run(string, func(*testing.T)) bool
	Fatalf(format string, args ...any)
	Error(args ...any)
}

func (b lifecycle[T, R]) afterArrange(t *testing.T, tc *T, arrangeRan, nilGivenFunc, emptyGivenString bool) {
	if f := b.hooks.AfterArrange; f != nil {
		f(t, AfterArrange[T]{tc, arrangeRan, nilGivenFunc, emptyGivenString})
	}
}

func (b lifecycle[T, R]) newI(t testingT, tableTestIndex int) func(testingT) {
	t.Helper()

	// getT converts a testingT to *testing.T
	//
	// under a self-test context it will return nil
	getT := b.getT
	if getT == nil {
		getT = defaultGetT
	}

	// runHook is an internal function reference supporting self-test contexts
	//
	// It is used to track run calls.
	runHook := b.runHook

	f := func(t testingT, tc T, prefix string) func(testingT) {
		t.Helper()

		b := b

		if tableTestIndex >= 0 {
			s := strconv.Itoa(tableTestIndex)
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

		test := func(t testingT) {
			t.Helper()

			if f := b.Describe; f != nil {
				r := f(getT(t), Describe[T]{tc, b.Given, b.When, b.Then})

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
				t.Fatalf(`when+then not run: BDD test not configured properly (prefix = "%s")`, prefix)
				return
			}

			whenStr := "when " + b.When
			if prefix != "" && !hasGivenPhase {
				whenStr = prefix + whenStr
			}

			t.Run(whenStr, func(t *testing.T) {
				nt := nillableT{t, runHook}
				nt.Helper()

				result := b.Act(t, tc)
				if f := b.hooks.AfterAct; f != nil {
					f(t, AfterAct[T, R]{&tc, &result})
				}

				nt.Run("then "+b.Then, func(t *testing.T) {
					nillableT{t, nil}.Helper()

					b.Assert(t, Assert[T, R]{tc, result})
					if f := b.hooks.AfterAssert; f != nil {
						f(t, AfterAssert[T, R]{&tc, &result})
					}
				})
			})
		}

		if hasGivenPhase {
			next := test

			test = func(t testingT) {
				t.Helper()

				var arrangeRan bool
				var given func(*testing.T)
				if f := b.Arrange; f != nil {
					arrangeRan = true
					b.Given, given = f(getT(t), Arrange[T, R]{&tc, &b.hooks, &b.Describe, &b.Act, &b.Assert, b.Given, &b.When, &b.Then})
					if given == nil {
						b.afterArrange(getT(t), &tc, arrangeRan, true, b.Given == "")
						t.Fatalf(`test setup not run: Arrange returned a nil given function (prefix = "%s")`, prefix)
						return
					}
				}

				b.afterArrange(getT(t), &tc, arrangeRan, given == nil, b.Given == "")

				if b.Given == "" {
					t.Fatalf(`test setup not run: Arrange function returned an empty Given string (prefix = "%s")`, prefix)
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
		} else {
			b.afterArrange(getT(t), &tc, false, true, true)

			if f := b.hooks.AfterGiven; f != nil {
				f(getT(t), AfterGiven[T]{&tc, &b.Given, &b.When, &b.Then, false})
			}
		}

		return test
	}

	return func(t testingT) {
		t.Helper()

		// `tc := b.TC` is required so the basis test works on a copy of the lifecycle's TC value.
		// The inner `tc := tc` plus optional CloneTC call let the basis test freely mutate its TC
		// without affecting:
		//   - the Lifecycle's stored TC, and
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
		for v := range variants(getT(t), tc) {
			i++

			if v.SkipTC {
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

func (b lifecycle[T, R]) new(t testingT) func(testingT) {
	t.Helper()

	return b.newI(t, -1)
}

// Lifecycle describes an execution process with a specific order to it.
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
type Lifecycle[T, R any] lifecycle[T, R]

// NewI takes a *testing.T and an index in a table driven test to construct
// sub-tests for a given Lifecycle configuration.
func (b Lifecycle[T, R]) NewI(t *testing.T, tableTestIndex int) func(testingT) {
	t.Helper()

	return (lifecycle[T, R])(b).newI(t, tableTestIndex)
}

// New takes a *testing.T to construct sub-tests for a given Lifecycle configuration.
func (b Lifecycle[T, R]) New(t *testing.T) func(testingT) {
	t.Helper()

	return (lifecycle[T, R])(b).new(t)
}

// GWT constructs a Lifecycle using the classic BDD shape
// “Given / When / Then” for a single test case tc.
//
//   - given is a human-readable description of the initial context.
//   - givenF is called during the Arrange/Given phase and may mutate tc
//     through the *T pointer to install any scenario-specific state.
//   - when is a human-readable description of the action under test.
//   - whenF is used as the Act function; it receives the (possibly mutated)
//     test case value and returns the result R.
//   - then is a human-readable description of the expected outcome.
//   - thenF is used as the Assert function; it receives the final test case
//     and result to perform assertions.
//
// The returned Lifecycle does not execute anything by itself; callers are
// expected to call New or NewI and execute the returned functions usually
// as part of table driven tests.
//
// GWT treats givenF as optional but panics if given is empty and givenF is
// not nil.
// GWT panics if when or then are empty, or if whenF or thenF are nil.
// This is treated as a programmer error in the test harness configuration.
//
// The arguments given and givenF can be empty and nil respectively and the
// resulting lifecycle will not produce any given context indicator in test
// descriptions. Should this be attractive try out the sugar function WT.
func GWT[T, R any](
	tc T,
	given string, givenF func(*testing.T, *T),
	when string, whenF func(*testing.T, T) R,
	then string, thenF func(*testing.T, T, R),
) Lifecycle[T, R] {

	var arrange func(*testing.T, Arrange[T, R]) (string, func(*testing.T))
	if givenF != nil {
		if given == "" {
			panic("tbdd.GWT: given description must be non-empty when given function is non-nil")
		}

		arrange = func(_ *testing.T, cfg Arrange[T, R]) (string, func(*testing.T)) {
			tc := cfg.TC
			return given, func(t *testing.T) {
				givenF(t, tc)
			}
		}
	}

	if when == "" {
		panic("tbdd.GWT: when description must be non-empty")
	}

	if whenF == nil {
		panic("tbdd.GWT: when function must be non-nil")
	}

	if then == "" {
		panic("tbdd.GWT: then description must be non-empty")
	}

	if thenF == nil {
		panic("tbdd.GWT: then function must be non-nil")
	}

	return Lifecycle[T, R]{
		TC:      tc,
		Given:   given,
		Arrange: arrange,
		When:    when,
		Act:     whenF,
		Then:    then,
		Assert: func(t *testing.T, cfg Assert[T, R]) {
			thenF(t, cfg.TC, cfg.Result)
		},
	}
}

// WT is a convenience wrapper around GWT for use when
// there is no given context to convey.
//
// See GWT for more detail.
func WT[T, R any](
	tc T,
	when string, whenF func(*testing.T, T) R,
	then string, thenF func(*testing.T, T, R),
) Lifecycle[T, R] {
	return GWT(
		tc,
		"", nil,
		when, whenF,
		then, thenF,
	)
}

//
// helpers
//

func defaultGetT(t testingT) *testing.T {
	v, _ := t.(*testing.T)
	if v == nil {
		panic("not a real *testing.T instance")
	}

	return v
}

type nillableT struct {
	t       *testing.T
	runHook func(string)
}

func (t nillableT) Helper() {
	if t.t != nil {
		t.t.Helper()
	}
}

func (t nillableT) Run(name string, f func(t *testing.T)) bool {
	if t.t != nil {
		return t.t.Run(name, f)
	}

	if f := t.runHook; f != nil {
		f(name)
	}
	f(nil)
	return true
}
