package tbdd

import (
	"iter"
	"slices"
	"strconv"
	"testing"
)

var _ testingT = (*testing.T)(nil)

func Test_testingT(t *testing.T) {
	t.Parallel()

	if v, ok := any(t).(testingT); !ok || v == nil {
		t.Fatal("somehow *testing.T no longer implements testingT")
	}

	{
		var testRan, isNil bool

		func(v testingT) {
			isNil = (v == nil)
			testRan = true
		}(t)

		if !testRan {
			t.Fatal("somehow test failed to run")
		}

		if isNil {
			t.Fatal("somehow casting *testing.T to testingT returned nil")
		}
	}
}

func Test_defaultGetT(t *testing.T) {
	t.Parallel()

	var panicked bool
	var r any
	func() {
		defer func() {
			r = recover()
		}()

		panicked = true
		defaultGetT(nil)
		panicked = false
	}()

	if !panicked {
		t.Error("expected a panic but one did not occur")
	}

	if r == nil {
		t.Error("panic occurred but recover returned nil")
	}

	if r != "not a real *testing.T instance" {
		t.Error("recover did not return the expected value")
	}
}

type fatalfCallData struct {
	format string
	args   []any
}

type mT struct {
	runCalls    []string
	fatalfCalls []fatalfCallData
	errorCalls  [][]any
}

func (t *mT) Helper() {
}

func (t *mT) Run(s string, f func(*testing.T)) bool {
	t.runCalls = append(t.runCalls, s)

	f(nil)
	return !t.Failed()
}

func (t *mT) Fatalf(format string, args ...any) {
	t.fatalfCalls = append(t.fatalfCalls, fatalfCallData{format, args})
}

func (t *mT) Error(args ...any) {
	t.errorCalls = append(t.errorCalls, args)
}

func (t *mT) Failed() bool {
	return len(t.errorCalls) != 0 || len(t.fatalfCalls) != 0
}

type mTC struct {
}

type tc struct {
	mt         *mT
	b          Lifecycle[mTC, mTCR]
	tciPlusOne int

	// expectations

	expRunCalls    []string
	expFatalfCalls []fatalfCallData
	expErrorCalls  [][]any
}

type tcr struct {
	runCalls    []string
	fatalfCalls []fatalfCallData
	errorCalls  [][]any
	setupPass   bool
}

type mTCR struct{}

func nilGetT(testingT) *testing.T {
	return nil
}

func descTC(t *testing.T, cfg Describe[tc]) DescribeResponse {
	t.Helper()

	tc := cfg.TC
	when := cfg.When
	then := cfg.Then

	if when == "" {
		t.Fatal("When was not defined for test case")
	}

	if then == "" {
		if len(tc.expErrorCalls) == 0 && len(tc.expFatalfCalls) == 0 {
			then = "should pass"
		} else {
			then = "should fail"
		}
	}

	return DescribeResponse{when, then}
}

func runTC(t *testing.T, tc tc) tcr {
	t.Helper()

	mt := tc.mt

	var f func(testingT)
	if tc.tciPlusOne == 0 {
		f = tc.b.New(mt)
	} else if tc.tciPlusOne > 0 {
		f = tc.b.NewI(mt, tc.tciPlusOne-1)
	} else {
		t.Fatal("invalid test config: tciPlusOne must not be negative")
	}

	var setupPass bool
	if !mt.Failed() {
		setupPass = true
		f(mt)
	}

	return tcr{
		runCalls:    mt.runCalls,
		fatalfCalls: mt.fatalfCalls,
		errorCalls:  mt.errorCalls,
		setupPass:   setupPass,
	}
}

