# Gossip Examples - Comprehensive Guide

This guide provides extensive documentation for using Gossip in real-world applications.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Basic Concepts](#basic-concepts)
3. [Setting Up Your Project](#setting-up-your-project)
4. [Real-World Examples](#real-world-examples)
5. [Advanced Patterns](#advanced-patterns)
6. [Best Practices](#best-practices)
7. [Common Pitfalls](#common-pitfalls)

---

## Getting Started

### Installation

```bash
go get github.com/seyallius/gossip
```

### Import in Your Code

```go
import gossip "github.com/seyallius/gossip/event"
```

---

## Basic Concepts

### 1. Event Types

Event types are strongly-typed string constants that identify events. **Never use raw strings.**

```go
// ✅ Good - Strongly typed
const (
    UserCreated gossip.EventType = "user.created"
    OrderPaid   gossip.EventType = "order.paid"
)

// ❌ Bad - Raw strings prone to typos
bus.Publish(gossip.NewEvent("user.created", data))
```

**Convention:** Use hierarchical naming with dots: `domain.entity.action`

Examples:
- `auth.user.created`
- `auth.login.success`
- `order.payment.completed`
- `inventory.stock.depleted`

### 2. Event Structure

Events contain:
- **Type**: Strongly-typed identifier
- **Timestamp**: Automatic creation time
- **Data**: Event-specific payload (any type)
- **Metadata**: Additional context (map)

```go
event := gossip.NewEvent(UserCreated, &UserData{
    UserID: "123",
    Email:  "user@example.com",
})

// Add metadata for context
event.WithMetadata("request_id", "req-abc-123")
event.WithMetadata("source", "api")
```

### 3. Processors

Processors are functions that process events:

```go
func myProcessor(ctx context.Context, event *event.Event) error {
    // Type assert the data
    data := event.Data.(*UserData)
    
    // Process the event
    log.Printf("Processing user: %s", data.UserID)
    
    // Return error if processing fails
    return nil
}
```

**Key points:**
- Always type-assert `event.Data` to expected type
- Return `error` if processing fails
- Use `ctx` for cancellation/timeout
- Processors should be idempotent when possible

---

## Setting Up Your Project

### Step 1: Define Event Types

Create a dedicated file for event types:

```go
// events/types.go
package events

import gossip "github.com/seyallius/gossip/event"

const (
    // User events
    UserCreated      gossip.EventType = "user.created"
    UserUpdated      gossip.EventType = "user.updated"
    UserDeleted      gossip.EventType = "user.deleted"
    
    // Auth events
    LoginSuccess     gossip.EventType = "auth.login.success"
    LoginFailed      gossip.EventType = "auth.login.failed"
    PasswordChanged  gossip.EventType = "auth.password.changed"
    
    // Order events
    OrderCreated     gossip.EventType = "order.created"
    OrderPaid        gossip.EventType = "order.paid"
    OrderShipped     gossip.EventType = "order.shipped"
)
```

### Step 2: Define Event Data Structures

Create structs for event payloads:

```go
// events/data.go
package events

type UserCreatedData struct {
    UserID   string
    Email    string
    Username string
}

type LoginSuccessData struct {
    UserID    string
    IPAddress string
    UserAgent string
}

type OrderCreatedData struct {
    OrderID    string
    CustomerID string
    Amount     float64
    Items      []string
}
```

### Step 3: Initialize Event Bus

In your main application:

```go
// main.go
package main

import (
	"log"

	"github.com/seyallius/gossip/event/bus"
)

var eventBus *bus.EventBus

func initEventBus() {
	config := &bus.Config{
		Workers:    10,   // Adjust based on load
		BufferSize: 1000, // Adjust based on event volume
	}

	eventBus = bus.NewEventBus(config)
	log.Println("Event bus initialized")
}

func main() {
	initEventBus()
	defer eventBus.Shutdown()

	// Register handlers
	registerProcessors()

	// Start your application
	// ...
}
```

### Step 4: Register Processors

Create handler registration function:

```go
// handlers/registration.go
package handlers

import (
	"github.com/seyallius/gossip"
	"github.com/seyallius/gossip/event/bus"

	"yourapp/events"
)

func RegisterProcessors(eventBus *bus.EventBus) {
	// User handlers
	eventBus.Subscribe(events.UserCreated, EmailNotificationProcessor)
	eventBus.Subscribe(events.UserCreated, AuditLogProcessor)
	eventBus.Subscribe(events.UserCreated, MetricsProcessor)

	// Auth handlers
	eventBus.Subscribe(events.LoginSuccess, SecurityProcessor)
	eventBus.Subscribe(events.LoginSuccess, AnalyticsProcessor)

	// Order handlers
	eventBus.Subscribe(events.OrderCreated, InventoryProcessor)
	eventBus.Subscribe(events.OrderPaid, PaymentProcessorProcessor)
}
```

### Step 5: Publish Events in Business Logic

```go
// services/user_service.go
package services

import (
	"github.com/seyallius/gossip"
	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"

	"yourapp/events"
)

type UserService struct {
	eventBus *bus.EventBus
}

func (s *UserService) CreateUser(email, username string) error {
	// Core business logic
	userID := generateUserID()

	// ... save to database ...

	// Publish event
	eventData := &events.UserCreatedData{
		UserID:   userID,
		Email:    email,
		Username: username,
	}

	event := event.NewEvent(events.UserCreated, eventData).
		WithMetadata("source", "api").
		WithMetadata("request_id", getRequestID())

	s.eventBus.Publish(event)

	return nil
}
```

---

## Real-World Examples

### 1. Authentication Service

**File: `examples/auth_service/main.go`**

This example demonstrates a complete authentication service using Gossip for event-driven architecture.

**Overview**

The example shows:
- User registration with welcome emails
- Login tracking and notifications
- Password change alerts
- Audit logging for all auth events
- Security monitoring

**Run:**
```bash
cd examples/auth_service
go run main.go
```

**Key Concepts Demonstrated**

1. **Event Fan-out**: Multiple handlers for single event
2. **Middleware Composition**: Combining retry, timeout, and recovery middleware
3. **Metadata Usage**: Using metadata for request tracking and context
4. **Security Patterns**: Implementing security monitoring with events

**Components Used**

- Event bus for decoupling services
- Middleware for error handling and recovery
- Metadata for request correlation

### 2. E-commerce Platform

**File: `examples/ecommerce/main.go`**

This example demonstrates an e-commerce platform using Gossip for event-driven order processing and notifications.

**Overview**

The example shows:
- Order creation and processing
- Batch email notifications
- Inventory management
- Conditional analytics (high-value orders)
- Payment processing pipeline

**Run:**
```bash
cd examples/ecommerce
go run main.go
```

**Key Concepts Demonstrated**

1. **Batch Processing**: Efficient processing of high-volume events
2. **Event Filtering**: Conditional processing based on event properties
3. **Async Operations**: Asynchronous inventory updates
4. **Pipeline Processing**: Sequential event processing patterns

**Components Used**

- Batch processors for efficient email sending
- Event filters for conditional processing
- Multiple event handlers for different business operations

### 3. Microservices Communication

**File: `examples/microservices/main.go`**

This example demonstrates microservices communication using Gossip for cross-service event-driven architecture.

**Overview**

The example shows:
- Cross-service event communication
- Service-to-service decoupling
- Multiple services reacting to same event
- Event chaining (service publishes events other services consume)

**Run:**
```bash
cd examples/microservices
go run main.go
```

**Key Concepts Demonstrated**

1. **Service Decoupling**: Loose coupling between services using events
2. **Event Chaining**: Services publishing events that trigger other services
3. **Shared Event Bus**: Using a common event bus across multiple services
4. **Self-Registering Handlers**: Services registering their own event handlers

**Components Used**

- Shared event bus for inter-service communication
- Self-registering event handlers
- Event-driven orchestration patterns

### 4. Error Handling with Conditional Logic

**File: `examples/error_handling/main.go`**

This example demonstrates how to use Gossip's standard error types for conditional error handling in event-driven applications.

**Overview**

The example shows:
- How to use different error types (`RetryableError`, `ValidationError`, `FatalError`, etc.)
- How to implement conditional error handling based on error types
- How to create custom middleware that handles errors differently based on their type
- How to combine error handling with other middleware like retry and timeout

**Run:**
```bash
cd examples/error_handling
go run main.go
```

**Key Concepts Demonstrated**

1. **Standard Error Types**: Using `gossip/event/errors` package to create typed errors
2. **Conditional Error Handling**: Using `IsRetryable`, `IsValidation`, etc. to handle different error types differently
3. **Integration with Middleware**: How error types work with existing middleware like retry and timeout
4. **Custom Error Handling Middleware**: Creating middleware that responds to specific error types

**Error Types Used**

- `RetryableError`: For transient failures that should be retried
- `ValidationError`: For input validation failures
- `FatalError`: For unrecoverable errors that should not be retried
- `TimeoutError`: For operations that exceed time limits
- `ProcessingError`: For general processing failures

### 5. Type Assertion Error Handling

**File: `examples/type_assertion_example/main.go`**

This example demonstrates proper handling of type assertion failures using Gossip's error handling patterns.

**Overview**

The example shows:
- How to properly handle type assertion failures using `TypeAssertionError`
- Similar to `redis.Nil` pattern for type mismatches
- How to implement conditional error handling for type-related issues
- Best practices for type-safe event data processing

**Run:**
```bash
cd examples/type_assertion_example
go run main.go
```

**Key Concepts Demonstrated**

1. **Safe Type Assertions**: Using proper type assertion patterns with error handling
2. **Type-Safe Processing**: Ensuring type-safe event data processing
3. **Conditional Error Handling**: Handling type-related errors differently based on context
4. **Error Type Integration**: How type assertion errors work with other error handling patterns

**Error Types Used**

- `TypeAssertionError`: For type assertion failures

---

## Advanced Patterns

### 1. Middleware Composition

Chain multiple middleware for robust handling:

```go
handler := middleware.Chain(
    middleware.WithRecovery(),                      // Catch panics
    middleware.WithRetry(3, 100*time.Millisecond),  // Retry on failure
    middleware.WithTimeout(5*time.Second),          // Prevent hanging
    middleware.WithLogging(),                       // Log execution
)(myProcessor)

bus.Subscribe(UserCreated, handler)
```

**Order matters:** Place recovery first to catch all panics.

### 2. Conditional Processing with Filters

Only execute handlers when conditions are met:

```go

// Only process events from specific source
apiFilter := filter.FilterByMetadata("source", "api")
bus.GetGlobalBus().Subscribe(UserCreated, gossip.NewFilteredProcessor(apiFilter, apiProcessor))

// Combine filters with AND/OR logic
complexFilter := filter.And(
    filter.FilterByMetadataExists("user_id"),
    filter.Or(
        filter.FilterByMetadata("source", "api"),
        filter.FilterByMetadata("source", "web"),
    ),
)
```

### 3. Batch Processing

Efficient processing of high-volume events:

```go
batchConfig := batch.BatchConfig{
    BatchSize:   100,              // Process 100 events at once
    FlushPeriod: 5 * time.Second,  // Or every 5 seconds
}

batchProcessor := func(ctx context.Context, events []*event.Event) error {
    // Process all events together
    log.Printf("Processing batch of %d events", len(events))
    
    // Bulk database insert, bulk email send, etc.
    return bulkInsertToDatabase(events)
}

processor := gossip.NewBatchProcessor(OrderCreated, batchConfig, batchProcessor)
defer processor.Shutdown()

bus.Subscribe(OrderCreated, processor.AsEventProcessor())
```

**When to use:**
- Email notifications (batch send)
- Database inserts (bulk insert)
- API calls (batch requests)
- Metric aggregation

### 4. Synchronous Processing

When you need immediate results:

```go
event := gossip.NewEvent(CriticalTransaction, data)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

errors := bus.PublishSync(ctx, event)
if len(errors) > 0 {
    // Some handlers failed
    log.Printf("Processor failures: %v", errors)
    return handleFailure(errors)
}

log.Println("All handlers succeeded")
```

**Use cases:**
- Critical transactions requiring validation
- Rollback scenarios
- Immediate feedback needed
- Testing


---

## Best Practices

### 1. Event Naming Conventions

✅ **Do:**
- Use hierarchical naming: `domain.entity.action`
- Be specific: `order.payment.completed` not `order.updated`
- Use past tense: `user.created` not `user.create`
- Group related events: `auth.login.*`, `order.payment.*`

❌ **Don't:**
- Use generic names: `updated`, `changed`
- Mix tenses: `user.creating`, `order.created`
- Use abbreviations: `usr.crtd`

### 2. Processor Design

✅ **Do:**
- Keep handlers small and focused
- Make handlers idempotent when possible
- Return errors for failures
- Use context for cancellation
- Log handler actions

❌ **Don't:**
- Block for long periods
- Panic in handlers (use middleware recovery)
- Modify shared state without locking
- Ignore errors

### 3. Error Handling

```go
func myProcessor(ctx context.Context, event *event.Event) error {
    // Type assertion with check
    data, ok := event.Data.(*UserData)
    if !ok {
        return fmt.Errorf("invalid event data type")
    }
    
    // Business logic with error handling
    if err := processUser(data); err != nil {
        return fmt.Errorf("failed to process user: %w", err)
    }
    
    return nil
}
```

### 4. Testing Processors

```go
func TestUserCreatedProcessor(t *testing.T) {
    // Create test event
    event := gossip.NewEvent(UserCreated, &UserData{
        UserID: "test-123",
        Email:  "test@example.com",
    })
    
    // Execute handler
    err := myProcessor(context.Background(), event)
    
    // Assert results
    assert.NoError(t, err)
    assert.True(t, emailWasSent("test@example.com"))
}
```

### 5. Configuration Tuning

```go
// Low volume (< 100 events/sec)
config := &bus.Config{
    Workers:    5,
    BufferSize: 500,
}

// Medium volume (100-1000 events/sec)
config := &bus.Config{
    Workers:    10,
    BufferSize: 1000,
}

// High volume (> 1000 events/sec)
config := &bus.Config{
    Workers:    20,
    BufferSize: 5000,
}
```

---

## Common Pitfalls

### 1. Type Assertion Panics

❌ **Wrong:**
```go
data := event.Data.(*UserData) // Panics if wrong type
```

✅ **Correct:**
```go
data, ok := event.Data.(*UserData)
if !ok {
    return fmt.Errorf("invalid data type")
}
```

### 2. Blocking Processors

❌ **Wrong:**
```go
func slowProcessor(ctx context.Context, event *event.Event) error {
    time.Sleep(30 * time.Second) // Blocks worker
    return nil
}
```

✅ **Correct:**
```go
func fastProcessor(ctx context.Context, event *event.Event) error {
    // Offload heavy work
    go processInBackground(event.Data)
    return nil
}
```

### 3. Forgetting Shutdown

❌ **Wrong:**
```go
func main() {
    bus := gossip.NewEventBus(config)
    // ... application code ...
    // No cleanup!
}
```

✅ **Correct:**
```go
func main() {
    bus := gossip.NewEventBus(config)
    defer bus.Shutdown() // Graceful cleanup
    // ... application code ...
}
```

### 4. Circular Event Dependencies

❌ **Wrong:**
```go
// Processor A publishes Event B
func handlerA(ctx context.Context, event *event.Event) error {
    bus.Publish(gossip.NewEvent(EventB, nil))
    return nil
}

// Processor B publishes Event A (circular!)
func handlerB(ctx context.Context, event *event.Event) error {
    bus.Publish(gossip.NewEvent(EventA, nil))
    return nil
}
```

✅ **Correct:**
- Avoid circular dependencies
- Use event metadata to track origin and prevent loops
- Design event flow as a DAG (Directed Acyclic Graph)

### 5. Overusing Synchronous Publishing

❌ **Wrong:**
```go
// Using sync for everything defeats the purpose
for _, item := range items {
    bus.PublishSync(ctx, event) // Slow!
}
```

✅ **Correct:**
```go
// Use async for most cases
for _, item := range items {
    bus.Publish(event) // Fast!
}

// Use sync only when necessary
criticalEvent := gossip.NewEvent(CriticalOp, data)
errors := bus.PublishSync(ctx, criticalEvent)
```

---

## Performance Tips

1. **Tune worker count** based on handler latency
2. **Use batch processing** for high-volume events
3. **Implement filtering** to avoid unnecessary work
4. **Monitor buffer fullness** - increase if events are dropped
5. **Profile handler performance** - optimize slow handlers
6. **Use middleware wisely** - each layer adds overhead

---

## Troubleshooting

### Events Not Being Processed

1. Check if handlers are registered before publishing
2. Verify event type matches subscription
3. Ensure bus hasn't been shutdown
4. Check logs for handler errors

### Slow Event Processing

1. Profile handler execution time
2. Check for blocking operations
3. Increase worker count
4. Use batch processing

### Memory Issues

1. Reduce buffer size
2. Implement event filtering
3. Fix handler memory leaks
4. Monitor goroutine count

---

## Additional Resources

- [Main README](../README.md) - Library overview
- [API Documentation](https://pkg.go.dev/github.com/seyallius/gossip)
- [GitHub Issues](https://github.com/seyallius/gossip/issues)

---

**Questions?** Open an issue on GitHub or reach out to the community!