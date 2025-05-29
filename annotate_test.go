package errors_test

import (
	"fmt"
	"testing"

	"github.com/memsql/errors"
	"github.com/stretchr/testify/require"
)

/*
type empty struct{}
type EmptyExample = WithPayload[empty]
var NewEmpty = errors.NewEmptyPayload[empty]

type value struct{
	value int
}
type ValueExample = WithPayload[value]
var NewValue = errors.NewWithPayload[T]

func TestPayloads(t *testing.T) {
	nilEmpty := NewEmpty(nil)
}

*/

func TestStringAsMarker(t *testing.T) {
	const signalOne errors.String = "signal one"
	const signalTwo errors.String = "signal two"

	err := errors.WithStack(fmt.Errorf("a test error"))
	require.False(t, errors.Is(err, signalOne), "non-signaled errors are not the signal error")
	require.NoError(t, signalOne.Wrap(nil), "wrapped nil is nil")

	require.True(t, errors.Is(signalOne.Wrap(err), signalOne), "wrapped errors are the signal error (Wrap)")
	require.False(t, errors.Is(signalTwo.Wrap(err), signalOne), "one signal error is not another (Wrap)")
	require.False(t, errors.Is(signalOne, signalTwo), "raw signal are not each other (Wrap)")
	require.True(t, errors.Is(signalOne, signalOne), "raw signal is itself (Wrap)")
	require.True(t, errors.Is(signalTwo.Wrap(signalOne.Wrap(err)), signalOne), "double wrapped is the signal #1 (Wrap)")
	require.True(t, errors.Is(signalOne.Wrap(signalTwo.Wrap(err)), signalOne), "double wrapped is the signal #2 (Wrap)")
	require.Equal(t, "a test error", signalOne.Wrap(signalTwo.Wrap(err)).Error(), "the string is untouched (Wrap)")

	require.True(t, errors.Is(signalOne.Errorf("%w", err), signalOne), "wrapped errors are the signal error (Errorf)")
	require.False(t, errors.Is(signalTwo.Errorf("%w", err), signalOne), "one signal error is not another (Errorf)")
	require.False(t, errors.Is(signalOne, signalTwo), "raw signal are not each other (Errorf)")
	require.True(t, errors.Is(signalOne, signalOne), "raw signal is itself (Errorf)")
	require.True(t, errors.Is(signalTwo.Errorf("%w", signalOne.Errorf("%w", err)), signalOne), "double wrapped is the signal #1 (Errorf)")
	require.True(t, errors.Is(signalOne.Errorf("%w", signalTwo.Errorf("%w", err)), signalOne), "double wrapped is the signal #2 (Errorf)")
	require.Equal(t, "a test error", signalOne.Errorf("%w", signalTwo.Errorf("%w", err)).Error(), "the string is untouched (Errorf)")
}
