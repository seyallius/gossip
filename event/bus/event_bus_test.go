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
		t.Error("Handler was not executed synchronously")
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
	receivedMetadata := make(map[string]interface{})
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
