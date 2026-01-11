// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package main. main demonstrates error handling with conditional logic in Gossip event system.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	gossip "github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
	goserr "github.com/seyallius/gossip/event/errors"
	"github.com/seyallius/gossip/event/middleware"
	"github.com/seyallius/gossip/event/sub"
)

// Event types for our example
const (
	UserCreated gossip.EventType = "user.created"
	UserUpdated gossip.EventType = "user.updated"
)

// User data structure
type UserData struct {
	UserID   string
	Username string
	Email    string
}

// -------------------------------------------- Example Processors with Error Handling --------------------------------------------

// emailNotificationProcessor sends email notifications when users are created
func emailNotificationProcessor(ctx context.Context, event *gossip.Event) error {
	// Type assertion with error handling - this is the common scenario you mentioned
	data, ok := event.Data.(*UserData)
	if !ok {
		// This is the common scenario you mentioned - when type assertion fails
		return goserr.NewTypeAssertionError("*UserData", event.Data, "expected *UserData for email notification")
	}

	// Check if event data is nil - another common scenario
	if data == nil {
		return goserr.NewNoDataError("event data is nil in email notification processor")
	}

	// Validate the email address
	if data.Email == "" {
		return goserr.NewValidationError("email", nil, "email address is required")
	}

	// Simulate email sending - sometimes this might fail temporarily
	if time.Now().Unix()%5 == 0 { // Simulate occasional transient failure
		return goserr.NewRetryableError(
			fmt.Errorf("email service temporarily unavailable"),
			"temporary email service failure",
		)
	}

	// Simulate successful email sending
	log.Printf("[Email Service] Sent welcome email to %s (%s)", data.Username, data.Email)
	return nil
}

// auditLogProcessor logs all user events for compliance
func auditLogProcessor(ctx context.Context, event *gossip.Event) error {
	data := event.Data.(*UserData)

	// Simulate audit logging
	log.Printf("[Audit] Event: %s for user %s (%s)", event.Type, data.Username, data.UserID)

	// Simulate a fatal error for invalid user IDs
	if data.UserID == "invalid" {
		return goserr.NewFatalError(
			fmt.Errorf("invalid user ID detected"),
			"security violation - invalid user ID",
		)
	}

	return nil
}

// metricsProcessor tracks user creation metrics
func metricsProcessor(ctx context.Context, event *gossip.Event) error {
	// Type assertion with error handling
	data, ok := event.Data.(*UserData)
	if !ok {
		return goserr.NewTypeAssertionError("*UserData", event.Data, "expected *UserData for metrics collection")
	}

	// Simulate metrics collection
	log.Printf("[Metrics] User %s created", data.Username)

	// Simulate timeout for slow metrics service
	if time.Now().Unix()%7 == 0 {
		return goserr.NewTimeoutError(
			"5s",
			fmt.Errorf("metrics service timeout"),
			"metrics collection timeout",
		)
	}

	return nil
}

// validationProcessor demonstrates validation and other error types
func validationProcessor(ctx context.Context, event *gossip.Event) error {
	// Type assertion with error handling
	data, ok := event.Data.(*UserData)
	if !ok {
		return goserr.NewTypeAssertionError("*UserData", event.Data, "expected *UserData for validation")
	}

	// Check for nil data
	if data == nil {
		return goserr.NewNoDataError("event data is nil in validation processor")
	}

	// Validate the event itself
	if data.UserID == "" {
		return goserr.NewInvalidEventError("", "user_id", "user ID cannot be empty")
	}

	// Check event type compatibility
	if event.Type != UserCreated && event.Type != UserUpdated {
		return goserr.NewUnsupportedEventTypeError(string(event.Type), []string{string(UserCreated), string(UserUpdated)})
	}

	// Simulate successful validation
	log.Printf("[Validation] Event %s passed validation for user %s", event.Type, data.Username)
	return nil
}

// -------------------------------------------- Error Handling with Conditional Logic --------------------------------------------

