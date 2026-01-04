# Error Handling Example

This example demonstrates how to use Gossip's standard error types for conditional error handling in event-driven applications.

## Overview

The error handling example shows:
- How to use different error types (`RetryableError`, `ValidationError`, `FatalError`, etc.)
- How to implement conditional error handling based on error types
- How to create custom middleware that handles errors differently based on their type
- How to combine error handling with other middleware like retry and timeout

## Key Concepts Demonstrated

### 1. Standard Error Types
Using `gossip/event/errors` package to create typed errors with specific meanings.

### 2. Conditional Error Handling
Using `IsRetryable`, `IsValidation`, etc. to handle different error types differently.

### 3. Integration with Middleware
How error types work with existing middleware like retry and timeout.

### 4. Custom Error Handling Middleware
Creating middleware that responds to specific error types.

## Code Walkthrough

### Event Types and Data

```go
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
```

### Email Notification Processor with Error Handling

```go
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
```

### Audit Log Processor

```go
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
```

### Error Handling with Conditional Logic

```go
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
```

### Custom Middleware with Error Handling

```go
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
```

### Main Function Setup

```go
func main() {
	// Initialize event bus with default in-memory provider
	config := &bus.Config{
		Driver:     "memory", // Use "redis" for distributed systems
		Workers:    10,
		BufferSize: 1000,
		// For Redis provider, configure:
		// RedisAddr:  "localhost:6379",
		// RedisPwd:   "",
		// RedisDB:    0,
	}
	eventBus := bus.NewEventBus(config)
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

	// Wait for events to be processed
	time.Sleep(2 * time.Second)
}
```

## Error Types Used

### RetryableError
For transient failures that can be retried. Used with retry middleware to automatically retry failed operations.

### ValidationError
For input validation failures. Indicates that the event data doesn't meet validation requirements.

### FatalError
For unrecoverable errors that should not be retried. These errors indicate serious problems that require manual intervention.

### TimeoutError
For operations that exceed time limits. Used with timeout middleware to prevent long-running operations from blocking.

### ProcessingError
For general processing failures. Generic error type for when other more specific types don't apply.

### TypeAssertionError
For type assertion failures. Similar to `redis.Nil` pattern for type mismatches.

### NoDataError
For missing event data when data was expected. Used when event.Data is nil but the processor requires data.

## Running the Example

To run this example:

```bash
cd examples/error_handling
go run main.go
```

## Key Takeaways

1. **Typed Errors**: Using specific error types allows for more sophisticated error handling logic.

2. **Conditional Handling**: Different error types can be handled differently based on their nature and severity.

3. **Middleware Integration**: Error types work seamlessly with existing middleware like retry and timeout.

4. **Monitoring**: Typed errors make it easier to monitor and alert on specific error conditions.

## Best Practices Demonstrated

- Use appropriate error types for different failure scenarios
- Implement conditional error handling based on error types
- Combine error handling with other middleware for robust processing
- Log different error types with appropriate severity levels
- Use error constants for common error conditions