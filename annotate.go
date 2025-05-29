package errors

func Annotate(err error, values ...any) error {
	if err == nil {
		return nil
	}
	if len(values) == 0 {
		return err
	}
	return &Error{
		error: err,
		arg:   values,
	}
}

// Annotation looks at the arguments supplied to Errrof(), Wrapf(), and Annotate(),
// looking to see if any of them match the type T. If so, it returns the value and true.
// Otherwise it returns an empty T and false.
func Annotation[T any](err error) (T, bool) {
	var rv T
	var found bool
	Walk(err, func(ex error) bool {
		withArg, ok := ex.(*Error)
		if !ok {
			return true
		}
		for _, arg := range withArg.arg {
			if v, ok := arg.(T); ok {
				rv = v
				found = true
				return false
			}
		}
		return true
	})
	return rv, found
}
