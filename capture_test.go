package errors_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/memsql/errors"
	"github.com/stretchr/testify/assert"
)

func TestAlert(t *testing.T) {
	var err error

	// Alert of nil should return nil
	assert.NoError(t, errors.Alert(err)) // nil

	err = errors.New("TestAlert")
	captured := errors.Alert(err)
	assert.Error(t, captured)
}

func TestCaptureArg(t *testing.T) {
	arg := []any{"one", "two"} // arbitrary

	var captured bool
	errors.RegisterCapture("TestCaptureArg", func(_ error, got ...any) errors.CaptureID {
		if diff := cmp.Diff(arg, got); diff != "" {
			t.Errorf("unexpected capture args:\n%s", diff)
		} else {
			t.Log("captured args as expected")
		}
		captured = true
		return "TestCaptureArg"
	})
	defer errors.UnregisterCapture("TestCaptureArg")

	_ = errors.Alertf("TestCaptureArg %v %v", arg...)
	assert.True(t, captured)
}

func TestCaptureLog(t *testing.T) {
	errors.RegisterCapture("capture_test", errors.LogCapture)
	defer errors.UnregisterCapture("capture_test")

	err := fmt.Errorf("this error text should be captured to log")
	captured := errors.Alert(err).(*errors.Captured)
	id := captured.ID("capture_test")

	// the captured ID should appear in message
	for _, format := range []string{"%s", "%q", "%v", "%+v"} {
		msg := fmt.Sprintf(format, captured)
		if !strings.Contains(msg, string(id)) {
			t.Errorf("captured error message (%q) does not contain capture ID (%q)", msg, id)
		}
	}

	time.Sleep(2 * time.Second) // enough time for a new log timestamp

	// a re-capture should have new id
	err2 := errors.Alertf("recapture: %w", captured)
	if errors.As(err2, &captured) {
		assert.NotEqual(t, id, captured.ID("capture_test"))
	} else {
		t.Error("result of errors.Alertf is not an errors.Captured")
	}

	// a new capture should have a new id
	err = errors.New("this error text, and stack trace, should be captured to log")
	captured = errors.Alert(err).(*errors.Captured)
	if captured.ID("capture_test") == id {
		t.Errorf("duplicate capture id (%q)", id)
	}
}

// TestCaptureRecurse checks that while a call to Alert succeeds, an Alert from that alert's handler will not.
func TestCaptureRecurse(t *testing.T) {
	depth := 0
	errors.RegisterCapture("recursive", func(exception error, arg ...interface{}) errors.CaptureID {
		if depth > 0 {
			// if recursing, break
			t.Errorf("recursive capture #%d", depth)
			return errors.CaptureID(fmt.Sprintf("recursion %d", depth))
		}
		depth++

		got := errors.Alertf("recursive alert, should fail")
		var captured *errors.Captured
		if errors.As(got, &captured) {
			t.Errorf("recursion not detected")
		}

		return errors.CaptureID(fmt.Sprintf("recursion %d", depth))
	})
	defer errors.UnregisterCapture("recursive")

	got := errors.Alertf("top level alert, should succeed")
	var captured *errors.Captured
	if !errors.As(got, &captured) {
		t.Errorf("alert did not capture")
	}
}
