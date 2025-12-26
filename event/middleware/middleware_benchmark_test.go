// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
)

// run from cwd: go test -bench=. -benchmem -cpuprofile=cpu.prof ./ -count=3
// cpu profilin: go tool pprof -http=:8080 cpu.prof
// ram profilin: go test -bench=. -benchmem -memprofile=mem.prof ./ -count=3

// -------------------------------------------- Middleware Benchmarks --------------------------------------------

// BenchmarkWithRetry
//
//	1000000000               0.3517 ns/op          0 B/op          0 allocs/op
//	1000000000               0.3546 ns/op          0 B/op          0 allocs/op
//	1000000000               0.3544 ns/op          0 B/op          0 allocs/op
func BenchmarkWithRetry(b *testing.B) {
	processor := func(ctx context.Context, e *event.Event) error {
		return nil
	}

	wrappedProcessor := WithRetry(3, 10*time.Millisecond)(processor)
	evt := event.NewEvent("test", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedProcessor(ctx, evt)
	}
}

// BenchmarkWithTimeout
//
//	423882              3169 ns/op             560 B/op          8 allocs/op
//	437866              3104 ns/op             560 B/op          8 allocs/op
//	455904              3138 ns/op             560 B/op          8 allocs/op
func BenchmarkWithTimeout(b *testing.B) {
	processor := func(ctx context.Context, e *event.Event) error {
		return nil
	}

	wrappedProcessor := WithTimeout(100 * time.Millisecond)(processor)
	evt := event.NewEvent("test", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedProcessor(ctx, evt)
	}
}

// BenchmarkChain
//
//	10000            173752 ns/op          488133 B/op         13 allocs/op
//	10000            162159 ns/op          488043 B/op         13 allocs/op
//	10000            164082 ns/op          487975 B/op         13 allocs/op
func BenchmarkChain(b *testing.B) {
	processor := func(ctx context.Context, e *event.Event) error {
		return nil
	}

	wrappedProcessor := Chain(
		WithRecovery(),
		WithRetry(2, 10*time.Millisecond),
		WithTimeout(100*time.Millisecond),
		WithLogging("", func(message string) { /* log.Println(message) comment to avoid filling the stdout */ }),
	)(processor)

	evt := event.NewEvent("test", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedProcessor(ctx, evt)
	}
}
