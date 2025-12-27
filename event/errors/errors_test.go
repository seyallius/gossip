// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// -------------------------------------------- Error Type Tests --------------------------------------------

func TestRetryableError(t *testing.T) {
	err := NewRetryableError(errors.New("network error"), "temporary network issue")

	assert.Contains(t, err.Error(), "retryable error")
	assert.Contains(t, err.Error(), "network error")
	assert.Contains(t, err.Error(), "temporary network issue")

	// Test unwrapping
	unwrapped := err.Unwrap()
	assert.Equal(t, "network error", unwrapped.Error())
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("email", errors.New("invalid format"), "email must be valid")

	assert.Contains(t, err.Error(), "validation error on field 'email'")
	assert.Contains(t, err.Error(), "invalid format")
	assert.Contains(t, err.Error(), "email must be valid")

	// Test unwrapping
	unwrapped := err.Unwrap()
	assert.Equal(t, "invalid format", unwrapped.Error())
}

func TestProcessingError(t *testing.T) {
	err := NewProcessingError(errors.New("db error"), "database operation failed")

	assert.Contains(t, err.Error(), "processing error")
	assert.Contains(t, err.Error(), "db error")
	assert.Contains(t, err.Error(), "database operation failed")

	// Test unwrapping
	unwrapped := err.Unwrap()
	assert.Equal(t, "db error", unwrapped.Error())
}

func TestTimeoutError(t *testing.T) {
	err := NewTimeoutError("5s", errors.New("operation timeout"), "operation exceeded time limit")

	assert.Contains(t, err.Error(), "timeout error after 5s")
	assert.Contains(t, err.Error(), "operation timeout")
	assert.Contains(t, err.Error(), "operation exceeded time limit")

	// Test unwrapping
	unwrapped := err.Unwrap()
	assert.Equal(t, "operation timeout", unwrapped.Error())
}

func TestFatalError(t *testing.T) {
	err := NewFatalError(errors.New("critical error"), "unrecoverable system failure")

	assert.Contains(t, err.Error(), "fatal error")
	assert.Contains(t, err.Error(), "critical error")
	assert.Contains(t, err.Error(), "unrecoverable system failure")

	// Test unwrapping
	unwrapped := err.Unwrap()
	assert.Equal(t, "critical error", unwrapped.Error())
}

// -------------------------------------------- Error Checking Function Tests --------------------------------------------

