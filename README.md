# Gossip (ﾉ◕ヮ◕)ﾉ*:･ﾟ✧

**Decoupled event handling for Go — publish once, let everyone listen in.** 🗣️

Gossip is a lightweight, type-safe event bus library for Go that implements the observer/pub-sub pattern. It enables clean separation between core business logic and side effects, making your codebase more maintainable and extensible.

## ✨ Features

- **🔒 Strongly-typed events** - No string typos with `EventType` constants
- **⚡ Async by default** - Non-blocking event dispatch with worker pools
- **🔌 Pluggable providers** - In-memory or Redis Pub/Sub support
- **🎯 Event filtering** - Conditional processor execution
- **📦 Batch processing** - Process multiple events efficiently
- **🔧 Middleware support** - Retry, timeout, logging, recovery
- **🎚️ Priority queues** - Handle critical events first
- **🛡️ Thread-safe** - Concurrent publish/subscribe operations
- **🧹 Graceful shutdown** - Wait for in-flight events

## 📦 Installation

```bash
go get github.com/seyallius/gossip
```

## 🚀 Quick Start

### Basic Event Bus Usage

```go
package main

import (
	"context"
	"log"
	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
)

// Define event types
const (
	UserCreated event.EventType = "user.created"
)

type UserData struct {
	UserID   string
	Username string
}

func main() {
	// Initialize event bus with default in-memory provider
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	// Subscribe processor
	eventBus.Subscribe(UserCreated, func(ctx context.Context, eventToPub *event.Event) error {
		data := eventToPub.Data.(*UserData)
		log.Printf("New user: %s", data.Username)
		return nil
	})

	// Publish event
	evnt := event.NewEvent(UserCreated, &UserData{
		UserID:   "123",
		Username: "alice",
	})
	eventBus.Publish(evnt)
}
```

### Using Redis Provider

```go
package main

import (
	"context"
	"log"
	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
)

func main() {
	// Initialize event bus with Redis provider
	config := &bus.Config{
		Driver:     "redis",
		RedisAddr:  "localhost:6379",
		RedisPwd:   "", // Set your Redis password
		RedisDB:    0,
		Workers:    10,
		BufferSize: 1000,
	}

	eventBus := bus.NewEventBus(config)
	defer eventBus.Shutdown()

	// Use the event bus as normal
	eventBus.Subscribe(UserCreated, func(ctx context.Context, eventToPub *event.Event) error {
		// Process event
		return nil
	})

	// Publish event
	evnt := event.NewEvent(UserCreated, &UserData{
		UserID:   "123",
		Username: "alice",
	})
	eventBus.Publish(evnt)
}
```

## ⚠️ Error Handling

Gossip provides standard error types for conditional logic and error handling, including both constructor functions and pre-defined constants similar to `io.EOF` or `redis.Nil`:

```go
import "github.com/seyallius/gossip/event/errors"

// In your event processor - using constructor functions for custom messages
func myProcessor(ctx context.Context, event *event.Event) error {
    // Validate input data
    if event.Data == nil {
        return errors.NewValidationError("data", nil, "event data cannot be nil")
    }

    // Process the event
    if err := processEvent(event); err != nil {
        // Return specific error types for middleware to handle appropriately
        if isTransientError(err) {
            return errors.NewRetryableError(err, "temporary processing failure")
        }
        return errors.NewProcessingError(err, "permanent processing failure")
    }

    return nil
}

// In your event processor - using pre-defined constants for common scenarios
func mySimpleProcessor(ctx context.Context, event *event.Event) error {
    // Type assertion with error handling
    data, ok := event.Data.(*UserData)
    if !ok {
        // Use the pre-defined constant for type assertion failures
        return errors.ErrTypeAssertionFailed
    }

    // Check for nil data
    if data == nil {
        // Use the pre-defined constant for nil data
        return errors.ErrNilData
    }

    // Process the event
    if data.Email == "" {
        // Use the pre-defined constant for validation failures
        return errors.ErrValidationFailed
    }

    return nil
}

// Check error types in your application code
func handleEventError(err error) {
    if errors.IsRetryable(err) {
        log.Println("Will retry the operation")
    } else if errors.IsFatal(err) {
        log.Println("Will not retry - fatal error")
    } else if errors.IsValidation(err) {
        log.Println("Validation failed - check input data")
    } else if errors.IsTypeAssertion(err) {
        log.Println("Type assertion failed - check event data type")
    } else if errors.IsNoData(err) {
        log.Println("No data found - event data is nil")
    }
}

// You can also check for specific constants directly
func handleSpecificError(err error) {
    switch {
    case errors.Is(err, errors.ErrNilData):
        log.Println("Event data is nil")
    case errors.Is(err, errors.ErrTypeAssertionFailed):
        log.Println("Type assertion failed")
    case errors.Is(err, errors.ErrValidationFailed):
        log.Println("Validation failed")
    }
}
```

