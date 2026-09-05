// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package bus

import (
	"context"
	"testing"
	"time"

	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/testdata"
)

// TestGlobalBus tests the global bus functionality.
func TestGlobalBus(t *testing.T) {
	// Get the global bus instance
	bus1 := GetGlobalBus()
	if bus1 == nil {
		t.Fatal("Expected non-nil global bus")
	}

	// Get it again - should be the same instance
	bus2 := GetGlobalBus()
	if bus1 != bus2 {
		t.Error("Expected same global bus instance")
	}

	// Test that we can use the global bus. The processor signals without blocking and the
	// subscription is removed again, because the global bus outlives this test: a leftover processor
	// parked on a full channel would wedge a worker for every later run.
	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		select {
		case executed <- true:
		default:
		}
		return nil
	}

	subscriptionID := bus1.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	if subscriptionID == "" {
		t.Fatal("Expected non-empty subscription ID")
	}
	defer bus1.Unsubscribe(subscriptionID)

	bus1.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
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
