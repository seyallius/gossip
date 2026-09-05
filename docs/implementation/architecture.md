# Architecture Deep Dive

This section provides a comprehensive look at the internal architecture of Gossip, including the concurrency model, thread safety patterns, and architectural decisions.

## Core Architecture Components

### EventBus Structure

Let me explain **EVERY. SINGLE. FIELD.**

```go
type EventBus struct {
    subscriptions   map[EventType][]*Subscription // Map of event types to processors
    eventChan       chan *Event                   // Buffered channel for events
    workers         int                           // Number of worker goroutines
    shutdownTimeout time.Duration                 // How long Shutdown waits for the drain
    subscriptionSeq atomic.Uint64                 // Monotonic counter behind every subscription ID
    ctx             context.Context               // Context handed to processors
    cancel          context.CancelFunc            // Function to cancel that context
    mu              sync.RWMutex                  // Read-write mutex for the subscription map
    closed          bool                          // Guarded by mu; true once Shutdown closed eventChan
    shutdownOnce    sync.Once                     // Makes Shutdown idempotent
    shutdownErr     error                         // Replayed to later Shutdown callers
    wg              sync.WaitGroup                // WaitGroup to track workers
}
```

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
eb.subscriptions[eventType] = append(eb.subscriptions[eventType], sub)
eb.mu.Unlock()
```

Note what the read side does **not** do: hold the lock while running processors. It reads the slice
header and lets go, so one slow processor cannot block every subscription change on the bus. That
puts the safety burden on the write side -- see *Why removal copies and addition does not* below.

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

This is the context handed to every processor, and it is cancelled at the **end** of `Shutdown`, once
the backlog has drained.

```go
ctx, cancel := context.WithCancel(context.Background())
```

It is tempting to cancel it first, as the signal that shutdown has begun. Do that and the final batch
of events -- the ones the drain exists to save -- all run under an already-cancelled context, so every
context-aware database write and HTTP call inside them fails instantly. Closing `eventChan` is the
stop signal; cancelling the context is the cleanup.

##### `wg sync.WaitGroup`

A **WaitGroup** is like a counter.

- When a worker starts: `wg.Add(1)`
- When a worker finishes: `wg.Done()`
- When shutting down: `wg.Wait()` blocks until all workers finish

This ensures **graceful shutdown** - we don't just kill workers mid-processing.

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
        Workers:    10,
        BufferSize: 1000,
    }
}
```

The configuration allows users to tune the event bus performance based on their workload characteristics. More workers can process events in parallel, while a larger buffer can handle bursty event loads.

### EventBus Creation

```go
// NewEventBus creates a new event bus with the given configuration.
func NewEventBus(cfg *Config) *EventBus {
    if cfg == nil {
        cfg = DefaultConfig()
    }

    // Out-of-range values are replaced, not honoured
    workers := cfg.Workers
    if workers <= 0 {
        workers = defaultWorkers
    }
    bufferSize := cfg.BufferSize
    if bufferSize <= 0 {
        bufferSize = defaultBufferSize
    }
    shutdownTimeout := cfg.ShutdownTimeout
    if shutdownTimeout <= 0 {
        shutdownTimeout = defaultShutdownTimeout
    }

    ctx, cancel := context.WithCancel(context.Background())

    bus := &EventBus{
        subscriptions:   make(map[EventType][]*Subscription),
        eventChan:       make(chan *Event, bufferSize),
        workers:         workers,
        shutdownTimeout: shutdownTimeout,
        ctx:             ctx,
        cancel:          cancel,
    }

    bus.start()
    return bus
}
```

Note that the event channel is buffered with `bufferSize`. This allows for non-blocking publish operations up to the buffer capacity.

**Why clamp the config?** Because none of the out-of-range values describes a usable bus. Zero workers
accepts events and never processes them. A zero buffer size makes the channel unbuffered, and since
`Publish` uses a non-blocking send, that means nearly every publish is dropped -- depending on whether
a worker happens to be parked on the receive at that instant.

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

**Why does uniqueness matter so much?** Because `Unsubscribe` removes the **first** subscription whose
ID matches. If two subscriptions can share an ID, then unsubscribing one silently removes the other,
and the handler you meant to remove stays live. Anything derived from the current length of the slice
has the same problem -- remove `user.created-0` and the next subscriber is handed that ID again.

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

### Why removal copies and addition does not

`dispatch` releases the read lock before running processors, so it walks a slice that `Subscribe` and
`Unsubscribe` may be changing underneath it. That makes what each of them does to the backing array
the whole safety argument -- and the two are not symmetric.

The obvious removal is `append(subscriptions[:i], subscriptions[i+1:]...)`, and it is a **data race**
here: it shifts elements down inside the existing backing array, over exactly the indices an
in-flight dispatch is reading. The race detector flags it. So removal builds a new slice:

```go
func removeSubscriptionAt(subscriptions []*Subscription, i int) []*Subscription {
    remaining := make([]*Subscription, 0, len(subscriptions)-1)
    remaining = append(remaining, subscriptions[:i]...)
    return append(remaining, subscriptions[i+1:]...)
}
```