func checkTC(t *testing.T, cfg Assert[tc, tcr]) {
	t.Helper()

	var failNow bool
	tc := cfg.TC
	r := cfg.Result

	if !r.setupPass {
		t.Errorf("expected setup ok status to be true but got false")
		failNow = true
	}

	{
		expPass := (len(tc.expErrorCalls) == 0 && len(tc.expFatalfCalls) == 0)
		actPass := (len(r.errorCalls) == 0 && len(r.fatalfCalls) == 0)

		if expPass != actPass {
			t.Errorf("expected %t run status but got %t", expPass, actPass)
			failNow = true
		}
	}

	if len(tc.expErrorCalls) != len(r.errorCalls) {
		t.Errorf("expected %d error calls but got %d", len(tc.expErrorCalls), len(r.errorCalls))
		failNow = true
	}

	for i, v := range tc.expErrorCalls {
		if i == len(r.errorCalls) {
			break
		}

		if len(v) != len(r.errorCalls[i]) {
			t.Errorf("expected error call %d to have %d arguments but got %d", i, len(v), len(r.errorCalls[i]))
			failNow = true
		}

		for i2, v := range v {
			if i2 == len(r.errorCalls[i]) {
				break
			}

			if v != r.errorCalls[i][i2] {
				t.Errorf("expected error call %d argument %d to be '%v' but got '%v'", i, i2, v, r.errorCalls[i][i2])
				failNow = true
			}
		}
	}

	if len(tc.expFatalfCalls) != len(r.fatalfCalls) {
		t.Errorf("expected %d fatalf calls but got %d", len(tc.expFatalfCalls), len(r.fatalfCalls))
		failNow = true
	}

	for i, v := range tc.expFatalfCalls {
		if i == len(r.fatalfCalls) {
			break
		}

		if v.format != r.fatalfCalls[i].format {
			t.Errorf("expected fatalf call %d to have format '%s' but got '%s'", i, v.format, r.fatalfCalls[i].format)
			failNow = true
		}

		if len(v.args) != len(r.fatalfCalls[i].args) {
			t.Errorf("expected fatalf call %d to have %d arguments but got %d", i, len(v.args), len(r.fatalfCalls[i].args))
			failNow = true
		}

		for i2, v := range v.args {
			if i2 == len(r.fatalfCalls[i].args) {
				break
			}

			if v != r.fatalfCalls[i].args[i2] {
				t.Errorf("expected fatalf call %d argument %d to be '%v' but got '%v'", i, i2, v, r.fatalfCalls[i].args[i2])
				failNow = true
			}
		}
	}

	if len(tc.expRunCalls) != len(r.runCalls) {
		t.Errorf("expected %d run calls but got %d", len(tc.expRunCalls), len(r.runCalls))
		failNow = true
	}

	for i, v := range tc.expRunCalls {
		if i == len(r.runCalls) {
			break
		}

		if v != r.runCalls[i] {
			t.Errorf("expected run call %d to be '%s' but got '%s'", i, v, r.runCalls[i])
			failNow = true
		}
	}

	if failNow {
		t.Fatal()
	}
}

func commonArrange(cfg Arrange[tc, tcr]) {
	mt := &mT{}
	cfg.TC.mt = mt
	cfg.TC.b.getT = nilGetT
	cfg.TC.b.runHook = func(s string) {
		mt.runCalls = append(mt.runCalls, s)
	}

	*cfg.Describe = descTC
	*cfg.Act = runTC
	*cfg.Assert = checkTC
}

func tcVariants(_ *testing.T, v tc) iter.Seq[TestVariant[tc]] {
	return func(yield func(TestVariant[tc]) bool) {
		if !yield(TestVariant[tc]{
			TC:     tc{},
			Kind:   "skipTC-true",
			SkipTC: true,
		}) {
			return
		}

		if v.tciPlusOne != 0 {
			return
		}

		ctc := cloneTC(v)
		ctc.tciPlusOne = 1
		if len(ctc.expRunCalls) > 0 {
			ctc.expRunCalls[0] = "0/" + ctc.expRunCalls[0]
		}
		for i := range ctc.expFatalfCalls {
			if len(ctc.expFatalfCalls[i].args) == 1 && ctc.expFatalfCalls[i].args[0] == "" {
				ctc.expFatalfCalls[i].args[0] = "0/"
			}
		}

		if !yield(TestVariant[tc]{
			TC:   ctc,
			Kind: "newI-variant",
		}) {
			return
		}
	}
}

