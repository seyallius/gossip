# API Reference

This section provides a complete reference to the Gossip API, including all functions, types, and methods available in the library.

## Event Types and Data Structures

### EventType

```go
type EventType string
```

EventType represents a strongly-typed event identifier to prevent typos and ensure consistency.

**Usage:**
```go
const (
    UserCreated EventType = "user.created"
    UserUpdated EventType = "user.updated"
)
```

### Event

```go
type Event struct {
    Type      EventType      // Strongly-typed event identifier
    Timestamp time.Time      // When the event occurred
    Data      any            // Event-specific payload
    Metadata  map[string]any // Additional context (e.g., request ID, user agent)
}
```

The Event struct carries information through the system.

#### NewEvent

```go
func NewEvent(eventType EventType, data any) *Event
```

Creates a new event with the given type and data. Automatically sets the timestamp to `time.Now()` and initializes an empty metadata map.

**Example:**
```go
event := NewEvent(UserCreated, &UserData{
    UserID: "123",
    Email:  "user@example.com",
})
```

#### WithMetadata

```go
func (e *Event) WithMetadata(key string, value any) *Event
```

Adds metadata to the event and returns the event for chaining. This enables fluent API usage.

**Example:**
```go
event := NewEvent(UserCreated, data).
    WithMetadata("request_id", "req-123").
    WithMetadata("source", "api")
```

## EventBus

### Config

```go
type Config struct {
    Workers         int           // Number of worker goroutines
    BufferSize      int           // Size of the event channel buffer
    ShutdownTimeout time.Duration // How long Shutdown waits for the backlog to drain
}
```

Configuration for the event bus. Any field left at zero (or negative) falls back to its default:
10 workers, a buffer of 1000, and a 5 second shutdown timeout. Out-of-range values are replaced
rather than honoured -- a bus with no workers would accept events and never process them, and an
unbuffered channel turns `Publish`'s non-blocking send into a drop whenever no worker is parked on
the receive.

### Methods

#### Subscribe

```go
func (eb *EventBus) Subscribe(eventType EventType, processor EventProcessor) string
```

Registers a processor for a specific event type and returns a subscription ID. The ID is unique for
the lifetime of the bus and is the only handle `Unsubscribe` accepts, so keep it if the subscription
is ever to be removed. Several processors may subscribe to the same event type; each one receives
every published event.

**Example:**
```go
id := eventBus.Subscribe(UserCreated, func(ctx context.Context, event *Event) error {
    data := event.Data.(*UserData)
    log.Printf("User created: %s", data.Username)
    return nil
})
```

#### SubscribeMultiple

```go
func (eb *EventBus) SubscribeMultiple(eventTypes []EventType, processor EventProcessor) []string
```

Registers a processor for multiple event types and returns subscription IDs.

Parameters:

1. `eventTypes`: Slice of event types to subscribe to
2. `processor`: The processor function that will handle all these event types

Returns: A slice of subscription IDs (one per event type)

**Example:**
```go
eventTypes := []gossip.EventType{UserCreated, UserUpdated, UserDeleted}
ids := eventBus.SubscribeMultiple(eventTypes, func(ctx context.Context, event *Event) error {
    switch event.Type {
    case UserCreated:
        log.Printf("User created: %v", event.Data)
    case UserUpdated:
        log.Printf("User updated: %v", event.Data)
    case UserDeleted:
        log.Printf("User deleted: %v", event.Data)
    }
    return nil
})
```

#### Unsubscribe

```go
func (eb *EventBus) Unsubscribe(subscriptionId string)
```

Removes a subscription by its ID.

**Example:**
```go
eventBus.Unsubscribe(subscriptionId)
```

#### UnsubscribeMultiple

```go
func (eb *EventBus) UnsubscribeMultiple(subscriptionIDs []string)
```

Removes multiple subscriptions by their IDs.

Parameters:

- `subscriptionIDs`: Slice of subscription IDs to remove

**Example:**
```go
eventBus.UnsubscribeMultiple([]string{id1, id2, id3})
```

#### Publish

```go
func (eb *EventBus) Publish(event *Event) bool
```

Publishes an event to the bus asynchronously, reporting whether the bus accepted it. It never blocks:
`false` means the event was dropped, either because the buffer was full or because the bus is shut
down. It also rejects a nil event.

