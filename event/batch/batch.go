// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package batch. batch provides batch processing capabilities for events.
// It enables efficient processing of high-volume events by collecting them into
// batches before invoking the processing logic, reducing overhead from
// repeated function calls, database operations, or network requests.
package batch

import (
	"context"
	"sync"
	"time"

	"github.com/seyallius/gossip/v2/event"
	goserr "github.com/seyallius/gossip/v2/event/errors"
	"github.com/seyallius/gossip/v2/event/sub"
)

// defaultBatchSize is used when BatchConfig.BatchSize is zero or negative. Without it, every Add
// satisfies len(buffer) >= batchSize and flushes a batch of one, which is the opposite of batching.
const defaultBatchSize = 100

// ----------------------------------------------- Types -------------------------------------------- //

// BatchProcessor is a function that processes multiple events at once.
// This is more efficient than processing events individually when:
//   - Making database bulk inserts
//   - Sending batch emails or notifications
//   - Aggregating metrics before writing
//   - Making batch API calls
//
// Example usage:
//
//	batchProcessor := func(ctx context.Context, events []*event.Event) error {
//	    // Process all events together
//	    return bulkInsertToDatabase(events)
//	}
type BatchProcessor func(ctx context.Context, events []*event.Event) error

// BatchErrorHandler is called when a BatchProcessor returns an error. It receives the same context
// and events the processor was given, so it can retry them, write them to a dead-letter queue, or
// just count them.
type BatchErrorHandler func(ctx context.Context, events []*event.Event, err error)

// Batch collects events and processes them in batches according to the
// configured batching strategy (size-based or time-based).
//
// The Batch struct implements a buffering mechanism where events are
// collected until either:
//  1. The buffer reaches BatchSize (size-based flushing)
//  2. The FlushPeriod elapses (time-based flushing)
//  3. Manual Flush() is called
//  4. Shutdown() is called (flushes remaining events)
//
// This pattern is commonly known as the "Buffered Processor" or
// "Accumulator" pattern in event-driven architectures.
type Batch struct {
	// eventType is the strongly-typed event identifier this batch processor
	// subscribes to. The batch processor only processes events of this type.
	// This ensures type safety and prevents processing unrelated events.
	eventType event.EventType

	// batchProcessor is the actual business logic that processes batched events.
	// This function receives a slice of events that have been accumulated
	// according to the batching rules. It's called once per batch.
	batchProcessor BatchProcessor

	// buffer holds events that have been added but not yet processed.
	// The buffer has a capacity equal to BatchSize to avoid reallocations.
	// When len(buffer) >= batchSize, the buffer is automatically flushed.
	buffer []*event.Event

	// batchSize defines the maximum number of events to collect before
	// automatically flushing the batch. When the buffer reaches this size,
	// flush() is called immediately.
	//
	// Example: If batchSize = 100, then every 100 events trigger processing.
	batchSize int

	// flushPeriod defines the maximum time to wait before flushing the buffer,
	// even if it hasn't reached batchSize. This ensures events aren't stuck
	// in the buffer indefinitely during low-traffic periods.
	//
	// Example: If flushPeriod = 5 seconds, events are processed at least
	// every 5 seconds, even if only 1 event was received.
	flushPeriod time.Duration

	// onError receives any error a BatchProcessor returns. Nil means errors are discarded, which is
	// what this type used to do unconditionally.
	onError BatchErrorHandler

	// ctx and cancel provide the context handed to the batch processor. It is cancelled at the very
	// end of Shutdown, once nothing is running under it any more -- cancelling it earlier would hand
	// the final batch a dead context, which is exactly the batch that most needs a live one.
	ctx    context.Context
	cancel context.CancelFunc

	// stop signals the periodic flush goroutine to exit. It is separate from ctx so that stopping the
	// ticker and cancelling the processor context can happen at different points in Shutdown.
	stop     chan struct{}
	stopOnce sync.Once

	// closed is set by Shutdown, after which Add refuses events: they would be buffered with nothing
	// left to flush them, and would restart the flush WaitGroup that Shutdown has already waited on.
	closed bool

	// mu protects concurrent access to the buffer.
	// Since Add() can be called from multiple goroutines (e.g., multiple
	// event handlers), we need a mutex to prevent race conditions when
	// modifying the buffer slice.
	mu sync.Mutex

	// wg tracks the periodic flush goroutine for graceful shutdown.
	// When Shutdown() is called, it waits for this goroutine to exit
	// before returning, ensuring no goroutine leaks.
	wg sync.WaitGroup

	// flushWg tracks the goroutines running batchProcessor. flush() hands the batch off
	// asynchronously, so without this Shutdown returned while the final batch was still being
	// written -- and the process could exit before it landed.
	flushWg sync.WaitGroup
}

