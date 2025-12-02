package tbdd

import (
	"testing"
)

var _ testingT = (*testing.T)(nil)

func Test_testingT(t *testing.T) {
	t.Parallel()

	{
		v, ok := any(t).(testingT)

		if !ok || v == nil {
			t.Fatal("somehow *testing.T no longer implements testingT")
		}
	}

	{
		var testRan, isNil bool

		func(v testingT) {
			testRan = true
			isNil = (v == nil)
		}(t)

		if !testRan {
			t.Fatal("somehow test failed to run")
		}

		if isNil {
			t.Fatal("somehow casting *testing.T to testingT returned nil")
		}
	}
}
