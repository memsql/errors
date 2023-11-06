package errors

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
)

// CaptureTimeout limits how long to wait for a capture ID to be returned from a capture handler.
var CaptureTimeout = 500 * time.Millisecond

type CaptureProvider string // i.e. "sentry"

type CaptureID string // may be a URL or any string that allows a captured error to be looked up

// CaptureFunc is a handler invoked to capture an error.
//
// Any mechanism that can save an error may provide a capture handler.  For example sentry, a log, etc... The
// CaptureID returned should be a way to find the error among other errors captured by the mechanism.
type CaptureFunc func(err error, arg ...interface{}) CaptureID

// capture tracks registered capture handlers.
var capture = map[CaptureProvider]CaptureFunc{}

// RegisterCapture adds a handler to the set that will be invoked each time an error is captured.
func RegisterCapture(name CaptureProvider, handler CaptureFunc) {
	if capture[name] != nil {
		log.Panicf("capture provider (%q) already registered", name)
	}

	capture[name] = handler
}

func UnregisterCapture(name CaptureProvider) {
	delete(capture, name)
}

// Captured marks and wraps an error that has been "captured", meaning it has been logged verbosely or stored in
// a way that can be looked up later.
type Captured struct {
	// error is the wrapper error
	error

	// id is a list of capture IDs, by provider
	id map[CaptureProvider]CaptureID
}

// Unwrap allows errors.Unwrap to return the parent error.
func (e *Captured) Unwrap() error { return e.error }

// Format produces a message with capture ID appended. The intention is to allow the captured details to be
// looked up, by engineers with access to the capture mechanism.
func (e *Captured) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%+v [%s]", e.error, e.allID())
			return
		}
		fallthrough
	case 's':
		fmt.Fprintf(f, "%s [%s]", e.error, e.allID())
	case 'q':
		fmt.Fprintf(f, "%q [%s]", e.error, e.allID())
	}
}

func (e *Captured) allID() string {
	id := make([]string, 0, len(e.id))
	for i := range e.id {
		id = append(id, string(e.id[i]))
	}
	return strings.Join(id, ", ")
}

// ID returns an identifier created when a capture handler recorded the error.
func (e *Captured) ID(provider CaptureProvider) CaptureID {
	id := e.id[provider]
	return id
}

// Alert sends an error to all registered capture handlers. Capture handlers produce verbose logs and alerts.
// This should be called only for errors that require human attention to address (our developers or SREs). It
// should not be called for run-of-the-mill errors that are handled in code or returned to portal users.
func Alert(err error) error {
	if err == nil {
		return nil
	}

	if len(capture) == 0 { // no capture handlers
		log.Printf("alert not captured: %+v", err)
		return WithStack(err)
	}

	return alert(err)
}

// Alertf produces an error and alerts. It is equivalent to calling Errorf() and then Alert().
func Alertf(format string, a ...interface{}) error {
	exception := &Error{
		// use fmt.Errorf here, to avoid a stack that is redundant with stack produced in alert()
		error: fmt.Errorf(format, a...),
		// don't lose track of arguments, as capture handlers may use them
		arg: a,
	}

	return alert(exception)
}

func alert(exception error) error {
	if exception == nil {
		return nil
	}

	// When alerting, we invoke registered handlers.  If those handlers in turn call (Force)Alert, we could get an
	// infinite recursion. Here, we try to prevent that. This is relatively expensive, but we're alerting, which
	// shouldn't happen often.
	pc := make([]uintptr, 42)
	runtime.Callers(1, pc) // skip 1 (the one skipped is runtime.Callers)
	cf := runtime.CallersFrames(pc)
	us, _ := cf.Next()
	for them, ok := cf.Next(); ok; them, ok = cf.Next() {
		// use HasPrefix here, not simple equality, because handlers are called from goroutine (below)
		if strings.HasPrefix(them.Func.Name(), us.Func.Name()) {
			log.Printf("cannot alert, recursion detected (%s): %+v", us.Func.Name(), exception)
			return exception // don't recurse again
		}
	}

	// pkgerrors.WithStack provides a stack trace to this alert call,
	// even if the wrapped error already has a stack.
	exception = pkgerrors.WithStack(exception)

	e := &Captured{
		error: exception,
		id:    map[CaptureProvider]CaptureID{},
	}

	// pass args to hander, if any
	var arg []any
	Walk(exception, func(ex error) bool {
		// we don't use As() here, because it could skip over joined errors, instead we walk the entire error tree.
		withArg, ok := ex.(*Error)
		if !ok {
			return true
		}

		arg = append(arg, withArg.arg...)
		return true
	})

	// Run handlers in goroutines, so that if one handler is deadlocked
	// it does not prevent others from running, or us from returning.
	
	timer := time.NewTimer(CaptureTimeout)
	defer timer.Stop()

	done := make(chan struct{})
	finish := sync.OnceFunc(func() {close(done)})
	var mu sync.Mutex
	
	// start a goroutine for each handler
	for provider, handler := range capture {
		provider := provider
		handler := handler
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("failed to capture exception (%q): %+v", provider, r)
				}
			}()

			id := handler(exception, arg...)

			mu.Lock()
			defer mu.Unlock()
			select {
			case <-done:
				// we are too late
			default:
				e.id[provider] = id
				if len(e.id) == len(capture) {
					finish()
				}
			}
		}()
	}

	// wait until done or timed out
waitLoop:
	for {
		select {
		case <- timer.C:
			mu.Lock()
			defer mu.Unlock()
			finish()
		case <- done:
			break waitLoop
		}
	}
	
	return e
}

// Walk visits each error in a tree of errors wrapping other errors.
//
// The handler func, f, takes in the error being visited.  The walk
// continues if the handler returns true, and does not continue if the
// handler returns false.
func Walk(exception error, f func(error) bool) {
	walk(exception, f)
}

func walk(exception error, f func(error) bool) bool {
	type join interface {
		Unwrap() []error
	}
	for exception != nil {
		ok := f(exception)
		if !ok {
			return false
		}

		if j, isJoin := exception.(join); isJoin {
			// if exception is a join, walk each
			for _, ex := range j.Unwrap() {
				ok := walk(ex, f)
				if !ok {
					return false
				}
			}
			return true
		}

		// if not a join, descend and continue loop
		exception = Unwrap(exception)
	}
	return true
}

// LogCapture is a simple capture handler that writes exception to log.
func LogCapture(exception error, arg ...interface{}) CaptureID {
	log.Printf("%+v", exception)
	return CaptureID(time.Now().Format("2006/01/02 15:04:05")) // use time as identifier, to help find the message in the log
}
