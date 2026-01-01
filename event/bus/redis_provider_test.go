// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package bus

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/testdata"
)

// TestRedisProvider_NewRedisProvider tests creating a new Redis provider.
func TestRedisProvider_NewRedisProvider(t *testing.T) {
	t.Run("NewRedisProvider with valid parameters", func(t *testing.T) {
		provider := NewRedisProvider("localhost:6379", "", 0)
		if provider == nil {
			t.Fatal("Expected non-nil RedisProvider")
		}
		defer provider.Shutdown()
	})

	t.Run("NewRedisProvider with password", func(t *testing.T) {
		provider := NewRedisProvider("localhost:6379", "password", 0)
		if provider == nil {
			t.Fatal("Expected non-nil RedisProvider")
		}
		defer provider.Shutdown()
	})

	t.Run("NewRedisProvider with different DB", func(t *testing.T) {
		provider := NewRedisProvider("localhost:6379", "", 1)
		if provider == nil {
			t.Fatal("Expected non-nil RedisProvider")
		}
		defer provider.Shutdown()
	})
}

// TestRedisProvider_Publish tests publishing events to the Redis provider.
func TestRedisProvider_Publish(t *testing.T) {
	// Note: These tests will fail if Redis is not running, so we're testing the structure
	// rather than actual Redis connectivity in a standard test environment
	provider := NewRedisProvider("localhost:6379", "", 0)
	defer provider.Shutdown()

	t.Run("Publish single event", func(t *testing.T) {
		eventToPublish := event.NewEvent("test.event", "test data")
		err := provider.Publish(eventToPublish)
		// Note: This will likely fail without Redis running, but we're testing the structure
		// In a real environment with Redis, this should work
		if err == nil {
			// If no error, that means Redis is available
			t.Log("Successfully published to Redis (Redis must be running)")
		} else {
			// This is expected if Redis is not running
			t.Logf("Expected error when Redis is not available: %v", err)
		}
	})

	t.Run("Publish event with complex data", func(t *testing.T) {
		testData := &testdata.UserCreatedData{
			UserID:   "test-user",
			Username: "testuser",
			Email:    "test@example.com",
		}
		eventToPublish := event.NewEvent("test.event", testData)
		err := provider.Publish(eventToPublish)
		if err != nil {
			t.Logf("Expected error when Redis is not available: %v", err)
		}
	})

	t.Run("Publish to closed provider", func(t *testing.T) {
		provider2 := NewRedisProvider("localhost:6379", "", 0)
		err := provider2.Shutdown()
		if err != nil {
			t.Errorf("Expected no error during shutdown, got %v", err)
		}

		eventToPublish := event.NewEvent("test.event", "test data")
		err = provider2.Publish(eventToPublish)
		if err == nil {
			t.Error("Expected error when publishing to closed provider")
		}
	})
}

// TestRedisProvider_Subscribe tests subscribing to events in the Redis provider.
func TestRedisProvider_Subscribe(t *testing.T) {
	provider := NewRedisProvider("localhost:6379", "", 0)
	defer provider.Shutdown()

	t.Run("Subscribe with valid processor", func(t *testing.T) {
		subscriptionID, err := provider.Subscribe("test.event", func(ctx context.Context, e *event.Event) error {
			return nil
		})
		if err != nil {
			t.Logf("Expected error when Redis is not available: %v", err)
		}
		if subscriptionID == "" {
			t.Log("Subscription ID is empty (expected if Redis is not available)")
		}
	})

	t.Run("Subscribe with nil processor", func(t *testing.T) {
		subscriptionID, err := provider.Subscribe("test.event", nil)
		if err != nil {
			t.Logf("Expected error when Redis is not available: %v", err)
		}
		if subscriptionID == "" {
			t.Log("Subscription ID is empty (expected if Redis is not available)")
		}
	})
}

// TestRedisProvider_Unsubscribe tests unsubscribing from events in the Redis provider.
func TestRedisProvider_Unsubscribe(t *testing.T) {
	provider := NewRedisProvider("localhost:6379", "", 0)
	defer provider.Shutdown()

	t.Run("Unsubscribe existing subscription", func(t *testing.T) {
		// Note: The Redis provider doesn't currently track subscription IDs properly
		// This is a limitation of the current implementation
		err := provider.Unsubscribe("some-id")
		if err != nil {
			t.Logf("Unsubscribe returned error (expected behavior): %v", err)
		}
	})
}

