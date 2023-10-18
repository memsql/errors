package errors_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/memsql/errors"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestThrottle(t *testing.T) {
	// test needs at least one capturer registered
	errors.RegisterCapture("throttle_test", errors.LogCapture)
	defer errors.UnregisterCapture("throttle_test")

	// this test should pass with any threshold > 0
	throttle := errors.Throttle{Scope: "TestThrottle", Threshold: random.Int31n(10) + 1}

	var captured *errors.Captured

	for i := int32(1); i <= throttle.Threshold; i++ {
		exception := throttle.Alertf("number %d, should not be throttled", i)
		if !errors.As(exception, &captured) {
			t.Errorf("throttle did not capture (%T): %+v", exception, exception)
		}
	}

	exception := throttle.Alertf("number %d, should be throttled (not captured)", throttle.Threshold+1)
	if errors.As(exception, &captured) {
		t.Errorf("throttle did capture (%T): %+v", exception, exception)
	}
}