func cloneTC(tc tc) tc {
	tc.expRunCalls = slices.Clone(tc.expRunCalls)
	tc.expFatalfCalls = slices.Clone(tc.expFatalfCalls)
	for i := range tc.expFatalfCalls {
		tc.expFatalfCalls[i].args = slices.Clone(tc.expFatalfCalls[i].args)
	}
	return tc
}

func TestLifecycle(t *testing.T) {
	t.Parallel()

	tcs := []Lifecycle[tc, tcr]{
		{
			When: "bear minimum is defined",
			Arrange: func(_ *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)

				b := &cfg.TC.b

				given := "When, Then, Act, and Assert are defined"
				return given, func(*testing.T) {
					b.When = "w"
					b.Then = "t"
					b.Act = func(*testing.T, mTC) mTCR {
						return mTCR{}
					}
					b.Assert = func(*testing.T, Assert[mTC, mTCR]) {
					}
				}
			},
			TC: tc{
				expRunCalls: []string{"when w", "then t"},
			},
		},
		{
			When: "bear minimum and hooks are defined",
			Arrange: func(t *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)

				b := &cfg.TC.b

				given := "When, Then, Act, and Assert are defined with all hooks"
				return given, func(*testing.T) {
					b.When = "w"
					b.Then = "t"
					b.Act = func(*testing.T, mTC) mTCR {
						return mTCR{}
					}
					b.Assert = func(*testing.T, Assert[mTC, mTCR]) {
					}

					h := &b.hooks

					h.AfterArrange = func(*testing.T, AfterArrange[mTC]) {
					}
					h.AfterGiven = func(*testing.T, AfterGiven[mTC]) {
					}
					h.AfterAct = func(*testing.T, AfterAct[mTC, mTCR]) {
					}
					h.AfterAssert = func(*testing.T, AfterAssert[mTC, mTCR]) {
					}
				}
			},
			TC: tc{
				expRunCalls: []string{"when w", "then t"},
			},
		},
		{
			When: "bear minimum and non-mock hooks are defined",
			Arrange: func(t *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)

				b := &cfg.TC.b

				h := cfg.Hooks

				h.AfterArrange = func(*testing.T, AfterArrange[tc]) {
				}
				h.AfterGiven = func(*testing.T, AfterGiven[tc]) {
				}
				h.AfterAct = func(*testing.T, AfterAct[tc, tcr]) {
				}
				h.AfterAssert = func(*testing.T, AfterAssert[tc, tcr]) {
				}

				given := "When, Then, Act, and Assert are defined with all hooks"
				return given, func(*testing.T) {
					b.When = "w"
					b.Then = "t"
					b.Act = func(*testing.T, mTC) mTCR {
						return mTCR{}
					}
					b.Assert = func(*testing.T, Assert[mTC, mTCR]) {
					}
				}
			},
			TC: tc{
				expRunCalls: []string{"when w", "then t"},
			},
		},
		{
			When: "nothing is defined",
			Arrange: func(_ *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)
				cfg.TC.expErrorCalls = [][]any{
					{"When string of BDD test must not be empty"},
					{"Then string of BDD test must not be empty"},
					{"Act function of BDD test is not defined"},
					{"Assert function of BDD test is not defined"},
				}

				given := "When, Then, Act, and Assert are NOT defined"
				return given, func(*testing.T) {}
			},
			TC: tc{
				expFatalfCalls: []fatalfCallData{
					{`when+then not run: BDD test not configured properly (prefix = "%s")`, []any{
						"",
					}},
				},
			},
		},
		{
			When: "arrange is not returning a function",
			Arrange: func(_ *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)

				b := &cfg.TC.b

				given := "When, then, Arrange, Act, and Assert are defined but Arrange returns a nil func"
				return given, func(*testing.T) {
					b.When = "w"
					b.Then = "t"
					b.Arrange = func(*testing.T, Arrange[mTC, mTCR]) (string, func(*testing.T)) {
						return "nil given", nil
					}
					b.Act = func(*testing.T, mTC) mTCR {
						return mTCR{}
					}
					b.Assert = func(*testing.T, Assert[mTC, mTCR]) {
					}
				}
			},
			TC: tc{
				expFatalfCalls: []fatalfCallData{
					{`test setup not run: Arrange returned a nil given function (prefix = "%s")`, []any{
						"",
					}},
				},
			},
		},
		{
			When: "arrange returns an empty given description",
			Arrange: func(_ *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)

				b := &cfg.TC.b

				given := "When, then, Arrange, Act, and Assert are defined but Arrange returns an empty given string"
				return given, func(*testing.T) {
					b.When = "w"
					b.Then = "t"
					b.Arrange = func(*testing.T, Arrange[mTC, mTCR]) (string, func(*testing.T)) {
						return "", func(*testing.T) {}
					}
					b.Act = func(*testing.T, mTC) mTCR {
						return mTCR{}
					}
					b.Assert = func(*testing.T, Assert[mTC, mTCR]) {
					}
				}
			},
			TC: tc{
				expFatalfCalls: []fatalfCallData{
					{`test setup not run: Arrange function returned an empty Given string (prefix = "%s")`, []any{
						"",
					}},
				},
			},
		},
		{
			When: "arrange returns an empty everything",
			Arrange: func(_ *testing.T, cfg Arrange[tc, tcr]) (string, func(*testing.T)) {
				commonArrange(cfg)

				b := &cfg.TC.b

				given := "When, then, Arrange, Act, and Assert are defined but Arrange returns an empty given string"
				return given, func(*testing.T) {
					b.When = "w"
					b.Then = "t"
					b.Arrange = func(*testing.T, Arrange[mTC, mTCR]) (string, func(*testing.T)) {
						return "", nil
					}
					b.Act = func(*testing.T, mTC) mTCR {
						return mTCR{}
					}
					b.Assert = func(*testing.T, Assert[mTC, mTCR]) {
					}
				}
			},
			TC: tc{
				expFatalfCalls: []fatalfCallData{
					{`test setup not run: Arrange returned a nil given function (prefix = "%s")`, []any{
						"",
					}},
				},
			},
		},
	}

	for i, tc := range tcs {
		tc.CloneTC = cloneTC
		tc.Variants = tcVariants
		f := tc.NewI(t, i)
		f(t)
	}
}

