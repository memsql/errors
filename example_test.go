package errors_test

import (
	"fmt"

	"github.com/memsql/errors"
)

func ExampleExpand() {
	// mustBeEven is at odds with odds.
	mustBeEven := func(input int) error {
		if (input % 2) == 1 {
			return errors.New("not even")
		}
		return nil
	}

	// processEvenNumber uses errors.Expand() to add details when returning non-nil error.
	processEvenNumber := func(input int) (err error) {
		//
		// Expand() adds details to non-nil err, leaves nil error as-is.
		//
		defer errors.Expand(&err, "failed to process number (%d)", input)

		return mustBeEven(input)
	}

	err := processEvenNumber(1) // fails because not even
	fmt.Println(err)
	err2 := processEvenNumber(42) // succeeds
	fmt.Println(err2)
	// Output:
	// failed to process number (1): not even
	// <nil>
}

func ExampleExpunge() {
	// mustBeEven is at odds with odds.
	mustBeEven := func(input int) error {
		if (input % 2) == 1 {
			return errors.New("not even")
		}
		return nil
	}

	// processEvenNumber uses errors.Expand() to add details when returning non-nil error.
	processEvenNumber := func(input int) (err error) {
		defer errors.Expand(&err, "failed to process number (%d)", input)

		return mustBeEven(input)
	}

	// ProcessSecretNumber uses errors.Expunge to control details presented when returning non-nil error from a
	// public-facing API.
	processSecretNumber := func(input int) (err error) {
		//
		// Expunge() redacts details from non-nil err, leaves nil error as-is.
		//
		defer errors.Expunge(&err, "unexpected error")

		return processEvenNumber(input)
	}

	err := processSecretNumber(1) // fails because not even
	fmt.Println(err)

	err2 := processSecretNumber(42) // succeeds
	fmt.Println(err2)
	// Output:
	// unexpected error: failed to process number
	// <nil>
}

func ExampleExpungeOnce() {
	// mustBeEven is at odds with odds.
	mustBeEven := func(input int) error {
		if (input % 2) == 1 {
			return errors.New("not even")
		}
		return nil
	}

	// processEvenNumber uses errors.Expand() to add details when returning non-nil error.
	processEvenNumber := func(input int) (err error) {
		defer errors.Expand(&err, "failed to process number (%d)", input)

		return mustBeEven(input)
	}

	// ProcessSecretNumber uses errors.Expunge to control details presented when returning non-nil error from a
	// public-facing API.
	processSecretNumber := func(input int) (err error) {
		//
		// ExpungeOnce() redacts details from non-nil err, leaves nil error as-is. When multiple ExpungeOnce() are
		// deferred, only one will affect the error.
		//
		defer errors.ExpungeOnce(&err, "secret number must be even")
		if err = processEvenNumber(input); err != nil {
			return err
		}

		defer errors.ExpungeOnce(&err, "secret number must be divisible by 4")
		return processEvenNumber(input / 2)
	}

	err := processSecretNumber(1) // fails because not even
	fmt.Println(err)

	err2 := processSecretNumber(42) // fails because not divisible by 4
	fmt.Println(err2)
	// Output:
	// secret number must be even: failed to process number
	// secret number must be divisible by 4: failed to process number
}
