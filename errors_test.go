package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/memsql/errors"

	"github.com/stretchr/testify/assert"
)

// TestError tests very basic features of errors.Errorf().
func TestError(t *testing.T) {
	var inner error = errors.Errorf("inner")
	assert.Equal(t, "inner", inner.Error())

	var withStack errors.StackTracer
	if !errors.As(inner, &withStack) {
		t.Errorf("errors.Errorf produced %T, without stack trace", inner)
	}

	outer := errors.Errorf("outer: %w", inner)
	assert.Equal(t, "outer: inner", outer.Error())

	isWrapped := false
	for cursor := error(outer); cursor != nil; cursor = errors.Unwrap(cursor) {
		if errors.Is(cursor, inner) {
			isWrapped = true
			break
		}
	}
	if !isWrapped {
		t.Errorf("outer error (%s) does not wrap inner error (%s)", outer, inner)
	}
}

func TestJoin(t *testing.T) {
	// all args should be passed in the final combined alert
	arg := []int{1, 2, 3}

	var exception error
	for i := range arg {
		if i == 0 {
			exception = errors.Errorf("TestJoin (%d)", arg[i])
		} else {
			exception = errors.Join(errors.Errorf("TestJoin (%d)", arg[i]), exception)
		}
	}

	errors.RegisterCapture("TestJoin", func(err error, have ...any) errors.CaptureID {
		// we should have all the args
		if len(arg) != len(have) {
			t.Errorf("want %d args, have %d", len(arg), len(have))
		}
	argLoop:
		for i := range arg {
			for j := range have {
				v, ok := have[j].(int)
				if !ok {
					continue
				}
				if v == arg[i] {
					continue argLoop
				}
			}
			t.Errorf("want arg (%d), not passed to capture handler", arg[i])
		}

		return "TestJoin"
	})
	defer errors.UnregisterCapture("TestJoin")
	_ = errors.Alert(exception) //nolint:errcheck // this is so our capture handler (above) gets called
}

func TestExpandArg(t *testing.T) {
	var err error

	needles := 0
	errors.RegisterCapture("TestExpandArg", func(exception error, arg ...any) errors.CaptureID {
		for _, a := range arg {
			if str, ok := a.(string); ok {
				if str == "needle" {
					needles++
				}
			}
		}
		return "TestExpandArg"
	})
	defer errors.UnregisterCapture("TestExpandArg")

	err = func() (err error) {
		defer errors.Expand(&err, "expanded (%s)", "needle") // we want this arg to make it to the capturer
		panic("TestExpandArg")
	}()
	assert.Error(t, err)
	assert.Equal(t, 1, needles, "expected argument to be passed to capture handler")
}

func TestExpandPanic(t *testing.T) {
	t.Parallel()
	var err error

	err = func() (err error) {
		defer errors.Expand(&err, "expanded text")
		dontKeepCalmAndCarryOn("panic text")
		return nil
	}()

	assert.Equal(t, "expanded text: panic text", err.Error())

	// confirmation that the stack trace includes the source of the panic
	verbose := fmt.Sprintf("%+v", err)
	if !strings.Contains(verbose, "dontKeepCalmAndCarryOn") {
		t.Errorf("expected stack trace to include panic, got:\n %s\n", verbose)
	}

	err = func() (err error) {
		defer errors.Expand(&err, "expanded text")
		panic(errors.New("panic error"))
	}()

	assert.Equal(t, "expanded text: panic error", err.Error())
}

func dontKeepCalmAndCarryOn(s string) {
	panic(s)
}

func TestExpungeArg(t *testing.T) {
	var err error

	needles := 0
	errors.RegisterCapture("TestExpungeArg", func(exception error, arg ...any) errors.CaptureID {
		for _, a := range arg {
			if str, ok := a.(string); ok {
				if str == "needle" {
					needles++
				}
			}
		}
		return "TestExpungeArg"
	})
	defer errors.UnregisterCapture("TestExpungeArg")

	err = func() (err error) {
		defer errors.Expunge(&err, "expunged (%s)", "needle") // we want this arg to make it to the capturer
		panic("TestExpungeArg")
	}()
	assert.Error(t, err)
	assert.Equal(t, 1, needles, "expected argument to be passed to capture handler")
}

func TestFromPanic(t *testing.T) {
	t.Parallel()
	needle := "needle"

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		defer func() {
			err := errors.FromPanic(recover())
			assert.Contains(t, err.Error(), needle)
		}()
		panic(needle)
	})

	t.Run("fmt.Stringer", func(t *testing.T) {
		t.Parallel()
		var s fmt.Stringer = myStringer{}
		defer func() {
			err := errors.FromPanic(recover())
			assert.Contains(t, err.Error(), s.String())
		}()
		panic(s)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		defer func() {
			err := errors.FromPanic(recover())
			assert.Contains(t, err.Error(), needle)
		}()
		panic(errors.New(needle))
	})
}

type myStringer struct{}

func (s myStringer) String() string { return "hello world" }
