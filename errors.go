package errors

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	pkgerrors "github.com/pkg/errors"
)

var (
	// Expose several helpers implemented by stdlib "errors", so that callers do not have to import multiple
	// "errors" packages.

	As     = errors.As
	Is     = errors.Is
	Join   = errors.Join
	Unwrap = errors.Unwrap
)

type StackTrace = pkgerrors.StackTrace

// Error implements Go's error interface; and can format verbose messages, including stack traces.
type Error struct {
	// error wraps an arbitrary error
	error

	// arg records the arguments used to construct an error message; it serves as metadata about the error
	arg []interface{}
}

// Unwrap allows errors.Unwrap to return the parent error.
func (e *Error) Unwrap() error { return e.error }

// Format is implemented to produce the most verbose version of the error message, in particular when "%+v" is
// the format string.
func (e *Error) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		// start with the error message
		_, _ = io.WriteString(f, e.Error()) // if this fails, not much we can do

		if f.Flag('+') {

			// If we've wrapped an error that implements fmt.Formatter, we defer to it's Format() behavior for
			// verbose output.  Typically this adds a stack trace, although it might add more details. It also
			// will display the wrapped message, which is usually redundant, included in the error message we've
			// already written. So the code that follows take steps to remove the redundant information.  We also
			// remove parts of stack traces that are internal to this package, as they may add several lines of
			// unimportant information that distracts from the real points of interest in the stack.

			var formatter interface {
				fmt.Formatter
				error
			}
			if As(e.error, &formatter) {
				// the wrapped error implements Format, so it might add additional verbose details
				buf := &bytes.Buffer{}
				_, _ = fmt.Fprintf(buf, "%+v", formatter)

				// omit leading lines that repeat text that already appears in error message
				// omit leading stack within this package
				leading := true
				scanner := bufio.NewScanner(buf)
				for scanner.Scan() {
					line := scanner.Text()
					if leading {
						if strings.Contains(e.Error(), line) {
							// line is redunant, a portion of the error message
							continue
						}
						if strings.HasPrefix(line, "github.com/memsql/errors.") {
							// line is stack trace within this package, not relevant to the human inspecting the stack
							if !scanner.Scan() { // skip two lines of stack trace
								break
							}
							continue
						}
					}
					_, _ = io.WriteString(f, "\n"+line) // if this fails, not much we can do
					leading = false
				}
			}

		}
	case 's':
		_, _ = fmt.Fprintf(f, "%s", e.error)
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", e.error)
	}
}

// New emulates the behavior of stdlib's errors.New(), and includes a stack trace with the error.
func New(text string) error {
	return WithStack(errors.New(text))
}

// FromPanic produces an error when passed non-nil input. It accepts input of any type, in order to support being
// invoked with what is returned from recover().
//
//	defer func() {
//	   if err = errors.FromPanic(recover()); err != nil { ... }
//	}()
func FromPanic(in interface{}) (exception error) {
	if in == nil {
		return nil
	}

	switch v := in.(type) {
	case error:
		return WithStack(v)
	case fmt.Stringer:
		return New(v.String())
	case string:
		return New(v)
	default:
		exception = Errorf("%+v", v)
	}
	return exception
}

// Errorf produces an error with a formatted message including dynamic arguments.
//
// Callers are encouraged to include all relevant arguments in a
// well-constructed error message. Most arguments are stored to
// provide additional metadata if the error is later
// captured. Arguments will not be stored when apparently redundant,
// for example a wrapped error or string included in error text.
func Errorf(format string, a ...interface{}) *Error {
	exception := &Error{
		error: WithStack(fmt.Errorf(format, a...)),
		arg:   a,
	}

	// if wrapping an error, no need to include it in args
	if strings.HasSuffix(format, " %w") && len(exception.arg) > 0 {
		exception.arg = exception.arg[:len(exception.arg)-1]
	}

	// if arg is text of the error, not need to include it in args
	if strings.HasPrefix(format, "%s") && len(exception.arg) > 0 {
		exception.arg = exception.arg[1:]
	}

	return exception
}

