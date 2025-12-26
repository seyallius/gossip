// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package batch

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -------------------------------------------- Batch Processor Tests --------------------------------------------

func TestNewBatchProcessor_CreatesProcessor(t *testing.T) {
	config := BatchConfig{
		BatchSize:   10,
		FlushPeriod: 1 * time.Second,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	require.NotNil(t, processor)
	defer processor.Shutdown()
}

func TestBatchProcessor_FlushesOnBatchSize(t *testing.T) {
	var mu sync.Mutex
	var batches []int

	config := BatchConfig{
		BatchSize:   3,
		FlushPeriod: 10 * time.Second, // Long period so size triggers flush
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		batches = append(batches, len(events))
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	// Add 7 events (should create 2 batches: 3 + 3, with 1 remaining)
	for i := 0; i < 7; i++ {
		processor.Add(event.NewEvent("test.event", i))
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, batches, 2, "Should have 2 batches")
	assert.Equal(t, 3, batches[0])
	assert.Equal(t, 3, batches[1])
}

func TestBatchProcessor_FlushesOnPeriod(t *testing.T) {
	var mu sync.Mutex
	var batches []int

	config := BatchConfig{
		BatchSize:   100,                    // Large batch size
		FlushPeriod: 100 * time.Millisecond, // Short period to trigger flush
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		batches = append(batches, len(events))
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	// Add 5 events (below batch size)
	for i := 0; i < 5; i++ {
		processor.Add(event.NewEvent("test.event", i))
	}

	// Wait for periodic flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	require.GreaterOrEqual(t, len(batches), 1, "Should have at least 1 batch")
	assert.Equal(t, 5, batches[0])
}

func TestBatchProcessor_ManualFlush(t *testing.T) {
	var mu sync.Mutex
	var processedEvents []*event.Event

	config := BatchConfig{
		BatchSize:   100,
		FlushPeriod: 10 * time.Second,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		processedEvents = append(processedEvents, events...)
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	// Add events
	for i := 0; i < 5; i++ {
		processor.Add(event.NewEvent("test.event", i))
	}

	// Manual flush
	processor.Flush()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, processedEvents, 5)
}

func TestBatchProcessor_EmptyBuffer(t *testing.T) {
	invoked := false

	config := BatchConfig{
		BatchSize:   10,
		FlushPeriod: 100 * time.Millisecond,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		invoked = true
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	// Flush without adding any events
	processor.Flush()
	time.Sleep(50 * time.Millisecond)

	assert.False(t, invoked, "Should not invoke handler for empty buffer")
}

func TestBatchProcessor_ShutdownFlushesRemaining(t *testing.T) {
	var mu sync.Mutex
	var processedEvents []*event.Event
	handlerDone := make(chan struct{})

	config := BatchConfig{
		BatchSize:   100,
		FlushPeriod: 10 * time.Second,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		defer close(handlerDone)

		processedEvents = append(processedEvents, events...)
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)

	// Add events (below batch size)
	for i := 0; i < 5; i++ {
		processor.Add(event.NewEvent("test.event", i))
	}

	// Shutdown should flush remaining events
	processor.Shutdown()

	// Wait for handler to complete
	select {
	case <-handlerDone:
		// Processor completed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Processor didn't complete in time")
	}

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, processedEvents, 5, "Shutdown should flush remaining events")
}

func TestBatchProcessor_AsEventProcessor(t *testing.T) {
	counter := int32(0)

	config := BatchConfig{
		BatchSize:   3,
		FlushPeriod: 10 * time.Second,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		atomic.AddInt32(&counter, int32(len(events)))
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	handler := processor.AsEventProcessor()

	// Use as regular event handler
	for i := 0; i < 5; i++ {
		err := handler(context.Background(), event.NewEvent("test.event", i))
		assert.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(3), atomic.LoadInt32(&counter), "Should process first batch of 3")
}

func TestBatchProcessor_WithEventBus(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	var mu sync.Mutex
	var batches []int

	config := BatchConfig{
		BatchSize:   5,
		FlushPeriod: 10 * time.Second,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		batches = append(batches, len(events))
		return nil
	}

	processor := NewBatchProcessor("order.created", config, batchProcessor)
	defer processor.Shutdown()

	eventBus.Subscribe("order.created", processor.AsEventProcessor())

	// Publish 12 events
	for i := 0; i < 12; i++ {
		eventBus.Publish(event.NewEvent("order.created", i))
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have 2 full batches of 5
	require.GreaterOrEqual(t, len(batches), 2)
	assert.Equal(t, 5, batches[0])
	assert.Equal(t, 5, batches[1])
}

func TestBatchProcessor_ConcurrentAdds(t *testing.T) {
	counter := int32(0)

	config := BatchConfig{
		BatchSize:   10,
		FlushPeriod: 100 * time.Millisecond,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		atomic.AddInt32(&counter, int32(len(events)))
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor.Add(event.NewEvent("test.event", nil))
		}()
	}

	wg.Wait()
	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, int32(numGoroutines), atomic.LoadInt32(&counter))
}

func TestBatchProcessor_PreserveEventData(t *testing.T) {
	type OrderData struct {
		OrderID string
		Amount  float64
	}

	var mu sync.Mutex
	var receivedOrders []OrderData

	config := BatchConfig{
		BatchSize:   3,
		FlushPeriod: 10 * time.Second,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()

		for _, evt := range events {
			data := evt.Data.(OrderData)
			receivedOrders = append(receivedOrders, data)
		}
		return nil
	}

	processor := NewBatchProcessor("order.created", config, batchProcessor)
	defer processor.Shutdown()

	orders := []OrderData{
		{OrderID: "order1", Amount: 99.99},
		{OrderID: "order2", Amount: 149.99},
		{OrderID: "order3", Amount: 199.99},
	}

	for _, order := range orders {
		processor.Add(event.NewEvent("order.created", order))
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, receivedOrders, 3)
	assert.Equal(t, "order1", receivedOrders[0].OrderID)
	assert.Equal(t, "order2", receivedOrders[1].OrderID)
	assert.Equal(t, "order3", receivedOrders[2].OrderID)
}

func TestBatchProcessor_MultipleBatches(t *testing.T) {
	var mu sync.Mutex
	var allEvents []*event.Event

	config := BatchConfig{
		BatchSize:   5,
		FlushPeriod: 50 * time.Millisecond,
	}

	batchProcessor := func(ctx context.Context, events []*event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		allEvents = append(allEvents, events...)
		return nil
	}

	processor := NewBatchProcessor("test.event", config, batchProcessor)
	defer processor.Shutdown()

	// Add 23 events (4 full batches of 5 + 1 batch of 3)
	for i := 0; i < 23; i++ {
		processor.Add(event.NewEvent("test.event", i))
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 23, len(allEvents))
}