**Example:**
```go
event := gossip.NewEvent(UserCreated, &UserData{
    UserID: "123",
    Username: "alice",
})
if !eventBus.Publish(event) {
    log.Printf("event bus rejected the %s event", UserCreated)
}
```

#### Shutdown

```go
func (eb *EventBus) Shutdown() error
```

Shuts down the event bus: it stops accepting new events, lets the workers finish everything `Publish`
has already accepted, and then cancels the bus context. It returns an error only if that drain was
still running when `ShutdownTimeout` elapsed. Calling it more than once is safe -- later calls do
nothing and return the first call's result.

#### DefaultConfig

```go
func DefaultConfig() *Config
```

Returns sensible default configuration: 10 workers, a buffer size of 1000, and a 5 second shutdown
timeout.

**Example:**
```go
config := DefaultConfig()
config.Workers = 20
config.BufferSize = 2000
```

### NewEventBus

```go
func NewEventBus(cfg *Config) *EventBus
```

Creates a new event bus with the given configuration. If config is nil, uses default configuration.

**Example:**
```go
config := &Config{
    Workers:    10,
    BufferSize: 1000,
}
eventBus := NewEventBus(config)
defer eventBus.Shutdown()
```

### Subscribe

```go
func (eb *EventBus) Subscribe(eventType EventType, processor EventProcessor) string
```

Registers a processor for a specific event type and returns a subscription ID. The processor will be called whenever an event of the specified type is published.

**Example:**
```go
subscriptionID := eventBus.Subscribe(UserCreated, func(ctx context.Context, event *Event) error {
    data := event.Data.(*UserData)
    log.Printf("New user: %s", data.Username)
    return nil
})
```

### Publish

```go
func (eb *EventBus) Publish(event *Event) bool
```

Sends an event to all registered processors asynchronously. This method returns immediately and does
not wait for processors to complete. A full buffer means the event is dropped -- which is what the
return value is for.

**Example:**
```go
event := NewEvent(UserCreated, &UserData{
    UserID: "123",
    Email:  "user@example.com",
})
if !eventBus.Publish(event) {
    // Buffer full or bus shut down: retry, log, or fall back to PublishSync
    log.Printf("event bus rejected the %s event", UserCreated)
}
```

### PublishSync

```go
func (eb *EventBus) PublishSync(ctx context.Context, event *Event) []error
```

Sends an event to all registered processors synchronously and waits for all processors to complete. Returns a slice of errors from processors that failed.

**Example:**
```go
errors := eventBus.PublishSync(context.Background(), event)
if len(errors) > 0 {
    log.Printf("Some processors failed: %v", errors)
}
```

### Unsubscribe

```go
func (eb *EventBus) Unsubscribe(subscriptionID string) bool
```

Removes a subscription by ID. Returns true if the subscription was found and removed, false otherwise.

**Example:**
```go
success := eventBus.Unsubscribe(subscriptionID)
if success {
    log.Println("Successfully unsubscribed")
}
```

### Shutdown

```go
func (eb *EventBus) Shutdown() error
```

Gracefully stops the event bus. It closes the bus to new events, lets the workers drain the backlog
(so events already accepted are delivered rather than discarded), waits for them, and only then
cancels the context processors were given -- cancelling it first would abort the very events the
drain exists to save.

It blocks for at most `Config.ShutdownTimeout`, after which it cancels the context anyway and returns
a `TimeoutError`, so one wedged processor cannot hang process exit. It is idempotent.

Because it drains, `Shutdown` is also the tidiest way to make a test deterministic: publish, shut
down, assert -- no `time.Sleep` to tune.

**Example:**
```go
defer func() {
    if err := eventBus.Shutdown(); err != nil {
        log.Printf("event bus shutdown did not finish cleanly: %v", err)
    }
}()
```

## EventProcessor

```go
type EventProcessor func(ctx context.Context, event *Event) error
```

EventProcessor is a function that processes events. It takes a context for cancellation/timeout and an event to process, and returns an error if processing fails.

**Example:**
```go
func emailProcessor(ctx context.Context, event *Event) error {
    data := event.Data.(*UserData)
    return sendEmail(data.Email, "Welcome!")
}
```