// BatchConfig holds configuration for batch processing.
type BatchConfig struct {
	// BatchSize is the maximum number of events to collect before
	// automatically flushing the batch. A larger batch size increases
	// efficiency but delays processing. A smaller size reduces latency
	// but increases overhead.
	BatchSize int

	// FlushPeriod is the maximum time to wait before flushing the buffer,
	// even if it hasn't reached BatchSize. This prevents events from
	// being stuck in the buffer during periods of low activity.
	//
	// Setting this to 0 (or a negative value) disables time-based flushing, leaving Add, Flush and
	// Shutdown as the only triggers.
	FlushPeriod time.Duration

	// OnError is called with the context, the batch, and the error whenever a BatchProcessor fails.
	// Leaving it nil discards errors, which is what this package used to do with no way to opt out.
	OnError BatchErrorHandler
}

// NewBatchProcessor creates a new batch processor with the given configuration.
// It also starts the configured number of worker goroutines and creates
// a context.WithCancel for cancellation.
// Parameters:
//   - eventType: The type of events this processor will handle
//   - config: Configuration for batch size and flush timing
//   - processor: The batch processing function that will be called with batched events
//
// Returns a pointer to a Batch that can be used to add events manually
// or converted to an EventProcessor for use with an EventBus.
//
// Example:
//
//	config := BatchConfig{BatchSize: 100, FlushPeriod: 5 * time.Second}
//	processor := NewBatchProcessor("user.login", config, batchLoginProcessor)
//	defer processor.Shutdown()
func NewBatchProcessor(eventType event.EventType, config BatchConfig, processor BatchProcessor) *Batch {
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	ctx, cancel := context.WithCancel(context.Background())
	bp := &Batch{
		eventType:      eventType,
		batchSize:      batchSize,
		flushPeriod:    config.FlushPeriod,
		batchProcessor: processor,
		onError:        config.OnError,
		buffer:         make([]*event.Event, 0, batchSize), // Pre-allocate capacity fixme: sync.pool?
		ctx:            ctx,
		cancel:         cancel,
		stop:           make(chan struct{}),
	}

	bp.start()
	return bp
}

// -------------------------------------------- Public API ------------------------------------------ //

// Add adds an event to the batch buffer.
//
// This method is thread-safe and can be called from multiple goroutines
// concurrently. If adding this event causes the buffer to reach or exceed
// batchSize, the buffer is automatically flushed (processed).
//
// Note: If the buffer is full and a flush is triggered, this method
// blocks until the flush completes (specifically, until the buffer is
// copied and cleared). For high-throughput scenarios, consider using
// a non-blocking approach or larger batch sizes.
//
// It reports whether the event was buffered. A false means the batch has been shut down: nothing is
// left to flush the buffer, so accepting the event would only lose it more quietly.
func (bp *Batch) Add(eventToAdd *event.Event) bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed || eventToAdd == nil {
		return false
	}

	bp.buffer = append(bp.buffer, eventToAdd)

	// Size-based trigger: If we've collected enough events, flush now
	if len(bp.buffer) >= bp.batchSize {
		bp.flush()
	}
	return true
}

// Flush processes all buffered events immediately.
//
// This method forces processing of any events currently in the buffer,
// regardless of whether batchSize has been reached. This is useful when:
//   - The application is shutting down and needs to process remaining events
//   - You need to ensure events are processed before taking some action
//   - Testing or debugging batch behavior
//
// The method is thread-safe and blocks until the buffer is cleared
// (but not necessarily until the batchProcessor completes, as it runs
// in a separate goroutine).
func (bp *Batch) Flush() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.flush()
}

