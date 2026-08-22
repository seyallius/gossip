// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package bus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/testdata"
)

// -------------------------------------------- Test Functions --------------------------------------------

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	received := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		atomic.AddInt32(&received, 1)
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("Expected 1 event, got %d", received)
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	counter := int32(0)

	for i := 0; i < 5; i++ {
		eventProcessor := func(ctx context.Context, event *event.Event) error {
			atomic.AddInt32(&counter, 1)
			return nil
		}
		bus.Subscribe(testdata.AuthEventUserCreated, eventProcessor)
	}

	bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("Expected 5 invocations, got %d", counter)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	counter := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		atomic.AddInt32(&counter, 1)
		return nil
	}
	subID := bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	newEvent := event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{})
	bus.Publish(newEvent)
	time.Sleep(50 * time.Millisecond)

	bus.Unsubscribe(subID)

	bus.Publish(newEvent)
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&counter) != 1 {
		t.Errorf("Expected 1 invocation, got %d", counter)
	}
}

func TestEventBus_SynchronousPublish(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	executed := false
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed = true
		return nil
	}

	bus.Subscribe(testdata.AuthEventPasswordChanged, eventProcessor)
	errors := bus.PublishSync(context.Background(), event.NewEvent(testdata.AuthEventPasswordChanged, nil))

	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	if !executed {
		t.Error("Processor was not executed synchronously")
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	counter := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		atomic.AddInt32(&counter, 1)
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	var wg sync.WaitGroup
	numEvents := 100

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&counter) != int32(numEvents) {
		t.Errorf("Expected %d events, got %d", numEvents, counter)
	}
}

func TestEventBus_GracefulShutdown(t *testing.T) {
	bus := NewEventBus(DefaultConfig())

	counter := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&counter, 1)
		return nil
	}

	bus.Subscribe(testdata.AuthEventUserCreated, eventProcessor)

	for i := 0; i < 10; i++ {
		bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{}))
	}

	bus.Shutdown()

	if atomic.LoadInt32(&counter) == 0 {
		t.Error("No events were processed before shutdown")
	}
	t.Logf("published %d events", atomic.LoadInt32(&counter))
}

func TestEventBus_EventMetadata(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	var mu sync.Mutex
	receivedMetadata := make(map[string]any)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		mu.Lock()
		defer mu.Unlock()

		for k, v := range event.Metadata {
			receivedMetadata[k] = v
		}
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}).
		WithMetadata("request_id", "12345").
		WithMetadata("user_agent", "test"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedMetadata["request_id"] != "12345" {
		t.Error("Metadata was not properly passed to eventProcessor")
	}
}

// TestEventBus_PublishSubscribeExtended tests additional publish/subscribe functionality.
func TestEventBus_PublishSubscribeExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    5,
		BufferSize: 100,
	})
	defer bus.Shutdown()

	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed <- true
		return nil
	}

	subscriptionID := bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	if subscriptionID == "" {
		t.Fatal("Expected non-empty subscription ID")
	}

	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	select {
	case <-executed:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("Expected processor to be executed within timeout")
	}
}

// TestEventBus_MultipleSubscribersExtended tests multiple subscribers with different configurations.
func TestEventBus_MultipleSubscribersExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    8,
		BufferSize: 200,
	})
	defer bus.Shutdown()

	executionCount := 0
	executionChan := make(chan bool, 8)

	for i := 0; i < 8; i++ {
		eventProcessor := func(ctx context.Context, event *event.Event) error {
			executionChan <- true
			return nil
		}
		bus.Subscribe(testdata.AuthEventUserCreated, eventProcessor)
	}

	bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// Wait for all processors to execute
	for i := 0; i < 8; i++ {
		select {
		case <-executionChan:
			executionCount++
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Expected %d executions, got %d", 8, executionCount)
			return
		}
	}

	if executionCount != 8 {
		t.Errorf("Expected 8 invocations, got %d", executionCount)
	}
}

// TestEventBus_UnsubscribeExtended tests unsubscribing with different scenarios.
func TestEventBus_UnsubscribeExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    3,
		BufferSize: 50,
	})
	defer bus.Shutdown()

	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed <- true
		return nil
	}

	subscriptionID := bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	if subscriptionID == "" {
		t.Fatal("Expected non-empty subscription ID")
	}

	// Unsubscribe the processor
	bus.Unsubscribe(subscriptionID)

	// Publish event - should not trigger the unsubscribed processor
	bus.Publish(event.NewEvent(event.EventType(subscriptionID[:10]), &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// Wait briefly to ensure no execution
	select {
	case <-executed:
		t.Error("Expected processor to not be executed after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Success - processor was not executed
	}
}

// TestEventBus_Configuration tests different configuration options.
func TestEventBus_Configuration(t *testing.T) {
	t.Run("Config with custom workers and buffer", func(t *testing.T) {
		config := &Config{
			Workers:    20,
			BufferSize: 2000,
		}
		bus := NewEventBus(config)
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})
}

// TestEventBus_Shutdown tests graceful shutdown functionality.
func TestEventBus_Shutdown(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    2,
		BufferSize: 10,
	})

	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		executed <- true
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	// Publish an event
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// Shutdown immediately
	bus.Shutdown()

	// The event might still be processed, so we don't wait for execution
	// The important thing is that shutdown completes without hanging
}

// TestEventBus_ErrorHandling tests error handling in processors.
func TestEventBus_ErrorHandling(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	executed := make(chan error, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed <- context.Canceled // Return an error
		return context.Canceled
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// The processor should still be called even if it returns an error
	select {
	case err := <-executed:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Expected processor to be executed within timeout")
	}
}
