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

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

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

	// ctx and cancel provide graceful shutdown capability.
	// The context is cancelled when Shutdown() is called, which signals
	// the periodic flush goroutine to stop.
	ctx    context.Context
	cancel context.CancelFunc

	// mu protects concurrent access to the buffer.
	// Since Add() can be called from multiple goroutines (e.g., multiple
	// event handlers), we need a mutex to prevent race conditions when
	// modifying the buffer slice.
	mu sync.Mutex

	// wg tracks the periodic flush goroutine for graceful shutdown.
	// When Shutdown() is called, it waits for this goroutine to exit
	// before returning, ensuring no goroutine leaks.
	wg sync.WaitGroup
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
	// Setting this to 0 disables time-based flushing (size-only).
	// Setting this to a very large value effectively disables it.
	FlushPeriod time.Duration
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
	ctx, cancel := context.WithCancel(context.Background())
	bp := &Batch{
		eventType:      eventType,
		batchSize:      config.BatchSize,
		flushPeriod:    config.FlushPeriod,
		batchProcessor: processor,
		buffer:         make([]*event.Event, 0, config.BatchSize), // Pre-allocate capacity fixme: sync.pool?
		ctx:            ctx,
		cancel:         cancel,
	}

	bp.start()
	return bp
}

// -------------------------------------------- Public Functions --------------------------------------------

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
func (bp *Batch) Add(event *event.Event) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.buffer = append(bp.buffer, event)

	// Size-based trigger: If we've collected enough events, flush now
	if len(bp.buffer) >= bp.batchSize {
		bp.flush()
	}
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
//  1. Cancels the context to signal the periodic flush goroutine to stop
//  2. Waits for the periodic flush goroutine to exit (wg.Wait())
//  3. Flushes any remaining events in the buffer
//
// It's important to call Shutdown() before the application exits to
// ensure all events are processed and goroutines are cleaned up.
// Typically used with defer:
//
//	processor := NewBatchProcessor(...)
//	defer processor.Shutdown()
func (bp *Batch) Shutdown() {
	bp.cancel()  // Signal periodic flush to stop
	bp.wg.Wait() // Wait for it to actually stop
	bp.Flush()   // Process any remaining events
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
	return func(ctx context.Context, event *event.Event) error {
		bp.Add(event)
		return nil
	}
}

// -------------------------------------------- Private Helper Functions --------------------------------------------

// start begins the periodic flush goroutine.
//
// This internal method starts a background goroutine that periodically
// flushes the buffer based on flushPeriod. The goroutine runs until
// the context is cancelled (via Shutdown()).
//
// The goroutine is tracked with a WaitGroup so Shutdown() can wait for
// it to exit cleanly.
func (bp *Batch) start() {
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
// for context cancellation to exit gracefully during shutdown.
func (bp *Batch) periodicFlush() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: // Time-based trigger: Flush if enough time has passed
			bp.Flush()

		case <-bp.ctx.Done(): // Context cancelled (shutdown signal) - exit goroutine
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
// Error handling: Errors from batchProcessor are ignored.
func (bp *Batch) flush() {
	if len(bp.buffer) == 0 {
		return
	}

	// Copy buffer contents to release lock quickly
	events := make([]*event.Event, len(bp.buffer))
	copy(events, bp.buffer)
	bp.buffer = bp.buffer[:0] // Clear buffer (keep capacity)

	// Process asynchronously to avoid blocking
	go func() {
		if err := bp.batchProcessor(bp.ctx, events); err != nil {
			// ignore error and don't block
			//todo: consider:
			// - Retry logic
			// - Dead letter queue
			// - Metrics/alerting
			// - Circuit breaker pattern
		}
	}()
}