## Middleware

### Middleware Type

```go
type Middleware func(EventProcessor) EventProcessor
```

Middleware wraps an EventProcessor with additional behavior. It takes an EventProcessor and returns a new EventProcessor that includes the middleware's behavior.

### WithRetry

```go
func WithRetry(maxRetries int, initialDelay time.Duration) Middleware
```

Creates a middleware that retries failed processors with exponential backoff. It runs the initial
attempt plus `maxRetries` retries, so `0` means a single attempt; a negative value is treated as `0`.

**Parameters:**
- `maxRetries`: Maximum number of retry attempts
- `initialDelay`: Initial delay between retries, doubled after each failure

**Example:**
```go
processor := WithRetry(3, 100*time.Millisecond)(myProcessor)
eventBus.Subscribe(UserCreated, processor)
```

### WithTimeout

```go
func WithTimeout(timeout time.Duration) Middleware
```

Creates a middleware that adds a timeout to processor execution. An expired deadline is reported as
an `errors.TimeoutError` wrapping `context.DeadlineExceeded`, so both `goserr.IsTimeout(err)` and
`errors.Is(err, context.DeadlineExceeded)` recognise it; a cancelled parent context is passed through
unchanged, because that is a cancellation and not a timeout. A non-positive timeout means no timeout.

Note that the deadline bounds how long the middleware waits, not the processor itself: a processor
that ignores `ctx` keeps running in its own goroutine after the timeout has been reported.

**Parameters:**
- `timeout`: Maximum time to allow for processor execution

**Example:**
```go
processor := WithTimeout(5*time.Second)(myProcessor)
eventBus.Subscribe(UserCreated, processor)
```

### WithRecovery

```go
func WithRecovery() Middleware
```

Creates a middleware that recovers from panics in processors, preventing them from crashing the
worker goroutine, and reports the panic as an `errors.ProcessingError`. If the panic value is itself
an error it stays reachable through `errors.As`. The stack is logged.

Put it innermost in a `Chain` (last in the argument list) so the middleware around it sees the error
and can act on it -- an outer `WithRecovery` catches the panic only after it has unwound past them.

**Example:**
```go
processor := WithRecovery()(myProcessor)
eventBus.Subscribe(UserCreated, processor)
```

### WithLogging

```go
func WithLogging(msg string, logFunc func(message string)) Middleware
```

Creates a middleware that logs processor execution time and errors, prefixing each line with `msg`
and handing it to `logFunc`.

**Parameters:**
- `msg`: Prefix for every logged line
- `logFunc`: Where to send the line -- `log.Println`, a structured logger, a test recorder

**Example:**
```go
processor := WithLogging("orders", func(message string) { log.Println(message) })(myProcessor)
eventBus.Subscribe(UserCreated, processor)
```

### Chain

```go
func Chain(middlewares ...Middleware) Middleware
```

Chains multiple middlewares together, applying them in reverse order (last to first).

**Example:**
```go
processor := Chain(
    WithLogging("orders", func(message string) { log.Println(message) }),
    WithRetry(3, 100*time.Millisecond),
    WithTimeout(5*time.Second),
    WithRecovery(), // innermost, so the retry and the log see a recovered panic as a failure
)(myProcessor)

eventBus.Subscribe(UserCreated, processor)
```

## Filters

### Filter Type

```go
type Filter func(*Event) bool
```

Filter determines if an event should be processed by a processor. Returns true if the event should be processed, false otherwise.

### FilteredProcessor

```go
func NewFilteredProcessor(filter Filter, processor EventProcessor) EventProcessor
```

Creates a processor that only executes when the filter returns true. A nil processor yields a
processor that does nothing, and a nil filter means "no condition", so the processor runs for every
event.

**Example:**
```go
highPriorityFilter := func(event *Event) bool {
    return event.Metadata["priority"] == "high"
}

filteredProcessor := NewFilteredProcessor(highPriorityFilter, myProcessor)
eventBus.Subscribe(UserCreated, filteredProcessor)
```

### FilterByMetadata

```go
func FilterByMetadata(key string, value any) Filter
```

Creates a filter that checks for specific metadata key-value pairs.

**Example:**
```go
apiFilter := FilterByMetadata("source", "api")
processor := NewFilteredProcessor(apiFilter, myProcessor)
```