// Shutdown stops the batch processor and flushes remaining events.
//
// This method performs graceful shutdown in the following order:
//  1. Signals the periodic flush goroutine to stop, and stops accepting new events
//  2. Waits for the periodic flush goroutine to exit
//  3. Flushes any remaining events in the buffer
//  4. Waits for every in-flight BatchProcessor to return
//  5. Cancels the context the processors were given
//
// Steps 4 and 5 are what make the guarantee real. flush() hands the batch to a goroutine, so
// Shutdown used to return while the final batch was still being written -- and it cancelled the
// context first, meaning that batch ran under an already-cancelled one and any context-aware write
// inside it failed immediately.
//
// It blocks until the final batch has been processed, so a BatchProcessor that never returns will
// hold up shutdown. Calling it more than once is safe; later calls wait for the first to finish.
//
// It's important to call Shutdown() before the application exits to
// ensure all events are processed and goroutines are cleaned up.
// Typically used with defer:
//
//	processor := NewBatchProcessor(...)
//	defer processor.Shutdown()
func (bp *Batch) Shutdown() {
	bp.stopOnce.Do(func() {
		bp.mu.Lock()
		bp.closed = true
		bp.mu.Unlock()

		close(bp.stop) // Signal periodic flush to stop
		bp.wg.Wait()   // Wait for it to actually stop

		bp.Flush()        // Process any remaining events
		bp.flushWg.Wait() // ...and wait for that batch to actually land

		bp.cancel() // Nothing runs under this context any more
	})
}

// AsEventProcessor returns a sub.EventProcessor that adds events to the batch.
//
// This adapter method allows a Batch to be used as a regular event handler
// with the bus.EventBus. When the returned function is called, it simply
// adds the event to the batch buffer.
//
// Example integration with bus.EventBus:
//
//	bus := NewEventBus(DefaultConfig())
//	batchProcessor := NewBatchProcessor(...)
//	bus.Subscribe("user.login", batchProcessor.AsEventProcessor())
//
// This enables seamless integration of batch processing into the existing
// event-driven architecture without modifying event publishing code.
func (bp *Batch) AsEventProcessor() sub.EventProcessor {
	return func(ctx context.Context, eventToAdd *event.Event) error {
		if !bp.Add(eventToAdd) {
			return goserr.NewProcessingErrorf("batch processor for %s is shut down", bp.eventType)
		}
		return nil
	}
}

// ----------------------------------------- Internal Helpers --------------------------------------- //

// start begins the periodic flush goroutine.
//
// This internal method starts a background goroutine that periodically
// flushes the buffer based on flushPeriod. The goroutine runs until
// Shutdown() signals it to stop.
//
// The goroutine is tracked with a WaitGroup so Shutdown() can wait for
// it to exit cleanly.
//
// Nothing is started when flushPeriod is not positive: that is the documented way to ask for
// size-only batching, and time.NewTicker panics on a non-positive interval, so the configuration the
// docs recommended used to take the process down at construction time.
func (bp *Batch) start() {
	if bp.flushPeriod <= 0 {
		return
	}
	bp.wg.Add(1)
	go bp.periodicFlush()
}

// periodicFlush flushes the buffer at regular intervals.
//
// This method runs in a separate goroutine and periodically triggers
// flushing based on flushPeriod. It ensures that events don't get stuck
// in the buffer during low-traffic periods.
//
// The method uses a ticker to trigger at regular intervals and listens
// for the stop signal to exit gracefully during shutdown.
func (bp *Batch) periodicFlush() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: // Time-based trigger: Flush if enough time has passed
			bp.Flush()

		case <-bp.stop: // Shutdown signal - exit goroutine
			return
		}
	}
}

// flush processes the current buffer (must be called with lock held).
//
// This is the core batching logic:
//  1. Checks if buffer is empty (nothing to do)
//  2. Copies the buffer to a new slice (to release the lock quickly)
//  3. Clears the original buffer (resets to empty)
//  4. Calls batchProcessor asynchronously with the copied events
//
// The buffer copy (events := make(...)) is important because:
//   - It allows the mutex to be released quickly (after copy, before processing)
//   - It prevents the batchProcessor from seeing buffer modifications
//   - It enables safe concurrent Adds while processing is happening
//
// The batchProcessor is called in a separate goroutine to:
//   - Avoid blocking the Add() method during slow processing
//   - Allow concurrent processing of multiple batches
//   - Maintain responsiveness of the event system
//
// The goroutine is tracked on flushWg so Shutdown can wait for it, and any error is handed to the
// configured BatchErrorHandler -- retry, dead-lettering and metrics are the caller's to decide, but
// they need to be told about the failure first.
func (bp *Batch) flush() {
	if len(bp.buffer) == 0 || bp.batchProcessor == nil {
		return
	}

	// Copy buffer contents to release lock quickly
	events := make([]*event.Event, len(bp.buffer))
	copy(events, bp.buffer)
	bp.buffer = bp.buffer[:0] // Clear buffer (keep capacity)

	// Process asynchronously to avoid blocking
	bp.flushWg.Add(1)
	go func() {
		defer bp.flushWg.Done()

		if err := bp.batchProcessor(bp.ctx, events); err != nil && bp.onError != nil {
			bp.onError(bp.ctx, events, err)
		}
	}()
}
