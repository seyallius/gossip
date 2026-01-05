# Implementation Overview

This section provides a comprehensive overview of how Gossip is implemented, including the core architecture, concurrency model, and design decisions.

## Core Architecture

The core of Gossip is built around several key components:

1. **EventBus** - The main coordinator that delegates operations to the underlying provider
2. **Provider** - Interface that abstracts the underlying transport mechanism (in-memory, Redis, etc.)
3. **Event** - The data structure that carries information through the system
4. **EventProcessor** - Functions that process events
5. **Middleware** - Pluggable functionality that wraps processors
6. **Filters** - Conditional execution of processors based on event properties

### Core Data Types

```go
// EventProcessor is a function that processes events.
type EventProcessor func(ctx context.Context, event *Event) error

// Subscription represents a registered event processor.
type Subscription struct {
    Id         string
    Type       EventType
    EProcessor EventProcessor
}

// EventBus manages the specific provider.
type EventBus struct {
    provider Provider
}

// Provider defines the behavior for an event transport engine (e.g., Memory, Redis, NATS).
type Provider interface {
    Publish(event *Event) error                                                                     // Publish sends an event to the underlying transport.
    Subscribe(eventType EventType, processor EventProcessor) (subscriptionId string, err error) // Subscribe registers a processor for a topic.
    Unsubscribe(subscriptionID string) error                                                              // Unsubscribe removes a subscription.
    Shutdown() error                                                                                      // Shutdown cleans up resources.
}

// EventBus provides methods for event handling.
type EventBus struct {
    Subscribe(eventType EventType, processor EventProcessor) string                    // Subscribe registers a processor for an event type.
    SubscribeMultiple(eventTypes []EventType, processor EventProcessor) []string       // SubscribeMultiple registers a processor for multiple event types.
    Unsubscribe(subscriptionId string)                                                 // Unsubscribe removes a subscription by ID.
    UnsubscribeMultiple(subscriptionIds []string)                                      // UnsubscribeMultiple removes multiple subscriptions by IDs.
    Publish(event *Event) error                                                        // Publish sends an event to the bus.
    Shutdown() error                                                                   // Shutdown cleans up resources.
}
```

### Provider Architecture

Gossip implements a flexible provider-based architecture that allows pluggable transport mechanisms. This design enables the same API to work with different underlying technologies like in-memory channels or distributed systems like Redis.

#### Provider Interface

The `Provider` interface defines the contract that all transport implementations must satisfy:

```go
type Provider interface {
    Publish(event *Event) error
    Subscribe(eventType EventType, processor EventProcessor) (subscriptionId string, err error)
    Unsubscribe(subscriptionID string) error
    Shutdown() error
}
```

#### Available Providers

1. **InMemoryProvider** - Uses Go channels for local, in-process event communication
2. **RedisProvider** - Uses Redis Pub/Sub for distributed event communication across multiple services

#### Configuration

The architecture introduces a driver-based configuration system:

```go
type Config struct {
    Driver     string // "memory" or "redis"
    Workers    int    // Number of worker goroutines
    BufferSize int    // Size of the event channel buffer
    RedisAddr  string // Redis address (for Redis provider)
    RedisPwd   string // Redis password (for Redis provider)
    RedisDB    int    // Redis database (for Redis provider)
}
```

## Event Types and Data Structures

### EventType

```go
// EventType represents a strongly-typed event identifier to prevent typos and ensure consistency.
type EventType string
```

We use a custom type `EventType` instead of raw strings to provide compile-time type safety. This prevents typos and ensures consistent event naming across the application.

**Why string?** Because events need names. But why not just use raw strings everywhere?

```go
// BAD - typos everywhere
bus.Publish("user.created", data)
bus.Subscribe("user.craeted", processor) // TYPO! Won't work
```

By using a **type alias**, we can define constants:

```go
const UserCreated EventType = "user.created"
```

Now the compiler helps us. If you type `UserCraeted` instead of `UserCreated`, **compile error**. Plus, your IDE autocompletes it.

