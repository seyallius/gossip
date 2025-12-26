// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. event_bus provides the core pub/sub event bus implementation.
package bus

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// EventBus manages event publishing and subscription.
type EventBus struct {
	subscriptions map[event.EventType][]*sub.Subscription // list of subscriptions the EventBus has to process (pub or sub) todo: use map for efficiency?
	eventChan     chan *event.Event                       // the actual even in a channel for async processing

	workers int // number of worker goroutines (concurrent)

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	wg     sync.WaitGroup
}

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

// NewEventBus creates a new event bus with the given configuration.
// It also starts the configured number of worker goroutines and creates
// a context.WithCancel for cancellation.
func NewEventBus(cfg *Config) *EventBus {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	bus := &EventBus{
		subscriptions: make(map[event.EventType][]*sub.Subscription),
		eventChan:     make(chan *event.Event, cfg.BufferSize), // create a buffered channel for async - drop events when full (fixme?)

		workers: cfg.Workers,

		ctx:    ctx,
		cancel: cancel,
	}

	bus.start()

	return bus
}

// -------------------------------------------- Public Functions --------------------------------------------

// Subscribe registers a sub.EventProcessor for a specific event type and returns a subscription ID.
func (eb *EventBus) Subscribe(eventType event.EventType, eventProcessor sub.EventProcessor) string {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscription := &sub.Subscription{
		Id:         fmt.Sprintf("%s-%d", eventType, rand.Intn(eb.workers)), //fixme: should be random? user processed? a sequence with previous?
		Type:       eventType,
		EProcessor: eventProcessor,
	}

	eb.subscriptions[eventType] = append(eb.subscriptions[eventType], subscription)
	return subscription.Id
}

// Unsubscribe removes a subscription from EventBus.
func (eb *EventBus) Unsubscribe(subscriptionId string) bool {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// O(n * m)! consider map for O(1)
	for eventType, subscriptions := range eb.subscriptions {
		for i, subscription := range subscriptions {
			if subscription.Id == subscriptionId {
				eb.subscriptions[eventType] = append(subscriptions[:i], subscriptions[i+1:]...) //unsubscribe the processor
				return true
			}
		}
	}
	return false
}

// PublishSync sends an event to all registered handlers synchronously.
func (eb *EventBus) PublishSync(ctx context.Context, eventToPub *event.Event) []error {
	return eb.dispatch(ctx, eventToPub)
}

// Publish sends an event to all registered handlers asynchronously.
func (eb *EventBus) Publish(eventToPub *event.Event) {
	select {
	case eb.eventChan <- eventToPub: // publish event by putting it on event channel
	case <-eb.ctx.Done(): // publish canceled - ignore and return
	default: // event buffer full - drop event and do nothing todo: log?
	}
}

// Shutdown gracefully stops the event bus and waits for all workers to finish.
func (eb *EventBus) Shutdown() {
	eb.cancel() // Cancel context to signal workers to stop
	go func() { // Drain the channel (process remaining events)
		for range eb.eventChan { // Discard remaining events or process them
		}
	}()
	time.Sleep(50 * time.Millisecond) // Close after a brief delay to allow draining
	close(eb.eventChan)               // Close the channel
	eb.wg.Wait()                      // Wait for all workers to finish processing
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
func (eb *EventBus) worker() {
	defer eb.wg.Done()

	for {
		select {
		case eventToPub, ok := <-eb.eventChan: // read event and dispatch
			if !ok { // channel closed - exit worker
				return
			}
			_ = eb.dispatch(eb.ctx, eventToPub) // dispatch and ignore error

		case <-eb.ctx.Done(): // cancel signal - shutdown worker
			return
		}
	}
}

// dispatch sends an event to all registered sub.EventProcessor(s).
func (eb *EventBus) dispatch(ctx context.Context, eventToPub *event.Event) []error {
	eb.mu.RLock()
	subscriptions := eb.subscriptions[eventToPub.Type]
	eb.mu.RUnlock()

	if len(subscriptions) == 0 {
		return nil
	}

	pubErrs := make([]error, 0, len(subscriptions)) //fixme: sync.pool?
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
