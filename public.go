package errors

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// parenReg matches parentheticals and preceding space.
var parenReg = regexp.MustCompile(` \([^()]*\)`)

type Public struct {
	msg string
	error
}

func (e Public) Error() string { return e.msg }

func (e Public) Unwrap() error { return e.error }

// Redact removes potential sensitive details from an error, making the message safe to display to an
// unprivileged user.
//
// Redact removes content in parenthesis.  That is, it expects only errors that follow the convention that
// potentially sensitive information appears in parentheses. Also that errors are relatively simple,
// i.e. without nested parentheses.
func Redact(err error) Public {
	p, ok := err.(Public)
	if ok {
		return p
	}

	long := err.Error()

	// remove the parts in parens
	long = parenReg.ReplaceAllString(long, "")

	// truncate at the first colon (shows the top error an not lower-level detail)
	split := strings.SplitN(long, ":", 2)
	short := split[0] // part preceding first ":"

	// append any capture IDs
	captured := &Captured{}
	if errors.As(err, &captured) {
		short = fmt.Sprintf("%s [%s]", short, captured.allID())
	}

	return Public{short, err} // public error is stripped of all dynamic detail
}
