// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package middleware. middleware provides composable middleware for event handlers.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/seyallius/gossip/v2/event"
	goserr "github.com/seyallius/gossip/v2/event/errors"
	"github.com/seyallius/gossip/v2/event/sub"
)

// ----------------------------------------------- Types -------------------------------------------- //

// Middleware wraps a sub.EventProcessor with additional behavior.
type Middleware func(sub.EventProcessor) sub.EventProcessor

// -------------------------------------------- Public API ------------------------------------------ //

// WithRetry retries failed event processors with exponential backoff.
// It executes initial + retries, so maxRetries of 0 means a single attempt.
//
// A negative maxRetries is treated as 0. It used to make the attempt loop body unreachable, so the
// processor never ran at all and the middleware reported success for work it had not done.
func WithRetry(maxRetries int, initialDelay time.Duration) Middleware {
	if maxRetries < 0 {
		maxRetries = 0
	}
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) error {
			var err error
			delay := initialDelay

			for attempt := 0; attempt <= maxRetries; attempt++ {
				if err = next(ctx, eventToPub); err == nil {
					return nil
				}

				if attempt < maxRetries {
					log.Printf("[Middleware] Retry attempt %d/%d for event %s after error: %v", attempt+1, maxRetries, eventToPub.Type, err)

					select {
					case <-time.After(delay): // wait and exponent delay
						delay *= 2

					case <-ctx.Done(): // cancelled while waiting
						return ctx.Err()
					}
				}
			}

			return err
		}
	}
}

// WithTimeout adds a timeout to event processor execution.
//
// An expired deadline is reported as an errors.TimeoutError wrapping context.DeadlineExceeded, so
// both goserr.IsTimeout and errors.Is(err, context.DeadlineExceeded) recognise it. A cancelled
// parent context is passed through unchanged, because that is a cancellation and not a timeout.
//
// A non-positive timeout means no timeout at all: context.WithTimeout would otherwise hand back an
// already-expired context, failing every event while still leaving the processor running detached.
//
// Note that the deadline bounds how long this middleware waits, not the processor itself. A
// processor that ignores ctx keeps running in its own goroutine after the timeout is reported.
func WithTimeout(timeout time.Duration) Middleware {
	if timeout <= 0 {
		return func(next sub.EventProcessor) sub.EventProcessor { return next }
	}
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1) // buffered so the processor's goroutine can always finish and exit
			go func() {
				done <- next(ctx, eventToPub)
			}()

			select {
			case err := <-done: // event processed (successfully or not)
				return err

			case <-ctx.Done(): // timeout fired, or the caller cancelled
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return goserr.NewTimeoutError(timeout.String(), ctx.Err(),
						fmt.Sprintf("processor for %s did not finish in time", eventToPub.Type))
				}
				return ctx.Err()
			}
		}
	}
}

// WithRecovery recovers from panics in event processor and reports them as an
// errors.ProcessingError, so a panicking handler fails the event instead of reporting success.
//
// It used to return nil, which made a panic indistinguishable from a clean run: PublishSync returned
// no error, WithRetry never retried, and WithLogging logged "completed". If the panic value is
// itself an error it stays reachable through errors.As.
//
// Put this innermost in a Chain (last in the argument list) so the middlewares around it see the
// error and can act on it.
func WithRecovery() Middleware {
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) (err error) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				log.Printf("[Middleware] Recovered from panic in handler for event %s: %v\n%s", eventToPub.Type, r, debug.Stack())

				cause, ok := r.(error)
				if !ok {
					cause = fmt.Errorf("%v", r)
				}
				err = goserr.NewProcessingError(cause, fmt.Sprintf("recovered from panic in handler for event %s", eventToPub.Type))
			}()

			return next(ctx, eventToPub)
		}
	}
}

// WithLogging logs processor execution, prefixing each line with msg.
//
// The formatted line is built into a local: assigning it back to msg appended to the prefix on every
// call, so the line grew without bound across the life of the process, and concurrent events racing
// on that shared string was a data race the -race detector flags.
func WithLogging(msg string, logFunc func(message string)) Middleware {
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) error {
			start := time.Now()
			err := next(ctx, eventToPub)
			duration := time.Since(start)

			if err != nil {
				logFunc(fmt.Sprintf("%s [Middleware] Processor for %s failed after %v: %v", msg, eventToPub.Type, duration, err))
			} else {
				logFunc(fmt.Sprintf("%s [Middleware] Processor for %s completed in %v", msg, eventToPub.Type, duration))
			}

			return err
		}
	}
}

// Chain chains multiple middlewares together.
func Chain(middlewares ...Middleware) Middleware {
	return func(processor sub.EventProcessor) sub.EventProcessor {
		for i := len(middlewares) - 1; i >= 0; i-- {
			processor = middlewares[i](processor)
		}
		return processor
	}
}
