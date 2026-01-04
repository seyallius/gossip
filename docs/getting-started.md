# Getting Started with Gossip

This guide will help you get up and running with Gossip, from installation to your first event-driven application.

## 📦 Installation

To install Gossip, use `go get`:

```bash
go get github.com/seyallius/gossip
```

## 🚀 Quick Start

### Basic Event Bus Usage

Here's a simple example to get you started:

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

For distributed systems, you can use the Redis provider:

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

## ⚙️ Configuration

### Basic Configuration

```go
config := &bus.Config{
    Workers:    20,     // Number of worker goroutines
    BufferSize: 2000,   // Event channel buffer size
}
eventBus := bus.NewEventBus(config)
defer eventBus.Shutdown()
```

### Provider Configuration

Gossip supports different transport providers:

**Memory Provider (Default):**
- Fastest option for single-application scenarios
- Events stay within the same process
- No external dependencies

**Redis Provider:**
- For distributed systems across multiple services
- Events can cross process boundaries
- Requires Redis server

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

## Next Steps

Now that you've learned the basics, explore more advanced topics:

- [Implementation Guide](./implementation/overview.md) - Deep dive into how Gossip works
- [Examples](./examples/README.md) - Real-world use cases
- [API Reference](./implementation/api.md) - Complete API documentation