func TestLifecycle_badVariants(t *testing.T) {
	t.Parallel()

	b := Lifecycle[mTC, mTCR]{
		When: "w",
		Then: "t",
		Act: func(*testing.T, mTC) mTCR {
			return mTCR{}
		},
		Assert: func(*testing.T, Assert[mTC, mTCR]) {
		},
	}

	mt := &mT{}

	b.getT = nilGetT
	b.runHook = func(s string) {
		mt.runCalls = append(mt.runCalls, s)
	}
	b.Variants = func(*testing.T, mTC) iter.Seq[TestVariant[mTC]] {
		return func(yield func(TestVariant[mTC]) bool) {
			yield(TestVariant[mTC]{
				Kind: "",
			})
		}
	}

	f := b.New(mt)
	f(mt)

	if len(mt.fatalfCalls) != 1 {
		t.Error("expected 1 fatal call but got " + strconv.Itoa(len(mt.fatalfCalls)))
	}

	if mt.fatalfCalls[0].format != "BDD configuration error: test case variant at index %d has no Kind detail" {
		t.Error("unexpected format found in fatalfCalls[0]")
	}

	if len(mt.fatalfCalls[0].args) != 1 {
		t.Error("expected fatalf call 0 to have 1 argument but got " + strconv.Itoa(len(mt.fatalfCalls[0].args)))
	}

	if mt.fatalfCalls[0].args[0] != int(0) {
		t.Errorf("expected fatalf call 0 to be int(0) but got %T(%v)", mt.fatalfCalls[0].args[0], mt.fatalfCalls[0].args[0])
	}
}