### Standard Error Types

- `RetryableError` - Transient failures that can be retried
- `ValidationError` - Input validation failures
- `ProcessingError` - General processing failures
- `TimeoutError` - Operations that exceeded time limits
- `FatalError` - Unrecoverable errors that should not be retried
- `TypeAssertionError` - Type assertion failures (similar to redis.Nil for type mismatches)
- `NoDataError` - Missing event data when data was expected
- `InvalidEventError` - Malformed or invalid events
- `UnsupportedEventTypeError` - Event types not supported by a processor

### Pre-defined Error Constants

- `errors.ErrNilData` - Pre-defined nil data error (similar to redis.Nil)
- `errors.ErrTypeAssertionFailed` - Pre-defined type assertion failure
- `errors.ErrInvalidEvent` - Pre-defined invalid event error
- `errors.ErrUnsupportedEventType` - Pre-defined unsupported event type error
- `errors.ErrRetryable` - Pre-defined retryable error
- `errors.ErrValidationFailed` - Pre-defined validation error
- `errors.ErrProcessingFailed` - Pre-defined processing error
- `errors.ErrTimeout` - Pre-defined timeout error
- `errors.ErrFatal` - Pre-defined fatal error

## 🧩 Components Quick Start

### 1. Event Bus (Core)

The event bus handles publishing and subscribing to events:

```go
// Initialize with custom config
config := &bus.Config{
    Workers:    20,     // Number of worker goroutines
    BufferSize: 2000,   // Event channel buffer size
}
eventBus := bus.NewEventBus(config)
defer eventBus.Shutdown()

// Subscribe to events
eventBus.Subscribe(UserCreated, myProcessor)

// Publish events (async)
eventBus.Publish(event)

// Publish events (sync - waits for all processors to complete)
errors := eventBus.PublishSync(ctx, event)
```

### 2. Event Filtering

Conditionally execute processors based on event properties:

```go
import "github.com/seyallius/gossip/event/filter"

// Filter by metadata
highPriorityFilter := filter.FilterByMetadata("priority", "high")

// Combine filters with AND logic
combinedFilter := filter.And(
    filter.FilterByMetadata("priority", "high"),
    filter.FilterByMetadata("source", "api"),
)

// Use filtered processor
filteredProcessor := filter.NewFilteredProcessor(
    highPriorityFilter,
    myProcessor,
)
eventBus.Subscribe(UserCreated, filteredProcessor)
```

### 3. Middleware

Chain behaviors around processors:

```go
import "github.com/seyallius/gossip/event/middleware"

// Chain multiple middleware
processor := middleware.Chain(
    middleware.WithRetry(3, 100*time.Millisecond),
    middleware.WithTimeout(5*time.Second),
    middleware.WithLogging("processing event", log.Println),
    middleware.WithRecovery(),
)(myProcessor)

eventBus.Subscribe(UserCreated, processor)
```

### 4. Batch Processing

Process events in groups for efficiency:

