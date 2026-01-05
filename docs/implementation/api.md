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
    Driver     string // "memory" or "redis"
    Workers    int    // Number of worker goroutines
    BufferSize int    // Size of the event channel buffer
    RedisAddr  string // Redis address (for Redis provider)
    RedisPwd   string // Redis password (for Redis provider)
    RedisDB    int    // Redis database (for Redis provider)
}
```

Configuration for the event bus.

### Methods

#### Subscribe

```go
func (eb *EventBus) Subscribe(eventType EventType, processor EventProcessor) string
```

Registers a processor for a specific event type and returns a subscription ID.

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

**Example:**
```go
eventBus.UnsubscribeMultiple([]string{id1, id2, id3})
```

#### Publish

```go
func (eb *EventBus) Publish(event *Event) error
```

Publishes an event to the bus.

**Example:**
```go
event := gossip.NewEvent(UserCreated, &UserData{
    UserID: "123",
    Username: "alice",
})
err := eventBus.Publish(event)
if err != nil {
    log.Printf("Failed to publish event: %v", err)
}
```

#### Shutdown

```go
func (eb *EventBus) Shutdown() error
```

Shuts down the event bus and cleans up resources.

#### DefaultConfig

```go
func DefaultConfig() *Config
```

Returns sensible default configuration with 10 workers and a buffer size of 1000.

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
func (eb *EventBus) Publish(event *Event)
```

Sends an event to all registered processors asynchronously. This method returns immediately and does not wait for processors to complete. If the event channel is full, the event may be dropped.

**Example:**
```go
event := NewEvent(UserCreated, &UserData{
    UserID: "123",
    Email:  "user@example.com",
})
eventBus.Publish(event)
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
func (eb *EventBus) Shutdown()
```

Gracefully stops the event bus and waits for all workers to finish. This method blocks until all workers have completed their current tasks.

**Example:**
```go
defer eventBus.Shutdown() // Good practice to defer shutdown
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

Creates a middleware that retries failed processors with exponential backoff.

**Parameters:**
- `maxRetries`: Maximum number of retry attempts
- `initialDelay`: Initial delay between retries

**Example:**
```go
processor := WithRetry(3, 100*time.Millisecond)(myProcessor)
eventBus.Subscribe(UserCreated, processor)
```

### WithTimeout

```go
func WithTimeout(timeout time.Duration) Middleware
```

Creates a middleware that adds a timeout to processor execution.

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

Creates a middleware that recovers from panics in processors, preventing them from crashing the worker goroutine.

**Example:**
```go
processor := WithRecovery()(myProcessor)
eventBus.Subscribe(UserCreated, processor)
```

### WithLogging

```go
func WithLogging() Middleware
```

Creates a middleware that logs processor execution time and errors.

**Example:**
```go
processor := WithLogging()(myProcessor)
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
    WithRecovery(),
    WithRetry(3, 100*time.Millisecond),
    WithTimeout(5*time.Second),
    WithLogging(),
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

Creates a processor that only executes when the filter returns true.

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
    BatchSize   int           // Max events per batch
    FlushPeriod time.Duration // Max time to wait before flushing
}
```

Configuration for batch processing.

### NewBatchProcessor

```go
func NewBatchProcessor(eventType EventType, config BatchConfig, processor BatchProcessor) *BatchProcessor
```

Creates a new batch processor that collects events and processes them in batches.

**Example:**
```go
batchConfig := BatchConfig{
    BatchSize:   100,
    FlushPeriod: 5 * time.Second,
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
defer processor.Shutdown()

eventBus.Subscribe(UserCreated, processor.AsEventProcessor())
```

### Add

```go
func (bp *BatchProcessor) Add(event *Event)
```

Adds an event to the batch buffer. If the buffer reaches capacity, it automatically flushes the batch.

### Flush

```go
func (bp *BatchProcessor) Flush()
```

Processes all buffered events immediately.

### AsEventProcessor

```go
func (bp *BatchProcessor) AsEventProcessor() EventProcessor
```

Returns an EventProcessor that adds events to the batch. This allows the batch processor to be used as a regular event processor.

### Shutdown

```go
func (bp *BatchProcessor) Shutdown()
```

Stops the batch processor and flushes remaining events.

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

### Global Publish

```go
func Publish(event *Event)
```

A convenience function to publish events to the global bus.

**Example:**
```go
event := NewEvent(UserCreated, data)
Publish(event)
```

### Global Subscribe

```go
func Subscribe(eventType EventType, processor EventProcessor) string
```

A convenience function to subscribe to the global bus.

**Example:**
```go
subscriptionID := Subscribe(UserCreated, myProcessor)
```

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

## Performance Recommendations

1. **Use Memory Provider** for single-application, high-throughput scenarios
2. **Use Redis Provider** for distributed systems where events need to cross process boundaries
3. **Batch Processing** is highly recommended for high-volume scenarios (1000+ events/second)
4. **Event Filtering** has minimal overhead and can reduce unnecessary processing
5. **Middleware** adds minimal overhead except for timeout middleware which requires goroutine management

## Next Steps

- [Implementation Overview](./overview.md) - High-level overview of implementation
- [Architecture Deep Dive](./architecture.md) - Detailed architecture information