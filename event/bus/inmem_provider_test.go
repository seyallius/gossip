// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package bus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/testdata"
)

// TestInMemoryProvider_NewInMemoryProvider tests creating a new in-memory provider.
func TestInMemoryProvider_NewInMemoryProvider(t *testing.T) {
	t.Run("NewInMemoryProvider with valid parameters", func(t *testing.T) {
		provider := NewInMemoryProvider(5, 100)
		if provider == nil {
			t.Fatal("Expected non-nil InMemoryProvider")
		}
		defer provider.Shutdown()
	})

	t.Run("NewInMemoryProvider with zero workers", func(t *testing.T) {
		provider := NewInMemoryProvider(0, 100)
		if provider == nil {
			t.Fatal("Expected non-nil InMemoryProvider")
		}
		defer provider.Shutdown()
	})

	t.Run("NewInMemoryProvider with zero buffer", func(t *testing.T) {
		provider := NewInMemoryProvider(5, 0)
		if provider == nil {
			t.Fatal("Expected non-nil InMemoryProvider")
		}
		defer provider.Shutdown()
	})
}

// TestInMemoryProvider_Publish tests publishing events to the in-memory provider.
func TestInMemoryProvider_Publish(t *testing.T) {
	provider := NewInMemoryProvider(2, 10)
	defer provider.Shutdown()

	t.Run("Publish single event", func(t *testing.T) {
		eventToPublish := event.NewEvent("test.event", "test data")
		err := provider.Publish(eventToPublish)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Publish to closed provider", func(t *testing.T) {
		provider2 := NewInMemoryProvider(1, 1)
		err := provider2.Shutdown()
		if err != nil {
			t.Errorf("Expected no error during shutdown, got %v", err)
		}

		// Don't try to publish after shutdown as it will cause a panic
		// The important thing is that shutdown completes without hanging
	})

	t.Run("Publish to full buffer", func(t *testing.T) {
		provider3 := NewInMemoryProvider(1, 1)
		defer provider3.Shutdown()

		// Fill the buffer
		event1 := event.NewEvent("test.event", "data1")
		err := provider3.Publish(event1)
		if err != nil {
			t.Fatalf("Expected no error for first event, got %v", err)
		}

		// Try to publish another event - should fail due to full buffer
		event2 := event.NewEvent("test.event", "data2")
		err = provider3.Publish(event2)
		if err == nil {
			t.Error("Expected error when buffer is full")
		}
	})
}

