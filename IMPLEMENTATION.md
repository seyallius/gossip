# Gossip Event Bus Implementation Documentation

## Table of Contents
1. [Overview](#overview)
2. [Core Architecture](#core-architecture)
3. [Provider Architecture](#provider-architecture)
4. [Event Types and Data Structures](#event-types-and-data-structures)
5. [Event Bus Implementation](#event-bus-implementation)
6. [Concurrency Model](#concurrency-model)
7. [Thread Safety](#thread-safety)
8. [Middleware System](#middleware-system)
9. [Event Filtering](#event-filtering)
10. [Batch Processing](#batch-processing)
11. [Global Event Bus](#global-event-bus)
12. [Testing Implementation](#testing-implementation)
13. [Performance Considerations](#performance-considerations)
14. [Design Decisions](#design-decisions)
15. [How to Extend This Library](#how-to-extend-this-library)
16. [Common Questions](#common-questions)
17. [Testing Tips](#testing-tips)

## Overview

Gossip is a lightweight, type-safe event bus library for Go that implements the observer/pub-sub pattern. It enables clean separation between core business logic and side effects, making your codebase more maintainable and extensible.

The implementation provides:
- Strongly-typed events with `EventType` constants
- Async by default with worker pools
- Pluggable transport providers (in-memory, Redis)
- Event filtering capabilities
- Batch processing features
- Middleware support
- Priority queues
- Thread-safe operations
- Graceful shutdown

## Why Does This Even Exist?

You know when you write code like this:

```go
func CreateUser(email, username string) error {
    // Save user to DB
    user := saveToDatabase(email, username)
    
    // Now we need to send email
    sendWelcomeEmail(user.Email)
    
    // And log it
    auditLog.Record("user_created", user.ID)
    
    // And update metrics
    metrics.Increment("users_created")
    
    // And notify admin
    notifyAdmin(user)
    
    return nil
}
```

This is **TIGHTLY COUPLED** garbage. Every time you want to add something new (like sending SMS, or posting to Slack), you have to **MODIFY** this function. That violates the **Open/Closed Principle** - code should be open for extension but closed for modification.

### The Event-Driven Solution

```go
func CreateUser(email, username string) error {
    // Save user to DB
    user := saveToDatabase(email, username)
    
    // Just publish an event - that's it!
    bus.Publish(NewEvent(UserCreated, user))
    
    return nil
}
```

Now `CreateUser` doesn't give a damn about emails, logs, metrics, notifications. It just says "Hey, a user was created" and **OTHER PARTS** of the code react to it. Want to add SMS? Just subscribe a new processor. No touching the original code.

## Provider Architecture

Gossip implements a flexible provider-based architecture that allows pluggable transport mechanisms. This design enables the same API to work with different underlying technologies like in-memory channels or distributed systems like Redis.

### Provider Interface

The `Provider` interface defines the contract that all transport implementations must satisfy:

```go
type Provider interface {
    Publish(event *Event) error
    Subscribe(eventType EventType, processor EventProcessor) (subscriptionId string, err error)
    Unsubscribe(subscriptionID string) error
    Shutdown() error
}
```

### Available Providers

1. **InMemoryProvider** - Uses Go channels for local, in-process event communication
2. **RedisProvider** - Uses Redis Pub/Sub for distributed event communication across multiple services

### Configuration

The new architecture introduces a driver-based configuration system:

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
```

Let me explain **EVERY. SINGLE. FIELD.**

##### `mu sync.RWMutex`

This is a **Read-Write Mutex**. Why do we need it?

Because multiple goroutines will:
- **Subscribe** processors (WRITE operation - modifying the map)
- **Publish** events (READ operation - reading the map to find processors)

Without a mutex, you get **race conditions** - two goroutines modifying the map at the same time = CRASH.

**RWMutex** is smart:
- Multiple readers can read at the same time (fast)
- Only one writer can write, and blocks all readers (safe)

```go
eb.mu.RLock()  // ← Lock for READING (multiple allowed)
processors := eb.subscriptions[eventType]
eb.mu.RUnlock()

eb.mu.Lock()   // ← Lock for WRITING (exclusive)
eb.subscriptions[eventType] = append(...)
eb.mu.Unlock()
```

##### `subscriptions map[EventType][]*Subscription`

This is the **registry**. It maps:
- **Key**: EventType (e.g., `UserCreated`)
- **Value**: Slice of processors that care about this event

Example:
```
UserCreated → [emailprocessor, auditprocessor, metricsprocessor]
OrderPaid   → [paymentprocessor, invoiceprocessor]
```

When you publish `UserCreated`, the bus looks up this map and runs all three processors.

Why a slice? Because **multiple processors** can subscribe to the same event.

##### `workers int`

Number of **worker goroutines**. Think of them as a thread pool.

When you publish an event, it goes into a channel. Workers are constantly pulling from that channel and processing events.

Why multiple workers? **Parallelism**. If you have 10 workers and 10 events arrive, all 10 can be processed simultaneously (if processors are independent).

##### `eventChan chan *Event`

This is a **buffered channel** - a queue for events.

```go
eventChan: make(chan *Event, cfg.BufferSize)
```

When you call `Publish()`, the event goes into this channel. Workers pull from it.

**Why buffered?** So `Publish()` doesn't block. If the channel is unbuffered and all workers are busy, `Publish()` would wait. With a buffer, it can dump 1000 events (or whatever `BufferSize` is) instantly, and workers process them when ready.

##### `ctx context.Context` and `cancel context.CancelFunc`

These are for **graceful shutdown**.

When you call `Shutdown()`, it calls `cancel()`. This sends a signal to all workers: "Stop accepting new work, finish what you're doing, and exit."

```go
ctx, cancel := context.WithCancel(context.Background())
```

Workers check:
```go
case <-eb.ctx.Done():
    return  // Exit
```

##### `wg sync.WaitGroup`

A **WaitGroup** is like a counter.

- When a worker starts: `wg.Add(1)`
- When a worker finishes: `wg.Done()`
- When shutting down: `wg.Wait()` blocks until all workers finish

This ensures **graceful shutdown** - we don't just kill workers mid-processing.

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

Let me explain each field:

- **Type**: What kind of event is this? (`UserCreated`, `OrderPaid`, etc.)
- **Timestamp**: When did this happen? Auto-set to `time.Now()` when you create the event
- **Data**: The actual payload. Could be `*UserData`, `*OrderData`, whatever. We use `any` because Go doesn't have generics in older versions (well, it does now, but `interface{}` is simpler for this use case)
- **Metadata**: Extra context like `request_id`, `user_agent`, `source`. Think of it like HTTP headers - not the main content, but useful info.

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

**Why a constructor?** So you don't forget to initialize `Metadata` or set `Timestamp`. Constructors enforce consistency.

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

### Predefined Event Types

The library includes example event types across different domains:

```go
//goland:noinspection GoCommentStart
const (
    // Authentication Events
    AuthEventUserCreated     EventType = "auth.user.created"
    AuthEventLoginSuccess    EventType = "auth.login.success"
    AuthEventLoginFailed     EventType = "auth.login.failed"
    // ... other event types
)
```

These serve as examples and can be extended based on application needs.

## Event Bus Implementation

### Configuration

```go
// Config holds configuration for the event bus.
type Config struct {
    Workers    int // Number of worker goroutines
    BufferSize int // Size of the event channel buffer
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
    return &Config{
        Workers:    10,
        BufferSize: 1000,
    }
}
```

The configuration allows users to tune the event bus performance based on their workload characteristics. More workers can process events in parallel, while a larger buffer can handle bursty event loads.

### Eventprocessor - What's a processor?

```go
type Eventprocessor func(ctx context.Context, event *Event) error
```

A processor is just a function that:
1. Takes a `context` (for cancellation/timeout)
2. Takes an `Event`
3. Returns an `error` (if something goes wrong)

Example:
```go
func emailprocessor(ctx context.Context, event *Event) error {
    data := event.Data.(*UserData)
    return sendEmail(data.Email, "Welcome!")
}
```

### EventBus Creation

```go
// NewEventBus creates a new event bus with the given configuration.
func NewEventBus(cfg *Config) *EventBus {
    if cfg == nil {
        cfg = DefaultConfig()
    }

    ctx, cancel := context.WithCancel(context.Background())

    bus := &EventBus{
        subscriptions: make(map[EventType][]*Subscription),
        workers:       cfg.Workers,
        eventChan:     make(chan *Event, cfg.BufferSize),
        ctx:           ctx,
        cancel:        cancel,
    }

    bus.start()
    return bus
}
```

Note that the event channel is buffered with `cfg.BufferSize`. This allows for non-blocking publish operations up to the buffer capacity. The `context.WithCancel` creates a cancellation context that will be used during shutdown.

**Why `context.WithCancel`?** So we can signal all workers to stop when `Shutdown()` is called.

### Subscription Management

```go
// Subscribe registers a processor for a specific event type and returns a subscription ID.
func (eb *EventBus) Subscribe(eventType EventType, processor Eventprocessor) string {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    subscriptionID := fmt.Sprintf("%s-%d", eventType, len(eb.subscriptions[eventType]))

    sub := &Subscription{
        ID:      subscriptionID,
        Type:    eventType,
        processor: processor,
    }

    eb.subscriptions[eventType] = append(eb.subscriptions[eventType], sub)

    return subscriptionID
}
```

The subscription ID is generated using the event type and the current count of processors for that type. This provides a unique but predictable ID for each subscription. The mutex lock ensures thread safety when modifying the subscription map.

**Step by step:**
1. **Lock** the subscriptions map (exclusive write access)
2. Generate a unique ID (like `user.created-0`, `user.created-1`)
3. Create a `Subscription` wrapper (ID + processor)
4. Append to the slice for this event type
5. Return the ID (so you can unsubscribe later)

**Why defer unlock?** In case something panics, the lock is always released. Otherwise, you'd deadlock the entire bus.

```go
// Unsubscribe removes a subscription by ID.
func (eb *EventBus) Unsubscribe(subscriptionID string) bool {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    for eventType, subs := range eb.subscriptions {
        for i, sub := range subs {
            if sub.ID == subscriptionID {
                // Unsubscribed processor
                eb.subscriptions[eventType] = append(subs[:i], subs[i+1:]...)
                return true
            }
        }
    }

    return false
}
```

The unsubscribe operation iterates through all subscription lists to find and remove the matching subscription. This is O(n) complexity but is acceptable since unsubscription typically happens infrequently.

### Event Publishing

#### Asynchronous Publishing

```go
// Publish sends an event to all registered processors asynchronously.
func (eb *EventBus) Publish(event *Event) {
    select {
    case eb.eventChan <- event:

    case <-eb.ctx.Done():

    default:
        // Event channel full, dropping event
    }
}
```

The publish operation uses a non-blocking select statement that has three cases:
1. Successful send to the event channel (most common case)
2. Context cancellation (the event bus is shutting down)
3. Default case where the channel buffer is full (event gets dropped)

This approach prevents blocking the caller when the event channel is full, which is important for maintaining system responsiveness.

**The `select` statement:**
- **First case**: Try to push event into the channel. If there's space, it succeeds immediately (non-blocking).
- **Second case**: If the bus is shutting down, don't accept new events.
- **Default case**: If the channel is full (buffer exhausted), drop the event with a warning.

**Why drop instead of block?** Because blocking the publisher could freeze your entire app. It's better to lose an event than hang the system. (You can adjust `BufferSize` to prevent this.)

#### Synchronous Publishing

```go
// PublishSync sends an event to all registered processors synchronously.
func (eb *EventBus) PublishSync(ctx context.Context, event *Event) []error {
    eb.mu.RLock()
    processors := eb.subscriptions[event.Type]
    eb.mu.RUnlock()

    if len(processors) == 0 {
        return nil
    }

    errors := make([]error, 0)

    for _, sub := range processors {
        if err := sub.processor(ctx, event); err != nil {
            errors = append(errors, fmt.Errorf("processor %s: %w", sub.ID, err))
        }
    }

    return errors
}
```

Synchronous publishing acquires a read lock to get the processors, then executes each processor in sequence. This is useful when immediate processing is required and the caller needs to know if any processors failed.

**Difference from `Publish()`:**
- **Synchronous**: Runs processors **right now** in the current goroutine.
- **Returns errors**: You get feedback immediately.

**When to use?**
- When you need to know if processors succeeded (like during a transaction).
- When order matters.
- In tests (to avoid waiting for async processing).

### Event Processing and Dispatch

#### Worker Goroutines

```go
// start initializes worker goroutines to process events.
func (eb *EventBus) start() {
    for i := 0; i < eb.workers; i++ {
        eb.wg.Add(1)
        go eb.worker()
    }
}
```

The start method creates the configured number of worker goroutines. Each worker processes events from the shared channel, enabling parallel processing.

```go
// worker processes events from the channel.
func (eb *EventBus) worker() {
    defer eb.wg.Done()

    for {
        select {
        case event, ok := <-eb.eventChan:
            if !ok {
                return // Worker stopped
            }
            eb.dispatch(event)

        case <-eb.ctx.Done():
            return // Worker shutting down
        }
    }
}
```

Each worker runs an infinite loop that selects from two channels:
1. The event channel - when an event arrives, it gets dispatched
2. The cancellation context - when cancelled, the worker exits

The `ok` check is important to detect when the event channel is closed during shutdown.

**The infinite loop:**
- **Wait** for an event from the channel
- **Or** wait for a shutdown signal
- If event arrives, dispatch it to processors
- If shutdown signal, exit gracefully

**Why `ok` check?** When a channel is closed, receivers get `ok == false`. This tells the worker to stop.

#### dispatch() - Running processors

```go
// dispatch sends an event to all registered processors.
func (eb *EventBus) dispatch(event *Event) {
    eb.mu.RLock()
    processors := eb.subscriptions[event.Type]
    eb.mu.RUnlock()

    if len(processors) == 0 {
        return
    }

    for _, sub := range processors {
        _ = sub.processor(eb.ctx, event)
    }
}
```

The dispatch method gets a read lock to access the processors (avoiding blocking while processing), then calls each processor. Errors are ignored in asynchronous dispatch (though they could be logged in a production system).

**Step by step:**
1. Look up processors for this event type
2. Run each processor
3. If a processor fails, log the error but **don't stop** other processors

**Why not stop on error?** Because processors are independent. If the email processor fails, the audit processor should still run.



### Graceful Shutdown

```go
// Shutdown gracefully stops the event bus and waits for all workers to finish.
func (eb *EventBus) Shutdown() {
    eb.cancel()
    close(eb.eventChan)
    eb.wg.Wait()
}
```

The shutdown process:
1. Cancels the context, which signals all workers to stop
2. Closes the event channel, which allows workers to detect shutdown and return
3. Waits for all workers to finish using the wait group

This ensures all in-flight events are processed before the event bus exits.

```go
func (eb *EventBus) Shutdown() {
    log.Println("Shutting down...")
    
    eb.cancel()           // ← Signal workers to stop
    close(eb.eventChan)   // ← Close the channel
    eb.wg.Wait()          // ← Wait for all workers to finish
    
    log.Println("Shutdown complete")
}
```

**Order matters:**
1. **Cancel context**: Workers stop pulling new events
2. **Close channel**: Workers process remaining events in the buffer, then exit
3. **Wait**: Block until all workers finish

This ensures **no event is lost** and **no goroutine leak**.

## Concurrency Model

Gossip uses a producer-consumer model with the following concurrency patterns:

1. **Publishers** - Any goroutine can call `Publish()`, which sends to a buffered channel
2. **Worker Pool** - Multiple goroutines consume from the shared event channel
3. **Subscription Management** - Protected by mutexes to ensure thread safety

The design separates the publishing path (which needs to be fast and non-blocking) from the processing path (which can be slower but needs to handle all events).

## Thread Safety

Thread safety is achieved through:

1. **RWMutex** for subscription map access:
   - `Subscribe()` and `Unsubscribe()` use write locks
   - `dispatch()` and `PublishSync()` use read locks
   - This allows concurrent reads while preventing concurrent writes/reads during modifications

2. **Channel-based Communication**:
   - The event channel is safe for concurrent use
   - Workers consume from the same channel without additional synchronization

3. **Immutable Event Objects**:
   - Once created, event objects are not modified during processing
   - This prevents race conditions on event data

## Middleware System

The middleware system provides a way to add cross-cutting concerns to event processors:

```go
// Middleware wraps an Eventprocessor with additional behavior.
type Middleware func(Eventprocessor) Eventprocessor
```

This functional approach allows for composable behavior around processors.

#### The Pattern

A middleware takes a processor and returns a **new processor** that does something extra.

### Retry Middleware

```go
// WithRetry retries failed processors with exponential backoff.
func WithRetry(maxRetries int, initialDelay time.Duration) Middleware {
    return func(next Eventprocessor) Eventprocessor {
        return func(ctx context.Context, event *Event) error {
            var err error
            delay := initialDelay

            for attempt := 0; attempt <= maxRetries; attempt++ {
                err = next(ctx, event)
                if err == nil {
                    return nil
                }

                if attempt < maxRetries {
                    log.Printf("[Middleware] Retry attempt %d/%d for event %s after error: %v", attempt+1, maxRetries, event.Type, err)

                    select {
                    case <-time.After(delay):
                        delay *= 2

                    case <-ctx.Done():
                        return ctx.Err()
                    }
                }
            }

            return err
        }
    }
}
```

The retry middleware implements exponential backoff with a maximum number of retries. It respects the context for cancellation, which is important for preventing infinite loops during shutdown.

**How it works:**
1. Wraps the original processor (`next`)
2. Tries to run it
3. If it fails, waits and retries
4. Each retry waits **longer** (exponential backoff: 100ms → 200ms → 400ms)

**Usage:**
```go
processor := WithRetry(3, 100*time.Millisecond)(myprocessor)
bus.Subscribe(UserCreated, processor)
```

### Timeout Middleware

```go
// WithTimeout adds a timeout to processor execution.
func WithTimeout(timeout time.Duration) Middleware {
    return func(next Eventprocessor) Eventprocessor {
        return func(ctx context.Context, event *Event) error {
            ctx, cancel := context.WithTimeout(ctx, timeout)
            defer cancel()

            done := make(chan error, 1)
            go func() {
                done <- next(ctx, event)
            }()

            select {
            case err := <-done:
                return err

            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
}
```

The timeout middleware creates a new context with a timeout and runs the processor in a goroutine. This prevents long-running processors from blocking the worker thread.

**Why a goroutine and channel?**
- Run the processor in a separate goroutine
- Wait for either the processor to finish OR the timeout
- If timeout hits first, return `context.DeadlineExceeded`

### Recovery Middleware

```go
// WithRecovery recovers from panics in processors.
func WithRecovery() Middleware {
    return func(next Eventprocessor) Eventprocessor {
        return func(ctx context.Context, event *Event) (err error) {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("[Middleware] Recovered from panic in processor for event %s: %v", event.Type, r)
                    err = nil
                }
            }()

            return next(ctx, event)
        }
    }
}
```

Recovery middleware uses defer and recover to catch panics in processors, preventing them from crashing the worker goroutine. This is essential for robust event processing.

### Logging Middleware

```go
// WithLogging logs processor execution.
func WithLogging() Middleware {
    return func(next Eventprocessor) Eventprocessor {
        return func(ctx context.Context, event *Event) error {
            start := time.Now()
            err := next(ctx, event)
            duration := time.Since(start)

            if err != nil {
                log.Printf("[Middleware] processor for %s failed after %v: %v", event.Type, duration, err)
            } else {
                log.Printf("[Middleware] processor for %s completed in %v", event.Type, duration)
            }

            return err
        }
    }
}
```

Logging middleware provides execution time and error information for debugging and monitoring.

### Middleware Chaining

```go
// Chain chains multiple middlewares together.
func Chain(middlewares ...Middleware) Middleware {
    return func(processor Eventprocessor) Eventprocessor {
        for i := len(middlewares) - 1; i >= 0; i-- {
            processor = middlewares[i](processor)
        }
        return processor
    }
}
```

The chain function applies middlewares in reverse order (last to first), which means the first middleware in the list will be executed first when a processor is called. This creates a chain where each middleware wraps the next one.

**Why reverse order?** So the first middleware in the list is the **outermost** wrapper.

```go
Chain(WithRecovery(), WithRetry(3, 100ms), WithLogging())(processor)
```

Becomes:
```
WithRecovery(
    WithRetry(
        WithLogging(
            processor
        )
    )
)
```

So recovery catches panics from retry, retry handles failures from logging, and logging wraps the original processor.

## Event Filtering

Event filtering allows processors to conditionally execute based on event properties:

```go
// Filter determines if an event should be processed by a processor.
type Filter func(*Event) bool

// Filteredprocessor wraps a processor with a filter condition.
type Filteredprocessor struct {
    filter  Filter
    processor Eventprocessor
}
```

#### Filter - What Is It?

A filter is a function that returns `true` if the event should be processed, `false` otherwise.

### Basic Filter Implementation

```go
// NewFilteredprocessor creates a processor that only executes when the filter returns true.
func NewFilteredprocessor(filter Filter, processor Eventprocessor) Eventprocessor {
    return func(ctx context.Context, event *Event) error {
        if filter(event) {
            return processor(ctx, event)
        }
        return nil
    }
}
```

The filtered processor only calls the wrapped processor if the filter returns true, otherwise it returns nil (no error).

**Usage:**
```go
highPriorityFilter := func(event *Event) bool {
    return event.Metadata["priority"] == "high"
}

bus.Subscribe(OrderCreated, NewFilteredprocessor(highPriorityFilter, urgentprocessor))
```

Now `urgentprocessor` only runs for high-priority orders.

### Metadata-Based Filters

```go
// FilterByMetadata creates a filter that checks for specific metadata key-value pairs.
func FilterByMetadata(key string, value interface{}) Filter {
    return func(event *Event) bool {
        if v, exists := event.Metadata[key]; exists {
            return v == value
        }
        return false
    }
}

// FilterByMetadataExists creates a filter that checks if metadata key exists.
func FilterByMetadataExists(key string) Filter {
    return func(event *Event) bool {
        _, exists := event.Metadata[key]
        return exists
    }
}
```

These utility functions create common types of filters based on event metadata, which is useful for conditional processing.

### Logical Filter Combinations

```go
// And combines multiple filters with AND logic.
func And(filters ...Filter) Filter {
    return func(event *Event) bool {
        for _, f := range filters {
            if !f(event) {
                return false
            }
        }
        return true
    }
}

// Or combines multiple filters with OR logic.
func Or(filters ...Filter) Filter {
    return func(event *Event) bool {
        for _, f := range filters {
            if f(event) {
                return true
            }
        }
        return false
    }
}

// Not negates a filter.
func Not(filter Filter) Filter {
    return func(event *Event) bool {
        return !filter(event)
    }
}
```

These combinators allow complex filtering logic by combining simple filters with logical operators.

**Usage:**
```go
complexFilter := And(
    FilterByMetadata("source", "api"),
    FilterByMetadata("priority", "high"),
)
```

Only runs if BOTH conditions are true.

## Batch Processing

Batch processing allows multiple events to be processed together for efficiency:

```go
// Batchprocessor processes multiple events at once.
type Batchprocessor func(ctx context.Context, events []*Event) error

// BatchProcessor collects events and processes them in batches.
type BatchProcessor struct {
    mu          sync.Mutex
    eventType   EventType
    batchSize   int
    flushPeriod time.Duration
    processor     Batchprocessor
    buffer      []*Event
    ctx         context.Context
    cancel      context.CancelFunc
    wg          sync.WaitGroup
}

// BatchConfig holds configuration for batch processing.
type BatchConfig struct {
    BatchSize   int           // Max events per batch
    FlushPeriod time.Duration // Max time to wait before flushing
}
```

#### Why Batch?

If you're sending 1000 emails, it's inefficient to make 1000 API calls. Better to batch them:
- Make 10 calls with 100 emails each
- Or wait 5 seconds and send all at once

### Batch Processor Implementation

```go
// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(eventType EventType, config BatchConfig, processor Batchprocessor) *BatchProcessor {
    ctx, cancel := context.WithCancel(context.Background())

    bp := &BatchProcessor{
        eventType:   eventType,
        batchSize:   config.BatchSize,
        flushPeriod: config.FlushPeriod,
        processor:     processor,
        buffer:      make([]*Event, 0, config.BatchSize),
        ctx:         ctx,
        cancel:      cancel,
    }

    bp.start()
    return bp
}
```

The batch processor creates a cancellation context and initializes its buffer. It also starts the periodic flush goroutine during initialization.

#### BatchProcessor - The Structure

```go
type BatchProcessor struct {
    buffer      []*Event
    batchSize   int
    flushPeriod time.Duration
    processor     Batchprocessor
    // ... synchronization stuff
}
```

**How it works:**
1. Events get added to `buffer`
2. When `buffer` reaches `batchSize`, flush it (call processor with all events)
3. Or, every `flushPeriod` seconds, flush whatever's in the buffer

### Event Addition and Flushing

```go
// Add adds an event to the batch buffer.
func (bp *BatchProcessor) Add(event *Event) {
    bp.mu.Lock()
    defer bp.mu.Unlock()

    bp.buffer = append(bp.buffer, event)

    if len(bp.buffer) >= bp.batchSize {
        bp.flush()
    }
}

// Flush processes all buffered events immediately.
func (bp *BatchProcessor) Flush() {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    bp.flush()
}
```

Events are added to the buffer under lock protection. When the buffer reaches capacity, it's automatically flushed to process the batch.

```go
// flush processes the current buffer (must be called with lock held).
func (bp *BatchProcessor) flush() {
    if len(bp.buffer) == 0 {
        return
    }

    events := make([]*Event, len(bp.buffer))
    copy(events, bp.buffer)
    bp.buffer = bp.buffer[:0]

    go func() {
        if err := bp.processor(bp.ctx, events); err != nil {
            // Log error but don't block
        }
    }()
}
```

The flush operation copies the buffer to avoid holding the lock while processing, then processes the batch in a separate goroutine to prevent blocking the caller.

#### Add() - Adding to Buffer

```go
func (bp *BatchProcessor) Add(event *Event) {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    
    bp.buffer = append(bp.buffer, event)
    
    if len(bp.buffer) >= bp.batchSize {
        bp.flush()  // ← Time to process!
    }
}
```

### Periodic Flushing

```go
// start begins the periodic flush goroutine.
func (bp *BatchProcessor) start() {
    bp.wg.Add(1)
    go bp.periodicFlush()
}

// periodicFlush flushes the buffer at regular intervals.
func (bp *BatchProcessor) periodicFlush() {
    defer bp.wg.Done()

    ticker := time.NewTicker(bp.flushPeriod)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            bp.Flush()

        case <-bp.ctx.Done():
            return
        }
    }
}
```

The periodic flush goroutine ensures that events don't stay in the buffer indefinitely. It flushes events either when the batch size is reached or when the flush period expires.

#### periodicFlush() - Time-Based Flushing

```go
func (bp *BatchProcessor) periodicFlush() {
    ticker := time.NewTicker(bp.flushPeriod)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            bp.Flush()  // ← Flush every N seconds
        case <-bp.ctx.Done():
            return
        }
    }
}
```

This runs in a goroutine, flushing the buffer periodically.

### Event processor Wrapper

```go
// AsEventprocessor returns an Eventprocessor that adds events to the batch.
func (bp *BatchProcessor) AsEventprocessor() Eventprocessor {
    return func(ctx context.Context, event *Event) error {
        bp.Add(event)
        return nil
    }
}
```

This method allows the batch processor to be used as a regular event processor by simply adding events to the batch.

### Batch Processor Shutdown

```go
// Shutdown stops the batch processor and flushes remaining events.
func (bp *BatchProcessor) Shutdown() {
    bp.cancel()
    bp.wg.Wait()
    bp.Flush()
}
```

The shutdown process cancels the context, waits for the periodic goroutine to finish, then flushes any remaining events in the buffer.

## Global Event Bus

Gossip provides a global singleton event bus for application-wide access:

```go
var (
    globalBus *EventBus
    once      sync.Once
)

// GetGlobalBus returns the singleton event bus instance.
func GetGlobalBus() *EventBus {
    once.Do(func() {
        globalBus = NewEventBus(DefaultConfig())
    })
    return globalBus
}
```

The global bus uses `sync.Once` to ensure it's initialized only once, even in concurrent scenarios.

### Global Convenience Functions

```go
// Publish is a convenience function to publish events to the global bus.
func Publish(event *Event) {
    GetGlobalBus().Publish(event)
}

// Subscribe is a convenience function to subscribe to the global bus.
func Subscribe(eventType EventType, processor Eventprocessor) string {
    return GetGlobalBus().Subscribe(eventType, processor)
}
```

These functions provide a simple API for applications that want to use a global event bus without managing the instance themselves.

## Testing Implementation

The implementation includes comprehensive tests covering various scenarios:

### Basic Publish/Subscribe Test

```go
func TestEventBus_PublishSubscribe(t *testing.T) {
    bus := event.NewEventBus(event.DefaultConfig())
    defer bus.Shutdown()

    received := int32(0)
    processor := func(ctx context.Context, event *event.Event) error {
        atomic.AddInt32(&received, 1)
        return nil
    }

    bus.Subscribe(event.AuthEventLoginSuccess, processor)

    event := event.NewEvent(event.AuthEventLoginSuccess, &event.LoginSuccessData{
        UserID:   "test-user",
        Username: "testuser",
    })

    bus.Publish(event)

    time.Sleep(100 * time.Millisecond)

    if atomic.LoadInt32(&received) != 1 {
        t.Errorf("Expected 1 event, got %d", received)
    }
}
```

This test verifies the basic functionality of subscribing to an event and receiving it once it's published.

### Multiple Subscribers Test

```go
func TestEventBus_MultipleSubscribers(t *testing.T) {
    bus := event.NewEventBus(event.DefaultConfig())
    defer bus.Shutdown()

    counter := int32(0)

    for i := 0; i < 5; i++ {
        bus.Subscribe(event.AuthEventUserCreated, func(ctx context.Context, event *event.Event) error {
            atomic.AddInt32(&counter, 1)
            return nil
        })
    }

    event := event.NewEvent(event.AuthEventUserCreated, &event.UserCreatedData{
        UserID:   "test-user",
        Username: "testuser",
    })

    bus.Publish(event)

    time.Sleep(100 * time.Millisecond)

    if atomic.LoadInt32(&counter) != 5 {
        t.Errorf("Expected 5 invocations, got %d", counter)
    }
}
```

This test ensures that when multiple processors are subscribed to the same event type, all of them receive the event.

### Concurrent Publish Test

```go
func TestEventBus_ConcurrentPublish(t *testing.T) {
    bus := event.NewEventBus(event.DefaultConfig())
    defer bus.Shutdown()

    counter := int32(0)
    processor := func(ctx context.Context, event *event.Event) error {
        atomic.AddInt32(&counter, 1)
        return nil
    }

    bus.Subscribe(event.AuthEventLoginSuccess, processor)

    var wg sync.WaitGroup
    numEvents := 100

    for i := 0; i < numEvents; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            event := event.NewEvent(event.AuthEventLoginSuccess, &event.LoginSuccessData{})
            bus.Publish(event)
        }()
    }

    wg.Wait()
    time.Sleep(200 * time.Millisecond)

    if atomic.LoadInt32(&counter) != int32(numEvents) {
        t.Errorf("Expected %d events, got %d", numEvents, counter)
    }
}
```

This test validates thread safety by publishing events concurrently from multiple goroutines and ensuring all events are processed.

## Performance Considerations

### Channel Buffering

The event channel is buffered to prevent blocking publishers. The buffer size should be tuned based on:
- Expected event burst rates
- Processing time per event
- Acceptable memory usage

### Worker Pool Size

The number of worker goroutines should match the workload characteristics:
- CPU-bound processors: Number of CPU cores
- I/O-bound processors: Higher number to allow other processors to run while one waits
- Mixed workloads: Experiment to find optimal balance

### Memory Management

- Events are processed in batches when possible to reduce allocation overhead
- Buffer sizes are pre-allocated where possible
- Event objects are immutable after creation to avoid copying overhead

### Concurrency Patterns

- Read-write mutexes allow concurrent reads during dispatch
- Worker goroutines consume from a shared channel for load balancing
- Batch processing reduces per-event overhead

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
- Sync publishing for scenarios where the caller needs to know if processors failed or must complete before continuing

### Why Middleware Instead of Hooks?

Middleware provides a more composable and reusable approach to cross-cutting concerns compared to hooks. Middlewares can be combined and reused across different processors, while hooks tend to be more rigid and harder to compose.

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

## Design Decisions - Why Did I Choose X Over Y?

### 1. Why `interface{}` for event data instead of generics?

**Answer:** Simplicity and Go version compatibility.

Generics in Go are still new (Go 1.18+). Using `interface{}` means:
- Works on older Go versions
- Simpler implementation
- Downside: No compile-time type safety (you have to type-assert)

If I used generics:
```go
type Event[T any] struct {
    Data T
}
```

But then `EventBus` would need to be generic too, and it gets messy when you have multiple event types in the same bus.

---

### 2. Why async by default?

**Answer:** Performance and decoupling.

If `Publish()` was synchronous, it would **block** until all processors finish. Imagine:

```go
func CreateUser() {
    user := saveUser()
    bus.Publish(UserCreated, user)  // ← Blocks for 5 seconds while sending email
}
```

Your API endpoint takes 5 seconds to respond! With async, `Publish()` returns instantly, and processors run in the background.

---

### 3. Why a channel instead of just calling processors directly?

**Answer:** Decoupling and buffering.

If you call processors directly in `Publish()`, the publisher waits for them. With a channel:
- Publisher dumps events and moves on
- Workers pull and process at their own pace
- Buffering handles traffic spikes

---

### 4. Why `RWMutex` instead of regular `Mutex`?

**Answer:** Performance.

- **Publishing** (reading subscriptions) happens WAY more than **subscribing** (writing subscriptions)
- `RWMutex` allows multiple readers simultaneously
- Regular `Mutex` would serialize all reads (slower)

---

### 5. Why log errors instead of panicking?

**Answer:** Resilience.

If one processor fails, others should still run. Panicking would crash the entire bus. Logging lets you monitor failures without bringing down the system.

---

## How to Extend This Library

### Adding a New Middleware

Want rate limiting? Create a middleware:

```go
func WithRateLimit(limit int, period time.Duration) Middleware {
    limiter := rate.NewLimiter(rate.Every(period), limit)
    
    return func(next Eventprocessor) Eventprocessor {
        return func(ctx context.Context, event *Event) error {
            if !limiter.Allow() {
                return fmt.Errorf("rate limit exceeded")
            }
            return next(ctx, event)
        }
    }
}
```

---

### Adding Event Persistence

Want to save events to a database before processing?

```go
func (eb *EventBus) PublishWithPersistence(event *Event) error {
    // Save to DB first
    if err := db.SaveEvent(event); err != nil {
        return err
    }
    
    // Then publish
    eb.Publish(event)
    return nil
}
```

---

### Adding Event Replay

Want to replay events from a log?

```go
func (eb *EventBus) Replay(events []*Event) {
    for _, event := range events {
        eb.Publish(event)
    }
}
```

---

## Common Questions

**Q: Can processors publish new events?**  
**A:** Yes! But be careful of infinite loops. Use metadata to track depth:

```go
func processor(ctx context.Context, event *Event) error {
    depth, _ := event.Metadata["depth"].(int)
    if depth > 5 {
        return nil  // Stop recursion
    }
    
    newEvent := NewEvent(AnotherEvent, data).
        WithMetadata("depth", depth+1)
    
    bus.Publish(newEvent)
    return nil
}
```

---

**Q: What if a processor is slow?**  
**A:** Use `WithTimeout()` middleware to prevent it from hanging.

---

**Q: What if I want guaranteed delivery?**  
**A:** Use `PublishSync()` and check errors. Or implement persistence (save to DB, then process).

---

**Q: Can I have multiple event buses?**  
**A:** Yes! Just create multiple instances. Useful for testing or isolating domains.

---

## Testing Tips

**Test processors independently:**
```go
func Testprocessor(t *testing.T) {
    event := NewEvent(UserCreated, &UserData{UserID: "123"})
    err := myprocessor(context.Background(), event)
    assert.NoError(t, err)
}
```

**Test the bus with sync publishing:**
```go
func TestBus(t *testing.T) {
    bus := NewEventBus(DefaultConfig())
    defer bus.Shutdown()
    
    called := false
    bus.Subscribe(UserCreated, func(ctx context.Context, e *Event) error {
        called = true
        return nil
    })
    
    bus.PublishSync(context.Background(), NewEvent(UserCreated, nil))
    assert.True(t, called)
}
```