```go
import "github.com/seyallius/gossip/event/batch"

// Batch processor function
batchProcessor := func(ctx context.Context, events []*event.Event) error {
    // Process all events together (e.g., bulk database insert)
    for _, evt := range events {
        data := evt.Data.(*UserData)
        // Process in batch
        log.Printf("Batch processing user: %s", data.Username)
    }
    return nil
}

// Configure batch processing
batchConfig := batch.BatchConfig{
    BatchSize:   100,              // Process in groups of 100
    FlushPeriod: 5 * time.Second,  // Flush every 5 seconds if not full
}

// Create batch processor
processor := batch.NewBatchProcessor(UserCreated, batchConfig, batchProcessor)
eventBus.Subscribe(UserCreated, processor.AsEventProcessor())
```

## 🏗️ Core Concepts

### Event Types
Define strongly-typed event identifiers:
```go
const (
    UserCreated event.EventType = "user.created"
    UserUpdated event.EventType = "user.updated"
)
```

### Events
Events carry data and metadata:
```go
event := event.NewEvent(UserCreated, userData).
    WithMetadata("request_id", "req-123").
    WithMetadata("source", "api")
```

### Processors
Functions that process events:
```go
func myProcessor(ctx context.Context, event *event.Event) error {
    // Process event
    return nil
}
```

## 🧪 Testing

```go
func TestMyProcessor(t *testing.T) {
    eventBus := bus.NewEventBus(bus.DefaultConfig())
    defer eventBus.Shutdown()

    received := false
    processor := func(ctx context.Context, event *event.Event) error {
        received = true
        return nil
    }

    eventBus.Subscribe(UserCreated, processor)
    eventBus.Publish(event.NewEvent(UserCreated, nil))

    time.Sleep(100 * time.Millisecond)
    assert.True(t, received)
}
```

## 🚀 Performance Report

The gossip event library is extremely lightweight and will have **minimal impact** on your application's performance:

- **Publishing Events**: Asynchronous publishing is extremely fast (~140ns) with zero memory allocations when there are no subscribers, and remains efficient (~70-90ns) even with multiple concurrent publishers and subscribers.
- **Subscribing to Events**: Subscription is efficient at ~220ns per operation with minimal memory allocation (120 bytes).
- **Memory Usage**: Most core operations have **zero memory allocations**. The event bus has a configurable buffer size (defaults to 1000 events).
- **Scalability**: The event bus can handle high-concurrency scenarios effectively with multiple worker goroutines processing events in parallel.

For detailed benchmark results, see:
- [Middleware Benchmarks](./event/middleware/middleware_benchmark_test.go)
- [Filter Benchmarks](./event/filter/filter_benchmark_test.go) 
- [Event Bus Benchmarks](./event/bus/event_bus_benchmark_test.go)
- [Batch Processor Benchmarks](./event/batch/batch_benchmark_test.go)

## 🎯 Best Practices

### 1. Event Naming
- Use hierarchical naming: `domain.entity.action`
- Examples: `auth.user.created`, `order.payment.completed`, `inventory.stock.updated`

### 2. Event Data
- Keep event data serializable
- Include only necessary information
- Use metadata for context (request_id, source, etc.)

### 3. Processor Design
- Make processors idempotent when possible
- Handle errors gracefully
- Use context for cancellation/timeout
- Avoid blocking operations when possible

### 4. Configuration
- Adjust `Workers` based on event volume
- Set `BufferSize` based on peak load expectations
- Use batch processing for high-volume scenarios

## 📚 Documentation

For comprehensive documentation, examples, and advanced usage patterns, see:

- **[Examples Directory](./examples/)** - Real-world use cases
    - [Auth Service](./examples/auth_service/) - Authentication with notifications
    - [E-commerce](./examples/ecommerce/) - Order processing with batch emails
    - [Microservices](./examples/microservices/) - Cross-service communication

## 🤝 Contributing

Contributions welcome! Please open an issue or submit a PR.

## 📄 License

MIT License - see LICENSE file for details

## 📖 Implementation Details

For the geeks, see [IMPLEMENTATION.md](./IMPLEMENTATION.md) - a comprehensive deep-dive into how Gossip is implemented, including the concurrency model, thread safety patterns, middleware system, and architectural decisions.

## 🙏 Acknowledgments

Inspired by the need for clean event-driven architecture in Go applications.

---

<p align="center" style="font-style: italic; font-family: 'Gochi Hand',fantasy">
    Built with ❤️ for the Go community
</p>