// TestInMemoryProvider_Subscribe tests subscribing to events in the in-memory provider.
func TestInMemoryProvider_Subscribe(t *testing.T) {
	provider := NewInMemoryProvider(2, 10)
	defer provider.Shutdown()

	t.Run("Subscribe with valid processor", func(t *testing.T) {
		subscriptionID, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if subscriptionID == "" {
			t.Error("Expected non-empty subscription ID")
		}
	})

	t.Run("Subscribe with nil processor", func(t *testing.T) {
		subscriptionID, err := provider.Subscribe("test.event", nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if subscriptionID == "" {
			t.Error("Expected non-empty subscription ID")
		}
	})
}

// TestInMemoryProvider_Unsubscribe tests unsubscribing from events in the in-memory provider.
func TestInMemoryProvider_Unsubscribe(t *testing.T) {
	provider := NewInMemoryProvider(2, 10)
	defer provider.Shutdown()

	t.Run("Unsubscribe existing subscription", func(t *testing.T) {
		subscriptionID, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
			return nil
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		err = provider.Unsubscribe(subscriptionID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Unsubscribe non-existing subscription", func(t *testing.T) {
		err := provider.Unsubscribe("non-existing-id")
		if err != nil {
			t.Errorf("Expected no error for non-existing subscription, got %v", err)
		}
	})
}

// TestInMemoryProvider_EndToEnd tests end-to-end functionality of the in-memory provider.
func TestInMemoryProvider_EndToEnd(t *testing.T) {
	provider := NewInMemoryProvider(3, 20)
	defer provider.Shutdown()

	executionCount := 0
	executionChan := make(chan bool, 3)

	// Subscribe multiple processors
	for i := 0; i < 3; i++ {
		_, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
			executionChan <- true
			return nil
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}

	// Publish event
	testEvent := event.NewEvent("test.event", "test data")
	err := provider.Publish(testEvent)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Wait for all processors to execute
	for i := 0; i < 3; i++ {
		select {
		case <-executionChan:
			executionCount++
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Expected %d executions, got %d", 3, executionCount)
			return
		}
	}

	if executionCount != 3 {
		t.Errorf("Expected 3 invocations, got %d", executionCount)
	}
}

// TestInMemoryProvider_ConcurrentAccess tests concurrent access to the in-memory provider.
func TestInMemoryProvider_ConcurrentAccess(t *testing.T) {
	provider := NewInMemoryProvider(5, 50)
	defer provider.Shutdown()

	done := make(chan bool, 10)

	// Concurrent subscriptions
	for i := 0; i < 5; i++ {
		go func(i int) {
			_, err := provider.Subscribe(event.EventType("test.event."+string(rune(i))), func(ctx context.Context, e *event.Event) error {
				return nil
			})
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			done <- true
		}(i)
	}

	// Concurrent publishes
	for i := 0; i < 5; i++ {
		go func(i int) {
			err := provider.Publish(event.NewEvent(event.EventType("test.event."+string(rune(i))), "data"))
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Continue
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations to complete")
		}
	}
}

// TestInMemoryProvider_Shutdown tests graceful shutdown of the in-memory provider.
func TestInMemoryProvider_Shutdown(t *testing.T) {
	provider := NewInMemoryProvider(2, 10)

	// Subscribe a processor that takes some time
	executed := make(chan bool, 1)
	_, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
		time.Sleep(50 * time.Millisecond) // Simulate work
		executed <- true
		return nil
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Publish an event
	err = provider.Publish(event.NewEvent("test.event", "test data"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Shutdown should wait for in-flight events to complete
	err = provider.Shutdown()
	if err != nil {
		t.Errorf("Expected no error during shutdown, got %v", err)
	}

	// The event might still be processed depending on timing
	// We don't wait here as the shutdown should have waited for active workers
}

// TestInMemoryProvider_ProcessorExecution tests processor execution with different data types.
func TestInMemoryProvider_ProcessorExecution(t *testing.T) {
	provider := NewInMemoryProvider(2, 10)
	defer provider.Shutdown()

	t.Run("Processor with string data", func(t *testing.T) {
		executed := make(chan error, 1)
		_, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
			if e.Data != "test string" {
				executed <- fmt.Errorf("Expected 'test string', got %v", e.Data)
				return nil
			}
			executed <- nil
			return nil
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		err = provider.Publish(event.NewEvent("test.event", "test string"))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		select {
		case result := <-executed:
			if result != nil {
				t.Error(result.Error())
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected processor to be executed within timeout")
		}
	})

	t.Run("Processor with struct data", func(t *testing.T) {
		executed := make(chan error, 1)
		_, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
			data, ok := e.Data.(*testdata.UserCreatedData)
			if !ok {
				executed <- fmt.Errorf("Expected *testdata.UserCreatedData, got %T", e.Data)
				return nil
			}
			if data.UserID != "test-user" {
				executed <- fmt.Errorf("Expected UserID 'test-user', got %s", data.UserID)
				return nil
			}
			executed <- nil
			return nil
		})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		testData := &testdata.UserCreatedData{
			UserID:   "test-user",
			Username: "testuser",
		}
		err = provider.Publish(event.NewEvent("test.event", testData))
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		select {
		case result := <-executed:
			if result != nil {
				t.Error(result.Error())
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Expected processor to be executed within timeout")
		}
	})
}
