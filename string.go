package errors

// String provides a simple mechanism to define const errors. This enable packages to export simple errors using
//
//	const ErrNoDroids = errors.String("these are not the droids you're looking for")
type String string

func (s String) Error() string { return string(s) }

// Is returnes true if the target is the same underlying
// String value (compared as a string)
func (s String) Is(target error) bool {
	return s == target
}

// Errorf returns an error which satisfies errors.Is(ex, s), without
// necessarily containing the text of string s.
func (s String) Errorf(format string, a ...interface{}) error {
	return errorString{
		error: Errorf(format, a...),
		s:     s,
	}
}

// Wrap preserves the existing error's string value. It just serves as
// a way to annotate an error so that errors.Is will return true when
// compared to the same base string
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

// Is returnes true if the target matches the same underlying
// String value (compared as a string)
func (e errorString) Is(target error) bool {
	return target == e.s
}

func (e errorString) Unwrap() error {
	return e.error
}
