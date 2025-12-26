// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package filter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
	"github.com/seyallius/gossip/event/middleware"
	"github.com/stretchr/testify/assert"
)

// -------------------------------------------- Filter Tests --------------------------------------------

func TestNewFilteredProcessor_PassesFilter(t *testing.T) {
	executed := false

	filter := func(e *event.Event) bool {
		return true
	}

	handler := func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	}

	filteredProcessor := NewFilteredProcessor(filter, handler)

	err := filteredProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed, "Processor should execute when filter returns true")
}

func TestNewFilteredProcessor_FailsFilter(t *testing.T) {
	executed := false

	filter := func(e *event.Event) bool {
		return false
	}

	handler := func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	}

	filteredProcessor := NewFilteredProcessor(filter, handler)

	err := filteredProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.False(t, executed, "Processor should NOT execute when filter returns false")
}

func TestFilterByMetadata_Matches(t *testing.T) {
	executed := false

	filter := FilterByMetadata("priority", "high")

	handler := NewFilteredProcessor(filter, func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	})

	evt := event.NewEvent("test", nil).WithMetadata("priority", "high")

	err := handler(context.Background(), evt)

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestFilterByMetadata_NoMatch(t *testing.T) {
	executed := false

	filter := FilterByMetadata("priority", "high")

	handler := NewFilteredProcessor(filter, func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	})

	evt := event.NewEvent("test", nil).WithMetadata("priority", "low")

	err := handler(context.Background(), evt)

	assert.NoError(t, err)
	assert.False(t, executed)
}

func TestFilterByMetadata_MissingKey(t *testing.T) {
	executed := false

	filter := FilterByMetadata("priority", "high")

	handler := NewFilteredProcessor(filter, func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	})

	evt := event.NewEvent("test", nil)

	err := handler(context.Background(), evt)

	assert.NoError(t, err)
	assert.False(t, executed, "Should not execute when metadata key is missing")
}

func TestFilterByMetadataExists_KeyExists(t *testing.T) {
	executed := false

	filter := FilterByMetadataExists("request_id")

	handler := NewFilteredProcessor(filter, func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	})

	evt := event.NewEvent("test", nil).WithMetadata("request_id", "req-123")

	err := handler(context.Background(), evt)

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestFilterByMetadataExists_KeyMissing(t *testing.T) {
	executed := false

	filter := FilterByMetadataExists("request_id")

	handler := NewFilteredProcessor(filter, func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	})

	evt := event.NewEvent("test", nil)

	err := handler(context.Background(), evt)

	assert.NoError(t, err)
	assert.False(t, executed)
}

// -------------------------------------------- Combinator Tests --------------------------------------------

func TestAnd_AllTrue(t *testing.T) {
	filter1 := func(e *event.Event) bool { return true }
	filter2 := func(e *event.Event) bool { return true }
	filter3 := func(e *event.Event) bool { return true }

	combinedFilter := And(filter1, filter2, filter3)

	result := combinedFilter(event.NewEvent("test", nil))
	assert.True(t, result, "AND should return true when all filters return true")
}

func TestAnd_OneFalse(t *testing.T) {
	filter1 := func(e *event.Event) bool { return true }
	filter2 := func(e *event.Event) bool { return false }
	filter3 := func(e *event.Event) bool { return true }

	combinedFilter := And(filter1, filter2, filter3)

	result := combinedFilter(event.NewEvent("test", nil))
	assert.False(t, result, "AND should return false when any filter returns false")
}

func TestAnd_AllFalse(t *testing.T) {
	filter1 := func(e *event.Event) bool { return false }
	filter2 := func(e *event.Event) bool { return false }

	combinedFilter := And(filter1, filter2)

	result := combinedFilter(event.NewEvent("test", nil))
	assert.False(t, result)
}

func TestAnd_EmptyFilters(t *testing.T) {
	combinedFilter := And()

	result := combinedFilter(event.NewEvent("test", nil))
	assert.True(t, result, "AND with no filters should return true")
}

func TestOr_AllTrue(t *testing.T) {
	filter1 := func(e *event.Event) bool { return true }
	filter2 := func(e *event.Event) bool { return true }

	combinedFilter := Or(filter1, filter2)

	result := combinedFilter(event.NewEvent("test", nil))
	assert.True(t, result)
}

func TestOr_OneTrue(t *testing.T) {
	filter1 := func(e *event.Event) bool { return false }
	filter2 := func(e *event.Event) bool { return true }
	filter3 := func(e *event.Event) bool { return false }

	combinedFilter := Or(filter1, filter2, filter3)

	result := combinedFilter(event.NewEvent("test", nil))
	assert.True(t, result, "OR should return true when any filter returns true")
}

func TestOr_AllFalse(t *testing.T) {
	filter1 := func(e *event.Event) bool { return false }
	filter2 := func(e *event.Event) bool { return false }

	combinedFilter := Or(filter1, filter2)

	result := combinedFilter(event.NewEvent("test", nil))
	assert.False(t, result, "OR should return false when all filters return false")
}

func TestOr_EmptyFilters(t *testing.T) {
	combinedFilter := Or()

	result := combinedFilter(event.NewEvent("test", nil))
	assert.False(t, result, "OR with no filters should return false")
}

func TestNot_True(t *testing.T) {
	filter := func(e *event.Event) bool { return true }

	negatedFilter := Not(filter)

	result := negatedFilter(event.NewEvent("test", nil))
	assert.False(t, result, "NOT should invert true to false")
}

func TestNot_False(t *testing.T) {
	filter := func(e *event.Event) bool { return false }

	negatedFilter := Not(filter)

	result := negatedFilter(event.NewEvent("test", nil))
	assert.True(t, result, "NOT should invert false to true")
}

