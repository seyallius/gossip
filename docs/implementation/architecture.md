# Architecture Deep Dive

This section provides a comprehensive look at the internal architecture of Gossip, including the concurrency model, thread safety patterns, and architectural decisions.

## Core Architecture Components

### EventBus Structure

Let me explain **EVERY. SINGLE. FIELD.**

```go
type EventBus struct {
    mu          sync.RWMutex                 // Read-write mutex for subscription map
    subscriptions map[EventType][]*Subscription // Map of event types to processors
    workers     int                          // Number of worker goroutines
    eventChan   chan *Event                  // Buffered channel for events
    ctx         context.Context              // Context for cancellation
    cancel      context.CancelFunc           // Function to cancel context
    wg          sync.WaitGroup               // WaitGroup to track workers
    provider    Provider                     // The underlying transport provider
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

    subscriptionID := fmt.Sprintf("%s-%d", eventType, len(eb.subscriptions[eventType]))

    sub := &Subscription{
        ID:         subscriptionID,
        Type:       eventType,
        Processor:  processor,
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
        // Event successfully sent to channel

    case <-eb.ctx.Done():
        // Event bus is shutting down

    default:
        // Event channel full, dropping event
        log.Printf("Event channel full, dropping event: %s", event.Type)
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
        if err := sub.Processor(ctx, event); err != nil {
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
        go func(s *Subscription) {
            if err := s.Processor(eb.ctx, event); err != nil {
                log.Printf("Processor %s failed for event %s: %v", s.ID, event.Type, err)
            }
        }(sub)
    }
}
```

The dispatch method gets a read lock to access the processors (avoiding blocking while processing), then calls each processor in a separate goroutine. This allows all processors for an event to run concurrently.

**Step by step:**
1. Look up processors for this event type
2. Run each processor in its own goroutine
3. If a processor fails, log the error but **don't stop** other processors

**Why run in goroutines?** So that if one processor is slow, it doesn't block other processors from running.

### Graceful Shutdown

```go
// Shutdown gracefully stops the event bus and waits for all workers to finish.
func (eb *EventBus) Shutdown() {
    log.Println("Shutting down event bus...")

    eb.cancel()           // ← Signal workers to stop
    close(eb.eventChan)   // ← Close the channel
    eb.wg.Wait()          // ← Wait for all workers to finish

    log.Println("Event bus shutdown complete")
}
```

The shutdown process:
1. Cancels the context, which signals all workers to stop
2. Closes the event channel, which allows workers to detect shutdown and return
3. Waits for all workers to finish using the wait group

This ensures all in-flight events are processed before the event bus exits.

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
func Subscribe(eventType EventType, processor EventProcessor) string {
    return GetGlobalBus().Subscribe(eventType, processor)
}
```

These functions provide a simple API for applications that want to use a global event bus without managing the instance themselves.

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

## Next Steps

- [API Reference](./api.md) - Complete API documentation
- [Implementation Details](../implementation/overview.md) - Overview of implementation