# Type Assertion Error Handling Example

This example demonstrates proper handling of type assertion failures using Gossip's error handling patterns.

## Overview

The type assertion example shows:
- How to properly handle type assertion failures using `TypeAssertionError`
- Similar to `redis.Nil` pattern for type mismatches
- How to implement conditional error handling for type-related issues
- Best practices for type-safe event data processing

## Key Concepts Demonstrated

### 1. Safe Type Assertions
Using proper type assertion patterns with error handling instead of panicking.

### 2. Type-Safe Processing
Ensuring type-safe event data processing to prevent runtime panics.

### 3. Conditional Error Handling
Handling type-related errors differently based on context.

### 4. Error Type Integration
How type assertion errors work with other error handling patterns.

## Code Walkthrough

### Event Types and Data Structures

```go
// Event types for our example
const (
	UserCreated gossip.EventType = "user.created"
)

// UserData represents user information
type UserData struct {
	UserID   string
	Username string
	Email    string
}

// BadData represents incorrect data type
type BadData struct {
	ID   int
	Name string
}
```

### Processor with Type Assertion Error Handling

```go
// userProcessor demonstrates proper type assertion with error handling
func userProcessor(ctx context.Context, event *gossip.Event) error {
	// The common scenario: type assertion with error handling
	data, ok := event.Data.(*UserData)
	if !ok {
		// This is the exact scenario you mentioned - instead of just panicking,
		// we return a specific error type that can be handled conditionally
		return goserr.NewTypeAssertionError("*UserData", event.Data, "expected *UserData but got different type")
	}

	// Process the correctly typed data
	log.Printf("[User Processor] Processing user: %s (ID: %s)", data.Username, data.UserID)
	return nil
}
```

### Bad Processor Example

```go
// badUserProcessor demonstrates what happens with wrong data type
func badUserProcessor(ctx context.Context, event *gossip.Event) error {
	// This will fail type assertion
	data, ok := event.Data.(*UserData)
	if !ok {
		// Return the specific TypeAssertionError
		return goserr.NewTypeAssertionError("*UserData", event.Data, "data type mismatch")
	}

	log.Printf("[Bad User Processor] Processing user: %s", data.Username)
	return nil
}
```

### Error Handler with Conditional Logic

```go
// handleError demonstrates conditional error handling
func handleError(err error) {
	switch {
	case goserr.IsTypeAssertion(err):
		// Handle type assertion errors specifically
		typeErr, _ := goserr.AsTypeAssertionError(err)
		log.Printf("[Error Handler] Type assertion failed: expected %s, got %T. This is similar to redis.Nil for type mismatches.",
			typeErr.ExpectedType, typeErr.Value)

	case goserr.IsNoData(err):
		// Handle missing data errors
		log.Printf("[Error Handler] No data found: %v", err)

	case goserr.IsValidation(err):
		// Handle validation errors
		log.Printf("[Error Handler] Validation failed: %v", err)

	// You can also check for specific constants directly
	case errors.Is(err, goserr.ErrNilData):
		log.Printf("[Error Handler] Event data is nil (using constant check)")

	case errors.Is(err, goserr.ErrTypeAssertionFailed):
		log.Printf("[Error Handler] Type assertion failed (using constant check)")

	default:
		// Handle other errors
		log.Printf("[Error Handler] Other error: %v", err)
	}
}
```

### Main Function Setup

```go
func main() {
	fmt.Println("Demonstrating type assertion error handling (similar to redis.Nil pattern)...")

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

	// Subscribe processors
	eventBus.Subscribe(UserCreated, func(ctx context.Context, event *gossip.Event) error {
		err := userProcessor(ctx, event)
		if err != nil {
			handleError(err) // Handle errors conditionally
			return err
		}
		return nil
	})

	// Publish events with correct data type
	fmt.Println("\n1. Publishing event with correct data type:")
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "user-123",
		Username: "alice",
		Email:    "alice@example.com",
	}))

	// Publish events with wrong data type to trigger TypeAssertionError
	fmt.Println("\n2. Publishing event with wrong data type (this will cause TypeAssertionError):")
	eventBus.Publish(gossip.NewEvent(UserCreated, &BadData{
		ID:   123,
		Name: "bob",
	}))

	// Publish event with nil data to demonstrate NoDataError
	fmt.Println("\n3. Publishing event with nil data:")
	eventBus.Publish(gossip.NewEvent(UserCreated, nil))

	// Wait for processing
	fmt.Println("\nProcessing events...")
}
```

## Error Types Used

### TypeAssertionError
For type assertion failures. This is similar to the `redis.Nil` pattern but for type mismatches. When you expect a specific type but get something else, this error type helps you handle the situation gracefully instead of panicking.

## Running the Example

To run this example:

```bash
cd examples/type_assertion_example
go run main.go
```

## Key Takeaways

1. **Safe Type Assertions**: Always use the `value, ok := interface.(Type)` pattern instead of direct type assertion that can panic.

2. **Error Handling**: Return specific error types instead of panicking when type assertions fail.

3. **Conditional Logic**: Use error type checking to handle different error scenarios appropriately.

4. **Type Safety**: The approach provides type safety while maintaining flexibility in the event system.

## Best Practices Demonstrated

- Always use safe type assertion patterns with error checking
- Return specific error types for different failure scenarios
- Implement conditional error handling based on error types
- Use error constants for common error conditions
- Handle type mismatches gracefully instead of panicking