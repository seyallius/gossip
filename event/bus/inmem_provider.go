// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. inmem_provider implements the in-memory Go channel based transport.
package bus

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// InMemoryProvider implements the Provider interface using Go channels.
type InMemoryProvider struct {
	subscriptions map[event.EventType][]*sub.Subscription // list of subscriptions the EventBus has to process (pub or sub) todo: use map for efficiency?
	eventChan     chan *event.Event                       // the actual even in a channel for async processing

	workers int // number of worker goroutines (concurrent)

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	wg     sync.WaitGroup
}

// NewInMemoryProvider creates the in-memory provider.
// It also starts the configured number of worker goroutines and creates
// a context.WithCancel for cancellation.
func NewInMemoryProvider(workers, bufferSize int) *InMemoryProvider {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &InMemoryProvider{
		subscriptions: make(map[event.EventType][]*sub.Subscription),
		eventChan:     make(chan *event.Event, bufferSize),
		workers:       workers,
		ctx:           ctx,
		cancel:        cancel,
	}
	provider.start()
	return provider
}

// -------------------------------------------- Public Functions --------------------------------------------

func (p *InMemoryProvider) Publish(eventToPub *event.Event) error {
	select {
	// publish event by putting it on event channel
	case p.eventChan <- eventToPub:
		return nil

	// publish canceled - ignore and return
	case <-p.ctx.Done():
		return errors.New("provider closed")

	// event buffer full - drop event todo: no error?
	default:
		return errors.New("event buffer full")
	}
}

func (p *InMemoryProvider) Subscribe(eventType event.EventType, processor sub.EventProcessor) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := subscriptionIdGenerator(eventType)
	subscription := &sub.Subscription{
		Id:         id,
		Type:       eventType,
		EProcessor: processor,
	}

	p.subscriptions[eventType] = append(p.subscriptions[eventType], subscription)
	return id, nil
}

func (p *InMemoryProvider) Unsubscribe(subscriptionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// optimize: O(n * m)! consider map for O(1)
	for eventType, subscriptions := range p.subscriptions {
		for i, subscription := range subscriptions {
			if subscription.Id == subscriptionID {
				p.subscriptions[eventType] = append(subscriptions[:i], subscriptions[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (p *InMemoryProvider) Shutdown() error {
	p.cancel()  // Cancel context to signal workers to stop
	go func() { // Drain the channel (process remaining events)
		for range p.eventChan { // Discard remaining events or process them
		}
	}()
	time.Sleep(50 * time.Millisecond) // Close after a brief delay to allow draining
	close(p.eventChan)                // Close the channel
	p.wg.Wait()                       // Wait for all workers to finish processing)
	return nil
}

// -------------------------------------------- Private Functions --------------------------------------------

// start initializes worker goroutines to process events.
func (p *InMemoryProvider) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker processes events from the channel.
func (p *InMemoryProvider) worker() {
	defer p.wg.Done()

	for {
		select {
		case evt, ok := <-p.eventChan: // read event and dispatch
			if !ok { // channel closed - exit worker
				return
			}
			p.dispatch(evt) // dispatch and ignore error

		case <-p.ctx.Done(): // cancel signal - shutdown worker
			return
		}
	}
}

// dispatch sends an event to all registered sub.EventProcessor(s).
func (p *InMemoryProvider) dispatch(evt *event.Event) {
	p.mu.RLock()
	subs := p.subscriptions[evt.Type]
	p.mu.RUnlock()

	for _, s := range subs {
		// In in-memory provider, we just call the function directly
		go func() {
			_ = s.EProcessor(p.ctx, evt)
		}()
	}
}
