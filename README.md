# Gossip (ﾉ◕ヮ◕)ﾉ*:･ﾟ✧

**Decoupled event handling for Go — publish once, let everyone listen in.** 🗣️

Gossip is a lightweight, type-safe event bus library for Go that implements the observer/pub-sub pattern. It enables clean separation between core business logic and side effects, making your codebase more maintainable and extensible.

## ✨ Features

- **🔒 Strongly-typed events** - No string typos with `EventType` constants
- **⚡ Async by default** - Non-blocking event dispatch with worker pools
- **🔄 Synchronous option** - When you need immediate processing
- **🎯 Event filtering** - Conditional processor execution
- **📦 Batch processing** - Process multiple events efficiently
- **🔄 Bulk subscription** - Subscribe to multiple event types with a single processor
- **🔧 Middleware support** - Retry, timeout, logging, recovery
- **🛡️ Thread-safe** - Concurrent publish/subscribe operations
- **🧹 Graceful shutdown** - Drains the backlog, so accepted events are not dropped on the way out

Gossip is **in-process**: `Publish` puts the event on a Go channel and a worker goroutine picks it up.
There is no broker and no network hop. It is a way to structure one application around events, not a
replacement for Kafka, NATS or Redis Pub/Sub.

## 📦 Installation

```bash
go get github.com/seyallius/gossip/v2
```

## 🚀 Quick Start

### Basic Event Bus Usage

```go
package main

import (
	"context"
	"log"
	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/bus"
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
	// Initialize event bus. Shutdown drains everything the bus accepted before it returns.
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer func() {
		if err := eventBus.Shutdown(); err != nil {
			log.Printf("event bus shutdown did not finish cleanly: %v", err)
		}
	}()

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
	if !eventBus.Publish(evnt) {
		log.Printf("event bus rejected the %s event", UserCreated)
	}
}
```

## ⚠️ Error Handling

Gossip provides standard error types for conditional logic and error handling, including both constructor functions and pre-defined constants similar to `io.EOF` or `redis.Nil`:

```go
import "github.com/seyallius/gossip/v2/event/errors"

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

Each type has a `New<Type>Error` constructor taking an explicit cause and a `New<Type>Errorf` that
formats one. The formatted variants put the text in the `Err` field, so the format string may use
`%w` to wrap a cause:

```go
err := errors.NewRetryableErrorf("connect to db: %w", cause)
errors.Is(err, cause)   // true
errors.IsRetryable(err) // true
```

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
// Initialize with custom config. Any field left at zero falls back to its default:
// 10 workers, a buffer of 1000, and a 5 second shutdown timeout.
config := &bus.Config{
    Workers:         20,              // Number of worker goroutines
    BufferSize:      2000,            // Event channel buffer size
    ShutdownTimeout: 5 * time.Second, // How long Shutdown waits for the backlog to drain
}
eventBus := bus.NewEventBus(config)
defer func() { _ = eventBus.Shutdown() }()

// Subscribe to events (keep the ID if you will ever unsubscribe)
subscriptionID := eventBus.Subscribe(UserCreated, myProcessor)

// Publish events (async) - false means the event was dropped
if !eventBus.Publish(event) {
    log.Printf("event bus rejected the %s event", UserCreated)
}

// Publish events (sync - waits for all processors to complete)
errors := eventBus.PublishSync(ctx, event)
```

### 2. Event Filtering

Conditionally execute processors based on event properties:

```go
import "github.com/seyallius/gossip/v2/event/filter"

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
import "github.com/seyallius/gossip/v2/event/middleware"

// Chain multiple middleware. Order is outermost-first, so WithRecovery goes last: it turns a panic
// into an error, and everything wrapped around it gets to see that error.
processor := middleware.Chain(
    middleware.WithRetry(3, 100*time.Millisecond),
    middleware.WithTimeout(5*time.Second),
    middleware.WithLogging("processing event", func(message string) { log.Println(message) }),
    middleware.WithRecovery(),
)(myProcessor)

eventBus.Subscribe(UserCreated, processor)
```

`WithRecovery` is not optional insurance: a processor that panics takes down the worker goroutine, and
with it the process. The bus does not recover on your behalf.

### 4. Batch Processing

Process events in groups for efficiency:

```go
import "github.com/seyallius/gossip/v2/event/batch"

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
    BatchSize:   100,             // Process in groups of 100
    FlushPeriod: 5 * time.Second, // Flush every 5 seconds if not full; 0 disables time-based flushing
    OnError: func(ctx context.Context, events []*event.Event, err error) {
        log.Printf("batch of %d events failed: %v", len(events), err)
    },
}

// Create batch processor
processor := batch.NewBatchProcessor(UserCreated, batchConfig, batchProcessor)
eventBus.Subscribe(UserCreated, processor.AsEventProcessor())

// Order matters on the way out: the bus feeds the batch, so drain the bus first
defer func() {
    _ = eventBus.Shutdown()
    processor.Shutdown()
}()
```

Without `OnError`, a failing `BatchProcessor` is silent -- the batch is gone and nothing is told.