### FilterByMetadataExists

```go
func FilterByMetadataExists(key string) Filter
```

Creates a filter that checks if a metadata key exists.

**Example:**
```go
hasUserFilter := FilterByMetadataExists("user_id")
processor := NewFilteredProcessor(hasUserFilter, myProcessor)
```

### Logical Filter Combinations

#### And

```go
func And(filters ...Filter) Filter
```

Combines multiple filters with AND logic. All filters must return true for the combined filter to return true.

**Example:**
```go
complexFilter := And(
    FilterByMetadata("source", "api"),
    FilterByMetadata("priority", "high"),
)
```

#### Or

```go
func Or(filters ...Filter) Filter
```

Combines multiple filters with OR logic. At least one filter must return true for the combined filter to return true.

**Example:**
```go
sourceFilter := Or(
    FilterByMetadata("source", "api"),
    FilterByMetadata("source", "web"),
)
```

#### Not

```go
func Not(filter Filter) Filter
```

Negates a filter.

**Example:**
```go
notTestFilter := Not(FilterByMetadata("env", "test"))
```

## Batch Processing

### BatchProcessor Type

```go
type BatchProcessor func(ctx context.Context, events []*Event) error
```

BatchProcessor processes multiple events at once.

### BatchConfig

```go
type BatchConfig struct {
    BatchSize   int               // Max events per batch
    FlushPeriod time.Duration     // Max time to wait before flushing
    OnError     BatchErrorHandler // Called with the batch and the error when the processor fails
}
```

Configuration for batch processing. A non-positive `BatchSize` falls back to 100; a non-positive
`FlushPeriod` disables time-based flushing entirely, leaving `Add`, `Flush` and `Shutdown` as the
only triggers. Leaving `OnError` nil discards processor errors.

```go
type BatchErrorHandler func(ctx context.Context, events []*Event, err error)
```

### NewBatchProcessor

```go
func NewBatchProcessor(eventType EventType, config BatchConfig, processor BatchProcessor) *Batch
```

Creates a new batch processor that collects events and processes them in batches.

**Example:**
```go
batchConfig := BatchConfig{
    BatchSize:   100,
    FlushPeriod: 5 * time.Second,
    OnError: func(ctx context.Context, events []*Event, err error) {
        log.Printf("batch of %d events failed: %v", len(events), err)
    },
}

batchProcessor := func(ctx context.Context, events []*Event) error {
    // Process all events together
    for _, event := range events {
        data := event.Data.(*UserData)
        log.Printf("Batch processing user: %s", data.Username)
    }
    return nil
}

processor := NewBatchProcessor(UserCreated, batchConfig, batchProcessor)

// Order matters: the bus feeds the batch, so drain the bus first
defer func() {
    _ = eventBus.Shutdown()
    processor.Shutdown()
}()

eventBus.Subscribe(UserCreated, processor.AsEventProcessor())
```

### Add

```go
func (bp *Batch) Add(event *Event) bool
```

Adds an event to the batch buffer, reporting whether it was buffered. If the buffer reaches capacity,
it automatically flushes the batch. It returns `false` once the batch has been shut down: nothing is
left to flush the buffer, so accepting the event would only lose it more quietly.

### Flush

```go
func (bp *Batch) Flush()
```

Processes all buffered events immediately. The processor runs in its own goroutine, so this returns
once the buffer has been handed off, not once the batch has been written.

### AsEventProcessor

```go
func (bp *Batch) AsEventProcessor() EventProcessor
```

Returns an EventProcessor that adds events to the batch. This allows the batch processor to be used
as a regular event processor. It returns an error if the batch has been shut down, so the bus learns
that the event went nowhere.

### Shutdown

```go
func (bp *Batch) Shutdown()
```

Stops the batch processor and flushes remaining events. It blocks until the final batch has actually
been processed and only then cancels the context the processors were given, so that last batch runs
under a live one. A `BatchProcessor` that never returns will hold up shutdown. It is idempotent.

## Global Event Bus

### GetGlobalBus

```go
func GetGlobalBus() *EventBus
```

Returns the singleton global event bus instance. The bus is initialized only once using sync.Once.

