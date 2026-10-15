// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package require reports a failed check and stops the test, with the same
// semantics as github.com/stretchr/testify/require for the functions
// provided.
package require

import "github.com/raftseg/protobuf-go-lite/internal/testutil"

// TestingT is what the checks need from a test.
type TestingT = testutil.T

func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Equal(t, expected, actual, msgAndArgs) {
		t.FailNow()
	}
}

func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.NotEqual(t, expected, actual, msgAndArgs) {
		t.FailNow()
	}
}

func NoError(t TestingT, err error, msgAndArgs ...any) {
	t.Helper()
	if !testutil.NoError(t, err, msgAndArgs) {
		t.FailNow()
	}
}

func Error(t TestingT, err error, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Error(t, err, msgAndArgs) {
		t.FailNow()
	}
}

func ErrorIs(t TestingT, err, target error, msgAndArgs ...any) {
	t.Helper()
	if !testutil.ErrorIs(t, err, target, msgAndArgs) {
		t.FailNow()
	}
}

func True(t TestingT, value bool, msgAndArgs ...any) {
	t.Helper()
	if !testutil.True(t, value, msgAndArgs) {
		t.FailNow()
	}
}

func False(t TestingT, value bool, msgAndArgs ...any) {
	t.Helper()
	if !testutil.False(t, value, msgAndArgs) {
		t.FailNow()
	}
}

func Len(t TestingT, object any, length int, msgAndArgs ...any) {
	t.Helper()
	if !testutil.LenIs(t, object, length, msgAndArgs) {
		t.FailNow()
	}
}

func Empty(t TestingT, object any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Empty(t, object, msgAndArgs) {
		t.FailNow()
	}
}

func NotEmpty(t TestingT, object any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.NotEmpty(t, object, msgAndArgs) {
		t.FailNow()
	}
}

func Zero(t TestingT, i any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Zero(t, i, msgAndArgs) {
		t.FailNow()
	}
}

func NotZero(t TestingT, i any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.NotZero(t, i, msgAndArgs) {
		t.FailNow()
	}
}

func Nil(t TestingT, object any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Nil(t, object, msgAndArgs) {
		t.FailNow()
	}
}

func NotNil(t TestingT, object any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.NotNil(t, object, msgAndArgs) {
		t.FailNow()
	}
}

func Panics(t TestingT, f func(), msgAndArgs ...any) {
	t.Helper()
	if !testutil.Panics(t, f, msgAndArgs) {
		t.FailNow()
	}
}

func GreaterOrEqual(t TestingT, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Ordered(t, e1, e2, func(c int) bool { return c >= 0 }, "greater than or equal to", msgAndArgs) {
		t.FailNow()
	}
}

func LessOrEqual(t TestingT, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Ordered(t, e1, e2, func(c int) bool { return c <= 0 }, "less than or equal to", msgAndArgs) {
		t.FailNow()
	}
}

func Greater(t TestingT, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Ordered(t, e1, e2, func(c int) bool { return c > 0 }, "greater than", msgAndArgs) {
		t.FailNow()
	}
}

func Less(t TestingT, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Ordered(t, e1, e2, func(c int) bool { return c < 0 }, "less than", msgAndArgs) {
		t.FailNow()
	}
}

func Condition(t TestingT, comp func() bool, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Condition(t, comp, msgAndArgs) {
		t.FailNow()
	}
}

func Contains(t TestingT, s, contains any, msgAndArgs ...any) {
	t.Helper()
	if !testutil.Contains(t, s, contains, msgAndArgs) {
		t.FailNow()
	}
}

func ErrorContains(t TestingT, err error, contains string, msgAndArgs ...any) {
	t.Helper()
	if !testutil.ErrorContains(t, err, contains, msgAndArgs) {
		t.FailNow()
	}
}

func Truef(t TestingT, value bool, msg string, args ...any) {
	t.Helper()
	if !testutil.True(t, value, append([]any{msg}, args...)) {
		t.FailNow()
	}
}

// Assertions binds the checks to a test, as testify's require.New does.
type Assertions struct{ t TestingT }

func New(t TestingT) *Assertions { return &Assertions{t} }

func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) {
	a.t.Helper()
	Equal(a.t, expected, actual, msgAndArgs...)
}

func (a *Assertions) Nil(object any, msgAndArgs ...any) {
	a.t.Helper()
	Nil(a.t, object, msgAndArgs...)
}

func (a *Assertions) NotNil(object any, msgAndArgs ...any) {
	a.t.Helper()
	NotNil(a.t, object, msgAndArgs...)
}
