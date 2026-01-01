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

// TestConfig_DefaultConfig tests the default configuration.
func TestConfig_DefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Driver != "memory" {
		t.Errorf("Expected Driver to be 'memory', got %s", config.Driver)
	}

	if config.Workers != 10 {
		t.Errorf("Expected Workers to be 10, got %d", config.Workers)
	}

	if config.BufferSize != 1000 {
		t.Errorf("Expected BufferSize to be 1000, got %d", config.BufferSize)
	}

	if config.RedisAddr != "" {
		t.Errorf("Expected RedisAddr to be empty, got %s", config.RedisAddr)
	}

	if config.RedisPwd != "" {
		t.Errorf("Expected RedisPwd to be empty, got %s", config.RedisPwd)
	}

	if config.RedisDB != 0 {
		t.Errorf("Expected RedisDB to be 0, got %d", config.RedisDB)
	}
}

// TestConfig_CustomConfig tests custom configuration options.
func TestConfig_CustomConfig(t *testing.T) {
	t.Run("Custom memory config", func(t *testing.T) {
		config := &Config{
			Driver:     "memory",
			Workers:    20,
			BufferSize: 2000,
			RedisAddr:  "localhost:6379",
			RedisPwd:   "password",
			RedisDB:    1,
		}

		if config.Driver != "memory" {
			t.Errorf("Expected Driver to be 'memory', got %s", config.Driver)
		}
		if config.Workers != 20 {
			t.Errorf("Expected Workers to be 20, got %d", config.Workers)
		}
		if config.BufferSize != 2000 {
			t.Errorf("Expected BufferSize to be 2000, got %d", config.BufferSize)
		}
		if config.RedisAddr != "localhost:6379" {
			t.Errorf("Expected RedisAddr to be 'localhost:6379', got %s", config.RedisAddr)
		}
		if config.RedisPwd != "password" {
			t.Errorf("Expected RedisPwd to be 'password', got %s", config.RedisPwd)
		}
		if config.RedisDB != 1 {
			t.Errorf("Expected RedisDB to be 1, got %d", config.RedisDB)
		}
	})

	t.Run("Custom redis config", func(t *testing.T) {
		config := &Config{
			Driver:     "redis",
			Workers:    5,
			BufferSize: 500,
			RedisAddr:  "192.168.1.100:6379",
			RedisPwd:   "mysecret",
			RedisDB:    2,
		}

		if config.Driver != "redis" {
			t.Errorf("Expected Driver to be 'redis', got %s", config.Driver)
		}
		if config.Workers != 5 {
			t.Errorf("Expected Workers to be 5, got %d", config.Workers)
		}
		if config.BufferSize != 500 {
			t.Errorf("Expected BufferSize to be 500, got %d", config.BufferSize)
		}
		if config.RedisAddr != "192.168.1.100:6379" {
			t.Errorf("Expected RedisAddr to be '192.168.1.100:6379', got %s", config.RedisAddr)
		}
		if config.RedisPwd != "mysecret" {
			t.Errorf("Expected RedisPwd to be 'mysecret', got %s", config.RedisPwd)
		}
		if config.RedisDB != 2 {
			t.Errorf("Expected RedisDB to be 2, got %d", config.RedisDB)
		}
	})
}

// TestConfig_NewEventBusWithConfig tests creating event buses with different configurations.
func TestConfig_NewEventBusWithConfig(t *testing.T) {
	t.Run("EventBus with memory config", func(t *testing.T) {
		config := &Config{
			Driver:     "memory",
			Workers:    8,
			BufferSize: 800,
		}
		bus := NewEventBus(config)
		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
		defer bus.Shutdown()
	})

	t.Run("EventBus with redis config", func(t *testing.T) {
		config := &Config{
			Driver:     "redis",
			RedisAddr:  "localhost:6379",
			RedisPwd:   "",
			RedisDB:    0,
			Workers:    6,
			BufferSize: 600,
		}
		bus := NewEventBus(config)
		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
		defer bus.Shutdown()
	})

	t.Run("EventBus with minimal config", func(t *testing.T) {
		config := &Config{
			Driver: "memory",
		}
		bus := NewEventBus(config)
		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
		defer bus.Shutdown()
	})
}

// TestConfig_ConfigValidation tests configuration validation (implicit through usage).
func TestConfig_ConfigValidation(t *testing.T) {
	t.Run("Config with zero workers", func(t *testing.T) {
		config := &Config{
			Driver:     "memory",
			Workers:    0,
			BufferSize: 100,
		}
		bus := NewEventBus(config)
		if bus == nil {
			t.Fatal("Expected non-nil EventBus even with 0 workers")
		}
		defer bus.Shutdown()
	})

	t.Run("Config with zero buffer size", func(t *testing.T) {
		config := &Config{
			Driver:     "memory",
			Workers:    5,
			BufferSize: 0,
		}
		bus := NewEventBus(config)
		if bus == nil {
			t.Fatal("Expected non-nil EventBus even with 0 buffer size")
		}
		defer bus.Shutdown()
	})

	t.Run("Config with empty redis address", func(t *testing.T) {
		config := &Config{
			Driver:     "redis",
			RedisAddr:  "",
			RedisPwd:   "",
			RedisDB:    0,
			Workers:    5,
			BufferSize: 100,
		}
		bus := NewEventBus(config)
		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
		defer bus.Shutdown()
	})
}

