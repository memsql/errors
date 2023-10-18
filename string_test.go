package errors

import (
	"testing"
)

func TestStringIs(t *testing.T) {
	const myErr String = "custom type of error"
	ex := myErr.Errorf("%s, we have a problem", "houston")
	if ex.Error() != "houston, we have a problem" {
		t.Errorf("mismatched text, have %q", ex.Error())
	}
	if !Is(ex, myErr) {
		t.Errorf("exception (%T) is not myErr (%T)", ex, myErr)
	}
}