### Event Structure

```go
// Event represents a generic event that flows through the system.
type Event struct {
    Type      EventType      // Strongly-typed event identifier
    Timestamp time.Time      // When the event occurred
    Data      any            // Event-specific payload
    Metadata  map[string]any // Additional context (e.g., request ID, user agent)
}
```

The `Event` struct is designed to be generic enough to carry any payload while providing essential metadata. The `Data` field uses `any` (interface{}) to allow any type of data payload, but type assertion should be used carefully in processors.

The `Metadata` field allows for additional contextual information like request IDs, user agents, or priority levels without requiring changes to the event data structure.

### Event Creation and Helper Methods

```go
// NewEvent creates a new event with the given type and data.
func NewEvent(eventType EventType, data any) *Event {
    return &Event{
        Type:      eventType,
        Timestamp: time.Now(),
        Data:      data,
        Metadata:  make(map[string]any),
    }
}

// WithMetadata adds metadata to the event and returns the event for chaining.
func (e *Event) WithMetadata(key string, value any) *Event {
    e.Metadata[key] = value
    return e
}
```

The `NewEvent` function automatically sets the timestamp when created, providing a consistent way to track when events occurred. The `WithMetadata` method enables method chaining for fluent API usage.

This is **method chaining**. It lets you write beautiful code:

```go
event := NewEvent(UserCreated, data).
    WithMetadata("request_id", "req-123").
    WithMetadata("source", "api").
    WithMetadata("priority", "high")
```

Instead of:

```go
event := NewEvent(UserCreated, data)
event.Metadata["request_id"] = "req-123"
event.Metadata["source"] = "api"
event.Metadata["priority"] = "high"
```

Much cleaner.

## Design Decisions

### Why `any` for Event Data?

We chose `any` (interface{}) for the event data field to maintain maximum flexibility while acknowledging that users should be careful with type assertions. This allows the event bus to carry any type of payload without requiring all events to implement a specific interface.

### Why Buffer Size in Channel?

The buffered channel provides decoupling between publishers and processors. When the system experiences bursts of events, the buffer absorbs the load temporarily, preventing publishers from being blocked. However, if the buffer fills up, events are dropped to maintain system responsiveness.

### Why Subscription IDs?

Subscription IDs allow for precise unsubscription, which is important when components need to be dynamically started and stopped. Rather than requiring users to remember both the event type and processor function, they only need to store the returned subscription ID.

### Why Two Publish Methods?

The split between `Publish` (async) and `PublishSync` (sync) allows the event bus to serve different use cases:
- Async publishing for performance-critical paths where the caller doesn't need to know if processors succeed
- Sync publishing for scenarios where the caller needs to know if processors failed or must complete before continuing.

## Thread Safety Trade-offs

### RWMutex for Subscription Map

We use a read-write mutex because:
- Reads (dispatching events) are much more frequent than writes (subscribing/unsubscribing)
- Multiple readers can access the map concurrently
- Write locks are held for minimal time during subscription changes

### Event Channel Safety

- The Go channel is inherently thread-safe for concurrent send/receive operations
- No additional synchronization is needed for the channel itself
- Workers consume from the same channel without conflicts

### Worker Isolation

Each worker processes events independently, which:
- Prevents one slow processor from blocking others
- Allows for parallel processing of different events
- Reduces contention between workers

## Error Handling Philosophy

### Async vs Sync Error Handling

In asynchronous operation, errors from processors are typically logged but not propagated to the publisher since the publisher has moved on. In synchronous operation, all processor errors are collected and returned to the caller.

### Graceful Degradation

The system is designed to continue operating even when individual processors fail:
- Failed processors don't prevent other processors from running
- Worker goroutines recover from panics to maintain availability
- Event drops during high load are preferred over system overload

This ensures that a single faulty processor or event doesn't bring down the entire event system.

## Next Steps

- [Architecture Deep Dive](./architecture.md) - Detailed look at the internal architecture
- [API Reference](./api.md) - Complete API documentation