// WithStack produces an error that includes a stack trace.  Note that if the wrapped error already has a stack,
// that error is returned without modification.  Thus only the first call to WithStack will produce a stack
// trace. In other words when an error is wrapped multiple times, it is the stack of the earliest wrapped error
// which has priority.
//
// To add information to an error message, use Errorf() instead. This function is provided to add a stack trace
// to a third-party error without otherwise altering the error text.
func WithStack(err error) error {
	if err == nil {
		return nil
	}

	var withStack StackTracer

	if errors.As(err, &withStack) {
		// already is/wraps an error with stack trace
		return err
	}

	// use the stack implementation from github.com/pkg/errors (in the future we may prefer runtime/debug)
	return pkgerrors.WithStack(err)
}

// StackTracer is exported so that external packages can detect whether a err has stack trace associated.
type StackTracer interface {
	StackTrace() pkgerrors.StackTrace
}

// Wrap returns nil when the exception passed in is nil; otherwise, it returns an error with message text that wraps exception.
//
// This function provides an alternative to
//
//	err = f()
//	if err != nil {
//	  return errors.Errorf("failed objective: %w", err)
//	}
//	return nil
//
// can be written as
//
//	return errors.Wrap(f(), "failed objective")
func Wrap(exception error, message string) error {
	if exception == nil {
		return nil
	}
	return Errorf("%s: %w", message, exception)
}

// Wrapf returns nil when the exception passed in is nil; otherwise, it produces text based on the format string
// and arguments, and returns an error with that text that wraps the exception.
//
// See Wrap() for rationale.
func Wrapf(exception error, format string, a ...interface{}) error {
	if exception == nil {
		return nil
	}
	return Errorf(format+": %w", append(a, exception)...)
}

// Expand rewites an error message, when an error is non-nil.
//
// This is intended to be invoked as a deferred function, as a convenient way to add details to an error
// immediately before returning it.
func Expand(exception *error, format string, a ...interface{}) {
	recovered := false
	if *exception == nil {
		*exception = FromPanic(recover())
		recovered = true
	}
	if *exception == nil {
		return // nothing to do
	}
	*exception = Errorf(format+": %w", append(a, exception)...)

	if recovered {
		*exception = Alert(*exception)
	}
}

// Expunge rewrites an error message, when an error is non-nil.  It removes potentially sensitive details from
// the exception and makes it less verbose, removing text of wrapped errors. It relies on text conventions, see Redact().
//
// Details are redacted from the exception passed in.  The format and arguments passed in become part of the new
// error message. Nothing is redacted from those additional details.
//
// This is intended to be invoked as a deferred function, as a convenient way to remove details from an error
// immediately before returning it from a public API.
func Expunge(exception *error, format string, a ...interface{}) {
	recovered := false
	if *exception == nil {
		*exception = FromPanic(recover())
		recovered = true
	}
	if *exception == nil {
		return // nothing to do
	}

	ex := Errorf("%s: %w", fmt.Sprintf(format, a...), Redact(*exception))
	ex.arg = append(ex.arg, a...)
	*exception = ex

	if recovered {
		*exception = Alert(*exception)
	}
}

// ExpungeOnce behaves like Expunge(), except that it leaves an exception as-is if the it has already been expunged.
//
// This is useful when a function has multiple stages, during which different details should be included in the
// exception.  Deferred functions are called in a specific order, and only one deferred ExpungeOnce() will
// affect the error. In other words, the last reached deferred ExpungeOnce() will determine the final error
// message.
func ExpungeOnce(exception *error, format string, a ...interface{}) {
	if *exception == nil {
		*exception = Alert(FromPanic(recover()))
	}
	if *exception == nil {
		return // nothing to do
	}

	var redacted Public
	if As(*exception, &redacted) {
		// already redacted/expunged, leave as is
		return
	}

	Expunge(exception, format, a...)
}
