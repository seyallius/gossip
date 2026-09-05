// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. event_bus provides the core pub/sub event bus implementation.
package bus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seyallius/gossip/v2/event"
	goserr "github.com/seyallius/gossip/v2/event/errors"
	"github.com/seyallius/gossip/v2/event/sub"
)

// ------------------------------------ Types, Variables & Constants -------------------------------- //

// Default configuration values, applied by NewEventBus to any Config field left at (or below) zero.
const (
	defaultWorkers         = 10
	defaultBufferSize      = 1000
	defaultShutdownTimeout = 5 * time.Second
)

// EventBus manages event publishing and subscription.
type EventBus struct {
	subscriptions map[event.EventType][]*sub.Subscription // subscriptions per event type; the slices are replaced, never edited in place
	eventChan     chan *event.Event                       // the actual event in a channel for async processing

	workers         int           // number of worker goroutines (concurrent)
	shutdownTimeout time.Duration // how long Shutdown waits for buffered events to drain

	subscriptionSeq atomic.Uint64 // monotonic counter behind every subscription ID

	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	closed       bool // guarded by mu; true once Shutdown has closed eventChan
	shutdownOnce sync.Once
	shutdownErr  error // result of the one real Shutdown, replayed to later callers
	wg           sync.WaitGroup
}

// Config holds configuration for the event bus.
type Config struct {
	Workers    int // Number of worker goroutines
	BufferSize int // Size of the event channel buffer

	// ShutdownTimeout bounds how long Shutdown waits for already-buffered events to be processed.
	// Zero or negative means the default of 5 seconds; there is no "wait forever" setting, because a
	// single wedged processor would otherwise hang process shutdown indefinitely.
	ShutdownTimeout time.Duration
}

// ------------------------------------------ Constructor(s) ---------------------------------------- //

// NewEventBus creates a new event bus with the given configuration.
// It also starts the configured number of worker goroutines and creates
// a context.WithCancel for cancellation.
//
// Out-of-range values are replaced by the defaults rather than honoured: a bus with zero workers
// would accept events and never process them, and a zero buffer size makes the event channel
// unbuffered, which turns Publish's non-blocking send into a drop whenever no worker happens to be
// parked on the receive.
func NewEventBus(cfg *Config) *EventBus {
	if cfg == nil {
		cfg = DefaultConfig()
	}

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
		subscriptions: make(map[event.EventType][]*sub.Subscription),
		eventChan:     make(chan *event.Event, bufferSize), // buffered channel for async publishing - Publish reports a full buffer rather than blocking

		workers:         workers,
		shutdownTimeout: shutdownTimeout,

		ctx:    ctx,
		cancel: cancel,
	}

	bus.start()

	return bus
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Workers:         defaultWorkers,
		BufferSize:      defaultBufferSize,
		ShutdownTimeout: defaultShutdownTimeout,
	}
}

// -------------------------------------------- Public API ------------------------------------------ //

// Subscribe registers a sub.EventProcessor for a specific event type and returns a subscription ID.
// The ID is unique for the lifetime of the bus and is the only handle Unsubscribe accepts, so keep
// it if the subscription is ever to be removed.
//
// Several processors may subscribe to the same event type; each one receives every published event.
func (eb *EventBus) Subscribe(eventType event.EventType, eventProcessor sub.EventProcessor) string {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscription := &sub.Subscription{
		Id:         fmt.Sprintf("%s-%d", eventType, eb.subscriptionSeq.Add(1)),
		Type:       eventType,
		EProcessor: eventProcessor,
	}

	eb.subscriptions[eventType] = appendSubscription(eb.subscriptions[eventType], subscription)
	return subscription.Id
}

// SubscribeMultiple registers a processor for multiple event types and returns subscription IDs.
// This is useful when the same logic applies to multiple event types (e.g., audit logging,
// metrics collection, or generic event handling).
func (eb *EventBus) SubscribeMultiple(eventTypes []event.EventType, processor sub.EventProcessor) []string {
	subscriptionIDs := make([]string, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		id := eb.Subscribe(eventType, processor)
		subscriptionIDs = append(subscriptionIDs, id)
	}
	return subscriptionIDs
}

// Unsubscribe removes a subscription from EventBus and reports whether one was found.
func (eb *EventBus) Unsubscribe(subscriptionId string) bool {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// O(n * m)! consider map for O(1)
	for eventType, subscriptions := range eb.subscriptions {
		for i, subscription := range subscriptions {
			if subscription.Id != subscriptionId {
				continue
			}
			if remaining := removeSubscriptionAt(subscriptions, i); len(remaining) > 0 {
				eb.subscriptions[eventType] = remaining
			} else {
				// Drop the key instead of leaving an empty slice behind, so an application that
				// churns through short-lived event types does not grow the map forever.
				delete(eb.subscriptions, eventType)
			}
			return true
		}
	}
	return false
}