Addition needs no such thing. A plain `append` either writes at index `len` -- which no in-flight
reader can observe, because its slice header carries the old, smaller length -- or allocates a new
array outright. Both are invisible to a dispatch already in progress.

That asymmetry is worth keeping. Applying copy-on-write to `Subscribe` as well looks tidier and makes
every subscription an O(n) copy, which is quadratic across a run: in this codebase it took
`BenchmarkEventBus_Subscribe` from ~355ns/op to ~183µs/op. Removal is rare enough to absorb the
allocation; addition is not obliged to pay for symmetry it does not need.

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
channel is a **panic**, not an error. Checking a flag and then sending is not enough on its own: the
close can land between the two. `Shutdown` takes the write lock before closing, so holding the read
lock across both the check and the send is what makes them mutually exclusive. Without it, a request
goroutine publishing at the moment the process starts shutting down takes the process down with it.

**Why drop instead of block?** Because blocking the publisher could freeze your entire app. It's better to lose an event than hang the system. (You can adjust `BufferSize` to prevent this.)

**Why return a bool?** Because a dropped event that nobody hears about is indistinguishable from one
that was processed. The caller is the only one who knows whether this particular event is worth a
retry, a log line, a metric, or a fall back to `PublishSync`.

**Why reject nil?** A nil event used to reach `dispatch` and nil-dereference on `event.Type` -- between
the `RLock` and the `RUnlock`, so the panic left the subscription lock held and the next `Subscribe`
or `Shutdown` deadlocked behind it. The bus wedged, a long way from the code that published nil.

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
hundred buffered events is as likely to exit as to process the next one. Everything the bus had
already accepted and acknowledged to its publishers evaporates. Closing the channel says "no more
events are coming"; the workers then finish what is in it and exit on their own.

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
processor **in the calling goroutine**, collecting rather than swallowing their errors. `PublishSync`
returns that slice to its caller; a worker discards it.

**Step by step:**
1. Look up processors for this event type
2. Run each in turn; a failure is collected and the rest still run
3. Return every error, tagged with the subscription that produced it

**Why serially, not one goroutine each?** Concurrency across events is what the worker pool is for, and
it is bounded. Spawning a goroutine per processor per event is unbounded: a burst of events with five
subscribers each fans out to five goroutines per event with nothing throttling them. Running them in
the worker keeps the parallelism at `Workers`, which is the number the user configured.

**Why does dispatch return errors instead of logging them?** Because a library that logs on your behalf
gives you one hard-coded destination and no way to react. `PublishSync` hands them back; a processor
that wants logging composes `middleware.WithLogging` with the destination of its choice.

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

- **Close the channel first, under the write lock.** This is the stop signal. Doing it under the lock
  is what stops a concurrent `Publish` from sending on a closed channel and panicking.
- **Then wait.** The workers are ranging over the channel, so they finish the buffer and exit on their
  own. Every event the bus accepted is delivered.
- **Cancel the context last.** It is what processors are handed. Cancelling it first aborts the very
  events this drain exists to save.

**Why the timeout?** Because "wait for the workers" and "one processor is wedged on a dead socket" are
the same state from here. Five seconds by default, then the context is cancelled -- so context-aware
processors unwind -- and the caller is told the drain did not finish.

**Why `sync.Once`?** Two independent defers on one bus is an ordinary thing to write, and closing an
already-closed channel is a panic. Later callers get the first call's result.

This ensures **no accepted event is lost**, **no goroutine leak**, and **no panic on the way out**.

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
   - `dispatch()`, `PublishSync()` and `Publish()` use read locks
   - This allows concurrent reads while preventing concurrent writes/reads during modifications
   - The read lock is released before processors run, which is only safe because the write side is
     copy-on-write (see above)
   - `Publish()` holds it across the closed-check and the channel send, which is what makes those
     mutually exclusive with `Shutdown()` closing the channel

2. **Channel-based Communication**:
   - The event channel is safe for concurrent use
   - Workers consume from the same channel without additional synchronization

3. **Immutable Event Objects**:
   - Once created, event objects are not modified during processing
   - This prevents race conditions on event data

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
Note that both functions share that `Once`: `InitGlobalBus` has to run before anything reaches for
`GetGlobalBus`, or the defaults win and the custom config is silently ignored. Call it early in
`main()`.

There are no package-level `Publish` or `Subscribe` shortcuts. Go through `GetGlobalBus()`, which
keeps it obvious at every call site that a process-wide singleton is involved.

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

- Buffer sizes are pre-allocated where possible
- `dispatch` allocates its error slice lazily, so the common case of no failures costs nothing
- Subscription changes allocate a fresh slice (copy-on-write); publishes allocate none
- Event objects are treated as immutable after creation to avoid copying overhead

### Concurrency Patterns

- Read-write mutexes allow concurrent reads during dispatch
- Worker goroutines consume from a shared channel for load balancing
- Batch processing reduces per-event overhead

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

The cost is that the publisher no longer learns whether processing worked -- and, when the buffer is
full, whether it even started. `Publish` returning a bool covers the second half of that: the caller
is told the event was dropped. For the first half, use `PublishSync`.

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

## Next Steps

- [API Reference](./api.md) - Complete API documentation
- [Implementation Details](../implementation/overview.md) - Overview of implementation