package errors_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/memsql/errors"
	"github.com/stretchr/testify/require"
)

func TestAnnotate(t *testing.T) {
	baseErr := fmt.Errorf("base error")

	t.Run("nil error returns nil", func(t *testing.T) {
		result := errors.Annotate(nil, "some", "values")
		require.Nil(t, result)
	})

	t.Run("no values returns original error", func(t *testing.T) {
		result := errors.Annotate(baseErr)
		require.Equal(t, baseErr, result)
	})

	t.Run("empty values returns original error", func(t *testing.T) {
		result := errors.Annotate(baseErr, []any{}...)
		require.Equal(t, baseErr, result)
	})

	t.Run("single value annotation", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "test-value")
		require.NotEqual(t, baseErr, annotated)
		require.Equal(t, "base error", annotated.Error())

		// Should be able to unwrap to get original error
		require.Equal(t, baseErr, errors.Unwrap(annotated))
	})

	t.Run("multiple values annotation", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "string", 42, true, struct{ Name string }{"test"})
		require.Equal(t, "base error", annotated.Error())
		require.Equal(t, baseErr, errors.Unwrap(annotated))
	})

	t.Run("annotation preserves error chain", func(t *testing.T) {
		wrapped := fmt.Errorf("wrapped: %w", baseErr)
		annotated := errors.Annotate(wrapped, "metadata")

		require.True(t, errors.Is(annotated, baseErr))
		require.True(t, errors.Is(annotated, wrapped))
	})
}

func TestAnnotation(t *testing.T) {
	baseErr := fmt.Errorf("base error")

	t.Run("no annotations returns zero value and false", func(t *testing.T) {
		value, found := errors.Annotation[string](baseErr)
		require.False(t, found)
		require.Equal(t, "", value)

		intValue, intFound := errors.Annotation[int](baseErr)
		require.False(t, intFound)
		require.Equal(t, 0, intValue)
	})

	t.Run("finds string annotation", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "test-string", 42)

		value, found := errors.Annotation[string](annotated)
		require.True(t, found)
		require.Equal(t, "test-string", value)
	})

	t.Run("finds int annotation", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "test-string", 42, true)

		value, found := errors.Annotation[int](annotated)
		require.True(t, found)
		require.Equal(t, 42, value)
	})

	t.Run("finds bool annotation", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "test-string", 42, true)

		value, found := errors.Annotation[bool](annotated)
		require.True(t, found)
		require.Equal(t, true, value)
	})

	t.Run("finds struct annotation", func(t *testing.T) {
		type TestStruct struct {
			Name string
			ID   int
		}
		testStruct := TestStruct{Name: "test", ID: 123}
		annotated := errors.Annotate(baseErr, "other", testStruct)

		value, found := errors.Annotation[TestStruct](annotated)
		require.True(t, found)
		require.Equal(t, testStruct, value)
	})

	t.Run("returns first matching annotation when multiple exist", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "first", "second", "third")

		value, found := errors.Annotation[string](annotated)
		require.True(t, found)
		require.Equal(t, "first", value)
	})

	t.Run("finds annotation in wrapped errors", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "inner-value")
		wrapped := fmt.Errorf("outer error: %w", annotated)

		value, found := errors.Annotation[string](wrapped)
		require.True(t, found)
		require.Equal(t, "inner-value", value)
	})

	t.Run("finds annotation in deeply nested errors", func(t *testing.T) {
		level1 := errors.Annotate(baseErr, "level1")
		level2 := errors.Annotate(level1, 42)
		level3 := fmt.Errorf("level3: %w", level2)
		level4 := errors.Annotate(level3, true)

		// Should find annotations from any level
		strValue, strFound := errors.Annotation[string](level4)
		require.True(t, strFound)
		require.Equal(t, "level1", strValue)

		intValue, intFound := errors.Annotation[int](level4)
		require.True(t, intFound)
		require.Equal(t, 42, intValue)

		boolValue, boolFound := errors.Annotation[bool](level4)
		require.True(t, boolFound)
		require.Equal(t, true, boolValue)
	})

	t.Run("works with Errorf annotations", func(t *testing.T) {
		type UserID int
		userID := UserID(12345)

		// Test that Annotation works with errors created by Errorf
		errWithArg := errors.Errorf("user operation failed for user %v", userID)

		value, found := errors.Annotation[UserID](errWithArg)
		require.True(t, found)
		require.Equal(t, userID, value)
	})

	t.Run("annotation survives error wrapping", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, "preserved-value")
		wrapped1 := fmt.Errorf("wrap1: %w", annotated)
		wrapped2 := fmt.Errorf("wrap2: %w", wrapped1)

		value, found := errors.Annotation[string](wrapped2)
		require.True(t, found)
		require.Equal(t, "preserved-value", value)
	})

	t.Run("handles pointer types", func(t *testing.T) {
		testStr := "pointer-test"
		annotated := errors.Annotate(baseErr, &testStr)

		value, found := errors.Annotation[*string](annotated)
		require.True(t, found)
		require.Equal(t, &testStr, value)
		require.Equal(t, "pointer-test", *value)
	})

	t.Run("handles interface types", func(t *testing.T) {
		annotated := errors.Annotate(baseErr, time.Second)

		value, found := errors.Annotation[fmt.Stringer](annotated)
		require.True(t, found)
		require.Equal(t, "1s", value.String())
	})
}

func TestAnnotateIntegration(t *testing.T) {
	t.Run("format preserves annotations", func(t *testing.T) {
		baseErr := fmt.Errorf("base error")
		annotated := errors.Annotate(baseErr, "metadata", 42)

		// The error message should be preserved
		require.Equal(t, "base error", annotated.Error())

		// And we should still be able to get annotations
		strValue, strFound := errors.Annotation[string](annotated)
		require.True(t, strFound)
		require.Equal(t, "metadata", strValue)

		intValue, intFound := errors.Annotation[int](annotated)
		require.True(t, intFound)
		require.Equal(t, 42, intValue)
	})
}
