package errors

// String provides a simple mechanism to define const errors. This enable packages to export simple errors using
//
//	const ErrNoDroids = errors.String("these are not the droids you're looking for")
type String string

func (s String) Error() string { return string(s) }

// Errorf returns an error which satisfies errors.Is(ex, s), without
// necessarily containing the text of string s.
func (s String) Errorf(format string, a ...interface{}) error {
	return errorString{
		error: Errorf(format, a...),
		s:     s,
	}
}

// Wrap returns nil when passed nil, otherwise it returns an error such
// that [errors.Is] returns true when compared to the String error or the
// passed in error.  It leaves the error text unchanged.
func (s String) Wrap(err error) error {
	if err == nil {
		return nil
	}
	return errorString{
		error: WithStack(err),
		s:     s,
	}
}

type errorString struct {
	error
	s String
}

// Is returns true if the target matches the same underlying
// String value (compared as a string)
func (e errorString) Is(target error) bool {
	return target == e.s
}

func (e errorString) Unwrap() error {
	return e.error
}