**Example:**
```go
globalBus := GetGlobalBus()
globalBus.Subscribe(UserCreated, myProcessor)
```

### InitGlobalBus

```go
func InitGlobalBus(cfg *Config)
```

Initializes the global bus with a custom configuration. Like `GetGlobalBus`, it runs only once: call
it during startup, before anything reaches for the bus, or the defaults win.

**Example:**
```go
func main() {
    bus.InitGlobalBus(&bus.Config{Workers: 20, BufferSize: 5000})
    defer func() { _ = bus.GetGlobalBus().Shutdown() }()
}
```

There are no package-level `Publish` or `Subscribe` shortcuts; go through `GetGlobalBus()`.

## Error Handling

### Standard Error Types

Gossip provides standard error types for conditional logic and error handling:

- `RetryableError` - Transient failures that can be retried
- `ValidationError` - Input validation failures
- `ProcessingError` - General processing failures
- `TimeoutError` - Operations that exceeded time limits
- `FatalError` - Unrecoverable errors that should not be retried
- `TypeAssertionError` - Type assertion failures
- `NoDataError` - Missing event data when data was expected
- `InvalidEventError` - Malformed or invalid events
- `UnsupportedEventTypeError` - Event types not supported by a processor

Each type has a `New<Type>Error` constructor taking an explicit cause, and a `New<Type>Errorf`
constructor that formats one. The formatted variants put the text in the `Err` field, which means the
format string may use `%w` to wrap a cause:

```go
err := errors.NewRetryableErrorf("connect to db: %w", cause)
errors.Is(err, cause)      // true
errors.IsRetryable(err)    // true
err.Error()                // "retryable error: connect to db: dial tcp: connection refused"
```

### Pre-defined Error Constants

- `errors.ErrNilData` - Pre-defined nil data error
- `errors.ErrTypeAssertionFailed` - Pre-defined type assertion failure
- `errors.ErrInvalidEvent` - Pre-defined invalid event error
- `errors.ErrUnsupportedEventType` - Pre-defined unsupported event type error
- `errors.ErrRetryable` - Pre-defined retryable error
- `errors.ErrValidationFailed` - Pre-defined validation error
- `errors.ErrProcessingFailed` - Pre-defined processing error
- `errors.ErrTimeout` - Pre-defined timeout error
- `errors.ErrFatal` - Pre-defined fatal error

### Error Checking Functions

#### IsRetryable

```go
func IsRetryable(err error) bool
```

Checks if an error is retryable.

#### IsFatal

```go
func IsFatal(err error) bool
```

Checks if an error is fatal (should not be retried).

#### IsValidation

```go
func IsValidation(err error) bool
```

Checks if an error is a validation error.

#### IsTypeAssertion

```go
func IsTypeAssertion(err error) bool
```

Checks if an error is a type assertion error.

#### IsNoData

```go
func IsNoData(err error) bool
```

Checks if an error indicates no data.

## Safe Type Casting (utils package)

The `utils` package provides utilities for safely casting event data with configurable nil-checking strategies.

### CastEventData

```go
func CastEventData[T any](e *event.Event, opts ...CastOption) (T, error)
```

Safely casts the event's `Data` field to type `T`. Returns an error if the event is nil, if the type
assertion fails, or if the data is nil.

**Options:**

- `WithReflectNilCheck()`: Uses reflection to check for typed nils (safest, but slower).
- `WithNoNilCheck()`: Disables typed nil checking entirely (fastest, default).
- `WithCustomNilCheck(checker NilChecker)`: Allows providing a custom, high-performance nil checker.

### NilChecker

```go
type NilChecker func (v any) bool
```

A function that returns true if the given value is considered nil. Used to detect "typed nils" without reflection.

## Performance Recommendations

1. **Tune `Workers` and `BufferSize`** to your workload; the bus is in-process, so events never leave the binary
2. **Batch Processing** is highly recommended for high-volume scenarios (1000+ events/second)
3. **Event Filtering** has minimal overhead and can reduce unnecessary processing
4. **Middleware** adds minimal overhead except for timeout middleware, which starts a goroutine per event

## Next Steps

- [Implementation Overview](./overview.md) - High-level overview of implementation
- [Architecture Deep Dive](./architecture.md) - Detailed architecture information