// handleProcessorError demonstrates how to handle different error types conditionally
func handleProcessorError(eventType gossip.EventType, err error) {
	switch {
	case goserr.IsRetryable(err):
		// Handle retryable errors - these can be retried by middleware
		log.Printf("[Error Handler] Retryable error for event %s: %v", eventType, err)
		// In a real application, you might want to log this for monitoring
		// or implement custom retry logic

	case goserr.IsValidation(err):
		// Handle validation errors - these indicate bad input data
		log.Printf("[Error Handler] Validation error for event %s: %v", eventType, err)
		// In a real application, you might want to alert developers
		// or log this for data quality monitoring

	case goserr.IsTimeout(err):
		// Handle timeout errors - these indicate operations taking too long
		log.Printf("[Error Handler] Timeout error for event %s: %v", eventType, err)
		// In a real application, you might want to adjust timeouts
		// or investigate performance issues

	case goserr.IsFatal(err):
		// Handle fatal errors - these should not be retried
		log.Printf("[Error Handler] Fatal error for event %s: %v", eventType, err)
		// In a real application, you might want to alert immediately
		// or implement circuit breaker logic

	case goserr.IsProcessing(err):
		// Handle general processing errors
		log.Printf("[Error Handler] Processing error for event %s: %v", eventType, err)
		// In a real application, you might want to log for debugging

	// You can also check for specific constants directly
	case errors.Is(err, goserr.ErrNilData):
		log.Printf("[Error Handler] No data found for event %s (using constant check)", eventType)

	case errors.Is(err, goserr.ErrTypeAssertionFailed):
		log.Printf("[Error Handler] Type assertion failed for event %s (using constant check)", eventType)

	default:
		// Handle other errors
		log.Printf("[Error Handler] Other error for event %s: %v", eventType, err)
	}
}

// -------------------------------------------- Custom Middleware with Error Handling --------------------------------------------

// WithErrorHandling is a custom middleware that handles errors conditionally
func WithErrorHandling() middleware.Middleware {
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, event *gossip.Event) error {
			err := next(ctx, event)
			if err != nil {
				// Handle the error based on its type
				handleProcessorError(event.Type, err)
			}
			return err
		}
	}
}

// -------------------------------------------- Main Function --------------------------------------------

func main() {
	// Initialize event bus
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	// Subscribe processors with error handling middleware
	eventBus.Subscribe(
		UserCreated,
		middleware.Chain(
			middleware.WithRetry(3, 100*time.Millisecond), // Retry retryable errors
			WithErrorHandling(),                           // Handle errors conditionally
		)(emailNotificationProcessor),
	)

	eventBus.Subscribe(
		UserCreated,
		middleware.Chain(
			WithErrorHandling(),
		)(auditLogProcessor),
	)

	eventBus.Subscribe(
		UserCreated,
		middleware.Chain(
			middleware.WithTimeout(3*time.Second), // Timeout for slow operations
			WithErrorHandling(),
		)(metricsProcessor),
	)

	eventBus.Subscribe(
		UserCreated,
		middleware.Chain(
			WithErrorHandling(),
		)(validationProcessor),
	)

	// Publish events with different scenarios
	fmt.Println("Publishing events with various error conditions...")

	// Scenario 1: Valid event
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "user-123",
		Username: "alice",
		Email:    "alice@example.com",
	}))

	// Scenario 2: Validation error (missing email)
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "user-456",
		Username: "bob",
		Email:    "", // Missing email will cause validation error
	}))

	// Scenario 3: Fatal error (invalid user ID)
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "invalid", // This will cause a fatal error
		Username: "charlie",
		Email:    "charlie@example.com",
	}))

	// Scenario 4: Retryable error (simulated by time condition)
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "user-789",
		Username: "diana",
		Email:    "diana@example.com",
	}))

	// Wait for events to be processed
	time.Sleep(2 * time.Second)

	fmt.Println("\nDemonstration complete! Check the logs to see how different error types were handled.")
}
