// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package errors provides standard error types for the Gossip event system.
// These error types allow users to implement conditional logic based on specific error conditions.
package errors

import (
	"errors"
	"fmt"
)

// -------------------------------------------- Error Types --------------------------------------------

// RetryableError indicates that an operation can be retried.
// This error type is useful for transient failures like network issues or rate limits.
type RetryableError struct {
	Err     error
	Message string
}

func (e *RetryableError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("retryable error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("retryable error: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// ValidationError indicates that input data failed validation.
// This error type is useful for rejecting events with invalid data.
type ValidationError struct {
	Err     error
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" && e.Message != "" {
		return fmt.Sprintf("validation error on field '%s': %s: %v", e.Field, e.Message, e.Err)
	} else if e.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %v", e.Field, e.Err)
	} else if e.Message != "" {
		return fmt.Sprintf("validation error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("validation error: %v", e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ProcessingError indicates that an event processor encountered an error during processing.
// This error type is useful for general processing failures that are not retryable.
type ProcessingError struct {
	Err     error
	Message string
}

func (e *ProcessingError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("processing error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("processing error: %v", e.Err)
}

func (e *ProcessingError) Unwrap() error {
	return e.Err
}

// TimeoutError indicates that an operation exceeded its time limit.
// This error type is useful for operations that should not take too long.
type TimeoutError struct {
	Err     error
	Message string
	Timeout string
}

func (e *TimeoutError) Error() string {
	if e.Timeout != "" && e.Message != "" {
		return fmt.Sprintf("timeout error after %s: %s: %v", e.Timeout, e.Message, e.Err)
	} else if e.Timeout != "" {
		return fmt.Sprintf("timeout error after %s: %v", e.Timeout, e.Err)
	} else if e.Message != "" {
		return fmt.Sprintf("timeout error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("timeout error: %v", e.Err)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

// FatalError indicates that an operation encountered an unrecoverable error.
// This error type signals that the operation should not be retried.
type FatalError struct {
	Err     error
	Message string
}

func (e *FatalError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("fatal error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("fatal error: %v", e.Err)
}

func (e *FatalError) Unwrap() error {
	return e.Err
}

// -------------------------------------------- Error Creation Functions --------------------------------------------

// NewRetryableError creates a new RetryableError.
func NewRetryableError(err error, message string) *RetryableError {
	return &RetryableError{
		Err:     err,
		Message: message,
	}
}

// NewRetryableErrorf creates a new RetryableError with formatted message.
func NewRetryableErrorf(format string, args ...interface{}) *RetryableError {
	return &RetryableError{
		Err:     fmt.Errorf(format, args...),
		Message: fmt.Sprintf(format, args...),
	}
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field string, err error, message string) *ValidationError {
	return &ValidationError{
		Err:     err,
		Field:   field,
		Message: message,
	}
}

// NewValidationErrorf creates a new ValidationError with formatted message.
func NewValidationErrorf(field, format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		Err:     fmt.Errorf(format, args...),
		Field:   field,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewProcessingError creates a new ProcessingError.
func NewProcessingError(err error, message string) *ProcessingError {
	return &ProcessingError{
		Err:     err,
		Message: message,
	}
}

// NewProcessingErrorf creates a new ProcessingError with formatted message.
func NewProcessingErrorf(format string, args ...interface{}) *ProcessingError {
	return &ProcessingError{
		Err:     fmt.Errorf(format, args...),
		Message: fmt.Sprintf(format, args...),
	}
}

// NewTimeoutError creates a new TimeoutError.
func NewTimeoutError(timeout string, err error, message string) *TimeoutError {
	return &TimeoutError{
		Err:     err,
		Message: message,
		Timeout: timeout,
	}
}

// NewTimeoutErrorf creates a new TimeoutError with formatted message.
func NewTimeoutErrorf(timeout, format string, args ...interface{}) *TimeoutError {
	return &TimeoutError{
		Err:     fmt.Errorf(format, args...),
		Message: fmt.Sprintf(format, args...),
		Timeout: timeout,
	}
}

// NewFatalError creates a new FatalError.
func NewFatalError(err error, message string) *FatalError {
	return &FatalError{
		Err:     err,
		Message: message,
	}
}

// NewFatalErrorf creates a new FatalError with formatted message.
func NewFatalErrorf(format string, args ...interface{}) *FatalError {
	return &FatalError{
		Err:     fmt.Errorf(format, args...),
		Message: fmt.Sprintf(format, args...),
	}
}

// -------------------------------------------- Error Checking Functions --------------------------------------------

// IsRetryable checks if an error is a RetryableError.
func IsRetryable(err error) bool {
	var retryableErr *RetryableError
	return errors.As(err, &retryableErr)
}

// IsValidation checks if an error is a ValidationError.
func IsValidation(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsProcessing checks if an error is a ProcessingError.
func IsProcessing(err error) bool {
	var processingErr *ProcessingError
	return errors.As(err, &processingErr)
}

// IsTimeout checks if an error is a TimeoutError.
func IsTimeout(err error) bool {
	var timeoutErr *TimeoutError
	return errors.As(err, &timeoutErr)
}

// IsFatal checks if an error is a FatalError.
func IsFatal(err error) bool {
	var fatalErr *FatalError
	return errors.As(err, &fatalErr)
}

// AsRetryable extracts a RetryableError from an error chain.
func AsRetryable(err error) (*RetryableError, bool) {
	var retryableErr *RetryableError
	ok := errors.As(err, &retryableErr)
	return retryableErr, ok
}

// AsValidationError extracts a ValidationError from an error chain.
func AsValidationError(err error) (*ValidationError, bool) {
	var validationErr *ValidationError
	ok := errors.As(err, &validationErr)
	return validationErr, ok
}

// AsProcessingError extracts a ProcessingError from an error chain.
func AsProcessingError(err error) (*ProcessingError, bool) {
	var processingErr *ProcessingError
	ok := errors.As(err, &processingErr)
	return processingErr, ok
}

// AsTimeoutError extracts a TimeoutError from an error chain.
func AsTimeoutError(err error) (*TimeoutError, bool) {
	var timeoutErr *TimeoutError
	ok := errors.As(err, &timeoutErr)
	return timeoutErr, ok
}

// AsFatalError extracts a FatalError from an error chain.
func AsFatalError(err error) (*FatalError, bool) {
	var fatalErr *FatalError
	ok := errors.As(err, &fatalErr)
	return fatalErr, ok
}

// -------------------------------------------- Common Event Processing Error Types --------------------------------------------

// TypeAssertionError indicates that a type assertion failed.
// This error is common when processing events where the expected type doesn't match the actual data.
type TypeAssertionError struct {
	ExpectedType string
	ActualType   string
	Value        interface{}
	Message      string
}

func (e *TypeAssertionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("type assertion error: %s (expected %s, got %T)", e.Message, e.ExpectedType, e.Value)
	}
	return fmt.Sprintf("type assertion error: expected %s, got %T", e.ExpectedType, e.Value)
}

func (e *TypeAssertionError) Unwrap() error {
	return nil
}

// NoDataError indicates that an event has no data when data was expected.
// This is similar to redis.Nil - it represents a missing value scenario.
type NoDataError struct {
	Message string
	Field   string
}

func (e *NoDataError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("no data error: %s", e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("no data error: field '%s' is empty", e.Field)
	}
	return "no data error: event data is nil"
}

func (e *NoDataError) Unwrap() error {
	return nil
}

// InvalidEventError indicates that an event is malformed or invalid.
type InvalidEventError struct {
	EventID string
	Field   string
	Reason  string
	Message string
}

func (e *InvalidEventError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("invalid event error: %s", e.Message)
	}
	if e.EventID != "" && e.Reason != "" {
		return fmt.Sprintf("invalid event error for event %s: %s", e.EventID, e.Reason)
	} else if e.Field != "" && e.Reason != "" {
		return fmt.Sprintf("invalid event error: field '%s' - %s", e.Field, e.Reason)
	}
	return "invalid event error: event is malformed"
}

func (e *InvalidEventError) Unwrap() error {
	return nil
}

// UnsupportedEventTypeError indicates that an event type is not supported by a processor.
type UnsupportedEventTypeError struct {
	EventType string
	Supported []string
	Message   string
}

func (e *UnsupportedEventTypeError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("unsupported event type error: %s", e.Message)
	}
	if e.EventType != "" && len(e.Supported) > 0 {
		return fmt.Sprintf("unsupported event type error: event type '%s' not supported, supported types: %v", e.EventType, e.Supported)
	}
	return fmt.Sprintf("unsupported event type error: event type '%s' not supported", e.EventType)
}

func (e *UnsupportedEventTypeError) Unwrap() error {
	return nil
}

// -------------------------------------------- Error Creation Functions for Common Errors --------------------------------------------

// NewTypeAssertionError creates a new TypeAssertionError.
func NewTypeAssertionError(expectedType string, actualValue interface{}, message string) *TypeAssertionError {
	return &TypeAssertionError{
		ExpectedType: expectedType,
		ActualType:   fmt.Sprintf("%T", actualValue),
		Value:        actualValue,
		Message:      message,
	}
}

// NewNoDataError creates a new NoDataError.
func NewNoDataError(message string) *NoDataError {
	return &NoDataError{
		Message: message,
	}
}

// NewNoDataErrorForField creates a new NoDataError for a specific field.
func NewNoDataErrorForField(field string) *NoDataError {
	return &NoDataError{
		Field: field,
	}
}

// NewInvalidEventError creates a new InvalidEventError.
func NewInvalidEventError(eventID, field, reason string) *InvalidEventError {
	return &InvalidEventError{
		EventID: eventID,
		Field:   field,
		Reason:  reason,
	}
}

// NewInvalidEventErrorf creates a new InvalidEventError with formatted message.
func NewInvalidEventErrorf(format string, args ...interface{}) *InvalidEventError {
	return &InvalidEventError{
		Message: fmt.Sprintf(format, args...),
	}
}

// NewUnsupportedEventTypeError creates a new UnsupportedEventTypeError.
func NewUnsupportedEventTypeError(eventType string, supported []string) *UnsupportedEventTypeError {
	return &UnsupportedEventTypeError{
		EventType: eventType,
		Supported: supported,
	}
}

// NewUnsupportedEventTypeErrorf creates a new UnsupportedEventTypeError with formatted message.
func NewUnsupportedEventTypeErrorf(format string, args ...interface{}) *UnsupportedEventTypeError {
	return &UnsupportedEventTypeError{
		Message: fmt.Sprintf(format, args...),
	}
}

// -------------------------------------------- Error Checking Functions for Common Errors --------------------------------------------

// IsTypeAssertion checks if an error is a TypeAssertionError.
func IsTypeAssertion(err error) bool {
	var typeAssertionErr *TypeAssertionError
	return errors.As(err, &typeAssertionErr)
}

// IsNoData checks if an error is a NoDataError.
func IsNoData(err error) bool {
	var noDataErr *NoDataError
	return errors.As(err, &noDataErr)
}

// IsInvalidEvent checks if an error is an InvalidEventError.
func IsInvalidEvent(err error) bool {
	var invalidEventErr *InvalidEventError
	return errors.As(err, &invalidEventErr)
}

// IsUnsupportedEventType checks if an error is an UnsupportedEventTypeError.
func IsUnsupportedEventType(err error) bool {
	var unsupportedEventTypeErr *UnsupportedEventTypeError
	return errors.As(err, &unsupportedEventTypeErr)
}

// AsTypeAssertionError extracts a TypeAssertionError from an error chain.
func AsTypeAssertionError(err error) (*TypeAssertionError, bool) {
	var typeAssertionErr *TypeAssertionError
	ok := errors.As(err, &typeAssertionErr)
	return typeAssertionErr, ok
}

// AsNoDataError extracts a NoDataError from an error chain.
func AsNoDataError(err error) (*NoDataError, bool) {
	var noDataErr *NoDataError
	ok := errors.As(err, &noDataErr)
	return noDataErr, ok
}

// AsInvalidEventError extracts an InvalidEventError from an error chain.
func AsInvalidEventError(err error) (*InvalidEventError, bool) {
	var invalidEventErr *InvalidEventError
	ok := errors.As(err, &invalidEventErr)
	return invalidEventErr, ok
}

// AsUnsupportedEventTypeError extracts an UnsupportedEventTypeError from an error chain.
func AsUnsupportedEventTypeError(err error) (*UnsupportedEventTypeError, bool) {
	var unsupportedEventTypeErr *UnsupportedEventTypeError
	ok := errors.As(err, &unsupportedEventTypeErr)
	return unsupportedEventTypeErr, ok
}

// -------------------------------------------- Public Error Constants --------------------------------------------

// Common error instances for frequently used scenarios.
// These are similar to how io.EOF or redis.Nil work - pre-defined error instances
// that can be returned directly without creating new instances.

// ErrNilData represents a nil data error - similar to redis.Nil but for event data.
var ErrNilData = &NoDataError{
	Message: "event data is nil",
}

// ErrTypeAssertionFailed represents a generic type assertion failure.
var ErrTypeAssertionFailed = &TypeAssertionError{
	ExpectedType: "unknown",
	ActualType:   "unknown",
	Message:      "type assertion failed",
	Value:        nil,
}

// ErrInvalidEvent represents a generic invalid event error.
var ErrInvalidEvent = &InvalidEventError{
	Message: "event is invalid",
}

// ErrUnsupportedEventType represents a generic unsupported event type error.
var ErrUnsupportedEventType = &UnsupportedEventTypeError{
	Message: "event type is not supported",
}

// ErrRetryable represents a generic retryable error.
var ErrRetryable = &RetryableError{
	Message: "operation is retryable",
	Err:     errors.New("retryable operation"),
}

// ErrValidationFailed represents a generic validation error.
var ErrValidationFailed = &ValidationError{
	Message: "validation failed",
	Err:     errors.New("validation error"),
}

// ErrProcessingFailed represents a generic processing error.
var ErrProcessingFailed = &ProcessingError{
	Message: "processing failed",
	Err:     errors.New("processing error"),
}

// ErrTimeout represents a generic timeout error.
var ErrTimeout = &TimeoutError{
	Message: "operation timed out",
	Timeout: "unknown",
	Err:     errors.New("timeout error"),
}

// ErrFatal represents a generic fatal error.
var ErrFatal = &FatalError{
	Message: "fatal error occurred",
	Err:     errors.New("fatal error"),
}
