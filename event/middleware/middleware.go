// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package middleware. middleware provides composable middleware for event handlers.
package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// Middleware wraps a sub.EventProcessor with additional behavior.
type Middleware func(sub.EventProcessor) sub.EventProcessor

// -------------------------------------------- Public Functions --------------------------------------------

// WithRetry retries failed event processors with exponential backoff.
// It executes initial + retries.
func WithRetry(maxRetries int, initialDelay time.Duration) Middleware {
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
func WithTimeout(timeout time.Duration) Middleware {
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1) // non blocking async
			go func() {
				done <- next(ctx, eventToPub)
			}()

			select {
			case err := <-done: // event failed
				return err

			case <-ctx.Done(): // timeout fired
				return ctx.Err()
			}
		}
	}
}

// WithRecovery recovers from panics in event processor.
func WithRecovery() Middleware {
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[Middleware] Recovered from panic in handler for event %s: %v", eventToPub.Type, r)
					err = nil
				}
			}()

			return next(ctx, eventToPub)
		}
	}
}

// WithLogging logs processor execution.
func WithLogging(msg string, logFunc func(message string)) Middleware {
	return func(next sub.EventProcessor) sub.EventProcessor {
		return func(ctx context.Context, eventToPub *event.Event) error {
			start := time.Now()
			err := next(ctx, eventToPub)
			duration := time.Since(start)

			if err != nil {
				msg = fmt.Sprintf("%s [Middleware] Processor for %s failed after %v: %v", msg, eventToPub.Type, duration, err)
				logFunc(msg)
			} else {
				msg = fmt.Sprintf("%s [Middleware] Processor for %s completed in %v", msg, eventToPub.Type, duration)
				logFunc(msg)
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