// UnsubscribeMultiple removes multiple subscriptions by their IDs.
func (eb *EventBus) UnsubscribeMultiple(subscriptionIDs []string) {
	for _, id := range subscriptionIDs {
		eb.Unsubscribe(id)
	}
}

// PublishSync sends an event to all registered handlers synchronously and returns their errors.
func (eb *EventBus) PublishSync(ctx context.Context, eventToPub *event.Event) []error {
	if eventToPub == nil {
		return []error{goserr.NewInvalidEventErrorf("cannot publish a nil event")}
	}
	return eb.dispatch(ctx, eventToPub)
}

// Publish sends an event to all registered handlers asynchronously and reports whether the bus
// accepted it. It never blocks: false means the event was dropped, either because the buffer was
// full or because the bus is shut down. Callers that cannot tolerate a silent drop should check the
// result and either retry, log it, or fall back to PublishSync.
func (eb *EventBus) Publish(eventToPub *event.Event) bool {
	if eventToPub == nil {
		return false
	}

	// The read lock is what makes the send safe against a concurrent Shutdown: Shutdown takes the
	// write lock before closing eventChan, so the channel cannot be closed between the closed check
	// and the send. Without it, publishing during shutdown panics with "send on closed channel".
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return false
	}

	select {
	case eb.eventChan <- eventToPub:
		return true
	default: // buffer full - drop the event and tell the caller
		return false
	}
}

// Shutdown gracefully stops the event bus: it stops accepting new events, lets the workers finish
// everything Publish has already accepted, and then cancels the bus context.
//
// It returns an error only if buffered events were still being processed when ShutdownTimeout
// elapsed; the context is cancelled in that case too, so context-aware processors unwind. Calling it
// more than once is safe -- later calls do nothing and return the first call's result.
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

		// Cancelled last, not first: the context is what processors are handed, so cancelling it up
		// front would abort the very events this shutdown is trying to drain.
		eb.cancel()
	})
	return eb.shutdownErr
}

// -------------------------------------------- Private Functions --------------------------------------------

// start initializes worker goroutines to process events.
func (eb *EventBus) start() {
	for i := 0; i < eb.workers; i++ {
		eb.wg.Add(1)
		go eb.worker()
	}
}

// worker processes events from the channel.
//
// It ranges over the channel rather than also selecting on ctx.Done(): closing eventChan in Shutdown
// ends the range once the buffer is empty, so events the bus already accepted are delivered instead
// of being discarded the moment shutdown starts.
func (eb *EventBus) worker() {
	defer eb.wg.Done()

	for eventToPub := range eb.eventChan {
		_ = eb.dispatch(eb.ctx, eventToPub) // dispatch and ignore error
	}
}

// dispatch sends an event to all registered sub.EventProcessor(s).
func (eb *EventBus) dispatch(ctx context.Context, eventToPub *event.Event) []error {
	eb.mu.RLock()
	subscriptions := eb.subscriptions[eventToPub.Type]
	eb.mu.RUnlock()

	// Iterating with the lock released is safe because Subscribe and Unsubscribe replace this slice
	// instead of editing it, which makes what we hold an immutable snapshot. Holding the lock for the
	// whole loop would instead let one slow processor block every subscription change on the bus.
	var pubErrs []error
	for _, subscription := range subscriptions {
		if subscription.EProcessor == nil {
			continue
		}
		if err := subscription.EProcessor(ctx, eventToPub); err != nil {
			pubErrs = append(pubErrs, fmt.Errorf("handler %s: %w", subscription.Id, err))
		}
	}

	return pubErrs
}

// appendSubscription returns a new slice holding subscriptions plus added.
//
// Copy-on-write, because dispatch keeps reading the old slice after releasing the read lock. Growing
// the slice in place would be safe on its own, but pairing it with an in-place removal is not, so
// both mutations use the same discipline. Subscriptions change far less often than events are
// published, which is what makes paying an allocation here the cheap side of the trade.
func appendSubscription(subscriptions []*sub.Subscription, added *sub.Subscription) []*sub.Subscription {
	updated := make([]*sub.Subscription, len(subscriptions), len(subscriptions)+1)
	copy(updated, subscriptions)
	return append(updated, added)
}

// removeSubscriptionAt returns a new slice holding every subscription except the one at index i.
func removeSubscriptionAt(subscriptions []*sub.Subscription, i int) []*sub.Subscription {
	remaining := make([]*sub.Subscription, 0, len(subscriptions)-1)
	remaining = append(remaining, subscriptions[:i]...)
	return append(remaining, subscriptions[i+1:]...)
}