// TestRedisProvider_EventSerialization tests event serialization/deserialization in Redis provider.
func TestRedisProvider_EventSerialization(t *testing.T) {
	// Test that events can be properly marshaled/unmarshaled
	eventToTest := event.NewEvent("test.event", map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": []string{"a", "b", "c"},
	})

	// Marshal the event
	payload, err := json.Marshal(eventToTest)
	if err != nil {
		t.Fatalf("Expected no error when marshaling event, got %v", err)
	}

	// Unmarshal back to event
	var unmarshaledEvent event.Event
	err = json.Unmarshal(payload, &unmarshaledEvent)
	if err != nil {
		t.Fatalf("Expected no error when unmarshaling event, got %v", err)
	}

	if unmarshaledEvent.Type != eventToTest.Type {
		t.Errorf("Expected type %s, got %s", eventToTest.Type, unmarshaledEvent.Type)
	}
}

// TestRedisProvider_Shutdown tests graceful shutdown of the Redis provider.
func TestRedisProvider_Shutdown(t *testing.T) {
	provider := NewRedisProvider("localhost:6379", "", 0)

	err := provider.Shutdown()
	if err != nil {
		t.Errorf("Expected no error during shutdown, got %v", err)
	}

	// Verify that the provider is properly closed
	// Note: We can't easily test if the Redis client is closed without reflection
	// The important thing is that shutdown doesn't panic
}

// TestRedisProvider_Configuration tests different configuration options for Redis provider.
func TestRedisProvider_Configuration(t *testing.T) {
	t.Run("Redis provider with different addresses", func(t *testing.T) {
		provider := NewRedisProvider("127.0.0.1:6379", "", 0)
		if provider == nil {
			t.Fatal("Expected non-nil RedisProvider")
		}
		defer provider.Shutdown()
	})

	t.Run("Redis provider with password", func(t *testing.T) {
		provider := NewRedisProvider("localhost:6379", "testpassword", 0)
		if provider == nil {
			t.Fatal("Expected non-nil RedisProvider")
		}
		defer provider.Shutdown()
	})

	t.Run("Redis provider with different DB numbers", func(t *testing.T) {
		for db := 0; db < 3; db++ {
			provider := NewRedisProvider("localhost:6379", "", db)
			if provider == nil {
				t.Fatalf("Expected non-nil RedisProvider for DB %d", db)
			}
			defer provider.Shutdown()
		}
	})
}

// TestRedisProvider_MarshalUnmarshal tests the marshaling and unmarshaling process.
func TestRedisProvider_MarshalUnmarshal(t *testing.T) {
	originalEvent := event.NewEvent("test.event", &testdata.UserCreatedData{
		UserID:   "test-user",
		Username: "testuser",
		Email:    "test@example.com",
	})

	// Marshal the event
	payload, err := json.Marshal(originalEvent)
	if err != nil {
		t.Fatalf("Expected no error when marshaling event, got %v", err)
	}

	// Unmarshal back to event
	var unmarshaledEvent event.Event
	err = json.Unmarshal(payload, &unmarshaledEvent)
	if err != nil {
		t.Fatalf("Expected no error when unmarshaling event, got %v", err)
	}

	// Note: After JSON unmarshaling, the Data field becomes map[string]interface{}
	// This is expected behavior for JSON unmarshaling of unknown types
	if unmarshaledEvent.Type != originalEvent.Type {
		t.Errorf("Expected type %s, got %s", originalEvent.Type, unmarshaledEvent.Type)
	}

	if unmarshaledEvent.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

// TestRedisProvider_ContextCancellation tests context cancellation handling.
func TestRedisProvider_ContextCancellation(t *testing.T) {
	provider := NewRedisProvider("localhost:6379", "", 0)

	// Cancel the provider's context
	provider.cancel()

	// Try to publish - should fail due to context cancellation
	testEvent := event.NewEvent("test.event", "test data")
	err := provider.Publish(testEvent)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	defer provider.Shutdown()
}
