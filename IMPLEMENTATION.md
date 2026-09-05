# Gossip Event Bus Implementation Documentation

## Table of Contents
1. [Overview](#overview)
2. [Core Architecture](#core-architecture)
3. [Event Types and Data Structures](#event-types-and-data-structures)
4. [Event Bus Implementation](#event-bus-implementation)
5. [Concurrency Model](#concurrency-model)
6. [Thread Safety](#thread-safety)
7. [Middleware System](#middleware-system)
8. [Event Filtering](#event-filtering)
9. [Batch Processing](#batch-processing)
10. [Global Event Bus](#global-event-bus)
11. [Testing Implementation](#testing-implementation)
12. [Performance Considerations](#performance-considerations)
13. [Design Decisions](#design-decisions)
14. [How to Extend This Library](#how-to-extend-this-library)
15. [Common Questions](#common-questions)
16. [Testing Tips](#testing-tips)

## Overview

Gossip is a lightweight, type-safe event bus library for Go that implements the observer/pub-sub pattern. It enables clean separation between core business logic and side effects, making your codebase more maintainable and extensible.

The implementation provides:
- Strongly-typed events with `EventType` constants
- Async by default with worker pools
- Synchronous publishing option
- Event filtering capabilities
- Batch processing features
- Middleware support
- Thread-safe operations
- Graceful shutdown that drains the backlog

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

## Core Architecture

The core of Gossip is built around several key components:

1. **EventBus** - The main coordinator that manages subscriptions and event dispatch
2. **Event** - The data structure that carries information through the system
3. **EventProcessor** - Functions that process events
4. **Worker Pool** - Goroutines that handle asynchronous event processing
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

// EventBus manages event publishing and subscription.
type EventBus struct {
    subscriptions   map[EventType][]*Subscription
    eventChan       chan *Event
    workers         int
    shutdownTimeout time.Duration
    subscriptionSeq atomic.Uint64 // monotonic counter behind every subscription ID
    ctx             context.Context
    cancel          context.CancelFunc
    mu              sync.RWMutex
    closed          bool      // guarded by mu; true once Shutdown closed eventChan
    shutdownOnce    sync.Once // makes Shutdown idempotent
    shutdownErr     error
    wg              sync.WaitGroup
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
    Workers         int           // Number of worker goroutines
    BufferSize      int           // Size of the event channel buffer
    ShutdownTimeout time.Duration // How long Shutdown waits for buffered events to drain
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
    return &Config{
        Workers:         10,
        BufferSize:      1000,
        ShutdownTimeout: 5 * time.Second,
    }
}
```

The configuration allows users to tune the event bus performance based on their workload characteristics. More workers can process events in parallel, while a larger buffer can handle bursty event loads.

`NewEventBus` replaces any field left at zero or below with its default, rather than honouring it.
None of those values describes a working bus: zero workers accepts events and never processes them,
and a zero buffer makes the channel unbuffered, which turns `Publish`'s non-blocking send into a drop
whenever no worker happens to be parked on the receive.

### EventProcessor - What's a processor?

```go
type EventProcessor func(ctx context.Context, event *Event) error
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
func (eb *EventBus) Subscribe(eventType EventType, processor EventProcessor) string {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    sub := &Subscription{
        Id:         fmt.Sprintf("%s-%d", eventType, eb.subscriptionSeq.Add(1)),
        Type:       eventType,
        EProcessor: processor,
    }

    eb.subscriptions[eventType] = append(eb.subscriptions[eventType], sub)

    return sub.Id
}
```

The subscription ID is the event type plus a **monotonic per-bus counter**, giving IDs like
`user.created-1`, `user.created-2`. The mutex lock ensures thread safety when modifying the
subscription map.

**Step by step:**
1. **Lock** the subscriptions map (exclusive write access)
2. Generate a unique ID
3. Create a `Subscription` wrapper (ID + processor)
4. Append to the slice for this event type
5. Return the ID (so you can unsubscribe later)

**Why does uniqueness matter?** Because `Unsubscribe` removes the **first** subscription whose ID
matches. If two can share an ID, unsubscribing one silently removes the other, and the handler you
meant to remove stays live. Deriving the ID from the current length has exactly that problem: remove
`user.created-0` and the next subscriber is handed that ID again.

**Why defer unlock?** In case something panics, the lock is always released. Otherwise, you'd deadlock the entire bus.

```go
// Unsubscribe removes a subscription by ID.
func (eb *EventBus) Unsubscribe(subscriptionId string) bool {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    for eventType, subscriptions := range eb.subscriptions {
        for i, subscription := range subscriptions {
            if subscription.Id != subscriptionId {
                continue
            }
            if remaining := removeSubscriptionAt(subscriptions, i); len(remaining) > 0 {
                eb.subscriptions[eventType] = remaining
            } else {
                delete(eb.subscriptions, eventType) // don't leak empty map keys
            }
            return true
        }
    }

    return false
}
```

The unsubscribe operation iterates through all subscription lists to find and remove the matching subscription. This is O(n) complexity but is acceptable since unsubscription typically happens infrequently.

Note that it builds a **new slice** rather than doing the obvious
`append(subscriptions[:i], subscriptions[i+1:]...)`. That expression shifts elements down inside the
existing backing array, and `dispatch` walks that same array after releasing the read lock -- a data
race the detector flags. Adding needs no such care: an `append` either writes at index `len`, which
no in-flight reader can observe, or allocates a new array outright.

### Event Publishing

#### Asynchronous Publishing

```go
// Publish sends an event to all registered processors asynchronously.
func (eb *EventBus) Publish(event *Event) bool {
    if event == nil {
        return false
    }

    eb.mu.RLock()
    defer eb.mu.RUnlock()

    if eb.closed {
        return false
    }

    select {
    case eb.eventChan <- event:
        return true
    default: // buffer full - drop the event and tell the caller
        return false
    }
}
```

The publish operation uses a non-blocking select, so it never blocks the caller, and returns whether
the bus accepted the event.

**Why the read lock around the send?** Because `Shutdown` closes `eventChan`, and sending on a closed
channel is a **panic**, not an error. Checking a flag and then sending is not enough on its own -- the
close can land in between. `Shutdown` takes the write lock before closing, so holding the read lock
across both the check and the send is what makes them mutually exclusive.

**Why drop instead of block?** Because blocking the publisher could freeze your entire app. It's better to lose an event than hang the system. (You can adjust `BufferSize` to prevent this.)

**Why return a bool?** Because a dropped event nobody hears about is indistinguishable from one that
was processed. Only the caller knows whether this particular event is worth a retry, a log line, a
metric, or a fall back to `PublishSync`.

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

    var errors []error

    for _, sub := range processors {
        if err := sub.EProcessor(ctx, event); err != nil {
            errors = append(errors, fmt.Errorf("handler %s: %w", sub.Id, err))
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

    for event := range eb.eventChan {
        _ = eb.dispatch(eb.ctx, event)
    }
}
```

Each worker ranges over the event channel. `range` ends when the channel is closed **and drained**,
which is exactly the shutdown behaviour we want.

**Why not also select on `ctx.Done()`?** That is the obvious shape, and it quietly throws away the
backlog. `select` picks a ready case at random, so once the context is cancelled a worker with a
hundred buffered events is as likely to exit as to process the next one -- and everything the bus had
already accepted and acknowledged to its publishers evaporates. Closing the channel says "no more
events are coming"; the workers finish what is in it and exit on their own.

#### dispatch() - Running processors

```go
// dispatch sends an event to all registered processors.
func (eb *EventBus) dispatch(ctx context.Context, event *Event) []error {
    eb.mu.RLock()
    subscriptions := eb.subscriptions[event.Type]
    eb.mu.RUnlock()

    var pubErrs []error
    for _, subscription := range subscriptions {
        if subscription.EProcessor == nil {
            continue
        }
        if err := subscription.EProcessor(ctx, event); err != nil {
            pubErrs = append(pubErrs, fmt.Errorf("handler %s: %w", subscription.Id, err))
        }
    }

    return pubErrs
}
```

The dispatch method takes the read lock only long enough to read the slice header, then runs each
processor and **collects** their errors rather than swallowing them. `PublishSync` returns that slice
to its caller; a worker discards it.

**Step by step:**
1. Look up processors for this event type
2. Run each in turn; a failure is collected and the rest still run
3. Return every error, tagged with the subscription that produced it

**Why not stop on error?** Because processors are independent. If the email processor fails, the audit processor should still run.

**Why collect instead of log?** Because a library that logs on your behalf gives you one hard-coded
destination and no way to react. Compose `middleware.WithLogging` with a destination of your choice.



### Graceful Shutdown

```go
// Shutdown gracefully stops the event bus and waits for all workers to finish.
func (eb *EventBus) Shutdown() error {
    eb.shutdownOnce.Do(func() {
        eb.mu.Lock()
        eb.closed = true
        close(eb.eventChan) // workers drain what is left, then their range loops end
        eb.mu.Unlock()

        drained := make(chan struct{})
        go func() {
            eb.wg.Wait()
            close(drained)
        }()

        select {
        case <-drained:
        case <-time.After(eb.shutdownTimeout):
            eb.shutdownErr = goserr.NewTimeoutErrorf(eb.shutdownTimeout.String(),
                "event bus shutdown timed out with events still in flight")
        }

        eb.cancel() // cancelled last: nothing runs under this context any more
    })
    return eb.shutdownErr
}
```

The shutdown process:
1. Marks the bus closed and closes the event channel, under the write lock
2. Waits for the workers to drain the backlog and exit
3. Gives up after `ShutdownTimeout` and reports it, rather than hanging forever
4. Cancels the context the processors were given

**Order matters, and it is the opposite of the intuitive one:**

- **Close the channel first, under the write lock.** This is the stop signal, and doing it under the
  lock is what stops a concurrent `Publish` from sending on a closed channel and panicking.
- **Then wait.** The workers are ranging over the channel, so they finish the buffer and exit on
  their own. Every event the bus accepted is delivered.
- **Cancel the context last.** It is what processors are handed. Cancelling it first aborts the very
  events this drain exists to save -- every context-aware write in that final batch fails instantly.

**Why the timeout?** Because "waiting for the workers" and "one processor is wedged on a dead socket"
look identical from here. Five seconds by default, then the context is cancelled so context-aware
processors unwind, and the caller is told the drain did not finish.

**Why `sync.Once`?** Two independent defers on one bus is an ordinary thing to write, and closing an
already-closed channel is a panic. Later callers get the first call's result.

This gives you **no accepted event lost**, **no goroutine leak**, and **no panic on the way out**.

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
// Middleware wraps an EventProcessor with additional behavior.
type Middleware func(EventProcessor) EventProcessor
```

This functional approach allows for composable behavior around processors.

#### The Pattern

A middleware takes a processor and returns a **new processor** that does something extra.

### Retry Middleware

```go
// WithRetry retries failed processors with exponential backoff.
func WithRetry(maxRetries int, initialDelay time.Duration) Middleware {
    return func(next EventProcessor) EventProcessor {
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
    if timeout <= 0 {
        return func(next EventProcessor) EventProcessor { return next }
    }
    return func(next EventProcessor) EventProcessor {
        return func(ctx context.Context, event *Event) error {
            ctx, cancel := context.WithTimeout(ctx, timeout)
            defer cancel()

            done := make(chan error, 1) // buffered so the processor's goroutine can always exit
            go func() {
                done <- next(ctx, event)
            }()

            select {
            case err := <-done:
                return err

            case <-ctx.Done():
                if errors.Is(ctx.Err(), context.DeadlineExceeded) {
                    return goserr.NewTimeoutError(timeout.String(), ctx.Err(),
                        fmt.Sprintf("processor for %s did not finish in time", event.Type))
                }
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
- If timeout hits first, report a `TimeoutError` wrapping `context.DeadlineExceeded`, so both
  `goserr.IsTimeout(err)` and `errors.Is(err, context.DeadlineExceeded)` recognise it

**Why the guard on a non-positive timeout?** `context.WithTimeout(ctx, 0)` returns an
already-expired context, so every event would fail instantly while its processor kept running,
detached, in a goroutine nothing waits on. A non-positive duration means no timeout.

**What this does not do:** the deadline bounds how long the *middleware* waits, not the processor.
A processor that ignores `ctx` keeps running after the timeout has been reported. The buffered
`done` channel is what stops that goroutine leaking when nobody is left to receive from it.

### Recovery Middleware

```go
// WithRecovery recovers from panics in processors and reports them as errors.
func WithRecovery() Middleware {
    return func(next EventProcessor) EventProcessor {
        return func(ctx context.Context, event *Event) (err error) {
            defer func() {
                r := recover()
                if r == nil {
                    return
                }
                log.Printf("[Middleware] Recovered from panic in handler for event %s: %v\n%s", event.Type, r, debug.Stack())

                cause, ok := r.(error)
                if !ok {
                    cause = fmt.Errorf("%v", r)
                }
                err = goserr.NewProcessingError(cause, fmt.Sprintf("recovered from panic in handler for event %s", event.Type))
            }()

            return next(ctx, event)
        }
    }
}
```

Recovery middleware uses defer and recover to catch panics in processors, preventing them from crashing the worker goroutine. This is essential for robust event processing.

**Why report the panic instead of swallowing it?** Returning nil made a panicking handler
indistinguishable from a clean one: `PublishSync` reported no error, `WithRetry` had nothing to
retry, and `WithLogging` logged "completed". A panic is a failure, and the caller should be able to
tell. If the panic value is itself an error it stays reachable through `errors.As`.

**Where to put it in a `Chain`:** innermost -- last in the argument list -- so the middleware wrapped
around it observes the failure. An outer `WithRecovery` catches the panic only after it has already
unwound past them.

### Logging Middleware

```go
// WithLogging logs processor execution, prefixing each line with msg.
func WithLogging(msg string, logFunc func(message string)) Middleware {
    return func(next EventProcessor) EventProcessor {
        return func(ctx context.Context, event *Event) error {
            start := time.Now()
            err := next(ctx, event)
            duration := time.Since(start)

            if err != nil {
                logFunc(fmt.Sprintf("%s [Middleware] Processor for %s failed after %v: %v", msg, event.Type, duration, err))
            } else {
                logFunc(fmt.Sprintf("%s [Middleware] Processor for %s completed in %v", msg, event.Type, duration))
            }

            return err
        }
    }
}
```

Logging middleware provides execution time and error information for debugging and monitoring. It
takes the destination as a parameter rather than reaching for `log` itself, so it composes with
whatever logger the application already has.

Note that the formatted line goes into a **local**. Assigning it back to the captured `msg` -- which
is easy to write and was the bug here for a while -- appends each line onto the prefix for the next
one, so the message grows without bound for the life of the wrapped processor, and concurrent events
race on that shared string.

### Middleware Chaining

```go
// Chain chains multiple middlewares together.
func Chain(middlewares ...Middleware) Middleware {
    return func(processor EventProcessor) EventProcessor {
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
Chain(WithLogging("orders", logFn), WithRetry(3, 100*time.Millisecond), WithRecovery())(processor)
```

Becomes:
```
WithLogging(
    WithRetry(
        WithRecovery(
            processor
        )
    )
)
```

So recovery turns a panic in the processor into an error, retry sees that error and retries, and
logging records the final outcome. Putting recovery outermost instead would catch the panic only
after it had unwound past retry and logging, so neither would ever see it.

## Event Filtering

Event filtering allows processors to conditionally execute based on event properties:

```go
// Filter determines if an event should be processed by a processor.
type Filter func(*Event) bool

// Filteredprocessor wraps a processor with a filter condition.
type Filteredprocessor struct {
    filter  Filter
    processor EventProcessor
}
```

#### Filter - What Is It?

A filter is a function that returns `true` if the event should be processed, `false` otherwise.

### Basic Filter Implementation

```go
// NewFilteredprocessor creates a processor that only executes when the filter returns true.
func NewFilteredprocessor(filter Filter, processor EventProcessor) EventProcessor {
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
// BatchProcessor processes multiple events at once.
type BatchProcessor func(ctx context.Context, events []*Event) error

// BatchErrorHandler is called when a BatchProcessor returns an error.
type BatchErrorHandler func(ctx context.Context, events []*Event, err error)

// Batch collects events and processes them in batches.
type Batch struct {
    mu             sync.Mutex
    eventType      EventType
    batchSize      int
    flushPeriod    time.Duration
    batchProcessor BatchProcessor
    onError        BatchErrorHandler
    buffer         []*Event
    ctx            context.Context
    cancel         context.CancelFunc
    stop           chan struct{} // stops the periodic flush, separately from ctx
    stopOnce       sync.Once
    closed         bool          // set by Shutdown; after which Add refuses events
    wg             sync.WaitGroup // the periodic flush goroutine
    flushWg        sync.WaitGroup // the goroutines running batchProcessor
}

// BatchConfig holds configuration for batch processing.
type BatchConfig struct {
    BatchSize   int               // Max events per batch; <= 0 falls back to 100
    FlushPeriod time.Duration     // Max time to wait before flushing; <= 0 disables time-based flushing
    OnError     BatchErrorHandler // Called with the batch and the error when the processor fails
}
```

#### Why Batch?

If you're sending 1000 emails, it's inefficient to make 1000 API calls. Better to batch them:
- Make 10 calls with 100 emails each
- Or wait 5 seconds and send all at once

### Batch Processor Implementation

```go
// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(eventType EventType, config BatchConfig, processor BatchProcessor) *Batch {
    batchSize := config.BatchSize
    if batchSize <= 0 {
        batchSize = defaultBatchSize // 100
    }

    ctx, cancel := context.WithCancel(context.Background())

    bp := &Batch{
        eventType:      eventType,
        batchSize:      batchSize,
        flushPeriod:    config.FlushPeriod,
        batchProcessor: processor,
        onError:        config.OnError,
        buffer:         make([]*Event, 0, batchSize),
        ctx:            ctx,
        cancel:         cancel,
        stop:           make(chan struct{}),
    }

    bp.start()
    return bp
}
```

The batch processor creates a cancellation context and initializes its buffer. It also starts the periodic flush goroutine during initialization.

**Why clamp `BatchSize`?** Because `len(buffer) >= 0` is true after every single `Add`, so a zero
batch size flushes a batch of one per event -- the exact opposite of batching.

#### Batch - The Structure

```go
type Batch struct {
    buffer         []*Event
    batchSize      int
    flushPeriod    time.Duration
    batchProcessor BatchProcessor
    onError        BatchErrorHandler
    // ... synchronization stuff
}
```

**How it works:**
1. Events get added to `buffer`
2. When `buffer` reaches `batchSize`, flush it (call processor with all events)
3. Or, every `flushPeriod` seconds, flush whatever's in the buffer

### Event Addition and Flushing

```go
// Add adds an event to the batch buffer, reporting whether it was buffered.
func (bp *Batch) Add(eventToAdd *Event) bool {
    bp.mu.Lock()
    defer bp.mu.Unlock()

    if bp.closed || eventToAdd == nil {
        return false
    }

    bp.buffer = append(bp.buffer, eventToAdd)

    if len(bp.buffer) >= bp.batchSize {
        bp.flush()
    }
    return true
}

// Flush processes all buffered events immediately.
func (bp *Batch) Flush() {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    bp.flush()
}
```

Events are added to the buffer under lock protection. When the buffer reaches capacity, it's automatically flushed to process the batch.

`Add` refuses events once the batch has been shut down: nothing is left to flush the buffer, so
accepting them would only lose them more quietly.

```go
// flush processes the current buffer (must be called with lock held).
func (bp *Batch) flush() {
    if len(bp.buffer) == 0 || bp.batchProcessor == nil {
        return
    }

    events := make([]*Event, len(bp.buffer))
    copy(events, bp.buffer)
    bp.buffer = bp.buffer[:0]

    bp.flushWg.Add(1)
    go func() {
        defer bp.flushWg.Done()

        if err := bp.batchProcessor(bp.ctx, events); err != nil && bp.onError != nil {
            bp.onError(bp.ctx, events, err)
        }
    }()
}
```

The flush operation copies the buffer to avoid holding the lock while processing, then processes the batch in a separate goroutine to prevent blocking the caller.

Two details carry weight. The goroutine is tracked on `flushWg`, which is what lets `Shutdown` wait
for the final batch instead of returning while it is still being written. And the error goes to the
configured `BatchErrorHandler`: retry, dead-lettering and metrics are the caller's decisions to make,
but they have to be told about the failure first.

### Periodic Flushing

```go
// start begins the periodic flush goroutine.
func (bp *Batch) start() {
    if bp.flushPeriod <= 0 {
        return
    }
    bp.wg.Add(1)
    go bp.periodicFlush()
}

// periodicFlush flushes the buffer at regular intervals.
func (bp *Batch) periodicFlush() {
    defer bp.wg.Done()

    ticker := time.NewTicker(bp.flushPeriod)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            bp.Flush()

        case <-bp.stop:
            return
        }
    }
}
```

The periodic flush goroutine ensures that events don't stay in the buffer indefinitely. It flushes events either when the batch size is reached or when the flush period expires.

**Why the guard in `start`?** `time.NewTicker` panics on a non-positive interval. Since a
`FlushPeriod` of zero is the documented way to ask for size-only batching, starting the ticker
anyway took the whole process down a moment after construction. Not starting the goroutine is what
size-only should have meant all along.

**Why a `stop` channel rather than `ctx.Done()`?** Because the two events happen at different points
in `Shutdown`: the ticker has to stop before the final flush, and the context has to stay live
through it.

### Event processor Wrapper

```go
// AsEventProcessor returns an EventProcessor that adds events to the batch.
func (bp *Batch) AsEventProcessor() EventProcessor {
    return func(ctx context.Context, eventToAdd *Event) error {
        if !bp.Add(eventToAdd) {
            return goserr.NewProcessingErrorf("batch processor for %s is shut down", bp.eventType)
        }
        return nil
    }
}
```

This method allows the batch processor to be used as a regular event processor by simply adding events to the batch. A refused add becomes an error, so the bus learns the event went nowhere.

### Batch Processor Shutdown

```go
// Shutdown stops the batch processor and flushes remaining events.
func (bp *Batch) Shutdown() {
    bp.stopOnce.Do(func() {
        bp.mu.Lock()
        bp.closed = true
        bp.mu.Unlock()

        close(bp.stop) // stop the periodic flush
        bp.wg.Wait()   // wait for it to actually stop

        bp.Flush()        // hand off whatever is left
        bp.flushWg.Wait() // ...and wait for that batch to land

        bp.cancel() // nothing runs under this context any more
    })
}
```

The shutdown process stops the periodic goroutine, flushes any remaining events, waits for that final
batch to be processed, and only then cancels the context.

**Why wait on `flushWg`?** Because `flush` hands the batch to a goroutine. Without the wait, Shutdown
returns while the final batch is still being written, and the process can exit before it lands --
which is the exact opposite of what "flushes remaining events" promises.

**Why cancel last?** Because `bp.ctx` is what the batch processor is given. Cancelling first means the
final batch runs under an already-cancelled context, and every context-aware database write inside it
fails immediately.

**Why `sync.Once`?** A second Shutdown would otherwise close an already-closed channel and panic.

## Global Event Bus

Gossip provides a global singleton event bus for application-wide access:

```go
var (
    globalBus     *EventBus
    globalBusOnce sync.Once
)

// GetGlobalBus returns the singleton event bus instance.
func GetGlobalBus() *EventBus {
    globalBusOnce.Do(func() {
        globalBus = NewEventBus(DefaultConfig())
    })
    return globalBus
}

// InitGlobalBus initializes the global event bus with custom configuration.
func InitGlobalBus(cfg *Config) {
    globalBusOnce.Do(func() {
        globalBus = NewEventBus(cfg)
    })
}
```

The global bus uses `sync.Once` to ensure it's initialized only once, even in concurrent scenarios.

Note that both functions share that `Once`. `InitGlobalBus` has to run before anything reaches for
`GetGlobalBus`, or the defaults win and the custom config is silently ignored -- call it early in
`main()`.

There are no package-level `Publish` or `Subscribe` shortcuts. Going through `GetGlobalBus()` keeps
it visible at every call site that a process-wide singleton is involved.

## Testing Implementation

The implementation includes comprehensive tests covering various scenarios.

One habit worth adopting before reading them: **synchronise on `Shutdown()`, not on `time.Sleep`.**
`Shutdown` drains everything the bus has accepted and returns once the workers are done, so an
assertion after it is exact and there is no timing to tune. The sleeps below are the older style and
work, but they are guesses -- they can be too short on a loaded CI box and are always too long
locally. `PublishSync` is the other option, when the test wants the processors' errors too.

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

    require.NoError(t, bus.Shutdown()) // drains: every accepted event is processed before this returns

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

    require.NoError(t, bus.Shutdown()) // drains: every accepted event is processed before this returns

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
    require.NoError(t, bus.Shutdown()) // drains: every accepted event is processed before this returns

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

### 6. Why Functional Options for Nil Checking in `CastEventData`?
**Answer:** Performance vs. Safety trade-offs.
In Go, an `interface{}` holding a typed nil (e.g., `var u *User = nil`) is NOT equal to `nil`. Checking for this "typed nil" requires reflection, which is slow in hot paths.
Instead of forcing reflection on everyone, or ignoring the problem, we used the **Functional Options Pattern**:
- **Default (Fast Path)**: Only checks for untyped `nil`. Zero overhead.
- **Opt-in (Safe Path)**: `WithReflectNilCheck()` for non-critical paths.
- **Opt-in (Hot Path)**: `WithCustomNilCheck()` allows users to provide a zero-allocation type assertion checker for their specific types.
  This perfectly aligns with the Open/Closed principle and gives developers exact control over their performance profiles.

---

## How to Extend This Library

### Adding a New Middleware

Want rate limiting? Create a middleware:

```go
func WithRateLimit(limit int, period time.Duration) Middleware {
    limiter := rate.NewLimiter(rate.Every(period), limit)
    
    return func(next EventProcessor) EventProcessor {
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