### 5. Safe Type Casting
Safely cast event data with configurable nil-checking strategies to avoid the infamous "typed nil" panic trap:
```go
import "github.com/seyallius/gossip/v2/event/utils"

// 🚀 Fast path (default): Only checks for untyped nils (Zero overhead)
data, err := utils.CastEventData[*UserData](event)

// 🛡️ Safe path: Uses reflection to catch typed nils
data, err := utils.CastEventData[*UserData](event, utils.WithReflectNilCheck())

// ⚡ Hot path: Zero-allocation custom checker (Best for high-throughput)
data, err := utils.CastEventData[*UserData](event, utils.WithCustomNilCheck(MyFastChecker))
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

### Bulk Subscription

Subscribe to multiple event types with a single processor:

```go
// Define multiple event types
const (
    UserCreated  event.EventType = "user.created"
    UserUpdated  event.EventType = "user.updated"
    UserDeleted  event.EventType = "user.deleted"
)

// Subscribe to all user events with one processor
eventTypes := []event.EventType{UserCreated, UserUpdated, UserDeleted}
subscriptionIDs := eventBus.SubscribeMultiple(eventTypes, func(ctx context.Context, e *event.Event) error {
    switch e.Type {
    case UserCreated:
        data := e.Data.(*UserData)
        log.Printf("New user created: %s", data.Username)
    case UserUpdated:
        data := e.Data.(*UserData)
        log.Printf("User updated: %s", data.Username)
    case UserDeleted:
        data := e.Data.(*UserData)
        log.Printf("User deleted: %s", data.Username)
    }
    return nil
})

// Clean up all subscriptions at once
defer eventBus.UnsubscribeMultiple(subscriptionIDs)
```

## ⚡ Performance

Gossip is designed for high-performance event processing. You can view the complete benchmark tests in these files:

- [event/bus/event_bus_benchmark_test.go](./event/bus/event_bus_benchmark_test.go) - Core EventBus benchmarks
- [event/batch/batch_benchmark_test.go](./event/batch/batch_benchmark_test.go) - Batch processing benchmarks
- [event/batch/batch_consolidated_benchmark_test.go](./event/batch/batch_consolidated_benchmark_test.go) - Table-driven batch benchmarks
- [event/filter/filter_benchmark_test.go](./event/filter/filter_benchmark_test.go) - Filter benchmarks
- [event/filter/filter_consolidated_benchmark_test.go](./event/filter/filter_consolidated_benchmark_test.go) - Table-driven filter benchmarks
- [event/middleware/middleware_benchmark_test.go](./event/middleware/middleware_benchmark_test.go) - Middleware benchmarks
- [event/middleware/middleware_consolidated_benchmark_test.go](./event/middleware/middleware_consolidated_benchmark_test.go) - Table-driven middleware benchmarks

### Where the Time Goes

Publishing is a non-blocking send on a buffered channel, so it costs a channel operation and nothing
else -- no serialization, no network, no allocation in the common path. The work happens on the worker
goroutines, where the cost is whatever your processors do.

The parts of the library that do allocate:

- **Subscribing** allocates the `Subscription` and, when the slice grows, a new backing array.
- **Unsubscribing** allocates a fresh slice, because dispatch iterates without holding the lock and
  shifting elements down under it would be a data race. It is the rare operation, deliberately paying
  so the frequent one does not.
- **`WithTimeout`** starts a goroutine and a channel per event. It is the one middleware with a real
  per-event cost.
- **Filters, `WithRetry` on the success path, and `WithLogging`** are function calls.

### Benchmark Results

Measured on an Intel Core Ultra 7 265K, `go test -bench=. -benchmem ./event/...`. Treat these as
orders of magnitude, not promises:

```
BenchmarkEventBus_PublishAsync_NoProcessors-20      133.9 ns/op      0 B/op    0 allocs/op
BenchmarkEventBus_Subscribe-20                      215.7 ns/op    137 B/op    3 allocs/op
BenchmarkBatchProcessor_Add-20                       23.3 ns/op      8 B/op    0 allocs/op
BenchmarkFilterByMetadata-20                         29.8 ns/op      0 B/op    0 allocs/op
BenchmarkWithTimeout-20                             1427   ns/op    560 B/op    8 allocs/op
```

The shape to notice: publishing and filtering allocate nothing, subscribing allocates a little, and
`WithTimeout` is an order of magnitude more expensive than everything else because it starts a
goroutine and a channel for every event.

### Performance Recommendations

1. **Tune `Workers` and `BufferSize`** to your workload -- workers for how much runs in parallel, buffer for how large a burst you can absorb before `Publish` starts reporting drops
2. **Batch Processing** is highly recommended for high-volume scenarios (1000+ events/second)
3. **Event Filtering** has minimal overhead and can reduce unnecessary processing
4. **Middleware** adds minimal overhead except for timeout middleware, which starts a goroutine per event

## 🧪 Testing

Synchronise on `Shutdown()`, not on a sleep. It drains everything the bus has accepted and returns
when the workers are done, so the assertion is exact and there is no timing to tune:

```go
func TestMyProcessor(t *testing.T) {
    eventBus := bus.NewEventBus(bus.DefaultConfig())
    defer eventBus.Shutdown() // idempotent, so this is just a safety net

    received := false
    processor := func(ctx context.Context, event *event.Event) error {
        received = true
        return nil
    }

    eventBus.Subscribe(UserCreated, processor)
    require.True(t, eventBus.Publish(event.NewEvent(UserCreated, nil)))

    require.NoError(t, eventBus.Shutdown())
    assert.True(t, received)
}
```

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

### 5. Check What the API Tells You
- `Publish` returns `false` when the event was dropped -- a full buffer or a bus shutting down
- `Shutdown` returns an error when the backlog outran `ShutdownTimeout`
- `BatchConfig.OnError` is the only way to learn that a batch failed
- Wrap processors in `middleware.WithRecovery()`; an unrecovered panic takes down the process

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