// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package middleware

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -------------------------------------------- Retry Middleware Tests --------------------------------------------

func TestWithRetry_SucceedsOnFirstAttempt(t *testing.T) {
	attempts := int32(0)

	processor := func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&attempts, 1)
		return nil
	}

	wrappedHandler := WithRetry(3, 10*time.Millisecond)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "Should succeed on first attempt")
}

func TestWithRetry_RetriesUntilSuccess(t *testing.T) {
	attempts := int32(0)

	processor := func(ctx context.Context, e *event.Event) error {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	wrappedHandler := WithRetry(3, 10*time.Millisecond)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts), "Should retry until success")
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	attempts := int32(0)
	expectedErr := errors.New("persistent error")

	processor := func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&attempts, 1)
		return expectedErr
	}

	wrappedHandler := WithRetry(3, 10*time.Millisecond)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, int32(4), atomic.LoadInt32(&attempts), "Initial attempt + 3 retries")
}

func TestWithRetry_ExponentialBackoff(t *testing.T) {
	attempts := make([]time.Time, 0)

	processor := func(ctx context.Context, e *event.Event) error {
		attempts = append(attempts, time.Now())
		return errors.New("error")
	}

	wrappedHandler := WithRetry(2, 50*time.Millisecond)(processor)

	_ = wrappedHandler(context.Background(), event.NewEvent("test", nil))

	require.Len(t, attempts, 3, "Should have 3 attempts total")

	// Check backoff intervals
	firstDelay := attempts[1].Sub(attempts[0])
	secondDelay := attempts[2].Sub(attempts[1])

	assert.GreaterOrEqual(t, firstDelay, 50*time.Millisecond, "First retry delay")
	assert.GreaterOrEqual(t, secondDelay, 100*time.Millisecond, "Second retry delay (doubled)")
}

func TestWithRetry_RespectsContext(t *testing.T) {
	attempts := int32(0)

	processor := func(ctx context.Context, e *event.Event) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("error")
	}

	wrappedHandler := WithRetry(10, 100*time.Millisecond)(processor)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := wrappedHandler(ctx, event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.Less(t, atomic.LoadInt32(&attempts), int32(11), "Should stop retrying when context is cancelled")
}

// -------------------------------------------- Timeout Middleware Tests --------------------------------------------

func TestWithTimeout_CompletesBeforeTimeout(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		time.Sleep(10 * time.Millisecond)
		executed = true
		return nil
	}

	wrappedHandler := WithTimeout(100 * time.Millisecond)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestWithTimeout_ExceedsTimeout(t *testing.T) {
	processor := func(ctx context.Context, e *event.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	wrappedHandler := WithTimeout(50 * time.Millisecond)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWithTimeout_HandlerError(t *testing.T) {
	expectedErr := errors.New("processor error")

	processor := func(ctx context.Context, e *event.Event) error {
		return expectedErr
	}

	wrappedHandler := WithTimeout(100 * time.Millisecond)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// -------------------------------------------- Recovery Middleware Tests --------------------------------------------

func TestWithRecovery_HandlerSucceeds(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	}

	wrappedHandler := WithRecovery()(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestWithRecovery_HandlerPanics(t *testing.T) {
	processor := func(ctx context.Context, e *event.Event) error {
		panic("something went wrong")
	}

	wrappedHandler := WithRecovery()(processor)

	// Should not panic
	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err, "Recovery should return nil after catching panic")
}

func TestWithRecovery_HandlerPanicsWithError(t *testing.T) {
	panicErr := errors.New("panic error")

	processor := func(ctx context.Context, e *event.Event) error {
		panic(panicErr)
	}

	wrappedHandler := WithRecovery()(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err, "Recovery should catch panic and return nil")
}

// -------------------------------------------- Logging Middleware Tests --------------------------------------------

func TestWithLogging_Success(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		time.Sleep(10 * time.Millisecond)
		executed = true
		return nil
	}

	wrappedHandler := WithLogging("", func(message string) { log.Println(message) })(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestWithLogging_Failure(t *testing.T) {
	expectedErr := errors.New("processor failed")

	processor := func(ctx context.Context, e *event.Event) error {
		return expectedErr
	}

	wrappedHandler := WithLogging("", func(message string) { log.Println(message) })(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// -------------------------------------------- Chain Middleware Tests --------------------------------------------

func TestChain_SingleMiddleware(t *testing.T) {
	attempts := int32(0)

	processor := func(ctx context.Context, e *event.Event) error {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			return errors.New("error")
		}
		return nil
	}

	wrappedHandler := Chain(
		WithRetry(2, 10*time.Millisecond),
	)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

func TestChain_MultipleMiddlewares(t *testing.T) {
	attempts := int32(0)

	processor := func(ctx context.Context, e *event.Event) error {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			return errors.New("error")
		}
		time.Sleep(20 * time.Millisecond)
		return nil
	}

	wrappedHandler := Chain(
		WithRecovery(),
		WithRetry(2, 10*time.Millisecond),
		WithTimeout(500*time.Millisecond),
		WithLogging("", func(message string) {
			log.Println(message)
		}),
	)(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
}

func TestChain_OrderMatters(t *testing.T) {
	// Recovery should be outermost to catch panics from other middleware
	processor := func(ctx context.Context, e *event.Event) error {
		panic("test panic")
	}

	// Recovery first - should catch panic
	wrappedHandler1 := Chain(
		WithRecovery(),
		WithLogging("", func(message string) {
			log.Println(message)
		}),
	)(processor)

	err1 := wrappedHandler1(context.Background(), event.NewEvent("test", nil))
	assert.NoError(t, err1, "Recovery should catch panic")

	// If logging was first, panic would escape (but WithRecovery still catches it here)
	// This test demonstrates the importance of middleware order
}

func TestChain_EmptyChain(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	}

	wrappedHandler := Chain()(processor)

	err := wrappedHandler(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

// -------------------------------------------- Integration Tests --------------------------------------------

func TestMiddleware_WithEventBus(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	attempts := int32(0)

	processor := func(ctx context.Context, e *event.Event) error {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	wrappedHandler := Chain(
		WithRetry(3, 10*time.Millisecond),
		WithTimeout(500*time.Millisecond),
	)(processor)

	eventBus.Subscribe("test.event", wrappedHandler)

	eventBus.Publish(event.NewEvent("test.event", nil))
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestMiddleware_MultipleHandlersWithDifferentMiddleware(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	var mu sync.Mutex
	processor1Called := false
	processor2Called := false

	// Handler 1 with retry
	processor1 := WithRetry(2, 10*time.Millisecond)(func(ctx context.Context, e *event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		processor1Called = true
		return nil
	})

	// Handler 2 with timeout
	processor2 := WithTimeout(100 * time.Millisecond)(func(ctx context.Context, e *event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		processor2Called = true
		return nil
	})

	eventBus.Subscribe("test.event", processor1)
	eventBus.Subscribe("test.event", processor2)

	eventBus.Publish(event.NewEvent("test.event", nil))
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, processor1Called)
	assert.True(t, processor2Called)
}
