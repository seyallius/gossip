# Implementation Overview

This section provides a comprehensive overview of how Gossip is implemented, including the core architecture, concurrency model, and design decisions.

## Core Architecture

The core of Gossip is built around several key components:

1. **EventBus** - Owns the subscription registry, the event channel, and the worker pool
2. **Event** - The data structure that carries information through the system
3. **EventProcessor** - Functions that process events
4. **Middleware** - Pluggable functionality that wraps processors
5. **Filters** - Conditional execution of processors based on event properties
6. **Batch** - An adapter that accumulates events and processes them in groups

Everything is in-process. There is no transport abstraction and no network hop: `Publish` puts the
event on a Go channel and a worker goroutine picks it up. Gossip is a way to structure a single
application around events, not a message broker.

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

// EventBus manages event publishing and subscription.
type EventBus struct {
    subscriptions map[EventType][]*Subscription // one slice of subscribers per event type
    eventChan     chan *Event                   // buffered hand-off to the worker pool
    // ... workers, context, mutex, wait group
}
```

### The EventBus API

```go
func (eb *EventBus) Subscribe(eventType EventType, processor EventProcessor) string
func (eb *EventBus) SubscribeMultiple(eventTypes []EventType, processor EventProcessor) []string
func (eb *EventBus) Unsubscribe(subscriptionId string) bool
func (eb *EventBus) UnsubscribeMultiple(subscriptionIds []string)
func (eb *EventBus) Publish(event *Event) bool
func (eb *EventBus) PublishSync(ctx context.Context, event *Event) []error
func (eb *EventBus) Shutdown() error
```

`Publish` and `Shutdown` both report back. `Publish` returning `false` means the event was dropped --
a full buffer or a bus that is shutting down -- and `Shutdown` returning an error means the backlog
was still draining when `ShutdownTimeout` elapsed. Neither can block, and neither is allowed to
fail silently.

### Dispatching Without Holding the Lock

`dispatch` takes the read lock only long enough to read the slice header for an event type, then
iterates it with the lock released -- so one slow processor cannot block every subscription change on
the bus. That puts the burden on the write side: `Unsubscribe` builds a new slice rather than
shifting elements down inside the array a publish may be walking, while `Subscribe` can append
normally, because an append is invisible to a reader holding the old, shorter length.

#### Configuration

```go
type Config struct {
    Workers         int           // Number of worker goroutines
    BufferSize      int           // Size of the event channel buffer
    ShutdownTimeout time.Duration // How long Shutdown waits for the backlog to drain
}
```

Any field left at zero (or negative) falls back to its default: 10 workers, a buffer of 1000, and a
5 second shutdown timeout. Out-of-range values are replaced rather than honoured, because none of
them describes a usable bus -- zero workers accepts events and never processes them, and a zero
buffer makes the channel unbuffered, which turns `Publish`'s non-blocking send into a drop whenever
no worker happens to be parked on the receive.

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
    if e.Metadata == nil {
        e.Metadata = make(map[string]any, 1)
    }
    e.Metadata[key] = value
    return e
}
```

The `NewEvent` function automatically sets the timestamp when created, providing a consistent way to track when events occurred. The `WithMetadata` method enables method chaining for fluent API usage.

`WithMetadata` creates the map when it is missing. `NewEvent` allocates one, but an `Event` built as a
struct literal -- which is what test fixtures and decoders produce -- carries a nil map, and assigning
into that panics.

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

The buffered channel provides decoupling between publishers and processors. When the system experiences bursts of events, the buffer absorbs the load temporarily, preventing publishers from being blocked. However, if the buffer fills up, events are dropped to maintain system responsiveness -- and `Publish` returns `false` so the caller can decide what that is worth: a retry, a log line, a metric, or a fall back to `PublishSync`.

### Why Subscription IDs?

Subscription IDs allow for precise unsubscription, which is important when components need to be dynamically started and stopped. Rather than requiring users to remember both the event type and processor function, they only need to store the returned subscription ID.

The ID is `"<event type>-<counter>"`, where the counter is per-bus and monotonic. It has to be unique:
`Unsubscribe` removes the first subscription whose ID matches, so two subscriptions sharing an ID
would mean removing one of them silently removes the other.

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
- Failed processors don't prevent other processors from running: `dispatch` collects errors per subscription and carries on
- Event drops during high load are preferred over system overload, and `Publish` reports each one

Panics are the exception. A processor that panics takes down the worker goroutine, and with it the
process -- the bus does not recover on your behalf, because silently swallowing a panic is how a
broken handler stays broken. Wrap processors in `middleware.WithRecovery()`, which converts the
panic into a `ProcessingError` the surrounding middleware can see.

This ensures that a single faulty processor or event doesn't bring down the entire event system.

## Next Steps

- [Architecture Deep Dive](./architecture.md) - Detailed look at the internal architecture
- [API Reference](./api.md) - Complete API documentation