// -------------------------------------------- Complex Filter Tests --------------------------------------------

func TestComplexFilter_AndOrCombination(t *testing.T) {
	// (priority = high OR source = api) AND status = active

	priorityHigh := FilterByMetadata("priority", "high")
	sourceAPI := FilterByMetadata("source", "api")
	statusActive := FilterByMetadata("status", "active")

	complexFilter := And(
		Or(priorityHigh, sourceAPI),
		statusActive,
	)

	// Test case 1: priority=high AND status=active -> true
	evt1 := event.NewEvent("test", nil).
		WithMetadata("priority", "high").
		WithMetadata("status", "active")
	assert.True(t, complexFilter(evt1))

	// Test case 2: source=api AND status=active -> true
	evt2 := event.NewEvent("test", nil).
		WithMetadata("source", "api").
		WithMetadata("status", "active")
	assert.True(t, complexFilter(evt2))

	// Test case 3: priority=high but status=inactive -> false
	evt3 := event.NewEvent("test", nil).
		WithMetadata("priority", "high").
		WithMetadata("status", "inactive")
	assert.False(t, complexFilter(evt3))

	// Test case 4: priority=low AND source=web AND status=active -> false
	evt4 := event.NewEvent("test", nil).
		WithMetadata("priority", "low").
		WithMetadata("source", "web").
		WithMetadata("status", "active")
	assert.False(t, complexFilter(evt4))
}

func TestComplexFilter_NestedLogic(t *testing.T) {
	// NOT(priority = low) AND (source = api OR source = web)

	priorityLow := FilterByMetadata("priority", "low")
	sourceAPI := FilterByMetadata("source", "api")
	sourceWeb := FilterByMetadata("source", "web")

	complexFilter := And(
		Not(priorityLow),
		Or(sourceAPI, sourceWeb),
	)

	// priority=high AND source=api -> true
	evt1 := event.NewEvent("test", nil).
		WithMetadata("priority", "high").
		WithMetadata("source", "api")
	assert.True(t, complexFilter(evt1))

	// priority=low AND source=api -> false (due to NOT)
	evt2 := event.NewEvent("test", nil).
		WithMetadata("priority", "low").
		WithMetadata("source", "api")
	assert.False(t, complexFilter(evt2))

	// priority=high AND source=mobile -> false (source not api or web)
	evt3 := event.NewEvent("test", nil).
		WithMetadata("priority", "high").
		WithMetadata("source", "mobile")
	assert.False(t, complexFilter(evt3))
}

// -------------------------------------------- Integration Tests --------------------------------------------

func TestFilter_WithEventBus(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	highPriorityCount := int32(0)
	lowPriorityCount := int32(0)

	highPriorityFilter := FilterByMetadata("priority", "high")
	lowPriorityFilter := FilterByMetadata("priority", "low")

	eventBus.Subscribe("test.event", NewFilteredProcessor(highPriorityFilter, func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&highPriorityCount, 1)
		return nil
	}))

	eventBus.Subscribe("test.event", NewFilteredProcessor(lowPriorityFilter, func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&lowPriorityCount, 1)
		return nil
	}))

	// Publish high priority event
	eventBus.Publish(event.NewEvent("test.event", nil).WithMetadata("priority", "high"))
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&highPriorityCount))
	assert.Equal(t, int32(0), atomic.LoadInt32(&lowPriorityCount))

	// Publish low priority event
	eventBus.Publish(event.NewEvent("test.event", nil).WithMetadata("priority", "low"))
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&highPriorityCount))
	assert.Equal(t, int32(1), atomic.LoadInt32(&lowPriorityCount))
}

func TestFilter_MultipleEventsWithDifferentFilters(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	apiCount := int32(0)
	webCount := int32(0)
	mobileCount := int32(0)

	apiFilter := FilterByMetadata("source", "api")
	webFilter := FilterByMetadata("source", "web")
	mobileFilter := FilterByMetadata("source", "mobile")

	eventBus.Subscribe("request.received", NewFilteredProcessor(apiFilter, func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&apiCount, 1)
		return nil
	}))

	eventBus.Subscribe("request.received", NewFilteredProcessor(webFilter, func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&webCount, 1)
		return nil
	}))

	eventBus.Subscribe("request.received", NewFilteredProcessor(mobileFilter, func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&mobileCount, 1)
		return nil
	}))

	// Publish events from different sources
	sources := []string{"api", "web", "mobile", "api", "web", "api"}

	for _, source := range sources {
		eventBus.Publish(event.NewEvent("request.received", nil).WithMetadata("source", source))
	}

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(3), atomic.LoadInt32(&apiCount))
	assert.Equal(t, int32(2), atomic.LoadInt32(&webCount))
	assert.Equal(t, int32(1), atomic.LoadInt32(&mobileCount))
}

func TestFilter_CombinedWithMiddleware(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	executedCount := int32(0)

	filter := FilterByMetadata("retry", "true")

	handler := NewFilteredProcessor(filter, func(ctx context.Context, e *event.Event) error {
		count := atomic.AddInt32(&executedCount, 1)
		if count < 2 {
			return assert.AnError
		}
		return nil
	})

	wrappedProcessor := middleware.WithRetry(2, 10*time.Millisecond)(handler)

	eventBus.Subscribe("test.event", wrappedProcessor)

	// Event without retry metadata - should not execute
	eventBus.Publish(event.NewEvent("test.event", nil))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&executedCount))

	// Event with retry metadata - should execute and retry
	eventBus.Publish(event.NewEvent("test.event", nil).WithMetadata("retry", "true"))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(2), atomic.LoadInt32(&executedCount))
}
