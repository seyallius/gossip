# Getting Started with Gossip

This guide will help you get up and running with Gossip, from installation to your first event-driven application.

## 📦 Installation

To install Gossip, use `go get`:

```bash
go get github.com/seyallius/gossip/v2
```

## 🚀 Quick Start

### Basic Event Bus Usage

Here's a simple example to get you started:

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

	// Example of SubscribeMultiple - subscribe to multiple event types with one processor
	const (
		UserUpdated event.EventType = "user.updated"
		UserDeleted event.EventType = "user.deleted"
	)

	type UserUpdatedData struct {
		UserID   string
		Username string
	}

	eventTypes := []event.EventType{UserCreated, UserUpdated, UserDeleted}
	subscriptionIDs := eventBus.SubscribeMultiple(eventTypes, func(ctx context.Context, eventToPub *event.Event) error {
		switch eventToPub.Type {
		case UserCreated:
			data := eventToPub.Data.(*UserData)
			log.Printf("New user: %s", data.Username)
		case UserUpdated:
			data := eventToPub.Data.(*UserUpdatedData)
			log.Printf("Updated user: %s", data.Username)
		case UserDeleted:
			log.Printf("User deleted")
		}
		return nil
	})

	// Example of UnsubscribeMultiple - clean up multiple subscriptions
	defer func() {
		eventBus.UnsubscribeMultiple(subscriptionIDs)
	}()
}
```

> We recommend using global bus singleton instance. See below:

### Basic event usage with global instance (recommended)

```go
package main

import (
	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/bus"
)

func main() {
	eventBus := bus.InitGlobalBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	// use eventBus just like above
}

// in another package, get the singleton instance like this:
package another

func DoSomething() {
	err := bus.GetGlobalBus().
		Publish(event.NewEvent(event.SomeEventType, "publishing some string event"))
    // do something with error
}
```

### Bulk Subscription

Subscribe to multiple event types with a single processor:

```go
package main

import (
	"context"
	"log"
	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/bus"
)

const (
	UserCreated event.EventType = "user.created"
	UserUpdated event.EventType = "user.updated"
	UserDeleted event.EventType = "user.deleted"
)

type UserData struct {
	UserID   string
	Username string
	Email    string
}

func main() {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	// Subscribe to multiple event types with one processor
	eventTypes := []event.EventType{UserCreated, UserUpdated, UserDeleted}
	subscriptionIDs := eventBus.SubscribeMultiple(eventTypes, func(ctx context.Context, e *event.Event) error {
		switch e.Type {
		case UserCreated:
			data := e.Data.(*UserData)
			log.Printf("New user: %s", data.Username)
		case UserUpdated:
			data := e.Data.(*UserData)
			log.Printf("User updated: %s", data.Username)
		case UserDeleted:
			data := e.Data.(*UserData)
			log.Printf("User deleted: %s", data.Username)
		}
		return nil
	})

	// Clean up all subscriptions
	defer eventBus.UnsubscribeMultiple(subscriptionIDs)

	// Publish events
	eventBus.Publish(event.NewEvent(UserCreated, &UserData{
		UserID:   "123",
		Username: "alice",
		Email:    "alice@example.com",
	}))

	eventBus.Publish(event.NewEvent(UserUpdated, &UserData{
		UserID:   "123",
		Username: "alice_updated",
		Email:    "alice@example.com",
	}))
}
```

### Subscribing to Multiple Event Types

One processor can serve several event types, which is what you want for audit logging, metrics, or
any other cross-cutting concern:

```go
package main

import (
	"context"
	"log"
	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/bus"
)

func main() {
	config := &bus.Config{
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

	// Example of SubscribeMultiple - subscribe to multiple event types with one processor
	const (
		UserUpdated event.EventType = "user.updated"
		UserDeleted event.EventType = "user.deleted"
	)

	type UserUpdatedData struct {
		UserID   string
		Username string
	}

	eventTypes := []event.EventType{UserCreated, UserUpdated, UserDeleted}
	subscriptionIDs := eventBus.SubscribeMultiple(eventTypes, func(ctx context.Context, eventToPub *event.Event) error {
		switch eventToPub.Type {
		case UserCreated:
			data := eventToPub.Data.(*UserData)
			log.Printf("New user: %s", data.Username)
		case UserUpdated:
			data := eventToPub.Data.(*UserUpdatedData)
			log.Printf("Updated user: %s", data.Username)
		case UserDeleted:
			log.Printf("User deleted")
		}
		return nil
	})

	// Example of UnsubscribeMultiple - clean up multiple subscriptions
	defer func() {
		eventBus.UnsubscribeMultiple(subscriptionIDs)
	}()
}
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

### Multiple Event Subscription
Subscribe to multiple event types with a single processor:
```go
eventTypes := []gossip.EventType{UserCreated, UserUpdated, UserDeleted}
subscriptionIDs := eventBus.SubscribeMultiple(eventTypes, func(ctx context.Context, event *event.Event) error {
    switch event.Type {
    case UserCreated:
        // Handle user created
    case UserUpdated:
        // Handle user updated
    case UserDeleted:
        // Handle user deleted
    }
    return nil
})
```

### Unsubscribing
Remove subscriptions when no longer needed:
```go
// Unsubscribe single subscription
eventBus.Unsubscribe(subscriptionID)

// Unsubscribe multiple subscriptions
eventBus.UnsubscribeMultiple(subscriptionIDs)
```

## ⚙️ Configuration

### Basic Configuration

```go
config := &bus.Config{
    Workers:         20,              // Number of worker goroutines
    BufferSize:      2000,            // Event channel buffer size
    ShutdownTimeout: 5 * time.Second, // How long Shutdown waits for the backlog to drain
}
eventBus := bus.NewEventBus(config)
defer func() {
    if err := eventBus.Shutdown(); err != nil {
        log.Printf("event bus shutdown did not finish cleanly: %v", err)
    }
}()
```

Every field falls back to its default when left at zero or below: 10 workers, a buffer of 1000, and a
5 second shutdown timeout. Out-of-range values are replaced rather than honoured, since none of them
describes a working bus -- zero workers accepts events and never processes them, and a zero buffer
makes the channel unbuffered, which turns `Publish`'s non-blocking send into a drop whenever no worker
happens to be free.

### Scope

Gossip is **in-process**. `Publish` puts the event on a Go channel and a worker goroutine picks it up:
there is no broker, no network hop, and no delivery to other processes. It is a way to structure one
application around events, and it is not a replacement for Kafka, NATS or Redis Pub/Sub.

A consequence worth stating plainly: events that are still in the buffer when the process dies are
gone. `Shutdown` drains what has been accepted, but nothing survives a crash.

## 🧩 Components Quick Start

### 1. Event Bus (Core)

The event bus handles publishing and subscribing to events:

```go
// Initialize with custom config
config := &bus.Config{
    Workers:    20,   // Number of worker goroutines
    BufferSize: 2000, // Event channel buffer size
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

Note that `WithRecovery` is not optional insurance. A processor that panics takes down the worker
goroutine, and with it the process -- the bus does not recover on your behalf.

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
`processor.Shutdown()` flushes the partial batch and blocks until it has actually been processed.

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

`PublishSync` is the other option, when the test wants the processors' errors as well.

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

## Next Steps

Now that you've learned the basics, explore more advanced topics:

- [Implementation Guide](./implementation/overview.md) - Deep dive into how Gossip works
- [Examples](./examples/README.md) - Real-world use cases
- [API Reference](./implementation/api.md) - Complete API documentation