// TestConfig_ConfigurationCombinations tests various configuration combinations.
func TestConfig_ConfigurationCombinations(t *testing.T) {
	configurations := []Config{
		{
			Driver:     "memory",
			Workers:    1,
			BufferSize: 1,
		},
		{
			Driver:     "memory",
			Workers:    100,
			BufferSize: 10000,
		},
		{
			Driver:     "redis",
			RedisAddr:  "localhost:6379",
			RedisPwd:   "password",
			RedisDB:    15,
			Workers:    1,
			BufferSize: 1,
		},
		{
			Driver:     "redis",
			RedisAddr:  "10.0.0.1:6380",
			RedisPwd:   "complex_password_123",
			RedisDB:    5,
			Workers:    50,
			BufferSize: 5000,
		},
	}

	for i, config := range configurations {
		t.Run("Config combination "+string(rune(i+'0')), func(t *testing.T) {
			bus := NewEventBus(&config)
			if bus == nil {
				t.Fatalf("Expected non-nil EventBus for config %d", i)
			}
			defer bus.Shutdown()
		})
	}
}

// TestConfig_ConfigEquality tests configuration equality (by checking values).
func TestConfig_ConfigEquality(t *testing.T) {
	config1 := &Config{
		Driver:     "memory",
		Workers:    10,
		BufferSize: 1000,
		RedisAddr:  "localhost:6379",
		RedisPwd:   "password",
		RedisDB:    1,
	}

	config2 := &Config{
		Driver:     "memory",
		Workers:    10,
		BufferSize: 1000,
		RedisAddr:  "localhost:6379",
		RedisPwd:   "password",
		RedisDB:    1,
	}

	// Check that all fields match
	if config1.Driver != config2.Driver {
		t.Errorf("Expected Driver to match: %s vs %s", config1.Driver, config2.Driver)
	}
	if config1.Workers != config2.Workers {
		t.Errorf("Expected Workers to match: %d vs %d", config1.Workers, config2.Workers)
	}
	if config1.BufferSize != config2.BufferSize {
		t.Errorf("Expected BufferSize to match: %d vs %d", config1.BufferSize, config2.BufferSize)
	}
	if config1.RedisAddr != config2.RedisAddr {
		t.Errorf("Expected RedisAddr to match: %s vs %s", config1.RedisAddr, config2.RedisAddr)
	}
	if config1.RedisPwd != config2.RedisPwd {
		t.Errorf("Expected RedisPwd to match: %s vs %s", config1.RedisPwd, config2.RedisPwd)
	}
	if config1.RedisDB != config2.RedisDB {
		t.Errorf("Expected RedisDB to match: %d vs %d", config1.RedisDB, config2.RedisDB)
	}
}

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

	time.Sleep(time.Millisecond) // or else there'll be no time for publishing before shutdown
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

// TestEventBus_NewEventBus tests creating a new event bus with different configurations.
func TestEventBus_NewEventBus(t *testing.T) {
	t.Run("NewEventBus with nil config uses default", func(t *testing.T) {
		bus := NewEventBus(nil)
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})

	t.Run("NewEventBus with memory driver", func(t *testing.T) {
		config := &Config{
			Driver:     "memory",
			Workers:    5,
			BufferSize: 100,
		}
		bus := NewEventBus(config)
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})

	t.Run("NewEventBus with redis driver", func(t *testing.T) {
		config := &Config{
			Driver:     "redis",
			RedisAddr:  "localhost:6379",
			RedisPwd:   "",
			RedisDB:    0,
			Workers:    5,
			BufferSize: 100,
		}
		bus := NewEventBus(config)
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})

	t.Run("NewEventBus with default config", func(t *testing.T) {
		bus := NewEventBus(DefaultConfig())
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})
}

// TestEventBus_PublishSubscribeExtended tests additional publish/subscribe functionality.
func TestEventBus_PublishSubscribeExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Driver:     "memory",
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
		Driver:     "memory",
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
		Driver:     "memory",
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
			Driver:     "memory",
			Workers:    20,
			BufferSize: 2000,
		}
		bus := NewEventBus(config)
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})

	t.Run("Config with redis settings", func(t *testing.T) {
		config := &Config{
			Driver:     "redis",
			RedisAddr:  "localhost:6379",
			RedisPwd:   "password",
			RedisDB:    1,
			Workers:    10,
			BufferSize: 1000,
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
		Driver:     "memory",
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
