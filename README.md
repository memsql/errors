# errors - API for creating exceptions and alerting

[![GoDoc](https://godoc.org/github.com/memsql/errors?status.svg)](https://pkg.go.dev/github.com/memsql/errors)
![unit tests](https://github.com/memsql/errors/actions/workflows/go.yml/badge.svg)
[![report card](https://goreportcard.com/badge/github.com/memsql/errors)](https://goreportcard.com/report/github.com/memsql/errors)
[![codecov](https://codecov.io/gh/memsql/errors/branch/main/graph/badge.svg)](https://codecov.io/gh/memsql/errors)

Install:

	go get github.com/memsql/errors

---

Package errors provides an API for creating exceptions and alerting.

This is intended to be a replacement for Go's standard library errors package.
You can import "github.com/memsql/errors" instead of "errors".

# Verbose Messages

Use New or Errorf to produce an error that can be formatted with verbose
details, including a stack trace. To see the verbose details, format using `%+v`
instead of `%v`, `%s` or calling err.Error().

# Alert and Capture

Applications can call RegisterCapture to persist errors to logs, a backend
database, or a third-party service. When Alert or Alertf is called, the error
will be passed to each registered handler, which is responsible for peristing.
The capture handler returns an identifier which becomes part of the error
message, as a suffix enclosed in square brackets.

# Expand and Expunge

The Expand helper adds information to an error, while Expunge is intended to
remove information that end users don't need to see. Both are intended to be
deferred, using a pattern like this,

    func FindThing(id string) (err error) {
      // include the id, and verbose stack, with all errors
      defer errors.Expand(&err, "thing (%q) not found", id)

      // ...
    }

When returning errors which wrap other errors, Expunge can be used to hide
underlying details, to make a more user-friendly message. It assumes that error
message text follows specific conventions, described below.

# Message Conventions

Error messages include static parts and dynamic parts. If the error is
“/tmp/foo.txt not found”, then “/tmp/foo.txt” is the dynamic part and “not
found” is the static part. The dynamic parts present detail to the reader. The
static part is useful as well. For example, to determine the line of code where
it originated, or how frequently the error is triggered. This package follows a
convention to easily work with both static and dynamic parts of error messages:

* Dynamic text SHOULD appear in parenthesis.

* Static text SHOULD be grammatically correct, with or without the dynamic
parts.

Following these guidelines, the backend might produce an error: “file
("/tmp/foo.txt”) not found”.

The static text should make sense even when the dynamic parts are removed.
Imagine running a whole log file through a regular expression that removes the
parentheticals. This stripped log file should still make sense to the reader.
In our example, the stripped text would be “file not found”, which makes sense
(while just “not found” would not make sense).

The colon character “:” has special significance in error message. It implies
that an error message has been “wrapped” with another message, providing
additional context.

    // avoid this!
    return errors.Errorf("invalid widget id: %s", id)

    // do this
    return errors.Errorf("failed to parse widget id (%q): %w", id, err)

* Error messages SHOULD use the colon “:” only when wrapping another error.

