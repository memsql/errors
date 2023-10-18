package errors

import (
	"fmt"
	"log"
	"sync/atomic"
)

// A Throttle will alert, until threshold is reached. After threshold is reached, errors are no longer
// alerted. See Throttle.Alert() for additional details. A throttle allows you to capture the first error(s)
// encountered on a given line of code, but not subsequent errors.
//
// Use a throttle when you believe that an error warrants investigation if it ever occurs, but it is not
// necessary to capture every time it occurs.
//
// The throttle is not persisted across restarts, so errors will be captured for each replica of a service and
// each time a replica is restarted. So, if you specify a Threshold of one, you might see two captures if the
// service has two replicas, or four after those replicas have restarted, etc.
type Throttle struct {
	Scope     string
	Threshold int32
	count     int32
}

func (t *Throttle) Alertf(format string, a ...interface{}) error {
	// use fmt.Errorf here, to avoid a stack that is redundant with stack produced in ForceAlert
	return t.Alert(fmt.Errorf(format, a...))
}

// Alert will capture an exception identically to errors.Alert, until some threshold number of errors has been
// alerted. After that threshold amount, subsequent errors are returned without capture.
//
// The goal is to log and capture errors, if they occur; while not capturing so many that noise exceeds signal
// in logs and sentry.
func (t *Throttle) Alert(exception error) error {
	if exception == nil {
		return nil
	}

	count := atomic.AddInt32(&t.count, 1)
	if count <= t.Threshold {
		return Alert(exception)
	}

	log.Printf("throttled an alert (%q) because threshold (%d) is reached (%d): %+v", t.Scope, t.Threshold, count, exception)

	// reset every once in a while so that capture is not totally silent despite thousands of errors.
	if count == 1_000 {
		Alert(fmt.Errorf("throttled excessive errors (%d in scope %q)", count, t.Scope)) //nolint:errcheck
		atomic.StoreInt32(&t.count, 0)
	}

	// return original exception, not alerted
	return exception
}