func TestIsRetryable(t *testing.T) {
	retryableErr := NewRetryableError(errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	assert.True(t, IsRetryable(retryableErr))
	assert.False(t, IsRetryable(otherErr))
}

func TestIsValidation(t *testing.T) {
	validationErr := NewValidationError("field", errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	assert.True(t, IsValidation(validationErr))
	assert.False(t, IsValidation(otherErr))
}

func TestIsProcessing(t *testing.T) {
	processingErr := NewProcessingError(errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	assert.True(t, IsProcessing(processingErr))
	assert.False(t, IsProcessing(otherErr))
}

func TestIsTimeout(t *testing.T) {
	timeoutErr := NewTimeoutError("5s", errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	assert.True(t, IsTimeout(timeoutErr))
	assert.False(t, IsTimeout(otherErr))
}

func TestIsFatal(t *testing.T) {
	fatalErr := NewFatalError(errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	assert.True(t, IsFatal(fatalErr))
	assert.False(t, IsFatal(otherErr))
}

// -------------------------------------------- As Functions Tests --------------------------------------------

func TestAsRetryable(t *testing.T) {
	retryableErr := NewRetryableError(errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsRetryable(retryableErr)
	assert.True(t, ok)
	assert.Equal(t, "test", extracted.Err.Error())
	assert.Equal(t, "test message", extracted.Message)

	// Test failed extraction
	_, ok = AsRetryable(otherErr)
	assert.False(t, ok)
}

func TestAsValidationError(t *testing.T) {
	validationErr := NewValidationError("field", errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsValidationError(validationErr)
	assert.True(t, ok)
	assert.Equal(t, "test", extracted.Err.Error())
	assert.Equal(t, "field", extracted.Field)
	assert.Equal(t, "test message", extracted.Message)

	// Test failed extraction
	_, ok = AsValidationError(otherErr)
	assert.False(t, ok)
}

func TestAsProcessingError(t *testing.T) {
	processingErr := NewProcessingError(errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsProcessingError(processingErr)
	assert.True(t, ok)
	assert.Equal(t, "test", extracted.Err.Error())
	assert.Equal(t, "test message", extracted.Message)

	// Test failed extraction
	_, ok = AsProcessingError(otherErr)
	assert.False(t, ok)
}

func TestAsTimeoutError(t *testing.T) {
	timeoutErr := NewTimeoutError("5s", errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsTimeoutError(timeoutErr)
	assert.True(t, ok)
	assert.Equal(t, "test", extracted.Err.Error())
	assert.Equal(t, "5s", extracted.Timeout)
	assert.Equal(t, "test message", extracted.Message)

	// Test failed extraction
	_, ok = AsTimeoutError(otherErr)
	assert.False(t, ok)
}

func TestAsFatalError(t *testing.T) {
	fatalErr := NewFatalError(errors.New("test"), "test message")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsFatalError(fatalErr)
	assert.True(t, ok)
	assert.Equal(t, "test", extracted.Err.Error())
	assert.Equal(t, "test message", extracted.Message)

	// Test failed extraction
	_, ok = AsFatalError(otherErr)
	assert.False(t, ok)
}

// -------------------------------------------- Error Formatting Tests --------------------------------------------

func TestRetryableErrorFormatting(t *testing.T) {
	// Test with message
	err := NewRetryableError(errors.New("inner"), "outer message")
	assert.Contains(t, err.Error(), "retryable error: outer message: inner")

	// Test without message
	err2 := NewRetryableError(errors.New("inner"), "")
	assert.Contains(t, err2.Error(), "retryable error: inner")
}

func TestValidationErrorFormatting(t *testing.T) {
	// Test with field and message
	err := NewValidationError("email", errors.New("invalid"), "must be valid")
	assert.Contains(t, err.Error(), "validation error on field 'email': must be valid: invalid")

	// Test with field only
	err2 := NewValidationError("email", errors.New("invalid"), "")
	assert.Contains(t, err2.Error(), "validation error on field 'email': invalid")

	// Test with message only
	err3 := NewValidationError("", errors.New("invalid"), "must be valid")
	assert.Contains(t, err3.Error(), "validation error: must be valid: invalid")

	// Test with error only
	err4 := NewValidationError("", errors.New("invalid"), "")
	assert.Contains(t, err4.Error(), "validation error: invalid")
}

func TestProcessingErrorFormatting(t *testing.T) {
	// Test with message
	err := NewProcessingError(errors.New("inner"), "outer message")
	assert.Contains(t, err.Error(), "processing error: outer message: inner")

	// Test without message
	err2 := NewProcessingError(errors.New("inner"), "")
	assert.Contains(t, err2.Error(), "processing error: inner")
}

func TestTimeoutErrorFormatting(t *testing.T) {
	// Test with timeout and message
	err := NewTimeoutError("5s", errors.New("inner"), "outer message")
	assert.Contains(t, err.Error(), "timeout error after 5s: outer message: inner")

	// Test with timeout only
	err2 := NewTimeoutError("5s", errors.New("inner"), "")
	assert.Contains(t, err2.Error(), "timeout error after 5s: inner")

	// Test with message only
	err3 := NewTimeoutError("", errors.New("inner"), "outer message")
	assert.Contains(t, err3.Error(), "timeout error: outer message: inner")

	// Test with error only
	err4 := NewTimeoutError("", errors.New("inner"), "")
	assert.Contains(t, err4.Error(), "timeout error: inner")
}

func TestFatalErrorFormatting(t *testing.T) {
	// Test with message
	err := NewFatalError(errors.New("inner"), "outer message")
	assert.Contains(t, err.Error(), "fatal error: outer message: inner")

	// Test without message
	err2 := NewFatalError(errors.New("inner"), "")
	assert.Contains(t, err2.Error(), "fatal error: inner")
}

// -------------------------------------------- Formatted Error Creation Tests --------------------------------------------

func TestFormattedErrorCreation(t *testing.T) {
	// Test formatted retryable error
	retryable := NewRetryableErrorf("network error: %s", "timeout")
	assert.Contains(t, retryable.Error(), "network error: timeout")

	// Test formatted validation error
	validation := NewValidationErrorf("email", "invalid email: %s", "not a valid address")
	assert.Contains(t, validation.Error(), "validation error on field 'email'")
	assert.Contains(t, validation.Error(), "invalid email: not a valid address")

	// Test formatted processing error
	processing := NewProcessingErrorf("db operation failed: %s", "connection lost")
	assert.Contains(t, processing.Error(), "processing error: db operation failed: connection lost")

	// Test formatted timeout error
	timeout := NewTimeoutErrorf("10s", "operation timed out: %s", "slow response")
	assert.Contains(t, timeout.Error(), "timeout error after 10s: operation timed out: slow response")

	// Test formatted fatal error
	fatal := NewFatalErrorf("system failure: %s", "disk full")
	assert.Contains(t, fatal.Error(), "system failure: disk full")
}

// -------------------------------------------- Common Error Type Tests --------------------------------------------

func TestTypeAssertionError(t *testing.T) {
	err := NewTypeAssertionError("*UserData", "string", "invalid type assertion")

	assert.Contains(t, err.Error(), "type assertion error")
	assert.Contains(t, err.Error(), "*UserData")
	assert.Contains(t, err.Error(), "string")

	// Test with custom message
	err2 := NewTypeAssertionError("*UserData", "string", "custom message")
	assert.Contains(t, err2.Error(), "custom message")
}

func TestNoDataError(t *testing.T) {
	// Test with message
	err := NewNoDataError("data is missing")
	assert.Contains(t, err.Error(), "no data error: data is missing")

	// Test with field
	err2 := NewNoDataErrorForField("user_id")
	assert.Contains(t, err2.Error(), "no data error: field 'user_id' is empty")

	// Test default message
	err3 := NewNoDataError("")
	assert.Contains(t, err3.Error(), "no data error: event data is nil")
}

func TestInvalidEventError(t *testing.T) {
	// Test with event ID and reason
	err := NewInvalidEventError("event-123", "", "malformed payload")
	assert.Contains(t, err.Error(), "invalid event error for event event-123: malformed payload")

	// Test with field and reason
	err2 := NewInvalidEventError("", "timestamp", "invalid format")
	assert.Contains(t, err2.Error(), "invalid event error: field 'timestamp' - invalid format")

	// Test with formatted message
	err3 := NewInvalidEventErrorf("event %s has invalid data", "event-456")
	assert.Contains(t, err3.Error(), "event event-456 has invalid data")

	// Test default message
	err4 := &InvalidEventError{}
	assert.Contains(t, err4.Error(), "invalid event error: event is malformed")
}

func TestUnsupportedEventTypeError(t *testing.T) {
	// Test with event type and supported types
	err := NewUnsupportedEventTypeError("user.deleted", []string{"user.created", "user.updated"})
	assert.Contains(t, err.Error(), "unsupported event type error: event type 'user.deleted' not supported, supported types: [user.created user.updated]")

	// Test with formatted message
	err2 := NewUnsupportedEventTypeErrorf("event type %s is not supported", "user.deleted")
	assert.Contains(t, err2.Error(), "event type user.deleted is not supported")

	// Test with just event type
	err3 := &UnsupportedEventTypeError{EventType: "user.deleted"}
	assert.Contains(t, err3.Error(), "unsupported event type error: event type 'user.deleted' not supported")
}

// -------------------------------------------- Common Error Checking Function Tests --------------------------------------------

func TestIsTypeAssertion(t *testing.T) {
	typeAssertionErr := NewTypeAssertionError("*UserData", "string", "test")
	otherErr := errors.New("regular error")

	assert.True(t, IsTypeAssertion(typeAssertionErr))
	assert.False(t, IsTypeAssertion(otherErr))
}

func TestIsNoData(t *testing.T) {
	noDataErr := NewNoDataError("missing data")
	otherErr := errors.New("regular error")

	assert.True(t, IsNoData(noDataErr))
	assert.False(t, IsNoData(otherErr))
}

func TestIsInvalidEvent(t *testing.T) {
	invalidEventErr := NewInvalidEventError("event-123", "", "malformed")
	otherErr := errors.New("regular error")

	assert.True(t, IsInvalidEvent(invalidEventErr))
	assert.False(t, IsInvalidEvent(otherErr))
}

func TestIsUnsupportedEventType(t *testing.T) {
	unsupportedErr := NewUnsupportedEventTypeError("user.deleted", []string{"user.created"})
	otherErr := errors.New("regular error")

	assert.True(t, IsUnsupportedEventType(unsupportedErr))
	assert.False(t, IsUnsupportedEventType(otherErr))
}

// -------------------------------------------- Common As Functions Tests --------------------------------------------

func TestAsTypeAssertionError(t *testing.T) {
	typeAssertionErr := NewTypeAssertionError("*UserData", "string", "test message")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsTypeAssertionError(typeAssertionErr)
	assert.True(t, ok)
	assert.Equal(t, "*UserData", extracted.ExpectedType)
	assert.Equal(t, "string", extracted.ActualType)
	assert.Equal(t, "test message", extracted.Message)

	// Test failed extraction
	_, ok = AsTypeAssertionError(otherErr)
	assert.False(t, ok)
}

func TestAsNoDataError(t *testing.T) {
	noDataErr := NewNoDataError("missing data")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsNoDataError(noDataErr)
	assert.True(t, ok)
	assert.Equal(t, "missing data", extracted.Message)

	// Test failed extraction
	_, ok = AsNoDataError(otherErr)
	assert.False(t, ok)
}

func TestAsInvalidEventError(t *testing.T) {
	invalidEventErr := NewInvalidEventError("event-123", "field", "reason")
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsInvalidEventError(invalidEventErr)
	assert.True(t, ok)
	assert.Equal(t, "event-123", extracted.EventID)
	assert.Equal(t, "field", extracted.Field)
	assert.Equal(t, "reason", extracted.Reason)

	// Test failed extraction
	_, ok = AsInvalidEventError(otherErr)
	assert.False(t, ok)
}

func TestAsUnsupportedEventTypeError(t *testing.T) {
	unsupportedErr := NewUnsupportedEventTypeError("user.deleted", []string{"user.created"})
	otherErr := errors.New("regular error")

	// Test successful extraction
	extracted, ok := AsUnsupportedEventTypeError(unsupportedErr)
	assert.True(t, ok)
	assert.Equal(t, "user.deleted", extracted.EventType)
	assert.Equal(t, []string{"user.created"}, extracted.Supported)

	// Test failed extraction
	_, ok = AsUnsupportedEventTypeError(otherErr)
	assert.False(t, ok)
}

// -------------------------------------------- Error Constants Tests --------------------------------------------

func TestErrorConstants(t *testing.T) {
	// Test ErrNilData
	assert.True(t, IsNoData(ErrNilData))
	assert.Contains(t, ErrNilData.Error(), "event data is nil")

	// Test ErrTypeAssertionFailed
	assert.True(t, IsTypeAssertion(ErrTypeAssertionFailed))
	assert.Contains(t, ErrTypeAssertionFailed.Error(), "type assertion failed")

	// Test ErrInvalidEvent
	assert.True(t, IsInvalidEvent(ErrInvalidEvent))
	assert.Contains(t, ErrInvalidEvent.Error(), "event is invalid")

	// Test ErrUnsupportedEventType
	assert.True(t, IsUnsupportedEventType(ErrUnsupportedEventType))
	assert.Contains(t, ErrUnsupportedEventType.Error(), "event type is not supported")

	// Test ErrRetryable
	assert.True(t, IsRetryable(ErrRetryable))
	assert.Contains(t, ErrRetryable.Error(), "operation is retryable")

	// Test ErrValidationFailed
	assert.True(t, IsValidation(ErrValidationFailed))
	assert.Contains(t, ErrValidationFailed.Error(), "validation failed")

	// Test ErrProcessingFailed
	assert.True(t, IsProcessing(ErrProcessingFailed))
	assert.Contains(t, ErrProcessingFailed.Error(), "processing failed")

	// Test ErrTimeout
	assert.True(t, IsTimeout(ErrTimeout))
	assert.Contains(t, ErrTimeout.Error(), "operation timed out")

	// Test ErrFatal
	assert.True(t, IsFatal(ErrFatal))
	assert.Contains(t, ErrFatal.Error(), "fatal error occurred")
}

func TestErrorConstantsUsage(t *testing.T) {
	// Test that constants can be used directly in conditional logic
	assert.True(t, IsNoData(ErrNilData))
	assert.True(t, IsTypeAssertion(ErrTypeAssertionFailed))
	assert.True(t, IsInvalidEvent(ErrInvalidEvent))
	assert.True(t, IsUnsupportedEventType(ErrUnsupportedEventType))
	assert.True(t, IsRetryable(ErrRetryable))
	assert.True(t, IsValidation(ErrValidationFailed))
	assert.True(t, IsProcessing(ErrProcessingFailed))
	assert.True(t, IsTimeout(ErrTimeout))
	assert.True(t, IsFatal(ErrFatal))
}