func TestWT(t *testing.T) {
	type TC struct{}
	type Result struct{}

	{
		var whenCalled bool
		var thenCalled bool
		b := WT(
			TC{},
			"w", func(*testing.T, TC) Result {
				whenCalled = true
				return Result{}
			},
			"t", func(*testing.T, TC, Result) {
				thenCalled = true
			},
		)

		var _ TestFactory = b

		f := b.New(t)
		f(t)

		if !whenCalled || !thenCalled {
			t.Error()
		}
	}

	//
	// validate panics
	//

	{
		exp := "tbdd.GWT: when description must be non-empty"

		var panicked bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			WT(
				TC{},
				"", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"t", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: when function must be non-nil"

		var panicked bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			WT(
				TC{},
				"w", nil,
				"t", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: then description must be non-empty"

		var panicked bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			WT(
				TC{},
				"w", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: then function must be non-nil"

		var panicked bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			WT(
				TC{},
				"w", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"t", nil,
			)

			panicked = false
		}()

		if !(panicked && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}
}

func TestGWT(t *testing.T) {
	type TC struct{}
	type Result struct{}

	{
		var givenCalled bool
		var whenCalled bool
		var thenCalled bool
		b := GWT(
			TC{},
			"G", func(*testing.T, *TC) {
				givenCalled = true
			},
			"w", func(*testing.T, TC) Result {
				whenCalled = true
				return Result{}
			},
			"t", func(*testing.T, TC, Result) {
				thenCalled = true
			},
		)

		var _ TestFactory = b

		f := b.New(t)
		f(t)

		if !givenCalled || !whenCalled || !thenCalled {
			t.Error()
		}
	}

	//
	// optional given semantics
	//

	{
		var whenCalled bool
		var thenCalled bool

		b := GWT(
			TC{},
			"", nil,
			"w", func(*testing.T, TC) Result {
				whenCalled = true
				return Result{}
			},
			"t", func(*testing.T, TC, Result) {
				thenCalled = true
			},
		)
		f := b.New(t)
		f(t)

		if !(whenCalled && thenCalled) {
			t.Error()
		}
	}

	{
		var whenCalled bool
		var thenCalled bool

		b := GWT(
			TC{},
			"g", nil,
			"w", func(*testing.T, TC) Result {
				whenCalled = true
				return Result{}
			},
			"t", func(*testing.T, TC, Result) {
				thenCalled = true
			},
		)
		f := b.New(t)
		f(t)

		if !(whenCalled && thenCalled) {
			t.Error()
		}
	}

	//
	// validate panics
	//

	{
		exp := "tbdd.GWT: given description must be non-empty when given function is non-nil"

		var panicked bool
		var givenCalled bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			GWT(
				TC{},
				"", func(*testing.T, *TC) {
					givenCalled = true
				},
				"w", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"t", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !givenCalled && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: when description must be non-empty"

		var panicked bool
		var givenCalled bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			GWT(
				TC{},
				"g", func(*testing.T, *TC) {
					givenCalled = true
				},
				"", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"t", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !givenCalled && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: when function must be non-nil"

		var panicked bool
		var givenCalled bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			GWT(
				TC{},
				"g", func(*testing.T, *TC) {
					givenCalled = true
				},
				"w", nil,
				"t", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !givenCalled && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: then description must be non-empty"

		var panicked bool
		var givenCalled bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			GWT(
				TC{},
				"g", func(*testing.T, *TC) {
					givenCalled = true
				},
				"w", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"", func(*testing.T, TC, Result) {
					thenCalled = true
				},
			)

			panicked = false
		}()

		if !(panicked && !givenCalled && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}

	{
		exp := "tbdd.GWT: then function must be non-nil"

		var panicked bool
		var givenCalled bool
		var whenCalled bool
		var thenCalled bool
		var r any

		func() {
			defer func() {
				r = recover()
			}()

			panicked = true
			GWT(
				TC{},
				"g", func(*testing.T, *TC) {
					givenCalled = true
				},
				"w", func(*testing.T, TC) Result {
					whenCalled = true
					return Result{}
				},
				"t", nil,
			)

			panicked = false
		}()

		if !(panicked && !givenCalled && !whenCalled && !thenCalled && exp == r) {
			t.Error()
		}
	}
}
