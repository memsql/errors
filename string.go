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

type errorString struct {
	error
	s String
}

func (e errorString) Is(target error) bool {
	return target == e.s
}
