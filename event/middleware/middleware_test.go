// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package middleware

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/bus"
	goserr "github.com/seyallius/gossip/v2/event/errors"
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

	wrappedProcessor := WithRetry(3, 10*time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

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

	wrappedProcessor := WithRetry(3, 10*time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

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

	wrappedProcessor := WithRetry(3, 10*time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

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

	wrappedProcessor := WithRetry(2, 50*time.Millisecond)(processor)

	_ = wrappedProcessor(context.Background(), event.NewEvent("test", nil))

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

	wrappedProcessor := WithRetry(10, 100*time.Millisecond)(processor)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := wrappedProcessor(ctx, event.NewEvent("test", nil))

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

	wrappedProcessor := WithTimeout(100 * time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestWithTimeout_ExceedsTimeout(t *testing.T) {
	processor := func(ctx context.Context, e *event.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	wrappedProcessor := WithTimeout(50 * time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWithTimeout_ProcessorError(t *testing.T) {
	expectedErr := errors.New("processor error")

	processor := func(ctx context.Context, e *event.Event) error {
		return expectedErr
	}

	wrappedProcessor := WithTimeout(100 * time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// -------------------------------------------- Recovery Middleware Tests --------------------------------------------

func TestWithRecovery_ProcessorSucceeds(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	}

	wrappedProcessor := WithRecovery()(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

// TestWithRecovery_ProcessorPanics pins the recovered panic being reported as a failure. It used to
// return nil, which made a panicking handler indistinguishable from a clean run.
func TestWithRecovery_ProcessorPanics(t *testing.T) {
	processor := func(ctx context.Context, e *event.Event) error {
		panic("something went wrong")
	}

	wrappedProcessor := WithRecovery()(processor)

	// Should not panic
	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	require.Error(t, err, "Recovery should report the panic as an error")
	assert.True(t, goserr.IsProcessing(err), "expected a ProcessingError, got %T", err)
	assert.Contains(t, err.Error(), "something went wrong", "the panic value should reach the caller")
}

// TestWithRecovery_ProcessorPanicsWithError covers a panic carrying an error value, which stays
// reachable through the returned error's chain.
func TestWithRecovery_ProcessorPanicsWithError(t *testing.T) {
	panicErr := errors.New("panic error")

	processor := func(ctx context.Context, e *event.Event) error {
		panic(panicErr)
	}

	wrappedProcessor := WithRecovery()(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	require.Error(t, err, "Recovery should report the panic as an error")
	assert.ErrorIs(t, err, panicErr, "the panicked error should stay in the chain")
}

// TestWithRecovery_PanicIsRetried is the consequence of reporting the panic: the middlewares wrapped
// around WithRecovery finally see the failure. A swallowed panic left WithRetry with nothing to do.
func TestWithRecovery_PanicIsRetried(t *testing.T) {
	attempts := 0
	processor := func(ctx context.Context, e *event.Event) error {
		attempts++
		if attempts < 3 {
			panic("transient")
		}
		return nil
	}

	wrappedProcessor := Chain(WithRetry(3, time.Millisecond), WithRecovery())(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "expected the panicking attempts to be retried")
}

// -------------------------------------------- Logging Middleware Tests --------------------------------------------

func TestWithLogging_Success(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		time.Sleep(10 * time.Millisecond)
		executed = true
		return nil
	}

	wrappedProcessor := WithLogging("", func(message string) { log.Println(message) })(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestWithLogging_Failure(t *testing.T) {
	expectedErr := errors.New("processor failed")

	processor := func(ctx context.Context, e *event.Event) error {
		return expectedErr
	}

	wrappedProcessor := WithLogging("", func(message string) { log.Println(message) })(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// TestWithLogging_MessageDoesNotAccumulate covers the log line itself. WithLogging assigned the
// formatted line back to the captured msg, so every call prefixed the previous line onto the next
// one: the message grew without bound for the life of the wrapped processor.
func TestWithLogging_MessageDoesNotAccumulate(t *testing.T) {
	var logged []string
	processor := func(ctx context.Context, e *event.Event) error { return nil }

	wrappedProcessor := WithLogging("PREFIX", func(message string) {
		logged = append(logged, message)
	})(processor)

	for i := 0; i < 4; i++ {
		require.NoError(t, wrappedProcessor(context.Background(), event.NewEvent("test", nil)))
	}

	require.Len(t, logged, 4)
	for i, message := range logged {
		assert.Equal(t, 1, strings.Count(message, "[Middleware]"), "line %d carries earlier lines: %q", i, message)
	}
}

// TestWithLogging_ConcurrentEvents covers the other half of the same bug: the shared msg string was
// written by every invocation, so concurrent events raced on it. Run under -race.
func TestWithLogging_ConcurrentEvents(t *testing.T) {
	wrappedProcessor := WithLogging("PREFIX", func(message string) {})(func(ctx context.Context, e *event.Event) error {
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_ = wrappedProcessor(context.Background(), event.NewEvent("test", nil))
		}()
	}
	wg.Wait()
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

	wrappedProcessor := Chain(
		WithRetry(2, 10*time.Millisecond),
	)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

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

	wrappedProcessor := Chain(
		WithRecovery(),
		WithRetry(2, 10*time.Millisecond),
		WithTimeout(500*time.Millisecond),
		WithLogging("", func(message string) {
			log.Println(message)
		}),
	)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
}

// TestChain_OrderMatters shows why WithRecovery belongs innermost. Either way the panic is caught
// and the caller gets an error, but only the inner placement lets the middleware around it observe
// the failure: an outer WithRecovery catches the panic after it has already unwound through them.
func TestChain_OrderMatters(t *testing.T) {
	processor := func(ctx context.Context, e *event.Event) error {
		panic("test panic")
	}

	var outermostRecoveryLogs []string
	outermostRecovery := Chain(
		WithRecovery(),
		WithLogging("", func(message string) {
			outermostRecoveryLogs = append(outermostRecoveryLogs, message)
		}),
	)(processor)

	err := outermostRecovery(context.Background(), event.NewEvent("test", nil))
	require.Error(t, err, "Recovery should catch the panic and report it")
	assert.Empty(t, outermostRecoveryLogs, "the panic unwound past the logger before recovery saw it")

	var innermostRecoveryLogs []string
	innermostRecovery := Chain(
		WithLogging("", func(message string) {
			innermostRecoveryLogs = append(innermostRecoveryLogs, message)
		}),
		WithRecovery(),
	)(processor)

	err = innermostRecovery(context.Background(), event.NewEvent("test", nil))
	require.Error(t, err, "Recovery should catch the panic and report it")
	require.Len(t, innermostRecoveryLogs, 1, "the logger should see the recovered failure")
	assert.Contains(t, innermostRecoveryLogs[0], "failed")
}

func TestChain_EmptyChain(t *testing.T) {
	executed := false

	processor := func(ctx context.Context, e *event.Event) error {
		executed = true
		return nil
	}

	wrappedProcessor := Chain()(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

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

	wrappedProcessor := Chain(
		WithRetry(3, 10*time.Millisecond),
		WithTimeout(500*time.Millisecond),
	)(processor)

	eventBus.Subscribe("test.event", wrappedProcessor)

	eventBus.Publish(event.NewEvent("test.event", nil))
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestMiddleware_MultipleProcessorsWithDifferentMiddleware(t *testing.T) {
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	var mu sync.Mutex
	processor1Called := false
	processor2Called := false

	// Processor 1 with retry
	processor1 := WithRetry(2, 10*time.Millisecond)(func(ctx context.Context, e *event.Event) error {
		mu.Lock()
		defer mu.Unlock()
		processor1Called = true
		return nil
	})

	// Processor 2 with timeout
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

// ------------------------------------- Degenerate Argument Regressions ---------------------------- //

// TestWithRetry_NegativeRetriesStillRunsOnce covers a negative retry count. The attempt loop ran
// while attempt <= maxRetries, so a negative count skipped the body entirely: the processor never
// ran and the middleware returned the nil error it started with.
func TestWithRetry_NegativeRetriesStillRunsOnce(t *testing.T) {
	expectedErr := errors.New("processor failed")
	attempts := 0
	processor := func(ctx context.Context, e *event.Event) error {
		attempts++
		return expectedErr
	}

	wrappedProcessor := WithRetry(-1, time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	assert.Equal(t, 1, attempts, "expected exactly one attempt")
	assert.ErrorIs(t, err, expectedErr, "expected the processor's error, not a silent success")
}

// TestWithTimeout_NonPositiveTimeoutDoesNotFailEveryEvent covers a zero or negative timeout.
// context.WithTimeout hands back an already-expired context for those, so every event failed
// immediately while its processor kept running, detached, in the background.
func TestWithTimeout_NonPositiveTimeoutDoesNotFailEveryEvent(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		executed := false
		processor := func(ctx context.Context, e *event.Event) error {
			executed = true
			return nil
		}

		wrappedProcessor := WithTimeout(timeout)(processor)

		err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

		assert.NoError(t, err, "timeout %v should mean no timeout", timeout)
		assert.True(t, executed, "timeout %v should still run the processor", timeout)
	}
}

// TestWithTimeout_ReportsATimeoutError covers the error an expired deadline produces: the library's
// own TimeoutError, still carrying context.DeadlineExceeded for callers who check for that.
func TestWithTimeout_ReportsATimeoutError(t *testing.T) {
	processor := func(ctx context.Context, e *event.Event) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	wrappedProcessor := WithTimeout(20 * time.Millisecond)(processor)

	err := wrappedProcessor(context.Background(), event.NewEvent("test", nil))

	require.Error(t, err)
	assert.True(t, goserr.IsTimeout(err), "expected a TimeoutError, got %T: %v", err, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "expected the context error to stay in the chain")
}

// TestWithTimeout_PassesThroughCancellation covers a cancelled parent, which is a cancellation rather
// than a timeout and must not be relabelled as one.
func TestWithTimeout_PassesThroughCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := func(ctx context.Context, e *event.Event) error {
		cancel()
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	wrappedProcessor := WithTimeout(5 * time.Second)(processor)

	err := wrappedProcessor(ctx, event.NewEvent("test", nil))

	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, goserr.IsTimeout(err), "a cancelled parent is not a timeout")
}
