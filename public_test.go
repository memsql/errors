package errors_test

import (
	"testing"

	"github.com/memsql/errors"
)

func TestRedact(t *testing.T) {
	table := []struct {
		error
		string
	}{
		{
			errors.Errorf("something bad happened (%s), and also (%s)", "secret 1", "secret 2"),
			"something bad happened, and also",
		},
	}

	for i := range table {
		redacted := errors.Redact(table[i].error)
		if redacted.Error() != table[i].string {
			t.Errorf("errors.Redact() converted %q into %q (wanted %q)", table[i].error, redacted, table[i].string)
		}
	}
}
